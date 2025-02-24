package volcengine

import (
	"ai-chatbot/initialization"
	customOpenai "ai-chatbot/services/openai"
	"context"
	"errors"
	"fmt"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	"io"
)

type Messages struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type VolcEngine struct {
	config *initialization.Config
}

func NewVolcEngine(config *initialization.Config) *VolcEngine {
	return &VolcEngine{config: config}
}

func (e *VolcEngine) StreamChat(ctx context.Context, msg []customOpenai.Messages, thinkStream, answerStream, refStream chan string) error {
	chatMsgs := make([]*model.ChatCompletionMessage, len(msg))
	for i, m := range msg {
		chatMsgs[i] = &model.ChatCompletionMessage{
			Role:    m.Role,
			Content: &model.ChatCompletionMessageContent{StringValue: volcengine.String(m.Content)},
		}
	}
	return e.StreamChatWithHistory(ctx, chatMsgs, 8092, thinkStream, answerStream, refStream)
}

func (e *VolcEngine) StreamChatWithHistory(ctx context.Context, msg []*model.ChatCompletionMessage, maxTokens int, thinkStream, answerStream, refStream chan string) error {
	client := arkruntime.NewClientWithApiKey(
		// 从环境变量中获取您的 API Key。此为默认方式，您可根据需要进行修改
		e.config.OpenaiApiKeys[0],
		// 此为默认路径，您可根据业务所在地域进行配置
		arkruntime.WithBaseUrl(e.config.OpenaiApiUrl),
	)
	req := model.BotChatCompletionRequest{
		BotId:       e.config.OpenaiModel,
		Messages:    msg,
		N:           1,
		Temperature: 0.7,
		MaxTokens:   maxTokens,
		TopP:        1,
	}
	stream, err := client.CreateBotChatCompletionStream(ctx, req)
	if err != nil {
		fmt.Errorf("CreateBotChatCompletionStream returned error: %v", err)
	}
	defer stream.Close()
	//isInReasoningContent := false
	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			fmt.Printf("Stream error: %v\n", err)
			return err
		}
		if len(response.Choices) > 0 {
			if response.References != nil {
				for _, ref := range response.References {
					refStream <- fmt.Sprintf("[%s](%s)\n", ref.Title, ref.Url)
				}
			}
			if response.Choices[0].Delta.ReasoningContent != nil {
				thinkStream <- *response.Choices[0].Delta.ReasoningContent
			} else {
				answerStream <- response.Choices[0].Delta.Content
			}
		}
	}
}
