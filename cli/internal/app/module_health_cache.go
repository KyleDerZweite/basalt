// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"time"

	"github.com/KyleDerZweite/basalt/internal/modules"
)

const (
	defaultHealthyModuleHealthTTL = 3 * time.Hour
	defaultOfflineModuleHealthTTL = 30 * time.Minute
)

// ModuleHealthCacheEntry stores cached module verification state.
type ModuleHealthCacheEntry struct {
	ModuleName string
	Version    string
	ConfigHash string
	Status     string
	Message    string
	CheckedAt  time.Time
	ExpiresAt  time.Time
}

func moduleHealthTTL(status modules.HealthStatus, override time.Duration) time.Duration {
	if override > 0 {
		return override
	}
	if status == modules.Offline {
		return defaultOfflineModuleHealthTTL
	}
	return defaultHealthyModuleHealthTTL
}
