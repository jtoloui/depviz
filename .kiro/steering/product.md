# Product Overview

depviz is a CLI tool that scans Go and JavaScript/TypeScript projects for import dependencies and renders an interactive HTML visualisation.

## Purpose

Help developers understand and explore the dependency landscape of their codebase — what each file imports and exports, how files relate to each other, and where external/internal/private boundaries are.

## Target Users

- Developers onboarding to a new codebase
- Tech leads auditing dependency health
- Teams reviewing architecture and coupling

## Current Features (Phase 1 ✅)

- Scan Go projects using go/ast (fast, full AST)
- Scan JS/TS projects using tree-sitter (AST-based, catches all import styles)
- 4-colour classification: stdlib, internal, private, external
- Interactive HTML output with search, category filters, VS Code file links
- `.depviz.yml` config for custom excludes, classification rules, and port
- Live server mode with `depviz serve`
- Concurrent scanning with worker pool
- Graceful shutdown, slog logging, version injection

## Current Features (Phase 2 ✅)

- Reverse index: click any import tag → "N files use this" link in code panel triggers sidebar list
- Import count sorting: most imports, most depended-on, alphabetical
- Sidebar panel showing reverse dependencies with VS Code links

## Current Features (Phase 5 ✅)

- Rich import data: kind (default, named, namespace, side-effect, require, dynamic, re-export, type), named bindings, aliases
- Hover tooltips on import tags showing kind + `{ named, bindings }`
- Code preview: floating panel appears near clicked import tag with syntax-highlighted snippet
- VS Code links with line numbers (`vscode://file/path:line`)
- Export capture: what each file exports (functions, classes, consts, vars, types, interfaces, re-exports)
- Go: both exported (public) and unexported (private) symbols with visual distinction
- JS/TS: export default, named exports, export clauses, re-exports shown in exports section
- Outlined export tags with kind labels, clickable to open VS Code at line
- Export line number tooltips on hover
- View toggle: button group for imports only / exports only / both (default: both)
- Labelled card sections (Imports / Exports) with counts
- Empty state messages when no imports/exports

## Current Features (UI Polish ✅)

- Dark/light theme toggle with localStorage persistence
- Devicon file type icons (React, TypeScript, Go, Vite, Tailwind, Jest, etc.) with filename-pattern priority
- Collapsible cards with chevron indicator, state persisted across re-renders
- Keyboard shortcuts: Esc closes panels, / focuses search
- Copy button on code preview snippet
- URL hash state: search, filters, view mode, sort persist in URL — shareable links
- Stats dashboard: total files, unique imports, total exports, avg imports/file, total lines
- GitHub-style language breakdown bar with percentages
- Category breakdown bar with percentages
- Top 5 most imported packages
- Coupling hotspots: files with 10+ imports
- Tooltips on stats section labels explaining each metric
- Configurable base font size via CSS variable
- Template split into separate CSS/JS/HTML source files for maintainability (single-file output preserved)

## What devs use it for today

- Onboarding — "show me the dependency landscape of this codebase"
- Spotting god files — which files import 20+ things
- Finding unused/orphan packages — imports nobody references
- Auditing external deps — "what third-party stuff are we pulling in?"
- Understanding file APIs — what does each file export?
- Go architecture review — public vs private symbols per file
- Codebase health — language breakdown, import density, coupling hotspots

---

## Roadmap

### High Value

**Phase 3 — Circular Dependency Detection**
- DFS cycle detection on internal imports
- Highlight circular deps in UI with warning badge
- List all cycles in a dedicated panel

**Phase 4 — Dependency Graph Visualisation**
- Interactive node + edge graph (D3.js or similar)
- File-to-file connections based on internal imports
- Highlight circular dependency cycles in the graph
- Zoom, pan, click-to-focus on a file and its connections
- Toggle between card view and graph view

### Medium Value

**Phase 6 — CI & Export**
- `depviz scan --format json` machine-readable output
- `depviz scan --fail-on-circular` exit code 1 if cycles found (CI gate)
- Diff mode: `depviz diff --base main` show dependency changes vs a git ref

**Phase 7 — Monorepo Support**
- Scan multiple packages, show cross-package dependencies
- Workspace-aware config (root config + per-package overrides)
- File glob ignore patterns in config (e.g. `**/*.generated.ts`)

### Nice to Have

**Phase 8 — Package Awareness**
- Read `package.json` / `go.mod` for version numbers alongside external deps
- Flag outdated or deprecated dependencies
- Treemap view — size-based visualisation of dependency weight

**Phase 9 — Usage Analysis**
- Track what's actually used from each import (e.g. `path.join`, `path.resolve`)
- Go: walk full AST for SelectorExpr nodes
- JS: tree-sitter query for member expressions
- Unused import detection — imported but never referenced

**Backlog**
- Export snippet preview — show the declaration source in the code panel when clicking an export chip
- Export "find references" — click an export to see which files import it (requires resolved-path cross-referencing)
- Sticky card headers when scrolling long cards
- Minimap/treemap view for visual file size overview
- Dependency depth indicator — hops from leaf to root
- Diff highlighting between successive scans
- Breadcrumb trail in code panel for navigation history
- Batch select — shift-click multiple imports to find intersection
- Move to own repo (`github.com/jtoloui/depviz`), set up goreleaser + GitHub Actions for releases
- Homebrew tap for macOS distribution
- Benchmarks — `testing.B` for scanner performance at 500/2000/5000 files, track regressions
- Custom colour themes in config
- Export as SVG/PNG for docs
- Watch mode — `depviz serve --watch` auto-refresh on file changes
- Plugin system for custom scanners (Python, Rust, etc.)
- `--offline` flag to use emoji icons instead of Devicon CDN

## Performance Benchmarks

| Files | JS/TS (tree-sitter) | Go (go/ast) |
|-------|-------------------|-------------|
| 500   | ~91ms             | ~25ms       |
| 2,000 | ~105ms            | ~82ms       |
| 5,000 | ~203ms            | ~195ms      |

Previous (before query caching): 500→430ms, 2000→1.6s, 5000→3.9s.
