package github

import (
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
