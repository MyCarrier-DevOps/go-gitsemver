package calculator

import (
	"testing"
	"time"

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
