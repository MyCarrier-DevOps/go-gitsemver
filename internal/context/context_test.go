package context

import (
	"go-gitsemver/internal/config"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetEffectiveConfiguration_Main(t *testing.T) {
	cfg, err := config.NewBuilder().Build()
	require.NoError(t, err)

	ctx := &GitVersionContext{FullConfiguration: cfg}

	ec, err := ctx.GetEffectiveConfiguration("main")
	require.NoError(t, err)
	require.True(t, ec.IsMainline)
	require.False(t, ec.IsReleaseBranch)
}

func TestGetEffectiveConfiguration_Release(t *testing.T) {
	cfg, err := config.NewBuilder().Build()
	require.NoError(t, err)

	ctx := &GitVersionContext{FullConfiguration: cfg}

	ec, err := ctx.GetEffectiveConfiguration("release/1.0")
	require.NoError(t, err)
	require.True(t, ec.IsReleaseBranch)
	require.False(t, ec.IsMainline)
}

func stringPtr(s string) *string { return &s }

func TestGetEffectiveConfiguration_NoMatch(t *testing.T) {
	cfg := &config.Config{
		Branches: map[string]*config.BranchConfig{
			"main": {Regex: stringPtr(`^main$`)},
		},
	}

	ctx := &GitVersionContext{FullConfiguration: cfg}

	_, err := ctx.GetEffectiveConfiguration("nonexistent")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no branch configuration matches")
}
