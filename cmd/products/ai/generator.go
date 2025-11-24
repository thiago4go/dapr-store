package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/ai/azopenai"
)

const generationTimeout = 5 * time.Second

func (c *Client) GenerateDescription(ctx context.Context, productName, currentDesc string) (string, error) {
	if !isPlaceholder(currentDesc) {
		return currentDesc, nil
	}

	ctx, cancel := context.WithTimeout(ctx, generationTimeout)
	defer cancel()

	prompt := fmt.Sprintf("Write a compelling 2-3 sentence product description for: %s", productName)
	
	systemMsg := "You are a creative product description writer. Write engaging, concise descriptions."
	
	messages := []azopenai.ChatRequestMessageClassification{
		&azopenai.ChatRequestSystemMessage{
			Content: &systemMsg,
		},
		&azopenai.ChatRequestUserMessage{
			Content: azopenai.NewChatRequestUserMessageContent(prompt),
		},
	}

	resp, err := c.openai.GetChatCompletions(ctx, azopenai.ChatCompletionsOptions{
		Messages:       messages,
		DeploymentName: &c.deployment,
		MaxTokens:      toPtr(int32(200)),
		Temperature:    toPtr(float32(0.7)),
	}, nil)

	if err != nil {
		return "", fmt.Errorf("failed to generate description: %w", err)
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message == nil || resp.Choices[0].Message.Content == nil {
		return "", fmt.Errorf("no description generated")
	}

	return *resp.Choices[0].Message.Content, nil
}

func isPlaceholder(desc string) bool {
	if len(desc) < 20 {
		return true
	}
	if strings.Contains(desc, "...") {
		return true
	}
	return false
}

func toPtr[T any](v T) *T {
	return &v
}
