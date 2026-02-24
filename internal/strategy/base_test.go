package strategy

import (
	"testing"

	"github.com/MyCarrier-DevOps/go-gitsemver/internal/git"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/semver"

	"github.com/stretchr/testify/require"
)

func TestBaseVersion_String(t *testing.T) {
	commit := git.Commit{Sha: "abc123def456789012345678901234567890abcd"}
	bv := BaseVersion{
		Source:            "Git tag 'v1.0.0'",
		ShouldIncrement:   true,
		SemanticVersion:   semver.SemanticVersion{Major: 1},
		BaseVersionSource: &commit,
	}
	s := bv.String()
	require.Contains(t, s, "Git tag 'v1.0.0'")
	require.Contains(t, s, "1.0.0")
	require.Contains(t, s, "abc123d")
	require.Contains(t, s, "increment: true")
}

func TestBaseVersion_String_NilSource(t *testing.T) {
	bv := BaseVersion{
		Source:          "NextVersion in configuration file",
		ShouldIncrement: false,
		SemanticVersion: semver.SemanticVersion{Major: 2},
	}
	s := bv.String()
	require.Contains(t, s, "external")
	require.Contains(t, s, "increment: false")
}

func TestExplanation_NilSafe(t *testing.T) {
	var e *Explanation
	// These should not panic.
	e.Add("step")
	e.Addf("step %d", 1)
	require.Nil(t, e)
}

func TestExplanation_RecordsSteps(t *testing.T) {
	e := NewExplanation("TestStrategy")
	require.Equal(t, "TestStrategy", e.Strategy)
	require.Empty(t, e.Steps)

	e.Add("first step")
	e.Addf("step %d: %s", 2, "second")

	require.Len(t, e.Steps, 2)
	require.Equal(t, "first step", e.Steps[0])
	require.Equal(t, "step 2: second", e.Steps[1])
}
