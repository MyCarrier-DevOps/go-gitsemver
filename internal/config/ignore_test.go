package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestIgnoreConfig_IsEmpty(t *testing.T) {
	tests := []struct {
		name   string
		config IgnoreConfig
		want   bool
	}{
		{"zero value", IgnoreConfig{}, true},
		{"only commits-before", IgnoreConfig{CommitsBefore: timePtr(time.Now())}, false},
		{"only sha", IgnoreConfig{Sha: []string{"abc123"}}, false},
		{"both set", IgnoreConfig{CommitsBefore: timePtr(time.Now()), Sha: []string{"abc"}}, false},
		{"empty sha slice", IgnoreConfig{Sha: []string{}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.config.IsEmpty())
		})
	}
}

func TestIgnoreConfig_UnmarshalYAML_DateOnly(t *testing.T) {
	data := []byte("commits-before: 2024-01-15\nsha:\n  - abc123\n  - def456\n")
	var cfg IgnoreConfig
	require.NoError(t, yaml.Unmarshal(data, &cfg))
	require.NotNil(t, cfg.CommitsBefore)
	require.Equal(t, 2024, cfg.CommitsBefore.Year())
	require.Equal(t, time.January, cfg.CommitsBefore.Month())
	require.Equal(t, 15, cfg.CommitsBefore.Day())
	require.Equal(t, []string{"abc123", "def456"}, cfg.Sha)
}

func TestIgnoreConfig_UnmarshalYAML_RFC3339(t *testing.T) {
	data := []byte("commits-before: 2024-06-15T10:30:00Z\n")
	var cfg IgnoreConfig
	require.NoError(t, yaml.Unmarshal(data, &cfg))
	require.NotNil(t, cfg.CommitsBefore)
	require.Equal(t, 2024, cfg.CommitsBefore.Year())
	require.Equal(t, time.June, cfg.CommitsBefore.Month())
	require.Equal(t, 10, cfg.CommitsBefore.Hour())
}

func TestIgnoreConfig_UnmarshalYAML_DateTimeNoTimezone(t *testing.T) {
	data := []byte("commits-before: 2024-03-20T14:00:00\n")
	var cfg IgnoreConfig
	require.NoError(t, yaml.Unmarshal(data, &cfg))
	require.NotNil(t, cfg.CommitsBefore)
	require.Equal(t, 2024, cfg.CommitsBefore.Year())
	require.Equal(t, time.March, cfg.CommitsBefore.Month())
}

func TestIgnoreConfig_UnmarshalYAML_Empty(t *testing.T) {
	data := []byte("{}\n")
	var cfg IgnoreConfig
	require.NoError(t, yaml.Unmarshal(data, &cfg))
	require.True(t, cfg.IsEmpty())
}

func TestIgnoreConfig_UnmarshalYAML_InvalidDate(t *testing.T) {
	data := []byte("commits-before: not-a-date\n")
	var cfg IgnoreConfig
	require.Error(t, yaml.Unmarshal(data, &cfg))
}

func timePtr(t time.Time) *time.Time {
	return &t
}
