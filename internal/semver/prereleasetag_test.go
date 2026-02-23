package semver

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func int64Ptr(n int64) *int64 {
	return &n
}

func TestPreReleaseTag_HasTag(t *testing.T) {
	tests := []struct {
		name string
		tag  PreReleaseTag
		want bool
	}{
		{"empty", PreReleaseTag{}, false},
		{"name only", PreReleaseTag{Name: "beta"}, true},
		{"number only", PreReleaseTag{Number: int64Ptr(4)}, true},
		{"name and number", PreReleaseTag{Name: "beta", Number: int64Ptr(4)}, true},
		{"zero number", PreReleaseTag{Number: int64Ptr(0)}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.tag.HasTag())
		})
	}
}

func TestPreReleaseTag_WithName(t *testing.T) {
	original := PreReleaseTag{Name: "alpha", Number: int64Ptr(1)}
	result := original.WithName("beta")

	require.Equal(t, "beta", result.Name)
	require.Equal(t, int64Ptr(1), result.Number)
	// Original unchanged (immutability)
	require.Equal(t, "alpha", original.Name)
}

func TestPreReleaseTag_WithNumber(t *testing.T) {
	original := PreReleaseTag{Name: "beta", Number: int64Ptr(1)}
	result := original.WithNumber(5)

	require.Equal(t, "beta", result.Name)
	require.Equal(t, int64(5), *result.Number)
	// Original unchanged (immutability)
	require.Equal(t, int64(1), *original.Number)
}

func TestPreReleaseTag_CompareTo(t *testing.T) {
	tests := []struct {
		name string
		a    PreReleaseTag
		b    PreReleaseTag
		want int
	}{
		{
			"both empty",
			PreReleaseTag{},
			PreReleaseTag{},
			0,
		},
		{
			"stable > pre-release",
			PreReleaseTag{},
			PreReleaseTag{Name: "beta", Number: int64Ptr(1)},
			1,
		},
		{
			"pre-release < stable",
			PreReleaseTag{Name: "beta", Number: int64Ptr(1)},
			PreReleaseTag{},
			-1,
		},
		{
			"same name same number",
			PreReleaseTag{Name: "beta", Number: int64Ptr(4)},
			PreReleaseTag{Name: "beta", Number: int64Ptr(4)},
			0,
		},
		{
			"same name lower number",
			PreReleaseTag{Name: "beta", Number: int64Ptr(3)},
			PreReleaseTag{Name: "beta", Number: int64Ptr(4)},
			-1,
		},
		{
			"same name higher number",
			PreReleaseTag{Name: "beta", Number: int64Ptr(5)},
			PreReleaseTag{Name: "beta", Number: int64Ptr(4)},
			1,
		},
		{
			"alpha < beta by name",
			PreReleaseTag{Name: "alpha", Number: int64Ptr(1)},
			PreReleaseTag{Name: "beta", Number: int64Ptr(1)},
			-1,
		},
		{
			"beta > alpha by name",
			PreReleaseTag{Name: "beta", Number: int64Ptr(1)},
			PreReleaseTag{Name: "alpha", Number: int64Ptr(1)},
			1,
		},
		{
			"case insensitive name comparison",
			PreReleaseTag{Name: "Beta", Number: int64Ptr(1)},
			PreReleaseTag{Name: "beta", Number: int64Ptr(1)},
			0,
		},
		{
			"name only vs name only",
			PreReleaseTag{Name: "alpha"},
			PreReleaseTag{Name: "beta"},
			-1,
		},
		{
			"number only comparison",
			PreReleaseTag{Number: int64Ptr(1)},
			PreReleaseTag{Number: int64Ptr(2)},
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

func TestPreReleaseTag_String(t *testing.T) {
	tests := []struct {
		name string
		tag  PreReleaseTag
		want string
	}{
		{"empty", PreReleaseTag{}, ""},
		{"name only", PreReleaseTag{Name: "beta"}, "beta"},
		{"number only", PreReleaseTag{Number: int64Ptr(4)}, "4"},
		{"name and number", PreReleaseTag{Name: "beta", Number: int64Ptr(4)}, "beta.4"},
		{"zero number", PreReleaseTag{Name: "rc", Number: int64Ptr(0)}, "rc.0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.tag.String())
		})
	}
}

func TestPreReleaseTag_Legacy(t *testing.T) {
	tests := []struct {
		name string
		tag  PreReleaseTag
		want string
	}{
		{"empty", PreReleaseTag{}, ""},
		{"name only", PreReleaseTag{Name: "beta"}, "beta"},
		{"number only", PreReleaseTag{Number: int64Ptr(4)}, "4"},
		{"name and number", PreReleaseTag{Name: "beta", Number: int64Ptr(4)}, "beta4"},
		{"zero number", PreReleaseTag{Name: "rc", Number: int64Ptr(0)}, "rc0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.tag.Legacy())
		})
	}
}

func TestPreReleaseTag_LegacyPadded(t *testing.T) {
	tests := []struct {
		name string
		tag  PreReleaseTag
		pad  int
		want string
	}{
		{"empty", PreReleaseTag{}, 4, ""},
		{"name only", PreReleaseTag{Name: "beta"}, 4, "beta"},
		{"number only", PreReleaseTag{Number: int64Ptr(4)}, 4, "0004"},
		{"name and number", PreReleaseTag{Name: "beta", Number: int64Ptr(4)}, 4, "beta0004"},
		{"larger number", PreReleaseTag{Name: "beta", Number: int64Ptr(12345)}, 4, "beta12345"},
		{"pad 6", PreReleaseTag{Name: "rc", Number: int64Ptr(1)}, 6, "rc000001"},
		{"zero number", PreReleaseTag{Name: "beta", Number: int64Ptr(0)}, 4, "beta0000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.tag.LegacyPadded(tt.pad))
		})
	}
}
