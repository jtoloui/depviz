package classify_test

import (
	"testing"

	"github.com/jtoloui/depviz/internal/classify"
	"github.com/jtoloui/depviz/internal/config"
)

func TestClassify_JS(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Language: "js",
		Classify: config.ClassifyRules{
			Internal: []string{`^\.\.?/.*`, `^@/.*`},
			Private:  []string{`^@jtoloui/.*`},
		},
	}

	cl, err := classify.New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	tests := []struct {
		name string
		imp  string
		want config.Category
	}{
		{"relative dot", "./utils", config.Internal},
		{"relative dotdot", "../config/db", config.Internal},
		{"alias @/", "@/src/helpers", config.Internal},
		{"private scope", "@jtoloui/ui-lib", config.Private},
		{"node builtin", "fs", config.Stdlib},
		{"node builtin path", "path", config.Stdlib},
		{"node: prefix", "node:crypto", config.Stdlib},
		{"npm package", "express", config.External},
		{"scoped npm", "@aws-sdk/client-s3", config.External},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := cl.Classify(tt.imp)
			if got != tt.want {
				t.Errorf("Classify(%q) = %q, want %q", tt.imp, got, tt.want)
			}
		})
	}
}

func TestClassify_Go(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Language: "go",
		Classify: config.ClassifyRules{
			Internal: []string{`^github\.com\/jtoloui\/depviz\/.*`},
			Private:  []string{`^github\.com\/jtoloui\/.*`},
		},
	}

	cl, err := classify.New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	tests := []struct {
		name string
		imp  string
		want config.Category
	}{
		{"stdlib fmt", "fmt", config.Stdlib},
		{"stdlib net/http", "net/http", config.Stdlib},
		{"internal package", "github.com/jtoloui/depviz/internal/config", config.Internal},
		{"private org", "github.com/jtoloui/other-repo", config.Private},
		{"external", "github.com/spf13/cobra", config.External},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := cl.Classify(tt.imp)
			if got != tt.want {
				t.Errorf("Classify(%q) = %q, want %q", tt.imp, got, tt.want)
			}
		})
	}
}

func TestClassify_JS_SubpathBuiltins(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Language: "js",
		Classify: config.ClassifyRules{
			Internal: []string{`^\.\.?/.*`},
		},
	}

	cl, err := classify.New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	builtins := []string{
		"fs/promises", "stream/web", "stream/consumers", "stream/promises",
		"timers/promises", "dns/promises", "readline/promises",
		"path/posix", "path/win32", "util/types", "assert/strict",
		"node:fs/promises", "node:stream/web", "node:timers/promises",
		"async_hooks", "diagnostics_channel", "inspector", "trace_events",
		"wasi", "test", "module", "repl", "console",
	}

	for _, imp := range builtins {
		t.Run(imp, func(t *testing.T) {
			t.Parallel()
			got := cl.Classify(imp)
			if got != config.Stdlib {
				t.Errorf("Classify(%q) = %q, want stdlib", imp, got)
			}
		})
	}
}

func TestNew_InvalidPrivatePattern(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Language: "js",
		Classify: config.ClassifyRules{
			Private: []string{`[invalid`},
		},
	}

	_, err := classify.New(cfg)
	if err == nil {
		t.Fatal("expected error for invalid private regex, got nil")
	}
}

func TestClassify_UnsupportedLang(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Language: "rust",
		Classify: config.ClassifyRules{},
	}

	cl, err := classify.New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Unsupported lang — isStdlib returns false, so everything falls to external
	got := cl.Classify("std::io")
	if got != config.External {
		t.Errorf("Classify(%q) = %q, want external", "std::io", got)
	}
}

func TestNew_InvalidPattern(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Language: "js",
		Classify: config.ClassifyRules{
			Internal: []string{`[invalid`},
		},
	}

	_, err := classify.New(cfg)
	if err == nil {
		t.Fatal("expected error for invalid regex, got nil")
	}
}
