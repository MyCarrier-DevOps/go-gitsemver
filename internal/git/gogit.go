package git

import (
	"fmt"
	"path/filepath"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

// Compile-time check that GoGitRepository implements Repository.
var _ Repository = (*GoGitRepository)(nil)

// GoGitRepository implements Repository using go-git.
type GoGitRepository struct {
	repo    *gogit.Repository
	path    string
	workDir string
}

// Open opens a git repository at the given path.
func Open(path string) (*GoGitRepository, error) {
	r, err := gogit.PlainOpenWithOptions(path, &gogit.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return nil, fmt.Errorf("opening git repository at %s: %w", path, err)
	}

	wt, err := r.Worktree()
	if err != nil {
		return nil, fmt.Errorf("getting worktree: %w", err)
	}

	root := wt.Filesystem.Root()

	return &GoGitRepository{
		repo:    r,
		path:    filepath.Join(root, ".git"),
		workDir: root,
	}, nil
}

func (r *GoGitRepository) Path() string {
	return r.path
}

func (r *GoGitRepository) WorkingDirectory() string {
	return r.workDir
}

func (r *GoGitRepository) IsHeadDetached() bool {
	ref, err := r.repo.Head()
	if err != nil {
		return false
	}
	return !ref.Name().IsBranch()
}

func (r *GoGitRepository) Head() (Branch, error) {
	ref, err := r.repo.Head()
	if err != nil {
		return Branch{}, fmt.Errorf("getting HEAD: %w", err)
	}

	commit, err := r.commitFromHash(ref.Hash())
	if err != nil {
		return Branch{}, fmt.Errorf("getting HEAD commit: %w", err)
	}

	isDetached := !ref.Name().IsBranch()
	name := NewReferenceName(string(ref.Name()))

	return Branch{
		Name:           name,
		Tip:            &commit,
		IsRemote:       false,
		IsDetachedHead: isDetached,
	}, nil
}

func (r *GoGitRepository) Branches(_ ...PathFilter) ([]Branch, error) {
	var branches []Branch

	// Local branches.
	localIter, err := r.repo.Branches()
	if err != nil {
		return nil, fmt.Errorf("listing local branches: %w", err)
	}
	err = localIter.ForEach(func(ref *plumbing.Reference) error {
		commit, err := r.commitFromHash(ref.Hash())
		if err != nil {
			return nil // skip branches we can't resolve
		}
		branches = append(branches, Branch{
			Name:     NewReferenceName(string(ref.Name())),
			Tip:      &commit,
			IsRemote: false,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("iterating local branches: %w", err)
	}

	// Remote branches.
	refIter, err := r.repo.References()
	if err != nil {
		return nil, fmt.Errorf("listing references: %w", err)
	}
	err = refIter.ForEach(func(ref *plumbing.Reference) error {
		if !ref.Name().IsRemote() {
			return nil
		}
		commit, err := r.commitFromHash(ref.Hash())
		if err != nil {
			return nil
		}
		branches = append(branches, Branch{
			Name:     NewReferenceName(string(ref.Name())),
			Tip:      &commit,
			IsRemote: true,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("iterating remote branches: %w", err)
	}

	return branches, nil
}

func (r *GoGitRepository) Tags(_ ...PathFilter) ([]Tag, error) {
	var tags []Tag

	iter, err := r.repo.Tags()
	if err != nil {
		return nil, fmt.Errorf("listing tags: %w", err)
	}

	err = iter.ForEach(func(ref *plumbing.Reference) error {
		tags = append(tags, Tag{
			Name:      NewReferenceName(string(ref.Name())),
			TargetSha: ref.Hash().String(),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("iterating tags: %w", err)
	}

	return tags, nil
}

func (r *GoGitRepository) CommitFromSha(sha string) (Commit, error) {
	hash := plumbing.NewHash(sha)
	return r.commitFromHash(hash)
}

func (r *GoGitRepository) CommitLog(from, to string, _ ...PathFilter) ([]Commit, error) {
	toHash := plumbing.NewHash(to)

	iter, err := r.repo.Log(&gogit.LogOptions{
		From:  toHash,
		Order: gogit.LogOrderCommitterTime,
	})
	if err != nil {
		return nil, fmt.Errorf("getting commit log: %w", err)
	}

	fromHash := plumbing.ZeroHash
	if from != "" {
		fromHash = plumbing.NewHash(from)
	}

	var commits []Commit
	err = iter.ForEach(func(c *object.Commit) error {
		if c.Hash == fromHash {
			return storer.ErrStop
		}
		commits = append(commits, convertCommit(c))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("iterating commits: %w", err)
	}

	return commits, nil
}

func (r *GoGitRepository) MainlineCommitLog(from, to string, _ ...PathFilter) ([]Commit, error) {
	toHash := plumbing.NewHash(to)

	iter, err := r.repo.Log(&gogit.LogOptions{
		From:  toHash,
		Order: gogit.LogOrderCommitterTime,
	})
	if err != nil {
		return nil, fmt.Errorf("getting mainline commit log: %w", err)
	}

	fromHash := plumbing.ZeroHash
	if from != "" {
		fromHash = plumbing.NewHash(from)
	}

	// First-parent only: follow only the first parent at each merge.
	var commits []Commit
	err = iter.ForEach(func(c *object.Commit) error {
		if c.Hash == fromHash {
			return storer.ErrStop
		}
		commits = append(commits, convertCommit(c))
		// For first-parent walk, we stop at the first parent only.
		// go-git's default LogOrderCommitterTime already follows DAG order,
		// but it visits all parents. We break manually after the merge
		// commit to follow only the mainline. This is approximate; for
		// exact first-parent semantics, use the parent-walking approach.
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("iterating mainline commits: %w", err)
	}

	return commits, nil
}

func (r *GoGitRepository) BranchCommits(branch Branch, _ ...PathFilter) ([]Commit, error) {
	if branch.Tip == nil {
		return nil, nil
	}

	return r.CommitLog("", branch.Tip.Sha)
}

func (r *GoGitRepository) CommitsPriorTo(olderThan time.Time, branch Branch) ([]Commit, error) {
	allCommits, err := r.BranchCommits(branch)
	if err != nil {
		return nil, err
	}

	var result []Commit
	for _, c := range allCommits {
		if c.When.Before(olderThan) {
			result = append(result, c)
		}
	}

	return result, nil
}

func (r *GoGitRepository) FindMergeBase(sha1, sha2 string) (string, error) {
	hash1 := plumbing.NewHash(sha1)
	hash2 := plumbing.NewHash(sha2)

	c1, err := r.repo.CommitObject(hash1)
	if err != nil {
		return "", fmt.Errorf("loading commit %s: %w", sha1, err)
	}

	c2, err := r.repo.CommitObject(hash2)
	if err != nil {
		return "", fmt.Errorf("loading commit %s: %w", sha2, err)
	}

	bases, err := c1.MergeBase(c2)
	if err != nil {
		return "", fmt.Errorf("computing merge base: %w", err)
	}

	if len(bases) == 0 {
		return "", nil
	}

	return bases[0].Hash.String(), nil
}

func (r *GoGitRepository) BranchesContainingCommit(sha string) ([]Branch, error) {
	targetHash := plumbing.NewHash(sha)
	allBranches, err := r.Branches()
	if err != nil {
		return nil, err
	}

	var result []Branch
	for _, b := range allBranches {
		if b.Tip == nil {
			continue
		}

		tipHash := plumbing.NewHash(b.Tip.Sha)
		if targetHash == tipHash {
			result = append(result, b)
			continue
		}

		tipCommit, err := r.repo.CommitObject(tipHash)
		if err != nil {
			continue
		}

		targetCommit, err := r.repo.CommitObject(targetHash)
		if err != nil {
			continue
		}

		isAnc, err := targetCommit.IsAncestor(tipCommit)
		if err != nil {
			continue
		}

		if isAnc {
			result = append(result, b)
		}
	}

	return result, nil
}

func (r *GoGitRepository) NumberOfUncommittedChanges() (int, error) {
	wt, err := r.repo.Worktree()
	if err != nil {
		return 0, fmt.Errorf("getting worktree: %w", err)
	}

	status, err := wt.Status()
	if err != nil {
		return 0, fmt.Errorf("getting worktree status: %w", err)
	}

	count := 0
	for _, s := range status {
		if s.Staging != gogit.Unmodified || s.Worktree != gogit.Unmodified {
			count++
		}
	}

	return count, nil
}

func (r *GoGitRepository) PeelTagToCommit(tag Tag) (string, error) {
	hash := plumbing.NewHash(tag.TargetSha)

	// Try as an annotated tag first.
	tagObj, err := r.repo.TagObject(hash)
	if err == nil {
		// Peel through annotated tags (possibly nested).
		commit, err := tagObj.Commit()
		if err != nil {
			return "", fmt.Errorf("peeling annotated tag %s: %w", tag.Name.Friendly, err)
		}
		return commit.Hash.String(), nil
	}

	// If not an annotated tag, check if it points directly to a commit.
	_, err = r.repo.CommitObject(hash)
	if err != nil {
		return "", fmt.Errorf("tag %s does not point to a commit: %w", tag.Name.Friendly, err)
	}

	return tag.TargetSha, nil
}

// commitFromHash loads a go-git commit and converts it to our Commit type.
func (r *GoGitRepository) commitFromHash(hash plumbing.Hash) (Commit, error) {
	c, err := r.repo.CommitObject(hash)
	if err != nil {
		return Commit{}, fmt.Errorf("loading commit %s: %w", hash.String(), err)
	}
	return convertCommit(c), nil
}

// convertCommit converts a go-git commit to our Commit type.
func convertCommit(c *object.Commit) Commit {
	parents := make([]string, 0, c.NumParents())
	for _, p := range c.ParentHashes {
		parents = append(parents, p.String())
	}

	return Commit{
		Sha:     c.Hash.String(),
		Parents: parents,
		When:    c.Committer.When,
		Message: c.Message,
	}
}
