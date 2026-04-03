// SPDX-License-Identifier: AGPL-3.0-or-later

package gitlab

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
	users := []map[string]interface{}{
		{
			"username":    "testuser",
			"name":        "Test User",
			"web_url":     "https://gitlab.com/testuser",
			"avatar_url":  "https://gitlab.com/uploads/-/system/user/avatar/123/avatar.png",
			"bio":         "Go developer",
			"website_url": "https://example.com",
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
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

	// Expect: account + full_name + avatar_url + website = 4 nodes
	if len(nodes) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(nodes))
	}
	if len(edges) != 4 {
		t.Fatalf("expected 4 edges, got %d", len(edges))
	}

	var foundAccount, foundName, foundAvatar, foundWebsite bool
	for _, n := range nodes {
		switch n.Type {
		case graph.NodeTypeAccount:
			foundAccount = true
			if n.Confidence != 0.90 {
				t.Errorf("expected account confidence 0.90, got %f", n.Confidence)
			}
			if n.Properties["bio"] != "Go developer" {
				t.Errorf("expected bio 'Go developer', got %v", n.Properties["bio"])
			}
		case graph.NodeTypeFullName:
			foundName = true
			if n.Label != "Test User" {
				t.Errorf("expected name 'Test User', got %q", n.Label)
			}
		case graph.NodeTypeAvatarURL:
			foundAvatar = true
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
	if !foundName {
		t.Error("expected full_name node")
	}
	if !foundAvatar {
		t.Error("expected avatar_url node")
	}
	if !foundWebsite {
		t.Error("expected website node")
	}
}

func TestExtractNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]interface{}{})
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
		t.Errorf("expected nil nodes for empty array, got %d", len(nodes))
	}
	if edges != nil {
		t.Errorf("expected nil edges for empty array, got %d", len(edges))
	}
}

func TestVerifyHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{"username": "root"},
		})
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
