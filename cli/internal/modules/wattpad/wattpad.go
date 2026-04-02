// SPDX-License-Identifier: AGPL-3.0-or-later

package wattpad

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
	"github.com/kyle/basalt/internal/modules"
)

const wattpadBaseURL = "https://www.wattpad.com"

// Module extracts profile data from Wattpad's public API.
type Module struct {
	baseURL string
}

// New creates a Wattpad module.
func New() *Module {
	return &Module{baseURL: wattpadBaseURL}
}

func (m *Module) Name() string        { return "wattpad" }
func (m *Module) Description() string { return "Extract profile data from Wattpad" }

func (m *Module) CanHandle(nodeType string) bool {
	return nodeType == "username"
}

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	username := node.Label
	apiURL := fmt.Sprintf("%s/api/v3/users/%s", m.baseURL, username)

	resp, err := client.Do(ctx, apiURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("wattpad user request: %w", err)
	}
	if resp.StatusCode == 404 || resp.StatusCode == 400 {
		// Check if it's error_code:1014 (user not found)
		return nil, nil, nil
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("wattpad returned %d", resp.StatusCode)
	}

	var user struct {
		Username             string `json:"username"`
		Name                 string `json:"name"`
		Avatar               string `json:"avatar"`
		Description          string `json:"description"`
		Location             string `json:"location"`
		Gender               string `json:"gender"`
		GenderCode           string `json:"genderCode"`
		CreateDate           int64  `json:"createDate"`
		Verified             bool   `json:"verified"`
		Website              string `json:"website"`
		Facebook             string `json:"facebook"`
		NumFollowers         int    `json:"numFollowers"`
		NumFollowing         int    `json:"numFollowing"`
		NumStoriesPublished  int    `json:"numStoriesPublished"`
		VotesReceived        int    `json:"votesReceived"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &user); err != nil {
		return nil, nil, fmt.Errorf("parsing wattpad response: %w", err)
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	// Create account node
	profileURL := fmt.Sprintf("%s/user/%s", m.baseURL, username)
	account := graph.NewAccountNode("wattpad", user.Username, profileURL, "wattpad")
	account.Confidence = 0.90
	account.Properties["name"] = user.Name
	account.Properties["description"] = user.Description
	account.Properties["location"] = user.Location
	account.Properties["gender"] = user.Gender
	account.Properties["genderCode"] = user.GenderCode
	account.Properties["createDate"] = fmt.Sprintf("%d", user.CreateDate)
	account.Properties["verified"] = fmt.Sprintf("%v", user.Verified)
	account.Properties["numFollowers"] = fmt.Sprintf("%d", user.NumFollowers)
	account.Properties["numFollowing"] = fmt.Sprintf("%d", user.NumFollowing)
	account.Properties["numStoriesPublished"] = fmt.Sprintf("%d", user.NumStoriesPublished)
	account.Properties["votesReceived"] = fmt.Sprintf("%d", user.VotesReceived)

	if user.Avatar != "" {
		account.Properties["avatar"] = user.Avatar
	}

	nodes = append(nodes, account)
	edges = append(edges, graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "wattpad"))

	// Extract website if present
	if user.Website != "" {
		websiteNode := graph.NewNode(graph.NodeTypeWebsite, user.Website, "wattpad")
		websiteNode.Pivot = true
		websiteNode.Confidence = 0.85
		nodes = append(nodes, websiteNode)
		edges = append(edges, graph.NewEdge(0, account.ID, websiteNode.ID, graph.EdgeTypeLinkedTo, "wattpad"))
	}

	// Extract Facebook username if present
	if user.Facebook != "" {
		// Extract just the username from potential Facebook URL or direct username
		facebookUsername := extractFacebookUsername(user.Facebook)
		facebookNode := graph.NewNode(graph.NodeTypeUsername, facebookUsername, "wattpad")
		facebookNode.Pivot = true
		facebookNode.Confidence = 0.80
		facebookNode.Properties["platform_hint"] = "facebook"
		nodes = append(nodes, facebookNode)
		edges = append(edges, graph.NewEdge(0, account.ID, facebookNode.ID, graph.EdgeTypeHasUsername, "wattpad"))
	}

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	apiURL := fmt.Sprintf("%s/api/v3/users/wattpad", m.baseURL)
	resp, err := client.Do(ctx, apiURL, nil)
	if err != nil {
		return modules.Offline, fmt.Sprintf("wattpad: %v", err)
	}
	if resp.StatusCode == 200 {
		var user struct {
			Username string `json:"username"`
		}
		if err := json.Unmarshal([]byte(resp.Body), &user); err == nil && strings.EqualFold(user.Username, "wattpad") {
			return modules.Healthy, "wattpad: OK"
		}
		return modules.Degraded, "wattpad: unexpected response format"
	}
	return modules.Offline, fmt.Sprintf("wattpad: status %d", resp.StatusCode)
}

// extractFacebookUsername extracts a username from a Facebook URL or direct username.
// Examples: "https://facebook.com/john.doe" -> "john.doe", "john.doe" -> "john.doe"
func extractFacebookUsername(fbInput string) string {
	fbInput = strings.TrimSpace(fbInput)
	// If it's a URL, extract the last path component
	if strings.Contains(fbInput, "/") {
		parts := strings.Split(fbInput, "/")
		for i := len(parts) - 1; i >= 0; i-- {
			if parts[i] != "" {
				return parts[i]
			}
		}
	}
	return fbInput
}
