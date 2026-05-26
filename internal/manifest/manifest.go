package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// DepType classifies which section of package.json a dependency lives in.
type DepType string

const (
	Dep    DepType = "dependency"
	DevDep DepType = "devDependency"
)

// DepStatus describes the relationship between a declared dep and actual usage.
type DepStatus string

const (
	Used    DepStatus = "used"
	Unused  DepStatus = "unused"
	Phantom DepStatus = "phantom" // imported but not declared in package.json
)

// DepReport is a single dependency's analysis result.
type DepReport struct {
	Name    string    `json:"name"`
	Version string    `json:"version,omitempty"`
	Type    DepType   `json:"type"`
	Status  DepStatus `json:"status"`
}

// PackageJSON represents the relevant fields of a package.json file.
type PackageJSON struct {
	Name            string            `json:"name"`
	Dir             string            `json:"-"` // directory containing this package.json
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
	Scripts         map[string]string `json:"scripts"`
	Workspaces      workspaceField    `json:"-"`
}

// workspaceField handles both string array and object forms of "workspaces".
type workspaceField struct {
	Patterns []string
}

// Parse reads and parses a package.json at the given path.
func Parse(path string) (*PackageJSON, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var pkg PackageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	// Parse workspaces field (can be []string or {packages: []string}).
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err == nil {
		if ws, ok := raw["workspaces"]; ok {
			pkg.Workspaces = parseWorkspaces(ws)
		}
	}

	pkg.Dir = filepath.Dir(path)
	if pkg.Dependencies == nil {
		pkg.Dependencies = map[string]string{}
	}
	if pkg.DevDependencies == nil {
		pkg.DevDependencies = map[string]string{}
	}

	return &pkg, nil
}

// parseWorkspaces handles both forms:
//   - ["packages/*", "apps/*"]
//   - {"packages": ["packages/*", "apps/*"]}
func parseWorkspaces(raw json.RawMessage) workspaceField {
	// Try array form first.
	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil {
		return workspaceField{Patterns: arr}
	}

	// Try object form (yarn/npm).
	var obj struct {
		Packages []string `json:"packages"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil {
		return workspaceField{Patterns: obj.Packages}
	}

	return workspaceField{}
}

// AllDeclared returns every dependency name from both deps and devDeps.
func (p *PackageJSON) AllDeclared() map[string]DepType {
	m := make(map[string]DepType, len(p.Dependencies)+len(p.DevDependencies))
	for name := range p.Dependencies {
		m[name] = Dep
	}
	for name := range p.DevDependencies {
		m[name] = DevDep
	}
	return m
}

// HasWorkspaces returns true if this package.json declares workspace patterns.
func (p *PackageJSON) HasWorkspaces() bool {
	return len(p.Workspaces.Patterns) > 0
}
