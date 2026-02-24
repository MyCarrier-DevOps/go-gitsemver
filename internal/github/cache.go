package github

import (
	"go-gitsemver/internal/git"
	"sort"
	"strings"
	"sync"
)

// apiCache provides in-memory caching for GitHub API responses.
// All fields are protected by a read-write mutex for concurrent safety.
// Caches have a single-run lifetime (not persisted).
type apiCache struct {
	mu sync.RWMutex

	// Ref-level caches (fetched all at once).
	branches        []git.Branch
	tags            []git.Tag
	branchesFetched bool
	tagsFetched     bool

	// SHA-keyed caches.
	commits    map[string]git.Commit   // sha → Commit
	tagPeels   map[string]string       // tag object sha → commit sha
	mergeBases map[string]string       // "sha1:sha2" (sorted) → merge base sha
	commitLogs map[string][]git.Commit // "from:to[:filter]" → commits

	// Head cache.
	headBranch *git.Branch
}

func newCache() *apiCache {
	return &apiCache{
		commits:    make(map[string]git.Commit),
		tagPeels:   make(map[string]string),
		mergeBases: make(map[string]string),
		commitLogs: make(map[string][]git.Commit),
	}
}

// Branches cache.

func (c *apiCache) getBranches() ([]git.Branch, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.branches, c.branchesFetched
}

func (c *apiCache) putBranches(branches []git.Branch) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.branches = branches
	c.branchesFetched = true
}

// Tags cache.

func (c *apiCache) getTags() ([]git.Tag, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tags, c.tagsFetched
}

func (c *apiCache) putTags(tags []git.Tag) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tags = tags
	c.tagsFetched = true
}

// Commit cache.

func (c *apiCache) getCommit(sha string) (git.Commit, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	commit, ok := c.commits[sha]
	return commit, ok
}

func (c *apiCache) putCommit(commit git.Commit) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.commits[commit.Sha] = commit
}

// Tag peel cache (tag object sha → commit sha).

func (c *apiCache) getTagPeel(tagSha string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	sha, ok := c.tagPeels[tagSha]
	return sha, ok
}

func (c *apiCache) putTagPeel(tagSha, commitSha string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tagPeels[tagSha] = commitSha
}

// Merge base cache.

func (c *apiCache) getMergeBase(sha1, sha2 string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	key := mergeBaseKey(sha1, sha2)
	base, ok := c.mergeBases[key]
	return base, ok
}

func (c *apiCache) putMergeBase(sha1, sha2, base string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := mergeBaseKey(sha1, sha2)
	c.mergeBases[key] = base
}

// Commit log cache.

func (c *apiCache) getCommitLog(key string) ([]git.Commit, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	log, ok := c.commitLogs[key]
	return log, ok
}

func (c *apiCache) putCommitLog(key string, commits []git.Commit) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.commitLogs[key] = commits
}

// Head cache.

func (c *apiCache) getHead() (*git.Branch, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.headBranch, c.headBranch != nil
}

func (c *apiCache) putHead(branch git.Branch) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.headBranch = &branch
}

// mergeBaseKey returns a deterministic cache key for two SHAs.
func mergeBaseKey(sha1, sha2 string) string {
	pair := []string{sha1, sha2}
	sort.Strings(pair)
	return pair[0] + ":" + pair[1]
}

// commitLogKey returns a cache key for a commit log query.
func commitLogKey(from, to string, filters ...git.PathFilter) string {
	var b strings.Builder
	b.WriteString(from)
	b.WriteByte(':')
	b.WriteString(to)
	for _, f := range filters {
		if f != "" {
			b.WriteByte(':')
			b.WriteString(string(f))
		}
	}
	return b.String()
}
