# Branch Workflows and Default Branch Configuration

GitVersion 5.12.0 supports different branching strategies through per-branch configuration. This document details the default configuration for each branch type and how the two main workflows differ.

Source: `GitVersion/src/GitVersion.Core/Configuration/ConfigurationBuilder.cs`

---

## Default Branch Configurations

### main (or master)

| Property | Value |
|----------|-------|
| Regex | `^master$\|^main$` |
| Increment | `Patch` |
| VersioningMode | ContinuousDelivery (inherited from global default) |
| Tag (pre-release label) | `""` (empty = stable, no pre-release) |
| IsMainline | `true` |
| IsReleaseBranch | `false` |
| TracksReleaseBranches | `false` |
| PreventIncrementOfMergedBranchVersion | `true` |
| TrackMergeTarget | `false` |
| SourceBranches | `develop`, `release` |
| PreReleaseWeight | 55000 |

**Behavior:** Produces stable versions (no pre-release tag). Increments by patch for each new commit/merge. Prevents merged branch versions from causing extra increments.

---

### develop

| Property | Value |
|----------|-------|
| Regex | `^dev(elop)?(ment)?$` |
| Increment | `Minor` |
| VersioningMode | ContinuousDeployment (special: overridden unless global is Mainline) |
| Tag (pre-release label) | `"alpha"` |
| IsMainline | `false` |
| IsReleaseBranch | `false` |
| TracksReleaseBranches | `true` |
| PreventIncrementOfMergedBranchVersion | `false` |
| TrackMergeTarget | `true` |
| SourceBranches | (empty) |
| PreReleaseWeight | 0 |

**Behavior:** Produces alpha pre-releases (e.g., `1.3.0-alpha.42`). Tracks release branches so it stays ahead of them. Uses ContinuousDeployment mode by default, meaning `CommitsSinceTag` is promoted to the pre-release number.

**Special VersioningMode logic:** If global mode is Mainline, develop also gets Mainline. Otherwise, develop always gets ContinuousDeployment regardless of global setting. (Source: `ConfigurationBuilder.FinalizeBranchConfiguration()`)

---

### release

| Property | Value |
|----------|-------|
| Regex | `^releases?[/-]` |
| Increment | `None` |
| VersioningMode | ContinuousDelivery (inherited) |
| Tag (pre-release label) | `"beta"` |
| IsMainline | `false` |
| IsReleaseBranch | `true` |
| TracksReleaseBranches | `false` |
| PreventIncrementOfMergedBranchVersion | `true` |
| TrackMergeTarget | `false` |
| SourceBranches | `develop`, `main`, `support`, `release` |
| PreReleaseWeight | 30000 |

**Behavior:** Version comes from the branch name (e.g., `release/1.2.0` → `1.2.0-beta.1`). No increment (version is fixed by branch name). Pre-release label is "beta" with incrementing number.

---

### feature

| Property | Value |
|----------|-------|
| Regex | `^features?[/-]` |
| Increment | `Inherit` |
| VersioningMode | ContinuousDelivery (inherited) |
| Tag (pre-release label) | `"{BranchName}"` |
| IsMainline | `false` |
| IsReleaseBranch | `false` |
| TracksReleaseBranches | `false` |
| PreventIncrementOfMergedBranchVersion | (default) |
| TrackMergeTarget | (default) |
| SourceBranches | `develop`, `main`, `release`, `feature`, `support`, `hotfix` |
| PreReleaseWeight | 30000 |

**Behavior:** Inherits increment strategy from its source branch. Pre-release label is the branch name itself (e.g., `feature/my-feature` → `1.3.0-my-feature.1`). The `{BranchName}` placeholder is replaced with the branch name part after the prefix.

---

### hotfix

| Property | Value |
|----------|-------|
| Regex | `^hotfix(es)?[/-]` |
| Increment | `Patch` |
| VersioningMode | ContinuousDelivery (inherited) |
| Tag (pre-release label) | `"beta"` |
| IsMainline | `false` |
| IsReleaseBranch | `false` |
| TracksReleaseBranches | `false` |
| PreventIncrementOfMergedBranchVersion | `false` |
| TrackMergeTarget | `false` |
| SourceBranches | `release`, `main`, `support`, `hotfix` |
| PreReleaseWeight | 30000 |

**Behavior:** Patches the version (e.g., `1.2.3` → `1.2.4-beta.1`). Similar to release branches but with Patch increment.

---

### pull-request

| Property | Value |
|----------|-------|
| Regex | `^(pull\|pull\-requests\|pr)[/-]` |
| Increment | `Inherit` |
| VersioningMode | ContinuousDelivery (inherited) |
| Tag (pre-release label) | `"PullRequest"` |
| TagNumberPattern | `[/-](?<number>\d+)` |
| IsMainline | `false` |
| IsReleaseBranch | `false` |
| SourceBranches | `develop`, `main`, `release`, `feature`, `support`, `hotfix` |
| PreReleaseWeight | 30000 |

**Behavior:** Uses PR number extracted from branch name as part of pre-release tag (e.g., `pull/123` → `1.3.0-PullRequest0123.1`). TagNumberPattern extracts the PR number, which gets padded and appended to the label.

---

### support

| Property | Value |
|----------|-------|
| Regex | `^support[/-]` |
| Increment | `Patch` |
| VersioningMode | ContinuousDelivery (inherited) |
| Tag (pre-release label) | `""` (empty = stable) |
| IsMainline | `true` |
| IsReleaseBranch | `false` |
| TracksReleaseBranches | `false` |
| PreventIncrementOfMergedBranchVersion | `true` |
| TrackMergeTarget | `false` |
| SourceBranches | `main` |
| PreReleaseWeight | 55000 |

**Behavior:** Acts like a secondary mainline for long-term support. Produces stable versions (no pre-release). Used for maintaining older major/minor versions.

---

## Versioning Modes

### ContinuousDelivery (default)

- Version stays as-is from calculation
- Pre-release number comes from tag scanning (finding existing tags with same label)
- Build metadata includes `CommitsSinceTag`
- Best for: teams that manually trigger releases
- Versions look like: `1.2.3-beta.1+3`

### ContinuousDeployment

- `CommitsSinceTag` is promoted to the pre-release number
- Every commit gets a unique, incrementing pre-release number
- If no pre-release tag is configured, falls back to `ContinuousDeploymentFallbackTag` (default: `"ci"`)
- Best for: auto-deploying every commit
- Versions look like: `1.2.3-alpha.42`

### Mainline

- Each commit on mainline increments the version
- Merge commits increment based on the merged branch's commit messages
- Direct commits on mainline each get their own increment
- Branches off mainline get one additional increment for "the act of branching"
- Pre-release tags are NOT supported on mainline branches
- Best for: trunk-based development
- Versions look like: `1.2.3` (on main) or `1.3.0-feature.1` (on branches)

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

## Key Branch Configuration Properties Explained

### IsMainline
Marks the branch as a mainline branch. Used by `MainlineVersionCalculator` to find the trunk. Produces stable versions (empty pre-release tag). `main` and `support` branches are mainline by default.

### IsReleaseBranch
Marks the branch as a release branch. Enables `VersionInBranchNameVersionStrategy` to extract version from the branch name. Used by `MergeMessageVersionStrategy` to identify release merges.

### TracksReleaseBranches
Makes the branch "aware" of active release branches. Enables `TrackReleaseBranchesVersionStrategy` which considers both release branch versions and main branch tags. Only `develop` has this by default.

### PreventIncrementOfMergedBranchVersion
When `true`, merging this branch type into another won't trigger an increment from the merge message. Used on `main` and `release` to prevent double-incrementing when merging releases.

### TrackMergeTarget
When `true`, the branch considers the merge target's version. Only `develop` has this by default.

### SourceBranches
Defines which branches this branch type can be created from. Used for:
1. Resolving `Inherit` increment strategy (walks up to source branch config)
2. Finding the commit where the branch was created (merge base with source)

### Tag (pre-release label)
- `""` (empty string) → stable version, no pre-release tag
- `"alpha"`, `"beta"`, etc. → literal pre-release label
- `"{BranchName}"` → replaced with actual branch name
- `"useBranchName"` → uses full branch name as label
