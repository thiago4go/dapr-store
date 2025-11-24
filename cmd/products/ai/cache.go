package ai

import (
	"context"
	"fmt"
	"time"

	dapr "github.com/dapr/go-sdk/client"
)

const (
	stateStoreName = "statestore"
	cacheTTL       = 24 * time.Hour
)

type Cache struct {
	daprClient dapr.Client
}

func NewCache(daprClient dapr.Client) *Cache {
	return &Cache{
		daprClient: daprClient,
	}
}

func (c *Cache) Get(ctx context.Context, productID string) (string, error) {
	key := fmt.Sprintf("ai-desc-%s", productID)
	item, err := c.daprClient.GetState(ctx, stateStoreName, key, nil)
	if err != nil {
		return "", err
	}
	if len(item.Value) == 0 {
		return "", nil
	}
	return string(item.Value), nil
}

func (c *Cache) Set(ctx context.Context, productID, description string) error {
	key := fmt.Sprintf("ai-desc-%s", productID)
	metadata := map[string]string{
		"ttlInSeconds": fmt.Sprintf("%d", int(cacheTTL.Seconds())),
	}
	return c.daprClient.SaveState(ctx, stateStoreName, key, []byte(description), metadata)
}
