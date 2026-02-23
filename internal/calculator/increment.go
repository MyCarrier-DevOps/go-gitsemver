// Package calculator implements the version calculation pipeline:
// strategy evaluation, base version selection, increment logic, and pre-release
// tag management.
package calculator

import (
	"go-gitsemver/internal/config"
	"go-gitsemver/internal/context"
	"go-gitsemver/internal/git"
	"go-gitsemver/internal/semver"
	"go-gitsemver/internal/strategy"
	"regexp"
	"strings"
)

// Conventional Commits patterns.
var (
	ccTypeRe         = regexp.MustCompile(`^(\w+)(?:\(.+?\))?(!)?:\s`)
	breakingFooterRe = regexp.MustCompile(`(?m)^BREAKING[ -]CHANGE:\s`)
)

// IncrementStrategyFinder scans commit messages to determine the version bump.
type IncrementStrategyFinder struct {
	store *git.RepositoryStore
}

// NewIncrementStrategyFinder creates a new IncrementStrategyFinder.
func NewIncrementStrategyFinder(store *git.RepositoryStore) *IncrementStrategyFinder {
	return &IncrementStrategyFinder{store: store}
}

// DetermineIncrementedField scans commits between the base version source and
// the current commit to find the highest version bump from commit messages.
// It respects the configured commit message convention and increment mode.
func (f *IncrementStrategyFinder) DetermineIncrementedField(
	ctx *context.GitVersionContext,
	bv strategy.BaseVersion,
	ec config.EffectiveConfiguration,
) (semver.VersionField, error) {
	// If commit message incrementing is disabled, use branch default.
	if ec.CommitMessageIncrementing == semver.CommitMessageIncrementDisabled {
		return f.branchDefault(bv, ec), nil
	}

	from := git.Commit{}
	if bv.BaseVersionSource != nil {
		from = *bv.BaseVersionSource
	}

	commits, err := f.store.GetCommitLog(from, ctx.CurrentCommit)
	if err != nil {
		return semver.VersionFieldNone, err
	}

	// Scan commits for highest bump.
	highest := semver.VersionFieldNone
	for _, c := range commits {
		// Skip the base version source commit itself.
		if bv.BaseVersionSource != nil && c.Sha == bv.BaseVersionSource.Sha {
			continue
		}

		field := f.analyzeCommit(c, ec)
		if field > highest {
			highest = field
		}
	}

	// If version < 1.0.0, cap at Minor (no Major bumps before 1.0).
	if bv.SemanticVersion.Major == 0 {
		if highest == semver.VersionFieldMajor {
			highest = semver.VersionFieldMinor
		}
	}

	// If ShouldIncrement and commit bump is less than branch default, use default.
	if bv.ShouldIncrement {
		branchField := f.branchDefault(bv, ec)
		if highest < branchField {
			highest = branchField
		}
	}

	return highest, nil
}

// branchDefault returns the branch's configured increment as a VersionField.
func (f *IncrementStrategyFinder) branchDefault(
	bv strategy.BaseVersion,
	ec config.EffectiveConfiguration,
) semver.VersionField {
	if !bv.ShouldIncrement {
		return semver.VersionFieldNone
	}
	field := ec.BranchIncrement.ToVersionField()
	if field == semver.VersionFieldNone {
		// Inherit falls back to Patch.
		return semver.VersionFieldPatch
	}
	return field
}

// AnalyzeCommitIncrement returns the version bump for a single commit message.
// Exported for use by MainlineVersionCalculator in per-commit mode.
func (f *IncrementStrategyFinder) AnalyzeCommitIncrement(
	c git.Commit,
	ec config.EffectiveConfiguration,
) semver.VersionField {
	return f.analyzeCommit(c, ec)
}

// analyzeCommit extracts the version bump from a single commit message.
func (f *IncrementStrategyFinder) analyzeCommit(
	c git.Commit,
	ec config.EffectiveConfiguration,
) semver.VersionField {
	// MergeMessageOnly: only analyze merge commits.
	if ec.CommitMessageIncrementing == semver.CommitMessageIncrementMergeMessageOnly && !c.IsMerge() {
		return semver.VersionFieldNone
	}

	highest := semver.VersionFieldNone

	switch ec.CommitMessageConvention {
	case semver.CommitMessageConventionConventionalCommits:
		highest = analyzeConventionalCommit(c.Message)
	case semver.CommitMessageConventionBumpDirective:
		highest = analyzeBumpDirective(c.Message, ec)
	case semver.CommitMessageConventionBoth:
		cc := analyzeConventionalCommit(c.Message)
		bd := analyzeBumpDirective(c.Message, ec)
		if cc > bd {
			highest = cc
		} else {
			highest = bd
		}
	}

	return highest
}

// analyzeConventionalCommit parses a Conventional Commits message.
// feat: → Minor, fix: → Patch, feat!: or BREAKING CHANGE: footer → Major
func analyzeConventionalCommit(msg string) semver.VersionField {
	firstLine := msg
	if idx := strings.IndexByte(msg, '\n'); idx >= 0 {
		firstLine = msg[:idx]
	}

	matches := ccTypeRe.FindStringSubmatch(firstLine)
	if matches == nil {
		return semver.VersionFieldNone
	}

	// Check for breaking change indicator (!) in first line.
	if matches[2] == "!" {
		return semver.VersionFieldMajor
	}

	// Check for BREAKING CHANGE footer in full message.
	if breakingFooterRe.MatchString(msg) {
		return semver.VersionFieldMajor
	}

	ccType := strings.ToLower(matches[1])
	switch ccType {
	case "feat":
		return semver.VersionFieldMinor
	case "fix":
		return semver.VersionFieldPatch
	default:
		// Other types (docs, chore, refactor, etc.) don't bump.
		return semver.VersionFieldNone
	}
}

// analyzeBumpDirective checks for +semver: directives in commit messages.
func analyzeBumpDirective(msg string, ec config.EffectiveConfiguration) semver.VersionField {
	if tryMatch(msg, ec.MajorVersionBumpMessage) {
		return semver.VersionFieldMajor
	}
	if tryMatch(msg, ec.MinorVersionBumpMessage) {
		return semver.VersionFieldMinor
	}
	if tryMatch(msg, ec.PatchVersionBumpMessage) {
		return semver.VersionFieldPatch
	}
	return semver.VersionFieldNone
}

// tryMatch returns true if the message matches the regex pattern.
func tryMatch(msg, pattern string) bool {
	if pattern == "" {
		return false
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(msg)
}
