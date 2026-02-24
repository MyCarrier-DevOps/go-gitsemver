// Package gitsemver provides a public Go API for calculating semantic versions
// from git history. It supports both local repositories (via go-git) and remote
// GitHub repositories (via the GitHub API).
//
// Basic usage:
//
//	result, err := gitsemver.Calculate(gitsemver.LocalOptions{
//	    Path: "/path/to/repo",
//	})
//	fmt.Println(result.Variables["SemVer"]) // "1.2.3"
//
//	result, err := gitsemver.CalculateRemote(gitsemver.RemoteOptions{
//	    Owner: "myorg",
//	    Repo:  "myrepo",
//	    Token: os.Getenv("GITHUB_TOKEN"),
//	})
//	fmt.Println(result.Variables["FullSemVer"]) // "1.2.3+5"
package gitsemver

import (
	"errors"
	"fmt"
	"go-gitsemver/internal/calculator"
	"go-gitsemver/internal/config"
	"go-gitsemver/internal/git"
	"go-gitsemver/internal/output"
	"go-gitsemver/internal/strategy"
	"os"
	"path/filepath"

	configctx "go-gitsemver/internal/context"

	ghprovider "go-gitsemver/internal/github"
)

// LocalOptions configures version calculation from a local git repository.
type LocalOptions struct {
	// Path to the git repository. Defaults to "." if empty.
	Path string

	// Branch overrides the target branch. Empty means use HEAD.
	Branch string

	// Commit overrides the branch tip with a specific SHA. Empty means use tip.
	Commit string

	// ConfigPath is the path to a gitsemver/GitVersion YAML config file.
	// If empty, auto-detects GitVersion.yml or gitsemver.yml in the repo root.
	ConfigPath string
}

// RemoteOptions configures version calculation via the GitHub API.
type RemoteOptions struct {
	// Owner is the GitHub repository owner (required).
	Owner string

	// Repo is the GitHub repository name (required).
	Repo string

	// Token is a GitHub personal access token or GITHUB_TOKEN.
	Token string

	// AppID is the GitHub App ID for app authentication.
	AppID int64

	// AppKeyPath is the path to a GitHub App private key PEM file.
	AppKeyPath string

	// BaseURL is a custom GitHub API base URL for GitHub Enterprise.
	BaseURL string

	// Ref is the git ref to version: branch, tag, or SHA. Defaults to the
	// repository's default branch.
	Ref string

	// MaxCommits is the hard cap on commit walk depth. Defaults to 1000.
	MaxCommits int

	// Branch overrides the target branch for context resolution.
	Branch string

	// Commit overrides the branch tip with a specific SHA.
	Commit string

	// ConfigPath is a local config file path that overrides remote config.
	ConfigPath string
}

// Result holds the calculated version and all output variables.
type Result struct {
	// Variables contains all 30+ output variables keyed by name.
	// Common keys: SemVer, FullSemVer, MajorMinorPatch, Major, Minor, Patch,
	// PreReleaseTag, PreReleaseNumber, CommitsSinceVersionSource, Sha, ShortSha,
	// BranchName, etc.
	Variables map[string]string
}

// configFileNames lists the files searched for configuration in order.
var configFileNames = []string{
	"GitVersion.yml",
	"gitsemver.yml",
}

// Calculate computes the next semantic version from a local git repository.
func Calculate(opts LocalOptions) (*Result, error) {
	path := opts.Path
	if path == "" {
		path = "."
	}

	// 1. Open repository.
	repo, err := git.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening repository: %w", err)
	}

	// 2. Load configuration.
	cfg, err := loadLocalConfig(opts.ConfigPath, repo.WorkingDirectory())
	if err != nil {
		return nil, fmt.Errorf("loading configuration: %w", err)
	}

	// 3. Run the shared calculation pipeline.
	return calculate(repo, cfg, opts.Branch, opts.Commit)
}

// CalculateRemote computes the next semantic version via the GitHub API.
func CalculateRemote(opts RemoteOptions) (*Result, error) {
	if opts.Owner == "" || opts.Repo == "" {
		return nil, errors.New("owner and repo are required")
	}

	// 1. Create GitHub client.
	client, err := ghprovider.NewClient(ghprovider.ClientConfig{
		Token:      opts.Token,
		AppID:      opts.AppID,
		AppKeyPath: opts.AppKeyPath,
		BaseURL:    opts.BaseURL,
		Owner:      opts.Owner,
	})
	if err != nil {
		return nil, fmt.Errorf("creating GitHub client: %w", err)
	}

	// 2. Create GitHubRepository.
	var ghOpts []ghprovider.Option
	if opts.Ref != "" {
		ghOpts = append(ghOpts, ghprovider.WithRef(opts.Ref))
	}
	maxCommits := opts.MaxCommits
	if maxCommits <= 0 {
		maxCommits = 1000
	}
	ghOpts = append(ghOpts, ghprovider.WithMaxCommits(maxCommits))
	if opts.BaseURL != "" {
		ghOpts = append(ghOpts, ghprovider.WithBaseURL(opts.BaseURL))
	}
	ghRepo := ghprovider.NewGitHubRepository(client, opts.Owner, opts.Repo, ghOpts...)

	// 3. Load configuration.
	cfg, err := loadRemoteConfig(opts.ConfigPath, ghRepo)
	if err != nil {
		return nil, fmt.Errorf("loading configuration: %w", err)
	}

	// 4. Run the shared calculation pipeline.
	return calculate(ghRepo, cfg, opts.Branch, opts.Commit)
}

// calculate runs the shared version calculation pipeline.
func calculate(repo git.Repository, cfg *config.Config, branch, commit string) (*Result, error) {
	store := git.NewRepositoryStore(repo)

	ctx, err := configctx.NewContext(store, repo, cfg, configctx.Options{
		TargetBranch: branch,
		CommitID:     commit,
	})
	if err != nil {
		return nil, fmt.Errorf("building context: %w", err)
	}

	ec, err := ctx.GetEffectiveConfiguration(ctx.CurrentBranch.FriendlyName())
	if err != nil {
		return nil, fmt.Errorf("resolving branch configuration: %w", err)
	}

	strategies := strategy.AllStrategies(store)
	calc := calculator.NewNextVersionCalculator(store, strategies)
	result, err := calc.Calculate(ctx, ec, false)
	if err != nil {
		return nil, fmt.Errorf("calculating version: %w", err)
	}

	vars := output.GetVariables(result.Version, ec)

	return &Result{Variables: vars}, nil
}

// loadLocalConfig loads configuration from a file path or auto-detects it.
func loadLocalConfig(configPath, workDir string) (*config.Config, error) {
	builder := config.NewBuilder()

	if configPath == "" {
		configPath = findConfigFile(workDir)
	}

	if configPath != "" {
		userCfg, err := config.LoadFromFile(configPath)
		if err != nil {
			return nil, err
		}
		builder.Add(userCfg)
	}

	return builder.Build()
}

// findConfigFile searches for a config file in the given directory.
func findConfigFile(dir string) string {
	for _, name := range configFileNames {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// loadRemoteConfig loads configuration from a local override or the remote repo.
func loadRemoteConfig(configPath string, ghRepo *ghprovider.GitHubRepository) (*config.Config, error) {
	builder := config.NewBuilder()

	if configPath != "" {
		userCfg, err := config.LoadFromFile(configPath)
		if err != nil {
			return nil, err
		}
		builder.Add(userCfg)
	} else {
		for _, name := range configFileNames {
			content, err := ghRepo.FetchFileContent(name)
			if err != nil {
				if ghprovider.IsNotFoundError(err) {
					continue
				}
				return nil, fmt.Errorf("fetching remote config %s: %w", name, err)
			}
			userCfg, err := config.LoadFromBytes([]byte(content))
			if err != nil {
				return nil, fmt.Errorf("parsing remote config %s: %w", name, err)
			}
			builder.Add(userCfg)
			break
		}
	}

	return builder.Build()
}
