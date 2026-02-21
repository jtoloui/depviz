# Project Structure

```
dep-visualiser/
├── cmd/
│   ├── root.go              ← Cobra root command, slog setup, -l/-v flags
│   ├── scan.go              ← depviz scan — config load, scan, render to file
│   └── serve.go             ← depviz serve — HTTP server, graceful shutdown, auto port
├── internal/
│   ├── classify/
│   │   ├── classifier.go    ← Classifier struct, pre-compiled regex, stdlib detection (Go + Node.js builtins)
│   │   └── classifier_test.go
│   ├── config/
│   │   ├── config.go        ← Config type, Load (reads .depviz.yml), validate
│   │   ├── config_test.go
│   │   └── defaults.go      ← DefaultFor(lang) — JS, Go, and multi built-in defaults
│   ├── render/
│   │   ├── html.go          ← HTML function, embeds template + CSS + JS via //go:embed
│   │   ├── html_test.go     ← Render tests: structure, JSON integrity, duplicate imports, empty results
│   │   ├── template.html    ← HTML skeleton with {{.CSS}}, {{.JS}}, {{.DataJSON}} placeholders
│   │   ├── styles.css       ← All CSS (14 themes, cards, sidebar, toolbar, tooltips, responsive)
│   │   └── app.js           ← All JS (render, search, filters, sort, icons, stats, file tree, keyboard shortcuts)
│   └── scanner/
│       ├── scanner.go       ← Scanner interface, FileImports, ImportDetail, ExportDetail types
│       ├── scanner_test.go  ← Scanner tests: Go, JS, tree-sitter, walk, concurrency, edge cases
│       ├── go.go            ← GoScanner — go/ast for imports (with aliases/blank/dot) + exported declarations + line counts
│       ├── js.go            ← JSScanner — regex-based import/require matching (legacy, kept for reference)
│       ├── treesitter.go    ← TreeSitterScanner — AST-based JS/TS parsing via pre-compiled tree-sitter queries + line counts
│       ├── multi.go         ← MultiScanner — delegates to GoScanner + TreeSitterScanner, merges results
│       └── walk.go          ← walkAndParse — concurrent fan-out worker pool (walker in WaitGroup, errors via channel)
├── e2e_test.go              ← End-to-end tests: full pipeline for Go and JS fixture projects
├── main.go                  ← Entry point, version injection via SetVersion
├── Makefile                 ← tidy → fmt → vet → test → lint → build; coverage target
├── .gitignore
├── README.md
├── go.mod
└── go.sum
```

## Package Responsibilities

- `cmd` — CLI orchestration only. Loads config, creates scanner + classifier, calls render. No business logic.
- `internal/scanner` — Knows how to walk directories and extract imports + exports + line counts. Language-specific parsers behind a shared Scanner interface. Concurrent via walk.go. JS/TS uses tree-sitter for AST-based parsing; Go uses go/ast.
- `internal/classify` — Knows how to categorise an import string. Owns stdlib lists (Go: no-dot heuristic, JS: comprehensive Node.js builtins map with subpath imports) and regex matching. Depends on config for patterns.
- `internal/config` — Knows how to read .depviz.yml and provide defaults. Pure data + validation. No behaviour beyond loading.
- `internal/render` — Knows how to turn scan results into HTML. Template split into three source files (HTML/CSS/JS) for maintainability, inlined at build time via `//go:embed` for single-file output. Depends on classify for category assignment.

## Data Flow

```
CLI flags + .depviz.yml → config.Load → Config
Config → scanner.New{Go,TreeSitter,Multi}Scanner → Scanner
Config → classify.New → Classifier
Scanner.Scan(root) → []FileImports (with Details + Exports + Lines + Lang)
[]FileImports + Classifier → render.HTML (ClassifyWithLang per file) → io.Writer (single HTML file)
```

## Key Interfaces

```go
type Scanner interface {
    Scan(root string) ([]FileImports, error)
}
```

All scanners implement this. New languages = new file in scanner package + compile-time check.

## Key Types

```go
type FileImports struct {
    File    string         // relative path from project root
    Lang    string         // source language ("go" or "js") for per-file stdlib classification
    Imports []string       // module paths (for backward compat + classifier)
    Details []ImportDetail // rich import data: kind, names, alias, snippet, line
    Exports []ExportDetail // what the file exports: name, kind, private flag, line
    Lines   int            // total line count of the file
}
```
