# Compliance and Security Use Cases

This guide demonstrates how to use goenv for compliance audits, security scanning, and regulatory requirements.

## Table of Contents

- [Inventory Management](#inventory-management)
- [SBOM Generation](#sbom-generation)
- [SOC 2 Compliance](#soc-2-compliance)
- [ISO 27001 Asset Management](#iso-27001-asset-management)
- [Vulnerability Management](#vulnerability-management)
- [Change Management](#change-management)
- [License Compliance](#license-compliance)
- [Multi-Environment Consistency](#multi-environment-consistency)

## Inventory Management

### Basic Inventory

```bash
# List all installed Go versions
goenv inventory go

# JSON output for automation
goenv inventory go --json

# Include SHA256 checksums for verification
goenv inventory go --checksums --json
```

### Automated Inventory Snapshots

```bash
# Monthly inventory snapshot
mkdir -p audits/$(date +%Y-%m)
goenv inventory go --json --checksums > audits/$(date +%Y-%m)/go-inventory.json

# Track in version control
git add audits/
git commit -m "Go inventory snapshot - $(date +%Y-%m-%d)"
```

### Inventory with Hooks

Automatically track all installations:

```yaml
# ~/.goenv/hooks.yaml
hooks:
  post_install:
    - action: run_command
      params:
        command: "goenv"
        args: ["inventory", "go", "--json"]
        capture_output: true

    - action: log_to_file
      params:
        path: ~/.goenv/audit/installations.log
        message: "[{timestamp}] Installed Go {version}"
```

## SBOM Generation

### Generate CycloneDX SBOM

```bash
# Install SBOM tool (once per Go version)
goenv tools install cyclonedx-gomod@v1.6.0

# Generate SBOM for your project
goenv sbom project \
  --tool=cyclonedx-gomod \
  --format=cyclonedx-json \
  --output=sbom.cdx.json

# Verify SBOM was generated
jq '.bomFormat' sbom.cdx.json
```

### Generate SPDX SBOM

```bash
# Install Syft (once per Go version)
goenv tools install syft@v1.0.0

# Generate SPDX SBOM
goenv sbom project \
  --tool=syft \
  --format=spdx-json \
  --output=sbom.spdx.json
```

### Automated SBOM Generation with Hooks

```yaml
# ~/.goenv/hooks.yaml
hooks:
  post_install:
    # Generate SBOM after each Go installation
    - action: run_command
      params:
        command: "goenv"
        args:
          - "sbom"
          - "project"
          - "--tool=cyclonedx-gomod"
          - "--output=/var/audit/sbom-{version}.json"
        timeout: "2m"
        fail_on_error: false

    - action: log_to_file
      params:
        path: ~/.goenv/sbom-log.txt
        message: "[{timestamp}] Generated SBOM for Go {version}"
```

## SOC 2 Compliance

### Control: Software Inventory (CC7.1)

**Requirement:** Maintain a current inventory of authorized software installations.

```bash
# Generate monthly inventory report
#!/bin/bash
YEAR_MONTH=$(date +%Y-%m)
REPORT_DIR="audits/soc2/$YEAR_MONTH"
mkdir -p "$REPORT_DIR"

# Go installations inventory
goenv inventory go --json --checksums > "$REPORT_DIR/go-inventory.json"

# Generate human-readable report
cat > "$REPORT_DIR/inventory-report.md" <<EOF
# SOC 2 Software Inventory Report
**Report Date:** $(date +%Y-%m-%d)
**Report Period:** $YEAR_MONTH

## Go Installations

$(goenv inventory go)

## Verification
All installations have been verified with SHA256 checksums.
EOF

# Commit to audit trail
git add audits/soc2/
git commit -m "SOC 2 inventory report - $YEAR_MONTH"
```

### Control: Change Management (CC8.1)

**Requirement:** Document and track software changes.

```yaml
# ~/.goenv/hooks.yaml - SOC 2 change tracking
hooks:
  post_install:
    - action: log_to_file
      params:
        path: /var/audit/change-log.txt
        message: "[{timestamp}] INSTALL: Go {version} by ${USER} on ${HOSTNAME}"

    - action: run_command
      params:
        command: "git"
        args:
          - "-C"
          - "/var/audit"
          - "add"
          - "change-log.txt"

    - action: run_command
      params:
        command: "git"
        args:
          - "-C"
          - "/var/audit"
          - "commit"
          - "-m"
          - "Go {version} installed - ${USER}@${HOSTNAME}"

  post_uninstall:
    - action: log_to_file
      params:
        path: /var/audit/change-log.txt
        message: "[{timestamp}] UNINSTALL: Go {version} by ${USER} on ${HOSTNAME}"
```

## ISO 27001 Asset Management

### A.8.1 - Inventory of Assets

```bash
# Generate asset register entry
goenv inventory go --json | jq '.[] | {
  asset_id: ("GO-" + .version),
  asset_type: "Development Tool",
  asset_name: "Go Programming Language",
  version: .version,
  location: .path,
  owner: "$USER",
  installed_date: .installed_at,
  verification: .sha256,
  platform: (.os + "/" + .arch),
  criticality: "Medium",
  status: "Active"
}' > asset-register-go.json
```

### A.8.2 - Information Classification

```bash
# Document Go toolchain as "Internal Use" asset
cat > go-asset-classification.yaml <<EOF
asset_class: Development Toolchain
classification: Internal Use
data_processed: Source Code
security_requirements:
  - integrity: HIGH
  - availability: MEDIUM
  - confidentiality: LOW
compliance_notes: |
  Go compiler processes proprietary source code.
  Versions tracked via goenv for auditability.
installed_versions:
$(goenv list --bare | sed 's/^/  - /')
verification_method: SHA256 checksums via 'goenv inventory go --checksums'
EOF
```

## Vulnerability Management

### Identify Vulnerable Versions

```bash
# List all installed versions
VERSIONS=$(goenv list --bare)

# Check each against vulnerability database
echo "# Go Version Vulnerability Scan - $(date)" > vuln-report.md
echo "" >> vuln-report.md

for version in $VERSIONS; do
  echo "## Go $version" >> vuln-report.md

  # Example: Check against govulncheck
  GOENV_VERSION=$version goenv exec govulncheck -version &>/dev/null && \
    echo "- govulncheck available" >> vuln-report.md || \
    echo "- ⚠️  govulncheck not installed" >> vuln-report.md

  # Add your vulnerability scanning here
  # Example: curl https://vuln-db.example.com/api/check?version=$version

  echo "" >> vuln-report.md
done

cat vuln-report.md
```

### Integration with Security Scanners

```bash
# Export inventory for Snyk
goenv inventory go --json | jq -r '.[] |
  "golang:\(.version),\(.binary_path),\(.sha256)"' \
  > snyk-import.csv

# Export for Trivy
goenv inventory go --json | jq -r '.[] |
  {
    name: "golang",
    version: .version,
    location: .binary_path,
    digest: .sha256
  }' > trivy-manifest.json

# Export for Grype
goenv inventory go --json | jq -r '.[] |
  "pkg:golang/go@\(.version)"' \
  > grype-packages.txt
```

### Automated Vulnerability Reporting

```bash
#!/bin/bash
# Daily vulnerability check

REPORT_DIR="security/$(date +%Y-%m-%d)"
mkdir -p "$REPORT_DIR"

# Generate inventory
goenv inventory go --json > "$REPORT_DIR/inventory.json"

# Check for known CVEs (example with fictional API)
for version in $(goenv list --bare); do
  echo "Checking Go $version..."
  # curl "https://cve-api.example.com/check?product=golang&version=$version" \
  #   >> "$REPORT_DIR/cve-findings.json"
done

# Alert if vulnerabilities found
# if jq -e '.vulnerabilities | length > 0' "$REPORT_DIR/cve-findings.json"; then
#   # Send alert (email, Slack, PagerDuty, etc.)
#   echo "ALERT: Vulnerabilities detected in installed Go versions"
# fi
```

## Change Management

### Pre-Change Documentation

```bash
# Before making changes (e.g., Go version upgrade)
cat > change-request-$(date +%Y%m%d).md <<EOF
# Change Request - Go Version Upgrade

**Date:** $(date +%Y-%m-%d)
**Requested By:** ${USER}
**Change Type:** Software Upgrade

## Current State

### Installed Go Versions
$(goenv inventory go)

### Active Version
$(goenv current)

## Proposed Changes

- Install Go 1.25.2
- Update projects to use Go 1.25.2

## Rollback Plan

Current installation backed up to: ~/.goenv/backups/pre-upgrade-$(date +%Y%m%d)/

## Approval

- [ ] Technical Lead
- [ ] Security Review
- [ ] Change Advisory Board

EOF
```

### Post-Change Verification

```bash
# After changes
cat >> change-request-$(date +%Y%m%d).md <<EOF

## Post-Change Verification

### New Installation State
$(goenv inventory go)

### Verification Steps
- [x] Go 1.25.2 installed successfully
- [x] All tests pass with new version
- [x] Production build successful

### Issues Encountered
None

**Change Status:** COMPLETED
**Completion Date:** $(date +%Y-%m-%d %H:%M:%S)
EOF
```

## License Compliance

### Verify Go License

Go is released under a BSD-style license. Verify license compliance:

```bash
# Check license file
for version in $(goenv list --bare); do
  LICENSE_FILE="$HOME/.goenv/versions/$version/LICENSE"
  if [ -f "$LICENSE_FILE" ]; then
    echo "Go $version: License found"
    head -1 "$LICENSE_FILE"
  else
    echo "Go $version: ⚠️  License file not found"
  fi
done
```

### Generate License Report

```bash
# License compliance report
cat > license-report.md <<EOF
# Go License Compliance Report

**Generated:** $(date +%Y-%m-%d)

## License Summary

Go is released under a BSD-style license (https://go.dev/LICENSE).

## Installed Versions

$(goenv inventory go)

## License Text

$(cat ~/.goenv/versions/$(goenv current --bare)/LICENSE)

## Compliance Status

✅ All installed Go versions comply with BSD-style license terms.
✅ No additional licensing fees required.
✅ Attribution requirements met in project documentation.

EOF
```

## Multi-Environment Consistency

### Compare Across Environments

```bash
#!/bin/bash
# Check Go version consistency across environments

ENVIRONMENTS=("dev-server" "staging-server" "prod-server")

for env in "${ENVIRONMENTS[@]}"; do
  echo "=== $env ===" > "$env-inventory.json"
  ssh $env "goenv inventory go --json" >> "$env-inventory.json"
done

# Compare versions
echo "# Environment Consistency Report" > consistency-report.md
echo "" >> consistency-report.md

for env in "${ENVIRONMENTS[@]}"; do
  echo "## $env" >> consistency-report.md
  jq -r '.[].version' "$env-inventory.json" | sort >> consistency-report.md
  echo "" >> consistency-report.md
done

# Check for discrepancies
echo "## Discrepancies" >> consistency-report.md
diff -u dev-server-inventory.json staging-server-inventory.json || \
  echo "⚠️  Dev and Staging have different versions"
```

### Sync Environments

```bash
#!/bin/bash
# Ensure all environments have required versions

REQUIRED_VERSIONS=("1.24.8" "1.25.2")

for env in dev-server staging-server prod-server; do
  echo "Checking $env..."
  for version in "${REQUIRED_VERSIONS[@]}"; do
    if ! ssh $env "goenv installed $version &>/dev/null"; then
      echo "Installing Go $version on $env..."
      ssh $env "goenv install $version"
    else
      echo "✓ $env has Go $version"
    fi
  done
done
```

## Automated Compliance Reporting

### CI/CD Integration

```yaml
# .github/workflows/compliance-audit.yml
name: Compliance Audit

on:
  schedule:
    - cron: '0 0 1 * *'  # Monthly on the 1st
  workflow_dispatch:      # Manual trigger

jobs:
  audit:
    runs-on: ubuntu-latest
    steps:
      - name: Setup goenv
        run: |
          curl -sfL https://raw.githubusercontent.com/go-nv/goenv/master/install.sh | bash
          echo "$HOME/.goenv/bin" >> $GITHUB_PATH

      - name: Generate Inventory
        run: |
          goenv inventory go --json --checksums > inventory.json

      - name: Generate SBOM
        run: |
          goenv tools install cyclonedx-gomod@v1.6.0
          goenv sbom project --tool=cyclonedx-gomod --output=sbom.json

      - name: Upload Artifacts
        uses: actions/upload-artifact@v4
        with:
          name: compliance-reports
          path: |
            inventory.json
            sbom.json

      - name: Commit to Audit Trail
        run: |
          mkdir -p audits/$(date +%Y-%m)
          cp inventory.json audits/$(date +%Y-%m)/
          cp sbom.json audits/$(date +%Y-%m)/
          git add audits/
          git commit -m "Monthly compliance audit - $(date +%Y-%m-%d)"
          git push
```

## Best Practices

### 1. Automated Tracking

Use hooks to automatically log all changes:

```yaml
# ~/.goenv/hooks.yaml
enabled: true
acknowledged_risks: true

settings:
  timeout: "10s"
  continue_on_error: true

hooks:
  post_install:
    - action: log_to_file
      params:
        path: /var/audit/goenv.log
        message: "[{timestamp}] INSTALL Go {version} - ${USER}@${HOSTNAME}"

    - action: http_webhook
      params:
        url: https://compliance-api.example.com/events
        body: |
          {
            "event": "software_install",
            "software": "golang",
            "version": "{version}",
            "timestamp": "{timestamp}",
            "user": "${USER}",
            "host": "${HOSTNAME}"
          }
```

### 2. Regular Audits

Schedule periodic inventory snapshots:

```bash
# Add to crontab
# Monthly inventory on the 1st at 2am
0 2 1 * * /usr/local/bin/goenv inventory go --json --checksums > /var/audit/go-inventory-$(date +\%Y-\%m).json
```

### 3. Version Control

Keep audit trails in version control:

```bash
# Initialize audit repository
mkdir -p ~/.goenv/audit
cd ~/.goenv/audit
git init
git config user.name "goenv-audit"
git config user.email "audit@example.com"

# Track changes
goenv inventory go > current-inventory.txt
git add current-inventory.txt
git commit -m "Initial inventory - $(date +%Y-%m-%d)"
```

### 4. Separation of Duties

Use different accounts for auditing and operations:

```bash
# Audit account (read-only)
useradd -m -s /bin/bash goenv-audit
chown -R goenv-audit:goenv-audit /var/audit

# Grant read-only access to goenv installations
setfacl -R -m u:goenv-audit:r-x ~/.goenv/versions
```

### 5. Retention Policies

Implement data retention for compliance:

```bash
# Keep audit logs for 7 years (SOC 2, ISO 27001)
find /var/audit -name "*.json" -mtime +2555 -delete

# Archive old reports
tar -czf audits-archive-$(date +%Y).tar.gz audits/$(date +%Y-*)/
mv audits-archive-*.tar.gz /long-term-storage/
```

## See Also

- [goenv inventory command](./reference/COMMANDS.md#goenv-inventory)
- [goenv sbom command](./reference/COMMANDS.md#goenv-sbom)
- [Hooks System](../reference/HOOKS_QUICKSTART.md)
- [CI/CD Integration Guide](CI_CD_GUIDE.md)
