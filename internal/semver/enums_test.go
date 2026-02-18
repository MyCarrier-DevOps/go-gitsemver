package semver

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionField_String(t *testing.T) {
	tests := []struct {
		field VersionField
		want  string
	}{
		{VersionFieldNone, "None"},
		{VersionFieldPatch, "Patch"},
		{VersionFieldMinor, "Minor"},
		{VersionFieldMajor, "Major"},
		{VersionField(99), "Unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			require.Equal(t, tt.want, tt.field.String())
		})
	}
}

func TestIncrementStrategy_String(t *testing.T) {
	tests := []struct {
		strategy IncrementStrategy
		want     string
	}{
		{IncrementStrategyNone, "None"},
		{IncrementStrategyMajor, "Major"},
		{IncrementStrategyMinor, "Minor"},
		{IncrementStrategyPatch, "Patch"},
		{IncrementStrategyInherit, "Inherit"},
		{IncrementStrategy(99), "Unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			require.Equal(t, tt.want, tt.strategy.String())
		})
	}
}

func TestIncrementStrategy_ToVersionField(t *testing.T) {
	tests := []struct {
		strategy IncrementStrategy
		want     VersionField
	}{
		{IncrementStrategyMajor, VersionFieldMajor},
		{IncrementStrategyMinor, VersionFieldMinor},
		{IncrementStrategyPatch, VersionFieldPatch},
		{IncrementStrategyNone, VersionFieldNone},
		{IncrementStrategyInherit, VersionFieldNone},
	}
	for _, tt := range tests {
		t.Run(tt.strategy.String(), func(t *testing.T) {
			require.Equal(t, tt.want, tt.strategy.ToVersionField())
		})
	}
}

func TestVersioningMode_String(t *testing.T) {
	tests := []struct {
		mode VersioningMode
		want string
	}{
		{VersioningModeContinuousDelivery, "ContinuousDelivery"},
		{VersioningModeContinuousDeployment, "ContinuousDeployment"},
		{VersioningModeMainline, "Mainline"},
		{VersioningMode(99), "Unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			require.Equal(t, tt.want, tt.mode.String())
		})
	}
}

func TestCommitMessageIncrementMode_String(t *testing.T) {
	tests := []struct {
		mode CommitMessageIncrementMode
		want string
	}{
		{CommitMessageIncrementEnabled, "Enabled"},
		{CommitMessageIncrementDisabled, "Disabled"},
		{CommitMessageIncrementMergeMessageOnly, "MergeMessageOnly"},
		{CommitMessageIncrementMode(99), "Unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			require.Equal(t, tt.want, tt.mode.String())
		})
	}
}

func TestCommitMessageConvention_String(t *testing.T) {
	tests := []struct {
		convention CommitMessageConvention
		want       string
	}{
		{CommitMessageConventionConventionalCommits, "ConventionalCommits"},
		{CommitMessageConventionBumpDirective, "BumpDirective"},
		{CommitMessageConventionBoth, "Both"},
		{CommitMessageConvention(99), "Unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			require.Equal(t, tt.want, tt.convention.String())
		})
	}
}
