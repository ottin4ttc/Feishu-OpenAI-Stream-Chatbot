package handlers

import (
	"ai-chatbot/dal/dsDb"
	"ai-chatbot/initialization"
	"ai-chatbot/model"
	"ai-chatbot/services/accesscontrol"
	"ai-chatbot/services/openai"
	"ai-chatbot/services/volcengine"
	"context"
	"encoding/json"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/ottin4ttc/go_common/db"
	"log"
	"strings"
	"time"

	"github.com/k0kubun/pp/v3"
)

type MessageAction struct { /*消息*/
	//chatgpt *chatgpt.ChatGPT
	volc *volcengine.VolcEngine
}

type CardUpdateMessage struct {
	think  string
	ref    string
	answer string
}

func (m *MessageAction) Execute(a *ActionInfo) bool {

	// Add access control
	if initialization.GetConfig().AccessControlEnable &&
		!accesscontrol.CheckAllowAccessThenIncrement(&a.info.userId) {

		msg := fmt.Sprintf("UserId: 【%s】 has accessed max count today! Max access count today %s: 【%d】",
			a.info.userId, accesscontrol.GetCurrentDateFlag(), initialization.GetConfig().AccessControlMaxCountPerUserPerDay)

		_ = sendMsg(*a.ctx, msg, a.info.chatId)
		return false
	}

	cardId, err2 := sendOnProcess(a)
	if err2 != nil {
		return false
	}

	thinkingAnswer := ""
	referenceAnswer := ""
	streamAnswer := ""

	refStream := make(chan string)
	thinkStream := make(chan string)
	answerResponseStream := make(chan string)
	done := make(chan struct{})

	noContentTimeout := time.AfterFunc(10*time.Second, func() {
		pp.Println("no content timeout")
		close(done)
		err := updateTextCardV2(*a.ctx, CardUpdateMessage{answer: "请求超时"}, cardId)
		if err != nil {
			return
		}
	})
	defer noContentTimeout.Stop()
	msg := a.handler.sessionCache.GetMsg(*a.info.sessionId)
	msg = append(msg, openai.Messages{
		Role: "user", Content: a.info.qParsed,
	})
	go func() {
		defer func() {
			if err := recover(); err != nil {
				err := updateTextCardV2(*a.ctx, CardUpdateMessage{answer: "聊天失败"}, cardId)
				if err != nil {
					printErrorMessage(a, msg, err)
					return
				}
			}
		}()

		//log.Printf("UserId: %s , Request: %s", a.info.userId, msg)

		if err := m.volc.StreamChat(*a.ctx, msg, thinkStream, answerResponseStream, refStream); err != nil {
			err := updateTextCardV2(*a.ctx, CardUpdateMessage{answer: "聊天失败"}, cardId)
			if err != nil {
				printErrorMessage(a, msg, err)
				return
			}
			close(done) // 关闭 done 信号
		}

		close(done) // 关闭 done 信号
	}()
	ticker := time.NewTicker(700 * time.Millisecond)
	defer ticker.Stop()
	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				updateMsg := CardUpdateMessage{
					think: thinkingAnswer,
					//ref:    referenceAnswer,
					answer: streamAnswer,
				}
				err := updateTextCardV2(*a.ctx, updateMsg, cardId)
				if err != nil {
					printErrorMessage(a, msg, err)
					return
				}
			}
		}
	}()

	for {
		select {
		case think, ok := <-thinkStream:
			if !ok {
				continue
			}
			noContentTimeout.Stop()
			thinkingAnswer += think
		case ref, ok := <-refStream:
			if !ok {
				continue
			}
			noContentTimeout.Stop()
			referenceAnswer += ref
		case res, ok := <-answerResponseStream:
			if !ok {
				continue
			}
			noContentTimeout.Stop()
			streamAnswer += res
		case <-done:
			updateMsg := CardUpdateMessage{
				think:  thinkingAnswer,
				ref:    referenceAnswer,
				answer: streamAnswer,
			}
			err := updateTextCardV2(*a.ctx, updateMsg, cardId)
			if err != nil {
				printErrorMessage(a, msg, err)
				return false
			}
			ticker.Stop()
			combinedAnswer := thinkingAnswer + "\n" + streamAnswer + "\n" + referenceAnswer
			msg = append(msg, openai.Messages{
				Role: "assistant", Content: combinedAnswer,
			})
			a.handler.sessionCache.SetMsg(*a.info.sessionId, msg)

			//if new topic
			//if len(msg) == 2 {
			//	//fmt.Println("new topic", msg[1].Content)
			//	//updateNewTextCard(*a.ctx, a.info.sessionId, a.info.msgId,
			//	//	completions.Content)
			//}

			jsonByteArray, err := json.Marshal(msg)
			if err != nil {
				log.Printf("Error marshaling JSON request: UserId: %s , Request: %s , Response: %s", a.info.userId, jsonByteArray, combinedAnswer)
			}
			jsonStr := strings.ReplaceAll(string(jsonByteArray), "\\n", "")
			jsonStr = strings.ReplaceAll(jsonStr, "\n", "")
			// // 将双向消息都存入数据库
			askMessage := &model.DsMessage{
				AppID:     a.messageEvent.EventV2Base.Header.AppID,
				TenantID:  a.messageEvent.EventV2Base.Header.TenantKey,
				UnionID:   *a.messageEvent.Event.Sender.SenderId.UnionId,
				UserID:    *a.messageEvent.Event.Sender.SenderId.UserId,
				SendType:  *a.messageEvent.Event.Sender.SenderType,
				MessageID: *a.messageEvent.Event.Message.MessageId,
				ChatID:    *a.messageEvent.Event.Message.ChatId,
				ChatType:  *a.messageEvent.Event.Message.ChatType,
				Content:   a.info.qParsed,
			}
			if a.messageEvent.Event.Message.RootId != nil {
				askMessage.RootID = *a.messageEvent.Event.Message.RootId
			}
			if a.messageEvent.Event.Message.ParentId != nil {
				askMessage.ParentID = *a.messageEvent.Event.Message.ParentId
			}
			jsonContent, _ := jsoniter.MarshalToString(a.messageEvent.Event)
			askMessage.EventJSON = jsonContent
			// reply
			replyMessage := &model.DsMessage{
				AppID:    a.messageEvent.EventV2Base.Header.AppID,
				TenantID: a.messageEvent.EventV2Base.Header.TenantKey,
				UnionID:  initialization.GetConfig().FeishuUnionId,
				SendType: askMessage.SendType,
				ChatType: askMessage.ChatType,
				ChatID:   askMessage.ChatID,
				Content:  combinedAnswer,
				RootID:   askMessage.RootID,
				ParentID: askMessage.MessageID,
			}
			ctx := context.Background()
			err = dsDb.NewDsMessageDao(context.Background(), db.GetPostgreSQL(ctx)).BatchCreate([]*model.DsMessage{askMessage, replyMessage})
			if err != nil {
				printErrorMessage(a, msg, err)
			} else {
				log.Printf("Success request plain jsonStr: UserId: %s , Request: %s , Response: %s",
					a.info.userId, jsonStr, combinedAnswer)
			}
			return false
		}
	}
}

func printErrorMessage(a *ActionInfo, msg []openai.Messages, err error) {
	log.Printf("Failed request: UserId: %s , Request: %s , Err: %s", a.info.userId, msg, err)
}

func sendOnProcess(a *ActionInfo) (*string, error) {
	// send 正在处理中
	cardId, err := sendOnProcessCardV2(*a.ctx, a.info.sessionId, a.info.msgId)
	if err != nil {
		return nil, err
	}
	return cardId, nil

}
