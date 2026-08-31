package calculator

import (
	"strings"
	"testing"
	"time"

	"github.com/MyCarrier-DevOps/go-gitsemver/internal/config"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/context"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/git"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/semver"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/strategy"

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
		MajorVersionBumpMessage:   `(\+semver:\s?(breaking|major)|bump major:)`,
		MinorVersionBumpMessage:   `(\+semver:\s?(feature|minor)|bump minor:)`,
		PatchVersionBumpMessage:   `(\+semver:\s?(fix|patch)|bump patch:)`,
		NoBumpMessage:             `(\+semver:\s?(none|skip)|bump (none|skip):)`,
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

func TestConventionalCommit_Types(t *testing.T) {
	tests := []struct {
		name string
		msg  string
		want semver.VersionField
	}{
		{"feat", "feat: add login", semver.VersionFieldMinor},
		{"fix", "fix: null pointer", semver.VersionFieldPatch},
		{"perf", "perf: cache tag lookups", semver.VersionFieldPatch},
		{"chore", "chore: update deps", semver.VersionFieldPatch},
		{"perf with scope", "perf(git): reuse the commit iterator", semver.VersionFieldPatch},
		{"chore with scope", "chore(deps): bump go-git", semver.VersionFieldPatch},
		{"build", "build: switch to CGO_ENABLED=0", semver.VersionFieldNone},
		{"ci", "ci: pin the runner image", semver.VersionFieldNone},
		{"docs", "docs: update readme", semver.VersionFieldNone},
		{"refactor", "refactor: extract the finder", semver.VersionFieldNone},
		{"revert", "revert: undo the auth change", semver.VersionFieldNone},
		{"style", "style: gofmt", semver.VersionFieldNone},
		{"test", "test: cover the parser", semver.VersionFieldNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, analyzeConventionalCommit(tt.msg))
		})
	}
}

func TestConventionalCommit_BreakingOnPatchTypes(t *testing.T) {
	require.Equal(t, semver.VersionFieldMajor, analyzeConventionalCommit("chore!: drop go 1.21"))
	require.Equal(t, semver.VersionFieldMajor, analyzeConventionalCommit("perf!: change cache key format"))
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
// AnalyzeCommitBump tests
// ---------------------------------------------------------------------------

func TestAnalyzeCommitBump_ConventionalCommit(t *testing.T) {
	store := git.NewRepositoryStore(&git.MockRepository{})
	finder := NewIncrementStrategyFinder(store)

	c := newCommit("aaa0000000000000000000000000000000000000", "feat: login")
	ec := defaultEC()
	ec.CommitMessageConvention = semver.CommitMessageConventionConventionalCommits

	field := finder.AnalyzeCommitBump(c, ec).Field
	require.Equal(t, semver.VersionFieldMinor, field)
}

func TestAnalyzeCommitBump_BumpDirective(t *testing.T) {
	store := git.NewRepositoryStore(&git.MockRepository{})
	finder := NewIncrementStrategyFinder(store)

	c := newCommit("aaa0000000000000000000000000000000000000", "change +semver: major")
	ec := defaultEC()
	ec.CommitMessageConvention = semver.CommitMessageConventionBumpDirective

	field := finder.AnalyzeCommitBump(c, ec).Field
	require.Equal(t, semver.VersionFieldMajor, field)
}

func TestAnalyzeCommitBump_MergeMessageOnly_NonMerge(t *testing.T) {
	store := git.NewRepositoryStore(&git.MockRepository{})
	finder := NewIncrementStrategyFinder(store)

	c := git.Commit{
		Sha: "aaa0000000000000000000000000000000000000", Parents: []string{"p1"},
		Message: "feat: should be skipped",
	}
	ec := defaultEC()
	ec.CommitMessageIncrementing = semver.CommitMessageIncrementMergeMessageOnly
	ec.CommitMessageConvention = semver.CommitMessageConventionConventionalCommits

	field := finder.AnalyzeCommitBump(c, ec).Field
	require.Equal(t, semver.VersionFieldNone, field, "non-merge should return None in MergeMessageOnly mode")
}

func TestAnalyzeCommitBump_Both_HighestWins(t *testing.T) {
	store := git.NewRepositoryStore(&git.MockRepository{})
	finder := NewIncrementStrategyFinder(store)

	c := newCommit("aaa0000000000000000000000000000000000000", "fix: bug +semver: minor")
	ec := defaultEC()
	ec.CommitMessageConvention = semver.CommitMessageConventionBoth

	field := finder.AnalyzeCommitBump(c, ec).Field
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

func TestBumpDirective_BumpMajorColon(t *testing.T) {
	ec := defaultEC()
	require.Equal(t, semver.VersionFieldMajor, analyzeBumpDirective("bump major: redesign API", ec))
}

func TestBumpDirective_BumpMinorColon(t *testing.T) {
	ec := defaultEC()
	require.Equal(t, semver.VersionFieldMinor, analyzeBumpDirective("bump minor: add new report type", ec))
}

func TestBumpDirective_BumpPatchColon(t *testing.T) {
	ec := defaultEC()
	require.Equal(t, semver.VersionFieldPatch, analyzeBumpDirective("bump patch: fix typo in output", ec))
}

func TestIsNoBump_Patterns(t *testing.T) {
	ec := defaultEC()
	tests := []struct {
		name string
		msg  string
		want bool
	}{
		{"+semver none", "update deps +semver: none", true},
		{"+semver skip", "update deps +semver: skip", true},
		{"+semver no space", "update deps +semver:none", true},
		{"bump none prefix", "bump none: update deps", true},
		{"bump skip prefix", "bump skip: update deps", true},
		{"in footer", "chore: update deps\n\n+semver: none", true},
		{"plain message", "chore: update deps", false},
		{"patch directive", "fix it +semver: patch", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, isNoBump(tt.msg, ec))
		})
	}
}

func TestAnalyzeCommitBump_NoBumpOverridesConventionalCommit(t *testing.T) {
	finder := NewIncrementStrategyFinder(nil)
	ec := defaultEC()

	got := finder.AnalyzeCommitBump(newCommit("aaa", "feat: add login\n\n+semver: none"), ec)
	require.Equal(t, semver.VersionFieldNone, got.Field)
	require.True(t, got.Suppressed, "explicit no-bump directive must beat the conventional-commit type")
}

func TestAnalyzeCommitBump_NoBumpIgnoredInConventionalCommitsMode(t *testing.T) {
	finder := NewIncrementStrategyFinder(nil)
	ec := defaultEC()
	ec.CommitMessageConvention = semver.CommitMessageConventionConventionalCommits

	got := finder.AnalyzeCommitBump(newCommit("aaa", "feat: add login\n\n+semver: none"), ec)
	require.Equal(t, semver.VersionFieldMinor, got.Field)
	require.False(t, got.Suppressed, "bump directives are not honoured in ConventionalCommits-only mode")
}

func TestAnalyzeCommitBump_ChoreIsPatch(t *testing.T) {
	finder := NewIncrementStrategyFinder(nil)
	got := finder.AnalyzeCommitBump(newCommit("aaa", "chore: update deps"), defaultEC())
	require.Equal(t, semver.VersionFieldPatch, got.Field)
	require.False(t, got.Suppressed)
}

func TestDetermineIncrement_NoBumpSuppressesBranchDefault(t *testing.T) {
	tip := newCommit("aaa0000000000000000000000000000000000000", "chore: update deps +semver: none")
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
	ec.BranchIncrement = semver.IncrementStrategyPatch

	finder := NewIncrementStrategyFinder(store)
	field, err := finder.DetermineIncrementedField(ctx, bv, ec)
	require.NoError(t, err)
	require.Equal(t, semver.VersionFieldNone, field, "no-bump must suppress the branch default increment")
}

func TestDetermineIncrement_NoBumpDoesNotSuppressRealBump(t *testing.T) {
	tip := newCommit("aaa0000000000000000000000000000000000000", "feat: add login")
	mid := newCommit("ccc0000000000000000000000000000000000000", "chore: deps +semver: none")
	source := newCommit("bbb0000000000000000000000000000000000000", "initial")

	mock := &git.MockRepository{
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip, mid, source}, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{CurrentCommit: tip}
	bv := strategy.BaseVersion{
		SemanticVersion:   semver.SemanticVersion{Major: 1},
		ShouldIncrement:   true,
		BaseVersionSource: &source,
	}

	finder := NewIncrementStrategyFinder(store)
	field, err := finder.DetermineIncrementedField(ctx, bv, defaultEC())
	require.NoError(t, err)
	require.Equal(t, semver.VersionFieldMinor, field, "a suppressed commit must not veto a real feat: bump")
}

func TestFirstLine(t *testing.T) {
	tests := []struct {
		name string
		msg  string
		want string
	}{
		{"single line", "feat: add login", "feat: add login"},
		{"multi line", "feat: add login\n\nbody text", "feat: add login"},
		{"leading newline", "\nfeat: add login", ""},
		{"empty", "", ""},
		{"only newline", "\n", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, firstLine(tt.msg))
		})
	}
}

func TestDetermineIncrementExplained_ReportsOnlyBumpingCommits(t *testing.T) {
	bumping := newCommit("aaa0000000000000000000000000000000000000", "feat: add login")
	quiet := newCommit("ccc0000000000000000000000000000000000000", "docs: update readme")
	source := newCommit("bbb0000000000000000000000000000000000000", "initial")

	mock := &git.MockRepository{
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{bumping, quiet, source}, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{CurrentCommit: bumping}
	bv := strategy.BaseVersion{
		SemanticVersion:   semver.SemanticVersion{Major: 1},
		ShouldIncrement:   true,
		BaseVersionSource: &source,
	}

	finder := NewIncrementStrategyFinder(store)
	result, err := finder.DetermineIncrementedFieldExplained(ctx, bv, defaultEC(), true)
	require.NoError(t, err)
	require.NotNil(t, result.Explanation)

	steps := strings.Join(result.Explanation.Steps, "\n")
	require.Contains(t, steps, "feat: add login", "a commit that bumps must be reported")
	require.NotContains(t, steps, "docs: update readme", "a commit that does not bump must not be reported")
}

// TestConventionName_BothConventionsAtSameLevel pins the cc >= bd boundary:
// when a commit carries a conventional-commit type AND a bump directive that
// request the same field, the conventional-commit label wins.
func TestConventionName_BothConventionsAtSameLevel(t *testing.T) {
	ec := defaultEC()
	ec.CommitMessageConvention = semver.CommitMessageConventionBoth

	require.Equal(t, "Conventional Commits", conventionName("fix: a bug +semver: patch", ec))
	require.Equal(t, "Conventional Commits", conventionName("feat: a thing +semver: minor", ec))
	require.Equal(t, "Bump Directive", conventionName("fix: a bug +semver: major", ec))
	require.Equal(t, "Bump Directive", conventionName("docs: readme +semver: patch", ec))
}
