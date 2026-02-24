# Branch Workflows and Default Configuration

go-gitsemver supports different branching strategies through per-branch configuration. This document details the default configuration for each branch type and how the versioning modes differ.

---

## Default Branch Configurations

go-gitsemver ships with 8 default branch configurations, ordered by priority. When a branch name matches multiple regexes, the highest priority wins.

### main (Priority: 100)

| Property | Value |
|----------|-------|
| Regex | `^master$\|^main$` |
| Increment | `Patch` |
| Tag | `""` (empty = stable, no pre-release) |
| IsMainline | `true` |
| IsReleaseBranch | `false` |
| TracksReleaseBranches | `false` |
| PreventIncrementOfMergedBranchVersion | `true` |
| SourceBranches | `develop`, `release` |
| PreReleaseWeight | 55000 |

**Behavior:** Produces stable versions (no pre-release tag). Increments by patch for each new commit/merge. Prevents merged branch versions from causing extra increments.

---

### release (Priority: 90)

| Property | Value |
|----------|-------|
| Regex | `^releases?[/-]` |
| Increment | `None` |
| Tag | `"beta"` |
| IsReleaseBranch | `true` |
| PreventIncrementOfMergedBranchVersion | `true` |
| SourceBranches | `develop`, `main`, `support`, `release` |
| PreReleaseWeight | 30000 |

**Behavior:** Version comes from the branch name (e.g., `release/1.2.0` → `1.2.0-beta.1`). No increment (version is fixed by branch name). Pre-release label is "beta" with incrementing number.

---

### hotfix (Priority: 80)

| Property | Value |
|----------|-------|
| Regex | `^hotfix(es)?[/-]` |
| Increment | `Patch` |
| Tag | `"beta"` |
| SourceBranches | `release`, `main`, `support`, `hotfix` |
| PreReleaseWeight | 30000 |

**Behavior:** Patches the version (e.g., `1.2.3` → `1.2.4-beta.1`). Similar to release branches but with Patch increment.

---

### support (Priority: 70)

| Property | Value |
|----------|-------|
| Regex | `^support[/-]` |
| Increment | `Patch` |
| Tag | `""` (empty = stable) |
| IsMainline | `true` |
| PreventIncrementOfMergedBranchVersion | `true` |
| SourceBranches | `main` |
| PreReleaseWeight | 55000 |

**Behavior:** Acts like a secondary mainline for long-term support. Produces stable versions (no pre-release). Used for maintaining older major/minor versions.

---

### develop (Priority: 60)

| Property | Value |
|----------|-------|
| Regex | `^dev(elop)?(ment)?$` |
| Increment | `Minor` |
| Tag | `"alpha"` |
| TracksReleaseBranches | `true` |
| TrackMergeTarget | `true` |
| PreReleaseWeight | 0 |

**Behavior:** Produces alpha pre-releases (e.g., `1.3.0-alpha.42`). Tracks release branches so it stays ahead of them.

**Special mode logic:** If no mode is explicitly set for develop:
- If global mode is `Mainline` → develop also gets `Mainline`
- Otherwise → develop gets `ContinuousDeployment` (regardless of global setting)

This ensures every commit on develop gets a unique, auto-incrementing version.

---

### feature (Priority: 50)

| Property | Value |
|----------|-------|
| Regex | `^features?[/-]` |
| Increment | `Inherit` |
| Tag | `"{BranchName}"` |
| SourceBranches | `develop`, `main`, `release`, `feature`, `support`, `hotfix` |
| PreReleaseWeight | 30000 |

**Behavior:** Inherits increment strategy from its source branch. Pre-release label is the branch name itself (e.g., `feature/my-feature` → `1.3.0-my-feature.1`). The `{BranchName}` placeholder is replaced with the branch name part after the prefix.

---

### pull-request (Priority: 40)

| Property | Value |
|----------|-------|
| Regex | `^(pull\|pull-requests\|pr)[/-]` |
| Increment | `Inherit` |
| Tag | `"PullRequest"` |
| TagNumberPattern | `[/-](?<number>\d+)` |
| SourceBranches | `develop`, `main`, `release`, `feature`, `support`, `hotfix` |
| PreReleaseWeight | 30000 |

**Behavior:** Uses PR number extracted from branch name as part of pre-release tag (e.g., `pull/123` → `1.3.0-PullRequest0123.1`).

---

### unknown (Priority: 0) — Catch-All

| Property | Value |
|----------|-------|
| Regex | `.*` |
| Increment | `Inherit` |
| Tag | `"{BranchName}"` |
| SourceBranches | `develop`, `main`, `release`, `feature`, `support`, `hotfix` |
| PreReleaseWeight | 30000 |

**Behavior:** Catches any branch that doesn't match a known pattern. Treated like a feature branch with `{BranchName}` as the pre-release tag. Ensures every branch gets a configuration — no errors, no surprises.

---

## Priority-Based Branch Matching

When resolving which config applies to a branch:

1. All branch configs with a regex matching the branch name are collected
2. Matches are sorted by priority (descending), then by config key name (for determinism)
3. The highest-priority match is used

This avoids the "first match wins" ambiguity of regex ordering. You can add custom branches with explicit priorities:

```yaml
branches:
  staging:
    regex: ^staging$
    increment: Patch
    tag: rc
    priority: 85    # between hotfix (80) and release (90)
```

---

## Versioning Modes

### ContinuousDelivery (default)

- Pre-release number comes from tag scanning (existing tags with same label)
- Build metadata includes `CommitsSinceTag`
- Stable versions are produced only when a tag is manually applied
- Best for: teams that manually trigger releases
- Versions look like: `1.2.3-beta.1+3`

### ContinuousDeployment

- `CommitsSinceTag` is promoted to the pre-release number
- Every commit gets a unique, monotonically increasing version
- If no pre-release tag is configured, uses `continuous-delivery-fallback-tag` (default: `"ci"`)
- Best for: auto-deploying every commit
- Versions look like: `1.2.3-alpha.42`

### Mainline

- Designed for trunk-based development
- **Aggregate mode (default):** highest increment from all commits applied once, commit count in metadata
- **EachCommit mode:** version incremented per commit individually
- Pre-release tags are NOT used on mainline branches (stable versions)
- Best for: trunk-based development
- Versions look like: `1.2.3+5` (on main) or `1.3.0-feature.1` (on branches)

---

## GitFlow Workflow Example

```
main:     1.0.0 ──────────── 1.1.0 ──────────── 1.1.1 ────
              \              /                  /
develop:       \─ 1.1.0-α.1 ── 1.1.0-α.2 ──  / ── 1.2.0-α.1
                          \                   /
release/1.1.0:             1.1.0-β.1 ── 1.1.0-β.2
                                                \
hotfix/1.1.1:                                    1.1.1-β.1
```

**Flow:**
1. `develop` at `1.1.0-alpha.1` (Minor increment from `1.0.0`, alpha label)
2. Create `release/1.1.0` → version from branch name: `1.1.0-beta.1`
3. Merge `release/1.1.0` to `main` → `1.1.0` (stable, empty tag)
4. `develop` sees the release merge → stays ahead at `1.2.0-alpha.1`
5. `hotfix/1.1.1` from main → `1.1.1-beta.1` (Patch increment)
6. Merge hotfix to main → `1.1.1`

---

## Key Branch Configuration Properties

### IsMainline
Marks the branch as a mainline branch. Produces stable versions (empty pre-release tag). `main` and `support` are mainline by default.

### IsReleaseBranch
Enables `VersionInBranchName` strategy to extract version from the branch name. Used by `MergeMessage` strategy to identify release merges.

### TracksReleaseBranches
Makes the branch aware of active release branches. Enables `TrackReleaseBranches` strategy. Only `develop` has this by default.

### PreventIncrementOfMergedBranchVersion
When `true`, merging this branch type into another won't trigger an increment from the merge message. Used on `main` and `release` to prevent double-incrementing.

### SourceBranches
Defines which branches this branch type can be created from. Used for resolving `Inherit` increment strategy (walks up to source branch config).

### Tag (pre-release label)
- `""` (empty) → stable version, no pre-release tag
- `"alpha"`, `"beta"`, etc. → literal pre-release label
- `"{BranchName}"` → replaced with actual branch name (prefix stripped)
