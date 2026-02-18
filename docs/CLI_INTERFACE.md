# CLI Interface and Output Variables

This document covers the GitVersion 5.12.0 CLI interface, output formats, and the complete set of output variables.

Source: `GitVersion/src/GitVersion.App/`

---

## CLI Usage

```
gitversion [path] [options]
```

### Key Arguments

| Argument | Description |
|----------|-------------|
| `[path]` | Repository path (defaults to current directory) |
| `/targetpath` | Path to the working directory |
| `/url` | Remote repository URL (for dynamic repos) |
| `/b` | Target branch name |
| `/c` | Target commit SHA |
| `/output` | Output type: `json`, `buildserver`, `file` |
| `/outputfile` | File path to write version info to |
| `/showvariable` | Show only a specific variable |
| `/showconfig` | Show the effective configuration |
| `/overrideconfig` | Override config values (e.g., `tag-prefix=custom`) |
| `/nocache` | Disable version caching |
| `/diag` | Enable diagnostic logging |
| `/verbosity` | Log verbosity level |
| `/updateassemblyinfo` | Update AssemblyInfo files |
| `/updateassemblyinfofilename` | Specific AssemblyInfo file to update |
| `/updateprojectfiles` | Update .NET project files |
| `/init` | Run interactive config wizard |

---

## Output Variables

The `VariableProvider` converts the calculated `SemanticVersion` into these output variables:

### Version Components

| Variable | Example | Description |
|----------|---------|-------------|
| `Major` | `1` | Major version number |
| `Minor` | `2` | Minor version number |
| `Patch` | `3` | Patch version number |
| `MajorMinorPatch` | `1.2.3` | Combined `Major.Minor.Patch` |

### SemVer Formats

| Variable | Example | Description |
|----------|---------|-------------|
| `SemVer` | `1.2.3-beta.4` | Default SemVer 2.0 format |
| `FullSemVer` | `1.2.3-beta.4+5` | SemVer with build metadata |
| `LegacySemVer` | `1.2.3-beta4` | Legacy format (no dot in pre-release) |
| `LegacySemVerPadded` | `1.2.3-beta0004` | Padded legacy format |
| `InformationalVersion` | `1.2.3-beta.4+5.Branch.main.Sha.abc` | Full informational string |

### Pre-Release

| Variable | Example | Description |
|----------|---------|-------------|
| `PreReleaseTag` | `beta.4` | Full pre-release tag |
| `PreReleaseTagWithDash` | `-beta.4` | Pre-release with dash prefix |
| `PreReleaseLabel` | `beta` | Pre-release label (name only) |
| `PreReleaseLabelWithDash` | `-beta` | Label with dash prefix |
| `PreReleaseNumber` | `4` | Pre-release number |
| `WeightedPreReleaseNumber` | `30004` | Number + configured weight |

### Build Metadata

| Variable | Example | Description |
|----------|---------|-------------|
| `BuildMetaData` | `5` | Commits since tag |
| `BuildMetaDataPadded` | `0005` | Padded commits since tag |
| `FullBuildMetaData` | `5.Branch.main.Sha.abc1234` | Complete metadata string |

### Git Information

| Variable | Example | Description |
|----------|---------|-------------|
| `BranchName` | `main` | Current branch name |
| `EscapedBranchName` | `main` | Branch name with special chars replaced by `-` |
| `Sha` | `abc1234def567890...` | Full commit SHA |
| `ShortSha` | `abc1234` | First 7 characters of SHA |
| `CommitDate` | `2024-01-15` | Commit date (format configurable) |

### Commit Tracking

| Variable | Example | Description |
|----------|---------|-------------|
| `VersionSourceSha` | `def5678...` | SHA of the base version source commit |
| `CommitsSinceVersionSource` | `5` | Number of commits since version source |
| `CommitsSinceVersionSourcePadded` | `0005` | Padded commit count |
| `UncommittedChanges` | `3` | Number of uncommitted changes |

### .NET Assembly

| Variable | Example | Description |
|----------|---------|-------------|
| `AssemblySemVer` | `1.2.3.0` | Assembly version (4-part) |
| `AssemblyFileSemVer` | `1.2.3.0` | File version (4-part) |

### NuGet

| Variable | Example | Description |
|----------|---------|-------------|
| `NuGetVersion` | `1.2.3-beta0004` | NuGet v2 compatible version |
| `NuGetVersionV2` | `1.2.3-beta0004` | Same as NuGetVersion |
| `NuGetPreReleaseTag` | `beta0004` | NuGet pre-release suffix |
| `NuGetPreReleaseTagV2` | `beta0004` | Same as NuGetPreReleaseTag |

---

## Output Formats

### JSON (default)
```json
{
  "Major": 1,
  "Minor": 2,
  "Patch": 3,
  "PreReleaseTag": "beta.4",
  "PreReleaseTagWithDash": "-beta.4",
  "PreReleaseLabel": "beta",
  "PreReleaseLabelWithDash": "-beta",
  "PreReleaseNumber": 4,
  "WeightedPreReleaseNumber": 30004,
  "BuildMetaData": 5,
  "BuildMetaDataPadded": "0005",
  "FullBuildMetaData": "5.Branch.main.Sha.abc1234",
  "MajorMinorPatch": "1.2.3",
  "SemVer": "1.2.3-beta.4",
  "LegacySemVer": "1.2.3-beta4",
  "LegacySemVerPadded": "1.2.3-beta0004",
  "AssemblySemVer": "1.2.3.0",
  "AssemblyFileSemVer": "1.2.3.0",
  "FullSemVer": "1.2.3-beta.4+5",
  "InformationalVersion": "1.2.3-beta.4+5.Branch.main.Sha.abc1234",
  "BranchName": "main",
  "EscapedBranchName": "main",
  "Sha": "abc1234def567890...",
  "ShortSha": "abc1234",
  "NuGetVersionV2": "1.2.3-beta0004",
  "NuGetVersion": "1.2.3-beta0004",
  "NuGetPreReleaseTagV2": "beta0004",
  "NuGetPreReleaseTag": "beta0004",
  "VersionSourceSha": "def5678...",
  "CommitsSinceVersionSource": 5,
  "CommitsSinceVersionSourcePadded": "0005",
  "UncommittedChanges": 0,
  "CommitDate": "2024-01-15"
}
```

### Build Server Integration

GitVersion can set environment variables and build numbers for:
- Azure DevOps / TFS
- GitHub Actions
- Jenkins
- TeamCity
- AppVeyor
- GitLab CI
- Bitbucket Pipelines

### Single Variable

```bash
gitversion /showvariable SemVer
# Output: 1.2.3-beta.4
```

---

## Caching

GitVersion caches calculated versions to avoid recalculation:
- Cache key based on: current commit SHA, branch name, config hash, override config
- Stored in the `.git` directory
- Bypass with `/nocache` flag

Source: `VersionCalculation/Cache/GitVersionCache.cs`, `GitVersionCacheKeyFactory.cs`

---

## Version Formatting Detail

### PromoteNumberOfCommitsToTagNumber

In ContinuousDeployment and Mainline modes, the `VariableProvider` promotes `CommitsSinceTag` to the pre-release number:

```
Before: 1.2.3-beta.1+5  (PreReleaseNumber=1, CommitsSinceTag=5)
After:  1.2.3-beta.5+5   (PreReleaseNumber=1+5-1=5, CommitsSinceTag=null)

If no existing PreReleaseNumber:
Before: 1.2.3+5          (CommitsSinceTag=5)
After:  1.2.3-ci.5       (PreReleaseNumber=5, using fallback tag "ci")
```

This ensures every commit in ContinuousDeployment mode gets a unique, monotonically increasing version.
