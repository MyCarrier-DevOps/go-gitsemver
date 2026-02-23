// Package semver provides immutable semantic versioning types.
package semver

import (
	"fmt"
	"strings"
)

// VersionField represents which field of a semantic version to increment.
type VersionField int

const (
	VersionFieldNone VersionField = iota
	VersionFieldPatch
	VersionFieldMinor
	VersionFieldMajor
)

func (f VersionField) String() string {
	switch f {
	case VersionFieldNone:
		return "None"
	case VersionFieldPatch:
		return "Patch"
	case VersionFieldMinor:
		return "Minor"
	case VersionFieldMajor:
		return "Major"
	default:
		return "Unknown"
	}
}

// IncrementStrategy represents the configured increment strategy for a branch.
type IncrementStrategy int

const (
	IncrementStrategyNone IncrementStrategy = iota
	IncrementStrategyMajor
	IncrementStrategyMinor
	IncrementStrategyPatch
	IncrementStrategyInherit
)

func (s IncrementStrategy) String() string {
	switch s {
	case IncrementStrategyNone:
		return "None"
	case IncrementStrategyMajor:
		return "Major"
	case IncrementStrategyMinor:
		return "Minor"
	case IncrementStrategyPatch:
		return "Patch"
	case IncrementStrategyInherit:
		return "Inherit"
	default:
		return "Unknown"
	}
}

// ToVersionField converts an IncrementStrategy to a VersionField.
// Inherit and None both map to VersionFieldNone.
func (s IncrementStrategy) ToVersionField() VersionField {
	switch s {
	case IncrementStrategyMajor:
		return VersionFieldMajor
	case IncrementStrategyMinor:
		return VersionFieldMinor
	case IncrementStrategyPatch:
		return VersionFieldPatch
	default:
		return VersionFieldNone
	}
}

// VersioningMode represents the versioning mode.
type VersioningMode int

const (
	VersioningModeContinuousDelivery VersioningMode = iota
	VersioningModeContinuousDeployment
	VersioningModeMainline
)

func (m VersioningMode) String() string {
	switch m {
	case VersioningModeContinuousDelivery:
		return "ContinuousDelivery"
	case VersioningModeContinuousDeployment:
		return "ContinuousDeployment"
	case VersioningModeMainline:
		return "Mainline"
	default:
		return "Unknown"
	}
}

// CommitMessageIncrementMode controls how commit messages affect version incrementing.
type CommitMessageIncrementMode int

const (
	CommitMessageIncrementEnabled CommitMessageIncrementMode = iota
	CommitMessageIncrementDisabled
	CommitMessageIncrementMergeMessageOnly
)

func (m CommitMessageIncrementMode) String() string {
	switch m {
	case CommitMessageIncrementEnabled:
		return "Enabled"
	case CommitMessageIncrementDisabled:
		return "Disabled"
	case CommitMessageIncrementMergeMessageOnly:
		return "MergeMessageOnly"
	default:
		return "Unknown"
	}
}

// CommitMessageConvention controls which commit message conventions are used
// for version incrementing.
type CommitMessageConvention int

const (
	CommitMessageConventionConventionalCommits CommitMessageConvention = iota
	CommitMessageConventionBumpDirective
	CommitMessageConventionBoth
)

func (c CommitMessageConvention) String() string {
	switch c {
	case CommitMessageConventionConventionalCommits:
		return "ConventionalCommits"
	case CommitMessageConventionBumpDirective:
		return "BumpDirective"
	case CommitMessageConventionBoth:
		return "Both"
	default:
		return "Unknown"
	}
}

// MainlineIncrementMode controls how mainline mode applies version increments.
type MainlineIncrementMode int

const (
	// MainlineIncrementAggregate finds the highest increment from all commits
	// since the last tag and applies it once. Commit count goes into build metadata.
	MainlineIncrementAggregate MainlineIncrementMode = iota
	// MainlineIncrementEachCommit increments the version for each commit
	// individually, matching GitVersion's per-commit behavior.
	MainlineIncrementEachCommit
)

func (m MainlineIncrementMode) String() string {
	switch m {
	case MainlineIncrementAggregate:
		return "Aggregate"
	case MainlineIncrementEachCommit:
		return "EachCommit"
	default:
		return "Unknown"
	}
}

// ParseMainlineIncrementMode parses a string into a MainlineIncrementMode.
// Matching is case-insensitive. Accepts hyphenated forms (e.g. "each-commit").
func ParseMainlineIncrementMode(s string) (MainlineIncrementMode, error) {
	switch strings.ToLower(s) {
	case "aggregate":
		return MainlineIncrementAggregate, nil
	case "eachcommit", "each-commit":
		return MainlineIncrementEachCommit, nil
	default:
		return 0, fmt.Errorf("unknown mainline increment mode: %q", s)
	}
}

// ParseVersioningMode parses a string into a VersioningMode.
// Matching is case-insensitive.
func ParseVersioningMode(s string) (VersioningMode, error) {
	switch strings.ToLower(s) {
	case "continuousdelivery":
		return VersioningModeContinuousDelivery, nil
	case "continuousdeployment":
		return VersioningModeContinuousDeployment, nil
	case "mainline":
		return VersioningModeMainline, nil
	default:
		return 0, fmt.Errorf("unknown versioning mode: %q", s)
	}
}

// ParseIncrementStrategy parses a string into an IncrementStrategy.
// Matching is case-insensitive.
func ParseIncrementStrategy(s string) (IncrementStrategy, error) {
	switch strings.ToLower(s) {
	case "none":
		return IncrementStrategyNone, nil
	case "major":
		return IncrementStrategyMajor, nil
	case "minor":
		return IncrementStrategyMinor, nil
	case "patch":
		return IncrementStrategyPatch, nil
	case "inherit":
		return IncrementStrategyInherit, nil
	default:
		return 0, fmt.Errorf("unknown increment strategy: %q", s)
	}
}

// ParseCommitMessageIncrementMode parses a string into a CommitMessageIncrementMode.
// Matching is case-insensitive.
func ParseCommitMessageIncrementMode(s string) (CommitMessageIncrementMode, error) {
	switch strings.ToLower(s) {
	case "enabled":
		return CommitMessageIncrementEnabled, nil
	case "disabled":
		return CommitMessageIncrementDisabled, nil
	case "mergemessageonly":
		return CommitMessageIncrementMergeMessageOnly, nil
	default:
		return 0, fmt.Errorf("unknown commit message increment mode: %q", s)
	}
}

// ParseCommitMessageConvention parses a string into a CommitMessageConvention.
// Matching is case-insensitive. Accepts hyphenated forms (e.g. "conventional-commits").
func ParseCommitMessageConvention(s string) (CommitMessageConvention, error) {
	switch strings.ToLower(s) {
	case "conventionalcommits", "conventional-commits":
		return CommitMessageConventionConventionalCommits, nil
	case "bumpdirective", "bump-directive":
		return CommitMessageConventionBumpDirective, nil
	case "both":
		return CommitMessageConventionBoth, nil
	default:
		return 0, fmt.Errorf("unknown commit message convention: %q", s)
	}
}
