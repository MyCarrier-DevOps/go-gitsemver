package main

import (
	"io"
	"os"
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
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)

	old := os.Stdout
	os.Stdout = w

	done := make(chan string, 1)
	go func() { b, _ := io.ReadAll(r); done <- string(b) }()

	fn()

	w.Close()
	os.Stdout = old
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

	out := captureStdout(t, func() {
		result, err := sdkCalculateForTest()
		require.NoError(t, err)
		printVersion("Custom", result)
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
