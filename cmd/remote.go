package cmd

import (
	"fmt"
	"go-gitsemver/internal/calculator"
	"go-gitsemver/internal/config"
	"go-gitsemver/internal/git"
	"go-gitsemver/internal/output"
	"go-gitsemver/internal/strategy"
	"os"
	"strings"

	configctx "go-gitsemver/internal/context"

	ghprovider "go-gitsemver/internal/github"

	"github.com/spf13/cobra"
)

var (
	flagToken      string
	flagAppID      int64
	flagAppKeyPath string
	flagGitHubURL  string
	flagRef        string
	flagMaxCommits int
)

var remoteCmd = &cobra.Command{
	Use:   "remote owner/repo",
	Short: "Calculate version from a GitHub repository via API",
	Long: `Calculate the next semantic version by reading git history from the
GitHub API. No local clone is required.

Authentication (checked in order):
  1. --token flag or GITHUB_TOKEN env var
  2. --github-app-id + --github-app-key flags or GH_APP_ID + GH_APP_PRIVATE_KEY env vars

Examples:
  GITHUB_TOKEN=ghp_xxx gitsemver remote myorg/myrepo
  gitsemver remote myorg/myrepo --token ghp_xxx --ref main
  gitsemver remote myorg/myrepo --github-app-id 12345 --github-app-key /path/to/key.pem`,
	Args: cobra.ExactArgs(1),
	RunE: remoteRunE,
}

func init() {
	remoteCmd.Flags().StringVar(&flagToken, "token", "", "GitHub token (or set GITHUB_TOKEN env var)")
	remoteCmd.Flags().Int64Var(&flagAppID, "github-app-id", 0, "GitHub App ID (or set GH_APP_ID env var)")
	remoteCmd.Flags().StringVar(&flagAppKeyPath, "github-app-key", "", "path to GitHub App private key PEM file (or set GH_APP_PRIVATE_KEY env var)")
	remoteCmd.Flags().StringVar(&flagGitHubURL, "github-url", "", "GitHub API base URL for GitHub Enterprise (or set GITHUB_API_URL env var)")
	remoteCmd.Flags().StringVar(&flagRef, "ref", "", "git ref to version: branch, tag, or SHA (default: repo default branch)")
	remoteCmd.Flags().IntVar(&flagMaxCommits, "max-commits", 1000, "maximum commit depth to walk via API")

	rootCmd.AddCommand(remoteCmd)
}

func remoteRunE(_ *cobra.Command, args []string) error {
	// 1. Parse owner/repo.
	owner, repo, err := parseOwnerRepo(args[0])
	if err != nil {
		return err
	}

	// 2. Resolve base URL from flag or env var so both client and repository use it.
	baseURL := ghprovider.ResolveBaseURL(flagGitHubURL)

	// 3. Create GitHub client.
	client, err := ghprovider.NewClient(ghprovider.ClientConfig{
		Token:      flagToken,
		AppID:      flagAppID,
		AppKeyPath: flagAppKeyPath,
		BaseURL:    baseURL,
		Owner:      owner,
	})
	if err != nil {
		return fmt.Errorf("creating GitHub client: %w", err)
	}

	// 4. Create GitHubRepository.
	var opts []ghprovider.Option
	if flagRef != "" {
		opts = append(opts, ghprovider.WithRef(flagRef))
	}
	if flagMaxCommits > 0 {
		opts = append(opts, ghprovider.WithMaxCommits(flagMaxCommits))
	}
	if baseURL != "" {
		opts = append(opts, ghprovider.WithBaseURL(baseURL))
	}
	ghRepo := ghprovider.NewGitHubRepository(client, owner, repo, opts...)

	// 5. Load configuration.
	cfg, err := loadRemoteConfig(ghRepo)
	if err != nil {
		return fmt.Errorf("loading configuration: %w", err)
	}

	// 6. Show config mode.
	if flagShowConfig {
		return showConfig(cfg)
	}

	// 7. Build context.
	store := git.NewRepositoryStore(ghRepo)
	ctx, err := configctx.NewContext(store, ghRepo, cfg, configctx.Options{
		TargetBranch: flagBranch,
		CommitID:     flagCommit,
	})
	if err != nil {
		return fmt.Errorf("building context: %w", err)
	}

	// 8. Resolve effective configuration for current branch.
	ec, err := ctx.GetEffectiveConfiguration(ctx.CurrentBranch.FriendlyName())
	if err != nil {
		return fmt.Errorf("resolving branch configuration: %w", err)
	}

	// 9. Calculate version.
	strategies := strategy.AllStrategies(store)
	calc := calculator.NewNextVersionCalculator(store, strategies)
	result, err := calc.Calculate(ctx, ec, flagExplain)
	if err != nil {
		return fmt.Errorf("calculating version: %w", err)
	}

	// 10. Write explain output to stderr if requested.
	if flagExplain {
		if err := output.WriteExplanation(os.Stderr, result); err != nil {
			return fmt.Errorf("writing explanation: %w", err)
		}
	}

	// 11. Compute output variables.
	vars := output.GetVariables(result.Version, ec)

	// 12. Write output.
	return writeOutput(vars)
}

func parseOwnerRepo(s string) (string, string, error) {
	parts := strings.SplitN(s, "/", 3)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid repository format %q, expected owner/repo", s)
	}
	return parts[0], parts[1], nil
}

// loadRemoteConfig fetches configuration from the remote repo or uses a local file.
func loadRemoteConfig(ghRepo *ghprovider.GitHubRepository) (*config.Config, error) {
	builder := config.NewBuilder()

	if flagConfig != "" {
		// Use explicit local config file.
		userCfg, err := config.LoadFromFile(flagConfig)
		if err != nil {
			return nil, err
		}
		builder.Add(userCfg)
	} else {
		// Try to fetch config from the remote repo root.
		for _, name := range configFileNames {
			content, err := ghRepo.FetchFileContent(name)
			if err != nil {
				// 404 means the file doesn't exist — try the next name.
				if ghprovider.IsNotFoundError(err) {
					continue
				}
				// Other errors (auth failure, rate limit, network) should not be silently ignored.
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
