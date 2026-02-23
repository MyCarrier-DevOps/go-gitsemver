package calculator

import (
	"go-gitsemver/internal/config"
	"go-gitsemver/internal/context"
	"go-gitsemver/internal/git"
	"go-gitsemver/internal/semver"
	"go-gitsemver/internal/strategy"
)

// MainlineVersionCalculator computes versions using the aggregate-increment
// approach (DI-10): find latest tag, collect commits since, determine the
// single highest increment, and apply it once.
type MainlineVersionCalculator struct {
	store     *git.RepositoryStore
	increment *IncrementStrategyFinder
}

// NewMainlineVersionCalculator creates a new MainlineVersionCalculator.
func NewMainlineVersionCalculator(
	store *git.RepositoryStore,
	increment *IncrementStrategyFinder,
) *MainlineVersionCalculator {
	return &MainlineVersionCalculator{store: store, increment: increment}
}

// FindMainlineModeVersion computes the mainline version.
// 1. Use the base version as the starting point.
// 2. Determine the single highest increment from commits.
// 3. Apply the increment once.
// 4. Commit count since base version source goes into build metadata.
func (m *MainlineVersionCalculator) FindMainlineModeVersion(
	ctx *context.GitVersionContext,
	bv strategy.BaseVersion,
	ec config.EffectiveConfiguration,
) (semver.SemanticVersion, error) {
	// Determine increment from commit messages.
	field, err := m.increment.DetermineIncrementedField(ctx, bv, ec)
	if err != nil {
		return semver.SemanticVersion{}, err
	}

	ver := bv.SemanticVersion

	// Apply increment if needed.
	if field != semver.VersionFieldNone {
		ver = ver.IncrementField(field)
	} else if bv.ShouldIncrement {
		// Fallback to branch default.
		defaultField := ec.BranchIncrement.ToVersionField()
		if defaultField == semver.VersionFieldNone {
			defaultField = semver.VersionFieldPatch
		}
		ver = ver.IncrementField(defaultField)
	}

	// Count commits since base version source for metadata.
	from := git.Commit{}
	if bv.BaseVersionSource != nil {
		from = *bv.BaseVersionSource
	}

	commits, err := m.store.GetCommitLog(from, ctx.CurrentCommit)
	if err != nil {
		return semver.SemanticVersion{}, err
	}

	// Exclude the base version source commit from the count.
	count := int64(len(commits))
	if bv.BaseVersionSource != nil {
		for _, c := range commits {
			if c.Sha == bv.BaseVersionSource.Sha {
				count--
				break
			}
		}
	}

	ver = ver.WithBuildMetaData(semver.BuildMetaData{
		CommitsSinceTag:           &count,
		Branch:                    ctx.CurrentBranch.FriendlyName(),
		Sha:                       ctx.CurrentCommit.Sha,
		ShortSha:                  ctx.CurrentCommit.ShortSha(),
		CommitsSinceVersionSource: count,
	})

	return ver, nil
}
