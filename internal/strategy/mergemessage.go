package strategy

import (
	"fmt"
	"go-gitsemver/internal/config"
	"go-gitsemver/internal/context"
	"go-gitsemver/internal/git"
	"go-gitsemver/internal/semver"
	"strings"
)

const maxMergeMessageResults = 5

// MergeMessageStrategy returns versions from merge commit messages.
type MergeMessageStrategy struct {
	store *git.RepositoryStore
}

// NewMergeMessageStrategy creates a new MergeMessageStrategy.
func NewMergeMessageStrategy(store *git.RepositoryStore) *MergeMessageStrategy {
	return &MergeMessageStrategy{store: store}
}

func (s *MergeMessageStrategy) Name() string { return "MergeMessage" }

func (s *MergeMessageStrategy) GetBaseVersions(
	ctx *context.GitVersionContext,
	ec config.EffectiveConfiguration,
	explain bool,
) ([]BaseVersion, error) {
	if ctx.CurrentBranch.Tip == nil {
		return nil, nil
	}

	var exp *Explanation
	if explain {
		exp = NewExplanation(s.Name())
	}

	commits, err := s.store.GetCommitLog(git.Commit{}, ctx.CurrentCommit)
	if err != nil {
		return nil, fmt.Errorf("getting commit log: %w", err)
	}

	exp.Addf("scanning %d commits for merge messages", len(commits))

	var results []BaseVersion

	// Pass 1: merge commits (2+ parents).
	for _, commit := range commits {
		if len(results) >= maxMergeMessageResults {
			break
		}
		if !commit.IsMerge() {
			continue
		}

		bv, ok := s.tryExtractVersion(ctx, ec, commit, explain, exp)
		if ok {
			results = append(results, bv)
		}
	}

	// Pass 2: squash merges (DI-8) — single-parent commits against squash formats.
	for _, commit := range commits {
		if len(results) >= maxMergeMessageResults {
			break
		}
		if commit.IsMerge() {
			continue
		}

		mm := git.ParseMergeMessage(commit.Message, nil)
		if mm.IsEmpty() || mm.MergedBranch == "" {
			continue
		}

		mergedBranch := trimRemotePrefix(mm.MergedBranch)
		if mergedBranch == "" || !ctx.FullConfiguration.IsReleaseBranch(mergedBranch) {
			continue
		}

		versionStr, ok := git.ExtractVersionFromBranch(mergedBranch, ec.TagPrefix)
		if !ok {
			continue
		}

		ver, err := semver.Parse(versionStr, "")
		if err != nil {
			continue
		}

		shouldIncrement := !ec.PreventIncrementOfMergedBranchVersion
		source := fmt.Sprintf("Squash merge '%s'", strings.TrimSpace(firstLine(commit.Message)))

		var bvExp *Explanation
		if explain {
			bvExp = NewExplanation(s.Name())
			bvExp.Addf("squash commit %s: branch %q (format: %s) -> %s",
				commit.ShortSha(), mergedBranch, mm.FormatName, ver.SemVer())
		}

		c := commit
		results = append(results, BaseVersion{
			Source:            source,
			ShouldIncrement:   shouldIncrement,
			SemanticVersion:   ver,
			BaseVersionSource: &c,
			Explanation:       bvExp,
		})
	}

	exp.Addf("found %d merge message versions", len(results))
	return results, nil
}

func (s *MergeMessageStrategy) tryExtractVersion(
	ctx *context.GitVersionContext,
	ec config.EffectiveConfiguration,
	commit git.Commit,
	explain bool,
	parentExp *Explanation,
) (BaseVersion, bool) {
	mm := git.ParseMergeMessage(commit.Message, ec.MergeMessageFormats)
	if mm.IsEmpty() {
		return BaseVersion{}, false
	}

	mergedBranch := trimRemotePrefix(mm.MergedBranch)
	if mergedBranch == "" {
		return BaseVersion{}, false
	}

	if !ctx.FullConfiguration.IsReleaseBranch(mergedBranch) {
		parentExp.Addf("commit %s: merged branch %q is not a release branch, skipping",
			commit.ShortSha(), mergedBranch)
		return BaseVersion{}, false
	}

	versionStr, ok := git.ExtractVersionFromBranch(mergedBranch, ec.TagPrefix)
	if !ok {
		parentExp.Addf("commit %s: no version in branch name %q", commit.ShortSha(), mergedBranch)
		return BaseVersion{}, false
	}

	ver, err := semver.Parse(versionStr, "")
	if err != nil {
		return BaseVersion{}, false
	}

	shouldIncrement := !ec.PreventIncrementOfMergedBranchVersion
	source := fmt.Sprintf("Merge message '%s'", strings.TrimSpace(firstLine(commit.Message)))

	var bvExp *Explanation
	if explain {
		bvExp = NewExplanation(s.Name())
		bvExp.Addf("commit %s: merge of %q (format: %s) -> %s, ShouldIncrement=%t",
			commit.ShortSha(), mergedBranch, mm.FormatName, ver.SemVer(), shouldIncrement)
	}

	c := commit
	return BaseVersion{
		Source:            source,
		ShouldIncrement:   shouldIncrement,
		SemanticVersion:   ver,
		BaseVersionSource: &c,
		Explanation:       bvExp,
	}, true
}

// trimRemotePrefix strips remote tracking prefixes from branch names.
func trimRemotePrefix(name string) string {
	name = strings.TrimPrefix(name, "refs/remotes/")
	name = strings.TrimPrefix(name, "origin/")
	return name
}

// firstLine returns the first line of a string.
func firstLine(s string) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return s[:idx]
	}
	return s
}
