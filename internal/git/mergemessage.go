package git

import (
	"regexp"
	"strconv"
	"strings"
)

// MergeMessageFormat defines a named regex pattern for merge messages.
type MergeMessageFormat struct {
	Name    string
	Pattern *regexp.Regexp
}

// MergeMessage represents a parsed merge commit or squash merge message.
type MergeMessage struct {
	FormatName          string
	MergedBranch        string
	TargetBranch        string
	PullRequestNumber   int
	IsMergedPullRequest bool
}

// IsEmpty returns true if the merge message did not match any format.
func (m MergeMessage) IsEmpty() bool {
	return m.FormatName == ""
}

// defaultFormats are the 6 built-in merge message formats.
var defaultFormats = []MergeMessageFormat{
	{
		Name:    "Default",
		Pattern: regexp.MustCompile(`(?i)^Merge (branch|tag) '(?P<SourceBranch>[^']*)'(?: into (?P<TargetBranch>\S*))*`),
	},
	{
		Name:    "SmartGit",
		Pattern: regexp.MustCompile(`(?i)^Finish (?P<SourceBranch>\S*)(?: into (?P<TargetBranch>\S*))*`),
	},
	{
		Name:    "BitBucketPull",
		Pattern: regexp.MustCompile(`(?i)^Merge pull request #(?P<PullRequestNumber>\d+) (?:from|in) (?P<Source>.*) from (?P<SourceBranch>\S*) to (?P<TargetBranch>\S*)`),
	},
	{
		Name:    "BitBucketPullv7",
		Pattern: regexp.MustCompile(`(?is)^Pull request #(?P<PullRequestNumber>\d+).*\n\nMerge in (?P<Source>.*) from (?P<SourceBranch>\S*) to (?P<TargetBranch>\S*)`),
	},
	{
		Name:    "GitHubPull",
		Pattern: regexp.MustCompile(`(?i)^Merge pull request #(?P<PullRequestNumber>\d+) (?:from|in) (?P<SourceBranch>\S*)(?: into (?P<TargetBranch>\S*))*`),
	},
	{
		Name:    "RemoteTracking",
		Pattern: regexp.MustCompile(`(?i)^Merge remote-tracking branch '(?P<SourceBranch>[^']*)'(?: into (?P<TargetBranch>\S*))*`),
	},
}

// squashFormats are merge message formats for squash merges.
var squashFormats = []MergeMessageFormat{
	{
		Name:    "GitHubSquash",
		Pattern: regexp.MustCompile(`^.+\(#(?P<PullRequestNumber>\d+)\)$`),
	},
	{
		Name:    "BitBucketSquash",
		Pattern: regexp.MustCompile(`(?i)^Merged in (?P<SourceBranch>\S*) \(pull request #(?P<PullRequestNumber>\d+)\)`),
	},
}

// DefaultMergeMessageFormats returns the 6 built-in merge message formats.
func DefaultMergeMessageFormats() []MergeMessageFormat {
	return defaultFormats
}

// SquashMergeMessageFormats returns the squash merge message formats.
func SquashMergeMessageFormats() []MergeMessageFormat {
	return squashFormats
}

// ParseMergeMessage parses a commit message against all known merge formats.
// Custom formats are tried first, then defaults, then squash formats.
// Returns a zero MergeMessage if no format matches.
func ParseMergeMessage(message string, customFormats map[string]string) MergeMessage {
	// Take only the first line for most patterns (except BitBucketPullv7 which is multiline)
	var allFormats []MergeMessageFormat

	// Custom formats first (highest priority).
	for name, pattern := range customFormats {
		re, err := regexp.Compile("(?i)" + pattern)
		if err != nil {
			continue
		}
		allFormats = append(allFormats, MergeMessageFormat{Name: name, Pattern: re})
	}

	allFormats = append(allFormats, defaultFormats...)
	allFormats = append(allFormats, squashFormats...)

	for _, format := range allFormats {
		match := format.Pattern.FindStringSubmatch(message)
		if match == nil {
			continue
		}

		result := MergeMessage{FormatName: format.Name}
		for i, name := range format.Pattern.SubexpNames() {
			if i == 0 || name == "" || match[i] == "" {
				continue
			}
			switch name {
			case "SourceBranch":
				result.MergedBranch = match[i]
			case "TargetBranch":
				result.TargetBranch = match[i]
			case "PullRequestNumber":
				if n, err := strconv.Atoi(match[i]); err == nil {
					result.PullRequestNumber = n
					result.IsMergedPullRequest = true
				}
			}
		}
		return result
	}

	return MergeMessage{}
}

// versionSegmentRe matches a semantic version segment (e.g., "1.2.0", "1.2", "2").
var versionSegmentRe = regexp.MustCompile(`^\d+(\.\d+){0,2}$`)

// ExtractVersionFromBranch attempts to extract a semantic version string from
// a branch name. It splits the branch name on '/' and '-', strips tag prefix,
// and looks for a segment that starts with a digit pattern. Segments like
// "JIRA-123" are not matched because the pre-dash portion is non-numeric.
func ExtractVersionFromBranch(branchName, tagPrefix string) (string, bool) {
	parts := strings.Split(branchName, "/")

	var prefixRe *regexp.Regexp
	if tagPrefix != "" {
		prefixRe, _ = regexp.Compile("^(?:" + tagPrefix + ")")
	}

	for _, part := range parts {
		// Try the whole part first (handles "1.2.0" directly).
		if v, ok := tryExtractVersion(part, prefixRe); ok {
			return v, true
		}
		// Then try splitting on '-' for cases like "release-1.3.0",
		// but only if the sub-segment is purely a version (no trailing text).
		dashParts := strings.SplitN(part, "-", 2)
		if len(dashParts) == 2 {
			if v, ok := tryExtractVersion(dashParts[1], prefixRe); ok {
				return v, true
			}
		}
	}

	return "", false
}

func tryExtractVersion(s string, prefixRe *regexp.Regexp) (string, bool) {
	cleaned := s
	if prefixRe != nil {
		cleaned = prefixRe.ReplaceAllString(s, "")
	}
	if cleaned == "" {
		return "", false
	}
	if versionSegmentRe.MatchString(cleaned) {
		return normalizeVersion(cleaned), true
	}
	return "", false
}

// normalizeVersion pads a version string to at least Major.Minor.Patch.
func normalizeVersion(v string) string {
	parts := strings.Split(v, ".")
	for len(parts) < 3 {
		parts = append(parts, "0")
	}
	return strings.Join(parts, ".")
}
