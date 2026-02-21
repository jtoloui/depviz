package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DefaultFor returns the built-in default config for a language.
// For Go, it reads the module path from go.mod to set internal patterns.
func DefaultFor(lang, root string) (*Config, error) {
	switch lang {
	case "js":
		return defaultJS(), nil
	case "go":
		return defaultGo(root)
	default:
		return nil, fmt.Errorf("unsupported language: %q", lang)
	}
}

func defaultJS() *Config {
	return &Config{
		Language: "js",
		Exclude:  []string{"node_modules", ".git", "dist", "build", ".next", "coverage"},
		Classify: ClassifyRules{
			Internal: []string{`^\.\.?/.*`},
		},
	}
}

func defaultGo(root string) (*Config, error) {
	mod, err := readModulePath(root)
	if err != nil {
		return nil, fmt.Errorf("reading go.mod: %w", err)
	}

	return &Config{
		Language: "go",
		Exclude:  []string{"vendor", ".git"},
		Classify: ClassifyRules{
			Internal: []string{`^` + regexpEscape(mod) + `/.*`},
		},
	}, nil
}

// readModulePath extracts the module path from go.mod.
func readModulePath(root string) (string, error) {
	f, err := os.Open(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module")), nil
		}
	}
	return "", fmt.Errorf("module directive not found in go.mod")
}

// regexp_escape escapes dots and slashes for use in regex patterns.
func regexpEscape(s string) string {
	s = strings.ReplaceAll(s, ".", `\.`)
	s = strings.ReplaceAll(s, "/", `\/`)
	return s
}
