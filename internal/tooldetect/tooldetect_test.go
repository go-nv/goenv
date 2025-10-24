package tooldetect

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name     string
		v1       string
		v2       string
		expected int
	}{
		{
			name:     "equal versions",
			v1:       "v1.2.3",
			v2:       "v1.2.3",
			expected: 0,
		},
		{
			name:     "equal versions without v prefix",
			v1:       "1.2.3",
			v2:       "1.2.3",
			expected: 0,
		},
		{
			name:     "v1 greater - major version",
			v1:       "v2.0.0",
			v2:       "v1.9.9",
			expected: 1,
		},
		{
			name:     "v1 greater - minor version",
			v1:       "v1.3.0",
			v2:       "v1.2.9",
			expected: 1,
		},
		{
			name:     "v1 greater - patch version",
			v1:       "v1.2.4",
			v2:       "v1.2.3",
			expected: 1,
		},
		{
			name:     "v2 greater - major version",
			v1:       "v1.0.0",
			v2:       "v2.0.0",
			expected: -1,
		},
		{
			name:     "v2 greater - minor version",
			v1:       "v1.2.0",
			v2:       "v1.3.0",
			expected: -1,
		},
		{
			name:     "v2 greater - patch version",
			v1:       "v1.2.3",
			v2:       "v1.2.4",
			expected: -1,
		},
		{
			name:     "unknown version v1",
			v1:       "unknown",
			v2:       "v1.2.3",
			expected: -1,
		},
		{
			name:     "unknown version v2",
			v1:       "v1.2.3",
			v2:       "unknown",
			expected: 1,
		},
		{
			name:     "both unknown",
			v1:       "unknown",
			v2:       "unknown",
			expected: 0,
		},
		{
			name:     "empty v1",
			v1:       "",
			v2:       "v1.2.3",
			expected: -1,
		},
		{
			name:     "empty v2",
			v1:       "v1.2.3",
			v2:       "",
			expected: 1,
		},
		{
			name:     "both empty",
			v1:       "",
			v2:       "",
			expected: 0,
		},
		{
			name:     "mixed v prefix",
			v1:       "v1.2.3",
			v2:       "1.2.3",
			expected: 0,
		},
		{
			name:     "prerelease versions",
			v1:       "v1.2.3-rc1",
			v2:       "v1.2.3-rc1",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompareVersions(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}

func TestListInstalledTools(t *testing.T) {
	// Create temporary goenv root
	tmpDir := t.TempDir()

	tests := []struct {
		name          string
		goVersion     string
		setupTools    []string // Tool binaries to create
		expectedCount int
		expectedNames []string
	}{
		{
			name:          "no tools installed",
			goVersion:     "1.21.0",
			setupTools:    []string{},
			expectedCount: 0,
			expectedNames: []string{},
		},
		{
			name:          "single tool installed",
			goVersion:     "1.21.0",
			setupTools:    []string{"gopls"},
			expectedCount: 1,
			expectedNames: []string{"gopls"},
		},
		{
			name:          "multiple tools installed",
			goVersion:     "1.21.0",
			setupTools:    []string{"gopls", "golangci-lint", "staticcheck"},
			expectedCount: 3,
			expectedNames: []string{"gopls", "golangci-lint", "staticcheck"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup directory structure
			gopathBin := filepath.Join(tmpDir, "versions", tt.goVersion, "gopath", "bin")
			if err := os.MkdirAll(gopathBin, 0755); err != nil {
				t.Fatalf("Failed to create GOPATH/bin: %v", err)
			}

			// Create tool binaries
			for _, tool := range tt.setupTools {
				toolPath := filepath.Join(gopathBin, tool)
			var content string
			if runtime.GOOS == "windows" {
				toolPath += ".bat"
				content = "@echo off\necho mock tool\n"
			} else {
				content = "#!/bin/sh\necho mock tool\n"
			}
			if err := os.WriteFile(toolPath, []byte(content), 0755); err != nil {
					t.Fatalf("Failed to create tool %s: %v", tool, err)
				}
				}
			}

			// List installed tools
			tools, err := ListInstalledTools(tmpDir, tt.goVersion)
			if err != nil {
				t.Fatalf("ListInstalledTools() error = %v", err)
			}

			// Check count
			if len(tools) != tt.expectedCount {
				t.Errorf("ListInstalledTools() returned %d tools, want %d", len(tools), tt.expectedCount)
			}

			// Check names
			toolNames := make(map[string]bool)
			for _, tool := range tools {
				toolNames[tool.Name] = true
			}

			for _, expectedName := range tt.expectedNames {
				if !toolNames[expectedName] {
					t.Errorf("Expected tool %q not found in results", expectedName)
				}
			}
		})
	}
}

func TestListInstalledTools_NonExistentVersion(t *testing.T) {
	tmpDir := t.TempDir()

	// Try to list tools for a version that doesn't exist
	tools, err := ListInstalledTools(tmpDir, "99.99.99")

	if err != nil {
		t.Errorf("ListInstalledTools() should not error for non-existent version, got: %v", err)
	}

	if len(tools) != 0 {
		t.Errorf("ListInstalledTools() should return empty list for non-existent version, got %d tools", len(tools))
	}
}

func TestListInstalledTools_SkipsDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	goVersion := "1.21.0"

	// Setup directory structure
	gopathBin := filepath.Join(tmpDir, "versions", goVersion, "gopath", "bin")
	if err := os.MkdirAll(gopathBin, 0755); err != nil {
		t.Fatalf("Failed to create GOPATH/bin: %v", err)
	}

	// Create a tool binary
	toolPath := filepath.Join(gopathBin, "gopls")
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
		toolPath += ext
	}
	if err := os.WriteFile(toolPath, []byte("mock binary"), 0755); err != nil {
		t.Fatalf("Failed to create tool: %v", err)
	}

	// Create a directory (should be skipped)
	dirPath := filepath.Join(gopathBin, "subdir")
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// List tools
	tools, err := ListInstalledTools(tmpDir, goVersion)
	if err != nil {
		t.Fatalf("ListInstalledTools() error = %v", err)
	}

	// Should only find the one tool, not the directory
	if len(tools) != 1 {
		t.Errorf("ListInstalledTools() returned %d tools, want 1 (directory should be skipped)", len(tools))
	}

	if len(tools) > 0 && tools[0].Name != "gopls" {
		t.Errorf("Expected tool name 'gopls', got %q", tools[0].Name)
	}
}

func TestListInstalledTools_WindowsExeHandling(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test")
	}

	tmpDir := t.TempDir()
	goVersion := "1.21.0"

	// Setup directory structure
	gopathBin := filepath.Join(tmpDir, "versions", goVersion, "gopath", "bin")
	if err := os.MkdirAll(gopathBin, 0755); err != nil {
		t.Fatalf("Failed to create GOPATH/bin: %v", err)
	}

	// Create .exe files
	exeTools := []string{"gopls.exe", "golangci-lint.exe"}
	for _, tool := range exeTools {
		toolPath := filepath.Join(gopathBin, tool)
		if err := os.WriteFile(toolPath, []byte("mock binary"), 0755); err != nil {
			t.Fatalf("Failed to create tool %s: %v", tool, err)
		}
	}

	// Create a non-.exe file (should be skipped on Windows)
	nonExePath := filepath.Join(gopathBin, "README")
	if err := os.WriteFile(nonExePath, []byte("readme file"), 0644); err != nil {
		t.Fatalf("Failed to create non-exe file: %v", err)
	}

	// List tools
	tools, err := ListInstalledTools(tmpDir, goVersion)
	if err != nil {
		t.Fatalf("ListInstalledTools() error = %v", err)
	}

	// Should find only .exe files
	if len(tools) != 2 {
		t.Errorf("ListInstalledTools() returned %d tools, want 2 (.exe files only)", len(tools))
	}

	// Check that .exe suffix is stripped from names
	for _, tool := range tools {
		if filepath.Ext(tool.Name) == ".exe" {
			t.Errorf("Tool name %q still has .exe extension, should be stripped", tool.Name)
		}
	}
}

func TestIsGoTool(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a mock Go binary that can be executed
	mockGoBinary := filepath.Join(tmpDir, "mock_tool")
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
		mockGoBinary += ext
	}

	// Write a simple binary-like file
	if err := os.WriteFile(mockGoBinary, []byte("mock binary"), 0755); err != nil {
		t.Fatalf("Failed to create mock binary: %v", err)
	}

	tests := []struct {
		name       string
		binaryPath string
		want       bool
	}{
		{
			name:       "non-existent file",
			binaryPath: filepath.Join(tmpDir, "nonexistent"+ext),
			want:       false,
		},
		{
			name:       "directory",
			binaryPath: tmpDir,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsGoTool(tt.binaryPath)
			if result != tt.want {
				t.Errorf("IsGoTool(%q) = %v, want %v", tt.binaryPath, result, tt.want)
			}
		})
	}
}

// Note: ExtractToolInfo and GetLatestVersion are harder to test without real Go binaries
// and network access. These would typically be integration tests or require mocking.

func TestExtractToolInfo_NonExistentBinary(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentPath := filepath.Join(tmpDir, "nonexistent")

	_, _, err := ExtractToolInfo(nonExistentPath)
	if err == nil {
		t.Error("ExtractToolInfo() should return error for non-existent binary")
	}
}

func TestGetLatestVersion_EmptyPackage(t *testing.T) {
	version, err := GetLatestVersion("")

	if err == nil {
		t.Error("GetLatestVersion() should return error for empty package path")
	}

	if version != "" {
		t.Errorf("GetLatestVersion() should return empty version on error, got %q", version)
	}
}

func TestGetLatestVersion_InvalidPackage(t *testing.T) {
	// Use an invalid package path
	version, err := GetLatestVersion("invalid/package/that/does/not/exist/anywhere")

	if err == nil {
		t.Error("GetLatestVersion() should return error for invalid package")
	}

	if version != "" {
		t.Errorf("GetLatestVersion() should return empty version on error, got %q", version)
	}
}
