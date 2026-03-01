// SPDX-License-Identifier: AGPL-3.0-or-later

package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/kyle/basalt/internal/engines/email"
	"github.com/kyle/basalt/internal/httpclient"
)

// Twitter checks if an email is registered on Twitter/X via the
// email availability endpoint used during signup.
type Twitter struct{}

func (t *Twitter) Name() string     { return "Twitter" }
func (t *Twitter) Category() string { return "social" }

func (t *Twitter) Check(ctx context.Context, addr string, client *httpclient.Client) email.ModuleResult {
	u := "https://api.twitter.com/i/users/email_available.json?email=" + url.QueryEscape(addr)

	resp, err := client.Do(ctx, u, map[string]string{
		"Accept": "application/json",
	})
	if err != nil {
		return email.ModuleResult{Err: err}
	}

	if r := email.CheckRateOrError(resp, 200); r != nil {
		return *r
	}

	var data struct {
		Taken bool `json:"taken"`
		Valid bool `json:"valid"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &data); err != nil {
		return email.ModuleResult{Err: fmt.Errorf("parsing response: %w", err)}
	}

	exists := data.Taken
	return email.ModuleResult{
		Exists: &exists,
		Method: "register",
	}
}
