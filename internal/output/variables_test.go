package output

import (
	"testing"

	"github.com/MyCarrier-DevOps/go-gitsemver/internal/config"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/semver"

	"github.com/stretchr/testify/require"
)

func TestGetVariables_Basic(t *testing.T) {
	count := int64(3)
	ver := semver.SemanticVersion{
		Major: 1, Minor: 2, Patch: 3,
		BuildMetaData: semver.BuildMetaData{
			CommitsSinceTag: &count,
			Branch:          "main",
			Sha:             "abc1234567890",
			ShortSha:        "abc1234",
		},
	}
	ec := config.EffectiveConfiguration{
		LegacySemVerPadding: 4,
		CommitDateFormat:    "2006-01-02",
		TagPreReleaseWeight: 60000,
	}

	vars := GetVariables(ver, ec)
	require.Equal(t, "1", vars["Major"])
	require.Equal(t, "2", vars["Minor"])
	require.Equal(t, "3", vars["Patch"])
	require.Equal(t, "1.2.3", vars["SemVer"])
	require.Equal(t, "1.2.3+3", vars["FullSemVer"])
	require.Equal(t, "main", vars["BranchName"])
}

func TestGetVariables_CDPromotion(t *testing.T) {
	count := int64(5)
	ver := semver.SemanticVersion{
		Major: 1, Minor: 2,
		BuildMetaData: semver.BuildMetaData{CommitsSinceTag: &count},
	}
	ec := config.EffectiveConfiguration{
		BranchMode:                      semver.VersioningModeContinuousDeployment,
		ContinuousDeploymentFallbackTag: "ci",
		LegacySemVerPadding:             4,
		CommitDateFormat:                "2006-01-02",
		TagPreReleaseWeight:             60000,
	}

	vars := GetVariables(ver, ec)
	// CD mode should promote commits to pre-release.
	require.Equal(t, "1.2.0-ci.5", vars["SemVer"])
}
