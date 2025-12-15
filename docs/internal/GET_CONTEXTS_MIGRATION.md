# GetContexts Migration Guide

## Overview

We've replaced `SetupContext()` with a new `GetContexts()` function that retrieves values from the command context instead of creating new instances every time.

## What Changed

### Old Signature (Deprecated)
```go
func SetupContext() (*config.Config, *manager.Manager)
```

### New Signature
```go
func GetContexts(cmd *cobra.Command, keys ...any) *Contexts

type Contexts struct {
    Config      *config.Config
    Manager     *manager.Manager
    Environment *utils.GoenvEnvironment
}
```

## Migration Examples

### Before
```go
func runMyCommand(cmd *cobra.Command, args []string) error {
    cfg, mgr := cmdutil.SetupContext()
    
    // Use cfg and mgr
    version := mgr.GetVersion()
    root := cfg.Root
    // ...
}
```

### After
```go
func runMyCommand(cmd *cobra.Command, args []string) error {
    ctx := cmdutil.GetContexts(cmd,
        config.ConfigContextKey,
        manager.ManagerContextKey,
    )
    
    // Use ctx.Config and ctx.Manager
    version := ctx.Manager.GetVersion()
    root := ctx.Config.Root
    // ...
}
```

### With Environment
```go
func runMyCommand(cmd *cobra.Command, args []string) error {
    ctx := cmdutil.GetContexts(cmd,
        config.ConfigContextKey,
        manager.ManagerContextKey,
        utils.EnvironmentContextKey,
    )
    
    // Check offline mode
    if ctx.Environment.Offline {
        return fmt.Errorf("cannot run in offline mode")
    }
    
    // Use ctx.Config and ctx.Manager
    // ...
}
```

### Only Need One
```go
func runMyCommand(cmd *cobra.Command, args []string) error {
    ctx := cmdutil.GetContexts(cmd, manager.ManagerContextKey)
    
    // Only manager is populated
    version := ctx.Manager.GetVersion()
    // ...
}
```

## Benefits

1. **Performance**: Reuses instances from `PersistentPreRun` instead of creating new ones
2. **Explicit Dependencies**: Clear which context values each command needs
3. **Type Safety**: Structured return type instead of multiple return values
4. **Optional Values**: Only request what you need
5. **Test Compatible**: Falls back to creating instances if context not set (for existing tests)

## Backward Compatibility

- `SetupContext()` still works but is deprecated
- `GetContexts()` automatically falls back to creating instances if context is not available
- Existing tests continue to work without modification
- All `FromContext()` functions handle nil context gracefully

## Available Context Keys

```go
import (
    "github.com/go-nv/goenv/internal/config"
    "github.com/go-nv/goenv/internal/manager"
    "github.com/go-nv/goenv/internal/utils"
)

// Use these keys with GetContexts
config.ConfigContextKey      // *config.Config
manager.ManagerContextKey    // *manager.Manager
utils.EnvironmentContextKey  // *utils.GoenvEnvironment
```

## Imports Needed

When migrating a command, add:
```go
import (
    "github.com/go-nv/goenv/internal/config"  // for ConfigContextKey
    // Only if using manager
    "github.com/go-nv/goenv/internal/manager" // for ManagerContextKey
    // Only if using environment
    "github.com/go-nv/goenv/internal/utils"   // for EnvironmentContextKey
)
```

## Example: Complete Migration

**File: `cmd/mypackage/mycommand.go`**

**Before:**
```go
package mypackage

import (
    "github.com/go-nv/goenv/internal/cmdutil"
    "github.com/spf13/cobra"
)

func runMyCommand(cmd *cobra.Command, args []string) error {
    cfg, mgr := cmdutil.SetupContext()
    
    versions, err := mgr.ListInstalledVersions()
    if err != nil {
        return err
    }
    
    for _, v := range versions {
        fmt.Fprintln(cmd.OutOrStdout(), v)
    }
    
    return nil
}
```

**After:**
```go
package mypackage

import (
    "github.com/go-nv/goenv/internal/cmdutil"
    "github.com/go-nv/goenv/internal/config"
    "github.com/go-nv/goenv/internal/manager"
    "github.com/spf13/cobra"
)

func runMyCommand(cmd *cobra.Command, args []string) error {
    ctx := cmdutil.GetContexts(cmd,
        config.ConfigContextKey,
        manager.ManagerContextKey,
    )
    
    versions, err := ctx.Manager.ListInstalledVersions()
    if err != nil {
        return err
    }
    
    for _, v := range versions {
        fmt.Fprintln(cmd.OutOrStdout(), v)
    }
    
    return nil
}
```

## Migration Checklist

- [ ] Replace `cfg, mgr := cmdutil.SetupContext()` with `ctx := cmdutil.GetContexts(cmd, ...)`
- [ ] Add necessary imports (`config`, `manager`, etc.)
- [ ] Change `cfg` to `ctx.Config`
- [ ] Change `mgr` to `ctx.Manager`
- [ ] Test the command still works
- [ ] Verify tests still pass

## Files Migrated

✅ `cmd/diagnostics/status.go` - Example migration complete

## Next Steps

Gradually migrate other commands:
1. Start with simple commands that only use config/manager
2. Move to commands that need environment
3. Update as you touch files for other changes
4. No rush - backward compatibility maintained
