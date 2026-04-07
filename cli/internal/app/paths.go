// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"os"
	"path/filepath"
)

// DefaultDataDir returns the local application data directory.
func DefaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ".basalt"
	}
	return filepath.Join(home, ".basalt")
}

// DefaultConfigPath returns the default API key config path.
func DefaultConfigPath(dataDir string) string {
	if dataDir == "" {
		dataDir = DefaultDataDir()
	}
	return filepath.Join(dataDir, "config")
}

func defaultDBPath(dataDir string) string {
	return filepath.Join(dataDir, "basalt.db")
}

func defaultDBDSN(dataDir string) string {
	return defaultDBPath(dataDir) + "?_pragma=foreign_keys(1)"
}
