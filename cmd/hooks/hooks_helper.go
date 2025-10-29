package hooks

import (
	"os"
	"time"

	"github.com/go-nv/goenv/internal/hooks"
)

// executeHooks runs hooks for the given hook point with provided variables.
// This is a shared helper used by install, uninstall, exec, and rehash commands.
// Errors are handled silently to prevent hooks from breaking normal operations.
func ExecuteHooks(hookPoint hooks.HookPoint, vars map[string]string) {
	// Load hooks configuration
	hooksConfig, err := hooks.LoadConfig(hooks.DefaultConfigPath())
	if err != nil || !hooksConfig.IsEnabled() {
		// Hooks disabled or config not found - skip silently
		return
	}

	// Early exit if no hooks configured for this point
	if len(hooksConfig.GetHooks(hookPoint.String())) == 0 {
		return
	}

	// Create executor only if we have hooks to execute
	executor := hooks.NewExecutor(hooksConfig)

	// Ensure hook and timestamp are always set
	if vars == nil {
		vars = make(map[string]string)
	}
	vars["hook"] = hookPoint.String()
	vars["timestamp"] = time.Now().Format(time.RFC3339)

	// Add file_arg if available (useful for exec hooks to know which file is being processed)
	if fileArg := os.Getenv("GOENV_FILE_ARG"); fileArg != "" {
		vars["file_arg"] = fileArg
	}

	// Execute hooks (errors are logged but don't fail the command)
	_ = executor.Execute(hookPoint, vars)
}
