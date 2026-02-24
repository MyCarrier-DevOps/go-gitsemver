package output

import (
	"bytes"
	"testing"

	"github.com/MyCarrier-DevOps/go-gitsemver/internal/calculator"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/git"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/semver"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/strategy"

	"github.com/stretchr/testify/require"
)

func makeCommit(sha, msg string) *git.Commit {
	return &git.Commit{Sha: sha, Message: msg}
}

func TestWriteExplanation_BasicTaggedCommit(t *testing.T) {
	source := makeCommit("abc1234567890abcdef1234567890abcdef123456", "v1.0.0")
	result := calculator.VersionResult{
		Version: semver.SemanticVersion{Major: 1, Minor: 1, Patch: 0},
		BaseVersion: strategy.BaseVersion{
			Source:            "TaggedCommit",
			ShouldIncrement:   true,
			SemanticVersion:   semver.SemanticVersion{Major: 1, Minor: 0, Patch: 0},
			BaseVersionSource: source,
			Explanation: &strategy.Explanation{
				Strategy: "TaggedCommit",
				Steps:    []string{"found tag v1.0.0 on commit abc1234"},
			},
		},
		AllCandidates: []strategy.BaseVersion{
			{
				Source:            "TaggedCommit",
				ShouldIncrement:   true,
				SemanticVersion:   semver.SemanticVersion{Major: 1, Minor: 0, Patch: 0},
				BaseVersionSource: source,
				Explanation: &strategy.Explanation{
					Strategy: "TaggedCommit",
					Steps:    []string{"found tag v1.0.0 on commit abc1234"},
				},
			},
			{
				Source:          "Fallback",
				ShouldIncrement: true,
				SemanticVersion: semver.SemanticVersion{Major: 0, Minor: 1, Patch: 0},
				Explanation: &strategy.Explanation{
					Strategy: "Fallback",
					Steps:    []string{"using default base version 0.1.0"},
				},
			},
		},
		IncrementExplanation: &calculator.IncrementExplanation{
			Steps: []string{
				"scanned 1 commits",
				`commit abc1234 "feat: add auth" -> Minor (Conventional Commits)`,
				"highest increment from commits: Minor",
			},
		},
	}

	var buf bytes.Buffer
	err := WriteExplanation(&buf, result)
	require.NoError(t, err)

	out := buf.String()

	// Check sections present.
	require.Contains(t, out, "Strategies evaluated:")
	require.Contains(t, out, "TaggedCommit:")
	require.Contains(t, out, "1.0.0")
	require.Contains(t, out, "Fallback:")
	require.Contains(t, out, "0.1.0")
	require.Contains(t, out, "Selected: TaggedCommit")
	require.Contains(t, out, "Increment:")
	require.Contains(t, out, "highest increment from commits: Minor")
	require.Contains(t, out, "Result: 1.1.0")

	// Strategies with no candidates should show (none).
	require.Contains(t, out, "ConfigNextVersion:")
	require.Contains(t, out, "(none)")
}

func TestWriteExplanation_NoIncrement(t *testing.T) {
	source := makeCommit("abc1234567890abcdef1234567890abcdef123456", "v2.0.0")
	result := calculator.VersionResult{
		Version: semver.SemanticVersion{Major: 2, Minor: 0, Patch: 0},
		BaseVersion: strategy.BaseVersion{
			Source:            "TaggedCommit",
			ShouldIncrement:   false,
			SemanticVersion:   semver.SemanticVersion{Major: 2, Minor: 0, Patch: 0},
			BaseVersionSource: source,
			Explanation: &strategy.Explanation{
				Strategy: "TaggedCommit",
				Steps:    []string{"current commit is tagged v2.0.0"},
			},
		},
		AllCandidates: []strategy.BaseVersion{
			{
				Source:            "TaggedCommit",
				ShouldIncrement:   false,
				SemanticVersion:   semver.SemanticVersion{Major: 2, Minor: 0, Patch: 0},
				BaseVersionSource: source,
				Explanation: &strategy.Explanation{
					Strategy: "TaggedCommit",
					Steps:    []string{"current commit is tagged v2.0.0"},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := WriteExplanation(&buf, result)
	require.NoError(t, err)

	out := buf.String()

	// No increment section when there's no IncrementExplanation.
	require.NotContains(t, out, "Increment:")
	require.Contains(t, out, "Result: 2.0.0")
}

func TestWriteExplanation_PreRelease(t *testing.T) {
	n := int64(1)
	result := calculator.VersionResult{
		Version: semver.SemanticVersion{
			Major: 1, Minor: 1, Patch: 0,
			PreReleaseTag: semver.PreReleaseTag{Name: "feature-login", Number: &n},
		},
		BaseVersion: strategy.BaseVersion{
			Source:          "Fallback",
			ShouldIncrement: true,
			SemanticVersion: semver.SemanticVersion{Major: 0, Minor: 1, Patch: 0},
			Explanation: &strategy.Explanation{
				Strategy: "Fallback",
				Steps:    []string{"using default base version 0.1.0"},
			},
		},
		AllCandidates: []strategy.BaseVersion{
			{
				Source:          "Fallback",
				ShouldIncrement: true,
				SemanticVersion: semver.SemanticVersion{Major: 0, Minor: 1, Patch: 0},
				Explanation: &strategy.Explanation{
					Strategy: "Fallback",
					Steps:    []string{"using default base version 0.1.0"},
				},
			},
		},
		IncrementExplanation: &calculator.IncrementExplanation{
			Steps: []string{"highest increment from commits: Minor"},
		},
		PreReleaseSteps: []string{
			`branch config tag="{BranchName}" -> "feature-login"`,
			"no existing tag for 1.1.0-feature-login -> number = 1",
		},
	}

	var buf bytes.Buffer
	err := WriteExplanation(&buf, result)
	require.NoError(t, err)

	out := buf.String()

	require.Contains(t, out, "Pre-release:")
	require.Contains(t, out, "feature-login")
	require.Contains(t, out, "number = 1")
	require.Contains(t, out, "Result: 1.1.0-feature-login.1")
}

func TestFormatExplanation_ReturnsString(t *testing.T) {
	result := calculator.VersionResult{
		Version: semver.SemanticVersion{Major: 0, Minor: 1, Patch: 0},
		BaseVersion: strategy.BaseVersion{
			Source:          "Fallback",
			SemanticVersion: semver.SemanticVersion{Major: 0, Minor: 1, Patch: 0},
		},
	}

	out := FormatExplanation(result)
	require.Contains(t, out, "Result: 0.1.0")
}
