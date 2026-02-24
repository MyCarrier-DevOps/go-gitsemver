package strategy

import (
	"errors"
	"fmt"

	"github.com/MyCarrier-DevOps/go-gitsemver/internal/config"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/context"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/git"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/semver"
)

// FallbackStrategy returns the base version (default 0.1.0) from the root commit.
type FallbackStrategy struct {
	store *git.RepositoryStore
}

// NewFallbackStrategy creates a new FallbackStrategy.
func NewFallbackStrategy(store *git.RepositoryStore) *FallbackStrategy {
	return &FallbackStrategy{store: store}
}

func (s *FallbackStrategy) Name() string { return "Fallback" }

func (s *FallbackStrategy) GetBaseVersions(
	ctx *context.GitVersionContext,
	ec config.EffectiveConfiguration,
	explain bool,
) ([]BaseVersion, error) {
	var exp *Explanation
	if explain {
		exp = NewExplanation(s.Name())
	}

	if ctx.CurrentBranch.Tip == nil {
		return nil, errors.New("no commits found on the current branch")
	}

	rootCommit, err := s.store.GetBaseVersionSource(ctx.CurrentCommit)
	if err != nil {
		return nil, fmt.Errorf("finding root commit: %w", err)
	}

	baseVersionStr := ec.BaseVersion
	ver, err := semver.Parse(baseVersionStr, "")
	if err != nil {
		// Hard fallback to 0.1.0 if config value is unparseable.
		ver = semver.SemanticVersion{Minor: 1}
	}

	exp.Addf("using base version %s from root commit %s", ver.SemVer(), rootCommit.ShortSha())

	return []BaseVersion{{
		Source:            "Fallback base version",
		ShouldIncrement:   false,
		SemanticVersion:   ver,
		BaseVersionSource: &rootCommit,
		Explanation:       exp,
	}}, nil
}
