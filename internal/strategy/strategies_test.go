package strategy

import (
	"testing"

	"github.com/MyCarrier-DevOps/go-gitsemver/internal/git"

	"github.com/stretchr/testify/require"
)

func TestAllStrategies_ReturnsAll(t *testing.T) {
	mock := &git.MockRepository{}
	store := git.NewRepositoryStore(mock)

	strategies := AllStrategies(store)
	require.Len(t, strategies, 6)

	names := make([]string, len(strategies))
	for i, s := range strategies {
		names[i] = s.Name()
	}

	require.Equal(t, []string{
		"ConfigNextVersion",
		"TaggedCommit",
		"MergeMessage",
		"VersionInBranchName",
		"TrackReleaseBranches",
		"Fallback",
	}, names)
}
