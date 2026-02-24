# GitHub Action: Setup go-gitsemver

A composite GitHub Action that installs the go-gitsemver CLI from GitHub Releases.

**Location:** `.github/actions/setup-go-gitsemver/action.yml`

---

## Inputs

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `version` | No | `latest` | Version to install (e.g., `v1.5.0`). Use `latest` for the most recent release. |
| `token` | No | | GitHub token for authenticated downloads (avoids rate limits). |
| `verify-checksum` | No | `true` | Verify SHA-256 checksum of the downloaded binary. |

---

## Usage

### Basic (latest version)

```yaml
jobs:
  version:
    runs-on: ubuntu-latest
    steps:
      - uses: MyCarrier-DevOps/go-gitsemver/.github/actions/setup-go-gitsemver@main

      - name: Calculate version
        run: go-gitsemver --show-variable SemVer
```

### Pinned version with authenticated download

```yaml
      - uses: MyCarrier-DevOps/go-gitsemver/.github/actions/setup-go-gitsemver@main
        with:
          version: v1.4.0
          token: ${{ secrets.GITHUB_TOKEN }}
```

### Local mode (full clone required)

```yaml
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Full history required for local mode

      - uses: MyCarrier-DevOps/go-gitsemver/.github/actions/setup-go-gitsemver@main
        with:
          token: ${{ secrets.GITHUB_TOKEN }}

      - name: Calculate version
        id: version
        run: echo "semver=$(go-gitsemver --show-variable SemVer)" >> "$GITHUB_OUTPUT"

      - name: Build
        run: docker build -t myapp:${{ steps.version.outputs.semver }} .
```

### Remote mode (no clone needed)

```yaml
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: MyCarrier-DevOps/go-gitsemver/.github/actions/setup-go-gitsemver@main
        with:
          token: ${{ secrets.GITHUB_TOKEN }}

      - name: Calculate version
        id: version
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: echo "semver=$(go-gitsemver remote ${{ github.repository }} --ref ${{ github.ref_name }} --show-variable SemVer)" >> "$GITHUB_OUTPUT"

      - name: Build
        run: docker build -t myapp:${{ steps.version.outputs.semver }} .
```

### With explain mode

```yaml
      - name: Calculate version with explanation
        id: version
        run: |
          go-gitsemver --explain --show-variable SemVer 2> explain.txt
          echo "semver=$(go-gitsemver --show-variable SemVer)" >> "$GITHUB_OUTPUT"
          cat explain.txt
```

---

## Supported platforms

| OS | Architecture |
|----|-------------|
| Linux | amd64, arm64 |
| macOS | amd64, arm64 |
| Windows | amd64 |

---

## Token usage

The action accepts two separate tokens for different purposes:

| Token | Purpose |
|-------|---------|
| `token` input on the action | Authenticates the binary download from GitHub Releases (avoids API rate limits) |
| `GITHUB_TOKEN` env var in run steps | Authenticates go-gitsemver remote mode API calls to read repo data |

These are independent — you can use the action without a token for downloading (public repo), but remote mode always needs `GITHUB_TOKEN` or App credentials.

---

## Checksum verification

By default, the action verifies the SHA-256 checksum of the downloaded binary against the `checksums.txt` file published alongside each release. This ensures the binary has not been tampered with or corrupted during download.

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
