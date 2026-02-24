# gitsemver

A Go application inspired by [GitVersion](https://github.com/GitTools/GitVersion) (v5.12.0). Automatic [Semantic Versioning](https://semver.org/) from your git history — no version files to maintain.

[![CI](../../actions/workflows/ci.yaml/badge.svg)](../../actions/workflows/ci.yaml)

## TL;DR

gitsemver calculates the next semantic version based on git history, tags, and branch conventions. Single static binary, zero runtime dependencies.

```bash
# Local mode — works with any git provider (GitHub, GitLab, Bitbucket, etc.)
# Requires a full clone (fetch-depth: 0 in CI)
gitsemver                                 # all version variables (key=value)
gitsemver --show-variable SemVer          # just the version string
gitsemver -o json                         # JSON output for CI
gitsemver --explain                       # show how the version was calculated

# Remote mode — GitHub and GitHub Enterprise only, no clone needed
# Requires a token (GITHUB_TOKEN) or GitHub App credentials
GITHUB_TOKEN=ghp_xxx gitsemver remote owner/repo
gitsemver remote owner/repo --token ghp_xxx --ref main
gitsemver remote owner/repo --github-app-id 12345 --github-app-key key.pem
```

**What it gives you:** `SemVer`, `FullSemVer`, `Major`, `Minor`, `Patch`, `BranchName`, `Sha`, `CommitDate`, `NuGetVersionV2`, and 20+ more output variables.

**What it understands:** Conventional Commits (`feat:`, `fix:`, `feat!:`), bump directives (`+semver: major`), 8 branch types (main, develop, release, feature, hotfix, pull-request, support, unknown), 3 versioning modes (ContinuousDelivery, ContinuousDeployment, Mainline), and squash merge formats from GitHub, GitLab, and Bitbucket.

**Configuration:** Drop a `go-gitsemver.yml` or `GitVersion.yml` in your repo root, or use `--config`. Works with zero config out of the box.

## Why gitsemver

- **Zero configuration required** — works out of the box with sensible defaults for GitFlow, trunk-based, and CD workflows
- **Single static binary** — no runtime dependencies, runs on Linux, macOS, and Windows
- **Two modes: local and remote** — run against a local clone, or version a GitHub repo via API without cloning
- **Go library** — embed version calculation in your own Go applications via `pkg/sdk`
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

### Local mode (default)

Run `gitsemver` inside a git repository with full history:

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

**Requires:** A local git clone with full history (`git clone` or `fetch-depth: 0` in CI). Reads tags, commits, and branches directly from the `.git` directory using go-git.

### Remote mode (GitHub API)

Version a GitHub repository without cloning it:

```bash
# Token auth (PAT, fine-grained token, or GitHub Actions GITHUB_TOKEN)
GITHUB_TOKEN=ghp_xxx gitsemver remote myorg/myrepo

# Specific branch
gitsemver remote myorg/myrepo --token ghp_xxx --ref main --show-variable SemVer

# GitHub App auth
gitsemver remote myorg/myrepo --github-app-id 12345 --github-app-key /path/to/key.pem

# GitHub Enterprise
gitsemver remote myorg/myrepo --token ghp_xxx --github-url https://ghe.example.com/api/v3
```

**Requires:** A GitHub token or GitHub App credentials. No clone, no checkout, no `fetch-depth: 0`. Reads tags, commits, and branches via the GitHub REST and GraphQL APIs. Configuration is fetched from the repo root (`go-gitsemver.yml` or `GitVersion.yml`) automatically.

### Example output

Both modes produce the same output:

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

### When to use which

| | Local mode | Remote mode |
|---|---|---|
| **Command** | `gitsemver` | `gitsemver remote owner/repo` |
| **Requires** | Local git clone with full history | GitHub token or App credentials |
| **Best for** | Developer machines, CI with full checkout | CI without clone, fast pipelines, large repos |
| **Git providers** | Any (GitHub, GitLab, Bitbucket, etc.) | GitHub and GitHub Enterprise only |
| **Working dir** | Detects uncommitted changes | N/A (no working directory) |
| **Speed** | Instant (reads local `.git`) | ~2-5 API calls for typical repos |
| **Config source** | Local filesystem | Fetched from repo via API (or `--config` local override) |

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

### Commands

| Command | Mode | Description |
|---------|------|-------------|
| `gitsemver [flags]` | Local | Calculate version from a local git repository (default) |
| `gitsemver remote owner/repo [flags]` | Remote | Calculate version from a GitHub repository via API |
| `gitsemver version` | — | Print the gitsemver binary version |

### Global flags (both local and remote)

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--branch` | `-b` | *(HEAD)* | Target branch name |
| `--commit` | `-c` | *(tip)* | Target commit SHA |
| `--config` | | *(auto)* | Path to config file |
| `--output` | `-o` | | Output format: `json` or default (key=value) |
| `--show-variable` | | | Show a single variable (e.g., `SemVer`) |
| `--show-config` | | | Print the effective configuration and exit |
| `--explain` | | | Show how the version was calculated |
| `--verbosity` | `-v` | `info` | Log verbosity: `quiet`, `info`, `debug` |

### Local-only flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--path` | `-p` | `.` | Path to the git repository |

### Remote-only flags

| Flag | Env var | Default | Description |
|------|---------|---------|-------------|
| `--token` | `GITHUB_TOKEN` | | GitHub personal access token or Actions token |
| `--github-app-id` | `GH_APP_ID` | | GitHub App ID |
| `--github-app-key` | `GH_APP_PRIVATE_KEY` | | Path to GitHub App private key PEM file |
| `--github-url` | `GITHUB_API_URL` | | GitHub Enterprise API base URL |
| `--ref` | | *(default branch)* | Branch, tag, or SHA to version |
| `--max-commits` | | `1000` | Maximum commit depth to walk via API |

Authentication is resolved in order: `--token`/`GITHUB_TOKEN` > `--github-app-id` + `--github-app-key` > error.

## Configuration

Place a `go-gitsemver.yml` (or `GitVersion.yml`) in your repository root. All fields are optional — defaults are applied automatically.

```yaml
# go-gitsemver.yml
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
| `CommitTag` | `25.03.abc1234` | Year.Week.ShortSha from commit date |
| `CommitsSinceVersionSource` | `5` | Commits since base version |
| `UncommittedChanges` | `0` | Dirty working tree count |
| `AssemblySemVer` | `1.2.3.0` | .NET assembly version |
| `NuGetVersionV2` | `1.2.3-beta0004` | NuGet-compatible version |
| `WeightedPreReleaseNumber` | `60004` | Sortable pre-release weight |

## CI/CD integration

### GitHub Actions (local mode)

```yaml
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Full history required for local mode

      - name: Calculate version
        id: version
        run: echo "semver=$(gitsemver --show-variable SemVer)" >> "$GITHUB_OUTPUT"

      - name: Build
        run: docker build -t myapp:${{ steps.version.outputs.semver }} .
```

### GitHub Actions (remote mode — no clone needed)

```yaml
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Calculate version
        id: version
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: echo "semver=$(gitsemver remote ${{ github.repository }} --ref ${{ github.ref_name }} --show-variable SemVer)" >> "$GITHUB_OUTPUT"

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

## Go library

gitsemver can be embedded in Go applications via the `pkg/sdk` package. This lets you calculate versions programmatically without shelling out to the CLI.

```go
import "go-gitsemver/pkg/sdk"

// Local mode — calculate from a local git repository
result, err := sdk.Calculate(sdk.LocalOptions{
    Path: ".",
})
fmt.Println(result.Variables["SemVer"]) // "1.2.3"

// Remote mode — calculate via GitHub API (no clone needed)
result, err := sdk.CalculateRemote(sdk.RemoteOptions{
    Owner: "myorg",
    Repo:  "myrepo",
    Token: os.Getenv("GITHUB_TOKEN"),
    Ref:   "main",
})
fmt.Println(result.Variables["FullSemVer"]) // "1.2.3+5"
```

`result.Variables` is a `map[string]string` containing all 30+ output variables (`SemVer`, `FullSemVer`, `Major`, `Minor`, `Patch`, `BranchName`, `Sha`, etc.).

See [example/main.go](example/main.go) for a runnable example.

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
| [Strategies and Modes](docs/STRATEGIES_AND_MODES.md) | Version strategies, versioning modes, and manual overrides |
| [Branch Workflows](docs/BRANCH_WORKFLOWS.md) | Branch types, defaults, and priority matching |
| [Version Strategies](docs/VERSION_STRATEGIES.md) | How the 6 version discovery strategies work |
| [Architecture](docs/ARCHITECTURE.md) | Package structure and design principles |
| [Features](docs/FEATURES.md) | Key features and design highlights |
| [Go Library Example](example/main.go) | Runnable example of the `pkg/sdk` library API |

## Acknowledgements

gitsemver is inspired by [GitVersion](https://github.com/GitTools/GitVersion) v5.12.0. It preserves the core versioning model — branch-aware strategies, three versioning modes, and `GitVersion.yml` configuration compatibility — while introducing improvements including immutable types, a single-increment pipeline, Conventional Commits support, squash merge awareness, and a simplified mainline calculator. See [Features](docs/FEATURES.md) for details.

## License

MIT
