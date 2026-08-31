package config

import (
	"testing"
	"time"

	"github.com/MyCarrier-DevOps/go-gitsemver/internal/semver"

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

func TestBuilder_MergeAllGlobalFields(t *testing.T) {
	mainlineIncr := semver.MainlineIncrementEachCommit
	commitConv := semver.CommitMessageConventionConventionalCommits
	commitIncr := semver.CommitMessageIncrementDisabled
	override := &Config{
		NextVersion:                      stringPtr("2.0.0"),
		Increment:                        incrementPtr(semver.IncrementStrategyMajor),
		ContinuousDeploymentFallbackTag:  stringPtr("beta"),
		CommitMessageIncrementing:        &commitIncr,
		CommitMessageConvention:          &commitConv,
		MajorVersionBumpMessage:          stringPtr("BREAKING:"),
		MinorVersionBumpMessage:          stringPtr("FEATURE:"),
		PatchVersionBumpMessage:          stringPtr("FIX:"),
		NoBumpMessage:                    stringPtr("SKIP:"),
		CommitDateFormat:                 stringPtr("20060102"),
		UpdateBuildNumber:                boolPtr(false),
		TagPreReleaseWeight:              int64Ptr(50000),
		LegacySemVerPadding:              intPtr(5),
		BuildMetaDataPadding:             intPtr(6),
		CommitsSinceVersionSourcePadding: intPtr(7),
		MainlineIncrement:                &mainlineIncr,
	}

	cfg, err := NewBuilder().Add(override).Build()
	require.NoError(t, err)

	require.Equal(t, "2.0.0", *cfg.NextVersion)
	require.Equal(t, semver.IncrementStrategyMajor, *cfg.Increment)
	require.Equal(t, "beta", *cfg.ContinuousDeploymentFallbackTag)
	require.Equal(t, semver.CommitMessageIncrementDisabled, *cfg.CommitMessageIncrementing)
	require.Equal(t, semver.CommitMessageConventionConventionalCommits, *cfg.CommitMessageConvention)
	require.Equal(t, "BREAKING:", *cfg.MajorVersionBumpMessage)
	require.Equal(t, "FEATURE:", *cfg.MinorVersionBumpMessage)
	require.Equal(t, "FIX:", *cfg.PatchVersionBumpMessage)
	require.Equal(t, "SKIP:", *cfg.NoBumpMessage)
	require.Equal(t, "20060102", *cfg.CommitDateFormat)
	require.Equal(t, false, *cfg.UpdateBuildNumber)
	require.Equal(t, int64(50000), *cfg.TagPreReleaseWeight)
	require.Equal(t, 5, *cfg.LegacySemVerPadding)
	require.Equal(t, 6, *cfg.BuildMetaDataPadding)
	require.Equal(t, 7, *cfg.CommitsSinceVersionSourcePadding)
	require.Equal(t, semver.MainlineIncrementEachCommit, *cfg.MainlineIncrement)
}

func TestBuilder_Validate_MissingBranchRegex(t *testing.T) {
	override := &Config{
		Branches: map[string]*BranchConfig{
			"bad": {
				Priority: intPtr(50),
			},
		},
	}

	_, err := NewBuilder().Add(override).Build()
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing regex")
}

func TestBuilder_IsSourceBranchFor_DuplicateIgnored(t *testing.T) {
	override := &Config{
		Branches: map[string]*BranchConfig{
			"develop": {
				IsSourceBranchFor: strSlicePtr([]string{"feature"}),
			},
		},
	}

	cfg, err := NewBuilder().Add(override).Build()
	require.NoError(t, err)

	featureSources := *cfg.Branches["feature"].SourceBranches
	count := 0
	for _, s := range featureSources {
		if s == "develop" {
			count++
		}
	}
	require.Equal(t, 1, count)
}

func TestNewEffectiveConfiguration_MainlineIncrement(t *testing.T) {
	mainlineIncr := semver.MainlineIncrementEachCommit
	cfg := &Config{
		MainlineIncrement: &mainlineIncr,
	}

	ec := NewEffectiveConfiguration(cfg, nil)
	require.Equal(t, semver.MainlineIncrementEachCommit, ec.MainlineIncrement)
}

func TestBranchConfig_MergeTo_AllFields(t *testing.T) {
	src := &BranchConfig{
		Regex:                                 stringPtr("^test$"),
		Increment:                             incrementPtr(semver.IncrementStrategyMajor),
		Mode:                                  versioningModePtr(semver.VersioningModeMainline),
		Tag:                                   stringPtr("rc"),
		SourceBranches:                        strSlicePtr([]string{"main"}),
		IsSourceBranchFor:                     strSlicePtr([]string{"feature"}),
		IsMainline:                            boolPtr(true),
		IsReleaseBranch:                       boolPtr(true),
		TracksReleaseBranches:                 boolPtr(true),
		PreventIncrementOfMergedBranchVersion: boolPtr(true),
		TrackMergeTarget:                      boolPtr(true),
		TagNumberPattern:                      stringPtr(`\d+`),
		CommitMessageIncrementing:             commitMsgIncrPtr(semver.CommitMessageIncrementDisabled),
		PreReleaseWeight:                      intPtr(100),
		Priority:                              intPtr(200),
	}
	target := &BranchConfig{}
	src.MergeTo(target)

	require.Equal(t, "^test$", *target.Regex)
	require.Equal(t, semver.IncrementStrategyMajor, *target.Increment)
	require.Equal(t, semver.VersioningModeMainline, *target.Mode)
	require.Equal(t, "rc", *target.Tag)
	require.Equal(t, []string{"main"}, *target.SourceBranches)
	require.Equal(t, []string{"feature"}, *target.IsSourceBranchFor)
	require.True(t, *target.IsMainline)
	require.True(t, *target.IsReleaseBranch)
	require.True(t, *target.TracksReleaseBranches)
	require.True(t, *target.PreventIncrementOfMergedBranchVersion)
	require.True(t, *target.TrackMergeTarget)
	require.Equal(t, `\d+`, *target.TagNumberPattern)
	require.Equal(t, semver.CommitMessageIncrementDisabled, *target.CommitMessageIncrementing)
	require.Equal(t, 100, *target.PreReleaseWeight)
	require.Equal(t, 200, *target.Priority)
}

func TestBranchConfig_MergeTo_NilSafe(t *testing.T) {
	var src *BranchConfig
	target := &BranchConfig{Regex: stringPtr("^main$")}
	src.MergeTo(target)
	require.Equal(t, "^main$", *target.Regex)

	src = &BranchConfig{Regex: stringPtr("^new$")}
	src.MergeTo(nil)
}

// TestMergeConfig_BaseVersionOverride pins that a set base-version is copied
// over an existing one, and that an unset one leaves the existing value alone.
func TestMergeConfig_BaseVersionOverride(t *testing.T) {
	base := CreateDefaultConfiguration()
	base.BaseVersion = stringPtr("1.0.0")

	overridden, err := NewBuilder().Add(base).Add(&Config{BaseVersion: stringPtr("2.5.0")}).Build()
	require.NoError(t, err)
	require.Equal(t, "2.5.0", *overridden.BaseVersion)

	untouched, err := NewBuilder().Add(base).Add(&Config{}).Build()
	require.NoError(t, err)
	require.Equal(t, "1.0.0", *untouched.BaseVersion)
}

// TestMergeConfig_IgnoreCommitsBefore pins the same for ignore.commits-before.
func TestMergeConfig_IgnoreCommitsBefore(t *testing.T) {
	when := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

	withIgnore, err := NewBuilder().
		Add(CreateDefaultConfiguration()).
		Add(&Config{Ignore: IgnoreConfig{CommitsBefore: &when}}).
		Build()
	require.NoError(t, err)
	require.NotNil(t, withIgnore.Ignore.CommitsBefore)
	require.Equal(t, when, *withIgnore.Ignore.CommitsBefore)

	withoutIgnore, err := NewBuilder().Add(CreateDefaultConfiguration()).Add(&Config{}).Build()
	require.NoError(t, err)
	require.Nil(t, withoutIgnore.Ignore.CommitsBefore)
}

// TestFinalizeBranches_IncrementInheritance pins that a branch without an
// increment inherits the global one, while a branch that sets its own keeps it.
func TestFinalizeBranches_IncrementInheritance(t *testing.T) {
	cfg := &Config{
		Increment: incrementPtr(semver.IncrementStrategyMajor),
		Branches: map[string]*BranchConfig{
			"inherits": {Regex: stringPtr("^inherits$")},
			"explicit": {Regex: stringPtr("^explicit$"), Increment: incrementPtr(semver.IncrementStrategyPatch)},
		},
	}

	finalizeBranches(cfg)

	require.NotNil(t, cfg.Branches["inherits"].Increment)
	require.Equal(t, semver.IncrementStrategyMajor, *cfg.Branches["inherits"].Increment,
		"a branch with no increment must inherit the global one")
	require.Equal(t, semver.IncrementStrategyPatch, *cfg.Branches["explicit"].Increment,
		"a branch with its own increment must keep it")
}

// TestFinalizeBranches_IsSourceBranchForInitialisesNilSlice pins that a target
// whose source-branches is nil gets one created rather than being skipped.
func TestFinalizeBranches_IsSourceBranchForInitialisesNilSlice(t *testing.T) {
	cfg := &Config{
		Branches: map[string]*BranchConfig{
			"feature": {Regex: stringPtr("^feature$"), IsSourceBranchFor: strSlicePtr([]string{"target", "seeded"})},
			"target":  {Regex: stringPtr("^target$")},
			"seeded":  {Regex: stringPtr("^seeded$"), SourceBranches: strSlicePtr([]string{"main"})},
		},
	}

	finalizeBranches(cfg)

	require.NotNil(t, cfg.Branches["target"].SourceBranches)
	require.Equal(t, []string{"feature"}, *cfg.Branches["target"].SourceBranches)
	require.Equal(t, []string{"main", "feature"}, *cfg.Branches["seeded"].SourceBranches)
}
