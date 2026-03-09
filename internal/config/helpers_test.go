package config

import (
	"testing"

	"github.com/MyCarrier-DevOps/go-gitsemver/internal/semver"
	"github.com/stretchr/testify/require"
)

func TestStringPtr(t *testing.T) {
	p := stringPtr("hello")
	require.NotNil(t, p)
	require.Equal(t, "hello", *p)

	// Ensure it's a new pointer each time.
	p2 := stringPtr("hello")
	require.NotSame(t, p, p2)
}

func TestIntPtr(t *testing.T) {
	p := intPtr(42)
	require.NotNil(t, p)
	require.Equal(t, 42, *p)
}

func TestInt64Ptr(t *testing.T) {
	p := int64Ptr(123456789)
	require.NotNil(t, p)
	require.Equal(t, int64(123456789), *p)
}

func TestBoolPtr(t *testing.T) {
	pTrue := boolPtr(true)
	require.NotNil(t, pTrue)
	require.True(t, *pTrue)

	pFalse := boolPtr(false)
	require.NotNil(t, pFalse)
	require.False(t, *pFalse)
}

func TestStrSlicePtr(t *testing.T) {
	ss := []string{"a", "b", "c"}
	p := strSlicePtr(ss)
	require.NotNil(t, p)
	require.Equal(t, ss, *p)
}

func TestIncrementPtr(t *testing.T) {
	p := incrementPtr(semver.IncrementStrategyMinor)
	require.NotNil(t, p)
	require.Equal(t, semver.IncrementStrategyMinor, *p)
}

func TestVersioningModePtr(t *testing.T) {
	p := versioningModePtr(semver.VersioningModeMainline)
	require.NotNil(t, p)
	require.Equal(t, semver.VersioningModeMainline, *p)
}

func TestCommitMsgIncrPtr(t *testing.T) {
	p := commitMsgIncrPtr(semver.CommitMessageIncrementDisabled)
	require.NotNil(t, p)
	require.Equal(t, semver.CommitMessageIncrementDisabled, *p)
}

func TestCommitMsgConvPtr(t *testing.T) {
	p := commitMsgConvPtr(semver.CommitMessageConventionBoth)
	require.NotNil(t, p)
	require.Equal(t, semver.CommitMessageConventionBoth, *p)
}
