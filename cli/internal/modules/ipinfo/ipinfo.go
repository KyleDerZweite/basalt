// SPDX-License-Identifier: AGPL-3.0-or-later

package ipinfo

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
	"github.com/kyle/basalt/internal/modules"
)

const defaultBaseURL = "https://ipinfo.io"

// Resolver abstracts DNS lookups for testing.
type Resolver interface {
	LookupHost(ctx context.Context, host string) ([]string, error)
}

type netResolver struct{}

func (r *netResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	return net.DefaultResolver.LookupHost(ctx, host)
}

// Module discovers IP geolocation and ASN data via ipinfo.io.
type Module struct {
	baseURL  string
	resolver Resolver
}

// New creates an IPinfo module with default settings.
func New() *Module {
	return &Module{
		baseURL:  defaultBaseURL,
		resolver: &netResolver{},
	}
}

func (m *Module) Name() string        { return "ipinfo" }
func (m *Module) Description() string { return "IP geolocation and ASN data via ipinfo.io" }

func (m *Module) CanHandle(nodeType string) bool {
	return nodeType == "domain"
}

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	domain := node.Label

	ips, err := m.resolver.LookupHost(ctx, domain)
	if err != nil || len(ips) == 0 {
		return nil, nil, fmt.Errorf("dns resolution failed for %s: %w", domain, err)
	}
	ip := ips[0]

	apiURL := fmt.Sprintf("%s/%s/json", m.baseURL, ip)
	resp, err := client.Do(ctx, apiURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("ipinfo request: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("ipinfo returned %d", resp.StatusCode)
	}

	var info struct {
		IP       string `json:"ip"`
		Hostname string `json:"hostname"`
		City     string `json:"city"`
		Region   string `json:"region"`
		Country  string `json:"country"`
		Loc      string `json:"loc"`
		Org      string `json:"org"`
		Postal   string `json:"postal"`
		Timezone string `json:"timezone"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &info); err != nil {
		return nil, nil, fmt.Errorf("parsing ipinfo response: %w", err)
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	account := graph.NewAccountNode("ipinfo", domain, "", "ipinfo")
	account.Confidence = 0.90
	if info.IP != "" {
		account.Properties["ip"] = info.IP
	}
	if info.Hostname != "" {
		account.Properties["hostname"] = info.Hostname
	}
	if info.City != "" {
		account.Properties["city"] = info.City
	}
	if info.Region != "" {
		account.Properties["region"] = info.Region
	}
	if info.Country != "" {
		account.Properties["country"] = info.Country
	}
	if info.Loc != "" {
		account.Properties["loc"] = info.Loc
	}
	if info.Org != "" {
		account.Properties["org"] = info.Org
	}
	if info.Postal != "" {
		account.Properties["postal"] = info.Postal
	}
	if info.Timezone != "" {
		account.Properties["timezone"] = info.Timezone
	}
	nodes = append(nodes, account)
	edges = append(edges, graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "ipinfo"))

	ipNode := graph.NewNode(graph.NodeTypeIP, ip, "ipinfo")
	ipNode.Confidence = 0.95
	ipNode.Pivot = false
	nodes = append(nodes, ipNode)
	edges = append(edges, graph.NewEdge(0, account.ID, ipNode.ID, graph.EdgeTypeResolvesTo, "ipinfo"))

	if info.Org != "" {
		orgNode := graph.NewNode(graph.NodeTypeOrganization, info.Org, "ipinfo")
		orgNode.Confidence = 0.80
		orgNode.Pivot = false
		nodes = append(nodes, orgNode)
		edges = append(edges, graph.NewEdge(0, account.ID, orgNode.ID, graph.EdgeTypeLinkedTo, "ipinfo"))
	}

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	apiURL := fmt.Sprintf("%s/8.8.8.8/json", m.baseURL)
	resp, err := client.Do(ctx, apiURL, nil)
	if err != nil {
		return modules.Offline, fmt.Sprintf("ipinfo: %v", err)
	}
	if resp.StatusCode != 200 {
		return modules.Degraded, fmt.Sprintf("ipinfo: status %d", resp.StatusCode)
	}

	var info struct {
		Org string `json:"org"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &info); err == nil && info.Org != "" {
		return modules.Healthy, "ipinfo: OK"
	}
	return modules.Degraded, "ipinfo: unexpected response format"
}
