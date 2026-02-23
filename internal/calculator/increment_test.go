package calculator

import (
	"go-gitsemver/internal/config"
	"go-gitsemver/internal/context"
	"go-gitsemver/internal/git"
	"go-gitsemver/internal/semver"
	"go-gitsemver/internal/strategy"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newCommit(sha, msg string) git.Commit {
	return git.Commit{Sha: sha, When: time.Now(), Message: msg}
}

func defaultEC() config.EffectiveConfiguration {
	return config.EffectiveConfiguration{
		TagPrefix:                 "[vV]",
		BranchIncrement:           semver.IncrementStrategyPatch,
		CommitMessageIncrementing: semver.CommitMessageIncrementEnabled,
		CommitMessageConvention:   semver.CommitMessageConventionBoth,
		MajorVersionBumpMessage:   `\+semver:\s?(breaking|major)`,
		MinorVersionBumpMessage:   `\+semver:\s?(feature|minor)`,
		PatchVersionBumpMessage:   `\+semver:\s?(fix|patch)`,
		NoBumpMessage:             `\+semver:\s?(none|skip)`,
	}
}

func TestConventionalCommit_Feat(t *testing.T) {
	require.Equal(t, semver.VersionFieldMinor, analyzeConventionalCommit("feat: add login"))
}

func TestConventionalCommit_FeatWithScope(t *testing.T) {
	require.Equal(t, semver.VersionFieldMinor, analyzeConventionalCommit("feat(auth): add login"))
}

func TestConventionalCommit_Fix(t *testing.T) {
	require.Equal(t, semver.VersionFieldPatch, analyzeConventionalCommit("fix: null pointer"))
}

func TestConventionalCommit_Breaking(t *testing.T) {
	require.Equal(t, semver.VersionFieldMajor, analyzeConventionalCommit("feat!: remove api"))
}

func TestConventionalCommit_BreakingFooter(t *testing.T) {
	msg := "feat: change API\n\nBREAKING CHANGE: removed old endpoint"
	require.Equal(t, semver.VersionFieldMajor, analyzeConventionalCommit(msg))
}

func TestConventionalCommit_BreakingChangeHyphen(t *testing.T) {
	msg := "feat: change API\n\nBREAKING-CHANGE: removed old endpoint"
	require.Equal(t, semver.VersionFieldMajor, analyzeConventionalCommit(msg))
}

func TestConventionalCommit_Chore(t *testing.T) {
	require.Equal(t, semver.VersionFieldNone, analyzeConventionalCommit("chore: update deps"))
}

func TestConventionalCommit_NotConventional(t *testing.T) {
	require.Equal(t, semver.VersionFieldNone, analyzeConventionalCommit("update readme"))
}

func TestBumpDirective_Major(t *testing.T) {
	ec := defaultEC()
	require.Equal(t, semver.VersionFieldMajor, analyzeBumpDirective("some change +semver: major", ec))
}

func TestBumpDirective_Minor(t *testing.T) {
	ec := defaultEC()
	require.Equal(t, semver.VersionFieldMinor, analyzeBumpDirective("add feature +semver: feature", ec))
}

func TestBumpDirective_Patch(t *testing.T) {
	ec := defaultEC()
	require.Equal(t, semver.VersionFieldPatch, analyzeBumpDirective("fix bug +semver: fix", ec))
}

func TestBumpDirective_None(t *testing.T) {
	ec := defaultEC()
	require.Equal(t, semver.VersionFieldNone, analyzeBumpDirective("regular commit", ec))
}

func TestDetermineIncrement_ConventionalCommits(t *testing.T) {
	tip := newCommit("aaa0000000000000000000000000000000000000", "feat: add login")
	source := newCommit("bbb0000000000000000000000000000000000000", "initial")

	mock := &git.MockRepository{
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip, source}, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{
		CurrentCommit: tip,
	}
	bv := strategy.BaseVersion{
		SemanticVersion:   semver.SemanticVersion{Major: 1, Minor: 0, Patch: 0},
		ShouldIncrement:   true,
		BaseVersionSource: &source,
	}
	ec := defaultEC()
	ec.CommitMessageConvention = semver.CommitMessageConventionConventionalCommits

	finder := NewIncrementStrategyFinder(store)
	field, err := finder.DetermineIncrementedField(ctx, bv, ec)
	require.NoError(t, err)
	require.Equal(t, semver.VersionFieldMinor, field)
}

func TestDetermineIncrement_BranchDefaultWhenHigher(t *testing.T) {
	tip := newCommit("aaa0000000000000000000000000000000000000", "docs: update readme")
	source := newCommit("bbb0000000000000000000000000000000000000", "initial")

	mock := &git.MockRepository{
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip, source}, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{CurrentCommit: tip}
	bv := strategy.BaseVersion{
		SemanticVersion:   semver.SemanticVersion{Major: 1},
		ShouldIncrement:   true,
		BaseVersionSource: &source,
	}
	ec := defaultEC()
	ec.BranchIncrement = semver.IncrementStrategyMinor

	finder := NewIncrementStrategyFinder(store)
	field, err := finder.DetermineIncrementedField(ctx, bv, ec)
	require.NoError(t, err)
	// docs: doesn't bump, so branch default (Minor) should be used.
	require.Equal(t, semver.VersionFieldMinor, field)
}

func TestDetermineIncrement_DisabledUsesBranchDefault(t *testing.T) {
	store := git.NewRepositoryStore(&git.MockRepository{})

	ctx := &context.GitVersionContext{}
	bv := strategy.BaseVersion{
		SemanticVersion: semver.SemanticVersion{Major: 1},
		ShouldIncrement: true,
	}
	ec := defaultEC()
	ec.CommitMessageIncrementing = semver.CommitMessageIncrementDisabled
	ec.BranchIncrement = semver.IncrementStrategyMinor

	finder := NewIncrementStrategyFinder(store)
	field, err := finder.DetermineIncrementedField(ctx, bv, ec)
	require.NoError(t, err)
	require.Equal(t, semver.VersionFieldMinor, field)
}

func TestDetermineIncrement_MergeMessageOnly(t *testing.T) {
	nonMerge := git.Commit{
		Sha: "aaa0000000000000000000000000000000000000", Parents: []string{"p1"},
		When: time.Now(), Message: "feat: add login",
	}
	merge := git.Commit{
		Sha: "bbb0000000000000000000000000000000000000", Parents: []string{"p1", "p2"},
		When: time.Now(), Message: "fix: resolve conflict",
	}
	source := newCommit("ccc0000000000000000000000000000000000000", "initial")

	mock := &git.MockRepository{
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{nonMerge, merge, source}, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{CurrentCommit: nonMerge}
	bv := strategy.BaseVersion{
		SemanticVersion:   semver.SemanticVersion{Major: 1},
		ShouldIncrement:   true,
		BaseVersionSource: &source,
	}
	ec := defaultEC()
	ec.CommitMessageIncrementing = semver.CommitMessageIncrementMergeMessageOnly
	ec.CommitMessageConvention = semver.CommitMessageConventionConventionalCommits

	finder := NewIncrementStrategyFinder(store)
	field, err := finder.DetermineIncrementedField(ctx, bv, ec)
	require.NoError(t, err)
	// Only merge commit analyzed: fix: → Patch. Non-merge feat: is skipped.
	require.Equal(t, semver.VersionFieldPatch, field)
}

func TestDetermineIncrement_CapMajorBelow1(t *testing.T) {
	tip := newCommit("aaa0000000000000000000000000000000000000", "feat!: breaking change")
	source := newCommit("bbb0000000000000000000000000000000000000", "initial")

	mock := &git.MockRepository{
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip, source}, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{CurrentCommit: tip}
	bv := strategy.BaseVersion{
		SemanticVersion:   semver.SemanticVersion{Major: 0, Minor: 5},
		ShouldIncrement:   true,
		BaseVersionSource: &source,
	}
	ec := defaultEC()
	ec.CommitMessageConvention = semver.CommitMessageConventionConventionalCommits

	finder := NewIncrementStrategyFinder(store)
	field, err := finder.DetermineIncrementedField(ctx, bv, ec)
	require.NoError(t, err)
	// Major capped to Minor when version < 1.0.0.
	require.Equal(t, semver.VersionFieldMinor, field)
}

func TestDetermineIncrement_BothConventions(t *testing.T) {
	tip := newCommit("aaa0000000000000000000000000000000000000", "fix: bug +semver: minor")
	source := newCommit("bbb0000000000000000000000000000000000000", "initial")

	mock := &git.MockRepository{
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip, source}, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{CurrentCommit: tip}
	bv := strategy.BaseVersion{
		SemanticVersion:   semver.SemanticVersion{Major: 1},
		ShouldIncrement:   true,
		BaseVersionSource: &source,
	}
	ec := defaultEC()

	finder := NewIncrementStrategyFinder(store)
	field, err := finder.DetermineIncrementedField(ctx, bv, ec)
	require.NoError(t, err)
	// fix: → Patch, +semver: minor → Minor. Both mode takes highest = Minor.
	require.Equal(t, semver.VersionFieldMinor, field)
}

func TestDetermineIncrement_NoShouldIncrementNoDefault(t *testing.T) {
	tip := newCommit("aaa0000000000000000000000000000000000000", "docs: update")
	source := newCommit("bbb0000000000000000000000000000000000000", "initial")

	mock := &git.MockRepository{
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip, source}, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{CurrentCommit: tip}
	bv := strategy.BaseVersion{
		SemanticVersion:   semver.SemanticVersion{Major: 1},
		ShouldIncrement:   false,
		BaseVersionSource: &source,
	}
	ec := defaultEC()

	finder := NewIncrementStrategyFinder(store)
	field, err := finder.DetermineIncrementedField(ctx, bv, ec)
	require.NoError(t, err)
	// docs: doesn't bump and ShouldIncrement is false → None.
	require.Equal(t, semver.VersionFieldNone, field)
}
