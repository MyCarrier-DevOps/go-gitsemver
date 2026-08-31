package sdk

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MyCarrier-DevOps/go-gitsemver/internal/calculator"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/git"
	ghprovider "github.com/MyCarrier-DevOps/go-gitsemver/internal/github"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/semver"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/strategy"

	gh "github.com/google/go-github/v68/github"
	"github.com/stretchr/testify/require"
)

// TestBuildExplainResult_CandidateSource pins both source spellings: a
// candidate anchored at a commit reports that commit's short SHA, and one with
// no anchor reports "external".
func TestBuildExplainResult_CandidateSource(t *testing.T) {
	anchor := git.Commit{Sha: "abcdef1234567890abcdef1234567890abcdef12"}

	er := buildExplainResult(calculator.VersionResult{
		Version:     semver.SemanticVersion{Major: 1, Minor: 2, Patch: 3},
		BaseVersion: strategy.BaseVersion{Source: "TaggedCommit"},
		AllCandidates: []strategy.BaseVersion{
			{Source: "TaggedCommit", SemanticVersion: semver.SemanticVersion{Major: 1}, BaseVersionSource: &anchor},
			{Source: "ConfigNextVersion", SemanticVersion: semver.SemanticVersion{Major: 2}},
		},
	})

	require.Len(t, er.Candidates, 2)
	require.Equal(t, anchor.ShortSha(), er.Candidates[0].Source)
	require.Equal(t, "external", er.Candidates[1].Source, "a candidate with no source commit is external")
}

// remoteRepoServing builds a GitHubRepository whose contents endpoint serves
// the given path -> body map and 404s everything else.
func remoteRepoServing(t *testing.T, files map[string]string) *ghprovider.GitHubRepository {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/contents/", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Path[len("/api/v3/repos/testowner/testrepo/contents/"):]
		body, ok := files[name]
		if !ok {
			http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"type":"file","encoding":"base64","content":"` +
			base64.StdEncoding.EncodeToString([]byte(body)) + `"}`))
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client, err := gh.NewClient(nil).WithEnterpriseURLs(server.URL+"/", server.URL+"/")
	require.NoError(t, err)
	return ghprovider.NewGitHubRepository(client, "testowner", "testrepo")
}

// TestLoadRemoteConfig_AutoDetectUsesFoundFile pins that a config file found
// during auto-detection is actually applied, and that a 404 is skipped rather
// than treated as a failure.
func TestLoadRemoteConfig_AutoDetectUsesFoundFile(t *testing.T) {
	// Only the third candidate name exists, so the first two must 404 and be skipped.
	ghRepo := remoteRepoServing(t, map[string]string{"GitVersion.yml": "next-version: 7.1.0\n"})

	cfg, err := loadRemoteConfig("", "", ghRepo)
	require.NoError(t, err)
	require.NotNil(t, cfg.NextVersion)
	require.Equal(t, "7.1.0", *cfg.NextVersion)
}

// TestLoadRemoteConfig_AutoDetectNoneFound falls back to defaults when the repo
// carries no configuration at all.
func TestLoadRemoteConfig_AutoDetectNoneFound(t *testing.T) {
	ghRepo := remoteRepoServing(t, nil)

	cfg, err := loadRemoteConfig("", "", ghRepo)
	require.NoError(t, err)
	require.Nil(t, cfg.NextVersion)
	require.NotEmpty(t, cfg.Branches)
}

const (
	sdkTipSha  = "1111111111111111111111111111111111111111"
	sdkMidSha  = "4444444444444444444444444444444444444444"
	sdkBaseSha = "2222222222222222222222222222222222222222"
	sdkAltSha  = "3333333333333333333333333333333333333333"
)

// sdkRemoteServer stands in for the GitHub API for a whole remote calculation.
// The repository default branch ("trunk") differs from "main", and history is
// paged with a breaking change on page two behind a failing compare API, so
// both the ref choice and the commit cap are observable in the result.
func sdkRemoteServer(t *testing.T) *httptest.Server {
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
	refNode := func(name, oid, msg, parent string) map[string]any {
		parents := map[string]any{"nodes": []any{}}
		if parent != "" {
			parents = map[string]any{"nodes": []map[string]any{{"oid": parent}}}
		}
		return map[string]any{"name": name, "target": map[string]any{
			"__typename": "Commit", "oid": oid, "message": msg,
			"committedDate": "2025-01-02T00:00:00Z", "parents": parents,
		}}
	}

	mux.HandleFunc("/api/v3/repos/testowner/testrepo/contents/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
	})
	mux.HandleFunc("/api/v3/repos/testowner/testrepo", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"name": "testrepo", "default_branch": "trunk"})
	})
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/branches/main", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"name": "main", "commit": commitJSON(sdkTipSha, "feat: add search", sdkMidSha)})
	})
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/branches/trunk", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"name": "trunk", "commit": commitJSON(sdkAltSha, "fix: a bug", sdkBaseSha)})
	})
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/compare/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"boom"}`, http.StatusInternalServerError)
	})
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/commits/", func(w http.ResponseWriter, r *http.Request) {
		switch strings.TrimPrefix(r.URL.Path, "/api/v3/repos/testowner/testrepo/commits/") {
		case sdkTipSha:
			writeJSON(w, commitJSON(sdkTipSha, "feat: add search", sdkMidSha))
		case sdkMidSha:
			writeJSON(w, commitJSON(sdkMidSha, "feat!: redesign the API", sdkBaseSha))
		case sdkAltSha:
			writeJSON(w, commitJSON(sdkAltSha, "fix: a bug", sdkBaseSha))
		default:
			writeJSON(w, commitJSON(sdkBaseSha, "initial"))
		}
	})
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/commits", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "2" {
			writeJSON(w, []map[string]any{
				commitJSON(sdkMidSha, "feat!: redesign the API", sdkBaseSha),
				commitJSON(sdkBaseSha, "initial"),
			})
			return
		}
		w.Header().Set("Link", `<https://example.com/commits?page=2>; rel="next"`)
		writeJSON(w, []map[string]any{commitJSON(sdkTipSha, "feat: add search", sdkMidSha)})
	})
	mux.HandleFunc("/graphql", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "refs/tags/") {
			writeJSON(w, map[string]any{"data": map[string]any{"repository": map[string]any{"refs": map[string]any{
				"nodes":    []map[string]any{refNode("v1.2.0", sdkBaseSha, "initial", "")},
				"pageInfo": map[string]any{"hasNextPage": false, "endCursor": ""},
			}}}})
			return
		}
		writeJSON(w, map[string]any{"data": map[string]any{"repository": map[string]any{"refs": map[string]any{
			"nodes": []map[string]any{
				refNode("main", sdkTipSha, "feat: add search", sdkMidSha),
				refNode("trunk", sdkAltSha, "fix: a bug", sdkBaseSha),
			},
			"pageInfo": map[string]any{"hasNextPage": false, "endCursor": ""},
		}}}})
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

// TestCalculateRemote_UsesRefAndUncappedWalk pins two things at once: Ref
// selects the branch rather than falling back to the repository default, and
// MaxCommits of zero means "unset" rather than a zero cap that would stop the
// walk before the breaking change on page two.
func TestCalculateRemote_UsesRefAndUncappedWalk(t *testing.T) {
	server := sdkRemoteServer(t)

	result, err := CalculateRemote(RemoteOptions{
		Owner:      "testowner",
		Repo:       "testrepo",
		Token:      "ghp_test",
		Ref:        "main",
		BaseURL:    server.URL + "/",
		MaxCommits: 0,
	})
	require.NoError(t, err)
	require.Equal(t, "main", result.Variables["BranchName"], "Ref must select the branch")
	require.Equal(t, "2.0.0", result.Variables["MajorMinorPatch"],
		"a zero MaxCommits must not cap the walk")
}

// TestCalculateRemote_NoRefUsesDefaultBranch pins the fallback when Ref is unset.
func TestCalculateRemote_NoRefUsesDefaultBranch(t *testing.T) {
	server := sdkRemoteServer(t)

	result, err := CalculateRemote(RemoteOptions{
		Owner:   "testowner",
		Repo:    "testrepo",
		Token:   "ghp_test",
		BaseURL: server.URL + "/",
	})
	require.NoError(t, err)
	require.Equal(t, "trunk", result.Variables["BranchName"])
}
