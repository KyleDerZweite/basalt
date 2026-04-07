// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/KyleDerZweite/basalt/internal/config"
	"github.com/KyleDerZweite/basalt/internal/graph"
	"github.com/KyleDerZweite/basalt/internal/httpclient"
	"github.com/KyleDerZweite/basalt/internal/modules"
)

type testHealthModule struct {
	name    string
	verifyN *atomic.Int64
}

func (m *testHealthModule) Name() string        { return m.name }
func (m *testHealthModule) Description() string { return "test module" }
func (m *testHealthModule) CanHandle(string) bool {
	return false
}
func (m *testHealthModule) Extract(context.Context, *graph.Node, *httpclient.Client) ([]*graph.Node, []*graph.Edge, error) {
	return nil, nil, nil
}
func (m *testHealthModule) Verify(context.Context, *httpclient.Client) (modules.HealthStatus, string) {
	m.verifyN.Add(1)
	return modules.Healthy, "ok"
}

func TestModuleHealthCacheHit(t *testing.T) {
	service, verifyCount, cleanup := testModuleHealthService(t)
	defer cleanup()

	req := ScanRequest{TimeoutSeconds: 1}
	if _, err := service.ModuleHealth(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ModuleHealth(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if got := verifyCount.Load(); got != 1 {
		t.Fatalf("expected cached second lookup, got %d verifies", got)
	}
}

func TestModuleHealthCacheExpires(t *testing.T) {
	service, verifyCount, cleanup := testModuleHealthService(t)
	defer cleanup()

	req := ScanRequest{TimeoutSeconds: 1}
	if _, err := service.ModuleHealth(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if got := verifyCount.Load(); got != 1 {
		t.Fatalf("expected first verification, got %d", got)
	}

	if _, err := service.store.db.Exec(`UPDATE module_health_cache SET expires_at = ?`, time.Now().UTC().Add(-time.Minute).Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}

	if _, err := service.ModuleHealth(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if got := verifyCount.Load(); got != 2 {
		t.Fatalf("expected expired cache to force a second verification, got %d", got)
	}
}

func TestModuleHealthCacheInvalidatesOnConfigChange(t *testing.T) {
	service, verifyCount, cleanup := testModuleHealthService(t)
	defer cleanup()

	configPath := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(configPath, []byte("TOKEN=one\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	req := ScanRequest{TimeoutSeconds: 1, ConfigPath: configPath}
	if _, err := service.ModuleHealth(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("TOKEN=two\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ModuleHealth(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if got := verifyCount.Load(); got != 2 {
		t.Fatalf("expected config hash change to invalidate cache, got %d verifies", got)
	}
}

func TestModuleHealthRefreshOverride(t *testing.T) {
	service, verifyCount, cleanup := testModuleHealthService(t)
	defer cleanup()

	req := ScanRequest{TimeoutSeconds: 1}
	if _, err := service.ModuleHealth(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	req.RefreshModuleHealth = true
	if _, err := service.ModuleHealth(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if got := verifyCount.Load(); got != 2 {
		t.Fatalf("expected refresh override to force a second verification, got %d", got)
	}
}

func TestModuleHealthClearOverride(t *testing.T) {
	service, verifyCount, cleanup := testModuleHealthService(t)
	defer cleanup()

	req := ScanRequest{TimeoutSeconds: 1}
	if _, err := service.ModuleHealth(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	req.ClearModuleHealthCache = true
	if _, err := service.ModuleHealth(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if got := verifyCount.Load(); got != 2 {
		t.Fatalf("expected clear override to force a second verification, got %d", got)
	}
}

func TestModuleHealthTTLOverridePersistsCustomExpiry(t *testing.T) {
	service, _, cleanup := testModuleHealthService(t)
	defer cleanup()

	req := ScanRequest{
		TimeoutSeconds:         1,
		RefreshModuleHealth:    true,
		ModuleHealthTTLSeconds: 90,
	}
	before := time.Now().UTC()
	if _, err := service.ModuleHealth(context.Background(), req); err != nil {
		t.Fatal(err)
	}

	var rawChecked string
	var rawExpires string
	if err := service.store.db.QueryRow(`SELECT checked_at, expires_at FROM module_health_cache LIMIT 1`).Scan(&rawChecked, &rawExpires); err != nil {
		t.Fatal(err)
	}
	checkedAt, err := time.Parse(time.RFC3339Nano, rawChecked)
	if err != nil {
		t.Fatal(err)
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, rawExpires)
	if err != nil {
		t.Fatal(err)
	}
	if expiresAt.Sub(checkedAt) != 90*time.Second {
		t.Fatalf("expected custom TTL of 90s, got %s", expiresAt.Sub(checkedAt))
	}
	if checkedAt.Before(before.Add(-time.Second)) {
		t.Fatalf("unexpected checked_at timestamp %s", checkedAt)
	}
}

func testModuleHealthService(t *testing.T) (*Service, *atomic.Int64, func()) {
	t.Helper()

	verifyCount := &atomic.Int64{}
	originalFactories := moduleFactories
	moduleFactories = []moduleFactory{
		func(*config.Config) modules.Module {
			return &testHealthModule{name: "test-module", verifyN: verifyCount}
		},
	}

	service, err := NewService("test-version", t.TempDir())
	if err != nil {
		moduleFactories = originalFactories
		t.Fatal(err)
	}

	cleanup := func() {
		moduleFactories = originalFactories
		_ = service.Close()
	}
	return service, verifyCount, cleanup
}
