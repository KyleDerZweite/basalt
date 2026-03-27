// SPDX-License-Identifier: AGPL-3.0-or-later

package httpclient

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net"
	"net/http"
	"time"
)

const (
	defaultTimeout         = 15 * time.Second
	defaultConnectTimeout  = 5 * time.Second
	defaultUserAgent       = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	maxBodySize            = 1 << 20 // 1 MB
	defaultMaxRetries      = 2
	defaultRetryBase       = 1 * time.Second
	defaultMaxIdleConns    = 200
	defaultMaxConnsPerHost = 20
)

// Response holds the relevant fields from an HTTP response.
type Response struct {
	StatusCode   int
	Body         string
	FinalURL     string // after redirects
	ResponseTime time.Duration
	RetryAfter   time.Duration // parsed from Retry-After header, 0 if absent
}

// Client is a wrapped HTTP client with configurable timeout, User-Agent, and retries.
type Client struct {
	http           *http.Client
	userAgent      string
	maxRetries     int
	retryBase      time.Duration
	connectTimeout time.Duration
	dnsCache       *DNSCache
}

// Option configures the Client.
type Option func(*Client)

// WithTimeout sets the HTTP request timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.http.Timeout = d
	}
}

// WithUserAgent sets the User-Agent header.
func WithUserAgent(ua string) Option {
	return func(c *Client) {
		c.userAgent = ua
	}
}

// WithTransport sets a custom http.Transport (e.g., for proxy support).
func WithTransport(t http.RoundTripper) Option {
	return func(c *Client) {
		c.http.Transport = t
	}
}

// WithRetries sets the maximum number of retries for transient failures.
func WithRetries(n int) Option {
	return func(c *Client) {
		c.maxRetries = n
	}
}

// WithConnectTimeout sets the connection establishment timeout.
func WithConnectTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.connectTimeout = d
	}
}

// WithDNSCache enables DNS caching for the client.
func WithDNSCache(cache *DNSCache) Option {
	return func(c *Client) {
		c.dnsCache = cache
		// If transport is set, update its DialContext.
		if t, ok := c.http.Transport.(*http.Transport); ok {
			t.DialContext = cache.DialContext
		}
	}
}

// defaultTransport returns a tuned HTTP transport with proper connection pooling.
func defaultTransport(concurrency int) *http.Transport {
	dialer := &net.Dialer{
		Timeout:   defaultConnectTimeout,
		KeepAlive: 30 * time.Second,
	}
	return &http.Transport{
		DialContext:           dialer.DialContext,
		MaxIdleConns:          defaultMaxIdleConns,
		MaxIdleConnsPerHost:   concurrency,
		MaxConnsPerHost:       concurrency,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
	}
}

// New creates a new HTTP client with the given options.
func New(opts ...Option) *Client {
	c := &Client{
		http: &http.Client{
			Timeout: defaultTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
			Transport: defaultTransport(defaultMaxConnsPerHost),
		},
		userAgent:      defaultUserAgent,
		maxRetries:     defaultMaxRetries,
		retryBase:      defaultRetryBase,
		connectTimeout: defaultConnectTimeout,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Do executes a GET request and returns the response.
// It retries on 429, 5xx, and transient network errors with exponential backoff.
func (c *Client) Do(ctx context.Context, url string, headers map[string]string) (*Response, error) {
	return c.DoRequest(ctx, http.MethodGet, url, nil, headers)
}

// DoRequest executes an HTTP request with the given method and optional body.
// It retries on 429, 5xx, and transient network errors with exponential backoff.
func (c *Client) DoRequest(ctx context.Context, method, url string, body io.Reader, headers map[string]string) (*Response, error) {
	var lastErr error

	// If the body is seekable, we can retry with it. Otherwise, read it once
	// and wrap in a bytes.Reader for retries.
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("reading request body: %w", err)
		}
	}

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			delay := c.retryBase * time.Duration(math.Pow(2, float64(attempt-1)))
			slog.Debug("retrying request", "url", url, "attempt", attempt, "delay", delay)

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		var reqBody io.Reader
		if bodyBytes != nil {
			reqBody = bytes.NewReader(bodyBytes)
		}

		resp, err := c.doOnce(ctx, method, url, reqBody, headers)
		if err != nil {
			if isRetryableError(err) && attempt < c.maxRetries {
				lastErr = err
				continue
			}
			return nil, err
		}

		// Retry on 429 (rate limited) and 5xx (server errors).
		if (resp.StatusCode == 429 || resp.StatusCode >= 500) && attempt < c.maxRetries {
			lastErr = fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)

			// Respect Retry-After header if present.
			if resp.RetryAfter > 0 {
				slog.Debug("retrying after Retry-After", "url", url, "delay", resp.RetryAfter)
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(resp.RetryAfter):
				}
				continue
			}

			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// doOnce executes a single HTTP request.
func (c *Client) doOnce(ctx context.Context, method, url string, body io.Reader, headers map[string]string) (*Response, error) {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	return &Response{
		StatusCode:   resp.StatusCode,
		Body:         string(respBytes),
		FinalURL:     resp.Request.URL.String(),
		ResponseTime: time.Since(start),
		RetryAfter:   parseRetryAfter(resp.Header.Get("Retry-After")),
	}, nil
}

// isRetryableError returns true for transient network errors.
func isRetryableError(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) && dnsErr.IsTemporary {
		return true
	}
	return false
}

// parseRetryAfter parses a Retry-After header value (seconds or HTTP date).
func parseRetryAfter(raw string) time.Duration {
	if raw == "" {
		return 0
	}
	// Try integer seconds first.
	if secs, err := time.ParseDuration(raw + "s"); err == nil && secs > 0 {
		return secs
	}
	// Try HTTP date format.
	if t, err := time.Parse(http.TimeFormat, raw); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}
	return 0
}
