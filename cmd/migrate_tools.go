package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/tooldetect"
	"github.com/spf13/cobra"
)

var migrateToolsCmd = &cobra.Command{
	Use:   "migrate-tools <source-version> <target-version>",
	Short: "Migrate installed Go tools from one Go version to another",
	Long: `Migrates all installed Go tools from a source Go version to a target Go version.

This command is useful when upgrading Go versions and wanting to maintain
your tool environment. It discovers all tools in the source version and
reinstalls them in the target version.

Examples:
  goenv migrate-tools 1.24.1 1.25.2           # Migrate all tools
  goenv migrate-tools 1.24.1 1.25.2 --dry-run # Preview migration
  goenv migrate-tools 1.24.1 1.25.2 --select gopls,delve  # Migrate specific tools
  goenv migrate-tools 1.24.1 1.25.2 --exclude staticcheck  # Exclude tools`,
	Args: cobra.ExactArgs(2),
	RunE: runMigrateTools,
}

var migrateToolsFlags struct {
	dryRun  bool
	select_ string // select is a keyword, use select_
	exclude string
}

func init() {
	migrateToolsCmd.Flags().BoolVar(&migrateToolsFlags.dryRun, "dry-run", false, "Show what would be migrated without actually migrating")
	migrateToolsCmd.Flags().StringVar(&migrateToolsFlags.select_, "select", "", "Comma-separated list of tools to migrate (e.g., gopls,delve)")
	migrateToolsCmd.Flags().StringVar(&migrateToolsFlags.exclude, "exclude", "", "Comma-separated list of tools to exclude from migration")
	rootCmd.AddCommand(migrateToolsCmd)
}

func runMigrateTools(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	sourceVersion := args[0]
	targetVersion := args[1]

	// Validate source and target versions
	if sourceVersion == targetVersion {
		return fmt.Errorf("source and target versions cannot be the same: %s", sourceVersion)
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
	fmt.Printf("🔍 Discovering tools in Go %s...\n", sourceVersion)
	sourceTools, err := tooldetect.ListInstalledTools(cfg.Root, sourceVersion)
	if err != nil {
		return fmt.Errorf("failed to list tools in source version: %w", err)
	}

	if len(sourceTools) == 0 {
		fmt.Printf("No Go tools found in Go %s.\n", sourceVersion)
		return nil
	}

	// Filter tools based on --select and --exclude flags
	toolsToMigrate := filterTools(sourceTools)

	if len(toolsToMigrate) == 0 {
		fmt.Println("No tools to migrate after applying filters.")
		return nil
	}

	// Display summary
	fmt.Printf("\n📦 Found %d tool(s) to migrate:\n", len(toolsToMigrate))
	for _, tool := range toolsToMigrate {
		version := tool.Version
		if version == "" {
			version = "unknown"
		}
		fmt.Printf("  • %s (%s) @ %s\n", tool.Name, tool.PackagePath, version)
	}
	fmt.Println()

	if migrateToolsFlags.dryRun {
		fmt.Println("🏃 Dry run mode - no tools will be installed")
		fmt.Printf("Would install %d tool(s) in Go %s\n", len(toolsToMigrate), targetVersion)
		return nil
	}

	// Confirm migration
	fmt.Printf("Migrating to Go %s...\n\n", targetVersion)

	// Install each tool in target version
	successCount := 0
	failCount := 0

	for i, tool := range toolsToMigrate {
		fmt.Printf("[%d/%d] Installing %s...\n", i+1, len(toolsToMigrate), tool.Name)

		// Construct install command
		packagePath := tool.PackagePath
		if tool.Version != "" && tool.Version != "unknown" {
			packagePath = fmt.Sprintf("%s@%s", tool.PackagePath, tool.Version)
		} else {
			packagePath = fmt.Sprintf("%s@latest", tool.PackagePath)
		}

		// Use target version's go binary
		goBinary := filepath.Join(targetPath, "go", "bin", "go")

		// On Windows, add .exe extension
		if runtime.GOOS == "windows" {
			goBinary += ".exe"
		}

		if _, err := os.Stat(goBinary); os.IsNotExist(err) {
			return fmt.Errorf("go binary not found in target version: %s", goBinary)
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
			fmt.Printf("  ❌ Failed to install %s: %v\n", tool.Name, err)
			failCount++
			continue
		}

		fmt.Printf("  ✅ Successfully installed %s\n", tool.Name)
		successCount++
	}

	// Summary
	fmt.Printf("\n✨ Migration complete!\n")
	fmt.Printf("  • Successfully migrated: %d tool(s)\n", successCount)
	if failCount > 0 {
		fmt.Printf("  • Failed: %d tool(s)\n", failCount)
	}

	// Trigger rehash for target version if it's the current version
	currentVersion, _, err := mgr.GetCurrentVersion()
	if err == nil && currentVersion == targetVersion {
		fmt.Println("\n🔄 Rehashing shims for current version...")
		if err := runRehash(cmd, []string{}); err != nil {
			fmt.Printf("Warning: Failed to rehash: %v\n", err)
		}
	}

	return nil
}

// filterTools applies --select and --exclude filters to the tool list
func filterTools(tools []tooldetect.Tool) []tooldetect.Tool {
	// Build set of selected tools
	var selectSet map[string]bool
	if migrateToolsFlags.select_ != "" {
		selectSet = make(map[string]bool)
		for _, name := range strings.Split(migrateToolsFlags.select_, ",") {
			selectSet[strings.TrimSpace(name)] = true
		}
	}

	// Build set of excluded tools
	var excludeSet map[string]bool
	if migrateToolsFlags.exclude != "" {
		excludeSet = make(map[string]bool)
		for _, name := range strings.Split(migrateToolsFlags.exclude, ",") {
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
