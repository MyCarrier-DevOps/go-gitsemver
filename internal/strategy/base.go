// Package strategy implements the 6 version strategies that discover candidate
// base versions from git history and configuration.
package strategy

import (
	"fmt"
	"go-gitsemver/internal/config"
	"go-gitsemver/internal/context"
	"go-gitsemver/internal/git"
	"go-gitsemver/internal/semver"
)

// BaseVersion represents a candidate version found by a strategy.
type BaseVersion struct {
	// Source is a human-readable description of where this version came from.
	Source string

	// ShouldIncrement indicates whether this version should be bumped before
	// comparison with other candidates.
	ShouldIncrement bool

	// SemanticVersion is the version found by the strategy.
	SemanticVersion semver.SemanticVersion

	// BaseVersionSource is the git commit this version originated from.
	// Nil for external sources (e.g., config next-version).
	BaseVersionSource *git.Commit

	// BranchNameOverride overrides the branch name used for pre-release
	// tag generation. Used by VersionInBranchName strategy.
	BranchNameOverride string

	// Explanation records the reasoning chain for how this version was derived.
	// Nil when explain mode is disabled (DI-9).
	Explanation *Explanation
}

// String returns a human-readable representation of the base version.
func (bv BaseVersion) String() string {
	source := "external"
	if bv.BaseVersionSource != nil {
		source = bv.BaseVersionSource.ShortSha()
	}
	return fmt.Sprintf("%s: %s (source: %s, increment: %t)",
		bv.Source, bv.SemanticVersion.SemVer(), source, bv.ShouldIncrement)
}

// Explanation records how a strategy derived a BaseVersion (DI-9).
type Explanation struct {
	// Strategy is the name of the strategy that produced this version.
	Strategy string

	// Steps records the reasoning chain in order.
	Steps []string
}

// NewExplanation creates a new Explanation for the given strategy name.
func NewExplanation(strategy string) *Explanation {
	return &Explanation{Strategy: strategy}
}

// Add appends a reasoning step. Nil-safe.
func (e *Explanation) Add(step string) {
	if e != nil {
		e.Steps = append(e.Steps, step)
	}
}

// Addf appends a formatted reasoning step. Nil-safe.
func (e *Explanation) Addf(format string, args ...any) {
	if e != nil {
		e.Steps = append(e.Steps, fmt.Sprintf(format, args...))
	}
}

// VersionStrategy is the interface implemented by all version discovery strategies.
type VersionStrategy interface {
	// Name returns the human-readable name of this strategy.
	Name() string

	// GetBaseVersions computes zero or more candidate base versions.
	// When explain is true, strategies populate Explanation on each
	// returned BaseVersion.
	GetBaseVersions(
		ctx *context.GitVersionContext,
		ec config.EffectiveConfiguration,
		explain bool,
	) ([]BaseVersion, error)
}
