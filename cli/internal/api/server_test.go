// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KyleDerZweite/basalt/internal/app"
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

func testServer(t *testing.T, opts Options) http.Handler {
	t.Helper()

	service, err := app.NewService("test", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = service.Close()
	})

	return NewServer(service, opts)
}
