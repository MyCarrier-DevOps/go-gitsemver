package config

import "github.com/MyCarrier-DevOps/go-gitsemver/internal/semver"

// CreateDefaultConfiguration returns a Config with all default values
// populated. This includes 8 branch configurations: main, develop,
// release, feature, hotfix, pull-request, support, and unknown (catch-all).
func CreateDefaultConfiguration() *Config {
	return &Config{
		Mode:                             versioningModePtr(semver.VersioningModeContinuousDelivery),
		TagPrefix:                        stringPtr("[vV]"),
		BaseVersion:                      stringPtr("0.1.0"),
		Increment:                        incrementPtr(semver.IncrementStrategyInherit),
		ContinuousDeploymentFallbackTag:  stringPtr("ci"),
		CommitMessageIncrementing:        commitMsgIncrPtr(semver.CommitMessageIncrementEnabled),
		CommitMessageConvention:          commitMsgConvPtr(semver.CommitMessageConventionBoth),
		MajorVersionBumpMessage:          stringPtr(`\+semver:\s?(breaking|major)`),
		MinorVersionBumpMessage:          stringPtr(`\+semver:\s?(feature|minor)`),
		PatchVersionBumpMessage:          stringPtr(`\+semver:\s?(fix|patch)`),
		NoBumpMessage:                    stringPtr(`\+semver:\s?(none|skip)`),
		CommitDateFormat:                 stringPtr("2006-01-02"),
		UpdateBuildNumber:                boolPtr(true),
		TagPreReleaseWeight:              int64Ptr(60000),
		LegacySemVerPadding:              intPtr(4),
		BuildMetaDataPadding:             intPtr(4),
		CommitsSinceVersionSourcePadding: intPtr(4),
		Branches:                         createDefaultBranches(),
	}
}

func createDefaultBranches() map[string]*BranchConfig {
	return map[string]*BranchConfig{
		"main":         defaultMain(),
		"develop":      defaultDevelop(),
		"release":      defaultRelease(),
		"feature":      defaultFeature(),
		"hotfix":       defaultHotfix(),
		"pull-request": defaultPullRequest(),
		"support":      defaultSupport(),
		"unknown":      defaultUnknown(),
	}
}

func defaultMain() *BranchConfig {
	return &BranchConfig{
		Regex:                                 stringPtr(`^master$|^main$`),
		Increment:                             incrementPtr(semver.IncrementStrategyPatch),
		Tag:                                   stringPtr(""),
		IsMainline:                            boolPtr(true),
		IsReleaseBranch:                       boolPtr(false),
		TracksReleaseBranches:                 boolPtr(false),
		PreventIncrementOfMergedBranchVersion: boolPtr(true),
		TrackMergeTarget:                      boolPtr(false),
		SourceBranches:                        strSlicePtr([]string{"develop", "release"}),
		PreReleaseWeight:                      intPtr(55000),
		Priority:                              intPtr(100),
	}
}

func defaultDevelop() *BranchConfig {
	return &BranchConfig{
		Regex:                                 stringPtr(`^dev(elop)?(ment)?$`),
		Increment:                             incrementPtr(semver.IncrementStrategyMinor),
		Tag:                                   stringPtr("alpha"),
		IsMainline:                            boolPtr(false),
		IsReleaseBranch:                       boolPtr(false),
		TracksReleaseBranches:                 boolPtr(true),
		PreventIncrementOfMergedBranchVersion: boolPtr(false),
		TrackMergeTarget:                      boolPtr(true),
		SourceBranches:                        strSlicePtr([]string{}),
		PreReleaseWeight:                      intPtr(0),
		Priority:                              intPtr(60),
	}
}

func defaultRelease() *BranchConfig {
	return &BranchConfig{
		Regex:                                 stringPtr(`^releases?[/-]`),
		Increment:                             incrementPtr(semver.IncrementStrategyNone),
		Tag:                                   stringPtr("beta"),
		IsMainline:                            boolPtr(false),
		IsReleaseBranch:                       boolPtr(true),
		TracksReleaseBranches:                 boolPtr(false),
		PreventIncrementOfMergedBranchVersion: boolPtr(true),
		TrackMergeTarget:                      boolPtr(false),
		SourceBranches:                        strSlicePtr([]string{"develop", "main", "support", "release"}),
		PreReleaseWeight:                      intPtr(30000),
		Priority:                              intPtr(90),
	}
}

func defaultFeature() *BranchConfig {
	return &BranchConfig{
		Regex:                                 stringPtr(`^features?[/-]`),
		Increment:                             incrementPtr(semver.IncrementStrategyInherit),
		Tag:                                   stringPtr("{BranchName}"),
		IsMainline:                            boolPtr(false),
		IsReleaseBranch:                       boolPtr(false),
		TracksReleaseBranches:                 boolPtr(false),
		PreventIncrementOfMergedBranchVersion: boolPtr(false),
		TrackMergeTarget:                      boolPtr(false),
		SourceBranches:                        strSlicePtr([]string{"develop", "main", "release", "feature", "support", "hotfix"}),
		PreReleaseWeight:                      intPtr(30000),
		Priority:                              intPtr(50),
	}
}

func defaultHotfix() *BranchConfig {
	return &BranchConfig{
		Regex:                                 stringPtr(`^hotfix(es)?[/-]`),
		Increment:                             incrementPtr(semver.IncrementStrategyPatch),
		Tag:                                   stringPtr("beta"),
		IsMainline:                            boolPtr(false),
		IsReleaseBranch:                       boolPtr(false),
		TracksReleaseBranches:                 boolPtr(false),
		PreventIncrementOfMergedBranchVersion: boolPtr(false),
		TrackMergeTarget:                      boolPtr(false),
		SourceBranches:                        strSlicePtr([]string{"release", "main", "support", "hotfix"}),
		PreReleaseWeight:                      intPtr(30000),
		Priority:                              intPtr(80),
	}
}

func defaultPullRequest() *BranchConfig {
	return &BranchConfig{
		Regex:                                 stringPtr(`^(pull|pull-requests|pr)[/-]`),
		Increment:                             incrementPtr(semver.IncrementStrategyInherit),
		Tag:                                   stringPtr("PullRequest"),
		TagNumberPattern:                      stringPtr(`[/-](?<number>\d+)`),
		IsMainline:                            boolPtr(false),
		IsReleaseBranch:                       boolPtr(false),
		TracksReleaseBranches:                 boolPtr(false),
		PreventIncrementOfMergedBranchVersion: boolPtr(false),
		TrackMergeTarget:                      boolPtr(false),
		SourceBranches:                        strSlicePtr([]string{"develop", "main", "release", "feature", "support", "hotfix"}),
		PreReleaseWeight:                      intPtr(30000),
		Priority:                              intPtr(40),
	}
}

func defaultSupport() *BranchConfig {
	return &BranchConfig{
		Regex:                                 stringPtr(`^support[/-]`),
		Increment:                             incrementPtr(semver.IncrementStrategyPatch),
		Tag:                                   stringPtr(""),
		IsMainline:                            boolPtr(true),
		IsReleaseBranch:                       boolPtr(false),
		TracksReleaseBranches:                 boolPtr(false),
		PreventIncrementOfMergedBranchVersion: boolPtr(true),
		TrackMergeTarget:                      boolPtr(false),
		SourceBranches:                        strSlicePtr([]string{"main"}),
		PreReleaseWeight:                      intPtr(55000),
		Priority:                              intPtr(70),
	}
}

func defaultUnknown() *BranchConfig {
	return &BranchConfig{
		Regex:                                 stringPtr(`.*`),
		Increment:                             incrementPtr(semver.IncrementStrategyInherit),
		Tag:                                   stringPtr("{BranchName}"),
		IsMainline:                            boolPtr(false),
		IsReleaseBranch:                       boolPtr(false),
		TracksReleaseBranches:                 boolPtr(false),
		PreventIncrementOfMergedBranchVersion: boolPtr(false),
		TrackMergeTarget:                      boolPtr(false),
		SourceBranches:                        strSlicePtr([]string{"develop", "main", "release", "feature", "support", "hotfix"}),
		PreReleaseWeight:                      intPtr(30000),
		Priority:                              intPtr(0),
	}
}
