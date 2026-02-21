# Code Conventions

## Go Style
- Follow https://go.dev/wiki/CodeReviewComments
- Follow https://github.com/uber-go/guide
- camelCase for functions/variables, PascalCase for exported
- Never underscores in Go identifiers

## Error Handling
- Wrap errors with `fmt.Errorf("context: %w", err)`
- Check with `errors.Is` for sentinel errors
- Validation inside constructors/loaders — never return half-valid state
- All file Close() calls must be checked (errcheck linter enforced)
- Read-only files: `defer func() { _ = f.Close() }()`
- Write files: explicit Close() with error check
- Tree-sitter CGo objects (Parser, Tree, Query, QueryCursor) must always be `defer x.Close()`'d — GC does not free C allocations

## Interfaces
- Compile-time checks: `var _ Interface = (*Type)(nil)`
- Keep interfaces small — one method where possible
- Define interfaces where they're consumed, not where they're implemented

## Testing
- Table-driven tests with `t.Run` subtests
- Black-box testing (package_test) where possible
- `t.Parallel()` on all tests and subtests
- `t.Helper()` on test helpers
- `t.TempDir()` for fixture directories (auto-cleanup)
- Race detector on all runs (`go test -race`)
- No `time.Sleep` in tests — use sync primitives

## Concurrency
- Worker pool pattern: walker goroutine → channel → N workers → results channel
- `runtime.NumCPU()` for worker count
- `sync.WaitGroup` for coordination
- Channels for communication, not shared memory

## Logging
- `log/slog` with JSON handler on stderr
- Debug level for internal details (config loaded, scanning path)
- Info level for user-facing events (scan complete, server started)
- `-v` flag enables debug level

## Config Priority
- .depviz.yml field > CLI flag > default value
- Config file is optional — defaults per language when absent

## Documentation

- When adding new packages, update `.kiro/steering/structure.md` with the package and its responsibility
- When adding new dependencies, update `.kiro/steering/tech.md`
- When completing a roadmap phase, mark it done in `.kiro/steering/product.md` and add a history entry in `.kiro/history/`
- When changing conventions or patterns, update `.kiro/steering/conventions.md`
- Keep `README.md` in sync with CLI flags, config options, and usage examples
- Code comments explain *why*, not *what* — don't restate the code
