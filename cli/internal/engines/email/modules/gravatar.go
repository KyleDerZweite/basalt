// SPDX-License-Identifier: AGPL-3.0-or-later

package modules

import (
	"context"
	"crypto/md5"
	"fmt"
	"strings"

	"github.com/kyle/basalt/internal/engines/email"
	"github.com/kyle/basalt/internal/httpclient"
)

// Gravatar checks if an email has a Gravatar profile by requesting the
// MD5-hashed avatar URL with d=404 (return 404 if no custom avatar).
type Gravatar struct{}

func (g *Gravatar) Name() string     { return "Gravatar" }
func (g *Gravatar) Category() string { return "social" }

func (g *Gravatar) Check(ctx context.Context, addr string, client *httpclient.Client) email.ModuleResult {
	hash := fmt.Sprintf("%x", md5.Sum([]byte(strings.ToLower(strings.TrimSpace(addr)))))
	u := "https://en.gravatar.com/" + hash + ".json"

	resp, err := client.Do(ctx, u, nil)
	if err != nil {
		return email.ModuleResult{Err: err}
	}

	if r := email.CheckRateOrError(resp, 0); r != nil {
		return *r
	}

	// 200 = profile exists, 404 = no profile.
	exists := resp.StatusCode == 200
	return email.ModuleResult{
		Exists:   &exists,
		Method:   "register",
		Metadata: map[string]string{"profile_url": "https://gravatar.com/" + hash},
	}
}
