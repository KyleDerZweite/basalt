// SPDX-License-Identifier: AGPL-3.0-or-later
//
// Importers for upstream OSINT site databases.
// Converts Maigret, Sherlock, and WhatsMyName JSON formats into
// Basalt's unified SiteDefinition YAML format.

package sitedb

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// UpstreamFormat identifies the source database format.
type UpstreamFormat string

const (
	FormatMaigret  UpstreamFormat = "maigret"
	FormatSherlock UpstreamFormat = "sherlock"
	FormatWMN      UpstreamFormat = "wmn"
)

// ImportUpstream reads an upstream JSON file and returns Basalt SiteDefinitions.
func ImportUpstream(path string, format UpstreamFormat) ([]SiteDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	switch format {
	case FormatMaigret:
		return importMaigret(data)
	case FormatSherlock:
		return importSherlock(data)
	case FormatWMN:
		return importWMN(data)
	default:
		return nil, fmt.Errorf("unknown format: %s", format)
	}
}

// normalizePlaceholder replaces a format-specific placeholder with {username}.
func normalizePlaceholder(template, placeholder string) string {
	return strings.ReplaceAll(template, placeholder, "{username}")
}

// extractMainURL extracts the scheme+host from a URL template.
func extractMainURL(urlTemplate string) string {
	// Strip placeholders before parsing.
	cleaned := strings.ReplaceAll(urlTemplate, "{username}", "x")
	cleaned = strings.ReplaceAll(cleaned, "{seed}", "x")

	parsed, err := url.Parse(cleaned)
	if err != nil {
		return urlTemplate
	}
	return parsed.Scheme + "://" + parsed.Host
}

// coalesce returns the first non-empty string.
func coalesce(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// guessCategory maps tags to a normalized category.
func guessCategory(tags []string) string {
	for _, t := range tags {
		t = strings.ToLower(t)
		switch {
		case t == "social" || t == "dating":
			return "social"
		case t == "coding" || t == "dev" || t == "git":
			return "coding"
		case t == "gaming" || t == "game":
			return "gaming"
		case t == "music" || t == "art":
			return "creative"
		case t == "finance" || t == "crypto":
			return "finance"
		case t == "news" || t == "blog":
			return "media"
		case t == "shopping" || t == "marketplace":
			return "shopping"
		}
	}
	return "other"
}

// normalizeCat maps a WMN category string to a normalized category.
func normalizeCat(cat string) string {
	c := strings.ToLower(strings.TrimSpace(cat))
	switch {
	case strings.Contains(c, "social"):
		return "social"
	case strings.Contains(c, "gaming"):
		return "gaming"
	case strings.Contains(c, "coding"), strings.Contains(c, "tech"):
		return "coding"
	case strings.Contains(c, "music"), strings.Contains(c, "art"):
		return "creative"
	case strings.Contains(c, "finance"), strings.Contains(c, "crypto"):
		return "finance"
	case strings.Contains(c, "news"), strings.Contains(c, "blog"):
		return "media"
	case strings.Contains(c, "shopping"), strings.Contains(c, "market"):
		return "shopping"
	case strings.Contains(c, "adult"), strings.Contains(c, "porn"):
		return "adult"
	default:
		return "other"
	}
}
