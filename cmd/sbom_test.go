package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/config"
)

func TestSBOMProject_FlagValidation(t *testing.T) {
	tests := []struct {
		name        string
		setupFlags  func()
		expectError bool
		errorText   string
	}{
		{
			name: "both image and dir specified",
			setupFlags: func() {
				sbomImage = "myimage:latest"
				sbomDir = "/some/dir"
			},
			expectError: true,
			errorText:   "cannot specify both --image and --dir",
		},
		{
			name: "image with non-syft tool",
			setupFlags: func() {
				sbomImage = "myimage:latest"
				sbomDir = "."
				sbomTool = "cyclonedx-gomod"
			},
			expectError: true,
			errorText:   "--image is only supported with --tool=syft",
		},
		{
			name: "valid cyclonedx-gomod",
			setupFlags: func() {
				sbomImage = ""
				sbomDir = "."
				sbomTool = "cyclonedx-gomod"
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			sbomImage = ""
			sbomDir = "."
			sbomTool = "cyclonedx-gomod"

			// Setup
			tt.setupFlags()

			// Create temp directory for test
			tmpDir := t.TempDir()
			os.Setenv("GOENV_ROOT", tmpDir)
			defer os.Unsetenv("GOENV_ROOT")

			// Run command
			cmd := sbomProjectCmd
			cmd.SetArgs([]string{})
			err := cmd.RunE(cmd, []string{})

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.errorText)
				} else if !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("Expected error containing %q, got %q", tt.errorText, err.Error())
				}
			} else if !tt.expectError && err != nil {
				// For valid cases, we expect tool-not-found error (since we don't have tools installed in test)
				if !strings.Contains(err.Error(), "not found") {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestResolveSBOMTool(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Root: tmpDir,
	}

	// Create host bin directory
	hostBinDir := cfg.HostBinDir()
	if err := os.MkdirAll(hostBinDir, 0755); err != nil {
		t.Fatalf("Failed to create host bin dir: %v", err)
	}

	// Create mock tool
	toolName := "cyclonedx-gomod"
	toolPath := filepath.Join(hostBinDir, toolName)
	if runtime.GOOS == "windows" {
		toolPath += ".exe"
	}

	if err := os.WriteFile(toolPath, []byte("#!/bin/sh\necho mock"), 0755); err != nil {
		t.Fatalf("Failed to create mock tool: %v", err)
	}

	// Test resolution
	resolvedPath, err := resolveSBOMTool(cfg, toolName)
	if err != nil {
		t.Fatalf("Failed to resolve tool: %v", err)
	}

	if resolvedPath != toolPath {
		t.Errorf("Expected path %q, got %q", toolPath, resolvedPath)
	}
}

func TestResolveSBOMTool_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Root: tmpDir,
	}

	// Test resolution for non-existent tool
	_, err := resolveSBOMTool(cfg, "nonexistent-tool")
	if err == nil {
		t.Error("Expected error for non-existent tool")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}

	if !strings.Contains(err.Error(), "goenv tools install") {
		t.Errorf("Expected installation instructions in error, got: %v", err)
	}
}

func TestBuildCycloneDXCommand(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}

	tests := []struct {
		name        string
		format      string
		modulesOnly bool
		output      string
		expectError bool
		expectArgs  []string
	}{
		{
			name:        "json format",
			format:      "cyclonedx-json",
			modulesOnly: false,
			output:      "sbom.json",
			expectArgs:  []string{"-output", "sbom.json", "-json"},
		},
		{
			name:        "modules only",
			format:      "cyclonedx-json",
			modulesOnly: true,
			output:      "sbom.json",
			expectArgs:  []string{"-output", "sbom.json", "-json", "-licenses", "-type", "library"},
		},
		{
			name:        "unsupported format",
			format:      "spdx-json",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set globals
			sbomFormat = tt.format
			sbomModulesOnly = tt.modulesOnly
			sbomOutput = tt.output
			sbomToolArgs = ""

			cmd, err := buildCycloneDXCommand("/mock/tool", cfg)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Check args
			for _, expectedArg := range tt.expectArgs {
				found := false
				for _, arg := range cmd.Args[1:] { // Skip binary name
					if arg == expectedArg {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected arg %q not found in %v", expectedArg, cmd.Args)
				}
			}
		})
	}
}

func TestBuildSyftCommand(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}

	tests := []struct {
		name        string
		format      string
		image       string
		dir         string
		output      string
		expectError bool
		expectArgs  []string
	}{
		{
			name:       "directory scan with cyclonedx",
			format:     "cyclonedx-json",
			dir:        ".",
			output:     "sbom.json",
			expectArgs: []string{".", "-o", "cyclonedx-json=sbom.json", "-q"},
		},
		{
			name:       "image scan with spdx",
			format:     "spdx-json",
			image:      "myimage:latest",
			output:     "sbom.json",
			expectArgs: []string{"myimage:latest", "-o", "spdx-json=sbom.json", "-q"},
		},
		{
			name:        "unsupported format",
			format:      "invalid-format",
			dir:         ".",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set globals
			sbomFormat = tt.format
			sbomImage = tt.image
			sbomDir = tt.dir
			sbomOutput = tt.output
			sbomToolArgs = ""

			cmd, err := buildSyftCommand("/mock/syft", cfg)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Check args
			for _, expectedArg := range tt.expectArgs {
				found := false
				for _, arg := range cmd.Args[1:] { // Skip binary name
					if arg == expectedArg {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected arg %q not found in %v", expectedArg, cmd.Args)
				}
			}
		})
	}
}
