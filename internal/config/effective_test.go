package config

import (
	"go-gitsemver/internal/semver"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewEffectiveConfiguration_FullyPopulated(t *testing.T) {
	cfg := CreateDefaultConfiguration()
	branch := cfg.Branches["main"]

	ec := NewEffectiveConfiguration(cfg, branch)

	// Global fields
	require.Equal(t, semver.VersioningModeContinuousDelivery, ec.Mode)
	require.Equal(t, "[vV]", ec.TagPrefix)
	require.Equal(t, "0.1.0", ec.BaseVersion)
	require.Equal(t, "", ec.NextVersion)
	require.Equal(t, semver.IncrementStrategyInherit, ec.Increment)
	require.Equal(t, "ci", ec.ContinuousDeploymentFallbackTag)
	require.Equal(t, semver.CommitMessageIncrementEnabled, ec.CommitMessageIncrementing)
	require.Equal(t, semver.CommitMessageConventionBoth, ec.CommitMessageConvention)
	require.Equal(t, true, ec.UpdateBuildNumber)
	require.Equal(t, int64(60000), ec.TagPreReleaseWeight)
	require.Equal(t, 4, ec.LegacySemVerPadding)

	// Branch-specific fields (main)
	require.Equal(t, `^master$|^main$`, ec.BranchRegex)
	require.Equal(t, semver.IncrementStrategyPatch, ec.BranchIncrement)
	require.Equal(t, "", ec.Tag)
	require.Equal(t, true, ec.IsMainline)
	require.Equal(t, false, ec.IsReleaseBranch)
	require.Equal(t, true, ec.PreventIncrementOfMergedBranchVersion)
	require.Equal(t, 55000, ec.PreReleaseWeight)
	require.Equal(t, 100, ec.Priority)
	require.Equal(t, []string{"develop", "release"}, ec.SourceBranches)
}

func TestNewEffectiveConfiguration_DevelopBranch(t *testing.T) {
	cfg := CreateDefaultConfiguration()
	branch := cfg.Branches["develop"]

	ec := NewEffectiveConfiguration(cfg, branch)

	require.Equal(t, semver.IncrementStrategyMinor, ec.BranchIncrement)
	require.Equal(t, "alpha", ec.Tag)
	require.Equal(t, true, ec.TracksReleaseBranches)
	require.Equal(t, true, ec.TrackMergeTarget)
	require.Equal(t, 0, ec.PreReleaseWeight)
	require.Equal(t, 60, ec.Priority)
}

func TestNewEffectiveConfiguration_NilBranch(t *testing.T) {
	cfg := CreateDefaultConfiguration()

	ec := NewEffectiveConfiguration(cfg, nil)

	// Global fields should still resolve
	require.Equal(t, semver.VersioningModeContinuousDelivery, ec.Mode)
	require.Equal(t, "[vV]", ec.TagPrefix)
	// Branch fields should have defaults
	require.Equal(t, "", ec.BranchRegex)
	require.Equal(t, false, ec.IsMainline)
	require.Equal(t, 0, ec.Priority)
}

func TestNewEffectiveConfiguration_PartialBranch(t *testing.T) {
	cfg := CreateDefaultConfiguration()
	branch := &BranchConfig{
		Regex: stringPtr("^custom$"),
		Tag:   stringPtr("rc"),
		// Increment is nil — should fall back to global
	}

	ec := NewEffectiveConfiguration(cfg, branch)

	require.Equal(t, "^custom$", ec.BranchRegex)
	require.Equal(t, "rc", ec.Tag)
	// Falls back to global increment
	require.Equal(t, semver.IncrementStrategyInherit, ec.BranchIncrement)
	// Falls back to global mode
	require.Equal(t, semver.VersioningModeContinuousDelivery, ec.BranchMode)
}

func TestNewEffectiveConfiguration_EmptyConfig(t *testing.T) {
	cfg := &Config{}
	branch := &BranchConfig{}

	ec := NewEffectiveConfiguration(cfg, branch)

	// All should resolve to hard-coded defaults
	require.Equal(t, semver.VersioningModeContinuousDelivery, ec.Mode)
	require.Equal(t, "[vV]", ec.TagPrefix)
	require.Equal(t, "0.1.0", ec.BaseVersion)
	require.Equal(t, semver.IncrementStrategyInherit, ec.Increment)
	require.Equal(t, "ci", ec.ContinuousDeploymentFallbackTag)
	require.Equal(t, 4, ec.LegacySemVerPadding)
	require.Equal(t, "{BranchName}", ec.Tag)
}

func TestNewEffectiveConfiguration_IgnoreConfig(t *testing.T) {
	now := time.Now()
	cfg := &Config{
		Ignore: IgnoreConfig{
			CommitsBefore: &now,
			Sha:           []string{"abc123", "def456"},
		},
	}

	ec := NewEffectiveConfiguration(cfg, nil)

	require.NotNil(t, ec.IgnoreCommitsBefore)
	require.Equal(t, now, *ec.IgnoreCommitsBefore)
	require.Equal(t, []string{"abc123", "def456"}, ec.IgnoreSha)
}

func TestNewEffectiveConfiguration_MergeMessageFormats(t *testing.T) {
	cfg := &Config{
		MergeMessageFormats: map[string]string{
			"azure": `^Merged PR (\d+)$`,
		},
	}

	ec := NewEffectiveConfiguration(cfg, nil)

	require.Equal(t, `^Merged PR (\d+)$`, ec.MergeMessageFormats["azure"])
}

func TestNewEffectiveConfiguration_BranchModeOverridesGlobal(t *testing.T) {
	cfg := CreateDefaultConfiguration()
	branch := &BranchConfig{
		Mode: versioningModePtr(semver.VersioningModeMainline),
	}

	ec := NewEffectiveConfiguration(cfg, branch)

	require.Equal(t, semver.VersioningModeContinuousDelivery, ec.Mode) // global
	require.Equal(t, semver.VersioningModeMainline, ec.BranchMode)     // branch override
}
