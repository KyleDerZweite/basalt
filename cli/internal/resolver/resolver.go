// SPDX-License-Identifier: AGPL-3.0-or-later

package resolver

import (
	"regexp"
	"strings"

	"github.com/kyle/basalt/internal/engine"
)

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	phoneRegex = regexp.MustCompile(`^\+?[0-9]{7,15}$`)
	// A domain must have at least one dot and no spaces, and not look like an email.
	domainRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?)+$`)
)

// looksLikeDomain performs a heuristic check: the TLD (last segment) must be
// 2-12 lowercase ASCII letters - this filters out usernames like "Sir.Cookie".
func looksLikeDomain(s string) bool {
	idx := strings.LastIndex(s, ".")
	if idx < 0 || idx == len(s)-1 {
		return false
	}
	tld := s[idx+1:]
	if len(tld) < 2 || len(tld) > 12 {
		return false
	}
	for _, c := range tld {
		if c < 'a' || c > 'z' {
			return false
		}
	}
	return true
}

// Resolve detects the type of a raw input string and returns a Seed.
func Resolve(input string) engine.Seed {
	input = strings.TrimSpace(input)

	switch {
	case emailRegex.MatchString(input):
		return engine.Seed{Value: input, Type: engine.SeedEmail}
	case phoneRegex.MatchString(strings.ReplaceAll(input, " ", "")):
		return engine.Seed{Value: strings.ReplaceAll(input, " ", ""), Type: engine.SeedPhone}
	case strings.Contains(input, ".") && !strings.Contains(input, " ") && domainRegex.MatchString(input) && looksLikeDomain(input):
		return engine.Seed{Value: input, Type: engine.SeedDomain}
	default:
		return engine.Seed{Value: input, Type: engine.SeedUsername}
	}
}
