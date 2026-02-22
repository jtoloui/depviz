package scanner_test

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/jtoloui/depviz/internal/config"
	"github.com/jtoloui/depviz/internal/scanner"
)

func TestGoScanner(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Create a fake Go file with imports.
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import (
	"fmt"
	"net/http"

	"github.com/example/pkg"
)

func main() {}
`)

	// Create a test file that should be skipped.
	writeFile(t, filepath.Join(dir, "main_test.go"), `package main

import "testing"

func TestFoo(t *testing.T) {}
`)

	// Create a vendor dir that should be skipped.
	writeFile(t, filepath.Join(dir, "vendor", "lib", "lib.go"), `package lib

import "fmt"

func Foo() { fmt.Println() }
`)

	cfg := &config.Config{
		Language: "go",
		Exclude:  []string{"vendor", ".git"},
	}

	s := scanner.NewGoScanner(cfg)
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("got %d files, want 1", len(results))
	}

	got := results[0]
	if got.File != "main.go" {
		t.Errorf("File = %q, want %q", got.File, "main.go")
	}

	wantImports := []string{"fmt", "net/http", "github.com/example/pkg"}
	sort.Strings(got.Imports)
	sort.Strings(wantImports)
	if !slicesEqual(got.Imports, wantImports) {
		t.Errorf("Imports = %v, want %v", got.Imports, wantImports)
	}
}

func TestJSScanner(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "index.ts"), `import express from 'express';
import { foo } from './utils';
const path = require('path');
`)

	writeFile(t, filepath.Join(dir, "empty.js"), `// no imports here
const x = 1;
`)

	// node_modules should be skipped.
	writeFile(t, filepath.Join(dir, "node_modules", "express", "index.js"), `import something from 'dep';`)

	cfg := &config.Config{
		Language: "js",
		Exclude:  []string{"node_modules", ".git"},
	}

	s := scanner.NewJSScanner(cfg)
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	// empty.js has no imports, should be excluded.
	if len(results) != 1 {
		t.Fatalf("got %d files, want 1", len(results))
	}

	got := results[0]
	if got.File != "index.ts" {
		t.Errorf("File = %q, want %q", got.File, "index.ts")
	}

	wantImports := []string{"express", "./utils", "path"}
	if !slicesEqual(got.Imports, wantImports) {
		t.Errorf("Imports = %v, want %v", got.Imports, wantImports)
	}
}

func TestTreeSitterScanner_Imports(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "app.tsx"), `import React from 'react';
import { useState, useEffect } from 'react';
import * as path from 'path';
import './styles.css';
const fs = require('fs');
const lazy = await import('./lazy');
import type { Config } from './config';
export { foo } from './foo';
export * from './bar';
`)

	cfg := &config.Config{Language: "js", Exclude: []string{".git"}}
	s := scanner.NewTreeSitterScanner(cfg)
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d files, want 1", len(results))
	}

	f := results[0]

	wantImports := []string{"react", "react", "path", "./styles.css", "fs", "./lazy", "./config", "./foo", "./bar"}
	if !slicesEqual(f.Imports, wantImports) {
		t.Errorf("Imports = %v, want %v", f.Imports, wantImports)
	}

	// Verify detail kinds
	wantKinds := []scanner.ImportKind{
		scanner.ImportDefault, scanner.ImportNamed, scanner.ImportNamespace,
		scanner.ImportSideEffect, scanner.ImportRequire, scanner.ImportDynamic,
		scanner.ImportType, scanner.ImportReExport, scanner.ImportReExportAll,
	}
	if len(f.Details) != len(wantKinds) {
		t.Fatalf("Details len = %d, want %d", len(f.Details), len(wantKinds))
	}
	for i, wk := range wantKinds {
		if f.Details[i].Kind != wk {
			t.Errorf("Details[%d].Kind = %q, want %q", i, f.Details[i].Kind, wk)
		}
	}

	// Named bindings
	if names := f.Details[1].Names; len(names) != 2 {
		t.Errorf("named import names = %v, want 2 entries", names)
	}

	// Namespace alias
	if f.Details[2].Alias != "path" {
		t.Errorf("namespace alias = %q, want %q", f.Details[2].Alias, "path")
	}

	// All details should have line numbers > 0
	for i, d := range f.Details {
		if d.Line == 0 {
			t.Errorf("Details[%d].Line = 0, want > 0", i)
		}
	}

	// Snippets present for all
	for i, d := range f.Details {
		if d.Snippet == "" {
			t.Errorf("Details[%d].Snippet empty", i)
		}
	}
}

func TestTreeSitterScanner_Exports(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "lib.ts"), `export function greet() {}
export class Widget {}
export const MAX = 100;
export type ID = string;
export interface Props { name: string; }
export default function main() {}
export { helper, util };
export { foo } from './foo';
export * from './bar';
`)

	cfg := &config.Config{Language: "js", Exclude: []string{".git"}}
	s := scanner.NewTreeSitterScanner(cfg)
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d files, want 1", len(results))
	}

	exports := results[0].Exports
	wantExports := []struct {
		name string
		kind scanner.ExportKind
	}{
		{"greet", scanner.ExportFunction},
		{"Widget", scanner.ExportClass},
		{"MAX", scanner.ExportConst},
		{"ID", scanner.ExportType},
		{"Props", scanner.ExportInterface},
		{"main", scanner.ExportDefault},
		{"helper", scanner.ExportNamed},
		{"util", scanner.ExportNamed},
		{"foo", scanner.ExportReExport},
		{"* from ./bar", scanner.ExportReExport},
	}

	if len(exports) != len(wantExports) {
		t.Fatalf("exports len = %d, want %d\ngot: %+v", len(exports), len(wantExports), exports)
	}
	for i, we := range wantExports {
		if exports[i].Name != we.name || exports[i].Kind != we.kind {
			t.Errorf("exports[%d] = {%q, %q}, want {%q, %q}", i, exports[i].Name, exports[i].Kind, we.name, we.kind)
		}
		if exports[i].Line == 0 {
			t.Errorf("exports[%d].Line = 0, want > 0", i)
		}
	}
}

func TestTreeSitterScanner_LineCount(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "five.ts"), "import a from 'a';\n\n\n\nexport const x = 1;")

	cfg := &config.Config{Language: "js", Exclude: []string{".git"}}
	s := scanner.NewTreeSitterScanner(cfg)
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if results[0].Lines != 5 {
		t.Errorf("Lines = %d, want 5", results[0].Lines)
	}
}

func TestTreeSitterScanner_AllExtensions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	exts := []string{".js", ".jsx", ".ts", ".tsx", ".mjs"}
	for _, ext := range exts {
		writeFile(t, filepath.Join(dir, "file"+ext), `import foo from 'bar';`)
	}
	writeFile(t, filepath.Join(dir, "style.css"), `@import 'reset';`)

	cfg := &config.Config{Language: "js", Exclude: []string{".git"}}
	s := scanner.NewTreeSitterScanner(cfg)
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(results) != len(exts) {
		t.Errorf("got %d files, want %d", len(results), len(exts))
	}
}

func TestTreeSitterScanner_SkipNodeModules(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "index.ts"), `import x from 'x';`)
	writeFile(t, filepath.Join(dir, "node_modules", "x", "index.js"), `import y from 'y';`)

	cfg := &config.Config{Language: "js", Exclude: []string{"node_modules"}}
	s := scanner.NewTreeSitterScanner(cfg)
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d files, want 1 (node_modules should be skipped)", len(results))
	}
}

func TestTreeSitterScanner_ExportDefaultExpression(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// export default <expression> — not a declaration, hits the hasExportDecl=false branch
	writeFile(t, filepath.Join(dir, "expr.ts"), `export default 42;
export default { key: 'value' };
`)

	cfg := &config.Config{Language: "js", Exclude: []string{".git"}}
	s := scanner.NewTreeSitterScanner(cfg)
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d files, want 1", len(results))
	}

	// Should have at least one default export
	hasDefault := false
	for _, e := range results[0].Exports {
		if e.Kind == scanner.ExportDefault {
			hasDefault = true
			break
		}
	}
	if !hasDefault {
		t.Errorf("expected default export, got: %+v", results[0].Exports)
	}
}

func TestTreeSitterScanner_SideEffectImport(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// Pure side-effect import — no clause, no bindings
	writeFile(t, filepath.Join(dir, "side.ts"), `import './polyfill';
import 'reflect-metadata';
`)

	cfg := &config.Config{Language: "js", Exclude: []string{".git"}}
	s := scanner.NewTreeSitterScanner(cfg)
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d files, want 1", len(results))
	}

	for _, d := range results[0].Details {
		if d.Kind != scanner.ImportSideEffect {
			t.Errorf("import %q kind = %q, want side-effect", d.Path, d.Kind)
		}
	}
}

func TestTreeSitterScanner_DefaultImportWithNamed(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// import React, { useState } from 'react' — hits collectNames with both identifier + named_imports
	writeFile(t, filepath.Join(dir, "combo.ts"), `import React, { useState, useEffect } from 'react';`)

	cfg := &config.Config{Language: "js", Exclude: []string{".git"}}
	s := scanner.NewTreeSitterScanner(cfg)
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	d := results[0].Details[0]
	if d.Kind != scanner.ImportDefault {
		t.Errorf("kind = %q, want default", d.Kind)
	}
	// Should have React, useState, useEffect in names
	if len(d.Names) != 3 {
		t.Errorf("names = %v, want 3 entries", d.Names)
	}
}

func TestTreeSitterScanner_VariableRequire(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// const x = require('y') — hits stmtSnippet walking up to lexical_declaration
	writeFile(t, filepath.Join(dir, "req.js"), `const express = require('express');
const { join } = require('path');
`)

	cfg := &config.Config{Language: "js", Exclude: []string{".git"}}
	s := scanner.NewTreeSitterScanner(cfg)
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	for _, d := range results[0].Details {
		if d.Kind != scanner.ImportRequire {
			t.Errorf("import %q kind = %q, want require", d.Path, d.Kind)
		}
		if d.Snippet == "" {
			t.Errorf("import %q missing snippet", d.Path)
		}
	}
}

func TestTreeSitterScanner_EmptyFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "empty.ts"), `const x = 1;`)
	writeFile(t, filepath.Join(dir, "has.ts"), `import a from 'a';`)

	cfg := &config.Config{Language: "js", Exclude: []string{".git"}}
	s := scanner.NewTreeSitterScanner(cfg)
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	// empty.ts has no imports or exports — should be excluded
	if len(results) != 1 {
		t.Errorf("got %d files, want 1", len(results))
	}
}

func TestGoScanner_ExportsAndDetails(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "lib.go"), `package lib

import (
	"fmt"
	_ "net/http/pprof"
	. "strings"
	myjson "encoding/json"
)

func Exported() {}
func unexported() {}
type MyType struct{}
type myPrivate struct{}
const MaxRetries = 3
const internal = 5
var GlobalVar = "x"
var localVar = "y"
`)

	cfg := &config.Config{Language: "go", Exclude: []string{".git"}}
	s := scanner.NewGoScanner(cfg)
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d files, want 1", len(results))
	}

	f := results[0]

	// Import details
	detailKinds := map[string]scanner.ImportKind{
		"fmt":            scanner.ImportNamed,
		"net/http/pprof": scanner.ImportBlank,
		"strings":        scanner.ImportDot,
		"encoding/json":  scanner.ImportAlias,
	}
	for _, d := range f.Details {
		want, ok := detailKinds[d.Path]
		if !ok {
			t.Errorf("unexpected detail path %q", d.Path)
			continue
		}
		if d.Kind != want {
			t.Errorf("detail %q kind = %q, want %q", d.Path, d.Kind, want)
		}
		if d.Line == 0 {
			t.Errorf("detail %q line = 0", d.Path)
		}
		if d.Snippet == "" {
			t.Errorf("detail %q snippet empty", d.Path)
		}
	}

	// Alias
	for _, d := range f.Details {
		if d.Path == "encoding/json" && d.Alias != "myjson" {
			t.Errorf("alias = %q, want %q", d.Alias, "myjson")
		}
	}

	// Exports
	wantExports := []struct {
		name    string
		kind    scanner.ExportKind
		private bool
	}{
		{"Exported", scanner.ExportFunction, false},
		{"unexported", scanner.ExportFunction, true},
		{"MyType", scanner.ExportType, false},
		{"myPrivate", scanner.ExportType, true},
		{"MaxRetries", scanner.ExportConst, false},
		{"internal", scanner.ExportConst, true},
		{"GlobalVar", scanner.ExportVar, false},
		{"localVar", scanner.ExportVar, true},
	}

	if len(f.Exports) != len(wantExports) {
		t.Fatalf("exports len = %d, want %d\ngot: %+v", len(f.Exports), len(wantExports), f.Exports)
	}
	for i, we := range wantExports {
		e := f.Exports[i]
		if e.Name != we.name || e.Kind != we.kind || e.Private != we.private {
			t.Errorf("exports[%d] = {%q, %q, private=%v}, want {%q, %q, private=%v}",
				i, e.Name, e.Kind, e.Private, we.name, we.kind, we.private)
		}
		if e.Line == 0 {
			t.Errorf("exports[%d].Line = 0", i)
		}
	}
}

func TestGoScanner_LineCount(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "three.go"), "package main\n\nimport \"fmt\"\n")

	cfg := &config.Config{Language: "go", Exclude: []string{".git"}}
	s := scanner.NewGoScanner(cfg)
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	// 3 lines + trailing newline = 4
	if results[0].Lines != 4 {
		t.Errorf("Lines = %d, want 4", results[0].Lines)
	}
}

func TestGoScanner_SkipMethods(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "svc.go"), `package svc

import "fmt"

type Svc struct{}

func (s *Svc) Handle() { fmt.Println() }
func Free() {}
`)

	cfg := &config.Config{Language: "go", Exclude: []string{".git"}}
	s := scanner.NewGoScanner(cfg)
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	// Should have Svc (type) and Free (func), but NOT Handle (method)
	var names []string
	for _, e := range results[0].Exports {
		if e.Kind == scanner.ExportFunction {
			names = append(names, e.Name)
		}
	}
	if len(names) != 1 || names[0] != "Free" {
		t.Errorf("function exports = %v, want [Free] (methods should be skipped)", names)
	}
}

func TestWalkAndParse_SkipDirs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.ts"), `import x from 'x';`)
	writeFile(t, filepath.Join(dir, "dist", "b.ts"), `import y from 'y';`)
	writeFile(t, filepath.Join(dir, ".git", "c.ts"), `import z from 'z';`)

	cfg := &config.Config{Language: "js", Exclude: []string{"dist", ".git"}}
	s := scanner.NewTreeSitterScanner(cfg)
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("got %d files, want 1 (dist and .git should be skipped)", len(results))
	}
}

func TestWalkAndParse_EmptyDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	cfg := &config.Config{Language: "js", Exclude: []string{".git"}}
	s := scanner.NewTreeSitterScanner(cfg)
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %d files, want 0", len(results))
	}
}

func TestWalkAndParse_Concurrency(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	// Create enough files to exercise multiple workers
	for i := range 50 {
		writeFile(t, filepath.Join(dir, fmt.Sprintf("f%d.ts", i)), `import a from 'a';
export const x = 1;`)
	}

	cfg := &config.Config{Language: "js", Exclude: []string{".git"}}
	s := scanner.NewTreeSitterScanner(cfg)
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(results) != 50 {
		t.Errorf("got %d files, want 50", len(results))
	}
}

func TestJSScanner_AllExtensions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	exts := []string{".js", ".jsx", ".ts", ".tsx", ".mjs"}

	for _, ext := range exts {
		writeFile(t, filepath.Join(dir, "file"+ext), `import foo from 'bar';`)
	}

	// .css should be ignored.
	writeFile(t, filepath.Join(dir, "style.css"), `@import 'reset';`)

	cfg := &config.Config{Language: "js", Exclude: []string{".git"}}
	s := scanner.NewJSScanner(cfg)
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(results) != len(exts) {
		t.Errorf("got %d files, want %d", len(results), len(exts))
	}
}

func TestMultiScanner(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "main.go"), `package main

import "fmt"

func main() { fmt.Println() }
`)

	writeFile(t, filepath.Join(dir, "app.ts"), `import express from 'express';
export const app = express();
`)

	cfg := &config.Config{
		Language: "multi",
		Exclude:  []string{"vendor", ".git", "node_modules"},
	}

	s := scanner.NewMultiScanner(cfg)
	results, err := s.Scan(dir)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("got %d files, want 2", len(results))
	}

	langs := map[string]bool{}
	for _, r := range results {
		langs[r.Lang] = true
	}
	if !langs["go"] || !langs["js"] {
		t.Errorf("expected both go and js, got %v", langs)
	}
}

func TestMultiScanner_ErrorPropagation(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Language: "multi",
		Exclude:  []string{"vendor", ".git", "node_modules"},
	}

	// Test Go scanner error (nonexistent path)
	s := scanner.NewMultiScanner(cfg)
	_, err := s.Scan("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestMultiScanner_JSErrorPropagation(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Valid Go file so Go scan succeeds
	writeFile(t, filepath.Join(dir, "main.go"), `package main

func main() {}
`)

	// Unreadable JS file so JS scan fails
	jsPath := filepath.Join(dir, "bad.ts")
	writeFile(t, jsPath, `import x from 'y';`)
	if err := os.Chmod(jsPath, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(jsPath, 0o644) })

	cfg := &config.Config{
		Language: "multi",
		Exclude:  []string{"vendor", ".git", "node_modules"},
	}

	s := scanner.NewMultiScanner(cfg)
	_, err := s.Scan(dir)
	if err == nil {
		t.Error("expected error from unreadable JS file")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
