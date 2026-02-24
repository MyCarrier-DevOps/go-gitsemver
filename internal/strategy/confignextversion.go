package strategy

import (
	"fmt"

	"github.com/MyCarrier-DevOps/go-gitsemver/internal/config"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/context"
	"github.com/MyCarrier-DevOps/go-gitsemver/internal/semver"
)

// ConfigNextVersionStrategy returns a version from the next-version config field.
type ConfigNextVersionStrategy struct{}

// NewConfigNextVersionStrategy creates a new ConfigNextVersionStrategy.
func NewConfigNextVersionStrategy() *ConfigNextVersionStrategy {
	return &ConfigNextVersionStrategy{}
}

func (s *ConfigNextVersionStrategy) Name() string { return "ConfigNextVersion" }

func (s *ConfigNextVersionStrategy) GetBaseVersions(
	ctx *context.GitVersionContext,
	ec config.EffectiveConfiguration,
	explain bool,
) ([]BaseVersion, error) {
	var exp *Explanation
	if explain {
		exp = NewExplanation(s.Name())
	}

	nextVersion := ec.NextVersion
	if nextVersion == "" {
		exp.Add("next-version not configured, skipping")
		return nil, nil
	}

	if ctx.IsCurrentCommitTagged {
		exp.Addf("next-version=%q but current commit is tagged, skipping", nextVersion)
		return nil, nil
	}

	// next-version is a bare version string (no tag prefix).
	ver, err := semver.Parse(nextVersion, "")
	if err != nil {
		return nil, fmt.Errorf("parsing next-version %q: %w", nextVersion, err)
	}

	exp.Addf("next-version=%q parsed as %s", nextVersion, ver.SemVer())

	return []BaseVersion{{
		Source:          "NextVersion in configuration file",
		ShouldIncrement: false,
		SemanticVersion: ver,
		Explanation:     exp,
	}}, nil
}
