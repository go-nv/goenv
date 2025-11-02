package shims

import (
	"fmt"
	"os"
	"path/filepath"

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

	cfg, mgr := cmdutil.SetupContext()

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
	currentVersion, _, _ := mgr.GetCurrentVersion()

	// If version is "system", don't export GOPATH/GOROOT
	if currentVersion == manager.SystemVersion || currentVersion == "" {
		return nil
	}

	// Check environment variables for disabling GOROOT/GOPATH
	disableGoroot := utils.GoenvEnvVarDisableGoroot.UnsafeValue() == "1"
	disableGopath := utils.GoenvEnvVarDisableGopath.UnsafeValue() == "1"
	gopathPrefix := utils.GoenvEnvVarGopathPrefix.UnsafeValue()
	appendGopath := utils.GoenvEnvVarAppendGopath.UnsafeValue() == "1"
	prependGopath := utils.GoenvEnvVarPrependGopath.UnsafeValue() == "1"
	existingGopath := os.Getenv(utils.EnvVarGopath)

	// Build GOPATH value
	var gopathValue string
	if gopathPrefix == "" {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			home = os.Getenv(utils.EnvVarHome) // Fallback
		}
		gopathValue = filepath.Join(home, "go", currentVersion)
	} else {
		gopathValue = filepath.Join(gopathPrefix, currentVersion)
	}

	// Handle GOPATH appending/prepending
	if existingGopath != "" && appendGopath {
		gopathValue = gopathValue + string(os.PathListSeparator) + existingGopath
	} else if existingGopath != "" && prependGopath {
		gopathValue = existingGopath + string(os.PathListSeparator) + gopathValue
	}

	// Build GOROOT value (version install path)
	gorootValue := filepath.Join(cfg.Root, "versions", currentVersion)

	// Generate shell-specific output
	if shellType == shellutil.ShellTypeFish {
		// Fish shell
		if !disableGoroot {
			fmt.Fprintf(cmd.OutOrStdout(), "set -gx GOROOT \"%s\"\n", gorootValue)
		}

		if !disableGopath {
			fmt.Fprintf(cmd.OutOrStdout(), "set -gx GOPATH \"%s\"\n", gopathValue)
		}

		// Fish doesn't support hash -r
	} else if shellType == shellutil.ShellTypePowerShell {
		// PowerShell
		if !disableGoroot {
			fmt.Fprintf(cmd.OutOrStdout(), "$env:GOROOT = \"%s\"\n", gorootValue)
		}

		if !disableGopath {
			fmt.Fprintf(cmd.OutOrStdout(), "$env:GOPATH = \"%s\"\n", gopathValue)
		}

		// PowerShell doesn't need hash -r equivalent
	} else if shellType == shellutil.ShellTypeCmd {
		// CMD
		if !disableGoroot {
			fmt.Fprintf(cmd.OutOrStdout(), "set GOROOT=%s\n", gorootValue)
		}

		if !disableGopath {
			fmt.Fprintf(cmd.OutOrStdout(), "set GOPATH=%s\n", gopathValue)
		}

		// CMD doesn't need hash -r equivalent
	} else {
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
