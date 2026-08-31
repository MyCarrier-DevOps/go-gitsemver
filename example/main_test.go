package main

import (
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/MyCarrier-DevOps/go-gitsemver/internal/testutil"
	"github.com/MyCarrier-DevOps/go-gitsemver/pkg/sdk"

	"github.com/stretchr/testify/require"
)

// runInRepo points the process at a throwaway git repository so the example's
// sdk.Calculate(Path: ".") calls resolve against known history.
func runInRepo(t *testing.T) {
	t.Helper()
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial")
	repo.CreateTag("v1.4.0", sha)
	repo.AddCommit("feat: add search")
	t.Chdir(repo.Path())
}

// captureStdout collects everything the example prints while fn runs.
func captureStdout(t *testing.T, fn func() error) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)

	old := os.Stdout
	os.Stdout = w

	done := make(chan string, 1)
	go func() { b, _ := io.ReadAll(r); done <- string(b) }()

	// Deferred: a require failure inside fn calls t.FailNow, which unwinds with
	// runtime.Goexit and runs only deferred calls. Restoring inline would leave
	// os.Stdout pointing at this pipe for every later test in the package, so
	// their output - including failure diagnostics - would be swallowed.
	restored := false
	restore := func() {
		if restored {
			return
		}
		restored = true
		os.Stdout = old
		w.Close()
	}
	defer restore()

	require.NoError(t, fn())

	restore()
	return <-done
}

func TestLocalVersion_PrintsVariables(t *testing.T) {
	runInRepo(t)

	out := captureStdout(t, localVersion)

	require.Contains(t, out, "=== Local Version ===")
	require.Contains(t, out, "SemVer")
	require.Contains(t, out, "1.5.0")
}

func TestLocalVersionExplain_PrintsExplanation(t *testing.T) {
	runInRepo(t)

	out := captureStdout(t, localVersionExplain)

	require.Contains(t, out, "=== Explain Output ===")
	require.Contains(t, out, "Final version:")
	require.Contains(t, out, "Selected source:")
	require.Contains(t, out, "Candidates:")
	require.NotContains(t, out, "Candidates: 0")
}

func TestPrintVersion_SortsVariablesAndLabels(t *testing.T) {
	runInRepo(t)

	out := captureStdout(t, func() error {
		result, err := sdkCalculateForTest()
		if err != nil {
			return err
		}
		printVersion("Custom", result)
		return nil
	})

	require.Contains(t, out, "=== Custom Version ===")

	// Variable names must be printed in sorted order.
	var names []string
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && !strings.HasPrefix(line, "===") {
			names = append(names, fields[0])
		}
	}
	require.NotEmpty(t, names)
	require.IsIncreasingf(t, names, "printVersion should sort variable names, got %v", names)
}

// sdkCalculateForTest mirrors the SDK call the example makes, so printVersion
// can be exercised against a real Result.
func sdkCalculateForTest() (*sdk.Result, error) {
	return sdk.Calculate(sdk.LocalOptions{Path: "."})
}

// TestRun_SkipsRemoteWithoutToken pins the token guard in run(). With no token
// the remote example must be skipped entirely - attempting it would fail
// authentication, so an empty token reaching remoteVersion surfaces as an error.
func TestRun_SkipsRemoteWithoutToken(t *testing.T) {
	runInRepo(t)

	out := captureStdout(t, func() error { return run("") })

	require.Contains(t, out, "=== Local Version ===")
	require.Contains(t, out, "=== Explain Output ===")
	require.NotContains(t, out, "=== Remote Version ===",
		"the remote example must be skipped when no token is set")
}

// TestMain_SucceedsInARepository runs main() in a subprocess, which is the only
// way to exercise a function that calls log.Fatal. It pins that a successful run
// exits zero: were the error check inverted, main would call log.Fatal on a nil
// error and exit non-zero instead.
func TestMain_SucceedsInARepository(t *testing.T) {
	if os.Getenv("GO_GITSEMVER_EXAMPLE_SUBPROCESS") == "1" {
		main()
		return
	}

	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial")
	repo.CreateTag("v1.4.0", sha)
	repo.AddCommit("feat: add search")

	cmd := exec.Command(os.Args[0], "-test.run=TestMain_SucceedsInARepository") //nolint:gosec // re-executes this test binary
	cmd.Dir = repo.Path()
	cmd.Env = append(os.Environ(), "GO_GITSEMVER_EXAMPLE_SUBPROCESS=1", "GITHUB_TOKEN=")

	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "main() should exit zero on success; output:\n%s", out)
	require.Contains(t, string(out), "=== Local Version ===")
	require.NotContains(t, string(out), "=== Remote Version ===")
}
