# SBOM Policy Validation Guide

## Overview

goenv's SBOM policy validation allows you to enforce security, compliance, and quality standards on generated SBOMs. Policies are defined in YAML files and validated using the `goenv sbom validate` command.

## Quick Start

1. **Create a policy file** (`.goenv-policy.yaml`):

```yaml
version: "1"
rules:
  - name: no-local-replaces
    type: supply-chain
    severity: error
    check: replace-directives
    blocked:
      - local-path
```

2. **Generate an SBOM**:

```bash
goenv sbom project -o sbom.json
```

3. **Validate against policy**:

```bash
goenv sbom validate sbom.json --policy=.goenv-policy.yaml
```

## Policy Structure

### Basic Structure

```yaml
version: "1"  # Policy format version

options:      # Global options
  fail_on_error: true
  fail_on_warning: false
  verbose: false

rules:        # Validation rules (array)
  - name: rule-name
    type: rule-type
    severity: error|warning|info
    description: Human-readable description
    # ... rule-specific fields
```

### Rule Types

#### 1. Supply Chain Security (`supply-chain`)

**Check replace directives:**

```yaml
- name: no-local-replaces
  type: supply-chain
  severity: error
  description: Prevent local path dependencies
  check: replace-directives
  blocked:
    - local-path    # Block local file paths
    - file-path     # Block file:// URLs
```

**Check vendoring status:**

```yaml
- name: no-vendoring
  type: supply-chain
  severity: warning
  check: vendoring-status
  blocked:
    - vendored
```

#### 2. Security Requirements (`security`)

**Block retracted versions:**

```yaml
- name: no-retracted-versions
  type: security
  severity: error
  check: retracted-versions
```

**Enforce CGO status:**

```yaml
- name: require-cgo-disabled
  type: security
  severity: warning
  check: cgo-disabled
  required:
    - "false"  # Must be string "false"
```

#### 3. Completeness Checks (`completeness`)

**Require components:**

```yaml
- name: require-stdlib
  type: completeness
  severity: warning
  check: required-components
  required:
    - golang-stdlib
    - myapp/internal/crypto
```

**Require metadata:**

```yaml
- name: require-build-metadata
  type: completeness
  severity: error
  check: required-metadata
  required:
    - goenv:go_version
    - goenv:platform
    - goenv:build_context.goos
    - goenv:build_context.goarch
    - goenv:module_context.go_mod_digest
```

#### 4. License Compliance (`license`)

**Block licenses:**

```yaml
- name: no-gpl-licenses
  type: license
  severity: error
  description: Copyleft licenses prohibited
  blocked:
    - GPL-2.0
    - GPL-3.0
    - AGPL-3.0
    - LGPL-2.1
```

## Command Usage

### Basic Validation

```bash
# Use default policy file
goenv sbom validate sbom.json

# Specify policy file
goenv sbom validate sbom.json --policy=custom-policy.yaml

# Verbose output
goenv sbom validate sbom.json --verbose

# Fail on warnings
goenv sbom validate sbom.json --fail-on-warning
```

### CI/CD Integration

**GitHub Actions:**

```yaml
- name: Generate SBOM
  run: goenv sbom project -o sbom.json --enhance --deterministic

- name: Validate SBOM
  run: goenv sbom validate sbom.json --policy=.goenv-policy.yaml
```

**GitLab CI:**

```yaml
sbom_validate:
  script:
    - goenv sbom project -o sbom.json
    - goenv sbom validate sbom.json --policy=.goenv-policy.yaml
  artifacts:
    paths:
      - sbom.json
```

**Pre-commit Hook:**

```bash
#!/bin/bash
# .git/hooks/pre-commit

if [ -f sbom.json ]; then
    goenv sbom validate sbom.json --policy=.goenv-policy.yaml
    if [ $? -ne 0 ]; then
        echo "SBOM validation failed! Run 'goenv sbom project' to regenerate."
        exit 1
    fi
fi
```

## Example Policies

### Enterprise Security

See `examples/policies/enterprise-strict.yaml` for:
- Zero tolerance for local dependencies
- No vendoring allowed
- CGO must be disabled
- Complete build provenance required
- Strict license controls

### CI/CD Pipeline

See `examples/policies/ci-cd.yaml` for:
- Warn on local replaces
- Block retracted versions
- Require build metadata
- Permissive license policy

### Open Source

See `examples/policies/open-source.yaml` for:
- Informational warnings only
- Encourage best practices
- No build failures

## Available Checks

| Check                  | Type         | Description                          |
| ---------------------- | ------------ | ------------------------------------ |
| `replace-directives`   | supply-chain | Validate module replace directives   |
| `vendoring-status`     | supply-chain | Check if dependencies are vendored   |
| `retracted-versions`   | security     | Detect retracted module versions     |
| `cgo-disabled`         | security     | Verify CGO enabled/disabled status   |
| `required-components`  | completeness | Ensure specific components exist     |
| `required-metadata`    | completeness | Ensure specific metadata fields      |
| `blocked-licenses`     | license      | Block specific license types         |

## Severity Levels

| Severity  | Behavior                          |
| --------- | --------------------------------- |
| `error`   | Fails validation, returns exit 1  |
| `warning` | Reports but passes by default     |
| `info`    | Informational only, always passes |

Use `--fail-on-warning` to treat warnings as errors.

## Best Practices

### 1. Start Permissive

Begin with warnings and info severity, then tighten to errors as teams adapt:

```yaml
# Phase 1: Learn
- name: check-replaces
  severity: info  # Just observe

# Phase 2: Warn
- name: check-replaces
  severity: warning  # Alert teams

# Phase 3: Enforce
- name: check-replaces
  severity: error  # Block builds
```

### 2. Version Control Policies

```bash
# Store in repo root
.goenv-policy.yaml

# Or per-environment
policies/
  dev.yaml
  staging.yaml
  production.yaml
```

### 3. Document Exceptions

```yaml
- name: allow-vendor-for-airgap
  type: supply-chain
  severity: info  # Don't block
  description: |
    Vendoring allowed for air-gapped deployments.
    See ADR-0015 for rationale.
```

### 4. Combine with Other Tools

```bash
# Generate enhanced SBOM
goenv sbom project -o sbom.json --enhance

# Validate policy
goenv sbom validate sbom.json

# Scan for vulnerabilities
grype sbom:sbom.json

# Upload to dependency tracking
upload-to-dependency-track sbom.json
```

## Troubleshooting

### Policy File Not Found

```
Error: policy file not found: .goenv-policy.yaml (use --policy to specify)
```

**Solution:** Create policy file or specify path:

```bash
goenv sbom validate sbom.json --policy=path/to/policy.yaml
```

### Invalid Policy Syntax

```
Error: failed to load policy: rule "xyz": invalid severity "critical"
```

**Solution:** Use valid severity (`error`, `warning`, `info`)

### Rule Not Matching

```
✓ SBOM validation passed (expected violations)
```

**Solution:** Check that:
1. SBOM was generated with `--enhance` flag
2. Property names match exactly (e.g., `goenv:go_version`)
3. Rule check type is correct for your validation

Enable verbose mode to debug:

```bash
goenv sbom validate sbom.json --verbose
```

## Roadmap

**v3.2 (Current):**
- ✅ YAML policy engine
- ✅ Supply chain, security, completeness, license checks
- ✅ CLI validation command

**v3.3 (Planned):**
- 🔲 Custom rule expressions (CEL/Rego)
- 🔲 Policy templates library
- 🔲 Remote policy fetching
- 🔲 Policy impact analysis

**v3.4 (Future):**
- 🔲 Policy-as-code testing
- 🔲 Automated remediation suggestions
- 🔲 Policy violation trending
- 🔲 Integration with OPA

## Resources

- [CycloneDX Specification](https://cyclonedx.org/specification/overview/)
- [SLSA Framework](https://slsa.dev/)
- [SSDF Guidelines](https://csrc.nist.gov/projects/ssdf)
- [Supply Chain Levels for Software Artifacts](https://slsa.dev/spec/v1.0/levels)

## Feedback

Found a bug or have a feature request? [Open an issue](https://github.com/go-nv/goenv/issues/new?labels=sbom,policy)

Want to contribute a policy example? See [CONTRIBUTING.md](../../CONTRIBUTING.md)
