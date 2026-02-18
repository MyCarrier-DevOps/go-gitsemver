package semver

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// BuildMetaData represents the build metadata of a semantic version.
// This type is immutable — all methods return new values.
type BuildMetaData struct {
	CommitsSinceTag           *int64
	Branch                    string
	Sha                       string
	ShortSha                  string
	VersionSourceSha          string
	CommitDate                time.Time
	CommitsSinceVersionSource int64
	UncommittedChanges        int64
}

// String returns the short metadata string (commits since tag count).
func (m BuildMetaData) String() string {
	if m.CommitsSinceTag == nil {
		return ""
	}
	return strconv.FormatInt(*m.CommitsSinceTag, 10)
}

// FullString returns the complete metadata string including branch and SHA.
// Format: "5.Branch.main.Sha.abc1234"
func (m BuildMetaData) FullString() string {
	var parts []string
	if m.CommitsSinceTag != nil {
		parts = append(parts, strconv.FormatInt(*m.CommitsSinceTag, 10))
	}
	if m.Branch != "" {
		parts = append(parts, "Branch."+m.Branch)
	}
	if m.Sha != "" {
		parts = append(parts, "Sha."+m.Sha)
	}
	return strings.Join(parts, ".")
}

// Padded returns the metadata string with zero-padded commits since tag.
func (m BuildMetaData) Padded(pad int) string {
	if m.CommitsSinceTag == nil {
		return ""
	}
	return fmt.Sprintf("%0*d", pad, *m.CommitsSinceTag)
}
