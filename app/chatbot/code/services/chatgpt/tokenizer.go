package chatgpt

import (
	"github.com/ottin4ttc/go-openai"
	"github.com/pandodao/tokenizer-go"
	"strings"
)

func CalcTokenLength(text string) int {
	text = strings.TrimSpace(text)
	return tokenizer.MustCalToken(text)
}

func CalcTokenFromMsgList(msgs []openai.ChatCompletionMessage) int {
	var total int
	for _, msg := range msgs {
		total += CalcTokenLength(msg.Content)
	}
	return total
}
