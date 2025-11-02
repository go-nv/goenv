package diagnostics

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
)

func TestRefreshCommand(t *testing.T) {
	// Create a temporary directory for test
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)

	// Reload config to pick up new GOENV_ROOT
	config.Load()

	tests := []struct {
		name          string
		setup         func()
		expectRemoved int
		expectError   bool
	}{
		{
			name: "remove existing caches",
			setup: func() {
				// Create dummy cache files
				testutil.WriteTestFile(t, filepath.Join(tmpDir, "versions-cache.json"), []byte("{}"), utils.PermFileDefault)
				testutil.WriteTestFile(t, filepath.Join(tmpDir, "releases-cache.json"), []byte("{}"), utils.PermFileDefault)
			},
			expectRemoved: 2,
			expectError:   false,
		},
		{
			name: "no cache files exist",
			setup: func() {
			},
			expectRemoved: 0,
			expectError:   false,
		},
		{
			name: "only one cache file exists",
			setup: func() {
				// Create only one cache file
				testutil.WriteTestFile(t, filepath.Join(tmpDir, "versions-cache.json"), []byte("{}"), utils.PermFileDefault)
			},
			expectRemoved: 1,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			tt.setup()

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
				if utils.PathExists(cacheFile) {
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
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)

	// Create a cache file
	cacheFile := filepath.Join(tmpDir, "versions-cache.json")
	testutil.WriteTestFile(t, cacheFile, []byte("{}"), utils.PermFileDefault)

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
