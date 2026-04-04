// SPDX-License-Identifier: AGPL-3.0-or-later

package securitytxt

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/KyleDerZweite/basalt/internal/graph"
	"github.com/KyleDerZweite/basalt/internal/httpclient"
	"github.com/KyleDerZweite/basalt/internal/modules"
)

const (
	defaultBaseURL = "https://%s"
	verifyDomain   = "securitytxt.org"
)

// Module extracts contacts and related URLs from security.txt files.
type Module struct {
	baseURL string
}

// New creates a security.txt module.
func New() *Module {
	return &Module{baseURL: defaultBaseURL}
}

func (m *Module) Name() string { return "securitytxt" }
func (m *Module) Description() string {
	return "Extract security contacts and disclosure URLs from security.txt"
}

func (m *Module) CanHandle(nodeType string) bool {
	return nodeType == graph.NodeTypeDomain
}

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	domain := strings.TrimSpace(node.Label)
	if domain == "" {
		return nil, nil, nil
	}

	data, fetchedURL, err := m.fetch(ctx, domain, client)
	if err != nil {
		return nil, nil, err
	}
	if data == nil {
		return nil, nil, nil
	}

	collector := newCollector()
	account := graph.NewAccountNode("securitytxt", domain, fetchedURL, "securitytxt")
	account.Confidence = 0.90
	setSliceProperty(account.Properties, "contacts", data.Contacts)
	setSliceProperty(account.Properties, "expires", data.Expires)
	setSliceProperty(account.Properties, "preferred_languages", data.PreferredLanguages)
	setSliceProperty(account.Properties, "canonical_urls", data.CanonicalURLs)
	setSliceProperty(account.Properties, "policy_urls", data.PolicyURLs)
	setSliceProperty(account.Properties, "acknowledgments_urls", data.AcknowledgmentsURLs)
	setSliceProperty(account.Properties, "hiring_urls", data.HiringURLs)
	setSliceProperty(account.Properties, "encryption_urls", data.EncryptionURLs)
	account.Properties["fetched_url"] = fetchedURL

	account = collector.addNode(account)
	collector.addEdge(graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeHasAccount, "securitytxt"))

	for _, contact := range data.Contacts {
		m.addContactTargets(collector, account.ID, domain, contact)
	}

	for _, website := range data.CanonicalURLs {
		m.addWebsiteTarget(collector, account.ID, domain, website, 0.70, false)
	}
	for _, website := range data.PolicyURLs {
		m.addWebsiteTarget(collector, account.ID, domain, website, 0.70, false)
	}
	for _, website := range data.AcknowledgmentsURLs {
		m.addWebsiteTarget(collector, account.ID, domain, website, 0.70, false)
	}
	for _, website := range data.HiringURLs {
		m.addWebsiteTarget(collector, account.ID, domain, website, 0.70, false)
	}
	for _, website := range data.EncryptionURLs {
		m.addWebsiteTarget(collector, account.ID, domain, website, 0.70, false)
	}

	return collector.nodes, collector.edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	verifyURL := joinURL(m.domainBaseURL(verifyDomain), "/.well-known/security.txt")

	resp, err := client.Do(ctx, verifyURL, map[string]string{"Accept": "text/plain"})
	if err != nil {
		return modules.Offline, fmt.Sprintf("securitytxt: %v", err)
	}
	if resp.StatusCode != 200 {
		return modules.Offline, fmt.Sprintf("securitytxt: status %d", resp.StatusCode)
	}

	data, fieldCount := parseSecurityTXT(resp.Body)
	if fieldCount == 0 {
		return modules.Degraded, "securitytxt: unexpected response format"
	}
	if len(data.Contacts) == 0 || len(data.Expires) == 0 {
		return modules.Degraded, "securitytxt: missing required fields"
	}

	return modules.Healthy, "securitytxt: OK"
}

func (m *Module) fetch(ctx context.Context, domain string, client *httpclient.Client) (*securityData, string, error) {
	baseURL := m.domainBaseURL(domain)
	paths := []string{"/.well-known/security.txt", "/security.txt"}

	for _, path := range paths {
		targetURL := joinURL(baseURL, path)
		resp, err := client.Do(ctx, targetURL, map[string]string{"Accept": "text/plain"})
		if err != nil {
			return nil, "", fmt.Errorf("securitytxt request: %w", err)
		}
		if resp.StatusCode == 404 {
			continue
		}
		if resp.StatusCode != 200 {
			return nil, "", fmt.Errorf("securitytxt returned %d", resp.StatusCode)
		}

		data, fieldCount := parseSecurityTXT(resp.Body)
		if fieldCount == 0 {
			continue
		}
		return &data, targetURL, nil
	}

	return nil, "", nil
}

func (m *Module) addContactTargets(collector *collector, accountID, seedDomain, raw string) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return
	}

	switch {
	case strings.HasPrefix(strings.ToLower(value), "mailto:"):
		for _, email := range mailtoAddresses(value) {
			emailNode := graph.NewNode(graph.NodeTypeEmail, email, "securitytxt")
			emailNode.Pivot = true
			emailNode.Confidence = 0.90
			emailNode = collector.addNode(emailNode)
			collector.addEdge(graph.NewEdge(0, accountID, emailNode.ID, graph.EdgeTypeHasEmail, "securitytxt"))
		}
	case isHTTPURL(value):
		m.addWebsiteTarget(collector, accountID, seedDomain, value, 0.80, true)
	}
}

func (m *Module) addWebsiteTarget(collector *collector, accountID, seedDomain, raw string, confidence float64, pivot bool) {
	parsed, ok := normalizeHTTPURL(raw)
	if !ok {
		return
	}

	websiteNode := graph.NewNode(graph.NodeTypeWebsite, parsed.String(), "securitytxt")
	websiteNode.Confidence = confidence
	websiteNode.Pivot = pivot
	websiteNode = collector.addNode(websiteNode)
	collector.addEdge(graph.NewEdge(0, accountID, websiteNode.ID, graph.EdgeTypeLinkedTo, "securitytxt"))

	host := normalizedHost(parsed.Hostname())
	if host == "" || host == normalizedHost(seedDomain) {
		return
	}

	domainNode := graph.NewNode(graph.NodeTypeDomain, host, "securitytxt")
	domainNode.Confidence = confidence
	domainNode.Pivot = true
	domainNode = collector.addNode(domainNode)
	collector.addEdge(graph.NewEdge(0, accountID, domainNode.ID, graph.EdgeTypeHasDomain, "securitytxt"))
}

func (m *Module) domainBaseURL(domain string) string {
	if strings.Contains(m.baseURL, "%s") {
		return strings.TrimRight(fmt.Sprintf(m.baseURL, domain), "/")
	}
	return strings.TrimRight(m.baseURL, "/")
}

type securityData struct {
	Contacts            []string
	Expires             []string
	PreferredLanguages  []string
	CanonicalURLs       []string
	PolicyURLs          []string
	AcknowledgmentsURLs []string
	HiringURLs          []string
	EncryptionURLs      []string
}

type collector struct {
	nodes    []*graph.Node
	edges    []*graph.Edge
	nodeByID map[string]*graph.Node
	edgeSeen map[string]struct{}
}

func newCollector() *collector {
	return &collector{
		nodeByID: make(map[string]*graph.Node),
		edgeSeen: make(map[string]struct{}),
	}
}

func (c *collector) addNode(node *graph.Node) *graph.Node {
	if existing, ok := c.nodeByID[node.ID]; ok {
		if node.Confidence > existing.Confidence {
			existing.Confidence = node.Confidence
		}
		existing.Pivot = existing.Pivot || node.Pivot
		for key, value := range node.Properties {
			if _, exists := existing.Properties[key]; !exists {
				existing.Properties[key] = value
			}
		}
		return existing
	}

	c.nodeByID[node.ID] = node
	c.nodes = append(c.nodes, node)
	return node
}

func (c *collector) addEdge(edge *graph.Edge) {
	key := edge.Source + "\x00" + edge.Target + "\x00" + edge.Type
	if _, ok := c.edgeSeen[key]; ok {
		return
	}
	c.edgeSeen[key] = struct{}{}
	c.edges = append(c.edges, edge)
}

func parseSecurityTXT(body string) (securityData, int) {
	var data securityData
	var fieldCount int

	body = strings.TrimPrefix(body, "\uFEFF")
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		name, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}

		name = strings.ToLower(strings.TrimSpace(name))
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		switch name {
		case "contact":
			data.Contacts = append(data.Contacts, value)
		case "expires":
			data.Expires = append(data.Expires, value)
		case "preferred-languages":
			data.PreferredLanguages = append(data.PreferredLanguages, value)
		case "canonical":
			data.CanonicalURLs = append(data.CanonicalURLs, value)
		case "policy":
			data.PolicyURLs = append(data.PolicyURLs, value)
		case "acknowledgments":
			data.AcknowledgmentsURLs = append(data.AcknowledgmentsURLs, value)
		case "hiring":
			data.HiringURLs = append(data.HiringURLs, value)
		case "encryption":
			data.EncryptionURLs = append(data.EncryptionURLs, value)
		default:
			continue
		}
		fieldCount++
	}

	return data, fieldCount
}

func setSliceProperty(props map[string]interface{}, key string, values []string) {
	if len(values) == 0 {
		return
	}
	copied := append([]string(nil), values...)
	props[key] = copied
}

func mailtoAddresses(raw string) []string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil
	}
	if !strings.EqualFold(parsed.Scheme, "mailto") {
		return nil
	}

	value := parsed.Opaque
	if value == "" {
		value = parsed.Path
	}
	if value == "" {
		return nil
	}

	var addresses []string
	seen := make(map[string]struct{})
	for _, part := range strings.Split(value, ",") {
		email, err := url.PathUnescape(strings.TrimSpace(part))
		if err != nil {
			continue
		}
		email = strings.TrimSpace(email)
		if email == "" {
			continue
		}
		if _, ok := seen[email]; ok {
			continue
		}
		seen[email] = struct{}{}
		addresses = append(addresses, email)
	}

	return addresses
}

func isHTTPURL(raw string) bool {
	parsed, ok := normalizeHTTPURL(raw)
	return ok && parsed != nil
}

func normalizeHTTPURL(raw string) (*url.URL, bool) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, false
	}
	if parsed.Host == "" {
		return nil, false
	}
	return parsed, true
}

func normalizedHost(host string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
}

func joinURL(baseURL, path string) string {
	return strings.TrimRight(baseURL, "/") + path
}
