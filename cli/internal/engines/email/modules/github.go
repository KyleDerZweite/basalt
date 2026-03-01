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

var githubAuthTokenPattern = regexp.MustCompile(`name="authenticity_token"\s+value="([^"]+)"`)

// GitHub checks if an email is registered on GitHub via the
// signup flow: GET /signup for authenticity token, then POST to
// /signup_check/email.
type GitHub struct{}

func (g *GitHub) Name() string     { return "GitHub" }
func (g *GitHub) Category() string { return "coding" }

func (g *GitHub) Check(ctx context.Context, addr string, client *httpclient.Client) email.ModuleResult {
	// Step 1: GET signup page for authenticity token.
	signupResp, err := client.Do(ctx, "https://github.com/signup", map[string]string{
		"Accept": "text/html",
	})
	if err != nil {
		return email.ModuleResult{Err: fmt.Errorf("fetching signup page: %w", err)}
	}

	if r := email.CheckRateOrError(signupResp, 0); r != nil {
		return *r
	}

	tokenMatch := githubAuthTokenPattern.FindStringSubmatch(signupResp.Body)
	if len(tokenMatch) < 2 {
		return email.ModuleResult{Err: fmt.Errorf("authenticity token not found")}
	}
	token := tokenMatch[1]

	// Step 2: POST to check email.
	checkResp, err := client.DoRequest(ctx, "POST",
		"https://github.com/signup_check/email",
		strings.NewReader("value="+url.QueryEscape(addr)+"&authenticity_token="+url.QueryEscape(token)),
		map[string]string{
			"Content-Type":     "application/x-www-form-urlencoded",
			"Accept":           "application/json",
			"X-Requested-With": "XMLHttpRequest",
			"Referer":          "https://github.com/signup",
		})
	if err != nil {
		return email.ModuleResult{Err: fmt.Errorf("checking email: %w", err)}
	}

	if r := email.CheckRateOrError(checkResp, 0); r != nil {
		return *r
	}

	// 422 = email already taken, 200 = available.
	exists := checkResp.StatusCode == 422
	return email.ModuleResult{
		Exists: &exists,
		Method: "register",
	}
}
