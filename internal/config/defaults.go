package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DefaultIgnoreUnused is the built-in list of packages that are commonly
// declared in package.json but never explicitly imported in source code.
var DefaultIgnoreUnused = []string{
	"@types/*",
	"typescript",
	// Git hooks & commit tooling
	"husky",
	"lint-staged",
	"commitlint*",
	"@commitlint/*",
	// Formatters & linters (CLI-only)
	"prettier",
	"eslint*",
	"@biomejs/*",
	"stylelint*",
	// Dev runners & CLI tools
	"concurrently",
	"nodemon",
	"ts-node",
	"tsx",
	"plop",
	"cross-env",
	"npm-watch",
	"patch-package",
	// Build tooling plugins (referenced in config, not imported)
	"@babel/*",
	"babel-*",
	"postcss*",
	"autoprefixer",
	"tailwindcss",
	"esbuild-plugin-*",
	"sass",
	"core-js",
	// Webpack loaders & plugins
	"@svgr/*",
	"*-loader",
	// Test tooling (referenced in config)
	"@vitest/coverage-*",
	"@swc/*",
	"identity-obj-proxy",
	// Storybook (addons referenced in .storybook/main config)
	"@storybook/*",
	"storybook",
	// Cypress plugins
	"cypress-*",
	"mochawesome*",
	// Prettier configs
	"*-prettier-config",
}

// DefaultFor returns the built-in default config for a language.
// For Go, it reads the module path from go.mod to set internal patterns.
func DefaultFor(lang, root string) (*Config, error) {
	switch lang {
	case "js":
		return defaultJS(), nil
	case "go":
		return defaultGo(root)
	case "multi":
		return defaultMulti(root)
	default:
		return nil, fmt.Errorf("unsupported language: %q", lang)
	}
}

func defaultJS() *Config {
	return &Config{
		Language: "js",
		Exclude:  []string{"node_modules", ".git", "dist", "build", ".next", "coverage", ".depviz"},
		Classify: ClassifyRules{
			Internal: []string{`^\.\.?/.*`},
		},
		Manifest: ManifestConfig{
			IgnoreUnused: DefaultIgnoreUnused,
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
		Exclude:  []string{"vendor", ".git", ".depviz"},
		Classify: ClassifyRules{
			Internal: []string{`^` + regexpEscape(mod) + `/.*`},
		},
	}, nil
}

func defaultMulti(root string) (*Config, error) {
	jsCfg := defaultJS()

	goCfg, err := defaultGo(root)
	if err != nil {
		// No go.mod — use JS defaults only (pure JS/TS project).
		return &Config{
			Language: "multi",
			Exclude:  jsCfg.Exclude,
			Classify: jsCfg.Classify,
			Manifest: ManifestConfig{
				IgnoreUnused: DefaultIgnoreUnused,
			},
		}, nil
	}

	exclude := append(goCfg.Exclude, jsCfg.Exclude...)
	internal := append(goCfg.Classify.Internal, jsCfg.Classify.Internal...)

	return &Config{
		Language: "multi",
		Exclude:  exclude,
		Classify: ClassifyRules{Internal: internal},
		Manifest: ManifestConfig{
			IgnoreUnused: DefaultIgnoreUnused,
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
