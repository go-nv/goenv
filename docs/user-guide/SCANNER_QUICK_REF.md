# Scanner Installation Quick Reference

Quick commands for installing and using security scanners with goenv.

## Installation

### All Open Source Scanners
```bash
goenv tools install grype trivy
```

### Add Commercial Scanner
```bash
goenv tools install snyk
```

### Install for All Go Versions
```bash
goenv tools install grype trivy snyk --all
```

## Verification

```bash
# List installed tools
goenv tools list

# Check versions
grype version
trivy --version
snyk --version

# Check if installed
goenv tools status
```

## Authentication

### Snyk
```bash
# Set token
export SNYK_TOKEN="your-token"

# Or browser auth
snyk auth

# Test
snyk test --help
```

### Veracode (Manual Installation)
```bash
# Download wrapper
mkdir -p $HOME/.veracode
wget https://downloads.veracode.com/securityscan/VeracodeJavaAPI.jar \
  -O $HOME/.veracode/VeracodeJavaAPI.jar

# Set credentials
export VERACODE_API_KEY_ID="your-key-id"
export VERACODE_API_KEY_SECRET="your-secret"
export VERACODE_WRAPPER_PATH="$HOME/.veracode/VeracodeJavaAPI.jar"

# Test
java -jar $VERACODE_WRAPPER_PATH -version
```

#### Alternate instructions

See the [Veracode CLI installation guide](https://docs.veracode.com/r/Install_the_Veracode_CLI).

## Usage Workflow

```bash
# 1. Install scanner
goenv tools install grype

# 2. Generate SBOM
goenv sbom project --enhance -o sbom.json

# 3. Scan
goenv sbom scan sbom.json

# 4. Try different scanners
goenv tools install trivy snyk
goenv sbom scan sbom.json --scanner=trivy
goenv sbom scan sbom.json --scanner=snyk --severity=high
```

## Updates

```bash
# Check for updates
goenv tools outdated

# Update specific scanner
goenv tools update grype

# Update all
goenv tools update grype trivy snyk
```

## Troubleshooting

### Scanner Not Found
```bash
# Check if installed
which grype

# Reinstall
goenv tools uninstall grype
goenv tools install grype
```

### Snyk Authentication Error
```bash
# Check token
echo $SNYK_TOKEN

# Re-authenticate
snyk auth

# Test connection
snyk test --org=your-org-id
```

### Version Conflicts
```bash
# List tools across versions
goenv tools list --all

# Sync tools between versions
goenv tools sync-tools 1.21.0 1.22.0
```

## Team Setup

Add to `.goenv/default-tools.yaml`:

```yaml
enabled: true
update_strategy: auto

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

Then team members just run:
```bash
goenv install 1.22.0  # Auto-installs tools
```

## CI/CD Examples

### GitHub Actions
```yaml
- name: Setup Go and scanners
  run: |
    goenv install 1.22.0
    goenv use 1.22.0
    goenv tools install grype snyk

- name: Scan with Grype
  run: |
    goenv sbom project -o sbom.json
    goenv sbom scan sbom.json --fail-on=high

- name: Scan with Snyk
  env:
    SNYK_TOKEN: ${{ secrets.SNYK_TOKEN }}
  run: |
    goenv sbom scan sbom.json --scanner=snyk --fail-on=high
```

### GitLab CI
```yaml
security_scan:
  script:
    - goenv install 1.22.0
    - goenv tools install grype trivy
    - goenv sbom project -o sbom.json
    - goenv sbom scan sbom.json --scanner=grype
    - goenv sbom scan sbom.json --scanner=trivy
```

## Comparison Matrix

| Scanner | Install Command | Auth Required | License | Best For |
|---------|----------------|---------------|---------|----------|
| Grype | `goenv tools install grype` | No | Free | CI/CD pipelines |
| Trivy | `goenv tools install trivy` | No | Free | Container workflows |
| Snyk | `goenv tools install snyk` | Yes | Freemium | Dev teams |
| Veracode | Manual (Java) | Yes | Enterprise | Compliance |
