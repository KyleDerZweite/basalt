// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"context"
	"time"
)

// SeedType represents the category of input data.
type SeedType string

const (
	SeedUsername SeedType = "username"
	SeedEmail    SeedType = "email"
	SeedPhone    SeedType = "phone"
	SeedDomain   SeedType = "domain"
)

// Seed is the input to an engine — a single data point to investigate.
type Seed struct {
	// Value is the raw input (e.g., "johndoe", "john@example.com").
	Value string

	// Type is the detected or declared seed type.
	Type SeedType

	// ParentID is the graph node ID of the discovery that produced this seed.
	// Empty for initial user-provided seeds.
	ParentID string

	// Depth tracks how many pivot hops away from the initial seed this is.
	Depth int
}

// Engine defines the contract for all discovery engines.
// Each engine handles one or more seed types and streams results as they arrive.
type Engine interface {
	// Name returns a human-readable engine identifier (e.g., "username-checker").
	Name() string

	// SeedTypes returns which seed types this engine can process.
	SeedTypes() []SeedType

	// Check performs discovery for the given seed and streams results.
	// The engine MUST close the results channel when done.
	// The engine MUST respect context cancellation for graceful shutdown.
	Check(ctx context.Context, seed Seed, results chan<- Result)
}

// Result represents a single site check outcome from an engine.
type Result struct {
	// EngineName identifies which engine produced this result.
	EngineName string

	// Seed is the input that was checked.
	Seed Seed

	// SiteName is the human-readable name of the site (e.g., "GitHub").
	SiteName string

	// SiteURL is the main URL of the site (e.g., "https://github.com").
	SiteURL string

	// Category classifies the site (e.g., "coding", "social", "gaming").
	Category string

	// ProfileURL is the resolved URL with the seed substituted.
	ProfileURL string

	// Confidence is the multi-signal score in [0.0, 1.0].
	Confidence float64

	// Exists indicates whether this result is considered a positive match
	// (Confidence >= threshold).
	Exists bool

	// Signals breaks down how the confidence was computed.
	Signals SignalScores

	// Metadata contains structured data extracted from the profile page.
	Metadata map[string]string

	// DiscoveredSeeds contains new seeds found on this profile (for auto-pivoting).
	DiscoveredSeeds []Seed

	// Err is set if the check failed (timeout, rate limit, DNS, etc.).
	// When Err is non-nil, Confidence is 0 and Exists is false.
	Err error

	// HTTPStatus is the HTTP response status code.
	HTTPStatus int

	// ResponseTime is how long the HTTP request took.
	ResponseTime time.Duration
}

// SignalScores breaks down the individual signals that contribute to confidence.
type SignalScores struct {
	// HTTPStatus: 1.0 if response status matches expected, 0.0 otherwise.
	HTTPStatus float64 `json:"http_status"`

	// BodyPresenceMatch: fraction of presence strings found in body [0.0, 1.0].
	BodyPresenceMatch float64 `json:"body_presence_match"`

	// BodyAbsenceCheck: fraction of absence strings that are absent from body [0.0, 1.0].
	BodyAbsenceCheck float64 `json:"body_absence_check"`

	// ContentDifferentiation: how different the response is from a known-nonexistent
	// user's response [0.0, 1.0]. 1.0 = completely different (likely real profile),
	// 0.0 = identical (soft 404). This is THE key signal for eliminating false positives.
	ContentDifferentiation float64 `json:"content_differentiation"`

	// NoRedirect: 1.0 if the URL didn't redirect, 0.0 if it did.
	NoRedirect float64 `json:"no_redirect"`
}
