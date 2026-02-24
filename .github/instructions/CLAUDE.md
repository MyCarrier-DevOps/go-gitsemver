# Claude Instructions for go-gitsemver

## After Making Changes

Run these commands in order and fix any issues before considering the work done:

```bash
make tidy    # clean up module dependencies
make fmt     # format code (gofumpt + goimports)
make lint    # run golangci-lint
make test    # run tests with coverage
```

Verify that test coverage stays **above 85%** overall.

## Code Conventions

- **Go style:** Follow `.github/instructions/go.instructions.md` for idiomatic Go practices
- **Testing:** Use `testify/require` — never `testify/assert` (enforced by linter). Code coverage must be above 85%.
- **Adapter patterns:** Git operations go behind the `Repository` interface (`internal/git/interfaces.go`)
- **Table-driven tests:** Prefer table-driven tests for functions with clear input/output
- **Formatting:** Code must pass `gofumpt` with extra rules and `goimports` with `go-gitsemver` as the local prefix

## Project Structure

- `cmd/` — CLI commands (cobra)
- `internal/semver/` — Pure semantic version types (zero external deps)
- `internal/config/` — YAML config loading, defaults, builder, effective config
- `internal/git/` — Git adapter interface, go-git implementation, mock, repository store, merge message parser
- `internal/context/` — GitVersionContext
- `internal/strategy/` — 6 version strategies
- `internal/calculator/` — Version calculators and increment strategy finder
- `internal/output/` — Variable provider, JSON output

## Reference Documentation

All docs are in `docs/`:
- `CONFIGURATION.md` — All config options with defaults
- `STRATEGIES_AND_MODES.md` — Version strategies, versioning modes, manual overrides
- `BRANCH_WORKFLOWS.md` — Branch types, defaults, priority matching
- `VERSION_STRATEGIES.md` — How the 6 version discovery strategies work
- `ARCHITECTURE.md` — Package structure and design principles
- `FEATURES.md` — Key features and design highlights
