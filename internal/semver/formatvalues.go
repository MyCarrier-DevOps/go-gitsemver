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
	// CommitDateFormat is the format for commit dates (default: "2006-01-02").
	// Accepts Go time layouts or .NET/Java-style formats (e.g. "yyyy-MM-dd").
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

	// Commit date and commit tag
	if !ver.BuildMetaData.CommitDate.IsZero() {
		goFmt := translateDateFormat(cfg.CommitDateFormat)
		vals["CommitDate"] = ver.BuildMetaData.CommitDate.Format(goFmt)
		year, week := ver.BuildMetaData.CommitDate.ISOWeek()
		vals["CommitTag"] = fmt.Sprintf("%02d.%02d.%s", year%100, week, ver.BuildMetaData.ShortSha)
	} else {
		vals["CommitDate"] = ""
		vals["CommitTag"] = ""
	}

	// Assembly info (output-only, no file updates)
	vals["AssemblySemVer"] = majorStr + "." + minorStr + ".0.0"
	vals["AssemblySemFileVer"] = majorStr + "." + minorStr + "." + patchStr + ".0"
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
// Order matters: longer tokens must be replaced before shorter ones (e.g. "yyyy" before "yy",
// "dddd" before "ddd" before "dd" before "d").
var dateFormatReplacements = []struct{ from, to string }{
	// Year
	{"yyyy", "2006"},
	{"yy", "06"},
	// Month
	{"MMMM", "January"},
	{"MMM", "Jan"},
	{"MM", "01"},
	{"M", "1"},
	// Day of week (before day-of-month)
	{"dddd", "Monday"},
	{"ddd", "Mon"},
	// Day of month
	{"dd", "02"},
	{"d", "2"},
	// Hour 24h
	{"HH", "15"},
	{"H", "15"},
	// Hour 12h
	{"hh", "03"},
	{"h", "3"},
	// Minute
	{"mm", "04"},
	{"m", "4"},
	// Second
	{"ss", "05"},
	{"s", "5"},
	// AM/PM
	{"tt", "PM"},
	// Timezone offset
	{"zzz", "-07:00"},
	{"zz", "-07"},
	{"z", "-7"},
	// Fractional seconds
	{"fff", "000"},
	{"ff", "00"},
	{"f", "0"},
}

// translateDateFormat converts a .NET/Java-style date format (e.g. "yyyy-MM-dd")
// to a Go time layout. If the format already looks like a Go layout (contains
// "2006" or "15:04"), it is returned as-is.
func translateDateFormat(format string) string {
	if strings.Contains(format, "2006") || strings.Contains(format, "15:04") {
		return format
	}
	result := format
	// Use placeholders to prevent Go reference values (e.g. "Monday")
	// from being corrupted by subsequent single-letter replacements.
	placeholders := make([]string, len(dateFormatReplacements))
	for i := range dateFormatReplacements {
		placeholders[i] = "\x00" + strconv.Itoa(i) + "\x00"
	}
	for i, r := range dateFormatReplacements {
		result = strings.ReplaceAll(result, r.from, placeholders[i])
	}
	for i, r := range dateFormatReplacements {
		result = strings.ReplaceAll(result, placeholders[i], r.to)
	}
	return result
}
