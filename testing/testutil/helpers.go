// Package testutil provides low-level test utilities with zero internal dependencies.
// This package can be safely imported by any test file without causing import cycles.
package testutil

import (
	"os"
	"strings"
	"testing"
)

// WriteTestFile writes a file for testing purposes with standardized error handling.
// If the write fails, it immediately fails the test with t.Fatalf.
// The optional msg parameter allows specifying a custom error message context.
// If not provided, defaults to "Failed to write test file <path>".
//
// This helper reduces test code boilerplate and provides consistent error messages.
// Unlike internal/cmdtest helpers, this has zero internal dependencies and can be
// used by any test package without creating import cycles.
//
// Accepts testing.TB so it works with both *testing.T and *testing.B.
//
// Example:
//
//	WriteTestFile(t, path, data, utils.PermFileDefault)  // Default error message
//	WriteTestFile(t, path, data, utils.PermFileDefault, "Failed to create config file")  // Custom message
func WriteTestFile(tb testing.TB, path string, content []byte, perm os.FileMode, msg ...string) {
	tb.Helper()
	if err := os.WriteFile(path, content, perm); err != nil {
		if len(msg) > 0 && msg[0] != "" {
			tb.Fatalf("%s: %v", msg[0], err)
		} else {
			tb.Fatalf("Failed to write test file %s: %v", path, err)
		}
	}
}

// StripDeprecationWarning removes deprecation warnings from command output.
// This is useful for testing legacy commands that now show deprecation warnings.
// The deprecation warning format is:
//
//	Deprecation warning: ...
//	  Modern command: ...
//	  See: ...
//	[blank line]
//	[actual output]
//
// This function removes the warning block and returns only the actual output.
func StripDeprecationWarning(output string) string {
	// Check if output contains a deprecation warning
	if !strings.Contains(output, "Deprecation warning:") {
		return strings.TrimSpace(output)
	}

	lines := strings.Split(output, "\n")

	// Find the blank line that separates warning from actual output
	// The warning block is: "Deprecation warning:" + 2 indent lines + blank line
	blankLineIdx := -1
	for i, line := range lines {
		if i > 0 && strings.HasPrefix(lines[0], "Deprecation warning:") && strings.TrimSpace(line) == "" {
			blankLineIdx = i
			break
		}
	}

	if blankLineIdx == -1 {
		// No blank line found, warning only
		return ""
	}

	// Return everything after the blank line
	if blankLineIdx+1 >= len(lines) {
		return ""
	}

	result := strings.Join(lines[blankLineIdx+1:], "\n")
	// Only trim trailing whitespace to preserve formatting of the output
	return strings.TrimRight(result, " \t\n\r")
}
