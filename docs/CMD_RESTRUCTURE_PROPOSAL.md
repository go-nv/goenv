# CMD Directory Restructure Proposal

## Current State
86 files in flat `/cmd` directory - hard to navigate

## Proposed Structure

```
cmd/
├── root.go                    # Cobra root command (stays at top)
├── util.go                    # Shared utilities (stays at top)
├── output_golden_test.go      # Shared test helpers (stays at top)
│
├── core/                      # Modern primary commands
│   ├── use.go                 # Primary version switcher
│   ├── current.go             # Show current version
│   ├── list.go                # List versions
│   ├── install.go
│   ├── install_test.go
│   ├── install_no_rehash_test.go
│   ├── uninstall.go
│   └── uninstall_test.go
│
├── legacy/                    # Deprecated commands (backwards compat)
│   ├── global.go              # Deprecated: use 'goenv use --global'
│   ├── global_test.go
│   ├── local.go               # Deprecated: use 'goenv use'
│   ├── local_test.go
│   ├── version.go             # Deprecated: use 'goenv current'
│   ├── version_test.go
│   ├── versions.go            # Deprecated: use 'goenv list'
│   ├── versions_test.go
│   ├── installed.go           # Specialized: use 'goenv list' instead
│   └── installed_test.go
│
├── version/                   # Version detection & file management
│   ├── version-name.go
│   ├── version-name_test.go
│   ├── version-origin.go
│   ├── version-origin_test.go
│   ├── version-file.go
│   ├── version-file_test.go
│   ├── version-file-read.go
│   ├── version-file-read_test.go
│   ├── version-file-write.go
│   ├── version-file-write_test.go
│   └── toolchain_edge_cases_test.go
│
├── tools/                     # Tool management commands
│   ├── tools.go               # Main tools command
│   ├── default_tools.go
│   ├── default_tools_test.go
│   ├── sync_tools.go
│   ├── sync_tools_test.go
│   ├── update_tools.go
│   └── update_tools_test.go
│
├── shims/                     # Shim system & execution
│   ├── exec.go
│   ├── exec_test.go
│   ├── exec_integration_test.go
│   ├── shims.go
│   ├── shims_test.go
│   ├── rehash.go              # (if exists)
│   ├── rehash_test.go
│   ├── sh-rehash.go
│   ├── sh-rehash_test.go
│   ├── sh-shell_test.go
│   ├── whence.go              # (if exists)
│   ├── whence_test.go
│   ├── which.go               # (if exists)
│   └── which_test.go
│
├── diagnostics/               # Health checks & maintenance
│   ├── doctor.go
│   ├── doctor_test.go
│   ├── doctor_exitcodes_test.go
│   ├── cache.go
│   ├── cache_test.go
│   ├── refresh.go
│   └── refresh_test.go
│
├── compliance/                # Security, audit, SBOM
│   ├── sbom.go
│   ├── sbom_test.go
│   ├── inventory.go
│   └── inventory_test.go
│
├── integrations/              # IDE & CI/CD integrations
│   ├── vscode.go
│   ├── vscode_test.go
│   ├── ci-setup.go
│   └── ci-setup_test.go
│
├── hooks/                     # Hook system
│   ├── hooks.go
│   └── hooks_helper.go
│
├── aliases/                   # Version aliases
│   ├── alias.go
│   ├── alias_test.go
│   └── unalias.go
│
├── shell/                     # Shell integration
│   ├── init.go
│   ├── init_test.go
│   ├── init_windows_test.go
│   ├── completion.go          # (if singular)
│   ├── completions.go
│   └── completions_test.go
│
└── meta/                      # Meta commands & system
    ├── commands.go
    ├── commands_test.go
    ├── help.go
    ├── help_test.go
    ├── update.go
    ├── update_test.go
    ├── goenv-root.go
    ├── goenv-root_test.go
    ├── prefix.go              # (if exists)
    └── prefix_test.go
```

## Key Benefits

### 1. Logical Grouping
- **core/** - Commands 95% of users interact with daily
- **legacy/** - Clearly marked deprecated commands
- **version/** - Internal version detection logic
- **tools/** - All tool management in one place
- **shims/** - Entire shim system together
- **diagnostics/** - Health & maintenance
- **compliance/** - Audit, security, SBOM
- **integrations/** - VS Code, CI/CD
- **hooks/** - Hook system
- **aliases/** - Version aliases
- **shell/** - Shell integration
- **meta/** - System/meta commands

### 2. Easier Navigation
- Want to deprecate a command? Look in `legacy/`
- Debugging version detection? Check `version/`
- Tool sync issues? All in `tools/`
- Doctor problems? All diagnostics in one folder

### 3. Clear Boundaries
- **legacy/** signals "don't build new features here"
- **core/** signals "primary user interface"
- **version/** signals "internal supporting logic"

### 4. Test Co-location
Tests stay with their commands, just in subdirectories

### 5. No Namespace Changes
All files stay in `package cmd` - no import changes needed

## Migration Strategy

### Phase 1: Create directories
```bash
mkdir -p cmd/{core,legacy,version,tools,shims,diagnostics,compliance,integrations,hooks,aliases,shell,meta}
```

### Phase 2: Move files (git mv preserves history)
```bash
# Example for core commands
git mv cmd/use.go cmd/core/
git mv cmd/current.go cmd/core/
git mv cmd/list.go cmd/core/
git mv cmd/install.go cmd/core/
git mv cmd/install_test.go cmd/core/
# ... etc
```

### Phase 3: Update imports (if any internal cross-references)
Most imports should be `"github.com/go-nv/goenv/cmd"` and still work

### Phase 4: Update build scripts/Makefile if needed
Ensure `go build ./cmd/...` still works (it should automatically)

## Priority Order

1. **High Priority: legacy/** - Immediate benefit for deprecation efforts
2. **High Priority: core/** - Clarifies primary interface
3. **Medium Priority: version/** - Helps internal maintainers
4. **Medium Priority: tools/** - Growing subsystem
5. **Low Priority: others** - Nice to have

## Alternative: Gradual Migration

Start with just 3 folders:
```
cmd/
├── core/       # Modern commands
├── legacy/     # Deprecated commands
└── (everything else stays flat temporarily)
```

Then add other folders as needed.

## Notes

- Package name stays `package cmd` everywhere
- Import paths unchanged: `github.com/go-nv/goenv/cmd`
- Cobra command registration in root.go unaffected
- Go's build system handles subdirectories automatically
- `go test ./cmd/...` tests all subdirectories

## Example File Header (no changes needed)

```go
package cmd  // ← stays the same

import (
    "github.com/spf13/cobra"
    // ... other imports
)
```

Works because Go allows a package to span multiple directories when they're all part of the same logical package.
