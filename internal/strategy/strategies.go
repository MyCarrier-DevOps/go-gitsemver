package strategy

import "github.com/MyCarrier-DevOps/go-gitsemver/internal/git"

// AllStrategies returns all version strategies in priority order.
// Strategies are evaluated in this order during base version selection:
//  1. ConfigNextVersion — explicit next-version override
//  2. TaggedCommit — version tags on branch history
//  3. MergeMessage — versions from merge/squash commit messages
//  4. VersionInBranchName — version extracted from release branch names
//  5. TrackReleaseBranches — release branch + main tag tracking (for develop)
//  6. Fallback — default base version when no other strategy matches
func AllStrategies(store *git.RepositoryStore) []VersionStrategy {
	return []VersionStrategy{
		NewConfigNextVersionStrategy(),
		NewTaggedCommitStrategy(store),
		NewMergeMessageStrategy(store),
		NewVersionInBranchNameStrategy(store),
		NewTrackReleaseBranchesStrategy(store),
		NewFallbackStrategy(store),
	}
}
