package semver

import "gopkg.in/yaml.v3"

// UnmarshalYAML implements yaml.Unmarshaler for VersioningMode.
func (m *VersioningMode) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	parsed, err := ParseVersioningMode(s)
	if err != nil {
		return err
	}
	*m = parsed
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler for IncrementStrategy.
func (s *IncrementStrategy) UnmarshalYAML(value *yaml.Node) error {
	var str string
	if err := value.Decode(&str); err != nil {
		return err
	}
	parsed, err := ParseIncrementStrategy(str)
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler for CommitMessageIncrementMode.
func (m *CommitMessageIncrementMode) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	parsed, err := ParseCommitMessageIncrementMode(s)
	if err != nil {
		return err
	}
	*m = parsed
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler for CommitMessageConvention.
func (c *CommitMessageConvention) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	parsed, err := ParseCommitMessageConvention(s)
	if err != nil {
		return err
	}
	*c = parsed
	return nil
}
