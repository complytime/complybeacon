package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
)

// CacheableClient wraps the basic client with that leverages a caching mechanism.
type CacheableClient struct {
	client *Client
	cache  *cache.Cache
	mu     sync.Mutex
	logger *zap.Logger
}

// NewCacheableClient creates a new enriched client with caching capabilities.
// If ttl is -1, the cache will never expire (go-cache treats durations < 1ns as no expiration).
// Otherwise, items will expire after the specified TTL with cleanup
// happening at half the TTL interval.
func NewCacheableClient(client *Client, logger *zap.Logger, ttl time.Duration) *CacheableClient {
	var cleanupInterval time.Duration
	if ttl == cache.NoExpiration {
		cleanupInterval = cache.NoExpiration
	} else {
		cleanupInterval = ttl / 2
	}

	cacheInstance := cache.New(ttl, cleanupInterval)

	ec := &CacheableClient{
		client: client,
		cache:  cacheInstance,
		logger: logger,
	}
	return ec
}

// Retrieve gets compliance data for using policy data lookup values.
// Cached metadata is used by default.
func (c *CacheableClient) Retrieve(ctx context.Context, policy Policy) (Compliance, error) {
	if value, found := c.cache.Get(policy.PolicyRuleId); found {
		return value.(Compliance), nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Fetch metadata from API on cache miss
	compliance, err := c.fetchMetadata(ctx, policy)
	if err != nil {
		return Compliance{}, fmt.Errorf("failed to fetch metadata: %w", err)
	}

	// Use the same expiration as configured for the cache
	c.cache.Set(policy.PolicyRuleId, compliance, cache.DefaultExpiration)

	return compliance, nil
}

func (c *CacheableClient) fetchMetadata(ctx context.Context, policy Policy) (Compliance, error) {
	req := EnrichmentRequest{Policy: policy}

	resp, err := c.callEnrich(ctx, req)
	if err != nil {
		return Compliance{}, fmt.Errorf("failed to call metadata API: %w", err)
	}

	return resp.Compliance, nil
}

func (c *CacheableClient) callEnrich(ctx context.Context, req EnrichmentRequest) (*EnrichmentResponse, error) {
	resp, err := c.client.PostV1Enrich(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	parsedResp, err := ParsePostV1EnrichResponse(resp)
	if err != nil {
		return nil, err
	}

	if parsedResp.JSON200 == nil {
		return nil, fmt.Errorf("unexpected response status: %s", resp.Status)
	}

	return parsedResp.JSON200, nil
}
