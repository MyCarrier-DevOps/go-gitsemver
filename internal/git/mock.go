package git

import "time"

// Compile-time check that MockRepository implements Repository.
var _ Repository = (*MockRepository)(nil)

// MockRepository is a configurable mock implementation of Repository for testing.
// Each method is backed by a function field. If the function field is nil,
// the method returns sensible zero values.
type MockRepository struct {
	PathFunc                       func() string
	WorkingDirectoryFunc           func() string
	IsHeadDetachedFunc             func() bool
	HeadFunc                       func() (Branch, error)
	BranchesFunc                   func(...PathFilter) ([]Branch, error)
	TagsFunc                       func(...PathFilter) ([]Tag, error)
	CommitFromShaFunc              func(string) (Commit, error)
	CommitLogFunc                  func(string, string, ...PathFilter) ([]Commit, error)
	MainlineCommitLogFunc          func(string, string, ...PathFilter) ([]Commit, error)
	BranchCommitsFunc              func(Branch, ...PathFilter) ([]Commit, error)
	CommitsPriorToFunc             func(time.Time, Branch) ([]Commit, error)
	FindMergeBaseFunc              func(string, string) (string, error)
	BranchesContainingCommitFunc   func(string) ([]Branch, error)
	NumberOfUncommittedChangesFunc func() (int, error)
	PeelTagToCommitFunc            func(Tag) (string, error)
}

func (m *MockRepository) Path() string {
	if m.PathFunc != nil {
		return m.PathFunc()
	}
	return ""
}

func (m *MockRepository) WorkingDirectory() string {
	if m.WorkingDirectoryFunc != nil {
		return m.WorkingDirectoryFunc()
	}
	return ""
}

func (m *MockRepository) IsHeadDetached() bool {
	if m.IsHeadDetachedFunc != nil {
		return m.IsHeadDetachedFunc()
	}
	return false
}

func (m *MockRepository) Head() (Branch, error) {
	if m.HeadFunc != nil {
		return m.HeadFunc()
	}
	return Branch{}, nil
}

func (m *MockRepository) Branches(filters ...PathFilter) ([]Branch, error) {
	if m.BranchesFunc != nil {
		return m.BranchesFunc(filters...)
	}
	return nil, nil
}

func (m *MockRepository) Tags(filters ...PathFilter) ([]Tag, error) {
	if m.TagsFunc != nil {
		return m.TagsFunc(filters...)
	}
	return nil, nil
}

func (m *MockRepository) CommitFromSha(sha string) (Commit, error) {
	if m.CommitFromShaFunc != nil {
		return m.CommitFromShaFunc(sha)
	}
	return Commit{}, nil
}

func (m *MockRepository) CommitLog(from, to string, filters ...PathFilter) ([]Commit, error) {
	if m.CommitLogFunc != nil {
		return m.CommitLogFunc(from, to, filters...)
	}
	return nil, nil
}

func (m *MockRepository) MainlineCommitLog(from, to string, filters ...PathFilter) ([]Commit, error) {
	if m.MainlineCommitLogFunc != nil {
		return m.MainlineCommitLogFunc(from, to, filters...)
	}
	return nil, nil
}

func (m *MockRepository) BranchCommits(branch Branch, filters ...PathFilter) ([]Commit, error) {
	if m.BranchCommitsFunc != nil {
		return m.BranchCommitsFunc(branch, filters...)
	}
	return nil, nil
}

func (m *MockRepository) CommitsPriorTo(olderThan time.Time, branch Branch) ([]Commit, error) {
	if m.CommitsPriorToFunc != nil {
		return m.CommitsPriorToFunc(olderThan, branch)
	}
	return nil, nil
}

func (m *MockRepository) FindMergeBase(sha1, sha2 string) (string, error) {
	if m.FindMergeBaseFunc != nil {
		return m.FindMergeBaseFunc(sha1, sha2)
	}
	return "", nil
}

func (m *MockRepository) BranchesContainingCommit(sha string) ([]Branch, error) {
	if m.BranchesContainingCommitFunc != nil {
		return m.BranchesContainingCommitFunc(sha)
	}
	return nil, nil
}

func (m *MockRepository) NumberOfUncommittedChanges() (int, error) {
	if m.NumberOfUncommittedChangesFunc != nil {
		return m.NumberOfUncommittedChangesFunc()
	}
	return 0, nil
}

func (m *MockRepository) PeelTagToCommit(tag Tag) (string, error) {
	if m.PeelTagToCommitFunc != nil {
		return m.PeelTagToCommitFunc(tag)
	}
	return tag.TargetSha, nil
}
