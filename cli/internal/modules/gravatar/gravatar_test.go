// SPDX-License-Identifier: AGPL-3.0-or-later

package gravatar

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
	if !m.CanHandle("email") {
		t.Error("should handle email")
	}
	if m.CanHandle("username") {
		t.Error("should not handle username")
	}
}

func TestExtractFound(t *testing.T) {
	profile := map[string]interface{}{
		"displayName":       "Kyle Test",
		"preferredUsername": "kyletest",
		"thumbnailUrl":      "https://gravatar.com/avatar/abc123",
		"urls": []interface{}{
			map[string]interface{}{"value": "https://kylehub.dev"},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(profile)
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL // override for testing

	node := graph.NewNode("email", "kyle@example.com", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}

	if len(nodes) == 0 {
		t.Fatal("expected at least one node")
	}

	// Check that we got an account node.
	var foundAccount bool
	for _, n := range nodes {
		if n.Type == graph.NodeTypeAccount {
			foundAccount = true
			if n.Confidence < 0.9 {
				t.Errorf("expected high confidence for found profile, got %f", n.Confidence)
			}
		}
	}
	if !foundAccount {
		t.Error("expected an account node")
	}

	if len(edges) == 0 {
		t.Error("expected at least one edge")
	}
}

func TestExtractNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	node := graph.NewNode("email", "nobody@example.com", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 0 {
		t.Errorf("expected no nodes for 404, got %d", len(nodes))
	}
	if len(edges) != 0 {
		t.Errorf("expected no edges for 404, got %d", len(edges))
	}
}

func TestVerifyHealthy(t *testing.T) {
	profile := map[string]interface{}{
		"displayName": "Test User",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(profile)
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
