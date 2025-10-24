package binarycheck

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestGetPlatformInfo(t *testing.T) {
	info := GetPlatformInfo()

	if info == nil {
		t.Fatal("GetPlatformInfo returned nil")
	}

	if info.OS != runtime.GOOS {
		t.Errorf("Expected OS %s, got %s", runtime.GOOS, info.OS)
	}

	if info.Arch != runtime.GOARCH {
		t.Errorf("Expected Arch %s, got %s", runtime.GOARCH, info.Arch)
	}

	if info.Details == nil {
		t.Error("Details map should not be nil")
	}

	t.Logf("Platform: %s/%s", info.OS, info.Arch)
	for k, v := range info.Details {
		t.Logf("  %s: %s", k, v)
	}
}

func TestCheckMacOSDeploymentTarget(t *testing.T) {
	if runtime.GOOS != "darwin" {
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
	if runtime.GOOS != "windows" {
		t.Skip("Windows compiler check only works on Windows")
	}

	info, issues := CheckWindowsCompiler()

	if info == nil {
		t.Fatal("CheckWindowsCompiler returned nil on Windows")
	}

	t.Logf("Compiler: %s", info.Compiler)
	t.Logf("Has cl.exe: %v", info.HasCLExe)
	t.Logf("Has VC Runtime: %v", info.HasVCRuntime)

	// Should detect at least one compiler or report it
	if info.Compiler == "unknown" && len(issues) == 0 {
		t.Error("No compiler detected but no issues reported")
	}

	if len(issues) > 0 {
		t.Log("Issues found:")
		for _, issue := range issues {
			t.Logf("  [%s] %s: %s", issue.Severity, issue.Message, issue.Hint)
		}
	}
}

func TestCheckWindowsARM64(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows ARM64 check only works on Windows")
	}

	info, issues := CheckWindowsARM64()

	if info == nil {
		t.Fatal("CheckWindowsARM64 returned nil on Windows")
	}

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
	if runtime.GOOS != "linux" {
		t.Skip("Linux kernel check only works on Linux")
	}

	info, issues := CheckLinuxKernelVersion()

	if info == nil {
		t.Fatal("CheckLinuxKernelVersion returned nil on Linux")
	}

	t.Logf("Kernel Version: %s", info.KernelVersion)
	t.Logf("Kernel Major: %d", info.KernelMajor)
	t.Logf("Kernel Minor: %d", info.KernelMinor)
	t.Logf("Kernel Patch: %d", info.KernelPatch)
	t.Logf("Libc Type: %s", info.LibcType)
	t.Logf("Glibc Version: %s", info.GlibcVersion)

	// Basic sanity checks
	if info.KernelVersion == "" {
		t.Error("Kernel version should not be empty")
	}

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
	if err := writeFile(tmpFile, []byte(scriptContent)); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	issues := CheckWindowsScriptShims(tmpFile)

	// On Windows, should suggest creating wrapper
	if runtime.GOOS == "windows" {
		if len(issues) == 0 {
			t.Error("Expected issues for script without .cmd/.ps1 extension on Windows")
		}

		hasWrapperSuggestion := false
		for _, issue := range issues {
			if issue.Severity == "warning" {
				hasWrapperSuggestion = true
			}
			t.Logf("[%s] %s: %s", issue.Severity, issue.Message, issue.Hint)
		}

		if !hasWrapperSuggestion {
			t.Error("Expected warning about wrapper shim on Windows")
		}
	} else {
		// On non-Windows, should return nil
		if issues != nil {
			t.Error("Expected nil issues on non-Windows platforms")
		}
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
			name:         "compatible versions",
			requiredGlibc: "2.27",
			currentGlibc: "2.31",
			expectIssues: false,
		},
		{
			name:           "incompatible versions",
			requiredGlibc:  "2.35",
			currentGlibc:   "2.27",
			expectIssues:   true,
			expectSeverity: "error",
		},
		{
			name:         "same versions",
			requiredGlibc: "2.31",
			currentGlibc: "2.31",
			expectIssues: false,
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
			if runtime.GOOS != "linux" {
				t.Skip("glibc compatibility check only works on Linux")
			}

			issues := SuggestGlibcCompatibility(tt.requiredGlibc, tt.currentGlibc)

			hasIssues := len(issues) > 0
			if hasIssues != tt.expectIssues {
				t.Errorf("Expected issues: %v, got issues: %v", tt.expectIssues, hasIssues)
			}

			if tt.expectIssues && len(issues) > 0 {
				// Check first issue has expected severity
				if issues[0].Severity != tt.expectSeverity {
					t.Errorf("Expected severity %s, got %s", tt.expectSeverity, issues[0].Severity)
				}

				// Should suggest container-based build
				foundContainerSuggestion := false
				for _, issue := range issues {
					if issue.Severity == "error" && contains(issue.Hint, "container") {
						foundContainerSuggestion = true
					}
					t.Logf("[%s] %s: %s", issue.Severity, issue.Message, issue.Hint)
				}

				if !foundContainerSuggestion {
					t.Error("Expected suggestion to build in older container")
				}
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
			if result != tt.expected {
				t.Errorf("isVersionCompatible(%s, %s) = %v, want %v", tt.current, tt.minimum, result, tt.expected)
			}
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
			if len(result) != len(tt.expected) {
				t.Errorf("parseVersion(%s) length = %d, want %d", tt.version, len(result), len(tt.expected))
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("parseVersion(%s)[%d] = %d, want %d", tt.version, i, result[i], tt.expected[i])
				}
			}
		})
	}
}

// Helper functions

func findGoBinary() (string, error) {
	// Try to find go binary
	goBinary := "/usr/local/go/bin/go"
	if runtime.GOOS == "windows" {
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
		if runtime.GOOS == "windows" {
			goBinary += ".exe"
		}
		if _, err := fileExists(goBinary); err == nil {
			return goBinary, nil
		}
	}

	return "", nil
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0755)
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && strings.Contains(s, substr))
}
