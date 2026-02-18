# Git Analysis - How Git Data Is Used

This document details exactly what git information GitVersion 5.12.0 reads and how it uses each piece of data.

Source: `GitVersion/src/GitVersion.Core/Core/RepositoryStore.cs`, `MergeBaseFinder.cs`, `GitVersionContextFactory.cs`

---

## Git Data Consumed

### 1. Tags

**How collected:** `RepositoryStore.GetValidVersionTags(tagPrefix, olderThan)`

- All tags in the repository are enumerated
- Each tag name is matched against the `tag-prefix` regex (default: `[vV]`)
- The prefix is stripped and the remainder is parsed as a semantic version
- Tags newer than `olderThan` (current commit date) are filtered out
- Tags on commits matching `ignore` config (SHA list or date filter) are excluded

**Used by:**
- `TaggedCommitVersionStrategy` - Primary source of base versions
- `TrackReleaseBranchesVersionStrategy` - Tags on main branch
- `IncrementStrategyFinder` - Tags used to limit commit message scanning window
- `GitVersionContextFactory` - Determines if current commit is tagged
- `NextVersionCalculator.UpdatePreReleaseTag()` - Finds last matching pre-release tag for number incrementing

**Key behavior:**
- If a tag is on the **current commit**, the tagged version is used directly (no increment)
- Multiple tags on the same commit: the highest semantic version wins
- Tag commit date is used for filtering, not the tag creation date

### 2. Commits

**How collected:** Multiple methods in `RepositoryStore`

**Current commit:** `GetCurrentCommit(branch, commitId)`
- From specific SHA (if provided via CLI) or branch tip

**Commit history:** `GetCommitLog(baseVersionSource, headCommit)`
- Linear count of commits between two points
- Used for `CommitsSinceTag` / `CommitsSinceVersionSource` in build metadata

**Reachable commits:** `GetCommitsReacheableFromHead(repo, headCommit)`
- Topological + reverse sorted (oldest first)
- Used by `IncrementStrategyFinder` to build the commit map for scanning

**Intermediate commits:** Commits between baseVersionSource and currentCommit
- Used for `+semver:` directive scanning

**Mainline commit log:** `GetMainlineCommitLog(baseVersionSource, mainlineTip)`
- Direct commits on mainline (first-parent walk)
- Used by `MainlineVersionCalculator` for per-commit incrementing

**Commit properties used:**
| Property | Usage |
|----------|-------|
| `Sha` | Unique identification, matching against tags, ignore filters |
| `Id.ToString(7)` | ShortSha in build metadata |
| `Message` | Scanned for `+semver:` directives and merge patterns |
| `Parents` | Detecting merge commits (count > 1), traversing merge graph |
| `When` | Commit date for filtering, tie-breaking, CommitDate output |

### 3. Branches

**How collected:** `RepositoryStore` methods

**Current branch:** `GetTargetBranch(targetBranchName)`
- Resolved from CLI argument or current HEAD
- If detached HEAD: `GetBranchesContainingCommit()` finds the best branch

**Branch properties used:**
| Property | Usage |
|----------|-------|
| `Name.Friendly` | Matched against branch config regex patterns |
| `Name.WithoutRemote` | Stripped of `origin/` prefix for matching |
| `Tip` | Latest commit on the branch |
| `Commits` | Commit enumeration for tag scanning |
| `IsDetachedHead` | Triggers branch detection from commit |
| `IsRemote` | Remote branch handling |

**Branch queries:**
- `FindMainBranch(config)` - Finds the branch matching main config (regex `^master$|^main$`)
- `GetReleaseBranches(config)` - All branches matching release regex
- `GetMainlineBranches(commit, config)` - Branches marked `IsMainline=true` containing the given commit
- `GetBranchesContainingCommit(commit)` - All branches reachable from a commit

### 4. Merge History

**Merge base:** `FindMergeBase(branch1, branch2)` or `FindMergeBase(commit1, commit2)`
- The common ancestor of two branches/commits
- Cached by pair of branch names

**Used for:**
- Determining where a branch was created from its parent
- Counting commits on a specific branch (between merge base and tip)
- `TrackReleaseBranchesVersionStrategy` - finding the fork point
- `MainlineVersionCalculator` - finding effective mainline tip for non-mainline branches
- `BaseVersionCalculator` - fixing merge message strategy source when release branch is deleted

**Merge commit analysis:**
- A commit with >1 parent is a merge commit
- `Parents.First()` = mainline parent (the branch being merged INTO)
- `Parents.Skip(1).First()` = merged head (the branch being merged FROM)
- Merge commits trigger `AggregateMergeCommitIncrement` in mainline mode

### 5. Uncommitted Changes

**How collected:** `GetNumberOfUncommittedChanges()`

- Counts files in the working tree that differ from HEAD
- Included in build metadata as `UncommittedChanges`
- Does NOT affect version calculation, only metadata

---

## Git Operations Flow

### On Application Start

```
1. Open repository at working directory (or specified path)
2. Resolve current branch (CLI override or HEAD)
3. Get current commit
4. Load all tags → parse as semantic versions
5. Check if current commit has a version tag
6. Count uncommitted changes
7. → GitVersionContext ready
```

### During Base Version Calculation

```
1. Match current branch name against all branch config regexes
2. For TaggedCommitVersionStrategy:
   a. Get all version tags
   b. Group by commit SHA
   c. Walk branch commits, match against tag groups
   d. Produce BaseVersions

3. For MergeMessageVersionStrategy:
   a. Get commits on branch prior to current commit date
   b. Filter to merge commits (>1 parent)
   c. Parse merge message with 6+ format patterns
   d. Check if merged branch matches release branch regex
   e. Produce BaseVersions

4. For VersionInBranchNameVersionStrategy:
   a. Check if current branch matches release branch regex
   b. Split branch name by / and -
   c. Try parsing each part as semver
   d. Find branch creation commit (merge base with source)

5. For TrackReleaseBranchesVersionStrategy:
   a. Find all release branches
   b. For each: find merge base with current branch
   c. Also get tags from main branch
   d. Combine results

6. For FallbackVersionStrategy:
   a. Walk to root commit (first commit reachable)
```

### During Increment Determination

```
1. Get all commits between baseVersionSource and currentCommit
2. Get all tags → build SHA set
3. Reverse commit list, take commits until hitting a tag
   (only scan commits since latest tag, per issue #3071)
4. If MergeMessageOnly mode: filter to merge commits
5. Scan each commit message against 4 regex patterns:
   major → minor → patch → none
6. Return highest match found
```

### During Mainline Calculation

```
1. Find mainline branch (IsMainline=true, containing currentCommit)
2. If current branch != mainline:
   a. Find merge base between current and mainline
   b. Find effective mainline tip (commit where merge base was integrated)
   c. Detect and handle forward merges
3. Get commit log from baseVersionSource to mainlineTip
4. Walk each commit:
   - Direct commits: queue for batch increment
   - Merge commits:
     a. Increment for queued direct commits
     b. Find merged branch commits (merge base to merged head)
     c. Scan those commits for +semver: directives
     d. Increment mainline version
5. Increment for remaining direct commits
6. If not on mainline: one more increment for branching
```

---

## Merge Message Parsing

**Source:** `GitVersion/src/GitVersion.Core/Model/MergeMessage.cs`

### Built-in Patterns

| Name | Regex Pattern |
|------|--------------|
| Default | `^Merge (branch\|tag) '(?<SourceBranch>[^']*)'(?: into (?<TargetBranch>[^)]*))?\s*$` |
| SmartGit | `^Finish (?<SourceBranch>[^\s]*)\s*$` |
| BitBucketPull | `^Merged in (?<SourceBranch>[^\s]*) \(pull request \#(?<PullRequestNumber>\d+)\)` |
| BitBucketPullv7 | `^Pull request \#(?<PullRequestNumber>\d+):.*GitVersion\.SourceBranch:(?<SourceBranch>[^\s]*)` |
| GitHubPull | `^Merge pull request \#(?<PullRequestNumber>\d+) (from\|in) (?<SourceBranch>[^\s]*)` |
| RemoteTracking | `^Merge remote-tracking branch '(?<SourceBranch>[^']*)'(?: into (?<TargetBranch>[^)]*))?\s*$` |

### Version Extraction from Merge Message
1. Parse merge message → extract `SourceBranch`
2. Check if SourceBranch matches release branch regex
3. Extract version from branch name (split by `/`, `-`, parse as semver)
4. If version found → create BaseVersion from merge commit

---

## Commit Count Accuracy

Commit counting is critical for build metadata and pre-release numbering. GitVersion ensures accuracy through:

1. **Oldest source selection** - When multiple strategies produce the same version, the oldest `BaseVersionSource` is used, ensuring the longest possible commit chain for counting
2. **Tag-limited scanning** - Commit message scanning stops at the most recent tag to avoid re-counting
3. **Deleted branch fixup** - When a release branch merge message points to a deleted branch, the source is rewritten to the merge base
4. **Mainline per-commit tracking** - In mainline mode, each commit is individually tracked and incremented
