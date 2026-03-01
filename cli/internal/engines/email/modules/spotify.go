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

// Spotify checks if an email is registered on Spotify via the
// signup validation endpoint.
type Spotify struct{}

func (s *Spotify) Name() string     { return "Spotify" }
func (s *Spotify) Category() string { return "music" }

func (s *Spotify) Check(ctx context.Context, addr string, client *httpclient.Client) email.ModuleResult {
	u := "https://spclient.wg.spotify.com/signup/public/v1/account?validate=1&email=" + url.QueryEscape(addr)

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
		Status int `json:"status"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &data); err != nil {
		return email.ModuleResult{Err: fmt.Errorf("parsing response: %w", err)}
	}

	// Status 20 means the email is already taken.
	exists := data.Status == 20
	return email.ModuleResult{
		Exists: &exists,
		Method: "register",
	}
}
