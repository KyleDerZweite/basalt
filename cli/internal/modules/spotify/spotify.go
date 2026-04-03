// SPDX-License-Identifier: AGPL-3.0-or-later

package spotify

import (
	"context"
	"fmt"
	"strings"

	"github.com/KyleDerZweite/basalt/internal/graph"
	"github.com/KyleDerZweite/basalt/internal/httpclient"
	"github.com/KyleDerZweite/basalt/internal/modules"
	"github.com/PuerkitoBio/goquery"
)

const defaultBaseURL = "https://open.spotify.com"

// Module extracts profile data from Spotify user pages.
type Module struct {
	baseURL string
}

// New creates a Spotify module.
func New() *Module {
	return &Module{baseURL: defaultBaseURL}
}

func (m *Module) Name() string                   { return "spotify" }
func (m *Module) Description() string            { return "Extract profile data from Spotify" }
func (m *Module) CanHandle(nodeType string) bool { return nodeType == "username" }

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	username := node.Label
	url := fmt.Sprintf("%s/user/%s", m.baseURL, username)

	resp, err := client.Do(ctx, url, spotifyHeaders())
	if err != nil {
		return nil, nil, fmt.Errorf("spotify request: %w", err)
	}
	if resp.StatusCode == 404 {
		return nil, nil, nil
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("spotify returned %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(resp.Body))
	if err != nil {
		return nil, nil, fmt.Errorf("parsing spotify HTML: %w", err)
	}

	title, hasTitle := doc.Find(`meta[property="og:title"]`).Attr("content")
	if !hasTitle || !isProfilePage(doc) {
		return nil, nil, nil
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	account := graph.NewAccountNode("spotify", username, url, "spotify")
	account.Confidence = 0.90

	if title != "" {
		account.Properties["display_name"] = title
	}
	if desc, exists := doc.Find(`meta[property="og:description"]`).Attr("content"); exists && desc != "" {
		account.Properties["description"] = desc
	}
	if image, exists := doc.Find(`meta[property="og:image"]`).Attr("content"); exists && image != "" {
		account.Properties["avatar_url"] = image
		avatarNode := graph.NewNode(graph.NodeTypeAvatarURL, image, "spotify")
		avatarNode.Confidence = 0.85
		avatarNode.Pivot = false
		nodes = append(nodes, avatarNode)
		edges = append(edges, graph.NewEdge(0, account.ID, avatarNode.ID, graph.EdgeTypeLinkedTo, "spotify"))
	}

	nodes = append([]*graph.Node{account}, nodes...)
	edges = append([]*graph.Edge{graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "spotify")}, edges...)

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	url := fmt.Sprintf("%s/user/spotify", m.baseURL)
	resp, err := client.Do(ctx, url, spotifyHeaders())
	if err != nil {
		return modules.Offline, fmt.Sprintf("spotify: %v", err)
	}
	if resp.StatusCode != 200 {
		return modules.Degraded, fmt.Sprintf("spotify: status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(resp.Body))
	if err != nil {
		return modules.Degraded, "spotify: failed to parse HTML"
	}
	if isProfilePage(doc) {
		return modules.Healthy, "spotify: OK"
	}
	return modules.Degraded, "spotify: unexpected response"
}

func isProfilePage(doc *goquery.Document) bool {
	if ogType, _ := doc.Find(`meta[property="og:type"]`).Attr("content"); ogType == "profile" {
		return true
	}

	title, hasTitle := doc.Find(`meta[property="og:title"]`).Attr("content")
	desc, _ := doc.Find(`meta[property="og:description"]`).Attr("content")
	canonical, _ := doc.Find(`link[rel="canonical"]`).Attr("href")

	return hasTitle && title != "" && strings.Contains(desc, "User") && strings.Contains(canonical, "/user/")
}

func spotifyHeaders() map[string]string {
	return map[string]string{
		"Accept":     "*/*",
		"User-Agent": "basalt/2.0",
	}
}
