package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MyCarrier-DevOps/go-gitsemver/internal/semver"

	"github.com/stretchr/testify/require"
)

func TestLoadFromBytes_Full(t *testing.T) {
	data := []byte(`
mode: Mainline
tag-prefix: 'release-'
base-version: 2.0.0
next-version: 3.0.0
increment: Minor
continuous-delivery-fallback-tag: ci
commit-message-incrementing: Disabled
commit-message-convention: bump-directive
major-version-bump-message: '\+semver:\s?(breaking|major)'
minor-version-bump-message: '\+semver:\s?(feature|minor)'
patch-version-bump-message: '\+semver:\s?(fix|patch)'
no-bump-message: '\+semver:\s?(none|skip)'
commit-date-format: '2006-01-02'
update-build-number: false
tag-pre-release-weight: 50000
legacy-semver-padding: 5
build-metadata-padding: 6
commits-since-version-source-padding: 3
branches:
  main:
    regex: ^main$
    increment: Patch
    tag: ''
    is-mainline: true
    priority: 100
merge-message-formats:
  custom: '^PR (\d+)$'
ignore:
  sha:
    - deadbeef
`)

	cfg, err := LoadFromBytes(data)
	require.NoError(t, err)

	require.Equal(t, semver.VersioningModeMainline, *cfg.Mode)
	require.Equal(t, "release-", *cfg.TagPrefix)
	require.Equal(t, "2.0.0", *cfg.BaseVersion)
	require.Equal(t, "3.0.0", *cfg.NextVersion)
	require.Equal(t, semver.IncrementStrategyMinor, *cfg.Increment)
	require.Equal(t, "ci", *cfg.ContinuousDeploymentFallbackTag)
	require.Equal(t, semver.CommitMessageIncrementDisabled, *cfg.CommitMessageIncrementing)
	require.Equal(t, semver.CommitMessageConventionBumpDirective, *cfg.CommitMessageConvention)
	require.Equal(t, false, *cfg.UpdateBuildNumber)
	require.Equal(t, int64(50000), *cfg.TagPreReleaseWeight)
	require.Equal(t, 5, *cfg.LegacySemVerPadding)
	require.Equal(t, 6, *cfg.BuildMetaDataPadding)
	require.Equal(t, 3, *cfg.CommitsSinceVersionSourcePadding)

	require.Len(t, cfg.Branches, 1)
	require.Equal(t, "^main$", *cfg.Branches["main"].Regex)
	require.Equal(t, `^PR (\d+)$`, cfg.MergeMessageFormats["custom"])
	require.Equal(t, []string{"deadbeef"}, cfg.Ignore.Sha)
}

func TestLoadFromBytes_Minimal(t *testing.T) {
	cfg, err := LoadFromBytes([]byte(""))
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Nil(t, cfg.Mode)
	require.Nil(t, cfg.Branches)
}

func TestLoadFromBytes_EnumsCaseInsensitive(t *testing.T) {
	tests := []struct {
		name  string
		yaml  string
		check func(t *testing.T, cfg *Config)
	}{
		{
			"mode lowercase",
			"mode: mainline",
			func(t *testing.T, cfg *Config) {
				t.Helper()
				require.Equal(t, semver.VersioningModeMainline, *cfg.Mode)
			},
		},
		{
			"increment mixed case",
			"increment: Inherit",
			func(t *testing.T, cfg *Config) {
				t.Helper()
				require.Equal(t, semver.IncrementStrategyInherit, *cfg.Increment)
			},
		},
		{
			"convention hyphenated",
			"commit-message-convention: conventional-commits",
			func(t *testing.T, cfg *Config) {
				t.Helper()
				require.Equal(t, semver.CommitMessageConventionConventionalCommits, *cfg.CommitMessageConvention)
			},
		},
		{
			"commit-message-incrementing",
			"commit-message-incrementing: MergeMessageOnly",
			func(t *testing.T, cfg *Config) {
				t.Helper()
				require.Equal(t, semver.CommitMessageIncrementMergeMessageOnly, *cfg.CommitMessageIncrementing)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := LoadFromBytes([]byte(tt.yaml))
			require.NoError(t, err)
			tt.check(t, cfg)
		})
	}
}

func TestLoadFromBytes_InvalidEnum(t *testing.T) {
	_, err := LoadFromBytes([]byte("mode: InvalidMode"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown versioning mode")
}

func TestLoadFromBytes_InvalidYAML(t *testing.T) {
	_, err := LoadFromBytes([]byte("::bad yaml{{"))
	require.Error(t, err)
}

func TestLoadFromFile_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "go-gitsemver.yml")
	require.NoError(t, os.WriteFile(path, []byte("tag-prefix: 'v'\n"), 0o644))

	cfg, err := LoadFromFile(path)
	require.NoError(t, err)
	require.Equal(t, "v", *cfg.TagPrefix)
}

func TestLoadFromFile_NotFound(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/go-gitsemver.yml")
	require.Error(t, err)
	require.Contains(t, err.Error(), "reading config file")
}
