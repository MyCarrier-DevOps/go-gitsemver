package git

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/MyCarrier-DevOps/go-gitsemver/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestOpen_InvalidPath(t *testing.T) {
	_, err := Open("/nonexistent/path")
	require.Error(t, err)
	require.Contains(t, err.Error(), "opening git repository")
}

func TestOpen_WorktreeConfigExtensionEnabled(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable not found")
	}

	repo := testutil.NewTestRepo(t)
	repo.AddCommit("initial commit")

	setCmd := exec.Command("git", "-C", repo.Path(), "config", "--local", "extensions.worktreeConfig", "true")
	require.NoError(t, setCmd.Run())

	// First, confirm that go-git itself rejects the repo when repair is disabled.
	// This ensures the recovery path is actually needed and exercised below.
	_, err := OpenWithOptions(repo.Path(), OpenOptions{RepairWorktreeConfig: false})
	require.Error(t, err, "expected open to fail when RepairWorktreeConfig is false")
	require.Contains(t, err.Error(), "worktreeconfig", "expected the unsupported-extension error")

	// Now open with repair enabled (the default) and assert it succeeds.
	opened, err := Open(repo.Path())
	require.NoError(t, err)
	require.NotNil(t, opened)

	// Assert the config key was actually removed from disk.
	getCmd := exec.Command("git", "-C", repo.Path(), "config", "--local", "--get-all", "extensions.worktreeConfig")
	output, getErr := getCmd.CombinedOutput()
	if getErr != nil {
		// Exit code 1 means the key does not exist — expected after recovery.
		var exitErr *exec.ExitError
		require.ErrorAs(t, getErr, &exitErr, "expected an exit error from git config --get-all")
		require.Equal(t, 1, exitErr.ExitCode(), "expected exit code 1 (key not found) from git config --get-all")
		return
	}

	require.Empty(t, strings.TrimSpace(string(output)))
}

func TestOpen_ValidRepository(t *testing.T) {
	// Use the go-gitsemver repository itself for testing.
	// Find the repo root by walking up from the test file location.
	dir, err := os.Getwd()
	require.NoError(t, err)

	// We're in internal/git/, walk up to the repo root.
	repo, err := Open(dir)
	require.NoError(t, err)
	require.NotEmpty(t, repo.Path())
	require.NotEmpty(t, repo.WorkingDirectory())
}

func TestOpen_IsHeadDetached(t *testing.T) {
	dir, err := os.Getwd()
	require.NoError(t, err)

	repo, err := Open(dir)
	require.NoError(t, err)

	// CI environments (GitHub Actions) check out in detached HEAD state.
	if os.Getenv("CI") != "" {
		t.Skip("skipping in CI: HEAD is expected to be detached")
	}

	// In a normal checkout, HEAD is not detached.
	require.False(t, repo.IsHeadDetached())
}

func TestOpen_Head(t *testing.T) {
	dir, err := os.Getwd()
	require.NoError(t, err)

	repo, err := Open(dir)
	require.NoError(t, err)

	head, err := repo.Head()
	require.NoError(t, err)
	require.NotEmpty(t, head.Name.Friendly)
	require.NotNil(t, head.Tip)
	require.NotEmpty(t, head.Tip.Sha)
}

func TestOpen_Tags(t *testing.T) {
	dir, err := os.Getwd()
	require.NoError(t, err)

	repo, err := Open(dir)
	require.NoError(t, err)

	tags, err := repo.Tags()
	require.NoError(t, err)
	// Tags may or may not exist; just verify no error.
	_ = tags
}

func TestOpen_Branches(t *testing.T) {
	dir, err := os.Getwd()
	require.NoError(t, err)

	repo, err := Open(dir)
	require.NoError(t, err)

	branches, err := repo.Branches()
	require.NoError(t, err)
	require.NotEmpty(t, branches, "expected at least one branch")
}

func TestOpen_NumberOfUncommittedChanges(t *testing.T) {
	dir, err := os.Getwd()
	require.NoError(t, err)

	repo, err := Open(dir)
	require.NoError(t, err)

	_, err = repo.NumberOfUncommittedChanges()
	require.NoError(t, err)
}

func TestOpen_CommitFromSha(t *testing.T) {
	dir, err := os.Getwd()
	require.NoError(t, err)

	repo, err := Open(dir)
	require.NoError(t, err)

	head, err := repo.Head()
	require.NoError(t, err)

	commit, err := repo.CommitFromSha(head.Tip.Sha)
	require.NoError(t, err)
	require.Equal(t, head.Tip.Sha, commit.Sha)
	require.NotEmpty(t, commit.Message)
	require.False(t, commit.When.IsZero())
}

func TestOpen_CommitFromSha_Invalid(t *testing.T) {
	dir, err := os.Getwd()
	require.NoError(t, err)

	repo, err := Open(dir)
	require.NoError(t, err)

	_, err = repo.CommitFromSha("0000000000000000000000000000000000000000")
	require.Error(t, err)
}

func TestOpen_CommitLog(t *testing.T) {
	dir, err := os.Getwd()
	require.NoError(t, err)

	repo, err := Open(dir)
	require.NoError(t, err)

	head, err := repo.Head()
	require.NoError(t, err)

	// Get all commits from HEAD (no "from" bound).
	commits, err := repo.CommitLog("", head.Tip.Sha)
	require.NoError(t, err)
	require.NotEmpty(t, commits)
	require.Equal(t, head.Tip.Sha, commits[0].Sha)
}

func TestOpen_MainlineCommitLog(t *testing.T) {
	dir, err := os.Getwd()
	require.NoError(t, err)

	repo, err := Open(dir)
	require.NoError(t, err)

	head, err := repo.Head()
	require.NoError(t, err)

	commits, err := repo.MainlineCommitLog("", head.Tip.Sha)
	require.NoError(t, err)
	require.NotEmpty(t, commits)
}

func TestOpen_BranchCommits(t *testing.T) {
	dir, err := os.Getwd()
	require.NoError(t, err)

	repo, err := Open(dir)
	require.NoError(t, err)

	head, err := repo.Head()
	require.NoError(t, err)

	branch := Branch{Name: head.Name, Tip: head.Tip}
	commits, err := repo.BranchCommits(branch)
	require.NoError(t, err)
	require.NotEmpty(t, commits)
}

func TestOpen_BranchCommits_NilTip(t *testing.T) {
	dir, err := os.Getwd()
	require.NoError(t, err)

	repo, err := Open(dir)
	require.NoError(t, err)

	branch := Branch{Name: NewReferenceName("refs/heads/test")}
	commits, err := repo.BranchCommits(branch)
	require.NoError(t, err)
	require.Empty(t, commits)
}

func TestOpen_CommitsPriorTo(t *testing.T) {
	dir, err := os.Getwd()
	require.NoError(t, err)

	repo, err := Open(dir)
	require.NoError(t, err)

	head, err := repo.Head()
	require.NoError(t, err)

	branch := Branch{Name: head.Name, Tip: head.Tip}

	// Use a time far in the past — should return no commits.
	ancient := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	commits, err := repo.CommitsPriorTo(ancient, branch)
	require.NoError(t, err)
	require.Empty(t, commits)

	// Use a time in the future — should return all commits.
	future := time.Now().Add(24 * time.Hour)
	commits, err = repo.CommitsPriorTo(future, branch)
	require.NoError(t, err)
	require.NotEmpty(t, commits)
}

func TestOpen_FindMergeBase(t *testing.T) {
	dir, err := os.Getwd()
	require.NoError(t, err)

	repo, err := Open(dir)
	require.NoError(t, err)

	head, err := repo.Head()
	require.NoError(t, err)

	// Merge base of a commit with itself is itself.
	base, err := repo.FindMergeBase(head.Tip.Sha, head.Tip.Sha)
	require.NoError(t, err)
	require.Equal(t, head.Tip.Sha, base)
}

func TestOpen_BranchesContainingCommit(t *testing.T) {
	dir, err := os.Getwd()
	require.NoError(t, err)

	repo, err := Open(dir)
	require.NoError(t, err)

	head, err := repo.Head()
	require.NoError(t, err)

	branches, err := repo.BranchesContainingCommit(head.Tip.Sha)
	require.NoError(t, err)
	require.NotEmpty(t, branches, "HEAD commit should be on at least one branch")
}

func TestOpen_PeelTagToCommit(t *testing.T) {
	dir, err := os.Getwd()
	require.NoError(t, err)

	repo, err := Open(dir)
	require.NoError(t, err)

	tags, err := repo.Tags()
	require.NoError(t, err)

	if len(tags) == 0 {
		t.Skip("no tags in repository")
	}

	// Peel the first tag to a commit SHA.
	sha, err := repo.PeelTagToCommit(tags[0])
	require.NoError(t, err)
	require.NotEmpty(t, sha)
	require.Len(t, sha, 40, "expected full SHA")
}
