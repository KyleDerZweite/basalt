// SPDX-License-Identifier: AGPL-3.0-or-later

package sitedb

import (
	"encoding/json"
	"fmt"
)

type sherlockSite struct {
	URL               string            `json:"url"`
	URLMain           string            `json:"urlMain"`
	URLProbe          string            `json:"urlProbe"`
	ErrorType         string            `json:"errorType"`
	ErrorMsg          interface{}       `json:"errorMsg"` // can be string or []string
	UsernameClaimed   string            `json:"username_claimed"`
	UsernameUnclaimed string            `json:"username_unclaimed"`
	RegexCheck        string            `json:"regexCheck"`
	Headers           map[string]string `json:"headers"`
	RequestMethod     string            `json:"request_method"`
	RequestPayload    interface{}       `json:"request_payload"`
	IsNSFW            bool              `json:"isNSFW"`
}

func importSherlock(data []byte) ([]SiteDefinition, error) {
	// Sherlock's JSON has non-object entries like "$schema" that we must skip.
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return nil, fmt.Errorf("parsing Sherlock JSON: %w", err)
	}

	sites := make([]SiteDefinition, 0, len(rawMap))
	for name, rawVal := range rawMap {
		if len(rawVal) == 0 || rawVal[0] != '{' {
			continue
		}

		var s sherlockSite
		if err := json.Unmarshal(rawVal, &s); err != nil || s.URL == "" {
			continue
		}

		// Sherlock uses {} as placeholder.
		urlTemplate := normalizePlaceholder(s.URL, "{}")

		check := Check{
			Method:         coalesce(s.RequestMethod, "GET"),
			ExpectedStatus: 200,
			Headers:        s.Headers,
		}

		if s.ErrorType == "message" {
			check.AbsenceStrings = extractErrorMsgs(s.ErrorMsg)
		}

		tags := []string{}
		if s.IsNSFW {
			tags = append(tags, "nsfw")
		}

		sd := SiteDefinition{
			Name:          name,
			URLTemplate:   urlTemplate,
			URLMain:       s.URLMain,
			Category:      "other",
			Tags:          tags,
			SeedTypes:     []string{"username"},
			Check:         check,
			UsernameRegex: s.RegexCheck,
			TestAccounts: TestAccounts{
				Claimed:   coalesce(s.UsernameClaimed, "blue"),
				Unclaimed: coalesce(s.UsernameUnclaimed, "zzz_basalt_nonexistent_user_zzz"),
			},
			Source: "sherlock",
		}

		if s.URLProbe != "" {
			sd.ProbeURL = normalizePlaceholder(s.URLProbe, "{}")
		}

		sites = append(sites, sd)
	}

	return sites, nil
}
