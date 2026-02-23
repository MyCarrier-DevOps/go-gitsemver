package git

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMockRepository_NilFuncsReturnDefaults(t *testing.T) {
	m := &MockRepository{}

	require.Equal(t, "", m.Path())
	require.Equal(t, "", m.WorkingDirectory())
	require.False(t, m.IsHeadDetached())

	head, err := m.Head()
	require.NoError(t, err)
	require.Equal(t, Branch{}, head)

	branches, err := m.Branches()
	require.NoError(t, err)
	require.Nil(t, branches)

	tags, err := m.Tags()
	require.NoError(t, err)
	require.Nil(t, tags)

	commit, err := m.CommitFromSha("abc")
	require.NoError(t, err)
	require.Equal(t, Commit{}, commit)

	log, err := m.CommitLog("a", "b")
	require.NoError(t, err)
	require.Nil(t, log)

	mainLog, err := m.MainlineCommitLog("a", "b")
	require.NoError(t, err)
	require.Nil(t, mainLog)

	bc, err := m.BranchCommits(Branch{})
	require.NoError(t, err)
	require.Nil(t, bc)

	prior, err := m.CommitsPriorTo(time.Now(), Branch{})
	require.NoError(t, err)
	require.Nil(t, prior)

	mb, err := m.FindMergeBase("a", "b")
	require.NoError(t, err)
	require.Equal(t, "", mb)

	containing, err := m.BranchesContainingCommit("abc")
	require.NoError(t, err)
	require.Nil(t, containing)

	changes, err := m.NumberOfUncommittedChanges()
	require.NoError(t, err)
	require.Equal(t, 0, changes)

	sha, err := m.PeelTagToCommit(Tag{TargetSha: "abc123"})
	require.NoError(t, err)
	require.Equal(t, "abc123", sha) // default returns TargetSha
}

func TestMockRepository_FuncFieldsCalled(t *testing.T) {
	expectedErr := errors.New("test error")
	expectedCommit := Commit{Sha: "abc123", Message: "test"}

	m := &MockRepository{
		PathFunc:             func() string { return "/repo/.git" },
		WorkingDirectoryFunc: func() string { return "/repo" },
		IsHeadDetachedFunc:   func() bool { return true },
		HeadFunc: func() (Branch, error) {
			return Branch{Name: NewBranchReferenceName("main")}, nil
		},
		CommitFromShaFunc: func(sha string) (Commit, error) {
			require.Equal(t, "abc123", sha)
			return expectedCommit, nil
		},
		FindMergeBaseFunc: func(sha1, sha2 string) (string, error) {
			return "", expectedErr
		},
		NumberOfUncommittedChangesFunc: func() (int, error) {
			return 5, nil
		},
	}

	require.Equal(t, "/repo/.git", m.Path())
	require.Equal(t, "/repo", m.WorkingDirectory())
	require.True(t, m.IsHeadDetached())

	head, err := m.Head()
	require.NoError(t, err)
	require.Equal(t, "main", head.FriendlyName())

	commit, err := m.CommitFromSha("abc123")
	require.NoError(t, err)
	require.Equal(t, expectedCommit, commit)

	_, err = m.FindMergeBase("a", "b")
	require.ErrorIs(t, err, expectedErr)

	changes, err := m.NumberOfUncommittedChanges()
	require.NoError(t, err)
	require.Equal(t, 5, changes)
}

func TestMockRepository_TagsWithFilters(t *testing.T) {
	var receivedFilters []PathFilter
	m := &MockRepository{
		TagsFunc: func(filters ...PathFilter) ([]Tag, error) {
			receivedFilters = filters
			return []Tag{{Name: NewReferenceName("refs/tags/v1.0.0"), TargetSha: "abc"}}, nil
		},
	}

	tags, err := m.Tags(PathFilter("src/"))
	require.NoError(t, err)
	require.Len(t, tags, 1)
	require.Equal(t, []PathFilter{PathFilter("src/")}, receivedFilters)
}

func TestMockRepository_CommitLogWithFilters(t *testing.T) {
	var gotFrom, gotTo string
	m := &MockRepository{
		CommitLogFunc: func(from, to string, filters ...PathFilter) ([]Commit, error) {
			gotFrom = from
			gotTo = to
			return []Commit{{Sha: "c1"}}, nil
		},
	}

	commits, err := m.CommitLog("abc", "def")
	require.NoError(t, err)
	require.Len(t, commits, 1)
	require.Equal(t, "abc", gotFrom)
	require.Equal(t, "def", gotTo)
}

func TestMockRepository_PeelTagToCommitCustom(t *testing.T) {
	m := &MockRepository{
		PeelTagToCommitFunc: func(tag Tag) (string, error) {
			return "peeled-sha", nil
		},
	}

	sha, err := m.PeelTagToCommit(Tag{TargetSha: "original"})
	require.NoError(t, err)
	require.Equal(t, "peeled-sha", sha)
}
