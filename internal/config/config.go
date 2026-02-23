// Package config provides YAML configuration loading, default branch configs,
// config merging, and effective configuration resolution for gitsemver.
package config

import "go-gitsemver/internal/semver"

// Config is the root configuration for gitsemver. All optional fields are
// pointers to support merge semantics during configuration building.
type Config struct {
	Mode                             *semver.VersioningMode             `yaml:"mode"`
	TagPrefix                        *string                            `yaml:"tag-prefix"`
	BaseVersion                      *string                            `yaml:"base-version"`
	NextVersion                      *string                            `yaml:"next-version"`
	Increment                        *semver.IncrementStrategy          `yaml:"increment"`
	ContinuousDeploymentFallbackTag  *string                            `yaml:"continuous-delivery-fallback-tag"`
	CommitMessageIncrementing        *semver.CommitMessageIncrementMode `yaml:"commit-message-incrementing"`
	CommitMessageConvention          *semver.CommitMessageConvention    `yaml:"commit-message-convention"`
	MajorVersionBumpMessage          *string                            `yaml:"major-version-bump-message"`
	MinorVersionBumpMessage          *string                            `yaml:"minor-version-bump-message"`
	PatchVersionBumpMessage          *string                            `yaml:"patch-version-bump-message"`
	NoBumpMessage                    *string                            `yaml:"no-bump-message"`
	CommitDateFormat                 *string                            `yaml:"commit-date-format"`
	UpdateBuildNumber                *bool                              `yaml:"update-build-number"`
	TagPreReleaseWeight              *int64                             `yaml:"tag-pre-release-weight"`
	LegacySemVerPadding              *int                               `yaml:"legacy-semver-padding"`
	BuildMetaDataPadding             *int                               `yaml:"build-metadata-padding"`
	CommitsSinceVersionSourcePadding *int                               `yaml:"commits-since-version-source-padding"`
	MainlineIncrement                *semver.MainlineIncrementMode      `yaml:"mainline-increment"`
	Branches                         map[string]*BranchConfig           `yaml:"branches"`
	Ignore                           IgnoreConfig                       `yaml:"ignore"`
	MergeMessageFormats              map[string]string                  `yaml:"merge-message-formats"`
}
