package output

import (
	"testing"

	"github.com/MyCarrier-DevOps/go-gitsemver/internal/semver"

	"github.com/stretchr/testify/require"
)

func TestPromote_ContinuousDeployment(t *testing.T) {
	count := int64(5)
	ver := semver.SemanticVersion{
		Major: 1, Minor: 2,
		BuildMetaData: semver.BuildMetaData{CommitsSinceTag: &count},
	}

	result := PromoteCommitsToPreRelease(ver, semver.VersioningModeContinuousDeployment, "ci")
	require.Equal(t, "ci", result.PreReleaseTag.Name)
	require.NotNil(t, result.PreReleaseTag.Number)
	require.Equal(t, int64(5), *result.PreReleaseTag.Number)
}

func TestPromote_ContinuousDeliveryUnchanged(t *testing.T) {
	count := int64(5)
	ver := semver.SemanticVersion{
		Major: 1, Minor: 2,
		BuildMetaData: semver.BuildMetaData{CommitsSinceTag: &count},
	}

	result := PromoteCommitsToPreRelease(ver, semver.VersioningModeContinuousDelivery, "ci")
	require.False(t, result.PreReleaseTag.HasTag())
}

func TestPromote_MainlineUnchanged(t *testing.T) {
	count := int64(5)
	ver := semver.SemanticVersion{
		Major:         1,
		BuildMetaData: semver.BuildMetaData{CommitsSinceTag: &count},
	}

	result := PromoteCommitsToPreRelease(ver, semver.VersioningModeMainline, "ci")
	require.False(t, result.PreReleaseTag.HasTag())
}

func TestPromote_AlreadyHasPreRelease(t *testing.T) {
	count := int64(5)
	num := int64(3)
	ver := semver.SemanticVersion{
		Major:         1,
		PreReleaseTag: semver.PreReleaseTag{Name: "beta", Number: &num},
		BuildMetaData: semver.BuildMetaData{CommitsSinceTag: &count},
	}

	result := PromoteCommitsToPreRelease(ver, semver.VersioningModeContinuousDeployment, "ci")
	// Already has a numbered pre-release, should not change.
	require.Equal(t, "beta", result.PreReleaseTag.Name)
	require.Equal(t, int64(3), *result.PreReleaseTag.Number)
}

func TestPromote_UsesExistingTagName(t *testing.T) {
	count := int64(2)
	ver := semver.SemanticVersion{
		Major:         1,
		PreReleaseTag: semver.PreReleaseTag{Name: "dev"},
		BuildMetaData: semver.BuildMetaData{CommitsSinceTag: &count},
	}

	result := PromoteCommitsToPreRelease(ver, semver.VersioningModeContinuousDeployment, "ci")
	require.Equal(t, "dev", result.PreReleaseTag.Name)
	require.NotNil(t, result.PreReleaseTag.Number)
	require.Equal(t, int64(2), *result.PreReleaseTag.Number)
}

func TestPromote_FallbackDefault(t *testing.T) {
	count := int64(1)
	ver := semver.SemanticVersion{
		Major:         1,
		BuildMetaData: semver.BuildMetaData{CommitsSinceTag: &count},
	}

	result := PromoteCommitsToPreRelease(ver, semver.VersioningModeContinuousDeployment, "")
	require.Equal(t, "ci", result.PreReleaseTag.Name)
}

func TestPromote_ZeroCommits(t *testing.T) {
	count := int64(0)
	ver := semver.SemanticVersion{
		Major:         1,
		BuildMetaData: semver.BuildMetaData{CommitsSinceTag: &count},
	}

	result := PromoteCommitsToPreRelease(ver, semver.VersioningModeContinuousDeployment, "ci")
	require.Equal(t, "ci", result.PreReleaseTag.Name)
	require.NotNil(t, result.PreReleaseTag.Number)
	require.Equal(t, int64(0), *result.PreReleaseTag.Number)
}
