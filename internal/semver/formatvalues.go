package semver

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// FormatConfig holds configuration for computing format values.
type FormatConfig struct {
	// Padding is the number of digits for zero-padded fields (default: 4).
	Padding int
	// CommitDateFormat is the Go time format for commit dates (default: "2006-01-02").
	CommitDateFormat string
	// TagPreReleaseWeight is added to pre-release numbers for weighted sorting (default: 60000).
	TagPreReleaseWeight int64
}

// DefaultFormatConfig returns a FormatConfig with default values.
func DefaultFormatConfig() FormatConfig {
	return FormatConfig{
		Padding:             4,
		CommitDateFormat:    "2006-01-02",
		TagPreReleaseWeight: 60000,
	}
}

var branchNameEscaper = regexp.MustCompile(`[^a-zA-Z0-9-]`)

func escapeBranchName(name string) string {
	return branchNameEscaper.ReplaceAllString(name, "-")
}

// ComputeFormatValues computes all output variable strings from a semantic version.
// This is a pure function with no side effects.
func ComputeFormatValues(ver SemanticVersion, cfg FormatConfig) map[string]string {
	if cfg.Padding <= 0 {
		cfg.Padding = 4
	}
	if cfg.CommitDateFormat == "" {
		cfg.CommitDateFormat = "2006-01-02"
	}
	if cfg.TagPreReleaseWeight == 0 {
		cfg.TagPreReleaseWeight = 60000
	}
	pad := cfg.Padding

	vals := make(map[string]string, 35)

	// Version components
	majorStr := strconv.FormatInt(ver.Major, 10)
	minorStr := strconv.FormatInt(ver.Minor, 10)
	patchStr := strconv.FormatInt(ver.Patch, 10)
	vals["Major"] = majorStr
	vals["Minor"] = minorStr
	vals["Patch"] = patchStr
	vals["MajorMinorPatch"] = majorStr + "." + minorStr + "." + patchStr

	// SemVer formats
	vals["SemVer"] = ver.SemVer()
	vals["FullSemVer"] = ver.FullSemVer()
	vals["LegacySemVer"] = ver.LegacySemVer()
	vals["LegacySemVerPadded"] = ver.LegacySemVerPadded(pad)
	vals["InformationalVersion"] = ver.InformationalVersion()

	// Pre-release
	preTag := ver.PreReleaseTag.String()
	vals["PreReleaseTag"] = preTag
	if preTag != "" {
		vals["PreReleaseTagWithDash"] = "-" + preTag
	} else {
		vals["PreReleaseTagWithDash"] = ""
	}
	vals["PreReleaseLabel"] = ver.PreReleaseTag.Name
	if ver.PreReleaseTag.Name != "" {
		vals["PreReleaseLabelWithDash"] = "-" + ver.PreReleaseTag.Name
	} else {
		vals["PreReleaseLabelWithDash"] = ""
	}
	if ver.PreReleaseTag.Number != nil {
		vals["PreReleaseNumber"] = strconv.FormatInt(*ver.PreReleaseTag.Number, 10)
		vals["WeightedPreReleaseNumber"] = strconv.FormatInt(
			cfg.TagPreReleaseWeight+*ver.PreReleaseTag.Number, 10,
		)
	} else {
		vals["PreReleaseNumber"] = ""
		vals["WeightedPreReleaseNumber"] = ""
	}

	// Build metadata
	vals["BuildMetaData"] = ver.BuildMetaData.String()
	vals["BuildMetaDataPadded"] = ver.BuildMetaData.Padded(pad)
	vals["FullBuildMetaData"] = ver.BuildMetaData.FullString()

	// Git information
	vals["BranchName"] = ver.BuildMetaData.Branch
	vals["EscapedBranchName"] = escapeBranchName(ver.BuildMetaData.Branch)
	vals["Sha"] = ver.BuildMetaData.Sha
	vals["ShortSha"] = ver.BuildMetaData.ShortSha

	// Commit tracking
	vals["VersionSourceSha"] = ver.BuildMetaData.VersionSourceSha
	vals["CommitsSinceVersionSource"] = strconv.FormatInt(
		ver.BuildMetaData.CommitsSinceVersionSource, 10,
	)
	vals["CommitsSinceVersionSourcePadded"] = fmt.Sprintf(
		"%0*d", pad, ver.BuildMetaData.CommitsSinceVersionSource,
	)
	vals["UncommittedChanges"] = strconv.FormatInt(ver.BuildMetaData.UncommittedChanges, 10)

	// Commit date
	if !ver.BuildMetaData.CommitDate.IsZero() {
		goFmt := translateDateFormat(cfg.CommitDateFormat)
		vals["CommitDate"] = ver.BuildMetaData.CommitDate.Format(goFmt)
	} else {
		vals["CommitDate"] = ""
	}

	// Assembly info (output-only, no file updates)
	assemblyVer := majorStr + "." + minorStr + "." + patchStr + ".0"
	vals["AssemblySemVer"] = assemblyVer
	vals["AssemblySemFileVer"] = assemblyVer
	vals["AssemblyInformationalVersion"] = ver.InformationalVersion()

	// NuGet (output-only)
	nugetVer := ver.LegacySemVerPadded(pad)
	vals["NuGetVersionV2"] = nugetVer
	vals["NuGetVersion"] = nugetVer
	nugetPreRelease := ver.PreReleaseTag.LegacyPadded(pad)
	vals["NuGetPreReleaseTagV2"] = nugetPreRelease
	vals["NuGetPreReleaseTag"] = nugetPreRelease

	return vals
}

// dateFormatReplacements maps .NET/Java date format tokens to Go reference time tokens.
// Order matters: longer tokens must be replaced before shorter ones (e.g. "yyyy" before "yy").
var dateFormatReplacements = []struct{ from, to string }{
	{"yyyy", "2006"},
	{"yy", "06"},
	{"MMMM", "January"},
	{"MMM", "Jan"},
	{"MM", "01"},
	{"dd", "02"},
	{"HH", "15"},
	{"hh", "03"},
	{"mm", "04"},
	{"ss", "05"},
	{"tt", "PM"},
	{"fff", "000"},
	{"ff", "00"},
	{"f", "0"},
}

// translateDateFormat converts a .NET/Java-style date format (e.g. "yyyy-MM-dd")
// to a Go time layout. If the format already looks like a Go layout (contains
// the reference year "2006"), it is returned as-is.
func translateDateFormat(format string) string {
	if strings.Contains(format, "2006") {
		return format
	}
	result := format
	for _, r := range dateFormatReplacements {
		result = strings.ReplaceAll(result, r.from, r.to)
	}
	return result
}
