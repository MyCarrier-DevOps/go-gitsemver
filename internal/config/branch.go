package config

import "go-gitsemver/internal/semver"

// BranchConfig holds per-branch configuration. All fields are pointers
// to support merge semantics: nil means "not set, inherit from parent".
type BranchConfig struct {
	Regex                                 *string                            `yaml:"regex"`
	Increment                             *semver.IncrementStrategy          `yaml:"increment"`
	Mode                                  *semver.VersioningMode             `yaml:"mode"`
	Tag                                   *string                            `yaml:"tag"`
	SourceBranches                        *[]string                          `yaml:"source-branches"`
	IsSourceBranchFor                     *[]string                          `yaml:"is-source-branch-for"`
	IsMainline                            *bool                              `yaml:"is-mainline"`
	IsReleaseBranch                       *bool                              `yaml:"is-release-branch"`
	TracksReleaseBranches                 *bool                              `yaml:"tracks-release-branches"`
	PreventIncrementOfMergedBranchVersion *bool                              `yaml:"prevent-increment-of-merged-branch-version"`
	TrackMergeTarget                      *bool                              `yaml:"track-merge-target"`
	TagNumberPattern                      *string                            `yaml:"tag-number-pattern"`
	CommitMessageIncrementing             *semver.CommitMessageIncrementMode `yaml:"commit-message-incrementing"`
	PreReleaseWeight                      *int                               `yaml:"pre-release-weight"`
	Priority                              *int                               `yaml:"priority"`
}

// MergeTo copies non-nil fields from bc into target. Used for overlay
// semantics: user config overrides defaults where specified.
func (bc *BranchConfig) MergeTo(target *BranchConfig) {
	if bc == nil || target == nil {
		return
	}
	if bc.Regex != nil {
		target.Regex = bc.Regex
	}
	if bc.Increment != nil {
		target.Increment = bc.Increment
	}
	if bc.Mode != nil {
		target.Mode = bc.Mode
	}
	if bc.Tag != nil {
		target.Tag = bc.Tag
	}
	if bc.SourceBranches != nil {
		target.SourceBranches = bc.SourceBranches
	}
	if bc.IsSourceBranchFor != nil {
		target.IsSourceBranchFor = bc.IsSourceBranchFor
	}
	if bc.IsMainline != nil {
		target.IsMainline = bc.IsMainline
	}
	if bc.IsReleaseBranch != nil {
		target.IsReleaseBranch = bc.IsReleaseBranch
	}
	if bc.TracksReleaseBranches != nil {
		target.TracksReleaseBranches = bc.TracksReleaseBranches
	}
	if bc.PreventIncrementOfMergedBranchVersion != nil {
		target.PreventIncrementOfMergedBranchVersion = bc.PreventIncrementOfMergedBranchVersion
	}
	if bc.TrackMergeTarget != nil {
		target.TrackMergeTarget = bc.TrackMergeTarget
	}
	if bc.TagNumberPattern != nil {
		target.TagNumberPattern = bc.TagNumberPattern
	}
	if bc.CommitMessageIncrementing != nil {
		target.CommitMessageIncrementing = bc.CommitMessageIncrementing
	}
	if bc.PreReleaseWeight != nil {
		target.PreReleaseWeight = bc.PreReleaseWeight
	}
	if bc.Priority != nil {
		target.Priority = bc.Priority
	}
}
