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

var snapchatXSRFPattern = regexp.MustCompile(`name="xsrf_token"\s+value="([^"]+)"`)

// Snapchat checks if an email is registered on Snapchat by probing
// the forgot-password flow: GET the page for XSRF token, then POST
// to check if the email exists.
type Snapchat struct{}

func (s *Snapchat) Name() string     { return "Snapchat" }
func (s *Snapchat) Category() string { return "social" }

func (s *Snapchat) Check(ctx context.Context, addr string, client *httpclient.Client) email.ModuleResult {
	// Step 1: GET forgot password page for XSRF token.
	pageResp, err := client.Do(ctx, "https://accounts.snapchat.com/accounts/password_reset_request", map[string]string{
		"Accept": "text/html",
	})
	if err != nil {
		return email.ModuleResult{Err: fmt.Errorf("fetching page: %w", err)}
	}

	if r := email.CheckRateOrError(pageResp, 0); r != nil {
		return *r
	}

	xsrfMatch := snapchatXSRFPattern.FindStringSubmatch(pageResp.Body)
	if len(xsrfMatch) < 2 {
		return email.ModuleResult{Err: fmt.Errorf("XSRF token not found")}
	}
	xsrfToken := xsrfMatch[1]

	// Step 2: POST to check email.
	body := fmt.Sprintf(`{"email":"%s","xsrf_token":"%s"}`, addr, xsrfToken)
	checkResp, err := client.DoRequest(ctx, "POST",
		"https://accounts.snapchat.com/accounts/password_reset_request",
		strings.NewReader(body),
		map[string]string{
			"Content-Type": "application/json",
			"Accept":       "application/json",
			"Referer":      "https://accounts.snapchat.com/accounts/password_reset_request",
		})
	if err != nil {
		return email.ModuleResult{Err: fmt.Errorf("checking email: %w", err)}
	}

	if r := email.CheckRateOrError(checkResp, 0); r != nil {
		return *r
	}

	// If the response indicates the email was found (password reset email sent).
	if strings.Contains(checkResp.Body, "hasSnapchat") || strings.Contains(checkResp.Body, "email_sent") {
		exists := true
		return email.ModuleResult{Exists: &exists, Method: "password_recovery"}
	}

	exists := false
	return email.ModuleResult{Exists: &exists, Method: "password_recovery"}
}
