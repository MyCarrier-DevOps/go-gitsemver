# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.6.0] - GitHub App Key Content Split

### Changed

- **`--github-app-key` now accepts PEM key content directly** instead of a file path — ideal for CI secrets where the key is stored as an environment variable or secret
- **New `--github-app-key-path` flag** accepts a file path to a PEM file (previous behavior of `--github-app-key`)
- **New `GH_APP_PRIVATE_KEY_PATH` env var** for file-path-based key configuration; `GH_APP_PRIVATE_KEY` now expects PEM content
- Auth resolution: content (`--github-app-key`/`GH_APP_PRIVATE_KEY`) takes precedence over file path (`--github-app-key-path`/`GH_APP_PRIVATE_KEY_PATH`)
- SDK `RemoteOptions` now has both `AppKey` (content) and `AppKeyPath` (file path) fields

## [1.5.0] - GitHub Action, Checksum Verification & CI Workflow

### Added

- **GitHub Action** (`.github/actions/setup-go-gitsemver/action.yml`) — composite action that installs go-gitsemver from GitHub Releases with cross-platform support (Linux, macOS, Windows)
- **`verify-checksum` input** on the GitHub Action — SHA-256 checksum verification against `checksums.txt` (enabled by default); disable with `verify-checksum: 'false'`
- **`token` input** on the GitHub Action — authenticated downloads to avoid rate limits on private repos
- **`docs/GITHUB_ACTION.md`** — full documentation for the GitHub Action with usage examples (local mode, remote mode, explain mode), supported platforms, and token usage
- **`make release-build` target** — cross-compiles for all supported platforms and generates `checksums.txt`
- **CI workflow updated** — release artifacts and checksums now use `go-gitsemver-*` binary naming

## [1.4.0] - CLI Rename & Documentation Overhaul

### Changed

- **CLI binary renamed from `gitsemver` to `go-gitsemver`** — all commands are now invoked as `go-gitsemver` (e.g. `go-gitsemver --show-variable SemVer`, `go-gitsemver remote owner/repo`)
- **All documentation updated** to use `go-gitsemver` consistently across README, CHANGELOG, FEATURES, CONFIGURATION, ARCHITECTURE, STRATEGIES_AND_MODES, VERSION_STRATEGIES, BRANCH_WORKFLOWS, and HIGHLIGHTS
- **`docs/ARCHITECTURE.md`** package tree updated to reflect `pkg/sdk/` (was still showing old `pkg/gitsemver/` paths)


## [1.3.0] - Explain Mode, Package Rename, Config Search Paths & Remote Config Path

### Added

- **`.github/` config auto-detection** — config file search now checks `.github/GitVersion.yml` and `.github/go-gitsemver.yml` before repo root, for both local and remote modes
- **`--remote-config-path` flag** — specify a config file path in the remote repo (e.g. `--remote-config-path .github/GitVersion.yml`) instead of relying on auto-detection
- **`RemoteConfigPath` SDK field** — `RemoteOptions.RemoteConfigPath` for programmatic control of remote config file location
- **`--explain` flag** — full transparency into version calculation, output to stderr
  - Shows all strategies evaluated with their candidates
  - Displays which strategy was selected and why
  - Records increment reasoning: which commits drove the bump and the convention used
  - Shows pre-release tag resolution steps for feature/develop branches
  - Structured output: Strategies evaluated → Selected → Increment → Pre-release → Result
- **`IncrementExplanation`** — new type tracking increment decision reasoning through the calculator pipeline
- **`ExplainResult` / `ExplainCandidate`** — public types in the SDK for programmatic access to explain data
- **`output.WriteExplanation()`** — formatter that renders structured explain output
- **`output.FormatExplanation()`** — returns explain output as a string (used by SDK)

### Changed

- **Renamed `pkg/gitsemver` → `pkg/sdk`** — import path is now `github.com/MyCarrier-DevOps/go-gitsemver/pkg/sdk`
- **Renamed config file `gitsemver.yml` → `go-gitsemver.yml`** — `GitVersion.yml` still supported as an alternative
- **Module path** updated to `github.com/MyCarrier-DevOps/go-gitsemver` for valid pkg.go.dev resolution
- **Default `base-version` changed from `0.1.0` to `1.0.0`** — new repos without tags now start at `1.0.0` instead of `0.1.0`
- SDK `LocalOptions` and `RemoteOptions` now include `Explain bool` field
- SDK `Result` now includes `ExplainResult *ExplainResult` (nil when explain is disabled)

## [1.2.0] - Go Library API

### Added

- **`pkg/sdk` public Go library** — embed version calculation in Go applications without shelling out to the CLI
  - `Calculate(LocalOptions)` for local git repositories
  - `CalculateRemote(RemoteOptions)` for GitHub API-based calculation
  - `Result.Variables` map with all 30+ output variables
  - Auto-detects `go-gitsemver.yml` / `GitVersion.yml` config files
- `example/main.go` — runnable example demonstrating library usage

## [1.1.0] - GitHub API Remote Provider

### Added

- **`go-gitsemver remote owner/repo` subcommand** — calculate semantic versions via the GitHub REST and GraphQL APIs without requiring a local clone. Eliminates the need for `fetch-depth: 0` in CI pipelines.
  - Token auth (`--token` / `GITHUB_TOKEN`) and GitHub App auth (`--github-app-id` + `--github-app-key`)
  - GitHub Enterprise support via `--github-url` / `GITHUB_API_URL`
  - GraphQL batch fetching for branches and tags (avoids N+1 REST calls)
  - Smart early termination for commit walks using version tag detection
  - In-memory caching across the run (branches, tags, commits, merge bases)
  - Configurable `--max-commits` safety cap (default 1000)
  - Remote config file fetching (`go-gitsemver.yml` / `GitVersion.yml` from repo root via API)
- `CommitTag` output variable — `YY.WW.ShortSha` format derived from the commit date
- Date format translation between Go and .NET/Python/strftime conventions
- JSON schema for go-gitsemver configuration

### Changed

- Updated GitHub Actions to latest versions (checkout v6, setup-go v6, upload-artifact v6, download-artifact v7)

## [1.0.0] - Initial Release

- Automatic semantic versioning from git history
- 6 version discovery strategies (ConfigNextVersion, TaggedCommit, MergeMessage, VersionInBranchName, TrackReleaseBranches, Fallback)
- 3 versioning modes (ContinuousDelivery, ContinuousDeployment, Mainline)
- Conventional Commits and bump directive support
- 8 built-in branch configurations with priority-based matching
- Squash merge awareness (GitHub, GitLab, Bitbucket formats)
- 30+ output variables with JSON, key=value, and single-variable output
- GitVersion.yml configuration compatibility
