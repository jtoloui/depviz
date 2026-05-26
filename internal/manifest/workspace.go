package manifest

import (
	"os"
	"path/filepath"
	"strings"
)

// resolveWorkspaces expands workspace glob patterns and parses each child package.json.
func resolveWorkspaces(root string, patterns []string) ([]*PackageJSON, error) {
	var pkgs []*PackageJSON

	for _, pattern := range patterns {
		dirs, err := expandWorkspacePattern(root, pattern)
		if err != nil {
			continue // Skip unresolvable patterns.
		}
		for _, dir := range dirs {
			pkgPath := filepath.Join(dir, "package.json")
			pkg, err := Parse(pkgPath)
			if err != nil {
				continue // Skip directories without valid package.json.
			}
			pkgs = append(pkgs, pkg)
		}
	}

	return pkgs, nil
}

// expandWorkspacePattern resolves a workspace pattern to directories.
// Supports: "packages/*", "apps/*", "packages/foo", etc.
func expandWorkspacePattern(root, pattern string) ([]string, error) {
	full := filepath.Join(root, pattern)

	// If pattern ends with /*, glob for directories.
	if strings.HasSuffix(pattern, "/*") || strings.HasSuffix(pattern, "\\*") {
		parent := filepath.Dir(full)
		entries, err := os.ReadDir(parent)
		if err != nil {
			return nil, err
		}
		var dirs []string
		for _, e := range entries {
			if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
				dirs = append(dirs, filepath.Join(parent, e.Name()))
			}
		}
		return dirs, nil
	}

	// Try filepath.Glob for other patterns.
	matches, err := filepath.Glob(full)
	if err != nil {
		return nil, err
	}

	var dirs []string
	for _, m := range matches {
		info, err := os.Stat(m)
		if err == nil && info.IsDir() {
			dirs = append(dirs, m)
		}
	}

	// If no glob matches, treat as a literal directory.
	if len(dirs) == 0 {
		info, err := os.Stat(full)
		if err == nil && info.IsDir() {
			dirs = append(dirs, full)
		}
	}

	return dirs, nil
}

// DetectWorkspaceRoot walks up from a directory to find the nearest package.json
// with a workspaces field. Returns the root path or empty string if not found.
func DetectWorkspaceRoot(dir string) string {
	for {
		pkgPath := filepath.Join(dir, "package.json")
		if pkg, err := Parse(pkgPath); err == nil && pkg.HasWorkspaces() {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}
