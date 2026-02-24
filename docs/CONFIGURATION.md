# Configuration Reference

gitsemver is configured via a `go-gitsemver.yml` or `GitVersion.yml` file in the repository root. All fields are optional — sensible defaults are applied.

---

## Configuration file

### Local mode

gitsemver searches for configuration in this order:

1. `.github/GitVersion.yml`
2. `.github/go-gitsemver.yml`
3. `GitVersion.yml` in the repository root
4. `go-gitsemver.yml` in the repository root
5. Path specified by `--config` flag (highest priority)

The first file found is used. If no file is found, built-in defaults are used.

### Remote mode (`gitsemver remote`)

When using the `remote` subcommand, configuration is fetched from the GitHub repository via API:

1. `.github/GitVersion.yml` (fetched via `GET /repos/{owner}/{repo}/contents/`)
2. `.github/go-gitsemver.yml`
3. `GitVersion.yml` in the repo root
4. `go-gitsemver.yml` in the repo root
5. Path specified by `--remote-config-path` flag (fetches a specific file from the remote repo)
6. Path specified by `--config` flag (local file override, highest priority)

If no remote config file exists and no override flags are provided, built-in defaults are used.

```yaml
# go-gitsemver.yml — minimal example
mode: ContinuousDelivery
tag-prefix: '[vV]'
next-version: 1.0.0
branches:
  main:
    regex: ^master$|^main$
    increment: Patch
    tag: ''
```

---

## Global options

### mode

| | |
|---|---|
| **Type** | Enum |
| **Default** | `ContinuousDelivery` |
| **Values** | `ContinuousDelivery`, `ContinuousDeployment`, `Mainline` |

Sets the default versioning mode for all branches. Individual branches can override this.

- **ContinuousDelivery** — Pre-release numbers from tag scanning. Stable versions require manual tagging.
- **ContinuousDeployment** — Commits-since-tag promoted to pre-release number. Every commit gets a unique version.
- **Mainline** — Highest increment from all commits applied once. Commit count in build metadata.

```yaml
mode: ContinuousDeployment
```

### tag-prefix

| | |
|---|---|
| **Type** | Regex string |
| **Default** | `[vV]` |

Regex pattern to identify and strip version tag prefixes. Applied when parsing tags like `v1.0.0`.

```yaml
tag-prefix: '[vV]'     # matches v1.0.0, V1.0.0
tag-prefix: ''          # no prefix, matches 1.0.0 directly
tag-prefix: 'release-'  # matches release-1.0.0
```

### base-version

| | |
|---|---|
| **Type** | Semver string |
| **Default** | `0.1.0` |

The starting version used by the Fallback strategy when no tags exist. This is permanent — it applies whenever no tags are found.

```yaml
base-version: 1.0.0    # start at 1.0.0 instead of 0.1.0
```

### next-version

| | |
|---|---|
| **Type** | Semver string |
| **Default** | *(none)* |

Explicitly sets the next version, overriding tag-based calculation.

- Ignored if the current commit is already tagged
- Does NOT increment (used as-is)
- Remove this field after tagging the release

```yaml
next-version: 2.0.0    # force next version to 2.0.0
```

### increment

| | |
|---|---|
| **Type** | Enum |
| **Default** | `Inherit` |
| **Values** | `None`, `Major`, `Minor`, `Patch`, `Inherit` |

Default increment strategy for branches that don't specify their own. `Inherit` resolves from the source branch hierarchy, with a final fallback to `Patch`.

### mainline-increment

| | |
|---|---|
| **Type** | Enum |
| **Default** | `Aggregate` |
| **Values** | `Aggregate`, `EachCommit` |

Controls how mainline mode applies version increments. Only relevant when `mode: Mainline`.

- **Aggregate** (default) — Finds the highest increment from all commits since the last tag and applies it once. Commit count goes into build metadata for uniqueness.
- **EachCommit** — Increments the version for each commit individually, matching GitVersion's per-commit behavior.

```yaml
mode: Mainline
mainline-increment: EachCommit
```

| Mode | Example (`v1.0.0` → fix → fix → feat → fix) | Result |
|------|-----------------------------------------------|--------|
| `Aggregate` | Highest = Minor, applied once | `1.1.0+4` |
| `EachCommit` | fix→1.0.1, fix→1.0.2, feat→1.1.0, fix→1.1.1 | `1.1.1` |

### continuous-delivery-fallback-tag

| | |
|---|---|
| **Type** | String |
| **Default** | `ci` |

Pre-release label used in ContinuousDeployment mode when no branch-specific tag is configured.

```yaml
continuous-delivery-fallback-tag: ci    # produces 1.0.1-ci.5
```

### commit-message-incrementing

| | |
|---|---|
| **Type** | Enum |
| **Default** | `Enabled` |
| **Values** | `Enabled`, `Disabled`, `MergeMessageOnly` |

Controls whether commit messages are scanned for version bump information.

- **Enabled** — All commits are scanned
- **Disabled** — Commit messages ignored; only branch config increment used
- **MergeMessageOnly** — Only merge commits (2+ parents) are scanned

### commit-message-convention

| | |
|---|---|
| **Type** | Enum |
| **Default** | `Both` |
| **Values** | `ConventionalCommits`, `BumpDirective`, `Both` |

Which commit message conventions to recognize.

- **ConventionalCommits** — `feat:`, `fix:`, `feat!:`, `BREAKING CHANGE:` footers
- **BumpDirective** — `+semver: major`, `+semver: minor`, `+semver: fix`, `+semver: skip`
- **Both** — Recognizes both conventions, highest bump wins

### Bump message patterns

| Option | Default | Triggers |
|--------|---------|----------|
| `major-version-bump-message` | `\+semver:\s?(breaking\|major)` | Major bump |
| `minor-version-bump-message` | `\+semver:\s?(feature\|minor)` | Minor bump |
| `patch-version-bump-message` | `\+semver:\s?(fix\|patch)` | Patch bump |
| `no-bump-message` | `\+semver:\s?(none\|skip)` | No bump |

These are regex patterns matched against commit messages. Override them to use your own conventions:

```yaml
major-version-bump-message: '\[major\]'
minor-version-bump-message: '\[minor\]'
patch-version-bump-message: '\[patch\]'
```

### merge-message-formats

| | |
|---|---|
| **Type** | Map of string to regex |
| **Default** | *(empty — uses 6 built-in + 2 squash formats)* |

Custom merge message regex patterns added on top of built-in formats. Must include a `(?P<SourceBranch>...)` capture group.

```yaml
merge-message-formats:
  azure-devops: '^Merged PR (?P<PullRequestNumber>\d+): Merge (?P<SourceBranch>.+) into .+'
```

### Formatting options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `commit-date-format` | String | `2006-01-02` | Go time format for `CommitDate` variable |
| `tag-pre-release-weight` | Integer | `60000` | Weight for tagged pre-release versions |
| `legacy-semver-padding` | Integer | `4` | Zero-padding for legacy semver (e.g., `beta0004`) |
| `build-metadata-padding` | Integer | `4` | Zero-padding for build metadata |
| `commits-since-version-source-padding` | Integer | `4` | Zero-padding for commits-since count |

---

## Branch configuration

Each branch type is configured under the `branches:` key. gitsemver includes 8 built-in branch configs that can be customized.

### Branch options

#### regex

| | |
|---|---|
| **Type** | Regex string |
| **Required** | Yes |

Pattern to match branch names. When multiple configs match, the one with the highest **priority** wins.

#### increment

| | |
|---|---|
| **Type** | Enum: `None`, `Major`, `Minor`, `Patch`, `Inherit` |

Increment strategy for this branch. `Inherit` walks up the source-branch hierarchy until a concrete value is found (fallback: `Patch`).

#### mode

| | |
|---|---|
| **Type** | Enum (same as global `mode`) |

Versioning mode override for this branch.

#### tag

| | |
|---|---|
| **Type** | String |

Pre-release label for this branch:

| Value | Effect | Example |
|-------|--------|---------|
| `""` (empty) | Stable version, no pre-release | `1.2.3` |
| `"{BranchName}"` | Replaced with cleaned branch name | `1.2.3-my-feature.1` |
| `"alpha"` | Literal label | `1.2.3-alpha.1` |

#### source-branches

| | |
|---|---|
| **Type** | List of branch config keys |

Which branch types this branch can be created from. Used for `Inherit` increment resolution and branch creation point detection.

#### is-source-branch-for

| | |
|---|---|
| **Type** | List of branch config keys |

Inverse of `source-branches`. Declaring `is-source-branch-for: [feature]` on `develop` adds `develop` to feature's `source-branches`.

#### is-mainline

| | |
|---|---|
| **Type** | Boolean |
| **Default** | `false` |

Marks as a mainline branch. Mainline branches produce stable versions and serve as the trunk reference in Mainline versioning mode.

#### is-release-branch

| | |
|---|---|
| **Type** | Boolean |
| **Default** | `false` |

Enables version extraction from the branch name (e.g., `release/2.0.0` → base version `2.0.0`).

#### tracks-release-branches

| | |
|---|---|
| **Type** | Boolean |
| **Default** | `false` |

Makes this branch aware of active release branches and main branch tags. Typically enabled for `develop`.

#### prevent-increment-of-merged-branch-version

| | |
|---|---|
| **Type** | Boolean |
| **Default** | varies by branch |

When `true`, merging from this branch won't trigger an increment from merge message version extraction.

#### tag-number-pattern

| | |
|---|---|
| **Type** | Regex with `(?<number>\d+)` capture group |

Extracts a number from the branch name to append to the pre-release tag. Used for pull-request branches:

```yaml
tag-number-pattern: '[/-](?<number>\d+)'
# PR branch: pull/123 → pre-release includes 123
```

#### priority

| | |
|---|---|
| **Type** | Integer |
| **Default** | varies by branch |

When multiple branch configs match a branch name, the highest priority wins. Built-in priorities: main=100, release=90, hotfix=80, support=70, develop=60, feature=50, pull-request=40, unknown=0.

#### pre-release-weight

| | |
|---|---|
| **Type** | Integer |

Weight for sorting pre-release versions. Used in `WeightedPreReleaseNumber` output calculation.

#### commit-message-incrementing

| | |
|---|---|
| **Type** | Enum (same as global) |

Override commit message incrementing mode for this branch only.

---

## Built-in branch defaults

| Key | Regex | Increment | Mode | Tag | Release | Mainline | Priority |
|-----|-------|-----------|------|-----|---------|----------|----------|
| `main` | `^master$\|^main$` | Patch | CD | `""` | No | Yes | 100 |
| `develop` | `^dev(elop)?(ment)?$` | Minor | CDeployment | `alpha` | No | No | 60 |
| `release` | `^releases?[/-]` | None | CD | `beta` | Yes | No | 90 |
| `feature` | `^features?[/-]` | Inherit | CD | `{BranchName}` | No | No | 50 |
| `hotfix` | `^hotfix(es)?[/-]` | Patch | CD | `beta` | No | No | 80 |
| `pull-request` | `^(pull\|pull-requests\|pr)[/-]` | Inherit | CD | `PullRequest` | No | No | 40 |
| `support` | `^support[/-]` | Patch | CD | `""` | No | Yes | 70 |
| `unknown` | `.*` | Inherit | CD | `{BranchName}` | No | No | 0 |

---

## Ignore configuration

Exclude specific commits from version calculation:

```yaml
ignore:
  sha:
    - abc1234def567890    # Ignore specific commit SHAs
  commits-before: 2020-01-01T00:00:00   # Ignore commits before this date
```

Ignored commits are excluded during base version selection. Their tags remain visible but are filtered out.

---

## Configuration resolution order

1. **Built-in defaults** — `CreateDefaultConfiguration()` with 8 branch configs
2. **Config file** — Values from `go-gitsemver.yml` / `GitVersion.yml` merged on top
3. **CLI flags** — `--config` file override
4. **Branch finalization** — Branch configs inherit from global config where unset
5. **Effective configuration** — Final resolved config for the specific branch

### Branch config inheritance

When a branch config property is unset (`null`), it inherits from the global config:

- `increment`: if unset → global `increment` → `Inherit`
- `mode`: if unset → global `mode` → `ContinuousDelivery`
  - Exception: `develop` gets `ContinuousDeployment` unless global is `Mainline`

### Increment "Inherit" resolution

When `increment = Inherit`, gitsemver walks up the source-branch hierarchy:

1. Find the source branch for the current branch
2. Check its `increment` setting
3. If also `Inherit`, continue walking up
4. Final fallback: `Patch`

---

## Example configurations

### Minimal (all defaults)

```yaml
# Empty file — all defaults apply
```

### GitFlow with custom labels

```yaml
mode: ContinuousDelivery
branches:
  main:
    tag: ''
  develop:
    tag: dev
  release:
    tag: rc
  feature:
    tag: '{BranchName}'
```

### Trunk-based development

```yaml
mode: Mainline
commit-message-convention: ConventionalCommits
branches:
  main:
    increment: Patch
    is-mainline: true
```

### ContinuousDeployment

```yaml
mode: ContinuousDeployment
continuous-delivery-fallback-tag: ci
```

### Force next major version

```yaml
next-version: 2.0.0
# Remove this line after tagging v2.0.0
```

### Custom bump patterns

```yaml
commit-message-convention: BumpDirective
major-version-bump-message: '\[major\]'
minor-version-bump-message: '\[minor\]'
patch-version-bump-message: '\[patch\]'
no-bump-message: '\[skip\]'
```

### Monorepo (per-service)

```yaml
# service-a/go-gitsemver.yml
tag-prefix: 'service-a/v'
```
