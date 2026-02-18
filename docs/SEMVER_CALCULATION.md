# Semantic Version Calculation - Deep Dive

This document provides a detailed walkthrough of how GitVersion 5.12.0 calculates the semantic version, step by step. Source references are to `GitVersion/src/GitVersion.Core/`.

## Overview

The calculation happens in three main phases:
1. **Context Creation** - Gather git state
2. **Base Version Selection** - Run strategies, pick the highest candidate
3. **Version Finalization** - Apply increment, pre-release tag, and build metadata

---

## Phase 1: Context Creation

**Source:** `Core/GitVersionContextFactory.cs`

```
Input:  Git repository + config file
Output: GitVersionContext {
    CurrentBranch,
    CurrentCommit,
    FullConfiguration,
    CurrentCommitTaggedVersion,    // null if current commit has no version tag
    NumberOfUncommittedChanges,
    IsCurrentCommitTagged          // derived: CurrentCommitTaggedVersion != null
}
```

### Steps:
1. Resolve target branch (from CLI arg or current HEAD)
2. Get current commit (specific SHA or branch tip)
3. Load configuration from `.gitversion.yml` (merged with defaults)
4. If HEAD is detached, find the most appropriate branch for the commit
5. Check if current commit has a version tag (using `tag-prefix` pattern)
6. Count uncommitted changes in the working tree

---

## Phase 2: Base Version Selection

**Source:** `VersionCalculation/BaseVersionCalculator.cs`

### Step 2a: Resolve Branch Configuration

```
CurrentBranch → BranchConfigurationCalculator → BranchConfig → EffectiveConfiguration
```

The branch name is matched against configured regex patterns (e.g., `^master$|^main$` for main). The matching `BranchConfig` is merged with global `Config` to produce an `EffectiveConfiguration` with all values guaranteed non-null.

### Step 2b: Run All Version Strategies

Each strategy produces zero or more `BaseVersion` candidates:

```csharp
BaseVersion {
    Source              // human-readable description of where this version came from
    ShouldIncrement     // whether this version needs to be incremented
    SemanticVersion     // the actual version number
    BaseVersionSource   // the git commit this version was derived from (for commit counting)
    BranchNameOverride  // optional override for branch-specific labeling
}
```

**Strategies executed (in registration order):**

1. **ConfigNextVersionVersionStrategy** - From `next-version` in config
2. **TaggedCommitVersionStrategy** - From git tags
3. **MergeMessageVersionStrategy** - From merge commit messages
4. **VersionInBranchNameVersionStrategy** - From branch name (release branches)
5. **TrackReleaseBranchesVersionStrategy** - From release branches (for develop)
6. **FallbackVersionStrategy** - Returns `0.1.0` as last resort

See [VERSION_STRATEGIES.md](VERSION_STRATEGIES.md) for details on each strategy.

### Step 2c: Filter Versions

- Each candidate (except Fallback) is checked against ignore filters (`ignore` config):
  - `MinDateVersionFilter` - Excludes versions from commits before a date
  - `ShaVersionFilter` - Excludes versions from specific commit SHAs

### Step 2d: Maybe-Increment Each Candidate

Before comparing, each candidate is **tentatively incremented** using `IncrementStrategyFinder`:

```
For each BaseVersion:
    IncrementedVersion = MaybeIncrement(baseVersion, effectiveConfig)

    MaybeIncrement:
        incrementField = IncrementStrategyFinder.DetermineIncrementedField(...)
        if incrementField == None: return baseVersion.SemanticVersion
        else: return baseVersion.SemanticVersion.IncrementVersion(incrementField)
```

In **Mainline mode**, candidates with pre-release tags are filtered out.

### Step 2e: Select Maximum Version

```
maxVersion = versions.Max(v => v.IncrementedVersion)
```

**Tie-breaking:** When multiple candidates produce the same incremented version:
- Find all matching versions that have a non-null `BaseVersionSource`
- Select the one with the **oldest** `BaseVersionSource.When` (commit date)
- This ensures accurate commit counting (more commits between source and HEAD = correct count)

### Step 2f: Special Fix for Deleted Release Branches

If a merge message strategy result references a release branch that was merged and deleted:
- Rewrite the `BaseVersionSource` to be the merge base of the merge commit's parents
- This ensures correct commit counting even after the release branch is gone

---

## Phase 3: Version Finalization

**Source:** `VersionCalculation/NextVersionCalculator.cs`

### Step 3a: Handle Tagged Current Commit

If the current commit is tagged with a version:
```
taggedSemanticVersion = tag version + build metadata (CommitsSinceTag = null)
```
This tagged version is held as a candidate to potentially replace the calculated version.

### Step 3b: Branch Between Mainline and Standard Mode

**If Mainline mode** (`VersioningMode.Mainline`):
```
semver = MainlineVersionCalculator.FindMainlineModeVersion(baseVersion)
```
See [Mainline Mode](#mainline-mode-calculation) below.

**If Standard mode** (ContinuousDelivery or ContinuousDeployment):
```
if taggedVersion.Sha != baseVersion.Sha:
    semver = PerformIncrement(baseVersion, configuration)
    semver.BuildMetaData = CreateVersionBuildMetaData(baseVersion.BaseVersionSource)
else:
    semver = baseVersion.SemanticVersion    // already correct
```

### Step 3c: Apply Increment (Standard Mode)

**Source:** `NextVersionCalculator.PerformIncrement()` → `IncrementStrategyFinder.DetermineIncrementedField()`

```
incrementField = IncrementStrategyFinder.DetermineIncrementedField(context, baseVersion, config)
semver = semver.IncrementVersion(incrementField)
```

**IncrementVersion behavior:**
- If current version has NO pre-release tag:
  - `Major`: Major++, Minor=0, Patch=0
  - `Minor`: Minor++, Patch=0
  - `Patch`: Patch++
  - `None`: No change
- If current version HAS a pre-release tag with a number:
  - Increment the pre-release number instead (e.g., `1.0.0-beta.3` → `1.0.0-beta.4`)

### Step 3d: Update Pre-Release Tag

**Source:** `NextVersionCalculator.UpdatePreReleaseTag()`

Runs when:
- Branch config has a pre-release tag configured but the version doesn't have one yet, OR
- The version has a pre-release tag but it doesn't match the branch config

```
tag = config.GetBranchSpecificTag(branchName, override)
    // Replaces {BranchName} placeholder with actual branch name

if config.IsMainline && tag.IsEmpty():
    // Mainline branches get empty pre-release tag (stable version)
    semver.PreReleaseTag = ("", null)
    return

lastTag = GetVersionTagsOnBranch() matching this pre-release name
if lastTag exists && Major.Minor.Patch match && lastTag has tag:
    number = lastTag.PreReleaseTag.Number + 1
else:
    number = 1

semver.PreReleaseTag = (tag, number)
```

### Step 3e: Compare With Tagged Version

If the current commit is tagged:
```
if calculatedVersion > taggedVersion:
    // Calculated version wins, discard tagged version
    use calculatedVersion
else:
    // Tagged version wins
    taggedVersion.CommitsSinceVersionSource = calculatedVersion.CommitsSinceVersionSource
    if preReleaseTag mismatch:
        taggedVersion.PreReleaseTag = calculatedVersion.PreReleaseTag
    use taggedVersion
```

### Step 3f: Generate Build Metadata

**Source:** `MainlineVersionCalculator.CreateVersionBuildMetaData()`

```
commitLog = GetCommitLog(baseVersionSource, currentCommit)
commitsSinceTag = commitLog.Count()

BuildMetaData = {
    VersionSourceSha:           baseVersionSource.Sha
    CommitsSinceTag:            commitsSinceTag
    Branch:                     currentBranch.Name
    Sha:                        currentCommit.Sha
    ShortSha:                   currentCommit.Sha[0:7]
    CommitDate:                 currentCommit.When
    UncommittedChanges:         count
    CommitsSinceVersionSource:  commitsSinceTag
}
```

### Step 3g: Apply Versioning Mode (VariableProvider)

**Source:** `VersionCalculation/VariableProvider.cs`

The `VariableProvider` applies the final versioning mode transformations:

**ContinuousDeployment mode** (when commit is not tagged):
- Forces a pre-release tag if none exists
- Uses branch-specific tag or `ContinuousDeploymentFallbackTag` (default: "ci")
- Promotes `CommitsSinceTag` to pre-release number:
  ```
  if PreReleaseTag.Number exists:
      PreReleaseTag.Number += CommitsSinceTag - 1
  else:
      PreReleaseTag.Number = CommitsSinceTag
  CommitsSinceVersionSource = CommitsSinceTag
  CommitsSinceTag = null
  ```

**Mainline mode:**
- Also promotes commits to pre-release number (same logic)

**ContinuousDelivery mode:**
- No special transformation (version as-is)

---

## Mainline Mode Calculation

**Source:** `VersionCalculation/MainlineVersionCalculator.cs`

Mainline mode walks the commit graph from the base version source to the mainline tip, incrementing for each commit and merge.

### Algorithm:

```
1. Start with baseVersion.SemanticVersion
2. Find the mainline branch (branch marked IsMainline=true)
3. If current branch != mainline:
   a. Find the effective mainline tip (the merge point where current branch was integrated)
   b. Handle forward merges (rewind mainline tip if needed)

4. Get the commit log from baseVersionSource to mainlineTip
5. Walk commits in order:
   For each commit:
       Add to directCommits list
       If commit is a merge commit (>1 parent):
           a. IncrementForEachCommit(directCommits)    // increment for each direct commit
           b. Clear directCommits
           c. Find merged branch commits
           d. Get increment from merged commits or merge message
           e. Apply that increment to mainlineVersion

6. IncrementForEachCommit(remaining directCommits)

7. If current branch != mainline:
   Apply one more increment for "the act of branching"
   (increment determined from commit messages on the branch)

8. Set BuildMetaData from mergeBase
```

### IncrementForEachCommit:
```
For each direct commit on mainline:
    increment = GetIncrementForCommits(commit) ?? FindDefaultIncrementForBranch(mainline)
    mainlineVersion = mainlineVersion.IncrementVersion(increment)
```

### FindMessageIncrement (for merge commits):
```
Get all commits between merge base and merged head
Check commit messages for +semver: directives
If none found, try extracting increment from merge message's branch name
If still none, fallback to branch default increment
```

---

## Increment Strategy Finder - Detail

**Source:** `VersionCalculation/IncrementStrategyFinder.cs`

### Commit Message Patterns (defaults):
| Pattern | Matches | Result |
|---------|---------|--------|
| `\+semver:\s?(breaking\|major)` | `+semver: breaking`, `+semver: major` | `Major` |
| `\+semver:\s?(feature\|minor)` | `+semver: feature`, `+semver: minor` | `Minor` |
| `\+semver:\s?(fix\|patch)` | `+semver: fix`, `+semver: patch` | `Patch` |
| `\+semver:\s?(none\|skip)` | `+semver: none`, `+semver: skip` | `None` |

### DetermineIncrementedField Logic:

```
1. Find commits between baseVersionSource and currentCommit
2. Filter: only keep commits up to the latest tag (ignore older commits)
3. If CommitMessageIncrementing == MergeMessageOnly:
     Filter: only merge commits (>1 parent)
4. Scan all matching commits for bump patterns
5. Take the MAXIMUM bump found (Major > Minor > Patch > None)

6. If no commit message bump found:
     return baseVersion.ShouldIncrement ? config.Increment : None

7. If version < 1.0.0:
     Cap commit message severity to Minor (protect 0.x versions)

8. If baseVersion.ShouldIncrement and commitBump < config.Increment:
     return config.Increment  (don't go lower than branch default)

9. return commitBump
```

---

## Complete End-to-End Example

### Scenario: Feature branch off develop, 3 commits, develop had tag `1.2.0-alpha.5`

```
develop:  A---B---C (tag: 1.2.0-alpha.5)
               \
feature/foo:    D---E---F (HEAD)
```

**Context:**
- CurrentBranch = `feature/foo`
- CurrentCommit = F
- Config: default ContinuousDelivery mode
- Feature config: Increment=Inherit (inherits Minor from develop), Tag=`{BranchName}`

**Strategy Results:**
1. ConfigNextVersion: (none, no next-version configured)
2. TaggedCommit: `1.2.0-alpha.5` from commit C, ShouldIncrement=true, Source=C
3. MergeMessage: (none, no merge commits)
4. VersionInBranchName: (none, not a release branch)
5. TrackReleaseBranches: (none, feature branch doesn't track)
6. Fallback: `0.1.0` from root, ShouldIncrement=false

**Maybe-Increment:**
- TaggedCommit: `1.2.0-alpha.5` → increment Minor → `1.3.0` (pre-release tag dropped since it incremented)
- Fallback: `0.1.0` (no increment)

**Select Maximum:** `1.3.0` from TaggedCommit strategy, source=C

**Finalize:**
- PerformIncrement: already handled in MaybeIncrement comparison
- Actual increment: `1.2.0-alpha.5` → Minor → `1.3.0`
- UpdatePreReleaseTag: branch config tag = `foo` (from `{BranchName}`)
  - No existing tag matching `foo` with `1.3.0` → number = 1
  - PreReleaseTag = `foo.1`
- BuildMetaData: 3 commits since C (D, E, F)
- **Final version: `1.3.0-foo.1+3`**
