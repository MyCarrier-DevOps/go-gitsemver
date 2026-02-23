package git

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseMergeMessage_DefaultFormat(t *testing.T) {
	tests := []struct {
		name         string
		message      string
		mergedBranch string
		targetBranch string
	}{
		{
			"branch only",
			"Merge branch 'release/1.2.0'",
			"release/1.2.0", "",
		},
		{
			"branch into target",
			"Merge branch 'release/1.2.0' into main",
			"release/1.2.0", "main",
		},
		{
			"tag merge",
			"Merge tag 'v1.0.0'",
			"v1.0.0", "",
		},
		{
			"tag into target",
			"Merge tag 'v2.0.0' into develop",
			"v2.0.0", "develop",
		},
		{
			"case insensitive",
			"merge branch 'feature/auth'",
			"feature/auth", "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := ParseMergeMessage(tt.message, nil)
			require.False(t, msg.IsEmpty())
			require.Equal(t, "Default", msg.FormatName)
			require.Equal(t, tt.mergedBranch, msg.MergedBranch)
			require.Equal(t, tt.targetBranch, msg.TargetBranch)
			require.False(t, msg.IsMergedPullRequest)
		})
	}
}

func TestParseMergeMessage_SmartGit(t *testing.T) {
	tests := []struct {
		name         string
		message      string
		mergedBranch string
		targetBranch string
	}{
		{
			"finish only",
			"Finish release/1.2.0",
			"release/1.2.0", "",
		},
		{
			"finish into target",
			"Finish release/1.2.0 into main",
			"release/1.2.0", "main",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := ParseMergeMessage(tt.message, nil)
			require.Equal(t, "SmartGit", msg.FormatName)
			require.Equal(t, tt.mergedBranch, msg.MergedBranch)
			require.Equal(t, tt.targetBranch, msg.TargetBranch)
		})
	}
}

func TestParseMergeMessage_BitBucketPull(t *testing.T) {
	msg := ParseMergeMessage(
		"Merge pull request #123 from myteam/myrepo from release/1.2.0 to main", nil,
	)
	require.Equal(t, "BitBucketPull", msg.FormatName)
	require.Equal(t, "release/1.2.0", msg.MergedBranch)
	require.Equal(t, "main", msg.TargetBranch)
	require.Equal(t, 123, msg.PullRequestNumber)
	require.True(t, msg.IsMergedPullRequest)
}

func TestParseMergeMessage_BitBucketPullv7(t *testing.T) {
	message := "Pull request #456: Feature X\n\nMerge in myproject from feature/x to main"
	msg := ParseMergeMessage(message, nil)
	require.Equal(t, "BitBucketPullv7", msg.FormatName)
	require.Equal(t, "feature/x", msg.MergedBranch)
	require.Equal(t, "main", msg.TargetBranch)
	require.Equal(t, 456, msg.PullRequestNumber)
	require.True(t, msg.IsMergedPullRequest)
}

func TestParseMergeMessage_GitHubPull(t *testing.T) {
	tests := []struct {
		name         string
		message      string
		prNum        int
		mergedBranch string
		targetBranch string
	}{
		{
			"from",
			"Merge pull request #42 from user/release/1.2.0",
			42, "user/release/1.2.0", "",
		},
		{
			"in",
			"Merge pull request #99 in release/1.2.0 into main",
			99, "release/1.2.0", "main",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := ParseMergeMessage(tt.message, nil)
			require.Equal(t, "GitHubPull", msg.FormatName)
			require.Equal(t, tt.mergedBranch, msg.MergedBranch)
			require.Equal(t, tt.targetBranch, msg.TargetBranch)
			require.Equal(t, tt.prNum, msg.PullRequestNumber)
			require.True(t, msg.IsMergedPullRequest)
		})
	}
}

func TestParseMergeMessage_RemoteTracking(t *testing.T) {
	tests := []struct {
		name         string
		message      string
		mergedBranch string
		targetBranch string
	}{
		{
			"branch only",
			"Merge remote-tracking branch 'origin/release/1.2.0'",
			"origin/release/1.2.0", "",
		},
		{
			"into target",
			"Merge remote-tracking branch 'origin/release/1.2.0' into develop",
			"origin/release/1.2.0", "develop",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := ParseMergeMessage(tt.message, nil)
			require.Equal(t, "RemoteTracking", msg.FormatName)
			require.Equal(t, tt.mergedBranch, msg.MergedBranch)
			require.Equal(t, tt.targetBranch, msg.TargetBranch)
		})
	}
}

func TestParseMergeMessage_GitHubSquash(t *testing.T) {
	msg := ParseMergeMessage("feat: add login page (#123)", nil)
	require.Equal(t, "GitHubSquash", msg.FormatName)
	require.Equal(t, 123, msg.PullRequestNumber)
	require.True(t, msg.IsMergedPullRequest)
	require.Equal(t, "", msg.MergedBranch) // squash doesn't include branch
}

func TestParseMergeMessage_BitBucketSquash(t *testing.T) {
	msg := ParseMergeMessage("Merged in feature/auth (pull request #42)", nil)
	require.Equal(t, "BitBucketSquash", msg.FormatName)
	require.Equal(t, "feature/auth", msg.MergedBranch)
	require.Equal(t, 42, msg.PullRequestNumber)
	require.True(t, msg.IsMergedPullRequest)
}

func TestParseMergeMessage_CustomFormat(t *testing.T) {
	custom := map[string]string{
		"azure": `^Merged PR (?P<PullRequestNumber>\d+): .*$`,
	}
	msg := ParseMergeMessage("Merged PR 789: Fix auth bug", custom)
	require.Equal(t, "azure", msg.FormatName)
	require.Equal(t, 789, msg.PullRequestNumber)
	require.True(t, msg.IsMergedPullRequest)
}

func TestParseMergeMessage_CustomFormatTakesPriority(t *testing.T) {
	// Custom format should match before built-in Default format.
	custom := map[string]string{
		"custom": `^Merge branch '(?P<SourceBranch>[^']*)'$`,
	}
	msg := ParseMergeMessage("Merge branch 'release/1.0.0'", custom)
	require.Equal(t, "custom", msg.FormatName)
	require.Equal(t, "release/1.0.0", msg.MergedBranch)
}

func TestParseMergeMessage_InvalidCustomFormatSkipped(t *testing.T) {
	custom := map[string]string{
		"bad": "[invalid",
	}
	msg := ParseMergeMessage("Merge branch 'main'", custom)
	require.Equal(t, "Default", msg.FormatName)
}

func TestParseMergeMessage_NoMatch(t *testing.T) {
	messages := []string{
		"feat: add login page",
		"fix: resolve crash on startup",
		"Initial commit",
		"",
	}
	for _, message := range messages {
		msg := ParseMergeMessage(message, nil)
		require.True(t, msg.IsEmpty(), "expected no match for %q", message)
	}
}

func TestDefaultMergeMessageFormats(t *testing.T) {
	formats := DefaultMergeMessageFormats()
	require.Len(t, formats, 6)
}

func TestSquashMergeMessageFormats(t *testing.T) {
	formats := SquashMergeMessageFormats()
	require.Len(t, formats, 2)
}

func TestExtractVersionFromBranch(t *testing.T) {
	tests := []struct {
		name      string
		branch    string
		tagPrefix string
		version   string
		ok        bool
	}{
		{"release slash", "release/1.2.0", "", "1.2.0", true},
		{"release dash", "release-1.3.0", "", "1.3.0", true},
		{"releases slash", "releases/2.0.0", "", "2.0.0", true},
		{"major only", "release/2", "", "2.0.0", true},
		{"major.minor", "release/1.3", "", "1.3.0", true},
		{"with v prefix", "release/v1.2.0", "[vV]", "1.2.0", true},
		{"with V prefix", "release/V3.0.0", "[vV]", "3.0.0", true},
		{"no version", "feature/auth", "", "", false},
		// In practice, ExtractVersionFromBranch is only called on release branches.
		{"feature with ticket", "feature/JIRA-123", "", "123.0.0", true},
		{"empty", "", "", "", false},
		{"nested release", "hotfix/release/1.0.0", "", "1.0.0", true},
		{"support branch", "support/1.x", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ver, ok := ExtractVersionFromBranch(tt.branch, tt.tagPrefix)
			require.Equal(t, tt.ok, ok)
			if ok {
				require.Equal(t, tt.version, ver)
			}
		})
	}
}

func TestMergeMessage_IsEmpty(t *testing.T) {
	require.True(t, MergeMessage{}.IsEmpty())
	require.False(t, MergeMessage{FormatName: "Default"}.IsEmpty())
}
