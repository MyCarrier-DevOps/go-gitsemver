package context

import (
	"fmt"

	"github.com/MyCarrier-DevOps/go-gitsemver/internal/config"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/git"
)

// Options configures what the factory resolves.
type Options struct {
	// TargetBranch overrides HEAD. Empty string means use HEAD.
	TargetBranch string

	// CommitID overrides the branch tip. Empty string means use tip.
	CommitID string
}

// NewContext creates a GitVersionContext by resolving the target branch,
// current commit, version tags, and uncommitted change count.
func NewContext(store *git.RepositoryStore, repo git.Repository, cfg *config.Config, opts Options) (*GitVersionContext, error) {
	// 1. Resolve target branch (from option or HEAD).
	currentBranch, err := store.GetTargetBranch(opts.TargetBranch)
	if err != nil {
		return nil, fmt.Errorf("resolving target branch: %w", err)
	}

	// 2. Get current commit (from SHA option or branch tip).
	currentCommit, err := store.GetCurrentCommit(currentBranch, opts.CommitID)
	if err != nil {
		return nil, fmt.Errorf("resolving current commit: %w", err)
	}

	// 3. Handle detached HEAD: find a branch containing this commit.
	if currentBranch.IsDetachedHead {
		branches, err := store.GetBranchesContainingCommit(currentCommit)
		if err != nil {
			return nil, fmt.Errorf("finding branches for detached HEAD: %w", err)
		}
		if best, ok := pickBestBranch(branches, cfg); ok {
			currentBranch = best
		}
	}

	// 4. Check for version tag on current commit.
	tagPrefix := "[vV]"
	if cfg.TagPrefix != nil {
		tagPrefix = *cfg.TagPrefix
	}
	taggedVersion, isTagged, err := store.GetCurrentCommitTaggedVersion(currentCommit, tagPrefix)
	if err != nil {
		return nil, fmt.Errorf("checking version tag: %w", err)
	}

	// 5. Count uncommitted changes.
	uncommitted, err := store.GetNumberOfUncommittedChanges()
	if err != nil {
		return nil, fmt.Errorf("counting uncommitted changes: %w", err)
	}

	return &GitVersionContext{
		CurrentBranch:              currentBranch,
		CurrentCommit:              currentCommit,
		FullConfiguration:          cfg,
		CurrentCommitTaggedVersion: taggedVersion,
		IsCurrentCommitTagged:      isTagged,
		NumberOfUncommittedChanges: uncommitted,
	}, nil
}

// pickBestBranch selects the best matching branch from candidates using
// config priority. Non-remote branches are preferred. Among matching
// branches, the one with the highest configured priority wins.
func pickBestBranch(branches []git.Branch, cfg *config.Config) (git.Branch, bool) {
	if len(branches) == 0 {
		return git.Branch{}, false
	}

	type scored struct {
		branch   git.Branch
		priority int
	}

	var candidates []scored
	for _, b := range branches {
		if b.IsRemote {
			continue
		}
		bc, _, err := cfg.GetBranchConfiguration(b.FriendlyName())
		if err != nil {
			candidates = append(candidates, scored{branch: b, priority: 0})
			continue
		}
		p := 0
		if bc.Priority != nil {
			p = *bc.Priority
		}
		candidates = append(candidates, scored{branch: b, priority: p})
	}

	if len(candidates) == 0 {
		// Fallback: return first non-remote, or first overall.
		for _, b := range branches {
			if !b.IsRemote {
				return b, true
			}
		}
		return branches[0], true
	}

	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.priority > best.priority {
			best = c
		}
	}
	return best.branch, true
}
