package cmd

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	gh "github.com/google/go-github/v68/github"
	"github.com/stretchr/testify/require"

	ghprovider "go-gitsemver/internal/github"
)

func TestParseOwnerRepo_Valid(t *testing.T) {
	owner, repo, err := parseOwnerRepo("myorg/myrepo")
	require.NoError(t, err)
	require.Equal(t, "myorg", owner)
	require.Equal(t, "myrepo", repo)
}

func TestParseOwnerRepo_NestedPath(t *testing.T) {
	// "owner/repo/extra" should only split on first "/".
	owner, repo, err := parseOwnerRepo("myorg/myrepo/extra")
	require.NoError(t, err)
	require.Equal(t, "myorg", owner)
	require.Equal(t, "myrepo/extra", repo)
}

func TestParseOwnerRepo_NoSlash(t *testing.T) {
	_, _, err := parseOwnerRepo("myrepo")
	require.Error(t, err)
	require.Contains(t, err.Error(), "expected owner/repo")
}

func TestParseOwnerRepo_EmptyOwner(t *testing.T) {
	_, _, err := parseOwnerRepo("/myrepo")
	require.Error(t, err)
}

func TestParseOwnerRepo_EmptyRepo(t *testing.T) {
	_, _, err := parseOwnerRepo("myorg/")
	require.Error(t, err)
}

func TestParseOwnerRepo_Empty(t *testing.T) {
	_, _, err := parseOwnerRepo("")
	require.Error(t, err)
}

func TestRemoteCmd_HasExpectedFlags(t *testing.T) {
	flags := remoteCmd.Flags()

	require.NotNil(t, flags.Lookup("token"))
	require.NotNil(t, flags.Lookup("github-app-id"))
	require.NotNil(t, flags.Lookup("github-app-key"))
	require.NotNil(t, flags.Lookup("github-url"))
	require.NotNil(t, flags.Lookup("ref"))
	require.NotNil(t, flags.Lookup("max-commits"))
}

func TestRemoteCmd_MaxCommitsDefault(t *testing.T) {
	f := remoteCmd.Flags().Lookup("max-commits")
	require.NotNil(t, f)
	require.Equal(t, "1000", f.DefValue)
}

func TestRemoteCmd_IsRegistered(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == "remote" {
			found = true
			break
		}
	}
	require.True(t, found, "remote subcommand should be registered")
}

func writeTestJSON(w http.ResponseWriter, v interface{}) {
	if err := json.NewEncoder(w).Encode(v); err != nil {
		panic(err)
	}
}

func newTestGHRepo(t *testing.T, mux *http.ServeMux) (*ghprovider.GitHubRepository, func()) {
	t.Helper()
	server := httptest.NewServer(mux)
	client, err := gh.NewClient(nil).WithEnterpriseURLs(server.URL+"/", server.URL+"/")
	require.NoError(t, err)
	repo := ghprovider.NewGitHubRepository(client, "testowner", "testrepo")
	return repo, server.Close
}

func TestLoadRemoteConfig_FetchesFromRemote(t *testing.T) {
	configYAML := "next-version: 3.0.0\n"
	encoded := base64.StdEncoding.EncodeToString([]byte(configYAML))

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/contents/GitVersion.yml", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(w, map[string]interface{}{
			"type":     "file",
			"encoding": "base64",
			"content":  encoded,
		})
	})

	ghRepo, cleanup := newTestGHRepo(t, mux)
	defer cleanup()

	// Clear flagConfig so it tries remote.
	flagConfig = ""
	defer func() { flagConfig = "" }()

	cfg, err := loadRemoteConfig(ghRepo)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Equal(t, "3.0.0", *cfg.NextVersion)
}

func TestLoadRemoteConfig_FallsBackToGitsemverYml(t *testing.T) {
	configYAML := "next-version: 4.0.0\n"
	encoded := base64.StdEncoding.EncodeToString([]byte(configYAML))

	mux := http.NewServeMux()
	// GitVersion.yml returns 404.
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/contents/GitVersion.yml", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
	})
	// gitsemver.yml succeeds.
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/contents/gitsemver.yml", func(w http.ResponseWriter, r *http.Request) {
		writeTestJSON(w, map[string]interface{}{
			"type":     "file",
			"encoding": "base64",
			"content":  encoded,
		})
	})

	ghRepo, cleanup := newTestGHRepo(t, mux)
	defer cleanup()

	flagConfig = ""
	defer func() { flagConfig = "" }()

	cfg, err := loadRemoteConfig(ghRepo)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Equal(t, "4.0.0", *cfg.NextVersion)
}

func TestLoadRemoteConfig_NoRemoteConfig_UsesDefaults(t *testing.T) {
	mux := http.NewServeMux()
	// Both config files return 404.
	mux.HandleFunc("/api/v3/repos/testowner/testrepo/contents/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
	})

	ghRepo, cleanup := newTestGHRepo(t, mux)
	defer cleanup()

	flagConfig = ""
	defer func() { flagConfig = "" }()

	cfg, err := loadRemoteConfig(ghRepo)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	// Should have default branches.
	require.NotNil(t, cfg.Branches)
}

func TestLoadRemoteConfig_LocalConfigOverride(t *testing.T) {
	mux := http.NewServeMux()
	ghRepo, cleanup := newTestGHRepo(t, mux)
	defer cleanup()

	// Create a local config file.
	dir := t.TempDir()
	path := filepath.Join(dir, "local.yml")
	require.NoError(t, os.WriteFile(path, []byte("next-version: 9.0.0\n"), 0o644))

	flagConfig = path
	defer func() { flagConfig = "" }()

	cfg, err := loadRemoteConfig(ghRepo)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Equal(t, "9.0.0", *cfg.NextVersion)
}

func TestLoadRemoteConfig_LocalConfigInvalid(t *testing.T) {
	mux := http.NewServeMux()
	ghRepo, cleanup := newTestGHRepo(t, mux)
	defer cleanup()

	flagConfig = "/nonexistent/config.yml"
	defer func() { flagConfig = "" }()

	_, err := loadRemoteConfig(ghRepo)
	require.Error(t, err)
}
