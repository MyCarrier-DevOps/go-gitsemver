// Package e2e contains end-to-end tests that exercise the full version
// calculation pipeline against real (temporary) git repositories.
//
// Each test creates a purpose-built git repo, runs the full pipeline, and
// asserts on the calculated version. This tests all layers together:
// git adapter → context → strategies → calculators → output.
package e2e

import (
	"go-gitsemver/internal/calculator"
	"go-gitsemver/internal/config"
	configctx "go-gitsemver/internal/context"
	"go-gitsemver/internal/git"
	"go-gitsemver/internal/output"
	"go-gitsemver/internal/strategy"
	"go-gitsemver/internal/testutil"
	"testing"

	"github.com/stretchr/testify/require"
)

// runPipeline executes the full version calculation pipeline against the given
// repo path, returning the computed output variables.
func runPipeline(t *testing.T, repoPath string) map[string]string {
	t.Helper()
	return runPipelineWithOpts(t, repoPath, configctx.Options{})
}

func runPipelineWithOpts(t *testing.T, repoPath string, opts configctx.Options) map[string]string {
	t.Helper()

	repo, err := git.Open(repoPath)
	require.NoError(t, err)

	cfg, err := config.NewBuilder().Build()
	require.NoError(t, err)

	store := git.NewRepositoryStore(repo)
	ctx, err := configctx.NewContext(store, repo, cfg, opts)
	require.NoError(t, err)

	ec, err := ctx.GetEffectiveConfiguration(ctx.CurrentBranch.FriendlyName())
	require.NoError(t, err)

	strategies := strategy.AllStrategies(store)
	calc := calculator.NewNextVersionCalculator(store, strategies)
	result, err := calc.Calculate(ctx, ec, false)
	require.NoError(t, err)

	return output.GetVariables(result.Version, ec)
}

func runPipelineWithConfig(t *testing.T, repoPath, configYAML string) map[string]string {
	t.Helper()

	repo, err := git.Open(repoPath)
	require.NoError(t, err)

	userCfg, err := config.LoadFromBytes([]byte(configYAML))
	require.NoError(t, err)

	cfg, err := config.NewBuilder().Add(userCfg).Build()
	require.NoError(t, err)

	store := git.NewRepositoryStore(repo)
	ctx, err := configctx.NewContext(store, repo, cfg, configctx.Options{})
	require.NoError(t, err)

	ec, err := ctx.GetEffectiveConfiguration(ctx.CurrentBranch.FriendlyName())
	require.NoError(t, err)

	strategies := strategy.AllStrategies(store)
	calc := calculator.NewNextVersionCalculator(store, strategies)
	result, err := calc.Calculate(ctx, ec, false)
	require.NoError(t, err)

	return output.GetVariables(result.Version, ec)
}

// ---------------------------------------------------------------------------
// Strategy: Fallback
// ---------------------------------------------------------------------------

func TestE2E_Fallback_NoTags(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.AddCommit("initial commit")
	repo.AddCommit("second commit")

	vars := runPipeline(t, repo.Path())

	// No tags → Fallback strategy with base version 0.1.0, branch default increment Patch.
	require.Equal(t, "0", vars["Major"])
	require.Equal(t, "1", vars["Minor"])
	require.Equal(t, "0", vars["Patch"])
}

func TestE2E_Fallback_CustomBaseVersion(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.AddCommit("initial commit")

	vars := runPipelineWithConfig(t, repo.Path(), "base-version: 1.0.0\n")

	require.Equal(t, "1", vars["Major"])
	require.Equal(t, "0", vars["Minor"])
	require.Equal(t, "0", vars["Patch"])
}

// ---------------------------------------------------------------------------
// Strategy: TaggedCommit
// ---------------------------------------------------------------------------

func TestE2E_TaggedCommit_CurrentCommitTagged(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial commit")
	repo.CreateTag("v1.0.0", sha)

	vars := runPipeline(t, repo.Path())

	// Current commit is tagged → exact version, no increment.
	require.Equal(t, "1.0.0", vars["SemVer"])
	require.Equal(t, "1", vars["Major"])
	require.Equal(t, "0", vars["Minor"])
	require.Equal(t, "0", vars["Patch"])
}

func TestE2E_TaggedCommit_CommitAfterTag(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial commit")
	repo.CreateTag("v1.0.0", sha)
	repo.AddCommit("feat: add new feature")

	vars := runPipeline(t, repo.Path())

	// One commit after v1.0.0 → incremented.
	require.Equal(t, "1", vars["Major"])
	require.Equal(t, "1", vars["CommitsSinceVersionSource"])
}

func TestE2E_TaggedCommit_MultipleTagsUseLatest(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha1 := repo.AddCommit("first release")
	repo.CreateTag("v1.0.0", sha1)
	sha2 := repo.AddCommit("second release")
	repo.CreateTag("v2.0.0", sha2)
	repo.AddCommit("after latest tag")

	vars := runPipeline(t, repo.Path())

	// Should use v2.0.0 as base (highest/latest tag).
	require.Equal(t, "2", vars["Major"])
}

func TestE2E_TaggedCommit_AnnotatedTag(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("release commit")
	repo.CreateAnnotatedTag("v3.0.0", sha, "Release 3.0.0")

	vars := runPipeline(t, repo.Path())

	require.Equal(t, "3.0.0", vars["SemVer"])
}

// ---------------------------------------------------------------------------
// Strategy: ConfigNextVersion
// ---------------------------------------------------------------------------

func TestE2E_ConfigNextVersion(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	repo.AddCommit("initial commit")

	vars := runPipelineWithConfig(t, repo.Path(), "next-version: 5.0.0\n")

	require.Equal(t, "5", vars["Major"])
	require.Equal(t, "0", vars["Minor"])
	require.Equal(t, "0", vars["Patch"])
}

func TestE2E_ConfigNextVersion_IgnoredWhenTagged(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("release commit")
	repo.CreateTag("v2.0.0", sha)

	vars := runPipelineWithConfig(t, repo.Path(), "next-version: 5.0.0\n")

	// next-version is ignored when current commit is tagged.
	require.Equal(t, "2.0.0", vars["SemVer"])
}

// ---------------------------------------------------------------------------
// Strategy: VersionInBranchName
// ---------------------------------------------------------------------------

func TestE2E_BranchName_ReleaseBranch(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial on main")
	repo.CreateBranch("release/3.0.0", sha)
	repo.Checkout("release/3.0.0")
	repo.AddCommit("release prep")

	vars := runPipeline(t, repo.Path())

	// Release branch version extracted from branch name.
	require.Equal(t, "3", vars["Major"])
	require.Equal(t, "0", vars["Minor"])
	require.Equal(t, "0", vars["Patch"])
}

// ---------------------------------------------------------------------------
// Strategy: MergeMessage
// ---------------------------------------------------------------------------

func TestE2E_MergeMessage_MergeFromRelease(t *testing.T) {
	repo := testutil.NewTestRepo(t)

	// Create a main branch with initial commit.
	mainSha := repo.AddCommit("initial on main")
	repo.CreateTag("v1.0.0", mainSha)

	// Create release branch.
	repo.CreateBranch("release/2.0.0", mainSha)
	repo.Checkout("release/2.0.0")
	releaseSha := repo.AddCommit("release work")

	// Back to master (go-git default) and merge.
	repo.Checkout("master")
	repo.MergeCommit("Merge branch 'release/2.0.0' into master", releaseSha)

	vars := runPipeline(t, repo.Path())

	// MergeMessage strategy should pick up version 2.0.0 from merge message.
	require.Equal(t, "2", vars["Major"])
}

// ---------------------------------------------------------------------------
// Conventional Commits
// ---------------------------------------------------------------------------

func TestE2E_ConventionalCommits_FeatBumpsMinor(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial")
	repo.CreateTag("v1.0.0", sha)
	repo.AddCommit("feat: add user authentication")

	vars := runPipelineWithConfig(t, repo.Path(), "commit-message-convention: ConventionalCommits\n")

	require.Equal(t, "1", vars["Major"])
	require.Equal(t, "1", vars["Minor"])
	require.Equal(t, "0", vars["Patch"])
}

func TestE2E_ConventionalCommits_FixBumpsPatch(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial")
	repo.CreateTag("v1.0.0", sha)
	repo.AddCommit("fix: resolve null pointer")

	vars := runPipelineWithConfig(t, repo.Path(), "commit-message-convention: ConventionalCommits\n")

	require.Equal(t, "1", vars["Major"])
	require.Equal(t, "0", vars["Minor"])
	require.Equal(t, "1", vars["Patch"])
}

func TestE2E_ConventionalCommits_BreakingBumpsMajor(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial")
	repo.CreateTag("v1.0.0", sha)
	repo.AddCommit("feat!: remove legacy API")

	vars := runPipelineWithConfig(t, repo.Path(), "commit-message-convention: ConventionalCommits\n")

	require.Equal(t, "2", vars["Major"])
	require.Equal(t, "0", vars["Minor"])
	require.Equal(t, "0", vars["Patch"])
}

func TestE2E_ConventionalCommits_BreakingFooter(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial")
	repo.CreateTag("v1.0.0", sha)
	repo.AddCommit("feat: change API\n\nBREAKING CHANGE: removed old endpoint")

	vars := runPipelineWithConfig(t, repo.Path(), "commit-message-convention: ConventionalCommits\n")

	require.Equal(t, "2", vars["Major"])
}

// ---------------------------------------------------------------------------
// Bump Directives
// ---------------------------------------------------------------------------

func TestE2E_BumpDirective_Major(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial")
	repo.CreateTag("v1.0.0", sha)
	repo.AddCommit("release changes +semver: major")

	vars := runPipeline(t, repo.Path())

	require.Equal(t, "2", vars["Major"])
}

func TestE2E_BumpDirective_Skip(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial")
	repo.CreateTag("v1.0.0", sha)
	repo.AddCommit("docs: update readme +semver: skip")

	vars := runPipeline(t, repo.Path())

	// Skip still applies branch default increment (Patch for main).
	require.Equal(t, "1", vars["Major"])
	require.Equal(t, "0", vars["Minor"])
	require.Equal(t, "1", vars["Patch"])
}

// ---------------------------------------------------------------------------
// Versioning Modes
// ---------------------------------------------------------------------------

func TestE2E_ContinuousDeployment_PromotesCommits(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial")
	repo.CreateTag("v1.0.0", sha)
	repo.AddCommit("second commit")
	repo.AddCommit("third commit")

	configYAML := `
mode: ContinuousDeployment
continuous-delivery-fallback-tag: ci
`
	vars := runPipelineWithConfig(t, repo.Path(), configYAML)

	// CD mode: commits-since-tag promoted to pre-release.
	require.Contains(t, vars["SemVer"], "ci.")
	require.Equal(t, "2", vars["CommitsSinceVersionSource"])
}

func TestE2E_Mainline_AggregateIncrement(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial")
	repo.CreateTag("v1.0.0", sha)
	repo.AddCommit("fix: bug 1")
	repo.AddCommit("feat: new feature")

	configYAML := `
mode: Mainline
commit-message-convention: ConventionalCommits
`
	vars := runPipelineWithConfig(t, repo.Path(), configYAML)

	// Mainline: highest increment (feat→Minor) applied once: 1.0.0 → 1.1.0.
	require.Equal(t, "1", vars["Major"])
	require.Equal(t, "1", vars["Minor"])
	require.Equal(t, "0", vars["Patch"])
}

func TestE2E_Mainline_EachCommit(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial")
	repo.CreateTag("v1.0.0", sha)
	repo.AddCommit("fix: bug 1")
	repo.AddCommit("fix: bug 2")
	repo.AddCommit("feat: new feature")
	repo.AddCommit("fix: bug 3")

	configYAML := `
mode: Mainline
mainline-increment: EachCommit
commit-message-convention: ConventionalCommits
`
	vars := runPipelineWithConfig(t, repo.Path(), configYAML)

	// Per-commit: fix→1.0.1, fix→1.0.2, feat→1.1.0, fix→1.1.1
	require.Equal(t, "1", vars["Major"])
	require.Equal(t, "1", vars["Minor"])
	require.Equal(t, "1", vars["Patch"])
}

// ---------------------------------------------------------------------------
// Feature Branches
// ---------------------------------------------------------------------------

func TestE2E_FeatureBranch_PreReleaseTag(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial on main")
	repo.CreateTag("v1.0.0", sha)
	repo.CreateBranch("feature/login", sha)
	repo.Checkout("feature/login")
	repo.AddCommit("feat: add login page")

	vars := runPipeline(t, repo.Path())

	// Feature branch gets branch-name pre-release tag.
	require.Equal(t, "login", vars["PreReleaseLabel"])
	require.Contains(t, vars["SemVer"], "login.")
	require.Equal(t, "feature/login", vars["BranchName"])
}

// ---------------------------------------------------------------------------
// Build Metadata
// ---------------------------------------------------------------------------

func TestE2E_BuildMetadata_CommitInfo(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial")
	repo.CreateTag("v1.0.0", sha)
	repo.AddCommit("second commit")

	vars := runPipeline(t, repo.Path())

	require.NotEmpty(t, vars["Sha"])
	require.NotEmpty(t, vars["ShortSha"])
	require.Len(t, vars["ShortSha"], 7)
	require.NotEmpty(t, vars["CommitDate"])
	require.Equal(t, "1", vars["CommitsSinceVersionSource"])
}

// ---------------------------------------------------------------------------
// Output Variables Completeness
// ---------------------------------------------------------------------------

func TestE2E_OutputVariables_AllPresent(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial commit")
	repo.CreateTag("v1.0.0", sha)
	repo.AddCommit("feat: something new")

	vars := runPipeline(t, repo.Path())

	expectedKeys := []string{
		"Major", "Minor", "Patch", "MajorMinorPatch",
		"SemVer", "FullSemVer", "LegacySemVer", "LegacySemVerPadded",
		"InformationalVersion", "BranchName", "EscapedBranchName",
		"Sha", "ShortSha", "CommitDate", "VersionSourceSha",
		"CommitsSinceVersionSource", "CommitsSinceVersionSourcePadded",
		"BuildMetaData", "BuildMetaDataPadded", "FullBuildMetaData",
		"PreReleaseTag", "PreReleaseTagWithDash", "PreReleaseLabel",
		"PreReleaseLabelWithDash", "PreReleaseNumber",
		"WeightedPreReleaseNumber",
		"AssemblySemVer", "AssemblySemFileVer", "AssemblyInformationalVersion",
		"NuGetVersion", "NuGetVersionV2", "NuGetPreReleaseTag", "NuGetPreReleaseTagV2",
		"UncommittedChanges",
	}

	for _, key := range expectedKeys {
		_, ok := vars[key]
		require.True(t, ok, "missing output variable %q", key)
	}
}

// ---------------------------------------------------------------------------
// Multiple Commits, Aggregate Behavior
// ---------------------------------------------------------------------------

func TestE2E_MultipleCommits_HighestWins(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial")
	repo.CreateTag("v1.0.0", sha)
	repo.AddCommit("fix: minor bug")
	repo.AddCommit("feat: big feature")
	repo.AddCommit("fix: another bug")

	configYAML := "commit-message-convention: ConventionalCommits\n"
	vars := runPipelineWithConfig(t, repo.Path(), configYAML)

	// Highest increment from all commits: feat → Minor.
	require.Equal(t, "1", vars["Major"])
	require.Equal(t, "1", vars["Minor"])
	require.Equal(t, "0", vars["Patch"])
	require.Equal(t, "3", vars["CommitsSinceVersionSource"])
}

// ---------------------------------------------------------------------------
// Hotfix Branch
// ---------------------------------------------------------------------------

func TestE2E_HotfixBranch(t *testing.T) {
	repo := testutil.NewTestRepo(t)
	sha := repo.AddCommit("initial on main")
	repo.CreateTag("v1.0.0", sha)
	repo.CreateBranch("hotfix/critical-fix", sha)
	repo.Checkout("hotfix/critical-fix")
	repo.AddCommit("fix: critical security patch")

	vars := runPipeline(t, repo.Path())

	// Hotfix branch: Patch increment, beta pre-release tag.
	require.Equal(t, "1", vars["Major"])
	require.Equal(t, "0", vars["Minor"])
	require.Equal(t, "1", vars["Patch"])
	require.Equal(t, "beta", vars["PreReleaseLabel"])
}
