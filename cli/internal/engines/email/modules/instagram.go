// SPDX-License-Identifier: AGPL-3.0-or-later

package modules

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/kyle/basalt/internal/engines/email"
	"github.com/kyle/basalt/internal/httpclient"
)

var instagramCSRFPattern = regexp.MustCompile(`"csrf_token":"([^"]+)"`)

// Instagram checks if an email is registered on Instagram via the
// web signup flow: GET signup page for CSRF token, then POST to the
// email validation endpoint.
type Instagram struct{}

func (i *Instagram) Name() string     { return "Instagram" }
func (i *Instagram) Category() string { return "social" }

func (i *Instagram) Check(ctx context.Context, addr string, client *httpclient.Client) email.ModuleResult {
	// Step 1: GET signup page to extract CSRF token.
	signupResp, err := client.Do(ctx, "https://www.instagram.com/accounts/emailsignup/", map[string]string{
		"Accept": "text/html",
	})
	if err != nil {
		return email.ModuleResult{Err: fmt.Errorf("fetching signup page: %w", err)}
	}

	if r := email.CheckRateOrError(signupResp, 0); r != nil {
		return *r
	}

	csrfMatch := instagramCSRFPattern.FindStringSubmatch(signupResp.Body)
	if len(csrfMatch) < 2 {
		return email.ModuleResult{Err: fmt.Errorf("CSRF token not found")}
	}
	csrfToken := csrfMatch[1]

	// Step 2: POST to check email availability.
	checkResp, err := client.DoRequest(ctx, "POST",
		"https://www.instagram.com/api/v1/web/accounts/web_create_ajax/attempt/",
		strings.NewReader("email="+url.QueryEscape(addr)),
		map[string]string{
			"Content-Type":     "application/x-www-form-urlencoded",
			"X-CSRFToken":      csrfToken,
			"X-Requested-With": "XMLHttpRequest",
			"Referer":          "https://www.instagram.com/accounts/emailsignup/",
		})
	if err != nil {
		return email.ModuleResult{Err: fmt.Errorf("checking email: %w", err)}
	}

	if r := email.CheckRateOrError(checkResp, 0); r != nil {
		return *r
	}

	// "email_is_taken" in response indicates the email is registered.
	exists := strings.Contains(checkResp.Body, "email_is_taken")
	return email.ModuleResult{
		Exists: &exists,
		Method: "register",
	}
}
