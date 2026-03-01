// SPDX-License-Identifier: AGPL-3.0-or-later

package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kyle/basalt/internal/engines/email"
	"github.com/kyle/basalt/internal/httpclient"
)

// Samsung checks if an email is registered with a Samsung account
// by probing the account existence check endpoint.
type Samsung struct{}

func (s *Samsung) Name() string     { return "Samsung" }
func (s *Samsung) Category() string { return "tech" }

func (s *Samsung) Check(ctx context.Context, addr string, client *httpclient.Client) email.ModuleResult {
	body := fmt.Sprintf(`{"id":"%s"}`, addr)

	resp, err := client.DoRequest(ctx, "POST",
		"https://account.samsung.com/accounts/v1/Samsung/checkEmailIDAvailability",
		strings.NewReader(body),
		map[string]string{
			"Content-Type": "application/json",
			"Accept":       "application/json",
		})
	if err != nil {
		return email.ModuleResult{Err: err}
	}

	if r := email.CheckRateOrError(resp, 0); r != nil {
		return *r
	}

	var data struct {
		ResultCode string `json:"resultCode"`
		Reason     string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &data); err != nil {
		return email.ModuleResult{Err: fmt.Errorf("parsing response: %w", err)}
	}

	// "000" = available (not registered), "008" or other = taken.
	switch {
	case data.ResultCode == "000":
		exists := false
		return email.ModuleResult{Exists: &exists, Method: "register"}
	case data.ResultCode != "":
		exists := true
		return email.ModuleResult{Exists: &exists, Method: "register"}
	default:
		return email.ModuleResult{Err: fmt.Errorf("unexpected result: %s", data.Reason)}
	}
}
