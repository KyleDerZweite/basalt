// SPDX-License-Identifier: AGPL-3.0-or-later

package steam

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KyleDerZweite/basalt/internal/graph"
	"github.com/KyleDerZweite/basalt/internal/httpclient"
	"github.com/KyleDerZweite/basalt/internal/modules"
)

const sampleHTML = `<html><head>
<meta property="og:title" content="Steam Community :: TestPlayer" />
<meta property="og:description" content="I play games" />
<meta property="og:image" content="https://avatars.steamstatic.com/test_full.jpg" />
</head><body></body></html>`

const errorHTML = `<html><head>
<meta property="og:title" content="Steam Community :: Error" />
</head><body></body></html>`

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

	node := graph.NewNode("username", "testplayer", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}

	// account + avatar = 2 nodes
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	if nodes[0].Type != graph.NodeTypeAccount {
		t.Errorf("expected account node, got %s", nodes[0].Type)
	}
	if nodes[0].Confidence != 0.90 {
		t.Errorf("expected confidence 0.90, got %f", nodes[0].Confidence)
	}
	if nodes[0].Properties["persona_name"] != "TestPlayer" {
		t.Errorf("expected persona_name TestPlayer, got %v", nodes[0].Properties["persona_name"])
	}
	if nodes[0].Properties["description"] != "I play games" {
		t.Errorf("expected description, got %v", nodes[0].Properties["description"])
	}
	if nodes[1].Type != graph.NodeTypeAvatarURL {
		t.Errorf("expected avatar node, got %s", nodes[1].Type)
	}
	// has_account + linked_to = 2 edges
	if len(edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(edges))
	}
}

func TestExtractNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(errorHTML))
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
		t.Errorf("expected no nodes, got %d", len(nodes))
	}
	if len(edges) != 0 {
		t.Errorf("expected no edges, got %d", len(edges))
	}
}

func TestVerifyHealthy(t *testing.T) {
	html := `<html><head>
<meta property="og:title" content="Steam Community :: Valve" />
</head><body></body></html>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
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
