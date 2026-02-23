package calculator

import (
	"go-gitsemver/internal/context"
	"go-gitsemver/internal/git"
	"go-gitsemver/internal/semver"
	"go-gitsemver/internal/strategy"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMainline_AggregateIncrement(t *testing.T) {
	tip := newCommit("aaa0000000000000000000000000000000000000", "feat: add new feature")
	mid := newCommit("bbb0000000000000000000000000000000000000", "fix: patch bug")
	source := newCommit("ccc0000000000000000000000000000000000000", "v1.0.0")

	mock := &git.MockRepository{
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip, mid, source}, nil
		},
	}
	store := git.NewRepositoryStore(mock)
	incr := NewIncrementStrategyFinder(store)
	calc := NewMainlineVersionCalculator(store, incr)

	ctx := &context.GitVersionContext{
		CurrentCommit: tip,
		CurrentBranch: git.Branch{Name: git.NewReferenceName("refs/heads/main")},
	}
	bv := strategy.BaseVersion{
		SemanticVersion:   semver.SemanticVersion{Major: 1},
		ShouldIncrement:   true,
		BaseVersionSource: &source,
	}
	ec := defaultEC()
	ec.CommitMessageConvention = semver.CommitMessageConventionConventionalCommits

	ver, err := calc.FindMainlineModeVersion(ctx, bv, ec)
	require.NoError(t, err)
	// feat: → Minor is highest; 1.0.0 → 1.1.0
	require.Equal(t, int64(1), ver.Major)
	require.Equal(t, int64(1), ver.Minor)
	require.Equal(t, int64(0), ver.Patch)
	// 2 commits since source (tip + mid, excluding source).
	require.NotNil(t, ver.BuildMetaData.CommitsSinceTag)
	require.Equal(t, int64(2), *ver.BuildMetaData.CommitsSinceTag)
}

func TestMainline_NoCommitMessages(t *testing.T) {
	tip := newCommit("aaa0000000000000000000000000000000000000", "docs: update")
	source := newCommit("bbb0000000000000000000000000000000000000", "v1.0.0")

	mock := &git.MockRepository{
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip, source}, nil
		},
	}
	store := git.NewRepositoryStore(mock)
	incr := NewIncrementStrategyFinder(store)
	calc := NewMainlineVersionCalculator(store, incr)

	ctx := &context.GitVersionContext{
		CurrentCommit: tip,
		CurrentBranch: git.Branch{Name: git.NewReferenceName("refs/heads/main")},
	}
	bv := strategy.BaseVersion{
		SemanticVersion:   semver.SemanticVersion{Major: 1},
		ShouldIncrement:   true,
		BaseVersionSource: &source,
	}
	ec := defaultEC()
	ec.CommitMessageConvention = semver.CommitMessageConventionConventionalCommits
	ec.BranchIncrement = semver.IncrementStrategyPatch

	ver, err := calc.FindMainlineModeVersion(ctx, bv, ec)
	require.NoError(t, err)
	// docs: doesn't bump; ShouldIncrement → fallback to Patch: 1.0.0 → 1.0.1
	require.Equal(t, int64(1), ver.Major)
	require.Equal(t, int64(0), ver.Minor)
	require.Equal(t, int64(1), ver.Patch)
}

func TestMainline_NilSource(t *testing.T) {
	tip := newCommit("aaa0000000000000000000000000000000000000", "feat: initial")

	mock := &git.MockRepository{
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip}, nil
		},
	}
	store := git.NewRepositoryStore(mock)
	incr := NewIncrementStrategyFinder(store)
	calc := NewMainlineVersionCalculator(store, incr)

	ctx := &context.GitVersionContext{
		CurrentCommit: tip,
		CurrentBranch: git.Branch{Name: git.NewReferenceName("refs/heads/main")},
	}
	bv := strategy.BaseVersion{
		SemanticVersion: semver.SemanticVersion{Minor: 1},
		ShouldIncrement: true,
	}
	ec := defaultEC()
	ec.CommitMessageConvention = semver.CommitMessageConventionConventionalCommits

	ver, err := calc.FindMainlineModeVersion(ctx, bv, ec)
	require.NoError(t, err)
	require.Equal(t, int64(0), ver.Major)
	require.Equal(t, int64(2), ver.Minor)
}
