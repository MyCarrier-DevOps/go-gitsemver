package calculator

import (
	"fmt"

	"github.com/MyCarrier-DevOps/go-gitsemver/internal/config"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/context"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/git"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/semver"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/strategy"
)

// VersionResult holds the calculated version and metadata.
type VersionResult struct {
	Version              semver.SemanticVersion
	BaseVersion          strategy.BaseVersion
	BranchName           string
	CommitsSince         int64
	AllCandidates        []strategy.BaseVersion
	IncrementExplanation *IncrementExplanation // nil when explain is false
	PreReleaseSteps      []string              // nil when explain is false
}

// NextVersionCalculator orchestrates the full version calculation pipeline.
type NextVersionCalculator struct {
	store    *git.RepositoryStore
	base     *BaseVersionCalculator
	mainline *MainlineVersionCalculator
	incr     *IncrementStrategyFinder
}

// NewNextVersionCalculator creates a NextVersionCalculator with all sub-calculators.
func NewNextVersionCalculator(
	store *git.RepositoryStore,
	strategies []strategy.VersionStrategy,
) *NextVersionCalculator {
	incr := NewIncrementStrategyFinder(store)
	return &NextVersionCalculator{
		store:    store,
		base:     NewBaseVersionCalculator(store, strategies, incr),
		mainline: NewMainlineVersionCalculator(store, incr),
		incr:     incr,
	}
}

// Calculate computes the next version for the given context.
func (c *NextVersionCalculator) Calculate(
	ctx *context.GitVersionContext,
	ec config.EffectiveConfiguration,
	explain bool,
) (VersionResult, error) {
	// Step 1: If current commit is already tagged, return the tagged version.
	if ctx.IsCurrentCommitTagged {
		return VersionResult{
			Version:    ctx.CurrentCommitTaggedVersion,
			BranchName: branchNameForTag(ctx, ec),
		}, nil
	}

	// Step 2: Get the winning base version from all strategies.
	baseResult, err := c.base.Calculate(ctx, ec, explain)
	if err != nil {
		return VersionResult{}, err
	}

	bv := baseResult.BaseVersion

	// Step 3: Branch to Mainline or Standard mode.
	var ver semver.SemanticVersion
	var incrExp *IncrementExplanation

	if ec.BranchMode == semver.VersioningModeMainline {
		ver, incrExp, err = c.mainline.FindMainlineModeVersion(ctx, bv, ec, explain)
		if err != nil {
			return VersionResult{}, err
		}
	} else {
		ver, incrExp, err = c.standardModeVersion(ctx, bv, ec, explain)
		if err != nil {
			return VersionResult{}, err
		}
	}

	// Step 4: Update pre-release tag for non-release branches.
	branchName := effectiveBranchName(ctx, bv, ec)
	ver, preReleaseSteps := c.updatePreReleaseTag(ver, ctx, ec, branchName, explain)

	// Step 5: Count commits since base version source.
	commitsSince := c.countCommitsSince(ctx, bv)

	// Step 6: Build metadata.
	ver = c.applyBuildMetadata(ver, ctx, bv, branchName, commitsSince)

	return VersionResult{
		Version:              ver,
		BaseVersion:          bv,
		BranchName:           branchName,
		CommitsSince:         commitsSince,
		AllCandidates:        baseResult.AllCandidates,
		IncrementExplanation: incrExp,
		PreReleaseSteps:      preReleaseSteps,
	}, nil
}

// standardModeVersion applies the increment pipeline for ContinuousDelivery
// and ContinuousDeployment modes.
func (c *NextVersionCalculator) standardModeVersion(
	ctx *context.GitVersionContext,
	bv strategy.BaseVersion,
	ec config.EffectiveConfiguration,
	explain bool,
) (semver.SemanticVersion, *IncrementExplanation, error) {
	result, err := c.incr.DetermineIncrementedFieldExplained(ctx, bv, ec, explain)
	if err != nil {
		return semver.SemanticVersion{}, nil, err
	}

	ver := bv.SemanticVersion
	if result.Field != semver.VersionFieldNone {
		ver = ver.IncrementField(result.Field)
	}

	return ver, result.Explanation, nil
}

// updatePreReleaseTag sets the pre-release tag based on branch config.
// Returns the updated version and optional explain steps.
func (c *NextVersionCalculator) updatePreReleaseTag(
	ver semver.SemanticVersion,
	ctx *context.GitVersionContext,
	ec config.EffectiveConfiguration,
	branchName string,
	explain bool,
) (semver.SemanticVersion, []string) {
	// Release branches and main branches don't get pre-release tags.
	if ec.Tag == "" || ec.IsReleaseBranch || ec.IsMainline {
		return ver, nil
	}

	tagName := config.GetBranchSpecificTag(branchName, ec.Tag)
	if tagName == "" {
		return ver, nil
	}

	var steps []string
	if explain {
		steps = append(steps, fmt.Sprintf("branch config tag=%q -> %q", ec.Tag, tagName))
	}

	// Find the next pre-release number by looking at existing tags.
	number := int64(1)
	existingTags, err := c.store.GetValidVersionTags(ec.TagPrefix, nil)
	if err == nil {
		for _, vt := range existingTags {
			if vt.Version.Major == ver.Major &&
				vt.Version.Minor == ver.Minor &&
				vt.Version.Patch == ver.Patch &&
				vt.Version.PreReleaseTag.Name == tagName &&
				vt.Version.PreReleaseTag.Number != nil {
				if *vt.Version.PreReleaseTag.Number >= number {
					number = *vt.Version.PreReleaseTag.Number + 1
				}
			}
		}
	}

	if explain {
		if number == 1 {
			steps = append(steps, fmt.Sprintf("no existing tag for %d.%d.%d-%s -> number = 1", ver.Major, ver.Minor, ver.Patch, tagName))
		} else {
			steps = append(steps, fmt.Sprintf("existing tag for %d.%d.%d-%s -> number = %d", ver.Major, ver.Minor, ver.Patch, tagName, number))
		}
	}

	return ver.WithPreReleaseTag(semver.PreReleaseTag{Name: tagName, Number: &number}), steps
}

// effectiveBranchName returns the branch name to use for pre-release tags.
func effectiveBranchName(
	ctx *context.GitVersionContext,
	bv strategy.BaseVersion,
	ec config.EffectiveConfiguration,
) string {
	if bv.BranchNameOverride != "" {
		return bv.BranchNameOverride
	}
	_ = ec // available for future use
	return ctx.CurrentBranch.FriendlyName()
}

// branchNameForTag returns the branch name used when the current commit is tagged.
func branchNameForTag(ctx *context.GitVersionContext, ec config.EffectiveConfiguration) string {
	_ = ec
	return ctx.CurrentBranch.FriendlyName()
}

// countCommitsSince counts commits between base version source and current commit.
func (c *NextVersionCalculator) countCommitsSince(
	ctx *context.GitVersionContext,
	bv strategy.BaseVersion,
) int64 {
	from := git.Commit{}
	if bv.BaseVersionSource != nil {
		from = *bv.BaseVersionSource
	}

	commits, err := c.store.GetCommitLog(from, ctx.CurrentCommit)
	if err != nil {
		return 0
	}

	count := int64(len(commits))
	if bv.BaseVersionSource != nil {
		for _, co := range commits {
			if co.Sha == bv.BaseVersionSource.Sha {
				count--
				break
			}
		}
	}
	return count
}

// applyBuildMetadata adds build metadata to the version.
func (c *NextVersionCalculator) applyBuildMetadata(
	ver semver.SemanticVersion,
	ctx *context.GitVersionContext,
	bv strategy.BaseVersion,
	branchName string,
	commitsSince int64,
) semver.SemanticVersion {
	versionSourceSha := ""
	if bv.BaseVersionSource != nil {
		versionSourceSha = bv.BaseVersionSource.Sha
	}

	return ver.WithBuildMetaData(semver.BuildMetaData{
		CommitsSinceTag:           &commitsSince,
		Branch:                    branchName,
		Sha:                       ctx.CurrentCommit.Sha,
		ShortSha:                  ctx.CurrentCommit.ShortSha(),
		VersionSourceSha:          versionSourceSha,
		CommitDate:                ctx.CurrentCommit.When,
		CommitsSinceVersionSource: commitsSince,
		UncommittedChanges:        int64(ctx.NumberOfUncommittedChanges),
	})
}
