// SPDX-License-Identifier: AGPL-3.0-or-later

package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/kyle/basalt/internal/engines/email"
	"github.com/kyle/basalt/internal/httpclient"
)

// Docker checks if an email is registered on Docker Hub via the
// registration validation endpoint.
type Docker struct{}

func (d *Docker) Name() string     { return "Docker" }
func (d *Docker) Category() string { return "coding" }

func (d *Docker) Check(ctx context.Context, addr string, client *httpclient.Client) email.ModuleResult {
	u := "https://hub.docker.com/v2/users/signup/"

	body := fmt.Sprintf(`{"email":"%s","username":"basalt_probe_%s","password":"Pr0be!Check#99"}`,
		addr, url.QueryEscape(strings.Split(addr, "@")[0]))

	resp, err := client.DoRequest(ctx, "POST", u,
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

	// Parse error response — if email is taken, the error message says so.
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(resp.Body), &data); err != nil {
		return email.ModuleResult{Err: fmt.Errorf("parsing response: %w", err)}
	}

	// Check if the email field has an error about being taken.
	if emailErrors, ok := data["email"]; ok {
		if errs, ok := emailErrors.([]interface{}); ok {
			for _, e := range errs {
				if s, ok := e.(string); ok && strings.Contains(strings.ToLower(s), "already") {
					exists := true
					return email.ModuleResult{Exists: &exists, Method: "register"}
				}
			}
		}
	}

	// If we got a 200/201, the email was available (signup would succeed).
	if resp.StatusCode == 200 || resp.StatusCode == 201 {
		exists := false
		return email.ModuleResult{Exists: &exists, Method: "register"}
	}

	exists := false
	return email.ModuleResult{Exists: &exists, Method: "register"}
}
