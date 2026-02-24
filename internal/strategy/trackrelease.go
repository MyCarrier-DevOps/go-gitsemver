package strategy

import (
	"fmt"

	"github.com/MyCarrier-DevOps/go-gitsemver/internal/config"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/context"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/git"
)

// TrackReleaseBranchesStrategy combines release branch versions and main
// branch tags for branches that track release branches (e.g., develop).
type TrackReleaseBranchesStrategy struct {
	store          *git.RepositoryStore
	tagStrategy    *TaggedCommitStrategy
	branchStrategy *VersionInBranchNameStrategy
}

// NewTrackReleaseBranchesStrategy creates a new TrackReleaseBranchesStrategy.
func NewTrackReleaseBranchesStrategy(store *git.RepositoryStore) *TrackReleaseBranchesStrategy {
	return &TrackReleaseBranchesStrategy{
		store:          store,
		tagStrategy:    NewTaggedCommitStrategy(store),
		branchStrategy: NewVersionInBranchNameStrategy(store),
	}
}

func (s *TrackReleaseBranchesStrategy) Name() string { return "TrackReleaseBranches" }

func (s *TrackReleaseBranchesStrategy) GetBaseVersions(
	ctx *context.GitVersionContext,
	ec config.EffectiveConfiguration,
	explain bool,
) ([]BaseVersion, error) {
	if !ec.TracksReleaseBranches {
		return nil, nil
	}

	var exp *Explanation
	if explain {
		exp = NewExplanation(s.Name())
	}

	releaseBranchVersions, err := s.releaseBranchBaseVersions(ctx, ec, explain)
	if err != nil {
		return nil, fmt.Errorf("release branch versions: %w", err)
	}

	mainTagVersions, err := s.mainTagsVersions(ctx, ec, explain)
	if err != nil {
		return nil, fmt.Errorf("main tag versions: %w", err)
	}

	exp.Addf("found %d release branch versions + %d main tag versions",
		len(releaseBranchVersions), len(mainTagVersions))

	results := make([]BaseVersion, 0, len(releaseBranchVersions)+len(mainTagVersions))
	results = append(results, releaseBranchVersions...)
	results = append(results, mainTagVersions...)
	return results, nil
}

func (s *TrackReleaseBranchesStrategy) releaseBranchBaseVersions(
	ctx *context.GitVersionContext,
	ec config.EffectiveConfiguration,
	explain bool,
) ([]BaseVersion, error) {
	releaseBranchConfig := ctx.FullConfiguration.GetReleaseBranchConfig()
	if len(releaseBranchConfig) == 0 {
		return nil, nil
	}

	releaseBranches, err := s.store.GetReleaseBranches(releaseBranchConfig)
	if err != nil {
		return nil, err
	}

	var results []BaseVersion

	for _, rb := range releaseBranches {
		mergeBase, found, err := s.store.FindMergeBase(rb, ctx.CurrentBranch)
		if err != nil || !found {
			continue
		}

		// Skip if merge base is the current commit (branch has no own commits).
		if mergeBase.Sha == ctx.CurrentCommit.Sha {
			continue
		}

		releaseEC, err := ctx.GetEffectiveConfiguration(rb.FriendlyName())
		if err != nil {
			continue
		}

		branchVersions, err := s.branchStrategy.getBaseVersionsForBranch(ctx, releaseEC, rb, explain)
		if err != nil {
			continue
		}

		// Remap: set ShouldIncrement=true, use merge base as source, drop branch override.
		for _, bv := range branchVersions {
			mb := mergeBase
			results = append(results, BaseVersion{
				Source:            "Release branch exists -> " + bv.Source,
				ShouldIncrement:   true,
				SemanticVersion:   bv.SemanticVersion,
				BaseVersionSource: &mb,
				Explanation:       bv.Explanation,
			})
		}
	}

	return results, nil
}

func (s *TrackReleaseBranchesStrategy) mainTagsVersions(
	ctx *context.GitVersionContext,
	ec config.EffectiveConfiguration,
	explain bool,
) ([]BaseVersion, error) {
	mainBranch, found, err := s.store.FindMainBranch(ctx.FullConfiguration)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}

	return s.tagStrategy.getTaggedVersions(ctx, ec, mainBranch, nil, explain)
}
