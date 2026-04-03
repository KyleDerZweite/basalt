// SPDX-License-Identifier: AGPL-3.0-or-later

package codeberg

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
		"login":           "forgejo",
		"full_name":       "Forgejo",
		"avatar_url":      "https://codeberg.org/avatars/forgejo.png",
		"html_url":        "https://codeberg.org/forgejo",
		"website":         "forgejo.org",
		"location":        "Europe",
		"description":     "Beyond coding. We forge.",
		"created":         "2022-11-06T07:18:11+01:00",
		"followers_count": 429,
		"following_count": 2,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	node := graph.NewNode("username", "forgejo", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 5 {
		t.Fatalf("expected 5 nodes, got %d", len(nodes))
	}
	if len(edges) != 5 {
		t.Fatalf("expected 5 edges, got %d", len(edges))
	}

	account := nodes[0]
	if account.Type != graph.NodeTypeAccount {
		t.Fatalf("expected account node, got %s", account.Type)
	}
	if account.Properties["followers_count"] != 429 {
		t.Errorf("expected followers_count 429, got %v", account.Properties["followers_count"])
	}

	if nodes[1].Type != graph.NodeTypeFullName || nodes[1].Label != "Forgejo" {
		t.Errorf("expected full_name node, got %s %q", nodes[1].Type, nodes[1].Label)
	}
	if nodes[2].Type != graph.NodeTypeAvatarURL {
		t.Errorf("expected avatar node, got %s", nodes[2].Type)
	}
	if nodes[3].Type != graph.NodeTypeWebsite || nodes[3].Label != "https://forgejo.org" {
		t.Errorf("expected normalized website node, got %s %q", nodes[3].Type, nodes[3].Label)
	}
	if !nodes[3].Pivot {
		t.Error("expected website node to be pivotable")
	}
	if nodes[4].Type != graph.NodeTypeDomain || nodes[4].Label != "forgejo.org" {
		t.Errorf("expected domain node, got %s %q", nodes[4].Type, nodes[4].Label)
	}
}

func TestExtractNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "missing"})
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	node := graph.NewNode("username", "missing", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 0 || len(edges) != 0 {
		t.Errorf("expected no results for 404, got %d nodes and %d edges", len(nodes), len(edges))
	}
}

func TestVerifyHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"login": "forgejo"})
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	client := httpclient.New()
	status, msg := m.Verify(context.Background(), client)
	if status != modules.Healthy {
		t.Fatalf("expected Healthy, got %d: %s", status, msg)
	}
}
