// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/KyleDerZweite/basalt/internal/app"
	"github.com/KyleDerZweite/basalt/internal/graph"
)

// Options configures the local HTTP API.
type Options struct {
	AuthToken      string
	AllowedOrigins []string
}

// NewServer creates the local HTTP API for Basalt clients.
func NewServer(service *app.Service, opts Options) http.Handler {
	server := &Server{
		service:        service,
		authToken:      strings.TrimSpace(opts.AuthToken),
		allowedOrigins: normalizeOrigins(opts.AllowedOrigins),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", server.handleRoot)
	mux.Handle("/api/settings", server.apiHandler(http.HandlerFunc(server.handleSettings)))
	mux.Handle("/api/modules/health", server.apiHandler(http.HandlerFunc(server.handleModuleHealth)))
	mux.Handle("/api/targets", server.apiHandler(http.HandlerFunc(server.handleTargets)))
	mux.Handle("/api/targets/", server.apiHandler(http.HandlerFunc(server.handleTargetByID)))
	mux.Handle("/api/scans", server.apiHandler(http.HandlerFunc(server.handleScans)))
	mux.Handle("/api/scans/", server.apiHandler(http.HandlerFunc(server.handleScanByID)))
	return mux
}

// Server serves the local API surface.
type Server struct {
	service        *app.Service
	authToken      string
	allowedOrigins []string
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"name":     "basalt",
		"product":  "local-api",
		"version":  s.service.Version(),
		"data_dir": s.service.DataDir(),
	})
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		settings, err := s.service.GetSettings()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, settings)
	case http.MethodPut:
		var settings app.Settings
		if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid settings payload: %w", err))
			return
		}
		if err := s.service.UpdateSettings(settings); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, settings)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleModuleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	req := app.ScanRequest{
		Depth:                  parseInt(r.URL.Query().Get("depth"), 2),
		Concurrency:            parseInt(r.URL.Query().Get("concurrency"), 5),
		TimeoutSeconds:         parseInt(r.URL.Query().Get("timeout"), 10),
		StrictMode:             r.URL.Query().Get("strict") == "1" || r.URL.Query().Get("strict") == "true",
		RefreshModuleHealth:    r.URL.Query().Get("refresh") == "1" || r.URL.Query().Get("refresh") == "true",
		ClearModuleHealthCache: r.URL.Query().Get("clear") == "1" || r.URL.Query().Get("clear") == "true",
		ModuleHealthTTLSeconds: parseDurationSeconds(r.URL.Query().Get("ttl"), 0),
	}
	health, err := s.service.ModuleHealth(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"modules": health})
}

func (s *Server) handleScans(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		limit := parseInt(r.URL.Query().Get("limit"), 20)
		scans, err := s.service.ListScans(limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"scans": scans})
	case http.MethodPost:
		var req app.ScanRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid scan payload: %w", err))
			return
		}
		record, err := s.service.StartScan(r.Context(), req)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusAccepted, record)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleTargets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		targets, err := s.service.ListTargets()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"targets": targets})
	case http.MethodPost:
		var target app.Target
		if err := json.NewDecoder(r.Body).Decode(&target); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid target payload: %w", err))
			return
		}
		created, err := s.service.CreateTarget(target)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusCreated, created)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleTargetByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/targets/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}

	ref := parts[0]
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			target, err := s.service.GetTarget(ref)
			if err != nil {
				writeError(w, http.StatusNotFound, err)
				return
			}
			writeJSON(w, http.StatusOK, target)
		case http.MethodPatch:
			var target app.Target
			if err := json.NewDecoder(r.Body).Decode(&target); err != nil {
				writeError(w, http.StatusBadRequest, fmt.Errorf("invalid target payload: %w", err))
				return
			}
			updated, err := s.service.UpdateTarget(ref, target)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusOK, updated)
		case http.MethodDelete:
			if err := s.service.DeleteTarget(ref); err != nil {
				writeError(w, http.StatusNotFound, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	switch parts[1] {
	case "aliases":
		if len(parts) == 2 {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var payload struct {
				SeedType  string `json:"seed_type"`
				SeedValue string `json:"seed_value"`
				Label     string `json:"label"`
				IsPrimary bool   `json:"is_primary"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				writeError(w, http.StatusBadRequest, fmt.Errorf("invalid alias payload: %w", err))
				return
			}
			alias, err := s.service.AddTargetAlias(ref, appSeed(payload.SeedType, payload.SeedValue), payload.Label, payload.IsPrimary)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusCreated, alias)
			return
		}
		if r.Method != http.MethodDelete {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := s.service.RemoveTargetAlias(ref, parts[2]); err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
	case "scans":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		limit := parseInt(r.URL.Query().Get("limit"), 20)
		scans, err := s.service.ListTargetScans(ref, limit)
		if err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"scans": scans})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleScanByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/scans/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}

	scanID := parts[0]
	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		record, err := s.service.GetScan(scanID)
		if err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeJSON(w, http.StatusOK, record)
		return
	}

	switch parts[1] {
	case "results":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		record, err := s.service.GetScan(scanID)
		if err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"scan_id": scanID,
			"graph":   record.Graph,
			"status":  record.Status,
		})
	case "workspace":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		workspace, err := s.service.BuildWorkspace(scanID)
		if err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeJSON(w, http.StatusOK, workspace)
	case "events":
		if acceptsSSE(r) {
			s.streamEvents(w, r, scanID)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		after := int64(parseInt(r.URL.Query().Get("after"), 0))
		events, err := s.service.GetEvents(scanID, after)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"events": events})
	case "export":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		format := r.URL.Query().Get("format")
		if format == "" {
			writeError(w, http.StatusBadRequest, fmt.Errorf("missing format query parameter"))
			return
		}
		switch format {
		case "json":
			w.Header().Set("Content-Type", "application/json")
		case "csv":
			w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		default:
			writeError(w, http.StatusBadRequest, fmt.Errorf("unsupported export format %q", format))
			return
		}
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", "basalt-"+scanID+"."+format))
		if err := s.service.WriteExport(scanID, format, w); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	case "cancel":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := s.service.CancelScan(scanID); err != nil {
			writeError(w, http.StatusConflict, err)
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]any{"scan_id": scanID, "status": "cancel_requested"})
	default:
		http.NotFound(w, r)
	}
}

func appSeed(seedType, seedValue string) graph.Seed {
	return graph.Seed{Type: strings.TrimSpace(seedType), Value: strings.TrimSpace(seedValue)}
}

func (s *Server) streamEvents(w http.ResponseWriter, r *http.Request, scanID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	after := int64(parseInt(r.URL.Query().Get("after"), 0))
	backlog, err := s.service.GetEvents(scanID, after)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	for _, event := range backlog {
		if err := writeSSE(w, event); err != nil {
			return
		}
	}
	flusher.Flush()

	events, cancel := s.service.Subscribe(scanID)
	defer cancel()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if _, err := fmt.Fprint(w, ": keep-alive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case event, ok := <-events:
			if !ok {
				return
			}
			if event.Sequence <= after {
				continue
			}
			if err := writeSSE(w, event); err != nil {
				return
			}
			flusher.Flush()
			after = event.Sequence
		}
	}
}

func (s *Server) apiHandler(next http.Handler) http.Handler {
	handler := next
	handler = s.withAuth(handler)
	handler = s.withCORS(handler)
	return handler
}

func (s *Server) withAuth(next http.Handler) http.Handler {
	if s.authToken == "" {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		token, err := bearerToken(r)
		if err != nil || token != s.authToken {
			writeError(w, http.StatusUnauthorized, errors.New("missing or invalid bearer token"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if corsOrigin, ok := s.allowedCORSOrigin(origin); ok {
			w.Header().Set("Access-Control-Allow-Origin", corsOrigin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		if r.Method == http.MethodOptions {
			if origin != "" && len(s.allowedOrigins) > 0 {
				if _, ok := s.allowedCORSOrigin(origin); !ok {
					http.Error(w, "origin not allowed", http.StatusForbidden)
					return
				}
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) allowedCORSOrigin(origin string) (string, bool) {
	if len(s.allowedOrigins) == 0 {
		if origin == "" {
			return "", false
		}
		return "*", true
	}
	if origin == "" {
		return "", false
	}
	if slices.Contains(s.allowedOrigins, origin) {
		return origin, true
	}
	return "", false
}

func normalizeOrigins(origins []string) []string {
	if len(origins) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(origins))
	out := make([]string, 0, len(origins))
	for _, origin := range origins {
		trimmed := strings.TrimSpace(origin)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func bearerToken(r *http.Request) (string, error) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return "", errors.New("missing authorization header")
	}
	if !strings.HasPrefix(header, "Bearer ") {
		return "", errors.New("invalid authorization scheme")
	}
	return strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")), nil
}

func writeSSE(w http.ResponseWriter, event app.ScanEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "event: %s\n", event.Type); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
		return err
	}
	return nil
}

func acceptsSSE(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "text/event-stream") || r.URL.Query().Get("stream") == "1"
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func parseInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	out, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return out
}

func parseDurationSeconds(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	if seconds, err := strconv.Atoi(value); err == nil {
		return seconds
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return int(duration / time.Second)
}
