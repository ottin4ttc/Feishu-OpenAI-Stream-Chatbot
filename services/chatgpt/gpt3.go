package chatgpt

import (
	"ai-chatbot/initialization"
	customOpenai "ai-chatbot/services/openai"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ottin4ttc/go-openai"
)

type Messages struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatGPT struct {
	config *initialization.Config
}

type Gpt3 interface {
	StreamChat(ctx context.Context,
		msg []customOpenai.Messages,
		responseStream chan string) error
	StreamChatWithHistory(ctx context.Context, msg []openai.ChatCompletionMessage, maxTokens int,
		responseStream chan string) error
}

func NewGpt3(config *initialization.Config) *ChatGPT {
	return &ChatGPT{config: config}
}

func (c *ChatGPT) StreamChat(ctx context.Context,
	msg []customOpenai.Messages,
	responseStream chan string) error {
	//change msg type from Messages to openai.ChatCompletionMessage
	chatMsgs := make([]openai.ChatCompletionMessage, len(msg))
	for i, m := range msg {
		chatMsgs[i] = openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
		}
	}
	return c.StreamChatWithHistory(ctx, chatMsgs, 8092,
		responseStream)
}

func (c *ChatGPT) StreamChatWithHistory(ctx context.Context, msg []openai.ChatCompletionMessage, maxTokens int,
	responseStream chan string,
) error {
	config := openai.DefaultConfig(c.config.OpenaiApiKeys[0])
	config.BaseURL = c.config.OpenaiApiUrl

	proxyClient, parseProxyError := customOpenai.GetProxyClient(c.config.HttpProxy)
	if parseProxyError != nil {
		return parseProxyError
	}
	config.HTTPClient = proxyClient

	client := openai.NewClientWithConfig(config)
	//pp.Printf("client: %v", client)
	req := openai.ChatCompletionRequest{
		Model:       c.config.OpenaiModel,
		Messages:    msg,
		N:           1,
		Temperature: 0.7,
		MaxTokens:   maxTokens,
		TopP:        1,
		//Moderation:     true,
		//ModerationStop: true,
	}
	stream, err := client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		fmt.Errorf("CreateCompletionStream returned error: %v", err)
	}

	defer stream.Close()
	isInReasoningContent := false
	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			fmt.Printf("Stream error: %v\n", err)
			return err
		}

		if response.Choices[0].Delta.ReasoningContent != "" {
			if !isInReasoningContent {
				responseStream <- "\n<think>\n"
				isInReasoningContent = true
			}
			responseStream <- response.Choices[0].Delta.ReasoningContent
		} else {
			if isInReasoningContent {
				responseStream <- "\n</think>\n"
				isInReasoningContent = false
			}
			responseStream <- response.Choices[0].Delta.Content
		}
	}
}
