// SPDX-License-Identifier: AGPL-3.0-or-later

package modules

import (
	"context"
	"fmt"
	"strings"

	"github.com/kyle/basalt/internal/engines/email"
	"github.com/kyle/basalt/internal/httpclient"
)

// Discord checks if an email is registered on Discord by attempting
// to register with the email. If the email is already taken, Discord
// returns an "EMAIL_ALREADY_REGISTERED" error.
type Discord struct{}

func (d *Discord) Name() string     { return "Discord" }
func (d *Discord) Category() string { return "social" }

func (d *Discord) Check(ctx context.Context, addr string, client *httpclient.Client) email.ModuleResult {
	body := fmt.Sprintf(`{"email":"%s","username":"basalt_probe","password":"Pr0be!Check#99","date_of_birth":"1990-01-01","consent":true}`, addr)

	resp, err := client.DoRequest(ctx, "POST",
		"https://discord.com/api/v9/auth/register",
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

	// Check for email already registered error.
	if strings.Contains(resp.Body, "EMAIL_ALREADY_REGISTERED") {
		exists := true
		return email.ModuleResult{Exists: &exists, Method: "register"}
	}

	// If we get captcha or other error without EMAIL_ALREADY_REGISTERED,
	// the email might be available, or we can't determine.
	if strings.Contains(resp.Body, "captcha") {
		return email.ModuleResult{Err: fmt.Errorf("captcha required")}
	}

	exists := false
	return email.ModuleResult{Exists: &exists, Method: "register"}
}
