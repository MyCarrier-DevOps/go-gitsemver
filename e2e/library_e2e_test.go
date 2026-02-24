// Package e2e contains end-to-end tests for the pkg/sdk library API.
//
// These tests exercise the public Calculate() and CalculateRemote() functions
// through the full pipeline, verifying that the library produces correct
// results against real git repos and mock GitHub API servers.
package e2e

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

// ---------------------------------------------------------------------------
// Library: Calculate() — local repos
// ---------------------------------------------------------------------------

func TestLibrary_Calculate_BasicRepo(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.AddCommit("initial commit")
	repo.AddCommit("second commit")

	result, err := sdk.Calculate(sdk.LocalOptions{
		Path: repo.Path(),
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "0", result.Variables["Major"])
	require.Equal(t, "1", result.Variables["Minor"])
	require.Equal(t, "0", result.Variables["Patch"])
}

func TestLibrary_Calculate_WithTag(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial")
	repo.CreateTag("v2.0.0", sha)

	result, err := sdk.Calculate(sdk.LocalOptions{
		Path: repo.Path(),
	})
	require.NoError(t, err)
	require.Equal(t, "2.0.0", result.Variables["SemVer"])
}

func TestLibrary_Calculate_CommitsAfterTag(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial")
	repo.CreateTag("v1.0.0", sha)
	repo.AddCommit("feat: add auth")

	result, err := sdk.Calculate(sdk.LocalOptions{
		Path: repo.Path(),
	})
	require.NoError(t, err)
	require.Equal(t, "1", result.Variables["Major"])
	require.Equal(t, "1", result.Variables["CommitsSinceVersionSource"])
}

func TestLibrary_Calculate_InvalidPath(t *testing.T) {
	_, err := sdk.Calculate(sdk.LocalOptions{
		Path: "/nonexistent/path",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "opening repository")
}

func TestLibrary_Calculate_WithConfigPath(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.AddCommit("initial")

	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte("next-version: 5.0.0\n"), 0o644))

	result, err := sdk.Calculate(sdk.LocalOptions{
		Path:       repo.Path(),
		ConfigPath: configPath,
	})
	require.NoError(t, err)
	require.Equal(t, "5.0.0", result.Variables["MajorMinorPatch"])
}

func TestLibrary_Calculate_AutoDetectsGitsemverYml(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.AddCommit("initial")
	repo.WriteConfig("next-version: 8.0.0\n")

	result, err := sdk.Calculate(sdk.LocalOptions{
		Path: repo.Path(),
	})
	require.NoError(t, err)
	require.Equal(t, "8.0.0", result.Variables["MajorMinorPatch"])
}

func TestLibrary_Calculate_AutoDetectsGitVersionYml(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.AddCommit("initial")

	// Write GitVersion.yml (the alternative config filename).
	configPath := filepath.Join(repo.Path(), "GitVersion.yml")
	require.NoError(t, os.WriteFile(configPath, []byte("next-version: 6.0.0\n"), 0o644))

	result, err := sdk.Calculate(sdk.LocalOptions{
		Path: repo.Path(),
	})
	require.NoError(t, err)
	require.Equal(t, "6.0.0", result.Variables["MajorMinorPatch"])
}

func TestLibrary_Calculate_WithBranch(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial on main")
	repo.CreateTag("v1.0.0", sha)
	repo.CreateBranch("feature/login", sha)
	repo.Checkout("feature/login")
	repo.AddCommit("feat: add login page")

	result, err := sdk.Calculate(sdk.LocalOptions{
		Path:   repo.Path(),
		Branch: "feature/login",
	})
	require.NoError(t, err)
	require.Equal(t, "login", result.Variables["PreReleaseLabel"])
	require.Contains(t, result.Variables["SemVer"], "login.")
}

func TestLibrary_Calculate_WithCommitOverride(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("release")
	repo.CreateTag("v3.0.0", sha)
	repo.AddCommit("after release")

	result, err := sdk.Calculate(sdk.LocalOptions{
		Path:   repo.Path(),
		Commit: sha,
	})
	require.NoError(t, err)
	require.Equal(t, "3.0.0", result.Variables["MajorMinorPatch"])
}

func TestLibrary_Calculate_ConventionalCommits(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial")
	repo.CreateTag("v1.0.0", sha)
	repo.AddCommit("feat: add user auth")

	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte("commit-message-convention: ConventionalCommits\n"), 0o644))

	result, err := sdk.Calculate(sdk.LocalOptions{
		Path:       repo.Path(),
		ConfigPath: configPath,
	})
	require.NoError(t, err)
	require.Equal(t, "1", result.Variables["Major"])
	require.Equal(t, "1", result.Variables["Minor"])
	require.Equal(t, "0", result.Variables["Patch"])
}

func TestLibrary_Calculate_BreakingChange(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial")
	repo.CreateTag("v1.0.0", sha)
	repo.AddCommit("feat!: remove legacy API")

	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte("commit-message-convention: ConventionalCommits\n"), 0o644))

	result, err := sdk.Calculate(sdk.LocalOptions{
		Path:       repo.Path(),
		ConfigPath: configPath,
	})
	require.NoError(t, err)
	require.Equal(t, "2", result.Variables["Major"])
	require.Equal(t, "0", result.Variables["Minor"])
	require.Equal(t, "0", result.Variables["Patch"])
}

func TestLibrary_Calculate_HotfixBranch(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial on main")
	repo.CreateTag("v1.0.0", sha)
	repo.CreateBranch("hotfix/critical-fix", sha)
	repo.Checkout("hotfix/critical-fix")
	repo.AddCommit("fix: critical patch")

	result, err := sdk.Calculate(sdk.LocalOptions{
		Path: repo.Path(),
	})
	require.NoError(t, err)
	require.Equal(t, "1", result.Variables["Major"])
	require.Equal(t, "0", result.Variables["Minor"])
	require.Equal(t, "1", result.Variables["Patch"])
	require.Equal(t, "beta", result.Variables["PreReleaseLabel"])
}

func TestLibrary_Calculate_ReleaseBranch(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial on main")
	repo.CreateBranch("release/3.0.0", sha)
	repo.Checkout("release/3.0.0")
	repo.AddCommit("release prep")

	result, err := sdk.Calculate(sdk.LocalOptions{
		Path: repo.Path(),
	})
	require.NoError(t, err)
	require.Equal(t, "3", result.Variables["Major"])
	require.Equal(t, "0", result.Variables["Minor"])
	require.Equal(t, "0", result.Variables["Patch"])
}

func TestLibrary_Calculate_AnnotatedTag(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("release")
	repo.CreateAnnotatedTag("v4.0.0", sha, "Release 4.0.0")

	result, err := sdk.Calculate(sdk.LocalOptions{
		Path: repo.Path(),
	})
	require.NoError(t, err)
	require.Equal(t, "4.0.0", result.Variables["SemVer"])
}

func TestLibrary_Calculate_AllOutputVariables(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial")
	repo.CreateTag("v1.0.0", sha)
	repo.AddCommit("second commit")

	result, err := sdk.Calculate(sdk.LocalOptions{
		Path: repo.Path(),
	})
	require.NoError(t, err)

	expectedKeys := []string{
		"Major", "Minor", "Patch", "MajorMinorPatch",
		"SemVer", "FullSemVer", "LegacySemVer",
		"InformationalVersion", "BranchName",
		"Sha", "ShortSha", "CommitDate", "CommitTag",
		"CommitsSinceVersionSource",
		"BuildMetaData", "FullBuildMetaData",
		"PreReleaseTag", "PreReleaseLabel", "PreReleaseNumber",
		"AssemblySemVer", "AssemblySemFileVer",
		"NuGetVersion", "NuGetVersionV2",
		"UncommittedChanges",
	}
	for _, key := range expectedKeys {
		require.Contains(t, result.Variables, key, "missing variable: %s", key)
	}
}

func TestLibrary_Calculate_ContinuousDeployment(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial")
	repo.CreateTag("v1.0.0", sha)
	repo.AddCommit("second")
	repo.AddCommit("third")

	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte("mode: ContinuousDeployment\ncontinuous-delivery-fallback-tag: ci\n"), 0o644))

	result, err := sdk.Calculate(sdk.LocalOptions{
		Path:       repo.Path(),
		ConfigPath: configPath,
	})
	require.NoError(t, err)
	require.Contains(t, result.Variables["SemVer"], "ci.")
	require.Equal(t, "2", result.Variables["CommitsSinceVersionSource"])
}

func TestLibrary_Calculate_Mainline(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial")
	repo.CreateTag("v1.0.0", sha)
	repo.AddCommit("fix: bug")
	repo.AddCommit("feat: new feature")

	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte("mode: Mainline\ncommit-message-convention: ConventionalCommits\n"), 0o644))

	result, err := sdk.Calculate(sdk.LocalOptions{
		Path:       repo.Path(),
		ConfigPath: configPath,
	})
	require.NoError(t, err)
	require.Equal(t, "1", result.Variables["Major"])
	require.Equal(t, "1", result.Variables["Minor"])
	require.Equal(t, "0", result.Variables["Patch"])
}

// ---------------------------------------------------------------------------
// Library: CalculateRemote() — mock GitHub server
// ---------------------------------------------------------------------------

// newLibraryMockServer sets up a complete mock GitHub API server for library
// CalculateRemote() tests. Returns the server URL.
func newLibraryMockServer(t *testing.T, tipSha, tipMessage, tipDate string, parents []string, tags []mockTag, commits []mockCommit) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	// Repository info.
	mux.HandleFunc("/api/v3/repos/testowner/testrepo", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/repos/testowner/testrepo" {
			http.NotFound(w, r)
			return
		}
		writeLibJSON(w, map[string]interface{}{
			"default_branch": "main",
		})
	})

	// Branch info.
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/branches/", func(w http.ResponseWriter, r *http.Request) {
		parentArr := make([]map[string]interface{}, 0, len(parents))
		for _, p := range parents {
			parentArr = append(parentArr, map[string]interface{}{"sha": p})
		}
		writeLibJSON(w, map[string]interface{}{
			"name": "main",
			"commit": map[string]interface{}{
				"sha": tipSha,
				"commit": map[string]interface{}{
					"message":   tipMessage,
					"committer": map[string]interface{}{"date": tipDate},
				},
				"parents": parentArr,
			},
		})
	})

	// Build commit index for quick lookup.
	commitIndex := make(map[string]mockCommit)
	for _, c := range commits {
		commitIndex[c.sha] = c
	}

	// GraphQL.
	mux.HandleFunc("/api/graphql", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Query     string                 `json:"query"`
			Variables map[string]interface{} `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if strings.Contains(body.Query, "refs/heads/") {
			parentNodes := make([]map[string]interface{}, 0, len(parents))
			for _, p := range parents {
				parentNodes = append(parentNodes, map[string]interface{}{"oid": p})
			}
			writeLibJSON(w, map[string]interface{}{
				"data": map[string]interface{}{
					"repository": map[string]interface{}{
						"refs": map[string]interface{}{
							"nodes": []map[string]interface{}{
								{
									"name": "main",
									"target": map[string]interface{}{
										"oid":           tipSha,
										"message":       tipMessage,
										"committedDate": tipDate,
										"parents":       map[string]interface{}{"nodes": parentNodes},
									},
								},
							},
							"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
						},
					},
				},
			})
		} else {
			// Tags.
			nodes := make([]map[string]interface{}, 0, len(tags))
			for _, tag := range tags {
				date := "2025-01-01T12:00:00Z"
				if c, ok := commitIndex[tag.commitSha]; ok {
					date = c.date
				}
				nodes = append(nodes, map[string]interface{}{
					"name": tag.name,
					"target": map[string]interface{}{
						"oid":           tag.commitSha,
						"message":       "",
						"committedDate": date,
						"parents":       map[string]interface{}{"nodes": []interface{}{}},
					},
				})
			}
			writeLibJSON(w, map[string]interface{}{
				"data": map[string]interface{}{
					"repository": map[string]interface{}{
						"refs": map[string]interface{}{
							"nodes":    nodes,
							"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
						},
					},
				},
			})
		}
	})

	// Paginated commit listing.
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/commits", func(w http.ResponseWriter, r *http.Request) {
		startSha := r.URL.Query().Get("sha")
		if startSha == "" {
			startSha = tipSha
		}

		var result []map[string]interface{}
		current := startSha
		for current != "" {
			c, ok := commitIndex[current]
			if !ok {
				break
			}
			ps := make([]map[string]interface{}, 0, len(c.parents))
			for _, p := range c.parents {
				ps = append(ps, map[string]interface{}{"sha": p})
			}
			result = append(result, map[string]interface{}{
				"sha": c.sha,
				"commit": map[string]interface{}{
					"message":   c.message,
					"committer": map[string]interface{}{"date": c.date},
				},
				"parents": ps,
			})
			if len(c.parents) > 0 {
				current = c.parents[0]
			} else {
				current = ""
			}
		}
		writeLibJSON(w, result)
	})

	// Individual commit lookup.
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/commits/", func(w http.ResponseWriter, r *http.Request) {
		cSha := strings.TrimPrefix(r.URL.Path, "/api/v3/repos/testowner/testrepo/commits/")
		c, ok := commitIndex[cSha]
		if !ok {
			http.NotFound(w, r)
			return
		}
		ps := make([]map[string]interface{}, 0, len(c.parents))
		for _, p := range c.parents {
			ps = append(ps, map[string]interface{}{"sha": p})
		}
		writeLibJSON(w, map[string]interface{}{
			"sha": c.sha,
			"commit": map[string]interface{}{
				"message":   c.message,
				"committer": map[string]interface{}{"date": c.date},
			},
			"parents": ps,
		})
	})

	// Compare API.
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/compare/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/v3/repos/testowner/testrepo/compare/")
		parts := strings.SplitN(path, "...", 2)
		if len(parts) != 2 {
			http.Error(w, "bad compare", http.StatusBadRequest)
			return
		}
		baseSha := parts[0]
		headSha := parts[1]

		var result []map[string]interface{}
		current := headSha
		for current != "" && current != baseSha {
			c, ok := commitIndex[current]
			if !ok {
				break
			}
			ps := make([]map[string]interface{}, 0, len(c.parents))
			for _, p := range c.parents {
				ps = append(ps, map[string]interface{}{"sha": p})
			}
			result = append(result, map[string]interface{}{
				"sha": c.sha,
				"commit": map[string]interface{}{
					"message":   c.message,
					"committer": map[string]interface{}{"date": c.date},
				},
				"parents": ps,
			})
			if len(c.parents) > 0 {
				current = c.parents[0]
			} else {
				current = ""
			}
		}
		// Reverse to forward order.
		for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
			result[i], result[j] = result[j], result[i]
		}

		var mergeBase map[string]interface{}
		if c, ok := commitIndex[baseSha]; ok {
			mergeBase = map[string]interface{}{
				"sha": c.sha,
				"commit": map[string]interface{}{
					"message":   c.message,
					"committer": map[string]interface{}{"date": c.date},
				},
			}
		}

		writeLibJSON(w, map[string]interface{}{
			"total_commits":     len(result),
			"commits":           result,
			"merge_base_commit": mergeBase,
			"status":            "ahead",
		})
	})

	// Config files — 404.
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/contents/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		writeLibJSON(w, &gh.ErrorResponse{
			Response: &http.Response{StatusCode: http.StatusNotFound},
			Message:  "Not Found",
		})
	})

	return httptest.NewServer(mux)
}

func writeLibJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		panic(err)
	}
}

func TestLibrary_CalculateRemote_NoTags(t *testing.T) {
	sha1 := sha("lr1111")
	sha2 := sha("lr2222")

	server := newLibraryMockServer(t,
		sha2, "second commit", "2025-01-01T12:01:00Z",
		[]string{sha1},
		nil, // no tags
		[]mockCommit{
			{sha: sha1, message: "initial", date: "2025-01-01T12:00:00Z"},
			{sha: sha2, message: "second commit", date: "2025-01-01T12:01:00Z", parents: []string{sha1}},
		},
	)
	defer server.Close()

	result, err := sdk.CalculateRemote(sdk.RemoteOptions{
		Owner:   "testowner",
		Repo:    "testrepo",
		Token:   "ghp_test",
		BaseURL: server.URL + "/api/v3",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "0", result.Variables["Major"])
	require.Equal(t, "1", result.Variables["Minor"])
	require.Equal(t, "0", result.Variables["Patch"])
}

func TestLibrary_CalculateRemote_WithTag(t *testing.T) {
	sha1 := sha("lr3333")

	server := newLibraryMockServer(t,
		sha1, "release commit", "2025-01-01T12:00:00Z",
		nil,
		[]mockTag{{name: "v2.0.0", commitSha: sha1}},
		[]mockCommit{
			{sha: sha1, message: "release commit", date: "2025-01-01T12:00:00Z"},
		},
	)
	defer server.Close()

	result, err := sdk.CalculateRemote(sdk.RemoteOptions{
		Owner:   "testowner",
		Repo:    "testrepo",
		Token:   "ghp_test",
		BaseURL: server.URL + "/api/v3",
	})
	require.NoError(t, err)
	require.Equal(t, "2.0.0", result.Variables["SemVer"])
}

func TestLibrary_CalculateRemote_CommitsAfterTag(t *testing.T) {
	sha1 := sha("lr4444")
	sha2 := sha("lr5555")

	server := newLibraryMockServer(t,
		sha2, "after tag", "2025-01-01T12:01:00Z",
		[]string{sha1},
		[]mockTag{{name: "v1.0.0", commitSha: sha1}},
		[]mockCommit{
			{sha: sha1, message: "initial", date: "2025-01-01T12:00:00Z"},
			{sha: sha2, message: "after tag", date: "2025-01-01T12:01:00Z", parents: []string{sha1}},
		},
	)
	defer server.Close()

	result, err := sdk.CalculateRemote(sdk.RemoteOptions{
		Owner:   "testowner",
		Repo:    "testrepo",
		Token:   "ghp_test",
		BaseURL: server.URL + "/api/v3",
	})
	require.NoError(t, err)
	require.Equal(t, "1", result.Variables["Major"])
	require.Equal(t, "1", result.Variables["CommitsSinceVersionSource"])
}

func TestLibrary_CalculateRemote_ValidationErrors(t *testing.T) {
	t.Run("missing owner", func(t *testing.T) {
		_, err := sdk.CalculateRemote(sdk.RemoteOptions{
			Repo:  "myrepo",
			Token: "ghp_test",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "owner and repo are required")
	})

	t.Run("missing repo", func(t *testing.T) {
		_, err := sdk.CalculateRemote(sdk.RemoteOptions{
			Owner: "myorg",
			Token: "ghp_test",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "owner and repo are required")
	})

	t.Run("no auth", func(t *testing.T) {
		t.Setenv("GITHUB_TOKEN", "")
		t.Setenv("GH_APP_ID", "")
		t.Setenv("GH_APP_PRIVATE_KEY", "")

		_, err := sdk.CalculateRemote(sdk.RemoteOptions{
			Owner: "myorg",
			Repo:  "myrepo",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "creating GitHub client")
	})
}

func TestLibrary_CalculateRemote_ConventionalCommits(t *testing.T) {
	sha1 := sha("lrc111")
	sha2 := sha("lrc222")

	server := newLibraryMockServer(t,
		sha2, "feat: add auth", "2025-01-01T12:01:00Z",
		[]string{sha1},
		[]mockTag{{name: "v1.0.0", commitSha: sha1}},
		[]mockCommit{
			{sha: sha1, message: "initial", date: "2025-01-01T12:00:00Z"},
			{sha: sha2, message: "feat: add auth", date: "2025-01-01T12:01:00Z", parents: []string{sha1}},
		},
	)
	defer server.Close()

	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte("commit-message-convention: ConventionalCommits\n"), 0o644))

	result, err := sdk.CalculateRemote(sdk.RemoteOptions{
		Owner:      "testowner",
		Repo:       "testrepo",
		Token:      "ghp_test",
		BaseURL:    server.URL + "/api/v3",
		ConfigPath: configPath,
	})
	require.NoError(t, err)
	require.Equal(t, "1", result.Variables["Major"])
	require.Equal(t, "1", result.Variables["Minor"])
	require.Equal(t, "0", result.Variables["Patch"])
}

func TestLibrary_CalculateRemote_AllOutputVariables(t *testing.T) {
	sha1 := sha("lrv111")
	sha2 := sha("lrv222")

	server := newLibraryMockServer(t,
		sha2, "after tag", "2025-01-01T12:01:00Z",
		[]string{sha1},
		[]mockTag{{name: "v1.0.0", commitSha: sha1}},
		[]mockCommit{
			{sha: sha1, message: "initial", date: "2025-01-01T12:00:00Z"},
			{sha: sha2, message: "after tag", date: "2025-01-01T12:01:00Z", parents: []string{sha1}},
		},
	)
	defer server.Close()

	result, err := sdk.CalculateRemote(sdk.RemoteOptions{
		Owner:   "testowner",
		Repo:    "testrepo",
		Token:   "ghp_test",
		BaseURL: server.URL + "/api/v3",
	})
	require.NoError(t, err)

	expectedKeys := []string{
		"Major", "Minor", "Patch", "MajorMinorPatch",
		"SemVer", "FullSemVer",
		"Sha", "ShortSha", "BranchName",
		"CommitsSinceVersionSource",
		"PreReleaseTag", "PreReleaseLabel",
		"NuGetVersion", "NuGetVersionV2",
	}
	for _, key := range expectedKeys {
		require.Contains(t, result.Variables, key, "missing variable: %s", key)
	}
}

// ---------------------------------------------------------------------------
// Library: Parity — local vs remote produce same version
// ---------------------------------------------------------------------------

func TestLibrary_Parity_LocalVsRemote(t *testing.T) {
	// Build a local repo.
	repo := testutil.NewTestRepo(t)
	tagSha := repo.AddCommit("initial")
	repo.CreateTag("v1.0.0", tagSha)
	tipSha := repo.AddCommit("second commit")

	// Calculate locally.
	localResult, err := sdk.Calculate(sdk.LocalOptions{
		Path: repo.Path(),
	})
	require.NoError(t, err)

	// Build equivalent mock server.
	server := newLibraryMockServer(t,
		tipSha, "second commit", "2025-01-01T12:01:00Z",
		[]string{tagSha},
		[]mockTag{{name: "v1.0.0", commitSha: tagSha}},
		[]mockCommit{
			{sha: tagSha, message: "initial", date: "2025-01-01T12:00:00Z"},
			{sha: tipSha, message: "second commit", date: "2025-01-01T12:01:00Z", parents: []string{tagSha}},
		},
	)
	defer server.Close()

	remoteResult, err := sdk.CalculateRemote(sdk.RemoteOptions{
		Owner:   "testowner",
		Repo:    "testrepo",
		Token:   "ghp_test",
		BaseURL: server.URL + "/api/v3",
	})
	require.NoError(t, err)

	// Core version fields must match.
	require.Equal(t, localResult.Variables["Major"], remoteResult.Variables["Major"])
	require.Equal(t, localResult.Variables["Minor"], remoteResult.Variables["Minor"])
	require.Equal(t, localResult.Variables["Patch"], remoteResult.Variables["Patch"])
	require.Equal(t, localResult.Variables["MajorMinorPatch"], remoteResult.Variables["MajorMinorPatch"])
	require.Equal(t, localResult.Variables["CommitsSinceVersionSource"], remoteResult.Variables["CommitsSinceVersionSource"])
}

// ---------------------------------------------------------------------------
// Library: Explain Mode
// ---------------------------------------------------------------------------

func TestLibrary_Calculate_WithExplain(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial release")
	repo.CreateTag("v1.0.0", sha)
	repo.AddCommit("feat: add dashboard")

	result, err := sdk.Calculate(sdk.LocalOptions{
		Path:    repo.Path(),
		Explain: true,
	})
	require.NoError(t, err)
	require.NotNil(t, result.ExplainResult)

	er := result.ExplainResult

	// Candidates should be populated.
	require.NotEmpty(t, er.Candidates)

	// At least one candidate should be TaggedCommit.
	hasTagged := false
	for _, c := range er.Candidates {
		if c.Strategy == "TaggedCommit" {
			hasTagged = true
			require.Equal(t, "1.0.0", c.Version)
			require.NotEmpty(t, c.Steps)
		}
	}
	require.True(t, hasTagged, "should have TaggedCommit candidate")

	// Selected source should be set.
	require.NotEmpty(t, er.SelectedSource)

	// Increment steps should be populated.
	require.NotEmpty(t, er.IncrementSteps)

	// Final version should be set.
	require.NotEmpty(t, er.FinalVersion)

	// Formatted output should contain key sections.
	require.Contains(t, er.FormattedOutput, "Strategies evaluated:")
	require.Contains(t, er.FormattedOutput, "Selected:")
	require.Contains(t, er.FormattedOutput, "Result:")
}

func TestLibrary_Calculate_WithoutExplain(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.AddCommit("initial commit")

	result, err := sdk.Calculate(sdk.LocalOptions{
		Path:    repo.Path(),
		Explain: false,
	})
	require.NoError(t, err)
	require.Nil(t, result.ExplainResult, "ExplainResult should be nil when Explain=false")
}

func TestLibrary_Calculate_ExplainFeatureBranch(t *testing.T) {
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

	// Feature branch should have pre-release steps.
	require.NotEmpty(t, result.ExplainResult.PreReleaseSteps)
}

func TestLibrary_CalculateRemote_WithExplain(t *testing.T) {
	tagSha := sha("tag100")
	tipSha := sha("tip200")

	server := newLibraryMockServer(t,
		tipSha, "feat: new feature", "2025-01-01T12:01:00Z",
		[]string{tagSha},
		[]mockTag{{name: "v1.0.0", commitSha: tagSha}},
		[]mockCommit{
			{sha: tagSha, message: "initial", date: "2025-01-01T12:00:00Z"},
			{sha: tipSha, message: "feat: new feature", date: "2025-01-01T12:01:00Z", parents: []string{tagSha}},
		},
	)
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

	er := result.ExplainResult
	require.NotEmpty(t, er.Candidates)
	require.NotEmpty(t, er.SelectedSource)
	require.NotEmpty(t, er.FinalVersion)
	require.Contains(t, er.FormattedOutput, "Strategies evaluated:")
}
