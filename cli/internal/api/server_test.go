// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/KyleDerZweite/basalt/internal/app"
	"github.com/KyleDerZweite/basalt/internal/graph"
)

func TestSettingsRoundTrip(t *testing.T) {
	handler := testServer(t, Options{})

	payload := []byte(`{"strict_mode":true,"disabled_modules":["github"]}`)
	req := httptest.NewRequest(http.MethodPut, "/api/settings", bytes.NewReader(payload))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT /api/settings returned %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/settings returned %d: %s", rec.Code, rec.Body.String())
	}

	var settings app.Settings
	if err := json.Unmarshal(rec.Body.Bytes(), &settings); err != nil {
		t.Fatal(err)
	}
	if !settings.StrictMode {
		t.Fatal("expected strict mode to persist")
	}
	if len(settings.DisabledModules) != 1 || settings.DisabledModules[0] != "github" {
		t.Fatalf("unexpected disabled modules: %v", settings.DisabledModules)
	}
}

func TestListScansEmpty(t *testing.T) {
	handler := testServer(t, Options{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/scans", nil)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/scans returned %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Body.String(); !bytes.Contains([]byte(got), []byte(`"scans"`)) {
		t.Fatalf("expected scans payload, got %s", got)
	}
}

func TestStartScanRequiresSeeds(t *testing.T) {
	handler := testServer(t, Options{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/scans", bytes.NewReader([]byte(`{"depth":1}`)))
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing seeds, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStartScanWithTargetAliasesOnly(t *testing.T) {
	harness := testHarness(t, Options{})

	target, err := harness.service.CreateTarget(app.Target{
		DisplayName: "Kyle",
		Slug:        "kyle",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := harness.service.AddTargetAlias(target.Slug, graph.Seed{Type: graph.NodeTypeUsername, Value: "kylederzweite"}, "", true); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/scans", bytes.NewReader([]byte(`{"target_ref":"kyle","depth":1}`)))
	harness.handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202 for target-only scan, got %d: %s", rec.Code, rec.Body.String())
	}

	var record app.ScanRecord
	if err := json.Unmarshal(rec.Body.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	if record.TargetID != target.ID {
		t.Fatalf("expected target id %q, got %q", target.ID, record.TargetID)
	}
	if len(record.Seeds) != 1 || record.Seeds[0].Value != "kylederzweite" {
		t.Fatalf("expected resolved target alias seed, got %+v", record.Seeds)
	}
}

func TestServerRequiresBearerAuthWhenConfigured(t *testing.T) {
	handler := testServer(t, Options{AuthToken: "secret-token"})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid auth, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestServerMirrorsAllowedOrigin(t *testing.T) {
	handler := testServer(t, Options{
		AuthToken:      "secret-token",
		AllowedOrigins: []string{"http://localhost:5173"},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/settings", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Authorization", "Bearer secret-token")
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 preflight, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("unexpected allow origin header: %q", got)
	}
}

func TestServerRejectsForbiddenOriginPreflight(t *testing.T) {
	handler := testServer(t, Options{
		AuthToken:      "secret-token",
		AllowedOrigins: []string{"http://localhost:5173"},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/settings", nil)
	req.Header.Set("Origin", "https://evil.example")
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 preflight rejection, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestTargetsRoundTrip(t *testing.T) {
	handler := testServer(t, Options{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/targets", bytes.NewReader([]byte(`{"display_name":"Kyle","slug":"kyle"}`)))
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/targets returned %d: %s", rec.Code, rec.Body.String())
	}

	var target app.Target
	if err := json.Unmarshal(rec.Body.Bytes(), &target); err != nil {
		t.Fatal(err)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/targets/kyle/aliases", bytes.NewReader([]byte(`{"seed_type":"username","seed_value":"kylederzweite","is_primary":true}`)))
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/targets/{id}/aliases returned %d: %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/targets/kyle", nil)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/targets/{id} returned %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Body.String(); !bytes.Contains([]byte(got), []byte(`"kylederzweite"`)) {
		t.Fatalf("expected alias in target payload, got %s", got)
	}
}

func TestWorkspaceEndpoint(t *testing.T) {
	harness := testHarness(t, Options{})
	now := time.Now().UTC().Round(time.Second)
	target, err := harness.service.CreateTarget(app.Target{
		DisplayName: "Kyle",
		Slug:        "kyle",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := harness.service.AddTargetAlias(target.Slug, graph.Seed{Type: graph.NodeTypeUsername, Value: "kylederzweite"}, "", true); err != nil {
		t.Fatal(err)
	}

	g := graph.New()
	account := graph.NewAccountNode("github", "KyleDerZweite", "https://github.com/KyleDerZweite", "github")
	account.Confidence = 0.95
	account.Properties["site_name"] = "github"
	domain := graph.NewNode(graph.NodeTypeDomain, "kylehub.dev", "github")
	domain.Confidence = 0.85
	if !g.AddNode(account) || !g.AddNode(domain) {
		t.Fatal("expected graph nodes")
	}
	record := &app.ScanRecord{
		ID:        "scan-1",
		TargetID:  target.ID,
		Status:    app.ScanStatusCompleted,
		StartedAt: now,
		UpdatedAt: now,
		Seeds:     []graph.Seed{{Type: graph.NodeTypeUsername, Value: "kylederzweite"}},
		Options:   app.ScanRequest{TargetRef: target.Slug},
		Health:    []app.ModuleStatus{{Name: "github", Status: "healthy", Message: "ok"}},
		Graph:     g,
	}
	record.Insights = ptr(app.BuildScanInsights(g, record.Health, record.Status))
	if err := harness.RunStoreCreateAndUpdate(record); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/scans/scan-1/workspace", nil)
	harness.handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/scans/{id}/workspace returned %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Body.String(); !bytes.Contains([]byte(got), []byte(`"headline"`)) || !bytes.Contains([]byte(got), []byte(`"target"`)) {
		t.Fatalf("expected workspace payload, got %s", got)
	}
}

func testServer(t *testing.T, opts Options) http.Handler {
	return testHarness(t, opts).handler
}

type harness struct {
	handler http.Handler
	service *app.Service
}

func testHarness(t *testing.T, opts Options) harness {
	t.Helper()

	service, err := app.NewService("test", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = service.Close()
	})

	return harness{
		handler: NewServer(service, opts),
		service: service,
	}
}

func (h harness) RunStoreCreateAndUpdate(record *app.ScanRecord) error {
	if err := h.service.Store().CreateScan(record); err != nil {
		return err
	}
	return h.service.Store().UpdateScan(record)
}

func ptr[T any](value T) *T {
	return &value
}
