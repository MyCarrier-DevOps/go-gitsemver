# Version Strategies

gitsemver uses 6 pluggable strategies to discover candidate base versions. Each strategy proposes zero or more `BaseVersion` candidates. The highest candidate (after computing an effective version) wins.

All strategies implement the `VersionStrategy` interface in `internal/strategy/`.

---

## BaseVersion Structure

Every strategy returns `BaseVersion` objects:

```
BaseVersion {
    Source:             string          // Human-readable description
    ShouldIncrement:    bool            // If true, version is incremented before comparison
    SemanticVersion:    SemanticVersion  // The actual version found
    BaseVersionSource:  *Commit         // Git commit this version came from
    BranchNameOverride: string          // Optional override for pre-release tag naming
}
```

---

## Strategy 1: ConfigNextVersion

**Source:** `internal/strategy/confignextversion.go`

Uses the `next-version` field from `gitsemver.yml` as the base version directly — no incrementing.

**When it produces a result:**
- `next-version` is set in config AND current commit is NOT tagged

**BaseVersion:**
- `ShouldIncrement`: **false** (version is exact)
- `BaseVersionSource`: **nil** (external source, not tied to a commit)

**Use case:** Bootstrapping a repo or forcing a version jump.

---

## Strategy 2: TaggedCommit

**Source:** `internal/strategy/taggedcommit.go`

Extracts versions from git tags on the current branch using the `tag-prefix` regex (default: `[vV]`).

**Algorithm:**
1. Get all valid version tags matching the tag prefix
2. Walk commits on the current branch, collect matching tags
3. If any tags are on the current commit, return only those (`ShouldIncrement = false`)
4. Otherwise return all found tags (`ShouldIncrement = true`)

**Key behavior:**
- Tag on current commit → that exact version is used (no increment)
- Multiple tags on the same commit → highest semantic version wins
- This is typically the most common source of base versions

---

## Strategy 3: MergeMessage

**Source:** `internal/strategy/mergemessage.go`

Extracts version information from merge commit messages and squash merge messages.

**Built-in formats (8 total):**

| Format | Pattern Example |
|--------|----------------|
| Default | `Merge branch 'release/1.2.0'` |
| SmartGit | `Finish release/1.2.0` |
| GitHub Pull | `Merge pull request #123 from release/1.2.0` |
| Bitbucket Pull | `Merged in release/1.2.0 (pull request #123)` |
| Bitbucket Pull v7 | `Pull request #123: release/1.2.0` |
| Remote Tracking | `Merge remote-tracking branch 'origin/release/1.2.0'` |
| GitHub Squash | `feat: add login page (#123)` |
| Bitbucket Squash | `Merged in feature/auth (pull request #123)` |

Custom formats can be added via `merge-message-formats` in config.

**Squash merge support:** Unlike tools that only detect two-parent merge commits, gitsemver also parses squash merge messages (single-parent commits), which are common on GitHub and GitLab.

---

## Strategy 4: VersionInBranchName

**Source:** `internal/strategy/branchname.go`

Extracts a version number from the branch name itself. Only active for branches with `is-release-branch: true`.

**Algorithm:**
1. Check if the current branch matches a release branch config
2. Split the branch name by `/` and `-`
3. Try to parse each segment as a semantic version

**Examples:**
- `release/1.2.0` → `1.2.0`
- `releases/1.2.0` → `1.2.0`
- `release-1.2.0` → `1.2.0`

**BaseVersion:**
- `ShouldIncrement`: **false** (version is exact from branch name)
- `BranchNameOverride`: branch name with version stripped

---

## Strategy 5: TrackReleaseBranches

**Source:** `internal/strategy/trackrelease.go`

For branches that track release branches (like `develop` in GitFlow). Collects versions from active release branches and main branch tags.

**When active:** Only when `tracks-release-branches: true` in the branch config (default for `develop`).

**Two sub-strategies:**
1. **Release branch versions** — find all active release branches, extract version from their names
2. **Main branch tags** — get tagged versions on main

**Use case:** Ensures `develop` stays ahead of any active release branches.

---

## Strategy 6: Fallback

**Source:** `internal/strategy/fallback.go`

Returns the `base-version` from config (default: `0.1.0`) from the root commit. Always present as a safety net.

**BaseVersion:**
- `ShouldIncrement`: **true** (version is a starting point)
- `BaseVersionSource`: root commit (first commit reachable from branch tip)

**Note:** Always bypasses ignore filters.

---

## Strategy Selection

**Source:** `internal/calculator/baseversion.go`

After all strategies return their candidates:

1. **Filter** — Apply ignore config (SHA and date filters) to each candidate except Fallback
2. **Compute effective version** — For each candidate, tentatively apply the increment for ranking (no mutation)
3. **Select maximum** — Pick the highest effective version
4. **Tie-break** — If multiple candidates produce the same effective version, the one with the **oldest** `BaseVersionSource` commit wins (ensures accurate commit-since-tag counting)
5. **Increment once** — The winning candidate is incremented a single time

```
Candidates:
  TaggedCommit:  1.2.0 (ShouldIncrement=true)  → effective 1.3.0
  MergeMessage:  (none)
  BranchName:    (none)
  TrackRelease:  (none)
  Fallback:      0.1.0 (ShouldIncrement=true)  → effective 0.1.1

Winner: TaggedCommit (effective 1.3.0)
Final increment: Minor → 1.3.0
```
