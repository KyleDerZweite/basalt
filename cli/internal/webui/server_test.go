// SPDX-License-Identifier: AGPL-3.0-or-later

package webui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KyleDerZweite/basalt/internal/app"
)

func TestRootServesIndex(t *testing.T) {
	handler := testServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET / returned %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("expected HTML content type, got %q", got)
	}
	if !strings.Contains(rec.Body.String(), "<div id=\"root\"></div>") {
		t.Fatalf("expected HTML shell, got %s", rec.Body.String())
	}
}

func TestBootstrapReturnsRuntimeConfig(t *testing.T) {
	service, err := app.NewService("test-version", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = service.Close() })

	handler := NewServer(service, Options{BaseURL: "http://127.0.0.1:9999"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/app/bootstrap", nil)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /app/bootstrap returned %d: %s", rec.Code, rec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["product"] != "web" {
		t.Fatalf("unexpected product: %#v", payload["product"])
	}
	if payload["version"] != "test-version" {
		t.Fatalf("unexpected version: %#v", payload["version"])
	}
	if payload["api_base_path"] != "/api" {
		t.Fatalf("unexpected api base path: %#v", payload["api_base_path"])
	}
}

func TestAPIRemainsMounted(t *testing.T) {
	handler := testServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/settings returned %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPathFallbackServesIndex(t *testing.T) {
	handler := testServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/scans/example-scan", nil)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /scans/example-scan returned %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "<div id=\"root\"></div>") {
		t.Fatalf("expected HTML shell, got %s", rec.Body.String())
	}
}

func TestAssetsAreServed(t *testing.T) {
	handler := testServer(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /assets/app.js returned %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "reactflow") && !strings.Contains(rec.Body.String(), "Investigation Workspace") {
		t.Fatalf("expected JavaScript asset, got %s", rec.Body.String())
	}
}

func testServer(t *testing.T) http.Handler {
	t.Helper()

	service, err := app.NewService("test", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = service.Close()
	})

	return NewServer(service, Options{BaseURL: "http://127.0.0.1:8788"})
}
