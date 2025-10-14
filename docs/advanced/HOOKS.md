# Hooks Guide

Hooks are scripts that execute at specific points during goenv command execution, allowing you to customize and extend goenv's behavior without modifying the core code.

> **✅ Cross-Platform Support:** Hooks work on all platforms! Use `.bash`/`.sh` scripts on Unix/WSL, `.ps1` scripts for PowerShell, or `.cmd`/`.bat` scripts for Windows Command Prompt. goenv automatically detects and uses the appropriate interpreter.

## Table of Contents

- [Overview](#overview)
- [Platform Support](#platform-support)
- [Hook Points](#hook-points)
- [Hook Locations](#hook-locations)
- [Environment Variables](#environment-variables)
- [Writing Hooks](#writing-hooks)
- [Examples](#examples)
- [Best Practices](#best-practices)

## Overview

Hooks provide a powerful extension mechanism for goenv. They run automatically at predefined points during command execution and can:

- Perform actions before/after operations (logging, notifications)
- Modify environment variables that commands will use
- Integrate with external tools and services
- Enforce policies or validation rules

## Platform Support

goenv hooks now support multiple script types and automatically detect the appropriate interpreter:

### Unix-like Systems (Linux, macOS, FreeBSD)
✅ **Fully Supported**
- **`.bash` scripts** - Executed with `bash` (preferred)
- **`.sh` scripts** - Executed with `sh` or `bash`
- **Shebang support** - Scripts without extensions use `#!/usr/bin/env bash` or similar
- **`.ps1` scripts** - Executed with `pwsh` (PowerShell Core) if installed

### Windows
✅ **Fully Supported** with multiple script types:

**Native Windows (PowerShell)**
- **`.ps1` scripts** - Executed with `powershell.exe` or `pwsh.exe`
- Automatic detection of PowerShell Core (`pwsh`) vs Windows PowerShell
- Execution policy automatically bypassed for hooks

**Native Windows (Command Prompt)**
- **`.cmd` scripts** - Executed with `cmd /c`
- **`.bat` scripts** - Executed with `cmd /c`

**Windows with Unix Shells (WSL, Git Bash, Cygwin, MSYS2)**
- **`.bash` scripts** - Executed with `bash`
- **`.sh` scripts** - Executed with `sh` or `bash`
- Full Unix-style hook support

### Script Type Priority

When multiple hook scripts exist for the same hook point, they are executed in alphabetical order by filename. Use numeric prefixes to control execution order:

```
install/
  ├── 00-check-disk-space.ps1    # Runs first
  ├── 10-validate.bash            # Runs second
  └── 20-notify.cmd               # Runs third
```

## Hook Points

goenv executes hooks at the following points:

| Hook Point | When It Runs | Common Use Cases |
|------------|--------------|------------------|
| `exec` | Before executing a Go command via `goenv exec` | Logging, environment setup, telemetry |
| `rehash` | After regenerating shims | Cache invalidation, post-install tasks |
| `which` | Before finding a binary location | Path customization, binary validation |
| `version-name` | When determining the current version name | Version aliasing, custom version resolution |
| `version-origin` | When determining where version was set | Custom origin messages, policy enforcement |
| `install` | Before installing a Go version | Pre-installation checks, disk space validation |
| `uninstall` | Before uninstalling a Go version | Backup creation, confirmation prompts |

## Hook Locations

Hooks are scripts located in directories specified by the `GOENV_HOOK_PATH` environment variable.

**Path separator:**
- **Unix/macOS/Linux**: Colon `:` separated list
- **Windows**: Semicolon `;` separated list

**Default hook search paths:**
```bash
# If GOENV_HOOK_PATH is not set, goenv searches:
# (No default paths - you must set GOENV_HOOK_PATH)

# Unix/macOS/Linux - colon separator
export GOENV_HOOK_PATH="$HOME/.goenv/hooks:/usr/local/etc/goenv/hooks"

# Windows PowerShell - semicolon separator
$env:GOENV_HOOK_PATH="$env:USERPROFILE\.goenv\hooks;C:\ProgramData\goenv\hooks"

# Windows CMD - semicolon separator
set GOENV_HOOK_PATH=%USERPROFILE%\.goenv\hooks;C:\ProgramData\goenv\hooks
```

**Hook file naming convention:**
- Hook files must have a supported extension or use shebang
- Supported extensions: `.bash`, `.sh`, `.ps1`, `.cmd`, `.bat`
- Files without extensions must have a valid shebang line
- Examples: `exec.bash`, `install.ps1`, `rehash.cmd`

**Directory structure examples:**

Unix/Linux:
```
$HOME/.goenv/hooks/
  ├── exec.bash
  ├── install.bash
```

Windows (PowerShell):
```
%USERPROFILE%\.goenv\hooks\
  ├── exec.ps1
  ├── install.ps1
```

Windows (Command Prompt):
```
%USERPROFILE%\.goenv\hooks\
  ├── exec.cmd
  ├── install.bat
```

Cross-platform setup:
```
$GOENV_ROOT/hooks/
  ├── exec.bash          # Unix/WSL/Git Bash
  ├── exec.ps1           # Windows PowerShell
  ├── exec.cmd           # Windows CMD
  ├── install.bash
  ├── install.ps1
  ├── uninstall.bash
  ├── rehash.bash
  ├── version-name.bash
  └── version-origin.bash
```

## Environment Variables

Hooks receive specific environment variables depending on the hook point:

### All Hooks

All hooks inherit the standard goenv environment:
- `GOENV_ROOT` - Root directory of goenv installation
- `GOENV_VERSION` - Currently active Go version (when applicable)
- `GOENV_DEBUG` - Debug mode flag

### Hook-Specific Variables

#### `exec` Hook
- `GOENV_VERSION` - The version being used to execute
- `GOENV_COMMAND` - The command being executed (e.g., `go`, `gofmt`)
- `GOENV_FILE_ARG` - Path to file if command has file arguments (e.g., `go run main.go` → `main.go`)

#### `install` Hook
- `GOENV_VERSION` - The version being installed

#### `uninstall` Hook
- `GOENV_VERSION` - The version being uninstalled

#### `version-name` Hook
- `GOENV_VERSION` - The current version name

#### `version-origin` Hook
- `GOENV_VERSION_ORIGIN` - The origin can be **set by the hook** to override the default origin message

## Writing Hooks

### Basic Hook Structure

**Unix/macOS/Linux:**
```bash
#!/usr/bin/env bash
# $GOENV_ROOT/hooks/exec/exec.bash

# Your hook code here
echo "Executing $GOENV_COMMAND with Go $GOENV_VERSION"
```

**Windows PowerShell:**
```powershell
# $env:GOENV_ROOT\hooks\exec\exec.ps1

# Your hook code here
Write-Host "Executing $env:GOENV_COMMAND with Go $env:GOENV_VERSION"
```

**Windows CMD:**
```batch
@echo off
REM %GOENV_ROOT%\hooks\exec\exec.cmd

REM Your hook code here
echo Executing %GOENV_COMMAND% with Go %GOENV_VERSION%
```

### Hook Execution

1. Hooks are executed in alphabetical order if multiple hooks exist
2. Hooks run in a subshell - they cannot modify the parent process
3. Hook exit codes are logged in debug mode but don't stop command execution
4. Hooks should be fast - they run on every command invocation

### Making Hooks Executable

**Unix/macOS/Linux:**
```bash
chmod +x $GOENV_ROOT/hooks/exec/exec.bash
```

**Windows:**
No action needed - all `.ps1`, `.cmd`, and `.bat` files are executable by default.

## Examples

> **Note:** These examples show Unix/Linux/macOS bash scripts. For cross-platform examples including PowerShell and CMD, see the [Cross-Platform Hook Examples](#cross-platform-hook-examples) section below.

### Example 1: Logging Exec Commands (Unix/Linux/macOS)

Log all Go commands to a file for audit purposes:

```bash
#!/usr/bin/env bash
# $GOENV_ROOT/hooks/exec/exec.bash

LOGFILE="$HOME/.goenv/exec.log"
echo "$(date '+%Y-%m-%d %H:%M:%S') | Version: $GOENV_VERSION | Command: $GOENV_COMMAND" >> "$LOGFILE"
```

### Example 2: Version Migration Warning (Unix/Linux/macOS)

Warn when using an old Go version:

```bash
#!/usr/bin/env bash
# $GOENV_ROOT/hooks/exec/exec.bash

# Parse major.minor version
if [[ $GOENV_VERSION =~ ^([0-9]+)\.([0-9]+) ]]; then
  major="${BASH_REMATCH[1]}"
  minor="${BASH_REMATCH[2]}"
  
  # Warn if using Go < 1.20
  if [[ $major -eq 1 && $minor -lt 20 ]]; then
    echo "⚠️  Warning: Go $GOENV_VERSION is outdated. Consider upgrading to 1.20+." >&2
  fi
fi
```

### Example 3: Pre-Install Disk Space Check (Unix/Linux/macOS)

Ensure sufficient disk space before installation:

```bash
#!/usr/bin/env bash
# $GOENV_ROOT/hooks/install/install.bash

# Check available disk space in GOENV_ROOT (in GB)
available=$(df -BG "$GOENV_ROOT" | awk 'NR==2 {print $4}' | sed 's/G//')

if [[ $available -lt 2 ]]; then
  echo "❌ Error: Insufficient disk space. Need at least 2GB free." >&2
  echo "   Available: ${available}GB" >&2
  exit 1
fi

echo "✓ Disk space check passed (${available}GB available)"
```

### Example 4: Custom Version Origin (Unix/Linux/macOS)

Override the version origin message:

```bash
#!/usr/bin/env bash
# $GOENV_ROOT/hooks/version-origin/version-origin.bash

# If version is from .go-version, add project info
if [[ -f .go-version ]]; then
  if [[ -f go.mod ]]; then
    module=$(grep '^module ' go.mod | awk '{print $2}')
    export GOENV_VERSION_ORIGIN="$PWD/.go-version (module: $module)"
  fi
fi
```

### Example 5: Integration with Development Tools (Unix/Linux/macOS)

Send telemetry to an analytics service:

```bash
#!/usr/bin/env bash
# $GOENV_ROOT/hooks/exec/exec.bash

# Only track in specific projects (adjust path pattern for your OS)
if [[ $PWD == $HOME/projects/* ]]; then
  # Send async telemetry (don't block execution)
  (
    curl -s -X POST "https://analytics.example.com/track" \
      -H "Content-Type: application/json" \
      -d "{\"event\": \"go_command\", \"version\": \"$GOENV_VERSION\", \"command\": \"$GOENV_COMMAND\"}" \
      > /dev/null 2>&1
  ) &
fi
```

### Example 6: Automatic GOPATH-installed Tool Rehashing (Unix/Linux/macOS)

Automatically rehash after installing tools with `go install`:

```bash
#!/usr/bin/env bash
# $GOENV_ROOT/hooks/exec/exec.bash

# If running 'go install', rehash after completion
if [[ $GOENV_COMMAND == "go" ]]; then
  # Check if 'install' is in the arguments
  for arg in "$@"; do
    if [[ $arg == "install" ]]; then
      # Schedule rehash to run after go install completes
      trap 'goenv rehash > /dev/null 2>&1' EXIT
      break
    fi
  done
fi
```

### Example 7: Environment-Specific Configuration (Unix/Linux/macOS)

Set different configurations based on the project:

```bash
#!/usr/bin/env bash
# $GOENV_ROOT/hooks/exec/exec.bash

# Load project-specific environment
if [[ -f .env.goenv ]]; then
  source .env.goenv
fi

# Set GOPRIVATE for corporate projects (adjust path pattern for your OS)
if [[ $PWD == $HOME/work/* ]]; then
  export GOPRIVATE="github.com/mycompany/*"
fi
```

## Cross-Platform Hook Examples

The same functionality can be implemented across all platforms using the appropriate script type:

### Logging Example - All Platforms

**Unix/Linux/macOS (exec.bash):**
```bash
#!/usr/bin/env bash
# $GOENV_ROOT/hooks/exec/exec.bash

LOGFILE="$HOME/.goenv/exec.log"
echo "$(date '+%Y-%m-%d %H:%M:%S') | Version: $GOENV_VERSION | Command: $GOENV_COMMAND" >> "$LOGFILE"
```

**Windows PowerShell (exec.ps1):**
```powershell
# $env:GOENV_ROOT\hooks\exec\exec.ps1

$LogFile = "$env:USERPROFILE\.goenv\exec.log"
$Timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
$LogEntry = "$Timestamp | Version: $env:GOENV_VERSION | Command: $env:GOENV_COMMAND"
Add-Content -Path $LogFile -Value $LogEntry
```

**Windows Command Prompt (exec.cmd):**
```batch
@echo off
REM %GOENV_ROOT%\hooks\exec\exec.cmd

set LOGFILE=%USERPROFILE%\.goenv\exec.log
echo %date% %time% ^| Version: %GOENV_VERSION% ^| Command: %GOENV_COMMAND% >> "%LOGFILE%"
```

### Disk Space Check Example - All Platforms

**Unix/Linux/macOS (install.bash):**
```bash
#!/usr/bin/env bash
# $GOENV_ROOT/hooks/install/install.bash

available=$(df -BG "$GOENV_ROOT" | awk 'NR==2 {print $4}' | sed 's/G//')
if [[ $available -lt 2 ]]; then
  echo "❌ Error: Insufficient disk space. Need at least 2GB free." >&2
  exit 1
fi
echo "✓ Disk space check passed (${available}GB available)"
```

**Windows PowerShell (install.ps1):**
```powershell
# $env:GOENV_ROOT\hooks\install\install.ps1

$Drive = (Get-Item $env:GOENV_ROOT).PSDrive.Name
$Disk = Get-PSDrive $Drive
$AvailableGB = [math]::Round($Disk.Free / 1GB, 2)

if ($AvailableGB -lt 2) {
    Write-Error "❌ Error: Insufficient disk space. Need at least 2GB free."
    Write-Error "   Available: ${AvailableGB}GB"
    exit 1
}
Write-Host "✓ Disk space check passed (${AvailableGB}GB available)"
```

**Windows Command Prompt (install.cmd):**
```batch
@echo off
REM %GOENV_ROOT%\hooks\install\install.cmd

REM Simple check - ensure GOENV_ROOT drive is accessible
if not exist "%GOENV_ROOT%\" (
    echo Error: Cannot access GOENV_ROOT
    exit /b 1
)
echo Disk space check passed
```

### Version Warning Example - All Platforms

**Unix/Linux/macOS (exec.bash):**
```bash
#!/usr/bin/env bash
# $GOENV_ROOT/hooks/exec/exec.bash

if [[ $GOENV_VERSION =~ ^1\.([0-9]+) ]]; then
  minor="${BASH_REMATCH[1]}"
  if [[ $minor -lt 20 ]]; then
    echo "⚠️  Warning: Go $GOENV_VERSION is outdated. Consider upgrading." >&2
  fi
fi
```

**Windows PowerShell (exec.ps1):**
```powershell
# $env:GOENV_ROOT\hooks\exec\exec.ps1

if ($env:GOENV_VERSION -match '^1\.(\d+)') {
    $minor = [int]$matches[1]
    if ($minor -lt 20) {
        Write-Warning "⚠️  Warning: Go $env:GOENV_VERSION is outdated. Consider upgrading."
    }
}
```

**Windows Command Prompt (exec.cmd):**
```batch
@echo off
REM %GOENV_ROOT%\hooks\exec\exec.cmd

REM Extract minor version (simplified check)
echo %GOENV_VERSION% | findstr /r "^1\.1[0-9]" >nul
if %errorlevel% equ 0 (
    echo Warning: Go %GOENV_VERSION% is outdated. Consider upgrading. >&2
)
```

## Best Practices

### Cross-Platform Compatibility

1. **Provide multiple script types** for broad compatibility
2. **Test on target platforms** before deploying
3. **Use platform-specific features** appropriately (PowerShell on Windows, bash on Unix)
4. **Document platform requirements** in hook comments

```bash
# Example: Provide both bash and PowerShell versions
# $GOENV_ROOT/hooks/exec/
#   ├── exec.bash    # For Unix/WSL/Git Bash
#   └── exec.ps1     # For Windows PowerShell
```

### Performance

1. **Keep hooks fast** - They run on every command
2. **Use background processes** for slow operations (network calls, disk I/O)
3. **Cache results** when possible
4. **Exit early** if conditions aren't met

```bash
# Good - exit early
[[ -z $GOENV_VERSION ]] && exit 0

# Good - background slow operations
(slow_operation) &
```

### Error Handling

1. **Don't fail silently** - Log errors for debugging
2. **Use debug mode** - Check `GOENV_DEBUG` for verbose output
3. **Don't break commands** - Hook failures shouldn't stop goenv

```bash
if [[ -n $GOENV_DEBUG ]]; then
  echo "Debug: Hook executing for version $GOENV_VERSION" >&2
fi
```

### Security

1. **Validate inputs** - Don't trust environment variables blindly
2. **Use full paths** for external commands
3. **Be careful with eval** and dynamic execution
4. **Set restrictive permissions** on hook files (Unix: chmod 700)

```bash
# Good - validate version format (bash)
if [[ ! $GOENV_VERSION =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  exit 0
fi

# Good - use full paths (bash)
/usr/bin/curl ...
```

```powershell
# Good - validate version format (PowerShell)
if ($env:GOENV_VERSION -notmatch '^\d+\.\d+\.\d+$') {
    exit 0
}

# Good - use cmdlets instead of external commands
Invoke-WebRequest ...
```

### Debugging

1. **Use GOENV_DEBUG** - Check if debug mode is enabled
2. **Log to stderr** - Keep stdout clean for command output
3. **Include timestamps** in logs
4. **Test hooks independently** before deploying

```bash
if [[ -n $GOENV_DEBUG ]]; then
  echo "[$(date '+%Y-%m-%d %H:%M:%S')] Hook: exec | Version: $GOENV_VERSION" >&2
fi
```

### Organization

1. **One hook per file** - Don't combine multiple hook points
2. **Comment your hooks** - Explain what and why
3. **Version control** - Keep hooks in your dotfiles repo
4. **Share team hooks** - Use `/usr/local/etc/goenv/hooks` for team-wide hooks

## Troubleshooting

### Hook Not Executing

1. Check `GOENV_HOOK_PATH` is set:
   ```bash
   # Unix/macOS
   echo $GOENV_HOOK_PATH
   
   # Windows PowerShell
   echo $env:GOENV_HOOK_PATH
   
   # Windows CMD
   echo %GOENV_HOOK_PATH%
   ```

2. Verify hook file exists:
   ```bash
   # Unix/macOS
   ls -la ~/.goenv/hooks/
   
   # Windows PowerShell
   Get-ChildItem $env:GOENV_ROOT\hooks\ -Recurse
   
   # Windows CMD
   dir %GOENV_ROOT%\hooks\ /s
   ```

3. Check hook file has correct extension:
   - Unix/macOS/WSL: `.bash`, `.sh`, or shebang
   - Windows PowerShell: `.ps1`
   - Windows CMD: `.cmd` or `.bat`

4. Verify interpreter is available:
   ```bash
   # Check bash (Unix/WSL/Git Bash)
   which bash
   
   # Check PowerShell (Windows)
   where.exe pwsh    # PowerShell Core
   where.exe powershell  # Windows PowerShell
   
   # Check cmd (Windows)
   where.exe cmd
   ```

5. Enable debug mode to see hook execution:
   ```bash
   # Unix/macOS
   GOENV_DEBUG=1 goenv exec go version
   
   # Windows PowerShell
   $env:GOENV_DEBUG=1; goenv exec go version
   
   # Windows CMD
   set GOENV_DEBUG=1 && goenv exec go version
   ```

### Hook Errors

View hook errors by enabling debug mode:
```bash
# Unix/macOS
GOENV_DEBUG=1 goenv <command>

# Windows PowerShell
$env:GOENV_DEBUG=1; goenv <command>

# Windows CMD
set GOENV_DEBUG=1 && goenv <command>
```

Test hook independently:

**Unix/macOS/WSL:**
```bash
GOENV_VERSION=1.22.5 GOENV_COMMAND=go bash "$GOENV_ROOT/hooks/exec/exec.bash"
```

**Windows PowerShell:**
```powershell
$env:GOENV_VERSION="1.22.5"
$env:GOENV_COMMAND="go"
& "$env:GOENV_ROOT\hooks\exec\exec.ps1"
```

**Windows CMD:**
```batch
set GOENV_VERSION=1.22.5
set GOENV_COMMAND=go
call "%GOENV_ROOT%\hooks\exec\exec.cmd"
```

## Further Reading

- [Environment Variables Reference](../reference/ENVIRONMENT_VARIABLES.md)
- [Command Reference](../reference/COMMANDS.md)
- [Advanced Configuration](ADVANCED_CONFIGURATION.md)
