# Hook Points Reference

This document provides a reference for all available hook points in goenv.

## Type-Safe Hook Points

All hook points are defined as constants in `hookpoints.go` to provide type safety and prevent typos.

### Usage

```go
import "github.com/go-nv/goenv/internal/hooks"

// Use the enum constants
executeHook(hooks.PreInstall, goVersion)
executeHook(hooks.PostInstall, goVersion)

// The HookPoint type automatically converts to string when needed
hookName := hooks.PreInstall.String() // "pre_install"
```

## Available Hook Points

### Installation Lifecycle

| Constant | String Value | Description | Available Variables |
|----------|-------------|-------------|---------------------|
| `hooks.PreInstall` | `"pre_install"` | Before installing a Go version | `{version}`, `{hook}`, `{timestamp}` |
| `hooks.PostInstall` | `"post_install"` | After installing a Go version | `{version}`, `{hook}`, `{timestamp}` |
| `hooks.PreUninstall` | `"pre_uninstall"` | Before uninstalling a Go version | `{version}`, `{hook}`, `{timestamp}` |
| `hooks.PostUninstall` | `"post_uninstall"` | After uninstalling a Go version | `{version}`, `{hook}`, `{timestamp}` |

### Execution Lifecycle

| Constant | String Value | Description | Available Variables |
|----------|-------------|-------------|---------------------|
| `hooks.PreExec` | `"pre_exec"` | Before executing a Go command | `{version}`, `{command}`, `{hook}`, `{timestamp}` |
| `hooks.PostExec` | `"post_exec"` | After executing a Go command | `{version}`, `{command}`, `{hook}`, `{timestamp}` |

### Maintenance Lifecycle

| Constant | String Value | Description | Available Variables |
|----------|-------------|-------------|---------------------|
| `hooks.PreRehash` | `"pre_rehash"` | Before regenerating shims | `{hook}`, `{timestamp}` |
| `hooks.PostRehash` | `"post_rehash"` | After regenerating shims | `{hook}`, `{timestamp}` |

## Helper Functions

### AllHookPoints()

Returns a slice of all valid hook points:

```go
allHooks := hooks.AllHookPoints()
// [PreInstall, PostInstall, PreUninstall, PostUninstall, PreExec, PostExec, PreRehash, PostRehash]
```

### IsValidHookPoint(s string)

Validates if a string represents a valid hook point:

```go
if hooks.IsValidHookPoint("pre_install") {
    // Valid hook point
}
```

## Integration Pattern

When integrating hooks into commands, use this pattern:

```go
func executeHook(hookPoint hooks.HookPoint, context string) {
    // Load config
    config, err := hooks.LoadConfig(hooks.DefaultConfigPath())
    if err != nil || !config.IsEnabled() {
        return // Skip silently
    }

    // Create executor
    executor := hooks.NewExecutor(config)

    // Prepare variables
    vars := map[string]string{
        "hook":      hookPoint.String(),
        "version":   context,
        "timestamp": time.Now().Format(time.RFC3339),
    }

    // Execute (errors don't fail commands)
    _ = executor.Execute(hookPoint, vars)
}
```

## Configuration Example

In `~/.goenv/hooks.yaml`, use the string values:

```yaml
enabled: true
acknowledged_risks: true

hooks:
  pre_install:
    - action: log_to_file
      params:
        path: /tmp/goenv.log
        message: "Installing version {version}"
  
  post_install:
    - action: notify_desktop
      params:
        title: "goenv"
        message: "Installed Go {version}"
```

## Benefits

1. **Type Safety**: Compiler catches typos at build time
2. **IDE Support**: Auto-completion and documentation
3. **Refactoring**: Easy to rename hook points across codebase
4. **Validation**: `IsValidHookPoint()` for runtime validation
5. **Documentation**: Single source of truth for all hook points
