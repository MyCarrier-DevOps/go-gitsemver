# go-gitsemver Project State

## Purpose

Go rewrite of GitVersion (v5.12.0 reference) with modifications. The goal is to build a Go-based semantic versioning tool that calculates versions from git history.

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

## Implementation Plan

See `../.claude/plans/jazzy-orbiting-graham.md` for the full plan.

8 phases: Bootstrap → Semver Types → Config → Git Adapter → Context/Strategies → Calculators → Output → CLI → Integration Tests

## Current Phase: Phase 0 — Project Bootstrap

### Completed
- Reference documentation written (7 docs)
- Implementation plan approved

### In Progress
- Initializing Go module (`go.mod`, `main.go`)
- Creating `CLAUDE.md` instructions file
- Rewriting Makefile for single-module build
- Updating CI pipeline (removing template values)
- Updating linter config (local-prefixes)

### Package Structure (planned)

```
go-gitsemver/
├── cmd/                    # CLI (cobra)
├── internal/
│   ├── semver/             # SemanticVersion, PreReleaseTag, BuildMetaData, enums
│   ├── config/             # YAML config, defaults, builder, effective config
│   ├── git/                # Repository interface, go-git impl, mock, repostore, merge message parser
│   ├── context/            # GitVersionContext
│   ├── strategy/           # 6 version strategies
│   ├── calculator/         # NextVersion, BaseVersion, Mainline, IncrementStrategyFinder
│   ├── output/             # VariableProvider, JSON, AssemblyInfo updater
│   └── fsadapter/          # FileSystem interface + OS/mock implementations
├── go.mod
└── main.go
```
