# GitHub Action: go-gitsemver

A composite GitHub Action that installs **and runs** go-gitsemver, exporting all version variables as `GO_GITSEMVER_*` outputs.

**Location:** `.github/actions/setup-go-gitsemver/action.yml`

---

## Inputs

| Input | Default | Description |
|-------|---------|-------------|
| `version` | `latest` | Version of go-gitsemver to install (e.g., `v1.5.0`). |
| `token` | | GitHub token for authenticated binary downloads (avoids rate limits). |
| `verify-checksum` | `true` | Verify SHA-256 checksum of the downloaded binary. |
| `path` | `.` | Path to the local git repository. |

All versioning behavior (branch detection, config file, commit resolution) is auto-detected from the checked-out repo. No additional flags are needed.

---

## Outputs

All outputs are prefixed with `GO_GITSEMVER_` and match the variable names from `go-gitsemver -o json`.

### Primary

| Output | Example | Description |
|--------|---------|-------------|
| `GO_GITSEMVER_SemVer` | `1.2.3-beta.4` | Semantic version |
| `GO_GITSEMVER_FullSemVer` | `1.2.3-beta.4+5` | Full SemVer with build metadata |
| `GO_GITSEMVER_Major` | `1` | Major version number |
| `GO_GITSEMVER_Minor` | `2` | Minor version number |
| `GO_GITSEMVER_Patch` | `3` | Patch version number |
| `GO_GITSEMVER_MajorMinorPatch` | `1.2.3` | Major.Minor.Patch |
| `GO_GITSEMVER_InformationalVersion` | `1.2.3-beta.4+5.Branch...` | Full informational version |

### Pre-release

| Output | Example | Description |
|--------|---------|-------------|
| `GO_GITSEMVER_PreReleaseTag` | `beta.4` | Pre-release tag |
| `GO_GITSEMVER_PreReleaseTagWithDash` | `-beta.4` | Pre-release tag with leading dash |
| `GO_GITSEMVER_PreReleaseLabel` | `beta` | Pre-release label name |
| `GO_GITSEMVER_PreReleaseLabelWithDash` | `-beta` | Pre-release label with leading dash |
| `GO_GITSEMVER_PreReleaseNumber` | `4` | Pre-release counter |
| `GO_GITSEMVER_WeightedPreReleaseNumber` | `60004` | Pre-release number with tag weight |

### Build metadata

| Output | Example | Description |
|--------|---------|-------------|
| `GO_GITSEMVER_BuildMetaData` | `5` | Commits since version source |
| `GO_GITSEMVER_BuildMetaDataPadded` | `0005` | Commits since version source (zero-padded) |
| `GO_GITSEMVER_FullBuildMetaData` | `5.Branch.main...` | Full build metadata string |

### Git info

| Output | Example | Description |
|--------|---------|-------------|
| `GO_GITSEMVER_BranchName` | `main` | Branch used for calculation |
| `GO_GITSEMVER_EscapedBranchName` | `main` | Branch name safe for identifiers |
| `GO_GITSEMVER_Sha` | `abc123...` | Full commit SHA |
| `GO_GITSEMVER_ShortSha` | `abc123` | Short commit SHA |
| `GO_GITSEMVER_VersionSourceSha` | `def456...` | SHA of the version source |
| `GO_GITSEMVER_CommitsSinceVersionSource` | `5` | Commits since version source |
| `GO_GITSEMVER_CommitsSinceVersionSourcePadded` | `0005` | Commits since version source (zero-padded) |
| `GO_GITSEMVER_UncommittedChanges` | `0` | Number of uncommitted changes |
| `GO_GITSEMVER_CommitDate` | `2025-01-15` | Commit date |
| `GO_GITSEMVER_CommitTag` | `25.03.abc123` | Commit tag (YY.WW.ShortSha) |

### Legacy / Assembly / NuGet

| Output | Description |
|--------|-------------|
| `GO_GITSEMVER_LegacySemVer` | Legacy SemVer format |
| `GO_GITSEMVER_LegacySemVerPadded` | Legacy SemVer, zero-padded |
| `GO_GITSEMVER_AssemblySemVer` | Assembly version (Major.Minor.0.0) |
| `GO_GITSEMVER_AssemblySemFileVer` | Assembly file version (Major.Minor.Patch.0) |
| `GO_GITSEMVER_AssemblyInformationalVersion` | Assembly informational version |
| `GO_GITSEMVER_NuGetVersionV2` | NuGet v2 compatible version |
| `GO_GITSEMVER_NuGetVersion` | NuGet version |
| `GO_GITSEMVER_NuGetPreReleaseTagV2` | NuGet v2 pre-release tag |
| `GO_GITSEMVER_NuGetPreReleaseTag` | NuGet pre-release tag |

### JSON

| Output | Description |
|--------|-------------|
| `GO_GITSEMVER_JSON` | All version variables as a JSON object |

---

## Usage

### Basic (latest version)

```yaml
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Full history required

      - uses: MyCarrier-DevOps/go-gitsemver/.github/actions/setup-go-gitsemver@main
        id: version

      - name: Use version
        run: echo "Version is ${{ steps.version.outputs.GO_GITSEMVER_SemVer }}"
```

### Pinned version with authenticated download

```yaml
      - uses: MyCarrier-DevOps/go-gitsemver/.github/actions/setup-go-gitsemver@main
        id: version
        with:
          version: v1.4.0
          token: ${{ secrets.GITHUB_TOKEN }}
```

### Docker build with calculated version

```yaml
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: MyCarrier-DevOps/go-gitsemver/.github/actions/setup-go-gitsemver@main
        id: version
        with:
          token: ${{ secrets.GITHUB_TOKEN }}

      - name: Build
        run: docker build -t myapp:${{ steps.version.outputs.GO_GITSEMVER_MajorMinorPatch }} .
```

### Using the JSON output

```yaml
      - uses: MyCarrier-DevOps/go-gitsemver/.github/actions/setup-go-gitsemver@main
        id: version

      - name: Parse version JSON
        run: |
          echo '${{ steps.version.outputs.GO_GITSEMVER_JSON }}' | jq .
```

---

## Supported platforms

| OS | Architecture |
|----|-------------|
| Linux | amd64, arm64 |
| macOS | amd64, arm64 |
| Windows | amd64 |

---

## Checksum verification

By default, the action verifies the SHA-256 checksum of the downloaded binary against the `checksums.txt` file published alongside each release.

To disable checksum verification (not recommended):

```yaml
      - uses: MyCarrier-DevOps/go-gitsemver/.github/actions/setup-go-gitsemver@main
        with:
          verify-checksum: 'false'
```

### Building release artifacts

The makefile includes a `release-build` target that cross-compiles binaries for all supported platforms and generates the checksums file:

```bash
make release-build VERSION=v1.5.0
```

This produces:
```
bin/
  go-gitsemver-linux-amd64
  go-gitsemver-linux-arm64
  go-gitsemver-darwin-amd64
  go-gitsemver-darwin-arm64
  go-gitsemver-windows-amd64.exe
  checksums.txt
```

Upload all files in `bin/` as release assets when creating a GitHub release.
