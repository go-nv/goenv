# VS Code Go Extension Internals

## How the Go Extension Manages Paths

This document explains how the VS Code Go extension interacts with goenv and why PATH issues occur.

## The Problem

The Go extension uses VS Code's **Environment Variable Collections API** to inject Go paths into terminal sessions. This causes conflicts with goenv because:

1. It runs **before shell initialization** (`.bashrc`, `.zshrc`, etc.)
2. It **caches** the Go path in VS Code's internal databases
3. It **persists across restarts** until explicitly cleared
4. It **cannot be overridden** by shell environment variables

### Cache Locations

The Go extension stores cached environment state in:

**macOS:**
```
~/Library/Application Support/Code/User/workspaceStorage/*/state.vscdb
~/Library/Application Support/Code/User/globalStorage/state.vscdb
```

**Linux:**
```
~/.config/Code/User/workspaceStorage/*/state.vscdb
~/.config/Code/User/globalStorage/state.vscdb
```

**Windows:**
```
%APPDATA%\Code\User\workspaceStorage\*\state.vscdb
%APPDATA%\Code\User\globalStorage\state.vscdb
```

### How It Works

1. **Extension activates** when you open a Go file
2. **Detects Go installation** by running `which go` or checking settings
3. **Caches GOROOT** in memory and state database
4. **Injects PATH** via `EnvironmentVariableCollection.prepend('PATH', ...)`
5. **All new terminals** get this PATH injection automatically

### Why This Breaks goenv

```
Terminal Startup Sequence:
1. VS Code creates PTY
2. Extension injects: PATH=/Users/you/.goenv/versions/1.23.2/bin:$PATH
3. Shell starts (bash/zsh)
4. Shell init runs: eval "$(goenv init -)"
5. goenv adds: PATH=/Users/you/.goenv/shims:$PATH

Result: /Users/you/.goenv/versions/1.23.2/bin:/Users/you/.goenv/shims:...
        ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^
        Extension's stale path takes precedence!
```

## goenv's Solution

goenv provides two-level configuration:

### Level 1: User Settings (Global)

Configure the extension to **not manage Go paths**:

```json
{
  "go.goroot": "",
  "go.gopath": "",
  "go.toolsManagement.autoUpdate": false,
  "go.alternateTools": {
    "go": "goenv exec go"
  }
}
```

**Effect:**
- `go.goroot: ""` → Extension doesn't inject PATH
- `go.gopath: ""` → Extension doesn't set GOPATH
- `go.alternateTools` → Extension uses goenv to find Go

**Modified by:** `goenv doctor --fix`

### Level 2: Workspace Settings (Project-Specific)

Tell the extension which Go version to use for IntelliSense:

```json
{
  "go.goroot": "${env:HOME}/.goenv/versions/1.25.4",
  "go.gopath": "${env:HOME}/go/1.25.4"
}
```

**Effect:**
- Extension uses correct Go for gopls, debugging, etc.
- Does NOT inject into terminal PATH (because user settings prevent it)

**Modified by:** `goenv vscode init/sync/setup`

## Environment Variable Collections API

This is the VS Code API the Go extension uses:

```typescript
// Inside Go extension code
const collection = context.environmentVariableCollection;

// Prepend Go to PATH
collection.prepend('PATH', '/path/to/go/bin' + delimiter);

// This affects ALL terminals, forever (until cleared)
```

### API Characteristics

- **Scope:** Global (all terminals) or per-workspace
- **Persistence:** Survives VS Code restart
- **Priority:** Applied before shell initialization
- **Visibility:** Hidden from users (no UI to inspect/clear in older VS Code versions)
- **Override:** Cannot be undone by shell environment

### Why Extensions Use It

The API was designed to solve a real problem:
- Users open VS Code from GUI (no shell environment)
- Extensions need to add tools to PATH
- Shell init files don't run in this scenario

But it conflicts with version managers like:
- goenv, nvm, rbenv, pyenv, etc.
- All rely on shell initialization
- All expect to control PATH themselves

## Detection Strategy

goenv detects PATH injection by:

1. **Checking user settings** for `go.goroot` or `go.gopath` with non-empty values
2. **Inspecting state databases** for cached GOROOT (if accessible)
3. **Comparing terminal PATH** to expected PATH (if running in VS Code terminal)

The `goenv doctor` command performs these checks and reports findings.

## Fix Strategy

goenv's fix is conservative:

1. **Never auto-fix** user settings without explicit consent
2. **Always create backup** before modifying settings
3. **Warn about side effects** (comment stripping)
4. **Only modify Go-related keys**, leave everything else intact

### What the Fix Changes

**Before:**
```json
{
  "go.goroot": "/Users/you/.goenv/versions/1.23.2",
  "go.gopath": "/Users/you/go",
  // ... 700 other settings ...
}
```

**After:**
```json
{
  "go.goroot": "",
  "go.gopath": "",
  "go.toolsManagement.autoUpdate": false,
  "go.alternateTools": {"go": "goenv exec go"},
  // ... 700 other settings (comments removed) ...
}
```

### Why Comments Are Lost

The fix uses `github.com/tidwall/jsonc` which:
1. Parses JSONC (JSON with Comments)
2. Converts to pure JSON
3. Modifies specific keys
4. Writes back

Step 2 strips comments because there's no Go library that can:
- Parse JSONC
- Modify specific keys
- Preserve comments and formatting
- Write back as JSONC

This is a known limitation. VS Code itself strips comments when modifying settings via its API.

## Alternative Solutions Considered

### 1. Manual Instructions

Simplest approach:
- Document what settings to add
- Let users edit manually
- No automation, no bugs

**Rejected because:** Most users won't do it correctly.

### 2. VS Code CLI

Use `code --user-data-dir ...` to modify settings:
```bash
code --user-data-dir "$HOME/Library/Application Support/Code" \
     settings set go.goroot ""
```

**Rejected because:**
- Not available in all VS Code installations
- Requires VS Code CLI in PATH
- Complex cross-platform support

### 3. Clear State Databases

Delete the cached state:
```bash
rm ~/Library/Application\ Support/Code/User/workspaceStorage/*/state.vscdb*
```

**Rejected because:**
- Removes ALL extension state, not just Go
- Extensions may break or require re-initialization
- Very invasive

### 4. Educate Go Extension Team

Work with Go extension maintainers to:
- Detect version managers (goenv, gvm, etc.)
- Skip PATH injection when detected
- Provide official integration API

**Status:** Potential future collaboration.

## For Extension Developers

If you're building a VS Code extension that manages tools:

### Best Practices

1. **Check for version managers first:**
   ```typescript
   const hasGoenv = await hasCommand('goenv');
   if (hasGoenv) {
     // Let goenv handle it
     return;
   }
   ```

2. **Make PATH injection optional:**
   ```json
   "go.manageToolPath": {
     "type": "boolean",
     "default": true,
     "description": "Let extension add Go to PATH"
   }
   ```

3. **Respect user settings:**
   ```typescript
   if (config.get('go.goroot') === '') {
     // User explicitly disabled - don't override
     return;
   }
   ```

4. **Provide integration hooks:**
   ```json
   "go.toolManager": {
     "type": "string",
     "enum": ["extension", "goenv", "gvm", "manual"],
     "default": "extension"
   }
   ```

### goenv-Friendly Extension

An ideal extension would:
- Detect goenv via `which goenv`
- Use `goenv exec go` instead of direct path
- Skip Environment Variable Collections API
- Let shell initialization handle PATH

## Testing

To verify the fix works:

```bash
# 1. Create a test scenario
goenv local 1.25.4
echo "$PATH" | grep 1.23.2  # Should NOT appear

# 2. Open VS Code terminal
code .
# In VS Code terminal:
which go                    # Should be: ~/.goenv/shims/go
go version                  # Should match: goenv current

# 3. Check extension behavior
# Look at VS Code status bar - should show correct version
# Run Go: Show All Commands → Go: Locate Configured Go Tools
# Should point to shims or current version

# 4. Verify no PATH pollution
echo "$PATH" | tr ':' '\n' | cat -n | grep go
# Should see:
#   N  /Users/you/.goenv/shims
#   M  /Users/you/.goenv/bin
# Should NOT see version-specific paths
```

## References

- [VS Code Environment Variable Collections API](https://code.visualstudio.com/api/references/vscode-api#EnvironmentVariableCollection)
- [Go Extension Source Code](https://github.com/golang/vscode-go)
- [VS Code Settings Precedence](https://code.visualstudio.com/docs/getstarted/settings)
- [Issue: Go extension PATH conflicts](https://github.com/golang/vscode-go/issues)

## Future Enhancements

### Planned

1. **Auto-detect and warn** when PATH injection is detected
2. **Interactive fix prompt** with clear explanation
3. **Settings validation** on every goenv version change
4. **VS Code extension** that integrates directly with goenv

### Under Consideration

1. **Hook into Go extension** via proposed collaboration
2. **Custom terminal profiles** that bypass injection
3. **Shell wrapper** that fixes PATH on startup
4. **VS Code workspace trust** integration
