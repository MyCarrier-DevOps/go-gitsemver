// Package context provides the GitVersionContext, the immutable snapshot of
// git state and configuration used for version calculation.
package context

import (
	"go-gitsemver/internal/config"
	"go-gitsemver/internal/git"
	"go-gitsemver/internal/semver"
)

// GitVersionContext holds the resolved state needed for version calculation.
// It is created once per invocation and passed to all strategies.
type GitVersionContext struct {
	// CurrentBranch is the branch being versioned.
	CurrentBranch git.Branch

	// CurrentCommit is the commit being versioned (branch tip or explicit SHA).
	CurrentCommit git.Commit

	// FullConfiguration is the merged configuration (defaults + user overrides).
	FullConfiguration *config.Config

	// CurrentCommitTaggedVersion is the semantic version tag on the current
	// commit, if any. Zero value with IsCurrentCommitTagged=false means no tag.
	CurrentCommitTaggedVersion semver.SemanticVersion

	// IsCurrentCommitTagged is true when the current commit has a version tag.
	IsCurrentCommitTagged bool

	// NumberOfUncommittedChanges counts dirty working directory entries.
	NumberOfUncommittedChanges int
}

// GetEffectiveConfiguration resolves the effective configuration for the
// given branch, using the context's full configuration.
func (ctx *GitVersionContext) GetEffectiveConfiguration(branchName string) (config.EffectiveConfiguration, error) {
	bc, _, err := ctx.FullConfiguration.GetBranchConfiguration(branchName)
	if err != nil {
		return config.EffectiveConfiguration{}, err
	}
	return config.NewEffectiveConfiguration(ctx.FullConfiguration, bc), nil
}
