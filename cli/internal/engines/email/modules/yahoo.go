// SPDX-License-Identifier: AGPL-3.0-or-later

package modules

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/kyle/basalt/internal/engines/email"
	"github.com/kyle/basalt/internal/httpclient"
)

var yahooAcrumbPattern = regexp.MustCompile(`name="acrumb"\s+value="([^"]+)"`)

// Yahoo checks if an email is registered on Yahoo by probing the
// login flow: GET login page for acrumb token, then POST to check
// if the identifier exists.
type Yahoo struct{}

func (y *Yahoo) Name() string     { return "Yahoo" }
func (y *Yahoo) Category() string { return "email_provider" }

func (y *Yahoo) Check(ctx context.Context, addr string, client *httpclient.Client) email.ModuleResult {
	// Step 1: GET login page for session and acrumb.
	loginResp, err := client.Do(ctx, "https://login.yahoo.com/", map[string]string{
		"Accept": "text/html",
	})
	if err != nil {
		return email.ModuleResult{Err: fmt.Errorf("fetching login page: %w", err)}
	}

	if r := email.CheckRateOrError(loginResp, 0); r != nil {
		return *r
	}

	acrumbMatch := yahooAcrumbPattern.FindStringSubmatch(loginResp.Body)
	if len(acrumbMatch) < 2 {
		return email.ModuleResult{Err: fmt.Errorf("acrumb token not found")}
	}
	acrumb := acrumbMatch[1]

	// Step 2: POST to check username.
	body := fmt.Sprintf(`{"acrumb":"%s","username":"%s"}`, acrumb, addr)
	checkResp, err := client.DoRequest(ctx, "POST",
		"https://login.yahoo.com/",
		strings.NewReader(body),
		map[string]string{
			"Content-Type": "application/json",
			"Accept":       "application/json",
			"Referer":      "https://login.yahoo.com/",
		})
	if err != nil {
		return email.ModuleResult{Err: fmt.Errorf("checking email: %w", err)}
	}

	if r := email.CheckRateOrError(checkResp, 0); r != nil {
		return *r
	}

	// If the response asks for password, the account exists.
	if strings.Contains(checkResp.Body, "IDENTIFIER_EXISTS") ||
		strings.Contains(checkResp.Body, "password") {
		exists := true
		return email.ModuleResult{Exists: &exists, Method: "login"}
	}

	// "IDENTIFIER_NOT_FOUND" means no account.
	if strings.Contains(checkResp.Body, "IDENTIFIER_NOT_FOUND") ||
		strings.Contains(checkResp.Body, "error") {
		exists := false
		return email.ModuleResult{Exists: &exists, Method: "login"}
	}

	return email.ModuleResult{Err: fmt.Errorf("ambiguous response")}
}
