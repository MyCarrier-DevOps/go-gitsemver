---
name: go-verify
description: Use when a task in a Go service repo is complete and needs the pre-handoff verification gate — before claiming success, committing, or opening a PR — and after meta changes (config, docs, dependency bumps) that are exempt from the TDD loop. Trigger on "verify", "run the checks", "is this ready", gofmt/lint failures, coverage dropping below the CI threshold, or surviving mutants from mutation testing.
---

# /go-verify

Run this **after** completing a task, before handing it off. Run it from the
repository root. Stop at the first failure and surface the real output — never
report success without the passing evidence.

## Steps

1. Format (auto-fixes gofmt / goimports / golines):

   ```bash
   make fmt
   ```

2. Lint — must be zero errors:

   ```bash
   make lint
   ```

3. Test with coverage (writes a `coverage.out` in each module directory):

   ```bash
   make test
   ```

4. Coverage gate — fail if total coverage is below the threshold declared in CI.
   The threshold is read live from `.github/workflows/ci.yml` and the module
   dir is auto-detected from the Makefile's `APPLICATION`, so this gate and CI
   can never drift (and it survives renaming the module directory). The gate
   ships with the plugin; run it from the repo root so it resolves the project's
   `Makefile` and `ci.yml`:

   ```bash
   "${CLAUDE_PLUGIN_ROOT}"/scripts/coverage-gate.sh
   ```

If the coverage gate fails, add table-driven tests for the uncovered code. Keep
`main()` a one-liner and test the `run()` helper instead. Do **not** lower the
threshold to pass.

5. Mutation testing — on any branch other than the base branch
   (feature/chore/fix/…), run it with the final coverage in place. Skip only
   when on the base branch itself (there is no diff to mutate). Fetch the base
   ref first — it is whatever `MUTATION_BASE` in the Makefile points at
   (`origin/main` by default):

   ```bash
   git fetch origin main    # or the repo's default branch
   make mutation
   ```

   This runs `mutest -diff <base> -threshold 100 ./...` in each module — only
   the lines this branch changed are mutated, and every mutant must be killed
   for the gate to pass.

   **mutest judges the working tree, not a commit**: staged and unstaged edits
   are included, and untracked `.go` files are mutated whole-file. Remove or
   gitignore stray `.go` scratch files before running, or the survivors won't
   be about your change.

   A non-zero exit usually means **surviving mutants**: the changed code has
   tests that execute it without actually asserting on its behavior. Add or
   strengthen table-driven tests (assert on boundary operators: `>` vs `>=`,
   `==` vs `!=`) until the survivors are killed — but read the output first;
   exit 1 also covers a failing test baseline or a mutest/tooling error. Do
   **not** skip the step or change `MUTATION_BASE` to dodge it.

Note: in checkouts armed by `/go-repo-init`, the plugin's pre-commit hook
re-runs fmt, lint, test, and (on non-base branches, when the run would judge
exactly what the commit stages) mutation before any `git commit`, blocking the
commit on failure — running `/go-verify` first means the commit gate is a
formality, not a surprise. If the hook says it skipped mutation, treat that as
work outstanding, not as a pass.

## Optional

Re-run the security scan if dependencies or the toolchain changed since preflight:

```bash
make check-sec
```
