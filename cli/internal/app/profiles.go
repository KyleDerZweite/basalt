// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"fmt"
	"sort"
	"strings"

	"github.com/KyleDerZweite/basalt/internal/graph"
)

// buildPersonProfiles clusters accounts into per-person profiles.
// Accounts merge when they share a pivot identifier (email, handle,
// domain, phone) or carry the same normalized handle. Identifiers,
// names, and avatars adjacent to member accounts are collected onto
// the profile. Output is capped and ordered by confidence.
func buildPersonProfiles(nodes []*graph.Node, edges []*graph.Edge) []PersonProfile {
	byID := make(map[string]*graph.Node, len(nodes))
	var accounts []*graph.Node
	for _, node := range nodes {
		byID[node.ID] = node
		if node.Type == graph.NodeTypeAccount {
			accounts = append(accounts, node)
		}
	}
	if len(accounts) == 0 {
		return nil
	}

	sets := newDisjointSets()
	for _, account := range accounts {
		sets.add(account.ID)
	}

	adjacent := make(map[string]map[string]*graph.Node)
	link := func(a, b string) {
		other, ok := byID[b]
		if !ok {
			return
		}
		if adjacent[a] == nil {
			adjacent[a] = make(map[string]*graph.Node)
		}
		adjacent[a][b] = other
	}
	identifierAccounts := make(map[string]map[string]struct{})
	for _, edge := range edges {
		source, ok := byID[edge.Source]
		if !ok {
			continue
		}
		target, ok := byID[edge.Target]
		if !ok {
			continue
		}
		link(edge.Source, edge.Target)
		link(edge.Target, edge.Source)
		for _, node := range []*graph.Node{source, target} {
			other := source
			if node == source {
				other = target
			}
			if node.Type != graph.NodeTypeAccount {
				continue
			}
			switch other.Type {
			case graph.NodeTypeEmail, graph.NodeTypeUsername, graph.NodeTypeDomain, graph.NodeTypePhone:
				key := other.Type + ":" + strings.ToLower(other.Label)
				if identifierAccounts[key] == nil {
					identifierAccounts[key] = make(map[string]struct{})
				}
				identifierAccounts[key][node.ID] = struct{}{}
			}
		}
	}
	for _, members := range identifierAccounts {
		mergeAll(sets, keys(members))
	}

	byHandle := make(map[string][]string)
	for _, account := range accounts {
		handle := accountHandle(account.Label)
		if len(normalizeHandle(handle)) < 4 {
			continue
		}
		byHandle[normalizeHandle(handle)] = append(byHandle[normalizeHandle(handle)], account.ID)
	}
	for _, members := range byHandle {
		mergeAll(sets, members)
	}

	clusters := make(map[string][]*graph.Node)
	for _, account := range accounts {
		root := sets.find(account.ID)
		clusters[root] = append(clusters[root], account)
	}

	out := make([]PersonProfile, 0, len(clusters))
	for _, members := range clusters {
		out = append(out, buildProfile(members, adjacent))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Confidence == out[j].Confidence {
			return out[i].PrimaryHandle < out[j].PrimaryHandle
		}
		return out[i].Confidence > out[j].Confidence
	})
	if len(out) > 10 {
		return out[:10]
	}
	return out
}

// buildProfile assembles one profile from a cluster of accounts.
func buildProfile(members []*graph.Node, adjacent map[string]map[string]*graph.Node) PersonProfile {
	sort.Slice(members, func(i, j int) bool {
		if members[i].Confidence == members[j].Confidence {
			return members[i].Label < members[j].Label
		}
		return members[i].Confidence > members[j].Confidence
	})

	primary := accountHandle(members[0].Label)
	usernames := make(map[string]struct{})
	names := make(map[string]struct{})
	emails := make(map[string]struct{})
	domains := make(map[string]struct{})
	modules := make(map[string]struct{})
	avatars := make(map[string]float64)
	nodeIDs := make(map[string]struct{})
	var reasons []string
	sharedEvidence := 0

	for _, member := range members {
		nodeIDs[member.ID] = struct{}{}
		if member.SourceModule != "" {
			modules[member.SourceModule] = struct{}{}
		}
		if handle := accountHandle(member.Label); handle != "" {
			usernames[handle] = struct{}{}
		}
		for _, neighbor := range adjacent[member.ID] {
			nodeIDs[neighbor.ID] = struct{}{}
			switch neighbor.Type {
			case graph.NodeTypeUsername:
				if handle := accountHandle(neighbor.Label); handle != "" {
					usernames[handle] = struct{}{}
				}
			case graph.NodeTypeFullName:
				if neighbor.Label != "" {
					names[neighbor.Label] = struct{}{}
				}
			case graph.NodeTypeEmail:
				if neighbor.Label != "" {
					emails[neighbor.Label] = struct{}{}
				}
			case graph.NodeTypeDomain, graph.NodeTypeWebsite:
				if host := neighbor.Label; host != "" {
					domains[host] = struct{}{}
				}
			case graph.NodeTypeAvatarURL:
				if neighbor.Label != "" {
					if confidence, ok := avatars[neighbor.Label]; !ok || neighbor.Confidence > confidence {
						avatars[neighbor.Label] = neighbor.Confidence
					}
				}
			}
		}
	}

	confidence := members[0].Confidence
	if len(members) > 1 {
		sharedEvidence += len(members) - 1
		reasons = append(reasons, fmt.Sprintf("%d linked accounts share identifiers", len(members)))
	} else {
		reasons = append(reasons, fmt.Sprintf("Single account on %s", members[0].SourceModule))
	}
	if len(emails) > 0 {
		sharedEvidence++
		reasons = append(reasons, fmt.Sprintf("%d email address%s attached", len(emails), pluralS(len(emails))))
	}
	confidence += 0.05 * float64(sharedEvidence)
	if confidence > 0.97 {
		confidence = 0.97
	}

	avatarURLs := make([]string, 0, len(avatars))
	for url := range avatars {
		avatarURLs = append(avatarURLs, url)
	}
	sort.Slice(avatarURLs, func(i, j int) bool {
		if avatars[avatarURLs[i]] == avatars[avatarURLs[j]] {
			return avatarURLs[i] < avatarURLs[j]
		}
		return avatars[avatarURLs[i]] > avatars[avatarURLs[j]]
	})
	if len(avatarURLs) > 4 {
		avatarURLs = avatarURLs[:4]
	}

	return PersonProfile{
		ID:            "profile:" + normalizeHandle(primary),
		PrimaryHandle: primary,
		Usernames:     sortedKeys(usernames),
		Names:         sortedKeys(names),
		Emails:        sortedKeys(emails),
		AvatarURLs:    avatarURLs,
		Domains:       sortedKeys(domains),
		Modules:       sortedKeys(modules),
		Confidence:    confidence,
		Reasons:       reasons,
		NodeIDs:       sortedKeys(nodeIDs),
	}
}

// accountHandle strips the platform prefix from account labels
// ("github - kyledev" becomes "kyledev").
func accountHandle(label string) string {
	if parts := strings.SplitN(label, " - ", 2); len(parts) == 2 {
		return parts[1]
	}
	return label
}

// disjointSets is a union-find over account IDs.
type disjointSets struct {
	parent map[string]string
}

func newDisjointSets() *disjointSets {
	return &disjointSets{parent: make(map[string]string)}
}

func (d *disjointSets) add(id string) {
	if _, ok := d.parent[id]; !ok {
		d.parent[id] = id
	}
}

func (d *disjointSets) find(id string) string {
	root, ok := d.parent[id]
	if !ok {
		return id
	}
	for root != d.parent[root] {
		d.parent[root] = d.parent[d.parent[root]]
		root = d.parent[root]
	}
	return root
}

func (d *disjointSets) union(a, b string) {
	d.add(a)
	d.add(b)
	d.parent[d.find(a)] = d.find(b)
}

func mergeAll(sets *disjointSets, members []string) {
	for i := 1; i < len(members); i++ {
		sets.union(members[0], members[i])
	}
}

func keys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	return out
}

func sortedKeys(set map[string]struct{}) []string {
	if len(set) == 0 {
		return nil
	}
	out := keys(set)
	sort.Strings(out)
	return out
}
