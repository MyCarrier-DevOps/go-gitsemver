# Installing and Using go-gitsemver Locally

This guide covers how to install go-gitsemver on your machine and use it to calculate semantic versions for your git repositories.

---

## Prerequisites

- **Git** — any recent version
- **Go 1.26+** — only required if building from source

---

## Installation

### Option 1: Download a pre-built binary

Go to the [GitHub Releases page](https://github.com/MyCarrier-DevOps/go-gitsemver/releases) and download the binary for your platform:

| Platform | Architecture | Binary |
|----------|-------------|--------|
| Linux | amd64 | `go-gitsemver-linux-amd64` |
| Linux | arm64 | `go-gitsemver-linux-arm64` |
| macOS | amd64 | `go-gitsemver-darwin-amd64` |
| macOS | arm64 (Apple Silicon) | `go-gitsemver-darwin-arm64` |
| Windows | amd64 | `go-gitsemver-windows-amd64.exe` |

#### Download from the browser

1. Open [https://github.com/MyCarrier-DevOps/go-gitsemver/releases](https://github.com/MyCarrier-DevOps/go-gitsemver/releases)
2. Find the latest release at the top of the page
3. Expand the **Assets** section
4. Click the binary that matches your OS and architecture to download it

#### Download from the command line

Replace `VERSION` with the desired release tag (e.g., `v1.2.0`) and `BINARY` with your platform binary name:

```bash
# Set your desired version and platform
VERSION="v1.2.0"   # replace with the latest release tag
BINARY="go-gitsemver-darwin-arm64"  # replace with your platform

# Download with curl
curl -Lo go-gitsemver "https://github.com/MyCarrier-DevOps/go-gitsemver/releases/download/${VERSION}/${BINARY}"

# Or download with wget
wget -O go-gitsemver "https://github.com/MyCarrier-DevOps/go-gitsemver/releases/download/${VERSION}/${BINARY}"
```

#### Verify the checksum (optional)

Each release includes a `checksums.txt` file. You can verify your download:

```bash
curl -Lo checksums.txt "https://github.com/MyCarrier-DevOps/go-gitsemver/releases/download/${VERSION}/checksums.txt"
shasum -a 256 --check --ignore-missing checksums.txt
```

#### Install the binary

Make it executable and move it to your PATH:

```bash
chmod +x go-gitsemver
sudo mv go-gitsemver /usr/local/bin/go-gitsemver
```

On **Windows**, move the `.exe` to a directory that is in your `PATH`, or add its location to your `PATH` environment variable.

### Option 2: Install with `go install`

```bash
go install github.com/MyCarrier-DevOps/go-gitsemver@latest
```

This places the binary in `$(go env GOPATH)/bin`. Make sure that directory is in your `PATH`.

### Option 3: Build from source

```bash
git clone https://github.com/MyCarrier-DevOps/go-gitsemver.git
cd go-gitsemver
make build
```

The binary is written to `bin/go-gitsemver`. You can move it to your PATH or run it directly:

```bash
./bin/go-gitsemver version
```

### Verify the installation

```bash
go-gitsemver version
```

---

## Basic usage

Run `go-gitsemver` from the root of any git repository with full history:

```bash
# Show all version variables (key=value format)
go-gitsemver

# Get just the semantic version string
go-gitsemver --show-variable SemVer

# JSON output
go-gitsemver -o json

# See how the version was calculated
go-gitsemver --explain

# View the effective configuration
go-gitsemver --show-config
```

### Important: full git history required

go-gitsemver needs access to your complete git history (tags, branches, merge commits) to calculate versions correctly. If you cloned with `--depth 1` or a shallow clone, fetch the full history first:

```bash
git fetch --unshallow
git fetch --tags
```

---

## Common workflows

### Get the version for scripting

```bash
VERSION=$(go-gitsemver --show-variable SemVer)
echo "Current version: $VERSION"
```

### Tag a release

```bash
VERSION=$(go-gitsemver --show-variable MajorMinorPatch)
git tag "v$VERSION"
git push origin "v$VERSION"
```

### Build a Docker image with the version

```bash
VERSION=$(go-gitsemver --show-variable SemVer)
docker build -t myapp:$VERSION .
```

### Check a different branch

```bash
go-gitsemver --branch release/2.0.0
```

### Point to a repo in a different directory

```bash
go-gitsemver --path /path/to/other/repo
```

---

## Configuration

go-gitsemver works with zero configuration out of the box. To customize behavior, create a config file in your repository:

```bash
# Preferred location
.github/go-gitsemver.yml

# Also supported (GitVersion compatibility)
.github/GitVersion.yml
GitVersion.yml
go-gitsemver.yml
```

A minimal configuration file:

```yaml
mode: ContinuousDelivery
tag-prefix: '[vV]'
branches:
  main:
    regex: ^master$|^main$
    increment: Patch
    tag: ''
```

You can also pass a config file explicitly:

```bash
go-gitsemver --config path/to/my-config.yml
```

See the full [Configuration Reference](CONFIGURATION.md) for all options.

---

## Versioning modes

go-gitsemver supports three versioning modes that control how versions are calculated. Set the mode in your config file or let the default apply.

### ContinuousDelivery (default)

Pre-release versions track the branch. Stable versions are produced only when you create a git tag manually.

```
develop:  1.1.0-alpha.1, 1.1.0-alpha.2, 1.1.0-alpha.3
main:     tag v1.1.0 → 1.1.0
```

### ContinuousDeployment

Every commit gets a unique, monotonically increasing version. No manual tagging needed.

```yaml
mode: ContinuousDeployment
```

```
main: 1.0.1-ci.1, 1.0.1-ci.2, 1.0.1-ci.3
```

### Mainline

The highest increment from all commits since the last tag is applied once. Commit count goes into build metadata.

```yaml
mode: Mainline
```

```
main: v1.0.0 ... 5 commits (fixes + feat) ... → 1.1.0+5
```

For per-commit incrementing (GitVersion-compatible), set `mainline-increment: EachCommit`:

```yaml
mode: Mainline
mainline-increment: EachCommit   # fix→1.0.1, fix→1.0.2, feat→1.1.0, fix→1.1.1
```

For a deep dive into modes and strategies, see [Strategies and Modes](STRATEGIES_AND_MODES.md) and [Branch Workflows](BRANCH_WORKFLOWS.md).

---

## Bumping versions manually

go-gitsemver determines the version bump automatically from your commit messages. You control what kind of bump happens by how you write your commits.

### Using Conventional Commits

Prefix your commit message with a type to signal the bump:

```bash
# Patch bump (bug fix)
git commit -m "fix: resolve null pointer in auth handler"

# Minor bump (new feature)
git commit -m "feat: add user profile endpoint"

# Major bump (breaking change — note the !)
git commit -m "feat!: redesign authentication API"

# Major bump (using a BREAKING CHANGE footer)
git commit -m "feat: change auth flow

BREAKING CHANGE: token format changed from JWT to opaque"
```

### Using bump directives

Use `bump major:`, `bump minor:`, or `bump patch:` as the prefix of your commit message:

```bash
# Major bump
git commit -m "bump major: redesign authentication API"

# Minor bump
git commit -m "bump minor: add new report type"

# Patch bump
git commit -m "bump patch: fix typo in output"
```

The `+semver:` style is also supported anywhere in the commit message:

```bash
git commit -m "overhaul config system +semver: major"
git commit -m "add new report type +semver: minor"
git commit -m "correct typo in output +semver: patch"
git commit -m "update docs +semver: skip"
```

### Using `next-version` in config

To force a specific version regardless of commit history, set `next-version` in your config file:

```yaml
# go-gitsemver.yml
next-version: 2.0.0
```

This overrides tag-based calculation. Remove this line after you tag the release.

### Using git tags directly

You can always set the version explicitly by creating a tag:

```bash
# Set the current version to 3.0.0
git tag v3.0.0
git push origin v3.0.0
```

All subsequent versions will be calculated relative to this tag.

### Summary

| Method | Example | Bump |
|--------|---------|------|
| `bump major:` | `bump major: redesign API` | Major |
| `bump minor:` | `bump minor: add search` | Minor |
| `bump patch:` | `bump patch: fix typo` | Patch |
| `feat:` commit | `feat: add search` | Minor |
| `fix:` commit | `fix: handle edge case` | Patch |
| `feat!:` or `BREAKING CHANGE:` | `feat!: new API` | Major |
| `+semver: major` | `refactor +semver: major` | Major |
| `+semver: minor` / `feature` | `add report +semver: minor` | Minor |
| `+semver: patch` / `fix` | `typo +semver: fix` | Patch |
| `+semver: skip` / `none` | `docs +semver: skip` | None |
| `next-version` config | `next-version: 2.0.0` | Exact version |
| Git tag | `git tag v3.0.0` | Sets base version |

By default, both Conventional Commits and bump directives are recognized (`commit-message-convention: Both`). You can restrict this in your config — see [Configuration Reference](CONFIGURATION.md).

---

## CLI flags reference

| Flag | Short | Description |
|------|-------|-------------|
| `--branch` | `-b` | Target branch name (default: HEAD) |
| `--commit` | `-c` | Target commit SHA |
| `--config` | | Path to config file |
| `--path` | `-p` | Path to the git repository (default: `.`) |
| `--output` | `-o` | Output format: `json` or key=value (default) |
| `--show-variable` | | Show a single variable (e.g., `SemVer`, `FullSemVer`) |
| `--show-config` | | Print the effective configuration and exit |
| `--explain` | | Show how the version was calculated |
| `--verbosity` | `-v` | Log verbosity: `quiet`, `info`, `debug` |

---

## Output variables

Running `go-gitsemver` without flags produces all version variables. The most commonly used ones:

| Variable | Example | Description |
|----------|---------|-------------|
| `SemVer` | `1.2.3-beta.4` | Semantic version with pre-release |
| `FullSemVer` | `1.2.3-beta.4+5` | SemVer with build metadata |
| `MajorMinorPatch` | `1.2.3` | Just Major.Minor.Patch |
| `Major` | `1` | Major version component |
| `Minor` | `2` | Minor version component |
| `Patch` | `3` | Patch version component |
| `BranchName` | `main` | Current branch name |
| `ShortSha` | `abc1234` | Short commit SHA |
| `CommitsSinceVersionSource` | `5` | Commits since the last version tag |

Run `go-gitsemver` with no flags to see the complete list of 30+ variables.

---

## Troubleshooting

### "not a git repository"

Make sure you are running the command inside a git repository, or use `--path` to point to one:

```bash
go-gitsemver --path /path/to/repo
```

### Version is always `1.0.0`

This usually means no version tags exist yet. Create an initial tag:

```bash
git tag v1.0.0
```

Or set a base version in your config:

```yaml
base-version: 0.1.0
```

### Unexpected version on a shallow clone

go-gitsemver needs full history. Fetch it:

```bash
git fetch --unshallow
git fetch --tags
```

### Debug output

Use `--explain` to see the version calculation steps, or `--verbosity debug` for detailed logs:

```bash
go-gitsemver --explain
go-gitsemver -v debug
```
