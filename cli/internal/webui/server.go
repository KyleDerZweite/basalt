// SPDX-License-Identifier: AGPL-3.0-or-later

package webui

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"

	"github.com/KyleDerZweite/basalt/internal/api"
	"github.com/KyleDerZweite/basalt/internal/app"
)

//go:embed dist dist/* dist/assets/*
var distFS embed.FS

// Options configures the browser-facing local product server.
type Options struct {
	BaseURL string
}

// NewServer creates the same-origin web UI and API server.
func NewServer(service *app.Service, opts Options) http.Handler {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err)
	}

	apiHandler := api.NewServer(service, api.Options{})
	fileServer := http.FileServer(http.FS(sub))

	mux := http.NewServeMux()
	mux.Handle("/api/", apiHandler)
	mux.Handle("/app/bootstrap", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"name":                "basalt",
			"product":             "web",
			"version":             service.Version(),
			"data_dir":            service.DataDir(),
			"default_config_path": app.DefaultConfigPath(service.DataDir()),
			"api_base_path":       "/api",
			"base_url":            opts.BaseURL,
		})
	}))
	mux.Handle("/assets/", fileServer)
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if shouldServeAsset(sub, r.URL.Path) {
			fileServer.ServeHTTP(w, r)
			return
		}
		if err := serveIndex(sub, w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))
	return mux
}

func shouldServeAsset(fsys fs.FS, requestPath string) bool {
	if requestPath == "/" || strings.HasPrefix(requestPath, "/api/") || requestPath == "/app/bootstrap" {
		return false
	}
	trimmed := strings.TrimPrefix(strings.TrimSpace(requestPath), "/")
	if trimmed == "" {
		return false
	}
	info, err := fs.Stat(fsys, trimmed)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func serveIndex(fsys fs.FS, w http.ResponseWriter) error {
	data, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(data)
	return err
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
