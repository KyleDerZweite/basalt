// SPDX-License-Identifier: AGPL-3.0-or-later

package app

import (
	"os"
	"testing"
	"time"
)

func TestServeRuntimeRoundTrip(t *testing.T) {
	dataDir := t.TempDir()
	now := time.Now().UTC().Round(time.Second)
	runtime := ServeRuntime{
		PID:           4242,
		ListenAddress: "127.0.0.1:8787",
		BaseURL:       "http://127.0.0.1:8787",
		Version:       "test",
		DataDir:       dataDir,
		LogFile:       DefaultServeLogPath(dataDir),
		StartedAt:     now,
	}

	if err := WriteServeRuntime(dataDir, runtime); err != nil {
		t.Fatal(err)
	}

	got, err := ReadServeRuntime(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if got.PID != runtime.PID || got.BaseURL != runtime.BaseURL || !got.StartedAt.Equal(now) {
		t.Fatalf("unexpected runtime payload: %+v", got)
	}

	if err := RemoveServeRuntime(dataDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(DefaultServeRuntimePath(dataDir)); !os.IsNotExist(err) {
		t.Fatalf("expected runtime file to be removed, stat err=%v", err)
	}
}
