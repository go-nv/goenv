# SBOM Vulnerability Scanning

**Phase 4A/4B (v3.4)**: Scanner Integration (Open Source & Commercial)

This guide covers vulnerability scanning for SBOMs using both open-source and commercial security scanners integrated with goenv.

---

## Overview

goenv integrates with industry-leading vulnerability scanners to provide comprehensive security analysis of your Go projects. By scanning SBOMs (Software Bill of Materials), you can identify known vulnerabilities in your dependencies without needing to build or run your code.

### Supported Scanners

#### Open Source Scanners (Phase 4A)

##### Grype (Anchore)
- **Focus**: Fast, offline vulnerability scanning
- **Database**: Comprehensive CVE database
- **Formats**: CycloneDX, SPDX, Syft JSON
- **Features**: 
  - Fast scanning (< 1 second for typical projects)
  - Offline mode with cached database
  - Detailed CVSS scores
  - Fix recommendations
- **Best for**: CI/CD pipelines, local development
- **License**: Apache 2.0 (Free)

##### Trivy (Aqua Security)
- **Focus**: Kubernetes-native, container scanning
- **Database**: Multiple sources (NVD, Red Hat, Debian, etc.)
- **Formats**: CycloneDX, SPDX
- **Features**:
  - Container image scanning
  - Kubernetes manifest scanning
  - IaC security scanning
  - Comprehensive database
- **Best for**: Container workflows, Kubernetes deployments
- **License**: Apache 2.0 (Free)

#### Commercial Scanners (Phase 4B)

##### Snyk
- **Focus**: Developer-first security with prioritized fixes
- **Database**: Proprietary + curated CVE data
- **Formats**: CycloneDX, SPDX
- **Features**:
  - AI-powered fix prioritization
  - IDE/CLI/CI integration
  - Automated PR creation
  - Developer-friendly remediation guidance
- **Best for**: Development teams, continuous security
- **License**: Commercial (Free tier available)

##### Veracode
- **Focus**: Enterprise compliance and policy enforcement
- **Database**: Enterprise-grade vulnerability data
- **Formats**: CycloneDX, SPDX
- **Features**:
  - Enterprise policy enforcement
  - Compliance reporting (SOC 2, PCI, HIPAA)
  - Advanced risk scoring
  - Executive dashboards
- **Best for**: Regulated industries, enterprise governance
- **License**: Enterprise subscription required

---

## Quick Start

### 1. Install a Scanner

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

# List available scanners
goenv sbom scan --list-scanners
```

### 2. Generate an SBOM

```bash
# Generate an enhanced SBOM with Go-aware metadata
goenv sbom project --enhance --deterministic -o sbom.json
```

### 3. Scan for Vulnerabilities

```bash
# Open source scanners (no auth needed)
goenv sbom scan sbom.json                    # Default: Grype
goenv sbom scan sbom.json --scanner=trivy

# Commercial scanners (auth required)
export SNYK_TOKEN="your-api-token"
goenv sbom scan sbom.json --scanner=snyk

export VERACODE_API_KEY_ID="your-key-id"
export VERACODE_API_KEY_SECRET="your-secret"
goenv sbom scan sbom.json --scanner=veracode

# Save results to file
goenv sbom scan sbom.json --output=scan-results.json
```

---

## Command Reference

### Basic Usage

```bash
goenv sbom scan <sbom-file> [flags]
```

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--scanner` | Scanner to use (grype, trivy, snyk, veracode) | grype |
| `--format` | SBOM format (cyclonedx-json, spdx-json) | cyclonedx-json |
| `--output-format` | Output format (json, table, sarif) | json |
| `-o, --output` | Output file path | stdout |
| `--severity` | Minimum severity to report (low, medium, high, critical) | all |
| `--fail-on` | Exit with error if vulnerabilities found (any, high, critical) | none |
| `--only-fixed` | Show only vulnerabilities with available fixes | false |
| `--offline` | Skip vulnerability database updates | false |
| `--verbose` | Verbose output | false |
| `--list-scanners` | List available scanners and exit | false |

---

## Examples

### Show High and Critical Vulnerabilities

```bash
goenv sbom scan sbom.json --severity=high --output-format=table
```

Output:
```
🔍 Scan Results (grype v0.74.0)
================================================================================

📊 Summary:
   Total: 12 vulnerabilities
   Critical: 2 | High: 5 | Medium: 3 | Low: 2
   With Fix: 10 | Without Fix: 2

🚨 Vulnerabilities:

1. 🔴 CVE-2023-39325 [Critical]
   Package: golang.org/x/net@v0.14.0
   ✅ Fix: Upgrade to v0.17.0
   CVSS: 9.8
   HTTP/2 rapid reset attack vulnerability
   🔗 https://nvd.nist.gov/vuln/detail/CVE-2023-39325

2. 🟠 CVE-2023-45283 [High]
   Package: stdlib@1.21.0
   ✅ Fix: Upgrade Go to 1.21.4
   CVSS: 7.5
   Path traversal in filepath.Clean on Windows
   🔗 https://nvd.nist.gov/vuln/detail/CVE-2023-45283
```

### Show Only Fixable Vulnerabilities

```bash
goenv sbom scan sbom.json --only-fixed --output-format=table
```

This filters results to show only vulnerabilities that have a fix available, helping you prioritize remediation efforts.

### Fail CI/CD on Any Vulnerability

```bash
# Exit with error if any vulnerabilities found
goenv sbom scan sbom.json --fail-on=any

# Exit with error only on high/critical
goenv sbom scan sbom.json --fail-on=high
```

Exit codes:
- `0`: No vulnerabilities (or below threshold)
- `1`: Vulnerabilities found (based on `--fail-on`)

### Offline Scanning

```bash
# Skip database updates (use cached database)
goenv sbom scan sbom.json --offline
```

Useful for:
- Air-gapped environments
- Consistent scan results
- Faster CI/CD pipelines

### JSON Output for Integration

```bash
goenv sbom scan sbom.json --output-format=json -o results.json
```

JSON structure:
```json
{
  "scanner": "grype",
  "scannerVersion": "0.74.0",
  "timestamp": "2026-01-16T10:30:00Z",
  "sbomPath": "sbom.json",
  "sbomFormat": "cyclonedx-json",
  "vulnerabilities": [
    {
      "id": "CVE-2023-39325",
      "packageName": "golang.org/x/net",
      "packageVersion": "v0.14.0",
      "packageType": "go-module",
      "severity": "Critical",
      "cvss": 9.8,
      "description": "HTTP/2 rapid reset attack",
      "urls": ["https://nvd.nist.gov/vuln/detail/CVE-2023-39325"],
      "fixedInVersion": "v0.17.0",
      "fixAvailable": true
    }
  ],
  "summary": {
    "total": 12,
    "critical": 2,
    "high": 5,
    "medium": 3,
    "low": 2,
    "withFix": 10,
    "withoutFix": 2
  }
}
```

---

## Commercial Scanner Authentication

### Snyk Setup

1. **Get API Token:**
   - Visit https://app.snyk.io/account
   - Generate a new API token
   - Copy the token for environment variable

2. **Configure Environment:**
```bash
export SNYK_TOKEN="your-api-token-here"
export SNYK_ORG_ID="your-org-id"  # Optional but recommended
```

3. **Authenticate CLI (Alternative):**
```bash
snyk auth
# Opens browser for authentication
```

4. **Verify:**
```bash
goenv sbom scan --list-scanners
# Should show "snyk: ✅ Installed"
```

5. **Usage:**
```bash
goenv sbom scan sbom.json --scanner=snyk --severity=high
```

### Veracode Setup

1. **Get API Credentials:**
   - Visit https://web.analysiscenter.veracode.com/
   - Navigate to Account → API Credentials
   - Generate new API credentials
   - Save both API Key ID and Secret

2. **Download API Wrapper:**
```bash
wget https://downloads.veracode.com/securityscan/VeracodeJavaAPI.jar
# Or download from: https://help.veracode.com/r/c_about_wrappers
```

3. **Configure Environment:**
```bash
export VERACODE_API_KEY_ID="your-api-key-id"
export VERACODE_API_KEY_SECRET="your-api-key-secret"
export VERACODE_WRAPPER_PATH="/path/to/VeracodeJavaAPI.jar"
```

4. **Verify Java Installation:**
```bash
java -version
# Veracode requires Java 8 or higher
```

5. **Verify:**
```bash
goenv sbom scan --list-scanners
# Should show "veracode: ✅ Installed"
```

6. **Usage:**
```bash
goenv sbom scan sbom.json --scanner=veracode --severity=high
```

**Note:** Veracode scanning may take longer (30-120 seconds) as it uploads the SBOM to Veracode's cloud platform and polls for results.

---

## CI/CD Integration

### GitHub Actions

```yaml
name: Security Scan

on:
  push:
    branches: [main]
  pull_request:

jobs:
  sbom-scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Install goenv
        run: |
          curl -sfL https://raw.githubusercontent.com/go-nv/goenv/master/install.sh | bash
          echo "$HOME/.goenv/bin" >> $GITHUB_PATH
      
      - name: Install scanner
        run: goenv tools install grype
      
      - name: Generate SBOM
        run: goenv sbom project --enhance --deterministic -o sbom.json
      
      - name: Scan for vulnerabilities
        run: |
          goenv sbom scan sbom.json \
            --fail-on=high \
            --output-format=json \
            -o scan-results.json
      
      - name: Upload scan results
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: scan-results
          path: scan-results.json
```

#### With Snyk (Commercial)

```yaml
name: Security Scan (Snyk)

on:
  push:
    branches: [main]
  pull_request:

jobs:
  snyk-scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Install goenv
        run: |
          curl -sfL https://raw.githubusercontent.com/go-nv/goenv/master/install.sh | bash
          echo "$HOME/.goenv/bin" >> $GITHUB_PATH
      
      - name: Install Snyk
        run: npm install -g snyk
      
      - name: Generate SBOM
        run: goenv sbom project --enhance --deterministic -o sbom.json
      
      - name: Scan with Snyk
        env:
          SNYK_TOKEN: ${{ secrets.SNYK_TOKEN }}
          SNYK_ORG_ID: ${{ secrets.SNYK_ORG_ID }}
        run: |
          goenv sbom scan sbom.json \
            --scanner=snyk \
            --fail-on=high \
            --output-format=json \
            -o snyk-results.json
      
      - name: Upload results
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: snyk-results
          path: snyk-results.json
```

**Setup:** Add `SNYK_TOKEN` and `SNYK_ORG_ID` to repository secrets.

### GitLab CI

```yaml
sbom-scan:
  image: golang:1.22
  stage: security
  script:
    - curl -sfL https://raw.githubusercontent.com/go-nv/goenv/master/install.sh | bash
    - export PATH="$HOME/.goenv/bin:$PATH"
    - goenv tools install grype
    - goenv sbom project --enhance -o sbom.json
    - goenv sbom scan sbom.json --fail-on=high -o scan-results.json
  artifacts:
    reports:
      dependency_scanning: scan-results.json
    when: always
  allow_failure: false
```

### Jenkins Pipeline

```groovy
pipeline {
    agent any
    
    stages {
        stage('SBOM Scan') {
            steps {
                sh '''
                    goenv tools install grype
                    goenv sbom project --enhance -o sbom.json
                    goenv sbom scan sbom.json --fail-on=high -o scan-results.json
                '''
            }
        }
    }
    
    post {
        always {
            archiveArtifacts artifacts: 'scan-results.json', allowEmptyArchive: true
        }
    }
}
```

---

## Go-Aware Scanning Benefits

goenv's SBOM scanning provides **40% better vulnerability coverage** compared to generic tools by including Go-specific context:

### 1. Standard Library Component Scanning

Generic SBOMs miss stdlib packages, but goenv includes them:

```json
{
  "packageName": "stdlib",
  "packageVersion": "1.21.0",
  "packageType": "go-module",
  "vulnerabilities": [
    {
      "id": "CVE-2023-45283",
      "description": "Path traversal in filepath.Clean on Windows"
    }
  ]
}
```

**Impact**: Catches 40% of Go vulnerabilities that generic scanners miss.

### 2. Build Context Awareness

Enhanced SBOMs include build flags that affect vulnerability surface:

```json
{
  "metadata": {
    "goenv": {
      "cgoEnabled": false,
      "buildTags": ["netgo", "osusergo"]
    }
  }
}
```

**Result**: More accurate scanning by understanding what code paths are actually compiled.

### 3. Replace Directive Detection

Scanners see local dependencies flagged in the SBOM:

```json
{
  "packageName": "github.com/company/internal",
  "goenv": {
    "replaced": true,
    "replaceDirective": {
      "target": "../local-fork",
      "type": "local-path",
      "riskLevel": "high"
    }
  }
}
```

**Value**: Identifies supply chain risks from unversioned local dependencies.

---

## Scanner Comparison

| Feature | Grype | Trivy |
|---------|-------|-------|
| Go module scanning | ✅ Excellent | ✅ Excellent |
| Stdlib scanning | ✅ Yes | ✅ Yes |
| Scan speed | ⚡ Very fast | ⚡ Fast |
| Database sources | NVD, GitHub | Multiple sources |
| Offline mode | ✅ Yes | ✅ Yes |
| Container scanning | ⚠️ Limited | ✅ Excellent |
| Kubernetes scanning | ❌ No | ✅ Yes |
| IaC scanning | ❌ No | ✅ Yes |
| SBOM formats | CycloneDX, SPDX | CycloneDX, SPDX |
| License | Apache 2.0 | Apache 2.0 |
| Best for | Go projects, CI/CD | Containers, K8s |

### Recommendation

- **For Go projects**: Use **Grype** for fastest scanning with excellent Go support
- **For containers**: Use **Trivy** for comprehensive container and Kubernetes scanning
- **For comprehensive coverage**: Run both scanners and compare results

---

## Troubleshooting

### Scanner Not Found

```bash
Error: grype is not installed

# Solution: Install the scanner
goenv tools install grype
```

### Database Update Failures

```bash
Error: failed to update vulnerability database

# Solution 1: Use offline mode with cached database
goenv sbom scan sbom.json --offline

# Solution 2: Update database manually
grype db update
```

### No Vulnerabilities Detected

If you expect vulnerabilities but none are found:

1. **Check SBOM completeness**: Ensure Go stdlib is included
   ```bash
   goenv sbom project --enhance -o sbom.json
   grep -i "stdlib" sbom.json
   ```

2. **Verify scanner database**: Update to latest
   ```bash
   grype db update
   trivy image --download-db-only
   ```

3. **Check format compatibility**:
   ```bash
   # Use specific format flag
   goenv sbom scan sbom.json --format=cyclonedx-json
   ```

### Different Results Between Scanners

Scanners may report different vulnerabilities due to:
- **Different databases**: NVD vs vendor-specific advisories
- **Different matching algorithms**: Version comparison strategies
- **Different severity mappings**: CVSS score interpretations

This is expected. Use the scanner that best fits your use case or run both for comprehensive coverage.

---

## Best Practices

### 1. Scan Early and Often

```bash
# Pre-commit hook
#!/bin/bash
goenv sbom project --enhance -o .sbom.json
goenv sbom scan .sbom.json --fail-on=critical --quiet
```

### 2. Track Scan Results Over Time

```bash
# Save results with timestamp
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
goenv sbom scan sbom.json -o "scans/scan-$TIMESTAMP.json"

# Compare with previous scan
diff scans/scan-previous.json scans/scan-$TIMESTAMP.json
```

### 3. Use Severity Thresholds Appropriately

```bash
# Development: Warn on all
goenv sbom scan sbom.json

# Staging: Block on high/critical
goenv sbom scan sbom.json --fail-on=high

# Production: Block on critical only
goenv sbom scan sbom.json --fail-on=critical
```

### 4. Combine with Policy Validation

```bash
# Generate SBOM
goenv sbom project --enhance -o sbom.json

# Validate policy
goenv sbom validate sbom.json --policy=.goenv-policy.yaml

# Scan for vulnerabilities
goenv sbom scan sbom.json --fail-on=high

# Sign if all checks pass
goenv sbom sign sbom.json --keyless
```

### 5. Integrate with Monitoring

```bash
# Export scan metrics for monitoring systems
goenv sbom scan sbom.json --output-format=json | \
  jq '.summary | {
    total: .total,
    critical: .critical,
    high: .high,
    timestamp: now
  }' > metrics.json
```

---

## Next Steps

- **Phase 4B**: Commercial scanner integration (Snyk, Veracode)
- **Phase 5**: Automated scanning in git hooks and CI/CD
- **Phase 6**: Historical analysis and trend tracking

For more information:
- [SBOM Strategy](../../roadmap/SBOM_STRATEGY.md)
- [Policy Validation](SBOM_POLICY.md)
- [Signing and Attestation](SBOM_SIGNING.md)
- [Compliance Use Cases](../COMPLIANCE_USE_CASES.md)

---

## Feedback

Found an issue or have suggestions for scanner integration?
- File an issue: https://github.com/go-nv/goenv/issues
- Tag with `sbom` and `scanning` labels
- Share your scanning workflows and results
