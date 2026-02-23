package git

import "time"

// Repository provides low-level git operations.
// This is the key abstraction point for testing and backend swapping.
// All methods that traverse commits or list refs accept optional PathFilter
// parameters for monorepo support (DI-11).
type Repository interface {
	// Path returns the path to the .git directory.
	Path() string

	// WorkingDirectory returns the path to the working directory.
	WorkingDirectory() string

	// IsHeadDetached returns true if HEAD is not pointing to a branch.
	IsHeadDetached() bool

	// Head returns the current HEAD branch.
	Head() (Branch, error)

	// Branches returns all branches in the repository.
	Branches(filters ...PathFilter) ([]Branch, error)

	// Tags returns all tags in the repository.
	Tags(filters ...PathFilter) ([]Tag, error)

	// CommitFromSha returns the commit with the given SHA.
	CommitFromSha(sha string) (Commit, error)

	// CommitLog returns commits reachable from 'to' but not from 'from',
	// in reverse chronological order. If from is empty, all ancestors of
	// 'to' are returned.
	CommitLog(from, to string, filters ...PathFilter) ([]Commit, error)

	// MainlineCommitLog returns first-parent-only commits reachable from
	// 'to' but not from 'from'. Used for mainline mode calculations.
	MainlineCommitLog(from, to string, filters ...PathFilter) ([]Commit, error)

	// BranchCommits returns commits on a specific branch in reverse
	// chronological order.
	BranchCommits(branch Branch, filters ...PathFilter) ([]Commit, error)

	// CommitsPriorTo returns branch commits whose date is older than the
	// given time.
	CommitsPriorTo(olderThan time.Time, branch Branch) ([]Commit, error)

	// FindMergeBase returns the best common ancestor of two commits.
	// Returns an empty string if no merge base exists.
	FindMergeBase(sha1, sha2 string) (string, error)

	// BranchesContainingCommit returns all branches that contain the
	// given commit SHA.
	BranchesContainingCommit(sha string) ([]Branch, error)

	// NumberOfUncommittedChanges returns the count of uncommitted changes
	// in the working directory.
	NumberOfUncommittedChanges() (int, error)

	// PeelTagToCommit resolves a tag to its target commit SHA.
	// For lightweight tags, returns the target directly.
	// For annotated tags, peels through to the commit.
	PeelTagToCommit(tag Tag) (string, error)
}
