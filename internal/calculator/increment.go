// Package calculator implements the version calculation pipeline:
// strategy evaluation, base version selection, increment logic, and pre-release
// tag management.
package calculator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/MyCarrier-DevOps/go-gitsemver/internal/config"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/context"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/git"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/semver"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/strategy"
)

// IncrementExplanation records the reasoning behind an increment decision.
type IncrementExplanation struct {
	Steps []string
}

// Add appends a reasoning step. Nil-safe.
func (e *IncrementExplanation) Add(step string) {
	if e != nil {
		e.Steps = append(e.Steps, step)
	}
}

// Addf appends a formatted reasoning step. Nil-safe.
func (e *IncrementExplanation) Addf(format string, args ...any) {
	if e != nil {
		e.Steps = append(e.Steps, fmt.Sprintf(format, args...))
	}
}

// IncrementResult holds the determined increment and optional reasoning.
type IncrementResult struct {
	Field       semver.VersionField
	Explanation *IncrementExplanation // nil when explain is false
}

// CommitBump is the increment signal carried by a single commit message.
type CommitBump struct {
	Field semver.VersionField
	// Suppressed reports that the message carried an explicit no-bump
	// directive. That is different from a Field of None, which only means the
	// message asked for nothing: a suppressed commit also declines the branch
	// default increment.
	Suppressed bool
}

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
	result, err := f.DetermineIncrementedFieldExplained(ctx, bv, ec, false)
	return result.Field, err
}

// DetermineIncrementedFieldExplained works like DetermineIncrementedField but
// records reasoning steps when explain is true.
func (f *IncrementStrategyFinder) DetermineIncrementedFieldExplained(
	ctx *context.GitVersionContext,
	bv strategy.BaseVersion,
	ec config.EffectiveConfiguration,
	explain bool,
) (IncrementResult, error) {
	var exp *IncrementExplanation
	if explain {
		exp = &IncrementExplanation{}
	}

	// If commit message incrementing is disabled, use branch default.
	if ec.CommitMessageIncrementing == semver.CommitMessageIncrementDisabled {
		field := f.branchDefault(bv, ec)
		exp.Addf("commit message incrementing disabled, using branch default: %s", field)
		return IncrementResult{Field: field, Explanation: exp}, nil
	}

	from := git.Commit{}
	if bv.BaseVersionSource != nil {
		from = *bv.BaseVersionSource
	}

	commits, err := f.store.GetCommitLog(from, ctx.CurrentCommit)
	if err != nil {
		return IncrementResult{}, err
	}

	exp.Addf("scanned %d commits", len(commits))

	// Scan commits for highest bump.
	highest := semver.VersionFieldNone
	suppressed := false
	for _, c := range commits {
		// Skip the base version source commit itself.
		if bv.BaseVersionSource != nil && c.Sha == bv.BaseVersionSource.Sha {
			continue
		}

		bump := f.AnalyzeCommitBump(c, ec)
		switch {
		case bump.Suppressed:
			suppressed = true
			exp.Addf("commit %s %q -> no-bump directive", c.ShortSha(), firstLine(c.Message))
		case bump.Field != semver.VersionFieldNone:
			convention := conventionName(c.Message, ec)
			exp.Addf("commit %s %q -> %s (%s)", c.ShortSha(), firstLine(c.Message), bump.Field, convention)
		}
		highest = max(highest, bump.Field)
	}

	exp.Addf("highest increment from commits: %s", highest)

	// If version < 1.0.0, cap at Minor (no Major bumps before 1.0).
	if bv.SemanticVersion.Major == 0 {
		if highest == semver.VersionFieldMajor {
			highest = semver.VersionFieldMinor
			exp.Add("pre-1.0: capping Major -> Minor")
		}
	}

	// An explicit no-bump directive suppresses the branch default increment.
	// It never overrides a real increment requested elsewhere in the range, so a
	// "+semver: none" chore cannot veto a feat: in the same set of commits.
	if suppressed && highest == semver.VersionFieldNone {
		exp.Add("no-bump directive present and no commit requested an increment: suppressing branch default")
		return IncrementResult{Field: semver.VersionFieldNone, Explanation: exp}, nil
	}

	// If ShouldIncrement and commit bump is less than branch default, use default.
	if bv.ShouldIncrement {
		branchField := f.branchDefault(bv, ec)
		if highest < branchField {
			exp.Addf("ShouldIncrement=true, branch default=%s > %s, using %s", branchField, highest, branchField)
			highest = branchField
		}
	}

	return IncrementResult{Field: highest, Explanation: exp}, nil
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

// AnalyzeCommitBump returns the increment signal carried by a single commit
// message. Exported for use by MainlineVersionCalculator in per-commit mode.
func (f *IncrementStrategyFinder) AnalyzeCommitBump(
	c git.Commit,
	ec config.EffectiveConfiguration,
) CommitBump {
	// MergeMessageOnly: only analyze merge commits.
	if ec.CommitMessageIncrementing == semver.CommitMessageIncrementMergeMessageOnly && !c.IsMerge() {
		return CommitBump{}
	}

	switch ec.CommitMessageConvention {
	case semver.CommitMessageConventionConventionalCommits:
		// Bump directives, including no-bump, are not honoured in this mode.
		return CommitBump{Field: analyzeConventionalCommit(c.Message)}

	case semver.CommitMessageConventionBumpDirective:
		if isNoBump(c.Message, ec) {
			return CommitBump{Suppressed: true}
		}
		return CommitBump{Field: analyzeBumpDirective(c.Message, ec)}

	case semver.CommitMessageConventionBoth:
		// An explicit no-bump directive is a manual override, so it wins over
		// whatever the conventional-commit type would otherwise ask for.
		if isNoBump(c.Message, ec) {
			return CommitBump{Suppressed: true}
		}
		return CommitBump{Field: max(analyzeConventionalCommit(c.Message), analyzeBumpDirective(c.Message, ec))}
	}

	return CommitBump{}
}

// analyzeConventionalCommit parses a Conventional Commits message.
// feat: → Minor; fix:, perf: and chore: → Patch; any type with a ! suffix or a
// BREAKING CHANGE: footer → Major.
func analyzeConventionalCommit(msg string) semver.VersionField {
	matches := ccTypeRe.FindStringSubmatch(firstLine(msg))
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
	case "fix", "perf", "chore":
		return semver.VersionFieldPatch
	default:
		// Other types (build, ci, docs, refactor, revert, style, test) don't bump.
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

// isNoBump reports whether the message carries the configured no-bump
// directive, which asks for the increment to be suppressed entirely.
func isNoBump(msg string, ec config.EffectiveConfiguration) bool {
	return tryMatch(msg, ec.NoBumpMessage)
}

// firstLine returns the first line of a commit message.
func firstLine(msg string) string {
	if idx := strings.IndexByte(msg, '\n'); idx >= 0 {
		return msg[:idx]
	}
	return msg
}

// conventionName returns a human-readable label for the convention that matched
// the commit message. Used in explain mode output.
func conventionName(msg string, ec config.EffectiveConfiguration) string {
	switch ec.CommitMessageConvention {
	case semver.CommitMessageConventionConventionalCommits:
		return "Conventional Commits"
	case semver.CommitMessageConventionBumpDirective:
		return "Bump Directive"
	case semver.CommitMessageConventionBoth:
		cc := analyzeConventionalCommit(msg)
		bd := analyzeBumpDirective(msg, ec)
		if cc >= bd && cc != semver.VersionFieldNone {
			return "Conventional Commits"
		}
		if bd != semver.VersionFieldNone {
			return "Bump Directive"
		}
		return "Conventional Commits"
	default:
		return "Bump Directive"
	}
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
