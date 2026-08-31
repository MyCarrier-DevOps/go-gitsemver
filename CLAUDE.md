# go-gitsemver

A GitVersion-compatible semantic-versioning engine for Git repositories, shipped
three ways: a Cobra CLI (`go-gitsemver`), a Go library (`pkg/sdk`), and a
composite GitHub Action (`.github/actions/run-go-gitsemver`). It derives a
version from branch shape, tags, merge messages, and commit conventions —
either from a local clone (via `go-git`) or from the GitHub API with no clone
at all.

This repo's Go workflow — the idiomatic-Go conventions, the RED-test-first delivery loop, the security preflight, and the coverage gate — is owned by the **go-devkit** plugin (an apm dependency declared in [apm.yml](apm.yml); its machinery is installed outside the repo tree under `apm_modules/` via `apm install`). The go-devkit block below is the authoritative description of that workflow and is kept in sync by `/go-repo-init` — do not edit it by hand. The sections after it record the project-specific facts the plugin cannot know: this repo's module layout, coverage policy, and house rules.

<!-- BEGIN go-devkit -->
## Development workflow (go-devkit)

This repository uses the **go-devkit** Claude Code plugin. The idiomatic Go
conventions this project enforces (naming, error handling, package layout,
concurrency, HTTP clients, testing, security) live in the plugin and are loaded
by the `go-tdd` skill — read them before writing Go code.

For features and bugfixes, use the **`go-tdd`** skill — it drives the full loop:

1. **RED test first (code changes only).** Write a failing table-driven test
   before the implementation, then make it pass, then refactor — confirm it
   fails for the intended reason first. This does **not** apply to meta changes
   (renaming the app / `APPLICATION`, config, docs, dependency bumps); make those
   directly and verify with `/go-verify`.
2. **Preflight before coding.** Run `/go-preflight` (`make check-sec`) after
   planning and before implementing. If `govulncheck` flags a Go standard-library
   CVE, upgrade the toolchain (`brew upgrade go`, or `mise use -g go@latest`)
   before continuing; if it flags a dependency, `make bump`.
3. **Verify after.** Run `/go-verify` when the task is done — it runs `make fmt`,
   `make lint`, `make test`, and the plugin's coverage gate (which reads the CI
   `threshold-total` live so local and CI never drift). On non-main branches it
   finishes with mutation testing (`make mutation`, `mutest -diff origin/main`) —
   surviving mutants mean missing assertions; add tests rather than skip.
4. **The commit is gated.** In checkouts armed by `/go-repo-init`, a pre-commit
   hook re-runs fmt, lint, test, and (on non-main branches, when the run would
   judge exactly what the commit stages) mutation before any `git commit`, and
   blocks the commit until they pass. If the hook reports it skipped mutation,
   that is not a pass — deal with the reason it names. Do not try to bypass
   the gate — fix the failure it reports.

**Pin the Go version to a full patch release, and keep it in sync.** The `go`
directive in the module's `go.mod` (e.g. `go 1.26.5`, not `go 1.26`) and the
builder image in the Dockerfile (`golang:1.26.5`) must name the same patch. CI
intentionally floats on the patch level (`go-version: "1.26"`); bump it by hand
for a new minor or major.
<!-- END go-devkit -->

## Module layout

Single Go module at the **repository root** (`./go.mod`, module
`github.com/MyCarrier-DevOps/go-gitsemver`). There is no `app/` directory — the
Makefile's `APPLICATION` is therefore `.`.

| Path | Responsibility |
| --- | --- |
| `main.go` | Thin entrypoint; injects the `version` ldflag into `cmd`. |
| `cmd/` | Cobra commands: `root`, `calculate`, `remote`, `version`. |
| `internal/semver/` | Pure semantic-version types and format values. Zero external deps. |
| `internal/config/` | YAML config loading, defaults, builder, effective config. |
| `internal/git/` | The `Repository` interface, its `go-git` implementation, a mock, a repository store, and the merge-message parser. |
| `internal/github/` | GitHub API backend: client, repository, GraphQL batch queries, request-scoped cache. |
| `internal/context/` | `GitVersionContext` — the state threaded through calculation. |
| `internal/strategy/` | The six version-discovery strategies (see below). |
| `internal/calculator/` | Base-version, next-version, increment, and mainline calculators. |
| `internal/output/` | Variable provider, JSON output, `--explain` formatter. |
| `internal/testutil/` | Test-only repository and config helpers. |
| `pkg/sdk/` | The **public** API — `sdk.Calculate` (local) and `sdk.CalculateRemote`. The only package outside `internal/`; treat its signatures as a compatibility surface. |
| `e2e/` | End-to-end suites that build real Git repositories with branches, tags, and commits. |
| `example/` | A runnable SDK usage example. |

**The six strategies** live one-per-file in `internal/strategy/`:
`confignextversion`, `taggedcommit`, `mergemessage`, `trackrelease`,
`fallback`, `branchname` — coordinated by `strategies.go` over the shared
`base.go`. Three versioning modes (ContinuousDelivery, ContinuousDeployment,
Mainline) and five branch types (mainline, develop, release, feature, unknown)
drive which strategy wins.

## Commands

The Makefile is the interface — `make help` lists everything. The ones that
matter day to day:

```bash
make test            # unit tests + coverage (excludes e2e and testutil)
make e2e             # end-to-end tests
make test-all        # both
make coverage-check  # test, then fail under 85%
make lint            # golangci-lint, config at .github/.golangci.yml
make fmt             # gofmt + gofumpt via golangci-lint
make check-sec       # govulncheck
make mutation        # mutest against origin/main, 100% kill required
make ci              # fmt + lint + test-all + coverage-check + build
make run ARGS="calculate --explain"
```

## Coverage policy

**85% overall, enforced in two places that must stay in step:** `make
coverage-check` in the Makefile, and `threshold-total: 85` in the `test` job of
`.github/workflows/ci.yaml`. The go-devkit coverage gate (run by `/go-verify`)
reads the CI value live, so change the workflow and the Makefile together or
local and CI will disagree. `COVER_PKGS` deliberately excludes `e2e` and
`testutil`.

## Project house rules

- **`testify/require`, never `testify/assert`.** This is enforced by the
  `depguard` linter, not convention — `assert` continues past a failure and
  produces misleading cascades. `io/ioutil` and `go-kit/kit/log` are denied for
  the same reason (there are better replacements).
- **Table-driven tests** for anything with a clear input/output relation. The
  RED-first rule from the go-devkit block above still applies to each case.
- **Git access goes through the `Repository` interface**
  (`internal/git/interfaces.go`), never directly against `go-git`. That
  interface is the seam the mock and the GitHub backend both implement; bypassing
  it makes the code untestable and breaks remote mode. Every commit-walking and
  ref-listing method takes optional `PathFilter` arguments — keep monorepo
  support intact when adding to it.
- **Formatting is `gofmt` + `gofumpt` with `extra-rules: true`**, configured
  under `formatters:` in `.github/.golangci.yml`. Run `make fmt`; do not
  hand-format.
- **Lint skips test files** (`run.tests: false`). Do not take a clean lint run
  as evidence that test code is clean.
- **`pkg/sdk` is public API.** Anything else belongs in `internal/`. Changing an
  exported signature there is a breaking change for downstream importers and for
  the GitHub Action.
- **Keep the CLI, the SDK, and the Action in agreement.** A new calculation flag
  usually needs a `cmd/` flag, an `Options` field in `pkg/sdk`, and an input or
  output in `.github/actions/run-go-gitsemver/action.yml`.
- **Do not hand-edit the go-devkit block above.** Change it in the plugin and
  re-run `/go-repo-init`.

## Configuration files this tool reads

Auto-detection order, first match wins: `.github/GitVersion.yml`,
`.github/go-gitsemver.yml`, `GitVersion.yml`, `go-gitsemver.yml`. The JSON
Schema for a config file is `go-gitsemver-schema.json` at the repo root — update
it when you add a config option.

## Reference documentation

Everything lives in `docs/`:

| Doc | Contents |
| --- | --- |
| `ARCHITECTURE.md` | Package structure and design principles. |
| `CONFIGURATION.md` | Every config option with its default. |
| `STRATEGIES_AND_MODES.md` | Strategies, versioning modes, manual overrides. |
| `VERSION_STRATEGIES.md` | How each of the six discovery strategies works. |
| `BRANCH_WORKFLOWS.md` | Branch types, defaults, priority matching. |
| `FEATURES.md` / `HIGHLIGHTS.md` | Feature and design highlights. |
| `GITHUB_ACTION.md` | Action setup, usage, checksum verification. |
| `LOCAL_INSTALL.md` | Local install instructions. |
| `examples/` | Ten worked config files (gitflow, trunk-based, mainline, …). |
