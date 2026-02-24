package strategy

import (
	"go-gitsemver/internal/config"
	"go-gitsemver/internal/context"
	"go-gitsemver/internal/git"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMergeMessage_DefaultFormat(t *testing.T) {
	mergeCommit := git.Commit{
		Sha:     "aaa0000000000000000000000000000000000000",
		Parents: []string{"p1", "p2"},
		Message: "Merge branch 'release/1.2.0' into main",
	}
	tip := newTestCommit("bbb0000000000000000000000000000000000000", "head")
	cfg := releaseBranchConfig(t)

	mock := &git.MockRepository{
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip, mergeCommit}, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{
		CurrentBranch:     git.Branch{Tip: &tip},
		CurrentCommit:     tip,
		FullConfiguration: cfg,
	}
	ec := config.EffectiveConfiguration{TagPrefix: "[vV]"}

	s := NewMergeMessageStrategy(store)
	require.Equal(t, "MergeMessage", s.Name())

	versions, err := s.GetBaseVersions(ctx, ec, false)
	require.NoError(t, err)
	require.Len(t, versions, 1)
	require.Equal(t, int64(1), versions[0].SemanticVersion.Major)
	require.Equal(t, int64(2), versions[0].SemanticVersion.Minor)
	require.True(t, versions[0].ShouldIncrement)
}

func TestMergeMessage_GitHubPull(t *testing.T) {
	mergeCommit := git.Commit{
		Sha:     "aaa0000000000000000000000000000000000000",
		Parents: []string{"p1", "p2"},
		Message: "Merge pull request #42 from release/1.3.0",
	}
	tip := newTestCommit("bbb0000000000000000000000000000000000000", "head")
	cfg := releaseBranchConfig(t)

	mock := &git.MockRepository{
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip, mergeCommit}, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{
		CurrentBranch:     git.Branch{Tip: &tip},
		CurrentCommit:     tip,
		FullConfiguration: cfg,
	}
	ec := config.EffectiveConfiguration{TagPrefix: "[vV]"}

	s := NewMergeMessageStrategy(store)
	versions, err := s.GetBaseVersions(ctx, ec, false)
	require.NoError(t, err)
	require.Len(t, versions, 1)
	require.Equal(t, int64(1), versions[0].SemanticVersion.Major)
	require.Equal(t, int64(3), versions[0].SemanticVersion.Minor)
}

func TestMergeMessage_NotReleaseBranch(t *testing.T) {
	mergeCommit := git.Commit{
		Sha:     "aaa0000000000000000000000000000000000000",
		Parents: []string{"p1", "p2"},
		Message: "Merge branch 'feature/auth' into main",
	}
	tip := newTestCommit("bbb0000000000000000000000000000000000000", "head")
	cfg := releaseBranchConfig(t)

	mock := &git.MockRepository{
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip, mergeCommit}, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{
		CurrentBranch:     git.Branch{Tip: &tip},
		CurrentCommit:     tip,
		FullConfiguration: cfg,
	}
	ec := config.EffectiveConfiguration{TagPrefix: "[vV]"}

	s := NewMergeMessageStrategy(store)
	versions, err := s.GetBaseVersions(ctx, ec, false)
	require.NoError(t, err)
	require.Empty(t, versions)
}

func TestMergeMessage_NonMergeCommit(t *testing.T) {
	singleParent := git.Commit{
		Sha:     "aaa0000000000000000000000000000000000000",
		Parents: []string{"p1"},
		Message: "feat: add new authentication module",
	}
	tip := newTestCommit("bbb0000000000000000000000000000000000000", "head")
	cfg := releaseBranchConfig(t)

	mock := &git.MockRepository{
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip, singleParent}, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{
		CurrentBranch:     git.Branch{Tip: &tip},
		CurrentCommit:     tip,
		FullConfiguration: cfg,
	}
	ec := config.EffectiveConfiguration{TagPrefix: "[vV]"}

	s := NewMergeMessageStrategy(store)
	versions, err := s.GetBaseVersions(ctx, ec, false)
	require.NoError(t, err)
	// Regular commit (not a merge or squash) produces no results.
	require.Empty(t, versions)
}

func TestMergeMessage_SquashMerge(t *testing.T) {
	// Single-parent commit with merge-style message is treated as a squash merge.
	squashCommit := git.Commit{
		Sha:     "aaa0000000000000000000000000000000000000",
		Parents: []string{"p1"},
		Message: "Merge branch 'release/1.2.0' into main",
	}
	tip := newTestCommit("bbb0000000000000000000000000000000000000", "head")
	cfg := releaseBranchConfig(t)

	mock := &git.MockRepository{
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip, squashCommit}, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{
		CurrentBranch:     git.Branch{Tip: &tip},
		CurrentCommit:     tip,
		FullConfiguration: cfg,
	}
	ec := config.EffectiveConfiguration{TagPrefix: "[vV]"}

	s := NewMergeMessageStrategy(store)
	versions, err := s.GetBaseVersions(ctx, ec, false)
	require.NoError(t, err)
	require.Len(t, versions, 1)
	require.Equal(t, int64(1), versions[0].SemanticVersion.Major)
	require.Equal(t, int64(2), versions[0].SemanticVersion.Minor)
	require.Contains(t, versions[0].Source, "Squash merge")
}

func TestMergeMessage_PreventIncrement(t *testing.T) {
	mergeCommit := git.Commit{
		Sha:     "aaa0000000000000000000000000000000000000",
		Parents: []string{"p1", "p2"},
		Message: "Merge branch 'release/1.2.0' into main",
	}
	tip := newTestCommit("bbb0000000000000000000000000000000000000", "head")
	cfg := releaseBranchConfig(t)

	mock := &git.MockRepository{
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip, mergeCommit}, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{
		CurrentBranch:     git.Branch{Tip: &tip},
		CurrentCommit:     tip,
		FullConfiguration: cfg,
	}
	ec := config.EffectiveConfiguration{
		TagPrefix:                             "[vV]",
		PreventIncrementOfMergedBranchVersion: true,
	}

	s := NewMergeMessageStrategy(store)
	versions, err := s.GetBaseVersions(ctx, ec, false)
	require.NoError(t, err)
	require.Len(t, versions, 1)
	require.False(t, versions[0].ShouldIncrement)
}

func TestMergeMessage_MaxResults(t *testing.T) {
	var commits []git.Commit
	tip := newTestCommit("bbb0000000000000000000000000000000000000", "head")
	commits = append(commits, tip)
	// Create 10 merge commits from release branches.
	for i := range 10 {
		commits = append(commits, git.Commit{
			Sha:     "aaa000000000000000000000000000000000000" + string(rune('0'+i)),
			Parents: []string{"p1", "p2"},
			Message: "Merge branch 'release/1." + string(rune('0'+i)) + ".0' into main",
		})
	}
	cfg := releaseBranchConfig(t)

	mock := &git.MockRepository{
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return commits, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{
		CurrentBranch:     git.Branch{Tip: &tip},
		CurrentCommit:     tip,
		FullConfiguration: cfg,
	}
	ec := config.EffectiveConfiguration{TagPrefix: "[vV]"}

	s := NewMergeMessageStrategy(store)
	versions, err := s.GetBaseVersions(ctx, ec, false)
	require.NoError(t, err)
	require.Len(t, versions, maxMergeMessageResults)
}

func TestMergeMessage_NilTip(t *testing.T) {
	store := git.NewRepositoryStore(&git.MockRepository{})

	ctx := &context.GitVersionContext{
		CurrentBranch: git.Branch{},
	}
	ec := config.EffectiveConfiguration{}

	s := NewMergeMessageStrategy(store)
	versions, err := s.GetBaseVersions(ctx, ec, false)
	require.NoError(t, err)
	require.Nil(t, versions)
}

func TestFirstLine(t *testing.T) {
	require.Equal(t, "first", firstLine("first\nsecond"))
	require.Equal(t, "only", firstLine("only"))
	require.Equal(t, "", firstLine(""))
}

func TestMergeMessage_WithExplain(t *testing.T) {
	mergeCommit := git.Commit{
		Sha:     "aaa0000000000000000000000000000000000000",
		Parents: []string{"p1", "p2"},
		Message: "Merge branch 'release/1.2.0' into main",
	}
	tip := newTestCommit("bbb0000000000000000000000000000000000000", "head")
	cfg := releaseBranchConfig(t)

	mock := &git.MockRepository{
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip, mergeCommit}, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{
		CurrentBranch:     git.Branch{Tip: &tip},
		CurrentCommit:     tip,
		FullConfiguration: cfg,
	}
	ec := config.EffectiveConfiguration{TagPrefix: "[vV]"}

	s := NewMergeMessageStrategy(store)
	versions, err := s.GetBaseVersions(ctx, ec, true)
	require.NoError(t, err)
	require.Len(t, versions, 1)
	require.NotNil(t, versions[0].Explanation)
	require.NotEmpty(t, versions[0].Explanation.Steps)
	require.Equal(t, "MergeMessage", versions[0].Explanation.Strategy)
}

func TestMergeMessage_SquashMerge_WithExplain(t *testing.T) {
	squashCommit := git.Commit{
		Sha:     "aaa0000000000000000000000000000000000000",
		Parents: []string{"p1"},
		Message: "Merge branch 'release/1.2.0' into main",
	}
	tip := newTestCommit("bbb0000000000000000000000000000000000000", "head")
	cfg := releaseBranchConfig(t)

	mock := &git.MockRepository{
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip, squashCommit}, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{
		CurrentBranch:     git.Branch{Tip: &tip},
		CurrentCommit:     tip,
		FullConfiguration: cfg,
	}
	ec := config.EffectiveConfiguration{TagPrefix: "[vV]"}

	s := NewMergeMessageStrategy(store)
	versions, err := s.GetBaseVersions(ctx, ec, true)
	require.NoError(t, err)
	require.Len(t, versions, 1)
	require.NotNil(t, versions[0].Explanation)
	require.Equal(t, "MergeMessage", versions[0].Explanation.Strategy)
}

func TestMergeMessage_RemotePrefix(t *testing.T) {
	require.Equal(t, "release/1.0.0", trimRemotePrefix("refs/remotes/release/1.0.0"))
	require.Equal(t, "release/1.0.0", trimRemotePrefix("origin/release/1.0.0"))
	require.Equal(t, "release/1.0.0", trimRemotePrefix("release/1.0.0"))
}

func TestMergeMessage_NoVersionInBranch(t *testing.T) {
	mergeCommit := git.Commit{
		Sha:     "aaa0000000000000000000000000000000000000",
		Parents: []string{"p1", "p2"},
		Message: "Merge branch 'release/next' into main",
	}
	tip := newTestCommit("bbb0000000000000000000000000000000000000", "head")
	cfg := releaseBranchConfig(t)

	mock := &git.MockRepository{
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{tip, mergeCommit}, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{
		CurrentBranch:     git.Branch{Tip: &tip},
		CurrentCommit:     tip,
		FullConfiguration: cfg,
	}
	ec := config.EffectiveConfiguration{TagPrefix: "[vV]"}

	s := NewMergeMessageStrategy(store)
	versions, err := s.GetBaseVersions(ctx, ec, false)
	require.NoError(t, err)
	require.Empty(t, versions, "branch without version should produce no results")
}
