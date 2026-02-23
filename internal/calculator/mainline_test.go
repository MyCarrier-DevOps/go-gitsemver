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

func TestMainline_EachCommit_IncrementPerCommit(t *testing.T) {
	// v1.0.0 → fix → fix → feat → fix
	// GitVersion behavior: 1.0.1 → 1.0.2 → 1.1.0 → 1.1.1
	tip := newCommit("aaa0000000000000000000000000000000000000", "fix: last fix")
	c3 := newCommit("bbb0000000000000000000000000000000000000", "feat: new feature")
	c2 := newCommit("ccc0000000000000000000000000000000000000", "fix: second fix")
	c1 := newCommit("ddd0000000000000000000000000000000000000", "fix: first fix")
	source := newCommit("eee0000000000000000000000000000000000000", "v1.0.0")

	mock := &git.MockRepository{
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip, c3, c2, c1, source}, nil
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
	ec.MainlineIncrement = semver.MainlineIncrementEachCommit

	ver, err := calc.FindMainlineModeVersion(ctx, bv, ec)
	require.NoError(t, err)
	// fix→1.0.1, fix→1.0.2, feat→1.1.0, fix→1.1.1
	require.Equal(t, int64(1), ver.Major)
	require.Equal(t, int64(1), ver.Minor)
	require.Equal(t, int64(1), ver.Patch)
	require.Equal(t, int64(4), *ver.BuildMetaData.CommitsSinceTag)
}

func TestMainline_EachCommit_AllFixes(t *testing.T) {
	tip := newCommit("aaa0000000000000000000000000000000000000", "fix: third")
	c2 := newCommit("bbb0000000000000000000000000000000000000", "fix: second")
	c1 := newCommit("ccc0000000000000000000000000000000000000", "fix: first")
	source := newCommit("ddd0000000000000000000000000000000000000", "v1.0.0")

	mock := &git.MockRepository{
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip, c2, c1, source}, nil
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
	ec.MainlineIncrement = semver.MainlineIncrementEachCommit

	ver, err := calc.FindMainlineModeVersion(ctx, bv, ec)
	require.NoError(t, err)
	// fix→1.0.1, fix→1.0.2, fix→1.0.3
	require.Equal(t, int64(1), ver.Major)
	require.Equal(t, int64(0), ver.Minor)
	require.Equal(t, int64(3), ver.Patch)
}

func TestMainline_EachCommit_NoMatchingMessages(t *testing.T) {
	tip := newCommit("aaa0000000000000000000000000000000000000", "chore: cleanup")
	c1 := newCommit("bbb0000000000000000000000000000000000000", "docs: update readme")
	source := newCommit("ccc0000000000000000000000000000000000000", "v1.0.0")

	mock := &git.MockRepository{
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip, c1, source}, nil
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
	ec.MainlineIncrement = semver.MainlineIncrementEachCommit

	ver, err := calc.FindMainlineModeVersion(ctx, bv, ec)
	require.NoError(t, err)
	// No CC match → fallback to branch default (Patch) per commit: 1.0.1, 1.0.2
	require.Equal(t, int64(1), ver.Major)
	require.Equal(t, int64(0), ver.Minor)
	require.Equal(t, int64(2), ver.Patch)
}
