package output

import (
	"go-gitsemver/internal/config"
	"go-gitsemver/internal/semver"
)

// GetVariables computes all output variables for a version, applying mode-specific
// transformations (ContinuousDeployment promotion) and then computing format values.
func GetVariables(
	ver semver.SemanticVersion,
	ec config.EffectiveConfiguration,
) map[string]string {
	// Apply ContinuousDeployment promotion if needed.
	promoted := PromoteCommitsToPreRelease(ver, ec.BranchMode, ec.ContinuousDeploymentFallbackTag)

	cfg := semver.FormatConfig{
		Padding:             ec.LegacySemVerPadding,
		CommitDateFormat:    ec.CommitDateFormat,
		TagPreReleaseWeight: ec.TagPreReleaseWeight,
	}

	return semver.ComputeFormatValues(promoted, cfg)
}
