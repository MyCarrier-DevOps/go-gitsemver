# GitVersion 5.12.0 Architecture Overview

This document describes the architecture of the original C# GitVersion 5.12.0 application, serving as the reference for the Go rewrite (`go-gitsemver`).

## Purpose

GitVersion is a tool that automatically calculates semantic versions (SemVer 2.0) from git history. It analyzes tags, commits, branches, and merge relationships to determine the next version number without manual version file management.

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     CLI / Entry Point                    │
│          (GitVersionApp, ArgumentParser, Output)         │
└────────────────────────┬────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────┐
│                  GitVersionContext                        │
│  (CurrentBranch, CurrentCommit, Config, TaggedVersion,   │
│   NumberOfUncommittedChanges)                             │
└────────────────────────┬────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────┐
│               NextVersionCalculator                      │
│  (Orchestrates: strategies → increment → deployment mode │
│   → pre-release tag → final version)                     │
└────────┬───────────────┬──────────────────┬─────────────┘
         │               │                  │
┌────────▼───┐  ┌───────▼──────┐  ┌────────▼─────────┐
│   Base      │  │  Increment   │  │   Mainline        │
│   Version   │  │  Strategy    │  │   Version         │
│   Calculator│  │  Finder      │  │   Calculator      │
│  (6 strats) │  │              │  │                   │
└─────────────┘  └──────────────┘  └───────────────────┘
         │               │                  │
┌────────▼───────────────▼──────────────────▼─────────┐
│                  RepositoryStore                      │
│   (Tags, Commits, Branches, Merge History queries)   │
│                via LibGit2Sharp                       │
└──────────────────────────────────────────────────────┘
```

## Core Source Files (v5.12.0)

All paths relative to `GitVersion/src/GitVersion.Core/`:

### Version Calculation
| File | Responsibility |
|------|---------------|
| `VersionCalculation/NextVersionCalculator.cs` | Main orchestrator - runs strategies, applies increment, handles pre-release tags |
| `VersionCalculation/BaseVersionCalculator.cs` | Runs all 6 strategies, selects maximum version, resolves ties by oldest source |
| `VersionCalculation/MainlineVersionCalculator.cs` | Mainline mode - walks commits, aggregates merge increments |
| `VersionCalculation/IncrementStrategyFinder.cs` | Scans commit messages for `+semver:` bump directives |
| `VersionCalculation/VariableProvider.cs` | Converts SemanticVersion into output variables (30+ vars) |

### Version Strategies
| File | Strategy |
|------|----------|
| `VersionCalculation/BaseVersionCalculators/ConfigNextVersionVersionStrategy.cs` | Uses `next-version` from config file |
| `VersionCalculation/BaseVersionCalculators/TaggedCommitVersionStrategy.cs` | Extracts version from git tags |
| `VersionCalculation/BaseVersionCalculators/MergeMessageVersionStrategy.cs` | Extracts version from merge commit messages |
| `VersionCalculation/BaseVersionCalculators/VersionInBranchNameVersionStrategy.cs` | Extracts version from branch name (release branches) |
| `VersionCalculation/BaseVersionCalculators/TrackReleaseBranchesVersionStrategy.cs` | Tracks release branch versions for develop-like branches |
| `VersionCalculation/BaseVersionCalculators/FallbackVersionStrategy.cs` | Returns `0.1.0` from root commit as last resort |
| `VersionCalculation/BaseVersionCalculators/BaseVersion.cs` | Base version value object (source, semanticVersion, shouldIncrement, baseVersionSource) |

### Semantic Versioning Types
| File | Responsibility |
|------|---------------|
| `VersionCalculation/SemanticVersioning/SemanticVersion.cs` | Core type: Major.Minor.Patch with parsing, comparison, increment, formatting |
| `VersionCalculation/SemanticVersioning/SemanticVersionPreReleaseTag.cs` | Pre-release tag: Name + Number (e.g., `beta.4`) |
| `VersionCalculation/SemanticVersioning/SemanticVersionBuildMetaData.cs` | Build metadata: CommitsSinceTag, Branch, SHA, etc. |
| `VersionCalculation/SemanticVersioning/SemanticVersionFormatValues.cs` | All 30+ output format values |
| `VersionCalculation/SemanticVersioning/VersionField.cs` | Enum: None, Patch, Minor, Major |
| `VersionCalculation/IncrementStrategy.cs` | Enum: None, Major, Minor, Patch, Inherit |
| `VersionCalculation/VersioningMode.cs` | Enum: ContinuousDelivery, ContinuousDeployment, Mainline |

### Configuration
| File | Responsibility |
|------|---------------|
| `Model/Configuration/Config.cs` | Main config model (YAML-mapped), branch regex patterns, default keys |
| `Model/Configuration/BranchConfig.cs` | Per-branch config: increment, tag, mode, source branches, flags |
| `Model/Configuration/EffectiveConfiguration.cs` | Resolved config (merged global + branch), all values guaranteed non-null |
| `Configuration/ConfigurationBuilder.cs` | Builds default config, applies overrides, finalizes branch configs |
| `Configuration/ConfigProvider.cs` | Loads config from `.gitversion.yml` file |

### Git Layer
| File | Responsibility |
|------|---------------|
| `Core/RepositoryStore.cs` | All git queries: tags, branches, commits, merge bases |
| `Core/GitVersionContextFactory.cs` | Creates GitVersionContext (branch, commit, tagged version, config) |
| `Core/MergeBaseFinder.cs` | Finds merge base between branches |
| `Core/MergeCommitFinder.cs` | Finds merge commits on a branch |
| `Model/MergeMessage.cs` | Parses merge messages (6 formats: Default, SmartGit, BitBucket, GitHub, etc.) |

## Semantic Version Structure

```
Major.Minor.Patch[-PreReleaseTag][+BuildMetaData]

Examples:
  1.2.3                                              (stable release)
  1.2.3-beta.4                                       (pre-release)
  1.2.3-beta.4+5                                     (full semver with build number)
  1.2.3-beta.4+5.Branch.main.Sha.abc1234             (informational)
```

### SemanticVersion Fields

| Field           | Type    | Description |
|-----------------|---------|-------------|
| `Major`         | `long`  | Breaking changes |
| `Minor`         | `long`  | New features (backwards compatible) |
| `Patch`         | `long`  | Bug fixes (backwards compatible) |
| `PreReleaseTag` | struct  | `.Name` (string, e.g. "beta") + `.Number` (long?, e.g. 4) |
| `BuildMetaData` | struct  | CommitsSinceTag, Branch, Sha, ShortSha, CommitDate, VersionSourceSha, CommitsSinceVersionSource, UncommittedChanges |

### Version Format Specifiers

| Specifier | Name           | Example                                             |
|-----------|----------------|-----------------------------------------------------|
| `j`       | Just version   | `1.2.3`                                             |
| `s`       | Default SemVer | `1.2.3-beta.4`                                      |
| `f`       | Full SemVer    | `1.2.3-beta.4+5`                                    |
| `i`       | Informational  | `1.2.3-beta.4+5.Branch.main.Sha.abc1234`            |
| `t`       | Tagged         | `1.2.3-beta.4`                                      |
| `l`       | Legacy         | `1.2.3-beta4` (no dot separator)                    |
| `lp`      | Legacy padded  | `1.2.3-beta0004`                                    |

## Calculation Flow Summary

1. **Build Context** - Determine current branch, commit, load config, check if commit is tagged, count uncommitted changes
2. **Run Strategies** - Execute all 6 version strategies to get candidate base versions
3. **Select Maximum** - Pick the highest base version; on tie, prefer the oldest source commit for accurate commit counting
4. **Determine Increment** - Scan commit messages for `+semver:` bump directives, fall back to branch config increment
5. **Apply Increment** - Bump the base version (Major/Minor/Patch) based on determined field
6. **Apply Versioning Mode** - For Mainline mode, use MainlineVersionCalculator instead of standard increment
7. **Update Pre-Release Tag** - Apply branch-specific pre-release label, auto-increment number
8. **Handle Tagged Commits** - If current commit is tagged and tag version >= calculated, use tag version
9. **Generate Variables** - Convert final SemanticVersion into 30+ output variables via VariableProvider

See [SEMVER_CALCULATION.md](SEMVER_CALCULATION.md) for the detailed algorithm walkthrough.
