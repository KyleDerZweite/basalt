// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"time"

	"github.com/KyleDerZweite/basalt/internal/graph"
)

// ScanStatus represents the lifecycle state of a scan.
type ScanStatus string

const (
	ScanStatusQueued    ScanStatus = "queued"
	ScanStatusVerifying ScanStatus = "verifying"
	ScanStatusRunning   ScanStatus = "running"
	ScanStatusCompleted ScanStatus = "completed"
	ScanStatusPartial   ScanStatus = "partial"
	ScanStatusFailed    ScanStatus = "failed"
	ScanStatusCanceled  ScanStatus = "canceled"
)

// Settings holds local product settings persisted on disk.
type Settings struct {
	StrictMode      bool       `json:"strict_mode"`
	DisabledModules []string   `json:"disabled_modules,omitempty"`
	LegalAcceptedAt *time.Time `json:"legal_accepted_at,omitempty"`
}

// ScanRequest describes a scan job request.
type ScanRequest struct {
	Seeds           []graph.Seed `json:"seeds"`
	Depth           int          `json:"depth"`
	Concurrency     int          `json:"concurrency"`
	TimeoutSeconds  int          `json:"timeout_seconds"`
	ConfigPath      string       `json:"config_path,omitempty"`
	StrictMode      bool         `json:"strict_mode,omitempty"`
	DisabledModules []string     `json:"disabled_modules,omitempty"`
}

// ModuleStatus is the JSON-safe health view for a module.
type ModuleStatus struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Message     string `json:"message"`
}

// ScanRecord is the persisted representation of a scan.
type ScanRecord struct {
	ID           string         `json:"id"`
	Status       ScanStatus     `json:"status"`
	StartedAt    time.Time      `json:"started_at"`
	CompletedAt  *time.Time     `json:"completed_at,omitempty"`
	UpdatedAt    time.Time      `json:"updated_at"`
	Seeds        []graph.Seed   `json:"seeds"`
	Options      ScanRequest    `json:"options"`
	Health       []ModuleStatus `json:"health,omitempty"`
	NodeCount    int            `json:"node_count"`
	EdgeCount    int            `json:"edge_count"`
	ErrorMessage string         `json:"error_message,omitempty"`
	Graph        *graph.Graph   `json:"graph,omitempty"`
}

// ScanEvent is a persisted and streamable scan progress event.
type ScanEvent struct {
	Sequence int64                  `json:"sequence"`
	ScanID   string                 `json:"scan_id"`
	Time     time.Time              `json:"time"`
	Type     string                 `json:"type"`
	Module   string                 `json:"module,omitempty"`
	NodeID   string                 `json:"node_id,omitempty"`
	EdgeID   string                 `json:"edge_id,omitempty"`
	Message  string                 `json:"message,omitempty"`
	Data     map[string]interface{} `json:"data,omitempty"`
}

// Normalize applies defaults to the request.
func (r *ScanRequest) Normalize() {
	if r.Depth <= 0 {
		r.Depth = 2
	}
	if r.Concurrency <= 0 {
		r.Concurrency = 5
	}
	if r.TimeoutSeconds <= 0 {
		r.TimeoutSeconds = 10
	}
	r.DisabledModules = dedupeStrings(r.DisabledModules)
}

func normalizeSettings(s Settings) Settings {
	s.DisabledModules = dedupeStrings(s.DisabledModules)
	return s
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
