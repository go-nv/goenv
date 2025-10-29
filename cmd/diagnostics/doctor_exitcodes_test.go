package diagnostics

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorCommand_JSONOutput(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)

	// Clear GOENV_VERSION to avoid picking up .go-version from repo
	t.Setenv("GOENV_VERSION", "system")

	// Add GOENV_ROOT/bin to PATH to avoid PATH configuration errors
	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", filepath.Join(tmpDir, "bin")+string(os.PathListSeparator)+oldPath)

	// Create basic directory structure
	if err := os.MkdirAll(filepath.Join(tmpDir, "bin"), 0755); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "shims"), 0755); err != nil {
		t.Fatalf("Failed to create shims directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "versions"), 0755); err != nil {
		t.Fatalf("Failed to create versions directory: %v", err)
	}

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

	err := doctorCmd.RunE(doctorCmd, []string{})
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

	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nOutput:\n%s", err, output)
	}

	// Verify schema version
	if result.SchemaVersion != "1" {
		t.Errorf("Expected schema_version '1', got %s", result.SchemaVersion)
	}

	// Verify checks have IDs
	if len(result.Checks) == 0 {
		t.Fatal("Expected at least one check in JSON output")
	}

	for i, check := range result.Checks {
		if check.ID == "" {
			t.Errorf("Check %d (%s) is missing 'id' field", i, check.Name)
		}
		if check.Name == "" {
			t.Errorf("Check %d is missing 'name' field", i)
		}
		if check.Status == "" {
			t.Errorf("Check %d (%s) is missing 'status' field", i, check.Name)
		}
		// Verify status is one of the valid values
		if check.Status != "ok" && check.Status != "warning" && check.Status != "error" {
			t.Errorf("Check %d (%s) has invalid status: %s", i, check.Name, check.Status)
		}
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

	if result.Summary.Total != len(result.Checks) {
		t.Errorf("Summary total (%d) doesn't match check count (%d)", result.Summary.Total, len(result.Checks))
	}
	if result.Summary.OK != okCount {
		t.Errorf("Summary OK count (%d) doesn't match actual (%d)", result.Summary.OK, okCount)
	}
	if result.Summary.Warnings != warningCount {
		t.Errorf("Summary warnings count (%d) doesn't match actual (%d)", result.Summary.Warnings, warningCount)
	}
	if result.Summary.Errors != errorCount {
		t.Errorf("Summary errors count (%d) doesn't match actual (%d)", result.Summary.Errors, errorCount)
	}

	t.Logf("JSON output verified: %d checks, %d OK, %d warnings, %d errors",
		result.Summary.Total, result.Summary.OK, result.Summary.Warnings, result.Summary.Errors)
}

func TestDoctorCommand_ExitCodes(t *testing.T) {
	tests := []struct {
		name         string
		failOn       string
		forceWarning bool // Simulate a warning condition
		forceError   bool // Simulate an error condition
		expectedExit int
	}{
		{
			name:         "no issues, default fail-on",
			failOn:       "error",
			forceWarning: false,
			forceError:   false,
			expectedExit: -1, // No exit call
		},
		{
			name:         "warnings only, fail-on error (default)",
			failOn:       "error",
			forceWarning: true,
			forceError:   false,
			expectedExit: -1, // No exit call (warnings don't trigger exit)
		},
		{
			name:         "warnings only, fail-on warning",
			failOn:       "warning",
			forceWarning: true,
			forceError:   false,
			expectedExit: 2, // Exit code 2 for warnings
		},
		{
			name:         "errors present, fail-on error",
			failOn:       "error",
			forceWarning: false,
			forceError:   true,
			expectedExit: 1, // Exit code 1 for errors
		},
		{
			name:         "errors present, fail-on warning",
			failOn:       "warning",
			forceWarning: false,
			forceError:   true,
			expectedExit: 1, // Exit code 1 for errors (takes precedence)
		},
		{
			name:         "both errors and warnings, fail-on warning",
			failOn:       "warning",
			forceWarning: true,
			forceError:   true,
			expectedExit: 1, // Exit code 1 (errors take precedence)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("GOENV_ROOT", tmpDir)
			t.Setenv("GOENV_DIR", tmpDir)

			// Set up environment based on test needs
			if tt.forceError {
				// Create error conditions:
				// 1. Set GOENV_VERSION to a non-existent version (causes "version not installed" error)
				t.Setenv("GOENV_VERSION", "999.999.999")
				// 2. Don't add bin to PATH (causes "PATH not configured" error)
				// Keep PATH as-is
			} else {
				// Clear GOENV_VERSION to avoid picking up .go-version from repo
				t.Setenv("GOENV_VERSION", "system")
				// Add GOENV_ROOT/bin to PATH to avoid PATH configuration errors
				oldPath := os.Getenv("PATH")
				t.Setenv("PATH", filepath.Join(tmpDir, "bin")+string(os.PathListSeparator)+oldPath)
			}

			// Create directories for non-error tests
			if !tt.forceError {
				if err := os.MkdirAll(filepath.Join(tmpDir, "bin"), 0755); err != nil {
					t.Fatalf("Failed to create bin directory: %v", err)
				}
				if err := os.MkdirAll(filepath.Join(tmpDir, "shims"), 0755); err != nil {
					t.Fatalf("Failed to create shims directory: %v", err)
				}
			}
			// Not creating versions directory can trigger warnings

			// Capture exit code
			exitCode := -1
			oldExit := doctorExit
			doctorExit = func(code int) {
				exitCode = code
			}
			defer func() { doctorExit = oldExit }()

			// Set fail-on flag
			oldFailOn := doctorFailOn
			doctorFailOn = tt.failOn
			defer func() { doctorFailOn = oldFailOn }()

			// Use JSON output for cleaner testing
			oldJSON := doctorJSON
			doctorJSON = true
			defer func() { doctorJSON = oldJSON }()

			buf := new(bytes.Buffer)
			doctorCmd.SetOut(buf)
			doctorCmd.SetErr(buf)

			err := doctorCmd.RunE(doctorCmd, []string{})
			if err != nil {
				t.Logf("RunE returned error (may be expected): %v", err)
			}

			if exitCode != tt.expectedExit {
				t.Errorf("Expected exit code %d, got %d\nOutput:\n%s", tt.expectedExit, exitCode, buf.String())
			}

			// Parse JSON to verify the check results
			var result struct {
				Summary struct {
					Warnings int `json:"warnings"`
					Errors   int `json:"errors"`
				} `json:"summary"`
			}
			if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			t.Logf("Exit code: %d, Warnings: %d, Errors: %d", exitCode, result.Summary.Warnings, result.Summary.Errors)
		})
	}
}

func TestDoctorCommand_CheckIDsConsistent(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)

	// Clear GOENV_VERSION to avoid picking up .go-version from repo
	t.Setenv("GOENV_VERSION", "system")

	// Add GOENV_ROOT/bin to PATH to avoid PATH configuration errors
	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", filepath.Join(tmpDir, "bin")+string(os.PathListSeparator)+oldPath)

	// Create basic structure
	if err := os.MkdirAll(filepath.Join(tmpDir, "bin"), 0755); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "shims"), 0755); err != nil {
		t.Fatalf("Failed to create shims directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "versions"), 0755); err != nil {
		t.Fatalf("Failed to create versions directory: %v", err)
	}

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

	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify all checks have IDs
	seenIDs := make(map[string]bool)
	for _, check := range result.Checks {
		if check.ID == "" {
			t.Errorf("Check '%s' is missing ID field", check.Name)
		}

		// Check for duplicate IDs (some checks may have same ID if they're conditional)
		if seenIDs[check.ID] {
			t.Logf("Note: Duplicate check ID '%s' for '%s' (may be intentional for conditional checks)", check.ID, check.Name)
		}
		seenIDs[check.ID] = true

		// Verify ID follows naming convention (lowercase, hyphen-separated)
		if !isValidCheckID(check.ID) {
			t.Errorf("Check ID '%s' doesn't follow naming convention (should be lowercase-with-hyphens)", check.ID)
		}
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
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)

	// Clear GOENV_VERSION to avoid picking up .go-version from repo
	t.Setenv("GOENV_VERSION", "system")

	// Add GOENV_ROOT/bin to PATH to avoid PATH configuration errors
	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", filepath.Join(tmpDir, "bin")+string(os.PathListSeparator)+oldPath)

	// Create directory structure
	if err := os.MkdirAll(filepath.Join(tmpDir, "bin"), 0755); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "shims"), 0755); err != nil {
		t.Fatalf("Failed to create shims directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "versions"), 0755); err != nil {
		t.Fatalf("Failed to create versions directory: %v", err)
	}

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
		if !strings.Contains(output, expected) {
			t.Errorf("Expected %q in human-readable output", expected)
		}
	}

	// Should not contain JSON markers
	if strings.Contains(output, `"schema_version"`) {
		t.Error("Human-readable output should not contain JSON")
	}
}
