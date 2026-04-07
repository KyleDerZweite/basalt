// SPDX-License-Identifier: AGPL-3.0-or-later

package modules

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/KyleDerZweite/basalt/internal/graph"
)

var usernameLikePattern = regexp.MustCompile(`^[A-Za-z0-9._-]{3,32}$`)

// InferUsernameFromDisplayName promotes handle-like single-token display names
// into a lower-confidence username node for selective module-level pivoting.
func InferUsernameFromDisplayName(displayName, knownUsername, sourceModule string) *graph.Node {
	candidate := strings.TrimSpace(displayName)
	if candidate == "" {
		return nil
	}
	if len(strings.Fields(candidate)) != 1 {
		return nil
	}
	if strings.EqualFold(candidate, strings.TrimSpace(knownUsername)) {
		return nil
	}
	if strings.Contains(candidate, "@") {
		return nil
	}
	if _, err := url.ParseRequestURI(candidate); err == nil {
		return nil
	}
	if !usernameLikePattern.MatchString(candidate) {
		return nil
	}
	if isNumericOnly(candidate) {
		return nil
	}

	node := graph.NewNode(graph.NodeTypeUsername, candidate, sourceModule)
	node.Pivot = true
	node.Confidence = 0.65
	node.Properties["inferred_from"] = graph.NodeTypeFullName
	node.Properties["inferred_by"] = "single_token_display_name"
	return node
}

func isNumericOnly(value string) bool {
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}
