package cmdutil

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
)

// SetupContext initializes the common context (config + manager) that most commands need.
// This is a convenience function to reduce boilerplate in command implementations.
//
// Example usage:
//
//	func runMyCommand(cmd *cobra.Command, args []string) error {
//	    cfg, mgr := cmdutil.SetupContext()
//	    // ... use cfg and mgr
//	}
func SetupContext() (*config.Config, *manager.Manager) {
	cfg := config.Load()
	mgr := manager.NewManager(cfg)
	return cfg, mgr
}

// OutputJSON encodes data as JSON and writes it to the given writer.
// This provides consistent JSON output formatting across all commands.
//
// Example usage:
//
//	if jsonFlag {
//	    return cmdutil.OutputJSON(cmd.OutOrStdout(), result)
//	}
func OutputJSON(w io.Writer, data interface{}) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// ValidateExactArgs returns an error if the number of arguments doesn't match expected.
// This provides consistent error messages for argument validation.
//
// Example usage:
//
//	if err := cmdutil.ValidateExactArgs(args, 1, "version"); err != nil {
//	    return err
//	}
func ValidateExactArgs(args []string, expected int, argName string) error {
	if len(args) != expected {
		if expected == 1 {
			return fmt.Errorf("expected %d argument (%s), got %d", expected, argName, len(args))
		}
		return fmt.Errorf("expected %d arguments, got %d", expected, len(args))
	}
	return nil
}

// ValidateMinArgs returns an error if there are fewer than the minimum number of arguments.
//
// Example usage:
//
//	if err := cmdutil.ValidateMinArgs(args, 1, "at least one version"); err != nil {
//	    return err
//	}
func ValidateMinArgs(args []string, min int, description string) error {
	if len(args) < min {
		return fmt.Errorf("expected %s, got %d", description, len(args))
	}
	return nil
}

// ValidateMaxArgs returns an error if there are more than the maximum number of arguments.
//
// Example usage:
//
//	if err := cmdutil.ValidateMaxArgs(args, 1, "at most one version"); err != nil {
//	    return err
//	}
func ValidateMaxArgs(args []string, max int, description string) error {
	if len(args) > max {
		return fmt.Errorf("expected %s, got %d", description, len(args))
	}
	return nil
}

// RequireInstalledVersion checks if a version is installed and returns a helpful error if not.
// This is a common pattern across many commands.
//
// Example usage:
//
//	if err := cmdutil.RequireInstalledVersion(mgr, version); err != nil {
//	    return err
//	}
func RequireInstalledVersion(mgr interface{ IsVersionInstalled(string) bool }, version string) error {
	if version == manager.SystemVersion {
		return nil // system is always "installed"
	}

	if !mgr.IsVersionInstalled(version) {
		return fmt.Errorf("version %s is not installed. Run: goenv install %s", version, version)
	}
	return nil
}

// MustGetVersion validates args and extracts the version argument.
// This combines argument validation with version extraction, a very common pattern.
//
// Example usage:
//
//	version, err := cmdutil.MustGetVersion(args)
//	if err != nil {
//	    return err
//	}
func MustGetVersion(args []string) (string, error) {
	if err := ValidateExactArgs(args, 1, "version"); err != nil {
		return "", err
	}
	return args[0], nil
}
