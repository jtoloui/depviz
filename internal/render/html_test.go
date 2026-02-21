package render_test

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/jtoloui/depviz/internal/classify"
	"github.com/jtoloui/depviz/internal/config"
	"github.com/jtoloui/depviz/internal/render"
	"github.com/jtoloui/depviz/internal/scanner"
)

func newClassifier(t *testing.T, lang string) *classify.Classifier {
	t.Helper()
	cl, err := classify.New(&config.Config{
		Language: lang,
		Classify: config.ClassifyRules{
			Internal: []string{`^\.\.?/.*`},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return cl
}

func TestHTML_ContainsStructure(t *testing.T) {
	t.Parallel()

	results := []scanner.FileImports{
		{
			File:    "src/app.ts",
			Lang:    "js",
			Imports: []string{"react", "./utils"},
			Details: []scanner.ImportDetail{
				{Path: "react", Kind: scanner.ImportDefault, Line: 1, Snippet: "import React from 'react';"},
				{Path: "./utils", Kind: scanner.ImportNamed, Names: []string{"helper"}, Line: 2, Snippet: "import { helper } from './utils';"},
			},
			Exports: []scanner.ExportDetail{
				{Name: "App", Kind: scanner.ExportDefault, Line: 5},
			},
			Lines: 10,
		},
	}

	var buf bytes.Buffer
	if err := render.HTML(&buf, "/project", results, newClassifier(t, "js")); err != nil {
		t.Fatalf("HTML: %v", err)
	}
	html := buf.String()

	// Structural checks
	checks := []struct {
		name    string
		pattern string
	}{
		{"doctype", "<!doctype html>"},
		{"css inlined", "<style>"},
		{"js inlined", "const data ="},
		{"root path", `const root = "/project"`},
		{"devicon cdn", "devicon.min.css"},
		{"sidebar", `class="sidebar"`},
		{"grid", `class="grid"`},
		{"theme select", "theme-select"},
	}
	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			if !strings.Contains(html, c.pattern) {
				t.Errorf("output missing %q", c.pattern)
			}
		})
	}
}

func TestHTML_JSONDataIntegrity(t *testing.T) {
	t.Parallel()

	results := []scanner.FileImports{
		{
			File:    "main.go",
			Lang:    "go",
			Imports: []string{"fmt", "github.com/example/pkg"},
			Details: []scanner.ImportDetail{
				{Path: "fmt", Kind: scanner.ImportNamed, Line: 3},
				{Path: "github.com/example/pkg", Kind: scanner.ImportNamed, Line: 4},
			},
			Exports: []scanner.ExportDetail{
				{Name: "Run", Kind: scanner.ExportFunction, Line: 10},
				{Name: "internal", Kind: scanner.ExportFunction, Private: true, Line: 15},
			},
			Lines: 20,
		},
	}

	var buf bytes.Buffer
	if err := render.HTML(&buf, "/go-project", results, newClassifier(t, "go")); err != nil {
		t.Fatalf("HTML: %v", err)
	}

	// Extract JSON from "const data = [...];"
	re := regexp.MustCompile(`const data = (\[.*?\]);\s*const root`)
	m := re.FindSubmatch(buf.Bytes())
	if m == nil {
		t.Fatal("could not extract JSON data from output")
	}

	var files []struct {
		File    string `json:"file"`
		Lines   int    `json:"lines"`
		Imports []struct {
			Name     string `json:"name"`
			Category string `json:"category"`
			Kind     string `json:"kind"`
			Line     int    `json:"line"`
		} `json:"imports"`
		Exports []struct {
			Name    string `json:"name"`
			Kind    string `json:"kind"`
			Private bool   `json:"private"`
			Line    int    `json:"line"`
		} `json:"exports"`
	}
	if err := json.Unmarshal(m[1], &files); err != nil {
		t.Fatalf("JSON unmarshal: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}

	f := files[0]
	if f.File != "main.go" {
		t.Errorf("file = %q, want %q", f.File, "main.go")
	}
	if f.Lines != 20 {
		t.Errorf("lines = %d, want 20", f.Lines)
	}

	// Classification
	if f.Imports[0].Category != "stdlib" {
		t.Errorf("fmt category = %q, want stdlib", f.Imports[0].Category)
	}
	if f.Imports[1].Category != "external" {
		t.Errorf("pkg category = %q, want external", f.Imports[1].Category)
	}

	// Import details preserved
	if f.Imports[0].Line != 3 {
		t.Errorf("fmt line = %d, want 3", f.Imports[0].Line)
	}

	// Exports
	if len(f.Exports) != 2 {
		t.Fatalf("exports len = %d, want 2", len(f.Exports))
	}
	if f.Exports[0].Name != "Run" || f.Exports[0].Kind != "func" {
		t.Errorf("export[0] = {%q, %q}, want {Run, func}", f.Exports[0].Name, f.Exports[0].Kind)
	}
	if !f.Exports[1].Private {
		t.Error("export[1] should be private")
	}
}

func TestHTML_DuplicateImportPaths(t *testing.T) {
	t.Parallel()

	// Same module imported twice with different kinds (e.g. value + type)
	results := []scanner.FileImports{
		{
			File:    "app.tsx",
			Lang:    "js",
			Imports: []string{"react", "react"},
			Details: []scanner.ImportDetail{
				{Path: "react", Kind: scanner.ImportDefault, Line: 1, Snippet: "import React from 'react';"},
				{Path: "react", Kind: scanner.ImportType, Names: []string{"FC"}, Line: 2, Snippet: "import type { FC } from 'react';"},
			},
			Lines: 5,
		},
	}

	var buf bytes.Buffer
	if err := render.HTML(&buf, "/project", results, newClassifier(t, "js")); err != nil {
		t.Fatalf("HTML: %v", err)
	}

	re := regexp.MustCompile(`const data = (\[.*?\]);\s*const root`)
	m := re.FindSubmatch(buf.Bytes())
	if m == nil {
		t.Fatal("could not extract JSON data")
	}

	var files []struct {
		Imports []struct {
			Kind string `json:"kind"`
		} `json:"imports"`
	}
	if err := json.Unmarshal(m[1], &files); err != nil {
		t.Fatalf("JSON unmarshal: %v", err)
	}

	// Both imports should preserve their distinct kinds
	if files[0].Imports[0].Kind != "default" {
		t.Errorf("import[0].kind = %q, want default", files[0].Imports[0].Kind)
	}
	if files[0].Imports[1].Kind != "type" {
		t.Errorf("import[1].kind = %q, want type", files[0].Imports[1].Kind)
	}
}

func TestHTML_EmptyResults(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	if err := render.HTML(&buf, "/empty", nil, newClassifier(t, "js")); err != nil {
		t.Fatalf("HTML: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, "<!doctype html>") {
		t.Error("empty results should still produce valid HTML")
	}
	if !strings.Contains(html, "const data = null;") && !strings.Contains(html, "const data = [];") {
		// Either null or [] is acceptable for empty
		if !strings.Contains(html, "const data =") {
			t.Error("output missing data injection")
		}
	}
}

func TestHTML_DeterministicOrder(t *testing.T) {
	t.Parallel()

	cl := newClassifier(t, "go")
	results := []scanner.FileImports{
		{File: "z/last.go", Lang: "go", Imports: []string{"fmt"}},
		{File: "a/first.go", Lang: "go", Imports: []string{"os"}},
		{File: "m/middle.go", Lang: "go", Imports: []string{"net"}},
	}

	var buf1, buf2 bytes.Buffer
	if err := render.HTML(&buf1, "/tmp", results, cl); err != nil {
		t.Fatal(err)
	}
	results[0], results[2] = results[2], results[0]
	if err := render.HTML(&buf2, "/tmp", results, cl); err != nil {
		t.Fatal(err)
	}

	if buf1.String() != buf2.String() {
		t.Error("HTML output differs for same files in different input order")
	}

	out := buf1.String()
	aIdx := strings.Index(out, "a/first.go")
	mIdx := strings.Index(out, "m/middle.go")
	zIdx := strings.Index(out, "z/last.go")
	if aIdx > mIdx || mIdx > zIdx {
		t.Errorf("files not sorted: a=%d m=%d z=%d", aIdx, mIdx, zIdx)
	}
}
