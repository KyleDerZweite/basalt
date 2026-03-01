// SPDX-License-Identifier: AGPL-3.0-or-later

package sitedb

import (
	"fmt"
	"regexp"
	"strings"
)

// Validate checks a SiteDefinition for required fields and correctness.
// Returns a list of validation errors (empty if valid).
func Validate(s *SiteDefinition) []string {
	var errs []string

	if s.Name == "" {
		errs = append(errs, "name is required")
	}

	if s.URLTemplate == "" {
		errs = append(errs, "url_template is required")
	} else if !strings.Contains(s.URLTemplate, "{username}") &&
		!strings.Contains(s.URLTemplate, "{seed}") {
		errs = append(errs, "url_template must contain {username} or {seed} placeholder")
	}

	if len(s.SeedTypes) == 0 {
		errs = append(errs, "seed_types is required (e.g., [\"username\"])")
	}

	if s.Check.ExpectedStatus == 0 {
		errs = append(errs, "check.expected_status is required (e.g., 200)")
	}

	if s.UsernameRegex != "" {
		if _, err := regexp.Compile(s.UsernameRegex); err != nil {
			errs = append(errs, fmt.Sprintf("username_regex is invalid: %v", err))
		}
	}

	if s.URLTemplate != "" && !strings.HasPrefix(s.URLTemplate, "http://") &&
		!strings.HasPrefix(s.URLTemplate, "https://") {
		errs = append(errs, "url_template must start with http:// or https://")
	}

	for i, rule := range s.Extract {
		if rule.Field == "" {
			errs = append(errs, fmt.Sprintf("extract[%d].field is required", i))
		}
		if rule.Selector == "" && rule.Regex == "" {
			errs = append(errs, fmt.Sprintf("extract[%d] needs either selector or regex", i))
		}
		if rule.Regex != "" {
			if _, err := regexp.Compile(rule.Regex); err != nil {
				errs = append(errs, fmt.Sprintf("extract[%d].regex is invalid: %v", i, err))
			}
		}
	}

	return errs
}

// ValidateAll validates a slice of site definitions and returns a map
// of site name → validation errors. Only sites with errors are included.
func ValidateAll(sites []SiteDefinition) map[string][]string {
	result := make(map[string][]string)
	for i := range sites {
		errs := Validate(&sites[i])
		if len(errs) > 0 {
			name := sites[i].Name
			if name == "" {
				name = fmt.Sprintf("(unnamed site #%d)", i)
			}
			result[name] = errs
		}
	}
	return result
}
