# go-gitsemver Project State

## Purpose

Go rewrite of [GitVersion](https://github.com/GitTools/GitVersion) (v5.12.0 reference) with modifications. The goal is to build a Go-based semantic versioning tool that calculates versions from git history.

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

## Current State

- Project skeleton created with Makefile, linting config, CI pipeline
- No Go source code yet
- Next step: Review docs, plan Go architecture and implementation priorities
