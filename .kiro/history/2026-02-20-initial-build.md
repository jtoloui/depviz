# 2026-02-20 — Initial Build + Phase 1 Complete

## Session Summary

Built depviz from scratch in a single session — from project scaffolding to a prod-ready CLI with an interactive HTML UI.

## What was built

### Core
- Cobra CLI with `scan` and `serve` subcommands
- Go scanner using go/ast (ImportsOnly flag)
- JS/TS scanner using regex (import/require patterns, .js/.jsx/.ts/.tsx/.mjs)
- Scanner interface with compile-time checks
- Concurrent file scanning with fan-out worker pool (walk.go)

### Config System
- `.depviz.yml` config file support
- Per-language defaults (JS: node_modules/dist/build excluded, Go: vendor excluded + module path from go.mod)
- Classify rules: internal/private regex patterns, stdlib hardcoded per language
- Config validation inside Load — language, regex validity, non-empty excludes
- Config priority: YAML field > CLI flag > default

### Classification
- Separate classify package with pre-compiled regexes
- 4 categories: stdlib (green), internal (purple), private (blue), external (orange)
- Node.js built-in list, Go no-dot stdlib rule

### Prod-Ready
- Graceful shutdown on serve (signal.NotifyContext + server.Shutdown)
- ReadHeaderTimeout (5s) on explicit http.Server
- Auto port selection — falls back to OS-assigned if in use
- slog JSON structured logging with -v debug flag
- --version flag with ldflags injection from git describe
- Makefile: fmt → vet → test → lint → build

### Phase 1 — UI Upgrade
- Refactored render to html/template + //go:embed
- Modern dark theme with sidebar layout
- Search bar — filter by filename or import name
- Category filter toggles with counts
- VS Code file links on all filenames
- Result counter, stats panel

### Tests (21 total, all passing with -race)
- Classifier: JS (9 cases), Go (5 cases), invalid pattern
- Config: load from file, JS/Go defaults, invalid language, invalid regex, empty exclude, flag fallback
- Scanner: Go with fixtures, JS with fixtures, all extensions, exclusion dirs

### Benchmarks
- 40 files (recipe-server): ~5ms
- 500 files (synthetic): 58ms
- 2000 files (synthetic): 47ms

## Files Created
- cmd/root.go, cmd/scan.go, cmd/serve.go
- internal/scanner/scanner.go, go.go, js.go, walk.go
- internal/config/config.go, defaults.go
- internal/classify/classifier.go
- internal/render/html.go, template.html
- main.go, Makefile, README.md
- Tests: classifier_test.go, config_test.go, scanner_test.go
- .kiro/steering/product.md, tech.md, structure.md, conventions.md

## Tested Against
- ~/Developer/go-teacher/milestone-2-url-shortener (Go, 10 files)
- ~/Developer/recipe-app/recipe-server (JS/TS, 40 files with .depviz.yml)

## Next Session
- Pick a final name (check `impx`, `depmap`, etc.)
- Move to own repo (`github.com/jtoloui/<name>`)
- Set up goreleaser + GitHub Actions for releases
- Tag `v0.1.0`
- Phase 2 remaining: "who depends on this file?" (Go-side import path resolution)
- Phase 3: Circular dependency detection

## Late additions (after initial build)
- `.depviz.yml` config: added `output` and `port` fields
- Reverse lookup: click import tag → sidebar shows which files use it
- Sort dropdown: name, most imports, most depended on
- Responsive CSS: mobile sidebar with hamburger menu, single-column grid
- Default output changed to `<project>/.depviz/deps.html`
- html/template + //go:embed refactor (Phase 1 UI upgrade)
- Kiro steering files created: product.md, tech.md, structure.md, conventions.md
