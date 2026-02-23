# gitsemver

A Go rewrite of [GitVersion](https://github.com/GitTools/GitVersion) (v5.12.0), redesigned with 12 architectural improvements. Automatic [Semantic Versioning](https://semver.org/) from your git history — no version files to maintain.

[![CI](../../actions/workflows/ci.yaml/badge.svg)](../../actions/workflows/ci.yaml)

## Why gitsemver

- **Zero configuration required** — works out of the box with sensible defaults for GitFlow, trunk-based, and CD workflows
- **Single static binary** — no runtime dependencies, runs on Linux, macOS, and Windows
- **Conventional Commits** — first-class support for `feat:`, `fix:`, `feat!:`, and `BREAKING CHANGE:` footers
- **Branch-aware** — eight built-in branch types with configurable pre-release labels, increment strategies, and versioning modes
- **30+ output variables** — `SemVer`, `FullSemVer`, `NuGetVersion`, `Sha`, `BranchName`, and more
- **Squash merge aware** — correctly parses GitHub, GitLab, and Bitbucket squash merge formats

## Installation

### Pre-built binaries

Download the latest release from [GitHub Releases](../../releases):

| Platform | Architecture |
|----------|-------------|
| Linux | amd64, arm64 |
| macOS | amd64, arm64 |
| Windows | amd64 |

### From source

```bash
go install go-gitsemver@latest
```

### Verify

```bash
gitsemver version
```

## Quick start

Run `gitsemver` in any git repository:

```bash
# Show all version variables
gitsemver

# Get just the semver string
gitsemver --show-variable SemVer
# Output: 1.2.3-beta.4

# JSON output for CI pipelines
gitsemver -o json

# See the effective configuration
gitsemver --show-config
```

### Example output

```
Major=1
Minor=2
Patch=3
SemVer=1.2.3
FullSemVer=1.2.3+5
MajorMinorPatch=1.2.3
BranchName=main
Sha=abc1234def567890...
ShortSha=abc1234
CommitsSinceVersionSource=5
...
```

## How it works

gitsemver analyzes your git repository in four steps:

1. **Find the base version** — scans tags, merge messages, branch names, and configuration for the latest version
2. **Determine the increment** — reads commit messages (Conventional Commits and/or bump directives) and branch config to decide major, minor, or patch
3. **Apply pre-release labels** — adds branch-specific labels (`alpha`, `beta`, `{BranchName}`, etc.) with auto-incrementing numbers
4. **Attach build metadata** — commit count, SHA, branch name, and commit date

```
main:       1.0.0 → 1.0.1 → 1.1.0
develop:    1.1.0-alpha.1 → 1.1.0-alpha.2
release:    1.1.0-beta.1 → 1.1.0-beta.2
feature:    1.1.0-my-feature.1
hotfix:     1.0.2-beta.1
```

## CLI reference

```
gitsemver [flags]
gitsemver [command]
```

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--path` | `-p` | `.` | Path to the git repository |
| `--branch` | `-b` | *(HEAD)* | Target branch name |
| `--commit` | `-c` | *(tip)* | Target commit SHA |
| `--config` | | *(auto)* | Path to config file |
| `--output` | `-o` | | Output format: `json` or default (key=value) |
| `--show-variable` | | | Show a single variable (e.g., `SemVer`) |
| `--show-config` | | | Print the effective configuration and exit |
| `--explain` | | | Show how the version was calculated |
| `--verbosity` | `-v` | `info` | Log verbosity: `quiet`, `info`, `debug` |

### Commands

| Command | Description |
|---------|-------------|
| `gitsemver` | Calculate and display version (default) |
| `gitsemver version` | Print the gitsemver binary version |

## Configuration

Place a `gitsemver.yml` (or `GitVersion.yml`) in your repository root. All fields are optional — defaults are applied automatically.

```yaml
# gitsemver.yml
mode: ContinuousDelivery          # ContinuousDelivery, ContinuousDeployment, or Mainline
tag-prefix: '[vV]'                 # Regex to match version tag prefixes
base-version: 0.1.0               # Starting version when no tags exist
commit-message-convention: Both    # ConventionalCommits, BumpDirective, or Both

branches:
  main:
    regex: ^master$|^main$
    increment: Patch
    tag: ''                        # Empty = stable version (no pre-release)
    is-mainline: true

  develop:
    regex: ^dev(elop)?(ment)?$
    increment: Minor
    tag: alpha

  release:
    regex: ^releases?[/-]
    increment: None
    tag: beta
    is-release-branch: true

  feature:
    regex: ^features?[/-]
    increment: Inherit
    tag: '{BranchName}'            # Replaced with branch name

ignore:
  sha: []
  commits-before: 2020-01-01
```

See the full [Configuration Reference](docs/CONFIGURATION.md) for all options.

### Versioning modes

#### ContinuousDelivery (default)

Pre-release versions track the branch. Stable versions are produced when you tag manually.

```
develop:  1.1.0-alpha.1, 1.1.0-alpha.2, 1.1.0-alpha.3
main:     tag v1.1.0 → 1.1.0
```

#### ContinuousDeployment

Every commit gets a unique, monotonically increasing version.

```
main: 1.0.1-ci.1, 1.0.1-ci.2, 1.0.1-ci.3
```

#### Mainline

By default, the highest increment from all commits since the last tag is applied once. Commit count goes into build metadata.

```
main: v1.0.0 ... 5 commits (fixes + feat) ... → 1.1.0+5
```

For GitVersion-compatible per-commit incrementing, set `mainline-increment: EachCommit`:

```yaml
mode: Mainline
mainline-increment: EachCommit   # fix→1.0.1, fix→1.0.2, feat→1.1.0, fix→1.1.1
```

### Commit message conventions

#### Conventional Commits

```
feat: add user authentication      → Minor
fix: resolve null pointer           → Patch
feat!: redesign API                 → Major

feat: change auth flow

BREAKING CHANGE: token format changed  → Major
```

#### Bump directives

```
some change +semver: major          → Major
some change +semver: feature        → Minor
some change +semver: fix            → Patch
some change +semver: skip           → No bump
```

### Branch defaults

| Branch | Regex | Increment | Pre-release tag | Priority |
|--------|-------|-----------|-----------------|----------|
| main | `^master$\|^main$` | Patch | *(stable)* | 100 |
| develop | `^dev(elop)?(ment)?$` | Minor | `alpha` | 60 |
| release | `^releases?[/-]` | None | `beta` | 90 |
| feature | `^features?[/-]` | Inherit | `{BranchName}` | 50 |
| hotfix | `^hotfix(es)?[/-]` | Patch | `beta` | 80 |
| pull-request | `^(pull\|pull-requests\|pr)[/-]` | Inherit | `PullRequest` | 40 |
| support | `^support[/-]` | Patch | *(stable)* | 70 |
| unknown | `.*` | Inherit | `{BranchName}` | 0 |

## Output variables

| Variable | Example | Description |
|----------|---------|-------------|
| `Major` | `1` | Major version component |
| `Minor` | `2` | Minor version component |
| `Patch` | `3` | Patch version component |
| `MajorMinorPatch` | `1.2.3` | Major.Minor.Patch |
| `SemVer` | `1.2.3-beta.4` | Semantic version with pre-release |
| `FullSemVer` | `1.2.3-beta.4+5` | SemVer with build metadata |
| `LegacySemVer` | `1.2.3-beta4` | Legacy format (no dot before number) |
| `LegacySemVerPadded` | `1.2.3-beta0004` | Legacy format with zero-padding |
| `InformationalVersion` | `1.2.3-beta.4+5.Branch.main.Sha.abc1234` | Full version string |
| `PreReleaseTag` | `beta.4` | Pre-release tag with number |
| `PreReleaseLabel` | `beta` | Pre-release label only |
| `PreReleaseNumber` | `4` | Pre-release number |
| `BuildMetaData` | `5` | Commits since tag |
| `FullBuildMetaData` | `5.Branch.main.Sha.abc1234` | Full build metadata string |
| `BranchName` | `main` | Current branch name |
| `Sha` | `abc1234def567...` | Full commit SHA |
| `ShortSha` | `abc1234` | Short commit SHA (7 chars) |
| `CommitDate` | `2025-01-15` | Commit date |
| `CommitsSinceVersionSource` | `5` | Commits since base version |
| `UncommittedChanges` | `0` | Dirty working tree count |
| `AssemblySemVer` | `1.2.3.0` | .NET assembly version |
| `NuGetVersionV2` | `1.2.3-beta0004` | NuGet-compatible version |
| `WeightedPreReleaseNumber` | `60004` | Sortable pre-release weight |

## CI/CD integration

### GitHub Actions

```yaml
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Full history required

      - name: Calculate version
        id: version
        run: echo "semver=$(gitsemver --show-variable SemVer)" >> "$GITHUB_OUTPUT"

      - name: Build
        run: docker build -t myapp:${{ steps.version.outputs.semver }} .
```

### GitLab CI

```yaml
build:
  script:
    - VERSION=$(gitsemver --show-variable SemVer)
    - echo "Building version $VERSION"
    - docker build -t myapp:$VERSION .
```

### Generic

```bash
VERSION=$(gitsemver --show-variable SemVer)
echo "Version: $VERSION"

# JSON output
gitsemver -o json > version.json
```

## Workflow examples

### GitFlow

```
main ───●───────────────────●───── 1.0.0 ────────── 1.1.0
         \                 /
develop   ●───●───●───●──● ────── 1.1.0-alpha.1..5
               \       /
feature/auth    ●───●  ────────── 1.1.0-auth.1..2
```

### Trunk-based (Mainline mode)

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

## Development

```bash
make build           # Build binary
make test            # Unit tests with coverage
make e2e             # End-to-end tests
make test-all        # Unit + e2e tests
make lint            # Run linter
make fmt             # Format code
make coverage-check  # Verify coverage >= 85%
make ci              # Full CI pipeline (fmt + lint + test-all + coverage + build)
```

## Documentation

| Document | Description |
|----------|-------------|
| [Configuration Reference](docs/CONFIGURATION.md) | All configuration options with defaults |
| [Architecture](docs/ARCHITECTURE.md) | System design and module layout |
| [SemVer Calculation](docs/SEMVER_CALCULATION.md) | How versions are calculated step by step |
| [Version Strategies](docs/VERSION_STRATEGIES.md) | The 6 strategies used to discover base versions |
| [Branch Workflows](docs/BRANCH_WORKFLOWS.md) | Branch types, versioning modes, and defaults |

## Acknowledgements

gitsemver is a ground-up rewrite of [GitVersion](https://github.com/GitTools/GitVersion) v5.12.0 (C#/.NET) in Go. It preserves the core versioning model — branch-aware strategies, three versioning modes, and `GitVersion.yml` configuration compatibility — while introducing 12 design improvements including immutable types, a single-increment pipeline, Conventional Commits support, squash merge awareness, and a simplified mainline calculator. See [COMPARISON.md](docs/COMPARISON.md) for a full breakdown.

## License

MIT
