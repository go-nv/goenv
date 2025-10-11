package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/shims"
	"github.com/spf13/cobra"
)

var rehashCmd = &cobra.Command{
	Use:   "rehash",
	Short: "Rehash goenv shims (run this after installing executables)",
	Long:  "Scans all installed Go versions and creates shim files for their executables",
	RunE:  runRehash,
}

var shimsCmd = &cobra.Command{
	Use:   "shims",
	Short: "List existing goenv shims",
	Long:  "Display all available shim files in the goenv shims directory",
	RunE:  runShims,
}

var whichCmd = &cobra.Command{
	Use:   "which <command>",
	Short: "Display the full path to an executable",
	Long:  "Shows the full path to the executable that goenv will invoke for the given command",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runWhich,
}

var whichFlags struct {
	complete bool
}

var whenceCmd = &cobra.Command{
	Use:   "whence <command>",
	Short: "List all Go versions that contain the given executable",
	Long:  "Display which installed Go versions have the specified command available",
	Args:  cobra.MaximumNArgs(2),
	RunE:  runWhence,
}

var whenceFlags struct {
	path     bool
	complete bool
}

var shimsFlags struct {
	short    bool
	complete bool
}

func init() {
	rootCmd.AddCommand(rehashCmd)
	rootCmd.AddCommand(shimsCmd)
	rootCmd.AddCommand(whichCmd)
	rootCmd.AddCommand(whenceCmd)

	shimsCmd.Flags().BoolVar(&shimsFlags.short, "short", false, "Show only shim names without full paths")
	shimsCmd.Flags().BoolVar(&shimsFlags.complete, "complete", false, "Show completion options")
	_ = shimsCmd.Flags().MarkHidden("complete")
	whichCmd.Flags().BoolVar(&whichFlags.complete, "complete", false, "Show completion options")
	_ = whichCmd.Flags().MarkHidden("complete")
	whenceCmd.Flags().BoolVar(&whenceFlags.path, "path", false, "Show full paths to executables")
	whenceCmd.Flags().BoolVar(&whenceFlags.complete, "complete", false, "Show completion options")
	_ = whenceCmd.Flags().MarkHidden("complete")
}

func runRehash(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	shimMgr := shims.NewShimManager(cfg)

	if cfg.Debug {
		cmd.Println("Debug: Rehashing goenv shims...")
	}

	cmd.Println("Rehashing...")
	if err := shimMgr.Rehash(); err != nil {
		return fmt.Errorf("failed to rehash shims: %w", err)
	}

	shimList, err := shimMgr.ListShims()
	if err != nil {
		return fmt.Errorf("failed to list shims: %w", err)
	}

	cmd.Printf("Rehashed %d shims\n", len(shimList))
	return nil
}

func runShims(cmd *cobra.Command, args []string) error {
	// Handle completion mode
	if shimsFlags.complete {
		cmd.Println("--short")
		return nil
	}

	cfg := config.Load()
	shimMgr := shims.NewShimManager(cfg)

	shimList, err := shimMgr.ListShims()
	if err != nil {
		return fmt.Errorf("failed to list shims: %w", err)
	}

	for _, shim := range shimList {
		if shimsFlags.short {
			cmd.Println(shim)
		} else {
			cmd.Println(filepath.Join(cfg.ShimsDir(), shim))
		}
	}

	return nil
}

func runWhich(cmd *cobra.Command, args []string) error {
	// Handle completion mode
	if whichFlags.complete {
		return nil
	}

	// Require command argument
	if len(args) == 0 {
		return fmt.Errorf("Usage: goenv which <command>")
	}

	commandName := args[0]
	cfg := config.Load()

	// Try using shim manager first (if available)
	shimMgr := shims.NewShimManager(cfg)
	binaryPath, err := shimMgr.WhichBinary(commandName)
	if err == nil {
		cmd.Println(binaryPath)
		return nil
	}

	// Fallback: implement manual search for testing compatibility
	return runWhichManual(cmd, commandName, cfg)
}

func runWhence(cmd *cobra.Command, args []string) error {
	// Handle completion mode
	if whenceFlags.complete {
		cmd.Println("--path")
		return nil
	}

	// Require command argument
	if len(args) == 0 {
		return fmt.Errorf("Usage: goenv whence [--path] <command>")
	}

	commandName := args[0]
	cfg := config.Load()

	// Try using shim manager first
	shimMgr := shims.NewShimManager(cfg)
	versions, err := shimMgr.WhenceVersions(commandName)
	if err == nil && len(versions) > 0 {
		for _, version := range versions {
			if whenceFlags.path {
				versionPath := filepath.Join(cfg.VersionsDir(), version, "bin", commandName)
				cmd.Println(versionPath)
			} else {
				cmd.Println(version)
			}
		}
		return nil
	}

	// Fallback: manual search
	return runWhenceManual(cmd, commandName, cfg)
}

// runWhichManual implements which command logic manually for testing/fallback
func runWhichManual(cmd *cobra.Command, commandName string, cfg *config.Config) error {
	mgr := manager.NewManager(cfg)

	// Get current version(s)
	versionSpec, source, err := mgr.GetCurrentVersion()
	if err != nil {
		return fmt.Errorf("no version set")
	}

	// Split multiple versions
	versions := splitVersionsWhich(versionSpec)

	// Track errors for missing versions
	var missingVersions []string

	// Try to find command in each version
	for _, version := range versions {
		// Check if version is installed
		if version != "system" && !mgr.IsVersionInstalled(version) {
			missingVersions = append(missingVersions, version)
			continue
		}

		// Handle system version
		if version == "system" {
			// Look for command in PATH, excluding GOENV_ROOT/shims
			commandPath, err := findInSystemPath(commandName, cfg.Root)
			if err == nil {
				cmd.Println(commandPath)
				return nil
			}
			continue
		}

		// Look for command in version's bin directory
		versionPath, err := mgr.GetVersionPath(version)
		if err != nil {
			continue
		}

		commandPath := filepath.Join(versionPath, "bin", commandName)
		if _, err := os.Stat(commandPath); err == nil {
			cmd.Println(commandPath)
			return nil
		}
	}

	// If there are missing versions, return error immediately
	if len(missingVersions) > 0 {
		for _, v := range missingVersions {
			cmd.PrintErrf("goenv: version '%s' is not installed (set by %s)\n", v, source)
		}
		return fmt.Errorf("goenv: version '%s' is not installed (set by %s)", missingVersions[0], source)
	}

	// Command not found - check if it exists in other versions
	allVersions, _ := mgr.ListInstalledVersions()
	var foundInVersions []string

	for _, v := range allVersions {
		versionPath, err := mgr.GetVersionPath(v)
		if err != nil {
			continue
		}

		commandPath := filepath.Join(versionPath, "bin", commandName)
		if _, err := os.Stat(commandPath); err == nil {
			foundInVersions = append(foundInVersions, v)
		}
	}

	// Format error message
	errMsg := fmt.Sprintf("goenv: '%s' command not found", commandName)

	if len(foundInVersions) > 0 {
		cmd.PrintErrf("%s\n", errMsg)
		cmd.PrintErrln()
		cmd.PrintErrf("The '%s' command exists in these Go versions:\n", commandName)
		for _, v := range foundInVersions {
			cmd.PrintErrf("  %s\n", v)
		}
		return fmt.Errorf("%s", errMsg)
	}

	return fmt.Errorf("%s", errMsg)
}

// splitVersionsWhich splits a version string by ':' delimiter
func splitVersionsWhich(version string) []string {
	if version == "" {
		return []string{}
	}

	result := []string{}
	current := ""

	for _, ch := range version {
		if ch == ':' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}

	if current != "" {
		result = append(result, current)
	}

	return result
}

// findInSystemPath searches for a command in PATH, excluding goenv shims
func findInSystemPath(commandName string, goenvRoot string) (string, error) {
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return "", fmt.Errorf("command not found")
	}

	shimsDir := filepath.Join(goenvRoot, "shims")
	pathDirs := strings.Split(pathEnv, string(os.PathListSeparator))

	for _, dir := range pathDirs {
		if dir == "" {
			continue
		}

		// Skip goenv shims directory
		if strings.Contains(dir, shimsDir) {
			continue
		}

		commandPath := filepath.Join(dir, commandName)
		info, err := os.Stat(commandPath)
		if err == nil && !info.IsDir() {
			// Check if executable
			if info.Mode()&0111 != 0 {
				return commandPath, nil
			}
		}
	}

	return "", fmt.Errorf("command not found")
}

// runWhenceManual implements whence command logic manually for testing/fallback
func runWhenceManual(cmd *cobra.Command, commandName string, cfg *config.Config) error {
	mgr := manager.NewManager(cfg)

	// Get all installed versions
	allVersions, err := mgr.ListInstalledVersions()
	if err != nil {
		return err
	}

	foundAny := false

	// Check each version for the executable
	for _, version := range allVersions {
		versionPath, err := mgr.GetVersionPath(version)
		if err != nil {
			continue
		}

		commandPath := filepath.Join(versionPath, "bin", commandName)
		info, err := os.Stat(commandPath)
		if err == nil && !info.IsDir() {
			// Check if executable
			if info.Mode()&0111 != 0 {
				foundAny = true
				if whenceFlags.path {
					cmd.Println(commandPath)
				} else {
					cmd.Println(version)
				}
			}
		}
	}

	if !foundAny {
		// Return error but don't print anything (BATS expects empty output + error)
		return fmt.Errorf("no versions found with %s", commandName)
	}

	return nil
}
