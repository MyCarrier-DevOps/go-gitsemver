# Version Strategies

GitVersion 5.12.0 uses 6 pluggable strategies to discover candidate base versions. Each strategy proposes zero or more `BaseVersion` values. The highest candidate (after tentative incrementing) wins.

All strategies are in `GitVersion/src/GitVersion.Core/VersionCalculation/BaseVersionCalculators/`.

---

## BaseVersion Structure

Every strategy returns `BaseVersion` objects:

```
BaseVersion {
    Source:             string     // Human-readable description (e.g., "Git tag 'v1.2.3'")
    ShouldIncrement:    bool       // If true, the version will be incremented before comparison
    SemanticVersion:    SemVer     // The actual version found
    BaseVersionSource:  ICommit?   // The git commit this version came from (used for commit counting)
    BranchNameOverride: string?    // Optional override for pre-release tag naming
}
```

---

## Strategy 1: ConfigNextVersionVersionStrategy

**Source:** `ConfigNextVersionVersionStrategy.cs`

**Purpose:** Allows explicitly setting the next version in the config file.

**When it produces a result:**
- `next-version` is set in `.gitversion.yml` AND current commit is NOT tagged

**BaseVersion:**
- `Source`: "NextVersion in GitVersion configuration file"
- `ShouldIncrement`: **false** (version is exact)
- `SemanticVersion`: parsed from `next-version` config value
- `BaseVersionSource`: **null** (external source, not tied to a commit)

**Notes:**
- If current commit is tagged, this strategy yields nothing (tag takes precedence)
- `next-version` accepts both `"2"` (becomes `"2.0"`) and `"2.1.0"` formats
- Use case: bootstrapping a repo or forcing a version jump

---

## Strategy 2: TaggedCommitVersionStrategy

**Source:** `TaggedCommitVersionStrategy.cs`

**Purpose:** Extracts versions from git tags on the current branch.

**Algorithm:**
1. Get all valid version tags (`GetValidVersionTags`) using `tag-prefix` regex (default: `[vV]`)
2. Filter to tags on commits that are **not newer** than the current commit
3. Group tags by commit SHA
4. Walk commits on current branch, collect matching version tags
5. For each matching tag, create a `BaseVersion`:
   - `ShouldIncrement = true` if tag commit != current commit
   - `ShouldIncrement = false` if tag IS on current commit
6. **If any tags are on the current commit**, return ONLY those (ShouldIncrement=false)
7. Otherwise return all found tags

**BaseVersion:**
- `Source`: "Git tag '{tagName}'"
- `ShouldIncrement`: `tagCommit.Sha != currentCommit.Sha`
- `SemanticVersion`: parsed from tag name
- `BaseVersionSource`: the tagged commit

**Notes:**
- This is typically the most common source of base versions
- Tag prefix is stripped before parsing (e.g., `v1.2.3` → `1.2.3`)
- Tags on current commit result in that exact version being used (no increment)

---

## Strategy 3: MergeMessageVersionStrategy

**Source:** `MergeMessageVersionStrategy.cs`

**Purpose:** Extracts versions from merge commit messages (e.g., "Merge branch 'release/1.2.0'").

**Algorithm:**
1. Get all commits on the current branch prior to the current commit's date
2. For each commit that is a merge commit (>1 parent):
   a. Parse the commit message using `MergeMessage` parser
   b. If a version is found AND the merged branch is a release branch:
      - Create a BaseVersion
3. Take at most 5 results

**Merge Message Formats Supported:**

| Format | Pattern Example |
|--------|----------------|
| Default | `Merge branch 'release/1.2.0'` |
| SmartGit | `Finish release/1.2.0` |
| BitBucketPull | `Merged in release/1.2.0 (pull request #123)` |
| BitBucketPullv7 | `Pull request #123: release/1.2.0` |
| GitHubPull | `Merge pull request #123 from release/1.2.0` |
| RemoteTracking | `Merge remote-tracking branch 'origin/release/1.2.0'` |

Custom formats can be added via `merge-message-formats` in config.

**BaseVersion:**
- `Source`: "Merge message '{commit message}'"
- `ShouldIncrement`: `!config.PreventIncrementOfMergedBranchVersion`
- `SemanticVersion`: extracted from branch name in merge message
- `BaseVersionSource`: the merge commit

---

## Strategy 4: VersionInBranchNameVersionStrategy

**Source:** `VersionInBranchNameVersionStrategy.cs`

**Purpose:** Extracts version from the branch name itself (for release branches).

**Algorithm:**
1. Check if the current branch matches a release branch regex
2. If yes, split the branch name by `/` and `-`
3. Try to parse each part as a semantic version
4. If found, find the commit where this branch was created from its parent

**Examples:**
- `release/1.2.0` → `1.2.0`
- `releases/1.2.0` → `1.2.0`
- `release-1.2.0` → `1.2.0`

**BaseVersion:**
- `Source`: "Version in branch name"
- `ShouldIncrement`: **false** (version is exact from branch name)
- `SemanticVersion`: parsed from branch name
- `BaseVersionSource`: commit where branch was created
- `BranchNameOverride`: branch name with version stripped (for pre-release labeling)

**Notes:**
- Only active for branches matching `IsReleaseBranch = true` configuration
- Commonly used in GitFlow where release branches are named `release/X.Y.Z`

---

## Strategy 5: TrackReleaseBranchesVersionStrategy

**Source:** `TrackReleaseBranchesVersionStrategy.cs`

**Purpose:** For branches that track release branches (like `develop` in GitFlow), finds versions from active release branches and main branch tags.

**When active:** Only when `TracksReleaseBranches = true` in the branch config (default for `develop`).

**Algorithm (two sub-strategies merged):**

**Sub-strategy A - Release Branch Versions:**
1. Find all branches matching release branch config
2. For each release branch:
   a. Find the merge base between it and the current branch
   b. If merge base == current commit, ignore (branch has no own commits)
   c. Use `VersionInBranchNameVersionStrategy` to extract version from release branch name
   d. Set `ShouldIncrement = true` and `BaseVersionSource = merge base`

**Sub-strategy B - Main Branch Tags:**
1. Find the main branch
2. Use `TaggedCommitVersionStrategy` to get all tagged versions on main
3. Return them as candidates

**Use case:** When on `develop`, this finds the version from any active `release/X.Y.Z` branch and also considers tags on `main`, ensuring `develop` stays ahead.

---

## Strategy 6: FallbackVersionStrategy

**Source:** `FallbackVersionStrategy.cs`

**Purpose:** Provides a baseline version when no other strategy produces a result.

**Always returns:**
- `Source`: "Fallback base version"
- `ShouldIncrement`: **false**
- `SemanticVersion`: `0.1.0`
- `BaseVersionSource`: root commit (the very first commit reachable from current branch tip)

**Notes:**
- This is the last resort, used only when there are no tags, no merge messages, no config, and no branch name version
- The root commit as source ensures commit counting starts from the beginning of history
- Always bypasses ignore filters (checked via `strategy is FallbackVersionStrategy` in BaseVersionCalculator)

---

## Strategy Selection Process

**Source:** `BaseVersionCalculator.cs`

```
1. Run all 6 strategies → collect all BaseVersion candidates
2. Apply ignore filters to each (except Fallback)
3. For each candidate, compute IncrementedVersion = MaybeIncrement(candidate)
4. In Mainline mode: filter out candidates with pre-release tags
5. Select the maximum IncrementedVersion
6. Tie-break: if multiple candidates produce the same IncrementedVersion,
   pick the one with the OLDEST BaseVersionSource (for accurate commit counting)
7. Return the winning BaseVersion + EffectiveBranchConfiguration
```

### Why Oldest Source on Tie?

If two strategies both produce `1.3.0` after incrementing, the one with the older source commit will have more commits between it and HEAD. This gives a more accurate `CommitsSinceVersionSource` count in the build metadata.

---

## Strategy Registration

Strategies are registered via dependency injection in `GitVersionCoreModule.cs`. The order of registration determines the iteration order, but since the maximum is selected, order only matters for logging/debugging.
