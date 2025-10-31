package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var (
	uninstallAllVersions bool
	uninstallGlobal      bool
	uninstallForce       bool
	uninstallDryRun      bool
	uninstallVerbose     bool
)

// NewUninstallCommand creates the tools uninstall command
func NewUninstallCommand(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "uninstall <tool[@version]>...",
		Short: "Uninstall Go tools",
		Long: `Uninstall Go tools from the current Go version or globally.

By default, uninstalls from the current active Go version's GOPATH/bin.
Use --all to uninstall from all installed Go versions.
Use --global to uninstall from the global GOPATH/bin.

Examples:
  # Uninstall from current Go version
  goenv tools uninstall gopls
  
  # Uninstall from all Go versions
  goenv tools uninstall gopls --all
  
  # Uninstall from global GOPATH
  goenv tools uninstall gopls --global
  
  # Uninstall multiple tools
  goenv tools uninstall gopls golangci-lint staticcheck
  
  # Preview what would be removed (dry run)
  goenv tools uninstall gopls --all --dry-run
  
  # Force removal without confirmation
  goenv tools uninstall gopls --force
  
  # Verbose output showing all files removed
  goenv tools uninstall gopls --verbose`,
		Example: `  goenv tools uninstall gopls
  goenv tools uninstall gopls --all
  goenv tools uninstall gopls golangci-lint --global
  goenv tools uninstall gopls --dry-run --verbose`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUninstall(cfg, args)
		},
	}

	cmd.Flags().BoolVar(&uninstallAllVersions, "all", false, "Uninstall from all installed Go versions")
	cmd.Flags().BoolVar(&uninstallGlobal, "global", false, "Uninstall from global GOPATH/bin")
	cmd.Flags().BoolVar(&uninstallForce, "force", false, "Skip confirmation prompts")
	cmd.Flags().BoolVar(&uninstallDryRun, "dry-run", false, "Show what would be removed without actually removing")
	cmd.Flags().BoolVarP(&uninstallVerbose, "verbose", "v", false, "Show detailed output")

	return cmd
}

type toolUninstallTarget struct {
	ToolName    string
	GoVersion   string // Empty for global
	BinPath     string
	Exists      bool
	BinaryFiles []string
}

func runUninstall(cfg *config.Config, toolNames []string) error {
	// Parse tool names (strip @version if present, we remove the binary regardless)
	var cleanToolNames []string
	for _, name := range toolNames {
		// Strip @version suffix
		if idx := strings.Index(name, "@"); idx != -1 {
			name = name[:idx]
		}
		cleanToolNames = append(cleanToolNames, name)
	}

	// Determine target Go versions
	var targets []toolUninstallTarget

	if uninstallGlobal {
		// Global GOPATH uninstall
		targets = append(targets, findGlobalToolTargets(cfg, cleanToolNames)...)
	} else if uninstallAllVersions {
		// All installed Go versions
		targets = append(targets, findAllVersionToolTargets(cfg, cleanToolNames)...)
	} else {
		// Current Go version only
		targets = append(targets, findCurrentVersionToolTargets(cfg, cleanToolNames)...)
	}

	if len(targets) == 0 {
		fmt.Fprintf(os.Stderr, "%s No tools found to uninstall\n", utils.EmojiOr("ℹ️  ", ""))
		return nil
	}

	// Filter to only existing tools
	var existingTargets []toolUninstallTarget
	for _, target := range targets {
		if target.Exists {
			existingTargets = append(existingTargets, target)
		}
	}

	if len(existingTargets) == 0 {
		fmt.Fprintf(os.Stderr, "%s No installed tools found matching: %s\n",
			utils.EmojiOr("ℹ️  ", ""),
			utils.Yellow(strings.Join(cleanToolNames, ", ")))
		return nil
	}

	// Show what will be removed
	if err := showUninstallPlan(existingTargets); err != nil {
		return err
	}

	// Dry run stops here
	if uninstallDryRun {
		fmt.Fprintf(os.Stdout, "\n%s Dry run mode - no changes made\n",
			utils.EmojiOr("ℹ️  ", ""))
		return nil
	}

	// Confirm unless --force
	if !uninstallForce {
		if !utils.PromptYesNoSimple(fmt.Sprintf("Remove %d tool(s)?", len(existingTargets))) {
			fmt.Fprintf(os.Stderr, "%s Cancelled\n", utils.EmojiOr("❌ ", ""))
			return nil
		}
	}

	// Execute uninstalls
	return executeUninstalls(existingTargets)
}

func findCurrentVersionToolTargets(cfg *config.Config, toolNames []string) []toolUninstallTarget {
	mgr := manager.NewManager(cfg)
	currentVersion, _, err := mgr.GetCurrentVersion()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s Failed to get current Go version: %v\n",
			utils.EmojiOr("⚠️  ", ""), err)
		return nil
	}

	if currentVersion == "" || currentVersion == "system" {
		fmt.Fprintf(os.Stderr, "%s Cannot uninstall tools from system Go\n",
			utils.EmojiOr("⚠️  ", ""))
		fmt.Fprintf(os.Stderr, "   Use --global to uninstall from global GOPATH\n")
		return nil
	}

	gopath := filepath.Join(cfg.Root, "versions", currentVersion, "gopath")
	binPath := filepath.Join(gopath, "bin")

	var targets []toolUninstallTarget
	for _, toolName := range toolNames {
		target := toolUninstallTarget{
			ToolName:  toolName,
			GoVersion: currentVersion,
			BinPath:   binPath,
		}

		// Find all related binaries
		target.BinaryFiles = findToolBinaries(binPath, toolName)
		target.Exists = len(target.BinaryFiles) > 0

		targets = append(targets, target)
	}

	return targets
}

func findAllVersionToolTargets(cfg *config.Config, toolNames []string) []toolUninstallTarget {
	versionsDir := filepath.Join(cfg.Root, "versions")
	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s Failed to read versions directory: %v\n",
			utils.EmojiOr("⚠️  ", ""), err)
		return nil
	}

	var targets []toolUninstallTarget
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		version := entry.Name()
		gopath := filepath.Join(versionsDir, version, "gopath")
		binPath := filepath.Join(gopath, "bin")

		// Check if bin directory exists
		if _, err := os.Stat(binPath); os.IsNotExist(err) {
			continue
		}

		for _, toolName := range toolNames {
			target := toolUninstallTarget{
				ToolName:  toolName,
				GoVersion: version,
				BinPath:   binPath,
			}

			// Find all related binaries
			target.BinaryFiles = findToolBinaries(binPath, toolName)
			target.Exists = len(target.BinaryFiles) > 0

			if target.Exists {
				targets = append(targets, target)
			}
		}
	}

	return targets
}

func findGlobalToolTargets(cfg *config.Config, toolNames []string) []toolUninstallTarget {
	// Global GOPATH from environment or default
	globalGopath := os.Getenv(utils.EnvVarGopath)
	if globalGopath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s Failed to get home directory: %v\n",
				utils.EmojiOr("⚠️  ", ""), err)
			return nil
		}
		globalGopath = filepath.Join(homeDir, "go")
	}

	binPath := filepath.Join(globalGopath, "bin")

	var targets []toolUninstallTarget
	for _, toolName := range toolNames {
		target := toolUninstallTarget{
			ToolName:  toolName,
			GoVersion: "", // Empty for global
			BinPath:   binPath,
		}

		// Find all related binaries
		target.BinaryFiles = findToolBinaries(binPath, toolName)
		target.Exists = len(target.BinaryFiles) > 0

		targets = append(targets, target)
	}

	return targets
}

func findToolBinaries(binPath, toolName string) []string {
	var binaries []string

	// Check if bin directory exists
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		return binaries
	}

	entries, err := os.ReadDir(binPath)
	if err != nil {
		return binaries
	}

	// Find exact matches and platform-specific variants
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Exact match
		if name == toolName {
			binaries = append(binaries, filepath.Join(binPath, name))
			continue
		}

		// Platform-specific variants (e.g., gopls.exe on Windows)
		if strings.HasPrefix(name, toolName+".") {
			binaries = append(binaries, filepath.Join(binPath, name))
			continue
		}
	}

	return binaries
}

func showUninstallPlan(targets []toolUninstallTarget) error {
	fmt.Fprintf(os.Stdout, "\n%s %s\n\n",
		utils.EmojiOr("🗑️  ", ""),
		utils.BoldBlue("Uninstall Plan"))

	// Group by Go version
	versionGroups := make(map[string][]toolUninstallTarget)
	for _, target := range targets {
		key := target.GoVersion
		if key == "" {
			key = "global"
		}
		versionGroups[key] = append(versionGroups[key], target)
	}

	// Display grouped
	for version, groupTargets := range versionGroups {
		if version == "global" {
			fmt.Fprintf(os.Stdout, "%s %s\n",
				utils.BoldCyan("Global GOPATH:"),
				utils.Gray(groupTargets[0].BinPath))
		} else {
			fmt.Fprintf(os.Stdout, "%s %s\n",
				utils.BoldCyan("Go "+version+":"),
				utils.Gray(groupTargets[0].BinPath))
		}

		for _, target := range groupTargets {
			fmt.Fprintf(os.Stdout, "  %s %s\n",
				utils.Red("✗"),
				utils.BoldWhite(target.ToolName))

			if uninstallVerbose {
				for _, binFile := range target.BinaryFiles {
					fmt.Fprintf(os.Stdout, "    %s %s\n",
						utils.Gray("→"),
						utils.Gray(filepath.Base(binFile)))
				}
			} else {
				fmt.Fprintf(os.Stdout, "    %s\n",
					utils.Gray(fmt.Sprintf("(%d file(s))", len(target.BinaryFiles))))
			}
		}
		fmt.Fprintln(os.Stdout)
	}

	// Summary
	totalFiles := 0
	for _, target := range targets {
		totalFiles += len(target.BinaryFiles)
	}

	fmt.Fprintf(os.Stdout, "%s\n",
		utils.Gray(fmt.Sprintf("Total: %d tool(s), %d file(s)", len(targets), totalFiles)))

	return nil
}

func executeUninstalls(targets []toolUninstallTarget) error {
	fmt.Fprintf(os.Stdout, "\n%s %s\n\n",
		utils.EmojiOr("🗑️  ", ""),
		utils.BoldBlue("Uninstalling Tools"))

	successCount := 0
	failureCount := 0

	for _, target := range targets {
		var label string
		if target.GoVersion != "" {
			label = fmt.Sprintf("%s (Go %s)", target.ToolName, target.GoVersion)
		} else {
			label = fmt.Sprintf("%s (global)", target.ToolName)
		}

		// Remove all binary files
		failed := false
		for _, binFile := range target.BinaryFiles {
			if err := os.Remove(binFile); err != nil {
				fmt.Fprintf(os.Stderr, "%s Failed to remove %s: %v\n",
					utils.Red("✗"),
					utils.Yellow(label),
					err)
				failed = true
				break
			}

			if uninstallVerbose {
				fmt.Fprintf(os.Stdout, "  %s Removed %s\n",
					utils.Gray("→"),
					utils.Gray(filepath.Base(binFile)))
			}
		}

		if failed {
			failureCount++
		} else {
			fmt.Fprintf(os.Stdout, "%s Uninstalled %s\n",
				utils.Green("✓"),
				utils.BoldWhite(label))
			successCount++
		}
	}

	// Summary
	fmt.Fprintln(os.Stdout)
	if failureCount == 0 {
		fmt.Fprintf(os.Stdout, "%s Successfully uninstalled %d tool(s)\n",
			utils.EmojiOr("✅ ", ""),
			successCount)
	} else {
		fmt.Fprintf(os.Stdout, "%s Uninstalled %d tool(s), %d failed\n",
			utils.EmojiOr("⚠️  ", ""),
			successCount,
			failureCount)
	}

	if failureCount > 0 {
		return fmt.Errorf("%d tool(s) failed to uninstall", failureCount)
	}

	return nil
}
