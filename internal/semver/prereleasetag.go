package semver

import (
	"fmt"
	"strconv"
	"strings"
)

// PreReleaseTag represents the pre-release portion of a semantic version.
// This type is immutable — all methods return new values.
type PreReleaseTag struct {
	Name   string
	Number *int64
}

// HasTag returns true when the pre-release tag has a name or number.
func (t PreReleaseTag) HasTag() bool {
	return t.Name != "" || t.Number != nil
}

// WithName returns a new PreReleaseTag with the given name.
func (t PreReleaseTag) WithName(name string) PreReleaseTag {
	return PreReleaseTag{Name: name, Number: t.Number}
}

// WithNumber returns a new PreReleaseTag with the given number.
func (t PreReleaseTag) WithNumber(n int64) PreReleaseTag {
	return PreReleaseTag{Name: t.Name, Number: &n}
}

// CompareTo compares two PreReleaseTags.
// Returns a negative value, zero, or a positive value.
// A stable version (no tag) is greater than a pre-release version.
// Pre-release versions are compared by name (case-insensitive), then by number.
func (t PreReleaseTag) CompareTo(other PreReleaseTag) int {
	if !t.HasTag() && !other.HasTag() {
		return 0
	}
	if !t.HasTag() {
		return 1 // stable > pre-release
	}
	if !other.HasTag() {
		return -1 // pre-release < stable
	}

	nameComp := strings.Compare(strings.ToLower(t.Name), strings.ToLower(other.Name))
	if nameComp != 0 {
		return nameComp
	}

	tNum := int64(0)
	if t.Number != nil {
		tNum = *t.Number
	}
	oNum := int64(0)
	if other.Number != nil {
		oNum = *other.Number
	}

	switch {
	case tNum < oNum:
		return -1
	case tNum > oNum:
		return 1
	default:
		return 0
	}
}

// String returns the dotted pre-release string (e.g., "beta.4").
func (t PreReleaseTag) String() string {
	if !t.HasTag() {
		return ""
	}
	if t.Number == nil {
		return t.Name
	}
	if t.Name == "" {
		return strconv.FormatInt(*t.Number, 10)
	}
	return t.Name + "." + strconv.FormatInt(*t.Number, 10)
}

// Legacy returns the pre-release string without a dot separator (e.g., "beta4").
func (t PreReleaseTag) Legacy() string {
	if !t.HasTag() {
		return ""
	}
	if t.Number == nil {
		return t.Name
	}
	if t.Name == "" {
		return strconv.FormatInt(*t.Number, 10)
	}
	return t.Name + strconv.FormatInt(*t.Number, 10)
}

// LegacyPadded returns the legacy format with zero-padded number (e.g., "beta0004").
func (t PreReleaseTag) LegacyPadded(pad int) string {
	if !t.HasTag() {
		return ""
	}
	if t.Number == nil {
		return t.Name
	}
	if t.Name == "" {
		return fmt.Sprintf("%0*d", pad, *t.Number)
	}
	return t.Name + fmt.Sprintf("%0*d", pad, *t.Number)
}
