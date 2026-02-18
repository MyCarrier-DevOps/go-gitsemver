# Claude Instructions for go-gitsemver

## Before Starting Any Work

1. **Always update `docs/PROJECT_STATE.md`** before beginning changes. Record what phase you're working on, what's in progress, and what's completed.

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
- **Testing:** Use `testify/require` — never `testify/assert` (enforced by linter)
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

All design docs are in `docs/`:
- `ARCHITECTURE.md` — System design overview
- `SEMVER_CALCULATION.md` — Step-by-step calculation algorithm
- `VERSION_STRATEGIES.md` — All 6 version strategies
- `BRANCH_WORKFLOWS.md` — Branch types and defaults
- `CONFIGURATION.md` — All config options
- `GIT_ANALYSIS.md` — How git data is used
- `CLI_INTERFACE.md` — CLI args and output variables
