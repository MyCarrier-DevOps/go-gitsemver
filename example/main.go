// Example program demonstrating the gitsemver library API.
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
	"go-gitsemver/pkg/gitsemver"
	"log"
	"os"
	"sort"
)

func main() {
	localVersion()

	if os.Getenv("GITHUB_TOKEN") != "" {
		remoteVersion()
	}
}

func localVersion() {
	result, err := gitsemver.Calculate(gitsemver.LocalOptions{
		Path: ".",
	})
	if err != nil {
		log.Fatalf("local calculation failed: %v", err)
	}

	printVersion("Local", result)
}

func remoteVersion() {
	result, err := gitsemver.CalculateRemote(gitsemver.RemoteOptions{
		Owner: "MyCarrier-DevOps",
		Repo:  "go-gitsemver",
		Token: os.Getenv("GITHUB_TOKEN"),
		Ref:   "main",
	})
	if err != nil {
		log.Fatalf("remote calculation failed: %v", err)
	}

	printVersion("Remote", result)
}

func printVersion(label string, result *gitsemver.Result) {
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
