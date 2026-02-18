package semver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuildMetaData_String(t *testing.T) {
	tests := []struct {
		name string
		meta BuildMetaData
		want string
	}{
		{"nil commits", BuildMetaData{}, ""},
		{"zero commits", BuildMetaData{CommitsSinceTag: int64Ptr(0)}, "0"},
		{"five commits", BuildMetaData{CommitsSinceTag: int64Ptr(5)}, "5"},
		{"large count", BuildMetaData{CommitsSinceTag: int64Ptr(12345)}, "12345"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.meta.String())
		})
	}
}

func TestBuildMetaData_FullString(t *testing.T) {
	tests := []struct {
		name string
		meta BuildMetaData
		want string
	}{
		{"empty", BuildMetaData{}, ""},
		{
			"commits only",
			BuildMetaData{CommitsSinceTag: int64Ptr(5)},
			"5",
		},
		{
			"commits and branch",
			BuildMetaData{CommitsSinceTag: int64Ptr(5), Branch: "main"},
			"5.Branch.main",
		},
		{
			"commits branch and sha",
			BuildMetaData{
				CommitsSinceTag: int64Ptr(5),
				Branch:          "main",
				Sha:             "abc1234",
			},
			"5.Branch.main.Sha.abc1234",
		},
		{
			"branch only",
			BuildMetaData{Branch: "feature/auth"},
			"Branch.feature/auth",
		},
		{
			"sha only",
			BuildMetaData{Sha: "abc1234def5678"},
			"Sha.abc1234def5678",
		},
		{
			"all fields",
			BuildMetaData{
				CommitsSinceTag:           int64Ptr(3),
				Branch:                    "develop",
				Sha:                       "deadbeef",
				ShortSha:                  "deadbee",
				VersionSourceSha:          "cafebabe",
				CommitDate:                time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
				CommitsSinceVersionSource: 3,
				UncommittedChanges:        2,
			},
			"3.Branch.develop.Sha.deadbeef",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.meta.FullString())
		})
	}
}

func TestBuildMetaData_Padded(t *testing.T) {
	tests := []struct {
		name string
		meta BuildMetaData
		pad  int
		want string
	}{
		{"nil commits", BuildMetaData{}, 4, ""},
		{"zero padded", BuildMetaData{CommitsSinceTag: int64Ptr(0)}, 4, "0000"},
		{"five padded", BuildMetaData{CommitsSinceTag: int64Ptr(5)}, 4, "0005"},
		{"large number", BuildMetaData{CommitsSinceTag: int64Ptr(12345)}, 4, "12345"},
		{"pad 6", BuildMetaData{CommitsSinceTag: int64Ptr(5)}, 6, "000005"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.meta.Padded(tt.pad))
		})
	}
}
