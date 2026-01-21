# SBOM Implementation Guide

**Purpose:** Practical guide for DevOps, security engineers, and developers to generate reproducible, Go-aware SBOMs.

**Audience:** Hands-on practitioners implementing SBOM workflows.

**Prerequisites:** goenv v3.0+ installed

---

## Quick Start

### Install SBOM Tool

```bash
# Install cyclonedx-gomod (recommended for Go projects)
goenv tools install cyclonedx-gomod@v1.6.0

# Or syft (for container images)
goenv tools install syft@v1.0.0

# Verify installation
goenv tools list
```

### Generate Basic SBOM

```bash
# CycloneDX format (JSON)
goenv sbom project \
  --tool=cyclonedx-gomod \
  --format=cyclonedx-json \
  --output=sbom.json

# CycloneDX format (XML)
goenv sbom project \
  --tool=cyclonedx-gomod \
  --format=cyclonedx-xml \
  --output=sbom.cdx.xml

# SPDX format (requires syft, not cyclonedx-gomod)
goenv sbom project \
  --tool=syft \
  --format=spdx-json \
  --output=sbom.spdx.json

# Verify output
jq '.bomFormat' sbom.json  # "CycloneDX"
jq '.components | length' sbom.json  # Component count
```

### Validate SBOM

```bash
# Check format
jq -e '.bomFormat == "CycloneDX"' sbom.json

# Check components exist
jq -e '.components | length > 0' sbom.json

# Check metadata
jq -e '.metadata.timestamp' sbom.json
jq -e '.metadata.component.name' sbom.json
```

---

## Reproducible SBOMs: The Foundation

**Goal:** Generate identical SBOMs across dev, CI, and production for the same code + dependencies.

**Why it matters:**

- **SOC 2/ISO 27001:** Prove SBOM matches release
- **SLSA Level 3:** Cryptographic verification
- **Audit trails:** Hash equality proves integrity
- **CI/CD trust:** Verify dev SBOM == production SBOM

---

### Reproducibility Levels

#### Level 0: No Reproducibility (Default)

```bash
# Different every time
goenv sbom project --output sbom.json
sha256sum sbom.json  # Different hash each run ❌
```

**Issues:** Timestamps vary, UUIDs random, array order non-deterministic

**Use case:** Quick development only, not for compliance

---

#### Level 1: Timestamp-Independent

```bash
goenv sbom project \
  --timestamp=build \
  --output sbom.json
```

**Guarantees:** Timestamp from go.mod mtime, not current time

**Reproducibility:** Within same day/build

---

#### Level 2: Content-Addressable

```bash
goenv sbom project \
  --deterministic \
  --timestamp=build \
  --output sbom.json
```

**Guarantees:**

- Sorted component arrays (lexicographic)
- Deterministic UUIDs (content-derived)
- Stable tool version in metadata
- Platform-independent output

**Reproducibility:** Across machines, same inputs

---

#### Level 3: Cryptographically Verifiable (RECOMMENDED)

```bash
# Step 1: Download dependencies (controlled)
go mod download
go mod verify

# Step 2: Generate deterministic SBOM offline
goenv sbom project \
  --deterministic \
  --offline \
  --embed-digests \
  --output sbom.json

# Step 3: Verify input digests
jq -r '.metadata.goenv.go_mod_digest' sbom.json
# sha256:abc123...

sha256sum go.mod
# abc123... ✅ Match!
```

**Guarantees:**

- Level 2 features +
- Input digests (go.mod, go.sum)
- Offline mode (no network variance)
- Build flags embedded

**Reproducibility:** Auditable, cryptographically verifiable

---

### Implementation Steps

#### Step 1: Enable Deterministic Mode

```bash
goenv sbom project \
  --deterministic \
  --output sbom.json
```

**What this does:**

1. Sorts components by purl (lexicographic)
2. Uses build timestamp, not current time
3. Derives serial number from content hash
4. Normalizes all purl formats
5. Orders all arrays consistently

**Verification:**

```bash
# Generate twice
goenv sbom project --deterministic --output sbom1.json
sleep 5
goenv sbom project --deterministic --output sbom2.json

# Should be identical
diff sbom1.json sbom2.json
# No differences ✅
```

---

#### Step 2: Offline Mode

```bash
# Download dependencies first (controlled step)
go mod download
go mod verify

# Generate SBOM offline (no network variance)
goenv sbom project \
  --deterministic \
  --offline \
  --output sbom.json
```

**Why offline matters:**

- No module proxy timeouts affecting metadata
- No DNS resolution variance
- No CDN routing differences
- Guaranteed same module versions

**Verification:**

```bash
# Should succeed without network
unshare -n goenv sbom project --offline --output sbom.json
# ✅ Works (no network access required)
```

---

#### Step 3: Pin Tool Versions

```bash
# Install specific SBOM tool version
goenv tools install cyclonedx-gomod@v1.6.0

# Verify version
goenv tools list | grep cyclonedx-gomod
# cyclonedx-gomod v1.6.0 ✓

# Tool version embedded in SBOM metadata
jq '.metadata.tools[] | select(.name == "cyclonedx-gomod")' sbom.json
# {
#   "name": "cyclonedx-gomod",
#   "version": "v1.6.0",
#   "vendor": "CycloneDX"
# }
```

---

#### Step 4: Embed Input Digests

```bash
# Generate with input digests
goenv sbom project \
  --deterministic \
  --offline \
  --embed-digests \
  --output sbom.json

# Verify go.mod unchanged
jq -r '.metadata.goenv.go_mod_digest' sbom.json
# sha256:abc123...

sha256sum go.mod
# abc123... ✓ Match!

# Verify go.sum unchanged
jq -r '.metadata.goenv.go_sum_digest' sbom.json
# sha256:def456...

sha256sum go.sum
# def456... ✓ Match!
```

---

#### Step 5: Hash Verification

```bash
# Generate canonical SBOM
goenv sbom project \
  --deterministic \
  --offline \
  --embed-digests \
  --output sbom.json

# Compute content hash (excluding metadata.timestamp)
jq 'del(.metadata.timestamp)' sbom.json | sha256sum
# xyz789... (consistent hash)

# Store with release
echo "xyz789... sbom.json" > SBOM.sha256
```

---

### Verification Commands

#### Basic Reproducibility Check

```bash
goenv sbom verify-reproducible \
  --sbom1 dev-sbom.json \
  --sbom2 ci-sbom.json
```

**Checks:**

- Component arrays identical (order-independent)
- Versions match exactly
- Licenses match
- PURLs canonical and identical

**Exit codes:**

- `0` - Reproducible ✅
- `1` - Not reproducible (shows diff)
- `2` - Invalid SBOM format

---

#### Hash-Based Verification

```bash
# Generate hash (excluding timestamp)
goenv sbom hash sbom.json
# sha256:xyz789...

# Verify against stored hash
echo "xyz789... sbom.json" > expected.sha256
goenv sbom verify-hash sbom.json expected.sha256
# ✓ Hash matches
```

---

#### Input Digest Verification

```bash
# Verify embedded digests match actual files
goenv sbom verify-digests sbom.json

# Output:
# ✓ go.mod digest matches
# ✓ go.sum digest matches
# ✓ No modifications since SBOM generation
```

---

### Troubleshooting Non-Reproducibility

#### Diagnosis Command

```bash
goenv sbom diff \
  --sbom1 sbom1.json \
  --sbom2 sbom2.json \
  --explain-differences
```

**Example output:**

```
Differences found:

1. Timestamp mismatch:
   sbom1: 2025-11-04T14:23:47Z
   sbom2: 2025-11-04T14:45:12Z
   → Use --timestamp=build

2. Component order differs:
   sbom1: [pkg-b, pkg-a]
   sbom2: [pkg-a, pkg-b]
   → Use --deterministic

3. Tool version mismatch:
   sbom1: cyclonedx-gomod v1.5.0
   sbom2: cyclonedx-gomod v1.6.0
   → Pin: goenv tools install cyclonedx-gomod@v1.6.0
```

---

#### Common Causes and Fixes

| Issue                  | Symptom                       | Fix                        |
| ---------------------- | ----------------------------- | -------------------------- |
| **Timestamp variance** | Different generation times    | `--timestamp=build`        |
| **Tool version**       | Different schemas             | Pin version in CI          |
| **Array ordering**     | Components in different order | `--deterministic`          |
| **Network variance**   | Module metadata differs       | `--offline` after download |
| **Environment vars**   | Platform-specific metadata    | Normalize in CI config     |
| **Random UUIDs**       | Serial number changes         | `--deterministic`          |
| **go.mod changes**     | Dependencies updated          | `--embed-digests`, verify  |

---

## Go-Aware Metadata Capture

**What makes goenv SBOMs different:** Go-specific build context that generic tools miss.

### Build Context Flags

```bash
# Build with specific configuration
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -tags netgo,osusergo -ldflags "-s -w"

# goenv captures this context
goenv sbom project \
  --capture-build-context \
  --output sbom.json
```

**SBOM includes:**

```json
{
  "metadata": {
    "goenv": {
      "go_version": "1.23.2",
      "build_context": {
        "tags": ["netgo", "osusergo"],
        "cgo_enabled": false,
        "goos": "linux",
        "goarch": "amd64",
        "ldflags": "-s -w"
      }
    }
  }
}
```

**Value:** Security teams know exact build configuration for vulnerability analysis.

---

### Replace Directive Detection

**Risky go.mod:**

```go
replace github.com/public/lib => ../local-fork  // Supply chain risk
```

**goenv SBOM:**

```json
{
  "components": [
    {
      "name": "github.com/public/lib",
      "version": "1.2.3",
      "goenv": {
        "replaced": true,
        "replace_directive": {
          "target": "../local-fork",
          "type": "local-path",
          "risk_level": "high"
        }
      }
    }
  ]
}
```

**Value:** Supply chain security sees hidden local dependencies.

---

### Standard Library Component

```json
{
  "components": [
    {
      "type": "library",
      "name": "golang-stdlib",
      "version": "1.23.2",
      "purl": "pkg:golang/stdlib@1.23.2",
      "goenv": {
        "type": "stdlib",
        "packages_included": ["net/http", "crypto/tls", "encoding/json"]
      }
    }
  ]
}
```

**Value:** Vulnerability scanning catches stdlib CVEs (40% of binary).

---

## CI/CD Integration

### GitHub Actions (Complete Example)

```yaml
name: Reproducible SBOM

on:
  push:
    branches: [main]
  pull_request:
  release:
    types: [published]

jobs:
  sbom-reproducibility:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - name: Setup goenv
        run: |
          curl -sfL https://goenv.sh | bash
          echo "$HOME/.goenv/bin" >> $GITHUB_PATH
          eval "$(goenv init -)"

      - name: Install Go (from .go-version)
        run: goenv install --skip-existing

      - name: Pin SBOM tool version
        run: goenv tools install cyclonedx-gomod@v1.6.0

      - name: Download dependencies (controlled)
        run: |
          goenv exec go mod download
          goenv exec go mod verify

      - name: Generate SBOM (deterministic)
        run: |
          goenv sbom project \
            --deterministic \
            --offline \
            --embed-digests \
            --output sbom-${{ github.sha }}.json

      - name: Compute SBOM hash
        id: sbom_hash
        run: |
          HASH=$(jq 'del(.metadata.timestamp)' sbom-${{ github.sha }}.json | sha256sum | cut -d' ' -f1)
          echo "hash=$HASH" >> $GITHUB_OUTPUT
          echo "$HASH sbom-${{ github.sha }}.json" > sbom.sha256

      - name: Verify reproducibility
        run: |
          # Generate again
          goenv sbom project \
            --deterministic \
            --offline \
            --embed-digests \
            --output sbom-verify.json

          # Compare hashes
          HASH1=$(jq 'del(.metadata.timestamp)' sbom-${{ github.sha }}.json | sha256sum | cut -d' ' -f1)
          HASH2=$(jq 'del(.metadata.timestamp)' sbom-verify.json | sha256sum | cut -d' ' -f1)

          if [ "$HASH1" != "$HASH2" ]; then
            echo "❌ SBOM not reproducible!"
            exit 1
          fi

          echo "✅ SBOM is reproducible (hash: $HASH1)"

      - name: Upload SBOM
        uses: actions/upload-artifact@v4
        with:
          name: sbom-${{ github.sha }}
          path: |
            sbom-${{ github.sha }}.json
            sbom.sha256
          retention-days: 90

      - name: Upload to release (on release only)
        if: github.event_name == 'release'
        uses: actions/upload-release-asset@v1
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        with:
          upload_url: ${{ github.event.release.upload_url }}
          asset_path: sbom-${{ github.sha }}.json
          asset_name: sbom.cdx.json
          asset_content_type: application/json
```

---

### GitLab CI (Complete Example)

```yaml
reproducible-sbom:
  stage: build
  image: golang:1.23

  before_script:
    - curl -sfL https://goenv.sh | bash
    - export PATH="$HOME/.goenv/bin:$PATH"
    - eval "$(goenv init -)"
    - goenv install --skip-existing
    - goenv tools install cyclonedx-gomod@v1.6.0

  script:
    # Controlled dependency download
    - go mod download
    - go mod verify

    # Generate deterministic SBOM
    - |
      goenv sbom project \
        --deterministic \
        --offline \
        --embed-digests \
        --output sbom-${CI_COMMIT_SHA}.json

    # Compute hash
    - jq 'del(.metadata.timestamp)' sbom-${CI_COMMIT_SHA}.json | sha256sum > sbom.sha256

    # Verify reproducibility
    - goenv sbom project --deterministic --offline --embed-digests --output sbom-verify.json
    - |
      HASH1=$(jq 'del(.metadata.timestamp)' sbom-${CI_COMMIT_SHA}.json | sha256sum | cut -d' ' -f1)
      HASH2=$(jq 'del(.metadata.timestamp)' sbom-verify.json | sha256sum | cut -d' ' -f1)
      if [ "$HASH1" != "$HASH2" ]; then
        echo "❌ SBOM not reproducible"
        exit 1
      fi
      echo "✅ SBOM reproducible: $HASH1"

  artifacts:
    reports:
      cyclonedx: sbom-${CI_COMMIT_SHA}.json
    paths:
      - sbom-${CI_COMMIT_SHA}.json
      - sbom.sha256
```

---

## Key Use Cases

### Use Case 1: License Compliance Scanning

**Goal:** Ensure no GPL or AGPL dependencies before release.

```bash
# Generate SBOM
goenv sbom project \
  --tool=cyclonedx-gomod \
  --output=sbom.json

# Check for prohibited licenses
PROHIBITED=("GPL-3.0" "AGPL-3.0" "SSPL")

for license in "${PROHIBITED[@]}"; do
  COUNT=$(jq -r --arg lic "$license" \
    '.components[] | select(.licenses[]?.license.id == $lic) | .name' \
    sbom.json | wc -l)

  if [ $COUNT -gt 0 ]; then
    echo "❌ Found $COUNT components with prohibited license: $license"
    jq -r --arg lic "$license" \
      '.components[] | select(.licenses[]?.license.id == $lic) |
       "  - \(.name) \(.version)"' \
      sbom.json
    exit 1
  fi
done

echo "✅ License compliance check passed"
```

---

### Use Case 2: Vulnerability Scanning Integration

**Goal:** Generate SBOM and scan for vulnerabilities in one workflow.

```bash
# 1. Generate SBOM
goenv sbom project \
  --deterministic \
  --offline \
  --output=sbom.json

# 2. Scan with Grype
grype sbom:sbom.json \
  --fail-on critical \
  --output json \
  > vulnerabilities.json

# 3. Check results
CRITICAL=$(jq '[.matches[] | select(.vulnerability.severity == "Critical")] | length' vulnerabilities.json)

if [ $CRITICAL -gt 0 ]; then
  echo "❌ Found $CRITICAL critical vulnerabilities"
  jq -r '.matches[] | select(.vulnerability.severity == "Critical") |
    "  - \(.artifact.name): \(.vulnerability.id)"' \
    vulnerabilities.json
  exit 1
fi

echo "✅ No critical vulnerabilities"
```

---

### Use Case 3: SOC 2 Compliance Evidence

**Goal:** Generate evidence package for auditors.

```bash
#!/bin/bash
# generate-soc2-evidence.sh

QUARTER="2026-Q1"
OUTPUT_DIR="compliance/soc2/${QUARTER}"
mkdir -p "${OUTPUT_DIR}"

# Generate SBOM
goenv sbom project \
  --deterministic \
  --offline \
  --embed-digests \
  --output="${OUTPUT_DIR}/sbom.json"

# Extract dependency list
jq -r '.components[] | "\(.name) \(.version) \(.licenses[0].license.id // "Unknown")"' \
  "${OUTPUT_DIR}/sbom.json" \
  | sort > "${OUTPUT_DIR}/dependencies.txt"

# Generate evidence summary
cat > "${OUTPUT_DIR}/evidence-summary.md" <<EOF
# SOC 2 Software Inventory Evidence

**Period:** ${QUARTER}
**Generated:** $(date -u +%Y-%m-%d)
**Go Version:** $(goenv version-name)

## Summary
- **Total Components:** $(jq '.components | length' "${OUTPUT_DIR}/sbom.json")
- **SBOM Hash:** $(jq 'del(.metadata.timestamp)' "${OUTPUT_DIR}/sbom.json" | sha256sum | cut -d' ' -f1)

## Control Mapping
- **CC7.1** - Software inventory maintained
- **CC7.2** - Dependencies documented with licenses
- **CC8.1** - Change tracking via git + SBOM hash
EOF

# Create auditor package
tar -czf "${OUTPUT_DIR}.tar.gz" -C compliance/soc2 "${QUARTER}"

echo "✅ SOC 2 evidence: ${OUTPUT_DIR}.tar.gz"
```

---

### Use Case 4: Multi-Project Aggregation

**Goal:** Consolidated view of all projects' dependencies for security team.

```bash
#!/bin/bash
# aggregate-sboms.sh

PROJECTS=("project-a" "project-b" "project-c")
SBOM_DIR="sboms/aggregate-$(date +%Y%m%d)"
mkdir -p "${SBOM_DIR}"

# Generate SBOM for each project
for project in "${PROJECTS[@]}"; do
  echo "Generating SBOM for ${project}..."
  (cd "../${project}" && \
    goenv sbom project \
      --deterministic \
      --offline \
      --output="${SBOM_DIR}/${project}-sbom.json")
done

# Merge all components (deduplicate)
jq -s '
  {
    bomFormat: "CycloneDX",
    specVersion: "1.5",
    metadata: {
      timestamp: (now | strftime("%Y-%m-%dT%H:%M:%SZ")),
      component: {
        type: "application",
        name: "enterprise-aggregate"
      }
    },
    components: ([.[].components[]] | unique_by(.purl))
  }
' "${SBOM_DIR}"/*-sbom.json > "${SBOM_DIR}/aggregate-sbom.json"

# Generate summary
echo "# Aggregated SBOM Summary" > "${SBOM_DIR}/summary.md"
echo "Generated: $(date)" >> "${SBOM_DIR}/summary.md"
echo "" >> "${SBOM_DIR}/summary.md"
echo "## Unique Dependencies" >> "${SBOM_DIR}/summary.md"
jq -r '.components[] | "\(.name) \(.version)"' "${SBOM_DIR}/aggregate-sbom.json" \
  | sort >> "${SBOM_DIR}/summary.md"
```

---

### Use Case 5: Container Image SBOM

**Goal:** Generate SBOM for final container image.

```bash
# Build container
docker build -t myapp:v1.0.0 .

# Generate SBOM for container image
goenv sbom project \
  --tool=syft \
  --format=spdx-json \
  --image=myapp:v1.0.0 \
  --output=container-sbom.spdx.json

# Verify includes all layers
jq '.packages | length' container-sbom.spdx.json
```

---

## Best Practices

### 1. Version Pinning

Always pin versions for reproducibility:

```bash
# .go-version
1.23.2

# Install specific tool versions
goenv tools install cyclonedx-gomod@v1.6.0
```

---

### 2. Offline Mode

Use `--offline` for reproducible, air-gapped builds:

```bash
# Download dependencies first
goenv exec go mod download
goenv exec go mod verify

# Generate SBOM offline
goenv sbom project --offline --output sbom.json
```

---

### 3. Validation

Always validate generated SBOMs:

```bash
# Format check
jq -e '.bomFormat == "CycloneDX"' sbom.json

# Component count
jq -e '.components | length > 0' sbom.json

# Required metadata
jq -e '.metadata.timestamp' sbom.json
```

---

### 4. Storage and Retention

Store SBOMs with proper organization:

```bash
# Organized by version
sboms/
├── v1.0.0/
│   └── sbom-20260104-abc123.json
├── v1.0.1/
│   └── sbom-20260115-def456.json
└── latest.json -> v1.0.1/sbom-20260115-def456.json
```

---

### 5. Audit Trail

Log all SBOM generation events:

```bash
echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] SBOM generated for $(goenv version-name) at $(git rev-parse HEAD)" \
  >> /var/log/sbom-audit.log
```

---

## Compliance Integration

### SLSA Level 3 Provenance

```json
{
  "predicateType": "https://slsa.dev/provenance/v1",
  "predicate": {
    "buildDefinition": {
      "buildType": "https://goenv.sh/BuildType/v1",
      "externalParameters": {
        "go_version": "1.23.2",
        "go_mod_digest": "sha256:abc123...",
        "go_sum_digest": "sha256:def456...",
        "sbom_digest": "sha256:xyz789..."
      }
    },
    "runDetails": {
      "builder": {
        "id": "https://github.com/go-nv/goenv@v3.1.0"
      },
      "metadata": {
        "reproducible": true,
        "sbom_tool": "cyclonedx-gomod@v1.6.0"
      }
    }
  }
}
```

---

## Migration from Direct Tool Usage

**Before (direct cyclonedx-gomod):**

```bash
cyclonedx-gomod -json -output sbom.json
```

**After (with goenv):**

```bash
goenv sbom project --tool=cyclonedx-gomod --output sbom.json
```

**Benefits:**

- Correct Go version guaranteed
- Reproducibility features
- Go-aware metadata
- Cross-platform consistency

---

## Troubleshooting

### Issue: SBOM Generation Fails

```bash
# Check Go version
goenv version

# Verify go.mod validity
go mod verify

# Check tool installation
goenv tools list

# Run with verbose output
goenv sbom project --tool=cyclonedx-gomod --output sbom.json --verbose
```

---

### Issue: Non-Reproducible SBOMs

```bash
# Use full deterministic mode
goenv sbom project \
  --deterministic \
  --offline \
  --embed-digests \
  --output sbom.json

# Diagnose differences
goenv sbom diff sbom1.json sbom2.json --explain-differences
```

---

### Issue: Missing Components

```bash
# Ensure dependencies downloaded
go mod download
go mod verify

# Check go.sum exists
ls -la go.sum

# Regenerate SBOM
goenv sbom project --tool=cyclonedx-gomod --output sbom.json
```

---

## Future Enhancements

See [SBOM_STRATEGY.md](../roadmap/SBOM_STRATEGY.md) for roadmap:

- **v3.1** - Reproducibility foundation, Go-aware metadata
- **v3.2** - Policy validation engine
- **v3.3** - Signing and attestation
- **v3.4+** - Diffing, vulnerability integration, compliance reporting

---

## Getting Help

- **Documentation:** [SBOM Strategy](../roadmap/SBOM_STRATEGY.md)
- **Issues:** https://github.com/go-nv/goenv/issues (tag: `sbom`)
- **Security:** security@goenv.io
- **Discussions:** Share your implementation patterns!

---

**Your feedback shapes future features.** Tell us what works, what's missing, and what you need for enterprise security workflows.
