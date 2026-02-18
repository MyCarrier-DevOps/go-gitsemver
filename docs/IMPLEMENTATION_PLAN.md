# Implementation Plan

## Overview

Rewriting GitVersion (C# v5.12.0) in Go as `go-gitsemver`. The goal is a well-organized, testable codebase using adapter patterns, with >85% code coverage.

8 phases: Bootstrap → Semver Types → Config → Git Adapter → Context/Strategies → Calculators → Output → CLI → Integration Tests

---

## Design Improvements Over GitVersion v5.12.0

These changes address pain points, missing features, and architectural issues identified in the C# reference implementation. Each improvement is tagged (DI-1 through DI-12) and referenced in the phases where it's applied.

### DI-1: Immutable SemanticVersion

All operations on `SemanticVersion` return new values instead of mutating in place. The C# code mutates `PreReleaseTag` and `BuildMetaData` throughout the pipeline (in NextVersionCalculator, VariableProvider, MainlineVersionCalculator), making it hard to trace state. Go types will be value-based — methods like `WithPreReleaseTag()`, `WithBuildMetaData()` return new structs.

### DI-2: Separate IncrementField from IncrementPreRelease

The C# `IncrementVersion(Major)` does two unrelated things depending on pre-release state:
- Without pre-release: `1.2.3` → `2.0.0` (bumps field)
- With pre-release: `1.2.3-beta.3` → `1.2.3-beta.4` (ignores field, bumps pre-release number)

Split into explicit methods: `IncrementField(Major/Minor/Patch)` always bumps the version field; `IncrementPreRelease()` always bumps the pre-release number. Caller decides which to use.

### DI-3: Single-Increment Pipeline

The C# `BaseVersionCalculator` tentatively increments every candidate via `MaybeIncrement` just to compare them, then `NextVersionCalculator` increments the winner again. This double-increment is error-prone. Instead: compute an `EffectiveVersion` for ranking (without mutation), select the winner, then increment once.

### DI-4: Commit Promotion as Pipeline Stage

The C# `PreReleaseTag.PromotedFromCommits` is a mutable boolean flag set during `VariableProvider` processing that changes how `HasTag()` reports. Replace with an explicit transformation function in the output pipeline: `PromoteCommitsToPreRelease(ver, mode) → ver`. No flag, no hidden state.

### DI-5: Named Methods Instead of Format Specifiers

Replace cryptic single-character format specifiers (`"s"`, `"f"`, `"j"`, `"i"`) with named methods: `SemVer()`, `FullSemVer()`, `InformationalVersion()`, etc. Self-documenting and IDE-friendly.

### DI-6: FormatValues as Pure Function

The C# `SemanticVersionFormatValues` is a class with 30+ computed properties. Replace with a pure function: `ComputeFormatValues(ver SemanticVersion, config EffectiveConfig) map[string]string`. No state, easy to test.

### DI-7: Conventional Commits Support

GitVersion only supports custom `+semver:` directives. Add first-class support for [Conventional Commits](https://www.conventionalcommits.org/):
- `feat:` / `feat(scope):` → Minor
- `fix:` / `fix(scope):` → Patch
- `feat!:` / `BREAKING CHANGE:` footer → Major

Also replace the verbose `+semver:` syntax with simpler `bump` directives:
- `bump major` → Major
- `bump minor` → Minor
- `bump patch` → Patch
- `bump none` / `bump skip` → None

Support both conventions simultaneously. Configurable via `commit-message-convention: conventional-commits | bump-directive | both`

### DI-8: Squash Merge Awareness

GitVersion's `MergeMessageVersionStrategy` relies on merge commits having multiple parents. Squash merges (very common on GitHub/GitLab) produce single-parent commits. Add parsing for squash merge formats:
- GitHub: `feat: thing (#123)` — extract PR number, optionally query branch name
- GitLab: `Merge branch 'feature/x' into 'main'` (squash variant)
- Generic: configurable squash message patterns

### DI-9: Explain Mode

Add `--explain` flag that outputs the full decision tree:

```
Strategies evaluated:
  TaggedCommit: 1.2.0 from tag v1.2.0 (3 commits ago) → effective 1.3.0
  Fallback: 0.1.0
Selected: TaggedCommit (1.3.0, oldest source)
Increment: Minor (from commit "feat: add auth" via Conventional Commits)
Pre-release: feature-login.1
Result: 1.3.0-feature-login.1+3
```

Implemented as a structured log/trace that the calculator pipeline emits, rendered by the CLI.

### DI-10: Simplified Mainline Calculation

The C# `MainlineVersionCalculator` walks the entire commit graph and increments the version for every commit individually. This causes version inflation (5 minor commits → `1.5.0`) and loses semantic meaning.

Replace with aggregate-increment approach:
1. Find the latest semver tag (e.g., `v1.2.0`)
2. Collect all commits since that tag
3. Determine the **single highest** increment type from commit messages (conventional commits / `bump` directives)
4. Apply that increment **once** → `1.3.0`
5. Commit count since tag goes into build metadata for uniqueness → `1.3.0+5`

To force a specific version bump, users can:
- Tag a commit manually (e.g., `v2.0.0`) — TaggedCommitStrategy handles this
- Use `bump major` or `feat!:` in a commit message

No per-commit incrementing. No graph walking. Version numbers stay semantically meaningful.

### DI-11: Monorepo-Ready Interfaces

Even though full monorepo support is deferred, design interfaces to allow future path-scoping:
- `Repository` interface methods accept optional path filters
- Tag prefix supports path-based patterns (`service-a/v1.0.0`)
- Commit scanning can be scoped to paths that changed

### DI-12: Branch Match Priority

Replace first-regex-match-wins with explicit priority ordering. Each branch config gets a `priority` field (default based on specificity). Most specific match wins. Fallback to first match for backward compatibility.

---

## Phase 0: Project Bootstrap

**Status:** Complete

- `go.mod` (go 1.26), `main.go` placeholder
- `.github/instructions/CLAUDE.md` with build/test instructions
- `.github/instructions/go.instructions.md` with Go coding standards
- `makefile` rewritten for single-module build
- `.golangci.yml` updated (local-prefixes: go-gitsemver)
- `.github/workflows/ci.yaml` with test/lint/vuln/build/status-check + release-artifacts (GitHub Release on `v*` tags)
- `.github/release.yml` for changelog categories

---

## Phase 1: Core Semver Types — `internal/semver/`

**Goal:** Pure, immutable value types with zero external dependencies. Foundational for everything else.

**Design improvements:** DI-1 (immutable types), DI-2 (separate increment methods), DI-5 (named format methods), DI-6 (pure format function)

### Files

| File | Contents |
|------|----------|
| `enums.go` | `VersionField` (None/Patch/Minor/Major), `IncrementStrategy` (None/Major/Minor/Patch/Inherit), `VersioningMode` (ContinuousDelivery/ContinuousDeployment/Mainline), `CommitMessageIncrementMode` (Enabled/Disabled/MergeMessageOnly), `CommitMessageConvention` (ConventionalCommits/BumpDirective/Both) |
| `prereleasetag.go` | `PreReleaseTag` struct (Name string, Number *int64). Immutable. `HasTag()`, `CompareTo()`, `WithNumber()`, `WithName()`. Named format methods: `String()` (dotted: `beta.4`), `Legacy()` (no dot: `beta4`), `LegacyPadded(pad)` (`beta0004`) |
| `buildmetadata.go` | `BuildMetaData` struct (CommitsSinceTag *int64, Branch, Sha, ShortSha, VersionSourceSha string, CommitDate time.Time, CommitsSinceVersionSource int64, UncommittedChanges int64). Immutable. `String()`, `ShortString()`, `FullString()`, `Padded(pad)` |
| `version.go` | `SemanticVersion` struct (Major, Minor, Patch int64, PreReleaseTag, BuildMetaData). Immutable. `Parse()`/`TryParse()` with tag-prefix regex, `CompareTo()`, `IncrementField(VersionField)` (always bumps version), `IncrementPreRelease()` (always bumps pre-release number), `WithPreReleaseTag()`, `WithBuildMetaData()`. Named format methods: `SemVer()`, `FullSemVer()`, `LegacySemVer()`, `LegacySemVerPadded()`, `InformationalVersion()` |
| `formatvalues.go` | `ComputeFormatValues(ver SemanticVersion, cfg FormatConfig) map[string]string` — pure function computing all 30+ output variable strings. `FormatConfig` holds padding, commit-date-format, tag-pre-release-weight |
| `*_test.go` | Table-driven tests for each file. Target: >95% coverage |

### Key behaviors

- `IncrementField(Major)`: `1.2.3` → `2.0.0` (zeros lower fields, strips pre-release)
- `IncrementField(Minor)`: `1.2.3` → `1.3.0`
- `IncrementField(Patch)`: `1.2.3` → `1.2.4`
- `IncrementField(None)`: no change
- `IncrementPreRelease()`: `1.2.3-beta.3` → `1.2.3-beta.4`; panics if no pre-release number
- `CompareTo`: stable > pre-release; then name (case-insensitive); then number
- `HasTag()`: true when Name non-empty OR Number has value
- Parse regex: `^(?<Major>\d+)(\.(?<Minor>\d+))?(\.(?<Patch>\d+))?(\.(?<FourthPart>\d+))?(-(?<Tag>[^\+]*))?(\+(?<BuildMetaData>.*))?$`

### Dependencies

stdlib only (`regexp`, `fmt`, `strconv`, `strings`, `time`)

---

## Phase 2: Configuration — `internal/config/`

**Goal:** YAML config loading, default branch configs, config merging, effective configuration resolution.

**Design improvements:** DI-7 (conventional commits config), DI-12 (branch priority)

### Files

| File | Contents |
|------|----------|
| `config.go` | `Config` struct with YAML tags matching `.gitversion.yml` format. All optional fields as pointers. New field: `CommitMessageConvention *CommitMessageConvention` |
| `branch.go` | `BranchConfig` struct (all pointer fields for merge semantics), `MergeTo()` method. New field: `Priority *int` |
| `defaults.go` | `CreateDefaultConfiguration()` — 7 branch defaults (main, develop, release, feature, hotfix, pull-request, support) with exact regex, increment, tag, source-branches, boolean flags. Default priorities: main=100, release=90, hotfix=80, support=70, develop=60, feature=50, pull-request=40 |
| `builder.go` | `Builder` — Add overrides, Build (apply overrides → finalize → validate) |
| `effective.go` | `EffectiveConfiguration` — resolved config with all fields guaranteed non-nil |
| `loader.go` | Load from `.gitversion.yml`, custom YAML unmarshalers for enums |
| `ignore.go` | `IgnoreConfig`, `ShaVersionFilter`, `MinDateVersionFilter` |
| `extensions.go` | `GetBranchConfiguration()` (priority-ordered regex match, highest priority wins), `GetReleaseBranchConfig()`, `GetBranchSpecificTag()` ({BranchName} replacement) |
| `*_test.go` | YAML loading, defaults verification, builder merging, validation, priority matching. Target: >90% |

### Dependencies

`gopkg.in/yaml.v3`, `internal/semver`

---

## Phase 3: Git Adapter Layer — `internal/git/`

**Goal:** Interface + go-git implementation + mock for testing.

**Design improvements:** DI-8 (squash merge parsing), DI-11 (monorepo-ready interfaces)

### Files

| File | Contents |
|------|----------|
| `interfaces.go` | `Repository`, `Branch`, `Commit`, `Tag`, `ObjectID` interfaces — narrow API surface. Methods like `CommitLog()`, `Tags()` accept optional `PathFilter` parameter |
| `gogit.go` | go-git implementation of `Repository` interface |
| `mock.go` | `MockRepository` with preconfigurable return values for unit tests |
| `repostore.go` | `RepositoryStore` — higher-level queries: `GetValidVersionTags`, `GetVersionTagsOnBranch`, `GetCurrentCommitTaggedVersion`, `FindMainBranch`, `GetReleaseBranches`, `GetBranchesContainingCommit`, `GetBaseVersionSource`, `GetMergeBaseCommits` |
| `mergemessage.go` | `MergeMessage` parser — 6 built-in merge formats + squash merge formats (GitHub squash `feat: thing (#123)`, generic squash patterns) + custom formats |
| `*_test.go` | Merge message parsing (table-driven, all formats including squash), repostore with mock. Target: >85% |

### Key design decisions

- `Repository` interface is deliberately narrow (git primitives only)
- `RepositoryStore` wraps `Repository` + `Config` for domain-level queries
- Merge-base: implement BFS common-ancestor in Go (go-git lacks built-in merge-base)
- Path filter parameter is `...PathFilter` (variadic optional) to keep the API clean when not scoping

### Dependencies

`github.com/go-git/go-git/v5`, `internal/semver`, `internal/config`

---

## Phase 4: Context & Version Strategies — `internal/context/`, `internal/strategy/`

**Goal:** GitVersionContext factory and all version strategies.

**Design improvements:** DI-8 (squash merge strategy), DI-9 (explanation traces)

### Context files

| File | Contents |
|------|----------|
| `context.go` | `GitVersionContext` struct (CurrentBranch, CurrentCommit, FullConfiguration, CurrentCommitTaggedVersion, NumberOfUncommittedChanges, IsCurrentCommitTagged) |
| `factory.go` | `NewContext()` — resolve branch, get commit, check tags, count uncommitted changes |

### Strategy files

| File | Contents |
|------|----------|
| `base.go` | `BaseVersion` struct (Source, ShouldIncrement, SemanticVersion, BaseVersionSource, BranchNameOverride), `VersionStrategy` interface, `Explanation` struct for tracing |
| `confignextversion.go` | From `next-version` config. ShouldIncrement=false, source=nil |
| `taggedcommit.go` | From git tags on branch. ShouldIncrement=true if tag not on current commit |
| `mergemessage.go` | From merge commit messages AND squash merge messages. Parses all formats, checks release branch |
| `branchname.go` | From release branch name. Splits by `/`/`-`, parses segments as semver |
| `trackrelease.go` | For develop-like branches. Combines release branch versions + main tags |
| `fallback.go` | Always returns 0.1.0 from root commit |
| `*_test.go` | Each strategy tested with MockRepository, including squash merge scenarios. Target: >90% |

### Dependencies

`internal/semver`, `internal/config`, `internal/git`, `internal/context`

---

## Phase 5: Calculators — `internal/calculator/`

**Goal:** Core orchestration logic with single-increment pipeline and Conventional Commits support.

**Design improvements:** DI-2 (explicit increment methods), DI-3 (single-increment pipeline), DI-7 (conventional commits), DI-9 (explanation traces), DI-10 (simplified mainline)

### Files

| File | Contents |
|------|----------|
| `increment.go` | `IncrementStrategyFinder` — scan commit messages for version bump directives. Supports: (1) Conventional Commits (`feat:`, `fix:`, `feat!:`, `BREAKING CHANGE:` footer), (2) `bump` directives (`bump major/minor/patch/none`), (3) both (configurable). `DetermineIncrementedField` logic: cap at Minor for <1.0.0, floor at branch default |
| `baseversion.go` | `BaseVersionCalculator` — run all strategies, compute effective version for ranking (no mutation), select max, tie-break by oldest source, fix deleted release branch source. No double-increment |
| `nextversion.go` | `NextVersionCalculator` — orchestrate: handle tagged commit → get base version → branch to Mainline or Standard → single increment using `IncrementField()`/`IncrementPreRelease()` → UpdatePreReleaseTag → compare with tagged version. Emits `Explanation` |
| `mainline.go` | `MainlineVersionCalculator` — aggregate-increment: find latest tag → collect commits since tag → scan all commit messages for highest increment type → apply once. Commit count goes into build metadata. No per-commit incrementing, no graph walking |
| `*_test.go` | Mock-based tests for each calculator, including Conventional Commits scenarios. Target: >85% |

### Dependencies

`internal/semver`, `internal/config`, `internal/git`, `internal/context`, `internal/strategy`

---

## Phase 6: Output — `internal/output/`

**Goal:** Variable generation, JSON output.

**Design improvements:** DI-4 (commit promotion as pure function), DI-6 (pure format function)

### Files

| File | Contents |
|------|----------|
| `variables.go` | `GetVariables()` — applies versioning mode via `PromoteCommitsToPreRelease()` (ContinuousDeployment: promote CommitsSinceTag to pre-release number; Mainline: same), then calls `ComputeFormatValues()` |
| `promote.go` | `PromoteCommitsToPreRelease(ver SemanticVersion, mode VersioningMode, fallbackTag string) SemanticVersion` — pure function, returns new version with commits promoted to pre-release number. No mutable flags |
| `json.go` | JSON pretty-print output, single-variable output |
| `*_test.go` | All three modes tested, promotion function tested independently. Target: >90% |

### Dependencies

`internal/semver`, `internal/config`

---

## Phase 7: CLI — `cmd/`

**Goal:** Wire everything together with cobra.

**Design improvements:** DI-9 (explain mode)

### Files

| File | Contents |
|------|----------|
| `root.go` | Root command, global flags (--path, --branch, --commit, --output, --show-variable, --show-config, --override-config, --no-cache, --verbosity, --explain) |
| `calculate.go` | Default command: open repo → load config → build context → run strategies → calculate → output. When `--explain`: render decision tree |
| `explain.go` | `Explanation` renderer — formats the structured trace into human-readable output showing strategies evaluated, effective versions, winner selection, increment source, pre-release update, and final result |
| `showconfig.go` | Print effective config as YAML |
| `version.go` | Print binary version (injected via ldflags) |
| `*_test.go` | Flag parsing, explain output formatting, integration test with temp git repo. Target: >75% |

### Dependencies

`github.com/spf13/cobra`, all internal packages

---

## Phase 8: Integration Tests & Coverage Hardening

**Goal:** End-to-end tests, edge cases, push coverage above 85%.

### Scenarios to test

- Full GitFlow: main → develop → feature → release → merge back → verify versions at each point
- Trunk-based (Mainline mode): direct commits, merge commits, per-commit increment
- ContinuousDeployment: CommitsSinceTag promotion to pre-release number
- Conventional Commits: `feat:`, `fix:`, `feat!:`, `BREAKING CHANGE:` footer in various positions
- Squash merges: GitHub-style squash messages, verify version extraction
- Explain mode: verify output format and completeness
- Branch priority: overlapping regex patterns, verify highest priority wins
- Edge cases: detached HEAD, no tags (fallback), multiple tags on same commit, deleted release branch, forward merge, <1.0.0 capping, next-version config, empty config

### Test infrastructure

`internal/testutil/` — helpers to create temp git repos: `CreateTestRepo`, `AddCommit`, `CreateTag`, `CreateBranch`, `MergeBranch`, `SquashMergeBranch`

---

## Package Dependency Graph

```
Phase 0: Bootstrap (go.mod, main.go, CI, Makefile) ✅
    ↓
Phase 1: internal/semver (zero deps)
    ↓
Phase 2: internal/config (→ semver)
    ↓
Phase 3: internal/git (→ semver, config)
    ↓
Phase 4: internal/context + internal/strategy (→ semver, config, git)
    ↓
Phase 5: internal/calculator (→ semver, config, git, context, strategy)
    ↓
Phase 6: internal/output (→ semver, config)
    ↓
Phase 7: cmd/ (→ everything)
    ↓
Phase 8: Integration tests
```

---

## External Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/go-git/go-git/v5` | Git operations |
| `github.com/spf13/cobra` | CLI framework |
| `gopkg.in/yaml.v3` | YAML config parsing |
| `github.com/stretchr/testify` | Test assertions (require only, NOT assert) |

---

## Verification

After each phase:

1. `make tidy` — module dependencies clean
2. `make fmt` — code formatted per gofumpt/goimports
3. `make lint` — passes golangci-lint
4. `make test` — all tests pass, coverage reported
5. Coverage check — verify >85% overall
