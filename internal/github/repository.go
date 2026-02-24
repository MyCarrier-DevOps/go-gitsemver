package github

import (
	"context"
	"fmt"
	"go-gitsemver/internal/git"
	"regexp"
	"time"

	gh "github.com/google/go-github/v68/github"
)

// Compile-time check that GitHubRepository implements git.Repository.
var _ git.Repository = (*GitHubRepository)(nil)

const defaultMaxCommits = 1000

// GitHubRepository implements git.Repository using the GitHub API.
type GitHubRepository struct {
	client     *gh.Client
	owner      string
	repo       string
	ref        string // target ref (branch name, tag, or SHA)
	baseURL    string // custom API base URL for GHE
	maxCommits int    // hard cap on commit walk depth
	cache      *apiCache
	ctx        context.Context // request context
	// versionTagSHAs is populated by Tags() and used by CommitLog for early termination.
	versionTagSHAs map[string]bool
}

// Option configures a GitHubRepository.
type Option func(*GitHubRepository)

// WithRef sets the target ref for HEAD resolution.
func WithRef(ref string) Option {
	return func(r *GitHubRepository) { r.ref = ref }
}

// WithMaxCommits sets the hard cap on commit walk depth.
func WithMaxCommits(n int) Option {
	return func(r *GitHubRepository) { r.maxCommits = n }
}

// WithBaseURL sets the GitHub API base URL for GitHub Enterprise.
func WithBaseURL(url string) Option {
	return func(r *GitHubRepository) { r.baseURL = url }
}

// NewGitHubRepository creates a new GitHubRepository.
func NewGitHubRepository(client *gh.Client, owner, repo string, opts ...Option) *GitHubRepository {
	r := &GitHubRepository{
		client:         client,
		owner:          owner,
		repo:           repo,
		maxCommits:     defaultMaxCommits,
		cache:          newCache(),
		versionTagSHAs: make(map[string]bool),
		ctx:            context.Background(),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func (r *GitHubRepository) Path() string {
	return fmt.Sprintf("github.com/%s/%s", r.owner, r.repo)
}

func (r *GitHubRepository) WorkingDirectory() string {
	return ""
}

var hexPattern = regexp.MustCompile(`^[0-9a-f]{40}$`)

func (r *GitHubRepository) IsHeadDetached() bool {
	return hexPattern.MatchString(r.ref)
}

func (r *GitHubRepository) Head() (git.Branch, error) {
	if branch, ok := r.cache.getHead(); ok {
		return *branch, nil
	}

	ref := r.ref

	if ref == "" {
		// Fetch the repository's default branch.
		repoInfo, _, err := r.client.Repositories.Get(r.ctx, r.owner, r.repo)
		if err != nil {
			return git.Branch{}, fmt.Errorf("getting repository info: %w", err)
		}
		ref = repoInfo.GetDefaultBranch()
	}

	// If ref is a SHA, build a detached head.
	if hexPattern.MatchString(ref) {
		commit, err := r.CommitFromSha(ref)
		if err != nil {
			return git.Branch{}, fmt.Errorf("getting HEAD commit: %w", err)
		}
		branch := git.Branch{
			Name:           git.NewReferenceName("HEAD"),
			Tip:            &commit,
			IsRemote:       false,
			IsDetachedHead: true,
		}
		r.cache.putHead(branch)
		return branch, nil
	}

	// Fetch the branch tip.
	ghBranch, _, err := r.client.Repositories.GetBranch(r.ctx, r.owner, r.repo, ref, 0)
	if err != nil {
		return git.Branch{}, fmt.Errorf("getting branch %s: %w", ref, err)
	}

	tipCommit := convertGitHubCommit(ghBranch.GetCommit())
	r.cache.putCommit(tipCommit)

	branch := git.Branch{
		Name:     git.NewBranchReferenceName(ref),
		Tip:      &tipCommit,
		IsRemote: false,
	}
	r.cache.putHead(branch)
	return branch, nil
}

func (r *GitHubRepository) Branches(_ ...git.PathFilter) ([]git.Branch, error) {
	if branches, ok := r.cache.getBranches(); ok {
		return branches, nil
	}

	branches, err := r.fetchAllBranchesGraphQL()
	if err != nil {
		return nil, err
	}

	r.cache.putBranches(branches)
	return branches, nil
}

func (r *GitHubRepository) Tags(_ ...git.PathFilter) ([]git.Tag, error) {
	if tags, ok := r.cache.getTags(); ok {
		return tags, nil
	}

	tags, err := r.fetchAllTagsGraphQL()
	if err != nil {
		return nil, err
	}

	// Build the versionTagSHAs set for early termination in CommitLog.
	// Every commit SHA that a tag resolves to is a potential stop point.
	for _, tag := range tags {
		if commitSha, ok := r.cache.getTagPeel(tag.TargetSha); ok {
			r.versionTagSHAs[commitSha] = true
		}
	}

	r.cache.putTags(tags)
	return tags, nil
}

func (r *GitHubRepository) CommitFromSha(sha string) (git.Commit, error) {
	if commit, ok := r.cache.getCommit(sha); ok {
		return commit, nil
	}

	ghCommit, _, err := r.client.Repositories.GetCommit(r.ctx, r.owner, r.repo, sha, nil)
	if err != nil {
		return git.Commit{}, fmt.Errorf("getting commit %s: %w", sha, err)
	}

	commit := convertGitHubRepoCommit(ghCommit)
	r.cache.putCommit(commit)
	return commit, nil
}

func (r *GitHubRepository) CommitLog(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
	key := commitLogKey(from, to, filters...)
	if log, ok := r.cache.getCommitLog(key); ok {
		return log, nil
	}

	var commits []git.Commit
	var err error

	if from != "" {
		// Bounded range: try compare API first.
		commits, err = r.commitLogCompare(from, to)
		if err != nil {
			// Fall back to paginated walk if compare fails (e.g., > 250 commits).
			commits, err = r.commitLogPaginated(from, to, filters...)
		}
	} else {
		// Full history walk with smart early termination.
		commits, err = r.commitLogPaginated(from, to, filters...)
	}
	if err != nil {
		return nil, err
	}

	r.cache.putCommitLog(key, commits)
	return commits, nil
}

// commitLogCompare uses the compare API for bounded commit ranges.
func (r *GitHubRepository) commitLogCompare(from, to string) ([]git.Commit, error) {
	comparison, _, err := r.client.Repositories.CompareCommits(r.ctx, r.owner, r.repo, from, to, nil)
	if err != nil {
		return nil, fmt.Errorf("comparing commits: %w", err)
	}

	// Compare API returns max 250 commits. If there are more, return error to trigger fallback.
	if comparison.GetTotalCommits() > len(comparison.Commits) {
		return nil, fmt.Errorf("compare API returned partial results (%d/%d commits)", len(comparison.Commits), comparison.GetTotalCommits())
	}

	// Compare API returns commits in forward chronological order; reverse them.
	commits := make([]git.Commit, 0, len(comparison.Commits))
	for i := len(comparison.Commits) - 1; i >= 0; i-- {
		commit := convertGitHubRepoCommit(comparison.Commits[i])
		r.cache.putCommit(commit)
		commits = append(commits, commit)
	}

	return commits, nil
}

// commitLogPaginated walks commits page-by-page with smart early termination.
func (r *GitHubRepository) commitLogPaginated(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
	opts := &gh.CommitsListOptions{
		SHA: to,
		ListOptions: gh.ListOptions{
			PerPage: 100,
		},
	}

	// Apply path filter for monorepo support.
	for _, f := range filters {
		if f != "" {
			opts.Path = string(f)
			break // GitHub API only supports one path filter.
		}
	}

	var commits []git.Commit
	foundTag := false
	bufferPages := 0

	for {
		ghCommits, resp, err := r.client.Repositories.ListCommits(r.ctx, r.owner, r.repo, opts)
		if err != nil {
			return nil, fmt.Errorf("listing commits: %w", err)
		}

		for _, ghCommit := range ghCommits {
			sha := ghCommit.GetSHA()

			// Stop if we've reached the 'from' boundary.
			if from != "" && sha == from {
				goto done
			}

			commit := convertGitHubRepoCommit(ghCommit)
			r.cache.putCommit(commit)
			commits = append(commits, commit)

			// Check for early termination: is this commit tagged?
			if r.versionTagSHAs[sha] {
				foundTag = true
			}
		}

		// Hard cap on total commits.
		if len(commits) >= r.maxCommits {
			break
		}

		// Smart early termination: if we found a tag, allow one more buffer page.
		if foundTag {
			bufferPages++
			if bufferPages > 1 {
				break
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

done:
	return commits, nil
}

func (r *GitHubRepository) MainlineCommitLog(from, to string, filters ...git.PathFilter) ([]git.Commit, error) {
	// Get full commit log, then filter to first-parent only.
	allCommits, err := r.CommitLog(from, to, filters...)
	if err != nil {
		return nil, err
	}

	if len(allCommits) == 0 {
		return nil, nil
	}

	// Build a first-parent chain starting from the first commit (newest).
	commitMap := make(map[string]git.Commit, len(allCommits))
	for _, c := range allCommits {
		commitMap[c.Sha] = c
	}

	var mainline []git.Commit
	current := allCommits[0]
	for {
		mainline = append(mainline, current)
		if len(current.Parents) == 0 {
			break
		}
		firstParent := current.Parents[0]
		if from != "" && firstParent == from {
			break
		}
		next, ok := commitMap[firstParent]
		if !ok {
			break
		}
		current = next
	}

	return mainline, nil
}

func (r *GitHubRepository) BranchCommits(branch git.Branch, filters ...git.PathFilter) ([]git.Commit, error) {
	if branch.Tip == nil {
		return nil, nil
	}
	return r.CommitLog("", branch.Tip.Sha, filters...)
}

func (r *GitHubRepository) CommitsPriorTo(olderThan time.Time, branch git.Branch) ([]git.Commit, error) {
	if branch.Tip == nil {
		return nil, nil
	}

	opts := &gh.CommitsListOptions{
		SHA:   branch.Tip.Sha,
		Until: olderThan,
		ListOptions: gh.ListOptions{
			PerPage: 100,
		},
	}

	var commits []git.Commit
	for {
		ghCommits, resp, err := r.client.Repositories.ListCommits(r.ctx, r.owner, r.repo, opts)
		if err != nil {
			return nil, fmt.Errorf("listing commits prior to %s: %w", olderThan, err)
		}

		for _, ghCommit := range ghCommits {
			commit := convertGitHubRepoCommit(ghCommit)
			r.cache.putCommit(commit)
			commits = append(commits, commit)
		}

		if len(commits) >= r.maxCommits || resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return commits, nil
}

func (r *GitHubRepository) FindMergeBase(sha1, sha2 string) (string, error) {
	if base, ok := r.cache.getMergeBase(sha1, sha2); ok {
		return base, nil
	}

	comparison, _, err := r.client.Repositories.CompareCommits(r.ctx, r.owner, r.repo, sha1, sha2, nil)
	if err != nil {
		return "", fmt.Errorf("comparing commits for merge base: %w", err)
	}

	base := ""
	if comparison.MergeBaseCommit != nil {
		base = comparison.MergeBaseCommit.GetSHA()
	}

	r.cache.putMergeBase(sha1, sha2, base)
	return base, nil
}

func (r *GitHubRepository) BranchesContainingCommit(sha string) ([]git.Branch, error) {
	branches, err := r.Branches()
	if err != nil {
		return nil, err
	}

	var result []git.Branch

	for _, b := range branches {
		if b.Tip == nil {
			continue
		}

		// Direct match: commit is the branch tip.
		if b.Tip.Sha == sha {
			result = append(result, b)
			continue
		}

		// Check ancestry via compare API.
		comparison, _, err := r.client.Repositories.CompareCommits(r.ctx, r.owner, r.repo, sha, b.Tip.Sha, nil)
		if err != nil {
			continue // skip branches we can't compare
		}

		// If status is "ahead" or "identical", the branch contains the commit.
		status := comparison.GetStatus()
		if status == "ahead" || status == "identical" {
			result = append(result, b)
		}
	}

	return result, nil
}

func (r *GitHubRepository) NumberOfUncommittedChanges() (int, error) {
	return 0, nil
}

func (r *GitHubRepository) PeelTagToCommit(tag git.Tag) (string, error) {
	// Check the pre-populated cache from Tags() GraphQL query.
	if commitSha, ok := r.cache.getTagPeel(tag.TargetSha); ok {
		return commitSha, nil
	}

	// Fallback: try to resolve via the git tags API.

	tagObj, _, err := r.client.Git.GetTag(r.ctx, r.owner, r.repo, tag.TargetSha)
	if err == nil && tagObj.GetObject() != nil {
		commitSha := tagObj.GetObject().GetSHA()
		r.cache.putTagPeel(tag.TargetSha, commitSha)
		return commitSha, nil
	}

	// If it's not an annotated tag object, it's a lightweight tag pointing directly to a commit.
	r.cache.putTagPeel(tag.TargetSha, tag.TargetSha)
	return tag.TargetSha, nil
}

// FetchFileContent fetches a file's content from the repository.
// Used to load configuration files from the remote repository.
func (r *GitHubRepository) FetchFileContent(path string) (string, error) {
	opts := &gh.RepositoryContentGetOptions{}
	if r.ref != "" {
		opts.Ref = r.ref
	}

	content, _, _, err := r.client.Repositories.GetContents(r.ctx, r.owner, r.repo, path, opts)
	if err != nil {
		return "", fmt.Errorf("fetching file %s: %w", path, err)
	}
	if content == nil {
		return "", fmt.Errorf("file %s not found", path)
	}

	decoded, err := content.GetContent()
	if err != nil {
		return "", fmt.Errorf("decoding file content: %w", err)
	}
	return decoded, nil
}

// convertGitHubRepoCommit converts a GitHub API RepositoryCommit to a git.Commit.
func convertGitHubRepoCommit(ghCommit *gh.RepositoryCommit) git.Commit {
	if ghCommit == nil {
		return git.Commit{}
	}

	var parents []string
	for _, p := range ghCommit.Parents {
		parents = append(parents, p.GetSHA())
	}

	var when time.Time
	var message string
	if ghCommit.Commit != nil {
		if ghCommit.Commit.Committer != nil && ghCommit.Commit.Committer.Date != nil {
			when = ghCommit.Commit.Committer.Date.Time
		}
		message = ghCommit.Commit.GetMessage()
	}

	return git.Commit{
		Sha:     ghCommit.GetSHA(),
		Parents: parents,
		When:    when,
		Message: message,
	}
}

// convertGitHubCommit converts a GitHub API Commit (from branch endpoint) to a git.Commit.
func convertGitHubCommit(ghCommit *gh.RepositoryCommit) git.Commit {
	return convertGitHubRepoCommit(ghCommit)
}
