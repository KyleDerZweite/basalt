// SPDX-License-Identifier: AGPL-3.0-or-later
//
// HTML sanitization for body comparison.
// Strips dynamic content (CSRF tokens, nonces, timestamps, scripts, etc.)
// that would cause false differences between target and control responses.

package engine

import (
	"regexp"
	"strings"
)

// Precompiled regexes for dynamic content removal.
var (
	// HTML elements that contain dynamic/non-deterministic content.
	reScript   = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	reStyle    = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	reComments = regexp.MustCompile(`<!--[\s\S]*?-->`)
	reSVG      = regexp.MustCompile(`(?is)<svg[^>]*>.*?</svg>`)

	// Attributes that commonly contain dynamic values.
	reCSRF      = regexp.MustCompile(`(?i)(csrf|authenticity.token|_token|__RequestVerificationToken)[^"']*["'][^"']*["']`)
	reNonce     = regexp.MustCompile(`(?i)nonce=["'][^"']*["']`)
	reDataAttrs = regexp.MustCompile(`(?i)\sdata-[a-z0-9-]+=["'][^"']*["']`)

	// Dynamic values embedded in HTML.
	reTimestamps = regexp.MustCompile(`\b\d{10,13}\b`)                                      // Unix timestamps (seconds or ms)
	reISO8601    = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[^\s"'<]*`)      // ISO 8601 datetimes
	reUUIDs      = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	reHexTokens  = regexp.MustCompile(`[0-9a-f]{32,}`)                                      // Long hex strings (session IDs, hashes)

	// Collapse whitespace for stable comparison.
	reWhitespace = regexp.MustCompile(`\s+`)
)

// SanitizeBody strips dynamic HTML content for stable comparison.
// The goal is to remove anything that changes between requests to the same
// page so that similarity comparison reflects actual content differences.
func SanitizeBody(body string) string {
	// Remove entire script/style/svg blocks.
	s := reScript.ReplaceAllString(body, "")
	s = reStyle.ReplaceAllString(s, "")
	s = reComments.ReplaceAllString(s, "")
	s = reSVG.ReplaceAllString(s, "")

	// Remove dynamic attributes.
	s = reCSRF.ReplaceAllString(s, "")
	s = reNonce.ReplaceAllString(s, "")
	s = reDataAttrs.ReplaceAllString(s, "")

	// Remove dynamic values.
	s = reTimestamps.ReplaceAllString(s, "")
	s = reISO8601.ReplaceAllString(s, "")
	s = reUUIDs.ReplaceAllString(s, "")
	s = reHexTokens.ReplaceAllString(s, "")

	// Normalize whitespace.
	s = reWhitespace.ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)

	return s
}
