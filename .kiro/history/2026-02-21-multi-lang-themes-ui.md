# 2026-02-21 — Multi-Language, Themes & UI

## Multi-Language Scanning
- Added `language: multi` mode (`depviz scan -l multi`) — runs GoScanner + TreeSitterScanner in a single pass
- New `MultiScanner` in `scanner/multi.go` delegates to both scanners and merges results
- `FileImports.Lang` field tags each file with its source language (`"go"` or `"js"`)
- `Classifier.ClassifyWithLang(imp, lang)` uses per-file language for correct stdlib detection in mixed repos
- `defaultMulti()` in config merges Go + JS excludes and classify rules
- `render.HTML` uses `ClassifyWithLang` so Go stdlib (no-dot heuristic) and JS stdlib (Node builtins) work correctly side by side

## Theme System
- Replaced dark/light toggle button with a theme selector dropdown (14 themes)
- Themes: Dark, Light, Solarized (dark/light), Catppuccin (Mocha/Latte), Nord, Dracula, Gruvbox (dark/light), Flat (dark/light), Lavender, Midnight, Slate, Sand, Melo, High Contrast
- Each theme is a CSS custom property block (`[data-theme="..."]`)
- Selection persisted in localStorage (`depviz-theme` key)

## File Tree
- Collapsible directory tree in sidebar between search and stats
- Built from file paths in data, shows directory hierarchy with file counts
- Click directory to collapse/expand, click file to scroll to its card with highlight animation
- Uses same Devicon file icons as the cards

## UI Layout
- Moved Categories, View, Sort controls from sidebar to sticky toolbar in main content area
- Toolbar groups have inline labels (Categories, View, Sort) for clarity
- Sidebar now contains: header, search, file tree, reverse panel, stats

## Other
- Inline SVG favicon using ◈ diamond (data URI, no external files)
- Reverse lookup state (`selectedImport`) persisted in URL hash (`#rev=...`) for shareable links
- Fixed `readHash()` ordering — moved after all variable declarations to prevent ReferenceError
- Fixed `parser.SetLanguage()` unchecked error return (errcheck lint)
- Fixed test assertions for lowercase `<!doctype html>` (formatter convention)

## Files Changed
- `internal/scanner/multi.go` (new)
- `internal/scanner/scanner.go` — added `Lang` field to `FileImports`
- `internal/scanner/go.go` — tags `Lang: "go"`
- `internal/scanner/treesitter.go` — tags `Lang: "js"`, fixed errcheck
- `internal/classify/classifier.go` — added `ClassifyWithLang`, refactored `isStdlib` to standalone `isStdlibFor`
- `internal/config/config.go` — added `"multi"` to `supportedLangs`
- `internal/config/defaults.go` — added `defaultMulti()`
- `internal/render/html.go` — uses `ClassifyWithLang(imp, r.Lang)`
- `internal/render/template.html` — favicon, theme dropdown, file tree container, sticky toolbar
- `internal/render/styles.css` — 14 theme blocks, toolbar styles, file tree styles
- `internal/render/app.js` — file tree, theme selector, reverse hash state, readHash reorder
- `internal/render/html_test.go` — updated for Lang field, doctype case, theme-select
- `cmd/scan.go` — added `"multi"` case to `getScanner`

## CLI Polish (2026-02-21)

### Added
- ASCII art banner (go-figure, purple "small" font) shown on scan/serve
- Coloured terminal output: `cli.ScanResult`, `cli.ServeResult`, `cli.InitResult`
- `depviz init` — interactive config generator using charmbracelet/huh forms
- `depviz stats` — terminal stats dashboard: summary, language bars, category bars, top 5 imports, coupling hotspots (8+ imports)
- SilenceUsage + SilenceErrors on root command, single error line + hint
- CI/CD: GitHub Actions CI + Release, GoReleaser (macOS-only), Homebrew tap
- `.depviz.yml` config for scanning the depviz codebase itself

### Files Changed
- `cmd/init.go` (new) — interactive huh form, detectLang, splitCSV
- `cmd/stats.go` (new) — scan pipeline → cli.Stats
- `internal/cli/output.go` (new) — Banner, ScanResult, ServeResult, InitResult
- `internal/cli/stats.go` (new) — Stats dashboard with coloured bars
- `cmd/root.go` — SilenceUsage, SilenceErrors, improved Execute()
- `cmd/scan.go` — Banner + ScanResult calls
- `cmd/serve.go` — Banner + ServeResult calls
- `internal/config/defaults.go` — `.depviz` in all default exclude lists
- `.github/workflows/ci.yml` (new)
- `.github/workflows/release.yml` (new)
- `.goreleaser.yml` (new)
- `.depviz.yml` (new)
