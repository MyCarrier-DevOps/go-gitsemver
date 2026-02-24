package config

import (
	"testing"

	"github.com/MyCarrier-DevOps/go-gitsemver/internal/semver"

	"github.com/stretchr/testify/require"
)

func TestBranchConfig_MergeTo_NilSource(t *testing.T) {
	target := &BranchConfig{Regex: stringPtr("^main$")}
	var nilBC *BranchConfig
	nilBC.MergeTo(target)
	require.Equal(t, "^main$", *target.Regex)
}

func TestBranchConfig_MergeTo_NilTarget(t *testing.T) {
	src := &BranchConfig{Regex: stringPtr("^main$")}
	src.MergeTo(nil) // should not panic
}

func TestBranchConfig_MergeTo_PartialOverride(t *testing.T) {
	target := &BranchConfig{
		Regex:     stringPtr("^main$"),
		Increment: incrementPtr(semver.IncrementStrategyPatch),
		Tag:       stringPtr(""),
		Priority:  intPtr(100),
	}
	src := &BranchConfig{
		Increment: incrementPtr(semver.IncrementStrategyMinor),
		Tag:       stringPtr("alpha"),
	}

	src.MergeTo(target)

	require.Equal(t, "^main$", *target.Regex, "regex should not change")
	require.Equal(t, semver.IncrementStrategyMinor, *target.Increment, "increment should be overridden")
	require.Equal(t, "alpha", *target.Tag, "tag should be overridden")
	require.Equal(t, 100, *target.Priority, "priority should not change")
}

func TestBranchConfig_MergeTo_FullOverride(t *testing.T) {
	target := &BranchConfig{}
	src := &BranchConfig{
		Regex:                                 stringPtr("^dev$"),
		Increment:                             incrementPtr(semver.IncrementStrategyMinor),
		Mode:                                  versioningModePtr(semver.VersioningModeContinuousDeployment),
		Tag:                                   stringPtr("alpha"),
		SourceBranches:                        strSlicePtr([]string{"main"}),
		IsSourceBranchFor:                     strSlicePtr([]string{"feature"}),
		IsMainline:                            boolPtr(false),
		IsReleaseBranch:                       boolPtr(false),
		TracksReleaseBranches:                 boolPtr(true),
		PreventIncrementOfMergedBranchVersion: boolPtr(false),
		TrackMergeTarget:                      boolPtr(true),
		TagNumberPattern:                      stringPtr(`[/-](\d+)`),
		CommitMessageIncrementing:             commitMsgIncrPtr(semver.CommitMessageIncrementEnabled),
		PreReleaseWeight:                      intPtr(30000),
		Priority:                              intPtr(60),
	}

	src.MergeTo(target)

	require.Equal(t, "^dev$", *target.Regex)
	require.Equal(t, semver.IncrementStrategyMinor, *target.Increment)
	require.Equal(t, semver.VersioningModeContinuousDeployment, *target.Mode)
	require.Equal(t, "alpha", *target.Tag)
	require.Equal(t, []string{"main"}, *target.SourceBranches)
	require.Equal(t, []string{"feature"}, *target.IsSourceBranchFor)
	require.Equal(t, false, *target.IsMainline)
	require.Equal(t, false, *target.IsReleaseBranch)
	require.Equal(t, true, *target.TracksReleaseBranches)
	require.Equal(t, false, *target.PreventIncrementOfMergedBranchVersion)
	require.Equal(t, true, *target.TrackMergeTarget)
	require.Equal(t, `[/-](\d+)`, *target.TagNumberPattern)
	require.Equal(t, semver.CommitMessageIncrementEnabled, *target.CommitMessageIncrementing)
	require.Equal(t, 30000, *target.PreReleaseWeight)
	require.Equal(t, 60, *target.Priority)
}

func TestBranchConfig_MergeTo_PointerIndependence(t *testing.T) {
	srcTag := "beta"
	src := &BranchConfig{Tag: &srcTag}
	target := &BranchConfig{}

	src.MergeTo(target)

	// Mutating the source pointer should not affect target since
	// both point to the same string, but strings are immutable in Go.
	require.Equal(t, "beta", *target.Tag)
}

func TestBranchConfig_MergeTo_EmptySliceNotNil(t *testing.T) {
	// An explicitly empty slice (not nil) should override.
	src := &BranchConfig{
		SourceBranches: strSlicePtr([]string{}),
	}
	target := &BranchConfig{
		SourceBranches: strSlicePtr([]string{"main", "develop"}),
	}

	src.MergeTo(target)
	require.NotNil(t, target.SourceBranches)
	require.Empty(t, *target.SourceBranches)
}
