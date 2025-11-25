package ai

import (
	"context"
	"fmt"
	"sync"
	"time"

	dapr "github.com/dapr/go-sdk/client"
)

const (
	stateStoreName = "statestore"
	cacheTTL       = 24 * time.Hour
)

type Cache struct {
	daprClient dapr.Client
	memCache   map[string]cacheEntry
	mu         sync.RWMutex
}

type cacheEntry struct {
	value     string
	expiresAt time.Time
}

func NewCache(daprClient dapr.Client) *Cache {
	return &Cache{
		daprClient: daprClient,
		memCache:   make(map[string]cacheEntry),
	}
}

// NewMemoryCache creates a cache without Dapr (in-memory only)
func NewMemoryCache() *Cache {
	return &Cache{
		memCache: make(map[string]cacheEntry),
	}
}

func (c *Cache) Get(ctx context.Context, productID string) (string, error) {
	key := fmt.Sprintf("ai-desc-%s", productID)
	
	// Try memory cache first
	c.mu.RLock()
	if entry, ok := c.memCache[key]; ok {
		if time.Now().Before(entry.expiresAt) {
			c.mu.RUnlock()
			return entry.value, nil
		}
	}
	c.mu.RUnlock()
	
	// Try Dapr if available
	if c.daprClient != nil {
		item, err := c.daprClient.GetState(ctx, stateStoreName, key, nil)
		if err != nil {
			return "", err
		}
		if len(item.Value) == 0 {
			return "", nil
		}
		return string(item.Value), nil
	}
	
	return "", nil
}

func (c *Cache) Set(ctx context.Context, productID, description string) error {
	key := fmt.Sprintf("ai-desc-%s", productID)
	
	// Store in memory cache
	c.mu.Lock()
	c.memCache[key] = cacheEntry{
		value:     description,
		expiresAt: time.Now().Add(cacheTTL),
	}
	c.mu.Unlock()
	
	// Store in Dapr if available
	if c.daprClient != nil {
		metadata := map[string]string{
			"ttlInSeconds": fmt.Sprintf("%d", int(cacheTTL.Seconds())),
		}
		return c.daprClient.SaveState(ctx, stateStoreName, key, []byte(description), metadata)
	}
	
	return nil
}
