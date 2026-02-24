package calculator

import (
	"go-gitsemver/internal/config"
	"go-gitsemver/internal/context"
	"go-gitsemver/internal/git"
	"go-gitsemver/internal/semver"
	"go-gitsemver/internal/strategy"
	"slices"
	"strings"
)

// MainlineVersionCalculator computes versions in mainline mode.
// Supports two increment modes controlled by ec.MainlineIncrement:
//   - Aggregate (default): highest increment applied once, commit count in build metadata
//   - EachCommit: version incremented per commit individually
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
func (m *MainlineVersionCalculator) FindMainlineModeVersion(
	ctx *context.GitVersionContext,
	bv strategy.BaseVersion,
	ec config.EffectiveConfiguration,
	explain bool,
) (semver.SemanticVersion, *IncrementExplanation, error) {
	if ec.MainlineIncrement == semver.MainlineIncrementEachCommit {
		return m.eachCommitVersion(ctx, bv, ec, explain)
	}
	return m.aggregateVersion(ctx, bv, ec, explain)
}

// aggregateVersion is the default approach: find the single highest
// increment from all commits since the last tag and apply it once.
// Commit count goes into build metadata.
func (m *MainlineVersionCalculator) aggregateVersion(
	ctx *context.GitVersionContext,
	bv strategy.BaseVersion,
	ec config.EffectiveConfiguration,
	explain bool,
) (semver.SemanticVersion, *IncrementExplanation, error) {
	result, err := m.increment.DetermineIncrementedFieldExplained(ctx, bv, ec, explain)
	if err != nil {
		return semver.SemanticVersion{}, nil, err
	}

	ver := bv.SemanticVersion

	if result.Field != semver.VersionFieldNone {
		ver = ver.IncrementField(result.Field)
	} else if bv.ShouldIncrement {
		defaultField := ec.BranchIncrement.ToVersionField()
		if defaultField == semver.VersionFieldNone {
			defaultField = semver.VersionFieldPatch
		}
		ver = ver.IncrementField(defaultField)
	}

	commits, count := m.commitsSince(bv, ctx)
	_ = commits

	ver = m.withBuildMetaData(ver, ctx, count)
	return ver, result.Explanation, nil
}

// eachCommitVersion walks each commit since the base version and increments
// the version individually for each one.
func (m *MainlineVersionCalculator) eachCommitVersion(
	ctx *context.GitVersionContext,
	bv strategy.BaseVersion,
	ec config.EffectiveConfiguration,
	explain bool,
) (semver.SemanticVersion, *IncrementExplanation, error) {
	commits, count := m.commitsSince(bv, ctx)

	var exp *IncrementExplanation
	if explain {
		exp = &IncrementExplanation{}
		exp.Addf("mainline EachCommit mode: walking %d commits", count)
	}

	ver := bv.SemanticVersion
	defaultField := ec.BranchIncrement.ToVersionField()
	if defaultField == semver.VersionFieldNone {
		defaultField = semver.VersionFieldPatch
	}

	// Walk commits oldest-first (commit log returns newest-first).
	slices.Reverse(commits)

	for _, c := range commits {
		// Skip the base version source commit.
		if bv.BaseVersionSource != nil && c.Sha == bv.BaseVersionSource.Sha {
			continue
		}

		field := m.increment.AnalyzeCommitIncrement(c, ec)

		// Cap Major to Minor for pre-1.0 versions.
		if ver.Major == 0 && field == semver.VersionFieldMajor {
			field = semver.VersionFieldMinor
		}

		if field != semver.VersionFieldNone {
			ver = ver.IncrementField(field)
		} else if bv.ShouldIncrement {
			ver = ver.IncrementField(defaultField)
		}

		if explain {
			firstLine := c.Message
			if idx := strings.IndexByte(firstLine, '\n'); idx >= 0 {
				firstLine = firstLine[:idx]
			}
			exp.Addf("commit %s %q -> %s -> %s", c.ShortSha(), firstLine, field, ver.SemVer())
		}
	}

	if explain {
		exp.Addf("final mainline version: %s", ver.SemVer())
	}

	ver = m.withBuildMetaData(ver, ctx, count)
	return ver, exp, nil
}

// commitsSince returns commits between base version source and current commit,
// along with the count (excluding the source commit itself).
func (m *MainlineVersionCalculator) commitsSince(
	bv strategy.BaseVersion,
	ctx *context.GitVersionContext,
) ([]git.Commit, int64) {
	from := git.Commit{}
	if bv.BaseVersionSource != nil {
		from = *bv.BaseVersionSource
	}

	commits, err := m.store.GetCommitLog(from, ctx.CurrentCommit)
	if err != nil {
		return nil, 0
	}

	count := int64(len(commits))
	if bv.BaseVersionSource != nil {
		for _, c := range commits {
			if c.Sha == bv.BaseVersionSource.Sha {
				count--
				break
			}
		}
	}

	return commits, count
}

// withBuildMetaData attaches build metadata to the version.
func (m *MainlineVersionCalculator) withBuildMetaData(
	ver semver.SemanticVersion,
	ctx *context.GitVersionContext,
	count int64,
) semver.SemanticVersion {
	return ver.WithBuildMetaData(semver.BuildMetaData{
		CommitsSinceTag:           &count,
		Branch:                    ctx.CurrentBranch.FriendlyName(),
		Sha:                       ctx.CurrentCommit.Sha,
		ShortSha:                  ctx.CurrentCommit.ShortSha(),
		CommitsSinceVersionSource: count,
	})
}
