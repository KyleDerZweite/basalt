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

// Adobe checks if an email is registered with Adobe by probing the
// authentication flow. This can extract partial recovery email and phone.
type Adobe struct{}

func (a *Adobe) Name() string     { return "Adobe" }
func (a *Adobe) Category() string { return "software" }

func (a *Adobe) Check(ctx context.Context, addr string, client *httpclient.Client) email.ModuleResult {
	// Step 1: POST to check auth state for the email.
	body := fmt.Sprintf(`{"username":"%s"}`, addr)
	resp, err := client.DoRequest(ctx, "POST",
		"https://auth.services.adobe.com/signin/v2/users/accounts",
		strings.NewReader(body),
		map[string]string{
			"Content-Type": "application/json",
			"Accept":       "application/json",
			"X-IMS-ClientId": "adobedotcom2",
		})
	if err != nil {
		return email.ModuleResult{Err: err}
	}

	if r := email.CheckRateOrError(resp, 0); r != nil {
		return *r
	}

	if resp.StatusCode == 404 {
		exists := false
		return email.ModuleResult{Exists: &exists, Method: "login"}
	}

	if resp.StatusCode != 200 {
		return email.ModuleResult{Err: fmt.Errorf("unexpected status %d", resp.StatusCode)}
	}

	// Parse the account info — may contain recovery hints.
	var accounts []struct {
		Type              string `json:"type"`
		AuthenticationType string `json:"authenticationType"`
		EmailHint         string `json:"emailHint"`
		PhoneHint         string `json:"phoneHint"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &accounts); err != nil {
		// Single object response.
		exists := true
		return email.ModuleResult{Exists: &exists, Method: "login"}
	}

	if len(accounts) == 0 {
		exists := false
		return email.ModuleResult{Exists: &exists, Method: "login"}
	}

	exists := true
	result := email.ModuleResult{
		Exists:   &exists,
		Method:   "login",
		Metadata: make(map[string]string),
	}

	// Extract recovery hints.
	for _, acc := range accounts {
		if acc.EmailHint != "" {
			result.EmailRecovery = acc.EmailHint
		}
		if acc.PhoneHint != "" {
			result.PhoneRecovery = acc.PhoneHint
		}
		if acc.Type != "" {
			result.Metadata["account_type"] = acc.Type
		}
	}

	return result
}
