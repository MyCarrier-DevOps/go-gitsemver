# go-gitsemver — Application Highlights

## What It Does

**go-gitsemver** is a Git-based semantic versioning tool that automatically calculates the next version for your project by analyzing your Git history — branches, tags, merge messages, and commit conventions.

Zero manual version bumping. Zero version files to maintain. The version is derived entirely from your repository state.

---

## Core Versioning Strategies

Six strategies are evaluated in priority order to determine the base version:

| Strategy | How It Works |
|----------|-------------|
| **ConfigNextVersion** | Explicit `next-version` set in config — highest priority override |
| **TaggedCommit** | Reads existing Git tags (e.g. `v1.2.3`) to find the latest release |
| **MergeMessage** | Parses merge commit messages like `Merge branch 'release/1.3.0'` |
| **VersionInBranchName** | Extracts version from branch names like `release/2.0.0` or `release-1.2` |
| **TrackReleaseBranches** | Watches release branches from a develop branch for version signals |
| **Fallback** | Default `1.0.0` when no other signal exists (configurable via `base-version`) |

The highest-priority strategy with a valid result wins.

---

## Increment Detection

Version bumps are determined automatically from commit messages using pluggable conventions:

- **Conventional Commits** — `feat:` → Minor, `fix:` → Patch, `feat!:` or `BREAKING CHANGE` → Major
- **Bump directives** — `+semver: major`, `+semver: minor`, `+semver: patch` (with aliases like `breaking`, `feature`, `fix`)

The highest increment across all commits since the last version tag is applied.

---

## Branch-Aware Versioning

Different branch types produce different version formats:

| Branch Type | Example Output |
|-------------|---------------|
| **main/master** | `1.2.3` |
| **develop** | `1.3.0-alpha.4` |
| **feature/login** | `1.3.0-login.2` |
| **release/2.0.0** | `2.0.0-beta.1` |
| **hotfix/fix-crash** | `1.2.4-fix-crash.1` |

Pre-release labels, numbering, and increment behavior are all configurable per branch pattern via regex matching.

---

## Mainline Mode

An alternative versioning strategy for trunk-based development:

- **Aggregate mode** — finds the highest increment across all commits since the last tag
- **Each-commit mode** — treats every commit as a potential version bump, stacking increments

Supports teams that ship from `main` without release branches.

---

## Configuration

YAML-based configuration (`GitVersion.yml` or `go-gitsemver.yml`) with sensible defaults:

```yaml
mode: ContinuousDelivery          # or ContinuousDeployment, Mainline
tag-prefix: "[vV]"
major-version-bump-message: "^\\+semver:\\s?(breaking|major)"
minor-version-bump-message: "^\\+semver:\\s?(feature|minor)"
patch-version-bump-message: "^\\+semver:\\s?(fix|patch)"
commit-message-incrementing: Enabled
branches:
  main:
    regex: "^master$|^main$"
    tag: ""
    increment: Patch
  develop:
    regex: "^dev(elop)?(ment)?$"
    tag: alpha
    increment: Minor
  feature:
    regex: "^features?[/-]"
    tag: "{BranchName}"
    increment: Inherit
  release:
    regex: "^releases?[/-]"
    is-release-branch: true
    tag: beta
    increment: None
```

Config file auto-detection searches `.github/` first, then the repo root. Branches are matched by regex. Unmatched branches inherit defaults. The `{BranchName}` placeholder dynamically inserts a sanitized branch name as the pre-release label.

---

## Explain Mode

The `--explain` flag provides full transparency into version calculation:

```
Strategies evaluated:
  TaggedCommit:          1.0.0 (source: abc1234, increment: true)
    → tag v1.0.0 on commit abc1234 → 1.0.0, ShouldIncrement=true
  Fallback:              1.0.0 (source: external, increment: true)
    → using fallback base version 1.0.0

Selected: TaggedCommit (1.0.0, source: abc1234)

Increment:
  → commit e5f6a78 "feat: add auth module" → Minor (ConventionalCommits)
  → highest increment from commits: Minor

Result: 1.1.0
```

Every decision — which strategy won, which commit drove the bump, how the pre-release tag was resolved — is visible and auditable.

---

## Output Formats

Multiple output modes for CI/CD integration:

| Flag | Output |
|------|--------|
| `--show-variable SemVer` | `1.2.3-alpha.4` |
| `--show-variable MajorMinorPatch` | `1.2.3` |
| `-o json` | Full JSON with all version variables |
| `--show-variable FullSemVer` | `1.2.3-alpha.4+5` |

Available variables: `Major`, `Minor`, `Patch`, `PreReleaseLabel`, `PreReleaseNumber`, `BuildMetaData`, `SemVer`, `FullSemVer`, `MajorMinorPatch`, `CommitsSinceVersionSource`, `Sha`, `ShortSha`, `CommitDate`, `CommitTag`.

---

## Go Library API

Embed versioning directly in Go applications — no CLI subprocess needed:

```go
import "github.com/MyCarrier-DevOps/go-gitsemver/pkg/sdk"

result, err := sdk.Calculate(sdk.LocalOptions{
    Path:    "/path/to/repo",
    Explain: true,
})

fmt.Println(result.Variables["SemVer"])       // "1.2.3-alpha.4"
fmt.Println(result.ExplainResult.FinalVersion) // "1.2.3-alpha.4+5"
```

Also supports remote calculation via GitHub API:

```go
result, err := sdk.CalculateRemote(sdk.RemoteOptions{
    Owner:   "MyCarrier-DevOps",
    Repo:    "my-service",
    Token:   os.Getenv("GITHUB_TOKEN"),
    Explain: true,
})
```

---

## Remote Mode

Calculate versions without cloning the repository:

```bash
go-gitsemver remote MyCarrier-DevOps/my-service --token $GITHUB_TOKEN

# Point to a specific config file in the remote repo
go-gitsemver remote MyCarrier-DevOps/my-service --remote-config-path .github/GitVersion.yml
```

Uses the GitHub API to fetch branches, tags, and commits. Supports `--remote-config-path` to fetch a specific config file from the remote repo, or auto-detects from `.github/` and repo root. Ideal for CI pipelines where a full clone is expensive.

---

## Architecture

```
cmd/                     CLI entry points (calculate, remote)
pkg/sdk/           Public Go library API
internal/
  calculator/            Version calculation engine
    nextversion.go       Main pipeline: strategy → increment → pre-release → result
    increment.go         Commit analysis and bump detection
    mainline.go          Mainline mode calculator
  strategy/              Base version strategies (6 strategies)
  git/                   Git abstraction layer (local + GitHub API)
  config/                YAML config loading and branch matching
  context/               Runtime context (current branch, commit, config)
  output/                Formatting (variables, JSON, explain)
  semver/                Semantic version types and parsing
e2e/                     End-to-end tests with real Git repos
```

---

## Test Coverage

- **589 tests** across unit, integration, and end-to-end suites
- **85% overall coverage**
- E2E tests create real Git repositories with branches, tags, and commits to validate the full pipeline
- Mock-based unit tests for isolated strategy and calculator testing

---

## Key Design Decisions

- **Pure Git analysis** — no version files, no build artifacts, no external state
- **Strategy pattern** — each version source is an independent, testable strategy
- **Branch config via regex** — flexible matching without hardcoding branch names
- **Stderr for explain** — stdout stays machine-parseable; diagnostics go to stderr
- **GitVersion.yml compatibility** — configuration format aligns with the established GitVersion ecosystem
