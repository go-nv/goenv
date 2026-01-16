# Scanner Tool Installation Enhancement

**Date:** January 16, 2026  
**Feature:** Added CLI tool installation support for security scanners  
**Related:** Phase 4A/4B SBOM Scanner Integration

---

## Overview

Enhanced `goenv tools install` to support security scanner CLIs, making it easier for developers to install and manage vulnerability scanners alongside their Go development tools.

---

## Changes Made

### 1. Tools Registry Update

**File:** `internal/tools/utils.go`

Added Snyk CLI to the common tools registry:

```go
var commonTools = map[string]string {
    // ... existing tools ...
    
    // Security scanners for SBOM vulnerability analysis (Phase 4A)
    "grype":  "github.com/anchore/grype/cmd/grype",
    "trivy":  "github.com/aquasecurity/trivy/cmd/trivy",
    
    // Commercial security scanners (Phase 4B)
    "snyk":   "github.com/snyk/cli/cmd/snyk",
}
```

**Rationale:**
- Snyk CLI is a Go-based tool that can be installed via `go install`
- Fits naturally into goenv's tool management system
- Provides consistent installation experience across scanners

### 2. Installation Instructions Updates

#### Snyk Scanner (`internal/sbom/snyk.go`)

Updated installation instructions to prioritize `goenv tools install`:

```go
func (s *SnykScanner) InstallationInstructions() string {
    return `Snyk CLI Installation:

1. Using goenv (recommended):
   goenv tools install snyk

2. Using npm:
   npm install -g snyk

3. Using Homebrew (macOS):
   brew install snyk/tap/snyk

4. Using binary:
   Download from https://github.com/snyk/cli/releases

5. Authenticate:
   snyk auth
   # Or set environment variables:
   export SNYK_TOKEN="your-api-token"
   export SNYK_ORG_ID="your-org-id"  # Optional but recommended
   ...
```

#### Veracode Scanner (`internal/sbom/veracode.go`)

Clarified that Veracode uses a Java-based wrapper (not a Go CLI):

```go
func (v *VeracodeScanner) InstallationInstructions() string {
    return `Veracode Setup:

Note: Veracode uses a Java-based API wrapper (not available via 'goenv tools install')

1. Install Java (required):
   # macOS
   brew install openjdk
   ...
```

### 3. Documentation Updates

#### User Guide (`docs/user-guide/SBOM_SCANNING.md`)

Updated installation section:

```bash
# Open Source Scanners (no authentication required)
goenv tools install grype    # Recommended for Go projects
goenv tools install trivy

# Commercial Scanners
goenv tools install snyk     # Snyk CLI (authentication required)
# Note: Veracode uses Java API wrapper, not available via 'goenv tools install'
#       Download from: https://downloads.veracode.com/securityscan/VeracodeJavaAPI.jar

# Alternative installation methods
npm install -g snyk          # Snyk via npm
brew install snyk/tap/snyk   # Snyk via Homebrew
```

#### Phase 4 Completion (`docs/roadmap/PHASE_4_COMPLETION.md`)

Added tool installation section showing all available methods.

---

## Supported Scanners via `goenv tools install`

| Scanner | Command | Type | Authentication |
|---------|---------|------|----------------|
| Grype | `goenv tools install grype` | Open Source | None required |
| Trivy | `goenv tools install trivy` | Open Source | None required |
| Snyk | `goenv tools install snyk` | Commercial | Token required |
| Veracode | Manual (Java wrapper) | Enterprise | API credentials |

---

## Usage Examples

### Install All Open Source Scanners

```bash
goenv tools install grype trivy
```

### Install Commercial Scanner

```bash
# Install Snyk CLI
goenv tools install snyk

# Authenticate
export SNYK_TOKEN="your-token-here"
snyk auth  # Browser-based authentication

# Verify installation
snyk --version
```

### Full Workflow

```bash
# 1. Install scanner
goenv tools install grype

# 2. Generate SBOM
goenv sbom project --enhance -o sbom.json

# 3. Scan for vulnerabilities
goenv sbom scan sbom.json

# 4. Install additional scanners as needed
goenv tools install snyk
export SNYK_TOKEN="..."
goenv sbom scan sbom.json --scanner=snyk
```

### Cross-Version Installation

```bash
# Install scanner for all Go versions
goenv tools install snyk --all

# Install for specific version
goenv use 1.22.0
goenv tools install snyk

# List installed tools
goenv tools list --all
```

---

## Why Not Veracode?

Veracode is **not** available via `goenv tools install` because:

1. **Not a Go tool**: Veracode uses a Java-based API wrapper (`VeracodeJavaAPI.jar`)
2. **Java runtime required**: Requires JVM to execute
3. **Manual download**: Must be downloaded from Veracode's secure portal
4. **Enterprise licensing**: Requires authentication before download
5. **Different paradigm**: Jar file, not a standalone binary

**Alternative approach:**
```bash
# Download Veracode wrapper
wget https://downloads.veracode.com/securityscan/VeracodeJavaAPI.jar

# Place in standard location
mkdir -p $HOME/.veracode
mv VeracodeJavaAPI.jar $HOME/.veracode/

# Configure environment
export VERACODE_API_KEY_ID="..."
export VERACODE_API_KEY_SECRET="..."
export VERACODE_WRAPPER_PATH="$HOME/.veracode/VeracodeJavaAPI.jar"

# Verify
java -jar $VERACODE_WRAPPER_PATH -version

# Use with goenv
goenv sbom scan sbom.json --scanner=veracode
```

---

## Benefits

### 1. Unified Installation Experience

Developers can install all Go-based security tools using the same command:

```bash
goenv tools install gopls golangci-lint staticcheck grype trivy snyk
```

### 2. Version Isolation

Tools are installed per Go version, preventing conflicts:

```
$GOENV_ROOT/
├── versions/
│   ├── 1.21.0/
│   │   └── pkgs/darwin_arm64/
│   │       └── bin/
│   │           ├── grype
│   │           ├── snyk
│   │           └── trivy
│   └── 1.22.0/
│       └── pkgs/darwin_arm64/
│           └── bin/
│               ├── grype
│               ├── snyk
│               └── trivy
```

### 3. Consistent Updates

```bash
# Check for scanner updates
goenv tools outdated

# Update all scanners
goenv tools update grype trivy snyk
```

### 4. Team Standardization

Teams can standardize on scanner versions:

```yaml
# .goenv/default-tools.yaml
tools:
  - name: grype
    package: github.com/anchore/grype/cmd/grype
    version: "@latest"
  
  - name: trivy
    package: github.com/aquasecurity/trivy/cmd/trivy
    version: "@latest"
  
  - name: snyk
    package: github.com/snyk/cli/cmd/snyk
    version: "@latest"
```

---

## Scanner Package Paths

| Scanner | Go Package Path | Repository |
|---------|----------------|------------|
| Grype | `github.com/anchore/grype/cmd/grype` | https://github.com/anchore/grype |
| Trivy | `github.com/aquasecurity/trivy/cmd/trivy` | https://github.com/aquasecurity/trivy |
| Snyk | `github.com/snyk/cli/cmd/snyk` | https://github.com/snyk/cli |

---

## Testing

### Verification Commands

```bash
# Build goenv
make build

# Test Snyk package path normalization
./goenv tools install snyk --dry-run

# Install and verify
goenv tools install snyk
snyk --version

# List installed
goenv tools list
```

### Expected Output

```
$ goenv tools install snyk
Installing tools for Go 1.22.0...
  ✓ snyk@latest

$ snyk --version
1.1042.0

$ goenv tools list
Installed tools for Go 1.22.0:
  ✓ gopls
  ✓ golangci-lint
  ✓ staticcheck
  ✓ grype
  ✓ trivy
  ✓ snyk
```

---

## Migration Guide

### Before (Phase 4A/4B Initial Release)

```bash
# Install Grype
brew install grype

# Install Trivy  
brew install trivy

# Install Snyk
npm install -g snyk
```

### After (Tool Installation Enhancement)

```bash
# Install all via goenv
goenv tools install grype trivy snyk

# Or install individually
goenv tools install snyk
```

### Benefits of Migration

1. **Version consistency** across team members
2. **No global npm/brew dependencies**
3. **Isolated per Go version**
4. **Unified update management**
5. **Supports offline/air-gapped environments** (after initial download)

---

## Implementation Status

- ✅ Added Snyk to tools registry
- ✅ Updated Snyk installation instructions
- ✅ Clarified Veracode manual installation
- ✅ Updated user guide documentation
- ✅ Updated Phase 4 completion documentation
- ✅ Build verification successful
- ✅ Maintains backward compatibility (npm/brew still work)

---

## Future Enhancements

### Potential Additions

1. **Veracode Pipeline Scanner**
   - If Veracode releases a standalone binary (non-Java)
   - Could be added to tools registry

2. **Other Scanners**
   - OSV Scanner (Google): `github.com/google/osv-scanner/cmd/osv-scanner`
   - Dependency Track CLI: If available as Go tool
   - GitHub Advisory Database Scanner: If available

3. **Auto-installation Hooks**
   - Auto-install scanner when running `goenv sbom scan` if missing
   - Interactive prompt: "Scanner 'grype' not found. Install now? [Y/n]"

4. **Scanner Version Pinning**
   ```yaml
   # Pin specific scanner versions for compliance
   tools:
     - name: grype
       version: "@v0.75.0"  # Pin for audit trail
   ```

---

## Conclusion

This enhancement makes security scanning more accessible by integrating scanner installation into goenv's existing tool management system. Developers can now install, update, and manage security scanners alongside their development tools using familiar commands.

**Key Achievement:**
- Unified installation experience for 3 out of 4 scanners
- Maintains clear documentation for Veracode's Java-based approach
- No breaking changes to existing installations
- Improved developer experience for security scanning workflows

**Commands Summary:**
```bash
# Install scanners
goenv tools install grype trivy snyk

# Generate SBOM
goenv sbom project --enhance -o sbom.json

# Scan with any scanner
goenv sbom scan sbom.json --scanner=grype
goenv sbom scan sbom.json --scanner=trivy
goenv sbom scan sbom.json --scanner=snyk

# Manage tools
goenv tools list
goenv tools outdated
goenv tools update grype trivy snyk
```
