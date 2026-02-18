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

## Current Phase: Phase 1 — Core Semver Types (`internal/semver/`)

### Phase 0 — Project Bootstrap (Complete)
- Reference documentation written (7 docs)
- Implementation plan approved with 12 design improvements (DI-1 through DI-12)
- Go module initialized (`go.mod` go 1.26, `main.go`)
- `.github/instructions/CLAUDE.md` instructions file
- `.github/instructions/go.instructions.md` Go coding standards
- Makefile rewritten for single-module build
- CI pipeline with test/lint/vuln/build/status-check + GitHub Release on `v*` tags
- `.github/release.yml` for changelog categories
- Linter config updated (local-prefixes: go-gitsemver)
- README written with full feature documentation

### Phase 1 — Core Semver Types (Complete)
- **Design improvements:** DI-1 (immutable types), DI-2 (separate increment methods), DI-5 (named format methods), DI-6 (pure format function)
- **Files:** `enums.go`, `prereleasetag.go`, `buildmetadata.go`, `version.go`, `formatvalues.go` + tests
- **Dependencies:** stdlib only + testify/require
- **Coverage:** 98.4%

### Next: Phase 2 — Configuration (`internal/config/`)

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
