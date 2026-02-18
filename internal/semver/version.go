package semver

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

var versionRegex = regexp.MustCompile(
	`^(\d+)(?:\.(\d+))?(?:\.(\d+))?(?:\.(\d+))?(?:-([^+]*))?(?:\+(.*))?$`,
)

// SemanticVersion represents a semantic version.
// This type is immutable — all methods return new values.
type SemanticVersion struct {
	Major         int64
	Minor         int64
	Patch         int64
	PreReleaseTag PreReleaseTag
	BuildMetaData BuildMetaData
}

// TryParse attempts to parse a version string with an optional tag prefix regex.
// If tagPrefix is non-empty, the string must start with a match for the prefix.
// Returns the parsed version and true if successful.
func TryParse(s, tagPrefix string) (SemanticVersion, bool) {
	v, err := Parse(s, tagPrefix)
	if err != nil {
		return SemanticVersion{}, false
	}
	return v, true
}

// Parse parses a version string with an optional tag prefix regex.
// If tagPrefix is non-empty, the string must start with a match for the prefix.
func Parse(s, tagPrefix string) (SemanticVersion, error) {
	remaining := s

	if tagPrefix != "" {
		prefixRegex, err := regexp.Compile("^(?:" + tagPrefix + ")")
		if err != nil {
			return SemanticVersion{}, errors.New("invalid tag prefix regex: " + err.Error())
		}
		loc := prefixRegex.FindStringIndex(remaining)
		if loc == nil {
			return SemanticVersion{}, errors.New("version string does not match tag prefix: " + s)
		}
		remaining = remaining[loc[1]:]
	}

	matches := versionRegex.FindStringSubmatch(remaining)
	if matches == nil {
		return SemanticVersion{}, errors.New("invalid version format: " + s)
	}

	var v SemanticVersion

	major, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return SemanticVersion{}, errors.New("invalid major version: " + matches[1])
	}
	v.Major = major

	if matches[2] != "" {
		minor, err := strconv.ParseInt(matches[2], 10, 64)
		if err != nil {
			return SemanticVersion{}, errors.New("invalid minor version: " + matches[2])
		}
		v.Minor = minor
	}

	if matches[3] != "" {
		patch, err := strconv.ParseInt(matches[3], 10, 64)
		if err != nil {
			return SemanticVersion{}, errors.New("invalid patch version: " + matches[3])
		}
		v.Patch = patch
	}

	// matches[4] is FourthPart — accepted for compatibility but not stored

	if matches[5] != "" {
		v.PreReleaseTag = parsePreReleaseTag(matches[5])
	}

	// Build metadata: parse commits-since-tag if it's a simple number
	if matches[6] != "" {
		if n, parseErr := strconv.ParseInt(matches[6], 10, 64); parseErr == nil {
			v.BuildMetaData = BuildMetaData{CommitsSinceTag: &n}
		}
	}

	return v, nil
}

// parsePreReleaseTag parses a pre-release tag string into a PreReleaseTag.
// Handles formats like "beta.4", "beta", "4", "alpha.1".
func parsePreReleaseTag(s string) PreReleaseTag {
	if s == "" {
		return PreReleaseTag{}
	}

	// Try splitting on the last dot
	lastDot := strings.LastIndex(s, ".")
	if lastDot >= 0 {
		name := s[:lastDot]
		numStr := s[lastDot+1:]
		if num, err := strconv.ParseInt(numStr, 10, 64); err == nil {
			return PreReleaseTag{Name: name, Number: &num}
		}
	}

	// Try parsing the whole string as a number
	if num, err := strconv.ParseInt(s, 10, 64); err == nil {
		return PreReleaseTag{Number: &num}
	}

	// It's just a name
	return PreReleaseTag{Name: s}
}

// CompareTo compares two SemanticVersions.
// Returns a negative value, zero, or a positive value.
// Build metadata is not considered in comparisons per SemVer 2.0 spec.
func (v SemanticVersion) CompareTo(other SemanticVersion) int {
	if v.Major != other.Major {
		if v.Major > other.Major {
			return 1
		}
		return -1
	}

	if v.Minor != other.Minor {
		if v.Minor > other.Minor {
			return 1
		}
		return -1
	}

	if v.Patch != other.Patch {
		if v.Patch > other.Patch {
			return 1
		}
		return -1
	}

	return v.PreReleaseTag.CompareTo(other.PreReleaseTag)
}

// IncrementField bumps the specified version field.
// Higher fields are preserved, lower fields are zeroed.
// Pre-release tag and build metadata are cleared.
// VersionFieldNone returns the version unchanged.
func (v SemanticVersion) IncrementField(field VersionField) SemanticVersion {
	switch field {
	case VersionFieldMajor:
		return SemanticVersion{Major: v.Major + 1}
	case VersionFieldMinor:
		return SemanticVersion{Major: v.Major, Minor: v.Minor + 1}
	case VersionFieldPatch:
		return SemanticVersion{Major: v.Major, Minor: v.Minor, Patch: v.Patch + 1}
	default:
		return v
	}
}

// IncrementPreRelease bumps the pre-release number.
// Panics if the version has no pre-release number.
func (v SemanticVersion) IncrementPreRelease() SemanticVersion {
	if v.PreReleaseTag.Number == nil {
		panic("cannot increment pre-release: no pre-release number set")
	}
	newNum := *v.PreReleaseTag.Number + 1
	return SemanticVersion{
		Major:         v.Major,
		Minor:         v.Minor,
		Patch:         v.Patch,
		PreReleaseTag: PreReleaseTag{Name: v.PreReleaseTag.Name, Number: &newNum},
		BuildMetaData: v.BuildMetaData,
	}
}

// WithPreReleaseTag returns a new SemanticVersion with the given pre-release tag.
func (v SemanticVersion) WithPreReleaseTag(tag PreReleaseTag) SemanticVersion {
	return SemanticVersion{
		Major:         v.Major,
		Minor:         v.Minor,
		Patch:         v.Patch,
		PreReleaseTag: tag,
		BuildMetaData: v.BuildMetaData,
	}
}

// WithBuildMetaData returns a new SemanticVersion with the given build metadata.
func (v SemanticVersion) WithBuildMetaData(meta BuildMetaData) SemanticVersion {
	return SemanticVersion{
		Major:         v.Major,
		Minor:         v.Minor,
		Patch:         v.Patch,
		PreReleaseTag: v.PreReleaseTag,
		BuildMetaData: meta,
	}
}

// SemVer returns the SemVer 2.0 format (e.g., "1.2.3" or "1.2.3-beta.4").
func (v SemanticVersion) SemVer() string {
	base := strconv.FormatInt(v.Major, 10) + "." +
		strconv.FormatInt(v.Minor, 10) + "." +
		strconv.FormatInt(v.Patch, 10)
	if tag := v.PreReleaseTag.String(); tag != "" {
		return base + "-" + tag
	}
	return base
}

// FullSemVer returns the SemVer with build metadata (e.g., "1.2.3-beta.4+5").
func (v SemanticVersion) FullSemVer() string {
	s := v.SemVer()
	if meta := v.BuildMetaData.String(); meta != "" {
		return s + "+" + meta
	}
	return s
}

// LegacySemVer returns the legacy format without dots in pre-release (e.g., "1.2.3-beta4").
func (v SemanticVersion) LegacySemVer() string {
	base := strconv.FormatInt(v.Major, 10) + "." +
		strconv.FormatInt(v.Minor, 10) + "." +
		strconv.FormatInt(v.Patch, 10)
	if tag := v.PreReleaseTag.Legacy(); tag != "" {
		return base + "-" + tag
	}
	return base
}

// LegacySemVerPadded returns the padded legacy format (e.g., "1.2.3-beta0004").
func (v SemanticVersion) LegacySemVerPadded(pad int) string {
	base := strconv.FormatInt(v.Major, 10) + "." +
		strconv.FormatInt(v.Minor, 10) + "." +
		strconv.FormatInt(v.Patch, 10)
	if tag := v.PreReleaseTag.LegacyPadded(pad); tag != "" {
		return base + "-" + tag
	}
	return base
}

// InformationalVersion returns the full informational string
// (e.g., "1.2.3-beta.4+5.Branch.main.Sha.abc1234").
func (v SemanticVersion) InformationalVersion() string {
	s := v.SemVer()
	if meta := v.BuildMetaData.FullString(); meta != "" {
		return s + "+" + meta
	}
	return s
}
