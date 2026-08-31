package calculator

import (
	"fmt"
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

func TestNextVersion_TaggedCommit(t *testing.T) {
	tip := newCommit("aaa0000000000000000000000000000000000000", "tagged")

	store := git.NewRepositoryStore(&git.MockRepository{})
	calc := NewNextVersionCalculator(store, nil)

	ctx := &context.GitVersionContext{
		CurrentBranch:              git.Branch{Name: git.NewReferenceName("refs/heads/main")},
		CurrentCommit:              tip,
		IsCurrentCommitTagged:      true,
		CurrentCommitTaggedVersion: semver.SemanticVersion{Major: 2, Minor: 1},
	}
	ec := defaultEC()
	ec.IsMainline = true

	result, err := calc.Calculate(ctx, ec, false)
	require.NoError(t, err)
	require.Equal(t, int64(2), result.Version.Major)
	require.Equal(t, int64(1), result.Version.Minor)
	require.Equal(t, "main", result.BranchName)
}

func TestNextVersion_StandardMode(t *testing.T) {
	tip := newCommit("aaa0000000000000000000000000000000000000", "feat: add login")
	source := newCommit("bbb0000000000000000000000000000000000000", "initial")

	mock := &git.MockRepository{
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip, source}, nil
		},
		TagsFunc: func(filters ...git.PathFilter) ([]git.Tag, error) { return nil, nil },
	}
	store := git.NewRepositoryStore(mock)

	vs := &stubStrategy{
		name: "test",
		versions: []strategy.BaseVersion{
			{
				Source:            "tag",
				SemanticVersion:   semver.SemanticVersion{Major: 1},
				ShouldIncrement:   true,
				BaseVersionSource: &source,
			},
		},
	}

	calc := NewNextVersionCalculator(store, []strategy.VersionStrategy{vs})

	ctx := &context.GitVersionContext{
		CurrentBranch: git.Branch{
			Name: git.NewReferenceName("refs/heads/main"),
			Tip:  &tip,
		},
		CurrentCommit: tip,
	}
	ec := defaultEC()
	ec.IsMainline = true
	ec.Tag = ""
	ec.CommitMessageConvention = semver.CommitMessageConventionConventionalCommits

	result, err := calc.Calculate(ctx, ec, false)
	require.NoError(t, err)
	// feat: → Minor increment: 1.0.0 → 1.1.0
	require.Equal(t, int64(1), result.Version.Major)
	require.Equal(t, int64(1), result.Version.Minor)
	require.Equal(t, int64(0), result.Version.Patch)
	require.Equal(t, int64(1), result.CommitsSince)
}

func TestNextVersion_MainlineMode(t *testing.T) {
	tip := newCommit("aaa0000000000000000000000000000000000000", "feat: add feature")
	mid := newCommit("bbb0000000000000000000000000000000000000", "fix: bug")
	source := newCommit("ccc0000000000000000000000000000000000000", "v1.0.0")

	mock := &git.MockRepository{
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip, mid, source}, nil
		},
		TagsFunc: func(filters ...git.PathFilter) ([]git.Tag, error) { return nil, nil },
	}
	store := git.NewRepositoryStore(mock)

	vs := &stubStrategy{
		name: "test",
		versions: []strategy.BaseVersion{
			{
				Source:            "tag",
				SemanticVersion:   semver.SemanticVersion{Major: 1},
				ShouldIncrement:   true,
				BaseVersionSource: &source,
			},
		},
	}

	calc := NewNextVersionCalculator(store, []strategy.VersionStrategy{vs})

	ctx := &context.GitVersionContext{
		CurrentBranch: git.Branch{
			Name: git.NewReferenceName("refs/heads/main"),
			Tip:  &tip,
		},
		CurrentCommit: tip,
	}
	ec := defaultEC()
	ec.BranchMode = semver.VersioningModeMainline
	ec.IsMainline = true
	ec.Tag = ""
	ec.CommitMessageConvention = semver.CommitMessageConventionConventionalCommits

	result, err := calc.Calculate(ctx, ec, false)
	require.NoError(t, err)
	// feat: → Minor is highest; 1.0.0 → 1.1.0
	require.Equal(t, int64(1), result.Version.Major)
	require.Equal(t, int64(1), result.Version.Minor)
}

func TestNextVersion_PreReleaseTag(t *testing.T) {
	tip := newCommit("aaa0000000000000000000000000000000000000", "feat: add login")
	source := newCommit("bbb0000000000000000000000000000000000000", "initial")

	mock := &git.MockRepository{
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip, source}, nil
		},
		TagsFunc: func(filters ...git.PathFilter) ([]git.Tag, error) { return nil, nil },
	}
	store := git.NewRepositoryStore(mock)

	vs := &stubStrategy{
		name: "test",
		versions: []strategy.BaseVersion{
			{
				Source:            "tag",
				SemanticVersion:   semver.SemanticVersion{Major: 1},
				ShouldIncrement:   true,
				BaseVersionSource: &source,
			},
		},
	}

	calc := NewNextVersionCalculator(store, []strategy.VersionStrategy{vs})

	ctx := &context.GitVersionContext{
		CurrentBranch: git.Branch{
			Name: git.NewReferenceName("refs/heads/feature/auth"),
			Tip:  &tip,
		},
		CurrentCommit: tip,
	}
	ec := defaultEC()
	ec.Tag = "{BranchName}"
	ec.CommitMessageConvention = semver.CommitMessageConventionConventionalCommits

	result, err := calc.Calculate(ctx, ec, false)
	require.NoError(t, err)
	require.Equal(t, "auth", result.Version.PreReleaseTag.Name)
	require.NotNil(t, result.Version.PreReleaseTag.Number)
	require.Equal(t, int64(1), *result.Version.PreReleaseTag.Number)
}

func TestNextVersion_PreReleaseIncrement(t *testing.T) {
	tip := newCommit("aaa0000000000000000000000000000000000000", "feat: more work")
	source := newCommit("bbb0000000000000000000000000000000000000", "initial")

	// Existing tag: 1.1.0-dev.3
	num := int64(3)
	existingTag := git.Tag{
		Name:      git.NewReferenceName("refs/tags/v1.1.0-dev.3"),
		TargetSha: "ttt0000000000000000000000000000000000000",
	}
	tagCommit := newCommit("ttt0000000000000000000000000000000000000", "tagged")

	mock := &git.MockRepository{
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip, source}, nil
		},
		TagsFunc: func(filters ...git.PathFilter) ([]git.Tag, error) {
			return []git.Tag{existingTag}, nil
		},
		PeelTagToCommitFunc: func(tag git.Tag) (string, error) { return tag.TargetSha, nil },
		CommitFromShaFunc: func(sha string) (git.Commit, error) {
			return tagCommit, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	vs := &stubStrategy{
		name: "test",
		versions: []strategy.BaseVersion{
			{
				Source:            "tag",
				SemanticVersion:   semver.SemanticVersion{Major: 1},
				ShouldIncrement:   true,
				BaseVersionSource: &source,
			},
		},
	}

	calc := NewNextVersionCalculator(store, []strategy.VersionStrategy{vs})

	ctx := &context.GitVersionContext{
		CurrentBranch: git.Branch{
			Name: git.NewReferenceName("refs/heads/develop"),
			Tip:  &tip,
		},
		CurrentCommit: tip,
	}
	ec := defaultEC()
	ec.Tag = "dev"
	ec.CommitMessageConvention = semver.CommitMessageConventionConventionalCommits

	result, err := calc.Calculate(ctx, ec, false)
	require.NoError(t, err)
	// feat: → Minor: 1.0.0 → 1.1.0-dev.N
	require.Equal(t, int64(1), result.Version.Major)
	require.Equal(t, int64(1), result.Version.Minor)
	require.Equal(t, "dev", result.Version.PreReleaseTag.Name)
	require.NotNil(t, result.Version.PreReleaseTag.Number)
	// Existing tag 1.1.0-dev.3 → next should be 4.
	require.Equal(t, num+1, *result.Version.PreReleaseTag.Number)
}

func TestNextVersion_BuildMetadata(t *testing.T) {
	tip := newCommit("aaa0000000000000000000000000000000000000", "fix: patch")
	tip.When = time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	source := newCommit("bbb0000000000000000000000000000000000000", "initial")

	mock := &git.MockRepository{
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip, source}, nil
		},
		TagsFunc: func(filters ...git.PathFilter) ([]git.Tag, error) { return nil, nil },
	}
	store := git.NewRepositoryStore(mock)

	vs := &stubStrategy{
		name: "test",
		versions: []strategy.BaseVersion{
			{
				Source:            "tag",
				SemanticVersion:   semver.SemanticVersion{Major: 1},
				ShouldIncrement:   true,
				BaseVersionSource: &source,
			},
		},
	}

	calc := NewNextVersionCalculator(store, []strategy.VersionStrategy{vs})

	ctx := &context.GitVersionContext{
		CurrentBranch: git.Branch{
			Name: git.NewReferenceName("refs/heads/main"),
			Tip:  &tip,
		},
		CurrentCommit:              tip,
		NumberOfUncommittedChanges: 2,
	}
	ec := defaultEC()
	ec.IsMainline = true
	ec.Tag = ""
	ec.CommitMessageConvention = semver.CommitMessageConventionConventionalCommits

	result, err := calc.Calculate(ctx, ec, false)
	require.NoError(t, err)
	require.NotNil(t, result.Version.BuildMetaData.CommitsSinceTag)
	require.Equal(t, int64(1), *result.Version.BuildMetaData.CommitsSinceTag)
	require.Equal(t, "main", result.Version.BuildMetaData.Branch)
	require.Equal(t, tip.Sha, result.Version.BuildMetaData.Sha)
	require.Equal(t, source.Sha, result.Version.BuildMetaData.VersionSourceSha)
	require.Equal(t, int64(2), result.Version.BuildMetaData.UncommittedChanges)
}

func TestNextVersion_ReleaseBranchNoPreRelease(t *testing.T) {
	tip := newCommit("aaa0000000000000000000000000000000000000", "fix: patch")
	source := newCommit("bbb0000000000000000000000000000000000000", "initial")

	mock := &git.MockRepository{
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip, source}, nil
		},
		TagsFunc: func(filters ...git.PathFilter) ([]git.Tag, error) { return nil, nil },
	}
	store := git.NewRepositoryStore(mock)

	vs := &stubStrategy{
		name: "test",
		versions: []strategy.BaseVersion{
			{
				Source:            "branch",
				SemanticVersion:   semver.SemanticVersion{Major: 1, Minor: 2},
				ShouldIncrement:   true,
				BaseVersionSource: &source,
			},
		},
	}

	calc := NewNextVersionCalculator(store, []strategy.VersionStrategy{vs})

	ctx := &context.GitVersionContext{
		CurrentBranch: git.Branch{
			Name: git.NewReferenceName("refs/heads/release/1.2.0"),
			Tip:  &tip,
		},
		CurrentCommit: tip,
	}
	ec := defaultEC()
	ec.IsReleaseBranch = true
	ec.Tag = "beta"
	ec.CommitMessageConvention = semver.CommitMessageConventionConventionalCommits

	result, err := calc.Calculate(ctx, ec, false)
	require.NoError(t, err)
	// Release branches don't get pre-release tags.
	require.False(t, result.Version.PreReleaseTag.HasTag())
}

func TestNextVersion_MainlinePreReleaseTag(t *testing.T) {
	tip := newCommit("aaa0000000000000000000000000000000000000", "feat: add feature")
	source := newCommit("bbb0000000000000000000000000000000000000", "initial")

	logFunc := func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
		return []git.Commit{tip, source}, nil
	}
	mock := &git.MockRepository{
		CommitLogFunc:         logFunc,
		MainlineCommitLogFunc: logFunc,
		TagsFunc:              func(filters ...git.PathFilter) ([]git.Tag, error) { return nil, nil },
	}
	store := git.NewRepositoryStore(mock)

	vs := &stubStrategy{
		name: "test",
		versions: []strategy.BaseVersion{
			{
				Source:            "tag",
				SemanticVersion:   semver.SemanticVersion{Major: 1},
				ShouldIncrement:   true,
				BaseVersionSource: &source,
			},
		},
	}

	calc := NewNextVersionCalculator(store, []strategy.VersionStrategy{vs})

	ctx := &context.GitVersionContext{
		CurrentBranch: git.Branch{
			Name: git.NewReferenceName("refs/heads/feature/login"),
			Tip:  &tip,
		},
		CurrentCommit: tip,
	}
	ec := defaultEC()
	ec.BranchMode = semver.VersioningModeMainline
	ec.Tag = "{BranchName}"
	ec.CommitMessageConvention = semver.CommitMessageConventionConventionalCommits

	result, err := calc.Calculate(ctx, ec, false)
	require.NoError(t, err)
	// Mainline mode with tag: pre-release number = commits since.
	require.Equal(t, "login", result.Version.PreReleaseTag.Name)
	require.NotNil(t, result.Version.PreReleaseTag.Number)
	require.True(t, *result.Version.PreReleaseTag.Number >= 1)
}

func TestNextVersion_ExplainMode(t *testing.T) {
	tip := newCommit("aaa0000000000000000000000000000000000000", "feat: add login")
	source := newCommit("bbb0000000000000000000000000000000000000", "initial")

	mock := &git.MockRepository{
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip, source}, nil
		},
		TagsFunc: func(filters ...git.PathFilter) ([]git.Tag, error) { return nil, nil },
	}
	store := git.NewRepositoryStore(mock)

	vs := &stubStrategy{
		name: "test",
		versions: []strategy.BaseVersion{
			{
				Source:            "tag",
				SemanticVersion:   semver.SemanticVersion{Major: 1},
				ShouldIncrement:   true,
				BaseVersionSource: &source,
			},
		},
	}

	calc := NewNextVersionCalculator(store, []strategy.VersionStrategy{vs})

	ctx := &context.GitVersionContext{
		CurrentBranch: git.Branch{
			Name: git.NewReferenceName("refs/heads/feature/auth"),
			Tip:  &tip,
		},
		CurrentCommit: tip,
	}
	ec := defaultEC()
	ec.Tag = "{BranchName}"
	ec.CommitMessageConvention = semver.CommitMessageConventionConventionalCommits

	result, err := calc.Calculate(ctx, ec, true)
	require.NoError(t, err)
	// Explain mode should populate IncrementExplanation and PreReleaseSteps.
	require.NotNil(t, result.IncrementExplanation)
	require.NotEmpty(t, result.IncrementExplanation.Steps)
	require.NotEmpty(t, result.PreReleaseSteps)
}

func TestNextVersion_BranchNameOverride(t *testing.T) {
	tip := newCommit("aaa0000000000000000000000000000000000000", "fix: patch")
	source := newCommit("bbb0000000000000000000000000000000000000", "initial")

	mock := &git.MockRepository{
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip, source}, nil
		},
		TagsFunc: func(filters ...git.PathFilter) ([]git.Tag, error) { return nil, nil },
	}
	store := git.NewRepositoryStore(mock)

	vs := &stubStrategy{
		name: "test",
		versions: []strategy.BaseVersion{
			{
				Source:             "tag",
				SemanticVersion:    semver.SemanticVersion{Major: 1},
				ShouldIncrement:    true,
				BaseVersionSource:  &source,
				BranchNameOverride: "custom-name",
			},
		},
	}

	calc := NewNextVersionCalculator(store, []strategy.VersionStrategy{vs})

	ctx := &context.GitVersionContext{
		CurrentBranch: git.Branch{
			Name: git.NewReferenceName("refs/heads/main"),
			Tip:  &tip,
		},
		CurrentCommit: tip,
	}
	ec := defaultEC()
	ec.IsMainline = true
	ec.Tag = ""

	result, err := calc.Calculate(ctx, ec, false)
	require.NoError(t, err)
	require.Equal(t, "custom-name", result.BranchName)
}

// newCountCalc builds a calculator whose commit log varies with the "from"
// argument, so tests can tell apart a scan anchored at the base version source
// from one anchored at the zero commit.
func newCountCalc(t *testing.T, fromSource, fromZero []git.Commit) *NextVersionCalculator {
	t.Helper()
	logFunc := func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
		if from == "" {
			return fromZero, nil
		}
		return fromSource, nil
	}
	mock := &git.MockRepository{CommitLogFunc: logFunc, MainlineCommitLogFunc: logFunc}
	store := git.NewRepositoryStore(mock)
	return NewNextVersionCalculator(store, nil)
}

// TestCountCommitsSince_AnchorsAtBaseVersionSource pins that the scan starts at
// the base version source, not at the zero commit.
func TestCountCommitsSince_AnchorsAtBaseVersionSource(t *testing.T) {
	tip := newCommit("aaa0000000000000000000000000000000000000", "feat: two")
	source := newCommit("bbb0000000000000000000000000000000000000", "v1.0.0")
	extra := newCommit("ddd0000000000000000000000000000000000000", "ancient")

	calc := newCountCalc(t,
		[]git.Commit{tip, source},
		[]git.Commit{tip, source, extra, extra, extra},
	)

	ctx := &context.GitVersionContext{CurrentCommit: tip}
	bv := strategy.BaseVersion{BaseVersionSource: &source}

	require.Equal(t, int64(1), calc.countCommitsSince(ctx, bv, defaultEC()))
}

// TestCountCommitsSince_SourceNotInLog pins that the count is only reduced when
// the base version source actually appears in the log.
func TestCountCommitsSince_SourceNotInLog(t *testing.T) {
	tip := newCommit("aaa0000000000000000000000000000000000000", "feat: two")
	mid := newCommit("ccc0000000000000000000000000000000000000", "feat: one")
	absent := newCommit("9990000000000000000000000000000000000000", "v1.0.0")

	calc := newCountCalc(t, []git.Commit{tip, mid}, nil)

	ctx := &context.GitVersionContext{CurrentCommit: tip}
	bv := strategy.BaseVersion{BaseVersionSource: &absent}

	require.Equal(t, int64(2), calc.countCommitsSince(ctx, bv, defaultEC()))
}

// TestCountCommitsSince_NoBaseVersionSource pins the nil-source path.
func TestCountCommitsSince_NoBaseVersionSource(t *testing.T) {
	tip := newCommit("aaa0000000000000000000000000000000000000", "feat: one")

	calc := newCountCalc(t, nil, []git.Commit{tip, tip, tip})

	ctx := &context.GitVersionContext{CurrentCommit: tip}
	require.Equal(t, int64(3), calc.countCommitsSince(ctx, strategy.BaseVersion{}, defaultEC()))
}

// preReleaseCalc builds a calculator whose tag list contains the given tag
// names, so pre-release numbering can be exercised.
func preReleaseCalc(t *testing.T, tagNames ...string) *NextVersionCalculator {
	t.Helper()
	tags := make([]git.Tag, 0, len(tagNames))
	for i, name := range tagNames {
		sha := fmt.Sprintf("%040d", i+1)
		tags = append(tags, git.Tag{Name: git.NewReferenceName("refs/tags/" + name), TargetSha: sha})
	}
	mock := &git.MockRepository{
		TagsFunc:            func(...git.PathFilter) ([]git.Tag, error) { return tags, nil },
		PeelTagToCommitFunc: func(tag git.Tag) (string, error) { return tag.TargetSha, nil },
		CommitFromShaFunc:   func(sha string) (git.Commit, error) { return git.Commit{Sha: sha}, nil },
	}
	return NewNextVersionCalculator(git.NewRepositoryStore(mock), nil)
}

func preReleaseEC() config.EffectiveConfiguration {
	ec := defaultEC()
	ec.Tag = "alpha"
	ec.IsMainline = false
	ec.IsReleaseBranch = false
	return ec
}

// TestUpdatePreReleaseTag_FirstTagStartsAtOne pins the no-existing-tag path,
// including the explain wording that distinguishes it.
func TestUpdatePreReleaseTag_FirstTagStartsAtOne(t *testing.T) {
	calc := preReleaseCalc(t)
	ctx := &context.GitVersionContext{}
	ver := semver.SemanticVersion{Major: 1, Minor: 3, Patch: 0}

	got, steps := calc.updatePreReleaseTag(ver, ctx, preReleaseEC(), "feature/x", 0, true)

	require.NotNil(t, got.PreReleaseTag.Number)
	require.Equal(t, int64(1), *got.PreReleaseTag.Number)
	require.Contains(t, strings.Join(steps, "\n"), "no existing tag")
}

// TestUpdatePreReleaseTag_ExistingTagIncrements pins the >= boundary: an
// existing tag numbered exactly 1 must push the next number to 2.
func TestUpdatePreReleaseTag_ExistingTagIncrements(t *testing.T) {
	calc := preReleaseCalc(t, "v1.3.0-alpha.1")
	ctx := &context.GitVersionContext{}
	ver := semver.SemanticVersion{Major: 1, Minor: 3, Patch: 0}

	got, steps := calc.updatePreReleaseTag(ver, ctx, preReleaseEC(), "feature/x", 0, true)

	require.NotNil(t, got.PreReleaseTag.Number)
	require.Equal(t, int64(2), *got.PreReleaseTag.Number)
	require.Contains(t, strings.Join(steps, "\n"), "existing tag")
	require.NotContains(t, strings.Join(steps, "\n"), "no existing tag")
}
