# gitsemver

A fast, cross-platform tool that automatically calculates [Semantic Versions](https://semver.org/) from your git history. No manual version files to maintain — versions are derived from tags, branches, commits, and merge history.

## Installation

### Pre-built binaries

Download the latest release from [GitHub Releases](../../releases) for your platform:

- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

### From source

```bash
go install go-gitsemver@latest
```

### Verify

```bash
gitsemver version
```

## Quick Start

Run `gitsemver` in any git repository:

```bash
# Full JSON output with all version variables
gitsemver

# Just the semver string
gitsemver --show-variable SemVer

# Show what's happening under the hood
gitsemver --explain
```

Output:

```json
{
  "Major": 1,
  "Minor": 2,
  "Patch": 3,
  "SemVer": "1.2.3-beta.4",
  "FullSemVer": "1.2.3-beta.4+5",
  "MajorMinorPatch": "1.2.3",
  "PreReleaseTag": "beta.4",
  "CommitsSinceVersionSource": 5,
  "Sha": "abc1234def567890...",
  "ShortSha": "abc1234",
  "BranchName": "release/1.2.3",
  ...
}
```

## How It Works

gitsemver analyzes your git repository to determine the current version:

1. **Finds the latest version tag** on the current branch (e.g., `v1.2.3`)
2. **Determines the increment** (major, minor, or patch) from commit messages and branch configuration
3. **Applies a pre-release label** based on the branch type (e.g., `alpha` for develop, `rc` for release branches)
4. **Attaches build metadata** including commit count, SHA, and branch name

```
main:       1.0.0 → 1.0.1 → 1.1.0
develop:    1.1.0-alpha.1 → 1.1.0-alpha.2
release:    1.1.0-rc.1 → 1.1.0-rc.2
feature:    1.1.0-my-feature.1
hotfix:     1.0.2-beta.1
```

## Features

### Automatic Versioning from Git History

No version files to maintain. gitsemver reads your tags, branches, commit messages, and merge history to calculate the correct version at any point in your repository.

### Conventional Commits

First-class support for [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add user authentication      → Minor bump
fix: resolve null pointer in login  → Patch bump
feat!: redesign API                 → Major bump

# Footer style also works:
refactor: change auth flow

BREAKING CHANGE: token format changed
```

Also supports explicit `bump` directives anywhere in a commit or merge message:

```
bump major
bump minor
bump patch
bump none       # or: bump skip
```

Configurable via `commit-message-convention`:

```yaml
commit-message-convention: conventional-commits  # or: bump-directive, both (default)
```

### Squash Merge Awareness

Works with squash merges out of the box. Parses common squash merge formats:

```
feat: add login page (#123)                         # GitHub squash
Merge branch 'feature/auth' into 'main'             # GitLab squash
Merged in feature/auth (pull request #123)           # Bitbucket
```

### Explain Mode

Understand exactly why gitsemver calculated a specific version:

```bash
gitsemver --explain
```

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

### Three Versioning Modes

#### ContinuousDelivery (default)

Pre-release versions track the branch; stable versions are produced when tags are applied manually.

```
develop:  1.1.0-alpha.1, 1.1.0-alpha.2, 1.1.0-alpha.3
main:     tag v1.1.0 → 1.1.0
```

#### ContinuousDeployment

Every commit gets a unique, monotonically increasing version. Commits-since-tag is promoted to the pre-release number.

```
main: 1.0.1-ci.1, 1.0.1-ci.2, 1.0.1-ci.3  (each commit is deployable)
```

#### Mainline

The highest increment from all commits since the last tag is applied once. Commit count goes into build metadata. Version numbers stay semantically meaningful.

```
main:     v1.0.0 ... 5 commits (fixes + feat) ... → 1.1.0+5
feature:  1.1.0-my-feature.1+2
```

To force a version jump, tag manually or use `bump major` / `feat!:` in a commit.

### Branch-Aware Defaults

Seven built-in branch configurations with sensible defaults:

| Branch | Regex | Increment | Tag | Example Version |
|--------|-------|-----------|-----|-----------------|
| main | `^master$\|^main$` | Patch | *(empty — stable)* | `1.2.3` |
| develop | `^dev(elop)?(ment)?$` | Minor | `alpha` | `1.3.0-alpha.4` |
| release | `^releases?[/-]` | None | `beta` | `1.3.0-beta.1` |
| feature | `^features?[/-]` | Inherit | `{BranchName}` | `1.3.0-my-feature.1` |
| hotfix | `^hotfix(es)?[/-]` | Patch | `beta` | `1.2.4-beta.1` |
| pull-request | `^(pull\|pull-requests\|pr)[/-]` | Inherit | `PullRequest` | `1.3.0-PullRequest0005.1` |
| support | `^support[/-]` | Patch | *(empty — stable)* | `1.2.4` |

### 30+ Output Variables

Full set of version variables for any CI/CD pipeline:

| Variable | Example |
|----------|---------|
| `Major`, `Minor`, `Patch` | `1`, `2`, `3` |
| `MajorMinorPatch` | `1.2.3` |
| `SemVer` | `1.2.3-beta.4` |
| `FullSemVer` | `1.2.3-beta.4+5` |
| `LegacySemVer` | `1.2.3-beta4` |
| `LegacySemVerPadded` | `1.2.3-beta0004` |
| `InformationalVersion` | `1.2.3-beta.4+5.Branch.main.Sha.abc1234` |
| `PreReleaseTag` | `beta.4` |
| `PreReleaseLabel` | `beta` |
| `PreReleaseNumber` | `4` |
| `WeightedPreReleaseNumber` | `60004` |
| `BuildMetaData` | `5` |
| `FullBuildMetaData` | `5.Branch.main.Sha.abc1234` |
| `BranchName` | `main` |
| `EscapedBranchName` | `main` |
| `Sha` | `abc1234def567890...` |
| `ShortSha` | `abc1234` |
| `CommitDate` | `2025-01-15` |
| `VersionSourceSha` | `def5678...` |
| `CommitsSinceVersionSource` | `5` |
| `UncommittedChanges` | `0` |

## CLI Reference

```
gitsemver [flags]
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--path` | `-p` | Repository path (defaults to current directory) |
| `--branch` | `-b` | Target branch name |
| `--commit` | `-c` | Target commit SHA |
| `--output` | `-o` | Output format: `json` (default), `buildserver`, `file` |
| `--output-file` | | File path to write version info to |
| `--show-variable` | | Show only a specific variable (e.g., `SemVer`) |
| `--show-config` | | Print the effective configuration |
| `--override-config` | | Override config values (e.g., `tag-prefix=custom`) |
| `--no-cache` | | Disable version caching |
| `--explain` | | Show the full decision tree for the version calculation |
| `--verbosity` | `-v` | Log verbosity: `quiet`, `normal`, `verbose`, `diagnostic` |

### Subcommands

| Command | Description |
|---------|-------------|
| `gitsemver version` | Print the gitsemver binary version |
| `gitsemver show-config` | Print the effective configuration as YAML |

## Configuration

Place a `.gitversion.yml` in your repository root. All fields are optional — sensible defaults are applied.

```yaml
# .gitversion.yml
mode: ContinuousDelivery
tag-prefix: '[vV]'
commit-message-convention: both          # conventional-commits, bump-directive, or both
increment: Inherit
continuous-delivery-fallback-tag: ci
commit-date-format: 'yyyy-MM-dd'

branches:
  main:
    regex: ^master$|^main$
    increment: Patch
    tag: ''
    is-mainline: true
    source-branches: [release, hotfix]

  develop:
    regex: ^dev(elop)?(ment)?$
    increment: Minor
    tag: alpha
    tracks-release-branches: true
    source-branches: [main, release, hotfix, support]

  release:
    regex: ^releases?[/-]
    increment: None
    tag: beta
    is-release-branch: true
    source-branches: [develop, main, support, release]

  feature:
    regex: ^features?[/-]
    increment: Inherit
    tag: '{BranchName}'
    source-branches: [develop, main, release, hotfix, support]

  hotfix:
    regex: ^hotfix(es)?[/-]
    increment: Patch
    tag: beta
    source-branches: [main, support]

  pull-request:
    regex: ^(pull|pull-requests|pr)[/-]
    increment: Inherit
    tag: PullRequest
    tag-number-pattern: '[/-](?<number>\d+)'
    source-branches: [develop, main, release, feature, hotfix, support]

  support:
    regex: ^support[/-]
    increment: Patch
    tag: ''
    is-mainline: true
    source-branches: [main]

ignore:
  sha: []
  commits-before: 2020-01-01

merge-message-formats:
  custom: '^Merged PR (?<PullRequestNumber>\d+): .*$'
```

### Configuration Resolution Order

1. Built-in defaults
2. `.gitversion.yml` file values
3. CLI `--override-config` flags

Branch configs inherit from global config where unset. The `Inherit` increment strategy walks up the source-branch hierarchy until it finds a concrete value (fallback: `Patch`).

## CI/CD Integration

### GitHub Actions

```yaml
- name: Calculate version
  id: version
  run: |
    echo "semver=$(gitsemver --show-variable SemVer)" >> "$GITHUB_OUTPUT"

- name: Build
  run: |
    docker build -t myapp:${{ steps.version.outputs.semver }} .
```

### GitLab CI

```yaml
variables:
  VERSION: $(gitsemver --show-variable SemVer)

build:
  script:
    - gitsemver --output buildserver
    - echo "Building version $VERSION"
```

### Generic

```bash
# Use in any CI system
VERSION=$(gitsemver --show-variable SemVer)
echo "Building version: $VERSION"

# Write to file
gitsemver --output file --output-file version.json
```

## Workflow Examples

### GitFlow

```
main ───●───────────────────●───── 1.0.0 ────────── 1.1.0
         \                 /
develop   ●───●───●───●──● ────── 1.1.0-alpha.1..5
               \       /
feature/auth    ●───●  ────────── 1.1.0-auth.1..2
```

### Trunk-Based (Mainline Mode)

```yaml
mode: Mainline
```

```
main ───●─────●─────●─────●
        v1.0.0              1.1.0+3
        (tag)  fix   fix   feat
                           ↑ highest = minor, applied once
```

### ContinuousDeployment

```yaml
mode: ContinuousDeployment
```

```
main ───●─────●─────●─────●
        1.0.0  1.0.1-ci.1  1.0.1-ci.2  1.0.1-ci.3
        (tag)  (auto)       (auto)       (auto)
```

## Documentation

- [Architecture](docs/ARCHITECTURE.md) — System design and module layout
- [SemVer Calculation](docs/SEMVER_CALCULATION.md) — How versions are calculated step by step
- [Version Strategies](docs/VERSION_STRATEGIES.md) — The strategies used to discover base versions
- [Branch Workflows](docs/BRANCH_WORKFLOWS.md) — Branch types, versioning modes, and defaults
- [Configuration](docs/CONFIGURATION.md) — All configuration options
- [Git Analysis](docs/GIT_ANALYSIS.md) — What git data is consumed and how
- [CLI Interface](docs/CLI_INTERFACE.md) — Commands, output variables, and formats

## License

MIT
