package strategy

import (
	"testing"

	"github.com/MyCarrier-DevOps/go-gitsemver/internal/config"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/context"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/git"

	"github.com/stretchr/testify/require"
)

func releaseBranchConfig(t *testing.T) *config.Config {
	t.Helper()
	cfg, err := config.NewBuilder().Build()
	require.NoError(t, err)
	return cfg
}

func TestBranchName_ReleaseBranch(t *testing.T) {
	branchPoint := newTestCommit("aaa0000000000000000000000000000000000000", "fork point")
	tip := newTestCommit("bbb0000000000000000000000000000000000000", "tip")
	branch := git.Branch{
		Name: git.NewReferenceName("refs/heads/release/1.2.0"),
		Tip:  &tip,
	}
	cfg := releaseBranchConfig(t)

	mock := &git.MockRepository{
		BranchesFunc: func(filters ...git.PathFilter) ([]git.Branch, error) {
			return []git.Branch{branch}, nil
		},
		FindMergeBaseFunc: func(sha1, sha2 string) (string, error) {
			return branchPoint.Sha, nil
		},
		CommitFromShaFunc: func(sha string) (git.Commit, error) {
			if sha == branchPoint.Sha {
				return branchPoint, nil
			}
			return tip, nil
		},
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip, branchPoint}, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{
		CurrentBranch:     branch,
		CurrentCommit:     tip,
		FullConfiguration: cfg,
	}
	ec := config.EffectiveConfiguration{TagPrefix: "[vV]", IsReleaseBranch: true}

	s := NewVersionInBranchNameStrategy(store)
	require.Equal(t, "VersionInBranchName", s.Name())

	versions, err := s.GetBaseVersions(ctx, ec, false)
	require.NoError(t, err)
	require.Len(t, versions, 1)
	require.Equal(t, int64(1), versions[0].SemanticVersion.Major)
	require.Equal(t, int64(2), versions[0].SemanticVersion.Minor)
	require.False(t, versions[0].ShouldIncrement)
	require.Equal(t, "Version in branch name", versions[0].Source)
}

func TestBranchName_ReleaseWithDash(t *testing.T) {
	tip := newTestCommit("bbb0000000000000000000000000000000000000", "tip")
	branchPoint := newTestCommit("aaa0000000000000000000000000000000000000", "fork")
	branch := git.Branch{
		Name: git.NewReferenceName("refs/heads/release-1.2.0"),
		Tip:  &tip,
	}
	cfg := releaseBranchConfig(t)

	mock := &git.MockRepository{
		BranchesFunc: func(filters ...git.PathFilter) ([]git.Branch, error) {
			return []git.Branch{branch}, nil
		},
		FindMergeBaseFunc: func(sha1, sha2 string) (string, error) {
			return branchPoint.Sha, nil
		},
		CommitFromShaFunc: func(sha string) (git.Commit, error) { return branchPoint, nil },
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip, branchPoint}, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{
		CurrentBranch:     branch,
		CurrentCommit:     tip,
		FullConfiguration: cfg,
	}
	ec := config.EffectiveConfiguration{TagPrefix: "[vV]"}

	s := NewVersionInBranchNameStrategy(store)
	versions, err := s.GetBaseVersions(ctx, ec, false)
	require.NoError(t, err)
	require.Len(t, versions, 1)
	require.Equal(t, int64(1), versions[0].SemanticVersion.Major)
	require.Equal(t, int64(2), versions[0].SemanticVersion.Minor)
}

func TestBranchName_NotReleaseBranch(t *testing.T) {
	tip := newTestCommit("bbb0000000000000000000000000000000000000", "tip")
	branch := git.Branch{
		Name: git.NewReferenceName("refs/heads/feature/auth"),
		Tip:  &tip,
	}
	cfg := releaseBranchConfig(t)

	store := git.NewRepositoryStore(&git.MockRepository{})

	ctx := &context.GitVersionContext{
		CurrentBranch:     branch,
		CurrentCommit:     tip,
		FullConfiguration: cfg,
	}
	ec := config.EffectiveConfiguration{TagPrefix: "[vV]"}

	s := NewVersionInBranchNameStrategy(store)
	versions, err := s.GetBaseVersions(ctx, ec, false)
	require.NoError(t, err)
	require.Nil(t, versions)
}

func TestBranchName_BranchNameOverride(t *testing.T) {
	tests := []struct {
		name     string
		branch   string
		version  string
		expected string
	}{
		{"release/version", "release/1.2.0", "1.2.0", "release"},
		{"release-version", "release-1.2.0", "1.2.0", "release"},
		{"releases/version", "releases/1.2.0", "1.2.0", "releases"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computeBranchNameOverride(tt.branch, tt.version)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestBranchName_PartialVersion(t *testing.T) {
	tip := newTestCommit("bbb0000000000000000000000000000000000000", "tip")
	branchPoint := newTestCommit("aaa0000000000000000000000000000000000000", "fork")
	branch := git.Branch{
		Name: git.NewReferenceName("refs/heads/release/1.2"),
		Tip:  &tip,
	}
	cfg := releaseBranchConfig(t)

	mock := &git.MockRepository{
		BranchesFunc: func(filters ...git.PathFilter) ([]git.Branch, error) {
			return []git.Branch{branch}, nil
		},
		FindMergeBaseFunc: func(sha1, sha2 string) (string, error) {
			return branchPoint.Sha, nil
		},
		CommitFromShaFunc: func(sha string) (git.Commit, error) { return branchPoint, nil },
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip, branchPoint}, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{
		CurrentBranch:     branch,
		CurrentCommit:     tip,
		FullConfiguration: cfg,
	}
	ec := config.EffectiveConfiguration{TagPrefix: "[vV]"}

	s := NewVersionInBranchNameStrategy(store)
	versions, err := s.GetBaseVersions(ctx, ec, false)
	require.NoError(t, err)
	require.Len(t, versions, 1)
	// 1.2 is normalized to 1.2.0
	require.Equal(t, int64(1), versions[0].SemanticVersion.Major)
	require.Equal(t, int64(2), versions[0].SemanticVersion.Minor)
	require.Equal(t, int64(0), versions[0].SemanticVersion.Patch)
}

func TestBranchName_WithExplain(t *testing.T) {
	branchPoint := newTestCommit("aaa0000000000000000000000000000000000000", "fork point")
	tip := newTestCommit("bbb0000000000000000000000000000000000000", "tip")
	branch := git.Branch{
		Name: git.NewReferenceName("refs/heads/release/2.0.0"),
		Tip:  &tip,
	}
	cfg := releaseBranchConfig(t)

	mock := &git.MockRepository{
		BranchesFunc: func(filters ...git.PathFilter) ([]git.Branch, error) {
			return []git.Branch{branch}, nil
		},
		FindMergeBaseFunc: func(sha1, sha2 string) (string, error) {
			return branchPoint.Sha, nil
		},
		CommitFromShaFunc: func(sha string) (git.Commit, error) {
			return branchPoint, nil
		},
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip, branchPoint}, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{
		CurrentBranch:     branch,
		CurrentCommit:     tip,
		FullConfiguration: cfg,
	}
	ec := config.EffectiveConfiguration{TagPrefix: "[vV]"}

	s := NewVersionInBranchNameStrategy(store)
	versions, err := s.GetBaseVersions(ctx, ec, true)
	require.NoError(t, err)
	require.Len(t, versions, 1)
	require.NotNil(t, versions[0].Explanation)
	require.Equal(t, "VersionInBranchName", versions[0].Explanation.Strategy)
	require.NotEmpty(t, versions[0].Explanation.Steps)
}

func TestBranchName_NotReleaseBranch_WithExplain(t *testing.T) {
	tip := newTestCommit("bbb0000000000000000000000000000000000000", "tip")
	branch := git.Branch{
		Name: git.NewReferenceName("refs/heads/feature/auth"),
		Tip:  &tip,
	}
	cfg := releaseBranchConfig(t)

	store := git.NewRepositoryStore(&git.MockRepository{})

	ctx := &context.GitVersionContext{
		CurrentBranch:     branch,
		CurrentCommit:     tip,
		FullConfiguration: cfg,
	}
	ec := config.EffectiveConfiguration{TagPrefix: "[vV]"}

	s := NewVersionInBranchNameStrategy(store)
	versions, err := s.GetBaseVersions(ctx, ec, true)
	require.NoError(t, err)
	require.Nil(t, versions)
}

func TestBranchName_NoVersionInName(t *testing.T) {
	tip := newTestCommit("bbb0000000000000000000000000000000000000", "tip")
	branch := git.Branch{
		Name: git.NewReferenceName("refs/heads/release/next"),
		Tip:  &tip,
	}
	cfg := releaseBranchConfig(t)

	store := git.NewRepositoryStore(&git.MockRepository{})

	ctx := &context.GitVersionContext{
		CurrentBranch:     branch,
		CurrentCommit:     tip,
		FullConfiguration: cfg,
	}
	ec := config.EffectiveConfiguration{TagPrefix: "[vV]"}

	s := NewVersionInBranchNameStrategy(store)
	versions, err := s.GetBaseVersions(ctx, ec, false)
	require.NoError(t, err)
	require.Nil(t, versions)
}
