// SPDX-License-Identifier: AGPL-3.0-or-later

package sitedb

import (
	"encoding/json"
	"fmt"
	"strings"
)

type wmnData struct {
	License []interface{} `json:"license"`
	Sites   []wmnSite     `json:"sites"`
}

type wmnSite struct {
	Name     string            `json:"name"`
	URICheck string            `json:"uri_check"`
	URIPretty string           `json:"uri_pretty"`
	ECode    int               `json:"e_code"`
	MCode    int               `json:"m_code"`
	EString  string            `json:"e_string"`
	MString  string            `json:"m_string"`
	Cat      string            `json:"cat"`
	Known    []string          `json:"known"`
	Headers  map[string]string `json:"headers"`
	PostBody string            `json:"post_body"`
	Valid    bool              `json:"valid"`
}

func importWMN(data []byte) ([]SiteDefinition, error) {
	var raw wmnData
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing WhatsMyName JSON: %w", err)
	}

	sites := make([]SiteDefinition, 0, len(raw.Sites))
	for _, w := range raw.Sites {
		if w.URICheck == "" {
			continue
		}

		urlTemplate := normalizePlaceholder(w.URICheck, "{account}")

		check := Check{
			ExpectedStatus: w.ECode,
			Headers:        w.Headers,
		}

		if w.PostBody != "" {
			check.Method = "POST"
			check.BodyTemplate = strings.ReplaceAll(w.PostBody, "{account}", "{seed}")
		} else {
			check.Method = "GET"
		}

		if w.EString != "" {
			check.PresenceStrings = []string{w.EString}
		}
		if w.MString != "" {
			check.AbsenceStrings = []string{w.MString}
		}

		claimed := ""
		if len(w.Known) > 0 {
			claimed = w.Known[0]
		}

		sd := SiteDefinition{
			Name:        w.Name,
			URLTemplate: urlTemplate,
			URLMain:     extractMainURL(urlTemplate),
			Category:    normalizeCat(w.Cat),
			Tags:        []string{},
			SeedTypes:   []string{"username"},
			Check:       check,
			TestAccounts: TestAccounts{
				Claimed:   claimed,
				Unclaimed: "zzz_basalt_nonexistent_user_zzz",
			},
			Source: "wmn",
		}

		sites = append(sites, sd)
	}

	return sites, nil
}
