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

// Imgur checks if an email is registered on Imgur via the
// signup email validation endpoint.
type Imgur struct{}

func (i *Imgur) Name() string     { return "Imgur" }
func (i *Imgur) Category() string { return "social" }

func (i *Imgur) Check(ctx context.Context, addr string, client *httpclient.Client) email.ModuleResult {
	u := "https://imgur.com/signin/ajax_email_available"

	resp, err := client.DoRequest(ctx, "POST", u,
		strings.NewReader("email="+url.QueryEscape(addr)),
		map[string]string{
			"Content-Type":     "application/x-www-form-urlencoded",
			"Accept":           "application/json",
			"X-Requested-With": "XMLHttpRequest",
		})
	if err != nil {
		return email.ModuleResult{Err: err}
	}

	if r := email.CheckRateOrError(resp, 0); r != nil {
		return *r
	}

	var data struct {
		Data struct {
			Available bool `json:"available"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &data); err != nil {
		return email.ModuleResult{Err: fmt.Errorf("parsing response: %w", err)}
	}

	// available=false means the email is taken.
	exists := !data.Data.Available
	return email.ModuleResult{
		Exists: &exists,
		Method: "register",
	}
}
