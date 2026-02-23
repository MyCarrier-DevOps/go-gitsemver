# go-gitsemver Project State

## Purpose

Go rewrite of GitVersion (v5.12.0 reference) with design improvements. The goal is to build a Go-based semantic versioning tool that calculates versions from git history.

## Reference Implementation

- **Source:** `../GitVersion/` (checked out at tag `5.12.0`)
- **Language:** C# / .NET
- **Key library:** LibGit2Sharp for git operations

## Documentation Index

| Document | Description |
|----------|-------------|
| [ARCHITECTURE.md](ARCHITECTURE.md) | High-level architecture, module layout, source file map, calculation flow summary |
| [SEMVER_CALCULATION.md](SEMVER_CALCULATION.md) | Detailed step-by-step algorithm: context creation, base version selection, increment logic, mainline mode, end-to-end example |
| [VERSION_STRATEGIES.md](VERSION_STRATEGIES.md) | All 6 version strategies: ConfigNextVersion, TaggedCommit, MergeMessage, VersionInBranchName, TrackReleaseBranches, Fallback |
| [BRANCH_WORKFLOWS.md](BRANCH_WORKFLOWS.md) | Default branch configs (main, develop, release, feature, hotfix, pull-request, support), versioning modes, GitFlow example |
| [CONFIGURATION.md](CONFIGURATION.md) | All global and branch config options with defaults, config file format, resolution order |
| [GIT_ANALYSIS.md](GIT_ANALYSIS.md) | How tags, commits, branches, merge history, and uncommitted changes are used |
| [CLI_INTERFACE.md](CLI_INTERFACE.md) | CLI arguments, 30+ output variables, output formats, caching, version formatting detail |
| [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md) | Full implementation plan with 12 design improvements and 8 phases |
| [STRATEGIES_AND_MODES.md](STRATEGIES_AND_MODES.md) | All 6 strategies, 3 versioning modes, manual overrides with examples and config files |
| [examples/](examples/) | Example `gitsemver.yml` configs for different workflows (GitFlow, trunk-based, CD, GitHub Flow, etc.) |
| [COMPARISON.md](COMPARISON.md) | What's better in gitsemver vs GitVersion v5.12.0 — all 12 DIs + additional improvements |

## Current Phase: Phase 4 — Version Context (`internal/context/`)

### Phase 0 — Project Bootstrap (Complete)
- Reference documentation written (7 docs)
- Implementation plan approved with 12 design improvements (DI-1 through DI-12)
- Go module initialized (`go.mod` go 1.26, `main.go`)
- `.github/instructions/CLAUDE.md` instructions file
- `.github/instructions/go.instructions.md` Go coding standards
- Makefile rewritten for single-module build
- CI pipeline with test/lint/vuln/build/status-check + GitHub Release on `v*` tags
- `.github/release.yml` for changelog categories
- Linter config updated for golangci-lint v2.9.0 (gofumpt with extra-rules)
- README written with full feature documentation

### Phase 1 — Core Semver Types (Complete)
- **Design improvements:** DI-1 (immutable types), DI-2 (separate increment methods), DI-5 (named format methods), DI-6 (pure format function)
- **Files:** `enums.go`, `prereleasetag.go`, `buildmetadata.go`, `version.go`, `formatvalues.go` + tests
- **Dependencies:** stdlib only + testify/require
- **Coverage:** 98.4%

### Phase 2 — Configuration (Complete)
- **Design improvements:** DI-7 (Conventional Commits config field), DI-12 (priority-based branch matching)
- **Semver additions:** `ParseXxx` functions for all 4 enum types, `UnmarshalYAML` methods (`yaml.go`, `parse_test.go`)
- **Config files:**
  - `helpers.go` — pointer helper functions
  - `ignore.go` — `IgnoreConfig` with flexible date parsing
  - `branch.go` — `BranchConfig` with 15 pointer fields + `MergeTo` coalesce
  - `config.go` — root `Config` struct with YAML tags
  - `loader.go` — `LoadFromFile` / `LoadFromBytes` via gopkg.in/yaml.v3
  - `defaults.go` — `CreateDefaultConfiguration` with 8 branch defaults and priority ordering
  - `builder.go` — `Builder` with overlay merging, develop special-case mode, `IsSourceBranchFor` processing, validation
  - `effective.go` — `EffectiveConfiguration` with all concrete types (no pointers)
  - `extensions.go` — `GetBranchConfiguration` (priority-sorted regex match), `GetReleaseBranchConfig`, `GetBranchSpecificTag`
- **All files have corresponding `_test.go` files**
- **Dependencies:** stdlib + testify/require + gopkg.in/yaml.v3
- **Coverage:** config 91.5%, semver 97.3%, overall 94.4%

### Phase 3 — Git Adapter (Complete)
- **Design improvements:** DI-8 (squash merge awareness), DI-11 (monorepo-ready PathFilter)
- **Git files:**
  - `types.go` — `PathFilter`, `ObjectID`, `Commit`, `ReferenceName`, `Branch`, `Tag`, `BranchCommit`, `VersionTag`
  - `interfaces.go` — `Repository` interface with 15 methods
  - `mergemessage.go` — 6 default + 2 squash merge message formats, `ParseMergeMessage`, `ExtractVersionFromBranch`
  - `mock.go` — `MockRepository` with function fields for all 15 methods
  - `repostore.go` — `RepositoryStore` with 18 domain query methods (tag/branch/commit/merge-base queries)
  - `gogit.go` — full go-git `Repository` implementation via `github.com/go-git/go-git/v5`
- **All files have corresponding `_test.go` files**
- **Dependencies:** stdlib + testify/require + gopkg.in/yaml.v3 + go-git/go-git/v5
- **Coverage:** git 84.5%, config 91.5%, semver 97.3%, overall 89.8%

### Next: Phase 4 — Version Context (`internal/context/`)

## Package Structure

```
go-gitsemver/
├── cmd/                    # CLI (cobra)
├── internal/
│   ├── semver/             # SemanticVersion, PreReleaseTag, BuildMetaData, enums
│   ├── config/             # YAML config, defaults, builder, effective config
│   ├── git/                # Repository interface, go-git impl, mock, repostore, merge message parser
│   ├── context/            # GitVersionContext
│   ├── strategy/           # Version strategies
│   ├── calculator/         # NextVersion, BaseVersion, Mainline, IncrementStrategyFinder
│   ├── output/             # VariableProvider, JSON output
│   └── testutil/           # Test helpers (temp git repos)
├── docs/                   # Reference docs and project state
├── go.mod
└── main.go
```
