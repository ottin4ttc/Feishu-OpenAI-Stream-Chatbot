package initialization

import (
	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
)

var larkClient *lark.Client
var larkWsClient *larkws.Client

func LoadLarkWsClient(config Config, handler *dispatcher.EventDispatcher) {
	larkWsClient = larkws.NewClient(config.FeishuAppId, config.FeishuAppSecret,
		larkws.WithEventHandler(handler),
		larkws.WithLogLevel(larkcore.LogLevelDebug))
}

func LoadLarkClient(config Config) {
	larkClient = lark.NewClient(config.FeishuAppId, config.FeishuAppSecret)
}

func GetLarkWsClient() *larkws.Client {
	return larkWsClient
}

func GetLarkClient() *lark.Client { return larkClient }
