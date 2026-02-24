package strategy

import (
	"go-gitsemver/internal/config"
	"go-gitsemver/internal/context"
	"go-gitsemver/internal/git"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTrackRelease_NotTracking(t *testing.T) {
	store := git.NewRepositoryStore(&git.MockRepository{})
	ctx := &context.GitVersionContext{}
	ec := config.EffectiveConfiguration{TracksReleaseBranches: false}

	s := NewTrackReleaseBranchesStrategy(store)
	require.Equal(t, "TrackReleaseBranches", s.Name())

	versions, err := s.GetBaseVersions(ctx, ec, false)
	require.NoError(t, err)
	require.Nil(t, versions)
}

func TestTrackRelease_ReleaseBranchVersions(t *testing.T) {
	developTip := newTestCommit("ddd0000000000000000000000000000000000000", "develop tip")
	releaseTip := newTestCommit("rrr0000000000000000000000000000000000000", "release tip")
	mergeBase := newTestCommit("mmm0000000000000000000000000000000000000", "merge base")
	branchPoint := newTestCommit("bbb0000000000000000000000000000000000000", "branch point")

	developBranch := git.Branch{
		Name: git.NewReferenceName("refs/heads/develop"),
		Tip:  &developTip,
	}
	releaseBranch := git.Branch{
		Name: git.NewReferenceName("refs/heads/release/1.0.0"),
		Tip:  &releaseTip,
	}

	cfg := releaseBranchConfig(t)

	mock := &git.MockRepository{
		BranchesFunc: func(filters ...git.PathFilter) ([]git.Branch, error) {
			return []git.Branch{developBranch, releaseBranch}, nil
		},
		FindMergeBaseFunc: func(sha1, sha2 string) (string, error) {
			return mergeBase.Sha, nil
		},
		CommitFromShaFunc: func(sha string) (git.Commit, error) {
			if sha == mergeBase.Sha {
				return mergeBase, nil
			}
			if sha == branchPoint.Sha {
				return branchPoint, nil
			}
			return releaseTip, nil
		},
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{releaseTip, branchPoint}, nil
		},
		TagsFunc: func(filters ...git.PathFilter) ([]git.Tag, error) { return nil, nil },
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{
		CurrentBranch:     developBranch,
		CurrentCommit:     developTip,
		FullConfiguration: cfg,
	}
	ec := config.EffectiveConfiguration{
		TracksReleaseBranches: true,
		TagPrefix:             "[vV]",
	}

	s := NewTrackReleaseBranchesStrategy(store)
	versions, err := s.GetBaseVersions(ctx, ec, false)
	require.NoError(t, err)
	require.NotEmpty(t, versions)

	// Should have release branch version with ShouldIncrement=true.
	found := false
	for _, v := range versions {
		if v.SemanticVersion.Major == 1 {
			found = true
			require.True(t, v.ShouldIncrement)
			require.Contains(t, v.Source, "Release branch exists")
			require.Equal(t, mergeBase.Sha, v.BaseVersionSource.Sha)
			require.Empty(t, v.BranchNameOverride, "BranchNameOverride should be dropped")
		}
	}
	require.True(t, found, "expected release branch version")
}

func TestTrackRelease_MainBranchTags(t *testing.T) {
	developTip := newTestCommit("ddd0000000000000000000000000000000000000", "develop tip")
	mainTip := newTestCommit("mmm0000000000000000000000000000000000000", "main tip")
	tagCommit := newTestCommit("ttt0000000000000000000000000000000000000", "tagged")

	developBranch := git.Branch{
		Name: git.NewReferenceName("refs/heads/develop"),
		Tip:  &developTip,
	}
	mainBranch := git.Branch{
		Name: git.NewReferenceName("refs/heads/main"),
		Tip:  &mainTip,
	}

	cfg := releaseBranchConfig(t)

	mock := &git.MockRepository{
		BranchesFunc: func(filters ...git.PathFilter) ([]git.Branch, error) {
			return []git.Branch{developBranch, mainBranch}, nil
		},
		TagsFunc: func(filters ...git.PathFilter) ([]git.Tag, error) {
			return []git.Tag{
				{Name: git.NewReferenceName("refs/tags/v1.0.0"), TargetSha: tagCommit.Sha},
			}, nil
		},
		PeelTagToCommitFunc: func(tag git.Tag) (string, error) { return tag.TargetSha, nil },
		CommitFromShaFunc:   func(sha string) (git.Commit, error) { return tagCommit, nil },
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{mainTip, tagCommit}, nil
		},
		FindMergeBaseFunc: func(sha1, sha2 string) (string, error) {
			// Return develop tip as merge base to skip release branch (same as current commit).
			return developTip.Sha, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{
		CurrentBranch:     developBranch,
		CurrentCommit:     developTip,
		FullConfiguration: cfg,
	}
	ec := config.EffectiveConfiguration{
		TracksReleaseBranches: true,
		TagPrefix:             "[vV]",
	}

	s := NewTrackReleaseBranchesStrategy(store)
	versions, err := s.GetBaseVersions(ctx, ec, false)
	require.NoError(t, err)
	// Should have main branch tag version.
	found := false
	for _, v := range versions {
		if v.SemanticVersion.Major == 1 {
			found = true
			require.Contains(t, v.Source, "Git tag")
		}
	}
	require.True(t, found, "expected main tag version")
}

func TestTrackRelease_ReleaseBranchNoOwnCommits(t *testing.T) {
	developTip := newTestCommit("ddd0000000000000000000000000000000000000", "develop tip")
	releaseTip := newTestCommit("rrr0000000000000000000000000000000000000", "release tip")

	developBranch := git.Branch{
		Name: git.NewReferenceName("refs/heads/develop"),
		Tip:  &developTip,
	}
	releaseBranch := git.Branch{
		Name: git.NewReferenceName("refs/heads/release/1.0.0"),
		Tip:  &releaseTip,
	}

	cfg := releaseBranchConfig(t)

	mock := &git.MockRepository{
		BranchesFunc: func(filters ...git.PathFilter) ([]git.Branch, error) {
			return []git.Branch{developBranch, releaseBranch}, nil
		},
		FindMergeBaseFunc: func(sha1, sha2 string) (string, error) {
			// Merge base equals current commit = branch has no own commits.
			return developTip.Sha, nil
		},
		CommitFromShaFunc: func(sha string) (git.Commit, error) { return developTip, nil },
		TagsFunc:          func(filters ...git.PathFilter) ([]git.Tag, error) { return nil, nil },
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return nil, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{
		CurrentBranch:     developBranch,
		CurrentCommit:     developTip,
		FullConfiguration: cfg,
	}
	ec := config.EffectiveConfiguration{
		TracksReleaseBranches: true,
		TagPrefix:             "[vV]",
	}

	s := NewTrackReleaseBranchesStrategy(store)
	versions, err := s.GetBaseVersions(ctx, ec, false)
	require.NoError(t, err)
	// Release branch should be skipped (merge base == current commit).
	// No main tags either.
	require.Empty(t, versions)
}

func TestTrackRelease_NoMainBranch(t *testing.T) {
	developTip := newTestCommit("ddd0000000000000000000000000000000000000", "develop tip")
	developBranch := git.Branch{
		Name: git.NewReferenceName("refs/heads/develop"),
		Tip:  &developTip,
	}

	// Config with no main branch.
	boolTrue := true
	regex := `^releases?[/-]`
	cfg := &config.Config{
		Branches: map[string]*config.BranchConfig{
			"release": {Regex: &regex, IsReleaseBranch: &boolTrue},
		},
	}

	mock := &git.MockRepository{
		BranchesFunc: func(filters ...git.PathFilter) ([]git.Branch, error) {
			return []git.Branch{developBranch}, nil
		},
		TagsFunc: func(filters ...git.PathFilter) ([]git.Tag, error) { return nil, nil },
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{
		CurrentBranch:     developBranch,
		CurrentCommit:     developTip,
		FullConfiguration: cfg,
	}
	ec := config.EffectiveConfiguration{
		TracksReleaseBranches: true,
		TagPrefix:             "[vV]",
	}

	s := NewTrackReleaseBranchesStrategy(store)
	versions, err := s.GetBaseVersions(ctx, ec, false)
	require.NoError(t, err)
	// No release branches match, no main branch → empty.
	require.Empty(t, versions)
}

func TestTrackRelease_WithExplain(t *testing.T) {
	developTip := newTestCommit("ddd0000000000000000000000000000000000000", "develop tip")
	releaseTip := newTestCommit("rrr0000000000000000000000000000000000000", "release tip")
	mergeBase := newTestCommit("mmm0000000000000000000000000000000000000", "merge base")
	branchPoint := newTestCommit("bbb0000000000000000000000000000000000000", "branch point")

	developBranch := git.Branch{
		Name: git.NewReferenceName("refs/heads/develop"),
		Tip:  &developTip,
	}
	releaseBranch := git.Branch{
		Name: git.NewReferenceName("refs/heads/release/1.0.0"),
		Tip:  &releaseTip,
	}

	cfg := releaseBranchConfig(t)

	mock := &git.MockRepository{
		BranchesFunc: func(filters ...git.PathFilter) ([]git.Branch, error) {
			return []git.Branch{developBranch, releaseBranch}, nil
		},
		FindMergeBaseFunc: func(sha1, sha2 string) (string, error) {
			return mergeBase.Sha, nil
		},
		CommitFromShaFunc: func(sha string) (git.Commit, error) {
			if sha == mergeBase.Sha {
				return mergeBase, nil
			}
			if sha == branchPoint.Sha {
				return branchPoint, nil
			}
			return releaseTip, nil
		},
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{releaseTip, branchPoint}, nil
		},
		TagsFunc: func(filters ...git.PathFilter) ([]git.Tag, error) { return nil, nil },
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{
		CurrentBranch:     developBranch,
		CurrentCommit:     developTip,
		FullConfiguration: cfg,
	}
	ec := config.EffectiveConfiguration{
		TracksReleaseBranches: true,
		TagPrefix:             "[vV]",
	}

	s := NewTrackReleaseBranchesStrategy(store)
	versions, err := s.GetBaseVersions(ctx, ec, true)
	require.NoError(t, err)
	require.NotEmpty(t, versions)

	// At least one version should have explanation.
	found := false
	for _, v := range versions {
		if v.Explanation != nil {
			found = true
			require.NotEmpty(t, v.Explanation.Strategy)
		}
	}
	require.True(t, found, "at least one candidate should have explanation with explain=true")
}

func TestTrackRelease_MainBranchTags_WithExplain(t *testing.T) {
	developTip := newTestCommit("ddd0000000000000000000000000000000000000", "develop tip")
	mainTip := newTestCommit("mmm0000000000000000000000000000000000000", "main tip")
	tagCommit := newTestCommit("ttt0000000000000000000000000000000000000", "tagged")

	developBranch := git.Branch{
		Name: git.NewReferenceName("refs/heads/develop"),
		Tip:  &developTip,
	}
	mainBranch := git.Branch{
		Name: git.NewReferenceName("refs/heads/main"),
		Tip:  &mainTip,
	}

	cfg := releaseBranchConfig(t)

	mock := &git.MockRepository{
		BranchesFunc: func(filters ...git.PathFilter) ([]git.Branch, error) {
			return []git.Branch{developBranch, mainBranch}, nil
		},
		TagsFunc: func(filters ...git.PathFilter) ([]git.Tag, error) {
			return []git.Tag{
				{Name: git.NewReferenceName("refs/tags/v1.0.0"), TargetSha: tagCommit.Sha},
			}, nil
		},
		PeelTagToCommitFunc: func(tag git.Tag) (string, error) { return tag.TargetSha, nil },
		CommitFromShaFunc:   func(sha string) (git.Commit, error) { return tagCommit, nil },
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{mainTip, tagCommit}, nil
		},
		FindMergeBaseFunc: func(sha1, sha2 string) (string, error) {
			return developTip.Sha, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{
		CurrentBranch:     developBranch,
		CurrentCommit:     developTip,
		FullConfiguration: cfg,
	}
	ec := config.EffectiveConfiguration{
		TracksReleaseBranches: true,
		TagPrefix:             "[vV]",
	}

	s := NewTrackReleaseBranchesStrategy(store)
	versions, err := s.GetBaseVersions(ctx, ec, true)
	require.NoError(t, err)

	found := false
	for _, v := range versions {
		if v.SemanticVersion.Major == 1 && v.Explanation != nil {
			found = true
			require.Equal(t, "TaggedCommit", v.Explanation.Strategy)
		}
	}
	require.True(t, found, "expected main tag version with explanation")
}
