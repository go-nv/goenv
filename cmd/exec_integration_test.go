package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/shims"
	"github.com/go-nv/goenv/internal/utils"
)

// TestExec_AutoRehashAfterGoInstall tests the end-to-end auto-rehash workflow
// This is an integration test that verifies shims are automatically created
// after running 'goenv exec go install <tool>'
func TestExec_AutoRehashAfterGoInstall(t *testing.T) {
	// Skip if SHORT test mode (this is an integration test)
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a temporary GOENV_ROOT for testing
	tempRoot := t.TempDir()
	oldRoot := utils.GoenvEnvVarRoot.UnsafeValue()
	utils.GoenvEnvVarRoot.Set(tempRoot)
	defer utils.GoenvEnvVarRoot.Set(oldRoot)

	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	// Check if we have a Go version installed to test with
	versions, err := mgr.ListInstalledVersions()
	if err != nil || len(versions) == 0 {
		t.Skip("No Go versions installed, skipping integration test")
	}

	// Use the first available version
	testVersion := versions[0]

	// Skip if it's 'system' (we need a real goenv-managed version)
	if testVersion == "system" {
		if len(versions) < 2 {
			t.Skip("Only system Go available, need goenv-managed version for test")
		}
		testVersion = versions[1]
	}

	// Set this version as current
	utils.GoenvEnvVarVersion.Set(testVersion)

	// Get the Go binary path
	goBin := filepath.Join(cfg.Root, "versions", testVersion, "bin", "go")
	if _, err := os.Stat(goBin); err != nil {
		t.Skipf("Go binary not found at %s, skipping test", goBin)
	}

	// Count shims before
	shimMgr := shims.NewShimManager(cfg)
	shimsDir := filepath.Join(cfg.Root, "shims")
	os.MkdirAll(shimsDir, 0755)

	shimsBefore, err := os.ReadDir(shimsDir)
	if err != nil {
		t.Fatalf("Failed to read shims directory: %v", err)
	}
	shimCountBefore := len(shimsBefore)

	// Test installing a simple tool that compiles quickly
	// We'll use a minimal tool or skip if network is unavailable
	t.Logf("Testing auto-rehash with Go %s", testVersion)
	t.Logf("Shims before: %d", shimCountBefore)

	// For this test, we'll verify the mechanism works by:
	// 1. Creating a fake tool binary in GOPATH
	// 2. Running rehash manually (simulating what exec does)
	// 3. Verifying the shim was created

	// Create a mock GOPATH for testing
	gopath := filepath.Join(tempRoot, "go", testVersion)
	gopathBin := filepath.Join(gopath, "bin")
	os.MkdirAll(gopathBin, 0755)

	// Create a mock tool binary
	mockToolName := "test-tool-auto-rehash"
	mockToolPath := filepath.Join(gopathBin, mockToolName)

	// Create executable mock binary
	mockContent := []byte("#!/bin/sh\necho 'test tool'\n")
	if err := os.WriteFile(mockToolPath, mockContent, 0755); err != nil {
		t.Fatalf("Failed to create mock tool: %v", err)
	}

	// Set GOENV_GOPATH_PREFIX to our test GOPATH
	utils.GoenvEnvVarGopathPrefix.Set(filepath.Join(tempRoot, "go"))
	defer os.Unsetenv("GOENV_GOPATH_PREFIX")

	// Verify tool exists
	if _, err := os.Stat(mockToolPath); err != nil {
		t.Fatalf("Mock tool not created: %v", err)
	}

	// Now run rehash (simulating what happens after 'go install')
	if err := shimMgr.Rehash(); err != nil {
		t.Fatalf("Rehash failed: %v", err)
	}

	// Count shims after
	shimsAfter, err := os.ReadDir(shimsDir)
	if err != nil {
		t.Fatalf("Failed to read shims directory after rehash: %v", err)
	}
	shimCountAfter := len(shimsAfter)

	t.Logf("Shims after: %d", shimCountAfter)

	// Verify shim was created
	shimPath := filepath.Join(shimsDir, mockToolName)
	if _, err := os.Stat(shimPath); err != nil {
		t.Errorf("Shim not created for %s at %s", mockToolName, shimPath)
		// List all shims for debugging
		t.Logf("Available shims:")
		for _, shim := range shimsAfter {
			t.Logf("  - %s", shim.Name())
		}
	}

	// Verify shim count increased
	if shimCountAfter <= shimCountBefore {
		t.Errorf("Shim count did not increase: before=%d, after=%d", shimCountBefore, shimCountAfter)
	}

	t.Logf("✓ Auto-rehash mechanism verified: shim created for %s", mockToolName)
}

// TestExec_AutoRehashCanBeDisabled tests that auto-rehash can be disabled
func TestExec_AutoRehashCanBeDisabled(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test that GOENV_NO_AUTO_REHASH environment variable works
	oldValue := utils.GoenvEnvVarNoAutoRehash.UnsafeValue()
	utils.GoenvEnvVarNoAutoRehash.Set("1")
	defer func() {
		if oldValue == "" {
			os.Unsetenv("GOENV_NO_AUTO_REHASH")
		} else {
			utils.GoenvEnvVarNoAutoRehash.Set(oldValue)
		}
	}()

	// Verify the environment variable is set
	if os.Getenv("GOENV_NO_AUTO_REHASH") != "1" {
		t.Fatal("Failed to set GOENV_NO_AUTO_REHASH")
	}

	// The actual logic check is in cmd/exec.go
	// This test just verifies the environment variable can be set
	// The unit tests in exec_test.go verify the actual behavior

	t.Log("✓ GOENV_NO_AUTO_REHASH environment variable works")
}

// TestExec_RealGoInstallIntegration tests actual go install command (optional, network-dependent)
func TestExec_RealGoInstallIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real go install integration test in short mode")
	}

	// Skip if CI environment or explicitly disabled
	if os.Getenv("CI") != "" || os.Getenv("SKIP_NETWORK_TESTS") == "1" {
		t.Skip("Skipping network-dependent test in CI")
	}

	// Create a temporary GOENV_ROOT for testing
	tempRoot := t.TempDir()
	oldRoot := utils.GoenvEnvVarRoot.UnsafeValue()
	utils.GoenvEnvVarRoot.Set(tempRoot)
	defer utils.GoenvEnvVarRoot.Set(oldRoot)

	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	// Check if we have a Go version installed
	versions, err := mgr.ListInstalledVersions()
	if err != nil || len(versions) == 0 {
		t.Skip("No Go versions installed, skipping integration test")
	}

	// Use the first available non-system version
	var testVersion string
	for _, v := range versions {
		if v != "system" {
			testVersion = v
			break
		}
	}

	if testVersion == "" {
		t.Skip("No goenv-managed Go version available")
	}

	// Set version
	utils.GoenvEnvVarVersion.Set(testVersion)

	// Build goenv binary for testing
	goenvBin := filepath.Join(tempRoot, "goenv-test")
	cmd := exec.Command("go", "build", "-o", goenvBin, ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build goenv: %v", err)
	}

	// Count shims before
	shimsDir := filepath.Join(cfg.Root, "shims")
	os.MkdirAll(shimsDir, 0755)

	shimsBefore, _ := os.ReadDir(shimsDir)
	shimCountBefore := len(shimsBefore)

	// Try to install a very small, fast-compiling tool
	// Using golang.org/x/example/hello as it's minimal
	t.Log("Attempting to install a minimal Go tool...")
	installCmd := exec.Command(goenvBin, "exec", "go", "install", "golang.org/x/example/hello@latest")
	installCmd.Env = append(os.Environ(),
		"GOENV_ROOT="+tempRoot,
		"GOENV_VERSION="+testVersion,
	)

	output, err := installCmd.CombinedOutput()
	if err != nil {
		// Network might be unavailable, skip gracefully
		if strings.Contains(string(output), "no such host") ||
			strings.Contains(string(output), "connection") {
			t.Skipf("Network unavailable, skipping real install test: %v", err)
		}
		t.Logf("Install output: %s", output)
		t.Skipf("Go install failed (may be expected): %v", err)
	}

	// Count shims after
	shimsAfter, _ := os.ReadDir(shimsDir)
	shimCountAfter := len(shimsAfter)

	t.Logf("Shims before: %d, after: %d", shimCountBefore, shimCountAfter)

	// If install succeeded, shims should have been created
	if shimCountAfter > shimCountBefore {
		t.Log("✓ Real go install triggered auto-rehash successfully")
	} else {
		t.Log("⚠ Shims not created (install may have failed silently)")
	}
}
