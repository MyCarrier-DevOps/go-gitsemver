package semver

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestUnmarshalVersioningMode(t *testing.T) {
	tests := []struct {
		input    string
		expected VersioningMode
		wantErr  bool
	}{
		{"ContinuousDelivery", VersioningModeContinuousDelivery, false},
		{"ContinuousDeployment", VersioningModeContinuousDeployment, false},
		{"Mainline", VersioningModeMainline, false},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var m VersioningMode
			err := yaml.Unmarshal([]byte(tt.input), &m)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, m)
			}
		})
	}
}

func TestUnmarshalIncrementStrategy(t *testing.T) {
	tests := []struct {
		input    string
		expected IncrementStrategy
		wantErr  bool
	}{
		{"None", IncrementStrategyNone, false},
		{"Major", IncrementStrategyMajor, false},
		{"Minor", IncrementStrategyMinor, false},
		{"Patch", IncrementStrategyPatch, false},
		{"Inherit", IncrementStrategyInherit, false},
		{"bogus", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var s IncrementStrategy
			err := yaml.Unmarshal([]byte(tt.input), &s)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, s)
			}
		})
	}
}

func TestUnmarshalCommitMessageIncrementMode(t *testing.T) {
	tests := []struct {
		input    string
		expected CommitMessageIncrementMode
		wantErr  bool
	}{
		{"Enabled", CommitMessageIncrementEnabled, false},
		{"Disabled", CommitMessageIncrementDisabled, false},
		{"MergeMessageOnly", CommitMessageIncrementMergeMessageOnly, false},
		{"nope", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var m CommitMessageIncrementMode
			err := yaml.Unmarshal([]byte(tt.input), &m)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, m)
			}
		})
	}
}

func TestUnmarshalMainlineIncrementMode(t *testing.T) {
	tests := []struct {
		input    string
		expected MainlineIncrementMode
		wantErr  bool
	}{
		{"Aggregate", MainlineIncrementAggregate, false},
		{"EachCommit", MainlineIncrementEachCommit, false},
		{"each-commit", MainlineIncrementEachCommit, false},
		{"wrong", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var m MainlineIncrementMode
			err := yaml.Unmarshal([]byte(tt.input), &m)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, m)
			}
		})
	}
}

func TestUnmarshalCommitMessageConvention(t *testing.T) {
	tests := []struct {
		input    string
		expected CommitMessageConvention
		wantErr  bool
	}{
		{"ConventionalCommits", CommitMessageConventionConventionalCommits, false},
		{"conventional-commits", CommitMessageConventionConventionalCommits, false},
		{"BumpDirective", CommitMessageConventionBumpDirective, false},
		{"bump-directive", CommitMessageConventionBumpDirective, false},
		{"Both", CommitMessageConventionBoth, false},
		{"unknown", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var c CommitMessageConvention
			err := yaml.Unmarshal([]byte(tt.input), &c)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, c)
			}
		})
	}
}

// Test that YAML unmarshaling works in struct context (like config loading).
func TestUnmarshalInStruct(t *testing.T) {
	type config struct {
		Mode       VersioningMode             `yaml:"mode"`
		Increment  IncrementStrategy          `yaml:"increment"`
		CommitMsg  CommitMessageIncrementMode `yaml:"commit-msg"`
		Mainline   MainlineIncrementMode      `yaml:"mainline"`
		Convention CommitMessageConvention    `yaml:"convention"`
	}

	input := `
mode: ContinuousDeployment
increment: Minor
commit-msg: Enabled
mainline: Aggregate
convention: Both
`
	var cfg config
	err := yaml.Unmarshal([]byte(input), &cfg)
	require.NoError(t, err)
	require.Equal(t, VersioningModeContinuousDeployment, cfg.Mode)
	require.Equal(t, IncrementStrategyMinor, cfg.Increment)
	require.Equal(t, CommitMessageIncrementEnabled, cfg.CommitMsg)
	require.Equal(t, MainlineIncrementAggregate, cfg.Mainline)
	require.Equal(t, CommitMessageConventionBoth, cfg.Convention)
}
