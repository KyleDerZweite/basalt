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

// Resolve detects the type of a raw input string and returns a Seed.
func Resolve(input string) engine.Seed {
	input = strings.TrimSpace(input)

	switch {
	case emailRegex.MatchString(input):
		return engine.Seed{Value: input, Type: engine.SeedEmail}
	case phoneRegex.MatchString(strings.ReplaceAll(input, " ", "")):
		return engine.Seed{Value: strings.ReplaceAll(input, " ", ""), Type: engine.SeedPhone}
	case strings.Contains(input, ".") && !strings.Contains(input, " ") && domainRegex.MatchString(input):
		return engine.Seed{Value: input, Type: engine.SeedDomain}
	default:
		return engine.Seed{Value: input, Type: engine.SeedUsername}
	}
}
