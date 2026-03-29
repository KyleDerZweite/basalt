// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	content := "STEAM_API_KEY=abc123\nGITHUB_TOKEN=ghp_test\n# comment\nEMPTY_VAL=\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if got := cfg.Get("STEAM_API_KEY"); got != "abc123" {
		t.Errorf("STEAM_API_KEY = %q, want %q", got, "abc123")
	}
	if got := cfg.Get("GITHUB_TOKEN"); got != "ghp_test" {
		t.Errorf("GITHUB_TOKEN = %q, want %q", got, "ghp_test")
	}
	if got := cfg.Get("EMPTY_VAL"); got != "" {
		t.Errorf("EMPTY_VAL = %q, want empty", got)
	}
	if got := cfg.Get("MISSING"); got != "" {
		t.Errorf("MISSING = %q, want empty", got)
	}
}

func TestLoadDefaultPath(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config")
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Get("ANYTHING"); got != "" {
		t.Errorf("expected empty for missing config, got %q", got)
	}
}

func TestLoadSkipsBlankLinesAndComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	content := "\n\n# A comment\n\nKEY=value\n\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Get("KEY"); got != "value" {
		t.Errorf("KEY = %q, want %q", got, "value")
	}
}

func TestLoadQuotedValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	content := "KEY=\"hello world\"\nKEY2='single quoted'\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.Get("KEY"); got != "hello world" {
		t.Errorf("KEY = %q, want %q", got, "hello world")
	}
	if got := cfg.Get("KEY2"); got != "single quoted" {
		t.Errorf("KEY2 = %q, want %q", got, "single quoted")
	}
}
