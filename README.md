# go-gitsemver

A Go-based tool that automatically calculates [Semantic Versions](https://semver.org/) from your git history. No manual version files to maintain — versions are derived from tags, branches, commits, and merge history.

## How It Works

go-gitsemver analyzes your git repository to determine the current version:

1. **Finds the latest version tag** on the current branch (e.g., `v1.2.3`)
2. **Determines the increment** (major, minor, or patch) based on branch configuration and commit messages
3. **Applies a pre-release label** based on the branch type (e.g., `alpha` for develop, `beta` for release branches)
4. **Attaches build metadata** including commit count, SHA, and branch name

### Example

```
main:       1.0.0 → 1.0.1 → 1.1.0
develop:    1.1.0-alpha.1 → 1.1.0-alpha.2
release:    1.1.0-beta.1 → 1.1.0-beta.2
feature:    1.1.0-my-feature.1
```

## Features

- Automatic semantic versioning from git history
- Support for GitFlow, GitHub Flow, and trunk-based workflows
- Branch-aware pre-release labels
- Commit message directives (`+semver: major`, `+semver: minor`, `+semver: patch`)
- Configurable via `.gitversion.yml`
- JSON output with 30+ version variables

## Configuration

Place a `.gitversion.yml` in your repository root:

```yaml
mode: ContinuousDelivery
branches:
  main:
    increment: Patch
    tag: ''
  develop:
    increment: Minor
    tag: alpha
  release:
    increment: None
    tag: beta
  feature:
    increment: Inherit
    tag: '{BranchName}'
```

See [docs/CONFIGURATION.md](docs/CONFIGURATION.md) for the full reference.

## Documentation

- [Architecture](docs/ARCHITECTURE.md) — System design and module layout
- [SemVer Calculation](docs/SEMVER_CALCULATION.md) — How versions are calculated step by step
- [Version Strategies](docs/VERSION_STRATEGIES.md) — The 6 strategies used to discover base versions
- [Branch Workflows](docs/BRANCH_WORKFLOWS.md) — Branch types, versioning modes, and defaults
- [Configuration](docs/CONFIGURATION.md) — All configuration options
- [Git Analysis](docs/GIT_ANALYSIS.md) — What git data is consumed and how
- [CLI Interface](docs/CLI_INTERFACE.md) — Commands, output variables, and formats

## License

MIT
