package ai

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/ai/azopenai"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

type Client struct {
	openai     *azopenai.Client
	deployment string
}

func NewClient(ctx context.Context) (*Client, error) {
	endpoint := os.Getenv("AZURE_OPENAI_ENDPOINT")
	deployment := os.Getenv("AZURE_OPENAI_DEPLOYMENT")

	if endpoint == "" || deployment == "" {
		return nil, fmt.Errorf("AZURE_OPENAI_ENDPOINT and AZURE_OPENAI_DEPLOYMENT must be set")
	}

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create credential: %w", err)
	}

	client, err := azopenai.NewClient(endpoint, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI client: %w", err)
	}

	return &Client{
		openai:     client,
		deployment: deployment,
	}, nil
}
