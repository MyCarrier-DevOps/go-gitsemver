# gitsemver vs GitVersion — What's Better

This document compares gitsemver (Go) against GitVersion v5.12.0 (C#/.NET) and highlights the improvements.

---

## At a Glance

| Area | GitVersion v5.12.0 | gitsemver |
|------|---------------------|-----------|
| Language | C# / .NET 6 | Go (single binary, zero runtime deps) |
| Git library | LibGit2Sharp (native C lib) | go-git (pure Go) |
| Binary size | ~50MB+ with .NET runtime | ~10-15MB standalone |
| Cross-platform | Requires .NET SDK or self-contained publish | Single static binary per OS/arch |
| Install | `dotnet tool install`, Chocolatey, Homebrew | Download binary, `go install`, or CI action |
| Config format | `.gitversion.yml` | `gitsemver.yml` |

---

## Design Improvements

### 1. Immutable Version Types (DI-1)

**GitVersion:** Mutates `SemanticVersion` in place throughout the pipeline. `PreReleaseTag` and `BuildMetaData` are modified by `NextVersionCalculator`, `VariableProvider`, and `MainlineVersionCalculator`. Hard to trace which code changed what.

**gitsemver:** All types are immutable value types. Methods like `WithPreReleaseTag()` and `WithBuildMetaData()` return new structs. No hidden state mutations.

---

### 2. Clear Increment Operations (DI-2)

**GitVersion:** `IncrementVersion(Major)` does two completely different things depending on whether the version has a pre-release tag:
- Without pre-release: `1.2.3` → `2.0.0` (bumps the version field)
- With pre-release: `1.2.3-beta.3` → `1.2.3-beta.4` (ignores the field argument, bumps pre-release number)

This implicit behavior caused bugs in the C# codebase.

**gitsemver:** Two explicit methods:
- `IncrementField(Major/Minor/Patch)` — always bumps the version field
- `IncrementPreRelease()` — always bumps the pre-release number

The caller decides which to use. No ambiguity.

---

### 3. No Double-Increment (DI-3)

**GitVersion:** The `BaseVersionCalculator` tentatively increments every candidate via `MaybeIncrement` just to compare them. Then `NextVersionCalculator` increments the winner *again*. This double-increment is a source of subtle version miscalculations.

**gitsemver:** Single-increment pipeline. Candidates are ranked by computing an effective version (without mutation). The winner is incremented exactly once.

---

### 4. Explicit Commit Promotion (DI-4)

**GitVersion:** Uses a mutable boolean `PreReleaseTag.PromotedFromCommits` flag that changes how `HasTag()` reports. This flag is set deep inside `VariableProvider`, making the behavior non-obvious.

**gitsemver:** Commit promotion is an explicit pure function: `PromoteCommitsToPreRelease(ver, mode) → ver`. No flag, no hidden state. Easy to test and reason about.

---

### 5. Self-Documenting API (DI-5)

**GitVersion:** Uses cryptic single-character format specifiers: `"s"` for SemVer, `"f"` for FullSemVer, `"j"` for JSON, `"i"` for InformationalVersion.

**gitsemver:** Named methods: `SemVer()`, `FullSemVer()`, `InformationalVersion()`. Self-documenting and IDE-friendly.

---

### 6. Pure Output Function (DI-6)

**GitVersion:** `SemanticVersionFormatValues` is a class with 30+ computed properties, each with its own logic and potential side effects.

**gitsemver:** Single pure function: `ComputeFormatValues(ver, config) → map[string]string`. No state, trivial to test.

---

### 7. Conventional Commits Support (DI-7)

**GitVersion:** Only supports its own `+semver: breaking/feature/fix` directives. No Conventional Commits support.

**gitsemver:** First-class support for both:
- **Conventional Commits:** `feat:` → Minor, `fix:` → Patch, `feat!:` / `BREAKING CHANGE:` → Major
- **Bump directives:** `bump major/minor/patch/none` (simpler than `+semver:`)
- Configurable: `commit-message-convention: conventional-commits | bump-directive | both`

---

### 8. Squash Merge Awareness (DI-8)

**GitVersion:** Relies on merge commits having two parents. Squash merges (single-parent commits) are invisible to the `MergeMessageVersionStrategy`. This is a major gap since squash merges are the default on GitHub and GitLab.

**gitsemver:** Parses squash merge formats out of the box:
- GitHub: `feat: add login page (#123)`
- GitLab: `Merge branch 'feature/x' into 'main'`
- Bitbucket: `Merged in feature/auth (pull request #123)`
- Custom patterns via config

---

### 9. Explain Mode (DI-9)

**GitVersion:** No way to understand *why* a specific version was calculated. Debugging requires reading source code or enabling verbose logging and parsing the output.

**gitsemver:** `--explain` flag outputs a structured decision tree:

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

---

### 10. Simplified Mainline Calculation (DI-10)

**GitVersion:** The `MainlineVersionCalculator` walks the entire commit graph and increments the version for **every commit individually**. This causes version inflation:

```
v1.0.0 → fix → fix → feat → fix → fix
GitVersion Mainline: 1.0.1 → 1.0.2 → 1.1.0 → 1.1.1 → 1.1.2
```

Five commits, five increments. Version numbers lose semantic meaning.

**gitsemver:** Aggregate-increment approach:

```
v1.0.0 → fix → fix → feat → fix → fix
gitsemver Mainline: 1.1.0+5
```

1. Find the highest increment from all commits since the last tag (Minor, from `feat`)
2. Apply it **once**
3. Commit count goes into build metadata for uniqueness

Version numbers stay semantically meaningful. To force a version jump, use `bump major` or `feat!:` in a commit message.

---

### 11. Monorepo-Ready Interfaces (DI-11)

**GitVersion:** Not designed for monorepos. Git operations don't support path scoping.

**gitsemver:** Interfaces accept optional path filters. Tag prefix supports path-based patterns. Commit scanning can be scoped to changed paths. Full monorepo support is a future addition, but the foundation is in place.

---

### 12. Branch Match Priority (DI-12)

**GitVersion:** First-regex-match-wins. Branch config order in the YAML file determines which config is used. If `feature/*` appears before a more specific pattern, it silently wins.

**gitsemver:** Explicit priority ordering. Each branch config has a `priority` field. Highest priority wins. Default priorities are based on specificity (main=100, release=90, ... unknown=0). No surprises.

---

## Additional Improvements

### Configurable Base Version

**GitVersion:** Fallback strategy hardcodes `0.1.0` when no tags exist. The only workaround is `next-version`, which is meant to be temporary.

**gitsemver:** `base-version` config option (default: `0.1.0`). Permanent setting — always applies when no tags exist. Separate from `next-version` which is a temporary override.

```yaml
base-version: 1.0.0
```

### Shallow Clone Protection

**GitVersion:** May produce incorrect versions on shallow clones without warning.

**gitsemver:** Detects shallow clones and exits with a fatal error by default. The `--allow-shallow` flag explicitly opts into running with potentially incomplete history. Clear error message suggests `git fetch --unshallow`.

### Catch-All Branch Config

**GitVersion:** Branches not matching any regex have no config. Behavior depends on fallback logic buried in the code.

**gitsemver:** Built-in `unknown` branch config (`.*` regex, priority 0) catches any branch that doesn't match a known pattern. Treated like a feature branch with `{BranchName}` as the pre-release tag. No surprises, no errors.

### No .NET Dependency

**GitVersion:** Requires .NET SDK or a self-contained publish (~50MB+). CI pipelines need a .NET setup step.

**gitsemver:** Single static binary. Download and run. ~10-15MB. No runtime dependencies.

### No C#-Specific Features

**GitVersion:** Includes assembly info updates, NuGet versioning, .NET project file updates — features only useful in the .NET ecosystem.

**gitsemver:** Generic tool. Works with any language, any build system. Output variables can be consumed by any CI/CD pipeline.

---

## Feature Parity

These features are carried over from GitVersion with equivalent or improved behavior:

| Feature | Status |
|---------|--------|
| 6 version strategies | Same 6 strategies, improved implementations |
| 3 versioning modes | ContinuousDelivery, ContinuousDeployment, Mainline (improved) |
| 8 default branch configs | 8 configs (7 original + unknown catch-all) |
| YAML config file | `gitsemver.yml` (renamed from `.gitversion.yml`) |
| Tag prefix regex | Same behavior |
| Source branch resolution | Same hierarchy with Inherit walk-up |
| Ignore filters (SHA, date) | Same behavior |
| Custom merge message formats | Same + squash merge support |
| 25+ output variables | Same variable set including Assembly/NuGet variables (output-only, no file updates) |
| JSON / buildserver / file output | Same output formats |
| CLI override config | Same `--override-config` behavior |
| Version caching | Same `--no-cache` behavior |
| Pre-release tag number patterns | Same `tag-number-pattern` behavior |
| Branch name in pre-release | Same `{BranchName}` replacement |

## Intentionally Removed

| Feature | Reason |
|---------|--------|
| Assembly info file updates | Output variables (`AssemblySemVer`, etc.) are still generated — but gitsemver doesn't write to files. Use CI/CD scripts to consume them |
| .NET project file updates | Same — output-only, no file writes |
| `+semver:` directive syntax | Replaced with simpler `bump` directives |
| Interactive config wizard (`/init`) | Unnecessary — config is simple YAML |
| Per-commit mainline incrementing | Replaced with aggregate increment (DI-10) |
