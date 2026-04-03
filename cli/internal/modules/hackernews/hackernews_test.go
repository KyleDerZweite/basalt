// SPDX-License-Identifier: AGPL-3.0-or-later

package hackernews

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
	if !m.CanHandle("username") {
		t.Error("should handle username")
	}
	if m.CanHandle("email") {
		t.Error("should not handle email")
	}
}

func TestExtractFound(t *testing.T) {
	payload := map[string]interface{}{
		"id":      "testuser",
		"created": 1234567890,
		"karma":   4200,
		"about":   "I love <i>hacking</i>",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	node := graph.NewNode("username", "testuser", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}

	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Type != graph.NodeTypeAccount {
		t.Errorf("expected account node, got %s", nodes[0].Type)
	}
	if nodes[0].Confidence != 0.95 {
		t.Errorf("expected confidence 0.95, got %f", nodes[0].Confidence)
	}
	if nodes[0].Properties["profile_url"] != "https://news.ycombinator.com/user?id=testuser" {
		t.Errorf("unexpected profile_url: %v", nodes[0].Properties["profile_url"])
	}
	// JSON numbers decoded via interface{} are float64.
	if karma, ok := nodes[0].Properties["karma"].(int); !ok || karma != 4200 {
		t.Errorf("expected karma 4200, got %v", nodes[0].Properties["karma"])
	}
	if created, ok := nodes[0].Properties["created"].(int64); !ok || created != 1234567890 {
		t.Errorf("expected created 1234567890, got %v", nodes[0].Properties["created"])
	}
	if nodes[0].Properties["about"] != "I love <i>hacking</i>" {
		t.Errorf("expected about with HTML, got %v", nodes[0].Properties["about"])
	}

	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}
	if edges[0].Type != graph.EdgeTypeHasAccount {
		t.Errorf("expected has_account edge, got %s", edges[0].Type)
	}
}

func TestExtractNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("null"))
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	node := graph.NewNode("username", "nonexistent", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 0 {
		t.Errorf("expected no nodes for null response, got %d", len(nodes))
	}
	if len(edges) != 0 {
		t.Errorf("expected no edges for null response, got %d", len(edges))
	}
}

func TestVerifyHealthy(t *testing.T) {
	payload := map[string]interface{}{
		"id":      "dang",
		"created": 1234567890,
		"karma":   20000,
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
