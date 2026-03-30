// SPDX-License-Identifier: AGPL-3.0-or-later

package youtube

import (
	"context"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
	"github.com/kyle/basalt/internal/modules"
)

const youtubeURL = "https://www.youtube.com"

// Module extracts profile data from YouTube channel pages.
type Module struct {
	baseURL string
}

// New creates a YouTube module.
func New() *Module {
	return &Module{baseURL: youtubeURL}
}

func (m *Module) Name() string                  { return "youtube" }
func (m *Module) Description() string           { return "Extract profile data from YouTube channel pages" }
func (m *Module) CanHandle(nodeType string) bool { return nodeType == "username" }

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	username := node.Label
	url := fmt.Sprintf("%s/@%s", m.baseURL, username)

	resp, err := client.Do(ctx, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("youtube request: %w", err)
	}
	if resp.StatusCode == 404 {
		return nil, nil, nil
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("youtube returned %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(resp.Body))
	if err != nil {
		return nil, nil, fmt.Errorf("parsing youtube HTML: %w", err)
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	profileURL := url
	if canonical, exists := doc.Find(`link[rel="canonical"]`).Attr("href"); exists {
		profileURL = canonical
	}

	account := graph.NewAccountNode("youtube", username, profileURL, "youtube")
	account.Confidence = 0.85
	nodes = append(nodes, account)
	edges = append(edges, graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "youtube"))

	if title, exists := doc.Find(`meta[property="og:title"]`).Attr("content"); exists && title != "" {
		nameNode := graph.NewNode(graph.NodeTypeFullName, title, "youtube")
		nameNode.Confidence = 0.75
		nodes = append(nodes, nameNode)
		edges = append(edges, graph.NewEdge(0, account.ID, nameNode.ID, graph.EdgeTypeLinkedTo, "youtube"))
	}

	if image, exists := doc.Find(`meta[property="og:image"]`).Attr("content"); exists && image != "" {
		avatarNode := graph.NewNode(graph.NodeTypeAvatarURL, image, "youtube")
		avatarNode.Confidence = 0.85
		nodes = append(nodes, avatarNode)
		edges = append(edges, graph.NewEdge(0, account.ID, avatarNode.ID, graph.EdgeTypeLinkedTo, "youtube"))
	}

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	url := fmt.Sprintf("%s/@YouTube", m.baseURL)
	resp, err := client.Do(ctx, url, nil)
	if err != nil {
		return modules.Offline, fmt.Sprintf("youtube: %v", err)
	}
	if resp.StatusCode == 200 {
		return modules.Healthy, "youtube: OK"
	}
	return modules.Degraded, fmt.Sprintf("youtube: unexpected status %d", resp.StatusCode)
}
