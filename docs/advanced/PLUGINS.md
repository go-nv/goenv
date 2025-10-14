# Plugins Guide

goenv supports a plugin system that allows you to extend functionality by adding custom commands. Plugins are automatically discovered and integrated seamlessly with goenv's command system.

## Table of Contents

- [Overview](#overview)
- [Plugin Structure](#plugin-structure)
- [Creating a Plugin](#creating-a-plugin)
- [Installing Plugins](#installing-plugins)
- [Plugin Discovery](#plugin-discovery)
- [Examples](#examples)
- [Best Practices](#best-practices)

## Overview

Plugins extend goenv by adding new commands without modifying the core codebase. They:

- **Integrate automatically** - No configuration needed once installed
- **Appear in `goenv commands`** - Discoverable like built-in commands
- **Support shell completion** - Can provide completion via `--complete` flag
- **Run as executables** - Can be written in any language

## Plugin Structure

### Directory Layout

Plugins live in the `plugins` directory under `$GOENV_ROOT`:

```
$GOENV_ROOT/
└── plugins/
    ├── my-plugin/
    │   ├── bin/
    │   │   ├── goenv-custom-command
    │   │   └── goenv-another-command
    │   ├── README.md
    │   └── LICENSE
    └── another-plugin/
        └── bin/
            └── goenv-special-feature
```

### Naming Convention

Plugin executables must follow this naming pattern:
- **Prefix:** `goenv-`
- **Command name:** lowercase with hyphens
- **Example:** `goenv-my-command` → invoked as `goenv my-command`

### File Requirements

1. **Executable permission** - Must be executable (`chmod +x`)
2. **Shebang line** - Should specify interpreter (e.g., `#!/usr/bin/env bash`)
3. **Location** - Must be in `$GOENV_ROOT/plugins/*/bin/`

## Creating a Plugin

### Step 1: Create Plugin Directory

```bash
mkdir -p "$GOENV_ROOT/plugins/my-plugin/bin"
```

### Step 2: Create Plugin Executable

```bash
cat > "$GOENV_ROOT/plugins/my-plugin/bin/goenv-hello" << 'EOF'
#!/usr/bin/env bash
# Summary: Print a friendly greeting
# Usage: goenv hello [name]
#
# Prints a greeting message. If name is provided, greets that person.
# Otherwise, greets the world.

set -e

name="${1:-World}"
echo "Hello, $name! You're using goenv $(goenv --version)"
EOF
```

### Step 3: Make Executable

```bash
chmod +x "$GOENV_ROOT/plugins/my-plugin/bin/goenv-hello"
```

### Step 4: Test Plugin

```bash
$ goenv hello
Hello, World! You're using goenv 3.0.0

$ goenv hello Alice
Hello, Alice! You're using goenv 3.0.0

$ goenv commands | grep hello
hello
```

## Installing Plugins

### Method 1: Manual Installation

```bash
# Clone or copy plugin to plugins directory
cd "$GOENV_ROOT/plugins"
git clone https://github.com/user/goenv-plugin-name.git

# Ensure executables are marked as executable
chmod +x goenv-plugin-name/bin/goenv-*
```

### Method 2: Git Submodule

```bash
cd "$GOENV_ROOT"
git submodule add https://github.com/user/goenv-plugin-name.git plugins/plugin-name
git submodule update --init --recursive
```

### Method 3: Symlink

```bash
# For development or testing
ln -s /path/to/my-plugin "$GOENV_ROOT/plugins/my-plugin"
```

## Plugin Discovery

### How Discovery Works

When you run `goenv commands`, goenv:

1. Scans `$GOENV_ROOT/plugins/*/bin/` for executables
2. Finds files matching `goenv-*` pattern
3. Checks executable permission (Unix-like systems)
4. Adds command names to the command list

### Discovery Example

```bash
# View all commands including plugins
$ goenv commands

# View only plugin commands (they're mixed with built-ins)
$ goenv commands | grep -v -E '^(global|local|shell|version|install)$'
```

## Examples

### Example 1: Version Info Plugin

Display detailed version information:

```bash
#!/usr/bin/env bash
# goenv-info
# Summary: Display detailed version information
# Usage: goenv info

set -e

version=$(goenv version-name)
origin=$(goenv version-origin)
prefix=$(goenv prefix)

echo "Current Go Version: $version"
echo "Set by: $origin"
echo "Location: $prefix"
echo ""
echo "GOROOT: ${GOROOT:-not set}"
echo "GOPATH: ${GOPATH:-not set}"
```

Usage:
```bash
$ goenv info
Current Go Version: 1.22.5
Set by: /home/user/project/.go-version
Location: /home/user/.goenv/versions/1.22.5

GOROOT: /home/user/.goenv/versions/1.22.5
GOPATH: /home/user/go/1.22.5
```

### Example 2: Quick Switch Plugin

Quickly switch between favorite versions:

```bash
#!/usr/bin/env bash
# goenv-quick
# Summary: Quickly switch between favorite Go versions
# Usage: goenv quick <alias>
#
# Aliases:
#   stable  - Latest stable release (1.22.5)
#   latest  - Bleeding edge (1.23.0)
#   legacy  - For old projects (1.20.0)

set -e

case "$1" in
  stable)
    goenv local 1.22.5
    echo "Switched to stable (1.22.5)"
    ;;
  latest)
    goenv local 1.23.0
    echo "Switched to latest (1.23.0)"
    ;;
  legacy)
    goenv local 1.20.0
    echo "Switched to legacy (1.20.0)"
    ;;
  *)
    echo "Usage: goenv quick <stable|latest|legacy>" >&2
    exit 1
    ;;
esac
```

### Example 3: Plugin with Completion

Add shell completion support:

```bash
#!/usr/bin/env bash
# goenv-project
# Summary: Project-specific goenv operations
# Usage: goenv project <init|status|clean>

set -e

# Handle completion
if [ "$1" = "--complete" ]; then
  echo init
  echo status
  echo clean
  exit 0
fi

case "$1" in
  init)
    echo "Initializing project..."
    goenv local 1.22.5
    echo "1.22.5" > .go-version
    echo "✓ Project initialized with Go 1.22.5"
    ;;
  status)
    if [ -f .go-version ]; then
      version=$(cat .go-version)
      echo "Project Go version: $version"
      echo "Active version: $(goenv version-name)"
    else
      echo "No .go-version file found"
    fi
    ;;
  clean)
    rm -f .go-version
    echo "✓ Removed .go-version file"
    ;;
  *)
    goenv help project
    exit 1
    ;;
esac
```

### Example 4: Multi-Language Plugin

Python plugin for managing Python alongside Go:

```bash
#!/usr/bin/env python3
# goenv-python
# Summary: Check Python version compatibility
# Usage: goenv python

import sys
import subprocess

def main():
    # Get Go version
    go_version = subprocess.check_output(['goenv', 'version-name']).decode().strip()
    
    # Get Python version
    py_version = f"{sys.version_info.major}.{sys.version_info.minor}.{sys.version_info.micro}"
    
    print(f"Go version: {go_version}")
    print(f"Python version: {py_version}")
    
    # Warn about compatibility
    if sys.version_info < (3, 8):
        print("⚠️  Warning: Python < 3.8 may have issues with Go tooling")

if __name__ == '__main__':
    main()
```

### Example 5: Cloud Deployment Plugin

Deploy Go apps to cloud:

```bash
#!/usr/bin/env bash
# goenv-deploy
# Summary: Deploy Go application
# Usage: goenv deploy <target>
#
# Targets:
#   staging    - Deploy to staging environment
#   production - Deploy to production environment

set -e

target="$1"
version=$(goenv version-name)

if [ -z "$target" ]; then
  echo "Usage: goenv deploy <staging|production>" >&2
  exit 1
fi

echo "Building with Go $version..."
go build -o app -ldflags "-X main.version=$(git describe --tags)"

echo "Deploying to $target..."
case "$target" in
  staging)
    scp app deploy@staging.example.com:/apps/myapp/
    ;;
  production)
    scp app deploy@production.example.com:/apps/myapp/
    ;;
  *)
    echo "Unknown target: $target" >&2
    exit 1
    ;;
esac

echo "✓ Deployed successfully"
```

## Best Practices

### 1. Follow Naming Conventions

```bash
# Good
goenv-my-command
goenv-custom-tool
goenv-project-init

# Bad
goenv_my_command   # Use hyphens, not underscores
my-command         # Missing goenv- prefix
goenv-MyCommand    # Use lowercase
```

### 2. Provide Help Documentation

Use standard goenv help format:

```bash
#!/usr/bin/env bash
# Summary: One-line description
# Usage: goenv command [options]
#
# Detailed description goes here.
# Can span multiple lines.
#
# Options:
#   -f, --flag    Flag description
#
# Examples:
#   goenv command --flag value
```

### 3. Support Completion

```bash
if [ "$1" = "--complete" ]; then
  # Output possible completions
  echo option1
  echo option2
  exit 0
fi
```

### 4. Use Standard goenv Commands

```bash
# Good - use goenv commands
version=$(goenv version-name)
prefix=$(goenv prefix)

# Avoid - direct environment variable access
version=$GOENV_VERSION  # May not be set
```

### 5. Handle Errors Gracefully

```bash
set -e  # Exit on error

if ! goenv version-name &>/dev/null; then
  echo "Error: No Go version set" >&2
  exit 1
fi
```

### 6. Make Commands Composable

```bash
# Allow piping and chaining
goenv info --json | jq '.version'
goenv quick stable && go test ./...
```

### 7. Add Logging for Debug

```bash
if [ -n "$GOENV_DEBUG" ]; then
  echo "Debug: Running command with version $(goenv version-name)" >&2
fi
```

## Testing Plugins

### Manual Testing

```bash
# Test plugin directly
"$GOENV_ROOT/plugins/my-plugin/bin/goenv-hello"

# Test via goenv
goenv hello

# Test completion
goenv hello --complete
```

### Automated Testing

```bash
#!/usr/bin/env bats
# test/plugin-hello.bats

@test "goenv hello without arguments" {
  run goenv hello
  [ "$status" -eq 0 ]
  [[ "$output" =~ "Hello, World!" ]]
}

@test "goenv hello with name" {
  run goenv hello Alice
  [ "$status" -eq 0 ]
  [[ "$output" =~ "Hello, Alice!" ]]
}
```

## Plugin Distribution

### GitHub Repository

Structure your plugin repository:

```
goenv-plugin-name/
├── bin/
│   └── goenv-my-command
├── README.md
├── LICENSE
└── test/
    └── plugin-test.bats
```

README.md should include:
- Installation instructions
- Usage examples
- Requirements
- License

### Installation Instructions

Provide clear installation steps in README:

```markdown
## Installation

```bash
git clone https://github.com/user/goenv-plugin-name.git \
  "$(goenv root)/plugins/plugin-name"
```
```

## Security Considerations

1. **Verify plugin sources** - Only install trusted plugins
2. **Review code** - Inspect plugin code before installation
3. **Check permissions** - Plugins run with your user permissions
4. **Use version control** - Pin plugin versions in production
5. **Audit regularly** - Review installed plugins periodically

## Troubleshooting

### Plugin Not Showing in Commands

```bash
# Check if plugin is in the right location
ls "$GOENV_ROOT/plugins/"

# Check if executable is properly named
ls "$GOENV_ROOT/plugins/*/bin/goenv-*"

# Check if executable has execute permission
ls -la "$GOENV_ROOT/plugins/my-plugin/bin/goenv-hello"

# Make executable if needed
chmod +x "$GOENV_ROOT/plugins/my-plugin/bin/goenv-hello"
```

### Plugin Not Executing

```bash
# Test directly
"$GOENV_ROOT/plugins/my-plugin/bin/goenv-hello"

# Check for errors
goenv hello 2>&1

# Enable debug mode
GOENV_DEBUG=1 goenv hello
```

## Example Plugins

Community plugins you might find useful:

1. **goenv-update** - Auto-update goenv and plugins
2. **goenv-doctor** - Health check for goenv installation
3. **goenv-aliases** - Version aliasing (e.g., `stable`, `latest`)
4. **goenv-sync** - Sync versions across machines

## Further Reading

- [Command Reference](../reference/COMMANDS.md)
- [Hooks Guide](HOOKS.md)
- [Advanced Configuration](ADVANCED_CONFIGURATION.md)
