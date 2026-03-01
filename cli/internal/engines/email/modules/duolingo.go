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

// Duolingo checks if an email is registered on Duolingo via the
// login endpoint — a non-existent email returns a specific error.
type Duolingo struct{}

func (d *Duolingo) Name() string     { return "Duolingo" }
func (d *Duolingo) Category() string { return "education" }

func (d *Duolingo) Check(ctx context.Context, addr string, client *httpclient.Client) email.ModuleResult {
	u := "https://www.duolingo.com/2017-06-30/login?fields="

	body := fmt.Sprintf(`{"identifier":"%s","password":"basalt_probe_invalid_pw"}`, addr)

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

	var data struct {
		Failure string `json:"failure"`
	}
	json.Unmarshal([]byte(resp.Body), &data)

	// "user_does_not_exist" = not registered, "invalid_password" = registered.
	switch {
	case data.Failure == "invalid_password" || resp.StatusCode == 403:
		exists := true
		return email.ModuleResult{Exists: &exists, Method: "login"}
	case data.Failure == "user_does_not_exist" || resp.StatusCode == 404:
		exists := false
		return email.ModuleResult{Exists: &exists, Method: "login"}
	default:
		return email.ModuleResult{Err: fmt.Errorf("unexpected response: %s", data.Failure)}
	}
}
