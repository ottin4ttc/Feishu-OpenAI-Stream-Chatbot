package handlers

import (
	"ai-chatbot/dal/dsDb"
	"ai-chatbot/initialization"
	"ai-chatbot/model"
	"ai-chatbot/services/accesscontrol"
	"ai-chatbot/services/chatgpt"
	"ai-chatbot/services/openai"
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
	chatgpt *chatgpt.ChatGPT
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

	answer := ""
	chatResponseStream := make(chan string)
	done := make(chan struct{}) // 添加 done 信号，保证 goroutine 正确退出
	noContentTimeout := time.AfterFunc(10*time.Second, func() {
		pp.Println("no content timeout")
		close(done)
		err := updateFinalCard(*a.ctx, "请求超时", cardId)
		if err != nil {
			return
		}
		return
	})
	defer noContentTimeout.Stop()
	msg := a.handler.sessionCache.GetMsg(*a.info.sessionId)
	msg = append(msg, openai.Messages{
		Role: "user", Content: a.info.qParsed,
	})
	go func() {
		defer func() {
			if err := recover(); err != nil {
				err := updateFinalCard(*a.ctx, "聊天失败", cardId)
				if err != nil {
					printErrorMessage(a, msg, err)
					return
				}
			}
		}()

		//log.Printf("UserId: %s , Request: %s", a.info.userId, msg)

		if err := m.chatgpt.StreamChat(*a.ctx, msg, chatResponseStream); err != nil {
			err := updateFinalCard(*a.ctx, "聊天失败", cardId)
			if err != nil {
				printErrorMessage(a, msg, err)
				return
			}
			close(done) // 关闭 done 信号
		}

		close(done) // 关闭 done 信号
	}()
	ticker := time.NewTicker(700 * time.Millisecond)
	defer ticker.Stop() // 注意在函数结束时停止 ticker
	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				err := updateTextCard(*a.ctx, answer, cardId)
				if err != nil {
					printErrorMessage(a, msg, err)
					return
				}
			}
		}
	}()

	for {
		select {
		case res, ok := <-chatResponseStream:
			if !ok {
				return false
			}
			noContentTimeout.Stop()
			answer += res
		case <-done: // 添加 done 信号的处理
			err := updateFinalCard(*a.ctx, answer, cardId)
			if err != nil {
				printErrorMessage(a, msg, err)
				return false
			}
			ticker.Stop()
			msg := append(msg, openai.Messages{
				Role: "assistant", Content: answer,
			})
			a.handler.sessionCache.SetMsg(*a.info.sessionId, msg)
			close(chatResponseStream)
			//if new topic
			//if len(msg) == 2 {
			//	//fmt.Println("new topic", msg[1].Content)
			//	//updateNewTextCard(*a.ctx, a.info.sessionId, a.info.msgId,
			//	//	completions.Content)
			//}

			jsonByteArray, err := json.Marshal(msg)
			if err != nil {
				log.Printf("Error marshaling JSON request: UserId: %s , Request: %s , Response: %s", a.info.userId, jsonByteArray, answer)
			}
			jsonStr := strings.ReplaceAll(string(jsonByteArray), "\\n", "")
			jsonStr = strings.ReplaceAll(jsonStr, "\n", "")
			// 将双向消息都存入数据库
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
				Content:  answer,
				RootID:   askMessage.RootID,
				ParentID: askMessage.MessageID,
			}
			ctx := context.Background()
			err = dsDb.NewDsMessageDao(context.Background(), db.GetPostgreSQL(ctx)).BatchCreate([]*model.DsMessage{askMessage, replyMessage})
			if err != nil {
				printErrorMessage(a, msg, err)
			} else {
				log.Printf("Success request plain jsonStr: UserId: %s , Request: %s , Response: %s",
					a.info.userId, jsonStr, answer)
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
	cardId, err := sendOnProcessCard(*a.ctx, a.info.sessionId, a.info.msgId)
	if err != nil {
		return nil, err
	}
	return cardId, nil

}
