package strategy

import (
	"fmt"
	"go-gitsemver/internal/config"
	"go-gitsemver/internal/context"
	"go-gitsemver/internal/git"
	"go-gitsemver/internal/semver"
	"regexp"
	"strings"
)

// VersionInBranchNameStrategy returns a version from the branch name for release branches.
type VersionInBranchNameStrategy struct {
	store *git.RepositoryStore
}

// NewVersionInBranchNameStrategy creates a new VersionInBranchNameStrategy.
func NewVersionInBranchNameStrategy(store *git.RepositoryStore) *VersionInBranchNameStrategy {
	return &VersionInBranchNameStrategy{store: store}
}

func (s *VersionInBranchNameStrategy) Name() string { return "VersionInBranchName" }

func (s *VersionInBranchNameStrategy) GetBaseVersions(
	ctx *context.GitVersionContext,
	ec config.EffectiveConfiguration,
	explain bool,
) ([]BaseVersion, error) {
	return s.getBaseVersionsForBranch(ctx, ec, ctx.CurrentBranch, explain)
}

// getBaseVersionsForBranch extracts version from a specific branch name.
// Also called by TrackReleaseBranchesStrategy for each release branch.
func (s *VersionInBranchNameStrategy) getBaseVersionsForBranch(
	ctx *context.GitVersionContext,
	ec config.EffectiveConfiguration,
	branch git.Branch,
	explain bool,
) ([]BaseVersion, error) {
	var exp *Explanation
	if explain {
		exp = NewExplanation(s.Name())
	}

	branchName := branch.Name.WithoutRemote

	// Check if this branch is a release branch.
	if !ctx.FullConfiguration.IsReleaseBranch(branchName) {
		exp.Addf("branch %q is not a release branch, skipping", branchName)
		return nil, nil
	}

	// Extract version from branch name.
	versionStr, ok := git.ExtractVersionFromBranch(branch.FriendlyName(), ec.TagPrefix)
	if !ok {
		exp.Addf("no version found in branch name %q", branch.FriendlyName())
		return nil, nil
	}

	ver, err := semver.Parse(versionStr, "")
	if err != nil {
		return nil, fmt.Errorf("parsing version from branch name %q: %w", versionStr, err)
	}

	// Find where this branch was created from its parent.
	branchPoint, err := s.store.FindCommitBranchWasBranchedFrom(branch, ctx.FullConfiguration)
	if err != nil {
		return nil, fmt.Errorf("finding branch point: %w", err)
	}

	var sourceCommit *git.Commit
	if !branchPoint.Commit.IsEmpty() {
		c := branchPoint.Commit
		sourceCommit = &c
	}

	// Compute branch name override: branch name with version segment stripped.
	branchNameOverride := computeBranchNameOverride(branch.FriendlyName(), versionStr)

	exp.Addf("branch %q -> version %s, override=%q", branch.FriendlyName(), ver.SemVer(), branchNameOverride)

	return []BaseVersion{{
		Source:             "Version in branch name",
		ShouldIncrement:    false,
		SemanticVersion:    ver,
		BaseVersionSource:  sourceCommit,
		BranchNameOverride: branchNameOverride,
		Explanation:        exp,
	}}, nil
}

// computeBranchNameOverride strips the version segment from the branch name.
func computeBranchNameOverride(branchName, version string) string {
	re := regexp.MustCompile(`[-/]` + regexp.QuoteMeta(version))
	result := re.ReplaceAllString(branchName, "")
	return strings.TrimRight(result, "/-")
}
