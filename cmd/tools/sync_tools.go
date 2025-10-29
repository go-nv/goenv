package tools

import (
	"github.com/go-nv/goenv/cmd/shims"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/pathutil"
	"github.com/go-nv/goenv/internal/tooldetect"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var syncToolsCmd = &cobra.Command{
	Use:   "sync-tools [source-version] [target-version]",
	Short: "Sync/replicate installed Go tools between versions",
	Long: `Replicates all installed Go tools from a source Go version to a target version.

This command discovers tools in the source version and reinstalls them (from source)
in the target version. The source version remains unchanged - think of this as
"syncing" or "replicating" your tool setup rather than "moving" tools.

Smart defaults when arguments are omitted:
  • No args: Sync from version with most tools → current version
  • One arg: Sync from that version → current version
  • Two args: Sync from source → target (explicit control)

This is useful when upgrading Go versions and wanting to maintain your tool
environment across versions.

Examples:
  goenv tools sync                         # Auto: best source → current version
  goenv tools sync 1.24.1                  # From 1.24.1 → current version
  goenv tools sync 1.24.1 1.25.2           # From 1.24.1 → 1.25.2 (explicit)
  goenv tools sync --dry-run               # Preview auto-sync
  goenv tools sync 1.24.1 --dry-run        # Preview sync from 1.24.1
  goenv tools sync --select gopls,delve    # Only sync specific tools
  goenv tools sync --exclude staticcheck   # Exclude certain tools`,
	Args: cobra.MaximumNArgs(2),
	RunE: runSyncTools,
}

var syncToolsFlags struct {
	dryRun  bool
	select_ string // select is a keyword, use select_
	exclude string
}

func init() {
	// Now registered as subcommand in tools.go
	syncToolsCmd.Flags().BoolVarP(&syncToolsFlags.dryRun, "dry-run", "n", false, "Show what would be synced without actually syncing")
	syncToolsCmd.Flags().StringVar(&syncToolsFlags.select_, "select", "", "Comma-separated list of tools to sync (e.g., gopls,delve)")
	syncToolsCmd.Flags().StringVar(&syncToolsFlags.exclude, "exclude", "", "Comma-separated list of tools to exclude from sync")
}

func runSyncTools(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	var sourceVersion, targetVersion string
	var err error

	// Handle smart defaults based on number of arguments
	switch len(args) {
	case 0:
		// No args: Auto-detect best source → current version
		sourceVersion, targetVersion, err = autoDetectVersions(cfg, mgr)
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%sAuto-detected: syncing from Go %s → Go %s\n", utils.Emoji("🔍 "), sourceVersion, targetVersion)

	case 1:
		// One arg: source → current version
		sourceVersion = args[0]
		targetVersion, _, err = mgr.GetCurrentVersion()
		if err != nil || targetVersion == "" {
			return fmt.Errorf("cannot determine current Go version: use 'goenv local' or 'goenv global' to set one")
		}
		if targetVersion == "system" {
			return fmt.Errorf("cannot sync tools to 'system' version")
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%sSyncing from Go %s → current Go %s\n", utils.Emoji("📦 "), sourceVersion, targetVersion)

	case 2:
		// Two args: explicit source → target
		sourceVersion = args[0]
		targetVersion = args[1]
	}

	// Validate source and target versions
	if sourceVersion == targetVersion {
		return fmt.Errorf("source and target versions are the same: %s (nothing to sync)", sourceVersion)
	}

	// Check if source version exists
	sourcePath := filepath.Join(cfg.Root, "versions", sourceVersion)
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("source Go version %s is not installed", sourceVersion)
	}

	// Check if target version exists
	targetPath := filepath.Join(cfg.Root, "versions", targetVersion)
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return fmt.Errorf("target Go version %s is not installed", targetVersion)
	}

	// Discover tools in source version
	fmt.Fprintf(cmd.OutOrStdout(), "%sDiscovering tools in Go %s...\n", utils.Emoji("🔍 "), sourceVersion)
	sourceTools, err := tooldetect.ListInstalledTools(cfg.Root, sourceVersion)
	if err != nil {
		return fmt.Errorf("failed to list tools in source version: %w", err)
	}

	if len(sourceTools) == 0 {
		fmt.Printf("No Go tools found in Go %s.\n", sourceVersion)
		return nil
	}

	// Filter tools based on --select and --exclude flags
	toolsToSync := filterTools(sourceTools)

	if len(toolsToSync) == 0 {
		fmt.Println("No tools to sync after applying filters.")
		return nil
	}

	// Display summary
	fmt.Fprintf(cmd.OutOrStdout(), "\n%sFound %d tool(s) to sync:\n", utils.Emoji("📦 "), len(toolsToSync))
	for _, tool := range toolsToSync {
		version := tool.Version
		if version == "" {
			version = "unknown"
		}
		fmt.Printf("  • %s (%s) @ %s\n", tool.Name, tool.PackagePath, version)
	}
	fmt.Println()

	if syncToolsFlags.dryRun {
		fmt.Fprintf(cmd.OutOrStdout(), "%sDry run mode - no tools will be installed\n", utils.Emoji("🏃 "))
		fmt.Printf("Would install %d tool(s) in Go %s\n", len(toolsToSync), targetVersion)
		return nil
	}

	// Confirm sync
	fmt.Printf("Syncing to Go %s...\n\n", targetVersion)

	// Install each tool in target version
	successCount := 0
	failCount := 0

	for i, tool := range toolsToSync {
		fmt.Printf("[%d/%d] Installing %s...\n", i+1, len(toolsToSync), tool.Name)

		// Construct install command
		var packagePath string
		if tool.Version != "" && tool.Version != "unknown" {
			packagePath = fmt.Sprintf("%s@%s", tool.PackagePath, tool.Version)
		} else {
			packagePath = fmt.Sprintf("%s@latest", tool.PackagePath)
		}

		// Use target version's go binary
		// The version directory IS the GOROOT (no extra 'go' subdirectory)
		goBinaryBase := filepath.Join(targetPath, "bin", "go")

		// Find the executable (handles .exe and .bat on Windows)
		goBinary, err := pathutil.FindExecutable(goBinaryBase)
		if err != nil {
			return fmt.Errorf("go binary not found in target version: %w", err)
		}

		// Set GOPATH for target version
		targetGOPATH := filepath.Join(targetPath, "gopath")
		if err := os.MkdirAll(targetGOPATH, 0755); err != nil {
			return fmt.Errorf("failed to create GOPATH directory: %w", err)
		}

		// Run go install
		installCmd := exec.Command(goBinary, "install", packagePath)
		installCmd.Env = append(os.Environ(),
			fmt.Sprintf("GOPATH=%s", targetGOPATH),
			fmt.Sprintf("GOBIN=%s", filepath.Join(targetGOPATH, "bin")),
		)
		installCmd.Stdout = os.Stdout
		installCmd.Stderr = os.Stderr

		if err := installCmd.Run(); err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "  %sFailed to install %s: %v\n", utils.Emoji("❌ "), tool.Name, err)
			failCount++
			continue
		}

		fmt.Fprintf(cmd.OutOrStdout(), "  %sSuccessfully installed %s\n", utils.Emoji("✅ "), tool.Name)
		successCount++
	}

	// Summary
	fmt.Fprintf(cmd.OutOrStdout(), "\n%sSync complete!\n", utils.Emoji("✨ "))
	fmt.Printf("  • Successfully synced: %d tool(s)\n", successCount)
	if failCount > 0 {
		fmt.Printf("  • Failed: %d tool(s)\n", failCount)
	}

	// Trigger rehash for target version if it's the current version
	currentVersion, _, err := mgr.GetCurrentVersion()
	if err == nil && currentVersion == targetVersion {
		fmt.Fprintf(cmd.OutOrStdout(), "\n%sRehashing shims for current version...\n", utils.Emoji("🔄 "))
		if err := shims.RunRehash(cmd, []string{}); err != nil {
			fmt.Printf("Warning: Failed to rehash: %v\n", err)
		}
	}

	return nil
}

// filterTools applies --select and --exclude filters to the tool list
func filterTools(tools []tooldetect.Tool) []tooldetect.Tool {
	// Build set of selected tools
	var selectSet map[string]bool
	if syncToolsFlags.select_ != "" {
		selectSet = make(map[string]bool)
		for _, name := range strings.Split(syncToolsFlags.select_, ",") {
			selectSet[strings.TrimSpace(name)] = true
		}
	}

	// Build set of excluded tools
	var excludeSet map[string]bool
	if syncToolsFlags.exclude != "" {
		excludeSet = make(map[string]bool)
		for _, name := range strings.Split(syncToolsFlags.exclude, ",") {
			excludeSet[strings.TrimSpace(name)] = true
		}
	}

	// Filter tools
	var filtered []tooldetect.Tool
	for _, tool := range tools {
		// If select is specified, only include selected tools
		if selectSet != nil && !selectSet[tool.Name] {
			continue
		}

		// If exclude is specified, skip excluded tools
		if excludeSet != nil && excludeSet[tool.Name] {
			continue
		}

		filtered = append(filtered, tool)
	}

	return filtered
}

// autoDetectVersions finds the best source and target versions for syncing
func autoDetectVersions(cfg *config.Config, mgr *manager.Manager) (source, target string, err error) {
	// Get current version as target
	currentVersion, _, err := mgr.GetCurrentVersion()
	if err != nil || currentVersion == "" {
		return "", "", fmt.Errorf("cannot determine current Go version: use 'goenv local' or 'goenv global' to set one")
	}

	if currentVersion == "system" {
		return "", "", fmt.Errorf("cannot sync tools to 'system' version")
	}

	// Get all installed versions
	installedVersions, err := mgr.ListInstalledVersions()
	if err != nil {
		return "", "", fmt.Errorf("failed to list installed versions: %w", err)
	}

	if len(installedVersions) < 2 {
		return "", "", fmt.Errorf("need at least 2 Go versions installed to auto-sync (found %d)", len(installedVersions))
	}

	// Find version with most tools (excluding current version)
	maxTools := 0
	bestSource := ""

	for _, version := range installedVersions {
		if version == currentVersion {
			continue
		}

		tools, err := tooldetect.ListInstalledTools(cfg.Root, version)
		if err != nil {
			continue
		}

		if len(tools) > maxTools {
			maxTools = len(tools)
			bestSource = version
		}
	}

	if bestSource == "" {
		return "", "", fmt.Errorf("no other Go versions have tools installed (current: %s)", currentVersion)
	}

	if maxTools == 0 {
		return "", "", fmt.Errorf("no tools found in any other Go version (current: %s)", currentVersion)
	}

	return bestSource, currentVersion, nil
}
