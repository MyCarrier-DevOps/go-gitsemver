package calculator

import (
	"testing"
	"time"

	"github.com/MyCarrier-DevOps/go-gitsemver/internal/config"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/context"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/git"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/semver"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/strategy"

	"github.com/stretchr/testify/require"
)

// stubStrategy returns fixed base versions.
type stubStrategy struct {
	name     string
	versions []strategy.BaseVersion
}

func (s *stubStrategy) Name() string { return s.name }
func (s *stubStrategy) GetBaseVersions(
	_ *context.GitVersionContext,
	_ config.EffectiveConfiguration,
	_ bool,
) ([]strategy.BaseVersion, error) {
	return s.versions, nil
}

func TestBaseVersionCalculator_SelectsHighest(t *testing.T) {
	older := newCommit("aaa0000000000000000000000000000000000000", "old")
	older.When = time.Now().Add(-1 * time.Hour)
	newer := newCommit("bbb0000000000000000000000000000000000000", "new")

	vs := &stubStrategy{
		name: "test",
		versions: []strategy.BaseVersion{
			{Source: "v1", SemanticVersion: semver.SemanticVersion{Major: 1}, BaseVersionSource: &older},
			{Source: "v2", SemanticVersion: semver.SemanticVersion{Major: 2}, BaseVersionSource: &newer},
		},
	}

	store := git.NewRepositoryStore(&git.MockRepository{})
	incr := NewIncrementStrategyFinder(store)
	calc := NewBaseVersionCalculator(store, []strategy.VersionStrategy{vs}, incr)

	ctx := &context.GitVersionContext{}
	result, err := calc.Calculate(ctx, defaultEC(), false)
	require.NoError(t, err)
	require.Equal(t, int64(2), result.BaseVersion.SemanticVersion.Major)
}

func TestBaseVersionCalculator_TieBreakOldest(t *testing.T) {
	older := newCommit("aaa0000000000000000000000000000000000000", "old")
	older.When = time.Now().Add(-2 * time.Hour)
	newer := newCommit("bbb0000000000000000000000000000000000000", "new")
	newer.When = time.Now()

	vs := &stubStrategy{
		name: "test",
		versions: []strategy.BaseVersion{
			{Source: "newer", SemanticVersion: semver.SemanticVersion{Major: 1}, BaseVersionSource: &newer},
			{Source: "older", SemanticVersion: semver.SemanticVersion{Major: 1}, BaseVersionSource: &older},
		},
	}

	store := git.NewRepositoryStore(&git.MockRepository{})
	incr := NewIncrementStrategyFinder(store)
	calc := NewBaseVersionCalculator(store, []strategy.VersionStrategy{vs}, incr)

	ctx := &context.GitVersionContext{}
	result, err := calc.Calculate(ctx, defaultEC(), false)
	require.NoError(t, err)
	// Tie-break: oldest source wins.
	require.Equal(t, "older", result.BaseVersion.Source)
}

func TestBaseVersionCalculator_EffectiveVersionRanking(t *testing.T) {
	older := newCommit("aaa0000000000000000000000000000000000000", "old")
	older.When = time.Now().Add(-1 * time.Hour)
	newer := newCommit("bbb0000000000000000000000000000000000000", "new")

	// v1.0.0 with ShouldIncrement=true → effective 1.0.1 (Patch)
	// v0.9.0 with ShouldIncrement=false → effective 0.9.0
	vs := &stubStrategy{
		name: "test",
		versions: []strategy.BaseVersion{
			{Source: "tag", SemanticVersion: semver.SemanticVersion{Minor: 9}, BaseVersionSource: &older},
			{Source: "branch", SemanticVersion: semver.SemanticVersion{Major: 1}, ShouldIncrement: true, BaseVersionSource: &newer},
		},
	}

	store := git.NewRepositoryStore(&git.MockRepository{})
	incr := NewIncrementStrategyFinder(store)
	calc := NewBaseVersionCalculator(store, []strategy.VersionStrategy{vs}, incr)

	ctx := &context.GitVersionContext{}
	ec := defaultEC()
	ec.BranchIncrement = semver.IncrementStrategyPatch

	result, err := calc.Calculate(ctx, ec, false)
	require.NoError(t, err)
	// 1.0.0+patch → effective 1.0.1 > 0.9.0
	require.Equal(t, "branch", result.BaseVersion.Source)
}

func TestBaseVersionCalculator_NoCandidates(t *testing.T) {
	vs := &stubStrategy{name: "empty", versions: nil}

	store := git.NewRepositoryStore(&git.MockRepository{})
	incr := NewIncrementStrategyFinder(store)
	calc := NewBaseVersionCalculator(store, []strategy.VersionStrategy{vs}, incr)

	ctx := &context.GitVersionContext{}
	_, err := calc.Calculate(ctx, defaultEC(), false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no base versions")
}

func TestBaseVersionCalculator_FilterIgnoreSha(t *testing.T) {
	ignored := newCommit("aaa0000000000000000000000000000000000000", "ignored")
	good := newCommit("bbb0000000000000000000000000000000000000", "good")

	vs := &stubStrategy{
		name: "test",
		versions: []strategy.BaseVersion{
			{Source: "v1", SemanticVersion: semver.SemanticVersion{Major: 2}, BaseVersionSource: &ignored},
			{Source: "v2", SemanticVersion: semver.SemanticVersion{Major: 1}, BaseVersionSource: &good},
		},
	}

	store := git.NewRepositoryStore(&git.MockRepository{})
	incr := NewIncrementStrategyFinder(store)
	calc := NewBaseVersionCalculator(store, []strategy.VersionStrategy{vs}, incr)

	ctx := &context.GitVersionContext{}
	ec := defaultEC()
	ec.IgnoreSha = []string{ignored.Sha}

	result, err := calc.Calculate(ctx, ec, false)
	require.NoError(t, err)
	require.Equal(t, "v2", result.BaseVersion.Source)
}

func TestBaseVersionCalculator_FilterIgnoreDate(t *testing.T) {
	old := newCommit("aaa0000000000000000000000000000000000000", "old")
	old.When = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	recent := newCommit("bbb0000000000000000000000000000000000000", "recent")
	recent.When = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	vs := &stubStrategy{
		name: "test",
		versions: []strategy.BaseVersion{
			{Source: "old-v", SemanticVersion: semver.SemanticVersion{Major: 3}, BaseVersionSource: &old},
			{Source: "new-v", SemanticVersion: semver.SemanticVersion{Major: 1}, BaseVersionSource: &recent},
		},
	}

	store := git.NewRepositoryStore(&git.MockRepository{})
	incr := NewIncrementStrategyFinder(store)
	calc := NewBaseVersionCalculator(store, []strategy.VersionStrategy{vs}, incr)

	ctx := &context.GitVersionContext{}
	ec := defaultEC()
	cutoff := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	ec.IgnoreCommitsBefore = &cutoff

	result, err := calc.Calculate(ctx, ec, false)
	require.NoError(t, err)
	require.Equal(t, "new-v", result.BaseVersion.Source)
}

func TestBaseVersionCalculator_AllFiltered(t *testing.T) {
	c := newCommit("aaa0000000000000000000000000000000000000", "only")

	vs := &stubStrategy{
		name: "test",
		versions: []strategy.BaseVersion{
			{Source: "v1", SemanticVersion: semver.SemanticVersion{Major: 1}, BaseVersionSource: &c},
		},
	}

	store := git.NewRepositoryStore(&git.MockRepository{})
	incr := NewIncrementStrategyFinder(store)
	calc := NewBaseVersionCalculator(store, []strategy.VersionStrategy{vs}, incr)

	ctx := &context.GitVersionContext{}
	ec := defaultEC()
	ec.IgnoreSha = []string{c.Sha}

	_, err := calc.Calculate(ctx, ec, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "filtered out")
}
