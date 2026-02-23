// Package testutil provides helpers for creating temporary git repositories
// for end-to-end testing.
package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	gogitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// TestRepo is a builder for creating temporary git repositories with
// controlled commit history, tags, and branches for e2e testing.
type TestRepo struct {
	t    testing.TB
	path string
	repo *gogit.Repository
	time time.Time
}

// NewTestRepo creates and initializes a new git repository in a temporary directory.
func NewTestRepo(t testing.TB) *TestRepo {
	t.Helper()
	dir := t.TempDir()

	repo, err := gogit.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	return &TestRepo{
		t:    t,
		path: dir,
		repo: repo,
		time: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
	}
}

// Path returns the repository root directory.
func (r *TestRepo) Path() string {
	return r.path
}

// AddCommit creates a new commit with the given message. A file named after
// the commit index is created to ensure each commit has changes.
// Returns the commit SHA.
func (r *TestRepo) AddCommit(message string) string {
	r.t.Helper()
	r.time = r.time.Add(time.Minute)

	wt, err := r.repo.Worktree()
	if err != nil {
		r.t.Fatalf("getting worktree: %v", err)
	}

	filename := fmt.Sprintf("file-%d.txt", r.time.Unix())
	path := filepath.Join(r.path, filename)
	if err := os.WriteFile(path, []byte(message), 0o644); err != nil {
		r.t.Fatalf("writing file: %v", err)
	}

	if _, err := wt.Add(filename); err != nil {
		r.t.Fatalf("staging file: %v", err)
	}

	hash, err := wt.Commit(message, &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  r.time,
		},
	})
	if err != nil {
		r.t.Fatalf("committing: %v", err)
	}

	return hash.String()
}

// CreateTag creates a lightweight tag pointing at the given SHA.
func (r *TestRepo) CreateTag(name, sha string) {
	r.t.Helper()
	ref := plumbing.NewReferenceFromStrings("refs/tags/"+name, sha)
	if err := r.repo.Storer.SetReference(ref); err != nil {
		r.t.Fatalf("creating tag %s: %v", name, err)
	}
}

// CreateAnnotatedTag creates an annotated tag pointing at the given SHA.
func (r *TestRepo) CreateAnnotatedTag(name, sha, message string) {
	r.t.Helper()
	r.time = r.time.Add(time.Second)

	hash := plumbing.NewHash(sha)
	_, err := r.repo.CreateTag(name, hash, &gogit.CreateTagOptions{
		Tagger: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  r.time,
		},
		Message: message,
	})
	if err != nil {
		r.t.Fatalf("creating annotated tag %s: %v", name, err)
	}
}

// CreateBranch creates a new branch pointing at the given SHA and checks it out.
func (r *TestRepo) CreateBranch(name, sha string) {
	r.t.Helper()

	ref := plumbing.NewReferenceFromStrings("refs/heads/"+name, sha)
	if err := r.repo.Storer.SetReference(ref); err != nil {
		r.t.Fatalf("creating branch %s: %v", name, err)
	}

	// Store branch config so go-git tracks it.
	cfg, err := r.repo.Config()
	if err != nil {
		r.t.Fatalf("reading config: %v", err)
	}
	cfg.Branches[name] = &gogitconfig.Branch{
		Name:   name,
		Remote: "",
		Merge:  plumbing.ReferenceName("refs/heads/" + name),
	}
	if err := r.repo.SetConfig(cfg); err != nil {
		r.t.Fatalf("saving config: %v", err)
	}
}

// Checkout switches HEAD to the given branch.
func (r *TestRepo) Checkout(branch string) {
	r.t.Helper()
	wt, err := r.repo.Worktree()
	if err != nil {
		r.t.Fatalf("getting worktree: %v", err)
	}

	err = wt.Checkout(&gogit.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branch),
	})
	if err != nil {
		r.t.Fatalf("checking out %s: %v", branch, err)
	}
}

// MergeCommit creates a merge commit with two parents: the current HEAD and
// the given SHA. Returns the merge commit SHA.
func (r *TestRepo) MergeCommit(message, otherSha string) string {
	r.t.Helper()
	r.time = r.time.Add(time.Minute)

	head, err := r.repo.Head()
	if err != nil {
		r.t.Fatalf("getting HEAD: %v", err)
	}

	wt, err := r.repo.Worktree()
	if err != nil {
		r.t.Fatalf("getting worktree: %v", err)
	}

	filename := fmt.Sprintf("merge-%d.txt", r.time.Unix())
	path := filepath.Join(r.path, filename)
	if err := os.WriteFile(path, []byte(message), 0o644); err != nil {
		r.t.Fatalf("writing merge file: %v", err)
	}

	if _, err := wt.Add(filename); err != nil {
		r.t.Fatalf("staging merge file: %v", err)
	}

	hash, err := wt.Commit(message, &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@example.com",
			When:  r.time,
		},
		Parents: []plumbing.Hash{head.Hash(), plumbing.NewHash(otherSha)},
	})
	if err != nil {
		r.t.Fatalf("merge commit: %v", err)
	}

	return hash.String()
}

// WriteConfig writes a gitsemver.yml file in the repo root.
func (r *TestRepo) WriteConfig(content string) {
	r.t.Helper()
	path := filepath.Join(r.path, "gitsemver.yml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		r.t.Fatalf("writing config: %v", err)
	}
}

// HeadSha returns the current HEAD commit SHA.
func (r *TestRepo) HeadSha() string {
	r.t.Helper()
	head, err := r.repo.Head()
	if err != nil {
		r.t.Fatalf("getting HEAD: %v", err)
	}
	return head.Hash().String()
}
