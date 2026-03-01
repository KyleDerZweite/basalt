// SPDX-License-Identifier: AGPL-3.0-or-later
//
// Metadata and seed extraction from HTML profile pages.
// Supports CSS selectors (goquery) and regex capture groups.

package pivot

import (
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/kyle/basalt/internal/engine"
	"github.com/kyle/basalt/internal/resolver"
	"github.com/kyle/basalt/internal/sitedb"
)

// ExtractionResult holds metadata and discovered seeds from a profile page.
type ExtractionResult struct {
	Metadata        map[string]string
	DiscoveredSeeds []engine.Seed
}

// Extract runs the site's extract rules against an HTML body.
// It returns extracted metadata and any discovered pivot seeds.
func Extract(body string, rules []sitedb.ExtractRule, parentNodeID string, depth int) ExtractionResult {
	result := ExtractionResult{
		Metadata: make(map[string]string),
	}

	if len(rules) == 0 || body == "" {
		return result
	}

	// Parse HTML once for all CSS selector rules.
	doc, docErr := goquery.NewDocumentFromReader(strings.NewReader(body))

	for _, rule := range rules {
		var value string

		// Try CSS selector first, then regex.
		if rule.Selector != "" && docErr == nil {
			value = extractBySelector(doc, rule)
		}
		if value == "" && rule.Regex != "" {
			value = extractByRegex(body, rule.Regex)
		}

		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		result.Metadata[rule.Field] = value

		// If this rule is a pivot rule, create a new seed from the extracted value.
		if rule.Pivot {
			seed := resolver.Resolve(value)
			seed.ParentID = parentNodeID
			seed.Depth = depth + 1
			result.DiscoveredSeeds = append(result.DiscoveredSeeds, seed)
		}
	}

	return result
}

// extractBySelector uses a CSS selector to extract text or an attribute.
func extractBySelector(doc *goquery.Document, rule sitedb.ExtractRule) string {
	sel := doc.Find(rule.Selector).First()
	if sel.Length() == 0 {
		return ""
	}

	if rule.Attribute != "" {
		val, exists := sel.Attr(rule.Attribute)
		if !exists {
			return ""
		}
		return val
	}

	return sel.Text()
}

// extractByRegex extracts the first capture group match.
func extractByRegex(body string, pattern string) string {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return ""
	}

	matches := re.FindStringSubmatch(body)
	if len(matches) < 2 {
		return ""
	}

	return matches[1]
}
