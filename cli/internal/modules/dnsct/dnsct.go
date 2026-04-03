// SPDX-License-Identifier: AGPL-3.0-or-later

package dnsct

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strings"

	"github.com/KyleDerZweite/basalt/internal/graph"
	"github.com/KyleDerZweite/basalt/internal/httpclient"
	"github.com/KyleDerZweite/basalt/internal/modules"
)

const defaultCTBaseURL = "https://crt.sh"

// Resolver abstracts DNS lookups for testing.
type Resolver interface {
	LookupHost(ctx context.Context, host string) ([]string, error)
	LookupMX(ctx context.Context, name string) ([]*net.MX, error)
}

// netResolver wraps the standard library DNS resolver.
type netResolver struct{}

func (r *netResolver) LookupHost(ctx context.Context, host string) ([]string, error) {
	return net.DefaultResolver.LookupHost(ctx, host)
}

func (r *netResolver) LookupMX(ctx context.Context, name string) ([]*net.MX, error) {
	return net.DefaultResolver.LookupMX(ctx, name)
}

// Module discovers DNS records and Certificate Transparency subdomains for a domain.
type Module struct {
	ctBaseURL string
	resolver  Resolver
}

// New creates a DNS/CT module with default settings.
func New() *Module {
	return &Module{
		ctBaseURL: defaultCTBaseURL,
		resolver:  &netResolver{},
	}
}

func (m *Module) Name() string { return "dnsct" }
func (m *Module) Description() string {
	return "DNS lookups and certificate transparency subdomain discovery"
}

func (m *Module) CanHandle(nodeType string) bool {
	return nodeType == "domain"
}

// ctEntry represents a single Certificate Transparency log entry from crt.sh.
type ctEntry struct {
	NameValue  string `json:"name_value"`
	CommonName string `json:"common_name"`
}

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	domain := node.Label

	// Collect DNS A records.
	aRecords, err := m.resolver.LookupHost(ctx, domain)
	if err != nil {
		slog.Debug("dns A lookup failed", "domain", domain, "error", err)
	}

	// Collect DNS MX records.
	var mxRecords []string
	mxEntries, err := m.resolver.LookupMX(ctx, domain)
	if err != nil {
		slog.Debug("dns MX lookup failed", "domain", domain, "error", err)
	}
	for _, mx := range mxEntries {
		mxRecords = append(mxRecords, strings.TrimSuffix(mx.Host, "."))
	}

	// Query crt.sh for CT subdomains.
	subdomains, ctErr := m.queryCT(ctx, domain, client)
	if ctErr != nil {
		slog.Debug("crt.sh query failed", "domain", domain, "error", ctErr)
	}

	// If everything failed, nothing to report.
	if len(aRecords) == 0 && len(mxRecords) == 0 && len(subdomains) == 0 {
		return nil, nil, nil
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	// Account node representing the domain's DNS/CT footprint.
	account := graph.NewAccountNode("dns", domain, "", "dnsct")
	account.Confidence = 0.90

	if len(aRecords) > 0 {
		account.Properties["a_records"] = aRecords
	}
	if len(mxRecords) > 0 {
		account.Properties["mx_records"] = mxRecords
	}

	nodes = append(nodes, account)
	edges = append(edges, graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeResolvesTo, "dnsct"))

	// Add subdomain nodes from CT logs (informational, no pivoting).
	for _, sub := range subdomains {
		subNode := graph.NewNode(graph.NodeTypeDomain, sub, "dnsct")
		subNode.Confidence = 0.70
		subNode.Pivot = false
		nodes = append(nodes, subNode)
		edges = append(edges, graph.NewEdge(0, account.ID, subNode.ID, graph.EdgeTypeHasDomain, "dnsct"))
	}

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	_, hostErr := m.resolver.LookupHost(ctx, "example.com")
	_, mxErr := m.resolver.LookupMX(ctx, "example.com")
	if hostErr != nil && mxErr != nil {
		return modules.Offline, fmt.Sprintf("dnsct: DNS unavailable (%v / %v)", hostErr, mxErr)
	}

	if _, err := m.queryCT(ctx, "example.com", client); err == nil {
		return modules.Healthy, "dnsct: OK"
	} else {
		return modules.Degraded, fmt.Sprintf("dnsct: DNS OK, crt.sh unavailable (%v)", err)
	}
}

// queryCT fetches Certificate Transparency subdomains from crt.sh.
func (m *Module) queryCT(ctx context.Context, domain string, client *httpclient.Client) ([]string, error) {
	ctURL := fmt.Sprintf("%s/?q=%s&output=json", m.ctBaseURL, url.QueryEscape(domain))

	resp, err := client.Do(ctx, ctURL, nil)
	if err != nil {
		return nil, fmt.Errorf("crt.sh request: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("crt.sh returned %d", resp.StatusCode)
	}

	var entries []ctEntry
	if err := json.Unmarshal([]byte(resp.Body), &entries); err != nil {
		return nil, fmt.Errorf("parsing crt.sh response: %w", err)
	}

	// Deduplicate and filter out wildcard entries.
	seen := make(map[string]bool)
	var subdomains []string
	for _, entry := range entries {
		// crt.sh name_value can contain newlines for multiple names.
		for _, name := range strings.Split(entry.NameValue, "\n") {
			name = strings.TrimSpace(strings.ToLower(name))
			if name == "" || strings.Contains(name, "*") {
				continue
			}
			// Skip the base domain itself.
			if name == strings.ToLower(domain) {
				continue
			}
			if !seen[name] {
				seen[name] = true
				subdomains = append(subdomains, name)
			}
		}
	}

	return subdomains, nil
}
