// SPDX-License-Identifier: AGPL-3.0-or-later

package instagram

import (
	"context"
	"fmt"
	"strings"

	"github.com/KyleDerZweite/basalt/internal/graph"
	"github.com/KyleDerZweite/basalt/internal/httpclient"
	"github.com/KyleDerZweite/basalt/internal/modules"
	"github.com/PuerkitoBio/goquery"
)

const instagramURL = "https://www.instagram.com"

// Module extracts profile data from Instagram profile pages.
type Module struct {
	baseURL string
}

// New creates an Instagram module.
func New() *Module {
	return &Module{baseURL: instagramURL}
}

func (m *Module) Name() string                   { return "instagram" }
func (m *Module) Description() string            { return "Extract profile data from Instagram profile pages" }
func (m *Module) CanHandle(nodeType string) bool { return nodeType == "username" }

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	username := node.Label
	url := fmt.Sprintf("%s/%s/", m.baseURL, username)

	resp, err := client.Do(ctx, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("instagram request: %w", err)
	}
	if resp.StatusCode == 404 {
		return nil, nil, nil
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("instagram returned %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(resp.Body))
	if err != nil {
		return nil, nil, fmt.Errorf("parsing instagram HTML: %w", err)
	}

	title, hasTitle := doc.Find(`meta[property="og:title"]`).Attr("content")
	image, hasImage := doc.Find(`meta[property="og:image"]`).Attr("content")

	// Instagram returns 200 for login walls with no og: metadata.
	// Require at least one og: signal to confirm the profile exists.
	if !hasTitle && !hasImage {
		return nil, nil, nil
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	profileURL := url
	account := graph.NewAccountNode("instagram", username, profileURL, "instagram")
	account.Confidence = 0.85
	nodes = append(nodes, account)
	edges = append(edges, graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "instagram"))

	if hasTitle && title != "" {
		nameNode := graph.NewNode(graph.NodeTypeFullName, title, "instagram")
		nameNode.Confidence = 0.75
		nodes = append(nodes, nameNode)
		edges = append(edges, graph.NewEdge(0, account.ID, nameNode.ID, graph.EdgeTypeLinkedTo, "instagram"))
	}

	if hasImage && image != "" {
		avatarNode := graph.NewNode(graph.NodeTypeAvatarURL, image, "instagram")
		avatarNode.Confidence = 0.85
		nodes = append(nodes, avatarNode)
		edges = append(edges, graph.NewEdge(0, account.ID, avatarNode.ID, graph.EdgeTypeLinkedTo, "instagram"))
	}

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	url := fmt.Sprintf("%s/instagram/", m.baseURL)
	resp, err := client.Do(ctx, url, nil)
	if err != nil {
		return modules.Offline, fmt.Sprintf("instagram: %v", err)
	}
	if resp.StatusCode == 200 {
		return modules.Healthy, "instagram: OK"
	}
	return modules.Degraded, fmt.Sprintf("instagram: unexpected status %d", resp.StatusCode)
}
