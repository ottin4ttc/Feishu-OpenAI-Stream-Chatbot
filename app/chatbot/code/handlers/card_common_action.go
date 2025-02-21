package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher/callback"
)

type CardHandlerMeta func(cardMsg CardMsg, m MessageHandler) CardHandlerFunc

type CardHandlerFunc func(ctx context.Context, event *callback.CardActionTriggerEvent) (
	*string, error)

var ErrNextHandler = fmt.Errorf("next handler")

func NewCardHandler(m MessageHandler) CardHandlerFunc {
	handlers := []CardHandlerMeta{
		NewClearCardHandler,
		NewPicResolutionHandler,
		NewPicTextMoreHandler,
		NewPicModeChangeHandler,
		NewRoleTagCardHandler,
		NewRoleCardHandler,
	}

	return func(ctx context.Context, event *callback.CardActionTriggerEvent) (*string, error) {
		var cardMsg CardMsg

		actionValue := event.Event.Action.Value
		actionValueJson, _ := json.Marshal(actionValue)
		json.Unmarshal(actionValueJson, &cardMsg)
		//pp.Println(cardMsg)
		for _, handler := range handlers {
			h := handler(cardMsg, m)
			retVal, err := h(ctx, event)
			if err == ErrNextHandler {
				continue
			}
			return retVal, err
		}
		return nil, nil
	}
}
