// Package semver provides immutable semantic versioning types.
package semver

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
