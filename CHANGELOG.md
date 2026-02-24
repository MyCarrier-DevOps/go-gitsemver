# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.0] - GitHub API Remote Provider

### Added

- **`gitsemver remote owner/repo` subcommand** — calculate semantic versions via the GitHub REST and GraphQL APIs without requiring a local clone. Eliminates the need for `fetch-depth: 0` in CI pipelines.
  - Token auth (`--token` / `GITHUB_TOKEN`) and GitHub App auth (`--github-app-id` + `--github-app-key`)
  - GitHub Enterprise support via `--github-url` / `GITHUB_API_URL`
  - GraphQL batch fetching for branches and tags (avoids N+1 REST calls)
  - Smart early termination for commit walks using version tag detection
  - In-memory caching across the run (branches, tags, commits, merge bases)
  - Configurable `--max-commits` safety cap (default 1000)
  - Remote config file fetching (`gitsemver.yml` / `GitVersion.yml` from repo root via API)
- `CommitTag` output variable — `YY.WW.ShortSha` format derived from the commit date
- Date format translation between Go and .NET/Python/strftime conventions
- JSON schema for gitsemver configuration

### Changed

- Updated GitHub Actions to latest versions (checkout v6, setup-go v6, upload-artifact v6, download-artifact v7)

## [1.0.0] - Initial Release

- Automatic semantic versioning from git history
- 6 version discovery strategies (ConfigNextVersion, TaggedCommit, MergeMessage, VersionInBranchName, TrackReleaseBranches, Fallback)
- 3 versioning modes (ContinuousDelivery, ContinuousDeployment, Mainline)
- Conventional Commits and bump directive support
- 8 built-in branch configurations with priority-based matching
- Squash merge awareness (GitHub, GitLab, Bitbucket formats)
- 30+ output variables with JSON, key=value, and single-variable output
- GitVersion.yml configuration compatibility
