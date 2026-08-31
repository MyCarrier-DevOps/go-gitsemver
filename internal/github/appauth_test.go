package github

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// testAppKeyPEM generates a PEM-encoded RSA private key usable as a GitHub App key.
func testAppKeyPEM(t *testing.T) []byte {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
}

// appInstallationServer serves the installations endpoint used to discover the
// installation ID for an owner.
func appInstallationServer(t *testing.T, owner string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/app/installations", func(w http.ResponseWriter, _ *http.Request) {
		id := int64(4242)
		require.NoError(t, json.NewEncoder(w).Encode([]map[string]any{
			{"id": id, "account": map[string]any{"login": owner}},
		}))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func TestNewClient_AppKeyContent_Succeeds(t *testing.T) {
	server := appInstallationServer(t, "testowner")

	client, err := NewClient(ClientConfig{
		AppID:   123,
		AppKey:  string(testAppKeyPEM(t)),
		Owner:   "testowner",
		BaseURL: server.URL + "/",
	})
	require.NoError(t, err)
	require.NotNil(t, client)
	require.Contains(t, client.BaseURL.String(), server.URL)
}

func TestNewClient_AppKeyFile_Succeeds(t *testing.T) {
	server := appInstallationServer(t, "testowner")

	keyPath := filepath.Join(t.TempDir(), "app.pem")
	require.NoError(t, os.WriteFile(keyPath, testAppKeyPEM(t), 0o600))

	client, err := NewClient(ClientConfig{
		AppID:      123,
		AppKeyPath: keyPath,
		Owner:      "testowner",
		BaseURL:    server.URL + "/",
	})
	require.NoError(t, err)
	require.NotNil(t, client)
	require.Contains(t, client.BaseURL.String(), server.URL)
}

func TestNewClient_AppKeyContent_InstallationNotFound(t *testing.T) {
	server := appInstallationServer(t, "someoneelse")

	_, err := NewClient(ClientConfig{
		AppID:   123,
		AppKey:  string(testAppKeyPEM(t)),
		Owner:   "testowner",
		BaseURL: server.URL + "/",
	})
	require.Error(t, err)
}

func TestNewClient_AppKeyFile_InstallationNotFound(t *testing.T) {
	server := appInstallationServer(t, "someoneelse")

	keyPath := filepath.Join(t.TempDir(), "app.pem")
	require.NoError(t, os.WriteFile(keyPath, testAppKeyPEM(t), 0o600))

	_, err := NewClient(ClientConfig{
		AppID:      123,
		AppKeyPath: keyPath,
		Owner:      "testowner",
		BaseURL:    server.URL + "/",
	})
	require.Error(t, err)
}

// appServerWithInstallationToken serves installation discovery, the
// access-token mint, and one repository read, so a test can drive a request
// all the way through the returned installation client.
func appServerWithInstallationToken(t *testing.T, owner string) (*httptest.Server, *int) {
	t.Helper()
	repoHits := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v3/app/installations", func(w http.ResponseWriter, _ *http.Request) {
		require.NoError(t, json.NewEncoder(w).Encode([]map[string]any{
			{"id": int64(4242), "account": map[string]any{"login": owner}},
		}))
	})
	mintToken := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"token":      "v1.installation-token",
			"expires_at": time.Now().Add(time.Hour).Format(time.RFC3339),
		}))
	}
	// The installation transport mints tokens from its own BaseURL, which has
	// no /api/v3 prefix; register both spellings.
	mux.HandleFunc("/app/installations/4242/access_tokens", mintToken)
	mux.HandleFunc("/api/v3/app/installations/4242/access_tokens", mintToken)
	mux.HandleFunc("/api/v3/repos/testowner/testrepo", func(w http.ResponseWriter, _ *http.Request) {
		repoHits++
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"name": "testrepo"}))
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server, &repoHits
}

// TestNewClient_AppKeyContent_InstallationTransportUsesBaseURL drives a real
// request through the returned client. The installation transport must mint its
// token from the configured base URL, not from github.com.
func TestNewClient_AppKeyContent_InstallationTransportUsesBaseURL(t *testing.T) {
	server, repoHits := appServerWithInstallationToken(t, "testowner")

	client, err := NewClient(ClientConfig{
		AppID:   123,
		AppKey:  string(testAppKeyPEM(t)),
		Owner:   "testowner",
		BaseURL: server.URL + "/",
	})
	require.NoError(t, err)

	repo, _, err := client.Repositories.Get(context.Background(), "testowner", "testrepo")
	require.NoError(t, err)
	require.Equal(t, "testrepo", repo.GetName())
	require.Equal(t, 1, *repoHits)
}

// TestNewClient_AppKeyFile_InstallationTransportUsesBaseURL is the same check
// for the key-file constructor.
func TestNewClient_AppKeyFile_InstallationTransportUsesBaseURL(t *testing.T) {
	server, repoHits := appServerWithInstallationToken(t, "testowner")

	keyPath := filepath.Join(t.TempDir(), "app.pem")
	require.NoError(t, os.WriteFile(keyPath, testAppKeyPEM(t), 0o600))

	client, err := NewClient(ClientConfig{
		AppID:      123,
		AppKeyPath: keyPath,
		Owner:      "testowner",
		BaseURL:    server.URL + "/",
	})
	require.NoError(t, err)

	repo, _, err := client.Repositories.Get(context.Background(), "testowner", "testrepo")
	require.NoError(t, err)
	require.Equal(t, "testrepo", repo.GetName())
	require.Equal(t, 1, *repoHits)
}
