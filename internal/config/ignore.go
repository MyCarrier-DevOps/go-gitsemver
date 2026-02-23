package config

import (
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// IgnoreConfig controls which commits are excluded from version calculation.
type IgnoreConfig struct {
	CommitsBefore *time.Time `yaml:"commits-before"`
	Sha           []string   `yaml:"sha"`
}

// IsEmpty returns true when no ignore rules are configured.
func (c IgnoreConfig) IsEmpty() bool {
	return c.CommitsBefore == nil && len(c.Sha) == 0
}

// flexTime wraps time.Time for flexible YAML date parsing.
// Supports both date-only ("2024-01-01") and RFC3339 formats.
type flexTime time.Time

func (ft *flexTime) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}

	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02",
	}

	for _, layout := range formats {
		if t, err := time.Parse(layout, s); err == nil {
			*ft = flexTime(t)
			return nil
		}
	}
	return fmt.Errorf("cannot parse date %q: expected RFC3339 or YYYY-MM-DD", s)
}

// UnmarshalYAML implements custom date parsing for IgnoreConfig.
func (c *IgnoreConfig) UnmarshalYAML(value *yaml.Node) error {
	var raw struct {
		CommitsBefore *flexTime `yaml:"commits-before"`
		Sha           []string  `yaml:"sha"`
	}
	if err := value.Decode(&raw); err != nil {
		return err
	}
	if raw.CommitsBefore != nil {
		t := time.Time(*raw.CommitsBefore)
		c.CommitsBefore = &t
	}
	c.Sha = raw.Sha
	return nil
}
