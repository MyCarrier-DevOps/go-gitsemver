package github

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	gh "github.com/google/go-github/v68/github"
	"github.com/stretchr/testify/require"
)

func TestResolveString_FlagTakesPrecedence(t *testing.T) {
	t.Setenv("TEST_VAR", "env_value")
	result := resolveString("flag_value", "TEST_VAR")
	require.Equal(t, "flag_value", result)
}

func TestResolveString_FallsBackToEnv(t *testing.T) {
	t.Setenv("TEST_VAR", "env_value")
	result := resolveString("", "TEST_VAR")
	require.Equal(t, "env_value", result)
}

func TestResolveString_ReturnsEmptyWhenBothEmpty(t *testing.T) {
	os.Unsetenv("TEST_VAR_EMPTY")
	result := resolveString("", "TEST_VAR_EMPTY")
	require.Equal(t, "", result)
}

func TestNewClient_NoAuth(t *testing.T) {
	// Ensure no auth env vars are set.
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_APP_ID", "")
	t.Setenv("GH_APP_PRIVATE_KEY", "")

	_, err := NewClient(ClientConfig{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no GitHub authentication provided")
}

func TestNewClient_TokenAuth(t *testing.T) {
	client, err := NewClient(ClientConfig{Token: "ghp_test_token"})
	require.NoError(t, err)
	require.NotNil(t, client)
}

func TestNewClient_TokenFromEnv(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "ghp_env_token")
	client, err := NewClient(ClientConfig{})
	require.NoError(t, err)
	require.NotNil(t, client)
}

func TestNewClient_TokenWithBaseURL(t *testing.T) {
	client, err := NewClient(ClientConfig{
		Token:   "ghp_test",
		BaseURL: "https://ghe.example.com/api/v3",
	})
	require.NoError(t, err)
	require.NotNil(t, client)
}

func TestNewClient_AppAuthMissingKey(t *testing.T) {
	// AppID set but no key path — should fall through to "no auth" error.
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_APP_PRIVATE_KEY", "")

	_, err := NewClient(ClientConfig{AppID: 12345})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no GitHub authentication provided")
}

func TestNewClient_AppAuthBadKeyFile(t *testing.T) {
	// Both AppID and key path set, but key file doesn't exist.
	t.Setenv("GITHUB_TOKEN", "")

	_, err := NewClient(ClientConfig{
		AppID:      12345,
		AppKeyPath: "/nonexistent/key.pem",
		Owner:      "testorg",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "creating GitHub App transport")
}

func TestNewClient_AppIDFromEnv(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_APP_ID", "99999")
	t.Setenv("GH_APP_PRIVATE_KEY", "/nonexistent/key.pem")

	_, err := NewClient(ClientConfig{Owner: "testorg"})
	require.Error(t, err)
	// Should get past the "no auth" check and fail on the key file.
	require.Contains(t, err.Error(), "creating GitHub App transport")
}

func TestFindInstallation_Found(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/app/installations", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, []map[string]interface{}{
			{
				"id":      int64(111),
				"account": map[string]interface{}{"login": "other-org"},
			},
			{
				"id":      int64(222),
				"account": map[string]interface{}{"login": "target-org"},
			},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client, err := gh.NewClient(nil).WithEnterpriseURLs(server.URL+"/", server.URL+"/")
	require.NoError(t, err)

	id, err := findInstallation(client, "target-org")
	require.NoError(t, err)
	require.Equal(t, int64(222), id)
}

func TestFindInstallation_NotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/app/installations", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		writeJSON(w, []map[string]interface{}{
			{
				"id":      int64(111),
				"account": map[string]interface{}{"login": "other-org"},
			},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client, err := gh.NewClient(nil).WithEnterpriseURLs(server.URL+"/", server.URL+"/")
	require.NoError(t, err)

	_, err = findInstallation(client, "missing-org")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no GitHub App installation found")
}

func TestFindInstallation_APIError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/app/installations", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"Unauthorized"}`, http.StatusUnauthorized)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client, err := gh.NewClient(nil).WithEnterpriseURLs(server.URL+"/", server.URL+"/")
	require.NoError(t, err)

	_, err = findInstallation(client, "any-org")
	require.Error(t, err)
	require.Contains(t, err.Error(), "listing GitHub App installations")
}

func TestNewClient_BaseURLFromEnv(t *testing.T) {
	t.Setenv("GITHUB_API_URL", "https://ghe.example.com/api/v3")
	client, err := NewClient(ClientConfig{Token: "ghp_test"})
	require.NoError(t, err)
	require.NotNil(t, client)
}

func TestNewClient_InvalidAppIDEnv(t *testing.T) {
	// Non-numeric GH_APP_ID should be ignored (falls through to no-auth error).
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_APP_ID", "not-a-number")
	t.Setenv("GH_APP_PRIVATE_KEY", "")

	_, err := NewClient(ClientConfig{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no GitHub authentication provided")
}

func TestIsNotFoundError(t *testing.T) {
	// 404 error should be detected.
	resp := &http.Response{StatusCode: 404}
	notFoundErr := &gh.ErrorResponse{Response: resp}
	require.True(t, IsNotFoundError(notFoundErr))

	// Wrapped 404 should still be detected.
	wrappedErr := fmt.Errorf("fetching file: %w", notFoundErr)
	require.True(t, IsNotFoundError(wrappedErr))

	// 401 error should not be detected as "not found".
	resp401 := &http.Response{StatusCode: 401}
	authErr := &gh.ErrorResponse{Response: resp401}
	require.False(t, IsNotFoundError(authErr))

	// Non-GitHub error should not match.
	require.False(t, IsNotFoundError(fmt.Errorf("some other error")))

	// Nil error should return false.
	require.False(t, IsNotFoundError(nil))
}

func TestResolveBaseURL(t *testing.T) {
	// Flag takes precedence.
	require.Equal(t, "https://ghe.example.com/api/v3", ResolveBaseURL("https://ghe.example.com/api/v3"))

	// Falls back to env.
	t.Setenv("GITHUB_API_URL", "https://ghe-env.example.com/api/v3")
	require.Equal(t, "https://ghe-env.example.com/api/v3", ResolveBaseURL(""))

	// Empty when both empty.
	t.Setenv("GITHUB_API_URL", "")
	require.Equal(t, "", ResolveBaseURL(""))
}
