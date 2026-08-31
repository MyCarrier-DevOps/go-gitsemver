package testutil

import (
	"os"
	"path/filepath"
	"testing"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/require"
)

// open reopens the repository from disk so assertions read committed state
// rather than the builder's in-memory handle.
func open(t *testing.T, r *TestRepo) *gogit.Repository {
	t.Helper()
	repo, err := gogit.PlainOpen(r.Path())
	require.NoError(t, err)
	return repo
}

func TestNewTestRepo_InitialisesGitRepository(t *testing.T) {
	r := NewTestRepo(t)

	require.NotEmpty(t, r.Path())
	require.DirExists(t, filepath.Join(r.Path(), ".git"))
	require.NotNil(t, open(t, r))
}

func TestAddCommit_ReturnsShaAndAdvancesHead(t *testing.T) {
	r := NewTestRepo(t)

	first := r.AddCommit("initial")
	require.Len(t, first, 40)
	require.Equal(t, first, r.HeadSha())

	second := r.AddCommit("feat: another")
	require.Len(t, second, 40)
	require.NotEqual(t, first, second)
	require.Equal(t, second, r.HeadSha())

	commit, err := open(t, r).CommitObject(plumbing.NewHash(second))
	require.NoError(t, err)
	require.Equal(t, "feat: another", commit.Message)
	require.Equal(t, 1, commit.NumParents())
}

func TestCreateTag_IsResolvable(t *testing.T) {
	r := NewTestRepo(t)
	sha := r.AddCommit("initial")

	r.CreateTag("v1.2.3", sha)

	ref, err := open(t, r).Reference(plumbing.ReferenceName("refs/tags/v1.2.3"), false)
	require.NoError(t, err)
	require.Equal(t, sha, ref.Hash().String())
}

func TestCreateAnnotatedTag_CreatesTagObject(t *testing.T) {
	r := NewTestRepo(t)
	sha := r.AddCommit("initial")

	r.CreateAnnotatedTag("v2.0.0", sha, "release two")

	tag, err := open(t, r).Tag("v2.0.0")
	require.NoError(t, err)

	obj, err := open(t, r).TagObject(tag.Hash())
	require.NoError(t, err)
	require.Equal(t, "release two\n", obj.Message)
	require.Equal(t, sha, obj.Target.String())
}

func TestCreateBranch_AndCheckout(t *testing.T) {
	r := NewTestRepo(t)
	base := r.AddCommit("initial")

	r.CreateBranch("feature/x", base)

	ref, err := open(t, r).Reference(plumbing.ReferenceName("refs/heads/feature/x"), false)
	require.NoError(t, err)
	require.Equal(t, base, ref.Hash().String())

	cfg, err := open(t, r).Config()
	require.NoError(t, err)
	require.Contains(t, cfg.Branches, "feature/x")

	r.Checkout("feature/x")

	head, err := open(t, r).Head()
	require.NoError(t, err)
	require.Equal(t, "refs/heads/feature/x", head.Name().String())
}

func TestMergeCommit_HasTwoParents(t *testing.T) {
	r := NewTestRepo(t)
	base := r.AddCommit("initial")

	r.CreateBranch("side", base)
	r.Checkout("side")
	side := r.AddCommit("feat: side work")

	r.CreateBranch("trunk", base)
	r.Checkout("trunk")

	merge := r.MergeCommit("Merge branch 'side'", side)
	require.Len(t, merge, 40)

	commit, err := open(t, r).CommitObject(plumbing.NewHash(merge))
	require.NoError(t, err)
	require.Equal(t, 2, commit.NumParents())
	require.Equal(t, "Merge branch 'side'", commit.Message)
	require.Equal(t, side, commit.ParentHashes[1].String())
}

func TestWriteConfig_WritesRepoRootFile(t *testing.T) {
	r := NewTestRepo(t)

	r.WriteConfig("tag-prefix: 'rel-'\n")

	data, err := os.ReadFile(filepath.Join(r.Path(), "go-gitsemver.yml"))
	require.NoError(t, err)
	require.Equal(t, "tag-prefix: 'rel-'\n", string(data))
}

func TestWriteConfigAt_CreatesNestedDirectories(t *testing.T) {
	r := NewTestRepo(t)

	r.WriteConfigAt(".github/GitVersion.yml", "next-version: 5.0.0\n")

	data, err := os.ReadFile(filepath.Join(r.Path(), ".github", "GitVersion.yml"))
	require.NoError(t, err)
	require.Equal(t, "next-version: 5.0.0\n", string(data))
}

func TestHeadSha_TracksLatestCommit(t *testing.T) {
	r := NewTestRepo(t)

	first := r.AddCommit("initial")
	require.Equal(t, first, r.HeadSha())

	second := r.AddCommit("second")
	require.Equal(t, second, r.HeadSha())
}
