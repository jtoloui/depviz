package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/jtoloui/depviz/internal/cli"
	"github.com/jtoloui/depviz/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Generate a .depviz.yml config file",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root := "."
		if len(args) > 0 {
			root = args[0]
		}
		root, err := filepath.Abs(root)
		if err != nil {
			return err
		}

		out := filepath.Join(root, ".depviz.yml")
		if _, err := os.Stat(out); err == nil {
			return fmt.Errorf(".depviz.yml already exists in %s", root)
		}

		detected := detectLang(root)

		var lang string
		var excludes []string
		var extraExcludes string
		var internalStr string
		var privateStr string
		var confirm bool

		defaults, _ := config.DefaultFor(detected, root)
		if defaults == nil {
			defaults = &config.Config{Language: detected}
		}

		// Pre-build exclude options from defaults
		excludeOpts := make([]huh.Option[string], 0, len(defaults.Exclude))
		for _, e := range defaults.Exclude {
			excludeOpts = append(excludeOpts, huh.NewOption(e, e).Selected(true))
		}

		// Pre-fill patterns as comma-separated
		defaultInternal := strings.Join(defaults.Classify.Internal, ", ")
		defaultPrivate := strings.Join(defaults.Classify.Private, ", ")

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Language").
					Description("Detected: "+detected).
					Options(
						huh.NewOption("Go", "go"),
						huh.NewOption("JavaScript/TypeScript", "js"),
						huh.NewOption("Multi (Go + JS/TS)", "multi"),
					).
					Value(&lang),

				huh.NewMultiSelect[string]().
					Title("Exclude directories").
					Options(excludeOpts...).
					Value(&excludes),

				huh.NewInput().
					Title("Additional excludes (comma-separated, optional)").
					Placeholder("e.g. bin, tmp, out").
					Value(&extraExcludes),

				huh.NewInput().
					Title("Internal import patterns (comma-separated)").
					Description("Regex patterns for your project's imports").
					Value(&internalStr).
					Placeholder(defaultInternal),

				huh.NewInput().
					Title("Private/org import patterns (comma-separated, optional)").
					Description("Regex patterns for your org's packages").
					Value(&privateStr).
					Placeholder(defaultPrivate),
			),

			huh.NewGroup(
				huh.NewConfirm().
					Title("Write .depviz.yml?").
					Value(&confirm),
			),
		)

		if err := form.Run(); err != nil {
			return err
		}

		if !confirm {
			fmt.Println("Cancelled.")
			return nil
		}

		cfg := &config.Config{
			Language: lang,
			Exclude:  excludes,
		}
		if extraExcludes != "" {
			excludes = append(excludes, splitCSV(extraExcludes)...)
			cfg.Exclude = excludes
		}
		if internalStr == "" {
			internalStr = defaultInternal
		}
		if internalStr != "" {
			cfg.Classify.Internal = splitCSV(internalStr)
		}
		if privateStr != "" {
			cfg.Classify.Private = splitCSV(privateStr)
		}

		data, err := yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("marshalling config: %w", err)
		}

		if err := os.WriteFile(out, data, 0o644); err != nil {
			return fmt.Errorf("writing config: %w", err)
		}

		cli.InitResult(lang, out)
		return nil
	},
}

func detectLang(root string) string {
	hasGo := fileExists(filepath.Join(root, "go.mod"))
	hasJS := fileExists(filepath.Join(root, "package.json"))

	if hasGo && hasJS {
		return "multi"
	}
	if hasJS {
		return "js"
	}
	return "go"
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func splitCSV(s string) []string {
	var out []string
	for _, v := range strings.Split(s, ",") {
		v = strings.TrimSpace(v)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}
