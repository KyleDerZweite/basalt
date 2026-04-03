// SPDX-License-Identifier: AGPL-3.0-or-later

package steam

import (
	"context"
	"fmt"
	"strings"

	"github.com/KyleDerZweite/basalt/internal/graph"
	"github.com/KyleDerZweite/basalt/internal/httpclient"
	"github.com/KyleDerZweite/basalt/internal/modules"
	"github.com/PuerkitoBio/goquery"
)

const defaultBaseURL = "https://steamcommunity.com"

// Module extracts profile data from Steam community pages.
type Module struct {
	baseURL string
}

// New creates a Steam module. No API key required.
func New() *Module {
	return &Module{baseURL: defaultBaseURL}
}

func (m *Module) Name() string                   { return "steam" }
func (m *Module) Description() string            { return "Extract profile data from Steam community pages" }
func (m *Module) CanHandle(nodeType string) bool { return nodeType == "username" }

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	username := node.Label
	url := fmt.Sprintf("%s/id/%s/", m.baseURL, username)

	resp, err := client.Do(ctx, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("steam request: %w", err)
	}
	if resp.StatusCode == 404 {
		return nil, nil, nil
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("steam returned %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(resp.Body))
	if err != nil {
		return nil, nil, fmt.Errorf("parsing steam HTML: %w", err)
	}

	title, _ := doc.Find(`meta[property="og:title"]`).Attr("content")
	// Non-existent profiles return "Steam Community :: Error".
	if title == "" || strings.Contains(title, ":: Error") {
		return nil, nil, nil
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	account := graph.NewAccountNode("steam", username, url, "steam")
	account.Confidence = 0.90

	// Title format: "Steam Community :: DisplayName"
	if parts := strings.SplitN(title, " :: ", 2); len(parts) == 2 && parts[1] != "" {
		account.Properties["persona_name"] = parts[1]
	}

	if desc, exists := doc.Find(`meta[property="og:description"]`).Attr("content"); exists && desc != "" {
		account.Properties["description"] = desc
	}

	if image, exists := doc.Find(`meta[property="og:image"]`).Attr("content"); exists && image != "" {
		account.Properties["avatar_url"] = image
		avatarNode := graph.NewNode(graph.NodeTypeAvatarURL, image, "steam")
		avatarNode.Confidence = 0.90
		avatarNode.Pivot = false
		nodes = append(nodes, avatarNode)
		edges = append(edges, graph.NewEdge(0, account.ID, avatarNode.ID, graph.EdgeTypeLinkedTo, "steam"))
	}

	nodes = append([]*graph.Node{account}, nodes...)
	edges = append([]*graph.Edge{graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "steam")}, edges...)

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	url := fmt.Sprintf("%s/id/valve/", m.baseURL)
	resp, err := client.Do(ctx, url, nil)
	if err != nil {
		return modules.Offline, fmt.Sprintf("steam: %v", err)
	}
	if resp.StatusCode != 200 {
		return modules.Degraded, fmt.Sprintf("steam: status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(resp.Body))
	if err != nil {
		return modules.Degraded, "steam: failed to parse HTML"
	}

	title, _ := doc.Find(`meta[property="og:title"]`).Attr("content")
	if strings.Contains(title, "Steam Community ::") && !strings.Contains(title, ":: Error") {
		return modules.Healthy, "steam: OK"
	}
	return modules.Degraded, "steam: unexpected response"
}
