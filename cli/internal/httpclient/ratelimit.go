// SPDX-License-Identifier: AGPL-3.0-or-later

package httpclient

import (
	"context"
	"net/url"
	"sync"

	"golang.org/x/time/rate"
)

// DomainRateLimiter maintains a per-domain token bucket rate limiter.
type DomainRateLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*rate.Limiter
	global   *rate.Limiter
}

// NewDomainRateLimiter creates a rate limiter with default global limits.
func NewDomainRateLimiter(rps float64, burst int) *DomainRateLimiter {
	return &DomainRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		global:   rate.NewLimiter(rate.Limit(rps), burst),
	}
}

// SetDomainLimit configures a specific rate limit for a domain.
func (d *DomainRateLimiter) SetDomainLimit(domain string, rps float64, burst int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.limiters[domain] = rate.NewLimiter(rate.Limit(rps), burst)
}

// Wait blocks until the rate limiter for the given URL's domain allows the request.
func (d *DomainRateLimiter) Wait(ctx context.Context, rawURL string) error {
	domain := extractDomain(rawURL)

	d.mu.RLock()
	limiter, ok := d.limiters[domain]
	d.mu.RUnlock()

	if !ok {
		limiter = d.global
	}

	return limiter.Wait(ctx)
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
