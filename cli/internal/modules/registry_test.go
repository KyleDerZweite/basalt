// SPDX-License-Identifier: AGPL-3.0-or-later

package modules

import (
	"context"
	"testing"

	"github.com/kyle/basalt/internal/graph"
	"github.com/kyle/basalt/internal/httpclient"
)

// stubModule is a minimal Module implementation for testing.
type stubModule struct {
	name      string
	handles   []string
	health    HealthStatus
	healthMsg string
}

func (s *stubModule) Name() string        { return s.name }
func (s *stubModule) Description() string { return "stub module for testing" }
func (s *stubModule) CanHandle(nodeType string) bool {
	for _, h := range s.handles {
		if h == nodeType {
			return true
		}
	}
	return false
}
func (s *stubModule) Extract(_ context.Context, _ *graph.Node, _ *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	return nil, nil, nil
}
func (s *stubModule) Verify(_ context.Context, _ *httpclient.Client) (HealthStatus, string) {
	return s.health, s.healthMsg
}

func TestRegistryRegisterAndLookup(t *testing.T) {
	reg := NewRegistry()
	m := &stubModule{name: "github", handles: []string{"username", "email"}}
	reg.Register(m)

	got := reg.ModulesFor("username")
	if len(got) != 1 || got[0].Name() != "github" {
		t.Errorf("expected github module for username, got %v", got)
	}

	got = reg.ModulesFor("email")
	if len(got) != 1 || got[0].Name() != "github" {
		t.Errorf("expected github module for email, got %v", got)
	}

	got = reg.ModulesFor("domain")
	if len(got) != 0 {
		t.Errorf("expected no modules for domain, got %v", got)
	}
}

func TestRegistryAll(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&stubModule{name: "a", handles: []string{"username"}})
	reg.Register(&stubModule{name: "b", handles: []string{"email"}})

	all := reg.All()
	if len(all) != 2 {
		t.Errorf("expected 2 modules, got %d", len(all))
	}
}

func TestRegistryModulesForNodeType(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&stubModule{name: "gravatar", handles: []string{"email"}})
	reg.Register(&stubModule{name: "github", handles: []string{"username", "email"}})
	reg.Register(&stubModule{name: "whois", handles: []string{"domain"}})

	emailModules := reg.ModulesFor("email")
	if len(emailModules) != 2 {
		t.Errorf("expected 2 email modules, got %d", len(emailModules))
	}
}
