# VS Code Integration Troubleshooting

## Common Issues and Solutions

### Issue: Terminal Shows Wrong Go Version

**Symptoms:**
```bash
# goenv says one thing
$ goenv current
1.25.4 (set by .go-version)

# But terminal uses another
$ which go
/Users/you/.goenv/versions/1.23.2/bin/go

$ go version
go version go1.23.2 darwin/arm64
```

**Cause:** The Go extension is injecting stale paths into terminal PATH via VS Code's Environment Variable Collections API. This happens **before** your shell initialization runs, bypassing goenv.

**Solution:**

1. Check if the issue exists:
   ```bash
   goenv doctor
   ```
   Look for warnings about "VS Code Go extension"

2. Fix it (with confirmation prompt):
   ```bash
   goenv doctor --fix
   ```
   
3. Reload VS Code: `⌘+Shift+P` → "Developer: Reload Window"

4. Open a **new** terminal and verify:
   ```bash
   which go
   go version
   goenv current
   ```

**What the fix does:**
- Updates your VS Code user settings (`~/Library/Application Support/Code/User/settings.json`)
- Sets `go.goroot: ""` and `go.gopath: ""` to prevent PATH injection
- Configures `go.alternateTools.go: "goenv exec go"` to use goenv

**⚠️ Important:** The fix will **remove comments** from your user settings file. A backup is created at `settings.json.backup`.

### Issue: Workspace Settings Out of Sync

**Symptoms:**
- VS Code shows Go 1.23.2 in status bar
- But `.go-version` file says 1.25.4
- Go extension features use wrong version

**Cause:** Workspace `.vscode/settings.json` has hardcoded paths to an old Go version.

**Solution:**

1. Update workspace settings to match current version:
   ```bash
   goenv vscode sync
   ```

2. Or do a complete setup:
   ```bash
   goenv vscode setup
   ```

3. Reload VS Code window

### Issue: Settings Keep Reverting

**Cause:** You have settings at multiple levels (user, workspace, folder) that conflict.

**Solution:**

1. Check all settings locations:
   - User: `~/Library/Application Support/Code/User/settings.json` (macOS)
   - Workspace: `.vscode/settings.json` 
   - Folder: If in multi-root workspace

2. Run doctor to see the status:
   ```bash
   goenv vscode doctor
   ```

3. Fix workspace settings:
   ```bash
   goenv vscode sync
   ```

4. If user settings have PATH injection, fix with:
   ```bash
   goenv doctor --fix
   ```

### Issue: Shims Not Found

**Symptoms:**
```bash
$ which go
/Users/you/.goenv/versions/1.23.2/bin/go
# Should be: /Users/you/.goenv/shims/go
```

**Cause:** VS Code Go extension is adding version-specific paths to terminal PATH.

**Solution:** Same as "Terminal Shows Wrong Go Version" above.

## Prevention

### First-Time Setup

Run this once when starting a new project:
```bash
cd /path/to/project
goenv local 1.25.4           # Set version for project
goenv vscode setup           # Configure VS Code integration
```

### After Changing Go Versions

The workspace settings need to be updated:
```bash
goenv local 1.26.0           # Change version
goenv vscode sync            # Update VS Code settings
```

**Future Enhancement:** Auto-sync is planned so `goenv local` will automatically update VS Code settings.

## Understanding the Architecture

VS Code has **two levels** of settings:

1. **User Settings** (`~/Library/Application Support/Code/User/settings.json`)
   - Global across all projects
   - Controls Go extension behavior
   - Can inject paths into terminal PATH
   - **Modified by:** `goenv doctor --fix`

2. **Workspace Settings** (`.vscode/settings.json`)
   - Project-specific
   - Tells Go extension which GOROOT/GOPATH to use
   - Controls gopls and other tools
   - **Modified by:** `goenv vscode init/sync/setup`

### Environment Variable Collections API

VS Code extensions can inject environment variables directly into terminals via a special API. This:
- Runs **before** shell initialization (`.bashrc`, `.zshrc`)
- Bypasses tools like goenv, nvm, rbenv
- Persists in cached state databases
- Cannot be overridden by shell init

The Go extension uses this to "help" by adding Go to PATH, but it caches stale paths.

## Known Limitations

### Comment Stripping

When `goenv doctor --fix` modifies your user settings, it uses a JSON library that **removes all comments** from the file. This is unavoidable with current Go JSON libraries.

**Workaround:**
- A backup is always created: `settings.json.backup`
- Restore comments manually if needed
- Or accept the loss (VS Code doesn't require comments)

**Why:** No Go library properly round-trips JSONC (JSON with Comments) while preserving formatting.

### Multi-Root Workspaces

VS Code multi-root workspaces have complex settings inheritance. goenv handles the common case (single workspace) but may not work correctly in multi-root scenarios.

## Diagnostic Commands

```bash
# Check overall goenv health
goenv doctor

# Check VS Code integration specifically
goenv vscode doctor

# Show current VS Code status
goenv vscode status

# See what sync would change (dry run)
goenv vscode sync --dry-run
```

## When to Use Each Command

| Command | Use When |
|---------|----------|
| `goenv vscode setup` | First time, or when everything is broken |
| `goenv vscode init` | Creating new project |
| `goenv vscode sync` | Changed Go version with `goenv local/global` |
| `goenv vscode doctor` | Debugging issues |
| `goenv doctor --fix` | Terminal PATH is wrong |

## Getting Help

If you're still having issues:

1. Run full diagnostics:
   ```bash
   goenv doctor --json > goenv-doctor.json
   goenv vscode doctor --json > vscode-doctor.json
   ```

2. Check your shell initialization is correct:
   ```bash
   # Should show goenv init command
   cat ~/.bashrc ~/.zshrc ~/.bash_profile 2>/dev/null | grep goenv
   ```

3. Verify PATH in a fresh terminal:
   ```bash
   echo "$PATH" | tr ':' '\n' | grep -E "goenv|go"
   ```

4. Check GitHub issues: https://github.com/go-nv/goenv/issues

5. Include the diagnostic output when reporting issues
