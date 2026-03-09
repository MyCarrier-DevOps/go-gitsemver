package github

import (
	"sync"
	"testing"

	"github.com/MyCarrier-DevOps/go-gitsemver/internal/git"
	"github.com/stretchr/testify/require"
)

func TestNewCache(t *testing.T) {
	c := newCache()
	require.NotNil(t, c.commits)
	require.NotNil(t, c.tagPeels)
	require.NotNil(t, c.mergeBases)
	require.NotNil(t, c.commitLogs)
	require.False(t, c.branchesFetched)
	require.False(t, c.tagsFetched)
}

func TestCacheBranches(t *testing.T) {
	c := newCache()

	// Not fetched yet.
	branches, ok := c.getBranches()
	require.False(t, ok)
	require.Nil(t, branches)

	// Put and get.
	expected := []git.Branch{
		{Name: git.NewBranchReferenceName("main")},
		{Name: git.NewBranchReferenceName("develop")},
	}
	c.putBranches(expected)

	branches, ok = c.getBranches()
	require.True(t, ok)
	require.Equal(t, expected, branches)
}

func TestCacheBranchesEmpty(t *testing.T) {
	c := newCache()
	c.putBranches([]git.Branch{})

	branches, ok := c.getBranches()
	require.True(t, ok)
	require.Empty(t, branches)
}

func TestCacheTags(t *testing.T) {
	c := newCache()

	tags, ok := c.getTags()
	require.False(t, ok)
	require.Nil(t, tags)

	expected := []git.Tag{
		{Name: git.NewReferenceName("refs/tags/v1.0.0"), TargetSha: "abc123"},
	}
	c.putTags(expected)

	tags, ok = c.getTags()
	require.True(t, ok)
	require.Equal(t, expected, tags)
}

func TestCacheCommit(t *testing.T) {
	c := newCache()

	_, ok := c.getCommit("sha1")
	require.False(t, ok)

	commit := git.Commit{Sha: "sha1", Message: "initial"}
	c.putCommit(commit)

	got, ok := c.getCommit("sha1")
	require.True(t, ok)
	require.Equal(t, commit, got)

	// Different SHA still missing.
	_, ok = c.getCommit("sha2")
	require.False(t, ok)
}

func TestCacheTagPeel(t *testing.T) {
	c := newCache()

	_, ok := c.getTagPeel("tag-sha")
	require.False(t, ok)

	c.putTagPeel("tag-sha", "commit-sha")

	got, ok := c.getTagPeel("tag-sha")
	require.True(t, ok)
	require.Equal(t, "commit-sha", got)
}

func TestCacheMergeBase(t *testing.T) {
	c := newCache()

	_, ok := c.getMergeBase("sha1", "sha2")
	require.False(t, ok)

	c.putMergeBase("sha1", "sha2", "base-sha")

	// Same order.
	got, ok := c.getMergeBase("sha1", "sha2")
	require.True(t, ok)
	require.Equal(t, "base-sha", got)

	// Reversed order should also hit cache (sorted key).
	got, ok = c.getMergeBase("sha2", "sha1")
	require.True(t, ok)
	require.Equal(t, "base-sha", got)
}

func TestCacheCommitLog(t *testing.T) {
	c := newCache()

	_, ok := c.getCommitLog("from:to")
	require.False(t, ok)

	commits := []git.Commit{{Sha: "a"}, {Sha: "b"}}
	c.putCommitLog("from:to", commits)

	got, ok := c.getCommitLog("from:to")
	require.True(t, ok)
	require.Equal(t, commits, got)
}

func TestCacheHead(t *testing.T) {
	c := newCache()

	head, ok := c.getHead()
	require.False(t, ok)
	require.Nil(t, head)

	branch := git.Branch{Name: git.NewBranchReferenceName("main")}
	c.putHead(branch)

	head, ok = c.getHead()
	require.True(t, ok)
	require.Equal(t, &branch, head)
}

func TestMergeBaseKey(t *testing.T) {
	// Deterministic regardless of order.
	require.Equal(t, mergeBaseKey("aaa", "bbb"), mergeBaseKey("bbb", "aaa"))
	require.Equal(t, "aaa:bbb", mergeBaseKey("aaa", "bbb"))
	require.Equal(t, "aaa:bbb", mergeBaseKey("bbb", "aaa"))

	// Same SHA.
	require.Equal(t, "x:x", mergeBaseKey("x", "x"))
}

func TestCommitLogKey(t *testing.T) {
	require.Equal(t, "from:to", commitLogKey("from", "to"))
	require.Equal(t, "from:to:path/filter", commitLogKey("from", "to", "path/filter"))
	require.Equal(t, "from:to:a:b", commitLogKey("from", "to", "a", "b"))

	// Empty filters are skipped.
	require.Equal(t, "from:to", commitLogKey("from", "to", ""))
	require.Equal(t, "from:to:a", commitLogKey("from", "to", "", "a", ""))
}

func TestCacheConcurrentAccess(t *testing.T) {
	c := newCache()
	var wg sync.WaitGroup

	// Concurrent writes and reads should not race.
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			c.putCommit(git.Commit{Sha: "sha" + string(rune('a'+i%26))})
		}(i)
		go func(i int) {
			defer wg.Done()
			c.getCommit("sha" + string(rune('a'+i%26)))
		}(i)
	}
	wg.Wait()
}
