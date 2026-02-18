package semver

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse_ValidVersions(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		tagPrefix string
		want      SemanticVersion
	}{
		{
			"major only",
			"1", "",
			SemanticVersion{Major: 1},
		},
		{
			"major.minor",
			"1.2", "",
			SemanticVersion{Major: 1, Minor: 2},
		},
		{
			"major.minor.patch",
			"1.2.3", "",
			SemanticVersion{Major: 1, Minor: 2, Patch: 3},
		},
		{
			"with pre-release name and number",
			"1.2.3-beta.4", "",
			SemanticVersion{
				Major:         1,
				Minor:         2,
				Patch:         3,
				PreReleaseTag: PreReleaseTag{Name: "beta", Number: int64Ptr(4)},
			},
		},
		{
			"with pre-release name only",
			"1.2.3-alpha", "",
			SemanticVersion{
				Major:         1,
				Minor:         2,
				Patch:         3,
				PreReleaseTag: PreReleaseTag{Name: "alpha"},
			},
		},
		{
			"with pre-release number only",
			"1.2.3-4", "",
			SemanticVersion{
				Major:         1,
				Minor:         2,
				Patch:         3,
				PreReleaseTag: PreReleaseTag{Number: int64Ptr(4)},
			},
		},
		{
			"with build metadata",
			"1.2.3+5", "",
			SemanticVersion{
				Major:         1,
				Minor:         2,
				Patch:         3,
				BuildMetaData: BuildMetaData{CommitsSinceTag: int64Ptr(5)},
			},
		},
		{
			"full version",
			"1.2.3-beta.4+5", "",
			SemanticVersion{
				Major:         1,
				Minor:         2,
				Patch:         3,
				PreReleaseTag: PreReleaseTag{Name: "beta", Number: int64Ptr(4)},
				BuildMetaData: BuildMetaData{CommitsSinceTag: int64Ptr(5)},
			},
		},
		{
			"four-part version",
			"1.2.3.4", "",
			SemanticVersion{Major: 1, Minor: 2, Patch: 3},
		},
		{
			"with v prefix",
			"v1.2.3", "[vV]",
			SemanticVersion{Major: 1, Minor: 2, Patch: 3},
		},
		{
			"with V prefix",
			"V1.2.3-rc.1", "[vV]",
			SemanticVersion{
				Major:         1,
				Minor:         2,
				Patch:         3,
				PreReleaseTag: PreReleaseTag{Name: "rc", Number: int64Ptr(1)},
			},
		},
		{
			"zero version",
			"0.0.0", "",
			SemanticVersion{},
		},
		{
			"pre-release with dots",
			"1.0.0-alpha.beta.1", "",
			SemanticVersion{
				Major:         1,
				PreReleaseTag: PreReleaseTag{Name: "alpha.beta", Number: int64Ptr(1)},
			},
		},
		{
			"non-numeric build metadata ignored",
			"1.2.3+sha.abc123", "",
			SemanticVersion{Major: 1, Minor: 2, Patch: 3},
		},
		{
			"pre-release zero",
			"1.0.0-beta.0", "",
			SemanticVersion{
				Major:         1,
				PreReleaseTag: PreReleaseTag{Name: "beta", Number: int64Ptr(0)},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input, tt.tagPrefix)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestParse_InvalidVersions(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		tagPrefix string
	}{
		{"empty string", "", ""},
		{"not a version", "hello", ""},
		{"missing prefix", "1.2.3", "[vV]"},
		{"wrong prefix", "x1.2.3", "[vV]"},
		{"invalid prefix regex", "v1.2.3", "[invalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input, tt.tagPrefix)
			require.Error(t, err)
		})
	}
}

func TestTryParse(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		v, ok := TryParse("1.2.3", "")
		require.True(t, ok)
		require.Equal(t, int64(1), v.Major)
		require.Equal(t, int64(2), v.Minor)
		require.Equal(t, int64(3), v.Patch)
	})

	t.Run("valid with prefix", func(t *testing.T) {
		v, ok := TryParse("v1.0.0", "[vV]")
		require.True(t, ok)
		require.Equal(t, int64(1), v.Major)
	})

	t.Run("invalid", func(t *testing.T) {
		_, ok := TryParse("not-a-version", "")
		require.False(t, ok)
	})

	t.Run("missing prefix", func(t *testing.T) {
		_, ok := TryParse("1.2.3", "[vV]")
		require.False(t, ok)
	})
}

func TestSemanticVersion_CompareTo(t *testing.T) {
	tests := []struct {
		name string
		a    SemanticVersion
		b    SemanticVersion
		want int
	}{
		{
			"equal",
			SemanticVersion{Major: 1, Minor: 2, Patch: 3},
			SemanticVersion{Major: 1, Minor: 2, Patch: 3},
			0,
		},
		{
			"major greater",
			SemanticVersion{Major: 2},
			SemanticVersion{Major: 1},
			1,
		},
		{
			"major less",
			SemanticVersion{Major: 1},
			SemanticVersion{Major: 2},
			-1,
		},
		{
			"minor greater",
			SemanticVersion{Major: 1, Minor: 3},
			SemanticVersion{Major: 1, Minor: 2},
			1,
		},
		{
			"minor less",
			SemanticVersion{Major: 1, Minor: 2},
			SemanticVersion{Major: 1, Minor: 3},
			-1,
		},
		{
			"patch greater",
			SemanticVersion{Major: 1, Minor: 2, Patch: 4},
			SemanticVersion{Major: 1, Minor: 2, Patch: 3},
			1,
		},
		{
			"patch less",
			SemanticVersion{Major: 1, Minor: 2, Patch: 3},
			SemanticVersion{Major: 1, Minor: 2, Patch: 4},
			-1,
		},
		{
			"stable > pre-release",
			SemanticVersion{Major: 1, Minor: 2, Patch: 3},
			SemanticVersion{
				Major:         1,
				Minor:         2,
				Patch:         3,
				PreReleaseTag: PreReleaseTag{Name: "beta", Number: int64Ptr(1)},
			},
			1,
		},
		{
			"pre-release < stable",
			SemanticVersion{
				Major:         1,
				Minor:         2,
				Patch:         3,
				PreReleaseTag: PreReleaseTag{Name: "beta", Number: int64Ptr(1)},
			},
			SemanticVersion{Major: 1, Minor: 2, Patch: 3},
			-1,
		},
		{
			"build metadata ignored",
			SemanticVersion{
				Major:         1,
				Minor:         2,
				Patch:         3,
				BuildMetaData: BuildMetaData{CommitsSinceTag: int64Ptr(5)},
			},
			SemanticVersion{
				Major:         1,
				Minor:         2,
				Patch:         3,
				BuildMetaData: BuildMetaData{CommitsSinceTag: int64Ptr(10)},
			},
			0,
		},
		{
			"pre-release name ordering",
			SemanticVersion{
				Major:         1,
				PreReleaseTag: PreReleaseTag{Name: "alpha", Number: int64Ptr(1)},
			},
			SemanticVersion{
				Major:         1,
				PreReleaseTag: PreReleaseTag{Name: "beta", Number: int64Ptr(1)},
			},
			-1,
		},
		{
			"pre-release number ordering",
			SemanticVersion{
				Major:         1,
				PreReleaseTag: PreReleaseTag{Name: "beta", Number: int64Ptr(2)},
			},
			SemanticVersion{
				Major:         1,
				PreReleaseTag: PreReleaseTag{Name: "beta", Number: int64Ptr(5)},
			},
			-1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.a.CompareTo(tt.b)
			switch {
			case tt.want < 0:
				require.Less(t, result, 0)
			case tt.want > 0:
				require.Greater(t, result, 0)
			default:
				require.Equal(t, 0, result)
			}
		})
	}
}

func TestSemanticVersion_IncrementField(t *testing.T) {
	tests := []struct {
		name  string
		ver   SemanticVersion
		field VersionField
		want  SemanticVersion
	}{
		{
			"major",
			SemanticVersion{Major: 1, Minor: 2, Patch: 3},
			VersionFieldMajor,
			SemanticVersion{Major: 2},
		},
		{
			"minor",
			SemanticVersion{Major: 1, Minor: 2, Patch: 3},
			VersionFieldMinor,
			SemanticVersion{Major: 1, Minor: 3},
		},
		{
			"patch",
			SemanticVersion{Major: 1, Minor: 2, Patch: 3},
			VersionFieldPatch,
			SemanticVersion{Major: 1, Minor: 2, Patch: 4},
		},
		{
			"none returns unchanged",
			SemanticVersion{
				Major:         1,
				Minor:         2,
				Patch:         3,
				PreReleaseTag: PreReleaseTag{Name: "beta", Number: int64Ptr(1)},
			},
			VersionFieldNone,
			SemanticVersion{
				Major:         1,
				Minor:         2,
				Patch:         3,
				PreReleaseTag: PreReleaseTag{Name: "beta", Number: int64Ptr(1)},
			},
		},
		{
			"major strips pre-release",
			SemanticVersion{
				Major:         1,
				Minor:         2,
				Patch:         3,
				PreReleaseTag: PreReleaseTag{Name: "beta", Number: int64Ptr(4)},
			},
			VersionFieldMajor,
			SemanticVersion{Major: 2},
		},
		{
			"minor strips pre-release",
			SemanticVersion{
				Major:         1,
				Minor:         2,
				Patch:         3,
				PreReleaseTag: PreReleaseTag{Name: "beta", Number: int64Ptr(4)},
			},
			VersionFieldMinor,
			SemanticVersion{Major: 1, Minor: 3},
		},
		{
			"patch strips pre-release",
			SemanticVersion{
				Major:         1,
				Minor:         2,
				Patch:         3,
				PreReleaseTag: PreReleaseTag{Name: "beta", Number: int64Ptr(4)},
			},
			VersionFieldPatch,
			SemanticVersion{Major: 1, Minor: 2, Patch: 4},
		},
		{
			"major strips build metadata",
			SemanticVersion{
				Major:         1,
				BuildMetaData: BuildMetaData{CommitsSinceTag: int64Ptr(5)},
			},
			VersionFieldMajor,
			SemanticVersion{Major: 2},
		},
		{
			"from zero",
			SemanticVersion{},
			VersionFieldPatch,
			SemanticVersion{Patch: 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ver.IncrementField(tt.field)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestSemanticVersion_IncrementPreRelease(t *testing.T) {
	t.Run("bumps number", func(t *testing.T) {
		v := SemanticVersion{
			Major:         1,
			Minor:         2,
			Patch:         3,
			PreReleaseTag: PreReleaseTag{Name: "beta", Number: int64Ptr(3)},
		}
		got := v.IncrementPreRelease()
		require.Equal(t, int64(1), got.Major)
		require.Equal(t, int64(2), got.Minor)
		require.Equal(t, int64(3), got.Patch)
		require.Equal(t, "beta", got.PreReleaseTag.Name)
		require.Equal(t, int64(4), *got.PreReleaseTag.Number)
	})

	t.Run("from zero", func(t *testing.T) {
		v := SemanticVersion{
			Major:         1,
			PreReleaseTag: PreReleaseTag{Name: "alpha", Number: int64Ptr(0)},
		}
		got := v.IncrementPreRelease()
		require.Equal(t, int64(1), *got.PreReleaseTag.Number)
	})

	t.Run("preserves build metadata", func(t *testing.T) {
		v := SemanticVersion{
			Major:         1,
			PreReleaseTag: PreReleaseTag{Name: "beta", Number: int64Ptr(1)},
			BuildMetaData: BuildMetaData{CommitsSinceTag: int64Ptr(5), Branch: "main"},
		}
		got := v.IncrementPreRelease()
		require.Equal(t, int64(2), *got.PreReleaseTag.Number)
		require.Equal(t, "main", got.BuildMetaData.Branch)
	})

	t.Run("panics without number", func(t *testing.T) {
		v := SemanticVersion{Major: 1, PreReleaseTag: PreReleaseTag{Name: "beta"}}
		require.Panics(t, func() { v.IncrementPreRelease() })
	})

	t.Run("panics without pre-release", func(t *testing.T) {
		v := SemanticVersion{Major: 1, Minor: 2, Patch: 3}
		require.Panics(t, func() { v.IncrementPreRelease() })
	})
}

func TestSemanticVersion_WithPreReleaseTag(t *testing.T) {
	original := SemanticVersion{Major: 1, Minor: 2, Patch: 3}
	tag := PreReleaseTag{Name: "beta", Number: int64Ptr(1)}
	result := original.WithPreReleaseTag(tag)

	require.Equal(t, int64(1), result.Major)
	require.Equal(t, "beta", result.PreReleaseTag.Name)
	// Original unchanged
	require.False(t, original.PreReleaseTag.HasTag())
}

func TestSemanticVersion_WithBuildMetaData(t *testing.T) {
	original := SemanticVersion{Major: 1, Minor: 2, Patch: 3}
	meta := BuildMetaData{CommitsSinceTag: int64Ptr(5), Branch: "main"}
	result := original.WithBuildMetaData(meta)

	require.Equal(t, int64(1), result.Major)
	require.Equal(t, "main", result.BuildMetaData.Branch)
	// Original unchanged
	require.Equal(t, "", original.BuildMetaData.Branch)
}

func TestSemanticVersion_SemVer(t *testing.T) {
	tests := []struct {
		name string
		ver  SemanticVersion
		want string
	}{
		{
			"basic",
			SemanticVersion{Major: 1, Minor: 2, Patch: 3},
			"1.2.3",
		},
		{
			"with pre-release",
			SemanticVersion{
				Major:         1,
				Minor:         2,
				Patch:         3,
				PreReleaseTag: PreReleaseTag{Name: "beta", Number: int64Ptr(4)},
			},
			"1.2.3-beta.4",
		},
		{
			"zero version",
			SemanticVersion{},
			"0.0.0",
		},
		{
			"ignores build metadata",
			SemanticVersion{
				Major:         1,
				Minor:         2,
				Patch:         3,
				BuildMetaData: BuildMetaData{CommitsSinceTag: int64Ptr(5)},
			},
			"1.2.3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.ver.SemVer())
		})
	}
}

func TestSemanticVersion_FullSemVer(t *testing.T) {
	tests := []struct {
		name string
		ver  SemanticVersion
		want string
	}{
		{
			"without metadata",
			SemanticVersion{Major: 1, Minor: 2, Patch: 3},
			"1.2.3",
		},
		{
			"with metadata",
			SemanticVersion{
				Major:         1,
				Minor:         2,
				Patch:         3,
				BuildMetaData: BuildMetaData{CommitsSinceTag: int64Ptr(5)},
			},
			"1.2.3+5",
		},
		{
			"full version",
			SemanticVersion{
				Major:         1,
				Minor:         2,
				Patch:         3,
				PreReleaseTag: PreReleaseTag{Name: "beta", Number: int64Ptr(4)},
				BuildMetaData: BuildMetaData{CommitsSinceTag: int64Ptr(5)},
			},
			"1.2.3-beta.4+5",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.ver.FullSemVer())
		})
	}
}

func TestSemanticVersion_LegacySemVer(t *testing.T) {
	tests := []struct {
		name string
		ver  SemanticVersion
		want string
	}{
		{
			"basic",
			SemanticVersion{Major: 1, Minor: 2, Patch: 3},
			"1.2.3",
		},
		{
			"with pre-release",
			SemanticVersion{
				Major:         1,
				Minor:         2,
				Patch:         3,
				PreReleaseTag: PreReleaseTag{Name: "beta", Number: int64Ptr(4)},
			},
			"1.2.3-beta4",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.ver.LegacySemVer())
		})
	}
}

func TestSemanticVersion_LegacySemVerPadded(t *testing.T) {
	tests := []struct {
		name string
		ver  SemanticVersion
		pad  int
		want string
	}{
		{
			"basic",
			SemanticVersion{Major: 1, Minor: 2, Patch: 3},
			4,
			"1.2.3",
		},
		{
			"with pre-release",
			SemanticVersion{
				Major:         1,
				Minor:         2,
				Patch:         3,
				PreReleaseTag: PreReleaseTag{Name: "beta", Number: int64Ptr(4)},
			},
			4,
			"1.2.3-beta0004",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.ver.LegacySemVerPadded(tt.pad))
		})
	}
}

func TestSemanticVersion_InformationalVersion(t *testing.T) {
	tests := []struct {
		name string
		ver  SemanticVersion
		want string
	}{
		{
			"basic",
			SemanticVersion{Major: 1, Minor: 2, Patch: 3},
			"1.2.3",
		},
		{
			"full version",
			SemanticVersion{
				Major:         1,
				Minor:         2,
				Patch:         3,
				PreReleaseTag: PreReleaseTag{Name: "beta", Number: int64Ptr(4)},
				BuildMetaData: BuildMetaData{
					CommitsSinceTag: int64Ptr(5),
					Branch:          "main",
					Sha:             "abc1234",
				},
			},
			"1.2.3-beta.4+5.Branch.main.Sha.abc1234",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.ver.InformationalVersion())
		})
	}
}

func TestParse_RoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"basic", "1.2.3"},
		{"with pre-release", "1.2.3-beta.4"},
		{"with build", "1.2.3+5"},
		{"full", "1.2.3-beta.4+5"},
		{"zero", "0.0.0"},
		{"large", "100.200.300"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := Parse(tt.input, "")
			require.NoError(t, err)
			// SemVer() should match the input (without build metadata)
			switch tt.input {
			case "1.2.3+5":
				require.Equal(t, "1.2.3", v.SemVer())
				require.Equal(t, "1.2.3+5", v.FullSemVer())
			case "1.2.3-beta.4+5":
				require.Equal(t, "1.2.3-beta.4", v.SemVer())
				require.Equal(t, "1.2.3-beta.4+5", v.FullSemVer())
			default:
				require.Equal(t, tt.input, v.SemVer())
			}
		})
	}
}
