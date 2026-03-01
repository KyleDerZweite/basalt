// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"strings"
)

// Signal weights - these sum to 1.0.
const (
	WeightHTTPStatus             = 0.15
	WeightBodyPresenceMatch      = 0.25
	WeightBodyAbsenceCheck       = 0.20
	WeightContentDifferentiation = 0.30
	WeightNoRedirect             = 0.10
)

// DefaultConfidenceThreshold is the minimum confidence to consider a match.
const DefaultConfidenceThreshold = 0.50

// CheckContext holds all data needed to compute a confidence score.
type CheckContext struct {
	// Site definition fields
	PresenceStrings []string
	AbsenceStrings  []string
	ExpectedStatus  int

	// Target response (the username being investigated)
	TargetStatus   int
	TargetBody     string
	TargetFinalURL string
	TargetURL      string // original URL before any redirects

	// Control response (known-nonexistent username for comparison)
	ControlStatus int
	ControlBody   string
}

// ComputeConfidence calculates a weighted confidence score from multiple signals.
// The content differentiation signal is the key innovation - it compares the target
// response against a control (nonexistent user) response to detect soft-404 pages.
func ComputeConfidence(ctx CheckContext) (float64, SignalScores) {
	var scores SignalScores

	// Signal 1: HTTP Status - does the status match what we expect for "exists"?
	if ctx.TargetStatus == ctx.ExpectedStatus {
		scores.HTTPStatus = 1.0
	}

	// Signal 2: Body Presence Match - do expected strings appear in the body?
	if len(ctx.PresenceStrings) > 0 {
		matched := 0
		for _, s := range ctx.PresenceStrings {
			if strings.Contains(ctx.TargetBody, s) {
				matched++
			}
		}
		scores.BodyPresenceMatch = float64(matched) / float64(len(ctx.PresenceStrings))
	} else {
		// No presence strings defined: use HTTP status as proxy.
		scores.BodyPresenceMatch = scores.HTTPStatus
	}

	// Signal 3: Body Absence Check - are error/404 strings absent from body?
	if len(ctx.AbsenceStrings) > 0 {
		absent := 0
		for _, s := range ctx.AbsenceStrings {
			if !strings.Contains(ctx.TargetBody, s) {
				absent++
			}
		}
		scores.BodyAbsenceCheck = float64(absent) / float64(len(ctx.AbsenceStrings))
	} else {
		// No absence strings defined: no red flags detected.
		scores.BodyAbsenceCheck = 1.0
	}

	// Signal 4: Content Differentiation - THE key signal.
	// Compare target body against control body to detect soft-404 pages.
	// If both responses are identical or near-identical, this is a soft 404.
	if ctx.ControlBody != "" {
		similarity := ComputeSimilarity(ctx.TargetBody, ctx.ControlBody)
		// similarity 1.0 = identical = soft 404 = differentiation 0.0
		// similarity 0.0 = completely different = real profile = differentiation 1.0
		scores.ContentDifferentiation = 1.0 - similarity
	} else {
		// No control response available: fall back to HTTP status signal.
		scores.ContentDifferentiation = scores.HTTPStatus
	}

	// Signal 5: No Redirect - did the URL stay the same?
	if ctx.TargetFinalURL == "" || ctx.TargetFinalURL == ctx.TargetURL {
		scores.NoRedirect = 1.0
	}

	// Clamp all signals to [0.0, 1.0].
	scores.HTTPStatus = clamp(scores.HTTPStatus)
	scores.BodyPresenceMatch = clamp(scores.BodyPresenceMatch)
	scores.BodyAbsenceCheck = clamp(scores.BodyAbsenceCheck)
	scores.ContentDifferentiation = clamp(scores.ContentDifferentiation)
	scores.NoRedirect = clamp(scores.NoRedirect)

	// Soft-404 penalty: when content differentiation is very low and we have
	// a control body to compare against, the page is almost certainly a soft 404.
	// Cap the maximum possible confidence to prevent false positives from sites
	// that return 200 with absence-string-free error pages.
	softPenalty := 1.0
	if ctx.ControlBody != "" && scores.ContentDifferentiation < 0.10 {
		// Scale penalty from 0.5 (at diff=0.10) down to 0.3 (at diff=0.0).
		softPenalty = 0.3 + scores.ContentDifferentiation*2.0
	}

	// Status mismatch penalty: when the HTTP status is a clear error (4xx/5xx)
	// but the expected status was 2xx, the account almost certainly doesn't exist.
	// High content differentiation alone shouldn't override a 404.
	statusPenalty := 1.0
	if scores.HTTPStatus == 0 && ctx.ExpectedStatus >= 200 && ctx.ExpectedStatus < 300 {
		if ctx.TargetStatus >= 400 {
			// Hard error: 404, 403, 410, 500, etc. - severely penalize.
			statusPenalty = 0.4
		}
	}

	// Weighted sum
	confidence := scores.HTTPStatus*WeightHTTPStatus +
		scores.BodyPresenceMatch*WeightBodyPresenceMatch +
		scores.BodyAbsenceCheck*WeightBodyAbsenceCheck +
		scores.ContentDifferentiation*WeightContentDifferentiation +
		scores.NoRedirect*WeightNoRedirect

	confidence *= softPenalty * statusPenalty

	return clamp(confidence), scores
}

func clamp(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// ComputeSimilarity returns a value in [0.0, 1.0] indicating how similar
// two response bodies are. 1.0 = identical, 0.0 = completely different.
//
// Bodies are sanitized first (strip scripts, CSRF tokens, timestamps, etc.)
// then compared using shingled Jaccard similarity on 5-gram character shingles.
// A length ratio is blended in as a fast-path differentiator.
func ComputeSimilarity(a, b string) float64 {
	if a == b {
		return 1.0
	}
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}

	// Sanitize to remove dynamic content before comparison.
	sa := SanitizeBody(a)
	sb := SanitizeBody(b)

	if sa == sb {
		return 1.0
	}
	if len(sa) == 0 || len(sb) == 0 {
		return 0.0
	}

	// Length ratio: if pages are very different lengths, they're likely different.
	lenA := float64(len(sa))
	lenB := float64(len(sb))
	lengthRatio := min(lenA, lenB) / max(lenA, lenB)

	// Shingled Jaccard similarity on 5-char shingles.
	const shingleSize = 5
	if len(sa) < shingleSize || len(sb) < shingleSize {
		return lengthRatio
	}

	shinglesA := shingleSet(sa, shingleSize)
	shinglesB := shingleSet(sb, shingleSize)

	// Jaccard = |A ∩ B| / |A ∪ B|
	intersection := 0
	for s := range shinglesA {
		if _, ok := shinglesB[s]; ok {
			intersection++
		}
	}
	union := len(shinglesA) + len(shinglesB) - intersection
	if union == 0 {
		return 1.0
	}

	jaccard := float64(intersection) / float64(union)

	// Blend length ratio (20%) and Jaccard (80%).
	return 0.2*lengthRatio + 0.8*jaccard
}

// shingleSet extracts unique character n-grams from a string.
func shingleSet(s string, n int) map[string]struct{} {
	set := make(map[string]struct{}, len(s)-n+1)
	for i := 0; i <= len(s)-n; i++ {
		set[s[i:i+n]] = struct{}{}
	}
	return set
}
