// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/KyleDerZweite/basalt/internal/api"
	"github.com/KyleDerZweite/basalt/internal/app"
)

var (
	flagListenAddr      string
	flagServeAuthToken  string
	flagServeOrigins    []string
	flagServeDetach     bool
	flagServeStatus     bool
	flagServeStop       bool
	flagServeForce      bool
	flagServeLogFile    string
	flagPrintListenJSON bool
)

type serveStartupInfo struct {
	ListenAddress string `json:"listen_address"`
	BaseURL       string `json:"base_url"`
	Version       string `json:"version"`
	DataDir       string `json:"data_dir"`
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the local Basalt product API",
	Long: `Run the local Basalt API used by local clients and browser-based UIs.

The server binds to localhost by default and persists scans, settings, and events
inside the Basalt data directory.`,
	RunE: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().StringVar(&flagDataDir, "data-dir", app.DefaultDataDir(), "Path to local app data directory")
	serveCmd.Flags().StringVar(&flagListenAddr, "listen", "127.0.0.1:8787", "Listen address for the local API")
	serveCmd.Flags().StringVar(&flagServeAuthToken, "auth-token", "", "Require this bearer token on all /api routes")
	serveCmd.Flags().StringSliceVar(&flagServeOrigins, "allow-origin", nil, "Allowed CORS origin for local clients (repeatable)")
	serveCmd.Flags().BoolVarP(&flagServeDetach, "detach", "d", false, "Start the local API in the background")
	serveCmd.Flags().BoolVar(&flagServeStatus, "status", false, "Show status for the managed local API")
	serveCmd.Flags().BoolVar(&flagServeStop, "stop", false, "Stop the managed local API")
	serveCmd.Flags().BoolVar(&flagServeForce, "force", false, "Force stop if graceful shutdown times out")
	serveCmd.Flags().StringVar(&flagServeLogFile, "log-file", "", "Path to the serve log file")
	serveCmd.Flags().BoolVar(&flagPrintListenJSON, "print-listen-json", false, "Print machine-readable startup JSON to stdout")
}

func runServe(cmd *cobra.Command, args []string) error {
	if err := validateServeMode(); err != nil {
		return err
	}
	if flagServeLogFile == "" {
		flagServeLogFile = app.DefaultServeLogPath(flagDataDir)
	}

	switch {
	case flagServeStatus:
		return runServeStatus()
	case flagServeStop:
		return runServeStop()
	case flagServeDetach:
		return runServeDetach()
	default:
		return runServeForeground()
	}
}

func runServeForeground() error {
	service, err := app.NewService(Version, flagDataDir)
	if err != nil {
		return err
	}
	defer service.Close()

	listener, err := net.Listen("tcp", flagListenAddr)
	if err != nil {
		return err
	}
	defer listener.Close()

	server := &http.Server{
		Addr: listener.Addr().String(),
		Handler: api.NewServer(service, api.Options{
			AuthToken:      flagServeAuthToken,
			AllowedOrigins: flagServeOrigins,
		}),
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	baseURL := "http://" + listener.Addr().String()
	runtime := app.ServeRuntime{
		PID:           os.Getpid(),
		ListenAddress: listener.Addr().String(),
		BaseURL:       baseURL,
		Version:       service.Version(),
		DataDir:       service.DataDir(),
		LogFile:       flagServeLogFile,
		StartedAt:     time.Now().UTC(),
	}
	if err := app.WriteServeRuntime(service.DataDir(), runtime); err != nil {
		return fmt.Errorf("writing serve runtime: %w", err)
	}
	defer func() {
		_ = app.RemoveServeRuntime(service.DataDir())
	}()

	startup := serveStartupInfo{
		ListenAddress: listener.Addr().String(),
		BaseURL:       baseURL,
		Version:       service.Version(),
		DataDir:       service.DataDir(),
	}
	if flagPrintListenJSON {
		if err := json.NewEncoder(os.Stdout).Encode(startup); err != nil {
			return fmt.Errorf("writing startup JSON: %w", err)
		}
	}

	fmt.Fprintf(os.Stderr, "Basalt local API listening on %s\n", baseURL)
	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func runServeDetach() error {
	if runtime, err := readServeRuntimeIfPresent(flagDataDir); err == nil {
		if ok, _ := serveHealthy(runtime.BaseURL); ok {
			fmt.Printf("Basalt serve is already running.\nPID: %d\nBase URL: %s\nData dir: %s\nLog file: %s\n", runtime.PID, runtime.BaseURL, runtime.DataDir, runtime.LogFile)
			return nil
		}
		_ = app.RemoveServeRuntime(flagDataDir)
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(flagServeLogFile), 0o755); err != nil {
		return fmt.Errorf("creating serve log dir: %w", err)
	}

	logFile, err := os.OpenFile(flagServeLogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening serve log file: %w", err)
	}
	defer logFile.Close()

	devNull, err := os.Open(os.DevNull)
	if err != nil {
		return fmt.Errorf("opening %s: %w", os.DevNull, err)
	}
	defer devNull.Close()

	args := []string{
		"serve",
		"--data-dir", flagDataDir,
		"--listen", flagListenAddr,
		"--log-file", flagServeLogFile,
	}
	if flagServeAuthToken != "" {
		args = append(args, "--auth-token", flagServeAuthToken)
	}
	for _, origin := range flagServeOrigins {
		args = append(args, "--allow-origin", origin)
	}
	if flagPrintListenJSON {
		args = append(args, "--print-listen-json")
	}

	binaryPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locating current executable: %w", err)
	}

	child := exec.Command(binaryPath, args...)
	child.Stdin = devNull
	child.Stdout = logFile
	child.Stderr = logFile
	child.SysProcAttr = detachedProcAttr()

	if err := child.Start(); err != nil {
		return fmt.Errorf("starting detached serve process: %w", err)
	}

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- child.Wait()
	}()

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-waitCh:
			if err != nil {
				return fmt.Errorf("serve process exited before ready, see %s: %w", flagServeLogFile, err)
			}
			return fmt.Errorf("serve process exited before ready, see %s", flagServeLogFile)
		default:
		}

		runtime, err := readServeRuntimeIfPresent(flagDataDir)
		if err == nil {
			ok, healthErr := serveHealthy(runtime.BaseURL)
			if ok {
				fmt.Printf("Basalt serve started in background.\nPID: %d\nBase URL: %s\nData dir: %s\nLog file: %s\n", runtime.PID, runtime.BaseURL, runtime.DataDir, runtime.LogFile)
				return nil
			}
			if healthErr != nil {
				time.Sleep(200 * time.Millisecond)
				continue
			}
		} else if !os.IsNotExist(err) {
			return err
		}
		time.Sleep(200 * time.Millisecond)
	}

	return fmt.Errorf("serve process did not become ready in time, see %s", flagServeLogFile)
}

func runServeStatus() error {
	runtime, err := readServeRuntimeIfPresent(flagDataDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Basalt serve is not running for data dir %s\n", flagDataDir)
			return nil
		}
		return err
	}

	ok, healthErr := serveHealthy(runtime.BaseURL)
	if ok {
		fmt.Printf("Basalt serve is running.\nPID: %d\nBase URL: %s\nData dir: %s\nLog file: %s\nStarted: %s\n", runtime.PID, runtime.BaseURL, runtime.DataDir, runtime.LogFile, runtime.StartedAt.Format(time.RFC3339))
		return nil
	}

	if healthErr != nil {
		fmt.Printf("Basalt serve runtime file exists but the API is unreachable.\nPID: %d\nBase URL: %s\nData dir: %s\nLog file: %s\n", runtime.PID, runtime.BaseURL, runtime.DataDir, runtime.LogFile)
		return nil
	}

	fmt.Printf("Basalt serve runtime file exists but the API reported an unhealthy response.\nPID: %d\nBase URL: %s\nData dir: %s\nLog file: %s\n", runtime.PID, runtime.BaseURL, runtime.DataDir, runtime.LogFile)
	return nil
}

func runServeStop() error {
	runtime, err := readServeRuntimeIfPresent(flagDataDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Basalt serve is not running for data dir %s\n", flagDataDir)
			return nil
		}
		return err
	}

	ok, healthErr := serveHealthy(runtime.BaseURL)
	if !ok {
		_ = app.RemoveServeRuntime(flagDataDir)
		if healthErr != nil {
			fmt.Printf("Basalt serve is not reachable at %s. Removed stale runtime metadata for %s.\n", runtime.BaseURL, runtime.DataDir)
			return nil
		}
		fmt.Printf("Basalt serve reported an unhealthy response at %s. Removed stale runtime metadata for %s.\n", runtime.BaseURL, runtime.DataDir)
		return nil
	}

	process, err := os.FindProcess(runtime.PID)
	if err != nil {
		_ = app.RemoveServeRuntime(flagDataDir)
		return fmt.Errorf("finding serve process %d: %w", runtime.PID, err)
	}

	if err := process.Signal(os.Interrupt); err != nil && !errors.Is(err, os.ErrProcessDone) {
		_ = app.RemoveServeRuntime(flagDataDir)
		return fmt.Errorf("signaling serve process %d: %w", runtime.PID, err)
	}

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		ok, _ := serveHealthy(runtime.BaseURL)
		if !ok {
			_ = app.RemoveServeRuntime(flagDataDir)
			fmt.Printf("Basalt serve stopped.\nPID: %d\nData dir: %s\n", runtime.PID, runtime.DataDir)
			return nil
		}
		time.Sleep(250 * time.Millisecond)
	}

	if !flagServeForce {
		return fmt.Errorf("timed out waiting for serve process %d to stop; retry with --stop --force", runtime.PID)
	}

	if err := process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return fmt.Errorf("force-stopping serve process %d: %w", runtime.PID, err)
	}
	_ = app.RemoveServeRuntime(flagDataDir)
	fmt.Printf("Basalt serve force-stopped.\nPID: %d\nData dir: %s\n", runtime.PID, runtime.DataDir)
	return nil
}

func validateServeMode() error {
	modes := 0
	for _, enabled := range []bool{flagServeDetach, flagServeStatus, flagServeStop} {
		if enabled {
			modes++
		}
	}
	if modes > 1 {
		return fmt.Errorf("only one of --detach, --status, or --stop may be used at a time")
	}
	if flagServeForce && !flagServeStop {
		return fmt.Errorf("--force can only be used with --stop")
	}
	return nil
}

func readServeRuntimeIfPresent(dataDir string) (*app.ServeRuntime, error) {
	runtime, err := app.ReadServeRuntime(dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, err
		}
		return nil, fmt.Errorf("reading serve runtime: %w", err)
	}
	return runtime, nil
}

func serveHealthy(baseURL string) (bool, error) {
	if baseURL == "" {
		return false, fmt.Errorf("empty base URL")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/", nil)
	if err != nil {
		return false, err
	}

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode == http.StatusOK, nil
}
