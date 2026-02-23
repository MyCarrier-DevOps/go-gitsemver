# Key Features

gitsemver is a semantic versioning tool that automatically calculates versions from git history. It is inspired by [GitVersion](https://gitversion.net/) and reimplements its core functionality in Go as a single static binary with zero runtime dependencies.

This document highlights gitsemver's key capabilities and design decisions.

---

## Single Binary, Zero Dependencies

gitsemver is a single static binary (~10-15MB). Download it and run — no runtime, no SDK, no package manager required. Works with any language and any build system.

Cross-platform builds are provided for Linux (amd64/arm64), macOS (amd64/arm64), and Windows (amd64).

---

## Immutable Version Types

All version types (`SemanticVersion`, `PreReleaseTag`, `BuildMetaData`) are immutable value types. Methods like `IncrementField()`, `WithPreReleaseTag()`, and `WithBuildMetaData()` return new values instead of mutating in place. This eliminates hidden state changes through the calculation pipeline.

---

## Clear Increment Operations

Two explicit methods for version bumping:
- `IncrementField(Major/Minor/Patch)` — always bumps the version field
- `IncrementPreRelease()` — always bumps the pre-release number

The caller decides which to use. No ambiguity, no implicit behavior changes based on pre-release state.

---

## Single-Increment Pipeline

Candidate base versions are ranked by computing an effective version for comparison (without mutation). The winning candidate is incremented exactly once. This avoids subtle double-increment bugs that arise from tentatively incrementing candidates during comparison and then incrementing the winner again.

---

## Conventional Commits Support

First-class support for the [Conventional Commits](https://www.conventionalcommits.org/) specification:

| Commit Pattern | Increment |
|----------------|-----------|
| `feat:` or `feat(scope):` | Minor |
| `fix:` or `fix(scope):` | Patch |
| `feat!:` or `fix!:` (any type with `!`) | Major |
| `BREAKING CHANGE:` in commit footer | Major |

Also supports simple bump directives:
- `bump major` / `bump minor` / `bump patch` / `bump none`

Configurable via `commit-message-convention: conventional-commits | bump-directive | both`

---

## Squash Merge Awareness

gitsemver parses squash merge formats out of the box, which is critical since squash merges are the default on GitHub and GitLab:

| Source | Pattern Example |
|--------|----------------|
| GitHub squash | `feat: add login page (#123)` |
| GitLab squash | `Merge branch 'feature/x' into 'main'` |
| Bitbucket squash | `Merged in feature/auth (pull request #123)` |

Custom patterns can be added via `merge-message-formats` in config.

---

## Explain Mode

The `--explain` flag outputs a structured decision tree showing exactly how the version was calculated:

```
Strategies evaluated:
  TaggedCommit:  1.2.0 from tag v1.2.0 (3 commits ago) → effective 1.3.0
  MergeMessage:  (none)
  BranchName:    (none)
  Fallback:      0.1.0

Selected: TaggedCommit (effective 1.3.0, oldest source at 2025-01-15)

Increment: Minor
  Source: commit abc1234 "feat: add user authentication" (Conventional Commits)

Pre-release: feature-login.1
  Branch config tag: {BranchName} → "feature-login"
  No existing tag for 1.3.0-feature-login → number = 1

Result: 1.3.0-feature-login.1+3
```

---

## Aggregate Mainline Calculation

In Mainline mode, gitsemver scans all commits since the last tag and applies the **single highest** increment once:

```
v1.0.0 → fix → fix → feat → fix → fix
Result: 1.1.0+5  (Minor applied once, 5 commits in metadata)
```

This keeps version numbers semantically meaningful. An alternative `EachCommit` mode is available via the `mainline-increment` config option for teams that prefer per-commit incrementing.

---

## Priority-Based Branch Matching

Branch configs are matched by regex with explicit priority ordering. The highest priority wins, eliminating the ambiguity of "first regex match wins." Default priorities:

| Branch | Priority |
|--------|----------|
| main | 100 |
| release | 90 |
| hotfix | 80 |
| support | 70 |
| develop | 60 |
| feature | 50 |
| pull-request | 40 |
| unknown (catch-all) | 0 |

Custom branches can specify their own priority:

```yaml
branches:
  staging:
    regex: ^staging$
    priority: 85
```

---

## Catch-All Branch Config

A built-in `unknown` branch config (`.*` regex, priority 0) catches any branch that doesn't match a known pattern. Unrecognized branches are treated like feature branches with `{BranchName}` as the pre-release tag. No errors, no surprises.

---

## Configurable Base Version

The `base-version` config option (default: `0.1.0`) sets the starting version when no tags exist. This is a permanent setting, separate from `next-version` which is a temporary override.

```yaml
base-version: 1.0.0
```

---

## Shallow Clone Protection

gitsemver detects shallow clones and exits with a clear error by default. The `--allow-shallow` flag explicitly opts into running with potentially incomplete history. The error message suggests `git fetch --unshallow`.

---

## Pure Output Functions

Output computation is handled by pure functions with no side effects:
- `ComputeFormatValues(ver, config) → map[string]string` generates all 25+ output variables
- `PromoteCommitsToPreRelease(ver, mode, fallbackTag) → ver` handles ContinuousDeployment mode

gitsemver outputs version information — it does not write to files. CI/CD scripts consume the output variables.

---

## 25+ Output Variables

gitsemver generates a comprehensive set of output variables compatible with various ecosystems:

- **SemVer formats:** `SemVer`, `FullSemVer`, `LegacySemVer`, `InformationalVersion`
- **Components:** `Major`, `Minor`, `Patch`, `MajorMinorPatch`
- **Pre-release:** `PreReleaseTag`, `PreReleaseLabel`, `PreReleaseNumber`
- **Build metadata:** `BuildMetaData`, `FullBuildMetaData`, `CommitsSinceVersionSource`
- **Git info:** `BranchName`, `Sha`, `ShortSha`, `CommitDate`
- **Assembly:** `AssemblySemVer`, `AssemblyFileSemVer`
- **NuGet:** `NuGetVersionV2`, `NuGetPreReleaseTagV2`

Output formats: JSON (default), key=value text, or single variable via `--show-variable`.
