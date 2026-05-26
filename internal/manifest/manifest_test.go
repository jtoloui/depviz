package manifest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jtoloui/depviz/internal/config"
	"github.com/jtoloui/depviz/internal/manifest"
	"github.com/jtoloui/depviz/internal/scanner"
)

func TestBarePackageName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{"lodash", "lodash", true},
		{"lodash/fp", "lodash", true},
		{"@aws-sdk/client-s3", "@aws-sdk/client-s3", true},
		{"@aws-sdk/client-s3/foo", "@aws-sdk/client-s3", true},
		{"@scope/pkg", "@scope/pkg", true},
		{"./utils", "", false},
		{"../config", "", false},
		{"/absolute/path", "", false},
		{"node:fs", "", false},
		{"node:path", "", false},
		{"react", "react", true},
		{"react-dom/client", "react-dom", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got, ok := manifest.BarePackageName(tt.input)
			if ok != tt.ok {
				t.Fatalf("BarePackageName(%q) ok = %v, want %v", tt.input, ok, tt.ok)
			}
			if got != tt.want {
				t.Errorf("BarePackageName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParse(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pkgJSON := `{
		"name": "test-project",
		"dependencies": { "express": "^4.18.0", "lodash": "^4.17.21" },
		"devDependencies": { "jest": "^29.0.0", "typescript": "^5.0.0" }
	}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkgJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	pkg, err := manifest.Parse(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatal(err)
	}

	if pkg.Name != "test-project" {
		t.Errorf("Name = %q, want %q", pkg.Name, "test-project")
	}
	if len(pkg.Dependencies) != 2 {
		t.Errorf("Dependencies count = %d, want 2", len(pkg.Dependencies))
	}
	if len(pkg.DevDependencies) != 2 {
		t.Errorf("DevDependencies count = %d, want 2", len(pkg.DevDependencies))
	}
	if pkg.Dependencies["express"] != "^4.18.0" {
		t.Errorf("express version = %q, want %q", pkg.Dependencies["express"], "^4.18.0")
	}
}

func TestParseWorkspacesArray(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pkgJSON := `{ "name": "root", "workspaces": ["packages/*", "apps/*"] }`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkgJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	pkg, err := manifest.Parse(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatal(err)
	}

	if !pkg.HasWorkspaces() {
		t.Fatal("expected HasWorkspaces() = true")
	}
}

func TestParseWorkspacesObject(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pkgJSON := `{ "name": "root", "workspaces": { "packages": ["packages/*"] } }`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkgJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	pkg, err := manifest.Parse(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatal(err)
	}

	if !pkg.HasWorkspaces() {
		t.Fatal("expected HasWorkspaces() = true")
	}
}

func TestAnalyzeUnusedAndPhantom(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pkgJSON := `{
		"name": "test",
		"dependencies": { "express": "^4.18.0", "lodash": "^4.17.21", "unused-pkg": "^1.0.0" },
		"devDependencies": { "jest": "^29.0.0" }
	}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkgJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	results := []scanner.FileImports{
		{
			File:    "index.ts",
			Lang:    "js",
			Imports: []string{"express", "lodash/fp", "./utils", "phantom-pkg"},
		},
		{
			File:    "test.spec.ts",
			Lang:    "js",
			Imports: []string{"jest"},
		},
	}

	cfg := &config.Config{
		Language: "js",
		Manifest: config.ManifestConfig{IgnoreUnused: nil},
	}

	report, err := manifest.Analyze(dir, results, cfg)
	if err != nil {
		t.Fatal(err)
	}

	statusMap := map[string]manifest.DepStatus{}
	for _, r := range report {
		statusMap[r.Name] = r.Status
	}

	if statusMap["express"] != manifest.Used {
		t.Errorf("express status = %q, want %q", statusMap["express"], manifest.Used)
	}
	if statusMap["lodash"] != manifest.Used {
		t.Errorf("lodash status = %q, want %q", statusMap["lodash"], manifest.Used)
	}
	if statusMap["unused-pkg"] != manifest.Unused {
		t.Errorf("unused-pkg status = %q, want %q", statusMap["unused-pkg"], manifest.Unused)
	}
	if statusMap["jest"] != manifest.Used {
		t.Errorf("jest status = %q, want %q", statusMap["jest"], manifest.Used)
	}
	if statusMap["phantom-pkg"] != manifest.Phantom {
		t.Errorf("phantom-pkg status = %q, want %q", statusMap["phantom-pkg"], manifest.Phantom)
	}
}

func TestAnalyzeIgnoreList(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pkgJSON := `{
		"name": "test",
		"dependencies": { "express": "^4.18.0" },
		"devDependencies": { "@types/node": "^20.0.0", "eslint-plugin-react": "^7.0.0" }
	}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkgJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	results := []scanner.FileImports{
		{File: "index.ts", Lang: "js", Imports: []string{"express"}},
	}

	cfg := &config.Config{
		Language: "js",
		Manifest: config.ManifestConfig{
			IgnoreUnused: []string{"@types/*", "eslint*"},
		},
	}

	report, err := manifest.Analyze(dir, results, cfg)
	if err != nil {
		t.Fatal(err)
	}

	for _, r := range report {
		if r.Name == "@types/node" || r.Name == "eslint-plugin-react" {
			t.Errorf("expected %q to be ignored, but got status %q", r.Name, r.Status)
		}
	}
}

func TestAnalyzeMonorepo(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Root package.json with workspaces.
	rootPkg := `{ "name": "root", "workspaces": ["packages/*"] }`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(rootPkg), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create workspace packages.
	coreDir := filepath.Join(dir, "packages", "core")
	webDir := filepath.Join(dir, "packages", "web")
	if err := os.MkdirAll(coreDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(webDir, 0o755); err != nil {
		t.Fatal(err)
	}

	corePkg := `{ "name": "@myorg/core", "dependencies": { "zod": "^3.0.0" } }`
	if err := os.WriteFile(filepath.Join(coreDir, "package.json"), []byte(corePkg), 0o644); err != nil {
		t.Fatal(err)
	}

	webPkg := `{ "name": "@myorg/web", "dependencies": { "@myorg/core": "workspace:*", "react": "^18.0.0" } }`
	if err := os.WriteFile(filepath.Join(webDir, "package.json"), []byte(webPkg), 0o644); err != nil {
		t.Fatal(err)
	}

	results := []scanner.FileImports{
		{File: "packages/web/index.ts", Lang: "js", Imports: []string{"@myorg/core", "react"}},
		{File: "packages/core/index.ts", Lang: "js", Imports: []string{"zod"}},
	}

	cfg := &config.Config{
		Language: "js",
		Manifest: config.ManifestConfig{IgnoreUnused: nil},
	}

	report, err := manifest.Analyze(dir, results, cfg)
	if err != nil {
		t.Fatal(err)
	}

	statusMap := map[string]manifest.DepStatus{}
	for _, r := range report {
		statusMap[r.Name] = r.Status
	}

	// @myorg/core should NOT appear — it's a workspace sibling.
	if _, exists := statusMap["@myorg/core"]; exists {
		t.Error("@myorg/core should be excluded as workspace sibling")
	}

	if statusMap["react"] != manifest.Used {
		t.Errorf("react status = %q, want %q", statusMap["react"], manifest.Used)
	}
	if statusMap["zod"] != manifest.Used {
		t.Errorf("zod status = %q, want %q", statusMap["zod"], manifest.Used)
	}
}

func TestAnalyzeNoPackageJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	results := []scanner.FileImports{
		{File: "index.ts", Lang: "js", Imports: []string{"express"}},
	}

	cfg := &config.Config{Language: "js"}

	report, err := manifest.Analyze(dir, results, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if report != nil {
		t.Errorf("expected nil report when no package.json, got %d entries", len(report))
	}
}
