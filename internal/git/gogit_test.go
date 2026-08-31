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

	// After the failed open with repair disabled, confirm the config key is still present.
	getBeforeRepairCmd := exec.Command("git", "-C", repo.Path(), "config", "--local", "--get-all", "extensions.worktreeConfig")
	beforeOutput, beforeErr := getBeforeRepairCmd.CombinedOutput()
	require.NoError(t, beforeErr, "expected extensions.worktreeConfig to still be present when repair is disabled")
	require.Equal(t, "true", strings.TrimSpace(string(beforeOutput)), "expected extensions.worktreeConfig to remain unchanged before repair")

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

// gitRun runs a git command in the repo, failing the test on error.
func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %s: %s", strings.Join(args, " "), out)
}

// TestCommitLog_StopsAtFromCommit pins that a non-empty "from" bounds the walk.
// Treating it as the zero hash would return the whole history instead.
func TestCommitLog_StopsAtFromCommit(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	first := repo.AddCommit("one")
	second := repo.AddCommit("two")
	third := repo.AddCommit("three")

	r, err := Open(repo.Path())
	require.NoError(t, err)

	bounded, err := r.CommitLog(second, third)
	require.NoError(t, err)
	require.Len(t, bounded, 1, "only commits after the from-commit should be returned")
	require.Equal(t, third, bounded[0].Sha)

	unbounded, err := r.CommitLog("", third)
	require.NoError(t, err)
	require.Len(t, unbounded, 3)
	require.Equal(t, first, unbounded[2].Sha)
}

// TestMainlineCommitLog_WalksAllParentsAndStopsAtFrom pins the first-parent walk:
// it must visit every commit down to "from", not stop at the first one that has
// a parent, and must not run past a non-empty "from".
func TestMainlineCommitLog_WalksAllParentsAndStopsAtFrom(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	first := repo.AddCommit("one")
	second := repo.AddCommit("two")
	third := repo.AddCommit("three")

	r, err := Open(repo.Path())
	require.NoError(t, err)

	all, err := r.MainlineCommitLog("", third)
	require.NoError(t, err)
	require.Len(t, all, 3, "the walk must follow first parents all the way to the root")
	require.Equal(t, third, all[0].Sha)
	require.Equal(t, first, all[2].Sha)

	bounded, err := r.MainlineCommitLog(second, third)
	require.NoError(t, err)
	require.Len(t, bounded, 1, "a non-empty from must bound the walk")
	require.Equal(t, third, bounded[0].Sha)
}

// TestBranchesContainingCommit_TipMatchAndAncestor pins both inclusion paths:
// a branch whose tip is exactly the commit, and one whose tip descends from it.
func TestBranchesContainingCommit_TipMatchAndAncestor(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	base := repo.AddCommit("base")

	repo.CreateBranch("at-base", base)
	repo.CreateBranch("ahead", base)
	repo.Checkout("ahead")
	aheadTip := repo.AddCommit("further work")

	r, err := Open(repo.Path())
	require.NoError(t, err)

	namesFor := func(sha string) []string {
		branches, err := r.BranchesContainingCommit(sha)
		require.NoError(t, err)
		names := make([]string, 0, len(branches))
		for _, b := range branches {
			names = append(names, b.Name.Friendly)
		}
		return names
	}

	atBase := namesFor(base)
	require.Contains(t, atBase, "at-base", "a branch whose tip is the commit must be included")
	require.Contains(t, atBase, "ahead", "a branch whose tip descends from the commit must be included")

	// The commit on "ahead" is not reachable from the "at-base" tip, so that
	// branch must be excluded. Treating a tip mismatch as a match would wrongly
	// include every branch here.
	atAheadTip := namesFor(aheadTip)
	require.Contains(t, atAheadTip, "ahead")
	require.NotContains(t, atAheadTip, "at-base",
		"a branch that does not contain the commit must be excluded")
}

// TestNumberOfUncommittedChanges_StagedAndUnstaged pins both halves of the
// dirty-file test: a staged change with a clean worktree, and a worktree change
// with a clean index. Each half is missed if the other operand is inverted.
func TestNumberOfUncommittedChanges_StagedAndUnstaged(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable not found")
	}

	t.Run("clean", func(t *testing.T) {
		repo := testutil.NewTestRepo(t)
		repo.AddCommit("initial")

		r, err := Open(repo.Path())
		require.NoError(t, err)

		n, err := r.NumberOfUncommittedChanges()
		require.NoError(t, err)
		require.Equal(t, 0, n)
	})

	t.Run("staged only", func(t *testing.T) {
		repo := testutil.NewTestRepo(t)
		repo.AddCommit("initial")
		require.NoError(t, os.WriteFile(repo.Path()+"/staged.txt", []byte("new"), 0o644))
		gitRun(t, repo.Path(), "add", "staged.txt")

		r, err := Open(repo.Path())
		require.NoError(t, err)

		n, err := r.NumberOfUncommittedChanges()
		require.NoError(t, err)
		require.Equal(t, 1, n, "a staged file with a clean worktree still counts as uncommitted")
	})

	t.Run("worktree only", func(t *testing.T) {
		repo := testutil.NewTestRepo(t)
		repo.AddCommit("initial")
		gitRun(t, repo.Path(), "config", "user.email", "test@example.com")
		gitRun(t, repo.Path(), "config", "user.name", "Test")
		require.NoError(t, os.WriteFile(repo.Path()+"/tracked.txt", []byte("v1"), 0o644))
		gitRun(t, repo.Path(), "add", "tracked.txt")
		gitRun(t, repo.Path(), "commit", "-m", "add tracked")

		// Modify without staging: index clean, worktree dirty.
		require.NoError(t, os.WriteFile(repo.Path()+"/tracked.txt", []byte("v2"), 0o644))

		r, err := Open(repo.Path())
		require.NoError(t, err)

		n, err := r.NumberOfUncommittedChanges()
		require.NoError(t, err)
		require.Equal(t, 1, n, "an unstaged worktree change still counts as uncommitted")
	})
}

// TestUnsetLocalWorktreeConfig_MissingKeyIsSuccess pins that git's exit code 5
// (key not found) is treated as success rather than as a failure.
func TestUnsetLocalWorktreeConfig_MissingKeyIsSuccess(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable not found")
	}

	repo := testutil.NewTestRepo(t)
	repo.AddCommit("initial")

	require.NoError(t, unsetLocalWorktreeConfig(repo.Path()),
		"unsetting a key that is not set must succeed")
}

// TestUnsetLocalWorktreeConfig_ReportsGitOutput pins that git's own message is
// carried into the returned error when the command fails for another reason.
func TestUnsetLocalWorktreeConfig_ReportsGitOutput(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git executable not found")
	}

	err := unsetLocalWorktreeConfig(t.TempDir())
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsetting local extensions.worktreeConfig")
	require.Contains(t, err.Error(), "fatal:",
		"git's own output should be included in the error")
}
