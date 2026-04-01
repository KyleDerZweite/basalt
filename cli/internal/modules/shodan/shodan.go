// SPDX-License-Identifier: AGPL-3.0-or-later

package shodan

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
	"github.com/kyle/basalt/internal/modules"
)

const defaultBaseURL = "https://internetdb.shodan.io"

// Resolver abstracts DNS lookups for testing.
type Resolver interface {
	LookupHost(ctx context.Context, host string) ([]string, error)
}

// netResolver wraps the standard library DNS resolver.
type netResolver struct{}

func (r *netResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	return net.DefaultResolver.LookupHost(ctx, host)
}

// internetDBResult represents the JSON response from the Shodan InternetDB API.
type internetDBResult struct {
	IP        string   `json:"ip"`
	Ports     []int    `json:"ports"`
	CPEs      []string `json:"cpes"`
	Hostnames []string `json:"hostnames"`
	Tags      []string `json:"tags"`
	Vulns     []string `json:"vulns"`
}

// Module discovers open ports, vulnerabilities, and services for a domain via Shodan InternetDB.
type Module struct {
	baseURL  string
	resolver Resolver
}

// New creates a Shodan InternetDB module with default settings.
func New() *Module {
	return &Module{
		baseURL:  defaultBaseURL,
		resolver: &netResolver{},
	}
}

func (m *Module) Name() string        { return "shodan" }
func (m *Module) Description() string { return "Open ports, vulnerabilities, and services via Shodan InternetDB" }

func (m *Module) CanHandle(nodeType string) bool {
	return nodeType == "domain"
}

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	domain := node.Label

	// Resolve domain to IP.
	ips, err := m.resolver.LookupHost(ctx, domain)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving %s: %w", domain, err)
	}
	if len(ips) == 0 {
		return nil, nil, fmt.Errorf("no IPs found for %s", domain)
	}
	ip := ips[0]

	// Query Shodan InternetDB.
	url := fmt.Sprintf("%s/%s", m.baseURL, ip)
	resp, err := client.Do(ctx, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("shodan request: %w", err)
	}

	// 404 means IP is not in the Shodan database.
	if resp.StatusCode == 404 {
		return nil, nil, nil
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("shodan returned HTTP %d for %s", resp.StatusCode, ip)
	}

	var result internetDBResult
	if err := json.Unmarshal([]byte(resp.Body), &result); err != nil {
		return nil, nil, fmt.Errorf("parsing shodan response: %w", err)
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	// Account node representing the Shodan footprint for this domain.
	account := graph.NewAccountNode("shodan", domain, "", "shodan")
	account.Confidence = 0.90

	if len(result.Ports) > 0 {
		account.Properties["ports"] = result.Ports
	}
	if len(result.CPEs) > 0 {
		account.Properties["cpes"] = result.CPEs
	}
	if len(result.Hostnames) > 0 {
		account.Properties["hostnames"] = result.Hostnames
	}
	if len(result.Tags) > 0 {
		account.Properties["tags"] = result.Tags
	}
	if len(result.Vulns) > 0 {
		account.Properties["vulns"] = result.Vulns
		account.Properties["vuln_count"] = len(result.Vulns)
	}
	if result.IP != "" {
		account.Properties["ip"] = result.IP
	}

	nodes = append(nodes, account)

	// Edge from seed node to account.
	edges = append(edges, graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "shodan"))

	// IP node.
	ipNode := graph.NewNode(graph.NodeTypeIP, ip, "shodan")
	ipNode.Confidence = 0.95
	ipNode.Pivot = false
	nodes = append(nodes, ipNode)

	// Edge from account to IP node.
	edges = append(edges, graph.NewEdge(0, account.ID, ipNode.ID, graph.EdgeTypeResolvesTo, "shodan"))

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	// Look up 8.8.8.8 (Google DNS) which is always in Shodan.
	url := fmt.Sprintf("%s/8.8.8.8", m.baseURL)
	resp, err := client.Do(ctx, url, nil)
	if err != nil {
		return modules.Offline, fmt.Sprintf("shodan: %v", err)
	}
	if resp.StatusCode != 200 {
		return modules.Offline, fmt.Sprintf("shodan: HTTP %d", resp.StatusCode)
	}

	var result internetDBResult
	if err := json.Unmarshal([]byte(resp.Body), &result); err != nil {
		return modules.Degraded, fmt.Sprintf("shodan: failed to parse response: %v", err)
	}
	if len(result.Ports) == 0 {
		return modules.Degraded, "shodan: no ports returned for 8.8.8.8"
	}

	return modules.Healthy, "shodan: OK"
}
