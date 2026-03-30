// SPDX-License-Identifier: AGPL-3.0-or-later

package matrix

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
	"github.com/kyle/basalt/internal/modules"
)

const matrixAPI = "https://matrix.org"

// Module extracts profile data from Matrix homeservers.
type Module struct {
	baseURL string
}

// New creates a Matrix module.
func New() *Module {
	return &Module{baseURL: matrixAPI}
}

func (m *Module) Name() string                  { return "matrix" }
func (m *Module) Description() string            { return "Extract profile data from Matrix" }
func (m *Module) CanHandle(nodeType string) bool { return nodeType == "username" }

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	username := node.Label
	homeserver := "matrix.org"

	// If the username contains ":", treat it as @user:homeserver.
	if idx := strings.Index(username, ":"); idx >= 0 {
		homeserver = username[idx+1:]
		username = username[:idx]
	}

	// Strip leading "@" if present.
	username = strings.TrimPrefix(username, "@")

	matrixID := fmt.Sprintf("@%s:%s", username, homeserver)
	apiURL := fmt.Sprintf("%s/_matrix/client/v3/profile/%s", m.baseURL, matrixID)

	resp, err := client.Do(ctx, apiURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("matrix request: %w", err)
	}
	if resp.StatusCode == 404 {
		return nil, nil, nil
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("matrix returned %d", resp.StatusCode)
	}

	var profile struct {
		DisplayName string `json:"displayname"`
		AvatarURL   string `json:"avatar_url"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &profile); err != nil {
		return nil, nil, fmt.Errorf("parsing matrix response: %w", err)
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	account := graph.NewAccountNode("matrix", matrixID, fmt.Sprintf("https://matrix.to/#/%s", matrixID), "matrix")
	account.Confidence = 0.85
	nodes = append(nodes, account)
	edges = append(edges, graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "matrix"))

	if profile.DisplayName != "" {
		nameNode := graph.NewNode(graph.NodeTypeFullName, profile.DisplayName, "matrix")
		nameNode.Confidence = 0.80
		nodes = append(nodes, nameNode)
		edges = append(edges, graph.NewEdge(0, account.ID, nameNode.ID, graph.EdgeTypeLinkedTo, "matrix"))
	}

	if profile.AvatarURL != "" {
		httpURL := mxcToHTTP(profile.AvatarURL)
		if httpURL != "" {
			avatarNode := graph.NewNode(graph.NodeTypeAvatarURL, httpURL, "matrix")
			avatarNode.Confidence = 0.85
			nodes = append(nodes, avatarNode)
			edges = append(edges, graph.NewEdge(0, account.ID, avatarNode.ID, graph.EdgeTypeLinkedTo, "matrix"))
		}
	}

	return nodes, edges, nil
}

// mxcToHTTP converts an mxc:// URL to an HTTPS download URL.
// Format: mxc://server/media_id -> https://matrix.org/_matrix/media/v3/download/server/media_id
func mxcToHTTP(mxcURL string) string {
	if !strings.HasPrefix(mxcURL, "mxc://") {
		return mxcURL
	}
	path := strings.TrimPrefix(mxcURL, "mxc://")
	return fmt.Sprintf("https://matrix.org/_matrix/media/v3/download/%s", path)
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	apiURL := fmt.Sprintf("%s/_matrix/client/v3/profile/@alice:matrix.org", m.baseURL)
	resp, err := client.Do(ctx, apiURL, nil)
	if err != nil {
		return modules.Offline, fmt.Sprintf("matrix: %v", err)
	}
	// Both 200 (found) and 404 (not found) mean the API is responding.
	if resp.StatusCode == 200 || resp.StatusCode == 404 {
		return modules.Healthy, "matrix: OK"
	}
	return modules.Degraded, fmt.Sprintf("matrix: status %d", resp.StatusCode)
}
