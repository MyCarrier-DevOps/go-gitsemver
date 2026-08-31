package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testTipSha  = "1111111111111111111111111111111111111111"
	testBaseSha = "2222222222222222222222222222222222222222"
	testAltSha  = "3333333333333333333333333333333333333333"
)

// remoteAPIServer stands in for the GitHub API for a whole remote version
// calculation. The repository's default branch ("trunk") deliberately differs
// from the branch under test ("main") so that ignoring --ref is observable.
func remoteAPIServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	writeJSON := func(w http.ResponseWriter, v any) {
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(v))
	}
	commitJSON := func(sha, msg string, parents ...string) map[string]any {
		ps := make([]map[string]any, 0, len(parents))
		for _, p := range parents {
			ps = append(ps, map[string]any{"sha": p})
		}
		return map[string]any{
			"sha": sha,
			"commit": map[string]any{
				"message":   msg,
				"committer": map[string]any{"date": "2025-01-02T00:00:00Z"},
			},
			"parents": ps,
		}
	}

	// No configuration in the remote repo: fall back to defaults.
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/contents/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
	})

	mux.HandleFunc("/api/v3/repos/testowner/testrepo", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"name": "testrepo", "default_branch": "trunk"})
	})

	mux.HandleFunc("/api/v3/repos/testowner/testrepo/branches/main", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"name": "main", "commit": commitJSON(testTipSha, "feat: add search", testBaseSha)})
	})
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/branches/trunk", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"name": "trunk", "commit": commitJSON(testAltSha, "fix: a bug", testBaseSha)})
	})

	mux.HandleFunc("/api/v3/repos/testowner/testrepo/commits/", func(w http.ResponseWriter, r *http.Request) {
		sha := strings.TrimPrefix(r.URL.Path, "/api/v3/repos/testowner/testrepo/commits/")
		switch sha {
		case testTipSha:
			writeJSON(w, commitJSON(testTipSha, "feat: add search", testBaseSha))
		case testAltSha:
			writeJSON(w, commitJSON(testAltSha, "fix: a bug", testBaseSha))
		default:
			writeJSON(w, commitJSON(testBaseSha, "initial"))
		}
	})

	mux.HandleFunc("/api/v3/repos/testowner/testrepo/commits", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, []map[string]any{
			commitJSON(testTipSha, "feat: add search", testBaseSha),
			commitJSON(testBaseSha, "initial"),
		})
	})

	mux.HandleFunc("/api/v3/repos/testowner/testrepo/compare/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"total_commits": 2,
			"commits": []map[string]any{
				commitJSON(testBaseSha, "initial"),
				commitJSON(testTipSha, "feat: add search", testBaseSha),
			},
		})
	})

	// GraphQL serves branch and tag refs; the query text says which.
	mux.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "refs/tags/") {
			writeJSON(w, map[string]any{"data": map[string]any{
				"repository": map[string]any{"refs": map[string]any{
					"nodes": []map[string]any{{
						"name":   "v1.2.0",
						"target": map[string]any{"__typename": "Commit", "oid": testBaseSha, "message": "initial", "committedDate": "2025-01-01T00:00:00Z", "parents": map[string]any{"nodes": []any{}}},
					}},
					"pageInfo": map[string]any{"hasNextPage": false, "endCursor": ""},
				}},
			}})
			return
		}
		writeJSON(w, map[string]any{"data": map[string]any{
			"repository": map[string]any{"refs": map[string]any{
				"nodes": []map[string]any{
					{"name": "main", "target": map[string]any{"__typename": "Commit", "oid": testTipSha, "message": "feat: add search", "committedDate": "2025-01-02T00:00:00Z", "parents": map[string]any{"nodes": []map[string]any{{"oid": testBaseSha}}}}},
					{"name": "trunk", "target": map[string]any{"__typename": "Commit", "oid": testAltSha, "message": "fix: a bug", "committedDate": "2025-01-02T00:00:00Z", "parents": map[string]any{"nodes": []map[string]any{{"oid": testBaseSha}}}}},
				},
				"pageInfo": map[string]any{"hasNextPage": false, "endCursor": ""},
			}},
		}})
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

// resetRemoteFlags restores the remote-specific globals after a test.
func resetRemoteFlags(t *testing.T) {
	t.Helper()
	saved := struct {
		token, appKey, appKeyPath, githubURL, ref, remoteConfigPath string
		appID                                                       int64
		maxCommits                                                  int
	}{flagToken, flagAppKey, flagAppKeyPath, flagGitHubURL, flagRef, flagRemoteConfigPath, flagAppID, flagMaxCommits}
	t.Cleanup(func() {
		flagToken, flagAppKey, flagAppKeyPath = saved.token, saved.appKey, saved.appKeyPath
		flagGitHubURL, flagRef, flagRemoteConfigPath = saved.githubURL, saved.ref, saved.remoteConfigPath
		flagAppID, flagMaxCommits = saved.appID, saved.maxCommits
	})
}

func TestRemoteRunE_UsesRequestedRef(t *testing.T) {
	resetFlags(t)
	resetRemoteFlags(t)
	server := remoteAPIServer(t)

	flagToken = "ghp_test"
	flagGitHubURL = server.URL + "/"
	flagRef = "main"
	flagMaxCommits = 1000
	flagOutput = "json"

	stdout, _, err := captureOutput(t, func() error { return remoteRunE(nil, []string{"testowner/testrepo"}) })
	require.NoError(t, err)

	var vars map[string]string
	require.NoError(t, json.Unmarshal([]byte(stdout), &vars))
	require.Equal(t, "main", vars["BranchName"], "--ref must select the branch, not the repo default")
	require.Equal(t, "1.3.0", vars["MajorMinorPatch"])
}

func TestRemoteRunE_DefaultsToRepositoryDefaultBranch(t *testing.T) {
	resetFlags(t)
	resetRemoteFlags(t)
	server := remoteAPIServer(t)

	flagToken = "ghp_test"
	flagGitHubURL = server.URL + "/"
	flagRef = ""
	flagMaxCommits = 1000
	flagShowVariable = "BranchName"

	stdout, _, err := captureOutput(t, func() error { return remoteRunE(nil, []string{"testowner/testrepo"}) })
	require.NoError(t, err)
	require.Equal(t, "trunk", strings.TrimSpace(stdout))
}

func TestRemoteRunE_ZeroMaxCommitsKeepsDefault(t *testing.T) {
	resetFlags(t)
	resetRemoteFlags(t)
	server := remoteAPIServer(t)

	flagToken = "ghp_test"
	flagGitHubURL = server.URL + "/"
	flagRef = "main"
	// Zero means "unset": the repository default must be kept rather than a
	// zero cap being applied, which would truncate every commit walk.
	flagMaxCommits = 0
	flagShowVariable = "MajorMinorPatch"

	stdout, _, err := captureOutput(t, func() error { return remoteRunE(nil, []string{"testowner/testrepo"}) })
	require.NoError(t, err)
	require.Equal(t, "1.3.0", strings.TrimSpace(stdout))
}

func TestRemoteRunE_InvalidOwnerRepo(t *testing.T) {
	resetFlags(t)
	resetRemoteFlags(t)

	_, _, err := captureOutput(t, func() error { return remoteRunE(nil, []string{"no-slash"}) })
	require.Error(t, err)
}

func TestRemoteRunE_NoAuth(t *testing.T) {
	resetFlags(t)
	resetRemoteFlags(t)
	t.Setenv("GITHUB_TOKEN", "")

	_, _, err := captureOutput(t, func() error { return remoteRunE(nil, []string{"testowner/testrepo"}) })
	require.Error(t, err)
	require.Contains(t, err.Error(), "creating GitHub client")
}

const testMidSha = "4444444444444444444444444444444444444444"

// paginatedRemoteAPIServer forces the paginated commit walk (the compare API
// fails) and splits history across two pages, with a breaking change on the
// second. A commit cap of zero would stop after page one and miss it.
func paginatedRemoteAPIServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	writeJSON := func(w http.ResponseWriter, v any) {
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(v))
	}
	commitJSON := func(sha, msg string, parents ...string) map[string]any {
		ps := make([]map[string]any, 0, len(parents))
		for _, p := range parents {
			ps = append(ps, map[string]any{"sha": p})
		}
		return map[string]any{
			"sha":     sha,
			"commit":  map[string]any{"message": msg, "committer": map[string]any{"date": "2025-01-02T00:00:00Z"}},
			"parents": ps,
		}
	}

	mux.HandleFunc("/api/v3/repos/testowner/testrepo/contents/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
	})
	mux.HandleFunc("/api/v3/repos/testowner/testrepo", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"name": "testrepo", "default_branch": "main"})
	})
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/branches/main", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"name": "main", "commit": commitJSON(testTipSha, "feat: add search", testMidSha)})
	})
	// Force the paginated fallback.
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/compare/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"boom"}`, http.StatusInternalServerError)
	})
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/commits/", func(w http.ResponseWriter, r *http.Request) {
		sha := strings.TrimPrefix(r.URL.Path, "/api/v3/repos/testowner/testrepo/commits/")
		switch sha {
		case testTipSha:
			writeJSON(w, commitJSON(testTipSha, "feat: add search", testMidSha))
		case testMidSha:
			writeJSON(w, commitJSON(testMidSha, "feat!: redesign the API", testBaseSha))
		default:
			writeJSON(w, commitJSON(testBaseSha, "initial"))
		}
	})
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/commits", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "2" {
			writeJSON(w, []map[string]any{
				commitJSON(testMidSha, "feat!: redesign the API", testBaseSha),
				commitJSON(testBaseSha, "initial"),
			})
			return
		}
		w.Header().Set("Link", `<https://example.com/commits?page=2>; rel="next"`)
		writeJSON(w, []map[string]any{commitJSON(testTipSha, "feat: add search", testMidSha)})
	})
	mux.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "refs/tags/") {
			writeJSON(w, map[string]any{"data": map[string]any{
				"repository": map[string]any{"refs": map[string]any{
					"nodes": []map[string]any{{
						"name":   "v1.2.0",
						"target": map[string]any{"__typename": "Commit", "oid": testBaseSha, "message": "initial", "committedDate": "2025-01-01T00:00:00Z", "parents": map[string]any{"nodes": []any{}}},
					}},
					"pageInfo": map[string]any{"hasNextPage": false, "endCursor": ""},
				}},
			}})
			return
		}
		writeJSON(w, map[string]any{"data": map[string]any{
			"repository": map[string]any{"refs": map[string]any{
				"nodes": []map[string]any{
					{"name": "main", "target": map[string]any{"__typename": "Commit", "oid": testTipSha, "message": "feat: add search", "committedDate": "2025-01-02T00:00:00Z", "parents": map[string]any{"nodes": []map[string]any{{"oid": testMidSha}}}}},
				},
				"pageInfo": map[string]any{"hasNextPage": false, "endCursor": ""},
			}},
		}})
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

// TestRemoteRunE_ZeroMaxCommitsDoesNotCapTheWalk pins that --max-commits=0
// means "unset". Applying it as a literal cap would stop the walk after the
// first page and miss the breaking change that lives on the second.
func TestRemoteRunE_ZeroMaxCommitsDoesNotCapTheWalk(t *testing.T) {
	resetFlags(t)
	resetRemoteFlags(t)
	server := paginatedRemoteAPIServer(t)

	flagToken = "ghp_test"
	flagGitHubURL = server.URL + "/"
	flagRef = "main"
	flagMaxCommits = 0
	flagShowVariable = "MajorMinorPatch"

	stdout, _, err := captureOutput(t, func() error { return remoteRunE(nil, []string{"testowner/testrepo"}) })
	require.NoError(t, err)
	require.Equal(t, "2.0.0", strings.TrimSpace(stdout),
		"the whole history must be walked, including the breaking change on page two")
}
