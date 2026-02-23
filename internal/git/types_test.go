package git

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestObjectID_ShortSha(t *testing.T) {
	tests := []struct {
		name   string
		sha    string
		n      int
		expect string
	}{
		{"normal", "abc1234567890", 7, "abc1234"},
		{"full length", "abc", 10, "abc"},
		{"exact length", "abc", 3, "abc"},
		{"zero", "abc1234", 0, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := ObjectID{Sha: tt.sha}
			require.Equal(t, tt.expect, id.ShortSha(tt.n))
		})
	}
}

func TestObjectID_String(t *testing.T) {
	id := ObjectID{Sha: "abc123"}
	require.Equal(t, "abc123", id.String())
}

func TestCommit_IsMerge(t *testing.T) {
	tests := []struct {
		name    string
		parents []string
		expect  bool
	}{
		{"no parents (root)", nil, false},
		{"one parent", []string{"abc"}, false},
		{"two parents (merge)", []string{"abc", "def"}, true},
		{"three parents (octopus)", []string{"a", "b", "c"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Commit{Parents: tt.parents}
			require.Equal(t, tt.expect, c.IsMerge())
		})
	}
}

func TestCommit_ShortSha(t *testing.T) {
	tests := []struct {
		name   string
		sha    string
		expect string
	}{
		{"normal", "abc1234567890def", "abc1234"},
		{"short sha", "abc", "abc"},
		{"exactly 7", "abc1234", "abc1234"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Commit{Sha: tt.sha}
			require.Equal(t, tt.expect, c.ShortSha())
		})
	}
}

func TestCommit_IsEmpty(t *testing.T) {
	require.True(t, Commit{}.IsEmpty())
	require.False(t, Commit{Sha: "abc"}.IsEmpty())
}

func TestNewReferenceName_LocalBranch(t *testing.T) {
	ref := NewReferenceName("refs/heads/main")
	require.Equal(t, "refs/heads/main", ref.Canonical)
	require.Equal(t, "main", ref.Friendly)
	require.Equal(t, "main", ref.WithoutRemote)
	require.True(t, ref.IsBranch())
	require.False(t, ref.IsRemoteBranch())
	require.False(t, ref.IsTag())
}

func TestNewReferenceName_NestedLocalBranch(t *testing.T) {
	ref := NewReferenceName("refs/heads/feature/auth")
	require.Equal(t, "feature/auth", ref.Friendly)
	require.Equal(t, "feature/auth", ref.WithoutRemote)
	require.True(t, ref.IsBranch())
}

func TestNewReferenceName_RemoteBranch(t *testing.T) {
	ref := NewReferenceName("refs/remotes/origin/main")
	require.Equal(t, "refs/remotes/origin/main", ref.Canonical)
	require.Equal(t, "origin/main", ref.Friendly)
	require.Equal(t, "main", ref.WithoutRemote)
	require.False(t, ref.IsBranch())
	require.True(t, ref.IsRemoteBranch())
	require.False(t, ref.IsTag())
}

func TestNewReferenceName_RemoteBranchNested(t *testing.T) {
	ref := NewReferenceName("refs/remotes/origin/feature/auth")
	require.Equal(t, "origin/feature/auth", ref.Friendly)
	require.Equal(t, "feature/auth", ref.WithoutRemote)
}

func TestNewReferenceName_RemoteBranchNoSlash(t *testing.T) {
	ref := NewReferenceName("refs/remotes/upstream")
	require.Equal(t, "upstream", ref.Friendly)
	require.Equal(t, "upstream", ref.WithoutRemote)
}

func TestNewReferenceName_Tag(t *testing.T) {
	ref := NewReferenceName("refs/tags/v1.0.0")
	require.Equal(t, "refs/tags/v1.0.0", ref.Canonical)
	require.Equal(t, "v1.0.0", ref.Friendly)
	require.Equal(t, "v1.0.0", ref.WithoutRemote)
	require.False(t, ref.IsBranch())
	require.False(t, ref.IsRemoteBranch())
	require.True(t, ref.IsTag())
}

func TestNewReferenceName_Unknown(t *testing.T) {
	ref := NewReferenceName("refs/stash")
	require.Equal(t, "refs/stash", ref.Canonical)
	require.Equal(t, "refs/stash", ref.Friendly)
	require.Equal(t, "refs/stash", ref.WithoutRemote)
	require.False(t, ref.IsBranch())
	require.False(t, ref.IsRemoteBranch())
	require.False(t, ref.IsTag())
}

func TestNewBranchReferenceName(t *testing.T) {
	ref := NewBranchReferenceName("develop")
	require.Equal(t, "refs/heads/develop", ref.Canonical)
	require.Equal(t, "develop", ref.Friendly)
	require.Equal(t, "develop", ref.WithoutRemote)
	require.True(t, ref.IsBranch())
}

func TestBranch_FriendlyName(t *testing.T) {
	b := Branch{Name: NewBranchReferenceName("feature/login")}
	require.Equal(t, "feature/login", b.FriendlyName())
}
