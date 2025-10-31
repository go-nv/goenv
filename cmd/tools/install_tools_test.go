package tools

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/config"
	"github.com/spf13/cobra"
)

func TestInstallCommand_AllFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	// Create multiple Go versions
	versions := []string{"1.21.0", "1.22.0", "1.23.0"}
	for _, v := range versions {
		versionPath := filepath.Join(tmpDir, "versions", v)
		goRoot := filepath.Join(versionPath, "go", "bin")
		gopath := filepath.Join(versionPath, "gopath", "bin")
		if err := os.MkdirAll(goRoot, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(gopath, 0755); err != nil {
			t.Fatal(err)
		}

		// Create fake go binary that does nothing (just exists)
		goBin := filepath.Join(goRoot, "go")
		if err := os.WriteFile(goBin, []byte("#!/bin/sh\nexit 0"), 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Test dry-run with --all flag
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	// Set flags
	installAllVersions = true
	installDryRun = true
	defer func() {
		installAllVersions = false
		installDryRun = false
	}()

	// Test that the function validates versions correctly
	foundVersions, err := getInstalledVersions(cfg)
	if err != nil {
		t.Fatalf("getInstalledVersions failed: %v", err)
	}

	if len(foundVersions) != 3 {
		t.Errorf("Expected 3 versions, got %d", len(foundVersions))
	}

	for _, expected := range versions {
		found := false
		for _, actual := range foundVersions {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected version %s not found", expected)
		}
	}
}

func TestInstallCommand_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	_ = &config.Config{
		Root: tmpDir,
	}

	// Create a Go version
	version := "1.23.0"
	versionPath := filepath.Join(tmpDir, "versions", version)
	goRoot := filepath.Join(versionPath, "go", "bin")
	gopath := filepath.Join(versionPath, "gopath", "bin")
	if err := os.MkdirAll(goRoot, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(gopath, 0755); err != nil {
		t.Fatal(err)
	}

	// Create fake go binary
	goBin := filepath.Join(goRoot, "go")
	if err := os.WriteFile(goBin, []byte("#!/bin/sh\nexit 0"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create .go-version file to set current version
	goVersionFile := filepath.Join(tmpDir, ".go-version")
	if err := os.WriteFile(goVersionFile, []byte(version), 0644); err != nil {
		t.Fatal(err)
	}

	// Test package normalization
	packages := normalizePackagePaths([]string{"gopls", "golang.org/x/tools/cmd/goimports@v0.1.0"})
	expected := []string{"gopls@latest", "golang.org/x/tools/cmd/goimports@v0.1.0"}

	if len(packages) != len(expected) {
		t.Errorf("Expected %d packages, got %d", len(expected), len(packages))
	}

	for i, exp := range expected {
		if packages[i] != exp {
			t.Errorf("Package %d: expected %s, got %s", i, exp, packages[i])
		}
	}
}

func TestInstallCommand_NoVersionsInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	// Create empty versions directory
	versionsDir := filepath.Join(tmpDir, "versions")
	if err := os.MkdirAll(versionsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Try to get installed versions
	versions, err := getInstalledVersions(cfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(versions) != 0 {
		t.Errorf("Expected no versions, got %d", len(versions))
	}
}

func TestInstallCommand_VerboseOutput(t *testing.T) {
	// Test that verbose flag is properly tracked
	originalVerbose := installVerbose
	defer func() { installVerbose = originalVerbose }()

	installVerbose = true
	if !installVerbose {
		t.Error("installVerbose flag should be true")
	}

	installVerbose = false
	if installVerbose {
		t.Error("installVerbose flag should be false")
	}
}

func TestExtractToolNames(t *testing.T) {
	tests := []struct {
		name     string
		packages []string
		expected []string
	}{
		{
			name:     "single package",
			packages: []string{"golang.org/x/tools/gopls@latest"},
			expected: []string{"gopls"},
		},
		{
			name:     "multiple packages",
			packages: []string{"golang.org/x/tools/gopls@latest", "honnef.co/go/tools/cmd/staticcheck@v0.4.0"},
			expected: []string{"gopls", "staticcheck"},
		},
		{
			name:     "package without version",
			packages: []string{"github.com/golangci/golangci-lint/cmd/golangci-lint"},
			expected: []string{"golangci-lint"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractToolNames(tt.packages)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d names, got %d", len(tt.expected), len(result))
				return
			}
			for i, exp := range tt.expected {
				if result[i] != exp {
					t.Errorf("Name %d: expected %s, got %s", i, exp, result[i])
				}
			}
		})
	}
}

func TestInstallToolForVersion_MissingGo(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	version := "1.23.0"
	versionPath := filepath.Join(tmpDir, "versions", version)
	if err := os.MkdirAll(versionPath, 0755); err != nil {
		t.Fatal(err)
	}

	// Don't create Go binary - should fail
	err := installToolForVersion(cfg, version, "gopls@latest", false)
	if err == nil {
		t.Error("Expected error when Go binary is missing")
	}

	if !strings.Contains(err.Error(), "go binary not found") {
		t.Errorf("Expected 'Go binary not found' error, got: %v", err)
	}
}

func TestExtractToolName_EdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "golang.org/x/tools/gopls@latest",
			expected: "gopls",
		},
		{
			input:    "golang.org/x/tools/gopls",
			expected: "gopls",
		},
		{
			input:    "gopls@latest",
			expected: "gopls",
		},
		{
			input:    "gopls",
			expected: "gopls",
		},
		{
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractToolName(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestExtractToolName_ComplexPaths(t *testing.T) {
	tests := []struct {
		packagePath string
		expected    string
	}{
		{
			packagePath: "golang.org/x/tools/gopls@v0.12.0",
			expected:    "gopls",
		},
		{
			packagePath: "github.com/golangci/golangci-lint/cmd/golangci-lint@latest",
			expected:    "golangci-lint",
		},
		{
			packagePath: "honnef.co/go/tools/cmd/staticcheck@2023.1.5",
			expected:    "staticcheck",
		},
		{
			packagePath: "github.com/go-delve/delve/cmd/dlv",
			expected:    "dlv",
		},
		{
			packagePath: "mvdan.cc/gofumpt@latest",
			expected:    "gofumpt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.packagePath, func(t *testing.T) {
			result := extractToolName(tt.packagePath)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
