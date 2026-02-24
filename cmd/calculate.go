package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/MyCarrier-DevOps/go-gitsemver/internal/calculator"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/config"
	configctx "github.com/MyCarrier-DevOps/go-gitsemver/internal/context"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/git"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/output"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/strategy"

	"github.com/spf13/cobra"
)

// configFileNames lists the files searched for configuration in order.
// Checks .github/ first, then repo root directory.
var configFileNames = []string{
	".github/GitVersion.yml",
	".github/go-gitsemver.yml",
	"GitVersion.yml",
	"go-gitsemver.yml",
}

func calculateRunE(_ *cobra.Command, _ []string) error {
	// 1. Open repository.
	repo, err := git.Open(flagPath)
	if err != nil {
		return fmt.Errorf("opening repository: %w", err)
	}

	// 2. Load configuration.
	cfg, err := loadConfig(repo.WorkingDirectory())
	if err != nil {
		return fmt.Errorf("loading configuration: %w", err)
	}

	// 3. Show config mode — print and exit.
	if flagShowConfig {
		return showConfig(cfg)
	}

	// 4. Build context.
	store := git.NewRepositoryStore(repo)
	ctx, err := configctx.NewContext(store, repo, cfg, configctx.Options{
		TargetBranch: flagBranch,
		CommitID:     flagCommit,
	})
	if err != nil {
		return fmt.Errorf("building context: %w", err)
	}

	// 5. Resolve effective configuration for current branch.
	ec, err := ctx.GetEffectiveConfiguration(ctx.CurrentBranch.FriendlyName())
	if err != nil {
		return fmt.Errorf("resolving branch configuration: %w", err)
	}

	// 6. Calculate version.
	strategies := strategy.AllStrategies(store)
	calc := calculator.NewNextVersionCalculator(store, strategies)
	result, err := calc.Calculate(ctx, ec, flagExplain)
	if err != nil {
		return fmt.Errorf("calculating version: %w", err)
	}

	// 7. Write explain output to stderr if requested.
	if flagExplain {
		if err := output.WriteExplanation(os.Stderr, result); err != nil {
			return fmt.Errorf("writing explanation: %w", err)
		}
	}

	// 8. Compute output variables.
	vars := output.GetVariables(result.Version, ec)

	// 9. Write output.
	return writeOutput(vars)
}

// loadConfig loads configuration from a file or defaults.
func loadConfig(workDir string) (*config.Config, error) {
	builder := config.NewBuilder()

	configPath := flagConfig
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

// findConfigFile searches for a config file in the working directory.
func findConfigFile(dir string) string {
	for _, name := range configFileNames {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// showConfig prints the effective configuration as JSON.
func showConfig(cfg *config.Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

// writeOutput writes the version variables in the requested format.
func writeOutput(vars map[string]string) error {
	w := os.Stdout

	if flagShowVariable != "" {
		return output.WriteVariable(w, vars, flagShowVariable)
	}

	switch flagOutput {
	case "json":
		return output.WriteJSON(w, vars)
	case "":
		return output.WriteAll(w, vars)
	default:
		return fmt.Errorf("unknown output format %q", flagOutput)
	}
}
