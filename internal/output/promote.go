// Package output handles version formatting, promotion, and serialization.
package output

import (
	"go-gitsemver/internal/semver"
)

// PromoteCommitsToPreRelease transforms a version for ContinuousDeployment
// or Mainline modes by setting CommitsSinceTag as the pre-release number.
// This is a pure function — no side effects.
//
// In ContinuousDeployment mode, commit count becomes the pre-release number:
//
//	1.2.0+5 → 1.2.0-ci.5 (using fallback tag "ci")
//
// In ContinuousDelivery mode, the version is returned unchanged.
func PromoteCommitsToPreRelease(
	ver semver.SemanticVersion,
	mode semver.VersioningMode,
	fallbackTag string,
) semver.SemanticVersion {
	if mode != semver.VersioningModeContinuousDeployment {
		return ver
	}

	// If already has a pre-release tag with a number, leave it.
	if ver.PreReleaseTag.HasTag() && ver.PreReleaseTag.Number != nil {
		return ver
	}

	commitsSince := int64(0)
	if ver.BuildMetaData.CommitsSinceTag != nil {
		commitsSince = *ver.BuildMetaData.CommitsSinceTag
	}

	tagName := ver.PreReleaseTag.Name
	if tagName == "" {
		tagName = fallbackTag
		if tagName == "" {
			tagName = "ci"
		}
	}

	return ver.WithPreReleaseTag(semver.PreReleaseTag{
		Name:   tagName,
		Number: &commitsSince,
	})
}
