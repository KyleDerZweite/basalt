// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"fmt"
	"sort"
	"strings"

	"github.com/KyleDerZweite/basalt/internal/graph"
)

// BuildWorkspace loads a target-centric graph and summary for a scan.
func (s *Service) BuildWorkspace(scanID string) (*ScanWorkspace, error) {
	record, err := s.GetScan(scanID)
	if err != nil {
		return nil, err
	}
	var target *Target
	if record.TargetID != "" {
		target, err = s.store.GetTarget(record.TargetID)
		if err != nil {
			return nil, err
		}
	}

	insights := record.Insights
	if insights == nil && record.Graph != nil {
		generated := BuildScanInsights(record.Graph, record.Health, record.Status)
		insights = &generated
	}
	record.Insights = insights

	workspace := &ScanWorkspace{
		Record:            record,
		Target:            target,
		Insights:          insights,
		RawGraphAvailable: record.Graph != nil,
		Graph:             BuildWorkspaceGraph(record, target),
	}
	if record.Graph != nil {
		nodes, edges := record.Graph.Collect()
		workspace.RawNodeCount = len(nodes)
		workspace.RawEdgeCount = len(edges)
	}
	return workspace, nil
}

// BuildScanInsights summarizes the most important findings for a scan.
func BuildScanInsights(g *graph.Graph, health []ModuleStatus, status ScanStatus) ScanInsights {
	insights := ScanInsights{}
	if g == nil {
		if status != "" {
			insights.Warnings = append(insights.Warnings, fmt.Sprintf("Scan status: %s", status))
		}
		return insights
	}

	nodes, edges := g.Collect()
	var accounts []*graph.Node
	identitySignals := make(map[string]struct{})
	infraSignals := make(map[string]struct{})
	var warnings []string

	for _, item := range health {
		if item.Status != "healthy" {
			warnings = append(warnings, fmt.Sprintf("%s: %s", item.Name, item.Message))
		}
	}
	if status == ScanStatusPartial || status == ScanStatusCanceled || status == ScanStatusFailed {
		warnings = append(warnings, fmt.Sprintf("Scan status: %s", status))
	}

	for _, node := range nodes {
		switch node.Type {
		case graph.NodeTypeAccount:
			accounts = append(accounts, node)
		case graph.NodeTypeFullName, graph.NodeTypeUsername, graph.NodeTypeEmail:
			if node.Label != "" {
				identitySignals[node.Label] = struct{}{}
			}
		case graph.NodeTypeWebsite, graph.NodeTypeDomain:
			if node.Label != "" {
				identitySignals[node.Label] = struct{}{}
			}
		case graph.NodeTypeIP, graph.NodeTypeOrganization:
			if node.Label != "" {
				infraSignals[node.Label] = struct{}{}
			}
		}
	}

	sort.Slice(accounts, func(i, j int) bool {
		if accounts[i].Confidence == accounts[j].Confidence {
			return accounts[i].Label < accounts[j].Label
		}
		return accounts[i].Confidence > accounts[j].Confidence
	})

	for _, node := range accounts {
		if node.Confidence < 0.8 {
			continue
		}
		finding := accountFinding(node)
		insights.HighConfidenceAccounts = append(insights.HighConfidenceAccounts, finding)
		if len(insights.TopFindings) < 5 {
			insights.TopFindings = append(insights.TopFindings, finding)
		}
	}
	if len(insights.TopFindings) < 5 {
		extra := topNonAccountFindings(nodes)
		for _, finding := range extra {
			if len(insights.TopFindings) >= 5 {
				break
			}
			insights.TopFindings = append(insights.TopFindings, finding)
		}
	}

	insights.IdentitySignals = mapKeys(identitySignals, 6)
	insights.InfrastructureSummary = mapKeys(infraSignals, 6)
	insights.Warnings = limitStrings(warnings, 6)
	insights.LinkedIdentities = linkIdentities(nodes, edges)
	insights.PossibleMatches = possibleHandleMatches(nodes)

	headlineParts := []string{}
	if len(accounts) > 0 {
		headlineParts = append(headlineParts, fmt.Sprintf("%d likely accounts", len(accounts)))
	}
	if countType(nodes, graph.NodeTypeDomain) > 0 {
		headlineParts = append(headlineParts, fmt.Sprintf("%d domains", countType(nodes, graph.NodeTypeDomain)))
	}
	if countType(nodes, graph.NodeTypeWebsite) > 0 {
		headlineParts = append(headlineParts, fmt.Sprintf("%d websites", countType(nodes, graph.NodeTypeWebsite)))
	}
	if countType(nodes, graph.NodeTypeIP)+countType(nodes, graph.NodeTypeOrganization) > 0 {
		headlineParts = append(headlineParts, fmt.Sprintf("%d infrastructure pivots", countType(nodes, graph.NodeTypeIP)+countType(nodes, graph.NodeTypeOrganization)))
	}
	if len(headlineParts) == 0 {
		headlineParts = append(headlineParts, "No high-signal findings yet")
	}
	insights.Headline = strings.Join(headlineParts, ", ")
	return insights
}

// BuildWorkspaceGraph builds the synthesized UI graph as a radial mindmap.
// The root node sits at the center (depth 0), seeds and category branches
// form the first ring (depth 1), and leaf discoveries sit in the outer
// ring (depth 2).
func BuildWorkspaceGraph(record *ScanRecord, target *Target) WorkspaceGraph {
	graphView := WorkspaceGraph{
		Layout: "concentric",
		Nodes:  []WorkspaceNode{},
		Edges:  []WorkspaceEdge{},
	}

	// Determine root node. Single-seed optimization: when there is no
	// target and exactly one seed, the seed itself becomes the root so
	// it sits at the very center of the mindmap.
	rootID := "scan-root"
	rootLabel := "Scan"
	singleSeedRoot := false
	if target != nil {
		rootID = "target:" + target.ID
		rootLabel = target.DisplayName
	} else if record != nil && len(record.Seeds) == 1 {
		seed := record.Seeds[0]
		rootID = "seed:" + seed.Type + ":" + seed.Value
		rootLabel = seed.Value
		singleSeedRoot = true
	}
	graphView.Nodes = append(graphView.Nodes, WorkspaceNode{
		ID:       rootID,
		Label:    rootLabel,
		Type:     "target",
		Category: "root",
		Depth:    0,
	})

	added := map[string]struct{}{rootID: {}}
	branchAdded := make(map[string]struct{})
	addBranchNode := func(category, label string) string {
		branchID := "category:" + category
		if _, ok := branchAdded[branchID]; ok {
			return branchID
		}
		branchAdded[branchID] = struct{}{}
		added[branchID] = struct{}{}
		graphView.Nodes = append(graphView.Nodes, WorkspaceNode{
			ID:       branchID,
			Label:    label,
			Type:     "category",
			Category: category,
			Depth:    1,
		})
		graphView.Edges = append(graphView.Edges, WorkspaceEdge{
			ID:     "edge:" + rootID + ":" + branchID,
			Source: rootID,
			Target: branchID,
			Type:   "branch",
		})
		return branchID
	}
	addItemNode := func(parentID string, node WorkspaceNode, edgeType string) {
		if _, ok := added[node.ID]; ok {
			return
		}
		added[node.ID] = struct{}{}
		graphView.Nodes = append(graphView.Nodes, node)
		graphView.Edges = append(graphView.Edges, WorkspaceEdge{
			ID:     "edge:" + parentID + ":" + node.ID,
			Source: parentID,
			Target: node.ID,
			Type:   edgeType,
		})
	}

	// Connect seeds/aliases directly to root (no intermediate branch).
	if target != nil {
		for _, alias := range target.Aliases {
			aliasNodeID := "alias:" + alias.ID
			label := alias.SeedValue
			if alias.Label != "" {
				label = alias.Label + " (" + alias.SeedValue + ")"
			}
			addItemNode(rootID, WorkspaceNode{
				ID:       aliasNodeID,
				Label:    label,
				Type:     alias.SeedType,
				Category: "seed",
				Depth:    1,
			}, "seed")
		}
	} else if record != nil && !singleSeedRoot {
		// Multiple seeds — connect each directly to root.
		for _, seed := range record.Seeds {
			nodeID := "seed:" + seed.Type + ":" + seed.Value
			addItemNode(rootID, WorkspaceNode{
				ID:       nodeID,
				Label:    seed.Value,
				Type:     seed.Type,
				Category: "seed",
				Depth:    1,
			}, "seed")
		}
	}

	if record == nil || record.Graph == nil {
		if record != nil && record.Insights != nil {
			branchID := ""
			for index, warning := range record.Insights.Warnings {
				if branchID == "" {
					branchID = addBranchNode("warnings", "Warnings")
				}
				warningID := fmt.Sprintf("warning:%d", index)
				addItemNode(branchID, WorkspaceNode{
					ID:       warningID,
					Label:    warning,
					Type:     "warning",
					Category: "warnings",
					Depth:    2,
				}, "warning")
			}
		}
		return graphView
	}

	nodes, _ := record.Graph.Collect()
	grouped := groupWorkspaceNodes(nodes)

	appendRawNodes(addBranchNode, addItemNode, "accounts", "Accounts", topNodes(grouped.accounts, 30), "account")
	appendRawNodes(addBranchNode, addItemNode, "identity", "Identity signals", topNodes(dedupeWorkspaceNodes(grouped.identity), 20), "signal")
	appendRawNodes(addBranchNode, addItemNode, "web", "Websites & domains", topNodes(dedupeWorkspaceNodes(grouped.web), 20), "asset")

	var people []*graph.Node
	for _, node := range nodes {
		if node.Type != graph.NodeTypeUsername {
			continue
		}
		if relationship, _ := node.Properties["relationship"].(string); relationship != "" {
			people = append(people, node)
		}
	}
	if len(people) > 0 {
		sortWorkspaceNodes(people)
		dedupedPeople := dedupeWorkspaceNodes(people)
		branchID := addBranchNode("people", "Friends & follows")
		visiblePeople := topNodes(dedupedPeople, 30)
		for _, node := range visiblePeople {
			addItemNode(branchID, workspaceNodeFromRaw(node, "people"), "social")
		}
		if hidden := len(dedupedPeople) - len(visiblePeople); hidden > 0 {
			addItemNode(branchID, WorkspaceNode{
				ID:             "summary:people",
				Label:          "Additional connections",
				Type:           "summary",
				Category:       "people",
				Depth:          2,
				CollapsedCount: hidden,
			}, "summary")
		}
	}

	if len(grouped.infra) > 0 {
		branchID := addBranchNode("infra", "Infrastructure")
		dedupedInfra := dedupeWorkspaceNodes(grouped.infra)
		visibleInfra := topNodes(dedupedInfra, 12)
		for _, node := range visibleInfra {
			addItemNode(branchID, workspaceNodeFromRaw(node, "infra"), "infra")
		}
		if hidden := len(dedupedInfra) - len(visibleInfra); hidden > 0 {
			addItemNode(branchID, WorkspaceNode{
				ID:             "summary:infra",
				Label:          "Additional infrastructure pivots",
				Type:           "summary",
				Category:       "infra",
				Depth:          2,
				CollapsedCount: hidden,
				RawNodeIDs:     collectNodeIDs(dedupedInfra[len(visibleInfra):]),
			}, "summary")
		}
	}

	if record.Insights != nil && len(record.Insights.Warnings) > 0 {
		branchID := addBranchNode("warnings", "Warnings")
		for index, warning := range limitStrings(record.Insights.Warnings, 3) {
			warningID := fmt.Sprintf("warning:%d", index)
			addItemNode(branchID, WorkspaceNode{
				ID:       warningID,
				Label:    warning,
				Type:     "warning",
				Category: "warnings",
				Depth:    2,
			}, "warning")
		}
		if hidden := len(record.Insights.Warnings) - minInt(len(record.Insights.Warnings), 3); hidden > 0 {
			addItemNode(branchID, WorkspaceNode{
				ID:             "summary:warnings",
				Label:          "Additional warnings",
				Type:           "summary",
				Category:       "warnings",
				Depth:          2,
				CollapsedCount: hidden,
			}, "summary")
		}
	}

	return graphView
}

type workspaceGroups struct {
	accounts []*graph.Node
	identity []*graph.Node
	web      []*graph.Node
	infra    []*graph.Node
}

func groupWorkspaceNodes(nodes []*graph.Node) workspaceGroups {
	grouped := workspaceGroups{}
	for _, node := range nodes {
		switch workspaceCategory(node) {
		case "accounts":
			grouped.accounts = append(grouped.accounts, node)
		case "identity":
			grouped.identity = append(grouped.identity, node)
		case "web":
			grouped.web = append(grouped.web, node)
		case "infra":
			grouped.infra = append(grouped.infra, node)
		}
	}
	sortWorkspaceNodes(grouped.accounts)
	sortWorkspaceNodes(grouped.identity)
	sortWorkspaceNodes(grouped.web)
	sortWorkspaceNodes(grouped.infra)
	return grouped
}

func sortWorkspaceNodes(nodes []*graph.Node) {
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Confidence == nodes[j].Confidence {
			return strings.ToLower(nodes[i].Label) < strings.ToLower(nodes[j].Label)
		}
		return nodes[i].Confidence > nodes[j].Confidence
	})
}

func topNodes(nodes []*graph.Node, limit int) []*graph.Node {
	if limit <= 0 || len(nodes) <= limit {
		return nodes
	}
	return nodes[:limit]
}

func dedupeWorkspaceNodes(nodes []*graph.Node) []*graph.Node {
	if len(nodes) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(nodes))
	out := make([]*graph.Node, 0, len(nodes))
	for _, node := range nodes {
		key := node.Type + ":" + strings.ToLower(node.Label)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, node)
	}
	return out
}

func appendRawNodes(
	addBranchNode func(category, label string) string,
	addItemNode func(branchID string, node WorkspaceNode, edgeType string),
	category string,
	label string,
	nodes []*graph.Node,
	edgeType string,
) {
	if len(nodes) == 0 {
		return
	}
	branchID := addBranchNode(category, label)
	for _, node := range nodes {
		addItemNode(branchID, workspaceNodeFromRaw(node, category), edgeType)
	}
}

func workspaceNodeFromRaw(node *graph.Node, category string) WorkspaceNode {
	out := WorkspaceNode{
		ID:         "raw:" + node.ID,
		Label:      node.Label,
		Type:       node.Type,
		Category:   category,
		Depth:      2,
		RawNodeIDs: []string{node.ID},
		ProfileURL: stringProperty(node.Properties, "profile_url"),
		Confidence: node.Confidence,
		Wave:       node.Wave,
		Pivot:      node.Pivot,
	}
	if node.SourceModule != "" {
		out.SourceModules = []string{node.SourceModule}
	}
	out.Properties = flattenProperties(node.Properties)
	return out
}

// flattenProperties copies scalar evidence properties into display strings.
// Compound values are skipped since the inspector cannot render them.
// Output is capped so a single noisy module cannot bloat the payload.
func flattenProperties(values map[string]interface{}) map[string]string {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		switch value := values[key].(type) {
		case string:
			if value != "" {
				out[key] = value
			}
		case bool:
			if value {
				out[key] = "true"
			} else {
				out[key] = "false"
			}
		case float64:
			out[key] = strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", value), "0"), ".")
		}
		if len(out) >= 12 {
			break
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func workspaceCategory(node *graph.Node) string {
	switch node.Type {
	case graph.NodeTypeAccount:
		return "accounts"
	case graph.NodeTypeWebsite, graph.NodeTypeDomain:
		return "web"
	case graph.NodeTypeIP, graph.NodeTypeOrganization:
		return "infra"
	case graph.NodeTypeFullName, graph.NodeTypeUsername, graph.NodeTypeEmail, graph.NodeTypePhone:
		return "identity"
	default:
		return ""
	}
}

func collectNodeIDs(nodes []*graph.Node) []string {
	if len(nodes) == 0 {
		return nil
	}
	ids := make([]string, 0, len(nodes))
	for _, node := range nodes {
		ids = append(ids, node.ID)
	}
	return ids
}

func accountFinding(node *graph.Node) InsightFinding {
	return InsightFinding{
		Title:       node.Label,
		Summary:     summaryForAccount(node),
		NodeIDs:     []string{node.ID},
		ProfileURL:  stringProperty(node.Properties, "profile_url"),
		Confidence:  node.Confidence,
		Category:    "account",
		SourceLabel: stringProperty(node.Properties, "site_name"),
	}
}

func topNonAccountFindings(nodes []*graph.Node) []InsightFinding {
	var findings []InsightFinding
	for _, node := range nodes {
		switch node.Type {
		case graph.NodeTypeDomain, graph.NodeTypeWebsite, graph.NodeTypeFullName:
		default:
			continue
		}
		findings = append(findings, InsightFinding{
			Title:      node.Label,
			Summary:    fmt.Sprintf("%s discovered via %s", node.Type, node.SourceModule),
			NodeIDs:    []string{node.ID},
			Confidence: node.Confidence,
			Category:   node.Type,
		})
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Confidence == findings[j].Confidence {
			return findings[i].Title < findings[j].Title
		}
		return findings[i].Confidence > findings[j].Confidence
	})
	return findings
}

func summaryForAccount(node *graph.Node) string {
	siteName := stringProperty(node.Properties, "site_name")
	if siteName == "" {
		siteName = node.SourceModule
	}
	parts := []string{fmt.Sprintf("High-confidence account on %s", siteName)}
	if fullName := stringProperty(node.Properties, "full_name"); fullName != "" {
		parts = append(parts, "name: "+fullName)
	}
	if location := stringProperty(node.Properties, "location"); location != "" {
		parts = append(parts, "location: "+location)
	}
	return strings.Join(parts, ", ")
}

// linkIdentities groups accounts that share one pivot identifier.
// Only identifiers claimed by at least two accounts are returned,
// ordered by group size. Output is capped to keep payloads small.
func linkIdentities(nodes []*graph.Node, edges []*graph.Edge) []LinkedIdentity {
	byID := make(map[string]*graph.Node, len(nodes))
	for _, node := range nodes {
		byID[node.ID] = node
	}

	type group struct {
		identifier string
		kind       string
		accounts   map[string]*graph.Node
		nodeIDs    map[string]struct{}
		modules    map[string]struct{}
	}
	groups := make(map[string]*group)
	for _, edge := range edges {
		target, ok := byID[edge.Target]
		if !ok {
			continue
		}
		switch target.Type {
		case graph.NodeTypeEmail, graph.NodeTypeUsername, graph.NodeTypeDomain, graph.NodeTypePhone:
		default:
			continue
		}
		source, ok := byID[edge.Source]
		if !ok || source.Type != graph.NodeTypeAccount {
			continue
		}
		key := target.Type + ":" + strings.ToLower(target.Label)
		entry := groups[key]
		if entry == nil {
			entry = &group{
				identifier: target.Label,
				kind:       target.Type,
				accounts:   make(map[string]*graph.Node),
				nodeIDs:    make(map[string]struct{}),
				modules:    make(map[string]struct{}),
			}
			groups[key] = entry
		}
		entry.accounts[source.ID] = source
		entry.nodeIDs[target.ID] = struct{}{}
		if source.SourceModule != "" {
			entry.modules[source.SourceModule] = struct{}{}
		}
	}

	out := make([]LinkedIdentity, 0, len(groups))
	for _, entry := range groups {
		if len(entry.accounts) < 2 {
			continue
		}
		labels := make([]string, 0, len(entry.accounts))
		nodeIDs := make([]string, 0, len(entry.nodeIDs)+1)
		for _, account := range entry.accounts {
			labels = append(labels, account.Label)
			nodeIDs = append(nodeIDs, account.ID)
		}
		for id := range entry.nodeIDs {
			nodeIDs = append(nodeIDs, id)
		}
		modules := make([]string, 0, len(entry.modules))
		for module := range entry.modules {
			modules = append(modules, module)
		}
		sort.Strings(labels)
		sort.Strings(modules)
		sort.Strings(nodeIDs)
		out = append(out, LinkedIdentity{
			Identifier:     entry.identifier,
			IdentifierType: entry.kind,
			AccountLabels:  labels,
			NodeIDs:        nodeIDs,
			Modules:        modules,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if len(out[i].AccountLabels) == len(out[j].AccountLabels) {
			return out[i].Identifier < out[j].Identifier
		}
		return len(out[i].AccountLabels) > len(out[j].AccountLabels)
	})
	if len(out) > 8 {
		return out[:8]
	}
	return out
}

// possibleHandleMatches flags handles that look like they belong to one
// person: exact matches after normalization across different modules,
// then small edit distances. Results carry the reason so the operator
// can judge them instead of trusting a score.
func possibleHandleMatches(nodes []*graph.Node) []PossibleMatch {
	type candidate struct {
		label  string
		norm   string
		module string
		id     string
	}
	var cands []candidate
	for _, node := range nodes {
		if node.Type != graph.NodeTypeAccount && node.Type != graph.NodeTypeUsername {
			continue
		}
		// Account labels carry a platform prefix ("github - kyledev").
		// Match on the handle portion so the same handle on two
		// platforms compares equal.
		handle := node.Label
		if parts := strings.SplitN(node.Label, " - ", 2); len(parts) == 2 {
			handle = parts[1]
		}
		norm := normalizeHandle(handle)
		if len(norm) < 4 {
			continue
		}
		cands = append(cands, candidate{label: node.Label, norm: norm, module: node.SourceModule, id: node.ID})
	}

	var out []PossibleMatch
	byNorm := make(map[string][]candidate)
	for _, cand := range cands {
		byNorm[cand.norm] = append(byNorm[cand.norm], cand)
	}
	for _, group := range byNorm {
		modules := make(map[string]struct{})
		for _, cand := range group {
			modules[cand.module] = struct{}{}
		}
		if len(modules) < 2 {
			continue
		}
		labels := make([]string, 0, len(group))
		ids := make([]string, 0, len(group))
		names := make([]string, 0, len(modules))
		for _, cand := range group {
			labels = append(labels, cand.label)
			ids = append(ids, cand.id)
		}
		for module := range modules {
			names = append(names, module)
		}
		sort.Strings(labels)
		sort.Strings(names)
		sort.Strings(ids)
		out = append(out, PossibleMatch{
			LabelA:     labels[0],
			LabelB:     labels[len(labels)-1],
			Reason:     "Same handle on " + strings.Join(names, ", "),
			NodeIDs:    ids,
			Confidence: 0.85,
		})
	}

	if len(cands) <= 300 {
		seen := make(map[string]struct{})
		for i := 0; i < len(cands); i++ {
			for j := i + 1; j < len(cands); j++ {
				a, b := cands[i], cands[j]
				if a.norm == b.norm || a.module == b.module {
					continue
				}
				if absInt(len(a.norm)-len(b.norm)) > 1 {
					continue
				}
				distance := editDistance(a.norm, b.norm)
				if distance == 0 || distance > 2 {
					continue
				}
				key := a.norm + "\x00" + b.norm
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				out = append(out, PossibleMatch{
					LabelA:     a.label,
					LabelB:     b.label,
					Reason:     fmt.Sprintf("Handles differ by %d character%s", distance, pluralS(distance)),
					NodeIDs:    []string{a.id, b.id},
					Confidence: 0.55,
				})
				if len(out) >= 16 {
					break
				}
			}
			if len(out) >= 16 {
				break
			}
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Confidence == out[j].Confidence {
			return out[i].LabelA < out[j].LabelA
		}
		return out[i].Confidence > out[j].Confidence
	})
	if len(out) > 8 {
		return out[:8]
	}
	return out
}

// normalizeHandle lowercases a handle and drops separators so
// "Kyle_DerZweite" and "kylederzweite" compare equal.
func normalizeHandle(label string) string {
	var builder strings.Builder
	for _, r := range strings.ToLower(label) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

// editDistance returns the Levenshtein distance between two short strings.
func editDistance(a, b string) int {
	ar, br := []rune(a), []rune(b)
	previous := make([]int, len(br)+1)
	for j := range previous {
		previous[j] = j
	}
	for i := 1; i <= len(ar); i++ {
		current := make([]int, len(br)+1)
		current[0] = i
		for j := 1; j <= len(br); j++ {
			cost := 0
			if ar[i-1] != br[j-1] {
				cost = 1
			}
			current[j] = minInt3(previous[j]+1, current[j-1]+1, previous[j-1]+cost)
		}
		previous = current
	}
	return previous[len(br)]
}

func minInt3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func pluralS(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

func stringProperty(values map[string]interface{}, key string) string {
	if values == nil {
		return ""
	}
	raw, ok := values[key]
	if !ok {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return value
}

func countType(nodes []*graph.Node, nodeType string) int {
	count := 0
	for _, node := range nodes {
		if node.Type == nodeType {
			count++
		}
	}
	return count
}

func mapKeys(values map[string]struct{}, limit int) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return limitStrings(out, limit)
}

func limitStrings(values []string, limit int) []string {
	if len(values) == 0 {
		return nil
	}
	if limit <= 0 || len(values) <= limit {
		return values
	}
	return values[:limit]
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
