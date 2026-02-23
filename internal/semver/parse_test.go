package semver

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestParseVersioningMode(t *testing.T) {
	tests := []struct {
		input string
		want  VersioningMode
	}{
		{"ContinuousDelivery", VersioningModeContinuousDelivery},
		{"continuousdelivery", VersioningModeContinuousDelivery},
		{"ContinuousDeployment", VersioningModeContinuousDeployment},
		{"continuousdeployment", VersioningModeContinuousDeployment},
		{"Mainline", VersioningModeMainline},
		{"mainline", VersioningModeMainline},
		{"MAINLINE", VersioningModeMainline},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseVersioningMode(tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestParseVersioningMode_Invalid(t *testing.T) {
	_, err := ParseVersioningMode("invalid")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown versioning mode")
}

func TestParseIncrementStrategy(t *testing.T) {
	tests := []struct {
		input string
		want  IncrementStrategy
	}{
		{"None", IncrementStrategyNone},
		{"none", IncrementStrategyNone},
		{"Major", IncrementStrategyMajor},
		{"major", IncrementStrategyMajor},
		{"Minor", IncrementStrategyMinor},
		{"minor", IncrementStrategyMinor},
		{"Patch", IncrementStrategyPatch},
		{"patch", IncrementStrategyPatch},
		{"Inherit", IncrementStrategyInherit},
		{"inherit", IncrementStrategyInherit},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseIncrementStrategy(tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestParseIncrementStrategy_Invalid(t *testing.T) {
	_, err := ParseIncrementStrategy("bogus")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown increment strategy")
}

func TestParseCommitMessageIncrementMode(t *testing.T) {
	tests := []struct {
		input string
		want  CommitMessageIncrementMode
	}{
		{"Enabled", CommitMessageIncrementEnabled},
		{"enabled", CommitMessageIncrementEnabled},
		{"Disabled", CommitMessageIncrementDisabled},
		{"disabled", CommitMessageIncrementDisabled},
		{"MergeMessageOnly", CommitMessageIncrementMergeMessageOnly},
		{"mergemessageonly", CommitMessageIncrementMergeMessageOnly},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseCommitMessageIncrementMode(tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestParseCommitMessageIncrementMode_Invalid(t *testing.T) {
	_, err := ParseCommitMessageIncrementMode("nope")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown commit message increment mode")
}

func TestParseCommitMessageConvention(t *testing.T) {
	tests := []struct {
		input string
		want  CommitMessageConvention
	}{
		{"ConventionalCommits", CommitMessageConventionConventionalCommits},
		{"conventional-commits", CommitMessageConventionConventionalCommits},
		{"conventionalcommits", CommitMessageConventionConventionalCommits},
		{"BumpDirective", CommitMessageConventionBumpDirective},
		{"bump-directive", CommitMessageConventionBumpDirective},
		{"bumpdirective", CommitMessageConventionBumpDirective},
		{"Both", CommitMessageConventionBoth},
		{"both", CommitMessageConventionBoth},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseCommitMessageConvention(tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestParseCommitMessageConvention_Invalid(t *testing.T) {
	_, err := ParseCommitMessageConvention("xyz")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown commit message convention")
}

func TestParseMainlineIncrementMode(t *testing.T) {
	tests := []struct {
		input string
		want  MainlineIncrementMode
	}{
		{"Aggregate", MainlineIncrementAggregate},
		{"aggregate", MainlineIncrementAggregate},
		{"EachCommit", MainlineIncrementEachCommit},
		{"eachcommit", MainlineIncrementEachCommit},
		{"each-commit", MainlineIncrementEachCommit},
		{"AGGREGATE", MainlineIncrementAggregate},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseMainlineIncrementMode(tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestParseMainlineIncrementMode_Invalid(t *testing.T) {
	_, err := ParseMainlineIncrementMode("percommit")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown mainline increment mode")
}

func TestMainlineIncrementMode_String(t *testing.T) {
	require.Equal(t, "Aggregate", MainlineIncrementAggregate.String())
	require.Equal(t, "EachCommit", MainlineIncrementEachCommit.String())
	require.Equal(t, "Unknown", MainlineIncrementMode(99).String())
}

func TestVersioningMode_UnmarshalYAML(t *testing.T) {
	var m VersioningMode
	require.NoError(t, yaml.Unmarshal([]byte(`Mainline`), &m))
	require.Equal(t, VersioningModeMainline, m)
}

func TestIncrementStrategy_UnmarshalYAML(t *testing.T) {
	var s IncrementStrategy
	require.NoError(t, yaml.Unmarshal([]byte(`Minor`), &s))
	require.Equal(t, IncrementStrategyMinor, s)
}

func TestCommitMessageIncrementMode_UnmarshalYAML(t *testing.T) {
	var m CommitMessageIncrementMode
	require.NoError(t, yaml.Unmarshal([]byte(`Disabled`), &m))
	require.Equal(t, CommitMessageIncrementDisabled, m)
}

func TestCommitMessageConvention_UnmarshalYAML(t *testing.T) {
	var c CommitMessageConvention
	require.NoError(t, yaml.Unmarshal([]byte(`conventional-commits`), &c))
	require.Equal(t, CommitMessageConventionConventionalCommits, c)
}

func TestVersioningMode_UnmarshalYAML_Invalid(t *testing.T) {
	var m VersioningMode
	require.Error(t, yaml.Unmarshal([]byte(`bad`), &m))
}

func TestIncrementStrategy_UnmarshalYAML_Invalid(t *testing.T) {
	var s IncrementStrategy
	require.Error(t, yaml.Unmarshal([]byte(`bad`), &s))
}

func TestCommitMessageIncrementMode_UnmarshalYAML_Invalid(t *testing.T) {
	var m CommitMessageIncrementMode
	require.Error(t, yaml.Unmarshal([]byte(`bad`), &m))
}

func TestCommitMessageConvention_UnmarshalYAML_Invalid(t *testing.T) {
	var c CommitMessageConvention
	require.Error(t, yaml.Unmarshal([]byte(`bad`), &c))
}

func TestMainlineIncrementMode_UnmarshalYAML(t *testing.T) {
	var m MainlineIncrementMode
	require.NoError(t, yaml.Unmarshal([]byte(`EachCommit`), &m))
	require.Equal(t, MainlineIncrementEachCommit, m)
}

func TestMainlineIncrementMode_UnmarshalYAML_HyphenForm(t *testing.T) {
	var m MainlineIncrementMode
	require.NoError(t, yaml.Unmarshal([]byte(`each-commit`), &m))
	require.Equal(t, MainlineIncrementEachCommit, m)
}

func TestMainlineIncrementMode_UnmarshalYAML_Invalid(t *testing.T) {
	var m MainlineIncrementMode
	require.Error(t, yaml.Unmarshal([]byte(`bad`), &m))
}
