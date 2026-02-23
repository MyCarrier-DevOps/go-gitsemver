package config

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type branchMatch struct {
	name     string
	branch   *BranchConfig
	priority int
}

// GetBranchConfiguration returns the best-matching BranchConfig for the given
// branch name, using priority-ordered regex matching (DI-12). Returns the
// matched BranchConfig, the branch config key name, and any error.
func (cfg *Config) GetBranchConfiguration(branchName string) (*BranchConfig, string, error) {
	var matches []branchMatch

	for name, branch := range cfg.Branches {
		if branch.Regex == nil {
			continue
		}
		re, err := regexp.Compile(*branch.Regex)
		if err != nil {
			return nil, "", fmt.Errorf("invalid regex for branch %q: %w", name, err)
		}
		if re.MatchString(branchName) {
			p := 0
			if branch.Priority != nil {
				p = *branch.Priority
			}
			matches = append(matches, branchMatch{
				name:     name,
				branch:   branch,
				priority: p,
			})
		}
	}

	if len(matches) == 0 {
		return nil, "", fmt.Errorf("no branch configuration matches %q", branchName)
	}

	// Sort by priority descending, then by name ascending for determinism
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].priority != matches[j].priority {
			return matches[i].priority > matches[j].priority
		}
		return matches[i].name < matches[j].name
	})

	return matches[0].branch, matches[0].name, nil
}

// GetReleaseBranchConfig returns all branch configurations where
// IsReleaseBranch is true.
func (cfg *Config) GetReleaseBranchConfig() map[string]*BranchConfig {
	result := make(map[string]*BranchConfig)
	for name, branch := range cfg.Branches {
		if branch.IsReleaseBranch != nil && *branch.IsReleaseBranch {
			result[name] = branch
		}
	}
	return result
}

var branchPrefixes = []string{
	"feature/", "features/",
	"hotfix/", "hotfixes/",
	"bugfix/", "bugfixes/",
	"release/", "releases/",
	"support/",
	"pull/", "pull-requests/", "pr/",
}

var branchNameCleaner = regexp.MustCompile(`[^a-zA-Z0-9-]`)

// GetBranchSpecificTag resolves the pre-release tag for a branch,
// replacing {BranchName} with the actual branch name (with prefix stripped
// and special characters replaced with hyphens).
func GetBranchSpecificTag(branchName, tag string) string {
	if !strings.Contains(tag, "{BranchName}") {
		return tag
	}
	cleaned := stripBranchPrefix(branchName)
	cleaned = branchNameCleaner.ReplaceAllString(cleaned, "-")
	return strings.ReplaceAll(tag, "{BranchName}", cleaned)
}

func stripBranchPrefix(name string) string {
	for _, prefix := range branchPrefixes {
		if strings.HasPrefix(name, prefix) {
			return name[len(prefix):]
		}
	}
	return name
}
