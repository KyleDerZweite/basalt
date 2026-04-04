// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ServeRuntime describes a running local API process.
type ServeRuntime struct {
	PID           int       `json:"pid"`
	ListenAddress string    `json:"listen_address"`
	BaseURL       string    `json:"base_url"`
	Version       string    `json:"version"`
	DataDir       string    `json:"data_dir"`
	LogFile       string    `json:"log_file"`
	StartedAt     time.Time `json:"started_at"`
}

// DefaultServeRuntimePath returns the runtime metadata path for the local API.
func DefaultServeRuntimePath(dataDir string) string {
	if dataDir == "" {
		dataDir = DefaultDataDir()
	}
	return filepath.Join(dataDir, "run", "serve.json")
}

// DefaultServeLogPath returns the default log file path for the local API.
func DefaultServeLogPath(dataDir string) string {
	if dataDir == "" {
		dataDir = DefaultDataDir()
	}
	return filepath.Join(dataDir, "logs", "serve.log")
}

// ReadServeRuntime loads runtime metadata from disk.
func ReadServeRuntime(dataDir string) (*ServeRuntime, error) {
	path := DefaultServeRuntimePath(dataDir)
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var runtime ServeRuntime
	if err := json.Unmarshal(payload, &runtime); err != nil {
		return nil, fmt.Errorf("decoding serve runtime %s: %w", path, err)
	}
	return &runtime, nil
}

// WriteServeRuntime persists runtime metadata atomically.
func WriteServeRuntime(dataDir string, runtime ServeRuntime) error {
	path := DefaultServeRuntimePath(dataDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating runtime dir: %w", err)
	}

	payload, err := json.MarshalIndent(runtime, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding serve runtime: %w", err)
	}
	payload = append(payload, '\n')

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, payload, 0o644); err != nil {
		return fmt.Errorf("writing runtime temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("renaming runtime file: %w", err)
	}
	return nil
}

// RemoveServeRuntime removes persisted runtime metadata.
func RemoveServeRuntime(dataDir string) error {
	path := DefaultServeRuntimePath(dataDir)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
