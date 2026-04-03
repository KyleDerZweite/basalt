// SPDX-License-Identifier: AGPL-3.0-or-later

package wayback

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KyleDerZweite/basalt/internal/graph"
	"github.com/KyleDerZweite/basalt/internal/httpclient"
	"github.com/KyleDerZweite/basalt/internal/modules"
)

func TestCanHandle(t *testing.T) {
	m := New()
	if !m.CanHandle("domain") {
		t.Error("should handle domain")
	}
	if m.CanHandle("username") {
		t.Error("should not handle username")
	}
	if m.CanHandle("email") {
		t.Error("should not handle email")
	}
}

func TestExtractFound(t *testing.T) {
	payload := map[string]interface{}{
		"url": "example.com",
		"archived_snapshots": map[string]interface{}{
			"closest": map[string]interface{}{
				"url":       "https://web.archive.org/web/20240101120000/https://example.com",
				"timestamp": "20240101120000",
				"status":    "200",
				"available": true,
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	node := graph.NewNode(graph.NodeTypeDomain, "example.com", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}

	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}

	account := nodes[0]
	if account.Type != graph.NodeTypeAccount {
		t.Errorf("expected account node, got %s", account.Type)
	}
	if account.Confidence != 0.85 {
		t.Errorf("expected confidence 0.85, got %f", account.Confidence)
	}
	if account.Properties["snapshot_url"] != "https://web.archive.org/web/20240101120000/https://example.com" {
		t.Errorf("unexpected snapshot_url: %v", account.Properties["snapshot_url"])
	}
	if account.Properties["timestamp"] != "20240101120000" {
		t.Errorf("unexpected timestamp: %v", account.Properties["timestamp"])
	}
	if account.Properties["first_seen"] != "2024" {
		t.Errorf("expected first_seen 2024, got %v", account.Properties["first_seen"])
	}

	edge := edges[0]
	if edge.Type != graph.EdgeTypeHasAccount {
		t.Errorf("expected has_account edge, got %s", edge.Type)
	}
	if edge.Source != node.ID {
		t.Errorf("edge source should be seed node ID")
	}
	if edge.Target != account.ID {
		t.Errorf("edge target should be account node ID")
	}
}

func TestExtractNotArchived(t *testing.T) {
	payload := map[string]interface{}{
		"url":                "nonexistent-domain-xyz.test",
		"archived_snapshots": map[string]interface{}{},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	node := graph.NewNode(graph.NodeTypeDomain, "nonexistent-domain-xyz.test", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}
	if nodes != nil {
		t.Errorf("expected nil nodes, got %d", len(nodes))
	}
	if edges != nil {
		t.Errorf("expected nil edges, got %d", len(edges))
	}
}

func TestVerifyHealthy(t *testing.T) {
	payload := map[string]interface{}{
		"url": "example.com",
		"archived_snapshots": map[string]interface{}{
			"closest": map[string]interface{}{
				"url":       "https://web.archive.org/web/20240101000000/https://example.com",
				"timestamp": "20240101000000",
				"status":    "200",
				"available": true,
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	client := httpclient.New()
	status, msg := m.Verify(context.Background(), client)
	if status != modules.Healthy {
		t.Errorf("expected Healthy, got %d: %s", status, msg)
	}
}

func TestVerifyHealthyWithoutSnapshot(t *testing.T) {
	payload := map[string]interface{}{
		"url":                "example.com",
		"archived_snapshots": map[string]interface{}{},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	client := httpclient.New()
	status, msg := m.Verify(context.Background(), client)
	if status != modules.Healthy {
		t.Errorf("expected Healthy, got %d: %s", status, msg)
	}
}
