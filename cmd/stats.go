package cmd

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/jtoloui/depviz/internal/classify"
	"github.com/jtoloui/depviz/internal/cli"
	"github.com/jtoloui/depviz/internal/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(statsCmd)
}

var statsCmd = &cobra.Command{
	Use:   "stats [path]",
	Short: "Show dependency statistics for a project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
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

		cli.Stats(results, cl)
		return nil
	},
}
