package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/bradleyfalzon/ghinstallation/v2"
	gh "github.com/google/go-github/v68/github"
	"golang.org/x/oauth2"
)

// ClientConfig holds the configuration for creating a GitHub API client.
type ClientConfig struct {
	// Token is a GitHub personal access token or GITHUB_TOKEN.
	// Falls back to GITHUB_TOKEN env var if empty.
	Token string

	// AppID is the GitHub App ID for app authentication.
	// Falls back to GH_APP_ID env var if zero.
	AppID int64

	// AppKeyPath is the path to a GitHub App private key PEM file.
	// Falls back to GH_APP_PRIVATE_KEY env var if empty.
	AppKeyPath string

	// BaseURL is a custom GitHub API base URL for GitHub Enterprise.
	// Falls back to GITHUB_API_URL env var if empty.
	BaseURL string

	// Owner is the repository owner, used for auto-detecting the app installation.
	Owner string
}

// NewClient creates an authenticated GitHub API client.
// Auth resolution order: Token flag → GITHUB_TOKEN env → App credentials → error.
func NewClient(cfg ClientConfig) (*gh.Client, error) {
	baseURL := resolveString(cfg.BaseURL, "GITHUB_API_URL")

	// Try token auth first.
	token := resolveString(cfg.Token, "GITHUB_TOKEN")
	if token != "" {
		return newTokenClient(token, baseURL)
	}

	// Try GitHub App auth.
	appID := cfg.AppID
	if appID == 0 {
		if s := os.Getenv("GH_APP_ID"); s != "" {
			if v, err := strconv.ParseInt(s, 10, 64); err == nil {
				appID = v
			}
		}
	}
	appKey := resolveString(cfg.AppKeyPath, "GH_APP_PRIVATE_KEY")

	if appID != 0 && appKey != "" {
		return newAppClient(appID, appKey, cfg.Owner, baseURL)
	}

	return nil, errors.New("no GitHub authentication provided: set GITHUB_TOKEN, use --token, or provide --github-app-id and --github-app-key")
}

func newTokenClient(token, baseURL string) (*gh.Client, error) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	httpClient := oauth2.NewClient(context.Background(), ts)

	if baseURL != "" {
		return gh.NewClient(httpClient).WithEnterpriseURLs(baseURL, baseURL)
	}
	return gh.NewClient(httpClient), nil
}

func newAppClient(appID int64, keyPath, owner, baseURL string) (*gh.Client, error) {
	// Create an app-level transport to discover the installation ID.
	appTransport, err := ghinstallation.NewAppsTransportKeyFromFile(http.DefaultTransport, appID, keyPath)
	if err != nil {
		return nil, fmt.Errorf("creating GitHub App transport: %w", err)
	}
	if baseURL != "" {
		appTransport.BaseURL = baseURL
	}

	// Find the installation for the target owner.
	appClient := gh.NewClient(&http.Client{Transport: appTransport})
	if baseURL != "" {
		appClient, err = appClient.WithEnterpriseURLs(baseURL, baseURL)
		if err != nil {
			return nil, fmt.Errorf("setting enterprise URL: %w", err)
		}
	}

	installationID, err := findInstallation(appClient, owner)
	if err != nil {
		return nil, err
	}

	// Create an installation-level transport with the discovered ID.
	installTransport, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, appID, installationID, keyPath)
	if err != nil {
		return nil, fmt.Errorf("creating installation transport: %w", err)
	}
	if baseURL != "" {
		installTransport.BaseURL = baseURL
	}

	client := gh.NewClient(&http.Client{Transport: installTransport})
	if baseURL != "" {
		return client.WithEnterpriseURLs(baseURL, baseURL)
	}
	return client, nil
}

// findInstallation finds the GitHub App installation for the given owner.
func findInstallation(client *gh.Client, owner string) (int64, error) {
	ctx := context.Background()
	opts := &gh.ListOptions{PerPage: 100}

	for {
		installations, resp, err := client.Apps.ListInstallations(ctx, opts)
		if err != nil {
			return 0, fmt.Errorf("listing GitHub App installations: %w", err)
		}

		for _, inst := range installations {
			if inst.GetAccount().GetLogin() == owner {
				return inst.GetID(), nil
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return 0, fmt.Errorf("no GitHub App installation found for owner %q", owner)
}

// IsNotFoundError returns true if the error represents an HTTP 404 response
// from the GitHub API. Used to distinguish "file not found" from auth failures,
// rate limits, and other errors that should not be silently ignored.
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	var ghErr *gh.ErrorResponse
	if errors.As(err, &ghErr) {
		return ghErr.Response != nil && ghErr.Response.StatusCode == 404
	}
	return false
}

// resolveString returns the flag value if non-empty, otherwise the env var value.
func resolveString(flag, envKey string) string {
	if flag != "" {
		return flag
	}
	return os.Getenv(envKey)
}

// ResolveBaseURL resolves the GitHub API base URL from the flag value or
// the GITHUB_API_URL environment variable. Returns empty string for github.com.
func ResolveBaseURL(flagValue string) string {
	return resolveString(flagValue, "GITHUB_API_URL")
}
