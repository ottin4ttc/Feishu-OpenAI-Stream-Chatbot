package handlers

import (
	"ai-chatbot/initialization"
	"ai-chatbot/services"
	"ai-chatbot/services/openai"
	"context"
	larkcard "github.com/larksuite/oapi-sdk-go/v3/card"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher/callback"
)

func NewRoleTagCardHandler(cardMsg CardMsg,
	m MessageHandler) CardHandlerFunc {
	return func(ctx context.Context, event *callback.CardActionTriggerEvent) (*larkcard.MessageCard, error) {

		if cardMsg.Kind == RoleTagsChooseKind {
			newCard, err, done := CommonProcessRoleTag(cardMsg, event,
				m.sessionCache)
			if done {
				return newCard, err
			}
			return nil, nil
		}
		return nil, ErrNextHandler
	}
}

func NewRoleCardHandler(cardMsg CardMsg,
	m MessageHandler) CardHandlerFunc {
	return func(ctx context.Context, event *callback.CardActionTriggerEvent) (*larkcard.MessageCard, error) {

		if cardMsg.Kind == RoleChooseKind {
			newCard, err, done := CommonProcessRole(cardMsg, event,
				m.sessionCache)
			if done {
				return newCard, err
			}
			return nil, nil
		}
		return nil, ErrNextHandler
	}
}

func CommonProcessRoleTag(msg CardMsg, event *callback.CardActionTriggerEvent, cache services.SessionServiceCacheInterface) (*larkcard.MessageCard, error, bool) {
	option := event.Event.Action.Option
	//replyMsg(context.Background(), "已选择tag:"+option,
	//	&msg.MsgId)
	roles := initialization.GetTitleListByTag(option)
	//fmt.Printf("roles: %s", roles)
	SendRoleListCard(context.Background(), &msg.SessionId,
		&msg.MsgId, option, *roles)
	return nil, nil, true
}

func CommonProcessRole(msg CardMsg, event *callback.CardActionTriggerEvent, cache services.SessionServiceCacheInterface) (*larkcard.MessageCard, error, bool) {
	option := event.Event.Action.Option
	contentByTitle, error := initialization.GetFirstRoleContentByTitle(option)
	if error != nil {
		return nil, error, true
	}
	cache.Clear(msg.SessionId)
	systemMsg := append([]openai.Messages{}, openai.Messages{
		Role: "system", Content: contentByTitle,
	})
	cache.SetMsg(msg.SessionId, systemMsg)
	//pp.Println("systemMsg: ", systemMsg)
	sendSystemInstructionCard(context.Background(), &msg.SessionId,
		&msg.MsgId, contentByTitle)
	//replyMsg(context.Background(), "已选择角色:"+contentByTitle,
	//	&msg.MsgId)
	return nil, nil, true
}
