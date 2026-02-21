package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jtoloui/depviz/internal/config"
)

func TestLoad_FromFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	yaml := `language: js
exclude:
  - node_modules
  - out
classify:
  internal:
    - "^\\.\\./.*"
  private:
    - "^@myorg/.*"
`
	if err := os.WriteFile(filepath.Join(dir, ".depviz.yml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(dir, "go") // flag says go, file says js — file wins
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Language != "js" {
		t.Errorf("Language = %q, want %q", cfg.Language, "js")
	}
	if len(cfg.Exclude) != 2 {
		t.Errorf("Exclude len = %d, want 2", len(cfg.Exclude))
	}
	if len(cfg.Classify.Private) != 1 {
		t.Errorf("Private len = %d, want 1", len(cfg.Classify.Private))
	}
}

func TestLoad_FallbackJS(t *testing.T) {
	t.Parallel()

	dir := t.TempDir() // no .depviz.yml

	cfg, err := config.Load(dir, "js")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Language != "js" {
		t.Errorf("Language = %q, want %q", cfg.Language, "js")
	}
	if len(cfg.Exclude) == 0 {
		t.Error("expected default excludes, got none")
	}
}

func TestLoad_FallbackGo(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gomod := "module github.com/example/test\n\ngo 1.22\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(dir, "go")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Language != "go" {
		t.Errorf("Language = %q, want %q", cfg.Language, "go")
	}
	if len(cfg.Classify.Internal) != 1 {
		t.Fatalf("Internal len = %d, want 1", len(cfg.Classify.Internal))
	}
	want := `^github\.com\/example\/test/.*`
	if cfg.Classify.Internal[0] != want {
		t.Errorf("Internal[0] = %q, want %q", cfg.Classify.Internal[0], want)
	}
}

func TestLoad_InvalidLanguage(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	yaml := "language: python\n"
	if err := os.WriteFile(filepath.Join(dir, ".depviz.yml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := config.Load(dir, "go")
	if err == nil {
		t.Fatal("expected error for unsupported language, got nil")
	}
}

func TestLoad_InvalidRegex(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	yaml := "language: js\nclassify:\n  internal:\n    - \"[bad\"\n"
	if err := os.WriteFile(filepath.Join(dir, ".depviz.yml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := config.Load(dir, "js")
	if err == nil {
		t.Fatal("expected error for invalid regex, got nil")
	}
}

func TestLoad_EmptyExclude(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	yaml := "language: js\nexclude:\n  - \"\"\n"
	if err := os.WriteFile(filepath.Join(dir, ".depviz.yml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := config.Load(dir, "js")
	if err == nil {
		t.Fatal("expected error for empty exclude, got nil")
	}
}

func TestLoad_LanguageFallbackFromFlag(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	yaml := "exclude:\n  - dist\n" // no language field
	if err := os.WriteFile(filepath.Join(dir, ".depviz.yml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(dir, "js")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Language != "js" {
		t.Errorf("Language = %q, want %q (from flag fallback)", cfg.Language, "js")
	}
}

func TestDefaultFor_UnsupportedLang(t *testing.T) {
	t.Parallel()

	_, err := config.DefaultFor("python", t.TempDir())
	if err == nil {
		t.Fatal("expected error for unsupported language")
	}
}

func TestDefaultFor_GoNoGoMod(t *testing.T) {
	t.Parallel()

	dir := t.TempDir() // no go.mod
	_, err := config.DefaultFor("go", dir)
	if err == nil {
		t.Fatal("expected error when go.mod is missing")
	}
}

func TestDefaultFor_GoMalformedGoMod(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// go.mod with no module directive
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("go 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := config.DefaultFor("go", dir)
	if err == nil {
		t.Fatal("expected error for go.mod without module directive")
	}
}
