# Hooks System Quick Start

Get started with goenv hooks in 5 minutes.

## What Are Hooks?

Hooks let you run automated actions when goenv events happen:
- **Log installations** for audit trails
- **Send team notifications** via Slack/Discord
- **Check disk space** before installing
- **Validate installations** with custom scripts

## 5-Minute Setup

### 1. Generate Configuration

```bash
goenv hooks init
```

This creates `~/.goenv/hooks.yaml` with examples.

### 2. Enable Hooks

Edit `~/.goenv/hooks.yaml`:

```yaml
enabled: true
acknowledged_risks: true
```

### 3. Add Your First Hook

Add a simple logging hook:

```yaml
hooks:
  post_install:
    - action: log_to_file
      params:
        path: ~/.goenv/install.log
        message: "[{timestamp}] Installed Go {version}"
```

### 4. Test It

```bash
# Validate configuration
goenv hooks validate

# Test without executing
goenv hooks test post_install

# Install a Go version to see it in action
goenv install 1.25.2
```

Check the log:
```bash
cat ~/.goenv/install.log
```

## Common Use Cases

### Installation Audit Trail

```yaml
hooks:
  post_install:
    - action: log_to_file
      params:
        path: ~/.goenv/audit.log
        message: "[{timestamp}] User installed Go {version}"
```

### Slack Notifications

```yaml
hooks:
  post_install:
    - action: http_webhook
      params:
        url: https://hooks.slack.com/services/YOUR/WEBHOOK/URL
        body: '{"text": "Go {version} installed on dev server"}'
```

### Disk Space Check

```yaml
hooks:
  pre_install:
    - action: check_disk_space
      params:
        min_space_mb: 500
        path: ~/.goenv/versions
```

### Desktop Notifications

```yaml
hooks:
  post_install:
    - action: notify_desktop
      params:
        title: "goenv"
        message: "Go {version} is ready to use"
```

### Installation Validation

```yaml
hooks:
  post_install:
    # Verify binary works
    - action: run_command
      params:
        command: "go"
        args: ["version"]
        capture_output: true
        log_output: true
```

## Security Notes

Hooks are **secure by default**:

- ✅ **HTTPS enforced** - HTTP URLs blocked by default
- ✅ **SSRF protection** - Internal/private IPs blocked
- ✅ **Timeout protection** - Actions can't hang forever
- ✅ **No shell injection** - Use `args` array for commands

**Default settings (recommended):**
```yaml
settings:
  allow_http: false         # HTTPS only
  allow_internal_ips: false # Block private IPs
  timeout: "5s"             # Fast fail
  continue_on_error: true   # Don't break goenv
```

Only override these settings for local development or trusted internal networks.

## Available Actions

| Action | Purpose | Example Use Case |
|--------|---------|------------------|
| `log_to_file` | Write to log files | Audit trail, debugging |
| `http_webhook` | Send HTTP POST requests | Slack/Discord notifications |
| `notify_desktop` | Show desktop notifications | Long-running operations |
| `check_disk_space` | Verify available space | Prevent failed installs |
| `set_env` | Set environment variables | Dynamic configuration |
| `run_command` | Execute commands | Validation scripts, tool installation |

## Hook Points

| Hook Point | When It Runs | Common Uses |
|------------|--------------|-------------|
| `pre_install` | Before installing Go | Check disk space, validate prerequisites |
| `post_install` | After installing Go | Log installation, notify team, validate |
| `pre_uninstall` | Before removing Go | Create backups, log removal |
| `post_uninstall` | After removing Go | Clean up, notify team |
| `pre_exec` | Before running Go command | Set env vars, log usage |
| `post_exec` | After running Go command | Track metrics, log execution |
| `pre_rehash` | Before regenerating shims | Backup shims |
| `post_rehash` | After regenerating shims | Update completions, notify tools |

## Template Variables

Use these variables in any hook parameter:

- `{version}` - Go version (e.g., "1.25.2")
- `{timestamp}` - ISO 8601 timestamp
- `{hook}` - Hook point name (e.g., "post_install")
- `{command}` - Command being executed (exec hooks only)

Example:
```yaml
message: "[{timestamp}] {hook}: Go {version}"
# Output: [2025-10-28T10:30:45Z] post_install: Go 1.25.2
```

## Replacing Custom Shell Functions

If you previously wrote custom shell functions to automate goenv workflows, hooks provide a cleaner, declarative alternative.

**Note:** Bash goenv did not have hooks. This section is for users who created their own custom automation.

### Before (Custom Shell Functions)

Manual shell scripts in your profile:
```bash
# ~/.bashrc
function goenv-post-install() {
    echo "$(date) - Installed Go $(goenv version-name)" >> ~/.goenv/log
    notify-send "Go installed"
}
```

### After (YAML-Based Hooks)

Declarative configuration:
```yaml
# ~/.goenv/hooks.yaml
hooks:
  post_install:
    - action: log_to_file
      params:
        path: ~/.goenv/log
        message: "[{timestamp}] Installed Go {version}"

    - action: notify_desktop
      params:
        title: "goenv"
        message: "Go {version} installed"
```

**Benefits:**
- ✅ No shell scripts to maintain
- ✅ Works on all platforms (Linux, macOS, Windows)
- ✅ Built-in security controls
- ✅ Validation and testing tools
- ✅ Declarative and version-controllable

## Troubleshooting

### Hooks not running?

1. Check both flags are set:
   ```yaml
   enabled: true
   acknowledged_risks: true
   ```

2. Validate configuration:
   ```bash
   goenv hooks validate
   ```

3. Check file location:
   ```bash
   ls -la ~/.goenv/hooks.yaml
   ```

### Webhook failing?

1. Test URL with curl:
   ```bash
   curl -X POST https://your-webhook-url \
     -H "Content-Type: application/json" \
     -d '{"text": "test"}'
   ```

2. Check security settings:
   ```yaml
   settings:
     allow_http: false         # Using HTTPS?
     allow_internal_ips: false # Webhook is public?
   ```

3. Check for SSRF blocks:
   - HTTP URLs blocked by default (use HTTPS)
   - Private IPs blocked by default (192.168.x.x, 10.x.x.x, 127.x.x.x)
   - Override only for development: `allow_http: true`, `allow_internal_ips: true`

### Desktop notifications not working?

**Linux:** Install libnotify:
```bash
sudo apt install libnotify-bin  # Ubuntu/Debian
sudo yum install libnotify       # RHEL/CentOS
```

**macOS/Windows:** Should work out of the box.

### Command execution failing?

Always use `args` array for security:
```yaml
# ✅ Secure
- action: run_command
  params:
    command: "go"
    args: ["build", "-o", "app"]

# ❌ Avoid
- action: run_command
  params:
    command: "go build -o app"
```

## Next Steps

- **Full documentation:** See [Hooks System Documentation](./HOOKS.md)
- **Security details:** See [Security Model](./HOOKS.md#security-model)
- **Advanced examples:** See [Advanced Patterns](./HOOKS.md#advanced-patterns)
- **Compliance use cases:** See below for SBOM and audit trail examples

## Compliance and Audit Examples

### SOC 2 Audit Trail

Track all version changes with timestamps:

```yaml
hooks:
  post_install:
    - action: log_to_file
      params:
        path: ~/.goenv/audit/installations.log
        message: "[{timestamp}] INSTALL: Go {version} by ${USER} on ${HOSTNAME}"

    - action: run_command
      params:
        command: "git"
        args: ["-C", "~/.goenv/audit", "add", "installations.log"]
        fail_on_error: false

    - action: run_command
      params:
        command: "git"
        args: ["-C", "~/.goenv/audit", "commit", "-m", "Installed Go {version}"]
        fail_on_error: false

  post_uninstall:
    - action: log_to_file
      params:
        path: ~/.goenv/audit/installations.log
        message: "[{timestamp}] UNINSTALL: Go {version} by ${USER} on ${HOSTNAME}"
```

### SBOM Generation

Automatically generate Software Bill of Materials after installation:

```yaml
hooks:
  post_install:
    - action: run_command
      params:
        command: "goenv"
        args: ["sbom", "project", "--tool=cyclonedx-gomod", "--output=/tmp/sbom-{version}.json"]
        timeout: "2m"
        fail_on_error: false

    - action: log_to_file
      params:
        path: ~/.goenv/sbom-log.txt
        message: "[{timestamp}] Generated SBOM for Go {version}"
```

### Change Management

Notify change management system:

```yaml
hooks:
  post_install:
    - action: http_webhook
      params:
        url: https://change-management.company.com/api/v1/changes
        body: |
          {
            "type": "software_installation",
            "component": "golang",
            "version": "{version}",
            "timestamp": "{timestamp}",
            "environment": "development",
            "user": "${USER}",
            "host": "${HOSTNAME}"
          }
```

## Example: Complete Development Workflow

```yaml
enabled: true
acknowledged_risks: true

settings:
  timeout: "10s"
  continue_on_error: true
  allow_http: false
  allow_internal_ips: false

hooks:
  # Before installation
  pre_install:
    - action: check_disk_space
      params:
        min_space_mb: 500

    - action: notify_desktop
      params:
        title: "goenv"
        message: "Installing Go {version}..."

  # After installation
  post_install:
    # Audit log
    - action: log_to_file
      params:
        path: ~/.goenv/audit.log
        message: "[{timestamp}] Installed Go {version}"

    # Validate installation
    - action: run_command
      params:
        command: "go"
        args: ["version"]
        capture_output: true
        log_output: true

    # Notify team
    - action: http_webhook
      params:
        url: https://hooks.slack.com/services/YOUR/WEBHOOK/URL
        body: '{"text": "Go {version} installed on development server"}'

    # Desktop notification
    - action: notify_desktop
      params:
        title: "goenv: Installation Complete"
        message: "Go {version} is ready to use"

  # Before running commands
  pre_exec:
    - action: set_env
      params:
        name: GOENV_ACTIVE_VERSION
        value: "{version}"

    - action: log_to_file
      params:
        path: ~/.goenv/commands.log
        message: "[{timestamp}] Running '{command}' with Go {version}"
```

## Getting Help

- **Validate config:** `goenv hooks validate`
- **Test hooks:** `goenv hooks test <hook-point>`
- **List available:** `goenv hooks list`
- **Full documentation:** [HOOKS.md](./HOOKS.md)
- **Report issues:** https://github.com/go-nv/goenv/issues
