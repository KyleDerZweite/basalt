// SPDX-License-Identifier: AGPL-3.0-or-later

package wayback

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
	"github.com/kyle/basalt/internal/modules"
)

const defaultBaseURL = "https://archive.org"

// Module checks the Wayback Machine for archived snapshots of a domain.
type Module struct {
	baseURL string
}

// New creates a Wayback Machine module.
func New() *Module {
	return &Module{baseURL: defaultBaseURL}
}

func (m *Module) Name() string        { return "wayback" }
func (m *Module) Description() string { return "Domain history via the Wayback Machine" }

func (m *Module) CanHandle(nodeType string) bool {
	return nodeType == "domain"
}

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	domain := node.Label

	apiURL := fmt.Sprintf("%s/wayback/available?url=%s", m.baseURL, url.QueryEscape(domain))

	resp, err := client.Do(ctx, apiURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("wayback availability request: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("wayback returned %d", resp.StatusCode)
	}

	var payload struct {
		URL               string `json:"url"`
		ArchivedSnapshots struct {
			Closest *struct {
				URL       string `json:"url"`
				Timestamp string `json:"timestamp"`
				Status    string `json:"status"`
				Available bool   `json:"available"`
			} `json:"closest"`
		} `json:"archived_snapshots"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &payload); err != nil {
		return nil, nil, fmt.Errorf("parsing wayback response: %w", err)
	}

	closest := payload.ArchivedSnapshots.Closest
	if closest == nil || !closest.Available {
		return nil, nil, nil
	}

	account := graph.NewAccountNode("wayback", domain, closest.URL, "wayback")
	account.Confidence = 0.85
	account.Properties["snapshot_url"] = closest.URL
	account.Properties["timestamp"] = closest.Timestamp
	account.Properties["archive_url"] = closest.URL

	if t, err := time.Parse("20060102150405", closest.Timestamp); err == nil {
		account.Properties["first_seen"] = fmt.Sprintf("%d", t.Year())
	}

	edge := graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "wayback")

	return []*graph.Node{account}, []*graph.Edge{edge}, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	apiURL := fmt.Sprintf("%s/wayback/available?url=%s", m.baseURL, url.QueryEscape("example.com"))

	resp, err := client.Do(ctx, apiURL, nil)
	if err != nil {
		return modules.Offline, fmt.Sprintf("wayback: %v", err)
	}
	if resp.StatusCode != 200 {
		return modules.Offline, fmt.Sprintf("wayback: status %d", resp.StatusCode)
	}

	var payload struct {
		ArchivedSnapshots struct {
			Closest *struct {
				Available bool `json:"available"`
			} `json:"closest"`
		} `json:"archived_snapshots"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &payload); err != nil {
		return modules.Degraded, fmt.Sprintf("wayback: invalid JSON: %v", err)
	}

	if payload.ArchivedSnapshots.Closest == nil || !payload.ArchivedSnapshots.Closest.Available {
		return modules.Degraded, "wayback: example.com not archived (unexpected)"
	}

	return modules.Healthy, "wayback: OK"
}
