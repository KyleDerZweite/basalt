// SPDX-License-Identifier: AGPL-3.0-or-later

package securitytxt

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KyleDerZweite/basalt/internal/graph"
	"github.com/KyleDerZweite/basalt/internal/httpclient"
	"github.com/KyleDerZweite/basalt/internal/modules"
)

func TestCanHandle(t *testing.T) {
	m := New()
	if !m.CanHandle(graph.NodeTypeDomain) {
		t.Error("should handle domain")
	}
	if m.CanHandle(graph.NodeTypeUsername) {
		t.Error("should not handle username")
	}
	if m.CanHandle(graph.NodeTypeEmail) {
		t.Error("should not handle email")
	}
}

func TestExtractFoundWellKnown(t *testing.T) {
	var requestedPaths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPaths = append(requestedPaths, r.URL.Path)
		if r.URL.Path != "/.well-known/security.txt" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}

		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("\uFEFF# comment\n" +
			"Contact: mailto:security@example.com\n" +
			"Contact: https://hackerone.com/example\n" +
			"Contact: xmpp:security@example.com\n" +
			"Expires: 2027-01-01T00:00:00Z\n" +
			"Preferred-Languages: en, fr, de\n" +
			"Canonical: https://security.example.net/.well-known/security.txt\n" +
			"Policy: https://example.com/security-policy\n" +
			"Acknowledgments: https://thanks.example.net/hall-of-fame\n" +
			"Hiring: https://jobs.example.net/security\n" +
			"Encryption: https://keys.example.net/pgp.txt\n"))
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	input := graph.NewNode(graph.NodeTypeDomain, "example.com", "seed")
	nodes, edges, err := m.Extract(context.Background(), input, httpclient.New())
	if err != nil {
		t.Fatal(err)
	}

	if len(requestedPaths) != 1 || requestedPaths[0] != "/.well-known/security.txt" {
		t.Fatalf("unexpected request sequence: %v", requestedPaths)
	}
	if len(nodes) != 13 {
		t.Fatalf("expected 13 nodes, got %d", len(nodes))
	}
	if len(edges) != 13 {
		t.Fatalf("expected 13 edges, got %d", len(edges))
	}

	account := findNode(t, nodes, graph.NodeTypeAccount, "securitytxt - example.com")
	if account.Confidence != 0.90 {
		t.Fatalf("expected account confidence 0.90, got %f", account.Confidence)
	}
	if account.Properties["fetched_url"] != srv.URL+"/.well-known/security.txt" {
		t.Fatalf("unexpected fetched_url: %v", account.Properties["fetched_url"])
	}
	assertStringSliceProperty(t, account, "contacts", []string{
		"mailto:security@example.com",
		"https://hackerone.com/example",
		"xmpp:security@example.com",
	})
	assertStringSliceProperty(t, account, "expires", []string{"2027-01-01T00:00:00Z"})

	emailNode := findNode(t, nodes, graph.NodeTypeEmail, "security@example.com")
	if !emailNode.Pivot || emailNode.Confidence != 0.90 {
		t.Fatalf("unexpected email node: pivot=%v confidence=%f", emailNode.Pivot, emailNode.Confidence)
	}

	contactWebsite := findNode(t, nodes, graph.NodeTypeWebsite, "https://hackerone.com/example")
	if !contactWebsite.Pivot || contactWebsite.Confidence != 0.80 {
		t.Fatalf("unexpected contact website: pivot=%v confidence=%f", contactWebsite.Pivot, contactWebsite.Confidence)
	}

	policyWebsite := findNode(t, nodes, graph.NodeTypeWebsite, "https://example.com/security-policy")
	if policyWebsite.Pivot || policyWebsite.Confidence != 0.70 {
		t.Fatalf("unexpected policy website: pivot=%v confidence=%f", policyWebsite.Pivot, policyWebsite.Confidence)
	}

	findNode(t, nodes, graph.NodeTypeDomain, "hackerone.com")
	findNode(t, nodes, graph.NodeTypeDomain, "security.example.net")
	findNode(t, nodes, graph.NodeTypeDomain, "thanks.example.net")
	findNode(t, nodes, graph.NodeTypeDomain, "jobs.example.net")
	findNode(t, nodes, graph.NodeTypeDomain, "keys.example.net")

	assertEdgeExists(t, edges, input.ID, account.ID, graph.EdgeTypeHasAccount)
	assertEdgeExists(t, edges, account.ID, emailNode.ID, graph.EdgeTypeHasEmail)
	assertEdgeExists(t, edges, account.ID, contactWebsite.ID, graph.EdgeTypeLinkedTo)
}

func TestExtractFallbackRootPath(t *testing.T) {
	var requestedPaths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPaths = append(requestedPaths, r.URL.Path)
		switch r.URL.Path {
		case "/.well-known/security.txt":
			http.NotFound(w, r)
		case "/security.txt":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("Contact: mailto:root@example.com\nExpires: 2027-01-01T00:00:00Z\n"))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	nodes, edges, err := m.Extract(context.Background(), graph.NewNode(graph.NodeTypeDomain, "example.com", "seed"), httpclient.New())
	if err != nil {
		t.Fatal(err)
	}

	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	if len(edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(edges))
	}
	if len(requestedPaths) != 2 || requestedPaths[0] != "/.well-known/security.txt" || requestedPaths[1] != "/security.txt" {
		t.Fatalf("unexpected request sequence: %v", requestedPaths)
	}

	account := findNode(t, nodes, graph.NodeTypeAccount, "securitytxt - example.com")
	if account.Properties["fetched_url"] != srv.URL+"/security.txt" {
		t.Fatalf("unexpected fetched_url: %v", account.Properties["fetched_url"])
	}
	findNode(t, nodes, graph.NodeTypeEmail, "root@example.com")
}

func TestExtractNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	nodes, edges, err := m.Extract(context.Background(), graph.NewNode(graph.NodeTypeDomain, "example.com", "seed"), httpclient.New())
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 0 || len(edges) != 0 {
		t.Fatalf("expected no results, got %d nodes and %d edges", len(nodes), len(edges))
	}
}

func TestExtractIgnoresUnsupportedContactSchemes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("Contact: xmpp:security@example.com\nExpires: 2027-01-01T00:00:00Z\n"))
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	nodes, edges, err := m.Extract(context.Background(), graph.NewNode(graph.NodeTypeDomain, "example.com", "seed"), httpclient.New())
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected only the account node, got %d nodes", len(nodes))
	}
	if len(edges) != 1 {
		t.Fatalf("expected only the seed edge, got %d edges", len(edges))
	}

	account := nodes[0]
	assertStringSliceProperty(t, account, "contacts", []string{"xmpp:security@example.com"})
}

func TestVerifyHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/security.txt" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("Contact: mailto:security@example.com\nExpires: 2027-01-01T00:00:00Z\n"))
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	status, msg := m.Verify(context.Background(), httpclient.New())
	if status != modules.Healthy {
		t.Fatalf("expected Healthy, got %d: %s", status, msg)
	}
}

func TestVerifyDegraded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("Contact: mailto:security@example.com\n"))
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL

	status, msg := m.Verify(context.Background(), httpclient.New())
	if status != modules.Degraded {
		t.Fatalf("expected Degraded, got %d: %s", status, msg)
	}
}

func findNode(t *testing.T, nodes []*graph.Node, nodeType, label string) *graph.Node {
	t.Helper()
	for _, node := range nodes {
		if node.Type == nodeType && node.Label == label {
			return node
		}
	}
	t.Fatalf("node not found: %s %q", nodeType, label)
	return nil
}

func assertStringSliceProperty(t *testing.T, node *graph.Node, key string, want []string) {
	t.Helper()
	value, ok := node.Properties[key]
	if !ok {
		t.Fatalf("missing property %q", key)
	}

	got, ok := value.([]string)
	if !ok {
		t.Fatalf("property %q has type %T, want []string", key, value)
	}
	if len(got) != len(want) {
		t.Fatalf("property %q length = %d, want %d", key, len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("property %q[%d] = %q, want %q", key, i, got[i], want[i])
		}
	}
}

func assertEdgeExists(t *testing.T, edges []*graph.Edge, source, target, edgeType string) {
	t.Helper()
	for _, edge := range edges {
		if edge.Source == source && edge.Target == target && edge.Type == edgeType {
			return
		}
	}
	t.Fatalf("edge not found: %s -> %s (%s)", source, target, edgeType)
}
