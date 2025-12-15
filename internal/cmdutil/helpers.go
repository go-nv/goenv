package cmdutil

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

// AllContextKeys is a slice containing all available context keys.
// Use this with GetContexts() when you need all context values.
var AllContextKeys = []any{
	config.ConfigContextKey,
	manager.ManagerContextKey,
	utils.EnvironmentContextKey,
}

// CmdContext holds all available context values that can be retrieved from command context.
// Use GetContexts to populate this struct with the values you need.
type CmdContext struct {
	Config      *config.Config
	Manager     *manager.Manager
	Environment *utils.GoenvEnvironment
}

// GetContexts retrieves multiple context values at once from the command's context.
// Pass the context keys you want to retrieve, and it returns a Contexts struct with the values.
// If no keys are provided, all available context values are retrieved.
//
// Available keys:
//   - config.ConfigContextKey
//   - manager.ManagerContextKey
//   - utils.EnvironmentContextKey
//
// Example usage:
//
//	func runMyCommand(cmd *cobra.Command, args []string) error {
//	    // Get all contexts
//	    ctx := cmdutil.GetContexts(cmd)
//
//	    // Or get specific contexts
//	    ctx := cmdutil.GetContexts(cmd,
//	        config.ConfigContextKey,
//	        manager.ManagerContextKey,
//	    )
//	    // Use ctx.Config and ctx.Manager
//	}
func GetContexts(cmd *cobra.Command, keys ...any) *CmdContext {
	// If no keys specified, default to all keys
	if len(keys) == 0 {
		keys = AllContextKeys
	}

	ctx := cmd.Context()
	// Handle nil context (common in tests)
	if ctx == nil {
		ctx = context.Background()
	}

	result := &CmdContext{}

	for _, key := range keys {
		switch key {
		case config.ConfigContextKey:
			result.Config = config.FromContext(ctx)
			// Fallback for tests that don't set up context
			if result.Config == nil {
				result.Config = config.Load()
			}
		case manager.ManagerContextKey:
			result.Manager = manager.FromContext(ctx)
			// Fallback for tests that don't set up context
			if result.Manager == nil {
				if result.Config == nil {
					result.Config = config.Load()
				}
				result.Manager = manager.NewManager(result.Config)
			}
		case utils.EnvironmentContextKey:
			result.Environment = utils.EnvironmentFromContext(ctx)
			// No fallback for environment - it's optional
		}
	}

	return result
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
