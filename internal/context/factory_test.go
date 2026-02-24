package context

import (
	"errors"
	"testing"
	"time"

	"github.com/MyCarrier-DevOps/go-gitsemver/internal/config"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/git"

	"github.com/stretchr/testify/require"
)

func defaultConfig(t *testing.T) *config.Config {
	t.Helper()
	cfg, err := config.NewBuilder().Build()
	require.NoError(t, err)
	return cfg
}

func newCommit(sha, message string) git.Commit {
	return git.Commit{Sha: sha, When: time.Now(), Message: message}
}

func newBranch(name string, tip *git.Commit) git.Branch {
	return git.Branch{
		Name: git.NewReferenceName("refs/heads/" + name),
		Tip:  tip,
	}
}

func TestNewContext_BasicBranch(t *testing.T) {
	tip := newCommit("abc123def456789012345678901234567890abcd", "initial commit")
	branch := newBranch("main", &tip)
	cfg := defaultConfig(t)

	mock := &git.MockRepository{
		HeadFunc:                       func() (git.Branch, error) { return branch, nil },
		NumberOfUncommittedChangesFunc: func() (int, error) { return 0, nil },
	}
	store := git.NewRepositoryStore(mock)

	ctx, err := NewContext(store, mock, cfg, Options{})
	require.NoError(t, err)
	require.Equal(t, "main", ctx.CurrentBranch.FriendlyName())
	require.Equal(t, tip.Sha, ctx.CurrentCommit.Sha)
	require.False(t, ctx.IsCurrentCommitTagged)
	require.Equal(t, 0, ctx.NumberOfUncommittedChanges)
}

func TestNewContext_WithTargetBranch(t *testing.T) {
	tip := newCommit("abc123def456789012345678901234567890abcd", "commit on develop")
	developBranch := newBranch("develop", &tip)
	cfg := defaultConfig(t)

	mock := &git.MockRepository{
		BranchesFunc: func(filters ...git.PathFilter) ([]git.Branch, error) {
			return []git.Branch{developBranch}, nil
		},
		NumberOfUncommittedChangesFunc: func() (int, error) { return 0, nil },
	}
	store := git.NewRepositoryStore(mock)

	ctx, err := NewContext(store, mock, cfg, Options{TargetBranch: "develop"})
	require.NoError(t, err)
	require.Equal(t, "develop", ctx.CurrentBranch.FriendlyName())
}

func TestNewContext_WithCommitID(t *testing.T) {
	tip := newCommit("abc123def456789012345678901234567890abcd", "tip commit")
	branch := newBranch("main", &tip)
	specificCommit := newCommit("def456789012345678901234567890abcdef1234", "older commit")
	cfg := defaultConfig(t)

	mock := &git.MockRepository{
		HeadFunc: func() (git.Branch, error) { return branch, nil },
		CommitFromShaFunc: func(sha string) (git.Commit, error) {
			if sha == specificCommit.Sha {
				return specificCommit, nil
			}
			return git.Commit{}, errors.New("not found")
		},
		NumberOfUncommittedChangesFunc: func() (int, error) { return 0, nil },
	}
	store := git.NewRepositoryStore(mock)

	ctx, err := NewContext(store, mock, cfg, Options{CommitID: specificCommit.Sha})
	require.NoError(t, err)
	require.Equal(t, specificCommit.Sha, ctx.CurrentCommit.Sha)
}

func TestNewContext_TaggedCommit(t *testing.T) {
	tip := newCommit("abc123def456789012345678901234567890abcd", "tagged commit")
	branch := newBranch("main", &tip)
	cfg := defaultConfig(t)

	mock := &git.MockRepository{
		HeadFunc: func() (git.Branch, error) { return branch, nil },
		TagsFunc: func(filters ...git.PathFilter) ([]git.Tag, error) {
			return []git.Tag{
				{Name: git.NewReferenceName("refs/tags/v1.0.0"), TargetSha: tip.Sha},
			}, nil
		},
		PeelTagToCommitFunc:            func(tag git.Tag) (string, error) { return tag.TargetSha, nil },
		CommitFromShaFunc:              func(sha string) (git.Commit, error) { return tip, nil },
		NumberOfUncommittedChangesFunc: func() (int, error) { return 0, nil },
	}
	store := git.NewRepositoryStore(mock)

	ctx, err := NewContext(store, mock, cfg, Options{})
	require.NoError(t, err)
	require.True(t, ctx.IsCurrentCommitTagged)
	require.Equal(t, int64(1), ctx.CurrentCommitTaggedVersion.Major)
}

func TestNewContext_UntaggedCommit(t *testing.T) {
	tip := newCommit("abc123def456789012345678901234567890abcd", "no tag")
	branch := newBranch("main", &tip)
	cfg := defaultConfig(t)

	mock := &git.MockRepository{
		HeadFunc:                       func() (git.Branch, error) { return branch, nil },
		TagsFunc:                       func(filters ...git.PathFilter) ([]git.Tag, error) { return nil, nil },
		NumberOfUncommittedChangesFunc: func() (int, error) { return 0, nil },
	}
	store := git.NewRepositoryStore(mock)

	ctx, err := NewContext(store, mock, cfg, Options{})
	require.NoError(t, err)
	require.False(t, ctx.IsCurrentCommitTagged)
}

func TestNewContext_DetachedHead_FindsBranch(t *testing.T) {
	tip := newCommit("abc123def456789012345678901234567890abcd", "detached")
	detached := git.Branch{
		Name:           git.NewReferenceName("refs/heads/HEAD"),
		Tip:            &tip,
		IsDetachedHead: true,
	}
	mainBranch := newBranch("main", &tip)
	cfg := defaultConfig(t)

	mock := &git.MockRepository{
		HeadFunc: func() (git.Branch, error) { return detached, nil },
		BranchesContainingCommitFunc: func(sha string) ([]git.Branch, error) {
			return []git.Branch{mainBranch}, nil
		},
		NumberOfUncommittedChangesFunc: func() (int, error) { return 0, nil },
	}
	store := git.NewRepositoryStore(mock)

	ctx, err := NewContext(store, mock, cfg, Options{})
	require.NoError(t, err)
	require.Equal(t, "main", ctx.CurrentBranch.FriendlyName())
	require.False(t, ctx.CurrentBranch.IsDetachedHead)
}

func TestNewContext_DetachedHead_PriorityPick(t *testing.T) {
	tip := newCommit("abc123def456789012345678901234567890abcd", "detached")
	detached := git.Branch{
		Name:           git.NewReferenceName("refs/heads/HEAD"),
		Tip:            &tip,
		IsDetachedHead: true,
	}
	featureBranch := newBranch("feature/auth", &tip)
	mainBranch := newBranch("main", &tip)
	cfg := defaultConfig(t)

	mock := &git.MockRepository{
		HeadFunc: func() (git.Branch, error) { return detached, nil },
		BranchesContainingCommitFunc: func(sha string) ([]git.Branch, error) {
			return []git.Branch{featureBranch, mainBranch}, nil
		},
		NumberOfUncommittedChangesFunc: func() (int, error) { return 0, nil },
	}
	store := git.NewRepositoryStore(mock)

	ctx, err := NewContext(store, mock, cfg, Options{})
	require.NoError(t, err)
	// main has higher default priority than feature in default config
	require.Equal(t, "main", ctx.CurrentBranch.FriendlyName())
}

func TestNewContext_DetachedHead_NoBranch(t *testing.T) {
	tip := newCommit("abc123def456789012345678901234567890abcd", "detached")
	detached := git.Branch{
		Name:           git.NewReferenceName("refs/heads/HEAD"),
		Tip:            &tip,
		IsDetachedHead: true,
	}
	cfg := defaultConfig(t)

	mock := &git.MockRepository{
		HeadFunc: func() (git.Branch, error) { return detached, nil },
		BranchesContainingCommitFunc: func(sha string) ([]git.Branch, error) {
			return nil, nil
		},
		NumberOfUncommittedChangesFunc: func() (int, error) { return 0, nil },
	}
	store := git.NewRepositoryStore(mock)

	ctx, err := NewContext(store, mock, cfg, Options{})
	require.NoError(t, err)
	require.True(t, ctx.CurrentBranch.IsDetachedHead)
}

func TestNewContext_UncommittedChanges(t *testing.T) {
	tip := newCommit("abc123def456789012345678901234567890abcd", "tip")
	branch := newBranch("main", &tip)
	cfg := defaultConfig(t)

	mock := &git.MockRepository{
		HeadFunc:                       func() (git.Branch, error) { return branch, nil },
		NumberOfUncommittedChangesFunc: func() (int, error) { return 5, nil },
	}
	store := git.NewRepositoryStore(mock)

	ctx, err := NewContext(store, mock, cfg, Options{})
	require.NoError(t, err)
	require.Equal(t, 5, ctx.NumberOfUncommittedChanges)
}

func TestNewContext_ErrorTargetBranch(t *testing.T) {
	cfg := defaultConfig(t)

	mock := &git.MockRepository{
		BranchesFunc: func(filters ...git.PathFilter) ([]git.Branch, error) {
			return nil, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	_, err := NewContext(store, mock, cfg, Options{TargetBranch: "nonexistent"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "resolving target branch")
}

func TestNewContext_ErrorCommit(t *testing.T) {
	branch := git.Branch{Name: git.NewReferenceName("refs/heads/main")} // nil tip
	cfg := defaultConfig(t)

	mock := &git.MockRepository{
		HeadFunc: func() (git.Branch, error) { return branch, nil },
	}
	store := git.NewRepositoryStore(mock)

	_, err := NewContext(store, mock, cfg, Options{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "resolving current commit")
}

func TestPickBestBranch_Empty(t *testing.T) {
	cfg := defaultConfig(t)
	_, ok := pickBestBranch(nil, cfg)
	require.False(t, ok)
}

func TestPickBestBranch_SingleCandidate(t *testing.T) {
	tip := newCommit("abc123def456789012345678901234567890abcd", "tip")
	branch := newBranch("main", &tip)
	cfg := defaultConfig(t)

	result, ok := pickBestBranch([]git.Branch{branch}, cfg)
	require.True(t, ok)
	require.Equal(t, "main", result.FriendlyName())
}

func TestPickBestBranch_SkipsRemote(t *testing.T) {
	tip := newCommit("abc123def456789012345678901234567890abcd", "tip")
	remote := git.Branch{
		Name:     git.NewReferenceName("refs/remotes/origin/main"),
		Tip:      &tip,
		IsRemote: true,
	}
	local := newBranch("develop", &tip)
	cfg := defaultConfig(t)

	result, ok := pickBestBranch([]git.Branch{remote, local}, cfg)
	require.True(t, ok)
	require.Equal(t, "develop", result.FriendlyName())
}

func TestPickBestBranch_AllRemote(t *testing.T) {
	tip := newCommit("abc123def456789012345678901234567890abcd", "tip")
	remote := git.Branch{
		Name:     git.NewReferenceName("refs/remotes/origin/main"),
		Tip:      &tip,
		IsRemote: true,
	}
	cfg := defaultConfig(t)

	result, ok := pickBestBranch([]git.Branch{remote}, cfg)
	require.True(t, ok)
	require.True(t, result.IsRemote) // fallback to first remote
}
