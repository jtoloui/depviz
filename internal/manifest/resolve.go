package manifest

import (
	"path/filepath"
	"strings"

	"github.com/jtoloui/depviz/internal/config"
	"github.com/jtoloui/depviz/internal/scanner"
)

// BarePackageName extracts the npm package name from an import specifier.
// Returns the package name and true, or ("", false) for relative/absolute paths.
// Examples: "lodash/fp" → "lodash", "@aws-sdk/client-s3/foo" → "@aws-sdk/client-s3"
func BarePackageName(imp string) (string, bool) {
	if strings.HasPrefix(imp, ".") || strings.HasPrefix(imp, "/") {
		return "", false
	}
	// Node built-in prefix — not a package.json dep.
	if strings.HasPrefix(imp, "node:") {
		return "", false
	}
	if strings.HasPrefix(imp, "@") {
		parts := strings.SplitN(imp, "/", 3)
		if len(parts) >= 2 {
			return parts[0] + "/" + parts[1], true
		}
		return imp, true
	}
	parts := strings.SplitN(imp, "/", 2)
	return parts[0], true
}

// Analyze cross-references scanned imports against package.json declarations
// and returns a report of unused, phantom, and used dependencies.
func Analyze(root string, results []scanner.FileImports, cfg *config.Config) ([]DepReport, error) {
	pkgs, err := discoverPackages(root)
	if err != nil {
		return nil, err
	}
	if len(pkgs) == 0 {
		return nil, nil
	}

	// Build set of workspace package names for exclusion.
	workspaceNames := make(map[string]bool)
	for _, pkg := range pkgs {
		if pkg.Name != "" {
			workspaceNames[pkg.Name] = true
		}
	}

	// Merge all declared deps across workspace packages.
	declared := make(map[string]depEntry)
	for _, pkg := range pkgs {
		for name, version := range pkg.Dependencies {
			if _, exists := declared[name]; !exists {
				declared[name] = depEntry{version: version, depType: Dep}
			}
		}
		for name, version := range pkg.DevDependencies {
			if _, exists := declared[name]; !exists {
				declared[name] = depEntry{version: version, depType: DevDep}
			}
		}
	}

	// Collect all bare package names actually imported.
	imported := make(map[string]bool)
	for _, r := range results {
		if r.Lang != "js" {
			continue
		}
		for _, imp := range r.Imports {
			if name, ok := BarePackageName(imp); ok {
				imported[name] = true
			}
		}
	}

	// Also mark packages referenced in package.json scripts as used.
	for name := range scriptsUsedPackages(pkgs, declared) {
		imported[name] = true
	}

	// Build ignore matcher from config.
	ignoreSet := buildIgnoreMatcher(cfg.Manifest.IgnoreUnused)

	var report []DepReport

	// Check declared deps: used or unused?
	for name, entry := range declared {
		// Skip workspace siblings — they're internal.
		if workspaceNames[name] {
			continue
		}
		// Skip ignored packages.
		if ignoreSet.matches(name) {
			continue
		}

		status := Unused
		if imported[name] {
			status = Used
		}
		report = append(report, DepReport{
			Name:    name,
			Version: entry.version,
			Type:    entry.depType,
			Status:  status,
		})
	}

	// Check for phantom deps: imported but not declared.
	for name := range imported {
		if _, isDeclared := declared[name]; isDeclared {
			continue
		}
		// Skip workspace siblings.
		if workspaceNames[name] {
			continue
		}
		// Skip node builtins (already handled by classifier, but be safe).
		if isNodeBuiltin(name) {
			continue
		}
		report = append(report, DepReport{
			Name:   name,
			Type:   Dep,
			Status: Phantom,
		})
	}

	return report, nil
}

type depEntry struct {
	version string
	depType DepType
}

// discoverPackages finds all package.json files relevant to the project.
func discoverPackages(root string) ([]*PackageJSON, error) {
	rootPkg := filepath.Join(root, "package.json")
	pkg, err := Parse(rootPkg)
	if err != nil {
		// No package.json at root — nothing to analyze.
		return nil, nil //nolint:nilerr
	}

	pkgs := []*PackageJSON{pkg}

	// If workspaces are declared, discover child packages.
	if pkg.HasWorkspaces() {
		children, err := resolveWorkspaces(root, pkg.Workspaces.Patterns)
		if err != nil {
			return pkgs, nil // Graceful: return root only.
		}
		pkgs = append(pkgs, children...)
	}

	return pkgs, nil
}

// ignoreMatcher matches package names against glob-like patterns.
type ignoreMatcher struct {
	exact    map[string]bool
	prefixes []string // patterns ending with *
}

func buildIgnoreMatcher(patterns []string) ignoreMatcher {
	m := ignoreMatcher{exact: make(map[string]bool)}
	for _, p := range patterns {
		if strings.HasSuffix(p, "*") {
			m.prefixes = append(m.prefixes, strings.TrimSuffix(p, "*"))
		} else {
			m.exact[p] = true
		}
	}
	return m
}

func (m ignoreMatcher) matches(name string) bool {
	if m.exact[name] {
		return true
	}
	for _, prefix := range m.prefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// scriptsUsedPackages extracts package names referenced in package.json scripts.
// Catches CLI tools like "cross-env", "webpack-cli", "storybook", etc.
func scriptsUsedPackages(pkgs []*PackageJSON, declared map[string]depEntry) map[string]bool {
	used := make(map[string]bool)
	// Separators that split commands in shell scripts.
	separators := []string{"&&", "||", ";", "|", "&"}

	for _, pkg := range pkgs {
		for _, script := range pkg.Scripts {
			tokens := tokeniseScript(script, separators)
			for _, token := range tokens {
				// Strip common prefixes.
				token = strings.TrimPrefix(token, "npx ")
				token = strings.TrimPrefix(token, "npx")
				if strings.HasPrefix(token, "./node_modules/.bin/") {
					token = strings.TrimPrefix(token, "./node_modules/.bin/")
				}
				// Check if this token is a declared package name.
				if _, ok := declared[token]; ok {
					used[token] = true
				}
				// Handle scoped packages: token might be just the first part.
				// e.g. script has "@datadog/datadog-ci" as a single token.
				for name := range declared {
					if strings.HasPrefix(name, "@") && token == name {
						used[name] = true
					}
				}
			}
		}
	}
	return used
}

// tokeniseScript splits a shell script string into command tokens.
func tokeniseScript(script string, separators []string) []string {
	// First split on separators.
	parts := []string{script}
	for _, sep := range separators {
		var next []string
		for _, p := range parts {
			next = append(next, strings.Split(p, sep)...)
		}
		parts = next
	}
	// Then extract the first word of each command (the binary name).
	var tokens []string
	for _, part := range parts {
		fields := strings.Fields(strings.TrimSpace(part))
		if len(fields) > 0 {
			tokens = append(tokens, fields[0])
		}
	}
	return tokens
}

// isNodeBuiltin checks if a package name is a Node.js built-in module.
func isNodeBuiltin(name string) bool {
	builtins := map[string]bool{
		"assert": true, "buffer": true, "child_process": true, "cluster": true,
		"console": true, "constants": true, "crypto": true, "dgram": true,
		"dns": true, "domain": true, "events": true, "fs": true, "http": true,
		"http2": true, "https": true, "module": true, "net": true, "os": true,
		"path": true, "perf_hooks": true, "process": true, "punycode": true,
		"querystring": true, "readline": true, "repl": true, "stream": true,
		"string_decoder": true, "sys": true, "timers": true, "tls": true,
		"tty": true, "url": true, "util": true, "v8": true, "vm": true,
		"worker_threads": true, "zlib": true,
	}
	return builtins[name]
}
