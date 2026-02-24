package github

import (
	"encoding/json"
	"fmt"
	"go-gitsemver/internal/git"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gh "github.com/google/go-github/v68/github"
	"github.com/stretchr/testify/require"
)

// writeJSON encodes v as JSON to the response writer. Panics on error (test only).
func writeJSON(w http.ResponseWriter, v interface{}) {
	if err := json.NewEncoder(w).Encode(v); err != nil {
		panic(err)
	}
}

// newTestServer creates a test HTTP server and a GitHub client pointed at it.
func newTestServer(t *testing.T, mux *http.ServeMux) (*gh.Client, func()) {
	t.Helper()
	server := httptest.NewServer(mux)
	client, err := gh.NewClient(nil).WithEnterpriseURLs(server.URL+"/", server.URL+"/")
	require.NoError(t, err)
	return client, server.Close
}

// newTestRepo creates a GitHubRepository backed by a test server.
func newTestRepo(t *testing.T, mux *http.ServeMux, opts ...Option) (*GitHubRepository, func()) {
	t.Helper()
	client, cleanup := newTestServer(t, mux)
	repo := NewGitHubRepository(client, "testowner", "testrepo", opts...)
	repo.baseURL = "" // graphql won't be used in REST-only tests
	return repo, cleanup
}

func TestPath(t *testing.T) {
	repo := NewGitHubRepository(nil, "myorg", "myrepo")
	require.Equal(t, "github.com/myorg/myrepo", repo.Path())
}

func TestWorkingDirectory(t *testing.T) {
	repo := NewGitHubRepository(nil, "myorg", "myrepo")
	require.Equal(t, "", repo.WorkingDirectory())
}

func TestIsHeadDetached(t *testing.T) {
	tests := []struct {
		ref      string
		detached bool
	}{
		{"main", false},
		{"feature/auth", false},
		{"", false},
		{"abc1234", false}, // too short
		{"abcdef1234567890abcdef1234567890abcdef12", true},
	}
	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			repo := NewGitHubRepository(nil, "o", "r", WithRef(tt.ref))
			require.Equal(t, tt.detached, repo.IsHeadDetached())
		})
	}
}

func TestNumberOfUncommittedChanges(t *testing.T) {
	repo := NewGitHubRepository(nil, "o", "r")
	n, err := repo.NumberOfUncommittedChanges()
	require.NoError(t, err)
	require.Equal(t, 0, n)
}

func TestHead_DefaultBranch(t *testing.T) {
	mux := http.NewServeMux()

	// GET /api/v3/repos/testowner/testrepo
	mux.HandleFunc("/api/v3/repos/testowner/testrepo", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, map[string]interface{}{
			"default_branch": "main",
		})
	})

	// GET /api/v3/repos/testowner/testrepo/branches/main
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/branches/main", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{
			"name": "main",
			"commit": map[string]interface{}{
				"sha": "abcdef1234567890abcdef1234567890abcdef12",
				"commit": map[string]interface{}{
					"message": "initial commit",
					"committer": map[string]interface{}{
						"date": "2025-01-15T00:00:00Z",
					},
				},
				"parents": []map[string]interface{}{},
			},
		})
	})

	repo, cleanup := newTestRepo(t, mux)
	defer cleanup()

	head, err := repo.Head()
	require.NoError(t, err)
	require.Equal(t, "main", head.FriendlyName())
	require.Equal(t, "abcdef1234567890abcdef1234567890abcdef12", head.Tip.Sha)
	require.False(t, head.IsDetachedHead)

	// Second call should hit cache.
	head2, err := repo.Head()
	require.NoError(t, err)
	require.Equal(t, head.Tip.Sha, head2.Tip.Sha)
}

func TestHead_ExplicitRef(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v3/repos/testowner/testrepo/branches/develop", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{
			"name": "develop",
			"commit": map[string]interface{}{
				"sha": "def5678abc1234567890123456789012345abcdef",
				"commit": map[string]interface{}{
					"message": "dev commit",
					"committer": map[string]interface{}{
						"date": "2025-06-01T00:00:00Z",
					},
				},
				"parents": []map[string]interface{}{},
			},
		})
	})

	repo, cleanup := newTestRepo(t, mux, WithRef("develop"))
	defer cleanup()

	head, err := repo.Head()
	require.NoError(t, err)
	require.Equal(t, "develop", head.FriendlyName())
}

func TestCommitFromSha(t *testing.T) {
	sha := "abcdef1234567890abcdef1234567890abcdef12"
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v3/repos/testowner/testrepo/commits/"+sha, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{
			"sha": sha,
			"commit": map[string]interface{}{
				"message": "test commit",
				"committer": map[string]interface{}{
					"date": "2025-03-15T14:30:00Z",
				},
			},
			"parents": []map[string]interface{}{
				{"sha": "parent1111111111111111111111111111111111111"},
			},
		})
	})

	repo, cleanup := newTestRepo(t, mux)
	defer cleanup()

	commit, err := repo.CommitFromSha(sha)
	require.NoError(t, err)
	require.Equal(t, sha, commit.Sha)
	require.Equal(t, "test commit", commit.Message)
	require.Len(t, commit.Parents, 1)
	require.Equal(t, "parent1111111111111111111111111111111111111", commit.Parents[0])
	require.Equal(t, 2025, commit.When.Year())

	// Second call should hit cache.
	commit2, err := repo.CommitFromSha(sha)
	require.NoError(t, err)
	require.Equal(t, commit.Sha, commit2.Sha)
}

func TestFindMergeBase(t *testing.T) {
	sha1 := "aaa1111111111111111111111111111111111111"
	sha2 := "bbb2222222222222222222222222222222222222"
	baseSha := "ccc3333333333333333333333333333333333333"

	mux := http.NewServeMux()

	mux.HandleFunc("/api/v3/repos/testowner/testrepo/compare/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{
			"merge_base_commit": map[string]interface{}{
				"sha": baseSha,
			},
		})
	})

	repo, cleanup := newTestRepo(t, mux)
	defer cleanup()

	base, err := repo.FindMergeBase(sha1, sha2)
	require.NoError(t, err)
	require.Equal(t, baseSha, base)

	// Cached (key is sorted, so reversing args still hits cache).
	base2, err := repo.FindMergeBase(sha2, sha1)
	require.NoError(t, err)
	require.Equal(t, baseSha, base2)
}

func TestPeelTagToCommit_Cached(t *testing.T) {
	repo := NewGitHubRepository(nil, "o", "r")
	tagSha := "tag1111111111111111111111111111111111111"
	commitSha := "commit2222222222222222222222222222222222"
	repo.cache.putTagPeel(tagSha, commitSha)

	result, err := repo.PeelTagToCommit(git.Tag{
		Name:      git.NewReferenceName("refs/tags/v1.0.0"),
		TargetSha: tagSha,
	})
	require.NoError(t, err)
	require.Equal(t, commitSha, result)
}

func TestCommitLog_Compare(t *testing.T) {
	fromSha := "from1111111111111111111111111111111111111"
	toSha := "to22222222222222222222222222222222222222"

	mux := http.NewServeMux()

	mux.HandleFunc("/api/v3/repos/testowner/testrepo/compare/", func(w http.ResponseWriter, r *http.Request) {
		// Return 3 commits in forward order.
		writeJSON(w, map[string]interface{}{
			"total_commits": 3,
			"commits": []map[string]interface{}{
				{
					"sha":     "commit_a",
					"commit":  map[string]interface{}{"message": "first", "committer": map[string]interface{}{"date": "2025-01-01T00:00:00Z"}},
					"parents": []map[string]interface{}{},
				},
				{
					"sha":     "commit_b",
					"commit":  map[string]interface{}{"message": "second", "committer": map[string]interface{}{"date": "2025-01-02T00:00:00Z"}},
					"parents": []map[string]interface{}{{"sha": "commit_a"}},
				},
				{
					"sha":     "commit_c",
					"commit":  map[string]interface{}{"message": "third", "committer": map[string]interface{}{"date": "2025-01-03T00:00:00Z"}},
					"parents": []map[string]interface{}{{"sha": "commit_b"}},
				},
			},
		})
	})

	repo, cleanup := newTestRepo(t, mux)
	defer cleanup()

	commits, err := repo.CommitLog(fromSha, toSha)
	require.NoError(t, err)
	require.Len(t, commits, 3)
	// Should be in reverse chronological order.
	require.Equal(t, "commit_c", commits[0].Sha)
	require.Equal(t, "commit_b", commits[1].Sha)
	require.Equal(t, "commit_a", commits[2].Sha)
}

func TestCommitLog_Paginated_EarlyTermination(t *testing.T) {
	mux := http.NewServeMux()
	page := 0

	mux.HandleFunc("/api/v3/repos/testowner/testrepo/commits", func(w http.ResponseWriter, r *http.Request) {
		page++
		var commits []map[string]interface{}
		for i := 0; i < 3; i++ {
			sha := fmt.Sprintf("commit_%d_%d", page, i)
			commits = append(commits, map[string]interface{}{
				"sha":     sha,
				"commit":  map[string]interface{}{"message": sha, "committer": map[string]interface{}{"date": "2025-01-01T00:00:00Z"}},
				"parents": []map[string]interface{}{},
			})
		}

		// Simulate 3 pages total.
		if page < 3 {
			w.Header().Set("Link", fmt.Sprintf(`<http://next?page=%d>; rel="next"`, page+1))
		}

		writeJSON(w, commits)
	})

	repo, cleanup := newTestRepo(t, mux, WithMaxCommits(100))
	defer cleanup()

	// Pre-populate versionTagSHAs so early termination kicks in.
	repo.versionTagSHAs["commit_1_2"] = true

	commits, err := repo.CommitLog("", "HEAD")
	require.NoError(t, err)
	// Should have fetched page 1 (found tag at commit_1_2) + 1 buffer page = 6 commits.
	require.Equal(t, 6, len(commits))
	// Should have stopped after page 2 (buffer).
	require.Equal(t, 2, page)
}

func TestCommitLog_Paginated_MaxCommitsCap(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v3/repos/testowner/testrepo/commits", func(w http.ResponseWriter, r *http.Request) {
		var commits []map[string]interface{}
		for i := 0; i < 100; i++ {
			sha := fmt.Sprintf("commit_%d", i)
			commits = append(commits, map[string]interface{}{
				"sha":     sha,
				"commit":  map[string]interface{}{"message": sha, "committer": map[string]interface{}{"date": "2025-01-01T00:00:00Z"}},
				"parents": []map[string]interface{}{},
			})
		}
		// Always claim there's a next page.
		w.Header().Set("Link", `<http://next?page=99>; rel="next"`)
		writeJSON(w, commits)
	})

	repo, cleanup := newTestRepo(t, mux, WithMaxCommits(50))
	defer cleanup()

	commits, err := repo.CommitLog("", "HEAD")
	require.NoError(t, err)
	// maxCommits=50 but we get 100 per page; should stop after first page (100 >= 50).
	require.LessOrEqual(t, 50, len(commits))
}

func TestMainlineCommitLog(t *testing.T) {
	mux := http.NewServeMux()

	// Simulate a merge commit scenario.
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/compare/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{
			"total_commits": 4,
			"commits": []map[string]interface{}{
				// A (root) -> B -> C (merge of D) -> E
				{"sha": "aaa", "commit": map[string]interface{}{"message": "A", "committer": map[string]interface{}{"date": "2025-01-01T00:00:00Z"}}, "parents": []map[string]interface{}{}},
				{"sha": "bbb", "commit": map[string]interface{}{"message": "B", "committer": map[string]interface{}{"date": "2025-01-02T00:00:00Z"}}, "parents": []map[string]interface{}{{"sha": "aaa"}}},
				{"sha": "ddd", "commit": map[string]interface{}{"message": "D (side branch)", "committer": map[string]interface{}{"date": "2025-01-02T12:00:00Z"}}, "parents": []map[string]interface{}{{"sha": "aaa"}}},
				{"sha": "ccc", "commit": map[string]interface{}{"message": "C (merge)", "committer": map[string]interface{}{"date": "2025-01-03T00:00:00Z"}}, "parents": []map[string]interface{}{{"sha": "bbb"}, {"sha": "ddd"}}},
			},
		})
	})

	repo, cleanup := newTestRepo(t, mux)
	defer cleanup()

	// Full log returns all 4 commits (reversed).
	commits, err := repo.CommitLog("base", "tip")
	require.NoError(t, err)
	require.Len(t, commits, 4)

	// Mainline should follow first-parent: ccc -> bbb -> aaa (skip ddd).
	mainline, err := repo.MainlineCommitLog("base", "tip")
	require.NoError(t, err)
	require.Len(t, mainline, 3)
	require.Equal(t, "ccc", mainline[0].Sha)
	require.Equal(t, "bbb", mainline[1].Sha)
	require.Equal(t, "aaa", mainline[2].Sha)
}

func TestCommitsPriorTo(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v3/repos/testowner/testrepo/commits", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, []map[string]interface{}{
			{
				"sha":     "old_commit",
				"commit":  map[string]interface{}{"message": "old", "committer": map[string]interface{}{"date": "2024-01-01T00:00:00Z"}},
				"parents": []map[string]interface{}{},
			},
		})
	})

	repo, cleanup := newTestRepo(t, mux)
	defer cleanup()

	tip := git.Commit{Sha: "tip123"}
	branch := git.Branch{
		Name: git.NewBranchReferenceName("main"),
		Tip:  &tip,
	}

	commits, err := repo.CommitsPriorTo(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), branch)
	require.NoError(t, err)
	require.Len(t, commits, 1)
	require.Equal(t, "old_commit", commits[0].Sha)
}

func TestFetchFileContent(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v3/repos/testowner/testrepo/contents/gitsemver.yml", func(w http.ResponseWriter, r *http.Request) {
		// GitHub Contents API returns base64-encoded content.
		writeJSON(w, map[string]interface{}{
			"type":     "file",
			"encoding": "base64",
			"content":  "bW9kZTogTWFpbmxpbmU=", // base64 of "mode: Mainline"
		})
	})

	repo, cleanup := newTestRepo(t, mux)
	defer cleanup()

	content, err := repo.FetchFileContent("gitsemver.yml")
	require.NoError(t, err)
	require.Equal(t, "mode: Mainline", content)
}

func TestFetchFileContent_NotFound(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v3/repos/testowner/testrepo/contents/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message": "Not Found"}`, http.StatusNotFound)
	})

	repo, cleanup := newTestRepo(t, mux)
	defer cleanup()

	_, err := repo.FetchFileContent("nonexistent.yml")
	require.Error(t, err)
}

func TestCacheHits(t *testing.T) {
	// Verify that repeated calls to the same method return cached data
	// and don't make additional API calls.
	callCount := 0
	mux := http.NewServeMux()

	sha := "abcdef1234567890abcdef1234567890abcdef12"
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/commits/"+sha, func(w http.ResponseWriter, r *http.Request) {
		callCount++
		writeJSON(w, map[string]interface{}{
			"sha":     sha,
			"commit":  map[string]interface{}{"message": "cached", "committer": map[string]interface{}{"date": "2025-01-01T00:00:00Z"}},
			"parents": []map[string]interface{}{},
		})
	})

	repo, cleanup := newTestRepo(t, mux)
	defer cleanup()

	// First call hits API.
	_, err := repo.CommitFromSha(sha)
	require.NoError(t, err)
	require.Equal(t, 1, callCount)

	// Second call should be cached.
	_, err = repo.CommitFromSha(sha)
	require.NoError(t, err)
	require.Equal(t, 1, callCount) // still 1
}

func TestBranchCommits_NilTip(t *testing.T) {
	repo := NewGitHubRepository(nil, "o", "r")
	commits, err := repo.BranchCommits(git.Branch{Name: git.NewBranchReferenceName("main")})
	require.NoError(t, err)
	require.Nil(t, commits)
}

// newTestRepoWithGraphQL creates a GitHubRepository with baseURL set so GraphQL
// calls are routed to the test server.
func newTestRepoWithGraphQL(t *testing.T, mux *http.ServeMux, opts ...Option) (*GitHubRepository, string, func()) {
	t.Helper()
	server := httptest.NewServer(mux)
	client, err := gh.NewClient(nil).WithEnterpriseURLs(server.URL+"/", server.URL+"/")
	require.NoError(t, err)
	repo := NewGitHubRepository(client, "testowner", "testrepo", opts...)
	repo.baseURL = server.URL // routes GraphQL to test server
	return repo, server.URL, server.Close
}

// graphQLBranchesResponse returns a JSON GraphQL response containing branches.
func graphQLBranchesResponse(branches []map[string]interface{}, hasNextPage bool, endCursor string) map[string]interface{} {
	return map[string]interface{}{
		"data": map[string]interface{}{
			"repository": map[string]interface{}{
				"refs": map[string]interface{}{
					"nodes": branches,
					"pageInfo": map[string]interface{}{
						"hasNextPage": hasNextPage,
						"endCursor":   endCursor,
					},
				},
			},
		},
	}
}

// graphQLTagsResponse returns a JSON GraphQL response containing tags.
func graphQLTagsResponse(tags []map[string]interface{}, hasNextPage bool, endCursor string) map[string]interface{} {
	return map[string]interface{}{
		"data": map[string]interface{}{
			"repository": map[string]interface{}{
				"refs": map[string]interface{}{
					"nodes": tags,
					"pageInfo": map[string]interface{}{
						"hasNextPage": hasNextPage,
						"endCursor":   endCursor,
					},
				},
			},
		},
	}
}

func TestBranches_GraphQL(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
		branches := []map[string]interface{}{
			{
				"name": "main",
				"target": map[string]interface{}{
					"oid":           "aaa1111111111111111111111111111111111111",
					"message":       "init",
					"committedDate": "2025-01-01T00:00:00Z",
					"parents":       map[string]interface{}{"nodes": []interface{}{}},
				},
			},
			{
				"name": "develop",
				"target": map[string]interface{}{
					"oid":           "bbb2222222222222222222222222222222222222",
					"message":       "dev work",
					"committedDate": "2025-02-01T00:00:00Z",
					"parents":       map[string]interface{}{"nodes": []map[string]interface{}{{"oid": "aaa1111111111111111111111111111111111111"}}},
				},
			},
		}
		writeJSON(w, graphQLBranchesResponse(branches, false, ""))
	})

	repo, _, cleanup := newTestRepoWithGraphQL(t, mux)
	defer cleanup()

	result, err := repo.Branches()
	require.NoError(t, err)
	require.Len(t, result, 2)
	require.Equal(t, "main", result[0].FriendlyName())
	require.Equal(t, "develop", result[1].FriendlyName())
	require.NotNil(t, result[0].Tip)
	require.Equal(t, "aaa1111111111111111111111111111111111111", result[0].Tip.Sha)
	require.False(t, result[0].IsRemote)

	// Second call should use cache.
	result2, err := repo.Branches()
	require.NoError(t, err)
	require.Len(t, result2, 2)
}

func TestTags_GraphQL_LightweightAndAnnotated(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
		tags := []map[string]interface{}{
			{
				// Lightweight tag: target is a Commit directly.
				"name": "v1.0.0",
				"target": map[string]interface{}{
					"__typename":    "Commit",
					"oid":           "commit_aaa",
					"message":       "release v1",
					"committedDate": "2025-01-01T00:00:00Z",
					"parents":       map[string]interface{}{"nodes": []interface{}{}},
				},
			},
			{
				// Annotated tag: target is a Tag that peels to a Commit.
				"name": "v2.0.0",
				"target": map[string]interface{}{
					"__typename": "Tag",
					"oid":        "tag_obj_bbb",
					"target": map[string]interface{}{
						"__typename":    "Commit",
						"oid":           "commit_bbb",
						"message":       "release v2",
						"committedDate": "2025-06-01T00:00:00Z",
						"parents":       map[string]interface{}{"nodes": []interface{}{}},
					},
				},
			},
		}
		writeJSON(w, graphQLTagsResponse(tags, false, ""))
	})

	repo, _, cleanup := newTestRepoWithGraphQL(t, mux)
	defer cleanup()

	result, err := repo.Tags()
	require.NoError(t, err)
	require.Len(t, result, 2)

	require.Equal(t, "refs/tags/v1.0.0", result[0].Name.Canonical)
	require.Equal(t, "commit_aaa", result[0].TargetSha) // lightweight: tag sha == commit sha

	require.Equal(t, "refs/tags/v2.0.0", result[1].Name.Canonical)
	require.Equal(t, "tag_obj_bbb", result[1].TargetSha) // annotated: tag sha is the tag object

	// Tag peels should be cached.
	commitSha, ok := repo.cache.getTagPeel("commit_aaa")
	require.True(t, ok)
	require.Equal(t, "commit_aaa", commitSha)

	commitSha2, ok := repo.cache.getTagPeel("tag_obj_bbb")
	require.True(t, ok)
	require.Equal(t, "commit_bbb", commitSha2)

	// versionTagSHAs should be populated for early termination.
	require.True(t, repo.versionTagSHAs["commit_aaa"])
	require.True(t, repo.versionTagSHAs["commit_bbb"])

	// Second call should use cache.
	result2, err := repo.Tags()
	require.NoError(t, err)
	require.Len(t, result2, 2)
}

func TestHead_DetachedSHA(t *testing.T) {
	sha := "abcdef1234567890abcdef1234567890abcdef12"
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v3/repos/testowner/testrepo/commits/"+sha, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{
			"sha":     sha,
			"commit":  map[string]interface{}{"message": "detached commit", "committer": map[string]interface{}{"date": "2025-03-01T00:00:00Z"}},
			"parents": []map[string]interface{}{},
		})
	})

	repo, cleanup := newTestRepo(t, mux, WithRef(sha))
	defer cleanup()

	head, err := repo.Head()
	require.NoError(t, err)
	require.True(t, head.IsDetachedHead)
	require.Equal(t, "HEAD", head.FriendlyName())
	require.Equal(t, sha, head.Tip.Sha)
}

func TestBranchesContainingCommit(t *testing.T) {
	targetSha := "target11111111111111111111111111111111111"

	mux := http.NewServeMux()

	// GraphQL returns 2 branches.
	mux.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
		branches := []map[string]interface{}{
			{
				"name": "main",
				"target": map[string]interface{}{
					"oid":           targetSha, // tip IS the target commit
					"message":       "latest",
					"committedDate": "2025-01-01T00:00:00Z",
					"parents":       map[string]interface{}{"nodes": []interface{}{}},
				},
			},
			{
				"name": "develop",
				"target": map[string]interface{}{
					"oid":           "dev_tip_22222222222222222222222222222222",
					"message":       "dev",
					"committedDate": "2025-02-01T00:00:00Z",
					"parents":       map[string]interface{}{"nodes": []interface{}{}},
				},
			},
		}
		writeJSON(w, graphQLBranchesResponse(branches, false, ""))
	})

	// Compare API: develop is "ahead" of target (contains it).
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/compare/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{
			"status":            "ahead",
			"merge_base_commit": map[string]interface{}{"sha": targetSha},
		})
	})

	repo, _, cleanup := newTestRepoWithGraphQL(t, mux)
	defer cleanup()

	result, err := repo.BranchesContainingCommit(targetSha)
	require.NoError(t, err)
	// main: direct match (tip == sha), develop: ahead → contains
	require.Len(t, result, 2)
}

func TestBranchesContainingCommit_Behind(t *testing.T) {
	targetSha := "target11111111111111111111111111111111111"

	mux := http.NewServeMux()

	mux.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
		branches := []map[string]interface{}{
			{
				"name": "old-branch",
				"target": map[string]interface{}{
					"oid":           "old_tip_222222222222222222222222222222222",
					"message":       "old",
					"committedDate": "2024-01-01T00:00:00Z",
					"parents":       map[string]interface{}{"nodes": []interface{}{}},
				},
			},
		}
		writeJSON(w, graphQLBranchesResponse(branches, false, ""))
	})

	// Compare says "behind" — branch does NOT contain the commit.
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/compare/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{
			"status":            "behind",
			"merge_base_commit": map[string]interface{}{"sha": "some_base"},
		})
	})

	repo, _, cleanup := newTestRepoWithGraphQL(t, mux)
	defer cleanup()

	result, err := repo.BranchesContainingCommit(targetSha)
	require.NoError(t, err)
	require.Empty(t, result)
}

func TestPeelTagToCommit_FallbackAPI(t *testing.T) {
	tagSha := "tag_obj_111111111111111111111111111111111"
	commitSha := "commit_22222222222222222222222222222222222"

	mux := http.NewServeMux()

	// Mock the git tags API for annotated tag resolution.
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/git/tags/"+tagSha, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{
			"sha": tagSha,
			"object": map[string]interface{}{
				"sha":  commitSha,
				"type": "commit",
			},
		})
	})

	repo, cleanup := newTestRepo(t, mux)
	defer cleanup()

	// Don't pre-populate cache — forces the API fallback path.
	result, err := repo.PeelTagToCommit(git.Tag{
		Name:      git.NewReferenceName("refs/tags/v3.0.0"),
		TargetSha: tagSha,
	})
	require.NoError(t, err)
	require.Equal(t, commitSha, result)

	// Should now be cached.
	cached, ok := repo.cache.getTagPeel(tagSha)
	require.True(t, ok)
	require.Equal(t, commitSha, cached)
}

func TestPeelTagToCommit_LightweightFallback(t *testing.T) {
	// When both cache miss and API error occur, treat as lightweight tag.
	tagSha := "lightweight_sha_111111111111111111111111"

	mux := http.NewServeMux()

	// Return 404 for git tags API (it's a lightweight tag, not an annotated tag object).
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/git/tags/"+tagSha, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
	})

	repo, cleanup := newTestRepo(t, mux)
	defer cleanup()

	result, err := repo.PeelTagToCommit(git.Tag{
		Name:      git.NewReferenceName("refs/tags/v1.0.0-light"),
		TargetSha: tagSha,
	})
	require.NoError(t, err)
	// For lightweight tags, target SHA == commit SHA.
	require.Equal(t, tagSha, result)
}

func TestCommitLog_CacheHit(t *testing.T) {
	repo := NewGitHubRepository(nil, "o", "r")

	// Pre-populate cache.
	expected := []git.Commit{{Sha: "cached_commit"}}
	key := commitLogKey("from", "to")
	repo.cache.putCommitLog(key, expected)

	result, err := repo.CommitLog("from", "to")
	require.NoError(t, err)
	require.Equal(t, expected, result)
}

func TestCommitLogKey_WithFilters(t *testing.T) {
	key := commitLogKey("abc", "def", git.PathFilter("src/"), git.PathFilter(""))
	// Empty filter should be skipped.
	require.Equal(t, "abc:def:src/", key)

	keyNoFilter := commitLogKey("abc", "def")
	require.Equal(t, "abc:def", keyNoFilter)
}

func TestConvertGitHubRepoCommit_Nil(t *testing.T) {
	commit := convertGitHubRepoCommit(nil)
	require.Equal(t, "", commit.Sha)
	require.Nil(t, commit.Parents)
	require.True(t, commit.When.IsZero())
}

func TestWithBaseURL(t *testing.T) {
	repo := NewGitHubRepository(nil, "o", "r", WithBaseURL("https://ghe.example.com/api/v3"))
	require.Equal(t, "https://ghe.example.com/api/v3", repo.baseURL)
}

func TestCommitsPriorTo_NilTip(t *testing.T) {
	repo := NewGitHubRepository(nil, "o", "r")
	branch := git.Branch{Name: git.NewBranchReferenceName("main")}
	result, err := repo.CommitsPriorTo(time.Now(), branch)
	require.NoError(t, err)
	require.Nil(t, result)
}

func TestBranchCommits_WithTip(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v3/repos/testowner/testrepo/commits", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, []map[string]interface{}{
			{
				"sha":     "commit_1",
				"commit":  map[string]interface{}{"message": "first", "committer": map[string]interface{}{"date": "2025-01-01T00:00:00Z"}},
				"parents": []map[string]interface{}{},
			},
		})
	})

	tip := git.Commit{Sha: "tip_sha"}
	repo, cleanup := newTestRepo(t, mux)
	defer cleanup()

	commits, err := repo.BranchCommits(git.Branch{
		Name: git.NewBranchReferenceName("main"),
		Tip:  &tip,
	})
	require.NoError(t, err)
	require.Len(t, commits, 1)
}

func TestMainlineCommitLog_Empty(t *testing.T) {
	repo := NewGitHubRepository(nil, "o", "r")
	// Pre-populate with empty commit log.
	key := commitLogKey("base", "tip")
	repo.cache.putCommitLog(key, []git.Commit{})

	mainline, err := repo.MainlineCommitLog("base", "tip")
	require.NoError(t, err)
	require.Nil(t, mainline)
}

func TestFetchFileContent_WithRef(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v3/repos/testowner/testrepo/contents/config.yml", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "develop", r.URL.Query().Get("ref"))
		writeJSON(w, map[string]interface{}{
			"type":     "file",
			"encoding": "base64",
			"content":  "dGVzdA==", // base64 of "test"
		})
	})

	repo, cleanup := newTestRepo(t, mux, WithRef("develop"))
	defer cleanup()

	content, err := repo.FetchFileContent("config.yml")
	require.NoError(t, err)
	require.Equal(t, "test", content)
}

func TestCommitLog_Paginated_WithFromBoundary(t *testing.T) {
	fromSha := "boundary_sha"
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v3/repos/testowner/testrepo/commits", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, []map[string]interface{}{
			{
				"sha":     "newer_commit",
				"commit":  map[string]interface{}{"message": "new", "committer": map[string]interface{}{"date": "2025-06-01T00:00:00Z"}},
				"parents": []map[string]interface{}{{"sha": fromSha}},
			},
			{
				"sha":     fromSha,
				"commit":  map[string]interface{}{"message": "boundary", "committer": map[string]interface{}{"date": "2025-01-01T00:00:00Z"}},
				"parents": []map[string]interface{}{},
			},
			{
				"sha":     "even_older",
				"commit":  map[string]interface{}{"message": "should not appear", "committer": map[string]interface{}{"date": "2024-01-01T00:00:00Z"}},
				"parents": []map[string]interface{}{},
			},
		})
	})

	repo, cleanup := newTestRepo(t, mux, WithMaxCommits(1000))
	defer cleanup()

	// Bounded paginated walk (compare API fails, so it falls back to paginated).
	// We trick it by having from != "" but making compare fail.
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/compare/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "error", http.StatusInternalServerError)
	})

	commits, err := repo.CommitLog(fromSha, "HEAD")
	require.NoError(t, err)
	// Should stop at fromSha boundary, so only "newer_commit" is returned.
	require.Len(t, commits, 1)
	require.Equal(t, "newer_commit", commits[0].Sha)
}

func TestCommitLog_Paginated_PathFilter(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v3/repos/testowner/testrepo/commits", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "src/", r.URL.Query().Get("path"))
		writeJSON(w, []map[string]interface{}{
			{
				"sha":     "filtered_commit",
				"commit":  map[string]interface{}{"message": "in src/", "committer": map[string]interface{}{"date": "2025-01-01T00:00:00Z"}},
				"parents": []map[string]interface{}{},
			},
		})
	})

	repo, cleanup := newTestRepo(t, mux)
	defer cleanup()

	commits, err := repo.CommitLog("", "HEAD", git.PathFilter("src/"))
	require.NoError(t, err)
	require.Len(t, commits, 1)
	require.Equal(t, "filtered_commit", commits[0].Sha)
}

func TestGraphQL_ErrorResponse(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{
			"errors": []map[string]interface{}{
				{"message": "something went wrong"},
			},
		})
	})

	repo, _, cleanup := newTestRepoWithGraphQL(t, mux)
	defer cleanup()

	_, err := repo.Branches()
	require.Error(t, err)
	require.Contains(t, err.Error(), "something went wrong")
}

func TestGraphQL_HTTPError(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})

	repo, _, cleanup := newTestRepoWithGraphQL(t, mux)
	defer cleanup()

	_, err := repo.Tags()
	require.Error(t, err)
	require.Contains(t, err.Error(), "401")
}

func TestCommitFromRefTarget(t *testing.T) {
	target := refTarget{
		OID:           "abc123",
		Message:       "test message",
		CommittedDate: "2025-06-15T10:30:00Z",
		Parents: parentList{
			Nodes: []struct {
				OID string `json:"oid"`
			}{
				{OID: "parent1"},
				{OID: "parent2"},
			},
		},
	}

	commit := commitFromRefTarget(target)
	require.Equal(t, "abc123", commit.Sha)
	require.Equal(t, "test message", commit.Message)
	require.Len(t, commit.Parents, 2)
	require.Equal(t, 2025, commit.When.Year())
}

func TestCommitFromRefTarget_NoDate(t *testing.T) {
	target := refTarget{
		OID: "abc123",
	}

	commit := commitFromRefTarget(target)
	require.Equal(t, "abc123", commit.Sha)
	require.True(t, commit.When.IsZero())
}

func TestHead_APIError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/repos/testowner/testrepo", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
	})

	repo, cleanup := newTestRepo(t, mux)
	defer cleanup()

	_, err := repo.Head()
	require.Error(t, err)
	require.Contains(t, err.Error(), "getting repository info")
}

func TestCommitFromSha_APIError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/commits/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
	})

	repo, cleanup := newTestRepo(t, mux)
	defer cleanup()

	_, err := repo.CommitFromSha("deadbeef12345678901234567890123456789012")
	require.Error(t, err)
	require.Contains(t, err.Error(), "getting commit")
}

func TestFindMergeBase_NilMergeBaseCommit(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/compare/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{
			"status": "diverged",
		})
	})

	repo, cleanup := newTestRepo(t, mux)
	defer cleanup()

	base, err := repo.FindMergeBase("sha1", "sha2")
	require.NoError(t, err)
	require.Equal(t, "", base)
}

func TestCommitLogCompare_PartialResults(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v3/repos/testowner/testrepo/compare/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{
			"total_commits": 500,
			"commits": []map[string]interface{}{
				{
					"sha":     "only_one",
					"commit":  map[string]interface{}{"message": "partial", "committer": map[string]interface{}{"date": "2025-01-01T00:00:00Z"}},
					"parents": []map[string]interface{}{},
				},
			},
		})
	})

	// Paginated fallback endpoint.
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/commits", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, []map[string]interface{}{
			{
				"sha":     "paginated_commit",
				"commit":  map[string]interface{}{"message": "from paginated", "committer": map[string]interface{}{"date": "2025-01-01T00:00:00Z"}},
				"parents": []map[string]interface{}{},
			},
		})
	})

	repo, cleanup := newTestRepo(t, mux)
	defer cleanup()

	commits, err := repo.CommitLog("from_sha", "to_sha")
	require.NoError(t, err)
	require.Len(t, commits, 1)
	// Should have fallen back to paginated.
	require.Equal(t, "paginated_commit", commits[0].Sha)
}

func TestBranches_GraphQL_Pagination(t *testing.T) {
	mux := http.NewServeMux()
	page := 0

	mux.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
		page++
		if page == 1 {
			branches := []map[string]interface{}{
				{
					"name": "main",
					"target": map[string]interface{}{
						"oid":           "aaa1111111111111111111111111111111111111",
						"message":       "init",
						"committedDate": "2025-01-01T00:00:00Z",
						"parents":       map[string]interface{}{"nodes": []interface{}{}},
					},
				},
			}
			writeJSON(w, graphQLBranchesResponse(branches, true, "cursor1"))
		} else {
			branches := []map[string]interface{}{
				{
					"name": "develop",
					"target": map[string]interface{}{
						"oid":           "bbb2222222222222222222222222222222222222",
						"message":       "dev",
						"committedDate": "2025-02-01T00:00:00Z",
						"parents":       map[string]interface{}{"nodes": []interface{}{}},
					},
				},
			}
			writeJSON(w, graphQLBranchesResponse(branches, false, ""))
		}
	})

	repo, _, cleanup := newTestRepoWithGraphQL(t, mux)
	defer cleanup()

	result, err := repo.Branches()
	require.NoError(t, err)
	require.Len(t, result, 2)
	require.Equal(t, 2, page)
}

func TestTags_GraphQL_Pagination(t *testing.T) {
	mux := http.NewServeMux()
	page := 0

	mux.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
		page++
		if page == 1 {
			tags := []map[string]interface{}{
				{
					"name": "v1.0.0",
					"target": map[string]interface{}{
						"__typename":    "Commit",
						"oid":           "commit_aaa",
						"message":       "v1",
						"committedDate": "2025-01-01T00:00:00Z",
						"parents":       map[string]interface{}{"nodes": []interface{}{}},
					},
				},
			}
			writeJSON(w, graphQLTagsResponse(tags, true, "cursor1"))
		} else {
			tags := []map[string]interface{}{
				{
					"name": "v2.0.0",
					"target": map[string]interface{}{
						"__typename":    "Commit",
						"oid":           "commit_bbb",
						"message":       "v2",
						"committedDate": "2025-06-01T00:00:00Z",
						"parents":       map[string]interface{}{"nodes": []interface{}{}},
					},
				},
			}
			writeJSON(w, graphQLTagsResponse(tags, false, ""))
		}
	})

	repo, _, cleanup := newTestRepoWithGraphQL(t, mux)
	defer cleanup()

	result, err := repo.Tags()
	require.NoError(t, err)
	require.Len(t, result, 2)
	require.Equal(t, 2, page)
}

func TestFetchFileContent_NilResponse(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/contents/empty.yml", func(w http.ResponseWriter, r *http.Request) {
		// Return null content (directory instead of file).
		_, _ = w.Write([]byte("null"))
	})

	repo, cleanup := newTestRepo(t, mux)
	defer cleanup()

	_, err := repo.FetchFileContent("empty.yml")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestHead_BranchAPIError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/branches/broken", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
	})

	repo, cleanup := newTestRepo(t, mux, WithRef("broken"))
	defer cleanup()

	_, err := repo.Head()
	require.Error(t, err)
	require.Contains(t, err.Error(), "getting branch")
}

func TestHead_DetachedSHA_CommitError(t *testing.T) {
	sha := "abcdef1234567890abcdef1234567890abcdef12"
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/commits/"+sha, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
	})

	repo, cleanup := newTestRepo(t, mux, WithRef(sha))
	defer cleanup()

	_, err := repo.Head()
	require.Error(t, err)
	require.Contains(t, err.Error(), "getting HEAD commit")
}

func TestFindMergeBase_APIError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/compare/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"server error"}`, http.StatusInternalServerError)
	})

	repo, cleanup := newTestRepo(t, mux)
	defer cleanup()

	_, err := repo.FindMergeBase("sha1", "sha2")
	require.Error(t, err)
	require.Contains(t, err.Error(), "comparing commits")
}

func TestMainlineCommitLog_WithFromBoundary(t *testing.T) {
	repo := NewGitHubRepository(nil, "o", "r")

	// Pre-populate cache with a chain: C -> B -> A
	commits := []git.Commit{
		{Sha: "ccc", Parents: []string{"bbb"}, Message: "C"},
		{Sha: "bbb", Parents: []string{"aaa"}, Message: "B"},
		{Sha: "aaa", Parents: []string{"base"}, Message: "A"},
	}
	key := commitLogKey("base", "ccc")
	repo.cache.putCommitLog(key, commits)

	// MainlineCommitLog should stop at "base" boundary.
	mainline, err := repo.MainlineCommitLog("base", "ccc")
	require.NoError(t, err)
	// Should follow ccc -> bbb -> aaa, stopping because aaa's first parent is "base".
	require.Len(t, mainline, 3)
	require.Equal(t, "ccc", mainline[0].Sha)
	require.Equal(t, "bbb", mainline[1].Sha)
	require.Equal(t, "aaa", mainline[2].Sha)
}

func TestCommitsPriorTo_APIError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/commits", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"error"}`, http.StatusInternalServerError)
	})

	repo, cleanup := newTestRepo(t, mux)
	defer cleanup()

	tip := git.Commit{Sha: "tip123"}
	branch := git.Branch{
		Name: git.NewBranchReferenceName("main"),
		Tip:  &tip,
	}

	_, err := repo.CommitsPriorTo(time.Now(), branch)
	require.Error(t, err)
	require.Contains(t, err.Error(), "listing commits prior to")
}

func TestBranchesContainingCommit_NilTipSkipped(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
		// Return a branch where target has no commit data (nil tip scenario).
		branches := []map[string]interface{}{
			{
				"name":   "empty-branch",
				"target": map[string]interface{}{},
			},
		}
		writeJSON(w, graphQLBranchesResponse(branches, false, ""))
	})

	repo, _, cleanup := newTestRepoWithGraphQL(t, mux)
	defer cleanup()

	result, err := repo.BranchesContainingCommit("some_sha")
	require.NoError(t, err)
	require.Empty(t, result)
}
