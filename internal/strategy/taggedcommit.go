package strategy

import (
	"fmt"
	"time"

	"github.com/MyCarrier-DevOps/go-gitsemver/internal/config"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/context"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/git"
)

// TaggedCommitStrategy returns versions from git tags on the current branch.
type TaggedCommitStrategy struct {
	store *git.RepositoryStore
}

// NewTaggedCommitStrategy creates a new TaggedCommitStrategy.
func NewTaggedCommitStrategy(store *git.RepositoryStore) *TaggedCommitStrategy {
	return &TaggedCommitStrategy{store: store}
}

func (s *TaggedCommitStrategy) Name() string { return "TaggedCommit" }

func (s *TaggedCommitStrategy) GetBaseVersions(
	ctx *context.GitVersionContext,
	ec config.EffectiveConfiguration,
	explain bool,
) ([]BaseVersion, error) {
	return s.getTaggedVersions(ctx, ec, ctx.CurrentBranch, &ctx.CurrentCommit.When, explain)
}

// getTaggedVersions is the internal implementation, also called by
// TrackReleaseBranchesStrategy for main branch tags.
func (s *TaggedCommitStrategy) getTaggedVersions(
	ctx *context.GitVersionContext,
	ec config.EffectiveConfiguration,
	branch git.Branch,
	olderThan *time.Time,
	explain bool,
) ([]BaseVersion, error) {
	if branch.Tip == nil {
		return nil, nil
	}

	versionTags, err := s.store.GetValidVersionTags(ec.TagPrefix, olderThan)
	if err != nil {
		return nil, fmt.Errorf("getting version tags: %w", err)
	}

	// Build lookup: commit SHA -> []VersionTag.
	tagsByCommit := make(map[string][]git.VersionTag)
	for _, vt := range versionTags {
		tagsByCommit[vt.Commit.Sha] = append(tagsByCommit[vt.Commit.Sha], vt)
	}

	// Walk commits on branch, collect matching tags.
	commits, err := s.store.GetCommitLog(git.Commit{}, *branch.Tip)
	if err != nil {
		return nil, fmt.Errorf("getting branch commits: %w", err)
	}

	var all []BaseVersion
	var onCurrent []BaseVersion

	for _, commit := range commits {
		tags, ok := tagsByCommit[commit.Sha]
		if !ok {
			continue
		}
		for _, vt := range tags {
			shouldIncrement := vt.Commit.Sha != ctx.CurrentCommit.Sha
			source := fmt.Sprintf("Git tag '%s'", vt.Tag.Name.Friendly)

			var bvExp *Explanation
			if explain {
				bvExp = NewExplanation("TaggedCommit")
				bvExp.Addf("tag %s on commit %s -> %s, ShouldIncrement=%t",
					vt.Tag.Name.Friendly, vt.Commit.ShortSha(),
					vt.Version.SemVer(), shouldIncrement)
			}

			c := vt.Commit
			bv := BaseVersion{
				Source:            source,
				ShouldIncrement:   shouldIncrement,
				SemanticVersion:   vt.Version,
				BaseVersionSource: &c,
				Explanation:       bvExp,
			}

			all = append(all, bv)
			if !shouldIncrement {
				onCurrent = append(onCurrent, bv)
			}
		}
	}

	// If any tags are on current commit, return only those.
	if len(onCurrent) > 0 {
		return onCurrent, nil
	}

	return all, nil
}
