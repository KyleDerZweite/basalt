// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"time"

	"github.com/spf13/cobra"

	"github.com/KyleDerZweite/basalt/internal/app"
	"github.com/KyleDerZweite/basalt/internal/webui"
)

var (
	flagWebListen string
	flagWebOpen   bool
	flagWebNoOpen bool
)

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Run the local Basalt web workspace",
	Long: `Run the local Basalt web workspace on top of the same persisted local backend.

The web command serves the browser UI and the API from the same localhost origin.`,
	RunE: runWeb,
}

func init() {
	rootCmd.AddCommand(webCmd)

	webCmd.Flags().StringVar(&flagDataDir, "data-dir", app.DefaultDataDir(), "Path to local app data directory")
	webCmd.Flags().StringVar(&flagWebListen, "listen", "127.0.0.1:8788", "Listen address for the local web workspace")
	webCmd.Flags().BoolVar(&flagWebOpen, "open", true, "Open the web workspace in the system browser after startup")
	webCmd.Flags().BoolVar(&flagWebNoOpen, "no-open", false, "Do not open the system browser automatically")
}

func runWeb(cmd *cobra.Command, args []string) error {
	if cmd.Flags().Changed("open") && cmd.Flags().Changed("no-open") {
		return fmt.Errorf("only one of --open or --no-open may be used at a time")
	}
	shouldOpen := flagWebOpen
	if flagWebNoOpen {
		shouldOpen = false
	}

	service, err := app.NewService(Version, flagDataDir)
	if err != nil {
		return err
	}
	defer service.Close()

	listener, err := net.Listen("tcp", flagWebListen)
	if err != nil {
		return err
	}
	defer listener.Close()

	baseURL := "http://" + listener.Addr().String()
	server := &http.Server{
		Addr:              listener.Addr().String(),
		Handler:           webui.NewServer(service, webui.Options{BaseURL: baseURL}),
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(listener)
	}()

	if err := waitForWebReady(baseURL); err != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
		return err
	}

	fmt.Fprintf(os.Stderr, "Basalt web workspace listening on %s\n", baseURL)
	fmt.Fprintf(os.Stderr, "Data dir: %s\n", service.DataDir())

	if shouldOpen {
		if err := openBrowser(baseURL); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not open browser automatically: %v\n", err)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
		err := <-errCh
		if err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	}
}

func waitForWebReady(baseURL string) error {
	client := &http.Client{}
	deadline := time.Now().Add(10 * time.Second)
	paths := []string{"/", "/app/bootstrap", "/api/settings"}
	for time.Now().Before(deadline) {
		allReady := true
		for _, route := range paths {
			ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+route, nil)
			if err != nil {
				cancel()
				return err
			}
			resp, err := client.Do(req)
			if err != nil {
				cancel()
				allReady = false
				time.Sleep(150 * time.Millisecond)
				break
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			cancel()
			if resp.StatusCode != http.StatusOK {
				allReady = false
				time.Sleep(150 * time.Millisecond)
				break
			}
		}
		if allReady {
			return nil
		}
		time.Sleep(150 * time.Millisecond)
	}
	return fmt.Errorf("web workspace did not become ready at %s in time", baseURL)
}

var openBrowser = func(target string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.Command("open", target)
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		command = exec.Command("xdg-open", target)
	}
	return command.Start()
}
