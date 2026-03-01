// SPDX-License-Identifier: AGPL-3.0-or-later
//
// UsernameEngine checks whether a username exists on various platforms
// by making HTTP requests and analyzing responses with multi-signal
// confidence scoring and dual-request verification.
//
// GDPR Note: This engine only queries publicly accessible profile URLs.
// No authentication bypass, no private data access, no API abuse.
// Results should only be used with the data subject's explicit consent.

package username

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/kyle/basalt/internal/engine"
	"github.com/kyle/basalt/internal/httpclient"
	"github.com/kyle/basalt/internal/pivot"
	"github.com/kyle/basalt/internal/sitedb"
)

// builtinSites are fallback sites used when no YAML files are loaded.
var builtinSites = []sitedb.SiteDefinition{
	{
		Name: "GitHub", URLTemplate: "https://github.com/{username}", URLMain: "https://github.com",
		Category: "coding", SeedTypes: []string{"username"},
		Check: sitedb.Check{Method: "GET", ExpectedStatus: 200,
			PresenceStrings: []string{"js-profile-editable-area"},
			AbsenceStrings:  []string{"This is not the web page you are looking for"}},
		TestAccounts: sitedb.TestAccounts{Claimed: "torvalds", Unclaimed: "zzz-basalt-nonexistent-user-zzz"},
	},
	{
		Name: "GitLab", URLTemplate: "https://gitlab.com/{username}", URLMain: "https://gitlab.com",
		Category: "coding", SeedTypes: []string{"username"},
		Check: sitedb.Check{Method: "GET", ExpectedStatus: 200,
			PresenceStrings: []string{"user-profile"},
			AbsenceStrings:  []string{"Page Not Found"}},
		TestAccounts: sitedb.TestAccounts{Claimed: "torvalds", Unclaimed: "zzz-basalt-nonexistent-user-zzz"},
	},
	{
		Name: "Reddit", URLTemplate: "https://www.reddit.com/user/{username}", URLMain: "https://www.reddit.com",
		Category: "social", SeedTypes: []string{"username"},
		Check: sitedb.Check{Method: "GET", ExpectedStatus: 200,
			AbsenceStrings: []string{"Sorry, nobody on Reddit goes by that name", "page not found"}},
		TestAccounts: sitedb.TestAccounts{Claimed: "spez", Unclaimed: "zzz_basalt_nonexistent_user_zzz"},
	},
	{
		Name: "HackerNews", URLTemplate: "https://news.ycombinator.com/user?id={username}", URLMain: "https://news.ycombinator.com",
		Category: "tech", SeedTypes: []string{"username"},
		Check: sitedb.Check{Method: "GET", ExpectedStatus: 200,
			PresenceStrings: []string{"created:"},
			AbsenceStrings:  []string{"No such user."}},
		TestAccounts: sitedb.TestAccounts{Claimed: "pg", Unclaimed: "zzz_basalt_nonexistent_user_zzz"},
	},
	{
		Name: "Keybase", URLTemplate: "https://keybase.io/{username}", URLMain: "https://keybase.io",
		Category: "security", SeedTypes: []string{"username"},
		Check: sitedb.Check{Method: "GET", ExpectedStatus: 200,
			AbsenceStrings: []string{"not found"}},
		TestAccounts: sitedb.TestAccounts{Claimed: "chris", Unclaimed: "zzz_basalt_nonexistent_user_zzz"},
	},
}

// Engine implements the engine.Engine interface for username lookups.
type Engine struct {
	client       *httpclient.Client
	rateLimiter  *httpclient.DomainRateLimiter
	controlCache *engine.ControlCache
	threshold    float64
	sites        []sitedb.SiteDefinition
	concurrency  int
}

// Option configures the Engine.
type Option func(*Engine)

// WithSites sets the site definitions to check.
func WithSites(sites []sitedb.SiteDefinition) Option {
	return func(e *Engine) { e.sites = sites }
}

// WithConcurrency sets the max concurrent site checks within this engine.
func WithConcurrency(n int) Option {
	return func(e *Engine) { e.concurrency = n }
}

// WithRateLimiter sets the per-domain rate limiter.
func WithRateLimiter(rl *httpclient.DomainRateLimiter) Option {
	return func(e *Engine) { e.rateLimiter = rl }
}

// WithControlCache sets the control response cache.
func WithControlCache(cc *engine.ControlCache) Option {
	return func(e *Engine) { e.controlCache = cc }
}

// New creates a new UsernameEngine.
func New(client *httpclient.Client, threshold float64, opts ...Option) *Engine {
	e := &Engine{
		client:      client,
		threshold:   threshold,
		sites:       builtinSites,
		concurrency: 20,
	}
	for _, opt := range opts {
		opt(e)
	}

	if e.rateLimiter != nil {
		for _, site := range e.sites {
			if site.RateLimit != nil {
				domain := httpclient.ExtractDomain(site.URLMain)
				e.rateLimiter.SetDomainLimit(domain, site.RateLimit.RequestsPerSecond, site.RateLimit.Burst)
			}
		}
	}

	return e
}

func (e *Engine) Name() string                 { return "username-checker" }
func (e *Engine) SeedTypes() []engine.SeedType  { return []engine.SeedType{engine.SeedUsername} }

// Check runs the username against all sites concurrently and streams results.
func (e *Engine) Check(ctx context.Context, seed engine.Seed, results chan<- engine.Result) {
	applicable := e.filterSites(seed.Value)
	slog.Info("checking sites", "count", len(applicable), "username", seed.Value)

	var wg sync.WaitGroup
	sem := make(chan struct{}, e.concurrency)

	for _, site := range applicable {
		wg.Add(1)
		sem <- struct{}{}

		go func(s sitedb.SiteDefinition) {
			defer wg.Done()
			defer func() { <-sem }()

			select {
			case <-ctx.Done():
				return
			default:
			}

			result := e.checkSite(ctx, seed, s)

			select {
			case results <- result:
			case <-ctx.Done():
			}
		}(site)
	}

	wg.Wait()
	close(results)
}

// filterSites returns sites applicable to this username.
func (e *Engine) filterSites(username string) []sitedb.SiteDefinition {
	var applicable []sitedb.SiteDefinition
	for _, site := range e.sites {
		if site.Disabled || !containsSeedType(site.SeedTypes, "username") {
			continue
		}
		if site.UsernameRegex != "" {
			re, err := regexp.Compile(site.UsernameRegex)
			if err == nil && !re.MatchString(username) {
				continue
			}
		}
		applicable = append(applicable, site)
	}
	return applicable
}

// waitRateLimit waits for rate limit clearance if a limiter is configured.
func (e *Engine) waitRateLimit(ctx context.Context, url string) error {
	if e.rateLimiter != nil {
		return e.rateLimiter.Wait(ctx, url)
	}
	return nil
}

// checkSite performs the dual-request verification for a single site.
func (e *Engine) checkSite(ctx context.Context, seed engine.Seed, site sitedb.SiteDefinition) engine.Result {
	targetURL := strings.ReplaceAll(site.URLTemplate, "{username}", seed.Value)

	result := engine.Result{
		EngineName: e.Name(),
		Seed:       seed,
		SiteName:   site.Name,
		SiteURL:    site.URLMain,
		Category:   site.Category,
		ProfileURL: targetURL,
	}

	if err := e.waitRateLimit(ctx, targetURL); err != nil {
		result.Err = fmt.Errorf("rate limit wait: %w", err)
		return result
	}

	// 1. Target request.
	targetResp, err := e.doTargetRequest(ctx, targetURL, seed.Value, site.Check)
	if err != nil {
		result.Err = fmt.Errorf("target request for %s: %w", site.Name, err)
		slog.Debug("site check failed", "site", site.Name, "error", err)
		return result
	}

	result.HTTPStatus = targetResp.StatusCode
	result.ResponseTime = targetResp.ResponseTime

	// 2. Control request.
	controlBody := e.fetchControlBody(ctx, site)

	// 3. Compute confidence.
	confidence, signals := engine.ComputeConfidence(engine.CheckContext{
		PresenceStrings: site.Check.PresenceStrings,
		AbsenceStrings:  site.Check.AbsenceStrings,
		ExpectedStatus:  site.Check.ExpectedStatus,
		TargetStatus:    targetResp.StatusCode,
		TargetBody:      targetResp.Body,
		TargetFinalURL:  targetResp.FinalURL,
		TargetURL:       targetURL,
		ControlBody:     controlBody,
	})
	result.Confidence = confidence
	result.Signals = signals
	result.Exists = confidence >= e.threshold

	// 4. Extract metadata and pivot seeds.
	if result.Exists && len(site.Extract) > 0 {
		e.extractMetadata(ctx, &result, seed, site, targetResp.Body)
	}

	return result
}

// doTargetRequest builds and executes the target HTTP request.
func (e *Engine) doTargetRequest(ctx context.Context, targetURL, seedValue string, check sitedb.Check) (*httpclient.Response, error) {
	method := check.Method
	if method == "" {
		method = http.MethodGet
	}

	var body io.Reader
	headers := check.Headers

	if check.BodyTemplate != "" {
		bodyStr := strings.ReplaceAll(check.BodyTemplate, "{username}", seedValue)
		bodyStr = strings.ReplaceAll(bodyStr, "{seed}", seedValue)
		body = strings.NewReader(bodyStr)

		if check.ContentType != "" {
			headers = mergeHeaders(headers, "Content-Type", check.ContentType)
		}
	}

	return e.client.DoRequest(ctx, method, targetURL, body, headers)
}

// mergeHeaders returns a copy of base headers with an additional key-value pair.
func mergeHeaders(base map[string]string, key, value string) map[string]string {
	merged := make(map[string]string, len(base)+1)
	for k, v := range base {
		merged[k] = v
	}
	merged[key] = value
	return merged
}

// fetchControlBody fetches the control response body for a site (cached when possible).
func (e *Engine) fetchControlBody(ctx context.Context, site sitedb.SiteDefinition) string {
	unclaimed := site.TestAccounts.Unclaimed
	if unclaimed == "" {
		unclaimed = "zzz_basalt_nonexistent_user_zzz"
	}
	controlURL := strings.ReplaceAll(site.URLTemplate, "{username}", unclaimed)

	if e.controlCache != nil {
		body, _, err := e.controlCache.Get(ctx, site.Name, controlURL, site.Check.Headers)
		if err != nil {
			slog.Debug("control cache fetch failed, proceeding without", "site", site.Name, "error", err)
			return ""
		}
		return body
	}

	// Fallback: direct request.
	if err := e.waitRateLimit(ctx, controlURL); err != nil {
		slog.Debug("rate limit wait for control failed", "site", site.Name, "error", err)
		return ""
	}
	controlResp, err := e.client.Do(ctx, controlURL, site.Check.Headers)
	if err != nil {
		slog.Debug("control request failed, proceeding without", "site", site.Name, "error", err)
		return ""
	}
	return controlResp.Body
}

// extractMetadata fetches probe data (if available) and extracts metadata + pivot seeds.
func (e *Engine) extractMetadata(ctx context.Context, result *engine.Result, seed engine.Seed, site sitedb.SiteDefinition, targetBody string) {
	accountNodeID := fmt.Sprintf("account:%s:%s", site.Name, seed.Value)
	extractBody := targetBody

	if site.ProbeURL != "" {
		if probeBody, ok := e.fetchProbeBody(ctx, site, seed.Value); ok {
			extractBody = probeBody
		}
	}

	extraction := pivot.Extract(extractBody, site.Extract, accountNodeID, seed.Depth)
	result.Metadata = extraction.Metadata
	result.DiscoveredSeeds = extraction.DiscoveredSeeds
}

// fetchProbeBody fetches the probe URL, retrying with lowercase if the first attempt fails.
func (e *Engine) fetchProbeBody(ctx context.Context, site sitedb.SiteDefinition, seedValue string) (string, bool) {
	probeURL := strings.ReplaceAll(site.ProbeURL, "{username}", seedValue)
	_ = e.waitRateLimit(ctx, probeURL)

	probeResp, err := e.client.Do(ctx, probeURL, site.Check.Headers)
	if err != nil || probeResp.StatusCode != 200 {
		// Some APIs are case-sensitive — retry with lowercase.
		probeLower := strings.ReplaceAll(site.ProbeURL, "{username}", strings.ToLower(seedValue))
		if probeLower != probeURL {
			_ = e.waitRateLimit(ctx, probeLower)
			probeResp, err = e.client.Do(ctx, probeLower, site.Check.Headers)
		}
	}

	if err == nil && probeResp.StatusCode == 200 {
		return probeResp.Body, true
	}
	slog.Debug("probe request failed, using HTML body", "site", site.Name, "error", err)
	return "", false
}

func containsSeedType(types []string, target string) bool {
	for _, t := range types {
		if t == target {
			return true
		}
	}
	return false
}
