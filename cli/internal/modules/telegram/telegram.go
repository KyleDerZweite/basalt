// SPDX-License-Identifier: AGPL-3.0-or-later

package telegram

import (
	"context"
	"fmt"
	"strings"

	"github.com/KyleDerZweite/basalt/internal/graph"
	"github.com/KyleDerZweite/basalt/internal/httpclient"
	"github.com/KyleDerZweite/basalt/internal/modules"
	"github.com/PuerkitoBio/goquery"
)

const defaultBaseURL = "https://t.me"

// Module extracts profile data from Telegram public pages.
type Module struct {
	baseURL string
}

// New creates a Telegram module.
func New() *Module {
	return &Module{baseURL: defaultBaseURL}
}

func (m *Module) Name() string                   { return "telegram" }
func (m *Module) Description() string            { return "Extract profile data from Telegram" }
func (m *Module) CanHandle(nodeType string) bool { return nodeType == "username" }

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	username := node.Label
	url := fmt.Sprintf("%s/%s", m.baseURL, username)

	resp, err := client.Do(ctx, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("telegram request: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, nil, nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(resp.Body))
	if err != nil {
		return nil, nil, fmt.Errorf("parsing telegram HTML: %w", err)
	}

	title, _ := doc.Find(`meta[property="og:title"]`).Attr("content")
	desc, _ := doc.Find(`meta[property="og:description"]`).Attr("content")
	image, hasImage := doc.Find(`meta[property="og:image"]`).Attr("content")

	// Non-existent users get "Telegram: Contact @username" as title
	// and an empty description. Real profiles have a display name as title
	// and a non-empty description (bio), or a custom avatar.
	if strings.HasPrefix(title, "Telegram: Contact") && desc == "" {
		return nil, nil, nil
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	account := graph.NewAccountNode("telegram", username, url, "telegram")
	account.Confidence = 0.90

	if title != "" && !strings.HasPrefix(title, "Telegram: Contact") {
		account.Properties["display_name"] = title
	}
	if desc != "" {
		account.Properties["bio"] = desc
	}

	if hasImage && image != "" && !strings.Contains(image, "telegram.org/img/t_logo") {
		account.Properties["avatar_url"] = image
		avatarNode := graph.NewNode(graph.NodeTypeAvatarURL, image, "telegram")
		avatarNode.Confidence = 0.85
		avatarNode.Pivot = false
		nodes = append(nodes, avatarNode)
		edges = append(edges, graph.NewEdge(0, account.ID, avatarNode.ID, graph.EdgeTypeLinkedTo, "telegram"))
	}

	nodes = append([]*graph.Node{account}, nodes...)
	edges = append([]*graph.Edge{graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "telegram")}, edges...)

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	url := fmt.Sprintf("%s/telegram", m.baseURL)
	resp, err := client.Do(ctx, url, nil)
	if err != nil {
		return modules.Offline, fmt.Sprintf("telegram: %v", err)
	}
	if resp.StatusCode != 200 {
		return modules.Degraded, fmt.Sprintf("telegram: status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(resp.Body))
	if err != nil {
		return modules.Degraded, "telegram: failed to parse HTML"
	}
	title, _ := doc.Find(`meta[property="og:title"]`).Attr("content")
	if title != "" && !strings.HasPrefix(title, "Telegram: Contact") {
		return modules.Healthy, "telegram: OK"
	}
	return modules.Degraded, "telegram: unexpected response"
}
