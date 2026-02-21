package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/jtoloui/depviz/internal/classify"
	"github.com/jtoloui/depviz/internal/cli"
	"github.com/jtoloui/depviz/internal/config"
	"github.com/jtoloui/depviz/internal/render"
	"github.com/spf13/cobra"
)

var port string

func init() {
	serveCmd.Flags().StringVarP(&port, "port", "p", "3000", "port to serve on")
	rootCmd.AddCommand(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:   "serve [path]",
	Short: "Scan a project and serve the dependency map in the browser",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cli.Banner()

		root, err := filepath.Abs(args[0])
		if err != nil {
			return err
		}

		cfg, err := config.Load(root, lang)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		slog.Debug("config loaded", "language", cfg.Language, "excludes", len(cfg.Exclude))

		s, err := getScanner(cfg)
		if err != nil {
			return err
		}

		cl, err := classify.New(cfg)
		if err != nil {
			return fmt.Errorf("creating classifier: %w", err)
		}

		slog.Debug("scanning", "root", root)
		results, err := s.Scan(root)
		if err != nil {
			return fmt.Errorf("scanning: %w", err)
		}

		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			if err := render.HTML(w, root, results, cl); err != nil {
				http.Error(w, "render error", http.StatusInternalServerError)
			}
		})

		addr := resolvePort(cfg, port)
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			slog.Debug("port in use, picking free port", "tried", addr)
			ln, err = net.Listen("tcp", ":0")
			if err != nil {
				return fmt.Errorf("finding free port: %w", err)
			}
		}

		srv := &http.Server{
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		}

		ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		go func() {
			<-ctx.Done()
			slog.Info("shutting down server")
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = srv.Shutdown(shutdownCtx)
		}()

		actualPort := ln.Addr().(*net.TCPAddr).Port
		cli.ServeResult(results, actualPort)

		if err := srv.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	},
}

func resolvePort(cfg *config.Config, flagPort string) string {
	if cfg.Port != 0 {
		return fmt.Sprintf(":%d", cfg.Port)
	}
	return ":" + flagPort
}
