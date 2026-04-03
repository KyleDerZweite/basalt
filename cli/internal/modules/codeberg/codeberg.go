// SPDX-License-Identifier: AGPL-3.0-or-later

package codeberg

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/KyleDerZweite/basalt/internal/graph"
	"github.com/KyleDerZweite/basalt/internal/httpclient"
	"github.com/KyleDerZweite/basalt/internal/modules"
)

const codebergBaseURL = "https://codeberg.org"

// Module extracts profile data from Codeberg.
type Module struct {
	baseURL string
}

// New creates a Codeberg module.
func New() *Module {
	return &Module{baseURL: codebergBaseURL}
}

func (m *Module) Name() string        { return "codeberg" }
func (m *Module) Description() string { return "Extract profile data from Codeberg" }

func (m *Module) CanHandle(nodeType string) bool {
	return nodeType == "username"
}

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	apiURL := fmt.Sprintf("%s/api/v1/users/%s", m.baseURL, url.PathEscape(node.Label))

	resp, err := client.Do(ctx, apiURL, map[string]string{"Accept": "application/json"})
	if err != nil {
		return nil, nil, fmt.Errorf("codeberg request: %w", err)
	}
	if resp.StatusCode == 404 {
		return nil, nil, nil
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("codeberg returned %d", resp.StatusCode)
	}

	var user struct {
		Login          string `json:"login"`
		FullName       string `json:"full_name"`
		AvatarURL      string `json:"avatar_url"`
		HTMLURL        string `json:"html_url"`
		Website        string `json:"website"`
		Location       string `json:"location"`
		Description    string `json:"description"`
		Created        string `json:"created"`
		FollowersCount int    `json:"followers_count"`
		FollowingCount int    `json:"following_count"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &user); err != nil {
		return nil, nil, fmt.Errorf("parsing codeberg response: %w", err)
	}
	if user.Login == "" {
		return nil, nil, nil
	}
	profileURL := user.HTMLURL
	if profileURL == "" {
		profileURL = fmt.Sprintf("%s/%s", m.baseURL, url.PathEscape(user.Login))
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	account := graph.NewAccountNode("codeberg", user.Login, profileURL, "codeberg")
	account.Confidence = 0.90
	if user.Location != "" {
		account.Properties["location"] = user.Location
	}
	if user.Description != "" {
		account.Properties["description"] = user.Description
	}
	if user.Created != "" {
		account.Properties["created"] = user.Created
	}
	if user.FollowersCount > 0 {
		account.Properties["followers_count"] = user.FollowersCount
	}
	if user.FollowingCount > 0 {
		account.Properties["following_count"] = user.FollowingCount
	}
	nodes = append(nodes, account)
	edges = append(edges, graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "codeberg"))

	if user.FullName != "" {
		nameNode := graph.NewNode(graph.NodeTypeFullName, user.FullName, "codeberg")
		nameNode.Confidence = 0.85
		nodes = append(nodes, nameNode)
		edges = append(edges, graph.NewEdge(0, account.ID, nameNode.ID, graph.EdgeTypeLinkedTo, "codeberg"))
	}

	if user.AvatarURL != "" {
		avatarNode := graph.NewNode(graph.NodeTypeAvatarURL, user.AvatarURL, "codeberg")
		avatarNode.Confidence = 0.90
		nodes = append(nodes, avatarNode)
		edges = append(edges, graph.NewEdge(0, account.ID, avatarNode.ID, graph.EdgeTypeLinkedTo, "codeberg"))
	}

	website := normalizeWebsite(user.Website)
	if website != "" {
		websiteNode := graph.NewNode(graph.NodeTypeWebsite, website, "codeberg")
		websiteNode.Pivot = true
		websiteNode.Confidence = 0.85
		nodes = append(nodes, websiteNode)
		edges = append(edges, graph.NewEdge(0, account.ID, websiteNode.ID, graph.EdgeTypeHasDomain, "codeberg"))

		if parsed, err := url.Parse(website); err == nil && parsed.Hostname() != "" {
			domainNode := graph.NewNode(graph.NodeTypeDomain, parsed.Hostname(), "codeberg")
			domainNode.Pivot = true
			domainNode.Confidence = 0.85
			nodes = append(nodes, domainNode)
			edges = append(edges, graph.NewEdge(0, account.ID, domainNode.ID, graph.EdgeTypeHasDomain, "codeberg"))
		}
	}

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	apiURL := fmt.Sprintf("%s/api/v1/users/forgejo", m.baseURL)

	resp, err := client.Do(ctx, apiURL, map[string]string{"Accept": "application/json"})
	if err != nil {
		return modules.Offline, fmt.Sprintf("codeberg: %v", err)
	}
	if resp.StatusCode != 200 {
		return modules.Offline, fmt.Sprintf("codeberg: status %d", resp.StatusCode)
	}

	var user struct {
		Login string `json:"login"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &user); err == nil && user.Login == "forgejo" {
		return modules.Healthy, "codeberg: OK"
	}
	return modules.Degraded, "codeberg: unexpected response format"
}

func normalizeWebsite(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	if !strings.Contains(value, "://") {
		value = "https://" + value
	}

	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	return parsed.String()
}
