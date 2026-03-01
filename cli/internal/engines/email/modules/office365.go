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

// Office365 checks if an email is associated with a Microsoft/Office365
// account by probing the GetCredentialType endpoint used during login.
type Office365 struct{}

func (o *Office365) Name() string     { return "Office365" }
func (o *Office365) Category() string { return "email_provider" }

func (o *Office365) Check(ctx context.Context, addr string, client *httpclient.Client) email.ModuleResult {
	body := fmt.Sprintf(`{"Username":"%s","isOtherIdpSupported":true,"checkPhones":false,"isRemoteNGCSupported":true,"isCookieBannerShown":false,"isFidoSupported":true,"forceotclogin":false,"otclogindisallowed":false,"isExternalFederationDisallowed":false,"isRemoteConnectSupported":false,"federationFlags":0,"isSignup":false,"flowToken":"","isAccessPassSupported":true}`, addr)

	resp, err := client.DoRequest(ctx, "POST",
		"https://login.microsoftonline.com/common/GetCredentialType?mkt=en-US",
		strings.NewReader(body),
		map[string]string{
			"Content-Type": "application/json",
			"Accept":       "application/json",
		})
	if err != nil {
		return email.ModuleResult{Err: err}
	}

	if r := email.CheckRateOrError(resp, 200); r != nil {
		return *r
	}

	var data struct {
		IfExistsResult   int    `json:"IfExistsResult"`
		ThrottleStatus   int    `json:"ThrottleStatus"`
		EstsProperties   struct {
			UserTenantBranding interface{} `json:"UserTenantBranding"`
		} `json:"EstsProperties"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &data); err != nil {
		return email.ModuleResult{Err: fmt.Errorf("parsing response: %w", err)}
	}

	if data.ThrottleStatus == 1 {
		return email.ModuleResult{RateLimit: true}
	}

	result := email.ModuleResult{
		Method:   "login",
		Metadata: make(map[string]string),
	}

	// IfExistsResult values:
	// 0 = exists, 1 = doesn't exist, 5 = exists (different IdP),
	// 6 = exists (different tenant)
	switch data.IfExistsResult {
	case 0, 5, 6:
		exists := true
		result.Exists = &exists
		if data.IfExistsResult == 5 {
			result.Metadata["federation"] = "external_idp"
		}
	case 1:
		exists := false
		result.Exists = &exists
	default:
		result.Err = fmt.Errorf("unknown IfExistsResult: %d", data.IfExistsResult)
	}

	return result
}
