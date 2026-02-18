# Configuration Reference

GitVersion 5.12.0 is configured via a `.gitversion.yml` file in the repository root. This document covers all configuration options, their defaults, and effects.

Source: `GitVersion/src/GitVersion.Core/Model/Configuration/Config.cs`, `BranchConfig.cs`, `ConfigurationBuilder.cs`

---

## Configuration File Format

```yaml
# .gitversion.yml
mode: ContinuousDelivery
tag-prefix: '[vV]'
next-version: 1.0.0
increment: Inherit
branches:
  main:
    regex: ^master$|^main$
    increment: Patch
    tag: ''
  develop:
    regex: ^dev(elop)?(ment)?$
    increment: Minor
    tag: alpha
  # ... more branches
ignore:
  sha: []
  commits-before: 2020-01-01
merge-message-formats:
  custom: '^Merged PR (?<PullRequestNumber>\d+): .*$'
```

---

## Global Configuration Options

### mode
**Type:** Enum
**Default:** `ContinuousDelivery`
**Values:** `ContinuousDelivery`, `ContinuousDeployment`, `Mainline`

Sets the default versioning mode for all branches. Individual branches can override this.

- **ContinuousDelivery** - Pre-release number from tag scanning; manual release trigger
- **ContinuousDeployment** - Commits-since-tag promoted to pre-release number; every commit unique
- **Mainline** - Each commit increments; no pre-release on mainline branches

### tag-prefix
**Type:** Regex string
**Default:** `[vV]`

Regex pattern to identify version tags. Stripped from tag names before parsing.
- `[vV]` matches tags like `v1.0.0` or `V1.0.0`
- Use `""` for tags without prefix like `1.0.0`

### next-version
**Type:** String (semver)
**Default:** (none)

Explicitly sets the next version. Overrides tag-based calculation.
- Accepts `"2"` (auto-converted to `"2.0"`) or full `"2.1.0"`
- Ignored if current commit is tagged
- Does NOT increment (used as-is)

### increment
**Type:** Enum
**Default:** `Inherit`
**Values:** `None`, `Major`, `Minor`, `Patch`, `Inherit`

Default increment strategy for branches that don't specify their own.
- `Inherit` means the branch inherits from its source/parent branch

### continuous-delivery-fallback-tag
**Type:** String
**Default:** `ci`

Pre-release label used in ContinuousDeployment mode when no branch-specific tag is configured.

### commit-message-incrementing
**Type:** Enum
**Default:** `Enabled`
**Values:** `Enabled`, `Disabled`, `MergeMessageOnly`

Controls whether commit messages are scanned for `+semver:` directives.
- **Enabled** - All commits scanned
- **Disabled** - Commit messages ignored, only branch config increment used
- **MergeMessageOnly** - Only merge commits (>1 parent) scanned

### major-version-bump-message
**Type:** Regex
**Default:** `\+semver:\s?(breaking|major)`

Commit message pattern that triggers a major version bump.

### minor-version-bump-message
**Type:** Regex
**Default:** `\+semver:\s?(feature|minor)`

Commit message pattern that triggers a minor version bump.

### patch-version-bump-message
**Type:** Regex
**Default:** `\+semver:\s?(fix|patch)`

Commit message pattern that triggers a patch version bump.

### no-bump-message
**Type:** Regex
**Default:** `\+semver:\s?(none|skip)`

Commit message pattern that explicitly prevents a version bump.

### merge-message-formats
**Type:** Dictionary<string, string>
**Default:** (empty)

Custom merge message regex patterns. Added on top of the 6 built-in formats.
```yaml
merge-message-formats:
  custom-azure: '^Merged PR (?<PullRequestNumber>\d+): .*$'
```

### commit-date-format
**Type:** String
**Default:** `yyyy-MM-dd`

Format string for the `CommitDate` output variable.

### update-build-number
**Type:** Boolean
**Default:** `true`

Whether to update the CI build number with the calculated version.

### tag-pre-release-weight
**Type:** Integer
**Default:** `60000`

Weight used for tagged pre-release versions in `WeightedPreReleaseNumber` output variable calculation.

### legacy-semver-padding
**Type:** Integer
**Default:** `4`

Padding for legacy semver pre-release number (e.g., `beta0004`).

### build-metadata-padding
**Type:** Integer
**Default:** `4`

Padding for build metadata number (e.g., `+0004`).

### commits-since-version-source-padding
**Type:** Integer
**Default:** `4`

Padding for `CommitsSinceVersionSourcePadded` output variable.

### assembly-versioning-scheme
**Type:** Enum
**Default:** `MajorMinorPatch`

.NET assembly versioning scheme. Controls which parts of the version are included.

### assembly-file-versioning-scheme
**Type:** Enum
**Default:** `MajorMinorPatch`

.NET assembly file versioning scheme.

### assembly-informational-format
**Type:** Format string
**Default:** (none, uses InformationalVersion)

Custom format for AssemblyInformationalVersion.

### assembly-versioning-format / assembly-file-versioning-format
**Type:** Format string
**Default:** (none)

Custom format strings for assembly versioning.

---

## Branch Configuration Options

Each branch type is configured under the `branches:` key.

### regex
**Type:** Regex string
**Required:** Yes

Pattern to match branch names against. First matching branch config wins.

**Default patterns:**
| Branch Key | Default Regex |
|------------|--------------|
| main | `^master$\|^main$` |
| develop | `^dev(elop)?(ment)?$` |
| release | `^releases?[/-]` |
| feature | `^features?[/-]` |
| hotfix | `^hotfix(es)?[/-]` |
| pull-request | `^(pull\|pull\-requests\|pr)[/-]` |
| support | `^support[/-]` |

### increment
**Type:** Enum
**Values:** `None`, `Major`, `Minor`, `Patch`, `Inherit`

Increment strategy for this branch. `Inherit` resolves from the source branch.

### mode
**Type:** Enum (same as global `mode`)

Versioning mode override for this branch.

### tag
**Type:** String
**Special values:**
- `""` (empty) → stable version, no pre-release label
- `"{BranchName}"` → replaced with the branch name (sans prefix)
- Any string → literal pre-release label (e.g., `"alpha"`, `"beta"`, `"rc"`)

### source-branches
**Type:** List of branch config keys
**Required:** Yes

Which branch types this branch can be created from. Used for:
1. Resolving `Inherit` increment strategy
2. Finding the branch creation point
3. Validation

### is-source-branch-for
**Type:** List of branch config keys

Inverse of `source-branches`. Declaring `is-source-branch-for: [feature]` on `develop` adds `develop` to feature's `source-branches`.

### is-mainline
**Type:** Boolean
**Default:** `false`

Marks as a mainline branch. Mainline branches produce stable versions and are used as the trunk reference in Mainline versioning mode.

### is-release-branch
**Type:** Boolean
**Default:** `false`

Enables version extraction from the branch name (`VersionInBranchNameVersionStrategy`).

### tracks-release-branches
**Type:** Boolean
**Default:** `false`

Enables `TrackReleaseBranchesVersionStrategy`. Makes the branch aware of active release branches and main branch tags.

### prevent-increment-of-merged-branch-version
**Type:** Boolean
**Default:** varies

When `true`, merging from this branch type won't cause an increment from merge message version extraction.

### track-merge-target
**Type:** Boolean
**Default:** `false`

Track the merge target branch's version.

### tag-number-pattern
**Type:** Regex with `(?<number>\d+)` capture group

Extracts a number from the branch name to append to the pre-release tag. Used for pull-request branches:
```yaml
tag-number-pattern: '[/-](?<number>\d+)'
```

### pre-release-weight
**Type:** Integer

Weight for sorting pre-release versions. Higher = more recent in sorting. Used in `WeightedPreReleaseNumber` calculation.

### commit-message-incrementing
**Type:** Enum (same as global)

Override commit message incrementing mode for this branch.

---

## Ignore Configuration

```yaml
ignore:
  sha:
    - abc1234def567890   # ignore specific commit SHAs
  commits-before: 2020-01-01T00:00:00   # ignore commits before this date
```

Commits matching ignore rules are excluded from version calculation. Their tags are still visible but filtered out during base version selection.

---

## Configuration Resolution Order

1. **CreateDefaultConfiguration()** - Hard-coded defaults in `ConfigurationBuilder`
2. **File override** - Values from `.gitversion.yml` merged on top
3. **CLI overrides** - Command-line argument overrides (highest priority)
4. **Branch finalization** - Branch configs inherit from global config where unset
5. **EffectiveConfiguration** - Final merged config for the specific branch context

### Branch Config Inheritance

When a branch config property is `null`, it inherits from global config:
- `Increment`: if null → global `Increment` → `Inherit`
- `VersioningMode`: if null → global `VersioningMode` → `ContinuousDelivery`
  - Exception: `develop` gets `ContinuousDeployment` unless global is `Mainline`

### Increment "Inherit" Resolution

When `Increment = Inherit`, GitVersion walks up the branch hierarchy:
1. Find the source branch for the current branch
2. Check its `Increment` setting
3. If also `Inherit`, continue walking up
4. Final fallback: `Patch`

---

## Example Configurations

### Minimal (all defaults)
```yaml
# .gitversion.yml
# Empty file - all defaults apply
```

### GitFlow with custom labels
```yaml
mode: ContinuousDelivery
branches:
  main:
    tag: ''
  develop:
    tag: dev
  release:
    tag: rc
  feature:
    tag: '{BranchName}'
```

### Trunk-Based Development
```yaml
mode: Mainline
branches:
  main:
    increment: Patch
    is-mainline: true
  feature:
    increment: Minor
```

### Force next major version
```yaml
next-version: 2.0.0
```
