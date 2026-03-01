// SPDX-License-Identifier: AGPL-3.0-or-later

package email

import (
	"context"
	"fmt"

	"github.com/kyle/basalt/internal/httpclient"
)

// Module is the interface every email-check site must implement.
// Each module is a self-contained check for one service.
type Module interface {
	// Name returns a human-readable name (e.g., "Instagram", "Discord").
	Name() string

	// Category returns a classification (e.g., "social", "coding", "shopping").
	Category() string

	// Check tests whether the email is registered on this service.
	// Returns a ModuleResult. Must respect context cancellation.
	Check(ctx context.Context, email string, client *httpclient.Client) ModuleResult
}

// ModuleResult is the standardized output from each module.
type ModuleResult struct {
	// Exists is true if the email is registered, false if not, nil if inconclusive.
	Exists *bool

	// RateLimit is true if the service rate-limited us.
	RateLimit bool

	// EmailRecovery is a partially obfuscated recovery email (e.g., "ex****e@gmail.com").
	EmailRecovery string

	// PhoneRecovery is a partially obfuscated phone (e.g., "***-***-**78").
	PhoneRecovery string

	// Method describes the technique used (e.g., "register", "login", "password_recovery").
	Method string

	// Metadata holds arbitrary extra data extracted from the service.
	Metadata map[string]string

	// Err is set if the check failed.
	Err error
}

// BoolPtr is a helper to create a *bool from a bool value.
func BoolPtr(b bool) *bool {
	return &b
}

// CheckRateOrError returns a ModuleResult if the response indicates rate limiting
// or an unexpected status. Returns nil if the response is OK to process.
func CheckRateOrError(resp *httpclient.Response, expectedStatus int) *ModuleResult {
	if resp.StatusCode == 429 {
		return &ModuleResult{RateLimit: true}
	}
	if expectedStatus > 0 && resp.StatusCode != expectedStatus {
		return &ModuleResult{Err: fmt.Errorf("unexpected status %d", resp.StatusCode)}
	}
	return nil
}
