# Installation

## Quick Install (Recommended - No Go Required!)

**The easiest way to install goenv is using pre-built binaries.** This method doesn't require Go to be installed on your system.

### Automatic Installation Script

```bash
# Linux/macOS
curl -sfL https://raw.githubusercontent.com/go-nv/goenv/master/install.sh | bash
```

```powershell
# Windows (PowerShell)
iwr -useb https://raw.githubusercontent.com/go-nv/goenv/master/install.ps1 | iex
```

### Manual Binary Installation

1. **Download the latest binary for your platform:**

   Visit [goenv releases](https://github.com/go-nv/goenv/releases/latest) and download the appropriate archive:

   - **Linux (x64)**: `goenv_*_linux_amd64.tar.gz`
   - **Linux (ARM64)**: `goenv_*_linux_arm64.tar.gz`
   - **macOS (Intel)**: `goenv_*_darwin_amd64.tar.gz`
   - **macOS (Apple Silicon)**: `goenv_*_darwin_arm64.tar.gz`
   - **Windows (x64)**: `goenv_*_windows_amd64.zip`
   - **FreeBSD (x64)**: `goenv_*_freebsd_amd64.tar.gz`

2. **Extract and install:**

   ```bash
   # Linux/macOS
   tar -xzf goenv_*_*.tar.gz
   mkdir -p ~/.goenv/bin
   mv goenv ~/.goenv/bin/
   chmod +x ~/.goenv/bin/goenv
   ```

   ```powershell
   # Windows
   Expand-Archive goenv_*_windows_amd64.zip
   mkdir $HOME\.goenv\bin -Force
   mv goenv.exe $HOME\.goenv\bin\
   ```

3. **Add to your shell** (see "Shell Setup" section below)

---

## Basic GitHub Checkout

This will get you going with the latest version of goenv and make it
easy to fork and contribute any changes back upstream.

**Note:** This method requires Go to be installed to build goenv. If you don't have Go installed, use the **Quick Install** method above.

1.  **Check out goenv where you want it installed.**
    A good place to choose is `$HOME/.goenv` (but you can install it somewhere else).

        git clone https://github.com/go-nv/goenv.git ~/.goenv

2.  **Build goenv** (requires Go to be installed):

    cd ~/.goenv
    make build

3.  **Define environment variable `GOENV_ROOT`** to point to the path where
    goenv repo is cloned and add `$GOENV_ROOT/bin` to your `$PATH` for access
    to the `goenv` command-line utility.

        echo 'export GOENV_ROOT="$HOME/.goenv"' >> ~/.bash_profile
        echo 'export PATH="$GOENV_ROOT/bin:$PATH"' >> ~/.bash_profile

    **Zsh note**: Modify your `~/.zshenv` file instead of `~/.bash_profile`.

    **Ubuntu note**: Modify your `~/.bashrc` file instead of `~/.bash_profile`.

4.  **Add `goenv init` to your shell** to enable shims, management of `GOPATH` and `GOROOT` and auto-completion.
    Please make sure `eval "$(goenv init -)"` is placed toward the end of the shell
    configuration file since it manipulates `PATH` during the initialization.

        echo 'eval "$(goenv init -)"' >> ~/.bash_profile

    **Zsh note**: Modify your `~/.zshenv` or `~/.zshrc` file instead of `~/.bash_profile`.

    **Ubuntu note**: Modify your `~/.bashrc` file instead of `~/.bash_profile`.

    **General warning**: There are some systems where the `BASH_ENV` variable is configured
    to point to `.bashrc`. On such systems you should almost certainly put the abovementioned line
    `eval "$(goenv init -)` into `.bash_profile`, and **not** into `.bashrc`. Otherwise you
    may observe strange behaviour, such as `goenv` getting into an infinite loop.
    See pyenv's issue [#264](https://github.com/pyenv/pyenv/issues/264) for details.

5.  **Restart your shell so the path changes take effect.**
    You can now begin using goenv.

        exec $SHELL

6.  **(Optional) Enable tab completion for faster command-line usage.**
    goenv includes built-in shell completion scripts that enable tab completion for commands, flags, and Go versions.

    **Quick Install (Recommended):**

    ```bash
    goenv completion --install
    ```

    This will:

    - Auto-detect your shell (bash/zsh/fish/powershell)
    - Install the completion script to the appropriate location
    - Display activation instructions

    **Manual Install:**

    ```bash
    # Output the completion script and add to your shell config manually
    goenv completion >> ~/.bashrc  # or ~/.zshrc, ~/.config/fish/config.fish, etc.
    ```

    **Restart your shell** to activate completions:

    ```bash
    exec $SHELL
    ```

    After setup, you can use tab completion:

    - `goenv <TAB>` - See all available commands
    - `goenv install <TAB>` - See available Go versions
    - `goenv use <TAB>` - See installed versions

7.  **Install Go versions into `$GOENV_ROOT/versions`.**
    For example, to download and install Go 1.12.0, run:

        goenv install 1.12.0

    **NOTE:** It downloads and places the prebuilt Go binaries provided by Google.

8.  **Set global Go version.**
    For example, to set the version to Go 1.12.0, run:

        goenv use 1.12.0 --global

---

## Shell Setup

After installing goenv (via binary or git checkout), add the following to your shell configuration:

### Bash (~/.bash_profile or ~/.bashrc)

```bash
export GOENV_ROOT="$HOME/.goenv"
export PATH="$GOENV_ROOT/bin:$PATH"
eval "$(goenv init -)"
```

### Zsh (~/.zshrc or ~/.zshenv)

```zsh
export GOENV_ROOT="$HOME/.goenv"
export PATH="$GOENV_ROOT/bin:$PATH"
eval "$(goenv init -)"
```

### Fish (~/.config/fish/config.fish)

```fish
set -Ux GOENV_ROOT $HOME/.goenv
set -U fish_user_paths $GOENV_ROOT/bin $fish_user_paths
status --is-interactive; and goenv init - | source
```

### PowerShell ($PROFILE)

```powershell
$env:GOENV_ROOT = "$HOME\.goenv"
$env:PATH = "$env:GOENV_ROOT\bin;$env:PATH"
& goenv init - | Invoke-Expression
```

Then restart your shell:

```bash
exec $SHELL
```

An example `.zshrc` that is properly configured may look like

```shell
export GOENV_ROOT="$HOME/.goenv"
export PATH="$GOENV_ROOT/bin:$PATH"
eval "$(goenv init -)"
```

## via ZPlug plugin manager for Zsh

Add the following line to your `.zshrc`:

`zplug "RiverGlide/zsh-goenv", from:gitlab`
Then install the plugin

```zsh
  source ~/.zshrc
  zplug install
```

The ZPlug plugin will install and initialise `goenv` and add `goenv` and `goenv-install` to your `PATH`

## Homebrew on macOS

**Recommended for macOS users who use Homebrew.**

You can install goenv using the [Homebrew](http://brew.sh) package manager for macOS (and Linux).

```bash
brew update
brew install goenv
```

**Advantages**:

- Automatic dependency handling
- Easy updates with `brew upgrade goenv`
- Integration with system package manager

**After installation**, you'll need to add the following to your shell profile (as shown in Homebrew caveats):

```bash
# Add to ~/.zshrc or ~/.bash_profile
export GOENV_ROOT="$HOME/.goenv"
export PATH="$GOENV_ROOT/bin:$PATH"
eval "$(goenv init -)"
```

**To see installation details again**:

```bash
brew info goenv
```

**To upgrade in the future**:

```bash
brew upgrade goenv
```

**To uninstall**:

```bash
brew uninstall goenv
```

---

## Windows FAQ

### Where are hook logs stored?

On Windows, hooks log to the same location as other platforms:

```powershell
# Default hooks log location
${env:USERPROFILE}\.goenv\hooks.log

# Example: View recent hook logs
Get-Content ${env:USERPROFILE}\.goenv\hooks.log -Tail 20
```

You can customize the log location in your hooks configuration:

```yaml
# ~/.goenv/hooks.yaml
settings:
  log_file: "C:\logs\goenv-hooks.log"  # Custom location
```

**Quick check:** Run `goenv hooks validate` to see your configured log file path.

### How do I enable desktop notifications (toast notifications)?

The `notify_desktop` hook action uses Windows toast notifications, which require proper PowerShell execution policy and Windows notification settings.

**1. Check PowerShell Execution Policy:**

```powershell
# Check current policy
Get-ExecutionPolicy

# If it's "Restricted", set it to RemoteSigned (allows local scripts)
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

**2. Enable Windows Notifications:**

- Open **Settings** → **System** → **Notifications**
- Ensure **"Get notifications from apps and other senders"** is ON
- Scroll down and ensure **PowerShell** is allowed to show notifications

**3. Test Notifications:**

```yaml
# ~/.goenv/hooks.yaml
version: 1
enabled: true
acknowledged_risks: true

hooks:
  post_install:
    - action: notify_desktop
      params:
        title: "goenv"
        message: "Installed Go {version}"
```

Then run:
```powershell
goenv install 1.23.2
# Should show a toast notification
```

**Troubleshooting:**

**Notifications don't appear:**
- Check Windows Event Viewer for PowerShell errors:
  - Open Event Viewer → Windows Logs → Application
  - Filter for PowerShell errors around the time you ran goenv
- Verify Focus Assist is not in "Priority only" or "Alarms only" mode:
  - Settings → System → Focus assist → Set to "Off"
- Check Action Center settings:
  - Settings → System → Notifications → Turn on "Get notifications from apps and other senders"
  - Scroll down and ensure PowerShell is in the list with notifications enabled

**Corporate/Enterprise environments:**
- Some Group Policies disable toast notifications - contact your IT department
- If you're on a managed device, you may need administrator approval
- Alternative: Use `log_to_file` hook action instead for audit trails

**Windows version issues:**
- Notifications require Windows 10 version 1709 or later
- Windows Server editions may not support toast notifications
- Check your version: `winver` command

**PowerShell version:**
- Works best with PowerShell 5.1+ (built into Windows 10/11)
- If using PowerShell Core (7+), ensure it's properly configured
- Test PowerShell notifications manually:
  ```powershell
  # Quick test of toast notifications
  $null = [Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime]
  # If this errors, your PowerShell can't access Windows.UI APIs
  ```

**Still not working?**
- Try running PowerShell as Administrator once to ensure proper setup
- Check if PowerShell is blocked by antivirus/security software
- Test with a simple PowerShell toast notification script from online tutorials
- Consider using Windows Terminal instead of traditional PowerShell console

### Where do installed tools (gopls, golangci-lint, etc.) land per version?

Tools installed via `go install` are isolated per Go version:

```powershell
# Check your GOPATH for the active version
go env GOPATH
# Example output: C:\Users\YourName\go\1.23.2

# Tools are installed to:
${env:USERPROFILE}\go\{version}\bin\

# Examples:
# gopls for Go 1.23.2:
C:\Users\YourName\go\1.23.2\bin\gopls.exe

# golangci-lint for Go 1.24.0:
C:\Users\YourName\go\1.24.0\bin\golangci-lint.exe
```

**Why per-version isolation?**
- Different Go versions may require different tool versions
- Prevents conflicts when switching between Go versions
- Tools automatically switch with `goenv use <version>`

**List all installed tools for current version:**

```powershell
# Via goenv (shows shimmed tools)
goenv which gopls
goenv which golangci-lint

# Directly list the bin directory
Get-ChildItem ${env:USERPROFILE}\go\$(go version | Select-String -Pattern 'go[\d.]+' | ForEach-Object { $_.Matches.Value })\bin
```

**Shims automatically route to version-specific tools:**

```powershell
# Install gopls for Go 1.23.2
goenv use 1.23.2
go install golang.org/x/tools/gopls@latest

# Shim created at:
${env:USERPROFILE}\.goenv\shims\gopls.exe
# Routes to: ${env:USERPROFILE}\go\1.23.2\bin\gopls.exe

# Switch Go version
goenv use 1.24.0
# Now gopls.exe routes to: ${env:USERPROFILE}\go\1.24.0\bin\gopls.exe
# (or shows "command not found" if not installed for this version)
```

See [GOPATH Integration](../advanced/GOPATH_INTEGRATION.md) for complete details on tool management.

### How do I use goenv in VS Code on Windows?

goenv integrates seamlessly with VS Code on Windows. Use the `goenv vscode setup` command:

```powershell
# Generate VS Code settings
goenv vscode setup

# Or specify custom template
goenv vscode setup --template advanced
```

This creates `.vscode/settings.json` with Windows-compatible paths:

```json
{
  "go.goroot": "${env:USERPROFILE}\\.goenv\\versions\\1.23.2",
  "go.gopath": "${env:USERPROFILE}\\go\\1.23.2"
}
```

**Path format notes:**
- Uses `${env:USERPROFILE}` for cross-user compatibility
- Backslashes are properly escaped in JSON (`\\`)
- VS Code automatically expands environment variables

See [VS Code Integration](VSCODE_INTEGRATION.md) for complete setup guide.

### Why does goenv use backslashes on Windows?

goenv automatically uses Windows-style paths (backslashes) on Windows and Unix-style paths (forward slashes) on macOS/Linux:

```powershell
# Windows
PS> goenv root
C:\Users\YourName\.goenv

# macOS/Linux
$ goenv root
/Users/YourName/.goenv
```

**PowerShell compatibility:**
- Both `\` and `/` work in PowerShell for most operations
- goenv uses `\` for consistency with Windows conventions
- All goenv commands handle path separators automatically

**When using paths in scripts:**
```powershell
# ✅ These all work in PowerShell:
cd $env:USERPROFILE\.goenv
cd $env:USERPROFILE/.goenv
cd "$env:USERPROFILE\.goenv"

# ✅ goenv commands handle both:
goenv install --list
goenv use 1.23.2
```

### Windows Terminal vs PowerShell vs CMD: Which should I use?

**TL;DR: Use Windows Terminal with PowerShell 7+ (recommended) or PowerShell 5.1 (built-in)**

| Shell | Recommended? | Notes |
|-------|--------------|-------|
| **Windows Terminal** + **PowerShell 7** | ✅ **Best** | Modern, cross-platform PowerShell with best features |
| **Windows Terminal** + **PowerShell 5.1** | ✅ **Good** | Built into Windows, no installation needed |
| **PowerShell ISE** | ⚠️ OK | Legacy IDE, limited features, being replaced |
| **CMD (Command Prompt)** | ⚠️ Works | Limited features, use only if required |
| **Git Bash** / **WSL** | ⚠️ Special | See [WSL/Git Bash notes](#using-goenv-with-wsl-or-git-bash) below |

**Recommended setup:**

1. **Install Windows Terminal** (if not already installed):
   ```powershell
   # Via winget (Windows 10 1809+)
   winget install Microsoft.WindowsTerminal

   # Or download from Microsoft Store
   ```

2. **Optional: Install PowerShell 7** (latest version):
   ```powershell
   # Via winget
   winget install Microsoft.PowerShell

   # Or download from: https://aka.ms/powershell
   ```

3. **Set PowerShell as default in Windows Terminal:**
   - Open Windows Terminal
   - Press `Ctrl+,` (Settings)
   - Set "Default profile" to "PowerShell" or "PowerShell 7"

**Why Windows Terminal?**
- Modern, GPU-accelerated rendering
- Multiple tabs and panes
- Unicode and emoji support (for goenv's output)
- Customizable themes and fonts
- Works with PowerShell, CMD, WSL, and Git Bash

**Why PowerShell over CMD?**
- Better scripting capabilities
- Native JSON parsing with `ConvertFrom-Json`
- Aliases work better (`ls`, `cp`, `rm`)
- goenv's interactive prompts work correctly
- Color output and emoji display

**CMD limitations:**
```cmd
REM ❌ CMD doesn't display goenv's emojis correctly
REM ❌ Some interactive prompts may not work
REM ❌ Limited scripting capabilities
REM ✅ Still works for basic goenv commands:
goenv install 1.23.2
goenv use 1.23.2
go version
```

**If you must use CMD:**
- Add `--plain` flag to suppress emojis: `goenv --plain install 1.23.2`
- Or set environment variable: `set NO_COLOR=1`

### Using goenv with WSL or Git Bash

**Windows Subsystem for Linux (WSL):**

If you're using WSL (Ubuntu, Debian, etc.), you should install goenv **inside WSL**, not in Windows:

```bash
# Inside WSL (Ubuntu/Debian)
git clone https://github.com/go-nv/goenv.git ~/.goenv
echo 'export GOENV_ROOT="$HOME/.goenv"' >> ~/.bashrc
echo 'export PATH="$GOENV_ROOT/bin:$PATH"' >> ~/.bashrc
echo 'eval "$(goenv init -)"' >> ~/.bashrc
source ~/.bashrc
```

**Why separate installations?**
- Windows and Linux are different operating systems
- Go binaries are platform-specific (Windows `.exe` vs Linux ELF)
- WSL has its own filesystem and PATH

**Git Bash (MSYS2):**

goenv works in Git Bash, but PowerShell is recommended for better Windows integration:

```bash
# Git Bash (if you prefer Unix-style shell on Windows)
git clone https://github.com/go-nv/goenv.git ~/.goenv
echo 'export GOENV_ROOT="$HOME/.goenv"' >> ~/.bash_profile
echo 'export PATH="$GOENV_ROOT/bin:$PATH"' >> ~/.bash_profile
echo 'eval "$(goenv init -)"' >> ~/.bash_profile
source ~/.bash_profile
```

**Gotchas with Git Bash:**
- Path translation (`C:\Users\...` vs `/c/Users/...`) can cause issues
- Some Windows-specific features may not work as expected
- Use PowerShell for VS Code integration setup

### Common Windows PATH Issues

**Problem: "goenv: command not found" or "'goenv' is not recognized"**

**Solution 1: Verify goenv is in PATH**

```powershell
# Check if goenv directory is in PATH
$env:PATH -split ';' | Select-String goenv

# Should show: C:\Users\YourName\.goenv\bin (or similar)
```

**Solution 2: Re-add goenv to PATH**

```powershell
# Add to current session (temporary)
$env:PATH = "$env:USERPROFILE\.goenv\bin;$env:PATH"

# Add permanently (user-level)
[Environment]::SetEnvironmentVariable(
    "PATH",
    "$env:USERPROFILE\.goenv\bin;$([Environment]::GetEnvironmentVariable('PATH', 'User'))",
    "User"
)

# Restart your terminal after adding permanently
```

**Solution 3: Verify PowerShell profile loaded**

```powershell
# Check if profile exists
Test-Path $PROFILE

# View profile contents
Get-Content $PROFILE

# Should contain goenv initialization
# If not, add it:
@'
$env:GOENV_ROOT = "$env:USERPROFILE\.goenv"
$env:PATH = "$env:GOENV_ROOT\bin;$env:PATH"
& goenv init - --no-rehash | Out-String | Invoke-Expression
'@ | Add-Content $PROFILE
```

**Problem: "go: command not found" after installing Go with goenv**

**Solution: Verify shims are in PATH**

```powershell
# Check if shims directory is in PATH
$env:PATH -split ';' | Select-String shims

# Should show: C:\Users\YourName\.goenv\shims

# If not, goenv init should add it
# Verify goenv init is in your $PROFILE
Get-Content $PROFILE | Select-String "goenv init"

# If missing, reinitialize:
goenv rehash
```

**Problem: PATH changes don't persist after restarting terminal**

**Causes:**
1. PowerShell profile not loading automatically
2. Execution policy blocking profile scripts
3. Incorrect profile location

**Solutions:**

```powershell
# 1. Check execution policy
Get-ExecutionPolicy
# If "Restricted", change it:
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser

# 2. Verify correct profile location
$PROFILE
# Common locations:
# PowerShell 5.1: C:\Users\YourName\Documents\WindowsPowerShell\Microsoft.PowerShell_profile.ps1
# PowerShell 7: C:\Users\YourName\Documents\PowerShell\Microsoft.PowerShell_profile.ps1

# 3. Test if profile loads
. $PROFILE
# Should run without errors

# 4. Check profile is set to load on startup (should be automatic)
# If issues persist, ensure profile directory exists:
New-Item -ItemType Directory -Force -Path (Split-Path -Parent $PROFILE)
```

**Problem: Multiple Go installations conflict (system Go vs goenv Go)**

**Solution: Ensure goenv shims come first in PATH**

```powershell
# Check PATH order
$env:PATH -split ';' | Select-String -Pattern 'go|Go'

# goenv shims should appear BEFORE system Go
# Correct order:
# C:\Users\YourName\.goenv\shims     <-- First (highest priority)
# C:\Users\YourName\.goenv\bin
# C:\Program Files\Go\bin            <-- Last (lowest priority)

# If system Go is first, reorder PATH in $PROFILE:
# Place goenv initialization at the TOP of your profile:
$env:GOENV_ROOT = "$env:USERPROFILE\.goenv"
$env:PATH = "$env:GOENV_ROOT\shims;$env:GOENV_ROOT\bin;$env:PATH"
& goenv init - --no-rehash | Out-String | Invoke-Expression
```

**Problem: Changes to PATH not visible in VS Code integrated terminal**

**Solution: Restart VS Code after PATH changes**

```powershell
# PATH changes require full VS Code restart
# "Reload Window" (Ctrl+Shift+P) is NOT enough

# Steps:
# 1. Make PATH changes in PowerShell profile
# 2. Close VS Code completely (Quit, not just close window)
# 3. Open new terminal to verify PATH:
$env:PATH -split ';' | Select-String goenv
# 4. Relaunch VS Code
# 5. Open integrated terminal and verify:
goenv version
```

### Windows-Specific Environment Variables

goenv respects Windows environment variables and provides Windows-compatible paths:

| Variable | Windows Value | Unix/macOS Value |
|----------|--------------|-------------------|
| `GOENV_ROOT` | `C:\Users\YourName\.goenv` | `~/.goenv` |
| `GOPATH` | `C:\Users\YourName\go\1.23.2` | `~/go/1.23.2` |
| `GOROOT` | `C:\Users\YourName\.goenv\versions\1.23.2` | `~/.goenv/versions/1.23.2` |

**Check current values:**

```powershell
# Show all Go-related environment variables
Get-ChildItem Env: | Where-Object { $_.Name -like 'GO*' }

# Or using goenv
goenv doctor
```

**Override for specific sessions:**

```powershell
# Temporary override (current session only)
$env:GOENV_ROOT = "D:\Development\goenv"

# Permanent override (user-level)
[Environment]::SetEnvironmentVariable("GOENV_ROOT", "D:\Development\goenv", "User")
```

---

## Upgrading

If you've installed goenv using the instructions above, you can
upgrade your installation at any time using git.

To upgrade to the latest development version of goenv, use `git pull`:

    cd ~/.goenv && git fetch --all && git pull

To upgrade to a specific release of goenv, check out the corresponding tag:

    cd ~/.goenv
    git fetch --all
    git tag
    v20160417
    git checkout v20160417

## Uninstalling goenv

The simplicity of goenv makes it easy to temporarily disable it, or
uninstall from the system.

1. To **disable** goenv managing your Go versions, simply remove the
   `goenv init` line from your shell startup configuration. This will
   remove goenv shims directory from PATH, and future invocations like
   `goenv` will execute the system Go version, as before goenv.

`goenv` will still be accessible on the command line, but your Go
apps won't be affected by version switching.

2.  To completely **uninstall** goenv, perform step (1) and then remove
    its root directory. This will **delete all Go versions** that were
    installed under `` `goenv root`/versions/ `` directory:

         rm -rf `goenv root`

    If you've installed goenv using a package manager, as a final step
    perform the goenv package removal. For instance, for Homebrew:

         brew uninstall goenv


## Uninstalling Go Versions

As time goes on, you will accumulate Go versions in your
`~/.goenv/versions` directory.

To remove old Go versions, `goenv uninstall` command to automate
the removal process.

Alternatively, simply `rm -rf` the directory of the version you want
to remove. You can find the directory of a particular Go version
with the `goenv prefix` command, e.g. `goenv prefix 1.6.2`.
