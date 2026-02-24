package gitsemver_test

import (
	"encoding/json"
	"go-gitsemver/internal/testutil"
	"go-gitsemver/pkg/gitsemver"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gh "github.com/google/go-github/v68/github"
	"github.com/stretchr/testify/require"
)

func TestCalculate_BasicRepo(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.AddCommit("initial commit")
	repo.AddCommit("second commit")

	result, err := gitsemver.Calculate(gitsemver.LocalOptions{
		Path: repo.Path(),
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Variables["SemVer"])
	require.NotEmpty(t, result.Variables["FullSemVer"])
	require.NotEmpty(t, result.Variables["MajorMinorPatch"])
}

func TestCalculate_WithTag(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial commit")
	repo.CreateTag("v1.0.0", sha)
	repo.AddCommit("feature work")

	result, err := gitsemver.Calculate(gitsemver.LocalOptions{
		Path: repo.Path(),
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Should be > 1.0.0 since we have a commit after the tag.
	require.Contains(t, result.Variables["MajorMinorPatch"], "1.0.")
}

func TestCalculate_TaggedCommitExact(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("release commit")
	repo.CreateTag("v2.0.0", sha)

	result, err := gitsemver.Calculate(gitsemver.LocalOptions{
		Path:   repo.Path(),
		Commit: sha,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "2.0.0", result.Variables["MajorMinorPatch"])
}

func TestCalculate_InvalidPath(t *testing.T) {
	_, err := gitsemver.Calculate(gitsemver.LocalOptions{
		Path: "/nonexistent/path",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "opening repository")
}

func TestCalculate_WithConfigFile(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial commit")
	repo.CreateTag("v1.0.0", sha)
	repo.AddCommit("next commit")

	// Write a config file with next-version override.
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte("next-version: 5.0.0\n"), 0o644))

	result, err := gitsemver.Calculate(gitsemver.LocalOptions{
		Path:       repo.Path(),
		ConfigPath: configPath,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "5.0.0", result.Variables["MajorMinorPatch"])
}

func TestCalculate_AutoDetectsConfig(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.AddCommit("initial commit")

	// Write gitsemver.yml in the repo root.
	repo.WriteConfig("next-version: 7.0.0\n")

	result, err := gitsemver.Calculate(gitsemver.LocalOptions{
		Path: repo.Path(),
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "7.0.0", result.Variables["MajorMinorPatch"])
}

func TestCalculate_InvalidConfigPath(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.AddCommit("initial commit")

	_, err := gitsemver.Calculate(gitsemver.LocalOptions{
		Path:       repo.Path(),
		ConfigPath: "/nonexistent/config.yml",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "loading configuration")
}

func TestCalculate_WithBranch(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial commit")
	repo.CreateTag("v1.0.0", sha)
	repo.CreateBranch("feature/test", sha)
	repo.Checkout("feature/test")
	repo.AddCommit("feature work")

	result, err := gitsemver.Calculate(gitsemver.LocalOptions{
		Path:   repo.Path(),
		Branch: "feature/test",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Variables["SemVer"])
}

func TestCalculate_ResultHasAllKeyVariables(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.AddCommit("initial commit")

	result, err := gitsemver.Calculate(gitsemver.LocalOptions{
		Path: repo.Path(),
	})
	require.NoError(t, err)

	expectedKeys := []string{
		"Major", "Minor", "Patch",
		"SemVer", "FullSemVer", "MajorMinorPatch",
		"Sha", "ShortSha", "BranchName",
		"CommitsSinceVersionSource",
	}
	for _, key := range expectedKeys {
		require.Contains(t, result.Variables, key, "missing variable: %s", key)
	}
}

func TestCalculateRemote_MissingOwner(t *testing.T) {
	_, err := gitsemver.CalculateRemote(gitsemver.RemoteOptions{
		Repo:  "myrepo",
		Token: "ghp_test",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "owner and repo are required")
}

func TestCalculateRemote_MissingRepo(t *testing.T) {
	_, err := gitsemver.CalculateRemote(gitsemver.RemoteOptions{
		Owner: "myorg",
		Token: "ghp_test",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "owner and repo are required")
}

func TestCalculateRemote_NoAuth(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_APP_ID", "")
	t.Setenv("GH_APP_PRIVATE_KEY", "")

	_, err := gitsemver.CalculateRemote(gitsemver.RemoteOptions{
		Owner: "myorg",
		Repo:  "myrepo",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "creating GitHub client")
}

// writeTestJSON encodes v as JSON to w.
func writeTestJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		panic(err)
	}
}

func TestCalculateRemote_WithMockServer(t *testing.T) {
	mux := http.NewServeMux()

	// Mock: Get repository (default branch).
	mux.HandleFunc("/api/v3/repos/testowner/testrepo", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(w, map[string]interface{}{
			"default_branch": "main",
		})
	})

	// Mock: Get branch.
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/branches/main", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(w, map[string]interface{}{
			"name": "main",
			"commit": map[string]interface{}{
				"sha": "abc123def456abc123def456abc123def456abc1",
				"commit": map[string]interface{}{
					"message": "initial commit",
					"committer": map[string]interface{}{
						"date": "2025-01-15T12:00:00Z",
					},
				},
				"parents": []map[string]interface{}{},
			},
		})
	})

	// Mock: GraphQL endpoint for branches and tags.
	mux.HandleFunc("/api/graphql", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Query     string                 `json:"query"`
			Variables map[string]interface{} `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Determine if this is a branches or tags query.
		if strings.Contains(body.Query, "refs/heads/") {
			writeTestJSON(w, map[string]interface{}{
				"data": map[string]interface{}{
					"repository": map[string]interface{}{
						"refs": map[string]interface{}{
							"nodes": []map[string]interface{}{
								{
									"name": "main",
									"target": map[string]interface{}{
										"oid":           "abc123def456abc123def456abc123def456abc1",
										"message":       "initial commit",
										"committedDate": "2025-01-15T12:00:00Z",
										"parents": map[string]interface{}{
											"nodes": []interface{}{},
										},
									},
								},
							},
							"pageInfo": map[string]interface{}{
								"hasNextPage": false,
								"endCursor":   "",
							},
						},
					},
				},
			})
		} else {
			// Tags query — return empty.
			writeTestJSON(w, map[string]interface{}{
				"data": map[string]interface{}{
					"repository": map[string]interface{}{
						"refs": map[string]interface{}{
							"nodes":    []interface{}{},
							"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
						},
					},
				},
			})
		}
	})

	// Mock: List commits (for commit log).
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/commits", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(w, []map[string]interface{}{
			{
				"sha": "abc123def456abc123def456abc123def456abc1",
				"commit": map[string]interface{}{
					"message": "initial commit",
					"committer": map[string]interface{}{
						"date": "2025-01-15T12:00:00Z",
					},
				},
				"parents": []interface{}{},
			},
		})
	})

	// Mock: Contents (config files — return 404).
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/contents/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		writeTestJSON(w, &gh.ErrorResponse{
			Response: &http.Response{StatusCode: http.StatusNotFound},
			Message:  "Not Found",
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	result, err := gitsemver.CalculateRemote(gitsemver.RemoteOptions{
		Owner:   "testowner",
		Repo:    "testrepo",
		Token:   "ghp_test",
		BaseURL: server.URL + "/api/v3",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Variables["SemVer"])
}
