// SPDX-License-Identifier: AGPL-3.0-or-later

package telegram

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KyleDerZweite/basalt/internal/graph"
	"github.com/KyleDerZweite/basalt/internal/httpclient"
	"github.com/KyleDerZweite/basalt/internal/modules"
)

const foundHTML = `<html><head>
<meta property="og:title" content="Test Channel" />
<meta property="og:description" content="This is a test bio" />
<meta property="og:image" content="https://cdn.telesco.pe/file/test.jpg" />
<meta property="og:site_name" content="Telegram" />
</head><body></body></html>`

const notFoundHTML = `<html><head>
<meta property="og:title" content="Telegram: Contact @nonexistent" />
<meta property="og:description" content="" />
<meta property="og:image" content="https://telegram.org/img/t_logo_2x.png" />
<meta property="og:site_name" content="Telegram" />
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
		w.Write([]byte(foundHTML))
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	node := graph.NewNode("username", "testchannel", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes (account+avatar), got %d", len(nodes))
	}
	if nodes[0].Type != graph.NodeTypeAccount {
		t.Errorf("expected account node, got %s", nodes[0].Type)
	}
	if nodes[0].Properties["display_name"] != "Test Channel" {
		t.Errorf("expected display_name, got %v", nodes[0].Properties["display_name"])
	}
	if nodes[0].Properties["bio"] != "This is a test bio" {
		t.Errorf("expected bio, got %v", nodes[0].Properties["bio"])
	}
	if len(edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(edges))
	}
}

func TestExtractNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(notFoundHTML))
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
	if len(nodes) != 0 || len(edges) != 0 {
		t.Error("expected no results for non-existent user")
	}
}

func TestVerifyHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(foundHTML))
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
