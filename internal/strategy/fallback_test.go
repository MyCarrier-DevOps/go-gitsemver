package strategy

import (
	"testing"
	"time"

	"github.com/MyCarrier-DevOps/go-gitsemver/internal/config"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/context"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/git"

	"github.com/stretchr/testify/require"
)

func TestFallback_ReturnsBaseVersion(t *testing.T) {
	rootCommit := git.Commit{Sha: "aaa0000000000000000000000000000000000000", When: time.Now(), Message: "initial"}
	tip := git.Commit{Sha: "bbb0000000000000000000000000000000000000", When: time.Now(), Message: "latest"}

	mock := &git.MockRepository{
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip, rootCommit}, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{
		CurrentBranch: git.Branch{Tip: &tip},
		CurrentCommit: tip,
	}
	ec := config.EffectiveConfiguration{BaseVersion: "1.0.0"}

	s := NewFallbackStrategy(store)
	require.Equal(t, "Fallback", s.Name())

	versions, err := s.GetBaseVersions(ctx, ec, false)
	require.NoError(t, err)
	require.Len(t, versions, 1)
	require.Equal(t, int64(1), versions[0].SemanticVersion.Major)
	require.Equal(t, int64(0), versions[0].SemanticVersion.Minor)
	require.False(t, versions[0].ShouldIncrement)
	require.Equal(t, rootCommit.Sha, versions[0].BaseVersionSource.Sha)
	require.Equal(t, "Fallback base version", versions[0].Source)
}

func TestFallback_CustomBaseVersion(t *testing.T) {
	tip := git.Commit{Sha: "bbb0000000000000000000000000000000000000", When: time.Now(), Message: "latest"}

	mock := &git.MockRepository{
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip}, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{
		CurrentBranch: git.Branch{Tip: &tip},
		CurrentCommit: tip,
	}
	// A configured base version must be used verbatim. Asserting on 1.0.0 here
	// would not distinguish it from the unparseable hard fallback below.
	ec := config.EffectiveConfiguration{BaseVersion: "2.5.3"}

	s := NewFallbackStrategy(store)
	versions, err := s.GetBaseVersions(ctx, ec, false)
	require.NoError(t, err)
	require.Len(t, versions, 1)
	require.Equal(t, "2.5.3", versions[0].SemanticVersion.SemVer())
}

// TestFallback_UnparseableBaseVersion pins the hard fallback to 1.0.0.
func TestFallback_UnparseableBaseVersion(t *testing.T) {
	tip := git.Commit{Sha: "bbb0000000000000000000000000000000000000", When: time.Now(), Message: "latest"}

	mock := &git.MockRepository{
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip}, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{
		CurrentBranch: git.Branch{Tip: &tip},
		CurrentCommit: tip,
	}
	ec := config.EffectiveConfiguration{BaseVersion: "not-a-version"}

	versions, err := NewFallbackStrategy(store).GetBaseVersions(ctx, ec, false)
	require.NoError(t, err)
	require.Len(t, versions, 1)
	require.Equal(t, "1.0.0", versions[0].SemanticVersion.SemVer())
}

func TestFallback_NilTip(t *testing.T) {
	store := git.NewRepositoryStore(&git.MockRepository{})

	ctx := &context.GitVersionContext{
		CurrentBranch: git.Branch{},
	}
	ec := config.EffectiveConfiguration{BaseVersion: "1.0.0"}

	s := NewFallbackStrategy(store)
	_, err := s.GetBaseVersions(ctx, ec, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no commits found")
}

func TestFallback_Explanation(t *testing.T) {
	tip := git.Commit{Sha: "bbb0000000000000000000000000000000000000", When: time.Now(), Message: "latest"}

	mock := &git.MockRepository{
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip}, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{
		CurrentBranch: git.Branch{Tip: &tip},
		CurrentCommit: tip,
	}
	ec := config.EffectiveConfiguration{BaseVersion: "1.0.0"}

	s := NewFallbackStrategy(store)
	versions, err := s.GetBaseVersions(ctx, ec, true)
	require.NoError(t, err)
	require.Len(t, versions, 1)
	require.NotNil(t, versions[0].Explanation)
	require.Equal(t, "Fallback", versions[0].Explanation.Strategy)
	require.NotEmpty(t, versions[0].Explanation.Steps)
}
