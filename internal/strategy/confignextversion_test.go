package strategy

import (
	"go-gitsemver/internal/config"
	"go-gitsemver/internal/context"
	"go-gitsemver/internal/semver"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigNextVersion_ReturnsVersion(t *testing.T) {
	ctx := &context.GitVersionContext{}
	ec := config.EffectiveConfiguration{NextVersion: "2.0.0"}

	s := NewConfigNextVersionStrategy()
	require.Equal(t, "ConfigNextVersion", s.Name())

	versions, err := s.GetBaseVersions(ctx, ec, false)
	require.NoError(t, err)
	require.Len(t, versions, 1)
	require.Equal(t, int64(2), versions[0].SemanticVersion.Major)
	require.False(t, versions[0].ShouldIncrement)
	require.Nil(t, versions[0].BaseVersionSource)
	require.Equal(t, "NextVersion in configuration file", versions[0].Source)
}

func TestConfigNextVersion_Empty(t *testing.T) {
	ctx := &context.GitVersionContext{}
	ec := config.EffectiveConfiguration{NextVersion: ""}

	s := NewConfigNextVersionStrategy()
	versions, err := s.GetBaseVersions(ctx, ec, false)
	require.NoError(t, err)
	require.Nil(t, versions)
}

func TestConfigNextVersion_TaggedSkips(t *testing.T) {
	ctx := &context.GitVersionContext{
		IsCurrentCommitTagged:      true,
		CurrentCommitTaggedVersion: semver.SemanticVersion{Major: 1},
	}
	ec := config.EffectiveConfiguration{NextVersion: "2.0.0"}

	s := NewConfigNextVersionStrategy()
	versions, err := s.GetBaseVersions(ctx, ec, false)
	require.NoError(t, err)
	require.Nil(t, versions)
}

func TestConfigNextVersion_ParseError(t *testing.T) {
	ctx := &context.GitVersionContext{}
	ec := config.EffectiveConfiguration{NextVersion: "not-a-version"}

	s := NewConfigNextVersionStrategy()
	_, err := s.GetBaseVersions(ctx, ec, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "parsing next-version")
}

func TestConfigNextVersion_Explanation(t *testing.T) {
	ctx := &context.GitVersionContext{}
	ec := config.EffectiveConfiguration{NextVersion: "3.0.0"}

	s := NewConfigNextVersionStrategy()
	versions, err := s.GetBaseVersions(ctx, ec, true)
	require.NoError(t, err)
	require.Len(t, versions, 1)
	require.NotNil(t, versions[0].Explanation)
	require.Equal(t, "ConfigNextVersion", versions[0].Explanation.Strategy)
	require.NotEmpty(t, versions[0].Explanation.Steps)
}

func TestConfigNextVersion_ExplanationWhenEmpty(t *testing.T) {
	ctx := &context.GitVersionContext{}
	ec := config.EffectiveConfiguration{NextVersion: ""}

	s := NewConfigNextVersionStrategy()
	versions, err := s.GetBaseVersions(ctx, ec, true)
	require.NoError(t, err)
	require.Nil(t, versions)
}
