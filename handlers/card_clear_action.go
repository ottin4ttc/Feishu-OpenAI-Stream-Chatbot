package handlers

import (
	"ai-chatbot/services"
	"context"
	larkcard "github.com/larksuite/oapi-sdk-go/v3/card"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher/callback"
)

func NewClearCardHandler(cardMsg CardMsg, m MessageHandler) CardHandlerFunc {
	return func(ctx context.Context, event *callback.CardActionTriggerEvent) (*larkcard.MessageCard, error) {
		if cardMsg.Kind == ClearCardKind {
			newCard, err, done := CommonProcessClearCache(cardMsg, m.sessionCache)
			if done {
				return newCard, err
			}
			return nil, nil
		}
		return nil, ErrNextHandler
	}
}

func CommonProcessClearCache(cardMsg CardMsg, session services.SessionServiceCacheInterface) (
	*larkcard.MessageCard, error, bool) {
	if cardMsg.Value == "1" {
		session.Clear(cardMsg.SessionId)
		newCard, _ := newSendCard(
			withHeader("️🆑 DeepSeek友情提示", larkcard.TemplateGrey),
			withMainMd("已删除此话题的上下文信息"),
			withNote("我们可以开始一个全新的话题，继续找我聊天吧"),
		)
		//fmt.Printf("session: %v", newCard)
		return newCard, nil, true
	}
	if cardMsg.Value == "0" {
		newCard, _ := newSendCard(
			withHeader("️🆑 DeepSeek友情提示", larkcard.TemplateGreen),
			withMainMd("依旧保留此话题的上下文信息"),
			withNote("我们可以继续探讨这个话题,期待和您聊天。如果您有其他问题或者想要讨论的话题，请告诉我哦"),
		)
		return newCard, nil, true
	}
	return nil, nil, false
}
