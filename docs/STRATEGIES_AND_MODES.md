# Strategies, Modes & Manual Overrides

This document explains how gitsemver calculates versions — the strategies that discover base versions, the modes that control versioning behavior, and the manual overrides available.

---

## Table of Contents

- [Version Strategies](#version-strategies)
  - [1. ConfigNextVersion](#1-confignextversion)
  - [2. TaggedCommit](#2-taggedcommit)
  - [3. MergeMessage](#3-mergemessage)
  - [4. VersionInBranchName](#4-versioninbranchname)
  - [5. TrackReleaseBranches](#5-trackreleasebranches)
  - [6. Fallback](#6-fallback)
  - [Strategy Selection](#strategy-selection)
- [Versioning Modes](#versioning-modes)
  - [ContinuousDelivery](#continuousdelivery)
  - [ContinuousDeployment](#continuousdeployment)
  - [Mainline](#mainline)
- [Manual Overrides](#manual-overrides)
  - [Conventional Commits](#conventional-commits)
  - [Bump Directives](#bump-directives)
  - [Manual Tags](#manual-tags)
  - [next-version Config](#next-version-config)
- [Configuration Examples](#configuration-examples)

---

## Version Strategies

When gitsemver runs, it evaluates **all 6 strategies** in parallel. Each strategy returns zero or more candidate base versions. After all candidates are collected, the best one is selected and used to compute the final version.

Each candidate contains:
- **SemanticVersion** — the raw version discovered (e.g., `1.2.0`)
- **ShouldIncrement** — whether this version needs to be incremented before use
- **Source** — human-readable description of where the version came from
- **BaseVersionSource** — the git commit associated with this version

### 1. ConfigNextVersion

Reads the `next-version` field from `.gitversion.yml` and uses it as the base version directly — no incrementing.

**When it activates:** `next-version` is set in config AND the current commit is not already tagged.

**Use case:** Force a version jump without tagging. Useful when starting a new major version or bootstrapping a project.

```yaml
# .gitversion.yml
next-version: 2.0.0
```

**Example:**

```
main ── A ── B ── C    (no tags anywhere)
                  ^ HEAD

Config: next-version: 2.0.0
Result: 2.0.0
```

Once you tag a commit with `v2.0.0`, remove `next-version` from config — the TaggedCommit strategy takes over from there.

**Candidate returned:**
| Field | Value |
|-------|-------|
| SemanticVersion | `2.0.0` |
| ShouldIncrement | `false` |
| Source | `"next-version in config"` |

---

### 2. TaggedCommit

Scans the current branch for git tags that match the configured `tag-prefix` pattern (default: `[vV]`). Each valid semver tag produces a candidate.

**When it activates:** There is at least one version tag reachable from the current branch.

**Use case:** This is the primary strategy for most workflows. Tags are the source of truth for released versions.

```yaml
# .gitversion.yml
tag-prefix: '[vV]'   # default — matches v1.0.0 and V1.0.0
```

**Example — tag on current commit:**

```
main ── A ── B ── C (tag: v1.2.0)
                  ^ HEAD

Result: 1.2.0   (ShouldIncrement = false, tag is on HEAD)
```

**Example — tag in history:**

```
main ── A (tag: v1.2.0) ── B ── C ── D
                                      ^ HEAD (3 commits since tag)

Base: 1.2.0 (ShouldIncrement = true)
Branch config increment: Patch
Result: 1.2.1 (with +3 in build metadata)
```

**Example — multiple tags, highest wins:**

```
main ── A (tag: v1.0.0) ── B (tag: v1.2.0) ── C ── D
                                                     ^ HEAD

Base: 1.2.0 (the higher version is selected)
```

**Custom tag prefix:**

```yaml
tag-prefix: 'release-'   # matches tags like release-1.0.0
```

---

### 3. MergeMessage

Extracts version information from merge commit messages. Supports both real merge commits (two parents) and squash merges (single parent).

**When it activates:** A merge commit or squash merge message is found in the commit history since the last version tag.

**Use case:** Captures version bumps from merged branches. Particularly useful when release branches carry version information.

**Built-in merge message formats:**

| Format | Pattern Example |
|--------|----------------|
| Default | `Merge branch 'release/1.2.0' into main` |
| SmartGit | `Finish release/1.2.0` |
| GitHub Pull | `Merge pull request #123 from release/1.2.0` |
| Bitbucket Pull | `Merged in release/1.2.0 (pull request #123)` |
| GitLab | `Merge branch 'release/1.2.0' into 'main'` |
| Remote Tracking | `Merge remote-tracking branch 'origin/release/1.2.0'` |

**Squash merge formats (DI-8):**

| Source | Pattern Example |
|--------|----------------|
| GitHub squash | `feat: add login page (#123)` |
| GitLab squash | `Merge branch 'feature/auth' into 'main'` |
| Bitbucket squash | `Merged in feature/auth (pull request #123)` |

**Custom merge message format:**

```yaml
merge-message-formats:
  azure-devops: '^Merged PR (?<PullRequestNumber>\d+): .*$'
```

**Example — release branch merge:**

```
main ── A (v1.0.0) ── M ── B
                      ^ merge commit: "Merge branch 'release/1.2.0' into main"

Strategy extracts: 1.2.0 from the merge message
Base: 1.2.0 (ShouldIncrement = false)
Result: 1.2.0
```

**Example — squash merge with PR number:**

```
main ── A (v1.0.0) ── S ── B
                      ^ squash commit: "feat: add auth (#42)"

Strategy extracts PR #42, identifies it was from a feature branch
Increment determined by commit message convention (feat: → Minor)
```

---

### 4. VersionInBranchName

Extracts a version number from the branch name itself. Only active for branches marked with `is-release-branch: true`.

**When it activates:** Current branch matches a branch config with `is-release-branch: true`, AND the branch name contains a parseable version.

**Use case:** Release branches named `release/1.2.0` carry their target version in the name.

```yaml
branches:
  release:
    regex: ^releases?[/-]
    is-release-branch: true
```

**Example:**

```
release/1.3.0 ── A ── B ── C
                            ^ HEAD

Branch name parsed: 1.3.0
Base: 1.3.0 (ShouldIncrement = false)
Pre-release tag: beta (from release branch config)
Result: 1.3.0-beta.1
```

**Branch name parsing rules:**
- Splits on `/` and `-`
- Tries to parse each segment and combination as semver
- `release/1.3.0` → `1.3.0`
- `release/1.3` → `1.3.0`
- `releases/v2` → `2.0.0`

---

### 5. TrackReleaseBranches

For branches that are aware of active release branches (like `develop` in GitFlow). Collects versions from both active release branches and the main branch.

**When it activates:** Current branch has `tracks-release-branches: true` in its config.

**Use case:** The `develop` branch needs to stay ahead of any active release branches. If `release/1.3.0` exists, develop should produce `1.4.0-alpha.X` (not `1.3.0-alpha.X`).

```yaml
branches:
  develop:
    regex: ^dev(elop)?(ment)?$
    tracks-release-branches: true
    source-branches: [main, release, hotfix, support]
```

**Example — develop tracking a release branch:**

```
main:          ── A (v1.0.0) ──────────────────
develop:          \── B ── C ── D ── E
release/1.1.0:            \── R1 ── R2

On develop (HEAD = E):
  TaggedCommit candidate: 1.0.0 → effective 1.1.0 (Minor increment)
  TrackRelease candidate: 1.1.0 (from release/1.1.0 branch name)

  TrackRelease wins (higher effective version)
  develop must be AHEAD of release: 1.2.0-alpha.1
```

**Example — develop after release merged to main:**

```
main:    ── A (v1.0.0) ── M (v1.1.0) ──
develop:     \── B ── C ──/── D ── E

On develop (HEAD = E):
  TaggedCommit: v1.1.0 on main → effective 1.2.0 (Minor increment)
  No active release branches
  Result: 1.2.0-alpha.2
```

---

### 6. Fallback

Returns `0.1.0` from the root commit. Always present as a safety net.

**When it activates:** Always. Provides a baseline if no other strategy produces a result.

**Use case:** Brand-new repositories with no tags.

**Example:**

```
main ── A ── B ── C    (no tags, no config)
                  ^ HEAD

All other strategies return nothing.
Fallback: 0.1.0 (ShouldIncrement = true)
Branch config: Patch increment
Result: 0.1.1 (on main) or 0.1.1-alpha.1 (on develop)
```

---

### Strategy Selection

After all strategies return their candidates:

1. **Compute effective version** for each candidate: if `ShouldIncrement`, tentatively apply the increment to get the effective version (this is for ranking only — no mutation).
2. **Select the maximum** effective version.
3. **Tie-break:** If two candidates have the same effective version, the one with the **oldest** `BaseVersionSource` commit wins. This ensures accurate commit-since-tag counting.
4. **Increment once** — the winning candidate is incremented a single time (DI-3: no double-increment).

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

---

## Versioning Modes

The versioning mode controls how the final version is assembled after the base version and increment are determined. Each branch can have its own mode.

### ContinuousDelivery

**The default mode.** Pre-release versions track the branch; stable versions are produced only when a tag is manually applied.

**How it works:**
1. Base version found (e.g., `1.2.0` from tag)
2. Increment applied (e.g., Minor → `1.3.0`)
3. Pre-release label from branch config (e.g., `alpha` for develop)
4. Pre-release number from tag scanning (how many tags exist for this pre-release series)
5. Build metadata: commits since tag

**On main/support (tag = empty):** Produces stable versions. You tag manually to release.

```
main:    v1.0.0 ── A ── B ── C
                               ^ HEAD

Version: 1.0.1+3   (Patch increment, no pre-release, 3 commits since tag)
                     Not released until you: git tag v1.0.1
```

**On develop (tag = alpha):**

```
develop: (from v1.0.0) ── A ── B ── C ── D
                                          ^ HEAD

Version: 1.1.0-alpha.1+4
         ├─ Minor increment (develop default)
         ├─ alpha label (develop config)
         ├─ .1 (no existing alpha tags for 1.1.0)
         └─ +4 (4 commits since version source)
```

**On feature (tag = {BranchName}):**

```
feature/login: (from develop) ── A ── B
                                       ^ HEAD

Version: 1.1.0-login.1+2
         ├─ Inherited increment (from develop: Minor)
         ├─ login (branch name with feature/ prefix stripped)
         └─ .1+2
```

**Config:** [examples/gitflow.yml](examples/gitflow.yml)

---

### ContinuousDeployment

Every commit gets a unique, monotonically increasing version. The commit count since the last tag is promoted into the pre-release number, making each version deployable.

**How it works:**
1. Base version found and incremented (same as ContinuousDelivery)
2. CommitsSinceTag is promoted to the pre-release number (DI-4)
3. If the branch has an empty tag (stable), the `continuous-delivery-fallback-tag` (default: `ci`) is used as the pre-release label

**On main:**

```
main:    v1.0.0 ── A ── B ── C
                               ^ HEAD

Version: 1.0.1-ci.3
         ├─ Patch increment
         ├─ ci (fallback tag since main has empty tag)
         └─ .3 (promoted from commits-since-tag)

Each commit gets a unique, incrementing version:
  A: 1.0.1-ci.1
  B: 1.0.1-ci.2
  C: 1.0.1-ci.3
```

**On develop:**

```
develop: (from v1.0.0) ── A ── B ── C
                                     ^ HEAD

Version: 1.1.0-alpha.3
         ├─ Minor increment
         ├─ alpha (develop config)
         └─ .3 (promoted from commits-since-tag)
```

**Key difference from ContinuousDelivery:** In CD mode, the pre-release number comes from commits-since-tag (always increasing). In ContinuousDelivery, it comes from scanning existing tags for that pre-release series.

**Config:** [examples/continuous-deployment.yml](examples/continuous-deployment.yml)

---

### Mainline

Designed for trunk-based development. The highest increment from all commits since the last tag is applied **once**. Commit count goes into build metadata for uniqueness.

**How it works (DI-10: Aggregate Increment):**
1. Find the latest semver tag (e.g., `v1.2.0`)
2. Collect all commits since that tag
3. Scan commit messages for the **single highest** increment type (via Conventional Commits / bump directives)
4. Apply that increment **once** → `1.3.0`
5. Commit count since tag goes into build metadata → `1.3.0+5`

**On main (mainline branch):**

```
main:    v1.0.0 ── fix ── fix ── feat ── fix ── fix
                                                  ^ HEAD (5 commits)

Commits scanned:
  "fix: resolve null pointer"     → Patch
  "fix: handle empty input"       → Patch
  "feat: add user profiles"       → Minor  ← highest
  "fix: profile validation"       → Patch
  "fix: edge case in profiles"    → Patch

Highest increment: Minor (applied ONCE)
Result: 1.1.0+5
         ├─ Minor applied once (not 5 times)
         └─ +5 (commit count in metadata for uniqueness)
```

**Contrast with per-commit incrementing (what we DON'T do):**

```
Per-commit (wrong):  1.0.1 → 1.0.2 → 1.1.0 → 1.1.1 → 1.1.2
Aggregate (correct): 1.1.0+5

The aggregate approach keeps version numbers semantically meaningful.
```

**On feature branches in Mainline mode:**

Feature branches still get pre-release labels:

```
feature/auth: (from main v1.0.0) ── A ── B ── C
                                              ^ HEAD

Version: 1.1.0-auth.1+3
         ├─ Increment inherited from scanning commits
         ├─ auth (branch name label)
         └─ .1+3
```

**Forcing a version jump:**

```
main:    v1.0.0 ── "bump major" ── fix ── fix
                                           ^ HEAD

Highest increment: Major (from "bump major")
Result: 2.0.0+3
```

Or tag manually:

```
main:    v1.0.0 ── A ── B ── v2.0.0 ── C ── D
                                             ^ HEAD

TaggedCommit: v2.0.0 (2 commits ago)
Highest increment from C, D: Patch
Result: 2.0.1+2
```

**Per-branch mode override:**

Mainline mode can be set globally or per-branch. This lets you mix modes — for example, Mainline on `main` but ContinuousDelivery on `develop` and `release`:

```yaml
mode: Mainline   # global default

branches:
  main:
    # inherits Mainline from global
    tag: ''
    is-mainline: true

  develop:
    mode: ContinuousDelivery   # override for this branch
    tag: alpha

  release:
    mode: ContinuousDelivery   # override for this branch
    tag: beta
```

**Config:** [examples/mainline.yml](examples/mainline.yml), [examples/trunk-based.yml](examples/trunk-based.yml)

---

## Manual Overrides

gitsemver provides several ways to manually control version bumps.

### Conventional Commits

First-class support for the [Conventional Commits](https://www.conventionalcommits.org/) specification. gitsemver scans commit messages for structured prefixes.

**Increment rules:**

| Commit Pattern | Increment |
|----------------|-----------|
| `feat:` or `feat(scope):` | Minor |
| `fix:` or `fix(scope):` | Patch |
| `feat!:` or `fix!:` (any type with `!`) | Major |
| `BREAKING CHANGE:` in commit footer | Major |

**Examples:**

```bash
# Minor bump
git commit -m "feat: add user authentication"
git commit -m "feat(auth): add OAuth2 support"

# Patch bump
git commit -m "fix: resolve null pointer in login"
git commit -m "fix(api): handle empty response body"

# Major bump
git commit -m "feat!: redesign REST API"
git commit -m "fix!: change error response format"

# Major bump via footer
git commit -m "refactor: change auth token format

BREAKING CHANGE: JWT tokens now use RS256 instead of HS256"
```

**Other conventional commit types** (`docs:`, `chore:`, `test:`, `refactor:`, `ci:`, `style:`, `perf:`, `build:`) do not trigger any increment by themselves. The branch default increment applies.

**Config:** [examples/conventional-commits.yml](examples/conventional-commits.yml)

---

### Bump Directives

Simple keywords placed anywhere in a commit or merge message. Alternative to Conventional Commits for teams that prefer explicit control.

| Directive | Increment |
|-----------|-----------|
| `bump major` | Major |
| `bump minor` | Minor |
| `bump patch` | Patch |
| `bump none` or `bump skip` | None (suppress increment) |

**Examples:**

```bash
# In a commit message
git commit -m "redesign the API

bump major"

# In a merge commit (squash or regular)
git merge feature/new-api -m "Merge feature/new-api

bump minor"

# Suppress increment entirely
git commit -m "update formatting only

bump none"
```

**Config:** [examples/bump-directives.yml](examples/bump-directives.yml)

---

### Using Both (Default)

By default, gitsemver recognizes **both** Conventional Commits and bump directives. If both are present in the same commit, the highest increment wins.

```bash
git commit -m "feat: add new endpoint

bump major"
# feat: would be Minor, but "bump major" overrides → Major
```

Configure which conventions are active:

```yaml
# Use only conventional commits
commit-message-convention: conventional-commits

# Use only bump directives
commit-message-convention: bump-directive

# Use both (default)
commit-message-convention: both
```

---

### Manual Tags

The most direct way to set a version. Tag any commit and gitsemver picks it up via the TaggedCommit strategy.

```bash
# Release a specific version
git tag v1.0.0
git tag v2.0.0-rc.1

# With a custom prefix
git tag release-1.0.0   # requires tag-prefix: 'release-'
```

**When to use:** Releasing stable versions in ContinuousDelivery mode, forcing a version jump, or bootstrapping versioning on an existing repo.

**Important:** When you tag the current commit, gitsemver returns that exact version (no increment). When the tag is in the past, gitsemver increments from it.

```
v1.0.0 on HEAD     → 1.0.0 (exact)
v1.0.0, 3 commits ago → 1.0.1+3 (incremented, on main with Patch default)
```

---

### next-version Config

Set the floor for the next version in `.gitversion.yml`. Useful for initial development or planned version jumps.

```yaml
next-version: 2.0.0
```

**Behavior:**
- Returns `2.0.0` with `ShouldIncrement = false`
- Overridden by tags (if a tag produces a higher version)
- Once you've tagged `v2.0.0`, remove `next-version` from config

**Example: Starting a new major version:**

```yaml
# Before any v2.x.x tags exist
next-version: 2.0.0
```

```
main ── A (v1.5.0) ── B ── C
                            ^ HEAD

Without next-version: 1.5.1+2
With next-version: 2.0.0 (next-version wins because 2.0.0 > 1.5.1)
```

---

## Configuration Examples

All example configurations are in the [examples/](examples/) folder:

| File | Workflow | Mode |
|------|----------|------|
| [gitflow.yml](examples/gitflow.yml) | GitFlow (main + develop + release + feature + hotfix) | ContinuousDelivery |
| [trunk-based.yml](examples/trunk-based.yml) | Trunk-based development (main + short-lived features) | Mainline |
| [mainline.yml](examples/mainline.yml) | Mainline with per-branch mode overrides (hybrid) | Mainline + CD |
| [continuous-deployment.yml](examples/continuous-deployment.yml) | Every commit deployable | ContinuousDeployment |
| [conventional-commits.yml](examples/conventional-commits.yml) | Conventional Commits only | ContinuousDelivery |
| [bump-directives.yml](examples/bump-directives.yml) | Bump directives only | ContinuousDelivery |
| [github-flow.yml](examples/github-flow.yml) | GitHub Flow (main + feature PRs) | ContinuousDelivery |
| [minimal.yml](examples/minimal.yml) | All defaults | ContinuousDelivery |
| [custom-branches.yml](examples/custom-branches.yml) | Custom branch patterns and priorities | ContinuousDelivery |
