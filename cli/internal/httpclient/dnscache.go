// SPDX-License-Identifier: AGPL-3.0-or-later
//
// DNSCache provides a lightweight in-process DNS cache that persists
// for the lifetime of a scan. It avoids redundant DNS lookups for
// the same hostname across hundreds of concurrent requests.

package httpclient

import (
	"context"
	"net"
	"sync"
	"time"
)

// dnsEntry holds a resolved address with an expiry time.
type dnsEntry struct {
	addrs    []string
	resolved time.Time
	ttl      time.Duration
}

// DNSCache caches DNS lookups in-process with a configurable TTL.
type DNSCache struct {
	mu      sync.RWMutex
	entries map[string]*dnsEntry
	ttl     time.Duration
}

// NewDNSCache creates a DNS cache with the given TTL.
func NewDNSCache(ttl time.Duration) *DNSCache {
	return &DNSCache{
		entries: make(map[string]*dnsEntry),
		ttl:     ttl,
	}
}

// Lookup returns the first resolved address for the hostname, using the cache
// if available. Falls back to the system resolver on cache miss.
func (d *DNSCache) Lookup(ctx context.Context, hostname string) (string, error) {
	d.mu.RLock()
	entry, ok := d.entries[hostname]
	d.mu.RUnlock()

	if ok && time.Since(entry.resolved) < entry.ttl && len(entry.addrs) > 0 {
		return entry.addrs[0], nil
	}

	// Cache miss -- resolve.
	addrs, err := net.DefaultResolver.LookupHost(ctx, hostname)
	if err != nil {
		return "", err
	}
	if len(addrs) == 0 {
		return "", &net.DNSError{Err: "no addresses found", Name: hostname}
	}

	d.mu.Lock()
	d.entries[hostname] = &dnsEntry{
		addrs:    addrs,
		resolved: time.Now(),
		ttl:      d.ttl,
	}
	d.mu.Unlock()

	return addrs[0], nil
}

// DialContext returns a DialContext function suitable for use with
// http.Transport. It resolves via the cache before dialing.
func (d *DNSCache) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return (&net.Dialer{}).DialContext(ctx, network, addr)
	}

	resolved, err := d.Lookup(ctx, host)
	if err != nil {
		// Fallback to direct dial on DNS failure.
		return (&net.Dialer{}).DialContext(ctx, network, addr)
	}

	return (&net.Dialer{}).DialContext(ctx, network, net.JoinHostPort(resolved, port))
}
