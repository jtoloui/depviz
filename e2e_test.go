package main_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/jtoloui/depviz/internal/classify"
	"github.com/jtoloui/depviz/internal/config"
	"github.com/jtoloui/depviz/internal/render"
	"github.com/jtoloui/depviz/internal/scanner"
)

// TestE2E_GoProject creates a fixture Go project, runs the full pipeline,
// and verifies the output HTML contains correct data.
func TestE2E_GoProject(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFixture(t, dir, "go.mod", "module github.com/test/proj\n\ngo 1.25\n")
	writeFixture(t, dir, "main.go", `package main

import (
	"fmt"
	"github.com/test/proj/internal/svc"
)

func main() { fmt.Println(svc.Run()) }
`)
	writeFixture(t, dir, "internal/svc/svc.go", `package svc

import "strings"

func Run() string { return strings.ToUpper("ok") }
func helper() string { return "h" }
type Config struct{ Name string }
`)

	// Load config (reads go.mod for internal pattern)
	cfg, err := config.Load(dir, "go")
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}

	// Scan
	s := scanner.NewGoScanner(cfg)
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("got %d files, want 2", len(results))
	}

	// Classify
	cl, err := classify.New(cfg)
	if err != nil {
		t.Fatalf("classify.New: %v", err)
	}

	// Render to file
	outPath := filepath.Join(dir, ".depviz", "deps.html")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := render.HTML(f, dir, results, cl); err != nil {
		_ = f.Close()
		t.Fatalf("render.HTML: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	// Verify output exists and is non-trivial
	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatalf("output file missing: %v", err)
	}
	if info.Size() < 1000 {
		t.Errorf("output too small: %d bytes", info.Size())
	}

	// Parse and verify JSON data
	html, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}

	re := regexp.MustCompile(`const data = (\[.*?\]);\s*const root`)
	m := re.FindSubmatch(html)
	if m == nil {
		t.Fatal("could not extract JSON data from output")
	}

	var files []struct {
		File    string `json:"file"`
		Lines   int    `json:"lines"`
		Imports []struct {
			Name     string `json:"name"`
			Category string `json:"category"`
		} `json:"imports"`
		Exports []struct {
			Name    string `json:"name"`
			Private bool   `json:"private"`
		} `json:"exports"`
	}
	if err := json.Unmarshal(m[1], &files); err != nil {
		t.Fatalf("JSON unmarshal: %v", err)
	}

	// Find svc.go
	var svcFile *struct {
		File    string `json:"file"`
		Lines   int    `json:"lines"`
		Imports []struct {
			Name     string `json:"name"`
			Category string `json:"category"`
		} `json:"imports"`
		Exports []struct {
			Name    string `json:"name"`
			Private bool   `json:"private"`
		} `json:"exports"`
	}
	for i := range files {
		if strings.HasSuffix(files[i].File, "svc.go") {
			svcFile = &files[i]
			break
		}
	}
	if svcFile == nil {
		t.Fatal("svc.go not found in output")
	}

	// strings should be stdlib
	if svcFile.Imports[0].Category != "stdlib" {
		t.Errorf("strings category = %q, want stdlib", svcFile.Imports[0].Category)
	}

	// Should have Run (public), helper (private), Config (public)
	if len(svcFile.Exports) != 3 {
		t.Fatalf("svc exports = %d, want 3", len(svcFile.Exports))
	}

	// Line count should be > 0
	if svcFile.Lines == 0 {
		t.Error("svc.go lines = 0")
	}
}

// TestE2E_JSProject creates a fixture JS project and verifies the full pipeline.
func TestE2E_JSProject(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFixture(t, dir, "src/App.tsx", `import React from 'react';
import { useState } from 'react';
import { helper } from './utils';
import fs from 'fs';

export default function App() { return null; }
export const VERSION = '1.0';
`)
	writeFixture(t, dir, "src/utils.ts", `import path from 'path';

export function helper() { return path.join('a', 'b'); }
export type Config = { name: string };
`)

	cfg := &config.Config{
		Language: "js",
		Exclude:  []string{"node_modules", ".git"},
		Classify: config.ClassifyRules{
			Internal: []string{`^\.\.?/.*`},
		},
	}

	s := scanner.NewTreeSitterScanner(cfg)
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d files, want 2", len(results))
	}

	cl, err := classify.New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	outPath := filepath.Join(dir, "deps.html")
	f, err := os.Create(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := render.HTML(f, dir, results, cl); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	html, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}

	re := regexp.MustCompile(`const data = (\[.*?\]);\s*const root`)
	m := re.FindSubmatch(html)
	if m == nil {
		t.Fatal("could not extract JSON data")
	}

	var files []struct {
		File    string `json:"file"`
		Imports []struct {
			Name     string `json:"name"`
			Category string `json:"category"`
			Kind     string `json:"kind"`
		} `json:"imports"`
		Exports []struct {
			Name string `json:"name"`
			Kind string `json:"kind"`
		} `json:"exports"`
	}
	if err := json.Unmarshal(m[1], &files); err != nil {
		t.Fatal(err)
	}

	// Find App.tsx
	var appFile *struct {
		File    string `json:"file"`
		Imports []struct {
			Name     string `json:"name"`
			Category string `json:"category"`
			Kind     string `json:"kind"`
		} `json:"imports"`
		Exports []struct {
			Name string `json:"name"`
			Kind string `json:"kind"`
		} `json:"exports"`
	}
	for i := range files {
		if strings.HasSuffix(files[i].File, "App.tsx") {
			appFile = &files[i]
			break
		}
	}
	if appFile == nil {
		t.Fatal("App.tsx not found")
	}

	// Verify classifications
	categories := map[string]string{}
	for _, imp := range appFile.Imports {
		categories[imp.Name] = imp.Category
	}
	if categories["react"] != "external" {
		t.Errorf("react = %q, want external", categories["react"])
	}
	if categories["./utils"] != "internal" {
		t.Errorf("./utils = %q, want internal", categories["./utils"])
	}
	if categories["fs"] != "stdlib" {
		t.Errorf("fs = %q, want stdlib", categories["fs"])
	}

	// Verify exports
	if len(appFile.Exports) != 2 {
		t.Fatalf("App exports = %d, want 2", len(appFile.Exports))
	}

	// Verify import kinds
	kinds := map[string]string{}
	for _, imp := range appFile.Imports {
		kinds[imp.Name+":"+imp.Kind] = imp.Kind
	}
	if _, ok := kinds["react:default"]; !ok {
		t.Error("missing react default import kind")
	}
}

// TestE2E_MultiProject creates a fixture with Go + JS files and runs the full multi pipeline.
func TestE2E_MultiProject(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFixture(t, dir, "go.mod", "module github.com/test/multi\n\ngo 1.25\n")
	writeFixture(t, dir, "main.go", `package main

import (
	"fmt"
	"github.com/test/multi/internal/svc"
)

func main() { fmt.Println(svc.Run()) }
`)
	writeFixture(t, dir, "internal/svc/svc.go", `package svc

func Run() string { return "ok" }
`)
	writeFixture(t, dir, "web/app.ts", `import express from 'express';
import { helper } from './utils';

export const app = express();
`)
	writeFixture(t, dir, "web/utils.ts", `export function helper() { return 'hi'; }
`)

	cfg, err := config.Load(dir, "multi")
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}

	s := scanner.NewMultiScanner(cfg)
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(results) != 4 {
		t.Fatalf("got %d files, want 4", len(results))
	}

	langs := map[string]int{}
	for _, r := range results {
		langs[r.Lang]++
	}
	if langs["go"] != 2 || langs["js"] != 2 {
		t.Errorf("lang counts = %v, want go:2 js:2", langs)
	}

	cl, err := classify.New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	outPath := filepath.Join(dir, "deps.html")
	f, err := os.Create(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := render.HTML(f, dir, results, cl); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	html, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}

	re := regexp.MustCompile(`const data = (\[.*?\]);\s*const root`)
	m := re.FindSubmatch(html)
	if m == nil {
		t.Fatal("could not extract JSON data")
	}

	var files []struct {
		File    string `json:"file"`
		Imports []struct {
			Name     string `json:"name"`
			Category string `json:"category"`
		} `json:"imports"`
	}
	if err := json.Unmarshal(m[1], &files); err != nil {
		t.Fatal(err)
	}

	// Verify both Go and JS imports are classified correctly
	categories := map[string]string{}
	for _, f := range files {
		for _, imp := range f.Imports {
			categories[imp.Name] = imp.Category
		}
	}
	if categories["fmt"] != "stdlib" {
		t.Errorf("fmt = %q, want stdlib", categories["fmt"])
	}
	if categories["express"] != "external" {
		t.Errorf("express = %q, want external", categories["express"])
	}
	if categories["./utils"] != "internal" {
		t.Errorf("./utils = %q, want internal", categories["./utils"])
	}
}

func writeFixture(t *testing.T, base, rel, content string) {
	t.Helper()
	p := filepath.Join(base, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
