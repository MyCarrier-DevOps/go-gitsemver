package semver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestComputeFormatValues_FullVersion(t *testing.T) {
	ver := SemanticVersion{
		Major: 1,
		Minor: 2,
		Patch: 3,
		PreReleaseTag: PreReleaseTag{
			Name:   "beta",
			Number: int64Ptr(4),
		},
		BuildMetaData: BuildMetaData{
			CommitsSinceTag:           int64Ptr(5),
			Branch:                    "main",
			Sha:                       "abc1234def567890",
			ShortSha:                  "abc1234",
			VersionSourceSha:          "def5678",
			CommitDate:                time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			CommitsSinceVersionSource: 5,
			UncommittedChanges:        0,
		},
	}
	cfg := DefaultFormatConfig()
	vals := ComputeFormatValues(ver, cfg)

	// Version components
	require.Equal(t, "1", vals["Major"])
	require.Equal(t, "2", vals["Minor"])
	require.Equal(t, "3", vals["Patch"])
	require.Equal(t, "1.2.3", vals["MajorMinorPatch"])

	// SemVer formats
	require.Equal(t, "1.2.3-beta.4", vals["SemVer"])
	require.Equal(t, "1.2.3-beta.4+5", vals["FullSemVer"])
	require.Equal(t, "1.2.3-beta4", vals["LegacySemVer"])
	require.Equal(t, "1.2.3-beta0004", vals["LegacySemVerPadded"])
	require.Equal(t, "1.2.3-beta.4+5.Branch.main.Sha.abc1234def567890", vals["InformationalVersion"])

	// Pre-release
	require.Equal(t, "beta.4", vals["PreReleaseTag"])
	require.Equal(t, "-beta.4", vals["PreReleaseTagWithDash"])
	require.Equal(t, "beta", vals["PreReleaseLabel"])
	require.Equal(t, "-beta", vals["PreReleaseLabelWithDash"])
	require.Equal(t, "4", vals["PreReleaseNumber"])
	require.Equal(t, "60004", vals["WeightedPreReleaseNumber"])

	// Build metadata
	require.Equal(t, "5", vals["BuildMetaData"])
	require.Equal(t, "0005", vals["BuildMetaDataPadded"])
	require.Equal(t, "5.Branch.main.Sha.abc1234def567890", vals["FullBuildMetaData"])

	// Git information
	require.Equal(t, "main", vals["BranchName"])
	require.Equal(t, "main", vals["EscapedBranchName"])
	require.Equal(t, "abc1234def567890", vals["Sha"])
	require.Equal(t, "abc1234", vals["ShortSha"])

	// Commit tracking
	require.Equal(t, "def5678", vals["VersionSourceSha"])
	require.Equal(t, "5", vals["CommitsSinceVersionSource"])
	require.Equal(t, "0005", vals["CommitsSinceVersionSourcePadded"])
	require.Equal(t, "0", vals["UncommittedChanges"])
	require.Equal(t, "2025-01-15", vals["CommitDate"])
	require.Equal(t, "25.03.abc1234", vals["CommitTag"])

	// Assembly info
	require.Equal(t, "1.2.0.0", vals["AssemblySemVer"])
	require.Equal(t, "1.2.3.0", vals["AssemblySemFileVer"])
	require.Equal(t, vals["InformationalVersion"], vals["AssemblyInformationalVersion"])

	// NuGet
	require.Equal(t, "1.2.3-beta0004", vals["NuGetVersionV2"])
	require.Equal(t, "1.2.3-beta0004", vals["NuGetVersion"])
	require.Equal(t, "beta0004", vals["NuGetPreReleaseTagV2"])
	require.Equal(t, "beta0004", vals["NuGetPreReleaseTag"])
}

func TestComputeFormatValues_StableVersion(t *testing.T) {
	ver := SemanticVersion{
		Major: 2,
		Minor: 0,
		Patch: 0,
		BuildMetaData: BuildMetaData{
			CommitsSinceTag:           int64Ptr(0),
			Branch:                    "main",
			Sha:                       "deadbeef12345678",
			ShortSha:                  "deadbee",
			CommitDate:                time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
			CommitsSinceVersionSource: 0,
		},
	}
	cfg := DefaultFormatConfig()
	vals := ComputeFormatValues(ver, cfg)

	require.Equal(t, "2.0.0", vals["SemVer"])
	require.Equal(t, "2.0.0+0", vals["FullSemVer"])
	require.Equal(t, "2.0.0", vals["LegacySemVer"])
	require.Equal(t, "2.0.0", vals["LegacySemVerPadded"])
	require.Equal(t, "", vals["PreReleaseTag"])
	require.Equal(t, "", vals["PreReleaseTagWithDash"])
	require.Equal(t, "", vals["PreReleaseLabel"])
	require.Equal(t, "", vals["PreReleaseLabelWithDash"])
	require.Equal(t, "", vals["PreReleaseNumber"])
	require.Equal(t, "", vals["WeightedPreReleaseNumber"])
	require.Equal(t, "2.0.0.0", vals["AssemblySemVer"])
	require.Equal(t, "2.0.0", vals["NuGetVersionV2"])
	require.Equal(t, "", vals["NuGetPreReleaseTagV2"])
	require.Equal(t, "25.22.deadbee", vals["CommitTag"])
}

func TestComputeFormatValues_EmptyBuildMetadata(t *testing.T) {
	ver := SemanticVersion{Major: 1, Minor: 0, Patch: 0}
	cfg := DefaultFormatConfig()
	vals := ComputeFormatValues(ver, cfg)

	require.Equal(t, "1.0.0", vals["SemVer"])
	require.Equal(t, "1.0.0", vals["FullSemVer"])
	require.Equal(t, "", vals["BuildMetaData"])
	require.Equal(t, "", vals["BuildMetaDataPadded"])
	require.Equal(t, "", vals["FullBuildMetaData"])
	require.Equal(t, "", vals["BranchName"])
	require.Equal(t, "", vals["EscapedBranchName"])
	require.Equal(t, "", vals["Sha"])
	require.Equal(t, "", vals["ShortSha"])
	require.Equal(t, "", vals["CommitDate"])
	require.Equal(t, "", vals["CommitTag"])
}

func TestComputeFormatValues_DefaultConfig(t *testing.T) {
	ver := SemanticVersion{
		Major:         1,
		PreReleaseTag: PreReleaseTag{Name: "beta", Number: int64Ptr(1)},
	}
	// Zero-value config should use defaults
	vals := ComputeFormatValues(ver, FormatConfig{})

	require.Equal(t, "1.0.0-beta0001", vals["LegacySemVerPadded"])
	require.Equal(t, "60001", vals["WeightedPreReleaseNumber"])
}

func TestEscapeBranchName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple", "main", "main"},
		{"with slash", "feature/auth", "feature-auth"},
		{"with dots", "release.1.0", "release-1-0"},
		{"with spaces", "my branch", "my-branch"},
		{"with underscores", "bug_fix_123", "bug-fix-123"},
		{"complex", "feature/user@auth#2", "feature-user-auth-2"},
		{"hyphens preserved", "feature-auth", "feature-auth"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, escapeBranchName(tt.input))
		})
	}
}

func TestDefaultFormatConfig(t *testing.T) {
	cfg := DefaultFormatConfig()
	require.Equal(t, 4, cfg.Padding)
	require.Equal(t, "2006-01-02", cfg.CommitDateFormat)
	require.Equal(t, int64(60000), cfg.TagPreReleaseWeight)
}

func TestComputeFormatValues_CustomWeight(t *testing.T) {
	ver := SemanticVersion{
		Major:         1,
		PreReleaseTag: PreReleaseTag{Name: "beta", Number: int64Ptr(4)},
	}
	cfg := FormatConfig{
		Padding:             4,
		CommitDateFormat:    "2006-01-02",
		TagPreReleaseWeight: 30000,
	}
	vals := ComputeFormatValues(ver, cfg)
	require.Equal(t, "30004", vals["WeightedPreReleaseNumber"])
}

func TestComputeFormatValues_CustomDateFormat(t *testing.T) {
	ver := SemanticVersion{
		Major: 1,
		BuildMetaData: BuildMetaData{
			CommitDate: time.Date(2025, 3, 15, 14, 30, 0, 0, time.UTC),
		},
	}
	cfg := FormatConfig{
		Padding:          4,
		CommitDateFormat: "2006/01/02 15:04",
	}
	vals := ComputeFormatValues(ver, cfg)
	require.Equal(t, "2025/03/15 14:30", vals["CommitDate"])
}

func TestTranslateDateFormat(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{"dotnet yyyy-MM-dd", "yyyy-MM-dd", "2006-01-02"},
		{"dotnet with time", "yyyy-MM-dd HH:mm:ss", "2006-01-02 15:04:05"},
		{"dotnet short date", "yy/MM/dd", "06/01/02"},
		{"dotnet 12h time", "hh:mm:ss tt", "03:04:05 PM"},
		{"dotnet milliseconds", "yyyy-MM-dd HH:mm:ss.fff", "2006-01-02 15:04:05.000"},
		{"go format passthrough", "2006-01-02", "2006-01-02"},
		{"go format with time", "2006-01-02 15:04:05", "2006-01-02 15:04:05"},
		{"go time-only passthrough", "15:04:05", "15:04:05"},
		{"month name long", "dd MMMM yyyy", "02 January 2006"},
		{"month name short", "dd MMM yyyy", "02 Jan 2006"},
		{"single letter month/day", "M/d/yyyy", "1/2/2006"},
		{"single letter time", "H:m:s", "15:4:5"},
		{"single letter 12h", "h:m:s tt", "3:4:5 PM"},
		{"day of week full", "dddd, dd MMMM yyyy", "Monday, 02 January 2006"},
		{"day of week short", "ddd, dd MMM yyyy", "Mon, 02 Jan 2006"},
		{"day of week with single day", "dddd, MMMM d, yyyy", "Monday, January 2, 2006"},
		{"timezone full offset", "yyyy-MM-dd HH:mm:ss zzz", "2006-01-02 15:04:05 -07:00"},
		{"timezone hours", "yyyy-MM-dd zz", "2006-01-02 -07"},
		{"iso8601 with T", "yyyy-MM-ddTHH:mm:ss", "2006-01-02T15:04:05"},
		{"empty string", "", ""},
		{"literal only", "T", "T"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expect, translateDateFormat(tt.input))
		})
	}
}

func TestComputeFormatValues_DotNetDateFormat(t *testing.T) {
	ver := SemanticVersion{
		Major: 1,
		BuildMetaData: BuildMetaData{
			CommitDate: time.Date(2025, 3, 15, 14, 30, 0, 0, time.UTC),
		},
	}
	cfg := FormatConfig{
		Padding:          4,
		CommitDateFormat: "yyyy-MM-dd",
	}
	vals := ComputeFormatValues(ver, cfg)
	require.Equal(t, "2025-03-15", vals["CommitDate"])
}

func TestComputeFormatValues_VariableCount(t *testing.T) {
	ver := SemanticVersion{
		Major:         1,
		Minor:         2,
		Patch:         3,
		PreReleaseTag: PreReleaseTag{Name: "beta", Number: int64Ptr(4)},
		BuildMetaData: BuildMetaData{
			CommitsSinceTag: int64Ptr(5),
			Branch:          "main",
			Sha:             "abc1234",
			ShortSha:        "abc1234",
		},
	}
	cfg := DefaultFormatConfig()
	vals := ComputeFormatValues(ver, cfg)

	// Should have at least 30 variables
	require.GreaterOrEqual(t, len(vals), 30)
}
