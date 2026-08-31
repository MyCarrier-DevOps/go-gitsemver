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
				SemanticVersion: semver.SemanticVersion{Major: 1, Minor: 0, Patch: 0},
				Explanation: &strategy.Explanation{
					Strategy: "Fallback",
					Steps:    []string{"using default base version 1.0.0"},
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
	require.Contains(t, out, "1.0.0")
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
			SemanticVersion: semver.SemanticVersion{Major: 1, Minor: 0, Patch: 0},
			Explanation: &strategy.Explanation{
				Strategy: "Fallback",
				Steps:    []string{"using default base version 1.0.0"},
			},
		},
		AllCandidates: []strategy.BaseVersion{
			{
				Source:          "Fallback",
				ShouldIncrement: true,
				SemanticVersion: semver.SemanticVersion{Major: 1, Minor: 0, Patch: 0},
				Explanation: &strategy.Explanation{
					Strategy: "Fallback",
					Steps:    []string{"using default base version 1.0.0"},
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
		Version: semver.SemanticVersion{Major: 1, Minor: 0, Patch: 0},
		BaseVersion: strategy.BaseVersion{
			Source:          "Fallback",
			SemanticVersion: semver.SemanticVersion{Major: 1, Minor: 0, Patch: 0},
		},
	}

	out := FormatExplanation(result)
	require.Contains(t, out, "Result: 1.0.0")
}

func TestWriteExplanation_MultipleCandidatesPerStrategy(t *testing.T) {
	source1 := makeCommit("aaa1234567890abcdef1234567890abcdef123456", "v1.0.0")
	source2 := makeCommit("bbb1234567890abcdef1234567890abcdef123456", "v0.9.0")
	result := calculator.VersionResult{
		Version: semver.SemanticVersion{Major: 1, Minor: 1, Patch: 0},
		BaseVersion: strategy.BaseVersion{
			Source:            "TaggedCommit",
			SemanticVersion:   semver.SemanticVersion{Major: 1, Minor: 0, Patch: 0},
			BaseVersionSource: source1,
			ShouldIncrement:   true,
			Explanation: &strategy.Explanation{
				Strategy: "TaggedCommit",
				Steps:    []string{"winner"},
			},
		},
		AllCandidates: []strategy.BaseVersion{
			{
				SemanticVersion:   semver.SemanticVersion{Major: 1, Minor: 0, Patch: 0},
				BaseVersionSource: source1,
				ShouldIncrement:   true,
				Explanation: &strategy.Explanation{
					Strategy: "TaggedCommit",
					Steps:    []string{"first tag"},
				},
			},
			{
				SemanticVersion:   semver.SemanticVersion{Major: 0, Minor: 9, Patch: 0},
				BaseVersionSource: source2,
				ShouldIncrement:   true,
				Explanation: &strategy.Explanation{
					Strategy: "TaggedCommit",
					Steps:    []string{"second tag"},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := WriteExplanation(&buf, result)
	require.NoError(t, err)

	out := buf.String()
	require.Contains(t, out, "1.0.0")
	require.Contains(t, out, "0.9.0")
	require.Contains(t, out, "first tag")
	require.Contains(t, out, "second tag")
}

func TestWriteExplanation_NilExplanation(t *testing.T) {
	result := calculator.VersionResult{
		Version: semver.SemanticVersion{Major: 1, Minor: 0, Patch: 0},
		BaseVersion: strategy.BaseVersion{
			Source:          "Fallback",
			SemanticVersion: semver.SemanticVersion{Major: 1, Minor: 0, Patch: 0},
		},
		AllCandidates: []strategy.BaseVersion{
			{
				SemanticVersion: semver.SemanticVersion{Major: 1, Minor: 0, Patch: 0},
				// Explanation is nil.
			},
		},
	}

	var buf bytes.Buffer
	err := WriteExplanation(&buf, result)
	require.NoError(t, err)
	require.Contains(t, buf.String(), "Result: 1.0.0")
}

func TestWriteExplanation_ExternalBaseVersionSource(t *testing.T) {
	result := calculator.VersionResult{
		Version: semver.SemanticVersion{Major: 1, Minor: 0, Patch: 0},
		BaseVersion: strategy.BaseVersion{
			Source:          "ConfigNextVersion",
			SemanticVersion: semver.SemanticVersion{Major: 1, Minor: 0, Patch: 0},
			// BaseVersionSource is nil → shows "external".
		},
		AllCandidates: []strategy.BaseVersion{
			{
				SemanticVersion: semver.SemanticVersion{Major: 1, Minor: 0, Patch: 0},
				Explanation: &strategy.Explanation{
					Strategy: "ConfigNextVersion",
					Steps:    []string{"from config"},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := WriteExplanation(&buf, result)
	require.NoError(t, err)

	out := buf.String()
	require.Contains(t, out, "external")
	require.Contains(t, out, "Selected: ConfigNextVersion")
}

func TestWriteExplanation_ErrorWriter(t *testing.T) {
	result := calculator.VersionResult{
		Version: semver.SemanticVersion{Major: 1, Minor: 0, Patch: 0},
		BaseVersion: strategy.BaseVersion{
			Source:          "Fallback",
			SemanticVersion: semver.SemanticVersion{Major: 1, Minor: 0, Patch: 0},
		},
	}

	err := WriteExplanation(&errWriter{n: 0}, result)
	require.Error(t, err)
}

// TestWriteExplanation_ErrorAtEveryWrite drives a failure at each successive
// write in a fully-populated explanation, covering every error-return branch in
// WriteExplanation rather than only the first one.
func TestWriteExplanation_ErrorAtEveryWrite(t *testing.T) {
	src := makeCommit("abc1234567890000000000000000000000000000", "feat: thing")
	result := calculator.VersionResult{
		Version: semver.SemanticVersion{Major: 1, Minor: 3, Patch: 0},
		BaseVersion: strategy.BaseVersion{
			Source:            "TaggedCommit",
			SemanticVersion:   semver.SemanticVersion{Major: 1, Minor: 2, Patch: 0},
			BaseVersionSource: src,
			ShouldIncrement:   true,
		},
		AllCandidates: []strategy.BaseVersion{
			{
				Source:            "TaggedCommit",
				SemanticVersion:   semver.SemanticVersion{Major: 1, Minor: 2, Patch: 0},
				BaseVersionSource: src,
				Explanation:       &strategy.Explanation{Strategy: "TaggedCommit", Steps: []string{"found tag v1.2.0"}},
			},
			{
				Source:          "TaggedCommit",
				SemanticVersion: semver.SemanticVersion{Major: 1, Minor: 1, Patch: 0},
				Explanation:     &strategy.Explanation{Strategy: "TaggedCommit", Steps: []string{"older tag"}},
			},
			{
				Source:          "Fallback",
				SemanticVersion: semver.SemanticVersion{Major: 1},
				Explanation:     &strategy.Explanation{Strategy: "Fallback", Steps: []string{"default base"}},
			},
		},
		IncrementExplanation: &calculator.IncrementExplanation{Steps: []string{"scanned 2 commits", "highest increment: Minor"}},
		PreReleaseSteps:      []string{"branch tag: alpha", "weight applied"},
	}

	// Establish how many writes a full run performs.
	var counter countingWriter
	require.NoError(t, WriteExplanation(&counter, result))
	require.Positive(t, counter.writes)

	for n := range counter.writes {
		err := WriteExplanation(&errWriter{n: n}, result)
		require.Errorf(t, err, "expected failure when the writer fails after %d writes", n)
	}
}

// countingWriter records how many Write calls it received.
type countingWriter struct{ writes int }

func (c *countingWriter) Write(p []byte) (int, error) {
	c.writes++
	return len(p), nil
}
