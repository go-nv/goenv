# Hooks System Documentation

## Table of Contents

- [Overview](#overview)
- [Security Model](#security-model)
- [Operational Limits](#operational-limits)
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

## Security Model

The goenv hooks system is designed with security as a priority. All actions have built-in protections, and security controls are **enabled by default**.

### Safe by Default

**SSRF Protection (Server-Side Request Forgery):**

The `http_webhook` action includes comprehensive SSRF protection that is **enabled by default**:

```yaml
settings:
  allow_http: false # ✓ Default: HTTPS only, HTTP blocked
  allow_internal_ips: false # ✓ Default: Internal IPs blocked
  strict_dns: false # ✓ Default: Lenient DNS validation
```

**What's Protected by Default:**

- ❌ HTTP URLs blocked (HTTPS required)
- ❌ Private IP ranges blocked (RFC1918: `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`)
- ❌ Loopback addresses blocked (`127.0.0.0/8`, `::1`)
- ❌ Link-local addresses blocked (`169.254.0.0/16`, `fe80::/10`)
- ❌ Carrier-grade NAT blocked (`100.64.0.0/10`)
- ❌ DNS resolution checks all IPs for private ranges
- ❌ Pattern matching catches `.internal`, `.local`, `localhost` even if DNS fails

**Example - Blocked by Default:**

```yaml
# ❌ These will be rejected with default settings
hooks:
  post_install:
    - action: http_webhook
      params:
        url: http://localhost:3000/api        # HTTP blocked
        url: https://192.168.1.10/webhook     # Private IP blocked
        url: https://internal.corp/api        # Internal domain blocked
        url: http://100.64.1.1/webhook        # CGNAT blocked
```

**Example - Allowed by Default:**

```yaml
# ✅ These work with default settings (HTTPS + public IPs)
hooks:
  post_install:
    - action: http_webhook
      params:
        url: https://hooks.slack.com/services/YOUR/WEBHOOK/URL
        body: '{"text": "Installed Go {version}"}'
```

### When to Override Defaults

Only override security defaults when you have a specific, trusted use case:

**Development/Testing - Allow HTTP and Internal IPs:**

```yaml
# ⚠️ Use only in development - NOT for production
settings:
  allow_http: true # Allow HTTP for local testing
  allow_internal_ips: true # Allow internal network webhooks
  strict_dns: false # Keep lenient for local DNS

hooks:
  post_install:
    - action: http_webhook
      params:
        url: http://localhost:8080/webhook
        body: '{"version": "{version}"}'
```

**Production - Maximum Security with Strict DNS:**

```yaml
# 🔒 Maximum security for untrusted environments
settings:
  allow_http: false # HTTPS only (default)
  allow_internal_ips: false # Block internal IPs (default)
  strict_dns: true # Reject URLs when DNS fails (strict)

hooks:
  post_install:
    - action: http_webhook
      params:
        url: https://webhook.site/unique-url
        body: '{"version": "{version}"}'
```

**Internal Network - Controlled Environment:**

```yaml
# ⚠️ Use only when targeting trusted internal services
settings:
  allow_http: false # Keep HTTPS only
  allow_internal_ips: true # Allow internal IPs (REQUIRED for internal webhooks)
  strict_dns: false # Lenient DNS

hooks:
  post_install:
    - action: http_webhook
      params:
        url: https://webhook.internal.company.com/api
        body: '{"version": "{version}"}'
```

### Understanding strict_dns

The `strict_dns` setting controls behavior when DNS resolution fails:

**Lenient Mode (Default: `strict_dns: false`):**

- If DNS fails, falls back to hostname pattern matching
- Blocks obvious patterns (`localhost`, `.internal`, `.local`)
- Allows legitimate URLs with temporary DNS issues
- **Recommended for most users**

**Strict Mode (`strict_dns: true`):**

- If DNS fails and `allow_internal_ips: false`, the URL is rejected
- Provides maximum SSRF protection
- May block legitimate webhooks during DNS outages
- **Recommended for:**
  - Untrusted environments
  - High-security deployments
  - When you can't tolerate any DNS resolution failures

**Example - When strict_dns Helps:**

```yaml
# Without strict_dns (default):
# - If DNS is down, "https://api.example.com" might be allowed
# - Pattern matching provides defense in depth

# With strict_dns: true
# - If DNS fails for "https://api.example.com", it's rejected
# - Prevents potential DNS rebinding attacks
# - Trade-off: legitimate URLs may be blocked during DNS issues

settings:
  strict_dns: true # Reject when DNS resolution fails

hooks:
  post_install:
    - action: http_webhook
      params:
        url: https://hooks.slack.com/services/YOUR/WEBHOOK/URL
        body: '{"text": "Installed {version}"}'
```

### Security Audit Checklist

Before enabling hooks in production, audit your configuration:

**1. Check HTTPS Enforcement:**

```bash
# Search for HTTP URLs in your configuration
grep -i "http://" ~/.goenv/hooks.yaml
# Should return nothing (or only commented examples)
```

**2. Review allow_internal_ips Setting:**

```yaml
# Verify setting matches your intent
settings:
  allow_internal_ips: false # ✓ Should be false unless you need internal webhooks
```

**3. Consider Strict DNS for Production:**

```yaml
# For maximum security
settings:
  strict_dns: true # Consider enabling for untrusted environments
```

**4. Audit Webhook URLs:**

```bash
# List all webhook URLs
grep -A 2 "http_webhook" ~/.goenv/hooks.yaml | grep "url:"
# Verify all URLs are:
# - HTTPS (not HTTP)
# - Public services (not internal IPs)
# - Trusted endpoints
```

**5. Test Configuration:**

```bash
# Validate hooks configuration
goenv hooks validate

# Test hooks without executing (shows what would run)
goenv hooks test post_install
```

### Additional Security Controls

**Command Execution Safety:**

- `run_command` uses args array (not shell evaluation)
- Control character validation on all string inputs
- Timeout protection (default 5s, configurable)
- No shell injection vulnerabilities

**File Operations:**

- Path traversal validation
- Restricted to user-writable locations
- Control character filtering in filenames

**Environment Variables:**

- Key/value validation
- No control characters allowed
- Scoped to hook execution context only

### Security Resources

- **SSRF Protection Details:** See [http_webhook Security Model](#security-model-1) section
- **Command Safety:** See [run_command](#run_command) documentation
- **Best Practices:** See [Best Practices - Security](#security-1) section

## Operational Limits

The hooks system has built-in limits and timeouts to prevent abuse, misconfiguration, and performance degradation.

### Default Limits

All limits are configurable but have sensible defaults:

| **Setting**         | **Default** | **Valid Range** | **Purpose**                            | **Location**                  |
| ------------------- | ----------- | --------------- | -------------------------------------- | ----------------------------- |
| `timeout`           | `5s`        | Any duration    | Maximum execution time per action      | `internal/hooks/config.go:70` |
| `max_actions`       | `10`        | `1` - `100`     | Maximum actions per hook point         | `internal/hooks/config.go:71` |
| `continue_on_error` | `true`      | boolean         | Whether to continue on action failures | `internal/hooks/config.go:73` |

**Implementation details:**

- **Per-action timeout** (`internal/hooks/executor.go:76-91`): Each action gets its own timeout, not shared across the hook point
- **Max actions validation** (`internal/hooks/config.go:187-190`): Enforced at config load time per hook point
- **Continue on error** (`internal/hooks/executor.go:50-56`): Logs errors but continues execution when enabled

### How Limits Work

**Timeout Behavior:**

```yaml
settings:
  timeout: "5s" # Each action gets 5 seconds max

hooks:
  post_install:
    - action: http_webhook # Gets 5s timeout
      params:
        url: https://slow.webhook.com/api
    - action: run_command # Gets separate 5s timeout
      params:
        command: sleep
        args: ["10"] # ❌ Will timeout after 5s
```

When an action times out:

- Error message: `action timed out after 5s`
- If `continue_on_error: true` (default): Next action executes
- If `continue_on_error: false`: Execution stops, goenv command fails

**Max Actions Limit:**

```yaml
settings:
  max_actions: 10 # Hard limit per hook point

hooks:
  post_install:
    # ✅ OK: 3 actions (under limit)
    - action: log_to_file
      params: { path: "/tmp/install.log", message: "Start" }
    - action: http_webhook
      params: { url: "https://api.example.com/webhook" }
    - action: notify_desktop
      params: { title: "Done", message: "Go {version} installed" }

  pre_install:
    # ❌ ERROR: 11 actions (exceeds limit)
    # Config validation will fail at load time
```

Validation error if exceeded:

```
Error: hook point post_install has 11 actions, max is 10
```

**Continue on Error:**

```yaml
settings:
  continue_on_error: true # Default: don't break goenv

hooks:
  post_install:
    - action: http_webhook
      params: { url: "https://flaky.service.com/webhook" }
      # ⚠️ Fails with network error

    - action: log_to_file
      params: { path: "/tmp/install.log", message: "Installed" }
      # ✅ Still executes (continue_on_error: true)
```

With `continue_on_error: false`:

```yaml
settings:
  continue_on_error: false # Stop on first error

hooks:
  post_install:
    - action: http_webhook
      params: { url: "https://broken.service.com/webhook" }
      # ❌ Fails

    - action: log_to_file
      # ❌ Never executes (stopped after first failure)
      # ❌ goenv install command fails
```

### Recommended Limits by Use Case

**Development/Testing:**

```yaml
settings:
  timeout: "10s" # Generous for debugging
  max_actions: 20 # Allow experimentation
  continue_on_error: true # Don't break workflow
```

**Production/CI:**

```yaml
settings:
  timeout: "5s" # Fast fail for reliability
  max_actions: 5 # Keep it simple
  continue_on_error: true # Never break builds
```

**Air-gapped/Offline:**

```yaml
settings:
  timeout: "30s" # Slow local systems
  max_actions: 3 # Minimal hooks
  continue_on_error: true # Resilience
```

### Preventing Abuse

The limits prevent common misconfigurations:

| **Problem**                | **Limit**                 | **Prevention**                            |
| -------------------------- | ------------------------- | ----------------------------------------- |
| Runaway webhook loops      | `timeout: 5s`             | Terminates slow/hanging webhooks          |
| Excessive action chains    | `max_actions: 10`         | Prevents accidentally complex hook chains |
| Hanging commands           | `timeout: 5s`             | Kills long-running commands               |
| Breaking normal operations | `continue_on_error: true` | Logs errors but doesn't fail goenv        |

**Example - Preventing runaway hooks:**

```yaml
# ❌ Bad: This would hang forever without timeout
hooks:
  post_install:
    - action: run_command
      params:
        command: tail
        args: ["-f", "/var/log/system.log"]  # Infinite stream
        # ✅ Timeout kills it after 5s

# ❌ Bad: This creates excessive work
hooks:
  post_install:
    # 50 webhook actions would be rejected
    # max_actions: 10 limit prevents this at config load
```

### Configuration Example

Complete example with all limits configured:

```yaml
version: 1
enabled: true
acknowledged_risks: true

settings:
  timeout: "10s" # Per-action timeout
  max_actions: 15 # Per hook point limit
  continue_on_error: true # Resilient execution
  log_file: "~/.goenv/hooks.log"

hooks:
  post_install:
    - action: log_to_file
      params:
        path: "~/.goenv/install-audit.log"
        message: "{timestamp} - Installed Go {version}"

    - action: http_webhook
      params:
        url: "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
        body: '{"text": "Installed Go {version}"}'
        headers:
          Content-Type: "application/json"
```

See [Configuration Options](#configuration-options) for complete settings documentation.

## Getting Started

> **🔒 Security First:** The hooks system is designed with security by default. HTTPS is enforced, internal IPs are blocked, and all actions have built-in protections. See the [Security Model](#security-model) section for details.

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

**Security Note:** The default configuration uses safe settings:

- `allow_http: false` - HTTPS only (recommended)
- `allow_internal_ips: false` - Blocks internal IPs (recommended)
- `strict_dns: false` - Lenient DNS validation (reasonable default)

Only override these settings if you have a specific, trusted use case (see [Security Model](#security-model)).

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

## Platform Requirements

Some hook actions have platform-specific dependencies or behaviors. This section documents what's needed on each platform.

### Desktop Notifications (`notify_desktop`)

Desktop notifications work on all platforms but require different system components:

| Platform    | Tool                      | Installation                                                                                                         | Notes                              |
| ----------- | ------------------------- | -------------------------------------------------------------------------------------------------------------------- | ---------------------------------- |
| **Linux**   | libnotify (`notify-send`) | `apt install libnotify-bin` (Debian/Ubuntu)<br>`dnf install libnotify` (Fedora/RHEL)<br>`pacman -S libnotify` (Arch) | Required for desktop notifications |
| **macOS**   | osascript                 | ✅ Built-in (no installation needed)                                                                                 | Uses AppleScript for notifications |
| **Windows** | PowerShell                | ✅ Built-in (no installation needed)                                                                                 | Uses Windows Toast notifications   |

**Verify Installation:**

```bash
# Linux
which notify-send && echo "✅ notify-send available" || echo "❌ Install libnotify"

# macOS
which osascript && echo "✅ osascript available"

# Windows (PowerShell)
Get-Command New-BurntToastNotification -ErrorAction SilentlyContinue
```

**Example Configuration:**

```yaml
hooks:
  post_install:
    - action: notify_desktop
      params:
        title: "Go Installation Complete"
        message: "Go {version} is now installed"
        # Works on all platforms - goenv handles the differences
```

### Command Execution (`run_command`)

The `run_command` action detects and uses the appropriate shell for each platform:

| Platform    | Shells              | Default Shell      | Notes                                            |
| ----------- | ------------------- | ------------------ | ------------------------------------------------ |
| **Linux**   | bash, sh, zsh, fish | bash               | Auto-detected from `$SHELL` environment variable |
| **macOS**   | bash, zsh, fish     | zsh (macOS 10.15+) | Auto-detected from `$SHELL` environment variable |
| **Windows** | cmd, PowerShell     | PowerShell         | Auto-detected, PowerShell preferred              |

**Cross-Platform Best Practice:**

Use the `args` array instead of shell-specific syntax for maximum compatibility:

```yaml
# ✅ RECOMMENDED: Cross-platform using args array
hooks:
  post_install:
    - action: run_command
      params:
        command: "echo"
        args:
          - "Go {version} installed successfully"

# ⚠️ PLATFORM-SPECIFIC: Works but requires specific shell
hooks:
  post_install:
    - action: run_command
      params:
        command: "bash"
        args:
          - "-c"
          - "echo 'Go {version} installed' && go version"
```

**Platform-Specific Examples:**

```yaml
# Linux/macOS: Bash script
hooks:
  post_install:
    - action: run_command
      params:
        command: "bash"
        args:
          - "-c"
          - "echo $HOME/.goenv/versions/{version} >> ~/.goenv/version-history"

# Windows: PowerShell script
hooks:
  post_install:
    - action: run_command
      params:
        command: "powershell"
        args:
          - "-Command"
          - "Add-Content -Path $env:USERPROFILE\\.goenv\\version-history -Value '{version}'"
```

### HTTP Webhooks (`http_webhook`)

HTTP webhooks work identically on all platforms. No platform-specific dependencies.

**Network Requirements:**

- HTTPS support (built into Go standard library)
- Outbound network access (if targeting external services)
- DNS resolution (for domain names)

**Platform Behavior:**

| Platform    | HTTPS Support | DNS                                  | Proxy Support                                            |
| ----------- | ------------- | ------------------------------------ | -------------------------------------------------------- |
| **Linux**   | ✅ Native     | systemd-resolved or /etc/resolv.conf | Respects `HTTP_PROXY`, `HTTPS_PROXY` env vars            |
| **macOS**   | ✅ Native     | macOS DNS resolver                   | Respects system proxy settings                           |
| **Windows** | ✅ Native     | Windows DNS resolver                 | Respects system proxy settings and `HTTP_PROXY` env vars |

**Example (works on all platforms):**

```yaml
hooks:
  post_install:
    - action: http_webhook
      params:
        url: "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
        body: '{"text": "Go {version} installed on {GOOS}/{GOARCH}"}'
```

### File Operations (`log_to_file`, `check_disk_space`)

File operations work on all platforms with automatic path handling:

**Path Separators:**

| Platform    | Separator                        | Example Path                            | Tilde Expansion                          |
| ----------- | -------------------------------- | --------------------------------------- | ---------------------------------------- |
| **Linux**   | Forward slash (`/`)              | `/home/user/.goenv/logs/install.log`    | ✅ `~/.goenv/` → `/home/user/.goenv/`    |
| **macOS**   | Forward slash (`/`)              | `/Users/user/.goenv/logs/install.log`   | ✅ `~/.goenv/` → `/Users/user/.goenv/`   |
| **Windows** | Backslash (`\`) or forward slash | `C:\Users\user\.goenv\logs\install.log` | ✅ `~/.goenv/` → `C:\Users\user\.goenv\` |

**Cross-Platform Path Examples:**

```yaml
# ✅ RECOMMENDED: Tilde expansion works everywhere
hooks:
  post_install:
    - action: log_to_file
      params:
        path: ~/.goenv/logs/install.log
        message: "{timestamp} - Go {version} installed"

# ✅ ALSO WORKS: Absolute paths (but not portable)
hooks:
  post_install:
    - action: log_to_file
      params:
        path: /var/log/goenv-install.log  # Linux/macOS
        # path: C:\Logs\goenv-install.log  # Windows
        message: "{timestamp} - Go {version} installed"
```

### Environment Variables (`set_env`)

Environment variables work on all platforms but have different scoping:

| Platform    | Scope               | Persistence    | Notes                                         |
| ----------- | ------------------- | -------------- | --------------------------------------------- |
| **Linux**   | Hook execution only | Not persistent | Child processes inherit during hook execution |
| **macOS**   | Hook execution only | Not persistent | Child processes inherit during hook execution |
| **Windows** | Hook execution only | Not persistent | Child processes inherit during hook execution |

**Important:** Environment variables set via `set_env` only affect the hook execution context. They do NOT persist after the goenv command completes.

**For persistent environment variables:**

```bash
# Linux/macOS: Add to shell profile
echo 'export MY_VAR="value"' >> ~/.bashrc

# Windows: Use PowerShell profile
Add-Content $PROFILE "Set-Variable -Name MY_VAR -Value 'value'"
```

### Platform Support Summary

| Action             | Linux | macOS | Windows | External Dependencies                       |
| ------------------ | ----- | ----- | ------- | ------------------------------------------- |
| `log_to_file`      | ✅    | ✅    | ✅      | None                                        |
| `http_webhook`     | ✅    | ✅    | ✅      | Network access (outbound HTTPS)             |
| `notify_desktop`   | ✅    | ✅    | ✅      | Linux: libnotify<br>macOS/Windows: Built-in |
| `check_disk_space` | ✅    | ✅    | ✅      | None                                        |
| `set_env`          | ✅    | ✅    | ✅      | None                                        |
| `run_command`      | ✅    | ✅    | ✅      | Target command must be available            |

**Platform-Specific Documentation:**

For more details on platform support, cross-platform compatibility, and architecture-specific features, see:

- **[Platform Support Matrix](PLATFORM_SUPPORT.md)** - Comprehensive OS/architecture compatibility
- **[CI/CD Guide](CI_CD_GUIDE.md)** - Platform-specific CI/CD examples (GitHub Actions, GitLab CI)
- **[Advanced Configuration](advanced/ADVANCED_CONFIGURATION.md)** - Platform-specific environment setup

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

| Parameter | Type   | Required | Description                                 |
| --------- | ------ | -------- | ------------------------------------------- |
| `path`    | string | Yes      | Absolute or tilde-prefixed path to log file |
| `message` | string | Yes      | Message to write (supports templates)       |

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

| Parameter | Type   | Required | Description                       |
| --------- | ------ | -------- | --------------------------------- |
| `url`     | string | Yes      | HTTP(S) endpoint URL              |
| `body`    | string | Yes      | JSON payload (supports templates) |

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

#### http_webhook Security Model

The `http_webhook` action includes comprehensive security controls to prevent Server-Side Request Forgery (SSRF) attacks and protect against malicious URLs.

> **💡 Quick Reference:** For a comprehensive security guide, see the main [Security Model](#security-model) section. This section provides technical details specific to the HTTP webhook action.

**Default Security Settings (Safe by Default):**

All security features are **enabled by default** and configured via global settings:

```yaml
settings:
  allow_http: false # ✓ Default: HTTPS only
  allow_internal_ips: false # ✓ Default: block private IPs (SSRF protection)
  strict_dns: false # ✓ Default: lenient DNS validation
```

**Security Features:**

1. **HTTPS Enforcement** (`allow_http`)

   - **Default:** `false` (HTTPS required)
   - When `false`: HTTP URLs are rejected with error message
   - When `true`: Both HTTP and HTTPS allowed
   - **Recommendation:** Keep at `false` for production
   - **Override only for:** Local development (localhost)

2. **SSRF Protection** (`allow_internal_ips`)

   - **Default:** `false` (blocks internal/private IPs)
   - **Blocks access to:**
     - RFC1918 private ranges: `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`
     - Carrier-grade NAT (CGNAT): `100.64.0.0/10`
     - Loopback: `127.0.0.0/8` (IPv4), `::1/128` (IPv6)
     - Link-local: `169.254.0.0/16` (IPv4), `fe80::/10` (IPv6)
     - IPv6 unique local: `fc00::/7`
   - **DNS resolution:** Resolves hostnames and checks all returned IPs
   - **Pattern matching:** Catches `localhost`, `.internal`, `.lan`, `.local` patterns even if DNS fails
   - **Recommendation:** Keep at `false` unless you need to webhook internal services
   - **Override only when:** You need to send webhooks to trusted internal services

3. **Strict DNS Mode** (`strict_dns`)

   - **Default:** `false` (lenient mode - recommended for most users)

   **Lenient Mode** (`strict_dns: false`, default):

   - If DNS resolution fails, falls back to hostname pattern checks
   - Blocks obvious internal patterns: `localhost`, `.internal`, `.lan`, `.local`
   - Allows URLs with DNS failures if no suspicious patterns detected
   - **Use case:** Production with reliable DNS, handles temporary DNS failures gracefully

   **Strict Mode** (`strict_dns: true`):

   - Rejects URLs when DNS resolution fails (if `allow_internal_ips: false`)
   - Provides maximum SSRF protection
   - May block legitimate URLs with temporary DNS issues
   - **Use case:** Untrusted environments, maximum security requirements, can tolerate DNS failure impacts

   **When strict_dns Helps:**

   - Prevents DNS rebinding attacks (attacker controls DNS, changes IP after validation)
   - Ensures every webhook URL can be validated against private IP ranges
   - Provides defense-in-depth when DNS infrastructure might be compromised

   **Trade-offs:**

   - `strict_dns: true` may fail legitimate webhooks during DNS outages
   - `strict_dns: false` (default) provides reasonable security with better availability

**Configuration Examples:**

```yaml
# ✅ Maximum security (recommended for production)
settings:
  allow_http: false # HTTPS only
  allow_internal_ips: false # Block internal IPs
  strict_dns: true # Reject on DNS failure (strict)

hooks:
  post_install:
    - action: http_webhook
      params:
        url: https://webhooks.example.com/notify
        body: '{"version": "{version}"}'
```

```yaml
# ✅ Balanced security (default, recommended for most users)
settings:
  allow_http: false # HTTPS only
  allow_internal_ips: false # Block internal IPs
  strict_dns: false # Allow DNS failures for legitimate services (default)

hooks:
  post_install:
    - action: http_webhook
      params:
        url: https://hooks.slack.com/services/YOUR/WEBHOOK/URL
        body: '{"text": "Installed {version}"}'
```

```yaml
# ⚠️ Development/testing only (NOT for production)
settings:
  allow_http: true # Allow HTTP for local testing
  allow_internal_ips: true # Allow internal IPs
  strict_dns: false # Lenient DNS

hooks:
  post_install:
    - action: http_webhook
      params:
        url: http://localhost:8080/webhook
        body: '{"version": "{version}"}'
```

```yaml
# ⚠️ Internal network only (trusted environment)
settings:
  allow_http: false # Keep HTTPS
  allow_internal_ips: true # REQUIRED: Allow internal IPs for corporate webhooks
  strict_dns: false # Lenient: Internal DNS may be flaky

hooks:
  post_install:
    - action: http_webhook
      params:
        url: https://webhook.internal.company.com/api
        body: '{"version": "{version}"}'
```

**What Gets Blocked by Default:**

```yaml
# ❌ All blocked by default (allow_internal_ips: false)
- http://localhost:3000/webhook # Loopback
- https://192.168.1.10/api # RFC1918 private
- http://internal.company.local/notify # Internal domain pattern
- https://10.0.0.5/webhook # RFC1918 private
- http://100.64.1.1/api # CGNAT range
- https://api.internal/v1/hook # Internal pattern
- http://172.16.50.10/webhook # RFC1918 private
- https://[::1]/api # IPv6 loopback
- https://[fe80::1]/webhook # IPv6 link-local
- https://[fc00::1]/api # IPv6 unique local

# ✅ All allowed by default (public HTTPS)
- https://hooks.slack.com/services/... # Public HTTPS
- https://discord.com/api/webhooks/... # Public HTTPS
- https://api.example.com/webhooks # Public HTTPS
- https://webhook.site/unique-url # Public HTTPS
```

**Error Messages You Might See:**

```
# HTTPS enforcement (allow_http: false)
❌ HTTP URLs are not allowed (use HTTPS or set allow_http: true)

# SSRF protection (allow_internal_ips: false)
❌ internal/private IP addresses are not allowed (set allow_internal_ips: true)
❌ hostname resolves to internal/private IP 192.168.1.10 (set allow_internal_ips: true)
❌ hostname appears to target internal resources (set allow_internal_ips: true)

# Strict DNS mode (strict_dns: true)
❌ DNS resolution failed and strict_dns is enabled: lookup failed
```

**Security Notes:**

- **CGNAT ranges** (`100.64.0.0/10`) are blocked by default to prevent attacks via carrier-grade NAT networks
- **DNS rebinding attacks** are mitigated by resolving hostnames and checking all returned IPs
- **Defense in depth:** Pattern matching provides protection even when DNS resolution fails
- **Explicit opt-in:** To use internal webhooks, you must explicitly set `allow_internal_ips: true`
- **Maximum security:** Enable `strict_dns: true` for environments where DNS cannot be trusted
- **Audit regularly:** Use the [Security Audit Checklist](#security-audit-checklist) to verify your configuration

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

| Parameter | Type   | Required | Description                             |
| --------- | ------ | -------- | --------------------------------------- |
| `title`   | string | Yes      | Notification title (supports templates) |
| `message` | string | Yes      | Notification body (supports templates)  |

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

| Parameter      | Type   | Required | Description                            |
| -------------- | ------ | -------- | -------------------------------------- |
| `min_space_mb` | number | Yes      | Minimum required space in megabytes    |
| `path`         | string | No       | Path to check (defaults to goenv root) |

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

| Parameter | Type   | Required | Description                         |
| --------- | ------ | -------- | ----------------------------------- |
| `name`    | string | Yes      | Environment variable name           |
| `value`   | string | Yes      | Variable value (supports templates) |

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

**🔒 SECURITY BEST PRACTICE: Always Use Args Array**

The `args` array form is the **ONLY recommended pattern** for running commands. It prevents shell injection by passing arguments directly without shell interpretation.

**Best practice (use this pattern):**

```yaml
# ✅ ALWAYS USE: Args array (secure by design)
- action: run_command
  params:
    command: "go"
    args: ["build", "-o", "/tmp/app"]
```

**Why args array is mandatory for security:**

- **No shell injection**: Arguments can't contain shell metacharacters (`;`, `|`, `$()`, backticks, etc.)
- **Predictable behavior**: Spaces, quotes, and special chars are treated as literals
- **Template safety**: Template variables with any content are safe (no escaping needed)
- **Cross-platform**: Works consistently on Windows, macOS, and Linux

**Avoid this pattern (shell string):**

```yaml
# ⚠️ AVOID: Shell string (only for static, fully trusted commands)
- action: run_command
  params:
    command: "go build -o /tmp/app" # Shell interprets - use args array instead!
```

Use shell strings ONLY when the command is static, hardcoded, and has zero variables or user input. **In all other cases, use args array.**

**Parameters:**

| Parameter        | Type    | Required | Description                                                     |
| ---------------- | ------- | -------- | --------------------------------------------------------------- |
| `command`        | string  | Yes      | Command to execute (supports templates)                         |
| `args`           | array   | No       | Command arguments (each supports templates)                     |
| `working_dir`    | string  | No       | Working directory for command execution                         |
| `timeout`        | string  | No       | Command timeout (default: "2m")                                 |
| `capture_output` | boolean | No       | Capture stdout/stderr (default: false)                          |
| `log_output`     | boolean | No       | Log command output (default: false)                             |
| `fail_on_error`  | boolean | No       | Fail hook if command fails (default: true)                      |
| `shell`          | string  | No       | Shell to use: auto, bash, sh, cmd, powershell (default: "auto") |

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
        command: "go version" # ⚠️ OK only if: no user input, no templates, fully trusted
        capture_output: true
```

**Security Anti-Patterns (DO NOT USE):**

```yaml
# ❌ INSECURE: Template in command string
- action: run_command
  params:
    command: "go install {package}@latest" # {package} could contain shell metacharacters

# ❌ INSECURE: User input in command string
- action: run_command
  params:
    command: "git clone {repo_url}" # {repo_url} could contain "; rm -rf /"

# ✅ SECURE: Use args array instead
- action: run_command
  params:
    command: "git"
    args: ["clone", "{repo_url}"] # Safe: repo_url treated as literal string
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
    command: "go build -o output-{version}" # Shell interprets {version}

# ✅ PREFER: Direct execution (command + args)
- action: run_command
  params:
    command: "go"
    args: ["build", "-o", "output-{version}"] # No shell, safer
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

> **📊 For detailed information about operational limits, timeouts, and max_actions, see the [Operational Limits](#operational-limits) section.**

### Global Settings

```yaml
enabled: true
acknowledged_risks: true

settings:
  timeout: "5s"
  max_actions: 10
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

**`settings.timeout`** (string, default: "5s")

- Maximum time for each action to complete
- Valid units: `s` (seconds), `m` (minutes), `h` (hours)
- Examples: `"5s"`, `"30s"`, `"1m"`, `"2m30s"`
- Actions exceeding timeout are terminated
- See [Operational Limits](#operational-limits) for detailed timeout behavior

**`settings.max_actions`** (integer, default: 10, range: 1-100)

- Maximum number of actions allowed per hook point
- Enforced at config load time
- Prevents accidentally complex hook chains
- Validation error if exceeded: `hook point X has Y actions, max is Z`
- See [Operational Limits](#operational-limits) for examples

**`settings.continue_on_error`** (boolean, default: true)

- If `true`: Log errors but continue executing remaining hooks
- If `false`: Stop executing hooks on first error
- Recommended: `true` (hooks shouldn't break normal operations)
- See [Operational Limits](#operational-limits) for detailed behavior

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

- Enhanced DNS validation for `http_webhook` actions
- When `true`: More strict DNS checking to prevent DNS rebinding attacks
- When `false`: Standard DNS resolution
- Recommended for high-security environments

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
- `{file_arg}` - File argument passed to command, if any (e.g., "main.go")

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

Log all Go commands for analytics, including file arguments:

```yaml
hooks:
  pre_exec:
    - action: log_to_file
      params:
        path: ~/.goenv/logs/usage.log
        message: "[{timestamp}] v{version}: {command} {file_arg}"

  post_exec:
    - action: http_webhook
      params:
        url: https://api.example.com/metrics
        body: |
          {
            "event": "go_command_executed",
            "version": "{version}",
            "command": "{command}",
            "file_arg": "{file_arg}",
            "timestamp": "{timestamp}"
          }
```

**Example log output:**

```
[2025-10-28T14:30:00Z] v1.23.2: go build main.go
[2025-10-28T14:30:15Z] v1.23.2: go run
[2025-10-28T14:31:00Z] v1.22.0: go test ./...
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
        fail_on_error: false # Don't fail install if test fails
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
        args:
          [
            "-p",
            "~/go/{version}/bin",
            "~/go/{version}/src",
            "~/go/{version}/pkg",
          ]
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
        fail_on_error: false # Non-critical

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

### Conditional Execution Patterns

While goenv hooks don't have built-in conditional logic, you can use shell commands to implement flexible conditional execution based on Go version, environment, OS, or any custom condition.

> **💡 Design Note:** goenv deliberately uses shell-based conditionals instead of a custom expression language. This provides maximum flexibility while keeping the hooks system simple and avoiding the need to maintain a complex condition evaluator.

#### Pattern 1: Version-Specific Actions

Execute actions only for specific Go versions or version ranges:

**Example: Go 1.21+ Only (New Features)**

```yaml
hooks:
  post_install:
    - action: run_command
      params:
        command: "bash"
        args:
          - "-c"
          - |
            # Only run for Go 1.21+
            VERSION=$(go version | grep -oE '[0-9]+\.[0-9]+' | head -1)
            if [[ "$VERSION" > "1.20" ]]; then
              echo "Go 1.21+ detected - enabling new toolchain features"
              # Your commands here
              go env -w GOTOOLCHAIN=local
            else
              echo "Go 1.20 or earlier - using legacy behavior"
            fi
```

**Example: Exact Version Match**

```yaml
hooks:
  post_install:
    - action: run_command
      params:
        command: "bash"
        args:
          - "-c"
          - |
            if [ "{version}" = "1.23.2" ]; then
              echo "Exact version match - running special setup"
              # Install version-specific tools
              go install golang.org/x/tools/gopls@v0.14.0
            fi
```

**Example: Version Range**

```yaml
hooks:
  post_install:
    - action: run_command
      params:
        command: "bash"
        args:
          - "-c"
          - |
            MAJOR=$(echo {version} | cut -d. -f1)
            MINOR=$(echo {version} | cut -d. -f2)

            # For Go 1.20.x to 1.22.x
            if [ "$MAJOR" -eq 1 ] && [ "$MINOR" -ge 20 ] && [ "$MINOR" -le 22 ]; then
              echo "Go 1.20-1.22 detected - applying compatibility patches"
              # Version-specific setup
            fi
```

**Example: Legacy Version Handling**

```yaml
hooks:
  post_install:
    - action: run_command
      params:
        command: "bash"
        args:
          - "-c"
          - |
            VERSION=$(go version | grep -oE '[0-9]+\.[0-9]+' | head -1)
            if [[ "$VERSION" < "1.18" ]]; then
              echo "Legacy Go version detected - installing compatibility tools"
              # Old Go versions need different gopls
              go get golang.org/x/tools/gopls@v0.7.5
            else
              echo "Modern Go version - using latest gopls"
              go install golang.org/x/tools/gopls@latest
            fi
```

#### Pattern 2: Environment-Based Conditionals

Execute different actions based on environment (CI, dev, prod):

**Example: CI vs Local Development**

```yaml
hooks:
  post_install:
    - action: run_command
      params:
        command: "bash"
        args:
          - "-c"
          - |
            if [ "$CI" = "true" ]; then
              echo "CI environment detected"
              # Minimal CI setup
              go install golang.org/x/tools/cmd/goimports@latest
              # Log to CI-friendly location
              echo "{version}" >> /var/log/goenv-ci.log
            else
              echo "Local development environment"
              # Full development tools
              go install golang.org/x/tools/gopls@latest
              go install github.com/go-delve/delve/cmd/dlv@latest
              # Desktop notification
              notify-send "Go {version} installed" "Development environment ready"
            fi
```

**Example: Multi-Environment Setup**

```yaml
hooks:
  post_install:
    - action: run_command
      params:
        command: "bash"
        args:
          - "-c"
          - |
            ENV=${DEPLOYMENT_ENV:-dev}

            case "$ENV" in
              production)
                echo "Production environment - minimal tooling"
                go install golang.org/x/tools/cmd/goimports@latest
                ;;
              staging)
                echo "Staging environment - testing tools"
                go install golang.org/x/tools/cmd/goimports@latest
                go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
                ;;
              dev)
                echo "Development environment - full toolset"
                go install golang.org/x/tools/gopls@latest
                go install github.com/go-delve/delve/cmd/dlv@latest
                go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
                ;;
              *)
                echo "Unknown environment: $ENV"
                ;;
            esac
```

**Example: GitHub Actions Detection**

```yaml
hooks:
  post_install:
    - action: run_command
      params:
        command: "bash"
        args:
          - "-c"
          - |
            if [ "$GITHUB_ACTIONS" = "true" ]; then
              echo "GitHub Actions detected"
              echo "::notice::Go {version} installed successfully"
              # GitHub Actions-specific logging
              echo "{version}" >> $GITHUB_STEP_SUMMARY
            fi
```

#### Pattern 3: OS-Specific Actions

Execute different commands based on operating system:

**Example: Cross-Platform Notifications**

```yaml
hooks:
  post_install:
    - action: run_command
      params:
        command: "bash"
        args:
          - "-c"
          - |
            case "$OSTYPE" in
              linux*)
                # Linux: use notify-send
                if command -v notify-send &> /dev/null; then
                  notify-send "goenv" "Go {version} installed"
                fi
                ;;
              darwin*)
                # macOS: use osascript
                osascript -e 'display notification "Go {version} installed" with title "goenv"'
                ;;
              msys*|cygwin*)
                # Windows: use PowerShell
                powershell -Command "New-BurntToastNotification -Text 'goenv', 'Go {version} installed'"
                ;;
            esac
```

**Example: Platform-Specific Tool Installation**

```yaml
hooks:
  post_install:
    - action: run_command
      params:
        command: "bash"
        args:
          - "-c"
          - |
            case "$OSTYPE" in
              linux*)
                echo "Linux: Installing development tools"
                # Linux-specific tools
                go install github.com/go-delve/delve/cmd/dlv@latest
                ;;
              darwin*)
                echo "macOS: Installing development tools"
                # macOS-specific setup
                go install github.com/go-delve/delve/cmd/dlv@latest
                # macOS-specific: code signing might be needed
                ;;
              msys*|cygwin*)
                echo "Windows: Installing development tools"
                # Windows may need different approach for some tools
                go install golang.org/x/tools/gopls@latest
                ;;
            esac
```

**Example: Architecture-Specific Setup**

```yaml
hooks:
  post_install:
    - action: run_command
      params:
        command: "bash"
        args:
          - "-c"
          - |
            ARCH=$(uname -m)

            case "$ARCH" in
              x86_64|amd64)
                echo "AMD64 architecture - standard setup"
                # Standard tools
                ;;
              arm64|aarch64)
                echo "ARM64 architecture - optimized binaries"
                # May need ARM-specific builds
                export GOARCH=arm64
                ;;
              *)
                echo "Unsupported architecture: $ARCH"
                exit 1
                ;;
            esac
```

#### Pattern 4: File/Directory Existence Checks

Execute actions based on file or directory presence:

**Example: Project Type Detection**

```yaml
hooks:
  post_install:
    - action: run_command
      params:
        command: "bash"
        args:
          - "-c"
          - |
            if [ -f "go.work" ]; then
              echo "Go workspace detected - multi-module project"
              # Install workspace-specific tools
              go install golang.org/x/tools/gopls@latest
            elif [ -f "go.mod" ]; then
              echo "Go module detected - standard project"
              # Install standard tools
              go install golang.org/x/tools/cmd/goimports@latest
            else
              echo "No Go project detected - basic setup only"
            fi
```

**Example: Configuration File Detection**

```yaml
hooks:
  post_install:
    - action: run_command
      params:
        command: "bash"
        args:
          - "-c"
          - |
            if [ -f "$HOME/.goenv/team-config.sh" ]; then
              echo "Team configuration found - applying team settings"
              source "$HOME/.goenv/team-config.sh"
              # Team-specific setup based on sourced config
            else
              echo "No team config - using defaults"
            fi
```

#### Pattern 5: Complex Multi-Condition Logic

Combine multiple conditions with AND/OR logic:

**Example: Version AND Environment**

```yaml
hooks:
  post_install:
    - action: run_command
      params:
        command: "bash"
        args:
          - "-c"
          - |
            VERSION=$(go version | grep -oE '[0-9]+\.[0-9]+' | head -1)

            # Go 1.21+ in CI environment
            if [[ "$VERSION" > "1.20" ]] && [ "$CI" = "true" ]; then
              echo "Go 1.21+ in CI - enabling new toolchain and minimal tools"
              go env -w GOTOOLCHAIN=local
              go install golang.org/x/tools/cmd/goimports@latest
            # Go 1.21+ in local development
            elif [[ "$VERSION" > "1.20" ]] && [ "$CI" != "true" ]; then
              echo "Go 1.21+ in dev - full feature set"
              go env -w GOTOOLCHAIN=local
              go install golang.org/x/tools/gopls@latest
              go install github.com/go-delve/delve/cmd/dlv@latest
            # Legacy version
            else
              echo "Legacy Go version - compatibility mode"
              # Old tools for old Go
            fi
```

**Example: Platform OR CI**

```yaml
hooks:
  post_install:
    - action: run_command
      params:
        command: "bash"
        args:
          - "-c"
          - |
            # Skip slow operations on Windows OR in CI
            if [[ "$OSTYPE" == "msys" ]] || [ "$CI" = "true" ]; then
              echo "Fast mode - skipping slow operations"
              # Quick setup only
            else
              echo "Full mode - running complete setup"
              # Full development environment
              go install golang.org/x/tools/gopls@latest
              go install github.com/go-delve/delve/cmd/dlv@latest
              go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
            fi
```

#### Pattern 6: Custom Condition Functions

Create reusable condition checks:

```yaml
hooks:
  post_install:
    - action: run_command
      params:
        command: "bash"
        args:
          - "-c"
          - |
            # Define helper functions
            is_go_21_plus() {
              VERSION=$(go version | grep -oE '[0-9]+\.[0-9]+' | head -1)
              [[ "$VERSION" > "1.20" ]]
            }

            is_ci() {
              [ "$CI" = "true" ] || [ "$GITHUB_ACTIONS" = "true" ] || [ "$GITLAB_CI" = "true" ]
            }

            is_linux() {
              [[ "$OSTYPE" == "linux"* ]]
            }

            # Use the functions
            if is_go_21_plus && is_ci && is_linux; then
              echo "Go 1.21+ in Linux CI - optimized setup"
              # Specific configuration
            fi
```

#### When to Use Conditional Patterns

**✅ Use shell-based conditionals for:**

- Complex version comparisons
- Environment detection (CI, dev, prod)
- OS-specific logic
- File existence checks
- Any custom condition you can express in shell

**✅ Benefits:**

- Extremely flexible - any condition possible
- Platform-aware (shell auto-detection)
- Leverages existing shell knowledge
- No additional goenv code to maintain

**⚠️ Trade-offs:**

- Requires shell knowledge
- Platform-specific syntax (use bash for consistency)
- More verbose than hypothetical native conditionals
- Harder to validate than structured conditionals

**💡 Best Practices:**

1. **Use bash for consistency:** Specify `command: "bash"` to ensure consistent behavior across platforms
2. **Test your conditions:** Use `goenv hooks test post_install` to verify logic
3. **Fail gracefully:** Set `fail_on_error: false` for non-critical conditionals
4. **Document your logic:** Add comments explaining complex conditions
5. **Extract to scripts:** For very complex logic, use external scripts and call them via `run_command`

**🚀 Future:** If native conditional hooks become common user requests, goenv may add built-in condition support. For now, shell-based patterns provide all necessary flexibility.

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

**Essential Security Practices:**

1. **Use Default SSRF Protection:** Keep `allow_http: false` and `allow_internal_ips: false` unless you have a specific need

   - See [Security Model](#security-model) for comprehensive SSRF protection details
   - Default settings block HTTP URLs and internal/private IP addresses

2. **Enable Strict DNS for Production/CI:** **Strongly recommended** for CI/CD pipelines and untrusted environments

   - Rejects URLs when DNS resolution fails (prevents DNS rebinding attacks)
   - Provides maximum protection against SSRF via DNS manipulation
   - **Especially important in CI/CD:** Prevents malicious PRs from exploiting DNS to reach internal services
   - Trade-off: may block legitimate URLs during DNS outages (acceptable in CI)

   **CI/CD recommendation:**

   ```yaml
   # ~/.goenv/hooks.yaml - Production/CI configuration
   settings:
     allow_http: false # HTTPS only
     allow_internal_ips: false # Block internal IPs
     strict_dns: true # ✅ RECOMMENDED for CI - reject on DNS failure
   ```

   **Why strict_dns matters in CI:**

   - CI environments may have access to internal networks
   - Malicious PRs could attempt DNS rebinding to probe infrastructure
   - `strict_dns: true` adds defense-in-depth against these attacks
   - Future goenv versions may default to `strict_dns: true` for enhanced security

3. **Audit Your Configuration:** Before enabling hooks, verify security settings

   ```bash
   # Check for HTTP URLs (should find none in production)
   grep -i "http://" ~/.goenv/hooks.yaml

   # Verify SSRF settings
   grep -A 2 "^settings:" ~/.goenv/hooks.yaml

   # Validate configuration
   goenv hooks validate
   ```

4. **Review Generated Configuration:** Always review `hooks.yaml` before enabling

   - Check that `allow_internal_ips: false` (unless you need internal webhooks)
   - Verify all webhook URLs use HTTPS
   - Ensure `acknowledged_risks: true` only after reviewing all actions

5. **Protect Webhook URLs:** Treat webhook URLs as secrets

   - Don't commit webhook URLs to public repositories
   - Use environment variables or secret management for sensitive URLs
   - Rotate webhook URLs if accidentally exposed

6. **Limit Actions:** Only enable actions you actually use

   - Remove unused hooks from configuration
   - Start with simple actions (log_to_file) before adding webhooks
   - Test with `goenv hooks test` before enabling

7. **Use HTTPS Exclusively:** For webhooks, always use HTTPS URLs

   - Default `allow_http: false` enforces this
   - Never override to allow HTTP in production
   - HTTP acceptable only for local development (localhost)

8. **Understand Override Risks:**

   ```yaml
   # ⚠️ Only override defaults when necessary
   settings:
     allow_http: true # Risk: Unencrypted traffic, MITM attacks
     allow_internal_ips: true # Risk: SSRF, access to internal services
     strict_dns: false # Risk: DNS rebinding (but default is reasonable)
   ```

9. **Test First:** Use `goenv hooks test` before enabling hooks

   ```bash
   goenv hooks test post_install --verbose
   ```

10. **Monitor Hook Execution:** Check logs for unexpected behavior
    - Look for rejected URLs in goenv output
    - Verify webhooks are reaching intended destinations
    - Watch for timeout errors (may indicate DNS issues)

**Security Configuration Examples:**

```yaml
# ✅ RECOMMENDED: Production with maximum security
settings:
  allow_http: false # HTTPS only
  allow_internal_ips: false # Block internal IPs
  strict_dns: true # Reject on DNS failure
  timeout: "5s"
  continue_on_error: true # Don't break goenv on hook failures

hooks:
  post_install:
    - action: http_webhook
      params:
        url: https://hooks.slack.com/services/YOUR/WEBHOOK/URL
        body: '{"text": "Go {version} installed"}'
```

```yaml
# ⚠️ DEVELOPMENT ONLY: Local testing with internal webhooks
settings:
  allow_http: true # Allow HTTP for localhost
  allow_internal_ips: true # Allow internal IPs for local services
  strict_dns: false # Lenient DNS
  timeout: "5s"
  continue_on_error: true

hooks:
  post_install:
    - action: http_webhook
      params:
        url: http://localhost:8080/webhook
        body: '{"version": "{version}"}'
```

```yaml
# ⚠️ INTERNAL NETWORK: Trusted internal services only
settings:
  allow_http: false # Keep HTTPS
  allow_internal_ips: true # REQUIRED for internal webhooks
  strict_dns: false # Lenient DNS (internal DNS may be flaky)
  timeout: "10s" # Longer timeout for internal networks
  continue_on_error: true

hooks:
  post_install:
    - action: http_webhook
      params:
        url: https://webhook.internal.company.com/api/v1/notify
        body: '{"version": "{version}", "host": "{hostname}"}'
```

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
