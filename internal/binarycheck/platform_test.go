package binarycheck

import (
	"runtime"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/osinfo"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPlatformInfo(t *testing.T) {
	info := GetPlatformInfo()

	require.NotNil(t, info, "GetPlatformInfo returned nil")

	assert.Equal(t, osinfo.OS(), info.OS, "Expected OS")

	assert.Equal(t, osinfo.Arch(), info.Arch, "Expected Arch")

	assert.NotNil(t, info.Details, "Details map should not be nil")

	t.Logf("Platform: %s/%s", info.OS, info.Arch)
	for k, v := range info.Details {
		t.Logf("  %s: %s", k, v)
	}
}

func TestCheckMacOSDeploymentTarget(t *testing.T) {
	if !osinfo.IsMacOS() {
		t.Skip("macOS deployment target check only works on macOS")
	}

	// Test with current go binary
	goBinary, err := findGoBinary()
	if err != nil {
		t.Skipf("Could not find go binary: %v", err)
	}

	info, issues := CheckMacOSDeploymentTarget(goBinary)

	if info == nil {
		t.Log("No macOS info returned (binary may not be Mach-O)")
		return
	}

	t.Logf("Deployment Target: %s", info.DeploymentTarget)
	t.Logf("Has Version Min: %v", info.HasVersionMin)

	if len(issues) > 0 {
		t.Log("Issues found:")
		for _, issue := range issues {
			t.Logf("  [%s] %s: %s", issue.Severity, issue.Message, issue.Hint)
		}
	}
}

func TestCheckWindowsCompiler(t *testing.T) {
	if !utils.IsWindows() {
		t.Skip("Windows compiler check only works on Windows")
	}

	info, issues := CheckWindowsCompiler()

	require.NotNil(t, info, "CheckWindowsCompiler returned nil on Windows")

	t.Logf("Compiler: %s", info.Compiler)
	t.Logf("Has cl.exe: %v", info.HasCLExe)
	t.Logf("Has VC Runtime: %v", info.HasVCRuntime)

	// Should detect at least one compiler or report it
	assert.False(t, info.Compiler == "unknown" && len(issues) == 0, "No compiler detected but no issues reported")

	if len(issues) > 0 {
		t.Log("Issues found:")
		for _, issue := range issues {
			t.Logf("  [%s] %s: %s", issue.Severity, issue.Message, issue.Hint)
		}
	}
}

func TestCheckWindowsARM64(t *testing.T) {
	if !utils.IsWindows() {
		t.Skip("Windows ARM64 check only works on Windows")
	}

	info, issues := CheckWindowsARM64()

	require.NotNil(t, info, "CheckWindowsARM64 returned nil on Windows")

	t.Logf("Process Mode: %s", info.ProcessMode)
	t.Logf("Is ARM64EC: %v", info.IsARM64EC)

	if len(issues) > 0 {
		t.Log("Issues found:")
		for _, issue := range issues {
			t.Logf("  [%s] %s: %s", issue.Severity, issue.Message, issue.Hint)
		}
	}
}

func TestCheckLinuxKernelVersion(t *testing.T) {
	if !osinfo.IsLinux() {
		t.Skip("Linux kernel check only works on Linux")
	}

	info, issues := CheckLinuxKernelVersion()

	require.NotNil(t, info, "CheckLinuxKernelVersion returned nil on Linux")

	t.Logf("Kernel Version: %s", info.KernelVersion)
	t.Logf("Kernel Major: %d", info.KernelMajor)
	t.Logf("Kernel Minor: %d", info.KernelMinor)
	t.Logf("Kernel Patch: %d", info.KernelPatch)
	t.Logf("Libc Type: %s", info.LibcType)
	t.Logf("Glibc Version: %s", info.GlibcVersion)

	// Basic sanity checks
	assert.NotEmpty(t, info.KernelVersion, "Kernel version should not be empty")

	if info.KernelMajor < 2 {
		t.Error("Kernel major version seems too low")
	}

	if len(issues) > 0 {
		t.Log("Issues found:")
		for _, issue := range issues {
			t.Logf("  [%s] %s: %s", issue.Severity, issue.Message, issue.Hint)
		}
	}
}

func TestCheckWindowsScriptShims(t *testing.T) {
	// Create a temporary script file
	tmpFile := t.TempDir() + "/test-script"

	// Write a bash script
	scriptContent := "#!/bin/bash\necho 'Hello World'\n"
	writeFile(t, tmpFile, []byte(scriptContent))

	issues := CheckWindowsScriptShims(tmpFile)

	// On Windows, should suggest creating wrapper
	if utils.IsWindows() {
		assert.NotEqual(t, 0, len(issues), "Expected issues for script without .cmd/.ps1 extension on Windows")

		hasWrapperSuggestion := false
		for _, issue := range issues {
			if issue.Severity == "warning" {
				hasWrapperSuggestion = true
			}
			t.Logf("[%s] %s: %s", issue.Severity, issue.Message, issue.Hint)
		}

		assert.True(t, hasWrapperSuggestion, "Expected warning about wrapper shim on Windows")
	} else {
		// On non-Windows, should return nil
		assert.Nil(t, issues, "Expected nil issues on non-Windows platforms")
	}
}

func TestSuggestGlibcCompatibility(t *testing.T) {
	tests := []struct {
		name           string
		requiredGlibc  string
		currentGlibc   string
		expectIssues   bool
		expectSeverity string
	}{
		{
			name:          "compatible versions",
			requiredGlibc: "2.27",
			currentGlibc:  "2.31",
			expectIssues:  false,
		},
		{
			name:           "incompatible versions",
			requiredGlibc:  "2.35",
			currentGlibc:   "2.27",
			expectIssues:   true,
			expectSeverity: "error",
		},
		{
			name:          "same versions",
			requiredGlibc: "2.31",
			currentGlibc:  "2.31",
			expectIssues:  false,
		},
		{
			name:           "major version mismatch",
			requiredGlibc:  "3.0",
			currentGlibc:   "2.35",
			expectIssues:   true,
			expectSeverity: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip on non-Linux
			if !osinfo.IsLinux() {
				t.Skip("glibc compatibility check only works on Linux")
			}

			issues := SuggestGlibcCompatibility(tt.requiredGlibc, tt.currentGlibc)

			hasIssues := len(issues) > 0
			assert.Equal(t, tt.expectIssues, hasIssues, "Expected issues")

			if tt.expectIssues && len(issues) > 0 {
				// Check first issue has expected severity
				assert.Equal(t, tt.expectSeverity, issues[0].Severity, "Expected severity")

				// Should suggest container-based build
				foundContainerSuggestion := false
				for _, issue := range issues {
					if issue.Severity == "error" && strings.Contains(issue.Hint, "container") {
						foundContainerSuggestion = true
					}
					t.Logf("[%s] %s: %s", issue.Severity, issue.Message, issue.Hint)
				}

				assert.True(t, foundContainerSuggestion, "Expected suggestion to build in older container")
			}
		})
	}
}

func TestIsVersionCompatible(t *testing.T) {
	tests := []struct {
		current  string
		minimum  string
		expected bool
	}{
		{"10.15.0", "10.14.0", true},
		{"10.15.0", "10.15.0", true},
		{"10.15.0", "10.16.0", false},
		{"11.0", "10.15", true},
		{"10.14", "10.15", false},
		{"12.5.1", "12.5.1", true},
		{"13.0", "12.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.current+"_vs_"+tt.minimum, func(t *testing.T) {
			result := isVersionCompatible(tt.current, tt.minimum)
			assert.Equal(t, tt.expected, result, "isVersionCompatible(, ) = %v %v", tt.current, tt.minimum)
		})
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		version  string
		expected []int
	}{
		{"2.31", []int{2, 31}},
		{"2.27.0", []int{2, 27, 0}},
		{"10.15.7", []int{10, 15, 7}},
		{"1.0", []int{1, 0}},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			result := parseVersion(tt.version)
			assert.Len(t, result, len(tt.expected), "parseVersion() length = %v", tt.version)
			for i := range result {
				assert.Equal(t, tt.expected[i], result[i], "parseVersion()[] = %v", tt.version)
			}
		})
	}
}

// Helper functions

func findGoBinary() (string, error) {
	// Try to find go binary
	goBinary := "/usr/local/go/bin/go"
	if utils.IsWindows() {
		goBinary = "C:\\Go\\bin\\go.exe"
	}

	// Check if exists
	if _, err := fileExists(goBinary); err == nil {
		return goBinary, nil
	}

	// Try current GOROOT
	goroot := runtime.GOROOT()
	if goroot != "" {
		goBinary = goroot + "/bin/go"
		if utils.IsWindows() {
			goBinary += ".exe"
		}
		if _, err := fileExists(goBinary); err == nil {
			return goBinary, nil
		}
	}

	return "", nil
}

func fileExists(path string) (bool, error) {
	exists := utils.PathExists(path)
	return exists, nil
}

func writeFile(t *testing.T, path string, data []byte) {
	testutil.WriteTestFile(t, path, data, utils.PermFileExecutable)
}
