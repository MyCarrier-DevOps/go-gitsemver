package config

import (
	"go-gitsemver/internal/semver"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestConfig_ZeroValue(t *testing.T) {
	var cfg Config
	require.Nil(t, cfg.Mode)
	require.Nil(t, cfg.TagPrefix)
	require.Nil(t, cfg.BaseVersion)
	require.Nil(t, cfg.NextVersion)
	require.Nil(t, cfg.Increment)
	require.Nil(t, cfg.Branches)
	require.True(t, cfg.Ignore.IsEmpty())
	require.Nil(t, cfg.MergeMessageFormats)
}

func TestConfig_YAMLRoundTrip(t *testing.T) {
	input := `mode: ContinuousDelivery
tag-prefix: '[vV]'
base-version: 1.0.0
increment: Patch
commit-message-convention: conventional-commits
branches:
  main:
    regex: ^main$
    increment: Patch
    tag: ''
    is-mainline: true
    priority: 100
  feature:
    regex: ^feature/
    increment: Inherit
    tag: '{BranchName}'
    priority: 50
merge-message-formats:
  azure: '^Merged PR (\d+)$'
ignore:
  commits-before: 2024-06-01
  sha:
    - abc123
`

	var cfg Config
	require.NoError(t, yaml.Unmarshal([]byte(input), &cfg))

	require.NotNil(t, cfg.Mode)
	require.Equal(t, semver.VersioningModeContinuousDelivery, *cfg.Mode)
	require.Equal(t, "[vV]", *cfg.TagPrefix)
	require.Equal(t, "1.0.0", *cfg.BaseVersion)
	require.Equal(t, semver.IncrementStrategyPatch, *cfg.Increment)
	require.Equal(t, semver.CommitMessageConventionConventionalCommits, *cfg.CommitMessageConvention)

	require.Len(t, cfg.Branches, 2)

	main := cfg.Branches["main"]
	require.NotNil(t, main)
	require.Equal(t, "^main$", *main.Regex)
	require.Equal(t, semver.IncrementStrategyPatch, *main.Increment)
	require.Equal(t, "", *main.Tag)
	require.Equal(t, true, *main.IsMainline)
	require.Equal(t, 100, *main.Priority)

	feature := cfg.Branches["feature"]
	require.NotNil(t, feature)
	require.Equal(t, "^feature/", *feature.Regex)
	require.Equal(t, semver.IncrementStrategyInherit, *feature.Increment)
	require.Equal(t, "{BranchName}", *feature.Tag)

	require.Equal(t, `^Merged PR (\d+)$`, cfg.MergeMessageFormats["azure"])

	require.False(t, cfg.Ignore.IsEmpty())
	require.NotNil(t, cfg.Ignore.CommitsBefore)
	require.Equal(t, []string{"abc123"}, cfg.Ignore.Sha)
}

func TestConfig_YAMLEmpty(t *testing.T) {
	var cfg Config
	require.NoError(t, yaml.Unmarshal([]byte("{}"), &cfg))
	require.Nil(t, cfg.Mode)
	require.Nil(t, cfg.Branches)
}
