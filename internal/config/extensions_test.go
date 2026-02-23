package config

import (
	"go-gitsemver/internal/semver"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetBranchConfiguration_StandardBranches(t *testing.T) {
	cfg, err := NewBuilder().Build()
	require.NoError(t, err)

	tests := []struct {
		branchName string
		wantKey    string
	}{
		{"main", "main"},
		{"master", "main"},
		{"develop", "develop"},
		{"dev", "develop"},
		{"development", "develop"},
		{"feature/auth", "feature"},
		{"features/login", "feature"},
		{"release/1.2.0", "release"},
		{"releases/1.3.0", "release"},
		{"release-1.2.0", "release"},
		{"hotfix/fix-crash", "hotfix"},
		{"hotfixes/urgent", "hotfix"},
		{"pull/123", "pull-request"},
		{"pr/456", "pull-request"},
		{"pull-requests/789", "pull-request"},
		{"support/1.x", "support"},
		{"some-random-branch", "unknown"},
		{"my-topic", "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.branchName, func(t *testing.T) {
			_, key, err := cfg.GetBranchConfiguration(tt.branchName)
			require.NoError(t, err)
			require.Equal(t, tt.wantKey, key)
		})
	}
}

func TestGetBranchConfiguration_PriorityWins(t *testing.T) {
	// Create a config where two branches match the same name
	// The higher priority should win
	cfg := &Config{
		Branches: map[string]*BranchConfig{
			"broad": {
				Regex:    stringPtr(`^feature.*`),
				Priority: intPtr(10),
			},
			"specific": {
				Regex:    stringPtr(`^feature/auth.*`),
				Priority: intPtr(90),
			},
		},
	}

	_, key, err := cfg.GetBranchConfiguration("feature/auth-module")
	require.NoError(t, err)
	require.Equal(t, "specific", key)
}

func TestGetBranchConfiguration_TiebreakByName(t *testing.T) {
	cfg := &Config{
		Branches: map[string]*BranchConfig{
			"beta": {
				Regex:    stringPtr(`.*`),
				Priority: intPtr(50),
			},
			"alpha": {
				Regex:    stringPtr(`.*`),
				Priority: intPtr(50),
			},
		},
	}

	_, key, err := cfg.GetBranchConfiguration("anything")
	require.NoError(t, err)
	require.Equal(t, "alpha", key) // alphabetically first
}

func TestGetBranchConfiguration_UnknownCatchAll(t *testing.T) {
	cfg, err := NewBuilder().Build()
	require.NoError(t, err)

	// A branch that doesn't match any specific pattern should match unknown
	_, key, err := cfg.GetBranchConfiguration("totally-random-branch-name")
	require.NoError(t, err)
	require.Equal(t, "unknown", key)
}

func TestGetBranchConfiguration_InvalidRegex(t *testing.T) {
	cfg := &Config{
		Branches: map[string]*BranchConfig{
			"bad": {
				Regex: stringPtr("[invalid"),
			},
		},
	}

	_, _, err := cfg.GetBranchConfiguration("test")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid regex")
}

func TestGetBranchConfiguration_NilRegex(t *testing.T) {
	cfg := &Config{
		Branches: map[string]*BranchConfig{
			"no-regex": {},
			"catch-all": {
				Regex:    stringPtr(".*"),
				Priority: intPtr(0),
			},
		},
	}

	_, key, err := cfg.GetBranchConfiguration("test")
	require.NoError(t, err)
	require.Equal(t, "catch-all", key)
}

func TestGetBranchConfiguration_NoMatch(t *testing.T) {
	cfg := &Config{
		Branches: map[string]*BranchConfig{
			"main": {
				Regex: stringPtr(`^main$`),
			},
		},
	}

	_, _, err := cfg.GetBranchConfiguration("develop")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no branch configuration matches")
}

func TestGetReleaseBranchConfig(t *testing.T) {
	cfg, err := NewBuilder().Build()
	require.NoError(t, err)

	releases := cfg.GetReleaseBranchConfig()
	require.Len(t, releases, 1)
	require.Contains(t, releases, "release")
	require.Equal(t, semver.IncrementStrategyNone, *releases["release"].Increment)
}

func TestGetReleaseBranchConfig_MultipleRelease(t *testing.T) {
	cfg, err := NewBuilder().Add(&Config{
		Branches: map[string]*BranchConfig{
			"hotfix": {
				IsReleaseBranch: boolPtr(true),
			},
		},
	}).Build()
	require.NoError(t, err)

	releases := cfg.GetReleaseBranchConfig()
	require.Len(t, releases, 2)
	require.Contains(t, releases, "release")
	require.Contains(t, releases, "hotfix")
}

func TestGetReleaseBranchConfig_None(t *testing.T) {
	cfg := &Config{
		Branches: map[string]*BranchConfig{
			"main": {IsReleaseBranch: boolPtr(false)},
		},
	}
	releases := cfg.GetReleaseBranchConfig()
	require.Empty(t, releases)
}

func TestIsReleaseBranch(t *testing.T) {
	cfg, err := NewBuilder().Build()
	require.NoError(t, err)

	tests := []struct {
		name       string
		branchName string
		want       bool
	}{
		{"release branch", "release/1.2.0", true},
		{"releases branch", "releases/1.3.0", true},
		{"release dash", "release-1.2.0", true},
		{"main is not release", "main", false},
		{"develop is not release", "develop", false},
		{"feature is not release", "feature/auth", false},
		{"hotfix is not release", "hotfix/fix-crash", false},
		{"unknown is not release", "some-random-branch", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, cfg.IsReleaseBranch(tt.branchName))
		})
	}
}

func TestIsReleaseBranch_NoBranches(t *testing.T) {
	cfg := &Config{Branches: map[string]*BranchConfig{}}
	require.False(t, cfg.IsReleaseBranch("release/1.0"))
}

func TestIsReleaseBranch_NilRelease(t *testing.T) {
	cfg := &Config{
		Branches: map[string]*BranchConfig{
			"release": {Regex: stringPtr(`^releases?[/-]`), IsReleaseBranch: nil},
		},
	}
	require.False(t, cfg.IsReleaseBranch("release/1.0"))
}

func TestGetBranchSpecificTag(t *testing.T) {
	tests := []struct {
		name       string
		branchName string
		tag        string
		want       string
	}{
		{"literal tag", "feature/auth", "beta", "beta"},
		{"empty tag", "main", "", ""},
		{"branch name replacement", "feature/auth", "{BranchName}", "auth"},
		{"branch name in template", "feature/my-feature", "pre-{BranchName}", "pre-my-feature"},
		{"strip feature prefix", "feature/login-page", "{BranchName}", "login-page"},
		{"strip features prefix", "features/login", "{BranchName}", "login"},
		{"strip hotfix prefix", "hotfix/fix-crash", "{BranchName}", "fix-crash"},
		{"strip release prefix", "release/1.2.0", "{BranchName}", "1-2-0"},
		{"strip pull prefix", "pull/123", "{BranchName}", "123"},
		{"strip pr prefix", "pr/456", "{BranchName}", "456"},
		{"strip support prefix", "support/1.x", "{BranchName}", "1-x"},
		{"no prefix to strip", "my-branch", "{BranchName}", "my-branch"},
		{"special chars cleaned", "feature/my_feature.v2", "{BranchName}", "my-feature-v2"},
		{"slashes cleaned", "feature/scope/sub", "{BranchName}", "scope-sub"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetBranchSpecificTag(tt.branchName, tt.tag)
			require.Equal(t, tt.want, got)
		})
	}
}
