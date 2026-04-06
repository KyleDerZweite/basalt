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

// Target represents a persisted real-world person or subject.
type Target struct {
	ID          string        `json:"id"`
	Slug        string        `json:"slug"`
	DisplayName string        `json:"display_name"`
	Notes       string        `json:"notes,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	Aliases     []TargetAlias `json:"aliases,omitempty"`
}

// TargetAlias associates a scannable identifier with a target.
type TargetAlias struct {
	ID        string    `json:"id"`
	TargetID  string    `json:"target_id"`
	SeedType  string    `json:"seed_type"`
	SeedValue string    `json:"seed_value"`
	Label     string    `json:"label,omitempty"`
	IsPrimary bool      `json:"is_primary,omitempty"`
	CreatedAt time.Time `json:"created_at"`
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
	TargetRef       string       `json:"target_ref,omitempty"`
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
	TargetID     string         `json:"target_id,omitempty"`
	Status       ScanStatus     `json:"status"`
	StartedAt    time.Time      `json:"started_at"`
	CompletedAt  *time.Time     `json:"completed_at,omitempty"`
	UpdatedAt    time.Time      `json:"updated_at"`
	Seeds        []graph.Seed   `json:"seeds"`
	Options      ScanRequest    `json:"options"`
	Health       []ModuleStatus `json:"health,omitempty"`
	Insights     *ScanInsights  `json:"insights,omitempty"`
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

// InsightFinding is a single summary bullet for a scan.
type InsightFinding struct {
	Title       string   `json:"title"`
	Summary     string   `json:"summary"`
	NodeIDs     []string `json:"node_ids,omitempty"`
	ProfileURL  string   `json:"profile_url,omitempty"`
	Confidence  float64  `json:"confidence,omitempty"`
	Category    string   `json:"category,omitempty"`
	SourceLabel string   `json:"source_label,omitempty"`
}

// ScanInsights summarizes the most important findings in a scan.
type ScanInsights struct {
	Headline               string           `json:"headline"`
	TopFindings            []InsightFinding `json:"top_findings,omitempty"`
	HighConfidenceAccounts []InsightFinding `json:"high_confidence_accounts,omitempty"`
	IdentitySignals        []string         `json:"identity_signals,omitempty"`
	InfrastructureSummary  []string         `json:"infrastructure_summary,omitempty"`
	Warnings               []string         `json:"warnings,omitempty"`
}

// WorkspaceNode is a node in the synthesized UI graph.
type WorkspaceNode struct {
	ID             string   `json:"id"`
	Label          string   `json:"label"`
	Type           string   `json:"type"`
	Category       string   `json:"category"`
	Depth          int      `json:"depth,omitempty"`
	RawNodeIDs     []string `json:"raw_node_ids,omitempty"`
	RawEdgeIDs     []string `json:"raw_edge_ids,omitempty"`
	ProfileURL     string   `json:"profile_url,omitempty"`
	Confidence     float64  `json:"confidence,omitempty"`
	CollapsedCount int      `json:"collapsed_count,omitempty"`
}

// WorkspaceEdge is an edge in the synthesized UI graph.
type WorkspaceEdge struct {
	ID         string   `json:"id"`
	Source     string   `json:"source"`
	Target     string   `json:"target"`
	Type       string   `json:"type"`
	RawEdgeIDs []string `json:"raw_edge_ids,omitempty"`
}

// WorkspaceGraph is the target-centric graph shown in the UI.
type WorkspaceGraph struct {
	Layout string          `json:"layout"`
	Nodes  []WorkspaceNode `json:"nodes"`
	Edges  []WorkspaceEdge `json:"edges"`
}

// ScanWorkspace is the target-centric workspace payload for the UI.
type ScanWorkspace struct {
	Record            *ScanRecord    `json:"record"`
	Target            *Target        `json:"target,omitempty"`
	Insights          *ScanInsights  `json:"insights,omitempty"`
	Graph             WorkspaceGraph `json:"graph"`
	RawGraphAvailable bool           `json:"raw_graph_available"`
	RawNodeCount      int            `json:"raw_node_count"`
	RawEdgeCount      int            `json:"raw_edge_count"`
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
