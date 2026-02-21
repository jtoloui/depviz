package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "depviz",
	Short: "Visualise Go and JS/TS project dependencies",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		var level slog.Level
		if verbose {
			level = slog.LevelDebug
		}
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: level,
		})))
	},
}

var (
	lang    string
	verbose bool
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&lang, "lang", "l", "go", "language: go, js")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable debug logging")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func SetVersion(v string) {
	rootCmd.Version = v
}
