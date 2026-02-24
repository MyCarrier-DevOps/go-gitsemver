# Project State

Tracks completed features, current work, and planned changes for go-gitsemver.

## Completed Features

### Core Version Calculation (Phase 0)
- Semantic versioning engine
- 6 version strategies: ConfigNextVersion, TaggedCommit, MergeMessage, TrackReleaseBranches, Fallback, VersionInBranchName
- 3 versioning modes: ContinuousDelivery, ContinuousDeployment, Mainline
- 5 branch types: mainline, develop, release, feature, unknown
- YAML configuration with `GitVersion.yml` / `gitsemver.yml` auto-detection
- CLI: `gitsemver calculate` with `--config`, `--output`, `--show-variable`, `--show-config`, `--explain`, `--branch`, `--commit`, `--path`
- 30+ output variables (SemVer, FullSemVer, MajorMinorPatch, etc.)
- Monorepo support via path filters

### GitHub API Remote Provider (Phase 1)
- `gitsemver remote owner/repo` subcommand — calculate versions via GitHub API, no clone required
- Token auth (`--token` / `GITHUB_TOKEN`) and GitHub App auth (`--github-app-id` / `--github-app-key`)
- GitHub Enterprise support (`--github-url` / `GITHUB_API_URL`)
- `--ref` flag for branch, tag, or SHA targeting
- `--max-commits` safety cap on commit walk depth (default 1000)
- Smart early termination: stops paginated commit walks once a semver tag is found
- GraphQL batch queries for branches and tags (avoids N+1 REST calls)
- In-memory request-scoped caching layer
- Remote config fetching from `GitVersion.yml` / `gitsemver.yml` in the repo
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
- Public Go package at `pkg/gitsemver/` for programmatic version calculation
- `Calculate(LocalOptions) (*Result, error)` — local repo via go-git
- `CalculateRemote(RemoteOptions) (*Result, error)` — remote via GitHub API
- `Result.Variables` map with all 30+ output variables
- Auto-detects `GitVersion.yml` / `gitsemver.yml` config files
- 13 unit tests covering local, remote (httptest mock), config, error cases
- Files: `pkg/gitsemver/gitsemver.go`, `pkg/gitsemver/gitsemver_test.go`

## Potential Future Work
- CLI refactor to use library API as thin wrapper (optional)
- Module path migration to `github.com/org/go-gitsemver` for external imports
- Separate `pkg/gitsemver/remote` sub-package to reduce dependency weight

## Documentation

- `docs/ARCHITECTURE.md` — package structure and design
- `docs/CONFIGURATION.md` — all config options with defaults
- `docs/STRATEGIES_AND_MODES.md` — strategies and versioning modes
- `docs/BRANCH_WORKFLOWS.md` — branch types and priority matching
- `docs/VERSION_STRATEGIES.md` — 6 version discovery strategies
- `docs/FEATURES.md` — key features and design highlights
- `docs/examples/` — example configuration files
- `docs/future/go-library-plan.md` — Go library API plan
