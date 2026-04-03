// SPDX-License-Identifier: AGPL-3.0-or-later

package shodan

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

// mockResolver returns a fixed IP for testing.
type mockResolver struct {
	ip  string
	err error
}

func (r *mockResolver) LookupHost(_ context.Context, _ string) ([]string, error) {
	if r.err != nil {
		return nil, r.err
	}
	return []string{r.ip}, nil
}

func TestCanHandle(t *testing.T) {
	m := New()
	if !m.CanHandle("domain") {
		t.Error("should handle domain")
	}
	if m.CanHandle("username") {
		t.Error("should not handle username")
	}
	if m.CanHandle("email") {
		t.Error("should not handle email")
	}
	if m.CanHandle("ip") {
		t.Error("should not handle ip")
	}
}

func TestExtractFound(t *testing.T) {
	shodanResp := internetDBResult{
		IP:        "1.2.3.4",
		Ports:     []int{80, 443, 8080},
		CPEs:      []string{"cpe:/a:apache:http_server:2.4.41"},
		Hostnames: []string{"example.com"},
		Tags:      []string{"cloud"},
		Vulns:     []string{"CVE-2021-44228", "CVE-2023-12345"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/1.2.3.4" {
			t.Errorf("unexpected request path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(shodanResp)
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL
	m.resolver = &mockResolver{ip: "1.2.3.4"}

	node := graph.NewNode("domain", "example.com", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}

	// Expect: 1 account node + 1 IP node.
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	if len(edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(edges))
	}

	// Verify account node.
	account := nodes[0]
	if account.Type != graph.NodeTypeAccount {
		t.Errorf("expected account node, got %s", account.Type)
	}
	if account.Confidence != 0.90 {
		t.Errorf("expected confidence 0.90, got %f", account.Confidence)
	}
	if account.SourceModule != "shodan" {
		t.Errorf("expected source_module shodan, got %s", account.SourceModule)
	}

	// Verify ports property.
	ports, ok := account.Properties["ports"].([]int)
	if !ok {
		t.Fatalf("expected ports as []int, got %T", account.Properties["ports"])
	}
	if len(ports) != 3 {
		t.Errorf("expected 3 ports, got %d", len(ports))
	}

	// Verify vulns property.
	vulns, ok := account.Properties["vulns"].([]string)
	if !ok {
		t.Fatalf("expected vulns as []string, got %T", account.Properties["vulns"])
	}
	if len(vulns) != 2 {
		t.Errorf("expected 2 vulns, got %d", len(vulns))
	}

	// Verify vuln_count property.
	vulnCount, ok := account.Properties["vuln_count"].(int)
	if !ok {
		t.Fatalf("expected vuln_count as int, got %T", account.Properties["vuln_count"])
	}
	if vulnCount != 2 {
		t.Errorf("expected vuln_count 2, got %d", vulnCount)
	}

	// Verify IP property.
	ipProp, ok := account.Properties["ip"].(string)
	if !ok || ipProp != "1.2.3.4" {
		t.Errorf("expected ip property 1.2.3.4, got %v", account.Properties["ip"])
	}

	// Verify CPEs, hostnames, tags.
	cpes, ok := account.Properties["cpes"].([]string)
	if !ok || len(cpes) != 1 {
		t.Errorf("expected 1 CPE, got %v", account.Properties["cpes"])
	}
	hostnames, ok := account.Properties["hostnames"].([]string)
	if !ok || len(hostnames) != 1 {
		t.Errorf("expected 1 hostname, got %v", account.Properties["hostnames"])
	}
	tags, ok := account.Properties["tags"].([]string)
	if !ok || len(tags) != 1 {
		t.Errorf("expected 1 tag, got %v", account.Properties["tags"])
	}

	// Verify IP node.
	ipNode := nodes[1]
	if ipNode.Type != graph.NodeTypeIP {
		t.Errorf("expected ip node, got %s", ipNode.Type)
	}
	if ipNode.Label != "1.2.3.4" {
		t.Errorf("expected IP label 1.2.3.4, got %s", ipNode.Label)
	}
	if ipNode.Confidence != 0.95 {
		t.Errorf("expected IP confidence 0.95, got %f", ipNode.Confidence)
	}
	if ipNode.Pivot {
		t.Error("IP node should not be pivotable")
	}

	// Verify edges.
	if edges[0].Type != graph.EdgeTypeHasAccount {
		t.Errorf("expected has_account edge, got %s", edges[0].Type)
	}
	if edges[0].Source != node.ID {
		t.Errorf("expected edge source %s, got %s", node.ID, edges[0].Source)
	}
	if edges[0].Target != account.ID {
		t.Errorf("expected edge target %s, got %s", account.ID, edges[0].Target)
	}
	if edges[1].Type != graph.EdgeTypeResolvesTo {
		t.Errorf("expected resolves_to edge, got %s", edges[1].Type)
	}
	if edges[1].Source != account.ID {
		t.Errorf("expected edge source %s, got %s", account.ID, edges[1].Source)
	}
	if edges[1].Target != ipNode.ID {
		t.Errorf("expected edge target %s, got %s", ipNode.ID, edges[1].Target)
	}
}

func TestExtractNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL
	m.resolver = &mockResolver{ip: "1.2.3.4"}

	node := graph.NewNode("domain", "unknown.example", "seed")
	client := httpclient.New()

	nodes, edges, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatalf("expected no error for 404, got: %v", err)
	}
	if nodes != nil {
		t.Errorf("expected nil nodes, got %d", len(nodes))
	}
	if edges != nil {
		t.Errorf("expected nil edges, got %d", len(edges))
	}
}

func TestVerifyHealthy(t *testing.T) {
	shodanResp := internetDBResult{
		IP:    "8.8.8.8",
		Ports: []int{53, 443},
		Vulns: []string{"CVE-2021-99999"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/8.8.8.8" {
			t.Errorf("verify should query 8.8.8.8, got %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(shodanResp)
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

func TestExtractNoVulns(t *testing.T) {
	// Verify that vuln_count is not set when there are no vulns.
	shodanResp := internetDBResult{
		IP:    "1.2.3.4",
		Ports: []int{80},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(shodanResp)
	}))
	defer srv.Close()

	m := New()
	m.baseURL = srv.URL
	m.resolver = &mockResolver{ip: "1.2.3.4"}

	node := graph.NewNode("domain", "example.com", "seed")
	client := httpclient.New()

	nodes, _, err := m.Extract(context.Background(), node, client)
	if err != nil {
		t.Fatal(err)
	}

	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}

	account := nodes[0]
	if _, ok := account.Properties["vulns"]; ok {
		t.Error("should not have vulns property when no vulns")
	}
	if _, ok := account.Properties["vuln_count"]; ok {
		t.Error("should not have vuln_count property when no vulns")
	}
}
