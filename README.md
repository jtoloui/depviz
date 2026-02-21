<h1 align="center">depviz</h1>

<p align="center">
  <strong>Visualise Go and JS/TS project dependencies as an interactive HTML map</strong>
</p>

<p align="center">
  <a href="https://github.com/jtoloui/depviz/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/jtoloui/depviz/ci.yml?branch=main&style=flat-square&label=CI" alt="CI" /></a>
  <a href="https://codecov.io/gh/jtoloui/depviz"><img src="https://img.shields.io/codecov/c/github/jtoloui/depviz?style=flat-square" alt="Coverage" /></a>
  <a href="https://github.com/jtoloui/depviz/releases/latest"><img src="https://img.shields.io/github/v/release/jtoloui/depviz?style=flat-square" alt="Release" /></a>
  <img src="https://img.shields.io/badge/go-1.25-00ADD8?style=flat-square&logo=go" alt="Go 1.25" />
  <img src="https://img.shields.io/badge/license-MIT-blue?style=flat-square" alt="MIT License" />
</p>

---

## What is depviz?

depviz is a CLI tool that scans your Go or JavaScript/TypeScript project, extracts all import dependencies and exports, and renders them as a colour-coded HTML visualisation with clickable VS Code file links.

### Features

- 🔍 **Go scanner** — uses `go/ast` to parse imports and exported declarations (fast, full AST)
- 📦 **JS/TS scanner** — tree-sitter AST parser catches all import styles: `import`, `require`, dynamic `import()`, re-exports, type-only imports
- 🌐 **Multi-language** — `depviz scan -l multi` scans Go + JS/TS in a single pass for mixed-language repos
- 🎨 **4-colour classification** — stdlib (green), internal (purple), private/org (blue), external (orange)
- 📋 **Rich import details** — hover any import to see kind (default/named/namespace/etc.) and named bindings
- 📤 **Export capture** — see what each file exports: functions, classes, consts, types, interfaces
- 🔐 **Public/private** — Go files show both exported and unexported symbols with visual distinction
- 💻 **Code preview** — click an import tag to see the actual import statement with syntax highlighting
- 🔗 **VS Code links** — click any filename or import to open it in your editor at the exact line
- ⚡ **Concurrent scanning** — fan-out worker pool scales to large monorepos
- 🔎 **Search & filter** — search by filename, import, or export; toggle categories on/off
- 🔄 **Reverse lookup** — click any import tag to see which files use it
- 📊 **Sorting** — sort by name, most imports, most depended on
- 👁️ **View toggle** — switch between imports only, exports only, or both
- 📄 **Config file** — `.depviz.yml` for custom excludes, classification rules, and port
- 🌐 **Live server** — `depviz serve` hosts the visualisation with graceful shutdown
- 📱 **Responsive** — works on mobile with collapsible sidebar
- 🎭 **14 themes** — Dark, Light, Solarized, Catppuccin, Nord, Dracula, Gruvbox, Flat UI, Lavender, Midnight, Slate, Sand, Melo, High Contrast — persisted in localStorage
- 🏷️ **File type icons** — Devicon icons for React, TypeScript, Go, Vite, Tailwind, Jest, etc.
- 🌳 **File tree** — collapsible directory tree in sidebar, click to scroll to card
- 📉 **Stats dashboard** — total files, imports, exports, lines, language breakdown, coupling hotspots
- ⌨️ **Keyboard shortcuts** — Esc closes panels, / focuses search
- 🔗 **Shareable URLs** — search, filters, view mode, sort, reverse lookup persist in URL hash
- ◈ **Favicon** — inline SVG favicon, no external files needed

---

## Installation

### Homebrew (macOS)

```bash
brew install jtoloui/tap/depviz
```

### Download binary

Grab the latest release from [GitHub Releases](https://github.com/jtoloui/depviz/releases/latest) — pre-built binaries for macOS, Linux, and Windows (amd64/arm64).

### From source

```bash
git clone https://github.com/jtoloui/depviz.git
cd depviz
make
```

The binary is built to `bin/depviz`.

---

## Usage

### `depviz scan`

Scan a project and generate a static HTML file.

```bash
depviz scan ./my-project
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--lang` | `-l` | `go` | Language: `go`, `js`, or `multi` |
| `--output` | `-o` | `<project>/.depviz/deps.html` | Output file path |
| `--verbose` | `-v` | `false` | Enable debug logging |

#### Examples

```bash
# Scan a Go project
depviz scan ./my-go-api

# Scan a JS/TS project
depviz scan -l js ./my-react-app

# Scan a mixed Go + JS/TS project
depviz scan -l multi ./my-fullstack-app

# Custom output path
depviz scan -o visualisation.html ./my-project

# With debug logging
depviz scan -v ./my-project
```

### `depviz serve`

Scan a project and serve the visualisation in the browser.

```bash
depviz serve ./my-project
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--lang` | `-l` | `go` | Language: `go`, `js`, or `multi` |
| `--port` | `-p` | `3000` | Port to serve on |
| `--verbose` | `-v` | `false` | Enable debug logging |

If the port is in use, depviz automatically picks a free one.

#### Examples

```bash
# Serve a Go project on default port
depviz serve ./my-go-api

# Serve a JS project on a custom port
depviz serve -l js -p 8080 ./my-react-app
```

### `depviz --version`

```bash
depviz --version
# depviz version v1.0.0
```

---

## Configuration

Create a `.depviz.yml` in your project root to customise behaviour. If no config file exists, depviz uses sensible defaults for the language.

```yaml
# .depviz.yml
language: js
port: 4000
output: docs/deps.html
exclude:
  - node_modules
  - .git
  - dist
  - out
  - coverage
  - "*.d.ts"
classify:
  internal:
    - "^\\.\\./.*"     # relative imports
    - "^@/.*"          # alias imports
  private:
    - "^@jtoloui/.*"   # your org packages
```

### Fields

| Field | Type | Description |
|-------|------|-------------|
| `language` | `string` | `go`, `js`, or `multi` — overrides the `-l` flag |
| `port` | `int` | Port for `depviz serve` — overrides the `-p` flag |
| `output` | `string` | Output file path for `depviz scan` — overrides the `-o` flag |
| `exclude` | `[]string` | Directory/file names to skip during scanning |
| `classify.internal` | `[]string` | Regex patterns for internal/relative imports |
| `classify.private` | `[]string` | Regex patterns for your org/private packages |

Anything not matched by `internal` or `private` patterns is classified as **external** (or **stdlib** if it's a known built-in).

### Defaults

When no `.depviz.yml` exists:

- Output: `<project>/.depviz/deps.html` (add `.depviz/` to your `.gitignore`)

**Go:**
- Excludes: `vendor`, `.git`
- Internal: your module path (read from `go.mod`)
- Stdlib: any import without a dot (`fmt`, `net/http`)

**JS/TS:**
- Excludes: `node_modules`, `.git`, `dist`, `build`, `.next`, `coverage`
- Internal: `./` and `../` relative imports
- Stdlib: Node.js built-ins (`fs`, `path`, `crypto`, etc.)

---

## Colour Legend

| Colour | Category | Examples |
|--------|----------|----------|
| 🟢 Green | stdlib / built-in | `fmt`, `net/http`, `fs`, `path` |
| 🟣 Purple | internal / relative | `./utils`, `../config`, `@/src/helpers` |
| 🔵 Blue | private / org | `@jtoloui/ui-lib`, `github.com/jtoloui/auth` |
| 🟠 Orange | external | `express`, `@aws-sdk/client-s3`, `github.com/spf13/cobra` |

---

## Project Structure

```
dep-visualiser/
├── cmd/
│   ├── root.go              ← Cobra root command, slog setup
│   ├── scan.go              ← depviz scan
│   └── serve.go             ← depviz serve (graceful shutdown)
├── internal/
│   ├── classify/
│   │   └── classifier.go    ← Import classification engine
│   ├── config/
│   │   ├── config.go        ← YAML config loading + validation
│   │   └── defaults.go      ← Per-language default configs
│   ├── render/
│   │   ├── html.go          ← HTML generation (embeds CSS/JS/template)
│   │   ├── template.html    ← HTML skeleton with placeholders
│   │   ├── styles.css       ← All CSS (themes, cards, sidebar, responsive)
│   │   └── app.js           ← All JS (render, search, filters, stats, icons)
│   └── scanner/
│       ├── scanner.go       ← Scanner interface + types
│       ├── go.go            ← Go scanner (go/ast)
│       ├── js.go            ← JS/TS scanner (regex, legacy)
│       ├── treesitter.go    ← JS/TS scanner (tree-sitter AST)
│       ├── multi.go         ← Multi-language scanner (Go + JS/TS)
│       └── walk.go          ← Concurrent file walker
├── e2e_test.go              ← End-to-end pipeline tests
├── main.go
├── Makefile
└── go.mod
```

---

## Development

```bash
# Full build chain: tidy → fmt → vet → test → lint → build
make

# Run tests with race detector
make test

# Generate coverage report (coverage.out + coverage.html)
make coverage

# Just build
make build

# Clean
make clean
```
