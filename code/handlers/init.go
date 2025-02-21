package handlers

import (
	"context"
	"start-feishubot/initialization"
	"start-feishubot/services/openai"

	larkcard "github.com/larksuite/oapi-sdk-go/v3/card"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher/callback"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

type MessageHandlerInterface interface {
	msgReceivedHandler(ctx context.Context, event *larkim.P2MessageReceiveV1) error
	cardHandler(ctx context.Context, event *callback.CardActionTriggerEvent) (*string, error)
}

type HandlerType string

const (
	GroupHandler = "group"
	UserHandler  = "personal"
)

// handlers 所有消息类型类型的处理器
var handlers MessageHandlerInterface

func InitHandlers(gpt *openai.ChatGPT, config initialization.Config) {
	handlers = NewMessageHandler(gpt, config)
}

func Handler(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
	return handlers.msgReceivedHandler(ctx, event)
}

func ReadHandler(ctx context.Context, event *larkim.P2MessageReadV1) error {
	_ = event.Event.Reader.ReaderId.OpenId
	//fmt.Printf("msg is read by : %v \n", *readerId)
	return nil
}

func CardHandler(ctx context.Context, event *callback.CardActionTriggerEvent) (*callback.CardActionTriggerResponse, error) {
	content, err := handlers.cardHandler(ctx, event)
	if err != nil {
		return nil, err
	}
	toast := &callback.Toast{
		Type:        "info",
		Content:     *content,
		I18nContent: nil,
	}
	return &callback.CardActionTriggerResponse{Toast: toast}, nil

}

func judgeCardType(cardAction *larkcard.CardAction) HandlerType {
	actionValue := cardAction.Action.Value
	chatType := actionValue["chatType"]
	//fmt.Printf("chatType: %v", chatType)
	if chatType == "group" {
		return GroupHandler
	}
	if chatType == "personal" {
		return UserHandler
	}
	return "otherChat"
}

func judgeChatType(event *larkim.P2MessageReceiveV1) HandlerType {
	chatType := event.Event.Message.ChatType
	if *chatType == "group" {
		return GroupHandler
	}
	if *chatType == "p2p" {
		return UserHandler
	}
	return "otherChat"
}
