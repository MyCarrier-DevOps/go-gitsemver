# Plan: Go Library API for gitsemver

## Problem

gitsemver is CLI-only. Teams that want to embed version calculation into Go applications, custom CI tooling, or GitHub Actions written in Go must shell out to the binary and parse stdout. This is fragile, slow, and loses type safety.

## Goal

Expose a public Go package (`pkg/gitsemver`) that lets consumers calculate versions programmatically — both local and remote — without importing `internal/` packages directly.

## Proposed API

```go
import "go-gitsemver/pkg/gitsemver"

// Local mode — calculate from a local git repo
result, err := gitsemver.Calculate(gitsemver.LocalOptions{
    Path:   "/path/to/repo",
    Branch: "main",
})
fmt.Println(result.Variables["SemVer"]) // "1.2.3"

// Remote mode — calculate via GitHub API
result, err := gitsemver.CalculateRemote(gitsemver.RemoteOptions{
    Owner: "myorg",
    Repo:  "myrepo",
    Token: os.Getenv("GITHUB_TOKEN"),
    Ref:   "main",
})
fmt.Println(result.Variables["FullSemVer"]) // "1.2.3+5"
```

### Types

```go
package gitsemver

type LocalOptions struct {
    Path       string // repo path (default ".")
    Branch     string // target branch (default: HEAD)
    Commit     string // target commit SHA (default: branch tip)
    ConfigPath string // config file path (default: auto-detect)
    Explain    bool   // include explanation in result
}

type RemoteOptions struct {
    Owner      string // GitHub owner (required)
    Repo       string // GitHub repo (required)
    Token      string // GitHub token (or use AppID + AppKeyPath)
    AppID      int64  // GitHub App ID
    AppKeyPath string // path to GitHub App private key PEM
    BaseURL    string // GitHub Enterprise API base URL
    Ref        string // branch, tag, or SHA (default: repo default branch)
    MaxCommits int    // max commit walk depth (default: 1000)
    ConfigPath string // local config file override
    Explain    bool
}

type Result struct {
    Variables map[string]string // all 30+ output variables
    Explain   string            // human-readable explanation (if requested)
}

func Calculate(opts LocalOptions) (*Result, error)
func CalculateRemote(opts RemoteOptions) (*Result, error)
```

## Architecture

```
External Go application
    imports pkg/gitsemver (public)
        uses internal/git       (allowed — same module)
        uses internal/github    (allowed — same module)
        uses internal/config    (allowed — same module)
        uses internal/context   (allowed — same module)
        uses internal/strategy  (allowed — same module)
        uses internal/calculator(allowed — same module)
        uses internal/output    (allowed — same module)
```

Go's `internal/` restriction only blocks imports from *outside* the module. `pkg/gitsemver/` lives inside the same module, so it can freely import all `internal/` packages. External consumers only see `pkg/gitsemver/` — a clean, stable API surface.

### What stays internal

Everything. No packages move. The `internal/` boundary protects consumers from breaking changes in implementation details (commit types, strategy interfaces, cache internals, etc.).

### What's new

One package: `pkg/gitsemver/gitsemver.go` (~100 lines). It wires together the same components that `cmd/calculate.go` and `cmd/remote.go` already use.

## Implementation

### File: `pkg/gitsemver/gitsemver.go`

`Calculate()` mirrors `calculateRunE` from `cmd/calculate.go`:
1. Open repo via `git.NewGoGitRepository(opts.Path)`
2. Load config via `config.LoadFromFile()` or auto-detect
3. Build context via `context.NewContext(store, repo, cfg, contextOpts)`
4. Resolve effective config for branch
5. Run calculator: `calculator.NewNextVersionCalculator(store, strategies).Calculate(ctx, ec, opts.Explain)`
6. Compute variables: `output.GetVariables(result.Version, ec)`
7. Return `&Result{Variables: vars}`

`CalculateRemote()` mirrors `remoteRunE` from `cmd/remote.go`:
1. Create GitHub client via `github.NewClient(clientConfig)`
2. Create `github.NewGitHubRepository(client, owner, repo, ghOpts...)`
3. Fetch remote config or use local override
4. Same steps 3-7 as local

### File: `pkg/gitsemver/gitsemver_test.go`

- Test `Calculate()` with a temp git repo (reuse `testutil` helpers)
- Test `CalculateRemote()` with httptest mock server
- Test default option handling
- Test error cases (bad path, no auth, etc.)

## CLI refactor (optional, not required)

After the library exists, `cmd/calculate.go` and `cmd/remote.go` can be simplified to thin wrappers:

```go
func calculateRunE(_ *cobra.Command, args []string) error {
    result, err := gitsemver.Calculate(gitsemver.LocalOptions{
        Path:   flagPath,
        Branch: flagBranch,
        Commit: flagCommit,
        // ...
    })
    if err != nil {
        return err
    }
    return writeOutput(result.Variables)
}
```

This is optional — the current CLI code works fine and the refactor can happen later.

## Scope

| Item | Effort | Priority |
|------|--------|----------|
| `pkg/gitsemver/gitsemver.go` | Small (~100 lines) | P0 |
| `pkg/gitsemver/gitsemver_test.go` | Medium (~200 lines) | P0 |
| Update `go.mod` module path if needed | None (same module) | — |
| CLI refactor to use library | Small | P1 (optional) |
| godoc comments on public types | Small | P0 |
| README section on library usage | Small | P1 |

## Considerations

- **Versioning:** The library API becomes part of the module's public contract. Breaking changes require a major version bump. Keep the API surface minimal — `LocalOptions`, `RemoteOptions`, `Result`, and two functions.
- **go.mod module path:** Currently `go-gitsemver` (not a full URL like `github.com/org/go-gitsemver`). For external imports to work, the module path needs to be a valid import path. This would be a separate migration.
- **Dependency weight:** Consumers who only need local mode still pull in `go-github`, `oauth2`, and `ghinstallation` via transitive deps. If this matters, the remote functionality could be in a separate sub-package (`pkg/gitsemver/remote`), but this adds complexity for marginal benefit.
