# Project State

Tracks completed features, current work, and planned changes for go-gitsemver.

## Completed Features

### Core Version Calculation (Phase 0)
- Semantic versioning engine
- 6 version strategies: ConfigNextVersion, TaggedCommit, MergeMessage, TrackReleaseBranches, Fallback, VersionInBranchName
- 3 versioning modes: ContinuousDelivery, ContinuousDeployment, Mainline
- 5 branch types: mainline, develop, release, feature, unknown
- YAML configuration with `GitVersion.yml` / `go-gitsemver.yml` auto-detection
- CLI: `go-gitsemver calculate` with `--config`, `--output`, `--show-variable`, `--show-config`, `--explain`, `--branch`, `--commit`, `--path`
- 30+ output variables (SemVer, FullSemVer, MajorMinorPatch, CommitTag, etc.)
- Monorepo support via path filters

### GitHub API Remote Provider (Phase 1)
- `go-gitsemver remote owner/repo` subcommand — calculate versions via GitHub API, no clone required
- Token auth (`--token` / `GITHUB_TOKEN`) and GitHub App auth (`--github-app-id` + `--github-app-key` for PEM content / `--github-app-key-path` for PEM file)
- GitHub Enterprise support (`--github-url` / `GITHUB_API_URL`)
- `--ref` flag for branch, tag, or SHA targeting
- `--max-commits` safety cap on commit walk depth (default 1000)
- Smart early termination: stops paginated commit walks once a semver tag is found
- GraphQL batch queries for branches and tags (avoids N+1 REST calls)
- In-memory request-scoped caching layer
- Remote config fetching from `GitVersion.yml` / `go-gitsemver.yml` in the repo (and `.github/` directory)
- `--remote-config-path` flag for explicit remote config file targeting
- Files: `internal/github/{client,repository,cache,graphql}.go`, `cmd/remote.go`

### Bug Fixes (Copilot Review)
1. GHE GraphQL endpoint: derives `/api/graphql` from `/api/v3` base URL
2. versionTagSHAs filter: only semver tags trigger early termination
3. Base URL consistency: resolved once and passed to both client and repository
4. loadRemoteConfig error handling: distinguishes 404 from auth/network errors
5. Head() tag resolution: falls back to tag lookup when branch returns 404
6. Empty OID handling: skips branches with empty target OID in GraphQL
7. parseOwnerRepo validation: rejects `owner/repo/extra` format

### Go Library API (Phase 2)
- Public Go package at `pkg/sdk/` for programmatic version calculation
- `sdk.Calculate(LocalOptions) (*Result, error)` — local repo via go-git
- `sdk.CalculateRemote(RemoteOptions) (*Result, error)` — remote via GitHub API
- `Result.Variables` map with all 30+ output variables
- Auto-detects `GitVersion.yml` / `go-gitsemver.yml` config files
- Files: `pkg/sdk/sdk.go`, `pkg/sdk/sdk_test.go`

### Explain Mode (Phase 3)
- `--explain` flag on CLI — full transparency into version calculation, output to stderr
- Shows all strategies evaluated with their candidates and reasoning
- Displays which strategy was selected and why
- Records increment reasoning: which commits drove the bump and the convention used
- Shows pre-release tag resolution steps for feature/develop branches
- `IncrementExplanation` type tracks increment decision reasoning through the pipeline
- `DetermineIncrementedFieldExplained()` — records per-commit bump analysis
- `output.WriteExplanation()` — formatter renders structured explain output to io.Writer
- `output.FormatExplanation()` — returns explain output as string (used by SDK)
- SDK: `ExplainResult` / `ExplainCandidate` public types for programmatic access
- SDK: `LocalOptions.Explain` and `RemoteOptions.Explain` fields
- SDK: `Result.ExplainResult` populated when explain is enabled
- Files: `internal/output/explain.go`, `internal/calculator/increment.go` (extended)

### Module & Package Rename (Phase 4)
- Module path updated to `github.com/MyCarrier-DevOps/go-gitsemver` for valid pkg.go.dev resolution
- Public package renamed from `pkg/gitsemver` to `pkg/sdk` — import: `github.com/MyCarrier-DevOps/go-gitsemver/pkg/sdk`
- Config file renamed from `gitsemver.yml` to `go-gitsemver.yml` (`GitVersion.yml` still supported)

### Config Search Paths & Remote Config Path (Phase 5)
- Config auto-detection expanded to search `.github/` directory before repo root
- Search order: `.github/GitVersion.yml`, `.github/go-gitsemver.yml`, `GitVersion.yml`, `go-gitsemver.yml`
- `--remote-config-path` CLI flag for remote mode — fetch a specific config file from the remote repo
- `RemoteOptions.RemoteConfigPath` SDK field for programmatic remote config path control
- `testutil.WriteConfigAt()` helper for testing config files at custom paths

### CLI Rename & Documentation Overhaul (Phase 6)
- CLI binary renamed from `gitsemver` to `go-gitsemver` — all commands, makefile, CI workflow updated
- All documentation updated to use `go-gitsemver` consistently
- `docs/ARCHITECTURE.md` package tree updated to reflect `pkg/sdk/`

### GitHub App Key Split, Checksum Verification & CI (Phase 7)
- `--github-app-key` now accepts PEM key content directly (ideal for CI secrets)
- New `--github-app-key-path` flag for file path to PEM file (previous behavior)
- New env vars: `GH_APP_PRIVATE_KEY` (content), `GH_APP_PRIVATE_KEY_PATH` (file path)
- Auth resolution: content takes precedence over file path
- SDK `RemoteOptions` has both `AppKey` (content) and `AppKeyPath` (file path) fields
- GitHub Action: SHA-256 checksum verification via `checksums.txt` (enabled by default)
- GitHub Action: `verify-checksum` input to disable verification
- `make release-build` target: cross-compiles all platforms and generates `checksums.txt`
- CI workflow updated: release artifacts use `go-gitsemver-*` naming
- Files: `internal/github/client.go`, `cmd/remote.go`, `pkg/sdk/sdk.go`, `.github/actions/setup-go-gitsemver/action.yml`, `.github/workflows/ci.yaml`, `makefile`

## Test Coverage
- 589 tests across unit, integration, and end-to-end suites
- 85% overall coverage
- E2E tests create real Git repositories with branches, tags, and commits
- Mock-based unit tests for isolated strategy and calculator testing

## Documentation

- `docs/HIGHLIGHTS.md` — application highlights for presentations
- `docs/ARCHITECTURE.md` — package structure and design
- `docs/CONFIGURATION.md` — all config options with defaults
- `docs/STRATEGIES_AND_MODES.md` — strategies and versioning modes
- `docs/BRANCH_WORKFLOWS.md` — branch types and priority matching
- `docs/VERSION_STRATEGIES.md` — 6 version discovery strategies
- `docs/FEATURES.md` — key features and design highlights
- `docs/examples/` — example configuration files
- `docs/GITHUB_ACTION.md` — GitHub Action setup, usage examples, checksum verification
