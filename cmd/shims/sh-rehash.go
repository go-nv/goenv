package shims

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	cmdpkg "github.com/go-nv/goenv/cmd"
	"github.com/go-nv/goenv/cmd/shell"

	"github.com/go-nv/goenv/internal/cmdutil"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/shellutil"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var shRehashCmd = &cobra.Command{
	Use:                "sh-rehash [--only-manage-paths]",
	Short:              "Calls goenv-rehash to rehash shims, manages GOPATH/GOROOT and rehashes shell executable",
	Hidden:             true,
	DisableFlagParsing: true, // Treat --only-manage-paths as argument, not flag
	RunE:               runShRehash,
}

func init() {
	cmdpkg.RootCmd.AddCommand(shRehashCmd)
}

func runShRehash(cmd *cobra.Command, args []string) error {
	// Handle completion request
	if len(args) == 1 && args[0] == "--complete" {
		// No completions for sh-rehash
		return nil
	}

	ctx := cmdutil.GetContexts(cmd)
	cfg := ctx.Config
	mgr := ctx.Manager
	env := ctx.Environment
	
	// Fallback: Load environment if not already in context (e.g., in tests)
	if env == nil {
		env, _ = utils.LoadEnvironment(cmd.Context())
	}

	// Determine shell type
	shellType := shell.ResolveShell("", true)

	// Call rehash unless --only-manage-paths is specified
	onlyManagePaths := false
	if len(args) > 0 && args[0] == "--only-manage-paths" {
		onlyManagePaths = true
	}

	if !onlyManagePaths {
		// Run rehash silently (we don't want its output mixed with shell commands)
		// Create a temporary command to capture rehash output
		rehashCmd := &cobra.Command{}
		rehashCmd.SetOut(cmd.OutOrStderr()) // Send rehash output to stderr or discard
		rehashCmd.SetErr(cmd.ErrOrStderr())
		if err := RunRehash(rehashCmd, []string{}); err != nil {
			return err
		}
	}

	// Get current version
	currentVersion, _, _, err := mgr.GetCurrentVersionResolved()
	if err != nil {
		return err
	}

	// If version is "system", don't export GOPATH/GOROOT
	if currentVersion == manager.SystemVersion || currentVersion == "" {
		return nil
	}

	// Check environment variables for disabling GOROOT/GOPATH
	disableGoroot := env.HasDisableGoroot()
	disableGopath := env.HasDisableGopath()

	// Build GOPATH value: $HOME/go/{version}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = os.Getenv(utils.EnvVarHome) // Fallback
	}
	gopathValue := filepath.Join(home, "go", currentVersion)

	// Preserve existing GOPATH by prepending version-specific path.
	// This allows users to keep source code in existing locations while
	// giving priority to version-specific installed tools/packages.
	// See: https://github.com/go-nv/goenv/issues/147
	existingGopath := os.Getenv(utils.EnvVarGopath)
	if existingGopath != "" {
		// Filter out any goenv-managed paths to prevent duplication on re-initialization
		goPathPattern := filepath.Join(home, "go")
		var filteredPaths []string
		for _, path := range filepath.SplitList(existingGopath) {
			// Skip paths that look like $HOME/go/{version}
			// Keep paths that are exactly $HOME/go or any other custom paths
			if !strings.HasPrefix(path, goPathPattern+string(filepath.Separator)) || path == goPathPattern {
				filteredPaths = append(filteredPaths, path)
			}
		}
		if len(filteredPaths) > 0 {
			gopathValue = gopathValue + string(os.PathListSeparator) + strings.Join(filteredPaths, string(os.PathListSeparator))
		}
	}

	// Build GOROOT value (version install path)
	gorootValue := filepath.Join(cfg.Root, "versions", currentVersion)

	// Generate shell-specific output
	switch shellType {
	case shellutil.ShellTypeFish:
		// Fish shell
		if !disableGoroot {
			fmt.Fprintf(cmd.OutOrStdout(), "set -gx GOROOT \"%s\"\n", gorootValue)
		}

		if !disableGopath {
			fmt.Fprintf(cmd.OutOrStdout(), "set -gx GOPATH \"%s\"\n", gopathValue)
		}

		// Fish doesn't support hash -r

	case shellutil.ShellTypePowerShell:
		// PowerShell
		if !disableGoroot {
			fmt.Fprintf(cmd.OutOrStdout(), "$env:GOROOT = \"%s\"\n", gorootValue)
		}

		if !disableGopath {
			fmt.Fprintf(cmd.OutOrStdout(), "$env:GOPATH = \"%s\"\n", gopathValue)
		}

		// PowerShell doesn't need hash -r equivalent

	case shellutil.ShellTypeCmd:
		// CMD
		if !disableGoroot {
			fmt.Fprintf(cmd.OutOrStdout(), "set GOROOT=%s\n", gorootValue)
		}

		if !disableGopath {
			fmt.Fprintf(cmd.OutOrStdout(), "set GOPATH=%s\n", gopathValue)
		}

		// CMD doesn't need hash -r equivalent

	default:
		// Bash, zsh, ksh, etc.
		if !disableGoroot {
			fmt.Fprintf(cmd.OutOrStdout(), "export GOROOT=\"%s\"\n", gorootValue)
		}

		if !disableGopath {
			fmt.Fprintf(cmd.OutOrStdout(), "export GOPATH=\"%s\"\n", gopathValue)
		}

		// Rehash binaries (Unix shells only)
		fmt.Fprintln(cmd.OutOrStdout(), "hash -r 2>/dev/null || true")
	}

	return nil
}
