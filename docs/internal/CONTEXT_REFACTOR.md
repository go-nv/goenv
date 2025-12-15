# Context-Based Configuration Refactor

## Summary

This refactor introduces a context-based approach for managing configuration, manager, and environment across all commands, improving performance and maintainability.

## What Changed

### 1. **Environment Variable Parsing** (`internal/utils/environment.go`)
- Added `GoenvEnvironment` struct with all GOENV_* env vars
- Integrated `github.com/sethvargo/go-envconfig` for proper parsing
- Uses `PrefixLookuper` to automatically map field names to `GOENV_*` vars
- Added context storage functions: `EnvironmentToContext()` and `EnvironmentFromContext()`

### 2. **Config Context** (`internal/config/context.go`)
- New file with type-safe context key
- `ToContext()` and `FromContext()` functions for storing/retrieving config

### 3. **Manager Context** (`internal/manager/context.go`)
- New file with type-safe context key
- `ToContext()` and `FromContext()` functions for storing/retrieving manager

### 4. **Config Loading** (`internal/config/config.go`)
- Added `LoadFromEnvironment()` function that accepts parsed `GoenvEnvironment`
- Existing `Load()` function kept for backward compatibility

### 5. **Root Command** (`cmd/root.go`)
- Added `PersistentPreRun` to initialize once per command execution
- Parses env vars, creates config/manager, stores in context
- All child commands automatically inherit the context

### 6. **Command Utilities** (`internal/cmdutil/helpers.go`)
- Added `GetConfig()` and `GetManager()` helper functions
- Deprecated `SetupContext()` but kept for backward compatibility
- Created `CONTEXT_PATTERN.go` with usage examples

## Benefits

✅ **Performance**: Config and manager created once, not per command  
✅ **Type Safety**: Proper env var parsing with struct tags  
✅ **Consistency**: All commands use the same instances  
✅ **Testability**: Can inject custom context in tests  
✅ **Clean Code**: No repeated `SetupContext()` calls  

## Migration Guide

### Old Pattern (Still Works):
```go
func runMyCommand(cmd *cobra.Command, args []string) error {
    cfg, mgr := cmdutil.SetupContext() // Creates new instances
    // ...
}
```

### New Pattern (Preferred):
```go
func runMyCommand(cmd *cobra.Command, args []string) error {
    cfg := cmdutil.GetConfig(cmd)   // From context
    mgr := cmdutil.GetManager(cmd)  // From context
    // Optional: env := utils.EnvironmentFromContext(cmd.Context())
    // ...
}
```

## Next Steps

Gradually migrate existing commands to use the new pattern:
1. Replace `cmdutil.SetupContext()` with `cmdutil.GetConfig()` and `cmdutil.GetManager()`
2. Remove the old `SetupContext()` function once all commands are migrated
3. Add tests that verify context propagation

## Files Added/Modified

**Added:**
- `internal/config/context.go`
- `internal/manager/context.go`
- `internal/cmdutil/CONTEXT_PATTERN.go`

**Modified:**
- `internal/utils/environment.go` - Added struct and context functions
- `internal/config/config.go` - Added `LoadFromEnvironment()`
- `cmd/root.go` - Added `PersistentPreRun`
- `internal/cmdutil/helpers.go` - Added new helper functions
- `go.mod` - Added `github.com/sethvargo/go-envconfig`
