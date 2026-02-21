# Technology Stack

## Language & Runtime
- Go 1.25
- Module: `github.com/jtoloui/depviz`

## Dependencies
- `github.com/spf13/cobra` — CLI framework (subcommands, flags, help, completions)
- `gopkg.in/yaml.v3` — YAML config parsing
- `github.com/tree-sitter/go-tree-sitter` — Tree-sitter Go bindings (CGo, wraps C library) for AST-based JS/TS parsing
- `github.com/tree-sitter/tree-sitter-javascript` — Tree-sitter JavaScript grammar (`.js`, `.jsx`, `.mjs`)
- `github.com/tree-sitter/tree-sitter-typescript` — Tree-sitter TypeScript/TSX grammars (`.ts`, `.tsx`)
- `github.com/mattn/go-pointer` — Indirect dep of go-tree-sitter (CGo pointer handling)
- `github.com/common-nighthawk/go-figure` — ASCII art banner for CLI output
- `github.com/charmbracelet/huh` — Interactive terminal forms for `depviz init`

## Tooling
- golangci-lint v2.10.1 with errcheck enabled
- Makefile: `tidy → fmt → vet → test → lint → build`; `coverage` target generates `coverage.out` + `coverage.html`
- ldflags version injection from `git describe`
- Race detector on all test runs (`go test -race`)
- CGo required for tree-sitter (C compiler needed at build time, not at runtime)

## CI/CD
- GitHub Actions CI: `make` + `make coverage` + codecov upload + golangci-lint, concurrency cancels in-progress on PRs
- GitHub Actions Release: GoReleaser on `v*` tags, macOS runner, Zig for cross-compilation
- GoReleaser: CGo cross-compile — clang for darwin, zig cc for linux (musl static) + windows
- Homebrew cask published to `jtoloui/homebrew-tap` via GoReleaser
- Badges: CI status, codecov coverage, latest release

## Key Stdlib Packages Used
- `go/ast` + `go/parser` — Go source parsing (full AST for imports + exported declarations)
- `html/template` + `//go:embed` — HTML rendering compiled into binary
- `log/slog` — structured JSON logging to stderr
- `net/http` — serve command with explicit `http.Server`, `ReadHeaderTimeout`
- `os/signal` — graceful shutdown via `signal.NotifyContext`
- `regexp` — config classify rules
- `sync` — WaitGroup for worker pool coordination
- `runtime` — NumCPU for worker count

## No External Dependencies For
- HTTP routing (stdlib `http.NewServeMux`)
- Logging (stdlib `log/slog`)
- Testing (stdlib `testing`)
- Template rendering (stdlib `html/template`)

## CGo / Cross-Compilation Notes
- Tree-sitter grammars compile C source at build time via CGo
- Native builds on macOS/Linux work out of the box (clang/gcc available)
- Cross-compilation requires a C cross-compiler for the target (e.g. `zig cc`, or build per-platform in CI)
- Distributed binaries are fully static — end users need no C toolchain

## External CDN
- Devicon (https://devicon.dev) — file type icons loaded via CDN stylesheet at runtime
- MIT licensed; brand logos are property of their respective owners
- Output HTML requires network access on first load (cached after)

## Theme System
- 14 built-in themes defined as CSS custom property blocks (`[data-theme="..."]`)
- Theme selection persisted in localStorage (`depviz-theme` key)
- No external dependencies — all theme colours are inline CSS variables
- Each theme defines: bg, surface, border, text, text-muted, accent, 4 category colours + backgrounds, 3 code highlight colours
