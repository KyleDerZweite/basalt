// SPDX-License-Identifier: AGPL-3.0-or-later

package sitedb

// SiteFile represents a YAML file containing one or more site definitions.
type SiteFile struct {
	Sites []SiteDefinition `yaml:"sites"`
}

// SiteDefinition describes how to check a single site for the presence of
// a username, email, or other seed type.
type SiteDefinition struct {
	// Name is a human-readable identifier (e.g., "GitHub").
	Name string `yaml:"name"`

	// URLTemplate is the profile URL with a {username} placeholder.
	URLTemplate string `yaml:"url_template"`

	// URLMain is the site's main URL (e.g., "https://github.com").
	URLMain string `yaml:"url_main"`

	// Category classifies the site (e.g., "coding", "social", "gaming").
	Category string `yaml:"category"`

	// Tags are freeform labels for filtering (e.g., ["developer", "git"]).
	Tags []string `yaml:"tags,omitempty"`

	// SeedTypes declares which seed types this site supports.
	// e.g., ["username"], ["email"], or ["username", "email"].
	SeedTypes []string `yaml:"seed_types"`

	// Check defines how to verify account existence.
	Check Check `yaml:"check"`

	// UsernameRegex is an optional regex to reject invalid usernames before
	// making any HTTP requests.
	UsernameRegex string `yaml:"username_regex,omitempty"`

	// TestAccounts provides known-existent and known-nonexistent accounts
	// for self-validation and dual-request control checks.
	TestAccounts TestAccounts `yaml:"test_accounts"`

	// RateLimit overrides the global rate limit for this site.
	RateLimit *RateLimit `yaml:"rate_limit,omitempty"`

	// ProbeURL is an alternative URL (e.g., an API endpoint) that may be
	// more reliable than the HTML profile page.
	ProbeURL string `yaml:"probe_url,omitempty"`

	// Extract defines rules for scraping metadata from discovered profiles.
	// Rules with Pivot=true feed extracted values back as new seeds.
	Extract []ExtractRule `yaml:"extract,omitempty"`

	// Disabled skips this site during scans.
	Disabled bool `yaml:"disabled,omitempty"`

	// Source tracks where this definition came from (e.g., "maigret", "custom").
	Source string `yaml:"source,omitempty"`
}

// Check defines the HTTP request and response matching configuration.
type Check struct {
	// Method is the HTTP method to use (default: "GET").
	Method string `yaml:"method,omitempty"`

	// Headers are additional HTTP headers to send.
	Headers map[string]string `yaml:"headers,omitempty"`

	// ExpectedStatus is the HTTP status code indicating account exists (e.g., 200).
	ExpectedStatus int `yaml:"expected_status"`

	// PresenceStrings must ALL appear in the response body for a positive match.
	PresenceStrings []string `yaml:"presence_strings,omitempty"`

	// AbsenceStrings must ALL be absent from the response body for a positive match.
	AbsenceStrings []string `yaml:"absence_strings,omitempty"`

	// BodyTemplate is used for POST-based checks (e.g., signup/reset API).
	// The placeholder {seed} is replaced with the actual seed value.
	BodyTemplate string `yaml:"body_template,omitempty"`

	// ContentType for POST requests (e.g., "application/json").
	ContentType string `yaml:"content_type,omitempty"`
}

// TestAccounts provides known accounts for validation and control checks.
type TestAccounts struct {
	// Claimed is a username known to exist on the site.
	Claimed string `yaml:"claimed"`

	// Unclaimed is a username known NOT to exist (used for dual-request verification).
	Unclaimed string `yaml:"unclaimed"`
}

// RateLimit configures per-site rate limiting.
type RateLimit struct {
	// RequestsPerSecond is the sustained rate.
	RequestsPerSecond float64 `yaml:"requests_per_second"`

	// Burst is the maximum burst size.
	Burst int `yaml:"burst"`
}

// ExtractRule defines how to scrape metadata from a profile page.
type ExtractRule struct {
	// Field is the name of the extracted datum (e.g., "email", "display_name").
	Field string `yaml:"field"`

	// Selector is a CSS selector for HTML extraction (via goquery).
	Selector string `yaml:"selector,omitempty"`

	// Regex is a regular expression with a capture group for extraction.
	Regex string `yaml:"regex,omitempty"`

	// Attribute is the HTML attribute to extract (e.g., "href").
	// Only used with Selector.
	Attribute string `yaml:"attribute,omitempty"`

	// Pivot indicates that extracted values should be fed back as new seeds.
	Pivot bool `yaml:"pivot,omitempty"`
}
