package diagnostics

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoctorCommand_JSONOutput(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	// Clear GOENV_VERSION to avoid picking up .go-version from repo
	t.Setenv(utils.GoenvEnvVarVersion.String(), "system")

	// Add GOENV_ROOT/bin to PATH to avoid PATH configuration errors
	oldPath := os.Getenv(utils.EnvVarPath)
	t.Setenv(utils.EnvVarPath, filepath.Join(tmpDir, "bin")+string(os.PathListSeparator)+oldPath)

	// Create basic directory structure
	err = utils.EnsureDir(filepath.Join(tmpDir, "bin"))
	require.NoError(t, err, "Failed to create bin directory")
	err = utils.EnsureDir(filepath.Join(tmpDir, "shims"))
	require.NoError(t, err, "Failed to create shims directory")
	err = utils.EnsureDir(filepath.Join(tmpDir, "versions"))
	require.NoError(t, err, "Failed to create versions directory")

	// Capture exit code
	exitCode := -1
	oldExit := doctorExit
	doctorExit = func(code int) {
		exitCode = code
	}
	defer func() { doctorExit = oldExit }()

	// Set JSON flag
	oldJSON := doctorJSON
	doctorJSON = true
	defer func() { doctorJSON = oldJSON }()

	buf := new(bytes.Buffer)
	doctorCmd.SetOut(buf)
	doctorCmd.SetErr(buf)

	err = doctorCmd.RunE(doctorCmd, []string{})
	if err != nil {
		t.Logf("RunE returned error (expected in some cases): %v", err)
	}

	t.Logf("Exit code captured: %d", exitCode)

	output := buf.String()

	// Verify JSON structure
	var result struct {
		SchemaVersion string `json:"schema_version"`
		Checks        []struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Status  string `json:"status"`
			Message string `json:"message"`
			Advice  string `json:"advice,omitempty"`
		} `json:"checks"`
		Summary struct {
			Total    int `json:"total"`
			OK       int `json:"ok"`
			Warnings int `json:"warnings"`
			Errors   int `json:"errors"`
		} `json:"summary"`
	}

	err = json.Unmarshal([]byte(output), &result)
	require.NoError(t, err, "Failed to parse JSON output: \\nOutput:\\n")

	// Verify schema version
	assert.Equal(t, "1", result.SchemaVersion, "Expected schema_version '1'")

	// Verify checks have IDs
	require.NotEmpty(t, result.Checks, "Expected at least one check in JSON output")

	for _, check := range result.Checks {
		assert.NotEmpty(t, check.ID, "Check () is missing 'id' field")
		assert.NotEmpty(t, check.Name, "Check is missing 'name' field")
		assert.NotEmpty(t, check.Status, "Check () is missing 'status' field")
		// Verify status is one of the valid values
		assert.False(t, check.Status != "ok" && check.Status != "warning" && check.Status != "error", "Check () has invalid status")
	}

	// Verify summary counts match
	okCount := 0
	warningCount := 0
	errorCount := 0
	for _, check := range result.Checks {
		switch check.Status {
		case "ok":
			okCount++
		case "warning":
			warningCount++
		case "error":
			errorCount++
		}
	}

	assert.Equal(t, len(result.Checks), result.Summary.Total, "Summary total () doesn't match check count ()")
	assert.Equal(t, okCount, result.Summary.OK, "Summary OK count () doesn't match actual ()")
	assert.Equal(t, warningCount, result.Summary.Warnings, "Summary warnings count () doesn't match actual ()")
	assert.Equal(t, errorCount, result.Summary.Errors, "Summary errors count () doesn't match actual ()")

	t.Logf("JSON output verified: %d checks, %d OK, %d warnings, %d errors",
		result.Summary.Total, result.Summary.OK, result.Summary.Warnings, result.Summary.Errors)
}

func TestDoctorCommand_ExitCodes(t *testing.T) {
	var err error
	tests := []struct {
		name         string
		failOn       FailOn
		forceWarning bool // Simulate a warning condition
		forceError   bool // Simulate an error condition
		expectedExit int
	}{
		{
			name:         "no issues, default fail-on",
			failOn:       FailOnError,
			forceWarning: false,
			forceError:   false,
			expectedExit: -1, // No exit call
		},
		{
			name:         "warnings only, fail-on error (default)",
			failOn:       FailOnError,
			forceWarning: true,
			forceError:   false,
			expectedExit: -1, // No exit call (warnings don't trigger exit)
		},
		{
			name:         "warnings only, fail-on warning",
			failOn:       FailOnWarning,
			forceWarning: true,
			forceError:   false,
			expectedExit: 2, // Exit code 2 for warnings
		},
		{
			name:         "errors present, fail-on error",
			failOn:       FailOnError,
			forceWarning: false,
			forceError:   true,
			expectedExit: 1, // Exit code 1 for errors
		},
		{
			name:         "errors present, fail-on warning",
			failOn:       FailOnWarning,
			forceWarning: false,
			forceError:   true,
			expectedExit: 1, // Exit code 1 for errors (takes precedence)
		},
		{
			name:         "both errors and warnings, fail-on warning",
			failOn:       FailOnWarning,
			forceWarning: true,
			forceError:   true,
			expectedExit: 1, // Exit code 1 (errors take precedence)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
			t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

			// Set HOME to tmpDir to avoid checking user's real profile files
			t.Setenv(utils.EnvVarHome, tmpDir)
			t.Setenv(utils.EnvVarUserProfile, tmpDir) // Windows uses USERPROFILE instead of HOME			// Clear GOENV_SHELL to skip shell initialization checks
			t.Setenv(utils.GoenvEnvVarShell.String(), "")

			// Set up environment based on test needs
			if tt.forceError {
				// Create error conditions:
				// 1. Set GOENV_VERSION to a non-existent version (causes "version not installed" error)
				t.Setenv(utils.GoenvEnvVarVersion.String(), "999.999.999")
				// 2. Don't add bin to PATH (causes "PATH not configured" error)
				// Keep PATH as-is
			} else {
				// Clear GOENV_VERSION to avoid picking up .go-version from repo
				t.Setenv(utils.GoenvEnvVarVersion.String(), "system")
				// Add both GOENV_ROOT/bin and shims to PATH to avoid PATH configuration errors
				oldPath := os.Getenv(utils.EnvVarPath)
				shimsPath := filepath.Join(tmpDir, "shims")
				binPath := filepath.Join(tmpDir, "bin")
				t.Setenv(utils.EnvVarPath, shimsPath+string(os.PathListSeparator)+binPath+string(os.PathListSeparator)+oldPath)
			}

			// Create directories for non-error tests
			if !tt.forceError {
				err = utils.EnsureDir(filepath.Join(tmpDir, "bin"))
				require.NoError(t, err, "Failed to create bin directory")
				err = utils.EnsureDir(filepath.Join(tmpDir, "shims"))
				require.NoError(t, err, "Failed to create shims directory")
				// Only create versions directory if not forcing warnings
				// (empty versions directory triggers "no versions installed" warning)
				if !tt.forceWarning {
					err = utils.EnsureDir(filepath.Join(tmpDir, "versions"))
					require.NoError(t, err, "Failed to create versions directory")
				}
			}

			// Capture exit code
			exitCode := -1
			oldExit := doctorExit
			doctorExit = func(code int) {
				exitCode = code
			}
			defer func() { doctorExit = oldExit }()

			// Set fail-on flag (need to set both the string and enum)
			oldFailOn := doctorFailOn
			oldFailOnStr := doctorFailOnStr
			doctorFailOn = tt.failOn
			doctorFailOnStr = string(tt.failOn)
			defer func() {
				doctorFailOn = oldFailOn
				doctorFailOnStr = oldFailOnStr
			}()

			// Use JSON output for cleaner testing
			oldJSON := doctorJSON
			doctorJSON = true
			defer func() { doctorJSON = oldJSON }()

			buf := new(bytes.Buffer)
			doctorCmd.SetOut(buf)
			doctorCmd.SetErr(buf)

			err = doctorCmd.RunE(doctorCmd, []string{})
			if err != nil {
				t.Logf("RunE returned error (may be expected): %v", err)
			}

			assert.Equal(t, tt.expectedExit, exitCode, "Expected exit code %v", buf.String())

			// Parse JSON to verify the check results
			var result struct {
				Summary struct {
					Warnings int `json:"warnings"`
					Errors   int `json:"errors"`
				} `json:"summary"`
			}
			err = json.Unmarshal(buf.Bytes(), &result)
			require.NoError(t, err, "Failed to parse JSON")

			t.Logf("Exit code: %d, Warnings: %d, Errors: %d", exitCode, result.Summary.Warnings, result.Summary.Errors)
		})
	}
}

func TestDoctorCommand_CheckIDsConsistent(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	// Clear GOENV_VERSION to avoid picking up .go-version from repo
	t.Setenv(utils.GoenvEnvVarVersion.String(), "system")

	// Add GOENV_ROOT/bin to PATH to avoid PATH configuration errors
	oldPath := os.Getenv(utils.EnvVarPath)
	t.Setenv(utils.EnvVarPath, filepath.Join(tmpDir, "bin")+string(os.PathListSeparator)+oldPath)

	// Create basic structure
	err = utils.EnsureDir(filepath.Join(tmpDir, "bin"))
	require.NoError(t, err, "Failed to create bin directory")
	err = utils.EnsureDir(filepath.Join(tmpDir, "shims"))
	require.NoError(t, err, "Failed to create shims directory")
	err = utils.EnsureDir(filepath.Join(tmpDir, "versions"))
	require.NoError(t, err, "Failed to create versions directory")

	// Override exit
	oldExit := doctorExit
	doctorExit = func(code int) {}
	defer func() { doctorExit = oldExit }()

	// Use JSON output
	oldJSON := doctorJSON
	doctorJSON = true
	defer func() { doctorJSON = oldJSON }()

	buf := new(bytes.Buffer)
	doctorCmd.SetOut(buf)
	doctorCmd.SetErr(buf)

	_ = doctorCmd.RunE(doctorCmd, []string{})

	// Parse JSON
	var result struct {
		Checks []struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"checks"`
	}

	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err, "Failed to parse JSON")

	// Verify all checks have IDs
	seenIDs := make(map[string]bool)
	for _, check := range result.Checks {
		assert.NotEmpty(t, check.ID)

		// Check for duplicate IDs (some checks may have same ID if they're conditional)
		if seenIDs[check.ID] {
			t.Logf("Note: Duplicate check ID '%s' for '%s' (may be intentional for conditional checks)", check.ID, check.Name)
		}
		seenIDs[check.ID] = true

		// Verify ID follows naming convention (lowercase, hyphen-separated)
		assert.True(t, isValidCheckID(check.ID))
	}

	t.Logf("Verified %d checks, %d unique IDs", len(result.Checks), len(seenIDs))
}

func isValidCheckID(id string) bool {
	if id == "" {
		return false
	}
	// Check if ID is lowercase with hyphens
	for _, c := range id {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			return false
		}
	}
	// Shouldn't start or end with hyphen
	if strings.HasPrefix(id, "-") || strings.HasSuffix(id, "-") {
		return false
	}
	return true
}

func TestDoctorCommand_HumanReadableOutput(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	// Clear GOENV_VERSION to avoid picking up .go-version from repo
	t.Setenv(utils.GoenvEnvVarVersion.String(), "system")

	// Add GOENV_ROOT/bin to PATH to avoid PATH configuration errors
	oldPath := os.Getenv(utils.EnvVarPath)
	t.Setenv(utils.EnvVarPath, filepath.Join(tmpDir, "bin")+string(os.PathListSeparator)+oldPath)

	// Create directory structure
	err = utils.EnsureDir(filepath.Join(tmpDir, "bin"))
	require.NoError(t, err, "Failed to create bin directory")
	err = utils.EnsureDir(filepath.Join(tmpDir, "shims"))
	require.NoError(t, err, "Failed to create shims directory")
	err = utils.EnsureDir(filepath.Join(tmpDir, "versions"))
	require.NoError(t, err, "Failed to create versions directory")

	// Override exit
	oldExit := doctorExit
	doctorExit = func(code int) {}
	defer func() { doctorExit = oldExit }()

	// Use human-readable output (not JSON)
	oldJSON := doctorJSON
	doctorJSON = false
	defer func() { doctorJSON = oldJSON }()

	buf := new(bytes.Buffer)
	doctorCmd.SetOut(buf)
	doctorCmd.SetErr(buf)

	_ = doctorCmd.RunE(doctorCmd, []string{})

	output := buf.String()

	// Verify human-readable format
	expectedStrings := []string{
		"Checking goenv installation",
		"Diagnostic Results",
		"Summary:",
		"OK,",
	}

	for _, expected := range expectedStrings {
		assert.Contains(t, output, expected, "Expected in human-readable output %v", expected)
	}

	// Should not contain JSON markers
	assert.NotContains(t, output, `"schema_version"`, "Human-readable output should not contain JSON")
}
