package git

import (
	"fmt"
	"go-gitsemver/internal/config"
	"go-gitsemver/internal/semver"
	"regexp"
	"sort"
	"time"
)

// RepositoryStore provides higher-level domain queries built on top of a
// Repository. It uses config and semver packages to interpret git data
// in the context of semantic versioning.
type RepositoryStore struct {
	repo Repository
}

// NewRepositoryStore creates a new RepositoryStore wrapping the given Repository.
func NewRepositoryStore(repo Repository) *RepositoryStore {
	return &RepositoryStore{repo: repo}
}

// --- Tag queries ---

// GetValidVersionTags returns all tags that parse as semantic versions,
// optionally filtered to tags on commits older than the given time.
func (s *RepositoryStore) GetValidVersionTags(tagPrefix string, olderThan *time.Time, filters ...PathFilter) ([]VersionTag, error) {
	tags, err := s.repo.Tags(filters...)
	if err != nil {
		return nil, fmt.Errorf("listing tags: %w", err)
	}

	var result []VersionTag
	for _, tag := range tags {
		ver, ok := semver.TryParse(tag.Name.Friendly, tagPrefix)
		if !ok {
			continue
		}

		commitSha, err := s.repo.PeelTagToCommit(tag)
		if err != nil {
			continue
		}

		commit, err := s.repo.CommitFromSha(commitSha)
		if err != nil {
			continue
		}

		if olderThan != nil && commit.When.After(*olderThan) {
			continue
		}

		result = append(result, VersionTag{Tag: tag, Version: ver, Commit: commit})
	}

	return result, nil
}

// GetVersionTagsOnBranch returns semantic versions from tags on the given branch.
// Results are sorted by version descending (highest first).
func (s *RepositoryStore) GetVersionTagsOnBranch(branch Branch, tagPrefix string, filters ...PathFilter) ([]semver.SemanticVersion, error) {
	versionTags, err := s.GetValidVersionTags(tagPrefix, nil, filters...)
	if err != nil {
		return nil, err
	}

	commits, err := s.repo.BranchCommits(branch, filters...)
	if err != nil {
		return nil, fmt.Errorf("getting branch commits: %w", err)
	}

	// Build a set of commit SHAs on this branch.
	commitSet := make(map[string]struct{}, len(commits))
	for _, c := range commits {
		commitSet[c.Sha] = struct{}{}
	}

	var versions []semver.SemanticVersion
	for _, vt := range versionTags {
		if _, ok := commitSet[vt.Commit.Sha]; ok {
			versions = append(versions, vt.Version)
		}
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].CompareTo(versions[j]) > 0
	})

	return versions, nil
}

// GetCurrentCommitTaggedVersion returns the highest semantic version tag on the
// given commit. Returns false if the commit has no version tag.
func (s *RepositoryStore) GetCurrentCommitTaggedVersion(commit Commit, tagPrefix string) (semver.SemanticVersion, bool, error) {
	versionTags, err := s.GetValidVersionTags(tagPrefix, nil)
	if err != nil {
		return semver.SemanticVersion{}, false, err
	}

	var best *semver.SemanticVersion
	for _, vt := range versionTags {
		if vt.Commit.Sha == commit.Sha {
			v := vt.Version
			if best == nil || v.CompareTo(*best) > 0 {
				best = &v
			}
		}
	}

	if best == nil {
		return semver.SemanticVersion{}, false, nil
	}
	return *best, true, nil
}

// --- Branch queries ---

// FindMainBranch returns the branch matching the main branch regex from config.
func (s *RepositoryStore) FindMainBranch(cfg *config.Config) (Branch, bool, error) {
	mainBC, ok := cfg.Branches["main"]
	if !ok || mainBC.Regex == nil {
		return Branch{}, false, nil
	}

	re, err := regexp.Compile(*mainBC.Regex)
	if err != nil {
		return Branch{}, false, fmt.Errorf("invalid main branch regex %q: %w", *mainBC.Regex, err)
	}

	branches, err := s.repo.Branches()
	if err != nil {
		return Branch{}, false, fmt.Errorf("listing branches: %w", err)
	}

	for _, b := range branches {
		if !b.IsRemote && re.MatchString(b.FriendlyName()) {
			return b, true, nil
		}
	}

	return Branch{}, false, nil
}

// GetReleaseBranches returns all branches matching any release branch config regex.
func (s *RepositoryStore) GetReleaseBranches(releaseBranchConfig map[string]*config.BranchConfig) ([]Branch, error) {
	branches, err := s.repo.Branches()
	if err != nil {
		return nil, fmt.Errorf("listing branches: %w", err)
	}

	var patterns []*regexp.Regexp
	for _, bc := range releaseBranchConfig {
		if bc.Regex == nil {
			continue
		}
		re, err := regexp.Compile(*bc.Regex)
		if err != nil {
			continue
		}
		patterns = append(patterns, re)
	}

	var result []Branch
	for _, b := range branches {
		if b.IsRemote {
			continue
		}
		for _, re := range patterns {
			if re.MatchString(b.FriendlyName()) {
				result = append(result, b)
				break
			}
		}
	}

	return result, nil
}

// GetBranchesContainingCommit returns branches that contain the given commit.
func (s *RepositoryStore) GetBranchesContainingCommit(commit Commit) ([]Branch, error) {
	if commit.IsEmpty() {
		return nil, nil
	}
	return s.repo.BranchesContainingCommit(commit.Sha)
}

// GetBranchesForCommit returns non-remote branches whose tip is the given commit.
func (s *RepositoryStore) GetBranchesForCommit(commit Commit) ([]Branch, error) {
	branches, err := s.repo.Branches()
	if err != nil {
		return nil, fmt.Errorf("listing branches: %w", err)
	}

	var result []Branch
	for _, b := range branches {
		if !b.IsRemote && b.Tip != nil && b.Tip.Sha == commit.Sha {
			result = append(result, b)
		}
	}

	return result, nil
}

// GetTargetBranch resolves the target branch from a name or HEAD.
func (s *RepositoryStore) GetTargetBranch(targetBranchName string) (Branch, error) {
	if targetBranchName == "" {
		return s.repo.Head()
	}

	branches, err := s.repo.Branches()
	if err != nil {
		return Branch{}, fmt.Errorf("listing branches: %w", err)
	}

	for _, b := range branches {
		if b.FriendlyName() == targetBranchName || b.Name.WithoutRemote == targetBranchName {
			return b, nil
		}
	}

	return Branch{}, fmt.Errorf("branch %q not found", targetBranchName)
}

// --- Commit queries ---

// GetCurrentCommit returns the commit from a SHA or the branch tip.
func (s *RepositoryStore) GetCurrentCommit(branch Branch, commitID string) (Commit, error) {
	if commitID != "" {
		return s.repo.CommitFromSha(commitID)
	}
	if branch.Tip == nil {
		return Commit{}, fmt.Errorf("branch %q has no tip commit", branch.FriendlyName())
	}
	return *branch.Tip, nil
}

// GetBaseVersionSource returns the root commit (first commit) reachable from tip.
func (s *RepositoryStore) GetBaseVersionSource(tip Commit) (Commit, error) {
	commits, err := s.repo.CommitLog("", tip.Sha)
	if err != nil {
		return Commit{}, fmt.Errorf("getting commit log: %w", err)
	}
	if len(commits) == 0 {
		return tip, nil
	}
	return commits[len(commits)-1], nil
}

// GetCommitLog returns commits between from and to.
func (s *RepositoryStore) GetCommitLog(from, to Commit) ([]Commit, error) {
	return s.repo.CommitLog(from.Sha, to.Sha)
}

// GetMainlineCommitLog returns first-parent-only commits between from and to.
func (s *RepositoryStore) GetMainlineCommitLog(from, to Commit) ([]Commit, error) {
	return s.repo.MainlineCommitLog(from.Sha, to.Sha)
}

// GetMergeBaseCommits returns commits reachable from mergedHead but not from mergeBase.
func (s *RepositoryStore) GetMergeBaseCommits(mergedHead, mergeBase Commit) ([]Commit, error) {
	return s.repo.CommitLog(mergeBase.Sha, mergedHead.Sha)
}

// --- Merge base ---

// FindMergeBase returns the merge base commit of two branches.
func (s *RepositoryStore) FindMergeBase(branch1, branch2 Branch) (Commit, bool, error) {
	if branch1.Tip == nil || branch2.Tip == nil {
		return Commit{}, false, nil
	}
	return s.FindMergeBaseFromCommits(*branch1.Tip, *branch2.Tip)
}

// FindMergeBaseFromCommits returns the merge base of two commits.
func (s *RepositoryStore) FindMergeBaseFromCommits(commit1, commit2 Commit) (Commit, bool, error) {
	sha, err := s.repo.FindMergeBase(commit1.Sha, commit2.Sha)
	if err != nil {
		return Commit{}, false, fmt.Errorf("finding merge base: %w", err)
	}
	if sha == "" {
		return Commit{}, false, nil
	}

	commit, err := s.repo.CommitFromSha(sha)
	if err != nil {
		return Commit{}, false, fmt.Errorf("loading merge base commit: %w", err)
	}

	return commit, true, nil
}

// FindCommitBranchWasBranchedFrom finds where a branch was forked from a source
// branch. It examines source branches defined in config and returns the branch
// and commit of the closest fork point.
func (s *RepositoryStore) FindCommitBranchWasBranchedFrom(branch Branch, cfg *config.Config, excludedBranches ...Branch) (BranchCommit, error) {
	if branch.Tip == nil {
		return BranchCommit{}, nil
	}

	// Get the branch config to find source branches.
	_, configName, err := cfg.GetBranchConfiguration(branch.FriendlyName())
	if err != nil {
		return BranchCommit{}, fmt.Errorf("getting branch configuration: %w", err)
	}

	bc := cfg.Branches[configName]
	if bc == nil || bc.SourceBranches == nil {
		return BranchCommit{}, nil
	}

	excludedSet := make(map[string]struct{})
	for _, eb := range excludedBranches {
		excludedSet[eb.FriendlyName()] = struct{}{}
	}

	allBranches, err := s.repo.Branches()
	if err != nil {
		return BranchCommit{}, fmt.Errorf("listing branches: %w", err)
	}

	var bestCommit Commit
	var bestBranch Branch
	found := false

	for _, sourceName := range *bc.SourceBranches {
		sourceBC := cfg.Branches[sourceName]
		if sourceBC == nil || sourceBC.Regex == nil {
			continue
		}
		re, err := regexp.Compile(*sourceBC.Regex)
		if err != nil {
			continue
		}

		for _, b := range allBranches {
			if b.IsRemote || b.Tip == nil {
				continue
			}
			if _, excluded := excludedSet[b.FriendlyName()]; excluded {
				continue
			}
			if b.FriendlyName() == branch.FriendlyName() {
				continue
			}
			if !re.MatchString(b.FriendlyName()) {
				continue
			}

			mb, err := s.repo.FindMergeBase(branch.Tip.Sha, b.Tip.Sha)
			if err != nil || mb == "" {
				continue
			}

			commit, err := s.repo.CommitFromSha(mb)
			if err != nil {
				continue
			}

			if !found || commit.When.After(bestCommit.When) {
				bestCommit = commit
				bestBranch = b
				found = true
			}
		}
	}

	if !found {
		return BranchCommit{}, nil
	}

	return BranchCommit{Branch: bestBranch, Commit: bestCommit}, nil
}

// --- Utility ---

// IsCommitOnBranch checks if a commit is reachable from the branch tip.
func (s *RepositoryStore) IsCommitOnBranch(commit Commit, branch Branch) (bool, error) {
	if branch.Tip == nil || commit.IsEmpty() {
		return false, nil
	}

	commits, err := s.repo.CommitLog("", branch.Tip.Sha)
	if err != nil {
		return false, fmt.Errorf("getting commit log: %w", err)
	}

	for _, c := range commits {
		if c.Sha == commit.Sha {
			return true, nil
		}
	}

	return false, nil
}

// GetNumberOfUncommittedChanges returns the number of uncommitted changes.
func (s *RepositoryStore) GetNumberOfUncommittedChanges() (int, error) {
	return s.repo.NumberOfUncommittedChanges()
}
