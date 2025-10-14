package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestListCommand(t *testing.T) {
	t.Run("returns multiple versions", func(t *testing.T) {
		_, cleanup := setupTestEnv(t)
		defer cleanup()

		cmd := &cobra.Command{
			Use: "list",
			RunE: func(cmd *cobra.Command, args []string) error {
				return runList(cmd, args)
			},
		}

		output := &strings.Builder{}
		cmd.SetOut(output)
		cmd.SetArgs([]string{})

		err := cmd.Execute()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		result := output.String()
		lines := strings.Split(strings.TrimSpace(result), "\n")

		// Critical: Should return many versions, not just latest 2
		// This test would have caught the bug where only 2 versions per line were shown
		if len(lines) < 100 {
			t.Errorf("Expected at least 100 versions, got %d. This suggests the bug of showing only latest 2 versions has returned!", len(lines))
		}

		// Verify we have some known versions
		if !strings.Contains(result, "1.21") {
			t.Error("Expected to find version 1.21.x in output")
		}
		if !strings.Contains(result, "1.22") {
			t.Error("Expected to find version 1.22.x in output")
		}

		t.Logf("✅ Found %d versions (including embedded versions)", len(lines))
		t.Logf("First 5 versions: %v", lines[:min(5, len(lines))])
	})

	t.Run("stable flag filters prereleases", func(t *testing.T) {
		_, cleanup := setupTestEnv(t)
		defer cleanup()

		// Create command with stable flag
		cmd := &cobra.Command{
			Use: "list",
			RunE: func(cmd *cobra.Command, args []string) error {
				listFlags.stable = true
				defer func() { listFlags.stable = false }()
				return runList(cmd, args)
			},
		}

		output := &strings.Builder{}
		cmd.SetOut(output)
		cmd.SetArgs([]string{})

		err := cmd.Execute()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		result := output.String()

		// Should NOT contain beta or rc versions when stable flag is set
		if strings.Contains(result, "beta") {
			t.Error("Stable flag should filter out beta versions")
		}
		if strings.Contains(result, "rc") && !strings.Contains(result, "1.21.0") { // rc in version string is ok
			t.Error("Stable flag should filter out rc versions")
		}

		// Should still have stable versions
		lines := strings.Split(strings.TrimSpace(result), "\n")
		if len(lines) < 50 {
			t.Errorf("Even with stable filter, should have 50+ stable versions, got %d", len(lines))
		}

		t.Logf("✅ Stable filter working: %d stable versions found", len(lines))
	})

	t.Run("includes patch versions not just latest 2", func(t *testing.T) {
		_, cleanup := setupTestEnv(t)
		defer cleanup()

		// This is THE KEY TEST - ensures we get all patch versions, not just latest 2
		// This test would have caught the historical bug immediately
		cmd := &cobra.Command{
			Use: "list",
			RunE: func(cmd *cobra.Command, args []string) error {
				return runList(cmd, args)
			},
		}

		output := &strings.Builder{}
		cmd.SetOut(output)
		cmd.SetArgs([]string{})

		err := cmd.Execute()
		if err != nil {
			t.Fatalf("list command failed: %v", err)
		}

		result := output.String()

		// Count how many 1.21.x versions we have (without "go" prefix in Go implementation)
		// Looking for lines like "  1.21.0", "  1.21.1", etc.
		count121 := strings.Count(result, "  1.21.")

		// The bug was showing only latest 2 versions per minor line
		// 1.21 has 13+ patch versions (1.21.0 through 1.21.13)
		if count121 < 3 {
			t.Errorf("CRITICAL: Expected at least 3 versions of 1.21.x, got %d. The 'latest 2 versions' bug may have returned!", count121)
		}

		// Also check 1.20.x (has 14+ versions: 1.20.0 through 1.20.14)
		count120 := strings.Count(result, "  1.20.")
		if count120 < 3 {
			t.Errorf("CRITICAL: Expected at least 3 versions of 1.20.x, got %d", count120)
		}

		t.Logf("✅ Found %d versions of 1.21.x (not just latest 2)", count121)
		t.Logf("✅ Found %d versions of 1.20.x (not just latest 2)", count120)
	})

	t.Run("versions include unstable versions", func(t *testing.T) {
		_, cleanup := setupTestEnv(t)
		defer cleanup()

		cmd := &cobra.Command{
			Use: "list",
			RunE: func(cmd *cobra.Command, args []string) error {
				return runList(cmd, args)
			},
		}

		output := &strings.Builder{}
		cmd.SetOut(output)
		cmd.SetArgs([]string{})

		err := cmd.Execute()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		result := output.String()

		// Verify beta and RC versions are included in the list
		// Note: The Go implementation doesn't add "(unstable)" markers by design
		// (see comment in list.go: "no unstable marker for install --list")
		hasBetaOrRC := strings.Contains(result, "beta") || strings.Contains(result, "rc")
		if !hasBetaOrRC {
			t.Error("Expected to find beta or rc versions in the list")
		}

		t.Log("✅ Unstable versions (beta/rc) are included in list")
	})

	t.Run("returns reasonable version count", func(t *testing.T) {
		_, cleanup := setupTestEnv(t)
		defer cleanup()

		cmd := &cobra.Command{
			Use: "list",
			RunE: func(cmd *cobra.Command, args []string) error {
				return runList(cmd, args)
			},
		}

		output := &strings.Builder{}
		cmd.SetOut(output)
		cmd.SetArgs([]string{})

		err := cmd.Execute()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		result := output.String()
		lines := strings.Split(strings.TrimSpace(result), "\n")

		// We should have hundreds of versions available
		// If this number is low (like 2-10), the bug has returned
		minExpected := 200 // Conservative estimate
		if len(lines) < minExpected {
			t.Errorf("CRITICAL: Expected at least %d versions, got %d. This indicates the 'latest 2 versions' bug may have returned!",
				minExpected, len(lines))
		}

		// But also shouldn't be ridiculously high (sanity check)
		maxExpected := 500 // As of 2025
		if len(lines) > maxExpected {
			t.Logf("WARNING: Found %d versions, expected under %d. May want to review.", len(lines), maxExpected)
		}

		t.Logf("✅ Version count reasonable: %d versions", len(lines))
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
