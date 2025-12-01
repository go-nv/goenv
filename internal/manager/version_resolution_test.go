package manager

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-nv/goenv/internal/config"
)

// setupTestVersions creates a test directory structure with installed versions
func setupTestVersions(t *testing.T, versions []string) (string, *Manager) {
	t.Helper()

	// Create temporary directory for test
	tmpDir := t.TempDir()

	// Create versions directory
	versionsDir := filepath.Join(tmpDir, "versions")
	if err := os.MkdirAll(versionsDir, 0755); err != nil {
		t.Fatalf("Failed to create versions dir: %v", err)
	}

	// Create version directories with go binaries
	for _, version := range versions {
		versionDir := filepath.Join(versionsDir, version, "bin")
		if err := os.MkdirAll(versionDir, 0755); err != nil {
			t.Fatalf("Failed to create version dir %s: %v", version, err)
		}

		// Create a dummy go binary (platform-appropriate)
		goBinaryName := "go"
		if runtime.GOOS == "windows" {
			goBinaryName = "go.exe"
		}
		goFile := filepath.Join(versionDir, goBinaryName)
		if err := os.WriteFile(goFile, []byte("#!/bin/sh\necho go"+version), 0755); err != nil {
			t.Fatalf("Failed to create go binary for %s: %v", version, err)
		}
	}

	// Create manager with test config
	cfg := &config.Config{}
	cfg.Root = tmpDir

	mgr := NewManager(cfg)
	return tmpDir, mgr
}

func TestValidateVersion_PartialVersionResolution(t *testing.T) {
	tests := []struct {
		name           string
		installedVersions []string
		versionToValidate string
		expectError    bool
		description    string
	}{
		{
			name:              "exact version match",
			installedVersions: []string{"1.25.4"},
			versionToValidate: "1.25.4",
			expectError:       false,
			description:       "Exact version should be valid",
		},
		{
			name:              "partial major.minor resolves to patch",
			installedVersions: []string{"1.25.2", "1.25.3", "1.25.4"},
			versionToValidate: "1.25",
			expectError:       false,
			description:       "1.25 should resolve to 1.25.4 (latest patch)",
		},
		{
			name:              "partial major.minor with single patch",
			installedVersions: []string{"1.24.7"},
			versionToValidate: "1.24",
			expectError:       false,
			description:       "1.24 should resolve to 1.24.7",
		},
		{
			name:              "partial version not installed",
			installedVersions: []string{"1.23.1", "1.24.7"},
			versionToValidate: "1.25",
			expectError:       true,
			description:       "1.25 has no matching versions",
		},
		{
			name:              "system version always valid",
			installedVersions: []string{"1.25.4"},
			versionToValidate: "system",
			expectError:       false,
			description:       "system is a special keyword",
		},
		{
			name:              "non-existent exact version",
			installedVersions: []string{"1.25.4"},
			versionToValidate: "1.26.0",
			expectError:       true,
			description:       "Exact version that doesn't exist should fail",
		},
		{
			name:              "major version resolution",
			installedVersions: []string{"1.20.0", "1.21.5", "1.22.8"},
			versionToValidate: "1",
			expectError:       false,
			description:       "Major version should resolve to latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mgr := setupTestVersions(t, tt.installedVersions)

			err := mgr.ValidateVersion(tt.versionToValidate)

			if tt.expectError && err == nil {
				t.Errorf("%s: expected error but got none", tt.description)
			}
			if !tt.expectError && err != nil {
				t.Errorf("%s: expected no error but got: %v", tt.description, err)
			}
		})
	}
}

func TestIsVersionInstalled_PartialVersionResolution(t *testing.T) {
	tests := []struct {
		name              string
		installedVersions []string
		versionToCheck    string
		expectInstalled   bool
		description       string
	}{
		{
			name:              "exact version installed",
			installedVersions: []string{"1.25.4"},
			versionToCheck:    "1.25.4",
			expectInstalled:   true,
			description:       "Exact match should return true",
		},
		{
			name:              "partial version resolves",
			installedVersions: []string{"1.25.2", "1.25.3", "1.25.4"},
			versionToCheck:    "1.25",
			expectInstalled:   true,
			description:       "1.25 resolves to an installed version",
		},
		{
			name:              "partial version not installed",
			installedVersions: []string{"1.24.7"},
			versionToCheck:    "1.25",
			expectInstalled:   false,
			description:       "No matching version for 1.25",
		},
		{
			name:              "system always installed",
			installedVersions: []string{"1.25.4"},
			versionToCheck:    "system",
			expectInstalled:   true,
			description:       "system is always considered installed",
		},
		{
			name:              "major version resolves",
			installedVersions: []string{"1.20.0", "1.21.5"},
			versionToCheck:    "1",
			expectInstalled:   true,
			description:       "Major version 1 should resolve",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mgr := setupTestVersions(t, tt.installedVersions)

			installed := mgr.IsVersionInstalled(tt.versionToCheck)

			if installed != tt.expectInstalled {
				t.Errorf("%s: expected installed=%v but got %v",
					tt.description, tt.expectInstalled, installed)
			}
		})
	}
}

func TestGetVersionPath_PartialVersionResolution(t *testing.T) {
	tests := []struct {
		name              string
		installedVersions []string
		versionToGet      string
		expectError       bool
		expectedVersion   string // The actual version dir that should be returned
		description       string
	}{
		{
			name:              "exact version path",
			installedVersions: []string{"1.25.4"},
			versionToGet:      "1.25.4",
			expectError:       false,
			expectedVersion:   "1.25.4",
			description:       "Exact match returns exact path",
		},
		{
			name:              "partial version resolves to latest patch",
			installedVersions: []string{"1.25.2", "1.25.3", "1.25.4"},
			versionToGet:      "1.25",
			expectError:       false,
			expectedVersion:   "1.25.4",
			description:       "1.25 resolves to 1.25.4 path",
		},
		{
			name:              "partial version not found",
			installedVersions: []string{"1.24.7"},
			versionToGet:      "1.25",
			expectError:       true,
			expectedVersion:   "",
			description:       "Non-existent partial version returns error",
		},
		{
			name:              "system version returns empty path",
			installedVersions: []string{"1.25.4"},
			versionToGet:      "system",
			expectError:       false,
			expectedVersion:   "",
			description:       "system returns empty path (uses PATH)",
		},
		{
			name:              "major version resolves to latest",
			installedVersions: []string{"1.20.0", "1.21.5", "1.22.8"},
			versionToGet:      "1",
			expectError:       false,
			expectedVersion:   "1.22.8",
			description:       "Major version resolves to latest 1.x",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, mgr := setupTestVersions(t, tt.installedVersions)

			path, err := mgr.GetVersionPath(tt.versionToGet)

			if tt.expectError && err == nil {
				t.Errorf("%s: expected error but got none", tt.description)
			}
			if !tt.expectError && err != nil {
				t.Errorf("%s: expected no error but got: %v", tt.description, err)
			}

			if !tt.expectError {
				expectedPath := ""
				if tt.expectedVersion != "" {
					expectedPath = filepath.Join(tmpDir, "versions", tt.expectedVersion)
				}

				if path != expectedPath {
					t.Errorf("%s: expected path %q but got %q",
						tt.description, expectedPath, path)
				}
			}
		})
	}
}

func TestPartialVersionResolution_GoModScenario(t *testing.T) {
	// This test simulates the real-world scenario that triggered the bug:
	// A go.mod file with "go 1.25" and installed versions like 1.25.2, 1.25.3, 1.25.4
	installedVersions := []string{"1.25.2", "1.25.3", "1.25.4"}
	_, mgr := setupTestVersions(t, installedVersions)

	// Test that all three methods work with the partial version from go.mod
	partialVersion := "1.25"

	// 1. ValidateVersion should not return error
	if err := mgr.ValidateVersion(partialVersion); err != nil {
		t.Errorf("ValidateVersion(%q) failed: %v (should resolve to 1.25.4)", partialVersion, err)
	}

	// 2. IsVersionInstalled should return true
	if !mgr.IsVersionInstalled(partialVersion) {
		t.Errorf("IsVersionInstalled(%q) returned false (should resolve to 1.25.4)", partialVersion)
	}

	// 3. GetVersionPath should return valid path
	path, err := mgr.GetVersionPath(partialVersion)
	if err != nil {
		t.Errorf("GetVersionPath(%q) failed: %v (should resolve to 1.25.4)", partialVersion, err)
	}
	if path == "" {
		t.Errorf("GetVersionPath(%q) returned empty path", partialVersion)
	}

	// Verify the path points to the latest patch version
	expectedVersion := "1.25.4"
	if filepath.Base(path) != expectedVersion {
		t.Errorf("GetVersionPath(%q) resolved to wrong version: got %s, expected %s",
			partialVersion, filepath.Base(path), expectedVersion)
	}
}

func TestPartialVersionResolution_EdgeCases(t *testing.T) {
	tests := []struct {
		name              string
		installedVersions []string
		partialVersion    string
		shouldResolve     bool
		description       string
	}{
		{
			name:              "rc versions",
			installedVersions: []string{"1.25rc1", "1.25rc2", "1.25.0"},
			partialVersion:    "1.25",
			shouldResolve:     true,
			description:       "Should resolve to stable version, not rc",
		},
		{
			name:              "beta versions",
			installedVersions: []string{"1.25beta1", "1.25.0"},
			partialVersion:    "1.25",
			shouldResolve:     true,
			description:       "Should resolve to stable version",
		},
		{
			name:              "multiple major versions",
			installedVersions: []string{"1.24.7", "1.25.4", "2.0.0"},
			partialVersion:    "1.25",
			shouldResolve:     true,
			description:       "Should resolve to correct minor version",
		},
		{
			name:              "only older versions available",
			installedVersions: []string{"1.23.1", "1.24.7"},
			partialVersion:    "1.25",
			shouldResolve:     false,
			description:       "Should not match older versions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, mgr := setupTestVersions(t, tt.installedVersions)

			installed := mgr.IsVersionInstalled(tt.partialVersion)
			if installed != tt.shouldResolve {
				t.Errorf("%s: IsVersionInstalled(%q) = %v, expected %v",
					tt.description, tt.partialVersion, installed, tt.shouldResolve)
			}

			err := mgr.ValidateVersion(tt.partialVersion)
			hasError := (err != nil)
			expectError := !tt.shouldResolve

			if hasError != expectError {
				t.Errorf("%s: ValidateVersion(%q) error state = %v, expected %v (err: %v)",
					tt.description, tt.partialVersion, hasError, expectError, err)
			}
		})
	}
}
