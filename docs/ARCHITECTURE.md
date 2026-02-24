# gitsemver Architecture

This document describes the architecture of gitsemver, a semantic versioning tool that automatically calculates versions from git history.

---

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     CLI (cmd/)                           │
│     Root command, remote subcommand, output formatting   │
└────────────────────────┬────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────┐
│              Go Library (pkg/gitsemver/)                  │
│  Calculate(LocalOptions), CalculateRemote(RemoteOptions) │
└────────────────────────┬────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────┐
│               GitVersionContext (context/)                │
│  CurrentBranch, CurrentCommit, Config, TaggedVersion,    │
│  NumberOfUncommittedChanges                              │
└────────────────────────┬────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────┐
│              NextVersionCalculator (calculator/)          │
│  Orchestrates: strategies → increment → mode branching   │
│  → pre-release tag → build metadata → final version      │
└────────┬───────────────┬──────────────────┬─────────────┘
         │               │                  │
┌────────▼───┐  ┌───────▼──────┐  ┌────────▼─────────┐
│   Base      │  │  Increment   │  │   Mainline        │
│   Version   │  │  Strategy    │  │   Version         │
│  Calculator │  │  Finder      │  │  Calculator       │
│ (6 strats)  │  │              │  │                   │
└─────────────┘  └──────────────┘  └───────────────────┘
         │               │                  │
┌────────▼───────────────▼──────────────────▼─────────┐
│                  RepositoryStore (git/)               │
│   Tags, commits, branches, merge history queries     │
│        via go-git (local) or GitHub API (remote)     │
└──────────────────────────────────────────────────────┘
```

---

## Package Structure

```
go-gitsemver/
├── cmd/                        # CLI commands (cobra)
│   ├── root.go                 # Root command with persistent flags
│   ├── calculate.go            # Default command: full calculation pipeline
│   ├── remote.go               # Remote subcommand: version via GitHub API
│   └── version.go              # Version subcommand
├── pkg/
│   └── gitsemver/              # Public Go library API
│       ├── gitsemver.go        # Calculate() and CalculateRemote() functions
│       └── gitsemver_test.go   # Unit tests with httptest mocks
├── example/
│   └── main.go                 # Runnable example showing library usage
├── internal/
│   ├── semver/                 # Semantic version types (immutable)
│   │   ├── version.go          # SemanticVersion: parse, compare, increment
│   │   ├── prereleasetag.go    # PreReleaseTag: name + number
│   │   ├── buildmetadata.go    # BuildMetaData: commits, branch, SHA, date
│   │   ├── enums.go            # VersionField, IncrementStrategy, VersioningMode, etc.
│   │   ├── formatvalues.go     # ComputeFormatValues() pure function
│   │   └── yaml.go             # YAML unmarshaling for enum types
│   ├── config/                 # YAML configuration and resolution
│   │   ├── config.go           # Root Config struct with YAML tags
│   │   ├── branch.go           # BranchConfig struct
│   │   ├── defaults.go         # 8 default branch configurations
│   │   ├── builder.go          # Config merging and finalization
│   │   ├── effective.go        # EffectiveConfiguration (all fields resolved)
│   │   ├── loader.go           # YAML file loading
│   │   ├── extensions.go       # Branch matching, tag resolution
│   │   └── ignore.go           # SHA and date ignore filters
│   ├── git/                    # Git abstraction layer
│   │   ├── interfaces.go       # Repository interface (15 methods)
│   │   ├── types.go            # Commit, Branch, Tag, ObjectID, VersionTag
│   │   ├── gogit.go            # go-git implementation of Repository
│   │   ├── repostore.go        # RepositoryStore: domain-level queries
│   │   ├── mergemessage.go     # Merge/squash message parsing (8 formats)
│   │   └── mock.go             # MockRepository for testing
│   ├── github/                 # GitHub API provider (remote mode)
│   │   ├── client.go           # Auth resolution, GitHub client factory
│   │   ├── repository.go       # GitHubRepository: implements git.Repository
│   │   ├── graphql.go          # Batch GraphQL queries for branches and tags
│   │   └── cache.go            # In-memory API response cache
│   ├── context/                # Immutable git state snapshot
│   │   ├── context.go          # GitVersionContext struct
│   │   └── factory.go          # NewContext() factory
│   ├── strategy/               # 6 version discovery strategies
│   │   ├── base.go             # BaseVersion type, VersionStrategy interface
│   │   ├── confignextversion.go
│   │   ├── taggedcommit.go
│   │   ├── mergemessage.go
│   │   ├── branchname.go
│   │   ├── trackrelease.go
│   │   ├── fallback.go
│   │   └── strategies.go       # AllStrategies() registry
│   ├── calculator/             # Version calculation pipeline
│   │   ├── nextversion.go      # NextVersionCalculator: full pipeline
│   │   ├── baseversion.go      # BaseVersionCalculator: strategy selection
│   │   ├── mainline.go         # MainlineVersionCalculator
│   │   └── increment.go        # IncrementStrategyFinder: commit scanning
│   ├── output/                 # Output formatting
│   │   ├── variables.go        # GetVariables(): compute all output vars
│   │   ├── promote.go          # PromoteCommitsToPreRelease(): CD mode
│   │   └── json.go             # JSON, text, single-variable output
│   └── testutil/               # Test helpers (temp repos, commits, tags)
├── e2e/                        # End-to-end tests
├── docs/                       # Documentation
├── main.go                     # Entry point
└── go.mod
```

---

## Calculation Flow

1. **Open Repository** — Open the git repo at the specified path using go-git (local), or connect via GitHub API (remote)
2. **Load Configuration** — Search for `gitsemver.yml` or `GitVersion.yml` (locally or via API), merge with defaults
3. **Build Context** — Resolve current branch, commit, check for version tags, count uncommitted changes
4. **Resolve Branch Config** — Match branch name against config regexes (priority-ordered), produce `EffectiveConfiguration`
5. **Run Strategies** — Execute all 6 version strategies to collect candidate base versions
6. **Select Winner** — Rank candidates by effective version, tie-break by oldest source commit
7. **Apply Increment** — Scan commit messages for Conventional Commits / bump directives, determine increment
8. **Branch on Mode** — Mainline uses aggregate or per-commit increment; Standard applies single increment
9. **Update Pre-Release Tag** — Apply branch-specific label, auto-increment number
10. **Build Metadata** — Compute commits-since-tag, branch, SHA, date
11. **Output** — Generate 25+ output variables, write as JSON/text/single-variable

---

## Key Interfaces

### Repository

```go
type Repository interface {
    Path() string
    IsHeadDetached() bool
    Head() (Branch, error)
    Branches() ([]Branch, error)
    Tags() ([]Tag, error)
    CommitLog(from, to string) ([]Commit, error)
    FindMergeBase(sha1, sha2 string) (string, error)
    // ... 15 methods total
}
```

Implemented by `GoGitRepository` (local, using `go-git/go-git/v5`) and `GitHubRepository` (remote, using GitHub REST + GraphQL APIs). `MockRepository` is provided for unit testing.

### VersionStrategy

```go
type VersionStrategy interface {
    Name() string
    GetBaseVersions(ctx, ec, explain) ([]BaseVersion, error)
}
```

Six implementations: ConfigNextVersion, TaggedCommit, MergeMessage, VersionInBranchName, TrackReleaseBranches, Fallback.

---

## Design Principles

### Immutable Types

`SemanticVersion`, `PreReleaseTag`, and `BuildMetaData` are immutable value types. All operations return new values:

```go
newVer := ver.IncrementField(semver.VersionFieldMinor)  // returns new version
newVer := ver.WithPreReleaseTag(tag)                     // returns new version
```

No hidden state mutations through the calculation pipeline.

### Single-Increment Pipeline

Candidate base versions are ranked by computing an effective version for comparison (without mutation). The winning candidate is incremented exactly once. No double-increment.

### Pure Functions

Output computation is a pure function: `ComputeFormatValues(ver, config) → map[string]string`. Commit promotion for ContinuousDeployment mode is also a pure function: `PromoteCommitsToPreRelease(ver, mode, fallbackTag) → ver`.

### Priority-Based Branch Matching

Branch configs are matched by regex with explicit priority ordering (main=100, release=90, hotfix=80, etc.). Highest priority wins. A catch-all `unknown` config (priority=0) ensures every branch gets a configuration.

---

## Package Dependency Graph

```
semver (zero external deps)
  ↓
config (→ semver, gopkg.in/yaml.v3)
  ↓
git (→ semver, config, go-git/go-git/v5)
  ↓
github (→ git, go-github/v68, oauth2, ghinstallation)
  ↓
context (→ semver, config, git)
  ↓
strategy (→ semver, config, git, context)
  ↓
calculator (→ semver, config, git, context, strategy)
  ↓
output (→ semver, config)
  ↓
pkg/gitsemver (→ all internal packages — public library API)
  ↓
cmd (→ all internal packages, github.com/spf13/cobra)
```

`pkg/gitsemver/` lives inside the same Go module, so it can import `internal/` packages freely. External consumers only see the public API surface: `Calculate()`, `CalculateRemote()`, `LocalOptions`, `RemoteOptions`, and `Result`.

---

## External Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/go-git/go-git/v5` | Pure-Go git operations (local mode) |
| `github.com/google/go-github/v68` | GitHub REST API client (remote mode) |
| `golang.org/x/oauth2` | Token-based HTTP auth for GitHub API |
| `github.com/bradleyfalzon/ghinstallation/v2` | GitHub App JWT authentication |
| `github.com/spf13/cobra` | CLI framework |
| `gopkg.in/yaml.v3` | YAML configuration parsing |
| `github.com/stretchr/testify` | Test assertions (require only) |
