package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRootCmd_HasExpectedFlags(t *testing.T) {
	flags := rootCmd.PersistentFlags()

	require.NotNil(t, flags.Lookup("path"))
	require.NotNil(t, flags.Lookup("branch"))
	require.NotNil(t, flags.Lookup("commit"))
	require.NotNil(t, flags.Lookup("config"))
	require.NotNil(t, flags.Lookup("output"))
	require.NotNil(t, flags.Lookup("show-variable"))
	require.NotNil(t, flags.Lookup("show-config"))
	require.NotNil(t, flags.Lookup("explain"))
	require.NotNil(t, flags.Lookup("verbosity"))
}

func TestRootCmd_HasVersionSubcommand(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == "version" {
			found = true
			break
		}
	}
	require.True(t, found, "version subcommand should be registered")
}
