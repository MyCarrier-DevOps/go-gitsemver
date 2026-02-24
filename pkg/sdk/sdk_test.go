package sdk_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MyCarrier-DevOps/go-gitsemver/internal/testutil"
	"github.com/MyCarrier-DevOps/go-gitsemver/pkg/sdk"

	gh "github.com/google/go-github/v68/github"
	"github.com/stretchr/testify/require"
)

func TestCalculate_BasicRepo(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.AddCommit("initial commit")
	repo.AddCommit("second commit")

	result, err := sdk.Calculate(sdk.LocalOptions{
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

	result, err := sdk.Calculate(sdk.LocalOptions{
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

	result, err := sdk.Calculate(sdk.LocalOptions{
		Path:   repo.Path(),
		Commit: sha,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "2.0.0", result.Variables["MajorMinorPatch"])
}

func TestCalculate_InvalidPath(t *testing.T) {
	_, err := sdk.Calculate(sdk.LocalOptions{
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

	result, err := sdk.Calculate(sdk.LocalOptions{
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

	// Write go-gitsemver.yml in the repo root.
	repo.WriteConfig("next-version: 7.0.0\n")

	result, err := sdk.Calculate(sdk.LocalOptions{
		Path: repo.Path(),
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "7.0.0", result.Variables["MajorMinorPatch"])
}

func TestCalculate_InvalidConfigPath(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.AddCommit("initial commit")

	_, err := sdk.Calculate(sdk.LocalOptions{
		Path:       repo.Path(),
		ConfigPath: "/nonexistent/config.yml",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "loading configuration")
}

func TestCalculate_AutoDetectsConfigInGitHubDir(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.AddCommit("initial commit")

	// Write config in .github/ directory (should be found before repo root).
	repo.WriteConfigAt(".github/GitVersion.yml", "next-version: 9.0.0\n")

	result, err := sdk.Calculate(sdk.LocalOptions{
		Path: repo.Path(),
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "9.0.0", result.Variables["MajorMinorPatch"])
}

func TestCalculate_GitHubDirTakesPrecedenceOverRoot(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.AddCommit("initial commit")

	// Write config in both locations — .github/ should win.
	repo.WriteConfigAt(".github/GitVersion.yml", "next-version: 8.0.0\n")
	repo.WriteConfig("next-version: 3.0.0\n")

	result, err := sdk.Calculate(sdk.LocalOptions{
		Path: repo.Path(),
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "8.0.0", result.Variables["MajorMinorPatch"])
}

func TestCalculate_WithBranch(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial commit")
	repo.CreateTag("v1.0.0", sha)
	repo.CreateBranch("feature/test", sha)
	repo.Checkout("feature/test")
	repo.AddCommit("feature work")

	result, err := sdk.Calculate(sdk.LocalOptions{
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

	result, err := sdk.Calculate(sdk.LocalOptions{
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
	_, err := sdk.CalculateRemote(sdk.RemoteOptions{
		Repo:  "myrepo",
		Token: "ghp_test",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "owner and repo are required")
}

func TestCalculateRemote_MissingRepo(t *testing.T) {
	_, err := sdk.CalculateRemote(sdk.RemoteOptions{
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
	t.Setenv("GH_APP_PRIVATE_KEY_PATH", "")

	_, err := sdk.CalculateRemote(sdk.RemoteOptions{
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

	result, err := sdk.CalculateRemote(sdk.RemoteOptions{
		Owner:   "testowner",
		Repo:    "testrepo",
		Token:   "ghp_test",
		BaseURL: server.URL + "/api/v3",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.Variables["SemVer"])
}

func TestCalculateRemote_WithRemoteConfigPath(t *testing.T) {
	mux := http.NewServeMux()
	tipSha := "abc123def456abc123def456abc123def456abc1"

	mux.HandleFunc("/api/v3/repos/testowner/testrepo", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(w, map[string]interface{}{"default_branch": "main"})
	})
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/branches/main", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(w, map[string]interface{}{
			"name": "main",
			"commit": map[string]interface{}{
				"sha":     tipSha,
				"commit":  map[string]interface{}{"message": "initial commit", "committer": map[string]interface{}{"date": "2025-01-15T12:00:00Z"}},
				"parents": []interface{}{},
			},
		})
	})
	mux.HandleFunc("/api/graphql", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(w, map[string]interface{}{
			"data": map[string]interface{}{
				"repository": map[string]interface{}{
					"refs": map[string]interface{}{
						"nodes": []map[string]interface{}{
							{
								"name": "main",
								"target": map[string]interface{}{
									"oid": tipSha, "message": "initial commit",
									"committedDate": "2025-01-15T12:00:00Z",
									"parents":       map[string]interface{}{"nodes": []interface{}{}},
								},
							},
						},
						"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
					},
				},
			},
		})
	})
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/commits", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(w, []map[string]interface{}{
			{
				"sha":     tipSha,
				"commit":  map[string]interface{}{"message": "initial commit", "committer": map[string]interface{}{"date": "2025-01-15T12:00:00Z"}},
				"parents": []interface{}{},
			},
		})
	})

	// Mock: Return config at custom path, 404 for everything else.
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/contents/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/custom/config.yml") {
			// Return base64-encoded YAML content.
			content := "bmV4dC12ZXJzaW9uOiAxMi4wLjAK" // base64("next-version: 12.0.0\n")
			writeTestJSON(w, map[string]interface{}{
				"type":     "file",
				"encoding": "base64",
				"content":  content,
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
		writeTestJSON(w, &gh.ErrorResponse{Response: &http.Response{StatusCode: http.StatusNotFound}, Message: "Not Found"})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	result, err := sdk.CalculateRemote(sdk.RemoteOptions{
		Owner:            "testowner",
		Repo:             "testrepo",
		Token:            "ghp_test",
		BaseURL:          server.URL + "/api/v3",
		RemoteConfigPath: "custom/config.yml",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "12.0.0", result.Variables["MajorMinorPatch"])
}

// ---------------------------------------------------------------------------
// Explain mode tests
// ---------------------------------------------------------------------------

func TestCalculate_ExplainEnabled(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial commit")
	repo.CreateTag("v1.0.0", sha)
	repo.AddCommit("feat: add dashboard")

	result, err := sdk.Calculate(sdk.LocalOptions{
		Path:    repo.Path(),
		Explain: true,
	})
	require.NoError(t, err)
	require.NotNil(t, result.ExplainResult)

	er := result.ExplainResult
	require.NotEmpty(t, er.Candidates)
	require.NotEmpty(t, er.SelectedSource)
	require.NotEmpty(t, er.FinalVersion)
	require.NotEmpty(t, er.FormattedOutput)
	require.Contains(t, er.FormattedOutput, "Strategies evaluated:")
	require.Contains(t, er.FormattedOutput, "Selected:")
	require.Contains(t, er.FormattedOutput, "Result:")

	// Each candidate should have strategy and version.
	for _, c := range er.Candidates {
		require.NotEmpty(t, c.Strategy)
		require.NotEmpty(t, c.Version)
		require.NotEmpty(t, c.Source)
	}
}

func TestCalculate_ExplainDisabled(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.AddCommit("initial commit")

	result, err := sdk.Calculate(sdk.LocalOptions{
		Path:    repo.Path(),
		Explain: false,
	})
	require.NoError(t, err)
	require.Nil(t, result.ExplainResult)
}

func TestCalculate_ExplainWithTag(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial")
	repo.CreateTag("v2.0.0", sha)

	result, err := sdk.Calculate(sdk.LocalOptions{
		Path:    repo.Path(),
		Explain: true,
	})
	require.NoError(t, err)
	require.Equal(t, "2.0.0", result.Variables["MajorMinorPatch"])
}

func TestCalculate_ExplainCandidateDetails(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial release")
	repo.CreateTag("v1.0.0", sha)
	repo.AddCommit("fix: patch fix")

	result, err := sdk.Calculate(sdk.LocalOptions{
		Path:    repo.Path(),
		Explain: true,
	})
	require.NoError(t, err)
	require.NotNil(t, result.ExplainResult)

	strategies := map[string]bool{}
	for _, c := range result.ExplainResult.Candidates {
		strategies[c.Strategy] = true
	}
	require.True(t, strategies["TaggedCommit"] || strategies["Fallback"],
		"should have at least TaggedCommit or Fallback candidate")
}

func TestCalculate_ExplainIncrementSteps(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial")
	repo.CreateTag("v1.0.0", sha)
	repo.AddCommit("feat: new feature")

	result, err := sdk.Calculate(sdk.LocalOptions{
		Path:    repo.Path(),
		Explain: true,
	})
	require.NoError(t, err)
	require.NotNil(t, result.ExplainResult)
	require.NotEmpty(t, result.ExplainResult.IncrementSteps)
}

func TestCalculate_ExplainFeatureBranch(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial on main")
	repo.CreateTag("v1.0.0", sha)
	repo.CreateBranch("feature/search", sha)
	repo.Checkout("feature/search")
	repo.AddCommit("feat: add search")

	result, err := sdk.Calculate(sdk.LocalOptions{
		Path:    repo.Path(),
		Explain: true,
	})
	require.NoError(t, err)
	require.NotNil(t, result.ExplainResult)
	require.NotEmpty(t, result.ExplainResult.PreReleaseSteps)
}

func TestCalculateRemote_ExplainEnabled(t *testing.T) {
	mux := http.NewServeMux()
	tipSha := "abc123def456abc123def456abc123def456abc1"

	mux.HandleFunc("/api/v3/repos/testowner/testrepo", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(w, map[string]interface{}{"default_branch": "main"})
	})
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/branches/main", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(w, map[string]interface{}{
			"name": "main",
			"commit": map[string]interface{}{
				"sha": tipSha,
				"commit": map[string]interface{}{
					"message":   "feat: new feature",
					"committer": map[string]interface{}{"date": "2025-01-15T12:00:00Z"},
				},
				"parents": []interface{}{},
			},
		})
	})
	mux.HandleFunc("/api/graphql", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(w, map[string]interface{}{
			"data": map[string]interface{}{
				"repository": map[string]interface{}{
					"refs": map[string]interface{}{
						"nodes": []map[string]interface{}{
							{
								"name": "main",
								"target": map[string]interface{}{
									"oid": tipSha, "message": "feat: new feature",
									"committedDate": "2025-01-15T12:00:00Z",
									"parents":       map[string]interface{}{"nodes": []interface{}{}},
								},
							},
						},
						"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
					},
				},
			},
		})
	})
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/commits", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(w, []map[string]interface{}{
			{
				"sha":     tipSha,
				"commit":  map[string]interface{}{"message": "feat: new feature", "committer": map[string]interface{}{"date": "2025-01-15T12:00:00Z"}},
				"parents": []interface{}{},
			},
		})
	})
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/contents/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		writeTestJSON(w, &gh.ErrorResponse{Response: &http.Response{StatusCode: http.StatusNotFound}, Message: "Not Found"})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	result, err := sdk.CalculateRemote(sdk.RemoteOptions{
		Owner:   "testowner",
		Repo:    "testrepo",
		Token:   "ghp_test",
		BaseURL: server.URL + "/api/v3",
		Explain: true,
	})
	require.NoError(t, err)
	require.NotNil(t, result.ExplainResult)
	require.NotEmpty(t, result.ExplainResult.Candidates)
	require.NotEmpty(t, result.ExplainResult.SelectedSource)
	require.NotEmpty(t, result.ExplainResult.FinalVersion)
	require.Contains(t, result.ExplainResult.FormattedOutput, "Strategies evaluated:")
}

func TestCalculateRemote_ExplainDisabled(t *testing.T) {
	mux := http.NewServeMux()
	tipSha := "abc123def456abc123def456abc123def456abc1"

	mux.HandleFunc("/api/v3/repos/testowner/testrepo", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(w, map[string]interface{}{"default_branch": "main"})
	})
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/branches/main", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(w, map[string]interface{}{
			"name": "main",
			"commit": map[string]interface{}{
				"sha": tipSha,
				"commit": map[string]interface{}{
					"message":   "initial commit",
					"committer": map[string]interface{}{"date": "2025-01-15T12:00:00Z"},
				},
				"parents": []interface{}{},
			},
		})
	})
	mux.HandleFunc("/api/graphql", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(w, map[string]interface{}{
			"data": map[string]interface{}{
				"repository": map[string]interface{}{
					"refs": map[string]interface{}{
						"nodes": []map[string]interface{}{
							{
								"name": "main",
								"target": map[string]interface{}{
									"oid": tipSha, "message": "initial commit",
									"committedDate": "2025-01-15T12:00:00Z",
									"parents":       map[string]interface{}{"nodes": []interface{}{}},
								},
							},
						},
						"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
					},
				},
			},
		})
	})
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/commits", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(w, []map[string]interface{}{
			{
				"sha":     tipSha,
				"commit":  map[string]interface{}{"message": "initial commit", "committer": map[string]interface{}{"date": "2025-01-15T12:00:00Z"}},
				"parents": []interface{}{},
			},
		})
	})
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/contents/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		writeTestJSON(w, &gh.ErrorResponse{Response: &http.Response{StatusCode: http.StatusNotFound}, Message: "Not Found"})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	result, err := sdk.CalculateRemote(sdk.RemoteOptions{
		Owner:   "testowner",
		Repo:    "testrepo",
		Token:   "ghp_test",
		BaseURL: server.URL + "/api/v3",
		Explain: false,
	})
	require.NoError(t, err)
	require.Nil(t, result.ExplainResult)
}
