// SPDX-License-Identifier: AGPL-3.0-or-later

package whois

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/KyleDerZweite/basalt/internal/graph"
	"github.com/KyleDerZweite/basalt/internal/httpclient"
	"github.com/KyleDerZweite/basalt/internal/modules"
)

const defaultBaseURL = "https://rdap.org"

// Module extracts domain registration data from RDAP (Registration Data Access Protocol).
type Module struct {
	baseURL string
}

// New creates an RDAP/WHOIS module with the default base URL.
func New() *Module {
	return &Module{baseURL: defaultBaseURL}
}

func (m *Module) Name() string        { return "whois" }
func (m *Module) Description() string { return "Extract domain registrant data via RDAP" }

func (m *Module) CanHandle(nodeType string) bool {
	return nodeType == "domain"
}

// rdapResponse represents the relevant fields from an RDAP domain lookup.
type rdapResponse struct {
	LDHName  string       `json:"ldhName"`
	Name     string       `json:"name"`
	Entities []rdapEntity `json:"entities"`
	Events   []rdapEvent  `json:"events"`
	Links    []rdapLink   `json:"links"`
}

type rdapEntity struct {
	Roles      []string      `json:"roles"`
	VCardArray []interface{} `json:"vcardArray"`
}

type rdapEvent struct {
	EventAction string `json:"eventAction"`
	EventDate   string `json:"eventDate"`
}

type rdapLink struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

// vcardContact holds contact info extracted from a vcard array.
type vcardContact struct {
	FullName     string
	Email        string
	Organization string
}

func (m *Module) Extract(ctx context.Context, node *graph.Node, client *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	domain := node.Label
	apiURL := fmt.Sprintf("%s/domain/%s", m.baseURL, url.PathEscape(domain))

	resp, err := client.Do(ctx, apiURL, rdapHeaders())
	if err != nil {
		return nil, nil, fmt.Errorf("rdap request: %w", err)
	}
	if resp.StatusCode == 404 {
		return nil, nil, nil
	}
	if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("rdap returned %d", resp.StatusCode)
	}

	var rdap rdapResponse
	if err := json.Unmarshal([]byte(resp.Body), &rdap); err != nil {
		return nil, nil, fmt.Errorf("parsing rdap response: %w", err)
	}

	domainName := rdap.LDHName
	if domainName == "" {
		domainName = rdap.Name
	}
	if domainName == "" {
		domainName = domain
	}

	var nodes []*graph.Node
	var edges []*graph.Edge

	// Account node for the domain registration.
	profileURL := rdapSelfLink(rdap.Links)
	if profileURL == "" {
		profileURL = apiURL
	}
	account := graph.NewAccountNode("rdap", domainName, profileURL, "whois")
	account.Confidence = 0.85

	// Attach registration/expiration dates as properties.
	for _, evt := range rdap.Events {
		switch evt.EventAction {
		case "registration":
			account.Properties["registration_date"] = evt.EventDate
		case "expiration":
			account.Properties["expiration_date"] = evt.EventDate
		}
	}

	nodes = append(nodes, account)
	edges = append(edges, graph.NewEdge(0, node.ID, account.ID, graph.EdgeTypeRegisteredTo, "whois"))

	// Extract registrant contact data from entities.
	for _, entity := range rdap.Entities {
		if !hasRole(entity.Roles, "registrant") {
			continue
		}
		contact := parseVCard(entity.VCardArray)
		if contact == nil {
			continue
		}
		if contact.FullName != "" {
			nameNode := graph.NewNode(graph.NodeTypeFullName, contact.FullName, "whois")
			nameNode.Confidence = 0.80
			nodes = append(nodes, nameNode)
			edges = append(edges, graph.NewEdge(0, account.ID, nameNode.ID, graph.EdgeTypeLinkedTo, "whois"))
		}
		if contact.Email != "" {
			emailNode := graph.NewNode(graph.NodeTypeEmail, contact.Email, "whois")
			emailNode.Pivot = true
			emailNode.Confidence = 0.80
			nodes = append(nodes, emailNode)
			edges = append(edges, graph.NewEdge(0, account.ID, emailNode.ID, graph.EdgeTypeHasEmail, "whois"))
		}
		if contact.Organization != "" {
			orgNode := graph.NewNode(graph.NodeTypeOrganization, contact.Organization, "whois")
			orgNode.Confidence = 0.80
			nodes = append(nodes, orgNode)
			edges = append(edges, graph.NewEdge(0, account.ID, orgNode.ID, graph.EdgeTypeLinkedTo, "whois"))
		}
	}

	return nodes, edges, nil
}

func (m *Module) Verify(ctx context.Context, client *httpclient.Client) (modules.HealthStatus, string) {
	apiURL := fmt.Sprintf("%s/domain/example.com", m.baseURL)
	resp, err := client.Do(ctx, apiURL, rdapHeaders())
	if err != nil {
		return modules.Offline, fmt.Sprintf("whois: %v", err)
	}
	if resp.StatusCode == 200 {
		var rdap rdapResponse
		if err := json.Unmarshal([]byte(resp.Body), &rdap); err == nil && rdap.LDHName != "" {
			return modules.Healthy, "whois: OK"
		}
		return modules.Degraded, "whois: unexpected response format"
	}
	return modules.Offline, fmt.Sprintf("whois: status %d", resp.StatusCode)
}

// hasRole checks whether a role list contains the target role.
func hasRole(roles []string, target string) bool {
	for _, r := range roles {
		if r == target {
			return true
		}
	}
	return false
}

// rdapSelfLink extracts the "self" link href from the links array.
func rdapSelfLink(links []rdapLink) string {
	for _, l := range links {
		if l.Rel == "self" {
			return l.Href
		}
	}
	return ""
}

// rdapHeaders returns headers appropriate for RDAP requests.
// rdap.org blocks browser-like User-Agents with 403.
func rdapHeaders() map[string]string {
	return map[string]string{
		"Accept":     "application/rdap+json",
		"User-Agent": "basalt/2.0",
	}
}

// parseVCard extracts contact info from an RDAP vcardArray.
// The vcardArray format is: ["vcard", [["version", {}, "text", "4.0"], ["fn", {}, "text", "Name"], ...]]
func parseVCard(vcardArray []interface{}) *vcardContact {
	if len(vcardArray) < 2 {
		return nil
	}
	properties, ok := vcardArray[1].([]interface{})
	if !ok {
		return nil
	}

	contact := &vcardContact{}
	for _, prop := range properties {
		arr, ok := prop.([]interface{})
		if !ok || len(arr) < 4 {
			continue
		}
		propName, _ := arr[0].(string)
		propValue, _ := arr[len(arr)-1].(string)
		if propValue == "" {
			continue
		}

		switch propName {
		case "fn":
			contact.FullName = propValue
		case "email":
			contact.Email = propValue
		case "org":
			contact.Organization = propValue
		}
	}

	if contact.FullName == "" && contact.Email == "" && contact.Organization == "" {
		return nil
	}
	return contact
}
