package git

import (
	"errors"
	"go-gitsemver/internal/config"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newTestCommit(sha string, when time.Time, msg string, parents ...string) Commit {
	return Commit{Sha: sha, When: when, Message: msg, Parents: parents}
}

func tagWithVersion(name, sha string) Tag {
	return Tag{Name: NewReferenceName("refs/tags/" + name), TargetSha: sha}
}

func branchWithTip(name string, tip *Commit) Branch {
	return Branch{Name: NewBranchReferenceName(name), Tip: tip}
}

func stringPtr(s string) *string { return &s }

// --- GetValidVersionTags ---

func TestGetValidVersionTags(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-time.Hour)
	later := now.Add(time.Hour)

	c1 := newTestCommit("sha1", earlier, "commit 1")
	c2 := newTestCommit("sha2", later, "commit 2")

	mock := &MockRepository{
		TagsFunc: func(filters ...PathFilter) ([]Tag, error) {
			return []Tag{
				tagWithVersion("v1.0.0", "sha1"),
				tagWithVersion("v2.0.0", "sha2"),
				tagWithVersion("not-a-version", "sha3"),
			}, nil
		},
		PeelTagToCommitFunc: func(tag Tag) (string, error) {
			return tag.TargetSha, nil
		},
		CommitFromShaFunc: func(sha string) (Commit, error) {
			switch sha {
			case "sha1":
				return c1, nil
			case "sha2":
				return c2, nil
			}
			return Commit{}, errors.New("not found")
		},
	}

	store := NewRepositoryStore(mock)

	t.Run("all tags", func(t *testing.T) {
		tags, err := store.GetValidVersionTags("[vV]", nil)
		require.NoError(t, err)
		require.Len(t, tags, 2)
	})

	t.Run("filtered by time", func(t *testing.T) {
		tags, err := store.GetValidVersionTags("[vV]", &now)
		require.NoError(t, err)
		require.Len(t, tags, 1)
		require.Equal(t, "sha1", tags[0].Commit.Sha)
	})
}

func TestGetValidVersionTags_Error(t *testing.T) {
	mock := &MockRepository{
		TagsFunc: func(filters ...PathFilter) ([]Tag, error) {
			return nil, errors.New("git error")
		},
	}
	store := NewRepositoryStore(mock)
	_, err := store.GetValidVersionTags("[vV]", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "listing tags")
}

// --- GetVersionTagsOnBranch ---

func TestGetVersionTagsOnBranch(t *testing.T) {
	c1 := newTestCommit("sha1", time.Now().Add(-2*time.Hour), "c1")
	c2 := newTestCommit("sha2", time.Now().Add(-time.Hour), "c2")
	c3 := newTestCommit("sha3", time.Now(), "c3")

	mock := &MockRepository{
		TagsFunc: func(filters ...PathFilter) ([]Tag, error) {
			return []Tag{
				tagWithVersion("v1.0.0", "sha1"),
				tagWithVersion("v2.0.0", "sha2"),
				tagWithVersion("v3.0.0", "sha3"),
			}, nil
		},
		PeelTagToCommitFunc: func(tag Tag) (string, error) {
			return tag.TargetSha, nil
		},
		CommitFromShaFunc: func(sha string) (Commit, error) {
			switch sha {
			case "sha1":
				return c1, nil
			case "sha2":
				return c2, nil
			case "sha3":
				return c3, nil
			}
			return Commit{}, errors.New("not found")
		},
		BranchCommitsFunc: func(branch Branch, filters ...PathFilter) ([]Commit, error) {
			// Only sha1 and sha2 are on this branch.
			return []Commit{c2, c1}, nil
		},
	}

	store := NewRepositoryStore(mock)
	tip := c2
	branch := branchWithTip("main", &tip)

	versions, err := store.GetVersionTagsOnBranch(branch, "[vV]")
	require.NoError(t, err)
	require.Len(t, versions, 2)
	// Sorted descending.
	require.Equal(t, int64(2), versions[0].Major)
	require.Equal(t, int64(1), versions[1].Major)
}

// --- GetCurrentCommitTaggedVersion ---

func TestGetCurrentCommitTaggedVersion(t *testing.T) {
	c1 := newTestCommit("sha1", time.Now(), "c1")

	mock := &MockRepository{
		TagsFunc: func(filters ...PathFilter) ([]Tag, error) {
			return []Tag{
				tagWithVersion("v1.0.0", "sha1"),
				tagWithVersion("v2.0.0", "sha1"), // two tags on same commit
				tagWithVersion("v0.5.0", "sha2"),
			}, nil
		},
		PeelTagToCommitFunc: func(tag Tag) (string, error) {
			return tag.TargetSha, nil
		},
		CommitFromShaFunc: func(sha string) (Commit, error) {
			switch sha {
			case "sha1":
				return c1, nil
			case "sha2":
				return newTestCommit("sha2", time.Now(), "c2"), nil
			}
			return Commit{}, errors.New("not found")
		},
	}

	store := NewRepositoryStore(mock)

	t.Run("highest tag wins", func(t *testing.T) {
		ver, ok, err := store.GetCurrentCommitTaggedVersion(c1, "[vV]")
		require.NoError(t, err)
		require.True(t, ok)
		require.Equal(t, int64(2), ver.Major)
	})

	t.Run("no matching tag", func(t *testing.T) {
		c := newTestCommit("sha-none", time.Now(), "no tag")
		_, ok, err := store.GetCurrentCommitTaggedVersion(c, "[vV]")
		require.NoError(t, err)
		require.False(t, ok)
	})
}

// --- FindMainBranch ---

func TestFindMainBranch(t *testing.T) {
	mainTip := newTestCommit("main-tip", time.Now(), "tip")
	mock := &MockRepository{
		BranchesFunc: func(filters ...PathFilter) ([]Branch, error) {
			return []Branch{
				branchWithTip("main", &mainTip),
				branchWithTip("develop", nil),
			}, nil
		},
	}

	store := NewRepositoryStore(mock)
	cfg := config.CreateDefaultConfiguration()

	branch, found, err := store.FindMainBranch(cfg)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "main", branch.FriendlyName())
}

func TestFindMainBranch_NotFound(t *testing.T) {
	mock := &MockRepository{
		BranchesFunc: func(filters ...PathFilter) ([]Branch, error) {
			return []Branch{branchWithTip("develop", nil)}, nil
		},
	}

	store := NewRepositoryStore(mock)
	cfg := config.CreateDefaultConfiguration()

	_, found, err := store.FindMainBranch(cfg)
	require.NoError(t, err)
	require.False(t, found)
}

func TestFindMainBranch_NoConfig(t *testing.T) {
	store := NewRepositoryStore(&MockRepository{})
	cfg := &config.Config{Branches: map[string]*config.BranchConfig{}}

	_, found, err := store.FindMainBranch(cfg)
	require.NoError(t, err)
	require.False(t, found)
}

// --- GetReleaseBranches ---

func TestGetReleaseBranches(t *testing.T) {
	mock := &MockRepository{
		BranchesFunc: func(filters ...PathFilter) ([]Branch, error) {
			return []Branch{
				branchWithTip("main", nil),
				branchWithTip("release/1.0", nil),
				branchWithTip("release/2.0", nil),
				branchWithTip("feature/auth", nil),
			}, nil
		},
	}

	store := NewRepositoryStore(mock)
	releaseCfg := map[string]*config.BranchConfig{
		"release": {Regex: stringPtr(`^releases?[/-]`)},
	}

	branches, err := store.GetReleaseBranches(releaseCfg)
	require.NoError(t, err)
	require.Len(t, branches, 2)
}

// --- GetBranchesContainingCommit ---

func TestGetBranchesContainingCommit(t *testing.T) {
	mock := &MockRepository{
		BranchesContainingCommitFunc: func(sha string) ([]Branch, error) {
			return []Branch{branchWithTip("main", nil)}, nil
		},
	}

	store := NewRepositoryStore(mock)
	branches, err := store.GetBranchesContainingCommit(Commit{Sha: "abc"})
	require.NoError(t, err)
	require.Len(t, branches, 1)
}

func TestGetBranchesContainingCommit_EmptyCommit(t *testing.T) {
	store := NewRepositoryStore(&MockRepository{})
	branches, err := store.GetBranchesContainingCommit(Commit{})
	require.NoError(t, err)
	require.Nil(t, branches)
}

// --- GetBranchesForCommit ---

func TestGetBranchesForCommit(t *testing.T) {
	tip := newTestCommit("sha1", time.Now(), "tip")
	otherTip := newTestCommit("sha2", time.Now(), "other")

	mock := &MockRepository{
		BranchesFunc: func(filters ...PathFilter) ([]Branch, error) {
			return []Branch{
				branchWithTip("main", &tip),
				branchWithTip("develop", &otherTip),
				{Name: NewReferenceName("refs/remotes/origin/main"), Tip: &tip, IsRemote: true},
			}, nil
		},
	}

	store := NewRepositoryStore(mock)
	branches, err := store.GetBranchesForCommit(tip)
	require.NoError(t, err)
	require.Len(t, branches, 1)
	require.Equal(t, "main", branches[0].FriendlyName())
}

// --- GetTargetBranch ---

func TestGetTargetBranch_ByName(t *testing.T) {
	mock := &MockRepository{
		BranchesFunc: func(filters ...PathFilter) ([]Branch, error) {
			return []Branch{
				branchWithTip("main", nil),
				branchWithTip("develop", nil),
			}, nil
		},
	}

	store := NewRepositoryStore(mock)
	branch, err := store.GetTargetBranch("develop")
	require.NoError(t, err)
	require.Equal(t, "develop", branch.FriendlyName())
}

func TestGetTargetBranch_Empty_UsesHead(t *testing.T) {
	mock := &MockRepository{
		HeadFunc: func() (Branch, error) {
			return branchWithTip("main", nil), nil
		},
	}

	store := NewRepositoryStore(mock)
	branch, err := store.GetTargetBranch("")
	require.NoError(t, err)
	require.Equal(t, "main", branch.FriendlyName())
}

func TestGetTargetBranch_NotFound(t *testing.T) {
	mock := &MockRepository{
		BranchesFunc: func(filters ...PathFilter) ([]Branch, error) {
			return nil, nil
		},
	}

	store := NewRepositoryStore(mock)
	_, err := store.GetTargetBranch("nonexistent")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

// --- GetCurrentCommit ---

func TestGetCurrentCommit_FromSha(t *testing.T) {
	expected := newTestCommit("sha1", time.Now(), "test")
	mock := &MockRepository{
		CommitFromShaFunc: func(sha string) (Commit, error) {
			return expected, nil
		},
	}

	store := NewRepositoryStore(mock)
	commit, err := store.GetCurrentCommit(Branch{}, "sha1")
	require.NoError(t, err)
	require.Equal(t, expected, commit)
}

func TestGetCurrentCommit_FromBranchTip(t *testing.T) {
	tip := newTestCommit("sha1", time.Now(), "tip")
	store := NewRepositoryStore(&MockRepository{})

	commit, err := store.GetCurrentCommit(branchWithTip("main", &tip), "")
	require.NoError(t, err)
	require.Equal(t, tip, commit)
}

func TestGetCurrentCommit_NilTip(t *testing.T) {
	store := NewRepositoryStore(&MockRepository{})
	_, err := store.GetCurrentCommit(branchWithTip("main", nil), "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no tip commit")
}

// --- GetBaseVersionSource ---

func TestGetBaseVersionSource(t *testing.T) {
	root := newTestCommit("root", time.Now().Add(-time.Hour), "initial")
	tip := newTestCommit("tip", time.Now(), "latest")

	mock := &MockRepository{
		CommitLogFunc: func(from, to string, filters ...PathFilter) ([]Commit, error) {
			return []Commit{tip, root}, nil
		},
	}

	store := NewRepositoryStore(mock)
	commit, err := store.GetBaseVersionSource(tip)
	require.NoError(t, err)
	require.Equal(t, "root", commit.Sha)
}

func TestGetBaseVersionSource_EmptyLog(t *testing.T) {
	tip := newTestCommit("tip", time.Now(), "only")
	mock := &MockRepository{
		CommitLogFunc: func(from, to string, filters ...PathFilter) ([]Commit, error) {
			return nil, nil
		},
	}

	store := NewRepositoryStore(mock)
	commit, err := store.GetBaseVersionSource(tip)
	require.NoError(t, err)
	require.Equal(t, "tip", commit.Sha)
}

// --- GetCommitLog / GetMainlineCommitLog / GetMergeBaseCommits ---

func TestGetCommitLog(t *testing.T) {
	expected := []Commit{{Sha: "c1"}, {Sha: "c2"}}
	mock := &MockRepository{
		CommitLogFunc: func(from, to string, filters ...PathFilter) ([]Commit, error) {
			require.Equal(t, "from-sha", from)
			require.Equal(t, "to-sha", to)
			return expected, nil
		},
	}

	store := NewRepositoryStore(mock)
	commits, err := store.GetCommitLog(Commit{Sha: "from-sha"}, Commit{Sha: "to-sha"})
	require.NoError(t, err)
	require.Equal(t, expected, commits)
}

func TestGetMainlineCommitLog(t *testing.T) {
	expected := []Commit{{Sha: "c1"}}
	mock := &MockRepository{
		MainlineCommitLogFunc: func(from, to string, filters ...PathFilter) ([]Commit, error) {
			return expected, nil
		},
	}

	store := NewRepositoryStore(mock)
	commits, err := store.GetMainlineCommitLog(Commit{Sha: "a"}, Commit{Sha: "b"})
	require.NoError(t, err)
	require.Equal(t, expected, commits)
}

func TestGetMergeBaseCommits(t *testing.T) {
	expected := []Commit{{Sha: "c1"}}
	mock := &MockRepository{
		CommitLogFunc: func(from, to string, filters ...PathFilter) ([]Commit, error) {
			require.Equal(t, "base-sha", from)
			require.Equal(t, "head-sha", to)
			return expected, nil
		},
	}

	store := NewRepositoryStore(mock)
	commits, err := store.GetMergeBaseCommits(Commit{Sha: "head-sha"}, Commit{Sha: "base-sha"})
	require.NoError(t, err)
	require.Equal(t, expected, commits)
}

// --- FindMergeBase ---

func TestFindMergeBase_Branches(t *testing.T) {
	tip1 := newTestCommit("sha1", time.Now(), "tip1")
	tip2 := newTestCommit("sha2", time.Now(), "tip2")
	base := newTestCommit("base", time.Now().Add(-time.Hour), "base")

	mock := &MockRepository{
		FindMergeBaseFunc: func(sha1, sha2 string) (string, error) {
			return "base", nil
		},
		CommitFromShaFunc: func(sha string) (Commit, error) {
			if sha == "base" {
				return base, nil
			}
			return Commit{}, errors.New("not found")
		},
	}

	store := NewRepositoryStore(mock)
	commit, found, err := store.FindMergeBase(
		branchWithTip("main", &tip1),
		branchWithTip("develop", &tip2),
	)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "base", commit.Sha)
}

func TestFindMergeBase_NilTip(t *testing.T) {
	store := NewRepositoryStore(&MockRepository{})
	_, found, err := store.FindMergeBase(
		branchWithTip("main", nil),
		branchWithTip("develop", nil),
	)
	require.NoError(t, err)
	require.False(t, found)
}

func TestFindMergeBase_NoCommonAncestor(t *testing.T) {
	tip1 := newTestCommit("sha1", time.Now(), "tip1")
	tip2 := newTestCommit("sha2", time.Now(), "tip2")

	mock := &MockRepository{
		FindMergeBaseFunc: func(sha1, sha2 string) (string, error) {
			return "", nil
		},
	}

	store := NewRepositoryStore(mock)
	_, found, err := store.FindMergeBase(
		branchWithTip("main", &tip1),
		branchWithTip("orphan", &tip2),
	)
	require.NoError(t, err)
	require.False(t, found)
}

func TestFindMergeBaseFromCommits(t *testing.T) {
	base := newTestCommit("base", time.Now(), "base")
	mock := &MockRepository{
		FindMergeBaseFunc: func(sha1, sha2 string) (string, error) {
			return "base", nil
		},
		CommitFromShaFunc: func(sha string) (Commit, error) {
			return base, nil
		},
	}

	store := NewRepositoryStore(mock)
	commit, found, err := store.FindMergeBaseFromCommits(
		Commit{Sha: "a"}, Commit{Sha: "b"},
	)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "base", commit.Sha)
}

// --- IsCommitOnBranch ---

func TestIsCommitOnBranch(t *testing.T) {
	tip := newTestCommit("tip", time.Now(), "tip")
	mock := &MockRepository{
		CommitLogFunc: func(from, to string, filters ...PathFilter) ([]Commit, error) {
			return []Commit{{Sha: "tip"}, {Sha: "target"}, {Sha: "root"}}, nil
		},
	}

	store := NewRepositoryStore(mock)

	t.Run("found", func(t *testing.T) {
		on, err := store.IsCommitOnBranch(Commit{Sha: "target"}, branchWithTip("main", &tip))
		require.NoError(t, err)
		require.True(t, on)
	})

	t.Run("not found", func(t *testing.T) {
		on, err := store.IsCommitOnBranch(Commit{Sha: "missing"}, branchWithTip("main", &tip))
		require.NoError(t, err)
		require.False(t, on)
	})

	t.Run("nil tip", func(t *testing.T) {
		on, err := store.IsCommitOnBranch(Commit{Sha: "x"}, branchWithTip("main", nil))
		require.NoError(t, err)
		require.False(t, on)
	})

	t.Run("empty commit", func(t *testing.T) {
		on, err := store.IsCommitOnBranch(Commit{}, branchWithTip("main", &tip))
		require.NoError(t, err)
		require.False(t, on)
	})
}

// --- GetNumberOfUncommittedChanges ---

func TestGetNumberOfUncommittedChanges(t *testing.T) {
	mock := &MockRepository{
		NumberOfUncommittedChangesFunc: func() (int, error) {
			return 3, nil
		},
	}

	store := NewRepositoryStore(mock)
	n, err := store.GetNumberOfUncommittedChanges()
	require.NoError(t, err)
	require.Equal(t, 3, n)
}

// --- FindCommitBranchWasBranchedFrom ---

func TestFindCommitBranchWasBranchedFrom(t *testing.T) {
	mainTip := newTestCommit("main-tip", time.Now(), "main tip")
	featureTip := newTestCommit("feat-tip", time.Now(), "feature tip")
	forkPoint := newTestCommit("fork", time.Now().Add(-time.Hour), "fork")

	mock := &MockRepository{
		BranchesFunc: func(filters ...PathFilter) ([]Branch, error) {
			return []Branch{
				branchWithTip("main", &mainTip),
				branchWithTip("feature/auth", &featureTip),
			}, nil
		},
		FindMergeBaseFunc: func(sha1, sha2 string) (string, error) {
			return "fork", nil
		},
		CommitFromShaFunc: func(sha string) (Commit, error) {
			if sha == "fork" {
				return forkPoint, nil
			}
			return Commit{}, errors.New("not found")
		},
	}

	store := NewRepositoryStore(mock)
	cfg, err := config.NewBuilder().Build()
	require.NoError(t, err)

	bc, err := store.FindCommitBranchWasBranchedFrom(
		branchWithTip("feature/auth", &featureTip), cfg,
	)
	require.NoError(t, err)
	require.Equal(t, "fork", bc.Commit.Sha)
	require.Equal(t, "main", bc.Branch.FriendlyName())
}

func TestFindCommitBranchWasBranchedFrom_NilTip(t *testing.T) {
	store := NewRepositoryStore(&MockRepository{})
	cfg, _ := config.NewBuilder().Build()

	bc, err := store.FindCommitBranchWasBranchedFrom(branchWithTip("main", nil), cfg)
	require.NoError(t, err)
	require.Equal(t, BranchCommit{}, bc)
}
