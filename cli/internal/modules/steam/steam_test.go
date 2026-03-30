// SPDX-License-Identifier: AGPL-3.0-or-later

package steam

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
	"github.com/kyle/basalt/internal/modules"
)

func TestCanHandle(t *testing.T) {
	m := New("testkey")
	if !m.CanHandle("username") {
		t.Error("should handle username")
	}
	if m.CanHandle("email") {
		t.Error("should not handle email")
	}
}

func TestVerifyOfflineWithoutKey(t *testing.T) {
	m := New("")
	client := httpclient.New()
	status, msg := m.Verify(context.Background(), client)
	if status != modules.Offline {
		t.Errorf("expected Offline without API key, got %d: %s", status, msg)
	}
}

func TestExtractFound(t *testing.T) {
	resolveResp := map[string]interface{}{
		"response": map[string]interface{}{
			"success": 1,
			"steamid": "76561198000000000",
		},
	}
	summaryResp := map[string]interface{}{
		"response": map[string]interface{}{
			"players": []interface{}{
				map[string]interface{}{
					"steamid":        "76561198000000000",
					"personaname":    "TestPlayer",
					"realname":       "Kyle Test",
					"profileurl":     "https://steamcommunity.com/id/testplayer/",
					"avatarfull":     "https://avatars.steamstatic.com/test.jpg",
					"loccountrycode": "DE",
				},
			},
		},
	}

	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if callCount == 0 {
			json.NewEncoder(w).Encode(resolveResp)
		} else {
			json.NewEncoder(w).Encode(summaryResp)
		}
		callCount++
	}))
	defer srv.Close()

	m := New("testkey")
	m.baseURL = srv.URL

	node := graph.NewNode("username", "testplayer", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}

	if len(nodes) == 0 {
		t.Fatal("expected at least one node")
	}
	if len(edges) == 0 {
		t.Fatal("expected at least one edge")
	}

	// Verify account node.
	var foundAccount bool
	for _, n := range nodes {
		if n.Type == graph.NodeTypeAccount {
			foundAccount = true
			if n.Confidence < 0.9 {
				t.Errorf("expected high confidence, got %f", n.Confidence)
			}
		}
	}
	if !foundAccount {
		t.Error("expected account node")
	}
}

func TestExtractNotFound(t *testing.T) {
	resolveResp := map[string]interface{}{
		"response": map[string]interface{}{
			"success": 42,
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resolveResp)
	}))
	defer srv.Close()

	m := New("testkey")
	m.baseURL = srv.URL

	node := graph.NewNode("username", "nonexistent_user_xyz", "seed")
	client := httpclient.New()

	nodes, _, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 0 {
		t.Errorf("expected no nodes, got %d", len(nodes))
	}
}

func TestVerifyHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"response": map[string]interface{}{
				"success": 1,
				"steamid": "123",
			},
		})
	}))
	defer srv.Close()

	m := New("testkey")
	m.baseURL = srv.URL

	client := httpclient.New()
	status, msg := m.Verify(context.Background(), client)
	if status != modules.Healthy {
		t.Errorf("expected Healthy, got %d: %s", status, msg)
	}
}
