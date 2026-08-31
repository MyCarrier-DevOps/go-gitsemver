package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MyCarrier-DevOps/go-gitsemver/internal/testutil"

	"github.com/stretchr/testify/require"
)

// resetFlags restores every global command flag to its zero/default value once
// the test finishes, so tests that drive calculateRunE cannot leak into others.
func resetFlags(t *testing.T) {
	t.Helper()
	saved := struct {
		path, branch, commit, config, output, showVariable, verbosity string
		showConfig, explain, noRepair                                 bool
	}{
		flagPath, flagBranch, flagCommit, flagConfig, flagOutput, flagShowVariable, flagVerbosity,
		flagShowConfig, flagExplain, flagNoRepairWorktreeConfig,
	}
	t.Cleanup(func() {
		flagPath, flagBranch, flagCommit = saved.path, saved.branch, saved.commit
		flagConfig, flagOutput, flagShowVariable = saved.config, saved.output, saved.showVariable
		flagVerbosity = saved.verbosity
		flagShowConfig, flagExplain = saved.showConfig, saved.explain
		flagNoRepairWorktreeConfig = saved.noRepair
	})
}

// captureOutput redirects stdout and stderr for the duration of fn.
func captureOutput(t *testing.T, fn func() error) (stdout, stderr string, err error) {
	t.Helper()

	outR, outW, pipeErr := os.Pipe()
	require.NoError(t, pipeErr)
	errR, errW, pipeErr := os.Pipe()
	require.NoError(t, pipeErr)

	oldOut, oldErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = outW, errW

	outCh := make(chan string, 1)
	errCh := make(chan string, 1)
	go func() { b, _ := io.ReadAll(outR); outCh <- string(b) }()
	go func() { b, _ := io.ReadAll(errR); errCh <- string(b) }()

	err = fn()

	outW.Close()
	errW.Close()
	os.Stdout, os.Stderr = oldOut, oldErr

	return <-outCh, <-errCh, err
}

func TestCalculateRunE_CalculatesVersion(t *testing.T) {
	resetFlags(t)
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial")
	repo.CreateTag("v1.2.0", sha)
	repo.AddCommit("feat: add search")

	flagPath = repo.Path()
	flagShowVariable = "SemVer"

	stdout, _, err := captureOutput(t, func() error { return calculateRunE(nil, nil) })
	require.NoError(t, err)
	require.Equal(t, "1.3.0", strings.TrimSpace(stdout))
}

func TestCalculateRunE_ChoreBumpsPatch(t *testing.T) {
	resetFlags(t)
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial")
	repo.CreateTag("v1.2.0", sha)
	repo.AddCommit("chore: bump dependencies")

	flagPath = repo.Path()
	flagShowVariable = "SemVer"

	stdout, _, err := captureOutput(t, func() error { return calculateRunE(nil, nil) })
	require.NoError(t, err)
	require.Equal(t, "1.2.1", strings.TrimSpace(stdout))
}

func TestCalculateRunE_NoBumpDirectiveSuppresses(t *testing.T) {
	resetFlags(t)
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial")
	repo.CreateTag("v1.2.0", sha)
	repo.AddCommit("chore: bump dependencies +semver: none")

	flagPath = repo.Path()
	flagShowVariable = "SemVer"

	stdout, _, err := captureOutput(t, func() error { return calculateRunE(nil, nil) })
	require.NoError(t, err)
	require.Equal(t, "1.2.0", strings.TrimSpace(stdout))
}

func TestCalculateRunE_JSONOutput(t *testing.T) {
	resetFlags(t)
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial")
	repo.CreateTag("v2.0.0", sha)
	repo.AddCommit("fix: a bug")

	flagPath = repo.Path()
	flagOutput = "json"

	stdout, _, err := captureOutput(t, func() error { return calculateRunE(nil, nil) })
	require.NoError(t, err)
	require.Contains(t, stdout, `"SemVer"`)
	require.Contains(t, stdout, "2.0.1")
}

func TestCalculateRunE_ExplainWritesToStderr(t *testing.T) {
	resetFlags(t)
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial")
	repo.CreateTag("v1.0.0", sha)
	repo.AddCommit("feat: something")

	flagPath = repo.Path()
	flagExplain = true
	flagShowVariable = "SemVer"

	stdout, stderr, err := captureOutput(t, func() error { return calculateRunE(nil, nil) })
	require.NoError(t, err)
	require.Equal(t, "1.1.0", strings.TrimSpace(stdout))
	require.NotEmpty(t, stderr, "explain output should go to stderr")
}

func TestCalculateRunE_ShowConfig(t *testing.T) {
	resetFlags(t)
	repo := testutil.NewTestRepo(t)
	repo.AddCommit("initial")

	flagPath = repo.Path()
	flagShowConfig = true

	stdout, _, err := captureOutput(t, func() error { return calculateRunE(nil, nil) })
	require.NoError(t, err)
	require.Contains(t, stdout, `"TagPrefix"`)
}

func TestCalculateRunE_UsesRepoConfigFile(t *testing.T) {
	resetFlags(t)
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial")
	repo.CreateTag("rel-1.0.0", sha)
	repo.WriteConfig("tag-prefix: 'rel-'\n")
	repo.AddCommit("fix: a bug")

	flagPath = repo.Path()
	flagShowVariable = "SemVer"

	stdout, _, err := captureOutput(t, func() error { return calculateRunE(nil, nil) })
	require.NoError(t, err)
	require.Equal(t, "1.0.1", strings.TrimSpace(stdout))
}

func TestCalculateRunE_OpenRepositoryError(t *testing.T) {
	resetFlags(t)
	flagPath = filepath.Join(t.TempDir(), "definitely-not-a-repo")

	_, _, err := captureOutput(t, func() error { return calculateRunE(nil, nil) })
	require.Error(t, err)
	require.Contains(t, err.Error(), "opening repository")
}

func TestCalculateRunE_InvalidConfigError(t *testing.T) {
	resetFlags(t)
	repo := testutil.NewTestRepo(t)
	repo.AddCommit("initial")

	bad := filepath.Join(t.TempDir(), "bad.yml")
	require.NoError(t, os.WriteFile(bad, []byte("mode: [not, a, string\n"), 0o644))

	flagPath = repo.Path()
	flagConfig = bad

	_, _, err := captureOutput(t, func() error { return calculateRunE(nil, nil) })
	require.Error(t, err)
	require.Contains(t, err.Error(), "loading configuration")
}

func TestCalculateRunE_UnknownOutputFormat(t *testing.T) {
	resetFlags(t)
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial")
	repo.CreateTag("v1.0.0", sha)
	repo.AddCommit("fix: a bug")

	flagPath = repo.Path()
	flagOutput = "yaml"

	_, _, err := captureOutput(t, func() error { return calculateRunE(nil, nil) })
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown output format")
}

func TestCalculateRunE_UnknownCommitError(t *testing.T) {
	resetFlags(t)
	repo := testutil.NewTestRepo(t)
	repo.AddCommit("initial")

	flagPath = repo.Path()
	flagCommit = "0000000000000000000000000000000000000000"

	_, _, err := captureOutput(t, func() error { return calculateRunE(nil, nil) })
	require.Error(t, err)
}

func TestExecute_RunsRootCommand(t *testing.T) {
	resetFlags(t)
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial")
	repo.CreateTag("v3.1.0", sha)
	repo.AddCommit("feat: add thing")

	oldArgs := os.Args
	t.Cleanup(func() {
		os.Args = oldArgs
		rootCmd.SetArgs(nil)
	})
	rootCmd.SetArgs([]string{"--path", repo.Path(), "--show-variable", "SemVer"})

	stdout, _, err := captureOutput(t, func() error { Execute(); return nil })
	require.NoError(t, err)
	require.Equal(t, "3.2.0", strings.TrimSpace(stdout))
}
