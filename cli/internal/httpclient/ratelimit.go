// SPDX-License-Identifier: AGPL-3.0-or-later

package httpclient

import (
	"context"
	"math/rand"
	"net/url"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// DomainRateLimiter maintains a per-domain token bucket rate limiter.
type DomainRateLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*rate.Limiter
	global   *rate.Limiter
	jitter   time.Duration
}

// NewDomainRateLimiter creates a rate limiter with default global limits.
func NewDomainRateLimiter(rps float64, burst int) *DomainRateLimiter {
	return &DomainRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		global:   rate.NewLimiter(rate.Limit(rps), burst),
		jitter:   50 * time.Millisecond,
	}
}

// SetDomainLimit configures a specific rate limit for a domain.
func (d *DomainRateLimiter) SetDomainLimit(domain string, rps float64, burst int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.limiters[domain] = rate.NewLimiter(rate.Limit(rps), burst)
}

// Wait blocks until the rate limiter for the given URL's domain allows the request.
// Adds a small random jitter to avoid burst patterns that look bot-like.
func (d *DomainRateLimiter) Wait(ctx context.Context, rawURL string) error {
	domain := extractDomain(rawURL)

	d.mu.RLock()
	limiter, ok := d.limiters[domain]
	d.mu.RUnlock()

	if !ok {
		limiter = d.global
	}

	if err := limiter.Wait(ctx); err != nil {
		return err
	}

	// Jitter: small random delay to break burst patterns.
	if d.jitter > 0 {
		j := time.Duration(rand.Int63n(int64(d.jitter)))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(j):
		}
	}

	return nil
}

// ExtractDomain returns the hostname from a URL string.
func ExtractDomain(rawURL string) string {
	return extractDomain(rawURL)
}

func extractDomain(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return u.Hostname()
}
