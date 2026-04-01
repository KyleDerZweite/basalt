// SPDX-License-Identifier: AGPL-3.0-or-later

package medium

import (
	"context"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
	"github.com/kyle/basalt/internal/modules"
)

const defaultBaseURL = "https://medium.com"

// Module extracts profile data from Medium user pages.
type Module struct {
	baseURL string
}

// New creates a Medium module.
func New() *Module {
	return &Module{baseURL: defaultBaseURL}
}

func (m *Module) Name() string                   { return "medium" }
func (m *Module) Description() string            { return "Extract profile data from Medium" }
func (m *Module) CanHandle(nodeType string) bool { return nodeType == "username" }

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	username := node.Label
	url := fmt.Sprintf("%s/@%s", m.baseURL, username)

	// Medium blocks default browser UAs but allows simple ones.
	resp, err := client.Do(ctx, url, map[string]string{
		"User-Agent": "basalt/2.0",
	})
	if err != nil {
		return nil, nil, fmt.Errorf("medium request: %w", err)
	}
	if resp.StatusCode == 404 {
		return nil, nil, nil
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("medium returned %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(resp.Body))
	if err != nil {
		return nil, nil, fmt.Errorf("parsing medium HTML: %w", err)
	}

	title, hasTitle := doc.Find(`meta[property="og:title"]`).Attr("content")
	// Non-existent users return 200 but only have og:site_name, no og:title.
	if !hasTitle || title == "" {
		return nil, nil, nil
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	account := graph.NewAccountNode("medium", username, url, "medium")
	account.Confidence = 0.90

	// Title format: "DisplayName – Medium"
	displayName := strings.TrimSuffix(title, " – Medium")
	if displayName != "" {
		account.Properties["display_name"] = displayName
	}
	if desc, exists := doc.Find(`meta[property="og:description"]`).Attr("content"); exists && desc != "" {
		account.Properties["bio"] = desc
	}
	if image, exists := doc.Find(`meta[property="og:image"]`).Attr("content"); exists && image != "" {
		account.Properties["avatar_url"] = image
		avatarNode := graph.NewNode(graph.NodeTypeAvatarURL, image, "medium")
		avatarNode.Confidence = 0.85
		avatarNode.Pivot = false
		nodes = append(nodes, avatarNode)
		edges = append(edges, graph.NewEdge(0, account.ID, avatarNode.ID, graph.EdgeTypeLinkedTo, "medium"))
	}

	nodes = append([]*graph.Node{account}, nodes...)
	edges = append([]*graph.Edge{graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "medium")}, edges...)

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	url := fmt.Sprintf("%s/@ev", m.baseURL)
	resp, err := client.Do(ctx, url, map[string]string{
		"User-Agent": "basalt/2.0",
	})
	if err != nil {
		return modules.Offline, fmt.Sprintf("medium: %v", err)
	}
	if resp.StatusCode != 200 {
		return modules.Degraded, fmt.Sprintf("medium: status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(resp.Body))
	if err != nil {
		return modules.Degraded, "medium: failed to parse HTML"
	}
	if title, exists := doc.Find(`meta[property="og:title"]`).Attr("content"); exists && title != "" {
		return modules.Healthy, "medium: OK"
	}
	return modules.Degraded, "medium: unexpected response"
}
