package config

import "go-gitsemver/internal/semver"

func stringPtr(s string) *string        { return &s }
func intPtr(n int) *int                 { return &n }
func int64Ptr(n int64) *int64           { return &n }
func boolPtr(b bool) *bool              { return &b }
func strSlicePtr(ss []string) *[]string { return &ss }

func incrementPtr(s semver.IncrementStrategy) *semver.IncrementStrategy {
	return &s
}

func versioningModePtr(m semver.VersioningMode) *semver.VersioningMode {
	return &m
}

func commitMsgIncrPtr(m semver.CommitMessageIncrementMode) *semver.CommitMessageIncrementMode {
	return &m
}

func commitMsgConvPtr(c semver.CommitMessageConvention) *semver.CommitMessageConvention {
	return &c
}
