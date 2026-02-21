package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/jtoloui/depviz/internal/classify"
	"github.com/jtoloui/depviz/internal/config"
	"github.com/jtoloui/depviz/internal/render"
	"github.com/jtoloui/depviz/internal/scanner"
	"github.com/spf13/cobra"
)

var output string

func init() {
	scanCmd.Flags().StringVarP(&output, "output", "o", "", "output file path (default: <project>/.depviz/deps.html)")
	rootCmd.AddCommand(scanCmd)
}

var scanCmd = &cobra.Command{
	Use:   "scan [path]",
	Short: "Scan a project and generate a dependency map",
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
		slog.Info("scan complete", "files", len(results))

		out := resolveOutput(cfg, output, root)
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return fmt.Errorf("creating output dir: %w", err)
		}

		f, err := os.Create(out)
		if err != nil {
			return fmt.Errorf("creating output: %w", err)
		}

		if err := render.HTML(f, root, results, cl); err != nil {
			_ = f.Close()
			return fmt.Errorf("rendering: %w", err)
		}

		if err := f.Close(); err != nil {
			return fmt.Errorf("closing output: %w", err)
		}

		slog.Info("output written", "path", out)
		return nil
	},
}

func getScanner(cfg *config.Config) (scanner.Scanner, error) {
	switch cfg.Language {
	case "go":
		return scanner.NewGoScanner(cfg), nil
	case "js":
		return scanner.NewTreeSitterScanner(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported language: %q", cfg.Language)
	}
}

func resolveOutput(cfg *config.Config, flagOutput, root string) string {
	if cfg.Output != "" {
		return cfg.Output
	}
	if flagOutput != "" {
		return flagOutput
	}
	return filepath.Join(root, ".depviz", "deps.html")
}
