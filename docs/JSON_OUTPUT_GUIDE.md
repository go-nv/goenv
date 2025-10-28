# JSON Output Guide

Complete reference for using goenv's JSON output in automation, CI/CD, and tooling integration.

## Table of Contents

- [Supported Commands](#supported-commands)
- [JSON Schemas](#json-schemas)
- [Common Use Cases](#common-use-cases)
- [CI/CD Integration](#cicd-integration)
- [Parsing Examples](#parsing-examples)
- [Error Handling](#error-handling)

## Supported Commands

### Commands with JSON Support

| Command | Flag | Purpose | Stability |
|---------|------|---------|-----------|
| `goenv list` | `--json` | List installed versions | ✅ Stable |
| `goenv list --remote` | `--json` | List available versions | ✅ Stable |
| `goenv current` | `--json` | Show active version | ✅ Stable |
| `goenv inventory go` | `--json` | Inventory report | ✅ Stable |
| `goenv doctor` | `--json` | Diagnostic results | ⚠️ Beta |

### Commands Without JSON Support

These commands provide text output only:

- `goenv install` - Installation progress (text)
- `goenv uninstall` - Confirmation prompts (text)
- `goenv use` - Success messages (text)
- `goenv cache` - Cache operations (text)
- `goenv tools` - Tool management (text)

**Note:** Use exit codes for success/failure detection with these commands.

## JSON Schemas

### `goenv list --json`

**Schema:**
```json
[
  {
    "version": "string",
    "path": "string",
    "active": boolean,
    "source": "string"
  }
]
```

**Example:**
```json
[
  {
    "version": "1.23.2",
    "path": "/Users/user/.goenv/versions/1.23.2",
    "active": false,
    "source": ""
  },
  {
    "version": "1.25.2",
    "path": "/Users/user/.goenv/versions/1.25.2",
    "active": true,
    "source": "/Users/user/project/.go-version"
  },
  {
    "version": "system",
    "path": "/usr/local/go",
    "active": false,
    "source": ""
  }
]
```

**Fields:**
- `version` - Version string (e.g., "1.25.2", "system")
- `path` - Absolute path to version directory
- `active` - Whether this version is currently active
- `source` - Where the active version was set (empty if not active)

### `goenv list --remote --json`

**Schema:**
```json
[
  {
    "version": "string",
    "stable": boolean,
    "remote": true
  }
]
```

**Example:**
```json
[
  {
    "version": "1.25.2",
    "stable": true,
    "remote": true
  },
  {
    "version": "1.25.1",
    "stable": true,
    "remote": true
  },
  {
    "version": "1.25rc2",
    "stable": false,
    "remote": true
  }
]
```

**Fields:**
- `version` - Version string available for installation
- `stable` - Whether this is a stable release (not rc, beta, etc.)
- `remote` - Always `true` for remote listings

### `goenv current --json`

**Schema:**
```json
{
  "version": "string",
  "source": "string",
  "path": "string"
}
```

**Example:**
```json
{
  "version": "1.25.2",
  "source": "/Users/user/project/.go-version",
  "path": "/Users/user/.goenv/versions/1.25.2"
}
```

**Fields:**
- `version` - Currently active version
- `source` - File or environment variable that set this version
- `path` - Absolute path to Go installation

### `goenv inventory go --json`

**Schema:**
```json
[
  {
    "version": "string",
    "path": "string",
    "binary_path": "string",
    "installed_at": "string (ISO 8601)",
    "sha256": "string (optional)",
    "os": "string",
    "arch": "string"
  }
]
```

**Example:**
```json
[
  {
    "version": "1.25.2",
    "path": "/Users/user/.goenv/versions/1.25.2",
    "binary_path": "/Users/user/.goenv/versions/1.25.2/bin/go",
    "installed_at": "2025-10-28T14:30:00Z",
    "sha256": "abc123def456...",
    "os": "darwin",
    "arch": "arm64"
  }
]
```

**Fields:**
- `version` - Go version string
- `path` - Installation directory
- `binary_path` - Path to go binary
- `installed_at` - Installation timestamp (ISO 8601)
- `sha256` - Binary checksum (if `--checksums` flag used)
- `os` - Operating system
- `arch` - CPU architecture

### `goenv doctor --json` (Beta)

**Schema:**
```json
{
  "status": "string",
  "checks": [
    {
      "name": "string",
      "status": "string",
      "message": "string"
    }
  ]
}
```

**Example:**
```json
{
  "status": "ok",
  "checks": [
    {
      "name": "goenv_root",
      "status": "ok",
      "message": "GOENV_ROOT is set"
    },
    {
      "name": "go_version",
      "status": "ok",
      "message": "Go 1.25.2 is installed"
    }
  ]
}
```

**Note:** This schema is in beta and may change in future versions.

## Common Use Cases

### 1. Check if Version is Installed

```bash
# Bash
VERSION="1.25.2"
if goenv list --json | jq -e --arg v "$VERSION" '.[] | select(.version == $v)' > /dev/null; then
  echo "Go $VERSION is installed"
else
  echo "Go $VERSION is not installed"
  goenv install "$VERSION"
fi
```

```python
# Python
import json
import subprocess

version = "1.25.2"
result = subprocess.run(["goenv", "list", "--json"], capture_output=True, text=True)
versions = json.loads(result.stdout)

if any(v["version"] == version for v in versions):
    print(f"Go {version} is installed")
else:
    print(f"Installing Go {version}...")
    subprocess.run(["goenv", "install", version])
```

### 2. Get Active Version

```bash
# Bash
ACTIVE_VERSION=$(goenv current --json | jq -r '.version')
echo "Using Go $ACTIVE_VERSION"
```

```javascript
// Node.js
const { execSync } = require('child_process');

const output = execSync('goenv current --json', { encoding: 'utf-8' });
const current = JSON.parse(output);
console.log(`Using Go ${current.version} from ${current.source}`);
```

### 3. List All Stable Remote Versions

```bash
# Bash
goenv list --remote --stable --json | jq -r '.[].version'
```

```python
# Python
import json
import subprocess

result = subprocess.run(
    ["goenv", "list", "--remote", "--stable", "--json"],
    capture_output=True,
    text=True
)
versions = json.loads(result.stdout)
stable_versions = [v["version"] for v in versions if v["stable"]]
print("\n".join(stable_versions))
```

### 4. Generate Inventory Report

```bash
# Bash - CSV format
echo "Version,Path,OS,Arch,SHA256"
goenv inventory go --json --checksums | jq -r \
  '.[] | [.version, .path, .os, .arch, .sha256] | @csv'
```

```python
# Python - Structured report
import json
import subprocess
from datetime import datetime

result = subprocess.run(
    ["goenv", "inventory", "go", "--json", "--checksums"],
    capture_output=True,
    text=True
)
inventory = json.loads(result.stdout)

print("Go Installation Inventory")
print(f"Generated: {datetime.now().isoformat()}\n")

for item in inventory:
    print(f"Version: {item['version']}")
    print(f"  Path: {item['path']}")
    print(f"  Installed: {item['installed_at']}")
    print(f"  Platform: {item['os']}/{item['arch']}")
    print(f"  SHA256: {item['sha256'][:16]}...")
    print()
```

### 5. Find Latest Installed Version

```bash
# Bash
LATEST=$(goenv list --json | jq -r \
  '[.[] | select(.version != "system")] | sort_by(.version) | last | .version')
echo "Latest installed: $LATEST"
```

```python
# Python with semantic versioning
import json
import subprocess
from packaging import version

result = subprocess.run(["goenv", "list", "--json"], capture_output=True, text=True)
versions = json.loads(result.stdout)

go_versions = [
    v["version"] for v in versions
    if v["version"] != "system" and not v["version"].startswith("1.2")
]

latest = max(go_versions, key=lambda v: version.parse(v))
print(f"Latest installed: {latest}")
```

### 6. Detect Version Mismatch

```bash
# Bash - Check if .go-version matches active version
REQUIRED=$(cat .go-version)
ACTIVE=$(goenv current --json | jq -r '.version')

if [ "$REQUIRED" != "$ACTIVE" ]; then
  echo "Warning: Using Go $ACTIVE but project requires $REQUIRED"
  goenv use "$REQUIRED"
fi
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Check Go Version

on: [push, pull_request]

jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install goenv
        run: |
          curl -sfL https://raw.githubusercontent.com/go-nv/goenv/master/install.sh | bash
          echo "$HOME/.goenv/bin" >> $GITHUB_PATH

      - name: Check project version
        id: go-version
        run: |
          VERSION=$(cat .go-version)
          echo "version=$VERSION" >> $GITHUB_OUTPUT

          # Check if installed
          if ! goenv list --json | jq -e --arg v "$VERSION" '.[] | select(.version == $v)' > /dev/null; then
            echo "Installing Go $VERSION..."
            goenv install "$VERSION"
          fi

          goenv use "$VERSION"

      - name: Verify version
        run: |
          CURRENT=$(goenv current --json | jq -r '.version')
          echo "Active Go version: $CURRENT"

          if [ "$CURRENT" != "${{ steps.go-version.outputs.version }}" ]; then
            echo "Error: Version mismatch!"
            exit 1
          fi

      - name: Generate inventory
        run: |
          goenv inventory go --json --checksums > go-inventory.json

      - name: Upload inventory
        uses: actions/upload-artifact@v4
        with:
          name: go-inventory
          path: go-inventory.json
```

### GitLab CI

```yaml
# .gitlab-ci.yml
variables:
  GOENV_OFFLINE: "1"  # Use offline mode for speed

stages:
  - setup
  - build
  - test

setup:
  stage: setup
  script:
    - curl -sfL https://raw.githubusercontent.com/go-nv/goenv/master/install.sh | bash
    - export PATH="$HOME/.goenv/bin:$PATH"

    # Install required version
    - VERSION=$(cat .go-version)
    - |
      if ! goenv list --json | jq -e --arg v "$VERSION" '.[] | select(.version == $v)' > /dev/null; then
        GOENV_OFFLINE=1 goenv install "$VERSION"
      fi

    # Generate inventory
    - goenv inventory go --json > go-inventory.json
  artifacts:
    paths:
      - go-inventory.json
    reports:
      dotenv: build.env
```

### Jenkins

```groovy
// Jenkinsfile
pipeline {
    agent any

    stages {
        stage('Setup Go') {
            steps {
                sh '''
                    # Install goenv if needed
                    if [ ! -d ~/.goenv ]; then
                        curl -sfL https://raw.githubusercontent.com/go-nv/goenv/master/install.sh | bash
                    fi

                    export PATH="$HOME/.goenv/bin:$PATH"

                    # Check installed versions
                    INSTALLED=$(goenv list --json)
                    echo "$INSTALLED" | jq '.'

                    # Install project version
                    VERSION=$(cat .go-version)
                    if ! echo "$INSTALLED" | jq -e --arg v "$VERSION" '.[] | select(.version == $v)' > /dev/null; then
                        goenv install "$VERSION"
                    fi

                    goenv use "$VERSION"

                    # Verify
                    CURRENT=$(goenv current --json)
                    echo "$CURRENT" | jq '.'
                '''
            }
        }
    }
}
```

## Parsing Examples

### jq Examples

```bash
# Get all version numbers
goenv list --json | jq -r '.[].version'

# Get active version only
goenv list --json | jq -r '.[] | select(.active) | .version'

# Get versions without system
goenv list --json | jq -r '.[] | select(.version != "system") | .version'

# Get paths for all versions
goenv list --json | jq -r '.[] | "\(.version): \(.path)"'

# Check if specific version is active
goenv list --json | jq -e '.[] | select(.version == "1.25.2" and .active)' > /dev/null

# Count installed versions
goenv list --json | jq 'length'

# Get versions sorted
goenv list --json | jq 'sort_by(.version)'

# Remote versions only stable
goenv list --remote --json | jq '.[] | select(.stable)'

# Latest stable version
goenv list --remote --stable --json | jq -r '.[0].version'
```

### Python Examples

```python
import json
import subprocess

def get_installed_versions():
    """Get list of installed Go versions."""
    result = subprocess.run(
        ["goenv", "list", "--json"],
        capture_output=True,
        text=True,
        check=True
    )
    return json.loads(result.stdout)

def get_active_version():
    """Get currently active Go version."""
    result = subprocess.run(
        ["goenv", "current", "--json"],
        capture_output=True,
        text=True,
        check=True
    )
    return json.loads(result.stdout)

def is_version_installed(version):
    """Check if a specific version is installed."""
    versions = get_installed_versions()
    return any(v["version"] == version for v in versions)

def get_remote_versions(stable_only=False):
    """Get available versions from remote."""
    cmd = ["goenv", "list", "--remote", "--json"]
    if stable_only:
        cmd.insert(3, "--stable")

    result = subprocess.run(cmd, capture_output=True, text=True, check=True)
    return json.loads(result.stdout)

# Usage
if __name__ == "__main__":
    print("Installed versions:")
    for v in get_installed_versions():
        active = " (active)" if v["active"] else ""
        print(f"  {v['version']}{active}")

    print(f"\nCurrent version: {get_active_version()['version']}")

    if not is_version_installed("1.25.2"):
        print("\nGo 1.25.2 not installed")
```

### Node.js Examples

```javascript
const { execSync } = require('child_process');

function getInstalledVersions() {
  const output = execSync('goenv list --json', { encoding: 'utf-8' });
  return JSON.parse(output);
}

function getCurrentVersion() {
  const output = execSync('goenv current --json', { encoding: 'utf-8' });
  return JSON.parse(output);
}

function isVersionInstalled(version) {
  const versions = getInstalledVersions();
  return versions.some(v => v.version === version);
}

function getRemoteVersions(stableOnly = false) {
  const cmd = stableOnly
    ? 'goenv list --remote --stable --json'
    : 'goenv list --remote --json';
  const output = execSync(cmd, { encoding: 'utf-8' });
  return JSON.parse(output);
}

// Usage
console.log('Installed versions:');
getInstalledVersions().forEach(v => {
  const active = v.active ? ' (active)' : '';
  console.log(`  ${v.version}${active}`);
});

const current = getCurrentVersion();
console.log(`\nCurrent version: ${current.version}`);
console.log(`Source: ${current.source}`);
```

## Error Handling

### Exit Codes

Commands follow standard Unix exit code conventions:

- **0** - Success
- **1** - General error
- **2** - Usage error (invalid arguments)

### JSON Error Format

When JSON output is requested but an error occurs, some commands may output errors to stderr and return non-zero exit code:

```bash
# Success (exit 0, JSON to stdout)
goenv list --json

# Failure (exit 1, error to stderr)
goenv current --json  # When no version is set
```

**Handling errors:**

```bash
# Bash
if OUTPUT=$(goenv current --json 2>&1); then
  VERSION=$(echo "$OUTPUT" | jq -r '.version')
  echo "Active version: $VERSION"
else
  echo "Error: $OUTPUT" >&2
  exit 1
fi
```

```python
# Python
import json
import subprocess

try:
    result = subprocess.run(
        ["goenv", "current", "--json"],
        capture_output=True,
        text=True,
        check=True  # Raises exception on non-zero exit
    )
    current = json.loads(result.stdout)
    print(f"Active version: {current['version']}")
except subprocess.CalledProcessError as e:
    print(f"Error: {e.stderr}", file=sys.stderr)
    sys.exit(1)
except json.JSONDecodeError as e:
    print(f"Invalid JSON: {e}", file=sys.stderr)
    sys.exit(1)
```

### Validation

Always validate JSON structure:

```python
import json
import sys

def safe_parse_goenv_list(output):
    """Safely parse goenv list --json output."""
    try:
        data = json.loads(output)

        # Validate structure
        if not isinstance(data, list):
            raise ValueError("Expected array")

        for item in data:
            required_fields = ["version", "path", "active"]
            if not all(field in item for field in required_fields):
                raise ValueError(f"Missing required fields in: {item}")

        return data
    except (json.JSONDecodeError, ValueError) as e:
        print(f"Error parsing JSON: {e}", file=sys.stderr)
        sys.exit(1)
```

## Best Practices

### 1. Always Validate JSON

```bash
# Bad - no validation
VERSION=$(goenv current --json | jq -r '.version')

# Good - validate output
if OUTPUT=$(goenv current --json 2>&1); then
  VERSION=$(echo "$OUTPUT" | jq -r '.version')
  if [ -z "$VERSION" ]; then
    echo "Error: No version in output"
    exit 1
  fi
else
  echo "Error: $OUTPUT"
  exit 1
fi
```

### 2. Use --bare for Simple Cases

For single-value extraction, `--bare` is simpler than JSON:

```bash
# Simple
VERSION=$(goenv current --bare)

# Overkill
VERSION=$(goenv current --json | jq -r '.version')
```

Use JSON when you need multiple fields or structured data.

### 3. Cache JSON Output

Avoid repeated calls:

```bash
# Bad - calls goenv 3 times
ACTIVE=$(goenv list --json | jq -r '.[] | select(.active) | .version')
PATH=$(goenv list --json | jq -r '.[] | select(.active) | .path')
SOURCE=$(goenv list --json | jq -r '.[] | select(.active) | .source')

# Good - call once, parse multiple times
LIST_JSON=$(goenv list --json)
ACTIVE=$(echo "$LIST_JSON" | jq -r '.[] | select(.active) | .version')
PATH=$(echo "$LIST_JSON" | jq -r '.[] | select(.active) | .path')
SOURCE=$(echo "$LIST_JSON" | jq -r '.[] | select(.active) | .source')
```

### 4. Handle Missing jq Gracefully

```bash
if ! command -v jq &> /dev/null; then
  echo "Error: jq is required for JSON parsing"
  echo "Install: apt-get install jq  # or  brew install jq"
  exit 1
fi
```

## See Also

- [Command Reference](./reference/COMMANDS.md) - All commands and flags
- [CI/CD Guide](./CI_CD_GUIDE.md) - CI/CD integration patterns
- [Automation Examples](./CI_CD_GUIDE.md#automation-examples) - More automation examples
