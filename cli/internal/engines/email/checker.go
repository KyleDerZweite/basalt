// SPDX-License-Identifier: AGPL-3.0-or-later
//
// EmailEngine checks whether an email address is registered on various services
// by running pluggable Go modules that implement the Module interface.
//
// GDPR Note: This engine only probes publicly accessible registration/login
// endpoints. No authentication bypass or private data access is performed.
// Results should only be used with the data subject's explicit consent.

package email

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/kyle/basalt/internal/engine"
	"github.com/kyle/basalt/internal/httpclient"
)

// Engine implements engine.Engine for email-based lookups.
type Engine struct {
	client      *httpclient.Client
	rateLimiter *httpclient.DomainRateLimiter
	modules     []Module
	concurrency int
	threshold   float64
}

// Option configures the email Engine.
type Option func(*Engine)

// WithModules sets the email check modules.
func WithModules(modules []Module) Option {
	return func(e *Engine) {
		e.modules = modules
	}
}

// WithConcurrency sets the max concurrent module checks.
func WithConcurrency(n int) Option {
	return func(e *Engine) {
		e.concurrency = n
	}
}

// WithRateLimiter sets the per-domain rate limiter.
func WithRateLimiter(rl *httpclient.DomainRateLimiter) Option {
	return func(e *Engine) {
		e.rateLimiter = rl
	}
}

// New creates a new email Engine.
func New(client *httpclient.Client, threshold float64, opts ...Option) *Engine {
	e := &Engine{
		client:      client,
		threshold:   threshold,
		concurrency: 20,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func (e *Engine) Name() string                 { return "email-checker" }
func (e *Engine) SeedTypes() []engine.SeedType  { return []engine.SeedType{engine.SeedEmail} }

// Check runs all registered modules concurrently and streams results.
func (e *Engine) Check(ctx context.Context, seed engine.Seed, results chan<- engine.Result) {
	defer close(results)

	if len(e.modules) == 0 {
		return
	}

	slog.Info("checking email modules", "count", len(e.modules), "email", seed.Value)

	var wg sync.WaitGroup
	sem := make(chan struct{}, e.concurrency)

	for _, mod := range e.modules {
		wg.Add(1)
		sem <- struct{}{} // acquire

		go func(m Module) {
			defer wg.Done()
			defer func() { <-sem }() // release

			select {
			case <-ctx.Done():
				return
			default:
			}

			result := e.runModule(ctx, seed, m)

			select {
			case results <- result:
			case <-ctx.Done():
			}
		}(mod)
	}

	wg.Wait()
}

// runModule executes a single email module and converts the result.
func (e *Engine) runModule(ctx context.Context, seed engine.Seed, mod Module) engine.Result {
	result := engine.Result{
		EngineName: e.Name(),
		Seed:       seed,
		SiteName:   mod.Name(),
		Category:   mod.Category(),
	}

	modResult := mod.Check(ctx, seed.Value, e.client)

	if modResult.Err != nil {
		result.Err = fmt.Errorf("%s: %w", mod.Name(), modResult.Err)
		slog.Debug("email module error", "module", mod.Name(), "error", modResult.Err)
		return result
	}

	if modResult.RateLimit {
		result.Err = fmt.Errorf("%s: rate limited", mod.Name())
		return result
	}

	// Inconclusive — skip.
	if modResult.Exists == nil {
		result.Err = fmt.Errorf("%s: inconclusive", mod.Name())
		return result
	}

	// Compute confidence.
	if *modResult.Exists {
		result.Confidence = 0.95
		result.Exists = true

		// Recovery info is proof of existence.
		if modResult.EmailRecovery != "" || modResult.PhoneRecovery != "" {
			result.Confidence = 1.0
		}
	} else {
		result.Confidence = 0.05
		result.Exists = false
	}

	// Build metadata.
	metadata := make(map[string]string)
	if modResult.Method != "" {
		metadata["method"] = modResult.Method
	}
	if modResult.EmailRecovery != "" {
		metadata["email_recovery"] = modResult.EmailRecovery
	}
	if modResult.PhoneRecovery != "" {
		metadata["phone_recovery"] = modResult.PhoneRecovery
	}
	for k, v := range modResult.Metadata {
		metadata[k] = v
	}
	if len(metadata) > 0 {
		result.Metadata = metadata
	}

	// Discovered recovery emails/phones become pivot seeds.
	if result.Exists {
		if modResult.EmailRecovery != "" {
			result.DiscoveredSeeds = append(result.DiscoveredSeeds, engine.Seed{
				Value:    modResult.EmailRecovery,
				Type:     engine.SeedEmail,
				ParentID: fmt.Sprintf("account:%s:%s", mod.Name(), seed.Value),
				Depth:    seed.Depth + 1,
			})
		}
	}

	return result
}
