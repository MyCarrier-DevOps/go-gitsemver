package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionCmd_Output(t *testing.T) {
	Version = "1.0.0-test"
	defer func() { Version = "dev" }()

	var buf bytes.Buffer
	versionCmd.SetOut(&buf)
	versionCmd.Run(versionCmd, nil)
	require.Equal(t, "1.0.0-test\n", buf.String())
}
