# SBOM Signing and Attestation Guide

This guide covers how to use goenv's SBOM signing and attestation features to secure your software supply chain.

## Overview

goenv provides three main commands for SBOM security:

- `goenv sbom sign` - Sign an SBOM with cryptographic keys or keyless signing
- `goenv sbom verify-signature` - Verify an SBOM signature
- `goenv sbom attest` - Generate SLSA provenance attestations

## Prerequisites

### For Key-Based Signing

No additional tools required - goenv includes built-in ECDSA P-256 signing.

### For Keyless Signing (Optional)

Install [cosign](https://github.com/sigstore/cosign):

```bash
# macOS
brew install cosign

# Linux
wget https://github.com/sigstore/cosign/releases/latest/download/cosign-linux-amd64
chmod +x cosign-linux-amd64
sudo mv cosign-linux-amd64 /usr/local/bin/cosign

# Verify installation
cosign version
```

## Key-Based Signing

### 1. Generate Keys

First, generate a key pair:

```bash
goenv sbom generate-keys \
  --private-key private.pem \
  --public-key public.pem
```

This creates:
- `private.pem` - Keep this secret! Use for signing
- `public.pem` - Share for verification

**Security Note:** Store private keys in a secure location. Consider using a secrets manager or hardware security module (HSM) for production environments.

### 2. Sign an SBOM

Sign an existing SBOM:

```bash
goenv sbom sign myapp.sbom.json \
  --key private.pem \
  --output myapp.sbom.json.sig
```

The signature file contains:
- Cryptographic signature (ECDSA-SHA256)
- Timestamp
- Key identifier
- Metadata

### 3. Verify a Signature

Verify an SBOM hasn't been tampered with:

```bash
goenv sbom verify-signature myapp.sbom.json \
  --signature myapp.sbom.json.sig \
  --key public.pem
```

Output:
```
✓ Signature verified successfully
  Algorithm: ECDSA-SHA256
  Signed: 2025-12-08T21:00:00Z
  Key ID: sha256:abc123...
```

## Keyless Signing (Sigstore/Fulcio)

Keyless signing uses certificate-based identity verification instead of long-lived keys.

### 1. Sign with OIDC Identity

```bash
goenv sbom sign myapp.sbom.json \
  --keyless \
  --oidc-issuer https://oauth2.sigstore.dev/auth \
  --output myapp.sbom.json.sig
```

This will:
1. Open your browser for OIDC authentication
2. Obtain a short-lived certificate from Fulcio
3. Sign the SBOM with the certificate
4. Log the signature to Rekor (transparency log)

### 2. Verify Keyless Signature

```bash
goenv sbom verify-signature myapp.sbom.json \
  --signature myapp.sbom.json.sig \
  --use-cosign
```

This verifies:
- Certificate validity
- Rekor transparency log entry
- Signature correctness

## SLSA Provenance Attestations

SLSA (Supply chain Levels for Software Artifacts) provenance provides verifiable information about how software was built.

### Generate Provenance

Create a SLSA v1.0 provenance attestation:

```bash
goenv sbom attest \
  --output provenance.json \
  --invocation-id "build-$(date +%s)"
```

The attestation includes:
- SBOM digest (SHA-256)
- Build environment (Go version, GOOS, GOARCH)
- Builder identity (goenv version)
- Build parameters (CGO, build tags, ldflags)
- Dependency digests (go.mod, go.sum)
- Reproducibility metadata

### Signed Provenance

Generate and sign provenance in one step:

```bash
goenv sbom attest \
  --output provenance.json \
  --sign \
  --key private.pem
```

### In-toto Attestation Format

Generate an in-toto attestation bundle:

```bash
goenv sbom attest \
  --output attestation.json \
  --in-toto \
  --sign \
  --key private.pem
```

In-toto format is useful for:
- Integration with in-toto supply chain frameworks
- Multi-signature workflows
- Policy-based verification

## Use Cases

### CI/CD Pipeline Integration

```bash
#!/bin/bash
# Generate SBOM
goenv sbom project --output myapp.sbom.json

# Generate and sign attestation
goenv sbom attest \
  --output provenance.json \
  --sign \
  --key "$SIGNING_KEY_PATH" \
  --invocation-id "$CI_PIPELINE_ID"

# Upload artifacts
aws s3 cp myapp.sbom.json s3://releases/myapp/v1.0.0/
aws s3 cp provenance.json s3://releases/myapp/v1.0.0/
```

### Release Verification

```bash
#!/bin/bash
# Download release artifacts
wget https://releases.example.com/myapp/v1.0.0/myapp.sbom.json
wget https://releases.example.com/myapp/v1.0.0/myapp.sbom.json.sig
wget https://releases.example.com/myapp/v1.0.0/public.pem

# Verify signature
goenv sbom verify-signature myapp.sbom.json \
  --signature myapp.sbom.json.sig \
  --key public.pem

# Verify SBOM against binary (if available)
# Additional checks...
```

### Continuous Compliance

```bash
#!/bin/bash
# Daily SBOM generation with attestations
for project in projects/*; do
  cd "$project"
  
  # Generate SBOM
  goenv sbom project --output sbom.json
  
  # Generate attestation
  goenv sbom attest \
    --output attestation.json \
    --sign \
    --key /secure/signing-key.pem
  
  # Archive for compliance
  cp sbom.json "$COMPLIANCE_ARCHIVE/$(basename $project)/$(date +%Y%m%d).sbom.json"
  cp attestation.json "$COMPLIANCE_ARCHIVE/$(basename $project)/$(date +%Y%m%d).attestation.json"
done
```

## Security Best Practices

### Key Management

1. **Never commit private keys** to version control
2. **Use separate keys** for different environments (dev/staging/prod)
3. **Rotate keys regularly** (quarterly or after personnel changes)
4. **Use hardware security modules (HSMs)** for production signing
5. **Implement key access logging** and monitoring

### Signing Policy

1. **Sign all production SBOMs** before distribution
2. **Verify signatures** before deployment
3. **Log all signing operations** with timestamps and identities
4. **Require attestations** for compliance artifacts
5. **Use keyless signing** for ephemeral CI/CD environments

### Attestation Best Practices

1. **Include unique invocation IDs** for traceability
2. **Generate attestations immediately** after SBOM creation
3. **Store attestations separately** from SBOMs (different backup strategy)
4. **Validate attestation schemas** before archiving
5. **Link attestations to git commits** for reproducibility

## Troubleshooting

### Signature Verification Fails

```
Error: signature verification failed: invalid signature
```

**Possible causes:**
- SBOM was modified after signing
- Wrong public key used
- Signature file corrupted

**Solutions:**
- Re-download SBOM and signature from trusted source
- Verify you're using the correct public key
- Check file permissions and integrity

### Cosign Not Found

```
Error: cosign binary not found in PATH
```

**Solution:**
Install cosign:
```bash
brew install cosign  # macOS
# or download from https://github.com/sigstore/cosign/releases
```

### OIDC Authentication Fails

```
Error: failed to obtain OIDC token
```

**Possible causes:**
- No browser available (headless environment)
- OIDC issuer unreachable
- Network proxy issues

**Solutions:**
- Use key-based signing instead for CI/CD
- Configure proxy settings
- Use alternative OIDC issuer

## Reference

### Signature Format

```json
{
  "value": "base64-encoded-signature",
  "algorithm": "ECDSA-SHA256",
  "keyID": "sha256:abc123...",
  "timestamp": "2025-12-08T21:00:00Z",
  "certificate": null
}
```

### SLSA Provenance Format

```json
{
  "_type": "https://in-toto.io/Statement/v1",
  "predicateType": "https://slsa.dev/provenance/v1",
  "subject": [{
    "name": "myapp.sbom.json",
    "digest": { "sha256": "abc123..." }
  }],
  "predicate": {
    "buildDefinition": {
      "buildType": "https://github.com/go-nv/goenv/SBOMBuild/v1",
      "externalParameters": { ... },
      "internalParameters": { ... }
    },
    "runDetails": { ... }
  }
}
```

## Related Commands

- `goenv sbom project` - Generate SBOMs
- `goenv sbom validate` - Validate SBOM format
- `goenv sbom policy` - Check policy compliance

## Further Reading

- [SLSA Framework](https://slsa.dev/)
- [Sigstore Project](https://www.sigstore.dev/)
- [In-toto Framework](https://in-toto.io/)
- [CycloneDX Specification](https://cyclonedx.org/)
- [SPDX Specification](https://spdx.dev/)
