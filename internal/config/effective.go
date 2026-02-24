package config

import (
	"time"

	"github.com/MyCarrier-DevOps/go-gitsemver/internal/semver"
)

// EffectiveConfiguration is a fully resolved configuration with all fields
// guaranteed to have values. Created from a Config and a specific BranchConfig.
type EffectiveConfiguration struct {
	// Global fields
	Mode                             semver.VersioningMode
	TagPrefix                        string
	BaseVersion                      string
	NextVersion                      string
	Increment                        semver.IncrementStrategy
	ContinuousDeploymentFallbackTag  string
	CommitMessageIncrementing        semver.CommitMessageIncrementMode
	CommitMessageConvention          semver.CommitMessageConvention
	MajorVersionBumpMessage          string
	MinorVersionBumpMessage          string
	PatchVersionBumpMessage          string
	NoBumpMessage                    string
	CommitDateFormat                 string
	UpdateBuildNumber                bool
	TagPreReleaseWeight              int64
	LegacySemVerPadding              int
	BuildMetaDataPadding             int
	CommitsSinceVersionSourcePadding int
	MainlineIncrement                semver.MainlineIncrementMode

	// Branch-specific fields
	BranchRegex                           string
	BranchIncrement                       semver.IncrementStrategy
	BranchMode                            semver.VersioningMode
	Tag                                   string
	SourceBranches                        []string
	IsMainline                            bool
	IsReleaseBranch                       bool
	TracksReleaseBranches                 bool
	PreventIncrementOfMergedBranchVersion bool
	TrackMergeTarget                      bool
	TagNumberPattern                      string
	BranchCommitMessageIncrementing       semver.CommitMessageIncrementMode
	PreReleaseWeight                      int
	Priority                              int

	// Ignore config
	IgnoreCommitsBefore *time.Time
	IgnoreSha           []string
	MergeMessageFormats map[string]string
}

// NewEffectiveConfiguration creates an EffectiveConfiguration by resolving
// all pointer fields from the given Config and BranchConfig to concrete values.
func NewEffectiveConfiguration(cfg *Config, branch *BranchConfig) EffectiveConfiguration {
	ec := EffectiveConfiguration{
		// Global fields with defaults
		Mode:                             derefVersioningMode(cfg.Mode, semver.VersioningModeContinuousDelivery),
		TagPrefix:                        derefString(cfg.TagPrefix, "[vV]"),
		BaseVersion:                      derefString(cfg.BaseVersion, "1.0.0"),
		NextVersion:                      derefString(cfg.NextVersion, ""),
		Increment:                        derefIncrementStrategy(cfg.Increment, semver.IncrementStrategyInherit),
		ContinuousDeploymentFallbackTag:  derefString(cfg.ContinuousDeploymentFallbackTag, "ci"),
		CommitMessageIncrementing:        derefCommitMsgIncr(cfg.CommitMessageIncrementing, semver.CommitMessageIncrementEnabled),
		CommitMessageConvention:          derefCommitMsgConv(cfg.CommitMessageConvention, semver.CommitMessageConventionBoth),
		MajorVersionBumpMessage:          derefString(cfg.MajorVersionBumpMessage, `\+semver:\s?(breaking|major)`),
		MinorVersionBumpMessage:          derefString(cfg.MinorVersionBumpMessage, `\+semver:\s?(feature|minor)`),
		PatchVersionBumpMessage:          derefString(cfg.PatchVersionBumpMessage, `\+semver:\s?(fix|patch)`),
		NoBumpMessage:                    derefString(cfg.NoBumpMessage, `\+semver:\s?(none|skip)`),
		CommitDateFormat:                 derefString(cfg.CommitDateFormat, "2006-01-02"),
		UpdateBuildNumber:                derefBool(cfg.UpdateBuildNumber, true),
		TagPreReleaseWeight:              derefInt64(cfg.TagPreReleaseWeight, 60000),
		LegacySemVerPadding:              derefInt(cfg.LegacySemVerPadding, 4),
		BuildMetaDataPadding:             derefInt(cfg.BuildMetaDataPadding, 4),
		CommitsSinceVersionSourcePadding: derefInt(cfg.CommitsSinceVersionSourcePadding, 4),
		MainlineIncrement:                derefMainlineIncrementMode(cfg.MainlineIncrement, semver.MainlineIncrementAggregate),

		// Ignore config
		IgnoreCommitsBefore: cfg.Ignore.CommitsBefore,
		IgnoreSha:           cfg.Ignore.Sha,
		MergeMessageFormats: cfg.MergeMessageFormats,
	}

	// Branch-specific fields
	if branch != nil {
		ec.BranchRegex = derefString(branch.Regex, "")
		ec.BranchIncrement = derefIncrementStrategy(branch.Increment, ec.Increment)
		ec.BranchMode = derefVersioningMode(branch.Mode, ec.Mode)
		ec.Tag = derefString(branch.Tag, "{BranchName}")
		if branch.SourceBranches != nil {
			ec.SourceBranches = *branch.SourceBranches
		}
		ec.IsMainline = derefBool(branch.IsMainline, false)
		ec.IsReleaseBranch = derefBool(branch.IsReleaseBranch, false)
		ec.TracksReleaseBranches = derefBool(branch.TracksReleaseBranches, false)
		ec.PreventIncrementOfMergedBranchVersion = derefBool(branch.PreventIncrementOfMergedBranchVersion, false)
		ec.TrackMergeTarget = derefBool(branch.TrackMergeTarget, false)
		ec.TagNumberPattern = derefString(branch.TagNumberPattern, "")
		ec.BranchCommitMessageIncrementing = derefCommitMsgIncr(branch.CommitMessageIncrementing, ec.CommitMessageIncrementing)
		ec.PreReleaseWeight = derefInt(branch.PreReleaseWeight, 0)
		ec.Priority = derefInt(branch.Priority, 0)
	}

	return ec
}

func derefString(p *string, fallback string) string {
	if p != nil {
		return *p
	}
	return fallback
}

func derefBool(p *bool, fallback bool) bool {
	if p != nil {
		return *p
	}
	return fallback
}

func derefInt(p *int, fallback int) int {
	if p != nil {
		return *p
	}
	return fallback
}

func derefInt64(p *int64, fallback int64) int64 {
	if p != nil {
		return *p
	}
	return fallback
}

func derefVersioningMode(p *semver.VersioningMode, fallback semver.VersioningMode) semver.VersioningMode {
	if p != nil {
		return *p
	}
	return fallback
}

func derefIncrementStrategy(p *semver.IncrementStrategy, fallback semver.IncrementStrategy) semver.IncrementStrategy {
	if p != nil {
		return *p
	}
	return fallback
}

func derefCommitMsgIncr(p *semver.CommitMessageIncrementMode, fallback semver.CommitMessageIncrementMode) semver.CommitMessageIncrementMode {
	if p != nil {
		return *p
	}
	return fallback
}

func derefCommitMsgConv(p *semver.CommitMessageConvention, fallback semver.CommitMessageConvention) semver.CommitMessageConvention {
	if p != nil {
		return *p
	}
	return fallback
}

func derefMainlineIncrementMode(p *semver.MainlineIncrementMode, fallback semver.MainlineIncrementMode) semver.MainlineIncrementMode {
	if p != nil {
		return *p
	}
	return fallback
}
