package main

import (
	"ai-chatbot/dal/dsDb"
	"ai-chatbot/handlers"
	"ai-chatbot/initialization"
	"ai-chatbot/services/openai"
	"ai-chatbot/utils"
	"context"
	"fmt"
	"github.com/ottin4ttc/go_common/db"
	"io"
	"log"
	"os"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/spf13/pflag"

	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher/callback"
)

func main() {
	initialization.InitRoleList()
	pflag.Parse()
	globalConfig := initialization.GetConfig()
	err := dsDb.InitAwsPostgreSQL(context.Background(), db.PostgreSQLConfig{DSN: globalConfig.Dsn})
	if err != nil {
		panic(err)
	}

	// 打印一下实际读取到的配置
	//globalConfigPrettyString, _ := json.MarshalIndent(globalConfig, "", "    ")
	//log.Println(string(globalConfigPrettyString))

	gpt := openai.NewChatGPT(*globalConfig)
	handlers.InitHandlers(gpt, *globalConfig)

	if globalConfig.EnableLog {
		logger := enableLog()
		defer utils.CloseLogger(logger)
	}

	eventHandler := dispatcher.NewEventDispatcher(
		globalConfig.FeishuAppVerificationToken, globalConfig.FeishuAppEncryptKey).
		OnP2MessageReceiveV1(handlers.Handler).
		OnP2MessageReadV1(func(ctx context.Context, event *larkim.P2MessageReadV1) error {
			return handlers.ReadHandler(ctx, event)
		}).
		OnP2CardActionTrigger(func(ctx context.Context, event *callback.CardActionTriggerEvent) (*callback.CardActionTriggerResponse, error) {
			//fmt.Printf("[ OnP2CardActionTrigger access ], data: %s\n", larkcore.Prettify(event))
			return handlers.CardHandler(ctx, event)
		}).
		// 监听「拉取链接预览数据 url.preview.get」
		OnP2CardURLPreviewGet(func(ctx context.Context, event *callback.URLPreviewGetEvent) (*callback.URLPreviewGetResponse, error) {
			fmt.Printf("[ OnP2URLPreviewAction access ], data: %s\n", larkcore.Prettify(event))
			return nil, nil
		})
	initialization.LoadLarkWsClient(*globalConfig, eventHandler)
	initialization.LoadLarkClient(*globalConfig)
	err = initialization.GetLarkWsClient().Start(context.Background())
	if err != nil {
		panic(err)
	}
}

func enableLog() *lumberjack.Logger {
	// Set up the logger
	var logger *lumberjack.Logger

	logger = &lumberjack.Logger{
		Filename: "logs/app.log",
		MaxSize:  100,      // megabytes
		MaxAge:   365 * 10, // days
	}

	fmt.Printf("logger %T\n", logger)

	// Set up the logger to write to both file and console
	log.SetOutput(io.MultiWriter(logger, os.Stdout))
	log.SetFlags(log.Ldate | log.Ltime)

	// Write some log messages
	log.Println("Starting application...")

	return logger
}
