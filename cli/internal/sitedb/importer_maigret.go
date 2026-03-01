// SPDX-License-Identifier: AGPL-3.0-or-later

package sitedb

import (
	"encoding/json"
	"fmt"
)

type maigretSite struct {
	URLMain           string            `json:"urlMain"`
	URL               string            `json:"url"`
	URLProbe          string            `json:"urlProbe"`
	CheckType         string            `json:"checkType"`
	UsernameClaimed   string            `json:"usernameClaimed"`
	UsernameUnclaimed string            `json:"usernameUnclaimed"`
	PresenseStrs      []string          `json:"presenseStrs"`
	AbsenceStrs       []string          `json:"absenceStrs"`
	RegexCheck        string            `json:"regexCheck"`
	Headers           map[string]string `json:"headers"`
	Tags              []string          `json:"tags"`
	Disabled          bool              `json:"disabled"`
	Engine            string            `json:"engine"`
}

func importMaigret(data []byte) ([]SiteDefinition, error) {
	// Maigret wraps sites under a "sites" key. Try that first, then fall back
	// to flat map.
	var wrapped struct {
		Sites map[string]maigretSite `json:"sites"`
	}
	raw := make(map[string]maigretSite)

	if err := json.Unmarshal(data, &wrapped); err == nil && len(wrapped.Sites) > 0 {
		raw = wrapped.Sites
	} else if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing Maigret JSON: %w", err)
	}

	sites := make([]SiteDefinition, 0, len(raw))
	for name, m := range raw {
		if m.Disabled || m.URL == "" {
			continue
		}

		// Maigret uses {username} and sometimes {0} as placeholders.
		urlTemplate := normalizePlaceholder(m.URL, "{0}")

		check := Check{
			Method:         "GET",
			ExpectedStatus: 200,
			Headers:        m.Headers,
		}

		switch m.CheckType {
		case "message":
			check.PresenceStrings = m.PresenseStrs
			check.AbsenceStrings = m.AbsenceStrs
		}

		tags := m.Tags
		if m.Engine != "" {
			tags = append(tags, m.Engine)
		}

		sd := SiteDefinition{
			Name:          name,
			URLTemplate:   urlTemplate,
			URLMain:       m.URLMain,
			Category:      guessCategory(tags),
			Tags:          tags,
			SeedTypes:     []string{"username"},
			Check:         check,
			UsernameRegex: m.RegexCheck,
			TestAccounts: TestAccounts{
				Claimed:   m.UsernameClaimed,
				Unclaimed: coalesce(m.UsernameUnclaimed, "zzz_basalt_nonexistent_user_zzz"),
			},
			Source: "maigret",
		}

		if m.URLProbe != "" {
			sd.ProbeURL = normalizePlaceholder(m.URLProbe, "{0}")
		}

		sites = append(sites, sd)
	}

	return sites, nil
}

// extractErrorMsgs handles Sherlock's errorMsg field which can be string or []string.
func extractErrorMsgs(v interface{}) []string {
	switch val := v.(type) {
	case string:
		if val != "" {
			return []string{val}
		}
	case []interface{}:
		var msgs []string
		for _, item := range val {
			if s, ok := item.(string); ok && s != "" {
				msgs = append(msgs, s)
			}
		}
		return msgs
	}
	return nil
}
