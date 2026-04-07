// SPDX-License-Identifier: AGPL-3.0-or-later
//
// Proxy pool with round-robin rotation.
// Supports HTTP and SOCKS5 proxies.

package httpclient

import (
	"bufio"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync/atomic"

	"golang.org/x/net/proxy"
)

// ProxyPool rotates through a list of proxy URLs using round-robin.
type ProxyPool struct {
	proxies []*url.URL
	index   atomic.Int64
}

// NewProxyPool creates a pool from proxy URL strings.
// Each proxy should be a URL like "http://host:port" or "socks5://host:port".
func NewProxyPool(proxyURLs []string) (*ProxyPool, error) {
	pool := &ProxyPool{}
	for _, raw := range proxyURLs {
		raw = strings.TrimSpace(raw)
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}
		u, err := url.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL %q: %w", raw, err)
		}
		if u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "socks5" {
			return nil, fmt.Errorf("unsupported proxy scheme %q (use http, https, or socks5)", u.Scheme)
		}
		pool.proxies = append(pool.proxies, u)
	}
	if len(pool.proxies) == 0 {
		return nil, fmt.Errorf("no valid proxies provided")
	}
	return pool, nil
}

// LoadProxyFile reads proxy URLs from a file (one per line).
func LoadProxyFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var proxies []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			proxies = append(proxies, line)
		}
	}
	return proxies, scanner.Err()
}

// Transport returns an http.RoundTripper that rotates through proxies.
func (p *ProxyPool) Transport() http.RoundTripper {
	t := defaultTransport(defaultMaxConnsPerHost, defaultConnectTimeout)
	t.Proxy = func(req *http.Request) (*url.URL, error) {
		return p.next(), nil
	}
	return t
}

// SOCKS5Transport returns an http.RoundTripper for a single SOCKS5 proxy.
// Use this when the pool has exactly one SOCKS5 proxy for simpler setup.
func SOCKS5Transport(proxyURL *url.URL) (http.RoundTripper, error) {
	auth := &proxy.Auth{}
	if proxyURL.User != nil {
		auth.User = proxyURL.User.Username()
		auth.Password, _ = proxyURL.User.Password()
	} else {
		auth = nil
	}

	dialer, err := proxy.SOCKS5("tcp", proxyURL.Host, auth, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("creating SOCKS5 dialer: %w", err)
	}

	contextDialer, ok := dialer.(proxy.ContextDialer)
	if !ok {
		return nil, fmt.Errorf("SOCKS5 dialer does not support DialContext")
	}

	return &http.Transport{
		DialContext: contextDialer.DialContext,
	}, nil
}

// next returns the next proxy in round-robin order.
func (p *ProxyPool) next() *url.URL {
	idx := p.index.Add(1) - 1
	return p.proxies[idx%int64(len(p.proxies))]
}

// Len returns the number of proxies in the pool.
func (p *ProxyPool) Len() int {
	return len(p.proxies)
}
