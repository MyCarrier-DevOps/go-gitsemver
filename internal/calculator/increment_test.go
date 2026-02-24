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

// ---------------------------------------------------------------------------
// DetermineIncrementedFieldExplained tests
// ---------------------------------------------------------------------------

func TestDetermineIncrementExplained_RecordsSteps(t *testing.T) {
	tip := newCommit("aaa0000000000000000000000000000000000000", "feat: add login")
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
	ec.CommitMessageConvention = semver.CommitMessageConventionConventionalCommits

	finder := NewIncrementStrategyFinder(store)
	result, err := finder.DetermineIncrementedFieldExplained(ctx, bv, ec, true)
	require.NoError(t, err)
	require.Equal(t, semver.VersionFieldMinor, result.Field)
	require.NotNil(t, result.Explanation)
	require.NotEmpty(t, result.Explanation.Steps)
}

func TestDetermineIncrementExplained_NoExplain(t *testing.T) {
	tip := newCommit("aaa0000000000000000000000000000000000000", "feat: add login")
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
	ec.CommitMessageConvention = semver.CommitMessageConventionConventionalCommits

	finder := NewIncrementStrategyFinder(store)
	result, err := finder.DetermineIncrementedFieldExplained(ctx, bv, ec, false)
	require.NoError(t, err)
	require.Equal(t, semver.VersionFieldMinor, result.Field)
	require.Nil(t, result.Explanation)
}

func TestDetermineIncrementExplained_Disabled(t *testing.T) {
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
	result, err := finder.DetermineIncrementedFieldExplained(ctx, bv, ec, true)
	require.NoError(t, err)
	require.Equal(t, semver.VersionFieldMinor, result.Field)
	require.NotNil(t, result.Explanation)
	require.NotEmpty(t, result.Explanation.Steps)
}

// ---------------------------------------------------------------------------
// conventionName tests
// ---------------------------------------------------------------------------

func TestConventionName_ConventionalCommits(t *testing.T) {
	ec := defaultEC()
	ec.CommitMessageConvention = semver.CommitMessageConventionConventionalCommits
	require.Equal(t, "Conventional Commits", conventionName("feat: add login", ec))
}

func TestConventionName_BumpDirective(t *testing.T) {
	ec := defaultEC()
	ec.CommitMessageConvention = semver.CommitMessageConventionBumpDirective
	require.Equal(t, "Bump Directive", conventionName("some change +semver: minor", ec))
}

func TestConventionName_Both_CCWins(t *testing.T) {
	ec := defaultEC()
	ec.CommitMessageConvention = semver.CommitMessageConventionBoth
	require.Equal(t, "Conventional Commits", conventionName("feat: add login", ec))
}

func TestConventionName_Both_BDWins(t *testing.T) {
	ec := defaultEC()
	ec.CommitMessageConvention = semver.CommitMessageConventionBoth
	require.Equal(t, "Bump Directive", conventionName("random commit +semver: major", ec))
}

func TestConventionName_Both_NeitherMatch(t *testing.T) {
	ec := defaultEC()
	ec.CommitMessageConvention = semver.CommitMessageConventionBoth
	require.Equal(t, "Conventional Commits", conventionName("docs: update readme", ec))
}

func TestConventionName_Unknown(t *testing.T) {
	ec := defaultEC()
	ec.CommitMessageConvention = 99 // unknown
	require.Equal(t, "Bump Directive", conventionName("foo", ec))
}

// ---------------------------------------------------------------------------
// tryMatch tests
// ---------------------------------------------------------------------------

func TestTryMatch_EmptyPattern(t *testing.T) {
	require.False(t, tryMatch("some message", ""))
}

func TestTryMatch_InvalidRegex(t *testing.T) {
	require.False(t, tryMatch("some message", "[invalid"))
}

func TestTryMatch_Matches(t *testing.T) {
	require.True(t, tryMatch("commit +semver: major", `\+semver:\s?(breaking|major)`))
}

func TestTryMatch_NoMatch(t *testing.T) {
	require.False(t, tryMatch("regular commit", `\+semver:\s?(breaking|major)`))
}

// ---------------------------------------------------------------------------
// branchDefault tests
// ---------------------------------------------------------------------------

func TestBranchDefault_NotShouldIncrement(t *testing.T) {
	store := git.NewRepositoryStore(&git.MockRepository{})
	finder := NewIncrementStrategyFinder(store)

	bv := strategy.BaseVersion{ShouldIncrement: false}
	ec := defaultEC()
	ec.BranchIncrement = semver.IncrementStrategyMinor

	field := finder.branchDefault(bv, ec)
	require.Equal(t, semver.VersionFieldNone, field)
}

func TestBranchDefault_InheritFallsToPatch(t *testing.T) {
	store := git.NewRepositoryStore(&git.MockRepository{})
	finder := NewIncrementStrategyFinder(store)

	bv := strategy.BaseVersion{ShouldIncrement: true}
	ec := defaultEC()
	ec.BranchIncrement = semver.IncrementStrategyInherit

	field := finder.branchDefault(bv, ec)
	require.Equal(t, semver.VersionFieldPatch, field)
}

func TestBranchDefault_ReturnsConfigured(t *testing.T) {
	store := git.NewRepositoryStore(&git.MockRepository{})
	finder := NewIncrementStrategyFinder(store)

	bv := strategy.BaseVersion{ShouldIncrement: true}
	ec := defaultEC()
	ec.BranchIncrement = semver.IncrementStrategyMajor

	field := finder.branchDefault(bv, ec)
	require.Equal(t, semver.VersionFieldMajor, field)
}

// ---------------------------------------------------------------------------
// AnalyzeCommitIncrement tests
// ---------------------------------------------------------------------------

func TestAnalyzeCommitIncrement_ConventionalCommit(t *testing.T) {
	store := git.NewRepositoryStore(&git.MockRepository{})
	finder := NewIncrementStrategyFinder(store)

	c := newCommit("aaa0000000000000000000000000000000000000", "feat: login")
	ec := defaultEC()
	ec.CommitMessageConvention = semver.CommitMessageConventionConventionalCommits

	field := finder.AnalyzeCommitIncrement(c, ec)
	require.Equal(t, semver.VersionFieldMinor, field)
}

func TestAnalyzeCommitIncrement_BumpDirective(t *testing.T) {
	store := git.NewRepositoryStore(&git.MockRepository{})
	finder := NewIncrementStrategyFinder(store)

	c := newCommit("aaa0000000000000000000000000000000000000", "change +semver: major")
	ec := defaultEC()
	ec.CommitMessageConvention = semver.CommitMessageConventionBumpDirective

	field := finder.AnalyzeCommitIncrement(c, ec)
	require.Equal(t, semver.VersionFieldMajor, field)
}

func TestAnalyzeCommitIncrement_MergeMessageOnly_NonMerge(t *testing.T) {
	store := git.NewRepositoryStore(&git.MockRepository{})
	finder := NewIncrementStrategyFinder(store)

	c := git.Commit{
		Sha: "aaa0000000000000000000000000000000000000", Parents: []string{"p1"},
		Message: "feat: should be skipped",
	}
	ec := defaultEC()
	ec.CommitMessageIncrementing = semver.CommitMessageIncrementMergeMessageOnly
	ec.CommitMessageConvention = semver.CommitMessageConventionConventionalCommits

	field := finder.AnalyzeCommitIncrement(c, ec)
	require.Equal(t, semver.VersionFieldNone, field, "non-merge should return None in MergeMessageOnly mode")
}

func TestAnalyzeCommitIncrement_Both_HighestWins(t *testing.T) {
	store := git.NewRepositoryStore(&git.MockRepository{})
	finder := NewIncrementStrategyFinder(store)

	c := newCommit("aaa0000000000000000000000000000000000000", "fix: bug +semver: minor")
	ec := defaultEC()
	ec.CommitMessageConvention = semver.CommitMessageConventionBoth

	field := finder.AnalyzeCommitIncrement(c, ec)
	// fix: → Patch, +semver: minor → Minor. Both mode takes highest = Minor.
	require.Equal(t, semver.VersionFieldMinor, field)
}

// ---------------------------------------------------------------------------
// IncrementExplanation nil-safety tests
// ---------------------------------------------------------------------------

func TestIncrementExplanation_NilSafe(t *testing.T) {
	var exp *IncrementExplanation
	// Should not panic.
	exp.Add("test")
	exp.Addf("test %s", "value")
	require.Nil(t, exp)
}

func TestIncrementExplanation_AddSteps(t *testing.T) {
	exp := &IncrementExplanation{}
	exp.Add("step 1")
	exp.Addf("step %d", 2)
	require.Len(t, exp.Steps, 2)
	require.Equal(t, "step 1", exp.Steps[0])
	require.Equal(t, "step 2", exp.Steps[1])
}

// ---------------------------------------------------------------------------
// Additional Conventional Commits edge cases
// ---------------------------------------------------------------------------

func TestConventionalCommit_FixWithScope(t *testing.T) {
	require.Equal(t, semver.VersionFieldPatch, analyzeConventionalCommit("fix(core): null pointer"))
}

func TestConventionalCommit_BreakingWithScope(t *testing.T) {
	require.Equal(t, semver.VersionFieldMajor, analyzeConventionalCommit("refactor(api)!: redesign"))
}

func TestConventionalCommit_Docs(t *testing.T) {
	require.Equal(t, semver.VersionFieldNone, analyzeConventionalCommit("docs: add README"))
}

func TestConventionalCommit_Refactor(t *testing.T) {
	require.Equal(t, semver.VersionFieldNone, analyzeConventionalCommit("refactor: clean up code"))
}

func TestConventionalCommit_Test(t *testing.T) {
	require.Equal(t, semver.VersionFieldNone, analyzeConventionalCommit("test: add unit tests"))
}

func TestConventionalCommit_CaseInsensitive(t *testing.T) {
	require.Equal(t, semver.VersionFieldMinor, analyzeConventionalCommit("Feat: uppercase feat"))
}

func TestConventionalCommit_MultilineBody(t *testing.T) {
	msg := "feat: add auth\n\nThis adds authentication support.\n\nSigned-off-by: dev"
	require.Equal(t, semver.VersionFieldMinor, analyzeConventionalCommit(msg))
}

func TestBumpDirective_BreakingAlias(t *testing.T) {
	ec := defaultEC()
	require.Equal(t, semver.VersionFieldMajor, analyzeBumpDirective("refactor +semver: breaking", ec))
}

func TestBumpDirective_MinorAlias(t *testing.T) {
	ec := defaultEC()
	require.Equal(t, semver.VersionFieldMinor, analyzeBumpDirective("add +semver: minor", ec))
}

func TestBumpDirective_PatchAlias(t *testing.T) {
	ec := defaultEC()
	require.Equal(t, semver.VersionFieldPatch, analyzeBumpDirective("fix +semver: patch", ec))
}
