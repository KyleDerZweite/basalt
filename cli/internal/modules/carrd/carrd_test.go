// SPDX-License-Identifier: AGPL-3.0-or-later

package carrd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KyleDerZweite/basalt/internal/graph"
	"github.com/KyleDerZweite/basalt/internal/httpclient"
	"github.com/KyleDerZweite/basalt/internal/modules"
)

const sampleHTML = `<!DOCTYPE html>
<html>
<head><title>testuser</title></head>
<body>
<a href="https://github.com/testuser">GitHub</a>
<a href="https://twitter.com/testuser">Twitter</a>
<a href="https://testuser.com">Website</a>
</body>
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
	m.overrideURL = srv.URL + "/"

	node := graph.NewNode("username", "testuser", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}

	if len(nodes) < 2 {
		t.Fatalf("expected at least 2 nodes (account + links), got %d", len(nodes))
	}
	if len(edges) < 2 {
		t.Fatalf("expected at least 2 edges, got %d", len(edges))
	}
}

func TestExtractNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	m := New()
	m.overrideURL = srv.URL + "/"

	node := graph.NewNode("username", "nonexistent", "seed")
	client := httpclient.New()

	nodes, _, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 0 {
		t.Errorf("expected no nodes for 404, got %d", len(nodes))
	}
}

func TestVerifyHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html><body>OK</body></html>"))
	}))
	defer srv.Close()

	m := New()
	m.verifyURL = srv.URL

	client := httpclient.New()
	status, _ := m.Verify(context.Background(), client)
	if status != modules.Healthy {
		t.Errorf("expected Healthy, got %d", status)
	}
}
