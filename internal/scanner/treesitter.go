package scanner

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	tree_sitter_typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"

	"github.com/jtoloui/depviz/internal/config"
)

var _ Scanner = (*TreeSitterScanner)(nil)

// S-expression query that captures import/export/require/dynamic-import sources.
const importQuery = `
(import_statement source: (string) @path)
(export_statement source: (string) @path)
(call_expression
  function: (identifier) @_fn
  arguments: (arguments (string) @path)
  (#eq? @_fn "require"))
(call_expression
  function: (import) @_fn
  arguments: (arguments (string) @path))
`

type TreeSitterScanner struct {
	cfg *config.Config
}

func NewTreeSitterScanner(cfg *config.Config) *TreeSitterScanner {
	return &TreeSitterScanner{cfg: cfg}
}

func (t *TreeSitterScanner) Scan(root string) ([]FileImports, error) {
	exts := map[string]bool{".js": true, ".jsx": true, ".ts": true, ".tsx": true, ".mjs": true}
	skip := toSet(t.cfg.Exclude)

	include := func(path string, info os.FileInfo) bool {
		return !info.IsDir() && exts[filepath.Ext(path)]
	}

	// Pre-compile queries per language (thread-safe for reads).
	queries := make(map[string]*tree_sitter.Query)
	for ext, lang := range map[string]*tree_sitter.Language{
		".js": languageForExt(".js"), ".jsx": languageForExt(".jsx"),
		".ts": languageForExt(".ts"), ".tsx": languageForExt(".tsx"),
		".mjs": languageForExt(".mjs"),
	} {
		q, err := tree_sitter.NewQuery(lang, importQuery)
		if err != nil {
			return nil, fmt.Errorf("query compile for %s: %w", ext, err)
		}
		defer q.Close()
		queries[ext] = q
	}

	return walkAndParse(root, skip, include, func(root, path string) (*FileImports, error) {
		return t.parseFile(root, path, queries[filepath.Ext(path)])
	})
}

func languageForExt(ext string) *tree_sitter.Language {
	switch ext {
	case ".ts":
		return tree_sitter.NewLanguage(unsafe.Pointer(tree_sitter_typescript.LanguageTypescript()))
	case ".tsx":
		return tree_sitter.NewLanguage(unsafe.Pointer(tree_sitter_typescript.LanguageTSX()))
	default:
		return tree_sitter.NewLanguage(unsafe.Pointer(tree_sitter_javascript.Language()))
	}
}

// stripQuotes removes surrounding quotes from a tree-sitter string node text.
func stripQuotes(s string) string {
	if len(s) >= 2 {
		return s[1 : len(s)-1]
	}
	return s
}

// nodeText returns the source text for a node.
func nodeText(src []byte, n *tree_sitter.Node) string {
	return string(src[n.StartByte():n.EndByte()])
}

func (t *TreeSitterScanner) parseFile(root, path string, query *tree_sitter.Query) (*FileImports, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	lang := languageForExt(filepath.Ext(path))

	parser := tree_sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(lang); err != nil {
		return nil, fmt.Errorf("set language for %s: %w", path, err)
	}

	tree := parser.Parse(src, nil)
	defer tree.Close()

	pathIdx, ok := query.CaptureIndexForName("path")
	if !ok {
		return nil, fmt.Errorf("capture @path not found in query")
	}

	cursor := tree_sitter.NewQueryCursor()
	defer cursor.Close()

	matches := cursor.Matches(query, tree.RootNode(), src)

	var imports []string
	var details []ImportDetail

	for m := matches.Next(); m != nil; m = matches.Next() {
		for _, cap := range m.Captures {
			if uint(cap.Index) != pathIdx {
				continue
			}
			modPath := stripQuotes(nodeText(src, &cap.Node))
			imports = append(imports, modPath)

			detail := extractDetail(src, &cap.Node, modPath)
			details = append(details, detail)
		}
	}

	exports := extractExports(src, tree.RootNode())

	if len(imports) == 0 && len(exports) == 0 {
		return nil, nil
	}

	rel, _ := filepath.Rel(root, path)
	return &FileImports{File: rel, Lang: "js", Imports: imports, Details: details, Exports: exports, Lines: bytes.Count(src, []byte{'\n'}) + 1}, nil
}

// extractDetail walks up from the captured string node to the statement
// and extracts kind + named bindings.
func extractDetail(src []byte, pathNode *tree_sitter.Node, modPath string) ImportDetail {
	stmt := pathNode.Parent()
	if stmt == nil {
		return ImportDetail{Path: modPath, Kind: ImportSideEffect, Line: int(pathNode.StartPosition().Row) + 1}
	}

	// require("x") or import("x") — parent is arguments, grandparent is call_expression
	if stmt.Kind() == "arguments" {
		call := stmt.Parent()
		if call != nil && call.Kind() == "call_expression" {
			s := stmtSnippet(src, call)
			line := int(call.StartPosition().Row) + 1
			fn := call.Child(0)
			if fn != nil && fn.Kind() == "import" {
				return ImportDetail{Path: modPath, Kind: ImportDynamic, Snippet: s, Line: line}
			}
			return ImportDetail{Path: modPath, Kind: ImportRequire, Snippet: s, Line: line}
		}
		return ImportDetail{Path: modPath, Kind: ImportRequire, Line: int(stmt.StartPosition().Row) + 1}
	}

	snippet := nodeText(src, stmt)
	line := int(stmt.StartPosition().Row) + 1

	switch stmt.Kind() {
	case "import_statement":
		d := extractImportStatement(src, stmt, modPath)
		d.Snippet = snippet
		d.Line = line
		return d
	case "export_statement":
		d := extractExportStatement(src, stmt, modPath)
		d.Snippet = snippet
		d.Line = line
		return d
	}

	return ImportDetail{Path: modPath, Kind: ImportSideEffect, Snippet: snippet, Line: line}
}

// stmtSnippet walks up to the nearest statement-level node for require/dynamic import.
func stmtSnippet(src []byte, node *tree_sitter.Node) string {
	for n := node; n != nil; n = n.Parent() {
		switch n.Kind() {
		case "lexical_declaration", "variable_declaration", "expression_statement":
			return nodeText(src, n)
		}
	}
	return nodeText(src, node)
}

func extractImportStatement(src []byte, stmt *tree_sitter.Node, modPath string) ImportDetail {
	isType := hasChildKind(stmt, "type")

	clause := childByKind(stmt, "import_clause")
	if clause == nil {
		// import "side-effect"
		kind := ImportSideEffect
		if isType {
			kind = ImportType
		}
		return ImportDetail{Path: modPath, Kind: kind}
	}

	// Walk import_clause children
	for i := uint(0); i < uint(clause.ChildCount()); i++ {
		child := clause.Child(i)
		switch child.Kind() {
		case "identifier":
			// default import
			kind := ImportDefault
			if isType {
				kind = ImportType
			}
			names := collectNames(src, clause)
			return ImportDetail{Path: modPath, Kind: kind, Names: names}
		case "named_imports":
			kind := ImportNamed
			if isType {
				kind = ImportType
			}
			return ImportDetail{Path: modPath, Kind: kind, Names: collectSpecifiers(src, child, "import_specifier")}
		case "namespace_import":
			kind := ImportNamespace
			if isType {
				kind = ImportType
			}
			alias := ""
			if id := childByKind(child, "identifier"); id != nil {
				alias = nodeText(src, id)
			}
			return ImportDetail{Path: modPath, Kind: kind, Alias: alias}
		}
	}

	return ImportDetail{Path: modPath, Kind: ImportSideEffect}
}

func extractExportStatement(src []byte, stmt *tree_sitter.Node, modPath string) ImportDetail {
	// export * from "x"
	if hasChildKind(stmt, "*") {
		return ImportDetail{Path: modPath, Kind: ImportReExportAll}
	}

	// export { a, b } from "x"
	if ec := childByKind(stmt, "export_clause"); ec != nil {
		return ImportDetail{Path: modPath, Kind: ImportReExport, Names: collectSpecifiers(src, ec, "export_specifier")}
	}

	return ImportDetail{Path: modPath, Kind: ImportReExport}
}

// collectNames gathers all identifier names from an import_clause that has
// both a default import and possibly named_imports (e.g. `import React, { useState } from "react"`).
func collectNames(src []byte, clause *tree_sitter.Node) []string {
	var names []string
	for i := uint(0); i < uint(clause.ChildCount()); i++ {
		child := clause.Child(i)
		switch child.Kind() {
		case "identifier":
			names = append(names, nodeText(src, child))
		case "named_imports":
			names = append(names, collectSpecifiers(src, child, "import_specifier")...)
		}
	}
	return names
}

// collectSpecifiers extracts names from import_specifier or export_specifier nodes.
func collectSpecifiers(src []byte, parent *tree_sitter.Node, specKind string) []string {
	var names []string
	for i := uint(0); i < uint(parent.ChildCount()); i++ {
		child := parent.Child(i)
		if child.Kind() == specKind {
			names = append(names, nodeText(src, child))
		}
	}
	return names
}

func childByKind(node *tree_sitter.Node, kind string) *tree_sitter.Node {
	for i := uint(0); i < uint(node.ChildCount()); i++ {
		if c := node.Child(i); c.Kind() == kind {
			return c
		}
	}
	return nil
}

func hasChildKind(node *tree_sitter.Node, kind string) bool {
	return childByKind(node, kind) != nil
}

// extractExports walks top-level export_statement nodes that are local exports
// (not re-exports, which have a source: string child).
func extractExports(src []byte, root *tree_sitter.Node) []ExportDetail {
	var exports []ExportDetail
	for i := uint(0); i < uint(root.ChildCount()); i++ {
		node := root.Child(i)
		if node.Kind() != "export_statement" {
			continue
		}
		// Re-exports (have a source string child) — also show as exports.
		if srcStr := childByKind(node, "string"); srcStr != nil {
			line := int(node.StartPosition().Row) + 1
			name := "* from " + stripQuotes(nodeText(src, srcStr))
			// export { foo, bar } from './x' — use named specifiers
			if clause := childByKind(node, "export_clause"); clause != nil {
				for k := uint(0); k < uint(clause.ChildCount()); k++ {
					spec := clause.Child(k)
					if spec.Kind() == "export_specifier" {
						if id := spec.Child(0); id != nil {
							exports = append(exports, ExportDetail{Name: nodeText(src, id), Kind: ExportReExport, Line: line})
						}
					}
				}
			} else {
				exports = append(exports, ExportDetail{Name: name, Kind: ExportReExport, Line: line})
			}
			continue
		}

		isDefault := hasChildKind(node, "default")
		line := int(node.StartPosition().Row) + 1

		for j := uint(0); j < uint(node.ChildCount()); j++ {
			child := node.Child(j)
			switch child.Kind() {
			case "function_declaration":
				name := "default"
				if id := childByKind(child, "identifier"); id != nil {
					name = nodeText(src, id)
				}
				kind := ExportFunction
				if isDefault {
					kind = ExportDefault
				}
				exports = append(exports, ExportDetail{Name: name, Kind: kind, Line: line})
			case "class_declaration":
				name := "default"
				if id := childByKind(child, "type_identifier"); id != nil {
					name = nodeText(src, id)
				}
				kind := ExportClass
				if isDefault {
					kind = ExportDefault
				}
				exports = append(exports, ExportDetail{Name: name, Kind: kind, Line: line})
			case "lexical_declaration":
				for k := uint(0); k < uint(child.ChildCount()); k++ {
					decl := child.Child(k)
					if decl.Kind() == "variable_declarator" {
						if id := childByKind(decl, "identifier"); id != nil {
							exports = append(exports, ExportDetail{Name: nodeText(src, id), Kind: ExportConst, Line: line})
						}
					}
				}
			case "type_alias_declaration":
				if id := childByKind(child, "type_identifier"); id != nil {
					exports = append(exports, ExportDetail{Name: nodeText(src, id), Kind: ExportType, Line: line})
				}
			case "interface_declaration":
				if id := childByKind(child, "type_identifier"); id != nil {
					exports = append(exports, ExportDetail{Name: nodeText(src, id), Kind: ExportInterface, Line: line})
				}
			case "export_clause":
				// export { foo, bar }
				for k := uint(0); k < uint(child.ChildCount()); k++ {
					spec := child.Child(k)
					if spec.Kind() == "export_specifier" {
						if id := spec.Child(0); id != nil {
							exports = append(exports, ExportDetail{Name: nodeText(src, id), Kind: ExportNamed, Line: line})
						}
					}
				}
			}
		}

		// export default <expression> (not a declaration)
		if isDefault && len(exports) == 0 || (isDefault && !hasExportDecl(node)) {
			// Check we haven't already added from a declaration child
			found := false
			for _, e := range exports {
				if e.Kind == ExportDefault {
					found = true
					break
				}
			}
			if !found {
				exports = append(exports, ExportDetail{Name: "default", Kind: ExportDefault, Line: line})
			}
		}
	}
	return exports
}

func hasExportDecl(node *tree_sitter.Node) bool {
	for i := uint(0); i < uint(node.ChildCount()); i++ {
		switch node.Child(i).Kind() {
		case "function_declaration", "class_declaration", "lexical_declaration":
			return true
		}
	}
	return false
}
