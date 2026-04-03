// SPDX-License-Identifier: AGPL-3.0-or-later

package matrix

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
	profile := map[string]interface{}{
		"displayname": "Alice Wonderland",
		"avatar_url":  "mxc://matrix.org/abc123def456",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(profile)
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	node := graph.NewNode("username", "alice", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}

	// Expect: account + full_name + avatar_url = 3 nodes
	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(nodes))
	}
	if len(edges) != 3 {
		t.Fatalf("expected 3 edges, got %d", len(edges))
	}

	var foundAccount, foundName, foundAvatar bool
	for _, n := range nodes {
		switch n.Type {
		case graph.NodeTypeAccount:
			foundAccount = true
			if n.Confidence != 0.85 {
				t.Errorf("expected account confidence 0.85, got %f", n.Confidence)
			}
			if n.Label != "matrix - @alice:matrix.org" {
				t.Errorf("expected label 'matrix - @alice:matrix.org', got %q", n.Label)
			}
		case graph.NodeTypeFullName:
			foundName = true
			if n.Label != "Alice Wonderland" {
				t.Errorf("expected name 'Alice Wonderland', got %q", n.Label)
			}
		case graph.NodeTypeAvatarURL:
			foundAvatar = true
			expected := "https://matrix.org/_matrix/media/v3/download/matrix.org/abc123def456"
			if n.Label != expected {
				t.Errorf("expected avatar URL %q, got %q", expected, n.Label)
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
}

func TestExtractNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
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
		t.Errorf("expected nil nodes for 404, got %d", len(nodes))
	}
	if edges != nil {
		t.Errorf("expected nil edges for 404, got %d", len(edges))
	}
}

func TestExtractWithHomeserver(t *testing.T) {
	profile := map[string]interface{}{
		"displayname": "Bob Builder",
	}

	var requestedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(profile)
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	// Username with homeserver notation.
	node := graph.NewNode("username", "bob:example.org", "seed")
	client := httpclient.New()

	nodes, _, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}

	// Should query for @bob:example.org
	expected := "/_matrix/client/v3/profile/@bob:example.org"
	if requestedPath != expected {
		t.Errorf("expected path %q, got %q", expected, requestedPath)
	}

	if len(nodes) < 1 {
		t.Fatal("expected at least one node")
	}

	// Account label should include the full Matrix ID.
	var foundAccount bool
	for _, n := range nodes {
		if n.Type == graph.NodeTypeAccount {
			foundAccount = true
			if n.Label != "matrix - @bob:example.org" {
				t.Errorf("expected label 'matrix - @bob:example.org', got %q", n.Label)
			}
		}
	}
	if !foundAccount {
		t.Error("expected account node")
	}
}

func TestMxcToHTTP(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"mxc://matrix.org/abc123", "https://matrix.org/_matrix/media/v3/download/matrix.org/abc123"},
		{"mxc://example.org/media456", "https://matrix.org/_matrix/media/v3/download/example.org/media456"},
		{"https://already-http.com/image.png", "https://already-http.com/image.png"},
	}

	for _, tt := range tests {
		got := mxcToHTTP(tt.input)
		if got != tt.expected {
			t.Errorf("mxcToHTTP(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestVerifyHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"displayname": "Alice",
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

func TestVerifyHealthyOn404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	client := httpclient.New()
	status, msg := m.Verify(context.Background(), client)
	if status != modules.Healthy {
		t.Errorf("expected Healthy on 404 (API is up), got %d: %s", status, msg)
	}
}
