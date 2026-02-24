package calculator

import (
	"testing"

	"github.com/MyCarrier-DevOps/go-gitsemver/internal/context"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/git"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/semver"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/strategy"

	"github.com/stretchr/testify/require"
)

func TestMainline_AggregateIncrement(t *testing.T) {
	tip := newCommit("aaa0000000000000000000000000000000000000", "feat: add new feature")
	mid := newCommit("bbb0000000000000000000000000000000000000", "fix: patch bug")
	source := newCommit("ccc0000000000000000000000000000000000000", "v1.0.0")

	logFunc := func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
		return []git.Commit{tip, mid, source}, nil
	}
	mock := &git.MockRepository{
		CommitLogFunc:         logFunc,
		MainlineCommitLogFunc: logFunc,
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

	ver, _, err := calc.FindMainlineModeVersion(ctx, bv, ec, false)
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

	logFunc := func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
		return []git.Commit{tip, source}, nil
	}
	mock := &git.MockRepository{
		CommitLogFunc:         logFunc,
		MainlineCommitLogFunc: logFunc,
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

	ver, _, err := calc.FindMainlineModeVersion(ctx, bv, ec, false)
	require.NoError(t, err)
	// docs: doesn't bump; ShouldIncrement → fallback to Patch: 1.0.0 → 1.0.1
	require.Equal(t, int64(1), ver.Major)
	require.Equal(t, int64(0), ver.Minor)
	require.Equal(t, int64(1), ver.Patch)
}

func TestMainline_NilSource(t *testing.T) {
	tip := newCommit("aaa0000000000000000000000000000000000000", "feat: initial")

	logFunc := func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
		return []git.Commit{tip}, nil
	}
	mock := &git.MockRepository{
		CommitLogFunc:         logFunc,
		MainlineCommitLogFunc: logFunc,
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

	ver, _, err := calc.FindMainlineModeVersion(ctx, bv, ec, false)
	require.NoError(t, err)
	require.Equal(t, int64(0), ver.Major)
	require.Equal(t, int64(2), ver.Minor)
}

func TestMainline_EachCommit_IncrementPerCommit(t *testing.T) {
	// v1.0.0 → fix → fix → feat → fix
	// Per-commit behavior: 1.0.1 → 1.0.2 → 1.1.0 → 1.1.1
	tip := newCommit("aaa0000000000000000000000000000000000000", "fix: last fix")
	c3 := newCommit("bbb0000000000000000000000000000000000000", "feat: new feature")
	c2 := newCommit("ccc0000000000000000000000000000000000000", "fix: second fix")
	c1 := newCommit("ddd0000000000000000000000000000000000000", "fix: first fix")
	source := newCommit("eee0000000000000000000000000000000000000", "v1.0.0")

	logFunc := func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
		return []git.Commit{tip, c3, c2, c1, source}, nil
	}
	mock := &git.MockRepository{
		CommitLogFunc:         logFunc,
		MainlineCommitLogFunc: logFunc,
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

	ver, _, err := calc.FindMainlineModeVersion(ctx, bv, ec, false)
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

	logFunc := func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
		return []git.Commit{tip, c2, c1, source}, nil
	}
	mock := &git.MockRepository{
		CommitLogFunc:         logFunc,
		MainlineCommitLogFunc: logFunc,
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

	ver, _, err := calc.FindMainlineModeVersion(ctx, bv, ec, false)
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

	logFunc := func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
		return []git.Commit{tip, c1, source}, nil
	}
	mock := &git.MockRepository{
		CommitLogFunc:         logFunc,
		MainlineCommitLogFunc: logFunc,
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

	ver, _, err := calc.FindMainlineModeVersion(ctx, bv, ec, false)
	require.NoError(t, err)
	// No CC match → fallback to branch default (Patch) per commit: 1.0.1, 1.0.2
	require.Equal(t, int64(1), ver.Major)
	require.Equal(t, int64(0), ver.Minor)
	require.Equal(t, int64(2), ver.Patch)
}

// ---------------------------------------------------------------------------
// Explain mode tests
// ---------------------------------------------------------------------------

func TestMainline_AggregateIncrement_Explain(t *testing.T) {
	tip := newCommit("aaa0000000000000000000000000000000000000", "feat: add new feature")
	source := newCommit("ccc0000000000000000000000000000000000000", "v1.0.0")

	logFunc := func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
		return []git.Commit{tip, source}, nil
	}
	mock := &git.MockRepository{
		CommitLogFunc:         logFunc,
		MainlineCommitLogFunc: logFunc,
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

	ver, exp, err := calc.FindMainlineModeVersion(ctx, bv, ec, true)
	require.NoError(t, err)
	require.Equal(t, int64(1), ver.Major)
	require.Equal(t, int64(1), ver.Minor)
	require.NotNil(t, exp, "explanation should not be nil with explain=true")
	require.NotEmpty(t, exp.Steps)
}

func TestMainline_AggregateIncrement_NoExplain(t *testing.T) {
	tip := newCommit("aaa0000000000000000000000000000000000000", "feat: add new feature")
	source := newCommit("ccc0000000000000000000000000000000000000", "v1.0.0")

	logFunc := func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
		return []git.Commit{tip, source}, nil
	}
	mock := &git.MockRepository{
		CommitLogFunc:         logFunc,
		MainlineCommitLogFunc: logFunc,
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

	ver, exp, err := calc.FindMainlineModeVersion(ctx, bv, ec, false)
	require.NoError(t, err)
	require.Equal(t, int64(1), ver.Major)
	require.Nil(t, exp, "explanation should be nil with explain=false")
}

func TestMainline_EachCommit_Explain(t *testing.T) {
	tip := newCommit("aaa0000000000000000000000000000000000000", "fix: last fix")
	c1 := newCommit("bbb0000000000000000000000000000000000000", "feat: feature")
	source := newCommit("ccc0000000000000000000000000000000000000", "v1.0.0")

	logFunc := func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
		return []git.Commit{tip, c1, source}, nil
	}
	mock := &git.MockRepository{
		CommitLogFunc:         logFunc,
		MainlineCommitLogFunc: logFunc,
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

	ver, exp, err := calc.FindMainlineModeVersion(ctx, bv, ec, true)
	require.NoError(t, err)
	// feat→1.1.0, fix→1.1.1
	require.Equal(t, int64(1), ver.Major)
	require.Equal(t, int64(1), ver.Minor)
	require.Equal(t, int64(1), ver.Patch)
	require.NotNil(t, exp, "explanation should not be nil with explain=true")
	require.NotEmpty(t, exp.Steps)

	// Should contain per-commit and final version entries.
	found := false
	for _, step := range exp.Steps {
		if len(step) > 0 && step[0:4] == "main" {
			found = true
		}
	}
	require.True(t, found, "should have mainline mode header step")
}

func TestMainline_EachCommit_PreV1_CapMajor(t *testing.T) {
	tip := newCommit("aaa0000000000000000000000000000000000000", "feat!: breaking change")
	source := newCommit("bbb0000000000000000000000000000000000000", "v0.1.0")

	logFunc := func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
		return []git.Commit{tip, source}, nil
	}
	mock := &git.MockRepository{
		CommitLogFunc:         logFunc,
		MainlineCommitLogFunc: logFunc,
	}
	store := git.NewRepositoryStore(mock)
	incr := NewIncrementStrategyFinder(store)
	calc := NewMainlineVersionCalculator(store, incr)

	ctx := &context.GitVersionContext{
		CurrentCommit: tip,
		CurrentBranch: git.Branch{Name: git.NewReferenceName("refs/heads/main")},
	}
	bv := strategy.BaseVersion{
		SemanticVersion:   semver.SemanticVersion{Major: 0, Minor: 1},
		ShouldIncrement:   true,
		BaseVersionSource: &source,
	}
	ec := defaultEC()
	ec.CommitMessageConvention = semver.CommitMessageConventionConventionalCommits
	ec.MainlineIncrement = semver.MainlineIncrementEachCommit

	ver, _, err := calc.FindMainlineModeVersion(ctx, bv, ec, false)
	require.NoError(t, err)
	// Major is capped to Minor for pre-1.0: 0.1.0 → 0.2.0
	require.Equal(t, int64(0), ver.Major)
	require.Equal(t, int64(2), ver.Minor)
	require.Equal(t, int64(0), ver.Patch)
}

func TestMainline_Aggregate_NoFieldAndShouldIncrement(t *testing.T) {
	tip := newCommit("aaa0000000000000000000000000000000000000", "docs: update readme")
	source := newCommit("bbb0000000000000000000000000000000000000", "v2.0.0")

	logFunc := func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
		return []git.Commit{tip, source}, nil
	}
	mock := &git.MockRepository{
		CommitLogFunc:         logFunc,
		MainlineCommitLogFunc: logFunc,
	}
	store := git.NewRepositoryStore(mock)
	incr := NewIncrementStrategyFinder(store)
	calc := NewMainlineVersionCalculator(store, incr)

	ctx := &context.GitVersionContext{
		CurrentCommit: tip,
		CurrentBranch: git.Branch{Name: git.NewReferenceName("refs/heads/main")},
	}
	bv := strategy.BaseVersion{
		SemanticVersion:   semver.SemanticVersion{Major: 2},
		ShouldIncrement:   true,
		BaseVersionSource: &source,
	}
	ec := defaultEC()
	ec.CommitMessageConvention = semver.CommitMessageConventionConventionalCommits
	ec.BranchIncrement = semver.IncrementStrategyMinor

	ver, _, err := calc.FindMainlineModeVersion(ctx, bv, ec, false)
	require.NoError(t, err)
	// docs: → None from CC, but ShouldIncrement uses branch default Minor: 2.0.0 → 2.1.0
	require.Equal(t, int64(2), ver.Major)
	require.Equal(t, int64(1), ver.Minor)
}

func TestMainline_Aggregate_NoFieldInheritFallsToPatch(t *testing.T) {
	tip := newCommit("aaa0000000000000000000000000000000000000", "chore: cleanup")
	source := newCommit("bbb0000000000000000000000000000000000000", "v3.0.0")

	logFunc := func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
		return []git.Commit{tip, source}, nil
	}
	mock := &git.MockRepository{
		CommitLogFunc:         logFunc,
		MainlineCommitLogFunc: logFunc,
	}
	store := git.NewRepositoryStore(mock)
	incr := NewIncrementStrategyFinder(store)
	calc := NewMainlineVersionCalculator(store, incr)

	ctx := &context.GitVersionContext{
		CurrentCommit: tip,
		CurrentBranch: git.Branch{Name: git.NewReferenceName("refs/heads/main")},
	}
	bv := strategy.BaseVersion{
		SemanticVersion:   semver.SemanticVersion{Major: 3},
		ShouldIncrement:   true,
		BaseVersionSource: &source,
	}
	ec := defaultEC()
	ec.CommitMessageConvention = semver.CommitMessageConventionConventionalCommits
	ec.BranchIncrement = semver.IncrementStrategyInherit

	ver, _, err := calc.FindMainlineModeVersion(ctx, bv, ec, false)
	require.NoError(t, err)
	// chore: → None, ShouldIncrement, Inherit → Patch fallback: 3.0.0 → 3.0.1
	require.Equal(t, int64(3), ver.Major)
	require.Equal(t, int64(0), ver.Minor)
	require.Equal(t, int64(1), ver.Patch)
}

func TestMainline_Aggregate_NotShouldIncrement(t *testing.T) {
	tip := newCommit("aaa0000000000000000000000000000000000000", "docs: update")
	source := newCommit("bbb0000000000000000000000000000000000000", "v1.5.0")

	logFunc := func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
		return []git.Commit{tip, source}, nil
	}
	mock := &git.MockRepository{
		CommitLogFunc:         logFunc,
		MainlineCommitLogFunc: logFunc,
	}
	store := git.NewRepositoryStore(mock)
	incr := NewIncrementStrategyFinder(store)
	calc := NewMainlineVersionCalculator(store, incr)

	ctx := &context.GitVersionContext{
		CurrentCommit: tip,
		CurrentBranch: git.Branch{Name: git.NewReferenceName("refs/heads/main")},
	}
	bv := strategy.BaseVersion{
		SemanticVersion:   semver.SemanticVersion{Major: 1, Minor: 5},
		ShouldIncrement:   false,
		BaseVersionSource: &source,
	}
	ec := defaultEC()
	ec.CommitMessageConvention = semver.CommitMessageConventionConventionalCommits

	ver, _, err := calc.FindMainlineModeVersion(ctx, bv, ec, false)
	require.NoError(t, err)
	// No CC match, ShouldIncrement=false → no increment
	require.Equal(t, int64(1), ver.Major)
	require.Equal(t, int64(5), ver.Minor)
	require.Equal(t, int64(0), ver.Patch)
}
