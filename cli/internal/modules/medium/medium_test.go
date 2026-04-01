// SPDX-License-Identifier: AGPL-3.0-or-later

package medium

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
	"github.com/kyle/basalt/internal/modules"
)

const foundHTML = `<html><head>
<meta property="og:site_name" content="Medium" />
<meta property="og:title" content="Ev Williams – Medium" />
<meta property="og:description" content="Curious human, cofounder @ Mozi" />
<meta property="og:image" content="https://miro.medium.com/v2/test.jpg" />
</head><body></body></html>`

const notFoundHTML = `<html><head>
<meta property="og:site_name" content="Medium" />
</head><body></body></html>`

func TestCanHandle(t *testing.T) {
	m := New()
	if !m.CanHandle("username") {
		t.Error("should handle username")
	}
	if m.CanHandle("domain") {
		t.Error("should not handle domain")
	}
}

func TestExtractFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "basalt/2.0" {
			t.Error("expected User-Agent basalt/2.0")
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(foundHTML))
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	node := graph.NewNode("username", "ev", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	if nodes[0].Properties["display_name"] != "Ev Williams" {
		t.Errorf("expected display_name Ev Williams, got %v", nodes[0].Properties["display_name"])
	}
	if nodes[0].Properties["bio"] != "Curious human, cofounder @ Mozi" {
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
