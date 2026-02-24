package config

import (
	"testing"

	"github.com/MyCarrier-DevOps/go-gitsemver/internal/semver"

	"github.com/stretchr/testify/require"
)

func TestCreateDefaultConfiguration_GlobalDefaults(t *testing.T) {
	cfg := CreateDefaultConfiguration()

	require.Equal(t, semver.VersioningModeContinuousDelivery, *cfg.Mode)
	require.Equal(t, "[vV]", *cfg.TagPrefix)
	require.Equal(t, "1.0.0", *cfg.BaseVersion)
	require.Equal(t, semver.IncrementStrategyInherit, *cfg.Increment)
	require.Equal(t, "ci", *cfg.ContinuousDeploymentFallbackTag)
	require.Equal(t, semver.CommitMessageIncrementEnabled, *cfg.CommitMessageIncrementing)
	require.Equal(t, semver.CommitMessageConventionBoth, *cfg.CommitMessageConvention)
	require.Equal(t, "2006-01-02", *cfg.CommitDateFormat)
	require.Equal(t, true, *cfg.UpdateBuildNumber)
	require.Equal(t, int64(60000), *cfg.TagPreReleaseWeight)
	require.Equal(t, 4, *cfg.LegacySemVerPadding)
	require.Equal(t, 4, *cfg.BuildMetaDataPadding)
	require.Equal(t, 4, *cfg.CommitsSinceVersionSourcePadding)
	require.Nil(t, cfg.NextVersion)
}

func TestCreateDefaultConfiguration_AllBranchesPresent(t *testing.T) {
	cfg := CreateDefaultConfiguration()
	expected := []string{"main", "develop", "release", "feature", "hotfix", "pull-request", "support", "unknown"}
	for _, name := range expected {
		require.Contains(t, cfg.Branches, name, "missing branch: %s", name)
	}
	require.Len(t, cfg.Branches, len(expected))
}

func TestCreateDefaultConfiguration_BranchDefaults(t *testing.T) {
	cfg := CreateDefaultConfiguration()

	tests := []struct {
		name             string
		regex            string
		increment        semver.IncrementStrategy
		tag              string
		isMainline       bool
		isRelease        bool
		tracksRelease    bool
		preventIncrement bool
		trackMergeTarget bool
		preReleaseWeight int
		priority         int
	}{
		{
			name: "main", regex: `^master$|^main$`,
			increment: semver.IncrementStrategyPatch, tag: "",
			isMainline: true, isRelease: false, tracksRelease: false,
			preventIncrement: true, trackMergeTarget: false,
			preReleaseWeight: 55000, priority: 100,
		},
		{
			name: "develop", regex: `^dev(elop)?(ment)?$`,
			increment: semver.IncrementStrategyMinor, tag: "alpha",
			isMainline: false, isRelease: false, tracksRelease: true,
			preventIncrement: false, trackMergeTarget: true,
			preReleaseWeight: 0, priority: 60,
		},
		{
			name: "release", regex: `^releases?[/-]`,
			increment: semver.IncrementStrategyNone, tag: "beta",
			isMainline: false, isRelease: true, tracksRelease: false,
			preventIncrement: true, trackMergeTarget: false,
			preReleaseWeight: 30000, priority: 90,
		},
		{
			name: "feature", regex: `^features?[/-]`,
			increment: semver.IncrementStrategyInherit, tag: "{BranchName}",
			isMainline: false, isRelease: false, tracksRelease: false,
			preventIncrement: false, trackMergeTarget: false,
			preReleaseWeight: 30000, priority: 50,
		},
		{
			name: "hotfix", regex: `^hotfix(es)?[/-]`,
			increment: semver.IncrementStrategyPatch, tag: "beta",
			isMainline: false, isRelease: false, tracksRelease: false,
			preventIncrement: false, trackMergeTarget: false,
			preReleaseWeight: 30000, priority: 80,
		},
		{
			name: "pull-request", regex: `^(pull|pull-requests|pr)[/-]`,
			increment: semver.IncrementStrategyInherit, tag: "PullRequest",
			isMainline: false, isRelease: false, tracksRelease: false,
			preventIncrement: false, trackMergeTarget: false,
			preReleaseWeight: 30000, priority: 40,
		},
		{
			name: "support", regex: `^support[/-]`,
			increment: semver.IncrementStrategyPatch, tag: "",
			isMainline: true, isRelease: false, tracksRelease: false,
			preventIncrement: true, trackMergeTarget: false,
			preReleaseWeight: 55000, priority: 70,
		},
		{
			name: "unknown", regex: `.*`,
			increment: semver.IncrementStrategyInherit, tag: "{BranchName}",
			isMainline: false, isRelease: false, tracksRelease: false,
			preventIncrement: false, trackMergeTarget: false,
			preReleaseWeight: 30000, priority: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := cfg.Branches[tt.name]
			require.NotNil(t, bc, "branch config should exist")

			require.Equal(t, tt.regex, *bc.Regex)
			require.Equal(t, tt.increment, *bc.Increment)
			require.Equal(t, tt.tag, *bc.Tag)
			require.Equal(t, tt.isMainline, *bc.IsMainline)
			require.Equal(t, tt.isRelease, *bc.IsReleaseBranch)
			require.Equal(t, tt.tracksRelease, *bc.TracksReleaseBranches)
			require.Equal(t, tt.preventIncrement, *bc.PreventIncrementOfMergedBranchVersion)
			require.Equal(t, tt.trackMergeTarget, *bc.TrackMergeTarget)
			require.Equal(t, tt.preReleaseWeight, *bc.PreReleaseWeight)
			require.Equal(t, tt.priority, *bc.Priority)
		})
	}
}

func TestCreateDefaultConfiguration_SourceBranches(t *testing.T) {
	cfg := CreateDefaultConfiguration()

	tests := []struct {
		name    string
		sources []string
	}{
		{"main", []string{"develop", "release"}},
		{"develop", []string{}},
		{"release", []string{"develop", "main", "support", "release"}},
		{"feature", []string{"develop", "main", "release", "feature", "support", "hotfix"}},
		{"hotfix", []string{"release", "main", "support", "hotfix"}},
		{"pull-request", []string{"develop", "main", "release", "feature", "support", "hotfix"}},
		{"support", []string{"main"}},
		{"unknown", []string{"develop", "main", "release", "feature", "support", "hotfix"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := cfg.Branches[tt.name]
			require.NotNil(t, bc.SourceBranches)
			require.Equal(t, tt.sources, *bc.SourceBranches)
		})
	}
}

func TestCreateDefaultConfiguration_PullRequestTagNumberPattern(t *testing.T) {
	cfg := CreateDefaultConfiguration()
	pr := cfg.Branches["pull-request"]
	require.NotNil(t, pr.TagNumberPattern)
	require.Equal(t, `[/-](?<number>\d+)`, *pr.TagNumberPattern)
}

func TestCreateDefaultConfiguration_PriorityOrdering(t *testing.T) {
	cfg := CreateDefaultConfiguration()

	// Verify priority ordering: main > release > hotfix > support > develop > feature > pull-request > unknown
	require.Greater(t, *cfg.Branches["main"].Priority, *cfg.Branches["release"].Priority)
	require.Greater(t, *cfg.Branches["release"].Priority, *cfg.Branches["hotfix"].Priority)
	require.Greater(t, *cfg.Branches["hotfix"].Priority, *cfg.Branches["support"].Priority)
	require.Greater(t, *cfg.Branches["support"].Priority, *cfg.Branches["develop"].Priority)
	require.Greater(t, *cfg.Branches["develop"].Priority, *cfg.Branches["feature"].Priority)
	require.Greater(t, *cfg.Branches["feature"].Priority, *cfg.Branches["pull-request"].Priority)
	require.Greater(t, *cfg.Branches["pull-request"].Priority, *cfg.Branches["unknown"].Priority)
}
