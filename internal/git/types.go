// Package git provides the git abstraction layer for semantic version calculation.
// It defines concrete entity types (Commit, Branch, Tag), a Repository interface,
// and higher-level domain queries via RepositoryStore.
package git

import (
	"go-gitsemver/internal/semver"
	"strings"
	"time"
)

const (
	localBranchPrefix          = "refs/heads/"
	remoteTrackingBranchPrefix = "refs/remotes/"
	tagRefPrefix               = "refs/tags/"
)

// PathFilter constrains git queries to files matching a path pattern.
// Used for monorepo support (DI-11). An empty PathFilter means no filtering.
type PathFilter string

// ObjectID represents a git object identifier.
type ObjectID struct {
	Sha string
}

// ShortSha returns the first n characters of the SHA.
func (id ObjectID) ShortSha(n int) string {
	if n >= len(id.Sha) {
		return id.Sha
	}
	return id.Sha[:n]
}

// String returns the full SHA.
func (id ObjectID) String() string {
	return id.Sha
}

// Commit represents a git commit.
type Commit struct {
	Sha     string
	Parents []string // parent SHAs; len > 1 means merge commit
	When    time.Time
	Message string
}

// IsMerge returns true if the commit has more than one parent.
func (c Commit) IsMerge() bool {
	return len(c.Parents) > 1
}

// ShortSha returns the first 7 characters of the SHA.
func (c Commit) ShortSha() string {
	if len(c.Sha) >= 7 {
		return c.Sha[:7]
	}
	return c.Sha
}

// IsEmpty returns true if the commit has no SHA (zero value).
func (c Commit) IsEmpty() bool {
	return c.Sha == ""
}

// ReferenceName represents a git reference with canonical and friendly forms.
type ReferenceName struct {
	Canonical     string // e.g., "refs/heads/main"
	Friendly      string // e.g., "main"
	WithoutRemote string // e.g., "main" (strips "origin/" from remote refs)
}

// NewReferenceName creates a ReferenceName from a canonical ref path.
func NewReferenceName(canonical string) ReferenceName {
	friendly := canonical
	withoutRemote := canonical

	switch {
	case strings.HasPrefix(canonical, localBranchPrefix):
		friendly = canonical[len(localBranchPrefix):]
		withoutRemote = friendly
	case strings.HasPrefix(canonical, remoteTrackingBranchPrefix):
		friendly = canonical[len(remoteTrackingBranchPrefix):]
		if idx := strings.Index(friendly, "/"); idx >= 0 {
			withoutRemote = friendly[idx+1:]
		} else {
			withoutRemote = friendly
		}
	case strings.HasPrefix(canonical, tagRefPrefix):
		friendly = canonical[len(tagRefPrefix):]
		withoutRemote = friendly
	}

	return ReferenceName{
		Canonical:     canonical,
		Friendly:      friendly,
		WithoutRemote: withoutRemote,
	}
}

// NewBranchReferenceName creates a ReferenceName for a local branch.
func NewBranchReferenceName(name string) ReferenceName {
	return NewReferenceName(localBranchPrefix + name)
}

// IsBranch returns true if this reference is a local branch.
func (r ReferenceName) IsBranch() bool {
	return strings.HasPrefix(r.Canonical, localBranchPrefix)
}

// IsRemoteBranch returns true if this reference is a remote tracking branch.
func (r ReferenceName) IsRemoteBranch() bool {
	return strings.HasPrefix(r.Canonical, remoteTrackingBranchPrefix)
}

// IsTag returns true if this reference is a tag.
func (r ReferenceName) IsTag() bool {
	return strings.HasPrefix(r.Canonical, tagRefPrefix)
}

// Branch represents a git branch.
type Branch struct {
	Name           ReferenceName
	Tip            *Commit
	IsRemote       bool
	IsDetachedHead bool
}

// FriendlyName returns the friendly name of the branch.
func (b Branch) FriendlyName() string {
	return b.Name.Friendly
}

// Tag represents a git tag.
type Tag struct {
	Name      ReferenceName
	TargetSha string // SHA of the commit this tag points to
}

// BranchCommit represents a branch and the commit where it was branched from.
type BranchCommit struct {
	Branch Branch
	Commit Commit
}

// VersionTag holds a tag, its parsed semantic version, and the target commit.
type VersionTag struct {
	Tag     Tag
	Version semver.SemanticVersion
	Commit  Commit
}
