package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/MyCarrier-DevOps/go-gitsemver/internal/calculator"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/strategy"
)

// strategyOrder defines the display order for strategies.
var strategyOrder = []string{
	"ConfigNextVersion",
	"TaggedCommit",
	"MergeMessage",
	"VersionInBranchName",
	"TrackReleaseBranches",
	"Fallback",
}

// WriteExplanation writes a structured explain output for the version
// calculation to w. It shows all strategy candidates, the selected winner,
// increment reasoning, pre-release tag resolution, and the final result.
func WriteExplanation(w io.Writer, result calculator.VersionResult) error {
	// Group candidates by strategy name.
	byStrategy := make(map[string][]strategy.BaseVersion)
	for _, c := range result.AllCandidates {
		name := ""
		if c.Explanation != nil {
			name = c.Explanation.Strategy
		}
		byStrategy[name] = append(byStrategy[name], c)
	}

	// --- Strategies evaluated ---
	fmt.Fprintln(w, "Strategies evaluated:")

	for _, name := range strategyOrder {
		candidates, ok := byStrategy[name]
		if !ok || len(candidates) == 0 {
			fmt.Fprintf(w, "  %-22s (none)\n", name+":")
			continue
		}
		for i, c := range candidates {
			label := name + ":"
			if i > 0 {
				label = ""
			}
			source := "external"
			if c.BaseVersionSource != nil {
				source = c.BaseVersionSource.ShortSha()
			}
			fmt.Fprintf(w, "  %-22s %s (source: %s, increment: %t)\n",
				label, c.SemanticVersion.SemVer(), source, c.ShouldIncrement)

			// Print explanation steps indented.
			if c.Explanation != nil {
				for _, step := range c.Explanation.Steps {
					fmt.Fprintf(w, "    %s %s\n", arrowPrefix, step)
				}
			}
		}
	}

	// --- Selected ---
	fmt.Fprintln(w)
	source := "external"
	if result.BaseVersion.BaseVersionSource != nil {
		source = result.BaseVersion.BaseVersionSource.ShortSha()
	}
	fmt.Fprintf(w, "Selected: %s (%s, source: %s)\n",
		result.BaseVersion.Source,
		result.BaseVersion.SemanticVersion.SemVer(),
		source,
	)

	// --- Increment ---
	if result.IncrementExplanation != nil && len(result.IncrementExplanation.Steps) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Increment:")
		for _, step := range result.IncrementExplanation.Steps {
			fmt.Fprintf(w, "  %s %s\n", arrowPrefix, step)
		}
	}

	// --- Pre-release ---
	if len(result.PreReleaseSteps) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Pre-release:")
		for _, step := range result.PreReleaseSteps {
			fmt.Fprintf(w, "  %s %s\n", arrowPrefix, step)
		}
	}

	// --- Result ---
	fmt.Fprintln(w)
	fmt.Fprintf(w, "Result: %s\n", result.Version.FullSemVer())

	return nil
}

const arrowPrefix = "\u2192"

// FormatExplanation returns the explain output as a string.
func FormatExplanation(result calculator.VersionResult) string {
	var sb strings.Builder
	_ = WriteExplanation(&sb, result)
	return sb.String()
}
