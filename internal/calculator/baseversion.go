package calculator

import (
	"errors"
	"fmt"
	"go-gitsemver/internal/config"
	"go-gitsemver/internal/context"
	"go-gitsemver/internal/git"
	"go-gitsemver/internal/semver"
	"go-gitsemver/internal/strategy"
)

// BaseVersionCalculator runs all strategies, computes effective versions for
// ranking, and selects the winning base version (DI-3).
type BaseVersionCalculator struct {
	store      *git.RepositoryStore
	strategies []strategy.VersionStrategy
	increment  *IncrementStrategyFinder
}

// NewBaseVersionCalculator creates a new BaseVersionCalculator.
func NewBaseVersionCalculator(
	store *git.RepositoryStore,
	strategies []strategy.VersionStrategy,
	increment *IncrementStrategyFinder,
) *BaseVersionCalculator {
	return &BaseVersionCalculator{
		store:      store,
		strategies: strategies,
		increment:  increment,
	}
}

// BaseVersionResult holds the selected base version and its effective config.
type BaseVersionResult struct {
	BaseVersion            strategy.BaseVersion
	EffectiveConfiguration config.EffectiveConfiguration
	AllCandidates          []strategy.BaseVersion
}

// Calculate runs all strategies, selects the highest effective version,
// and returns the winning base version.
func (c *BaseVersionCalculator) Calculate(
	ctx *context.GitVersionContext,
	ec config.EffectiveConfiguration,
	explain bool,
) (BaseVersionResult, error) {
	var allCandidates []strategy.BaseVersion

	// Run all strategies and collect candidates.
	for _, s := range c.strategies {
		versions, err := s.GetBaseVersions(ctx, ec, explain)
		if err != nil {
			return BaseVersionResult{}, fmt.Errorf("strategy %s: %w", s.Name(), err)
		}
		allCandidates = append(allCandidates, versions...)
	}

	if len(allCandidates) == 0 {
		return BaseVersionResult{}, errors.New("no base versions found from any strategy")
	}

	// Filter by ignore config.
	candidates := filterCandidates(allCandidates, ec)
	if len(candidates) == 0 {
		return BaseVersionResult{}, errors.New("all base versions were filtered out by ignore config")
	}

	// Select winner by computing effective versions (DI-3).
	winner := c.selectWinner(ctx, candidates, ec)

	return BaseVersionResult{
		BaseVersion:            winner,
		EffectiveConfiguration: ec,
		AllCandidates:          allCandidates,
	}, nil
}

// filterCandidates removes base versions that match ignore config.
func filterCandidates(candidates []strategy.BaseVersion, ec config.EffectiveConfiguration) []strategy.BaseVersion {
	if len(ec.IgnoreSha) == 0 && ec.IgnoreCommitsBefore == nil {
		return candidates
	}

	ignoreShaSet := make(map[string]struct{}, len(ec.IgnoreSha))
	for _, sha := range ec.IgnoreSha {
		ignoreShaSet[sha] = struct{}{}
	}

	var filtered []strategy.BaseVersion
	for _, bv := range candidates {
		if bv.BaseVersionSource != nil {
			if _, ignored := ignoreShaSet[bv.BaseVersionSource.Sha]; ignored {
				continue
			}
			if ec.IgnoreCommitsBefore != nil && bv.BaseVersionSource.When.Before(*ec.IgnoreCommitsBefore) {
				continue
			}
		}
		filtered = append(filtered, bv)
	}
	return filtered
}

// selectWinner selects the base version with the highest "effective version."
// DI-3: if ShouldIncrement, tentatively increment to compute effective version
// for ranking. The actual increment happens later in the pipeline.
// Tie-break: oldest BaseVersionSource (more commits → more accurate count).
func (c *BaseVersionCalculator) selectWinner(
	ctx *context.GitVersionContext,
	candidates []strategy.BaseVersion,
	ec config.EffectiveConfiguration,
) strategy.BaseVersion {
	best := candidates[0]
	bestEffective := c.effectiveVersion(ctx, best, ec)

	for _, bv := range candidates[1:] {
		effective := c.effectiveVersion(ctx, bv, ec)

		cmp := effective.CompareTo(bestEffective)
		if cmp > 0 {
			best = bv
			bestEffective = effective
		} else if cmp == 0 {
			// Tie-break: oldest source commit wins (more history = more accurate count).
			if bv.BaseVersionSource != nil && best.BaseVersionSource != nil {
				if bv.BaseVersionSource.When.Before(best.BaseVersionSource.When) {
					best = bv
					bestEffective = effective
				}
			}
		}
	}

	return best
}

// effectiveVersion computes the version that would result if we incremented.
// Used only for ranking — no mutation occurs.
func (c *BaseVersionCalculator) effectiveVersion(
	ctx *context.GitVersionContext,
	bv strategy.BaseVersion,
	ec config.EffectiveConfiguration,
) semver.SemanticVersion {
	if !bv.ShouldIncrement {
		return bv.SemanticVersion
	}
	field := ec.BranchIncrement.ToVersionField()
	if field == semver.VersionFieldNone {
		field = semver.VersionFieldPatch
	}
	return bv.SemanticVersion.IncrementField(field)
}
