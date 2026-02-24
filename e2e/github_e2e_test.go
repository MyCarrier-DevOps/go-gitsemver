// Package e2e contains end-to-end tests that exercise the full version
// calculation pipeline via the GitHub API mock server.
//
// These tests construct realistic mock GitHub API responses (REST + GraphQL)
// and run the full pipeline through GitHubRepository → RepositoryStore →
// context → strategies → calculator → output.
package e2e

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"go-gitsemver/internal/calculator"
	"go-gitsemver/internal/config"
	configctx "go-gitsemver/internal/context"
	"go-gitsemver/internal/git"
	ghprovider "go-gitsemver/internal/github"
	"go-gitsemver/internal/output"
	"go-gitsemver/internal/strategy"
	"go-gitsemver/internal/testutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gh "github.com/google/go-github/v68/github"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock GitHub server helpers
// ---------------------------------------------------------------------------

// ghMock builds a mock GitHub API server for e2e tests. It supports
// registering branches, tags, and commits that the pipeline will query.
type ghMock struct {
	mux     *http.ServeMux
	commits map[string]mockCommit // SHA → commit
	tags    []mockTag
	branch  string // default branch name
	tipSha  string // tip of default branch
}

type mockCommit struct {
	sha     string
	message string
	date    string
	parents []string
}

type mockTag struct {
	name      string
	commitSha string
}

func newGHMock(defaultBranch, tipSha string) *ghMock {
	return &ghMock{
		mux:     http.NewServeMux(),
		commits: make(map[string]mockCommit),
		branch:  defaultBranch,
		tipSha:  tipSha,
	}
}

func (m *ghMock) addCommit(sha, message, date string, parents ...string) {
	m.commits[sha] = mockCommit{sha: sha, message: message, date: date, parents: parents}
}

func (m *ghMock) addTag(name, commitSha string) {
	m.tags = append(m.tags, mockTag{name: name, commitSha: commitSha})
}

func (m *ghMock) register() {
	// GET /api/v3/repos/{owner}/{repo} — default branch info.
	m.mux.HandleFunc("/api/v3/repos/testowner/testrepo", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/repos/testowner/testrepo" {
			http.NotFound(w, r)
			return
		}
		writeGHJSON(w, map[string]interface{}{
			"default_branch": m.branch,
		})
	})

	// GET /api/v3/repos/{owner}/{repo}/branches/{branch}
	m.mux.HandleFunc("/api/v3/repos/testowner/testrepo/branches/", func(w http.ResponseWriter, r *http.Request) {
		branchName := strings.TrimPrefix(r.URL.Path, "/api/v3/repos/testowner/testrepo/branches/")
		if branchName == "" {
			http.NotFound(w, r)
			return
		}

		tip, ok := m.commits[m.tipSha]
		if !ok {
			http.NotFound(w, r)
			return
		}

		parents := make([]map[string]interface{}, 0, len(tip.parents))
		for _, p := range tip.parents {
			parents = append(parents, map[string]interface{}{"sha": p})
		}

		writeGHJSON(w, map[string]interface{}{
			"name": branchName,
			"commit": map[string]interface{}{
				"sha": tip.sha,
				"commit": map[string]interface{}{
					"message":   tip.message,
					"committer": map[string]interface{}{"date": tip.date},
				},
				"parents": parents,
			},
		})
	})

	// GraphQL — branches and tags.
	m.mux.HandleFunc("/api/graphql", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Query     string                 `json:"query"`
			Variables map[string]interface{} `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if strings.Contains(body.Query, "refs/heads/") {
			m.handleBranchesGraphQL(w)
		} else {
			m.handleTagsGraphQL(w)
		}
	})

	// GET /api/v3/repos/{owner}/{repo}/commits/{sha}
	m.mux.HandleFunc("/api/v3/repos/testowner/testrepo/commits/", func(w http.ResponseWriter, r *http.Request) {
		sha := strings.TrimPrefix(r.URL.Path, "/api/v3/repos/testowner/testrepo/commits/")
		if sha == "" {
			http.NotFound(w, r)
			return
		}

		c, ok := m.commits[sha]
		if !ok {
			http.NotFound(w, r)
			return
		}

		parents := make([]map[string]interface{}, 0, len(c.parents))
		for _, p := range c.parents {
			parents = append(parents, map[string]interface{}{"sha": p})
		}

		writeGHJSON(w, map[string]interface{}{
			"sha": c.sha,
			"commit": map[string]interface{}{
				"message":   c.message,
				"committer": map[string]interface{}{"date": c.date},
			},
			"parents": parents,
		})
	})

	// GET /api/v3/repos/{owner}/{repo}/commits — paginated commit listing.
	m.mux.HandleFunc("/api/v3/repos/testowner/testrepo/commits", func(w http.ResponseWriter, r *http.Request) {
		tipSha := r.URL.Query().Get("sha")
		if tipSha == "" {
			tipSha = m.tipSha
		}

		// Walk backward from tipSha following first parent.
		var result []map[string]interface{}
		current := tipSha
		for current != "" {
			c, ok := m.commits[current]
			if !ok {
				break
			}

			parents := make([]map[string]interface{}, 0, len(c.parents))
			for _, p := range c.parents {
				parents = append(parents, map[string]interface{}{"sha": p})
			}

			result = append(result, map[string]interface{}{
				"sha": c.sha,
				"commit": map[string]interface{}{
					"message":   c.message,
					"committer": map[string]interface{}{"date": c.date},
				},
				"parents": parents,
			})

			if len(c.parents) > 0 {
				current = c.parents[0]
			} else {
				current = ""
			}
		}

		writeGHJSON(w, result)
	})

	// GET /api/v3/repos/{owner}/{repo}/compare/{base}...{head}
	m.mux.HandleFunc("/api/v3/repos/testowner/testrepo/compare/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/v3/repos/testowner/testrepo/compare/")
		parts := strings.SplitN(path, "...", 2)
		if len(parts) != 2 {
			http.Error(w, "bad compare path", http.StatusBadRequest)
			return
		}
		baseSha := parts[0]
		headSha := parts[1]

		// Walk from head backward to base to collect commits.
		var commits []map[string]interface{}
		current := headSha
		for current != "" && current != baseSha {
			c, ok := m.commits[current]
			if !ok {
				break
			}
			parents := make([]map[string]interface{}, 0, len(c.parents))
			for _, p := range c.parents {
				parents = append(parents, map[string]interface{}{"sha": p})
			}
			commits = append(commits, map[string]interface{}{
				"sha": c.sha,
				"commit": map[string]interface{}{
					"message":   c.message,
					"committer": map[string]interface{}{"date": c.date},
				},
				"parents": parents,
			})
			if len(c.parents) > 0 {
				current = c.parents[0]
			} else {
				current = ""
			}
		}

		// Reverse to forward chronological order (compare API returns forward).
		for i, j := 0, len(commits)-1; i < j; i, j = i+1, j-1 {
			commits[i], commits[j] = commits[j], commits[i]
		}

		// Find merge base commit.
		var mergeBase map[string]interface{}
		if c, ok := m.commits[baseSha]; ok {
			mergeBase = map[string]interface{}{
				"sha": c.sha,
				"commit": map[string]interface{}{
					"message":   c.message,
					"committer": map[string]interface{}{"date": c.date},
				},
			}
		}

		writeGHJSON(w, map[string]interface{}{
			"total_commits":     len(commits),
			"commits":           commits,
			"merge_base_commit": mergeBase,
			"status":            "ahead",
		})
	})

	// Config files — return 404 by default.
	m.mux.HandleFunc("/api/v3/repos/testowner/testrepo/contents/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		writeGHJSON(w, &gh.ErrorResponse{
			Response: &http.Response{StatusCode: http.StatusNotFound},
			Message:  "Not Found",
		})
	})
}

// registerConfigFile overrides the default 404 handler with a config file response.
func (m *ghMock) registerConfigFile(filename, yamlContent string) {
	encoded := base64.StdEncoding.EncodeToString([]byte(yamlContent))
	m.mux.HandleFunc("/api/v3/repos/testowner/testrepo/contents/"+filename, func(w http.ResponseWriter, r *http.Request) {
		writeGHJSON(w, map[string]interface{}{
			"type":     "file",
			"encoding": "base64",
			"content":  encoded,
		})
	})
}

func (m *ghMock) handleBranchesGraphQL(w http.ResponseWriter) {
	tip := m.commits[m.tipSha]
	parentNodes := make([]map[string]interface{}, 0, len(tip.parents))
	for _, p := range tip.parents {
		parentNodes = append(parentNodes, map[string]interface{}{"oid": p})
	}

	writeGHJSON(w, map[string]interface{}{
		"data": map[string]interface{}{
			"repository": map[string]interface{}{
				"refs": map[string]interface{}{
					"nodes": []map[string]interface{}{
						{
							"name": m.branch,
							"target": map[string]interface{}{
								"oid":           tip.sha,
								"message":       tip.message,
								"committedDate": tip.date,
								"parents":       map[string]interface{}{"nodes": parentNodes},
							},
						},
					},
					"pageInfo": map[string]interface{}{"hasNextPage": false, "endCursor": ""},
				},
			},
		},
	})
}

func (m *ghMock) handleTagsGraphQL(w http.ResponseWriter) {
	nodes := make([]map[string]interface{}, 0, len(m.tags))
	for _, tag := range m.tags {
		nodes = append(nodes, map[string]interface{}{
			"name": tag.name,
			"target": map[string]interface{}{
				"oid":           tag.commitSha,
				"message":       "",
				"committedDate": m.commits[tag.commitSha].date,
				"parents":       map[string]interface{}{"nodes": []interface{}{}},
			},
		})
	}

	writeGHJSON(w, map[string]interface{}{
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

func writeGHJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		panic(err)
	}
}

// runGitHubPipeline creates a GitHubRepository from the mock server and runs
// the full calculation pipeline, returning the output variables.
func runGitHubPipeline(t *testing.T, mock *ghMock) map[string]string {
	t.Helper()
	return runGitHubPipelineWithConfig(t, mock, nil)
}

func runGitHubPipelineWithConfig(t *testing.T, mock *ghMock, userCfg *config.Config) map[string]string {
	t.Helper()

	mock.register()
	server := httptest.NewServer(mock.mux)
	defer server.Close()

	client, err := gh.NewClient(nil).WithEnterpriseURLs(server.URL+"/api/v3", server.URL+"/api/v3")
	require.NoError(t, err)

	ghRepo := ghprovider.NewGitHubRepository(client, "testowner", "testrepo",
		ghprovider.WithBaseURL(server.URL+"/api/v3"),
	)

	builder := config.NewBuilder()
	if userCfg != nil {
		builder.Add(userCfg)
	}
	cfg, err := builder.Build()
	require.NoError(t, err)

	store := git.NewRepositoryStore(ghRepo)
	ctx, err := configctx.NewContext(store, ghRepo, cfg, configctx.Options{})
	require.NoError(t, err)

	ec, err := ctx.GetEffectiveConfiguration(ctx.CurrentBranch.FriendlyName())
	require.NoError(t, err)

	strategies := strategy.AllStrategies(store)
	calc := calculator.NewNextVersionCalculator(store, strategies)
	result, err := calc.Calculate(ctx, ec, false)
	require.NoError(t, err)

	return output.GetVariables(result.Version, ec)
}

// sha generates a deterministic 40-char hex SHA from a short identifier.
func sha(id string) string {
	base := fmt.Sprintf("%040s", id)
	return base[len(base)-40:]
}

// ---------------------------------------------------------------------------
// Tests: Fallback strategy via GitHub API
// ---------------------------------------------------------------------------

func TestGitHub_Fallback_NoTags(t *testing.T) {
	sha1 := sha("aaa111")
	sha2 := sha("bbb222")

	mock := newGHMock("main", sha2)
	mock.addCommit(sha1, "initial commit", "2025-01-01T12:00:00Z")
	mock.addCommit(sha2, "second commit", "2025-01-01T12:01:00Z", sha1)

	vars := runGitHubPipeline(t, mock)

	require.Equal(t, "0", vars["Major"])
	require.Equal(t, "1", vars["Minor"])
	require.Equal(t, "0", vars["Patch"])
}

// ---------------------------------------------------------------------------
// Tests: TaggedCommit strategy via GitHub API
// ---------------------------------------------------------------------------

func TestGitHub_TaggedCommit_ExactTag(t *testing.T) {
	sha1 := sha("ccc333")

	mock := newGHMock("main", sha1)
	mock.addCommit(sha1, "release commit", "2025-01-01T12:00:00Z")
	mock.addTag("v1.0.0", sha1)

	vars := runGitHubPipeline(t, mock)

	require.Equal(t, "1.0.0", vars["SemVer"])
	require.Equal(t, "1", vars["Major"])
	require.Equal(t, "0", vars["Minor"])
	require.Equal(t, "0", vars["Patch"])
}

func TestGitHub_TaggedCommit_CommitsAfterTag(t *testing.T) {
	sha1 := sha("ddd444")
	sha2 := sha("eee555")
	sha3 := sha("fff666")

	mock := newGHMock("main", sha3)
	mock.addCommit(sha1, "initial", "2025-01-01T12:00:00Z")
	mock.addCommit(sha2, "tagged release", "2025-01-01T12:01:00Z", sha1)
	mock.addCommit(sha3, "after tag", "2025-01-01T12:02:00Z", sha2)
	mock.addTag("v1.0.0", sha2)

	vars := runGitHubPipeline(t, mock)

	require.Equal(t, "1", vars["Major"])
	require.Equal(t, "1", vars["CommitsSinceVersionSource"])
}

func TestGitHub_TaggedCommit_MultipleTags(t *testing.T) {
	sha1 := sha("111aaa")
	sha2 := sha("222bbb")
	sha3 := sha("333ccc")

	mock := newGHMock("main", sha3)
	mock.addCommit(sha1, "first release", "2025-01-01T12:00:00Z")
	mock.addCommit(sha2, "second release", "2025-01-01T12:01:00Z", sha1)
	mock.addCommit(sha3, "after latest tag", "2025-01-01T12:02:00Z", sha2)
	mock.addTag("v1.0.0", sha1)
	mock.addTag("v2.0.0", sha2)

	vars := runGitHubPipeline(t, mock)

	require.Equal(t, "2", vars["Major"])
}

// ---------------------------------------------------------------------------
// Tests: Conventional Commits via GitHub API
// ---------------------------------------------------------------------------

func TestGitHub_ConventionalCommits_FeatBumpsMinor(t *testing.T) {
	sha1 := sha("cc1111")
	sha2 := sha("cc2222")

	mock := newGHMock("main", sha2)
	mock.addCommit(sha1, "initial", "2025-01-01T12:00:00Z")
	mock.addCommit(sha2, "feat: add auth", "2025-01-01T12:01:00Z", sha1)
	mock.addTag("v1.0.0", sha1)

	userCfg, err := config.LoadFromBytes([]byte("commit-message-convention: ConventionalCommits\n"))
	require.NoError(t, err)

	vars := runGitHubPipelineWithConfig(t, mock, userCfg)

	require.Equal(t, "1", vars["Major"])
	require.Equal(t, "1", vars["Minor"])
	require.Equal(t, "0", vars["Patch"])
}

func TestGitHub_ConventionalCommits_FixBumpsPatch(t *testing.T) {
	sha1 := sha("cc3333")
	sha2 := sha("cc4444")

	mock := newGHMock("main", sha2)
	mock.addCommit(sha1, "initial", "2025-01-01T12:00:00Z")
	mock.addCommit(sha2, "fix: null pointer", "2025-01-01T12:01:00Z", sha1)
	mock.addTag("v1.0.0", sha1)

	userCfg, err := config.LoadFromBytes([]byte("commit-message-convention: ConventionalCommits\n"))
	require.NoError(t, err)

	vars := runGitHubPipelineWithConfig(t, mock, userCfg)

	require.Equal(t, "1", vars["Major"])
	require.Equal(t, "0", vars["Minor"])
	require.Equal(t, "1", vars["Patch"])
}

func TestGitHub_ConventionalCommits_BreakingBumpsMajor(t *testing.T) {
	sha1 := sha("cc5555")
	sha2 := sha("cc6666")

	mock := newGHMock("main", sha2)
	mock.addCommit(sha1, "initial", "2025-01-01T12:00:00Z")
	mock.addCommit(sha2, "feat!: remove legacy API", "2025-01-01T12:01:00Z", sha1)
	mock.addTag("v1.0.0", sha1)

	userCfg, err := config.LoadFromBytes([]byte("commit-message-convention: ConventionalCommits\n"))
	require.NoError(t, err)

	vars := runGitHubPipelineWithConfig(t, mock, userCfg)

	require.Equal(t, "2", vars["Major"])
	require.Equal(t, "0", vars["Minor"])
	require.Equal(t, "0", vars["Patch"])
}

func TestGitHub_ConventionalCommits_MultipleCommitsHighestWins(t *testing.T) {
	sha1 := sha("mc1111")
	sha2 := sha("mc2222")
	sha3 := sha("mc3333")
	sha4 := sha("mc4444")

	mock := newGHMock("main", sha4)
	mock.addCommit(sha1, "initial", "2025-01-01T12:00:00Z")
	mock.addCommit(sha2, "fix: minor bug", "2025-01-01T12:01:00Z", sha1)
	mock.addCommit(sha3, "feat: big feature", "2025-01-01T12:02:00Z", sha2)
	mock.addCommit(sha4, "fix: another bug", "2025-01-01T12:03:00Z", sha3)
	mock.addTag("v1.0.0", sha1)

	userCfg, err := config.LoadFromBytes([]byte("commit-message-convention: ConventionalCommits\n"))
	require.NoError(t, err)

	vars := runGitHubPipelineWithConfig(t, mock, userCfg)

	require.Equal(t, "1", vars["Major"])
	require.Equal(t, "1", vars["Minor"])
	require.Equal(t, "0", vars["Patch"])
	require.Equal(t, "3", vars["CommitsSinceVersionSource"])
}

// ---------------------------------------------------------------------------
// Tests: Bump directives via GitHub API
// ---------------------------------------------------------------------------

func TestGitHub_BumpDirective_Major(t *testing.T) {
	sha1 := sha("bd1111")
	sha2 := sha("bd2222")

	mock := newGHMock("main", sha2)
	mock.addCommit(sha1, "initial", "2025-01-01T12:00:00Z")
	mock.addCommit(sha2, "release changes +semver: major", "2025-01-01T12:01:00Z", sha1)
	mock.addTag("v1.0.0", sha1)

	vars := runGitHubPipeline(t, mock)

	require.Equal(t, "2", vars["Major"])
}

// ---------------------------------------------------------------------------
// Tests: ConfigNextVersion via GitHub API
// ---------------------------------------------------------------------------

func TestGitHub_ConfigNextVersion(t *testing.T) {
	sha1 := sha("nv1111")

	mock := newGHMock("main", sha1)
	mock.addCommit(sha1, "initial", "2025-01-01T12:00:00Z")

	userCfg, err := config.LoadFromBytes([]byte("next-version: 5.0.0\n"))
	require.NoError(t, err)

	vars := runGitHubPipelineWithConfig(t, mock, userCfg)

	require.Equal(t, "5", vars["Major"])
	require.Equal(t, "0", vars["Minor"])
	require.Equal(t, "0", vars["Patch"])
}

// ---------------------------------------------------------------------------
// Tests: ContinuousDeployment mode via GitHub API
// ---------------------------------------------------------------------------

func TestGitHub_ContinuousDeployment(t *testing.T) {
	sha1 := sha("cd1111")
	sha2 := sha("cd2222")
	sha3 := sha("cd3333")

	mock := newGHMock("main", sha3)
	mock.addCommit(sha1, "initial", "2025-01-01T12:00:00Z")
	mock.addCommit(sha2, "second", "2025-01-01T12:01:00Z", sha1)
	mock.addCommit(sha3, "third", "2025-01-01T12:02:00Z", sha2)
	mock.addTag("v1.0.0", sha1)

	userCfg, err := config.LoadFromBytes([]byte("mode: ContinuousDeployment\ncontinuous-delivery-fallback-tag: ci\n"))
	require.NoError(t, err)

	vars := runGitHubPipelineWithConfig(t, mock, userCfg)

	require.Contains(t, vars["SemVer"], "ci.")
	require.Equal(t, "2", vars["CommitsSinceVersionSource"])
}

// ---------------------------------------------------------------------------
// Tests: Mainline mode via GitHub API
// ---------------------------------------------------------------------------

func TestGitHub_Mainline_AggregateIncrement(t *testing.T) {
	sha1 := sha("ml1111")
	sha2 := sha("ml2222")
	sha3 := sha("ml3333")

	mock := newGHMock("main", sha3)
	mock.addCommit(sha1, "initial", "2025-01-01T12:00:00Z")
	mock.addCommit(sha2, "fix: bug 1", "2025-01-01T12:01:00Z", sha1)
	mock.addCommit(sha3, "feat: new feature", "2025-01-01T12:02:00Z", sha2)
	mock.addTag("v1.0.0", sha1)

	userCfg, err := config.LoadFromBytes([]byte("mode: Mainline\ncommit-message-convention: ConventionalCommits\n"))
	require.NoError(t, err)

	vars := runGitHubPipelineWithConfig(t, mock, userCfg)

	require.Equal(t, "1", vars["Major"])
	require.Equal(t, "1", vars["Minor"])
	require.Equal(t, "0", vars["Patch"])
}

func TestGitHub_Mainline_EachCommit(t *testing.T) {
	sha1 := sha("me1111")
	sha2 := sha("me2222")
	sha3 := sha("me3333")
	sha4 := sha("me4444")
	sha5 := sha("me5555")

	mock := newGHMock("main", sha5)
	mock.addCommit(sha1, "initial", "2025-01-01T12:00:00Z")
	mock.addCommit(sha2, "fix: bug 1", "2025-01-01T12:01:00Z", sha1)
	mock.addCommit(sha3, "fix: bug 2", "2025-01-01T12:02:00Z", sha2)
	mock.addCommit(sha4, "feat: new feature", "2025-01-01T12:03:00Z", sha3)
	mock.addCommit(sha5, "fix: bug 3", "2025-01-01T12:04:00Z", sha4)
	mock.addTag("v1.0.0", sha1)

	userCfg, err := config.LoadFromBytes([]byte("mode: Mainline\nmainline-increment: EachCommit\ncommit-message-convention: ConventionalCommits\n"))
	require.NoError(t, err)

	vars := runGitHubPipelineWithConfig(t, mock, userCfg)

	// Per-commit: fix→1.0.1, fix→1.0.2, feat→1.1.0, fix→1.1.1
	require.Equal(t, "1", vars["Major"])
	require.Equal(t, "1", vars["Minor"])
	require.Equal(t, "1", vars["Patch"])
}

// ---------------------------------------------------------------------------
// Tests: Output completeness via GitHub API
// ---------------------------------------------------------------------------

func TestGitHub_OutputVariables_AllPresent(t *testing.T) {
	sha1 := sha("ov1111")
	sha2 := sha("ov2222")

	mock := newGHMock("main", sha2)
	mock.addCommit(sha1, "initial", "2025-01-01T12:00:00Z")
	mock.addCommit(sha2, "feat: something new", "2025-01-01T12:01:00Z", sha1)
	mock.addTag("v1.0.0", sha1)

	vars := runGitHubPipeline(t, mock)

	expectedKeys := []string{
		"Major", "Minor", "Patch", "MajorMinorPatch",
		"SemVer", "FullSemVer", "LegacySemVer", "LegacySemVerPadded",
		"InformationalVersion", "BranchName", "EscapedBranchName",
		"Sha", "ShortSha", "CommitDate", "CommitTag", "VersionSourceSha",
		"CommitsSinceVersionSource", "CommitsSinceVersionSourcePadded",
		"BuildMetaData", "BuildMetaDataPadded", "FullBuildMetaData",
		"PreReleaseTag", "PreReleaseTagWithDash", "PreReleaseLabel",
		"PreReleaseLabelWithDash", "PreReleaseNumber",
		"WeightedPreReleaseNumber",
		"AssemblySemVer", "AssemblySemFileVer", "AssemblyInformationalVersion",
		"NuGetVersion", "NuGetVersionV2", "NuGetPreReleaseTag", "NuGetPreReleaseTagV2",
		"UncommittedChanges",
	}

	for _, key := range expectedKeys {
		_, ok := vars[key]
		require.True(t, ok, "missing output variable %q", key)
	}
}

// ---------------------------------------------------------------------------
// Tests: Build metadata via GitHub API
// ---------------------------------------------------------------------------

func TestGitHub_BuildMetadata(t *testing.T) {
	sha1 := sha("bm1111")
	sha2 := sha("bm2222")

	mock := newGHMock("main", sha2)
	mock.addCommit(sha1, "initial", "2025-01-01T12:00:00Z")
	mock.addCommit(sha2, "second commit", "2025-01-01T12:01:00Z", sha1)
	mock.addTag("v1.0.0", sha1)

	vars := runGitHubPipeline(t, mock)

	require.NotEmpty(t, vars["Sha"])
	require.NotEmpty(t, vars["ShortSha"])
	require.Len(t, vars["ShortSha"], 7)
	require.NotEmpty(t, vars["CommitDate"])
	require.Equal(t, "1", vars["CommitsSinceVersionSource"])
}

// ---------------------------------------------------------------------------
// Tests: Custom base version via GitHub API
// ---------------------------------------------------------------------------

func TestGitHub_CustomBaseVersion(t *testing.T) {
	sha1 := sha("bv1111")

	mock := newGHMock("main", sha1)
	mock.addCommit(sha1, "initial", "2025-01-01T12:00:00Z")

	userCfg, err := config.LoadFromBytes([]byte("base-version: 1.0.0\n"))
	require.NoError(t, err)

	vars := runGitHubPipelineWithConfig(t, mock, userCfg)

	require.Equal(t, "1", vars["Major"])
	require.Equal(t, "0", vars["Minor"])
	require.Equal(t, "0", vars["Patch"])
}

// ---------------------------------------------------------------------------
// Tests: Remote config file loading
// ---------------------------------------------------------------------------

func TestGitHub_RemoteConfig(t *testing.T) {
	sha1 := sha("rc1111")

	mock := newGHMock("main", sha1)
	mock.addCommit(sha1, "initial", "2025-01-01T12:00:00Z")
	mock.registerConfigFile("gitsemver.yml", "next-version: 9.0.0\n")

	// Run the pipeline using the library API which handles remote config fetching.
	mock.register()
	server := httptest.NewServer(mock.mux)
	defer server.Close()

	client, err := gh.NewClient(nil).WithEnterpriseURLs(server.URL+"/api/v3", server.URL+"/api/v3")
	require.NoError(t, err)

	ghRepo := ghprovider.NewGitHubRepository(client, "testowner", "testrepo",
		ghprovider.WithBaseURL(server.URL+"/api/v3"),
	)

	// Fetch config from remote.
	content, err := ghRepo.FetchFileContent("gitsemver.yml")
	require.NoError(t, err)

	userCfg, err := config.LoadFromBytes([]byte(content))
	require.NoError(t, err)

	cfg, err := config.NewBuilder().Add(userCfg).Build()
	require.NoError(t, err)

	store := git.NewRepositoryStore(ghRepo)
	ctx, err := configctx.NewContext(store, ghRepo, cfg, configctx.Options{})
	require.NoError(t, err)

	ec, err := ctx.GetEffectiveConfiguration(ctx.CurrentBranch.FriendlyName())
	require.NoError(t, err)

	strategies := strategy.AllStrategies(store)
	calc := calculator.NewNextVersionCalculator(store, strategies)
	result, err := calc.Calculate(ctx, ec, false)
	require.NoError(t, err)

	vars := output.GetVariables(result.Version, ec)

	require.Equal(t, "9", vars["Major"])
	require.Equal(t, "0", vars["Minor"])
	require.Equal(t, "0", vars["Patch"])
}

// ---------------------------------------------------------------------------
// Tests: Parity with local tests — verify GitHub pipeline matches local
// ---------------------------------------------------------------------------

func TestGitHub_ParityWithLocal_FallbackNoTags(t *testing.T) {
	// Run same scenario locally and via GitHub mock, results should match.
	repo := testutil.NewTestRepo(t)
	repo.AddCommit("initial commit")
	localSha := repo.AddCommit("second commit")

	localVars := runPipeline(t, repo.Path())

	// Build equivalent GitHub mock.
	sha1 := sha("par111")

	mock := newGHMock("main", localSha)
	mock.addCommit(sha1, "initial commit", "2025-01-01T12:00:00Z")
	mock.addCommit(localSha, "second commit", "2025-01-01T12:01:00Z", sha1)

	ghVars := runGitHubPipeline(t, mock)

	// Core version fields should match.
	require.Equal(t, localVars["Major"], ghVars["Major"])
	require.Equal(t, localVars["Minor"], ghVars["Minor"])
	require.Equal(t, localVars["Patch"], ghVars["Patch"])
	require.Equal(t, localVars["MajorMinorPatch"], ghVars["MajorMinorPatch"])
}

func TestGitHub_ParityWithLocal_TaggedCommit(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	localSha := repo.AddCommit("release commit")
	repo.CreateTag("v3.0.0", localSha)

	localVars := runPipeline(t, repo.Path())

	mock := newGHMock("main", localSha)
	mock.addCommit(localSha, "release commit", "2025-01-01T12:00:00Z")
	mock.addTag("v3.0.0", localSha)

	ghVars := runGitHubPipeline(t, mock)

	require.Equal(t, localVars["SemVer"], ghVars["SemVer"])
	require.Equal(t, localVars["Major"], ghVars["Major"])
	require.Equal(t, localVars["Minor"], ghVars["Minor"])
	require.Equal(t, localVars["Patch"], ghVars["Patch"])
}
