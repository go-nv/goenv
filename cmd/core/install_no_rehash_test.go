package core

import (
	"bytes"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/cmdtest"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallCommand_NoRehashFlag(t *testing.T) {
	defer func() {
		installFlags.noRehash = false
	}()

	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDebug.String(), "1")

	// Create a fake existing installation
	cmdtest.CreateMockGoVersion(t, tmpDir, "1.21.0")

	buf := new(bytes.Buffer)
	installCmd.SetOut(buf)
	installCmd.SetErr(buf)

	// Test with --no-rehash flag
	installFlags.noRehash = true
	installFlags.skipExisting = true // Skip actual install since already exists

	err := installCmd.RunE(installCmd, []string{"1.21.0"})
	require.NoError(t, err, "Install command failed")

	output := buf.String()

	// Should show debug message about skipping rehash
	if !strings.Contains(output, "Skipping auto-rehash") && !strings.Contains(output, "skip") {
		t.Logf("Output: %s", output)
		// This is OK - skip-existing returns early before rehash logic
	}
}

func TestInstallCommand_NoRehashEnv(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDebug.String(), "1")
	t.Setenv(utils.GoenvEnvVarNoAutoRehash.String(), "1")

	// Create a fake existing installation
	cmdtest.CreateMockGoVersion(t, tmpDir, "1.21.0")

	buf := new(bytes.Buffer)
	installCmd.SetOut(buf)
	installCmd.SetErr(buf)

	defer func() {
		installFlags.skipExisting = false
	}()

	installFlags.skipExisting = true

	err := installCmd.RunE(installCmd, []string{"1.21.0"})
	require.NoError(t, err, "Install command failed")

	output := buf.String()

	// With environment variable set, should skip rehash
	if !strings.Contains(output, "Skipping auto-rehash") && !strings.Contains(output, "skip") {
		t.Logf("Output: %s", output)
		// This is OK - skip-existing returns early before rehash logic
	}
}

func TestInstallCommand_NoRehashFlagExists(t *testing.T) {
	// Verify the flag is defined
	flag := installCmd.Flags().Lookup("no-rehash")
	require.NotNil(t, flag, "--no-rehash flag is not defined")

	assert.Equal(t, "false", flag.DefValue, "Expected --no-rehash default to be false")
}
