# Hooks System Documentation

## Table of Contents
- [Overview](#overview)
- [Getting Started](#getting-started)
- [Hook Points](#hook-points)
- [Available Actions](#available-actions)
  - [log_to_file](#log_to_file)
  - [http_webhook](#http_webhook)
  - [notify_desktop](#notify_desktop)
  - [check_disk_space](#check_disk_space)
  - [set_env](#set_env)
  - [run_command](#run_command)
- [Configuration Options](#configuration-options)
- [Template Variables](#template-variables)
- [Common Use Cases](#common-use-cases)
- [Advanced Patterns](#advanced-patterns)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Overview

The goenv hooks system allows you to execute automated actions at specific points in the goenv lifecycle. Hooks are declarative, defined in a YAML configuration file, and provide a safe way to extend goenv's functionality without writing shell scripts.

**Key Features:**
- **Declarative:** Define what should happen, not how to do it
- **Safe:** Limited to predefined actions with security controls
- **Non-blocking:** Hook failures don't break goenv commands
- **Template support:** Dynamic variable interpolation in all actions
- **Cross-platform:** Works on Linux, macOS, and Windows

**Common Use Cases:**
- Log Go version installations for audit trails
- Send webhook notifications to team channels
- Check disk space before installing large Go versions
- Set environment variables after version changes
- Display desktop notifications for long-running operations

## Getting Started

### 1. Initialize Configuration

Generate a template configuration file with examples:

```bash
goenv hooks init
```

This creates `~/.goenv/hooks.yaml` with commented examples of all available actions.

### 2. Enable Hooks

Edit `~/.goenv/hooks.yaml` and set:

```yaml
enabled: true
acknowledged_risks: true
```

**Important:** You must set `acknowledged_risks: true` to acknowledge that hooks can execute actions on your system. Review the generated configuration carefully.

### 3. Configure Your First Hook

Add a simple logging hook:

```yaml
hooks:
  post_install:
    - action: log_to_file
      params:
        path: ~/.goenv/install.log
        message: "Installed Go {version} at {timestamp}"
```

### 4. Test Your Configuration

Validate your configuration:

```bash
goenv hooks validate
```

Test hooks without executing them:

```bash
goenv hooks test post_install
```

### 5. List Available Actions

See all available actions and hook points:

```bash
goenv hooks list
```

## Hook Points

Hooks can be triggered at 8 different points in the goenv lifecycle:

### Installation Hooks

**`pre_install`**
- **When:** Before installing a Go version
- **Use cases:** Check disk space, validate prerequisites, send notifications
- **Variables:** `{version}`, `{hook}`, `{timestamp}`

**`post_install`**
- **When:** After successfully installing a Go version
- **Use cases:** Log installation, notify team, set up environment
- **Variables:** `{version}`, `{hook}`, `{timestamp}`

### Uninstallation Hooks

**`pre_uninstall`**
- **When:** Before removing a Go version
- **Use cases:** Create backups, log removal intent, confirm disk space
- **Variables:** `{version}`, `{hook}`, `{timestamp}`

**`post_uninstall`**
- **When:** After successfully removing a Go version
- **Use cases:** Log removal, clean up related files, notify team
- **Variables:** `{version}`, `{hook}`, `{timestamp}`

### Execution Hooks

**`pre_exec`**
- **When:** Before executing a Go command through goenv
- **Use cases:** Log command usage, check environment, validate version
- **Variables:** `{version}`, `{command}`, `{hook}`, `{timestamp}`

**`post_exec`**
- **When:** After executing a Go command through goenv
- **Use cases:** Log execution time, track usage, send metrics
- **Variables:** `{version}`, `{command}`, `{hook}`, `{timestamp}`

### Rehash Hooks

**`pre_rehash`**
- **When:** Before regenerating shims
- **Use cases:** Backup existing shims, log rehash events
- **Variables:** `{hook}`, `{timestamp}`

**`post_rehash`**
- **When:** After regenerating shims
- **Use cases:** Verify shims, update shell completions, notify tools
- **Variables:** `{hook}`, `{timestamp}`

## Available Actions

### log_to_file

Write log messages to a file. Great for audit trails and debugging.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `path` | string | Yes | Absolute or tilde-prefixed path to log file |
| `message` | string | Yes | Message to write (supports templates) |

**Features:**
- Creates parent directories automatically
- Appends to existing files
- Thread-safe writes
- Supports `~` expansion for home directory
- Template variable interpolation

**Example:**

```yaml
hooks:
  post_install:
    - action: log_to_file
      params:
        path: ~/.goenv/logs/install.log
        message: "[{timestamp}] Installed Go {version}"
  
  pre_exec:
    - action: log_to_file
      params:
        path: ~/.goenv/logs/commands.log
        message: "[{timestamp}] Executing '{command}' with Go {version}"
```

**Log Output:**
```
[2025-10-16T10:30:45Z] Installed Go 1.21.3
[2025-10-16T11:15:22Z] Executing 'go build' with Go 1.21.3
```

---

### http_webhook

Send HTTP POST requests with JSON payloads. Perfect for team notifications and integrations.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `url` | string | Yes | HTTP(S) endpoint URL |
| `body` | string | Yes | JSON payload (supports templates) |

**Features:**
- Automatic JSON content-type header
- 5-second timeout (configurable in settings)
- Template variable interpolation in URL and body
- Supports HTTPS
- Non-blocking (errors logged but don't fail commands)

**Security:**
- Only HTTP and HTTPS protocols allowed
- No authentication secrets in config (use environment variables in URL)
- Timeout protection against hanging requests
- SSRF protection (see Security Model below)

#### Security Model

The `http_webhook` action includes comprehensive security controls to prevent Server-Side Request Forgery (SSRF) attacks and protect against malicious URLs.

**Default Security Settings:**

All security features are **enabled by default** and configured via global settings:

```yaml
settings:
  allow_http: false         # Default: HTTPS only
  allow_internal_ips: false # Default: block private IPs
  strict_dns: false         # Default: lenient DNS validation
```

**Security Features:**

1. **HTTPS Enforcement** (`allow_http`)
   - Default: `false` (HTTPS required)
   - When `false`: HTTP URLs are rejected
   - When `true`: Both HTTP and HTTPS allowed
   - Recommendation: Keep at `false` for production

2. **SSRF Protection** (`allow_internal_ips`)
   - Default: `false` (blocks internal/private IPs)
   - Blocks access to:
     - RFC1918 private ranges: `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`
     - Carrier-grade NAT (CGNAT): `100.64.0.0/10`
     - Loopback: `127.0.0.0/8`
     - Link-local: `169.254.0.0/16`
     - IPv6 private ranges: `::1/128`, `fe80::/10`, `fc00::/7`
   - DNS resolution: Resolves hostnames and checks all returned IPs
   - Pattern matching: Catches `localhost`, `.internal`, `.lan` patterns even if DNS fails

3. **Strict DNS Mode** (`strict_dns`)
   - Default: `false` (lenient mode)
   - **Lenient mode** (`strict_dns: false`):
     - If DNS resolution fails, falls back to hostname pattern checks
     - Allows URLs with DNS failures if no suspicious patterns detected
     - Recommended for most users
   - **Strict mode** (`strict_dns: true`):
     - Rejects URLs when DNS resolution fails (if `allow_internal_ips: false`)
     - Provides maximum SSRF protection
     - May block legitimate URLs with temporary DNS issues

**Configuration Examples:**

```yaml
# Maximum security (recommended for production)
settings:
  allow_http: false         # HTTPS only
  allow_internal_ips: false # Block internal IPs
  strict_dns: true          # Reject on DNS failure

hooks:
  post_install:
    - action: http_webhook
      url: https://webhooks.example.com/notify
      body: '{"version": "{version}"}'
```

```yaml
# Development/testing (allow internal webhooks)
settings:
  allow_http: true          # Allow HTTP for local testing
  allow_internal_ips: true  # Allow internal IPs
  strict_dns: false         # Lenient DNS

hooks:
  post_install:
    - action: http_webhook
      url: http://localhost:8080/webhook
      body: '{"version": "{version}"}'
```

```yaml
# Balanced (public HTTPS webhooks with lenient DNS)
settings:
  allow_http: false         # HTTPS only
  allow_internal_ips: false # Block internal IPs
  strict_dns: false         # Allow DNS failures for legitimate services

hooks:
  post_install:
    - action: http_webhook
      url: https://hooks.slack.com/services/YOUR/WEBHOOK/URL
      body: '{"text": "Installed {version}"}'
```

**What Gets Blocked:**

```yaml
# ❌ Blocked by default (allow_internal_ips: false)
- http://localhost:3000/webhook
- https://192.168.1.10/api
- http://internal.company.local/notify
- https://10.0.0.5/webhook
- http://100.64.1.1/api          # CGNAT range
- https://api.internal/v1/hook

# ✅ Allowed by default
- https://hooks.slack.com/services/...
- https://discord.com/api/webhooks/...
- https://api.example.com/webhooks
- https://webhook.site/unique-url
```

**Security Notes:**

- **CGNAT ranges** (`100.64.0.0/10`) are blocked by default to prevent attacks via carrier-grade NAT networks
- DNS rebinding attacks are mitigated by resolving hostnames and checking all returned IPs
- If you need to send webhooks to internal services, explicitly set `allow_internal_ips: true` and understand the risks
- For maximum security in untrusted environments, enable `strict_dns: true`

**Slack Example:**

```yaml
hooks:
  post_install:
    - action: http_webhook
      params:
        url: https://hooks.slack.com/services/YOUR/WEBHOOK/URL
        body: |
          {
            "text": "🎉 Go {version} installed successfully",
            "username": "goenv-bot",
            "channel": "#dev-notifications"
          }
```

**Discord Example:**

```yaml
hooks:
  post_install:
    - action: http_webhook
      params:
        url: https://discord.com/api/webhooks/YOUR_WEBHOOK_URL
        body: |
          {
            "content": "✅ Go {version} installed at {timestamp}",
            "username": "goenv"
          }
```

**Custom API Example:**

```yaml
hooks:
  pre_exec:
    - action: http_webhook
      params:
        url: https://api.example.com/metrics/go-usage
        body: |
          {
            "event": "go_command_executed",
            "version": "{version}",
            "command": "{command}",
            "timestamp": "{timestamp}",
            "host": "{hostname}"
          }
```

---

### notify_desktop

Display native desktop notifications. Great for long-running operations.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `title` | string | Yes | Notification title (supports templates) |
| `message` | string | Yes | Notification body (supports templates) |

**Features:**
- Native system notifications (macOS, Windows, Linux)
- Template variable interpolation
- Non-intrusive (notifications timeout automatically)
- Works even when terminal is in background

**Platform Support:**
- **macOS:** Uses `osascript` (always available)
- **Linux:** Uses `notify-send` (requires libnotify)
- **Windows:** Uses PowerShell notifications

**Example:**

```yaml
hooks:
  post_install:
    - action: notify_desktop
      params:
        title: "goenv: Installation Complete"
        message: "Go {version} is ready to use"
  
  pre_install:
    - action: notify_desktop
      params:
        title: "goenv: Installing Go"
        message: "Installing Go {version}... This may take a few minutes."
```

**Notification Appearance:**

On macOS, notifications appear in the Notification Center:
```
┌─────────────────────────────────────┐
│ goenv: Installation Complete        │
│ Go 1.21.3 is ready to use          │
└─────────────────────────────────────┘
```

---

### check_disk_space

Verify sufficient disk space before operations. Prevents failed installations.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `min_space_mb` | number | Yes | Minimum required space in megabytes |
| `path` | string | No | Path to check (defaults to goenv root) |

**Features:**
- Checks available space on filesystem
- Configurable threshold in MB
- Cross-platform (Linux, macOS, Windows)
- Fails hook execution if insufficient space
- Useful for `pre_install` and `pre_rehash` hooks

**How It Works:**
1. Checks available space on the filesystem containing `path`
2. Compares to `min_space_mb` threshold
3. Returns error if insufficient space (stops hook execution)
4. Otherwise continues silently

**Example:**

```yaml
hooks:
  pre_install:
    - action: check_disk_space
      params:
        min_space_mb: 500
        path: ~/.goenv/versions
    
    - action: log_to_file
      params:
        path: ~/.goenv/install.log
        message: "Starting installation of Go {version}"
```

**Why Use This:**
- Go SDK downloads can be 100-200 MB
- Prevents partial installations from running out of space
- Provides clear error message before wasting time downloading
- Especially useful on systems with limited disk space

**Error Message Example:**
```
Error: Insufficient disk space at ~/.goenv/versions
Required: 500 MB, Available: 245 MB
```

---

### set_env

Set environment variables dynamically. Useful for version-specific configuration.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes | Environment variable name |
| `value` | string | Yes | Variable value (supports templates) |

**Features:**
- Sets environment variables in the current process
- Template variable interpolation
- Persists for the goenv command execution
- Great for `pre_exec` hooks to configure Go behavior

**Important Notes:**
- Variables are set in goenv's process and child processes
- Does NOT persist across shell sessions
- Does NOT modify shell configuration files
- Use for temporary, command-specific configuration

**Example:**

```yaml
hooks:
  pre_exec:
    - action: set_env
      params:
        name: GO_VERSION_INFO
        value: "go{version}"
    
    - action: set_env
      params:
        name: GOENV_COMMAND
        value: "{command}"
```

**Use Cases:**
- Set `GOPROXY` based on Go version
- Configure `GOPRIVATE` for specific projects
- Set build tags dynamically
- Pass version info to build scripts

**Advanced Example:**

```yaml
hooks:
  pre_exec:
    - action: set_env
      params:
        name: GOPROXY
        value: "https://proxy.golang.org,direct"
    
    - action: set_env
      params:
        name: GOENV_ACTIVE_VERSION
        value: "{version}"
    
    - action: log_to_file
      params:
        path: ~/.goenv/exec.log
        message: "Executing {command} with Go {version}"
```

---

### run_command

Execute shell commands during hook execution. Enables custom validation, automation, and integration.

**⚠️ SECURITY FIRST: Always Use Args Array**

The `args` array form is the **secure pattern** for running commands. It prevents shell injection by passing arguments directly without shell interpretation.

```yaml
# ✅ SECURE: Args array (recommended)
- action: run_command
  params:
    command: "go"
    args: ["build", "-o", "/tmp/app"]

# ⚠️ AVOID: Shell string (only for simple, trusted commands)
- action: run_command
  params:
    command: "go build -o /tmp/app"  # Shell interprets spaces, quotes, etc.
```

**Why args array matters:**
- **No shell injection**: Arguments can't contain shell metacharacters (`;`, `|`, `$()`, etc.)
- **Predictable behavior**: Spaces and quotes are treated as literals
- **Template safety**: Even template variables with special chars are safe
- **Cross-platform**: Works consistently on Windows, macOS, Linux

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `command` | string | Yes | Command to execute (supports templates) |
| `args` | array | No | Command arguments (each supports templates) |
| `working_dir` | string | No | Working directory for command execution |
| `timeout` | string | No | Command timeout (default: "2m") |
| `capture_output` | boolean | No | Capture stdout/stderr (default: false) |
| `log_output` | boolean | No | Log command output (default: false) |
| `fail_on_error` | boolean | No | Fail hook if command fails (default: true) |
| `shell` | string | No | Shell to use: auto, bash, sh, cmd, powershell (default: "auto") |

**Features:**
- Execute custom validation scripts
- Run platform-specific commands
- Capture and log output
- Template variable interpolation in command and args
- Cross-platform shell support (auto-detects OS)
- Configurable timeout and error handling
- Security: Protected against control characters, with timeout limits

**How It Works:**
1. Interpolates template variables in command and arguments
2. Selects appropriate shell based on `shell` parameter (or auto-detects)
3. Executes command with optional timeout
4. Optionally captures output to context variables
5. Returns error or continues based on `fail_on_error`

**Example 1: Basic Command with Args (Recommended Pattern)**

```yaml
hooks:
  post_install:
    - action: run_command
      params:
        command: "go"
        args: ["version"]
        capture_output: true
        log_output: true
```

**Example 2: Command with Multiple Arguments (Secure)**

```yaml
hooks:
  post_install:
    - action: run_command
      params:
        command: "go"
        args:
          - "build"
          - "-o"
          - "/tmp/test-{version}"
          - "."
        working_dir: "/tmp"
        timeout: "1m"
```

**Example 3: Template Variables in Args (Safe)**

```yaml
hooks:
  post_install:
    # Templates in args array are safe from injection
    - action: run_command
      params:
        command: "echo"
        args:
          - "Installed version {version} at {timestamp}"
        log_output: true
```

**Example 4: Shell Script (When Necessary)**

If you need shell features (pipes, redirects, env var expansion), use `bash -c`:

```yaml
hooks:
  post_install:
    - action: run_command
      params:
        command: "bash"
        args:
          - "-c"
          - "go version && go env GOROOT"
        capture_output: true
        timeout: "30s"
```

**Example 5: Simple Command String (Only for Trusted, Static Commands)**

```yaml
hooks:
  post_install:
    - action: run_command
      params:
        command: "go version"  # ⚠️ OK only if: no user input, no templates, fully trusted
        capture_output: true
```

**Security Anti-Patterns (DO NOT USE):**

```yaml
# ❌ INSECURE: Template in command string
- action: run_command
  params:
    command: "go install {package}@latest"  # {package} could contain shell metacharacters

# ❌ INSECURE: User input in command string
- action: run_command
  params:
    command: "git clone {repo_url}"  # {repo_url} could contain "; rm -rf /"

# ✅ SECURE: Use args array instead
- action: run_command
  params:
    command: "git"
    args: ["clone", "{repo_url}"]  # Safe: repo_url treated as literal string
```

**Example: Test Compilation**

```yaml
hooks:
  post_install:
    - action: run_command
      params:
        command: "go"
        args:
          - "build"
          - "-o"
          - "/tmp/test-go-{version}"
          - "-"
        working_dir: "/tmp"
        timeout: "1m"
        fail_on_error: true

    - action: log_to_file
      params:
        path: ~/.goenv/install.log
        message: "[{timestamp}] Go {version} - compilation test passed"
```

**Example: Platform-Specific Commands**

```yaml
hooks:
  post_install:
    # macOS: Add to Spotlight index
    - action: run_command
      params:
        command: "mdutil -i on ~/.goenv/versions/{version}"
        shell: "bash"
        fail_on_error: false

    # Windows: Register with Windows Search
    - action: run_command
      params:
        command: "attrib +I ~/.goenv/versions/{version}"
        shell: "cmd"
        fail_on_error: false
```

**Example: Integration with CI/CD**

```yaml
hooks:
  post_install:
    - action: run_command
      params:
        command: "curl"
        args:
          - "-X"
          - "POST"
          - "https://api.example.com/notify"
          - "-d"
          - "version={version}&timestamp={timestamp}"
        timeout: "10s"
        fail_on_error: false
```

**Captured Variables:**

When `capture_output: true` is set, the following variables are added to the hook context:
- `{command_stdout}` - Standard output from the command
- `{command_stderr}` - Standard error from the command
- `{command_exit_code}` - Exit code (currently "0" on success)

**Example: Using Captured Output**

```yaml
hooks:
  post_install:
    - action: run_command
      params:
        command: "go version"
        capture_output: true

    - action: log_to_file
      params:
        path: ~/.goenv/install.log
        message: "[{timestamp}] Installed: {command_stdout}"
```

**Security Considerations:**
- ⚠️ **Command execution is powerful** - Only use commands you trust
- ✅ Commands are validated for control characters
- ✅ Timeout prevents hanging processes (default 2 minutes)
- ✅ Path traversal protection on `working_dir`
- ✅ **Best practice:** Use `args` array for direct execution (bypasses shell)
- ✅ Variables are safe - they come from goenv (version, timestamp, etc.)
- ⚠️ **Protect hooks.yaml** - Treat as executable code, use file permissions (0600)
- ⚠️ **Review changes** - Always review hooks.yaml before committing to version control
- 💡 Consider setting `fail_on_error: false` for non-critical validations

**Shell Injection Prevention:**
```yaml
# ❌ AVOID: Shell interpretation (command as string)
- action: run_command
  params:
    command: "go build -o output-{version}"  # Shell interprets {version}

# ✅ PREFER: Direct execution (command + args)
- action: run_command
  params:
    command: "go"
    args: ["build", "-o", "output-{version}"]  # No shell, safer
```

**Use Cases:**
- Verify Go binary works after installation
- Run custom validation or compliance scripts
- Test compilation capabilities
- Integrate with external tools or APIs
- Execute platform-specific setup commands
- Trigger CI/CD pipelines or deployments

**Platform Notes:**
- **Auto shell detection:** Uses `sh` on Unix, `cmd` on Windows
- **Bash/sh:** Commands are executed as `bash -c "command"` or `sh -c "command"`
- **Windows cmd:** Commands are executed as `cmd /C "command"`
- **PowerShell:** Commands are executed as `powershell -Command "command"`
- **Direct execution:** When `args` are provided, command is executed directly without shell

---

## Configuration Options

### Global Settings

```yaml
enabled: true
acknowledged_risks: true

settings:
  timeout: "5s"
  continue_on_error: true
  allow_http: false
  allow_internal_ips: false
  strict_dns: false
```

**`enabled`** (boolean, required)
- Controls whether hooks are executed
- Set to `false` to disable all hooks without removing configuration
- Default: `false`

**`acknowledged_risks`** (boolean, required)
- Safety flag to confirm you've reviewed the configuration
- Both `enabled` and `acknowledged_risks` must be `true` for hooks to run
- Default: `false`

### Operational Limits

The hooks system includes built-in limits to prevent abuse and ensure reliable operation:

| Limit | Default | Maximum | Description |
|-------|---------|---------|-------------|
| **Action Timeout** | `5s` | `10m` | Maximum time for each action to complete |
| **Actions per Hook** | Unlimited | ~1000 | Number of actions per hook point (soft limit) |
| **Total Hooks** | Unlimited | ~50 | Number of configured hooks across all hook points |
| **Webhook Body Size** | 64KB | 1MB | Maximum size for HTTP webhook payloads |
| **Log File Size** | Unlimited | N/A | No size limit for log_to_file actions |

**Notes:**
- Timeouts are enforced per-action, not per-hook-point
- Multiple actions on the same hook point run sequentially
- Failed actions don't count against limits when `continue_on_error: true`
- Limits are designed to prevent misconfiguration, not malicious use

**Example configuration with custom limits:**

```yaml
settings:
  timeout: "10s"  # Increase for slow webhooks or commands
  continue_on_error: true  # Don't break goenv on hook failures

hooks:
  post_install:
    # Keep action lists reasonable (< 10 actions per hook is a good rule)
    - action: log_to_file
      params:
        path: ~/.goenv/install.log
        message: "Installed Go {version}"

    - action: http_webhook
      params:
        url: https://webhooks.example.com/notify
        body: '{"version": "{version}", "timestamp": "{timestamp}"}'
```

**`settings.timeout`** (string, default: "5s")
- Maximum time for each action to complete
- Valid units: `s` (seconds), `m` (minutes), `h` (hours)
- Examples: `"5s"`, `"30s"`, `"1m"`, `"2m30s"`
- Actions exceeding timeout are terminated
- Recommended: `5s` for webhooks, `10s` for commands, `30s` for disk checks

**`settings.continue_on_error`** (boolean, default: true)
- If `true`: Log errors but continue executing remaining hooks
- If `false`: Stop executing hooks on first error
- Recommended: `true` (hooks shouldn't break normal operations)

**`settings.allow_http`** (boolean, default: false)
- Controls whether HTTP URLs are allowed in `http_webhook` actions
- When `false`: Only HTTPS URLs accepted (recommended for production)
- When `true`: Both HTTP and HTTPS URLs allowed
- See [http_webhook Security Model](#security-model) for details

**`settings.allow_internal_ips`** (boolean, default: false)
- Controls SSRF protection for `http_webhook` actions
- When `false`: Blocks requests to private/internal IP ranges (RFC1918, CGNAT, loopback, link-local)
- When `true`: Allows requests to any IP address
- Protects against Server-Side Request Forgery (SSRF) attacks
- See [http_webhook Security Model](#security-model) for complete list of blocked ranges

**`settings.strict_dns`** (boolean, default: false)
-- Enhanced DNS validation for `http_webhook` actions
-- When `true`: More strict DNS checking to prevent DNS rebinding attacks
-- When `false`: Standard DNS resolution
-- Recommended for high-security environments

### Operational Limits

To prevent abuse and ensure system stability, the following limits are enforced:

| Limit | Default | Description |
|-------|---------|-------------|
| **Max Actions per Hook** | 10 | Maximum number of actions that can be defined for a single hook point |
| **Default Timeout** | 5s | Default timeout for each action if not specified |
| **Max Timeout** | 10m | Maximum timeout value allowed for any action |
| **Command Timeout** | 2m | Default timeout for `run_command` actions |
| **Webhook Timeout** | 10s | Maximum timeout for `http_webhook` requests |
| **Max Retries** | 3 | Maximum retry attempts for failed actions (if retry is enabled) |

**Notes:**
- Timeouts are enforced per-action, not per-hook
- Actions exceeding timeout are terminated gracefully
- `continue_on_error: true` (default) ensures one failing action doesn't block others
- Limits prevent runaway hooks from degrading system performance
- See [http_webhook Security Model](#security-model) for detailed behavior

### Hook Configuration

Each hook point can have multiple actions:

```yaml
hooks:
  post_install:
    - action: log_to_file
      params:
        path: ~/.goenv/install.log
        message: "Installed {version}"
    
    - action: notify_desktop
      params:
        title: "Installation Complete"
        message: "Go {version} ready"
```

**Execution Order:**
- Actions execute in the order defined
- If `continue_on_error: false`, execution stops on first error
- If `continue_on_error: true`, all actions attempt to execute

## Template Variables

All action parameters support template variable interpolation using `{variable}` syntax.

### Available Variables

**Always Available:**
- `{hook}` - Current hook point name (e.g., "post_install")
- `{timestamp}` - ISO 8601 timestamp (e.g., "2025-10-16T10:30:45Z")

**Installation/Uninstallation Hooks:**
- `{version}` - Go version being installed/uninstalled (e.g., "1.21.3")

**Execution Hooks:**
- `{version}` - Active Go version (e.g., "1.21.3")
- `{command}` - Command being executed (e.g., "go build")

**Rehash Hooks:**
- (No additional variables beyond hook and timestamp)

### Template Examples

```yaml
# Simple variable interpolation
message: "Installed Go {version}"
# Output: "Installed Go 1.21.3"

# Multiple variables
message: "[{timestamp}] {hook}: Go {version}"
# Output: "[2025-10-16T10:30:45Z] post_install: Go 1.21.3"

# JSON with templates
body: |
  {
    "version": "{version}",
    "hook": "{hook}",
    "time": "{timestamp}"
  }

# Complex message
message: "User installed Go {version} at {timestamp} via {hook} hook"
```

### Template Escaping

To use literal braces, double them:

```yaml
message: "This is not a variable: {{version}}"
# Output: "This is not a variable: {version}"
```

## Common Use Cases

### 1. Installation Audit Trail

Track all Go version installations with timestamps and details:

```yaml
hooks:
  post_install:
    - action: log_to_file
      params:
        path: ~/.goenv/audit/installations.log
        message: "[{timestamp}] INSTALL: Go {version}"
    
    - action: notify_desktop
      params:
        title: "goenv: Installation Complete"
        message: "Go {version} is now available"

  post_uninstall:
    - action: log_to_file
      params:
        path: ~/.goenv/audit/installations.log
        message: "[{timestamp}] UNINSTALL: Go {version}"
```

### 2. Team Notifications

Notify your team when Go versions are installed:

```yaml
hooks:
  post_install:
    - action: http_webhook
      params:
        url: https://hooks.slack.com/services/YOUR/WEBHOOK/URL
        body: |
          {
            "text": "🎉 Go {version} installed on development server",
            "channel": "#dev-notifications",
            "username": "goenv-bot"
          }
```

### 3. Pre-Installation Checks

Ensure sufficient resources before installing:

```yaml
hooks:
  pre_install:
    - action: check_disk_space
      params:
        min_space_mb: 500
        path: ~/.goenv/versions
    
    - action: notify_desktop
      params:
        title: "goenv: Starting Installation"
        message: "Installing Go {version}... Please wait."
    
    - action: log_to_file
      params:
        path: ~/.goenv/install.log
        message: "[{timestamp}] Starting installation of Go {version}"
```

### 4. Command Usage Tracking

Log all Go commands for analytics:

```yaml
hooks:
  pre_exec:
    - action: log_to_file
      params:
        path: ~/.goenv/logs/usage.log
        message: "[{timestamp}] v{version}: {command}"
  
  post_exec:
    - action: http_webhook
      params:
        url: https://api.example.com/metrics
        body: |
          {
            "event": "go_command_executed",
            "version": "{version}",
            "command": "{command}",
            "timestamp": "{timestamp}"
          }
```

### 5. Environment Configuration

Set version-specific environment variables:

```yaml
hooks:
  pre_exec:
    - action: set_env
      params:
        name: GOENV_ACTIVE_VERSION
        value: "{version}"
    
    - action: set_env
      params:
        name: GOPROXY
        value: "https://proxy.golang.org,direct"
```

### 6. Development Workflow

Complete workflow with checks, logs, and notifications:

```yaml
enabled: true
acknowledged_risks: true

settings:
  timeout: "10s"
  continue_on_error: true

hooks:
  pre_install:
    - action: check_disk_space
      params:
        min_space_mb: 500

    - action: notify_desktop
      params:
        title: "goenv"
        message: "Installing Go {version}..."

  post_install:
    - action: log_to_file
      params:
        path: ~/.goenv/install.log
        message: "[{timestamp}] ✓ Installed Go {version}"

    - action: notify_desktop
      params:
        title: "goenv: Installation Complete"
        message: "Go {version} is ready to use"

    - action: http_webhook
      params:
        url: https://hooks.slack.com/services/YOUR/WEBHOOK
        body: |
          {
            "text": "✅ Go {version} installed successfully",
            "channel": "#dev-team"
          }

  pre_exec:
    - action: set_env
      params:
        name: GOENV_VERSION
        value: "{version}"
```

### 7. Installation Validation

Verify Go installations work correctly using `run_command`:

```yaml
hooks:
  post_install:
    # Test 1: Verify go version command works
    - action: run_command
      params:
        command: "go"
        args: ["version"]
        capture_output: true
        fail_on_error: true
        timeout: "10s"

    # Test 2: Verify go env works
    - action: run_command
      params:
        command: "go"
        args: ["env", "GOROOT"]
        capture_output: true
        fail_on_error: true

    # Test 3: Simple compilation test (optional)
    - action: run_command
      params:
        command: "go"
        args: ["run", "-"]
        working_dir: "/tmp"
        timeout: "30s"
        fail_on_error: false  # Don't fail install if test fails
        shell: "bash"

    # Log validation results
    - action: log_to_file
      params:
        path: ~/.goenv/install.log
        message: "[{timestamp}] Go {version} validated: {command_stdout}"
```

### 8. Configuration File Management

Copy team configuration files after installation:

```yaml
hooks:
  post_install:
    # Copy team's gopls configuration
    - action: run_command
      params:
        command: "cp"
        args:
          - "-f"
          - "~/.goenv/templates/gopls.json"
          - "~/go/{version}/gopls.json"
        fail_on_error: false

    # Copy golangci-lint configuration
    - action: run_command
      params:
        command: "cp"
        args:
          - "-f"
          - "~/.goenv/templates/.golangci.yml"
          - "~/go/{version}/.golangci.yml"
        fail_on_error: false

    # Create standard project structure
    - action: run_command
      params:
        command: "mkdir"
        args: ["-p", "~/go/{version}/bin", "~/go/{version}/src", "~/go/{version}/pkg"]
        fail_on_error: false
```

### 9. Backup Before Uninstall

Archive Go version before removing it:

```yaml
hooks:
  pre_uninstall:
    # Create backup directory
    - action: run_command
      params:
        command: "mkdir"
        args: ["-p", "~/.goenv/backups"]
        fail_on_error: false

    # Archive the version
    - action: run_command
      params:
        command: "tar"
        args:
          - "-czf"
          - "~/.goenv/backups/go-{version}-{timestamp}.tar.gz"
          - "-C"
          - "~/.goenv/versions"
          - "{version}"
        timeout: "5m"
        fail_on_error: false

    - action: log_to_file
      params:
        path: ~/.goenv/backups/uninstall.log
        message: "[{timestamp}] Archived Go {version} before removal"
```

### 10. Comprehensive Installation Validation

Multi-stage validation with detailed logging:

```yaml
hooks:
  post_install:
    # Stage 1: Basic binary check
    - action: run_command
      params:
        command: "go"
        args: ["version"]
        capture_output: true
        log_output: true
        fail_on_error: true

    # Stage 2: Environment verification
    - action: run_command
      params:
        command: "sh"
        args:
          - "-c"
          - "go env GOROOT && go env GOCACHE && go env GOMODCACHE"
        capture_output: true
        log_output: true
        fail_on_error: true

    # Stage 3: Standard library check
    - action: run_command
      params:
        command: "go"
        args: ["list", "std"]
        capture_output: true
        timeout: "30s"
        fail_on_error: true

    # Stage 4: Simple compilation test
    - action: run_command
      params:
        command: "sh"
        args:
          - "-c"
          - "echo 'package main\nfunc main() {}' | go build -o /tmp/goenv-test-{version} -x -"
        timeout: "1m"
        fail_on_error: false  # Non-critical

    # Log comprehensive validation success
    - action: log_to_file
      params:
        path: ~/.goenv/install.log
        message: "[{timestamp}] Go {version} fully validated - all checks passed"

    - action: notify_desktop
      params:
        title: "goenv: Validation Complete"
        message: "Go {version} installed and validated successfully"
```

### 11. Cross-Platform File Operations

Handle file operations across different operating systems:

```yaml
hooks:
  post_install:
    # Unix/macOS: Use cp and mkdir
    - action: run_command
      params:
        command: "sh"
        args:
          - "-c"
          - "mkdir -p ~/go/{version}/config && cp ~/.goenv/templates/* ~/go/{version}/config/"
        shell: "bash"
        fail_on_error: false

    # Windows: Use PowerShell
    - action: run_command
      params:
        command: "powershell"
        args:
          - "-Command"
          - "New-Item -ItemType Directory -Force -Path ~/go/{version}/config; Copy-Item ~/.goenv/templates/* ~/go/{version}/config/"
        shell: "powershell"
        fail_on_error: false
```

### 12. Tool Installation After Go Install

Automatically install commonly used Go tools:

```yaml
hooks:
  post_install:
    # Install gopls (Go language server)
    - action: run_command
      params:
        command: "go"
        args: ["install", "golang.org/x/tools/gopls@latest"]
        timeout: "5m"
        fail_on_error: false

    # Install golangci-lint
    - action: run_command
      params:
        command: "sh"
        args:
          - "-c"
          - "curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ~/go/{version}/bin"
        timeout: "3m"
        fail_on_error: false

    # Install delve debugger
    - action: run_command
      params:
        command: "go"
        args: ["install", "github.com/go-delve/delve/cmd/dlv@latest"]
        timeout: "3m"
        fail_on_error: false

    - action: log_to_file
      params:
        path: ~/.goenv/install.log
        message: "[{timestamp}] Go {version} - development tools installed"
```

## Advanced Patterns

### Conditional Execution Based on Go Version

Execute actions only for specific Go versions:

```yaml
hooks:
  post_install:
    # For Go 1.21+ only: Enable new features
    - action: run_command
      params:
        command: "sh"
        args:
          - "-c"
          - "if [ \"$(go version | grep -oE '[0-9]+\\.[0-9]+' | head -1)\" \\> \"1.20\" ]; then echo 'Go 1.21+ detected'; fi"
        capture_output: true
        fail_on_error: false
```

### Integration with Version Control

Commit installation records to Git:

```yaml
hooks:
  post_install:
    - action: run_command
      params:
        command: "sh"
        args:
          - "-c"
          - "cd ~/.goenv && git add install.log && git commit -m 'Installed Go {version} at {timestamp}' || true"
        fail_on_error: false
```

### Health Check Script

Run custom health check scripts:

```yaml
hooks:
  post_install:
    - action: run_command
      params:
        command: "bash"
        args: ["~/.goenv/scripts/health-check.sh", "{version}"]
        timeout: "2m"
        capture_output: true
        log_output: true
        fail_on_error: true
```

Example `~/.goenv/scripts/health-check.sh`:
```bash
#!/bin/bash
VERSION=$1

echo "Running health checks for Go $VERSION..."

# Check 1: Binary exists and is executable
if ! command -v go &> /dev/null; then
    echo "ERROR: go binary not found"
    exit 1
fi

# Check 2: Version matches
ACTUAL_VERSION=$(go version | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
if [[ ! "$ACTUAL_VERSION" =~ "$VERSION" ]]; then
    echo "ERROR: Version mismatch. Expected $VERSION, got $ACTUAL_VERSION"
    exit 1
fi

# Check 3: Can compile
echo 'package main; func main() {}' | go build -o /tmp/test-$$ -x - 2>&1
if [ $? -ne 0 ]; then
    echo "ERROR: Compilation test failed"
    exit 1
fi
rm -f /tmp/test-$$

echo "All health checks passed for Go $VERSION"
exit 0
```

## Best Practices

### Security

1. **Review Generated Configuration:** Always review `hooks.yaml` before enabling
2. **Use HTTPS:** For webhooks, always use HTTPS URLs
3. **Protect Webhook URLs:** Treat webhook URLs as secrets (don't commit to public repos)
4. **Limit Actions:** Only use actions you need
5. **Test First:** Use `goenv hooks test` before enabling hooks

### Performance

1. **Keep Timeouts Short:** Default 5s is usually sufficient
2. **Use `continue_on_error: true`:** Prevent hooks from breaking normal operations
3. **Avoid Blocking Operations:** Webhooks and notifications are designed to be fast
4. **Log to Fast Storage:** Use local SSDs for log files, not network drives

### Configuration Management

1. **Version Control:** Keep `hooks.yaml` in version control (without secrets)
2. **Document Custom Hooks:** Add comments explaining why each hook exists
3. **Use Templates:** Leverage variables for flexibility
4. **Test Regularly:** Use `goenv hooks validate` and `goenv hooks test`

### Debugging

1. **Start Simple:** Begin with one log_to_file hook
2. **Check Logs:** Look for hook execution errors in output
3. **Test Incrementally:** Add one action at a time
4. **Validate Often:** Run `goenv hooks validate` after changes

### Example: Well-Documented Configuration

```yaml
# goenv hooks configuration
# Purpose: Track installations and notify team
# Last updated: 2025-10-16

enabled: true
acknowledged_risks: true

settings:
  # 10 second timeout for webhook latency
  timeout: "10s"
  # Don't break goenv if hooks fail
  continue_on_error: true

hooks:
  # Track all installations for compliance
  post_install:
    - action: log_to_file
      params:
        path: ~/.goenv/audit/install.log
        message: "[{timestamp}] Installed Go {version}"
    
    # Notify team in Slack #dev-notifications
    - action: http_webhook
      params:
        url: https://hooks.slack.com/services/YOUR/WEBHOOK
        body: '{"text": "Go {version} installed on dev server"}'
  
  # Ensure we don't run out of disk space
  pre_install:
    - action: check_disk_space
      params:
        min_space_mb: 500
```

## Troubleshooting

### Hooks Not Executing

**Problem:** Hooks aren't running when expected.

**Solutions:**
1. Check `enabled: true` in `hooks.yaml`
2. Check `acknowledged_risks: true` in `hooks.yaml`
3. Verify `hooks.yaml` is in the correct location (`~/.goenv/hooks.yaml`)
4. Run `goenv hooks validate` to check for errors
5. Check for YAML syntax errors

### Validation Errors

**Problem:** `goenv hooks validate` reports errors.

**Solutions:**
1. Check YAML syntax (indentation, quotes, colons)
2. Verify all required parameters are present
3. Check parameter types (strings vs numbers)
4. Ensure hook point names are valid
5. Verify action names are spelled correctly

### Action Failures

**Problem:** Specific actions fail during execution.

**Solutions:**

**log_to_file:**
- Ensure parent directory exists or can be created
- Check file permissions
- Verify path is valid for your OS

**http_webhook:**
- Test URL in a browser or with `curl`
- Verify JSON syntax in body
- Check network connectivity
- Ensure URL uses HTTP or HTTPS

**notify_desktop:**
- **Linux:** Install `libnotify` (`sudo apt install libnotify-bin`)
- **macOS:** Should work out of the box
- **Windows:** PowerShell must be available

**check_disk_space:**
- Verify `path` exists
- Check `min_space_mb` is a number, not a string
- Ensure sufficient permissions to check disk space

**set_env:**
- Verify `name` is a valid environment variable name
- Check for typos in variable names

### Timeout Errors

**Problem:** Actions timing out.

**Solutions:**
1. Increase `settings.timeout` value
2. Check network latency for webhooks
3. Verify external services are responsive
4. Consider if the action is appropriate for hooks

### Template Variables Not Working

**Problem:** Variables show as literal `{variable}` instead of values.

**Solutions:**
1. Verify variable names are correct (case-sensitive)
2. Ensure variable is available for that hook point
3. Check for typos in variable names
4. Use `goenv hooks test <hookpoint>` to see variable values

### Getting Help

If you're still having issues:

1. **Check Configuration:**
   ```bash
   goenv hooks validate
   goenv hooks list
   ```

2. **Test Hooks:**
   ```bash
   goenv hooks test post_install
   ```

3. **Enable Verbose Output:**
   Look for hook-related error messages in goenv command output

4. **Simplify Configuration:**
   Remove all hooks except one log_to_file action to isolate the issue

5. **Check File Permissions:**
   Ensure `~/.goenv/hooks.yaml` is readable

6. **Report Issues:**
   If you find a bug, report it on the goenv GitHub repository

## Complete Example Configuration

Here's a comprehensive example using all features:

```yaml
# ~/.goenv/hooks.yaml
# Complete example configuration

enabled: true
acknowledged_risks: true

settings:
  timeout: "10s"
  continue_on_error: true

hooks:
  # Before installation
  pre_install:
    - action: check_disk_space
      params:
        min_space_mb: 500
        path: ~/.goenv/versions
    
    - action: notify_desktop
      params:
        title: "goenv"
        message: "Installing Go {version}..."
    
    - action: log_to_file
      params:
        path: ~/.goenv/logs/install.log
        message: "[{timestamp}] PRE_INSTALL: Go {version}"

  # After installation
  post_install:
    - action: log_to_file
      params:
        path: ~/.goenv/logs/install.log
        message: "[{timestamp}] POST_INSTALL: Go {version} - SUCCESS"
    
    - action: notify_desktop
      params:
        title: "goenv: Installation Complete"
        message: "Go {version} is ready to use"
    
    - action: http_webhook
      params:
        url: https://hooks.slack.com/services/YOUR/WEBHOOK/URL
        body: |
          {
            "text": "✅ Go {version} installed successfully",
            "username": "goenv-bot",
            "channel": "#dev-notifications"
          }

  # Before uninstallation
  pre_uninstall:
    - action: log_to_file
      params:
        path: ~/.goenv/logs/install.log
        message: "[{timestamp}] PRE_UNINSTALL: Go {version}"

  # After uninstallation
  post_uninstall:
    - action: log_to_file
      params:
        path: ~/.goenv/logs/install.log
        message: "[{timestamp}] POST_UNINSTALL: Go {version} - REMOVED"
    
    - action: notify_desktop
      params:
        title: "goenv: Uninstallation Complete"
        message: "Go {version} has been removed"

  # Before command execution
  pre_exec:
    - action: set_env
      params:
        name: GOENV_VERSION
        value: "{version}"
    
    - action: log_to_file
      params:
        path: ~/.goenv/logs/commands.log
        message: "[{timestamp}] EXEC: {command} (Go {version})"

  # After command execution
  post_exec:
    - action: http_webhook
      params:
        url: https://api.example.com/metrics/go-usage
        body: |
          {
            "event": "go_command",
            "version": "{version}",
            "command": "{command}",
            "timestamp": "{timestamp}"
          }

  # Before rehash
  pre_rehash:
    - action: log_to_file
      params:
        path: ~/.goenv/logs/rehash.log
        message: "[{timestamp}] PRE_REHASH"

  # After rehash
  post_rehash:
    - action: log_to_file
      params:
        path: ~/.goenv/logs/rehash.log
        message: "[{timestamp}] POST_REHASH - Shims updated"
    
    - action: notify_desktop
      params:
        title: "goenv"
        message: "Shims regenerated successfully"
```

---

## Next Steps

1. **Generate Configuration:**
   ```bash
   goenv hooks init
   ```

2. **Review and Edit:**
   ```bash
   nano ~/.goenv/hooks.yaml
   ```

3. **Validate:**
   ```bash
   goenv hooks validate
   ```

4. **Test:**
   ```bash
   goenv hooks test post_install
   ```

5. **Enable and Use:**
   Set `enabled: true` and `acknowledged_risks: true`, then use goenv normally!

For more information, see the [goenv documentation](https://github.com/go-nv/goenv).
