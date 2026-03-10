package compliance

import (
	"fmt"

	"github.com/go-nv/goenv/internal/sbom"
	"github.com/spf13/cobra"
)

var (
	hookForce       bool
	hookFailOnError bool
	hookOutputPath  string
	hookFormat      string
	hookQuiet       bool
)

var sbomHooksCmd = &cobra.Command{
	Use:   "hooks",
	Short: "Manage Git hooks for SBOM generation",
	Long: `Manage Git hooks for automatic SBOM generation.

The hooks command allows you to install, uninstall, and manage Git pre-commit 
hooks that automatically generate SBOMs when go.mod or go.sum files change.

Examples:
  # Install pre-commit hook with defaults
  goenv sbom hooks install

  # Install with custom output path
  goenv sbom hooks install --output sbom.cyclonedx.json

  # Install without failing on errors
  goenv sbom hooks install --no-fail-on-error

  # Check hook status
  goenv sbom hooks status

  # Uninstall hook
  goenv sbom hooks uninstall`,
}

var sbomHooksInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install pre-commit hook for SBOM generation",
	Long: `Install a Git pre-commit hook that automatically generates SBOMs.

The hook will:
- Detect when go.mod or go.sum files are modified
- Automatically generate an updated SBOM
- Stage the SBOM file for commit
- Optionally fail the commit if SBOM generation fails

Configuration options allow you to customize the hook behavior.`,
	RunE: runHooksInstall,
}

var sbomHooksUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall pre-commit hook",
	Long:  `Uninstall the goenv-managed Git pre-commit hook.`,
	RunE:  runHooksUninstall,
}

var sbomHooksStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show hook installation status",
	Long:  `Display information about the current hook installation status.`,
	RunE:  runHooksStatus,
}

func init() {
	// Add hooks subcommands
	sbomHooksCmd.AddCommand(sbomHooksInstallCmd)
	sbomHooksCmd.AddCommand(sbomHooksUninstallCmd)
	sbomHooksCmd.AddCommand(sbomHooksStatusCmd)

	// Install flags
	sbomHooksInstallCmd.Flags().BoolVar(&hookForce, "force", false, "Overwrite existing hook")
	sbomHooksInstallCmd.Flags().BoolVar(&hookFailOnError, "fail-on-error", true, "Prevent commits if SBOM generation fails")
	sbomHooksInstallCmd.Flags().StringVar(&hookOutputPath, "output", "sbom.json", "SBOM output path (relative to repo root)")
	sbomHooksInstallCmd.Flags().StringVar(&hookFormat, "format", "cyclonedx", "SBOM format (cyclonedx, spdx, syft)")
	sbomHooksInstallCmd.Flags().BoolVar(&hookQuiet, "quiet", false, "Suppress hook output")

	// Add to parent
	sbomCmd.AddCommand(sbomHooksCmd)
}

func runHooksInstall(cmd *cobra.Command, args []string) error {
	// Create hook manager
	manager, err := sbom.NewHookManager("")
	if err != nil {
		return fmt.Errorf("failed to initialize hook manager: %w", err)
	}

	// Check if already installed
	installed, err := manager.IsHookInstalled()
	if err != nil {
		return fmt.Errorf("failed to check hook status: %w", err)
	}

	if installed && !hookForce {
		fmt.Println("✓ Pre-commit hook is already installed")
		fmt.Println("\nTo reinstall, use --force flag")
		return nil
	}

	// Create config
	config := sbom.HookConfig{
		AutoGenerate: true,
		FailOnError:  hookFailOnError,
		OutputPath:   hookOutputPath,
		Format:       hookFormat,
		Quiet:        hookQuiet,
	}

	// Install hook
	if err := manager.InstallHook(config); err != nil {
		return fmt.Errorf("failed to install hook: %w", err)
	}

	// Print success message
	fmt.Println("✅ Pre-commit hook installed successfully!")
	fmt.Println()
	fmt.Println("Configuration:")
	fmt.Printf("  Output: %s\n", hookOutputPath)
	fmt.Printf("  Format: %s\n", hookFormat)
	fmt.Printf("  Fail on error: %t\n", hookFailOnError)
	fmt.Printf("  Quiet mode: %t\n", hookQuiet)
	fmt.Println()
	fmt.Println("The hook will automatically generate SBOMs when go.mod or go.sum changes.")
	fmt.Println()
	fmt.Println("To uninstall: goenv sbom hooks uninstall")

	return nil
}

func runHooksUninstall(cmd *cobra.Command, args []string) error {
	// Create hook manager
	manager, err := sbom.NewHookManager("")
	if err != nil {
		return fmt.Errorf("failed to initialize hook manager: %w", err)
	}

	// Check if installed
	installed, err := manager.IsHookInstalled()
	if err != nil {
		return fmt.Errorf("failed to check hook status: %w", err)
	}

	if !installed {
		fmt.Println("ℹ️  No goenv-managed pre-commit hook found")
		return nil
	}

	// Uninstall hook
	if err := manager.UninstallHook(); err != nil {
		return fmt.Errorf("failed to uninstall hook: %w", err)
	}

	fmt.Println("✅ Pre-commit hook uninstalled successfully")
	return nil
}

func runHooksStatus(cmd *cobra.Command, args []string) error {
	// Create hook manager
	manager, err := sbom.NewHookManager("")
	if err != nil {
		return fmt.Errorf("failed to initialize hook manager: %w", err)
	}

	// Get status
	status, err := manager.GetHookStatus()
	if err != nil {
		return fmt.Errorf("failed to get hook status: %w", err)
	}

	// Display status
	fmt.Println("Git Hook Status")
	fmt.Println("═══════════════════════════════════════")
	fmt.Println()
	fmt.Printf("Repository: %s\n", status["repo_root"])
	fmt.Printf("Hook path:  %s\n", status["hook_path"])
	fmt.Printf("Goenv path: %s\n", status["goenv_path"])
	fmt.Println()

	installed := status["installed"].(bool)
	if installed {
		fmt.Println("Status: ✅ Installed")
		fmt.Println()
		fmt.Println("Hook will automatically generate SBOMs when go.mod or go.sum changes.")
		fmt.Println()
		fmt.Println("To uninstall: goenv sbom hooks uninstall")
	} else {
		fmt.Println("Status: ❌ Not installed")
		fmt.Println()
		fmt.Println("To install: goenv sbom hooks install")
	}

	return nil
}
