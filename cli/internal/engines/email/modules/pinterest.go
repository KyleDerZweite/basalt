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

// Pinterest checks if an email is registered on Pinterest via the
// email existence check endpoint.
type Pinterest struct{}

func (p *Pinterest) Name() string     { return "Pinterest" }
func (p *Pinterest) Category() string { return "social" }

func (p *Pinterest) Check(ctx context.Context, addr string, client *httpclient.Client) email.ModuleResult {
	u := "https://www.pinterest.com/resource/EmailExistsResource/get/?source_url=/&data=" +
		url.QueryEscape(fmt.Sprintf(`{"options":{"email":"%s"}}`, addr))

	resp, err := client.Do(ctx, u, map[string]string{
		"Accept":           "application/json",
		"X-Requested-With": "XMLHttpRequest",
	})
	if err != nil {
		return email.ModuleResult{Err: err}
	}

	if r := email.CheckRateOrError(resp, 200); r != nil {
		return *r
	}

	var data struct {
		ResourceResponse struct {
			Data bool `json:"data"`
		} `json:"resource_response"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &data); err != nil {
		return email.ModuleResult{Err: fmt.Errorf("parsing response: %w", err)}
	}

	exists := data.ResourceResponse.Data
	return email.ModuleResult{
		Exists: &exists,
		Method: "register",
	}
}
