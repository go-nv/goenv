# SBOM Policy Examples

This directory contains example policy configurations for different use cases and security postures.

## Quick Start

```bash
# Use a policy
goenv sbom validate sbom.json --policy=examples/policies/ci-cd.yaml

# With verbose output
goenv sbom validate sbom.json --policy=examples/policies/enterprise-strict.yaml --verbose
```

## Available Policies

### 1. `enterprise-strict.yaml`

**Use case:** Regulated industries, high-security environments, SOC 2 compliance

**Features:**
- Zero tolerance for local dependencies
- No vendoring allowed  
- CGO must be disabled
- Complete build provenance required
- Strict license controls (no GPL/AGPL)
- All warnings treated as failures

**When to use:**
- Financial services
- Healthcare (HIPAA)
- Government contracts
- PCI-DSS compliance

```bash
goenv sbom validate sbom.json --policy=examples/policies/enterprise-strict.yaml
```

### 2. `ci-cd.yaml`

**Use case:** Continuous integration pipelines, automated builds

**Features:**
- Warns on local replaces (doesn't block)
- Blocks retracted versions
- Requires build metadata for provenance
- Permissive license policy
- Balanced between security and developer productivity

**When to use:**
- GitHub Actions workflows
- GitLab CI/CD
- Jenkins pipelines
- Automated releases

```bash
goenv sbom validate sbom.json --policy=examples/policies/ci-cd.yaml
```

### 3. `open-source.yaml`

**Use case:** Public open source projects, community contributions

**Features:**
- Informational warnings only
- No build failures
- Encourages best practices
- Helps contributors learn

**When to use:**
- Public GitHub repositories
- Community-driven projects
- Learning environments
- Development/staging

```bash
goenv sbom validate sbom.json --policy=examples/policies/open-source.yaml
```

## Integration Examples

### GitHub Actions

`.github/workflows/sbom.yml`:

```yaml
name: SBOM Validation

on: [push, pull_request]

jobs:
  sbom:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Install goenv
        run: |
          curl -fsSL https://github.com/go-nv/goenv/releases/latest/download/install.sh | bash
          echo "$HOME/.goenv/bin" >> $GITHUB_PATH
      
      - name: Generate SBOM
        run: goenv sbom project -o sbom.json --enhance --deterministic
      
      - name: Validate SBOM
        run: goenv sbom validate sbom.json --policy=examples/policies/ci-cd.yaml
      
      - name: Upload SBOM
        uses: actions/upload-artifact@v4
        with:
          name: sbom
          path: sbom.json
```

### GitLab CI

`.gitlab-ci.yml`:

```yaml
sbom:generate:
  script:
    - goenv sbom project -o sbom.json --enhance
    - goenv sbom validate sbom.json --policy=examples/policies/ci-cd.yaml
  artifacts:
    paths:
      - sbom.json
    reports:
      cyclonedx: sbom.json
```

### Pre-commit Hook

`.git/hooks/pre-commit`:

```bash
#!/bin/bash

# Regenerate SBOM if go.mod changed
if git diff --cached --name-only | grep -q "go.mod\|go.sum"; then
    echo "Regenerating SBOM..."
    goenv sbom project -o sbom.json --enhance
    
    echo "Validating SBOM..."
    if ! goenv sbom validate sbom.json --policy=.goenv-policy.yaml; then
        echo "SBOM validation failed!"
        exit 1
    fi
    
    git add sbom.json
fi
```

## Customizing Policies

### Start with a Template

Copy an example policy and modify it:

```bash
cp examples/policies/ci-cd.yaml .goenv-policy.yaml
# Edit .goenv-policy.yaml to match your needs
```

### Progressive Enhancement

Start permissive, tighten over time:

**Week 1-2: Learn**
```yaml
- name: check-local-replaces
  severity: info  # Just observe patterns
```

**Week 3-4: Warn**
```yaml
- name: check-local-replaces
  severity: warning  # Alert but don't block
```

**Week 5+: Enforce**
```yaml
- name: check-local-replaces
  severity: error  # Block builds
```

### Environment-Specific Policies

```
policies/
├── development.yaml    # Permissive
├── staging.yaml        # Moderate
└── production.yaml     # Strict
```

```bash
# Use environment variable
export POLICY_FILE="policies/${ENVIRONMENT}.yaml"
goenv sbom validate sbom.json --policy=$POLICY_FILE
```

## Testing Policies

### Dry Run (Info Only)

Temporarily change all rules to `info` severity to see what would fail:

```bash
# Copy policy
cp .goenv-policy.yaml test-policy.yaml

# Edit: Change all severity: error to severity: info

# Test
goenv sbom validate sbom.json --policy=test-policy.yaml --verbose
```

### Gradual Rollout

1. **Measure:** Run with `--verbose` to see violations
2. **Communicate:** Share results with team
3. **Remediate:** Fix violations incrementally
4. **Enforce:** Enable errors when team is ready

## Common Patterns

### Block Specific Licenses

```yaml
- name: no-copyleft
  type: license
  severity: error
  blocked:
    - GPL-2.0
    - GPL-3.0
    - AGPL-3.0
```

### Require Security Metadata

```yaml
- name: require-security-context
  type: completeness
  severity: error
  check: required-metadata
  required:
    - goenv:build_context.cgo_enabled
    - goenv:module_context.vendored
    - goenv:module_context.go_mod_digest
```

### Development vs Production

```yaml
# development.yaml - Allow local replaces
- name: local-replaces-allowed
  type: supply-chain
  severity: info
  check: replace-directives

# production.yaml - Block local replaces  
- name: no-local-replaces
  type: supply-chain
  severity: error
  check: replace-directives
  blocked:
    - local-path
```

## Troubleshooting

### "Policy file not found"

Ensure the policy file exists:

```bash
ls -la .goenv-policy.yaml
# or
goenv sbom validate sbom.json --policy=examples/policies/ci-cd.yaml
```

### "Required metadata missing"

Regenerate SBOM with enhancement:

```bash
goenv sbom project -o sbom.json --enhance
```

### "Component not found"

Some components (like `golang-stdlib`) require specific SBOM generation options. Check the [SBOM Policy Guide](../../docs/user-guide/SBOM_POLICY_GUIDE.md) for details.

## Contributing

Have a policy that works well for your organization? Consider contributing it!

1. Anonymize any sensitive information
2. Add clear documentation
3. Submit a PR with your policy in `examples/policies/`

See [CONTRIBUTING.md](../../CONTRIBUTING.md) for guidelines.

## Resources

- [Full Policy Guide](../../docs/user-guide/SBOM_POLICY_GUIDE.md)
- [SBOM Strategy](../../docs/roadmap/SBOM_STRATEGY.md)
- [CycloneDX Specification](https://cyclonedx.org/)
- [SLSA Framework](https://slsa.dev/)
