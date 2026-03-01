// SPDX-License-Identifier: AGPL-3.0-or-later
//
// ControlCache caches control (nonexistent user) HTTP responses per site.
// This avoids making duplicate control requests for the same site within a
// scan, cutting HTTP requests nearly in half for large site databases.

package engine

import (
	"context"
	"sync"
	"time"

	"github.com/kyle/basalt/internal/httpclient"
)

// controlEntry holds a cached control response.
type controlEntry struct {
	body      string
	status    int
	fetchedAt time.Time
}

// ControlCache provides thread-safe caching of control responses per site.
type ControlCache struct {
	mu      sync.RWMutex
	entries map[string]*controlEntry
	client  *httpclient.Client
	limiter *httpclient.DomainRateLimiter
	ttl     time.Duration
}

// NewControlCache creates a cache with the given TTL.
func NewControlCache(client *httpclient.Client, limiter *httpclient.DomainRateLimiter, ttl time.Duration) *ControlCache {
	return &ControlCache{
		entries: make(map[string]*controlEntry),
		client:  client,
		limiter: limiter,
		ttl:     ttl,
	}
}

// Get returns the cached control body for a site, fetching it if needed.
// The key should be unique per site (e.g., site name or URL template).
// controlURL is the fully-resolved URL with the nonexistent username substituted.
func (c *ControlCache) Get(ctx context.Context, key string, controlURL string, headers map[string]string) (body string, status int, err error) {
	// Check cache first.
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if ok && time.Since(entry.fetchedAt) < c.ttl {
		return entry.body, entry.status, nil
	}

	// Cache miss or expired - fetch.
	if c.limiter != nil {
		if err := c.limiter.Wait(ctx, controlURL); err != nil {
			return "", 0, err
		}
	}

	resp, err := c.client.Do(ctx, controlURL, headers)
	if err != nil {
		return "", 0, err
	}

	// Store in cache.
	c.mu.Lock()
	c.entries[key] = &controlEntry{
		body:      resp.Body,
		status:    resp.StatusCode,
		fetchedAt: time.Now(),
	}
	c.mu.Unlock()

	return resp.Body, resp.StatusCode, nil
}

// Len returns the number of cached entries.
func (c *ControlCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}
