// SPDX-License-Identifier: AGPL-3.0-or-later

package stackexchange

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
	if m.CanHandle("domain") {
		t.Error("should not handle domain")
	}
}

func TestExtractFound(t *testing.T) {
	data := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{
				"display_name": "testuser",
				"link":         "https://stackoverflow.com/users/12345/testuser",
				"website_url":  "https://example.com",
				"location":     "Berlin, Germany",
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
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

	// Expect: account + website = 2 nodes
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	if len(edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(edges))
	}

	var foundAccount, foundWebsite bool
	for _, n := range nodes {
		switch n.Type {
		case graph.NodeTypeAccount:
			foundAccount = true
			if n.Confidence != 0.80 {
				t.Errorf("expected confidence 0.80, got %f", n.Confidence)
			}
			if n.Properties["location"] != "Berlin, Germany" {
				t.Errorf("expected location 'Berlin, Germany', got %v", n.Properties["location"])
			}
		case graph.NodeTypeWebsite:
			foundWebsite = true
			if !n.Pivot {
				t.Error("website node should be pivotable")
			}
			if n.Label != "https://example.com" {
				t.Errorf("expected website 'https://example.com', got %q", n.Label)
			}
		}
	}
	if !foundAccount {
		t.Error("expected account node")
	}
	if !foundWebsite {
		t.Error("expected website node")
	}
}

func TestExtractNotFound(t *testing.T) {
	data := map[string]interface{}{
		"items": []interface{}{},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
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
	if nodes != nil {
		t.Errorf("expected nil nodes for empty items, got %d", len(nodes))
	}
	if edges != nil {
		t.Errorf("expected nil edges for empty items, got %d", len(edges))
	}
}

func TestExtractNoMatch(t *testing.T) {
	// Items returned but none match the username (case-insensitive).
	data := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{
				"display_name": "otheruser",
				"link":         "https://stackoverflow.com/users/99999/otheruser",
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
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
	if nodes != nil {
		t.Errorf("expected nil nodes when no display_name matches, got %d", len(nodes))
	}
	if edges != nil {
		t.Errorf("expected nil edges when no display_name matches, got %d", len(edges))
	}
}

func TestVerifyHealthy(t *testing.T) {
	data := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{
				"display_name": "jonathon",
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
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
