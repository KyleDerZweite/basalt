// SPDX-License-Identifier: AGPL-3.0-or-later

package tiktok

import (
	"context"
	"fmt"
	"strings"

	"github.com/KyleDerZweite/basalt/internal/graph"
	"github.com/KyleDerZweite/basalt/internal/httpclient"
	"github.com/KyleDerZweite/basalt/internal/modules"
	"github.com/PuerkitoBio/goquery"
)

const tiktokURL = "https://www.tiktok.com"

// Module extracts profile data from TikTok profile pages.
type Module struct {
	baseURL string
}

// New creates a TikTok module.
func New() *Module {
	return &Module{baseURL: tiktokURL}
}

func (m *Module) Name() string                   { return "tiktok" }
func (m *Module) Description() string            { return "Extract profile data from TikTok profile pages" }
func (m *Module) CanHandle(nodeType string) bool { return nodeType == "username" }

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	username := node.Label
	url := fmt.Sprintf("%s/@%s", m.baseURL, username)

	resp, err := client.Do(ctx, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("tiktok request: %w", err)
	}
	if resp.StatusCode == 404 {
		return nil, nil, nil
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("tiktok returned %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(resp.Body))
	if err != nil {
		return nil, nil, fmt.Errorf("parsing tiktok HTML: %w", err)
	}

	title, hasTitle := doc.Find(`meta[property="og:title"]`).Attr("content")
	image, hasImage := doc.Find(`meta[property="og:image"]`).Attr("content")

	// TikTok returns 200 for non-existent profiles with no og: metadata.
	// Require at least one og: signal to confirm the profile exists.
	if !hasTitle && !hasImage {
		return nil, nil, nil
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	profileURL := url
	if canonical, exists := doc.Find(`link[rel="canonical"]`).Attr("href"); exists {
		profileURL = canonical
	}

	account := graph.NewAccountNode("tiktok", username, profileURL, "tiktok")
	account.Confidence = 0.85
	nodes = append(nodes, account)
	edges = append(edges, graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "tiktok"))

	if hasTitle && title != "" {
		nameNode := graph.NewNode(graph.NodeTypeFullName, title, "tiktok")
		nameNode.Confidence = 0.75
		nodes = append(nodes, nameNode)
		edges = append(edges, graph.NewEdge(0, account.ID, nameNode.ID, graph.EdgeTypeLinkedTo, "tiktok"))
	}

	if hasImage && image != "" {
		avatarNode := graph.NewNode(graph.NodeTypeAvatarURL, image, "tiktok")
		avatarNode.Confidence = 0.85
		nodes = append(nodes, avatarNode)
		edges = append(edges, graph.NewEdge(0, account.ID, avatarNode.ID, graph.EdgeTypeLinkedTo, "tiktok"))
	}

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	url := fmt.Sprintf("%s/@tiktok", m.baseURL)
	resp, err := client.Do(ctx, url, nil)
	if err != nil {
		return modules.Offline, fmt.Sprintf("tiktok: %v", err)
	}
	if resp.StatusCode == 200 {
		return modules.Healthy, "tiktok: OK"
	}
	return modules.Degraded, fmt.Sprintf("tiktok: unexpected status %d", resp.StatusCode)
}
