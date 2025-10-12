package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-nv/goenv/internal/config"
)

func TestRefreshCommand(t *testing.T) {
	// Create a temporary directory for test
	tmpDir := t.TempDir()
	originalRoot := os.Getenv("GOENV_ROOT")
	os.Setenv("GOENV_ROOT", tmpDir)
	defer os.Setenv("GOENV_ROOT", originalRoot)

	// Reload config to pick up new GOENV_ROOT
	config.Load()

	tests := []struct {
		name          string
		setup         func() error
		expectRemoved int
		expectError   bool
	}{
		{
			name: "remove existing caches",
			setup: func() error {
				// Create dummy cache files
				if err := os.WriteFile(filepath.Join(tmpDir, "versions-cache.json"), []byte("{}"), 0644); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(tmpDir, "releases-cache.json"), []byte("{}"), 0644); err != nil {
					return err
				}
				return nil
			},
			expectRemoved: 2,
			expectError:   false,
		},
		{
			name: "no cache files exist",
			setup: func() error {
				// Don't create any files
				return nil
			},
			expectRemoved: 0,
			expectError:   false,
		},
		{
			name: "only one cache file exists",
			setup: func() error {
				// Create only one cache file
				return os.WriteFile(filepath.Join(tmpDir, "versions-cache.json"), []byte("{}"), 0644)
			},
			expectRemoved: 1,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if err := tt.setup(); err != nil {
				t.Fatalf("setup failed: %v", err)
			}

			// Run command
			cmd := refreshCmd
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs([]string{})

			err := runRefresh(cmd, []string{})

			// Check error expectation
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Verify cache files were removed
			cacheFiles := []string{
				filepath.Join(tmpDir, "versions-cache.json"),
				filepath.Join(tmpDir, "releases-cache.json"),
			}

			for _, cacheFile := range cacheFiles {
				if _, err := os.Stat(cacheFile); err == nil {
					if tt.expectRemoved > 0 {
						t.Errorf("cache file still exists: %s", cacheFile)
					}
				}
			}
		})
	}
}

func TestRefreshVerboseFlag(t *testing.T) {
	tmpDir := t.TempDir()
	originalRoot := os.Getenv("GOENV_ROOT")
	os.Setenv("GOENV_ROOT", tmpDir)
	defer os.Setenv("GOENV_ROOT", originalRoot)

	// Create a cache file
	cacheFile := filepath.Join(tmpDir, "versions-cache.json")
	if err := os.WriteFile(cacheFile, []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Run with verbose flag
	refreshFlags.verbose = true
	defer func() { refreshFlags.verbose = false }()

	cmd := refreshCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	if err := runRefresh(cmd, []string{}); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	// With verbose flag, we expect to see detailed output
	// The output is written directly to stdout via fmt.Printf, not through cmd.SetOut
	// So this test just verifies the command runs successfully with the verbose flag
}
