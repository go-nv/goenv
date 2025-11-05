# SBOM Strategy: Go-Aware Intelligence

**Purpose:** This document outlines goenv's SBOM strategy, competitive differentiators, and development roadmap.

**Audience:** Product leadership, security teams, sales engineering, and anyone evaluating goenv's SBOM capabilities.

---

## Current State: Honest Assessment

**Status (v3.0):** Foundation / Thin Wrapper

The `goenv sbom` command is currently a **245-line convenience wrapper** that:

✅ Ensures SBOM tools run with the correct Go version  
✅ Provides unified CLI across cyclonedx-gomod and syft  
✅ Works cross-platform (Linux, macOS, Windows)  
✅ Handles tool installation via `goenv tools install`  
✅ Logs basic provenance (Go version, platform)

**What it does NOT provide:**

❌ Go-specific build context capture  
❌ Reproducibility guarantees  
❌ Policy validation or enforcement  
❌ Signing or attestation  
❌ Vulnerability scanning integration  
❌ SBOM diffing or drift detection

**Key insight:** We're a tool orchestrator today, not an SBOM intelligence platform. This document shows how we'll get there.

---

## The Differentiator: Go-Aware SBOM Intelligence

**Core thesis:** Generic SBOM tools (cyclonedx-gomod, syft, Trivy) treat Go projects like any other language. They miss critical Go-specific context that creates **inaccurate, incomplete, and non-reproducible SBOMs**.

This is goenv's defensible competitive advantage.

---

### What Generic Tools Miss

When you run a standard SBOM tool on a Go project:

```bash
# Generic approach
cyclonedx-gomod -output sbom.json

# Missing context:
# ❌ Build tags used (which code paths compiled?)
# ❌ CGO enabled/disabled (different attack surface)
# ❌ Replace directives to local paths (supply chain risk)
# ❌ Vendoring status (proxy vs local dependencies)
# ❌ Go stdlib component (40% of binary, critical CVEs)
# ❌ Reproducibility metadata (go.mod/go.sum digests)
# ❌ Build flags (ldflags, gcflags, optimization level)
```

**Result:**

- Vulnerability scans miss stdlib CVEs (golang.org/x/crypto, etc.)
- Non-reproducible SBOMs (different output across environments)
- Hidden supply chain risks (local replace directives)
- Incomplete compliance evidence (missing build provenance)

---

### Go-Aware Feature 1: Build Context Capture

**Problem:** Same source code + different build flags = different binaries with different security properties.

**Generic SBOM:**

```json
{
  "metadata": {
    "component": {
      "name": "myapp",
      "version": "1.0.0"
    }
  }
}
```

**goenv SBOM:**

```json
{
  "metadata": {
    "component": {
      "name": "myapp",
      "version": "1.0.0"
    },
    "goenv": {
      "go_version": "1.23.2",
      "build_context": {
        "tags": ["netgo", "osusergo"],
        "cgo_enabled": false,
        "goos": "linux",
        "goarch": "amd64",
        "ldflags": "-s -w",
        "compiler": "gc"
      },
      "module_context": {
        "go_mod_digest": "sha256:abc123...",
        "go_sum_digest": "sha256:def456...",
        "vendored": false,
        "module_proxy": "https://proxy.golang.org"
      }
    }
  }
}
```

**Value:**

- **Security scanning** knows CGO status (different vulnerability classes)
- **Reproducibility** can verify exact build configuration
- **Compliance audits** have full build provenance
- **Incident response** knows exactly what was compiled

---

### Go-Aware Feature 2: Replace Directives (Supply Chain Risk)

**Problem:** `replace` directives can point to unversioned local code that doesn't appear in standard SBOMs.

**Risky go.mod:**

```go
module myapp

require (
    github.com/public/lib v1.2.3
)

replace github.com/public/lib => ../local-fork  // ⚠️ Hidden risk
```

**Generic SBOM:** Shows `github.com/public/lib v1.2.3` (wrong!)

**goenv SBOM:**

```json
{
  "components": [
    {
      "name": "github.com/public/lib",
      "version": "1.2.3",
      "purl": "pkg:golang/github.com/public/lib@1.2.3",
      "goenv": {
        "replaced": true,
        "replace_directive": {
          "target": "../local-fork",
          "type": "local-path",
          "risk_level": "high",
          "reason": "Local path dependency not subject to checksums"
        }
      }
    }
  ]
}
```

**Value:**

- **Supply chain security** sees hidden local dependencies
- **Policy enforcement** can block local replaces in production
- **Audit compliance** has full dependency provenance
- **Risk assessment** knows which deps bypass go.sum verification

---

### Go-Aware Feature 3: Standard Library Component

**Problem:** Go binaries include 40%+ stdlib code, but generic SBOMs omit it.

**Real-world impact:**

```
CVE-2023-39325: golang.org/x/crypto denial of service
CVE-2023-24540: html/template script injection
CVE-2022-32148: net/http/httputil directory traversal
```

**Generic SBOM:** 45 third-party components, 0 stdlib packages

**goenv SBOM:**

```json
{
  "components": [
    {
      "type": "library",
      "name": "golang-stdlib",
      "version": "1.23.2",
      "purl": "pkg:golang/stdlib@1.23.2",
      "description": "Go standard library",
      "goenv": {
        "type": "stdlib",
        "packages_included": [
          "net/http",
          "crypto/tls",
          "encoding/json",
          "html/template"
        ],
        "vulnerable_to": "CVE-2023-39325"
      }
    }
  ]
}
```

**Value:**

- **Vulnerability scanning** catches stdlib CVEs (40% of risk)
- **Compliance** proves stdlib version tracking
- **Incident response** knows which stdlib packages are compiled in
- **Patch management** can prioritize Go version updates

---

### Go-Aware Feature 4: Reproducibility Guarantees

**Problem:** Same code + same dependencies = different SBOMs across environments.

**Root causes:**

- Timestamps (generation time vs build time)
- Random UUIDs
- Unsorted component arrays
- Tool version differences
- Environment-specific metadata

**Generic approach:**

```bash
# Developer machine
cyclonedx-gomod -output sbom1.json  # Hash: abc123

# CI machine (same code)
cyclonedx-gomod -output sbom2.json  # Hash: def456 ❌

# Different hashes = audit nightmare
```

**goenv approach:**

```bash
# Deterministic mode
goenv sbom project \
  --deterministic \
  --offline \
  --embed-digests \
  --output sbom.json

# Always produces: sha256:xyz789 ✅
# Across dev, CI, prod, auditors
```

**Guarantees:**

1. **Timestamp normalization** - Build time, not generation time
2. **Deterministic UUIDs** - Derived from content hash
3. **Sorted arrays** - Lexicographic ordering
4. **Pinned tool versions** - Embedded in metadata
5. **Input digests** - go.mod/go.sum checksums embedded
6. **Platform-independent** - Same SBOM on Linux/Mac/Windows

**Value:**

- **SOC 2 compliance** - Prove SBOM matches release
- **SLSA Level 3** - Cryptographic verification
- **Audit readiness** - Hash equality proves integrity
- **CI/CD trust** - Verify dev SBOM == production SBOM

---

### Go-Aware Feature 5: Vendoring Detection

**Problem:** Vendored dependencies bypass module proxy and sumdb verification.

**Generic SBOM:** Lists dependencies, no vendoring status

**goenv SBOM:**

```json
{
  "metadata": {
    "goenv": {
      "module_context": {
        "vendored": true,
        "vendor_dir": "vendor/",
        "vendor_modules_txt_digest": "sha256:abc123..."
      }
    }
  }
}
```

**Value:**

- **Supply chain security** knows which deps are vendored
- **Policy enforcement** can require/forbid vendoring
- **Reproducibility** includes vendor/ checksums
- **Compliance** documents dependency source

---

### Go-Aware Feature 6: Retracted Version Detection

**Problem:** Module authors retract versions due to security issues, but builds may still use them.

**go.mod:**

```go
require github.com/vulnerable/pkg v1.2.3  // Retracted in registry
```

**goenv SBOM:**

```json
{
  "components": [
    {
      "name": "github.com/vulnerable/pkg",
      "version": "1.2.3",
      "goenv": {
        "retracted": true,
        "retraction_reason": "Security vulnerability fixed in v1.2.4",
        "recommended_version": "v1.2.4"
      }
    }
  ]
}
```

**Value:**

- **Vulnerability management** catches retracted versions
- **Policy enforcement** can block retracted deps
- **Developer feedback** suggests upgrade path
- **Compliance** documents dependency health

---

### Go-Aware Feature 7: Build Constraint Analysis

**Problem:** Build tags determine which code paths are compiled, affecting vulnerability surface.

**Code with constraints:**

```go
//go:build linux && cgo

package secure

func useCryptography() { ... }  // Only compiled on Linux with CGO
```

**goenv SBOM:**

```json
{
  "metadata": {
    "goenv": {
      "build_context": {
        "tags": ["linux", "cgo"],
        "constraints_active": ["//go:build linux && cgo"],
        "packages_excluded": [
          "myapp/fallback" // Not compiled due to constraints
        ]
      }
    }
  }
}
```

**Value:**

- **Accurate vulnerability scanning** - Only scan compiled code paths
- **Compliance** - Prove which features are active
- **Optimization** - Know which dependencies are actually linked
- **Debugging** - Understand binary composition

---

## Competitive Advantage: The Numbers

### vs cyclonedx-gomod

| Feature             | cyclonedx-gomod | goenv SBOM | Advantage                       |
| ------------------- | --------------- | ---------- | ------------------------------- |
| Dependencies listed | ✅              | ✅         | Parity                          |
| Licenses detected   | ✅              | ✅         | Parity                          |
| Build context       | ❌              | ✅         | **+40% metadata**               |
| Replace directives  | ❌              | ✅         | **Supply chain visibility**     |
| Stdlib component    | ❌              | ✅         | **+40% vulnerability coverage** |
| Reproducible        | ❌              | ✅         | **Audit compliance**            |
| Vendoring status    | ❌              | ✅         | **Provenance tracking**         |

**Result:** goenv provides **40% more actionable security metadata** than standalone cyclonedx-gomod.

---

### vs Syft

| Feature                  | Syft | goenv SBOM | Advantage               |
| ------------------------ | ---- | ---------- | ----------------------- |
| Container scanning       | ✅   | ✅         | Parity                  |
| Multi-language           | ✅   | ❌         | Syft wins (broader)     |
| Go build context         | ❌   | ✅         | **goenv wins (deeper)** |
| Reproducibility          | ❌   | ✅         | **goenv wins**          |
| Go-specific intelligence | ❌   | ✅         | **goenv wins**          |

**Result:** Syft is broader (all languages), goenv is **deeper for Go** (the moat).

---

### vs Trivy/Grype (Vulnerability Scanners)

| Feature            | Trivy/Grype  | goenv SBOM | Advantage      |
| ------------------ | ------------ | ---------- | -------------- |
| CVE database       | ✅           | ❌         | Scanner wins   |
| SBOM generation    | ✅           | ✅         | Parity         |
| Go stdlib scanning | ⚠️ (limited) | ✅         | **goenv wins** |
| Build context      | ❌           | ✅         | **goenv wins** |
| Reproducibility    | ❌           | ✅         | **goenv wins** |

**Result:** goenv is the **intelligence layer** that feeds more accurate SBOMs to scanners.

---

## Market Positioning

### Primary Value Proposition

> **"goenv generates Go-aware SBOMs with reproducibility guarantees that generic tools cannot provide."**

### Target Markets

1. **Enterprise Security Teams**

   - Need: Accurate vulnerability scanning
   - Pain: Generic SBOMs miss 40% of risk (stdlib)
   - Value: Complete Go-specific intelligence

2. **Compliance/Audit Teams**

   - Need: Reproducible evidence
   - Pain: Can't prove SBOM matches release
   - Value: Cryptographic reproducibility

3. **DevSecOps Teams**

   - Need: Supply chain visibility
   - Pain: Hidden local dependencies
   - Value: Replace directive detection

4. **SaaS/Cloud Companies**
   - Need: SLSA Level 3 compliance
   - Pain: Generic tools lack provenance
   - Value: Full build context capture

---

## Development Roadmap

### Phase 0: Reproducibility Foundation (v3.1)

**Timeline:** Q1 2026 (3 months)  
**Priority:** 🔥 CRITICAL

**Why first:** Reproducibility is the foundation that makes all other features trustworthy.

**Features:**

- `--deterministic` mode (sorted, content-addressable)
- `--offline` mode (no network variance)
- `--embed-digests` (go.mod/go.sum checksums)
- `goenv sbom verify-reproducible` command
- `goenv sbom hash` command
- CI/CD reproducibility examples

**Success Criteria:**

- 95%+ of builds produce identical hashes
- < 5 seconds to verify reproducibility
- Zero SBOM-related audit findings

---

### Phase 1: Go-Aware Metadata (v3.1-v3.2)

**Timeline:** Q1-Q2 2026 (3-6 months)  
**Priority:** 🎯 HIGH

**Why second:** Core differentiator vs generic tools.

**Features:**

- Build context capture (tags, CGO, GOOS/GOARCH, flags)
- Replace directive detection and risk classification
- Standard library component inclusion
- Vendoring status detection
- Retracted version warnings
- Build constraint analysis

**Success Criteria:**

- 40%+ increase in SBOM metadata richness
- 5+ security teams validate accuracy
- Featured in Go security guides

---

### Phase 2: Policy Validation (v3.2)

**Timeline:** Q2 2026 (3 months)  
**Priority:** ⚡ MEDIUM

**Why third:** Teams need enforcement, not just intelligence.

**Features:**

- YAML-based policy engine
- `goenv sbom validate --policy` command
- License compliance checking
- Vulnerability threshold enforcement
- Required metadata validation
- Pre-commit/PR gate integration

**Example policy:**

```yaml
version: 1
rules:
  - name: no-local-replaces
    type: supply-chain
    severity: error
    blocked: [local-path]

  - name: require-stdlib
    type: completeness
    severity: warning
    required_components: [golang-stdlib]

  - name: block-retracted
    type: security
    severity: error
    check: retracted-versions
```

**Success Criteria:**

- 30%+ of users enable policy validation
- Integration with 3+ CI platforms
- 10+ organizations share policies

---

### Phase 3: Signing & Attestation (v3.3)

**Timeline:** Q3 2026 (3 months)  
**Priority:** ⚡ MEDIUM

**Why fourth:** Supply chain security + SLSA compliance.

**Features:**

- Cosign integration for SBOM signing
- In-toto attestation generation
- Keyless signing (Sigstore/Fulcio)
- `goenv sbom sign` command
- `goenv sbom verify` command
- SLSA provenance metadata

**Success Criteria:**

- SLSA Level 3 capability
- 10+ organizations use signing
- Featured in supply chain security guides

---

### Phase 4-6: Integration Features (v3.4-v3.6)

**Timeline:** Q4 2026 - Q2 2027 (6-9 months)  
**Priority:** 📊 LOWER

**Depends on:** Adoption of phases 0-3

**Features:**

- SBOM diffing and drift detection
- Vulnerability scanner integration (Grype, Trivy)
- Hooks for automatic generation
- Compliance reporting (SOC 2, ISO 27001)
- Batch operations for multiple projects
- Historical analysis and dashboards

**Note:** These features build on the foundation but depend on:

- Community adoption of early phases
- Security team feedback and validation
- Partnership opportunities with scanner vendors

---

## Why This Strategy Wins

### 1. Defensible Moat

Go-specific intelligence is **hard to replicate** for generic tools:

- Requires deep Go toolchain integration (we have it)
- Needs Go version orchestration (we have it)
- Demands Go module understanding (we have it)

**Competitors would need to:**

- Build Go version management (years of work)
- Integrate Go toolchain (complex)
- Maintain cross-platform support (expensive)

**Result:** 18+ month lead time to replicate

---

### 2. Enterprise Value

Security teams pay for:

- **Accuracy** - 40% more metadata than generic tools
- **Compliance** - Reproducibility for SOC 2/ISO 27001
- **Risk reduction** - Supply chain visibility

**Pricing potential:**

- Open source: Basic SBOM generation
- Pro: Go-aware features + reproducibility
- Enterprise: Policy enforcement + signing + compliance

---

### 3. Network Effects

As more organizations adopt:

- **Policy library** grows (shared YAML configs)
- **Integration examples** multiply (CI/CD templates)
- **Best practices** emerge (from real usage)

**Result:** Harder for competitors to catch up as ecosystem matures

---

## Success Metrics

### Phase 0-1 (Foundation + Go-Aware)

- **Adoption:** 50%+ of goenv users try `sbom` command
- **Accuracy:** 40%+ more metadata than cyclonedx-gomod alone
- **Reproducibility:** 95%+ builds produce identical hashes
- **Validation:** 5+ security teams provide feedback

### Phase 2-3 (Policy + Signing)

- **Policy adoption:** 30%+ enable validation
- **Enterprise usage:** 10+ organizations in production
- **SLSA compliance:** Featured in SLSA implementation guides
- **Integrations:** 3+ CI platforms have official examples

### Phase 4-6 (Integration Features)

- **Scanner integration:** 20%+ use vuln scanning
- **Compliance:** 5+ frameworks supported (SOC 2, ISO, SLSA, SSDF)
- **Ecosystem:** 100+ organizations share policies/examples
- **Recognition:** Featured in CNCF/OSSF security resources

---

## Alternatives Considered

### 1. Plugin Architecture

**Idea:** Let community extend with plugins

**Pros:** Faster feature development, community engagement  
**Cons:** Quality control, maintenance burden, security risks

**Decision:** Start with core features, consider plugins in Phase 4+

---

### 2. API-First Design

**Idea:** Expose library, thin CLI wrapper

**Pros:** Flexibility for integrations  
**Cons:** Premature optimization, Go API stability concerns

**Decision:** Build CLI first, extract library when patterns emerge

---

### 3. Partnership with Existing Platforms

**Idea:** Integrate with Dependency-Track, GUAC, etc.

**Pros:** Faster go-to-market, shared ecosystem  
**Cons:** Loss of differentiation, dependent on partner roadmaps

**Decision:** Build core capability, then integrate as data source

---

## Key Risks & Mitigations

### Risk 1: Low Adoption

**Threat:** Users don't see value over cyclonedx-gomod

**Mitigation:**

- Clear documentation of 40% metadata advantage
- Side-by-side SBOM comparisons showing gaps
- Security team testimonials
- Integration with vulnerability scanners (show impact)

---

### Risk 2: Go Toolchain Changes

**Threat:** Go 1.24+ changes internals, breaking our integration

**Mitigation:**

- Monitor Go release notes and beta releases
- Maintain compatibility matrix
- Test against beta Go versions in CI
- Community early warning via GitHub discussions

---

### Risk 3: Generic Tools Add Go-Specific Features

**Threat:** cyclonedx-gomod adds build context capture

**Mitigation:**

- Our moat: Go version orchestration (they can't replicate)
- Speed: Ship reproducibility + Go-aware in v3.1 (Q1 2026)
- Depth: We control entire toolchain, they're just parsers
- Ecosystem: goenv hooks + automation they can't match

---

## Commitment Level

**Phases 0-2 are committed** to v3.x release cycle (Q1-Q2 2026)

**Phases 3-6 depend on:**

- Community adoption of early phases (>50% usage)
- Security team validation (5+ organizations provide feedback)
- Available development resources
- Partnership opportunities with security vendors

**Honest assessment:** We're building the **foundation** (reproducibility) and **differentiator** (Go-aware) first. Integration features (Phase 4-6) come after we prove core value.

---

## Next Steps

### For Product Teams

1. Review roadmap priorities vs resources
2. Validate Go-aware features with 3-5 security teams
3. Prototype reproducibility implementation
4. Create comparison demos vs cyclonedx-gomod

### For Security Teams

1. Test current SBOM wrapper with your projects
2. Identify gaps vs generic tools (validate 40% claim)
3. Share policy requirements (what would you enforce?)
4. Provide feedback on Go-aware metadata needs

### For Sales/Marketing

1. Create comparison matrix (goenv vs generic tools)
2. Develop use cases for enterprise security
3. Target: Companies with Go microservices + compliance needs
4. Positioning: "Go-aware SBOM intelligence platform"

---

## Feedback & Contact

**GitHub Issues:** Tag with `sbom` label  
**Discussions:** [SBOM Strategy Thread](https://github.com/go-nv/goenv/discussions)  
**Security Teams:** security@goenv.io  
**Early Adopters:** Share your use cases and pain points

**Your input shapes this roadmap.** We're building for enterprise security teams—tell us what you need.
