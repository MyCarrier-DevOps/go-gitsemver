package config

import (
	"fmt"
	"go-gitsemver/internal/semver"
	"regexp"
)

// Builder constructs a Config by layering overrides on top of defaults.
type Builder struct {
	overrides []*Config
}

// NewBuilder creates a new configuration builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// Add adds a configuration override. Overrides are applied in order:
// later overrides take precedence over earlier ones.
func (b *Builder) Add(override *Config) *Builder {
	if override != nil {
		b.overrides = append(b.overrides, override)
	}
	return b
}

// Build constructs the final configuration by starting with defaults,
// applying all overrides, finalizing branch configs, and validating.
func (b *Builder) Build() (*Config, error) {
	cfg := CreateDefaultConfiguration()

	for _, override := range b.overrides {
		mergeConfig(cfg, override)
	}

	finalizeBranches(cfg)

	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// mergeConfig applies non-nil fields from src to dst.
func mergeConfig(dst, src *Config) {
	if src.Mode != nil {
		dst.Mode = src.Mode
	}
	if src.TagPrefix != nil {
		dst.TagPrefix = src.TagPrefix
	}
	if src.BaseVersion != nil {
		dst.BaseVersion = src.BaseVersion
	}
	if src.NextVersion != nil {
		dst.NextVersion = src.NextVersion
	}
	if src.Increment != nil {
		dst.Increment = src.Increment
	}
	if src.ContinuousDeploymentFallbackTag != nil {
		dst.ContinuousDeploymentFallbackTag = src.ContinuousDeploymentFallbackTag
	}
	if src.CommitMessageIncrementing != nil {
		dst.CommitMessageIncrementing = src.CommitMessageIncrementing
	}
	if src.CommitMessageConvention != nil {
		dst.CommitMessageConvention = src.CommitMessageConvention
	}
	if src.MajorVersionBumpMessage != nil {
		dst.MajorVersionBumpMessage = src.MajorVersionBumpMessage
	}
	if src.MinorVersionBumpMessage != nil {
		dst.MinorVersionBumpMessage = src.MinorVersionBumpMessage
	}
	if src.PatchVersionBumpMessage != nil {
		dst.PatchVersionBumpMessage = src.PatchVersionBumpMessage
	}
	if src.NoBumpMessage != nil {
		dst.NoBumpMessage = src.NoBumpMessage
	}
	if src.CommitDateFormat != nil {
		dst.CommitDateFormat = src.CommitDateFormat
	}
	if src.UpdateBuildNumber != nil {
		dst.UpdateBuildNumber = src.UpdateBuildNumber
	}
	if src.TagPreReleaseWeight != nil {
		dst.TagPreReleaseWeight = src.TagPreReleaseWeight
	}
	if src.LegacySemVerPadding != nil {
		dst.LegacySemVerPadding = src.LegacySemVerPadding
	}
	if src.BuildMetaDataPadding != nil {
		dst.BuildMetaDataPadding = src.BuildMetaDataPadding
	}
	if src.CommitsSinceVersionSourcePadding != nil {
		dst.CommitsSinceVersionSourcePadding = src.CommitsSinceVersionSourcePadding
	}
	if src.MainlineIncrement != nil {
		dst.MainlineIncrement = src.MainlineIncrement
	}

	// Branch configs: merge per-key
	if src.Branches != nil {
		if dst.Branches == nil {
			dst.Branches = make(map[string]*BranchConfig)
		}
		for name, srcBranch := range src.Branches {
			if dstBranch, ok := dst.Branches[name]; ok {
				srcBranch.MergeTo(dstBranch)
			} else {
				dst.Branches[name] = srcBranch
			}
		}
	}

	// Merge message formats: merge maps
	if src.MergeMessageFormats != nil {
		if dst.MergeMessageFormats == nil {
			dst.MergeMessageFormats = make(map[string]string)
		}
		for k, v := range src.MergeMessageFormats {
			dst.MergeMessageFormats[k] = v
		}
	}

	// Ignore config
	if src.Ignore.CommitsBefore != nil {
		dst.Ignore.CommitsBefore = src.Ignore.CommitsBefore
	}
	if src.Ignore.Sha != nil {
		dst.Ignore.Sha = src.Ignore.Sha
	}
}

// finalizeBranches applies global config inheritance and the develop
// special-case VersioningMode logic.
func finalizeBranches(cfg *Config) {
	for name, branch := range cfg.Branches {
		// Inherit increment from global if not set
		if branch.Increment == nil && cfg.Increment != nil {
			inc := *cfg.Increment
			branch.Increment = &inc
		}

		// Inherit mode from global if not set
		if branch.Mode == nil && cfg.Mode != nil {
			if name == "develop" {
				// Special case: develop gets ContinuousDeployment unless global is Mainline
				if *cfg.Mode == semver.VersioningModeMainline {
					m := semver.VersioningModeMainline
					branch.Mode = &m
				} else {
					m := semver.VersioningModeContinuousDeployment
					branch.Mode = &m
				}
			} else {
				m := *cfg.Mode
				branch.Mode = &m
			}
		}

		// Inherit CommitMessageIncrementing from global if not set
		if branch.CommitMessageIncrementing == nil && cfg.CommitMessageIncrementing != nil {
			cmi := *cfg.CommitMessageIncrementing
			branch.CommitMessageIncrementing = &cmi
		}
	}

	// Process is-source-branch-for (inverse source-branches)
	for name, branch := range cfg.Branches {
		if branch.IsSourceBranchFor == nil {
			continue
		}
		for _, targetName := range *branch.IsSourceBranchFor {
			target, ok := cfg.Branches[targetName]
			if !ok {
				continue
			}
			if target.SourceBranches == nil {
				empty := []string{}
				target.SourceBranches = &empty
			}
			sources := *target.SourceBranches
			if !sliceContains(sources, name) {
				sources = append(sources, name)
				target.SourceBranches = &sources
			}
		}
	}
}

// validate checks the configuration for errors.
func validate(cfg *Config) error {
	if cfg.TagPrefix != nil {
		if _, err := regexp.Compile(*cfg.TagPrefix); err != nil {
			return fmt.Errorf("invalid tag-prefix regex %q: %w", *cfg.TagPrefix, err)
		}
	}

	for name, branch := range cfg.Branches {
		if branch.Regex == nil {
			return fmt.Errorf("branch %q missing regex", name)
		}
		if _, err := regexp.Compile(*branch.Regex); err != nil {
			return fmt.Errorf("branch %q has invalid regex %q: %w", name, *branch.Regex, err)
		}
	}

	return nil
}

func sliceContains(ss []string, s string) bool {
	for _, item := range ss {
		if item == s {
			return true
		}
	}
	return false
}
