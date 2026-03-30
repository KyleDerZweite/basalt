// SPDX-License-Identifier: AGPL-3.0-or-later

package twitch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
	"github.com/kyle/basalt/internal/modules"
)

const sampleHTML = `<!DOCTYPE html>
<html>
<head>
<meta property="og:title" content="Test Streamer">
<meta property="og:description" content="Playing games all day">
<meta property="og:image" content="https://example.com/twitch-avatar.jpg">
</head>
<body></body>
</html>`

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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(sampleHTML))
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	node := graph.NewNode("username", "teststreamer", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}

	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes (account + name + avatar), got %d", len(nodes))
	}

	// Check account node.
	if nodes[0].Type != graph.NodeTypeAccount {
		t.Errorf("expected account node, got %s", nodes[0].Type)
	}
	if nodes[0].Confidence != 0.85 {
		t.Errorf("expected confidence 0.85, got %f", nodes[0].Confidence)
	}

	// Check full name node.
	if nodes[1].Type != graph.NodeTypeFullName {
		t.Errorf("expected full_name node, got %s", nodes[1].Type)
	}
	if nodes[1].Label != "Test Streamer" {
		t.Errorf("expected label 'Test Streamer', got %q", nodes[1].Label)
	}

	// Check avatar node.
	if nodes[2].Type != graph.NodeTypeAvatarURL {
		t.Errorf("expected avatar_url node, got %s", nodes[2].Type)
	}
	if nodes[2].Label != "https://example.com/twitch-avatar.jpg" {
		t.Errorf("expected avatar URL, got %q", nodes[2].Label)
	}

	if len(edges) != 3 {
		t.Fatalf("expected 3 edges, got %d", len(edges))
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
	if len(nodes) != 0 {
		t.Errorf("expected no nodes for 404, got %d", len(nodes))
	}
	if len(edges) != 0 {
		t.Errorf("expected no edges for 404, got %d", len(edges))
	}
}

func TestVerifyHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
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

func TestVerifyDegraded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	client := httpclient.New()
	status, _ := m.Verify(context.Background(), client)
	if status != modules.Degraded {
		t.Errorf("expected Degraded, got %d", status)
	}
}
