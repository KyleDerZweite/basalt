// SPDX-License-Identifier: AGPL-3.0-or-later

package roblox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/KyleDerZweite/basalt/internal/graph"
	"github.com/KyleDerZweite/basalt/internal/httpclient"
	"github.com/KyleDerZweite/basalt/internal/modules"
)

const (
	defaultUsersBaseURL = "https://users.roblox.com"
	defaultThumbBaseURL = "https://thumbnails.roblox.com"
)

// Module extracts profile data from Roblox via username lookup.
type Module struct {
	usersBaseURL string
	thumbBaseURL string
}

// New creates a Roblox module.
func New() *Module {
	return &Module{
		usersBaseURL: defaultUsersBaseURL,
		thumbBaseURL: defaultThumbBaseURL,
	}
}

func (m *Module) Name() string                   { return "roblox" }
func (m *Module) Description() string            { return "Extract profile data from Roblox via username lookup" }
func (m *Module) CanHandle(nodeType string) bool { return nodeType == "username" }

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	username := node.Label

	// Step 1: Resolve username to user ID via POST
	userID, userInfo, err := m.resolveUsername(ctx, client, username)
	if err != nil {
		return nil, nil, err
	}
	if userID == 0 {
		// User not found
		return nil, nil, nil
	}

	// Step 2: Get full profile by ID
	profile, err := m.getProfile(ctx, client, userID)
	if err != nil {
		return nil, nil, err
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	// Create account node
	profileURL := fmt.Sprintf("https://www.roblox.com/users/%d/profile", userID)
	account := graph.NewAccountNode("roblox", username, profileURL, "roblox")
	account.Confidence = 0.90

	// Store properties
	if userInfo.DisplayName != "" {
		account.Properties["display_name"] = userInfo.DisplayName
	}
	if userInfo.Name != "" {
		account.Properties["name"] = userInfo.Name
	}
	if userInfo.HasVerifiedBadge {
		account.Properties["verified"] = "true"
	}
	if profile.Description != "" {
		account.Properties["description"] = profile.Description
	}
	if profile.Created != "" {
		account.Properties["created"] = profile.Created
	}
	if profile.IsBanned {
		account.Properties["banned"] = "true"
	}

	nodes = append(nodes, account)
	edges = append(edges, graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "roblox"))

	// Add display name as a separate node if it differs from username
	if userInfo.DisplayName != "" && userInfo.DisplayName != username {
		nameNode := graph.NewNode(graph.NodeTypeFullName, userInfo.DisplayName, "roblox")
		nameNode.Confidence = 0.85
		nodes = append(nodes, nameNode)
		edges = append(edges, graph.NewEdge(0, account.ID, nameNode.ID, graph.EdgeTypeLinkedTo, "roblox"))
	}

	return nodes, edges, nil
}

// resolveUsername performs a POST request to resolve username to user ID.
// Returns userID (0 if not found), user info, and error.
func (m *Module) resolveUsername(ctx context.Context, client *httpclient.Client, username string) (int64, *usernameResponse, error) {
	url := fmt.Sprintf("%s/v1/usernames/users", m.usersBaseURL)

	reqBody := map[string]interface{}{
		"usernames":          []string{username},
		"excludeBannedUsers": false,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return 0, nil, fmt.Errorf("marshaling request body: %w", err)
	}

	headers := map[string]string{"Content-Type": "application/json"}
	resp, err := client.DoRequest(ctx, "POST", url, bytes.NewReader(bodyBytes), headers)
	if err != nil {
		return 0, nil, fmt.Errorf("roblox username lookup: %w", err)
	}
	if resp.StatusCode != 200 {
		return 0, nil, fmt.Errorf("roblox username lookup returned %d", resp.StatusCode)
	}

	var result struct {
		Data []usernameResponse `json:"data"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &result); err != nil {
		return 0, nil, fmt.Errorf("parsing roblox username response: %w", err)
	}

	if len(result.Data) == 0 {
		// User not found
		return 0, nil, nil
	}

	userInfo := result.Data[0]
	return userInfo.ID, &userInfo, nil
}

// getProfile retrieves full profile information by user ID.
func (m *Module) getProfile(ctx context.Context, client *httpclient.Client, userID int64) (*profileResponse, error) {
	url := fmt.Sprintf("%s/v1/users/%d", m.usersBaseURL, userID)

	resp, err := client.Do(ctx, url, nil)
	if err != nil {
		return nil, fmt.Errorf("roblox profile request: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("roblox profile returned %d", resp.StatusCode)
	}

	var profile profileResponse
	if err := json.Unmarshal([]byte(resp.Body), &profile); err != nil {
		return nil, fmt.Errorf("parsing roblox profile response: %w", err)
	}

	return &profile, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	// Verify against the official Roblox account
	_, userInfo, err := m.resolveUsername(ctx, client, "roblox")
	if err != nil {
		return modules.Offline, fmt.Sprintf("roblox: %v", err)
	}
	if userInfo != nil && userInfo.ID > 0 {
		return modules.Healthy, "roblox: OK"
	}
	return modules.Degraded, "roblox: could not verify official account"
}

// usernameResponse represents the response data from username lookup.
type usernameResponse struct {
	RequestedUsername string `json:"requestedUsername"`
	HasVerifiedBadge  bool   `json:"hasVerifiedBadge"`
	ID                int64  `json:"id"`
	Name              string `json:"name"`
	DisplayName       string `json:"displayName"`
}

// profileResponse represents the full profile data from the user endpoint.
type profileResponse struct {
	Description            string      `json:"description"`
	Created                string      `json:"created"`
	IsBanned               bool        `json:"isBanned"`
	ExternalAppDisplayName interface{} `json:"externalAppDisplayName"`
	HasVerifiedBadge       bool        `json:"hasVerifiedBadge"`
	ID                     int64       `json:"id"`
	Name                   string      `json:"name"`
	DisplayName            string      `json:"displayName"`
}
