// SPDX-License-Identifier: AGPL-3.0-or-later

package modules

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/kyle/basalt/internal/engines/email"
	"github.com/kyle/basalt/internal/httpclient"
)

// Amazon checks if an email is registered on Amazon by probing the
// forgot-password flow. If the email exists, Amazon asks for a captcha
// or shows a success message.
type Amazon struct{}

func (a *Amazon) Name() string     { return "Amazon" }
func (a *Amazon) Category() string { return "shopping" }

func (a *Amazon) Check(ctx context.Context, addr string, client *httpclient.Client) email.ModuleResult {
	// Step 1: GET forgot password page.
	pageResp, err := client.Do(ctx, "https://www.amazon.com/ap/forgotpassword?openid.pape.max_auth_age=0", map[string]string{
		"Accept": "text/html",
	})
	if err != nil {
		return email.ModuleResult{Err: fmt.Errorf("fetching page: %w", err)}
	}

	if r := email.CheckRateOrError(pageResp, 0); r != nil {
		return *r
	}

	// Step 2: POST the email to the forgot password form.
	formData := url.Values{
		"email": {addr},
	}

	checkResp, err := client.DoRequest(ctx, "POST",
		"https://www.amazon.com/ap/forgotpassword",
		strings.NewReader(formData.Encode()),
		map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
			"Accept":       "text/html",
			"Referer":      "https://www.amazon.com/ap/forgotpassword",
		})
	if err != nil {
		return email.ModuleResult{Err: fmt.Errorf("checking email: %w", err)}
	}

	if r := email.CheckRateOrError(checkResp, 0); r != nil {
		return *r
	}

	body := strings.ToLower(checkResp.Body)

	// If Amazon asks to verify or shows OTP page, the account exists.
	if strings.Contains(body, "verification code") ||
		strings.Contains(body, "otp") ||
		strings.Contains(body, "we sent a code") {
		exists := true
		return email.ModuleResult{Exists: &exists, Method: "password_recovery"}
	}

	// If "no account found" or similar error.
	if strings.Contains(body, "no account found") ||
		strings.Contains(body, "cannot find an account") {
		exists := false
		return email.ModuleResult{Exists: &exists, Method: "password_recovery"}
	}

	// Captcha block - inconclusive.
	if strings.Contains(body, "captcha") {
		return email.ModuleResult{Err: fmt.Errorf("captcha required")}
	}

	return email.ModuleResult{Err: fmt.Errorf("ambiguous response")}
}
