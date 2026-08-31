// Example program demonstrating the sdk library API.
//
// Run from the repo root:
//
//	go run ./example/
//
// With remote mode (set GITHUB_TOKEN first):
//
//	GITHUB_TOKEN=ghp_xxx go run ./example/
package main

import (
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/MyCarrier-DevOps/go-gitsemver/pkg/sdk"
)

// main keeps the process-exiting behaviour a demo wants; run holds the logic so
// it stays callable - and testable - without terminating the caller.
func main() {
	if err := run(os.Getenv("GITHUB_TOKEN")); err != nil {
		log.Fatal(err)
	}
}

// run takes the token rather than reading the environment itself, so the
// remote-example guard can be exercised without a live GitHub call.
func run(githubToken string) error {
	if err := localVersion(); err != nil {
		return err
	}
	if err := localVersionExplain(); err != nil {
		return err
	}

	if githubToken != "" {
		return remoteVersion(githubToken)
	}
	return nil
}

func localVersion() error {
	result, err := sdk.Calculate(sdk.LocalOptions{
		Path: ".",
	})
	if err != nil {
		return fmt.Errorf("local calculation failed: %w", err)
	}

	printVersion("Local", result)
	return nil
}

func localVersionExplain() error {
	result, err := sdk.Calculate(sdk.LocalOptions{
		Path:    ".",
		Explain: true,
	})
	if err != nil {
		return fmt.Errorf("local explain calculation failed: %w", err)
	}

	fmt.Println("=== Explain Output ===")
	if result.ExplainResult != nil {
		fmt.Println(result.ExplainResult.FormattedOutput)
		fmt.Printf("Final version: %s\n", result.ExplainResult.FinalVersion)
		fmt.Printf("Selected source: %s\n", result.ExplainResult.SelectedSource)
		fmt.Printf("Candidates: %d\n", len(result.ExplainResult.Candidates))
	}
	fmt.Println()
	return nil
}

func remoteVersion(githubToken string) error {
	result, err := sdk.CalculateRemote(sdk.RemoteOptions{
		Owner: "MyCarrier-DevOps",
		Repo:  "go-gitsemver",
		Token: githubToken,
		Ref:   "main",
	})
	if err != nil {
		return fmt.Errorf("remote calculation failed: %w", err)
	}

	printVersion("Remote", result)
	return nil
}

func printVersion(label string, result *sdk.Result) {
	fmt.Printf("=== %s Version ===\n", label)

	keys := make([]string, 0, len(result.Variables))
	for k := range result.Variables {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		fmt.Printf("%-40s %s\n", k, result.Variables[k])
	}
	fmt.Println()
}
