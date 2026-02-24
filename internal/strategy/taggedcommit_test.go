package strategy

import (
	"testing"
	"time"

	"github.com/MyCarrier-DevOps/go-gitsemver/internal/config"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/context"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/git"

	"github.com/stretchr/testify/require"
)

func newTestCommit(sha, msg string) git.Commit {
	return git.Commit{Sha: sha, When: time.Now(), Message: msg}
}

func TestTaggedCommit_SingleTag(t *testing.T) {
	tagCommit := newTestCommit("aaa0000000000000000000000000000000000000", "tagged")
	headCommit := newTestCommit("bbb0000000000000000000000000000000000000", "head")

	mock := &git.MockRepository{
		TagsFunc: func(filters ...git.PathFilter) ([]git.Tag, error) {
			return []git.Tag{
				{Name: git.NewReferenceName("refs/tags/v1.0.0"), TargetSha: tagCommit.Sha},
			}, nil
		},
		PeelTagToCommitFunc: func(tag git.Tag) (string, error) { return tag.TargetSha, nil },
		CommitFromShaFunc: func(sha string) (git.Commit, error) {
			if sha == tagCommit.Sha {
				return tagCommit, nil
			}
			return git.Commit{}, nil
		},
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{headCommit, tagCommit}, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{
		CurrentBranch: git.Branch{Tip: &headCommit},
		CurrentCommit: headCommit,
	}
	ec := config.EffectiveConfiguration{TagPrefix: "[vV]"}

	s := NewTaggedCommitStrategy(store)
	require.Equal(t, "TaggedCommit", s.Name())

	versions, err := s.GetBaseVersions(ctx, ec, false)
	require.NoError(t, err)
	require.Len(t, versions, 1)
	require.Equal(t, int64(1), versions[0].SemanticVersion.Major)
	require.True(t, versions[0].ShouldIncrement)
	require.Equal(t, tagCommit.Sha, versions[0].BaseVersionSource.Sha)
}

func TestTaggedCommit_TagOnCurrentCommit(t *testing.T) {
	headCommit := newTestCommit("aaa0000000000000000000000000000000000000", "tagged head")

	mock := &git.MockRepository{
		TagsFunc: func(filters ...git.PathFilter) ([]git.Tag, error) {
			return []git.Tag{
				{Name: git.NewReferenceName("refs/tags/v2.0.0"), TargetSha: headCommit.Sha},
			}, nil
		},
		PeelTagToCommitFunc: func(tag git.Tag) (string, error) { return tag.TargetSha, nil },
		CommitFromShaFunc:   func(sha string) (git.Commit, error) { return headCommit, nil },
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{headCommit}, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{
		CurrentBranch: git.Branch{Tip: &headCommit},
		CurrentCommit: headCommit,
	}
	ec := config.EffectiveConfiguration{TagPrefix: "[vV]"}

	s := NewTaggedCommitStrategy(store)
	versions, err := s.GetBaseVersions(ctx, ec, false)
	require.NoError(t, err)
	require.Len(t, versions, 1)
	require.False(t, versions[0].ShouldIncrement)
}

func TestTaggedCommit_MultipleTagsOnCurrent(t *testing.T) {
	headCommit := newTestCommit("aaa0000000000000000000000000000000000000", "multi-tagged")

	mock := &git.MockRepository{
		TagsFunc: func(filters ...git.PathFilter) ([]git.Tag, error) {
			return []git.Tag{
				{Name: git.NewReferenceName("refs/tags/v1.0.0"), TargetSha: headCommit.Sha},
				{Name: git.NewReferenceName("refs/tags/v2.0.0"), TargetSha: headCommit.Sha},
			}, nil
		},
		PeelTagToCommitFunc: func(tag git.Tag) (string, error) { return tag.TargetSha, nil },
		CommitFromShaFunc:   func(sha string) (git.Commit, error) { return headCommit, nil },
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{headCommit}, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{
		CurrentBranch: git.Branch{Tip: &headCommit},
		CurrentCommit: headCommit,
	}
	ec := config.EffectiveConfiguration{TagPrefix: "[vV]"}

	s := NewTaggedCommitStrategy(store)
	versions, err := s.GetBaseVersions(ctx, ec, false)
	require.NoError(t, err)
	require.Len(t, versions, 2)
	for _, v := range versions {
		require.False(t, v.ShouldIncrement)
	}
}

func TestTaggedCommit_MixedTags(t *testing.T) {
	tagCommit := newTestCommit("aaa0000000000000000000000000000000000000", "older tagged")
	headCommit := newTestCommit("bbb0000000000000000000000000000000000000", "head tagged")

	mock := &git.MockRepository{
		TagsFunc: func(filters ...git.PathFilter) ([]git.Tag, error) {
			return []git.Tag{
				{Name: git.NewReferenceName("refs/tags/v1.0.0"), TargetSha: tagCommit.Sha},
				{Name: git.NewReferenceName("refs/tags/v2.0.0"), TargetSha: headCommit.Sha},
			}, nil
		},
		PeelTagToCommitFunc: func(tag git.Tag) (string, error) { return tag.TargetSha, nil },
		CommitFromShaFunc: func(sha string) (git.Commit, error) {
			if sha == tagCommit.Sha {
				return tagCommit, nil
			}
			return headCommit, nil
		},
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{headCommit, tagCommit}, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{
		CurrentBranch: git.Branch{Tip: &headCommit},
		CurrentCommit: headCommit,
	}
	ec := config.EffectiveConfiguration{TagPrefix: "[vV]"}

	s := NewTaggedCommitStrategy(store)
	versions, err := s.GetBaseVersions(ctx, ec, false)
	require.NoError(t, err)
	// Only the tag on the current commit should be returned.
	require.Len(t, versions, 1)
	require.False(t, versions[0].ShouldIncrement)
	require.Equal(t, int64(2), versions[0].SemanticVersion.Major)
}

func TestTaggedCommit_NoTags(t *testing.T) {
	headCommit := newTestCommit("bbb0000000000000000000000000000000000000", "head")

	mock := &git.MockRepository{
		TagsFunc:            func(filters ...git.PathFilter) ([]git.Tag, error) { return nil, nil },
		PeelTagToCommitFunc: func(tag git.Tag) (string, error) { return tag.TargetSha, nil },
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{headCommit}, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{
		CurrentBranch: git.Branch{Tip: &headCommit},
		CurrentCommit: headCommit,
	}
	ec := config.EffectiveConfiguration{TagPrefix: "[vV]"}

	s := NewTaggedCommitStrategy(store)
	versions, err := s.GetBaseVersions(ctx, ec, false)
	require.NoError(t, err)
	require.Empty(t, versions)
}

func TestTaggedCommit_NilTip(t *testing.T) {
	store := git.NewRepositoryStore(&git.MockRepository{})

	ctx := &context.GitVersionContext{
		CurrentBranch: git.Branch{},
		CurrentCommit: newTestCommit("bbb0000000000000000000000000000000000000", "head"),
	}
	ec := config.EffectiveConfiguration{TagPrefix: "[vV]"}

	s := NewTaggedCommitStrategy(store)
	versions, err := s.GetBaseVersions(ctx, ec, false)
	require.NoError(t, err)
	require.Empty(t, versions)
}

func TestTaggedCommit_Explanation(t *testing.T) {
	headCommit := newTestCommit("aaa0000000000000000000000000000000000000", "tagged head")

	mock := &git.MockRepository{
		TagsFunc: func(filters ...git.PathFilter) ([]git.Tag, error) {
			return []git.Tag{
				{Name: git.NewReferenceName("refs/tags/v1.0.0"), TargetSha: headCommit.Sha},
			}, nil
		},
		PeelTagToCommitFunc: func(tag git.Tag) (string, error) { return tag.TargetSha, nil },
		CommitFromShaFunc:   func(sha string) (git.Commit, error) { return headCommit, nil },
		CommitLogFunc: func(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
			return []git.Commit{headCommit}, nil
		},
	}
	store := git.NewRepositoryStore(mock)

	ctx := &context.GitVersionContext{
		CurrentBranch: git.Branch{Tip: &headCommit},
		CurrentCommit: headCommit,
	}
	ec := config.EffectiveConfiguration{TagPrefix: "[vV]"}

	s := NewTaggedCommitStrategy(store)
	versions, err := s.GetBaseVersions(ctx, ec, true)
	require.NoError(t, err)
	require.Len(t, versions, 1)
	require.NotNil(t, versions[0].Explanation)
	require.NotEmpty(t, versions[0].Explanation.Steps)
}
