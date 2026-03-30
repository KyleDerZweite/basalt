// SPDX-License-Identifier: AGPL-3.0-or-later

package twitch

import (
	"context"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
	"github.com/kyle/basalt/internal/modules"
)

const twitchURL = "https://www.twitch.tv"

// Module extracts profile data from Twitch channel pages.
type Module struct {
	baseURL string
}

// New creates a Twitch module.
func New() *Module {
	return &Module{baseURL: twitchURL}
}

func (m *Module) Name() string                  { return "twitch" }
func (m *Module) Description() string           { return "Extract profile data from Twitch channel pages" }
func (m *Module) CanHandle(nodeType string) bool { return nodeType == "username" }

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	username := node.Label
	url := fmt.Sprintf("%s/%s", m.baseURL, username)

	resp, err := client.Do(ctx, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("twitch request: %w", err)
	}
	if resp.StatusCode == 404 {
		return nil, nil, nil
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("twitch returned %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(resp.Body))
	if err != nil {
		return nil, nil, fmt.Errorf("parsing twitch HTML: %w", err)
	}

	title, hasTitle := doc.Find(`meta[property="og:title"]`).Attr("content")

	// Twitch returns 200 for non-existent users with og:title="Twitch".
	// A real profile has "{Username} - Twitch" in the title.
	if !hasTitle || !strings.Contains(title, " - ") {
		return nil, nil, nil
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	profileURL := url
	account := graph.NewAccountNode("twitch", username, profileURL, "twitch")
	account.Confidence = 0.85
	nodes = append(nodes, account)
	edges = append(edges, graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "twitch"))

	nameNode := graph.NewNode(graph.NodeTypeFullName, title, "twitch")
	nameNode.Confidence = 0.75
	nodes = append(nodes, nameNode)
	edges = append(edges, graph.NewEdge(0, account.ID, nameNode.ID, graph.EdgeTypeLinkedTo, "twitch"))

	if image, exists := doc.Find(`meta[property="og:image"]`).Attr("content"); exists && image != "" {
		avatarNode := graph.NewNode(graph.NodeTypeAvatarURL, image, "twitch")
		avatarNode.Confidence = 0.85
		nodes = append(nodes, avatarNode)
		edges = append(edges, graph.NewEdge(0, account.ID, avatarNode.ID, graph.EdgeTypeLinkedTo, "twitch"))
	}

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	url := fmt.Sprintf("%s/twitch", m.baseURL)
	resp, err := client.Do(ctx, url, nil)
	if err != nil {
		return modules.Offline, fmt.Sprintf("twitch: %v", err)
	}
	if resp.StatusCode == 200 {
		return modules.Healthy, "twitch: OK"
	}
	return modules.Degraded, fmt.Sprintf("twitch: unexpected status %d", resp.StatusCode)
}
