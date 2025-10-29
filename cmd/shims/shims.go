package shims

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	cmdhooks "github.com/go-nv/goenv/cmd/hooks"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/go-nv/goenv/internal/hooks"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/shims"
	"github.com/spf13/cobra"
)

var rehashCmd = &cobra.Command{
	Use:   "rehash",
	Short: "Rehash goenv shims (run this after installing executables)",
	Long:  "Scans all installed Go versions and creates shim files for their executables",
	RunE:  RunRehash,
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
	Args: func(cmd *cobra.Command, args []string) error {
		// Handle completion mode
		if whichFlags.complete {
			return nil
		}
		if len(args) == 0 {
			return fmt.Errorf("Usage: goenv which <command>")
		}
		if len(args) > 1 {
			return fmt.Errorf("Usage: goenv which <command>")
		}
		return nil
	},
	RunE: runWhich,
}

var whichFlags struct {
	complete bool
}

var whenceCmd = &cobra.Command{
	Use:   "whence <command>",
	Short: "List all Go versions that contain the given executable",
	Long:  "Display which installed Go versions have the specified command available",
	Args: func(cmd *cobra.Command, args []string) error {
		// Handle completion mode
		if whenceFlags.complete {
			return nil
		}
		if len(args) == 0 {
			return fmt.Errorf("Usage: goenv whence [--path] <command>")
		}
		if len(args) > 1 {
			return fmt.Errorf("Usage: goenv whence [--path] <command>")
		}
		return nil
	},
	RunE: runWhence,
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
	cmdpkg.RootCmd.AddCommand(rehashCmd)
	cmdpkg.RootCmd.AddCommand(shimsCmd)
	cmdpkg.RootCmd.AddCommand(whichCmd)
	cmdpkg.RootCmd.AddCommand(whenceCmd)

	shimsCmd.Flags().BoolVar(&shimsFlags.short, "short", false, "Show only shim names without full paths")
	shimsCmd.Flags().BoolVar(&shimsFlags.complete, "complete", false, "Show completion options")
	_ = shimsCmd.Flags().MarkHidden("complete")
	whichCmd.Flags().BoolVar(&whichFlags.complete, "complete", false, "Show completion options")
	_ = whichCmd.Flags().MarkHidden("complete")

	// Apply custom help text
	helptext.SetCommandHelp(rehashCmd)
	helptext.SetCommandHelp(shimsCmd)
	helptext.SetCommandHelp(whichCmd)
	helptext.SetCommandHelp(whenceCmd)
	whenceCmd.Flags().BoolVarP(&whenceFlags.path, "path", "p", false, "Show full paths to executables")
	whenceCmd.Flags().BoolVar(&whenceFlags.complete, "complete", false, "Show completion options")
	_ = whenceCmd.Flags().MarkHidden("complete")
}

func RunRehash(cmd *cobra.Command, args []string) error {
	// Validate: rehash command takes no arguments
	if len(args) > 0 {
		return fmt.Errorf("Usage: goenv rehash")
	}

	cfg := config.Load()
	shimMgr := shims.NewShimManager(cfg)

	if cfg.Debug {
		fmt.Fprintln(cmd.OutOrStdout(), "Debug: Rehashing goenv shims...")
	}

	// Execute pre-rehash hooks
	cmdhooks.ExecuteHooks(hooks.PreRehash, nil)

	fmt.Fprintln(cmd.OutOrStdout(), "Rehashing...")
	if err := shimMgr.Rehash(); err != nil {
		return fmt.Errorf("failed to rehash shims: %w", err)
	}

	shimList, err := shimMgr.ListShims()
	if err != nil {
		return fmt.Errorf("failed to list shims: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Rehashed %d shims\n", len(shimList))

	// Execute post-rehash hooks
	cmdhooks.ExecuteHooks(hooks.PostRehash, nil)

	return nil
}

func runShims(cmd *cobra.Command, args []string) error {
	// Handle completion mode
	if shimsFlags.complete {
		fmt.Fprintln(cmd.OutOrStdout(), "--short")
		return nil
	}

	// Validate: shims command takes no positional arguments (only --short flag)
	if len(args) > 0 {
		return fmt.Errorf("Usage: goenv shims [--short]")
	}

	cfg := config.Load()
	shimMgr := shims.NewShimManager(cfg)

	shimList, err := shimMgr.ListShims()
	if err != nil {
		return fmt.Errorf("failed to list shims: %w", err)
	}

	for _, shim := range shimList {
		if shimsFlags.short {
			fmt.Fprintln(cmd.OutOrStdout(), shim)
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), filepath.Join(cfg.ShimsDir(), shim))
		}
	}

	return nil
}

func runWhich(cmd *cobra.Command, args []string) error {
	// Handle completion mode
	if whichFlags.complete {
		return nil
	}

	if len(args) == 0 {
		return fmt.Errorf("Usage: goenv which <command>")
	}

	commandName := args[0]
	cfg := config.Load()

	// Try using shim manager first (if available)
	shimMgr := shims.NewShimManager(cfg)
	binaryPath, err := shimMgr.WhichBinary(commandName)
	if err == nil {
		fmt.Fprintln(cmd.OutOrStdout(), binaryPath)
		return nil
	}

	// Fallback: implement manual search for testing compatibility
	return runWhichManual(cmd, commandName, cfg)
}

func runWhence(cmd *cobra.Command, args []string) error {
	// Handle completion mode
	if whenceFlags.complete {
		fmt.Fprintln(cmd.OutOrStdout(), "--path")
		return nil
	}

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
				// Build the path without extension first
				versionPath := filepath.Join(cfg.VersionsDir(), version, "bin", commandName)
				// On Windows, try to find the actual file with extension
				if runtime.GOOS == "windows" {
					if foundPath, err := findExecutable(versionPath); err == nil {
						// Strip the extension for display (show logical command name)
						displayPath := strings.TrimSuffix(strings.TrimSuffix(foundPath, ".exe"), ".bat")
						fmt.Fprintln(cmd.OutOrStdout(), displayPath)
					} else {
						fmt.Fprintln(cmd.OutOrStdout(), versionPath)
					}
				} else {
					fmt.Fprintln(cmd.OutOrStdout(), versionPath)
				}
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), version)
			}
		}
		return nil
	}

	// Fallback: manual search
	return runWhenceManual(cmd, commandName, cfg)
}

// findExecutable looks for an executable file, handling Windows extensions
func findExecutable(basePath string) (string, error) {
	// On Windows, try common executable extensions
	if runtime.GOOS == "windows" {
		extensions := []string{".bat", ".cmd", ".exe"}
		for _, ext := range extensions {
			path := basePath + ext
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				return path, nil
			}
		}
		// Also try without extension
		if info, err := os.Stat(basePath); err == nil && !info.IsDir() {
			return basePath, nil
		}
		return "", fmt.Errorf("executable not found")
	}

	// On Unix, check if file exists and is executable
	info, err := os.Stat(basePath)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("is a directory")
	}
	// Check executable bit
	if info.Mode()&0111 == 0 {
		return "", fmt.Errorf("not executable")
	}
	return basePath, nil
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
				fmt.Fprintln(cmd.OutOrStdout(), commandPath)
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
		if foundPath, err := findExecutable(commandPath); err == nil {
			fmt.Fprintln(cmd.OutOrStdout(), foundPath)
			return nil
		}
	}

	// If there are missing versions, return error immediately
	if len(missingVersions) > 0 {
		// Use enhanced error message for the first missing version
		return errors.VersionNotInstalledDetailed(missingVersions[0], source, mgr)
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
		if _, err := findExecutable(commandPath); err == nil {
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

	// Normalize shims directory to absolute path for accurate comparison
	shimsAbs, err := filepath.Abs(shimsDir)
	if err != nil {
		// If we can't get absolute path, fall back to original path
		shimsAbs = shimsDir
	}
	// Clean the path to remove . and .. components
	shimsAbs = filepath.Clean(shimsAbs)

	pathDirs := strings.Split(pathEnv, string(os.PathListSeparator))

	for _, dir := range pathDirs {
		if dir == "" {
			continue
		}

		// Normalize the PATH directory to absolute path
		dirAbs, err := filepath.Abs(dir)
		if err != nil {
			// If we can't resolve it, use as-is
			dirAbs = dir
		}
		dirAbs = filepath.Clean(dirAbs)

		// Skip goenv shims directory using proper path comparison
		// Check exact match or if dir is a subdirectory of shims
		if dirAbs == shimsAbs || strings.HasPrefix(dirAbs+string(filepath.Separator), shimsAbs+string(filepath.Separator)) {
			continue
		}

		commandPath := filepath.Join(dir, commandName)
		if foundPath, err := findExecutable(commandPath); err == nil {
			return foundPath, nil
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
		if foundPath, err := findExecutable(commandPath); err == nil {
			foundAny = true
			if whenceFlags.path {
				// On Windows, strip extension for display (show logical command name)
				displayPath := foundPath
				if runtime.GOOS == "windows" {
					displayPath = strings.TrimSuffix(strings.TrimSuffix(foundPath, ".exe"), ".bat")
				}
				fmt.Fprintln(cmd.OutOrStdout(), displayPath)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), version)
			}
		}
	}

	if !foundAny {
		// Return error but don't print anything (BATS expects empty output + error)
		return fmt.Errorf("no versions found with %s", commandName)
	}

	return nil
}
