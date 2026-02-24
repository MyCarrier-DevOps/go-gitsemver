package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFindConfigFile_Found(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "go-gitsemver.yml")
	require.NoError(t, os.WriteFile(path, []byte("mode: ContinuousDelivery\n"), 0o644))

	found := findConfigFile(dir)
	require.Equal(t, path, found)
}

func TestFindConfigFile_GitVersionYml(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "GitVersion.yml")
	require.NoError(t, os.WriteFile(path, []byte("mode: ContinuousDelivery\n"), 0o644))

	found := findConfigFile(dir)
	require.Equal(t, path, found)
}

func TestFindConfigFile_NotFound(t *testing.T) {
	dir := t.TempDir()
	found := findConfigFile(dir)
	require.Empty(t, found)
}

func TestFindConfigFile_PrefersGitVersionYml(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "GitVersion.yml"), []byte(""), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go-gitsemver.yml"), []byte(""), 0o644))

	found := findConfigFile(dir)
	require.Equal(t, filepath.Join(dir, "GitVersion.yml"), found)
}

func TestLoadConfig_NoFile(t *testing.T) {
	dir := t.TempDir()
	flagConfig = ""
	cfg, err := loadConfig(dir)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.NotNil(t, cfg.Branches)
}

func TestLoadConfig_WithFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "go-gitsemver.yml")
	require.NoError(t, os.WriteFile(path, []byte("next-version: 5.0.0\n"), 0o644))

	flagConfig = path
	defer func() { flagConfig = "" }()

	cfg, err := loadConfig(dir)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Equal(t, "5.0.0", *cfg.NextVersion)
}

func TestLoadConfig_InvalidFile(t *testing.T) {
	flagConfig = "/nonexistent/path/config.yml"
	defer func() { flagConfig = "" }()

	_, err := loadConfig(t.TempDir())
	require.Error(t, err)
}

func TestWriteOutput_ShowVariable(t *testing.T) {
	vars := map[string]string{"SemVer": "1.2.3"}

	// Save and restore stdout.
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	flagShowVariable = "SemVer"
	defer func() { flagShowVariable = "" }()

	err := writeOutput(vars)
	require.NoError(t, err)

	w.Close()
	os.Stdout = old

	buf := make([]byte, 128)
	n, _ := r.Read(buf)
	require.Equal(t, "1.2.3\n", string(buf[:n]))
}

func TestWriteOutput_JSON(t *testing.T) {
	vars := map[string]string{"A": "1"}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	flagOutput = "json"
	flagShowVariable = ""
	defer func() { flagOutput = "" }()

	err := writeOutput(vars)
	require.NoError(t, err)

	w.Close()
	os.Stdout = old

	buf := make([]byte, 256)
	n, _ := r.Read(buf)
	require.Contains(t, string(buf[:n]), `"A": "1"`)
}

func TestWriteOutput_Default(t *testing.T) {
	vars := map[string]string{"A": "1", "B": "2"}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	flagOutput = ""
	flagShowVariable = ""

	err := writeOutput(vars)
	require.NoError(t, err)

	w.Close()
	os.Stdout = old

	buf := make([]byte, 256)
	n, _ := r.Read(buf)
	out := string(buf[:n])
	require.Contains(t, out, "A=1")
	require.Contains(t, out, "B=2")
}

func TestWriteOutput_UnknownFormat(t *testing.T) {
	flagOutput = "xml"
	flagShowVariable = ""
	defer func() { flagOutput = "" }()

	err := writeOutput(map[string]string{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown output format")
}

func TestShowConfig(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cfg, err := loadConfig(t.TempDir())
	require.NoError(t, err)

	err = showConfig(cfg)
	require.NoError(t, err)

	w.Close()
	os.Stdout = old

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	out := string(buf[:n])
	require.Contains(t, out, "TagPrefix")
}
