package cli_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/jtoloui/depviz/internal/classify"
	"github.com/jtoloui/depviz/internal/cli"
	"github.com/jtoloui/depviz/internal/config"
	"github.com/jtoloui/depviz/internal/scanner"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	fn()

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestStats(t *testing.T) {
	cfg := &config.Config{
		Language: "go",
		Classify: config.ClassifyRules{
			Internal: []string{`^github\.com/jtoloui/depviz`},
		},
	}
	cl, err := classify.New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	results := []scanner.FileImports{
		{
			File:    "cmd/root.go",
			Lang:    "go",
			Imports: []string{"fmt", "os", "github.com/jtoloui/depviz/internal/config", "github.com/spf13/cobra"},
			Exports: []scanner.ExportDetail{{Name: "Execute", Kind: "func"}},
			Lines:   50,
		},
		{
			File:    "cmd/scan.go",
			Lang:    "go",
			Imports: []string{"fmt", "path/filepath", "github.com/jtoloui/depviz/internal/config", "github.com/jtoloui/depviz/internal/scanner", "github.com/jtoloui/depviz/internal/render", "github.com/jtoloui/depviz/internal/classify", "github.com/spf13/cobra", "os", "log/slog"},
			Exports: []scanner.ExportDetail{{Name: "ScanCmd", Kind: "var"}},
			Lines:   90,
		},
	}

	out := captureStdout(t, func() { cli.Stats(results, cl) })

	tests := []struct {
		name string
		want string
	}{
		{"header", "depviz stats"},
		{"files", "2"},
		{"lines", "140"},
		{"imports count", "13"},
		{"exports count", "2"},
		{"language", "go"},
		{"stdlib category", "stdlib"},
		{"internal category", "internal"},
		{"external category", "external"},
		{"top import fmt", "fmt"},
		{"top import cobra", "cobra"},
		{"coupling header", "Coupling Hotspots"},
		{"hotspot file", "cmd/scan.go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if !strings.Contains(out, tt.want) {
				t.Errorf("output missing %q", tt.want)
			}
		})
	}
}

func TestStatsEmpty(t *testing.T) {
	cfg := &config.Config{Language: "go"}
	cl, err := classify.New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() { cli.Stats(nil, cl) })

	if !strings.Contains(out, "Files") {
		t.Error("expected Files label in empty stats")
	}
	if strings.Contains(out, "Coupling Hotspots") {
		t.Error("should not show hotspots for empty input")
	}
}

func TestStatsMultiLang(t *testing.T) {
	cfg := &config.Config{
		Language: "multi",
		Classify: config.ClassifyRules{
			Internal: []string{`^\.\.?/`},
		},
	}
	cl, err := classify.New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	results := []scanner.FileImports{
		{File: "main.go", Lang: "go", Imports: []string{"fmt"}, Lines: 10},
		{File: "app.ts", Lang: "js", Imports: []string{"./utils", "express"}, Lines: 20},
	}

	out := captureStdout(t, func() { cli.Stats(results, cl) })

	for _, want := range []string{"go", "js", "stdlib", "internal", "external"} {
		if !strings.Contains(out, want) {
			t.Errorf("multi-lang output missing %q", want)
		}
	}
}

func TestStatsNoImports(t *testing.T) {
	cfg := &config.Config{Language: "go"}
	cl, err := classify.New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	results := []scanner.FileImports{
		{File: "empty.go", Lang: "go", Lines: 5},
	}

	out := captureStdout(t, func() { cli.Stats(results, cl) })

	if !strings.Contains(out, "Imports") {
		t.Error("expected Imports label")
	}
	if !strings.Contains(out, "Avg/file") {
		t.Error("expected Avg/file label")
	}
}

func TestBanner(t *testing.T) {
	out := captureStdout(t, func() { cli.Banner() })
	if len(out) == 0 {
		t.Error("banner produced no output")
	}
}

func TestScanResult(t *testing.T) {
	results := []scanner.FileImports{
		{
			File:    "main.go",
			Imports: []string{"fmt", "os"},
			Exports: []scanner.ExportDetail{{Name: "Main", Kind: "func"}},
			Lines:   30,
		},
	}

	out := captureStdout(t, func() { cli.ScanResult(results, "/tmp/deps.html") })

	for _, want := range []string{"Scan complete", "Files", "Imports", "Exports", "Lines", "/tmp/deps.html"} {
		if !strings.Contains(out, want) {
			t.Errorf("ScanResult missing %q", want)
		}
	}
}

func TestServeResult(t *testing.T) {
	results := []scanner.FileImports{
		{File: "main.go"},
		{File: "util.go"},
	}

	out := captureStdout(t, func() { cli.ServeResult(results, 8080) })

	for _, want := range []string{"Scan complete", "2 files", "8080"} {
		if !strings.Contains(out, want) {
			t.Errorf("ServeResult missing %q", want)
		}
	}
}

func TestInitResult(t *testing.T) {
	out := captureStdout(t, func() { cli.InitResult("go", "/tmp/.depviz.yml") })

	for _, want := range []string{"Config created", "go", "/tmp/.depviz.yml"} {
		if !strings.Contains(out, want) {
			t.Errorf("InitResult missing %q", want)
		}
	}
}
