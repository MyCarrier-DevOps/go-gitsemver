package config

import (
	"go-gitsemver/internal/semver"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuilder_NoOverrides(t *testing.T) {
	cfg, err := NewBuilder().Build()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Len(t, cfg.Branches, 8)
	require.Equal(t, semver.VersioningModeContinuousDelivery, *cfg.Mode)
}

func TestBuilder_GlobalOverrides(t *testing.T) {
	override := &Config{
		Mode:        versioningModePtr(semver.VersioningModeMainline),
		TagPrefix:   stringPtr("release-"),
		BaseVersion: stringPtr("1.0.0"),
	}

	cfg, err := NewBuilder().Add(override).Build()
	require.NoError(t, err)
	require.Equal(t, semver.VersioningModeMainline, *cfg.Mode)
	require.Equal(t, "release-", *cfg.TagPrefix)
	require.Equal(t, "1.0.0", *cfg.BaseVersion)
	// Defaults still present for unoverridden fields
	require.Equal(t, "ci", *cfg.ContinuousDeploymentFallbackTag)
}

func TestBuilder_BranchOverride_ExistingKey(t *testing.T) {
	override := &Config{
		Branches: map[string]*BranchConfig{
			"main": {
				Regex:    stringPtr(`^master$|^main$|^prod$`),
				Priority: intPtr(200),
			},
		},
	}

	cfg, err := NewBuilder().Add(override).Build()
	require.NoError(t, err)

	main := cfg.Branches["main"]
	require.Equal(t, `^master$|^main$|^prod$`, *main.Regex)
	require.Equal(t, 200, *main.Priority)
	// Default fields preserved
	require.Equal(t, semver.IncrementStrategyPatch, *main.Increment)
	require.Equal(t, "", *main.Tag)
	require.Equal(t, true, *main.IsMainline)
}

func TestBuilder_BranchOverride_NewKey(t *testing.T) {
	override := &Config{
		Branches: map[string]*BranchConfig{
			"staging": {
				Regex:     stringPtr(`^staging$`),
				Increment: incrementPtr(semver.IncrementStrategyNone),
				Tag:       stringPtr("rc"),
				Priority:  intPtr(85),
			},
		},
	}

	cfg, err := NewBuilder().Add(override).Build()
	require.NoError(t, err)
	require.Contains(t, cfg.Branches, "staging")
	require.Equal(t, "^staging$", *cfg.Branches["staging"].Regex)
	require.Equal(t, 9, len(cfg.Branches)) // 8 defaults + 1 new
}

func TestBuilder_MultipleOverrides(t *testing.T) {
	first := &Config{TagPrefix: stringPtr("v")}
	second := &Config{TagPrefix: stringPtr("release-")}

	cfg, err := NewBuilder().Add(first).Add(second).Build()
	require.NoError(t, err)
	require.Equal(t, "release-", *cfg.TagPrefix) // second wins
}

func TestBuilder_NilOverride(t *testing.T) {
	cfg, err := NewBuilder().Add(nil).Build()
	require.NoError(t, err)
	require.NotNil(t, cfg)
}

func TestBuilder_DevelopSpecialCase_DefaultMode(t *testing.T) {
	// Default mode is ContinuousDelivery, develop should get ContinuousDeployment
	cfg, err := NewBuilder().Build()
	require.NoError(t, err)
	require.Equal(t, semver.VersioningModeContinuousDeployment, *cfg.Branches["develop"].Mode)
}

func TestBuilder_DevelopSpecialCase_MainlineMode(t *testing.T) {
	// When global is Mainline, develop also gets Mainline
	override := &Config{Mode: versioningModePtr(semver.VersioningModeMainline)}
	cfg, err := NewBuilder().Add(override).Build()
	require.NoError(t, err)
	require.Equal(t, semver.VersioningModeMainline, *cfg.Branches["develop"].Mode)
}

func TestBuilder_DevelopSpecialCase_ExplicitOverride(t *testing.T) {
	// If user explicitly sets develop mode, it should be preserved
	override := &Config{
		Branches: map[string]*BranchConfig{
			"develop": {
				Mode: versioningModePtr(semver.VersioningModeContinuousDelivery),
			},
		},
	}
	cfg, err := NewBuilder().Add(override).Build()
	require.NoError(t, err)
	require.Equal(t, semver.VersioningModeContinuousDelivery, *cfg.Branches["develop"].Mode)
}

func TestBuilder_OtherBranchesInheritGlobalMode(t *testing.T) {
	cfg, err := NewBuilder().Build()
	require.NoError(t, err)
	// All non-develop branches should inherit ContinuousDelivery
	for name, branch := range cfg.Branches {
		if name == "develop" {
			continue
		}
		require.NotNil(t, branch.Mode, "branch %s should have mode set", name)
		require.Equal(t, semver.VersioningModeContinuousDelivery, *branch.Mode,
			"branch %s should inherit ContinuousDelivery", name)
	}
}

func TestBuilder_IsSourceBranchFor(t *testing.T) {
	override := &Config{
		Branches: map[string]*BranchConfig{
			"staging": {
				Regex:             stringPtr(`^staging$`),
				IsSourceBranchFor: strSlicePtr([]string{"feature"}),
				Priority:          intPtr(85),
			},
		},
	}

	cfg, err := NewBuilder().Add(override).Build()
	require.NoError(t, err)

	featureSources := *cfg.Branches["feature"].SourceBranches
	require.Contains(t, featureSources, "staging")
}

func TestBuilder_MergeMessageFormats(t *testing.T) {
	override := &Config{
		MergeMessageFormats: map[string]string{
			"azure": `^Merged PR (\d+)$`,
		},
	}

	cfg, err := NewBuilder().Add(override).Build()
	require.NoError(t, err)
	require.Equal(t, `^Merged PR (\d+)$`, cfg.MergeMessageFormats["azure"])
}

func TestBuilder_IgnoreConfig(t *testing.T) {
	override := &Config{
		Ignore: IgnoreConfig{
			Sha: []string{"abc123"},
		},
	}

	cfg, err := NewBuilder().Add(override).Build()
	require.NoError(t, err)
	require.Equal(t, []string{"abc123"}, cfg.Ignore.Sha)
}

func TestBuilder_Validate_InvalidBranchRegex(t *testing.T) {
	override := &Config{
		Branches: map[string]*BranchConfig{
			"bad": {
				Regex: stringPtr("[invalid"),
			},
		},
	}

	_, err := NewBuilder().Add(override).Build()
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid regex")
}

func TestBuilder_Validate_InvalidTagPrefix(t *testing.T) {
	override := &Config{
		TagPrefix: stringPtr("[invalid"),
	}

	_, err := NewBuilder().Add(override).Build()
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid tag-prefix regex")
}

func TestBuilder_InheritIncrement(t *testing.T) {
	override := &Config{
		Increment: incrementPtr(semver.IncrementStrategyMinor),
	}

	cfg, err := NewBuilder().Add(override).Build()
	require.NoError(t, err)

	// Branches that had nil increment should inherit from global
	// But defaults already set increment, so this tests the override path
	require.Equal(t, semver.IncrementStrategyMinor, *cfg.Increment)
}

func TestBuilder_InheritCommitMessageIncrementing(t *testing.T) {
	cfg, err := NewBuilder().Build()
	require.NoError(t, err)

	// All branches should have CommitMessageIncrementing set after finalization
	for name, branch := range cfg.Branches {
		require.NotNil(t, branch.CommitMessageIncrementing,
			"branch %s should have CommitMessageIncrementing set", name)
	}
}
