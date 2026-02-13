# cmdutil - Command Utilities Package

This package provides common helper functions for goenv commands to reduce boilerplate and ensure consistency across the codebase.

## Purpose

As goenv grew, patterns emerged across commands:
- Most commands need to load config and create a manager
- Many commands output JSON with consistent formatting
- Argument validation follows similar patterns

The `cmdutil` package consolidates these patterns into reusable functions.

## Functions

### GetContexts(cmd, keys...)

**NEW**: Retrieves multiple context values at once from the command's context.
This is the recommended way to access config, manager, and environment in commands.

**Usage:**
```go
func runMyCommand(cmd *cobra.Command, args []string) error {
    ctx := cmdutil.GetContexts(cmd,
        config.ConfigContextKey,
        manager.ManagerContextKey,
    )
    
    cfg := ctx.Config
    mgr := ctx.Manager
    // ...
}
```

**Available Keys:**
- `config.ConfigContextKey` - Retrieves `*config.Config`
- `manager.ManagerContextKey` - Retrieves `*manager.Manager`
- `utils.EnvironmentContextKey` - Retrieves `*utils.GoenvEnvironment`

**Benefits:**
- Only retrieves what you need
- Single call to get all contexts
- Type-safe access
- Reuses instances from `PersistentPreRun` (efficient)

### SetupContext() [DEPRECATED]

**OLD**: Initializes the common context (config + manager) that most commands need.

**⚠️ DEPRECATED**: Use `GetContexts()` instead for better performance and consistency.

**Before:**
```go
func runMyCommand(cmd *cobra.Command, args []string) error {
    cfg := config.Load()
    mgr := manager.NewManager(cfg)
    // ...
}
```

**Old Way:**
```go
func runMyCommand(cmd *cobra.Command, args []string) error {
    cfg, mgr := cmdutil.SetupContext()
    // ...
}
```

**New Way:**
```go
func runMyCommand(cmd *cobra.Command, args []string) error {
    ctx := cmdutil.GetContexts(cmd,
        config.ConfigContextKey,
        manager.ManagerContextKey,
    )
    cfg := ctx.Config
    mgr := ctx.Manager
    // ...
}
```

### OutputJSON(writer, data)

Encodes data as JSON with consistent formatting (2-space indentation).

**Before:**
```go
encoder := json.NewEncoder(cmd.OutOrStdout())
encoder.SetIndent("", "  ")
return encoder.Encode(result)
```

**After:**
```go
return cmdutil.OutputJSON(cmd.OutOrStdout(), result)
```

### ValidateExactArgs(args, expected, argName)

Validates argument count with consistent error messages.

```go
if err := cmdutil.ValidateExactArgs(args, 1, "version"); err != nil {
    return err
}
```

### ValidateMinArgs(args, min, description)

Ensures minimum number of arguments.

```go
if err := cmdutil.ValidateMinArgs(args, 1, "at least one version"); err != nil {
    return err
}
```

### ValidateMaxArgs(args, max, description)

Ensures maximum number of arguments.

```go
if err := cmdutil.ValidateMaxArgs(args, 2, "at most two versions"); err != nil {
    return err
}
```

## Benefits

1. **Reduced Boilerplate**: Commands are more concise and focused on their core logic
2. **Consistency**: All commands use the same patterns for common operations
3. **Maintainability**: Changes to common patterns only need to be made once
4. **Testability**: Helpers are well-tested and reliable

## Migration Guide

### For New Commands

Always use `cmdutil` helpers when writing new commands:

```go
func runNewCommand(cmd *cobra.Command, args []string) error {
    // Setup
    cfg, mgr := cmdutil.SetupContext()

    // JSON output
    if jsonFlag {
        return cmdutil.OutputJSON(cmd.OutOrStdout(), result)
    }

    // Continue with command logic...
}
```

### For Existing Commands

Migration can happen gradually:
1. New commands should always use cmdutil
2. When modifying existing commands, consider refactoring to use cmdutil
3. No urgency - both patterns work fine

## Examples

### Example 1: Simple Command with JSON Output

```go
func runExampleCmd(cmd *cobra.Command, args []string) error {
    cfg, mgr := cmdutil.SetupContext()

    result := map[string]string{
        "root": cfg.Root,
        "version": "1.23.0",
    }

    if jsonFlag {
        return cmdutil.OutputJSON(cmd.OutOrStdout(), result)
    }

    fmt.Fprintf(cmd.OutOrStdout(), "Root: %s\n", cfg.Root)
    return nil
}
```

### Example 2: Command with Argument Validation

```go
func runValidateCmd(cmd *cobra.Command, args []string) error {
    // Validate exactly one argument
    if err := cmdutil.ValidateExactArgs(args, 1, "version"); err != nil {
        return err
    }

    version := args[0]
    cfg, mgr := cmdutil.SetupContext()

    // Continue with logic...
}
```

## Testing

The package has 100% test coverage:

```bash
go test ./internal/cmdutil -v
```

All helper functions are thoroughly tested with multiple scenarios.

## See Also

- [internal/platform](../platform/README.md) - Platform detection utilities
- [internal/utils](../utils/README.md) - General utilities
