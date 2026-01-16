package compliance

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-nv/goenv/internal/sbom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSBOMSigning_KeyGeneration(t *testing.T) {
	tmpDir := t.TempDir()

	privateKeyPath := filepath.Join(tmpDir, "private.pem")
	publicKeyPath := filepath.Join(tmpDir, "public.pem")

	// Test key generation
	err := sbom.GenerateKeyPair(privateKeyPath, publicKeyPath)
	require.NoError(t, err, "Key generation should succeed")

	// Verify files exist
	assert.FileExists(t, privateKeyPath, "Private key should be created")
	assert.FileExists(t, publicKeyPath, "Public key should be created")

	// Verify files are not empty
	privInfo, err := os.Stat(privateKeyPath)
	require.NoError(t, err)
	assert.Greater(t, privInfo.Size(), int64(0), "Private key should not be empty")

	pubInfo, err := os.Stat(publicKeyPath)
	require.NoError(t, err)
	assert.Greater(t, pubInfo.Size(), int64(0), "Public key should not be empty")
}

func TestSBOMSigning_SignAndVerify(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping signing test in short mode")
	}

	tmpDir := t.TempDir()

	// Create test SBOM
	sbomPath := filepath.Join(tmpDir, "test.sbom.json")
	testSBOM := []byte(`{
		"bomFormat": "CycloneDX",
		"specVersion": "1.5",
		"version": 1,
		"metadata": {
			"component": {
				"name": "test-component",
				"version": "1.0.0"
			}
		}
	}`)
	err := os.WriteFile(sbomPath, testSBOM, 0644)
	require.NoError(t, err)

	// Generate keys
	privateKeyPath := filepath.Join(tmpDir, "private.pem")
	publicKeyPath := filepath.Join(tmpDir, "public.pem")
	err = sbom.GenerateKeyPair(privateKeyPath, publicKeyPath)
	require.NoError(t, err)

	// Sign SBOM
	signaturePath := filepath.Join(tmpDir, "test.sbom.json.sig")

	signer := sbom.NewSigner(sbom.SignatureOptions{
		KeyPath: privateKeyPath,
	})
	sig, err := signer.SignSBOM(sbomPath)
	require.NoError(t, err, "Signing should succeed")

	err = signer.WriteSignature(sig, signaturePath)
	require.NoError(t, err)
	assert.FileExists(t, signaturePath, "Signature file should be created")

	// Verify signature
	verifier := sbom.NewVerifier(sbom.VerificationOptions{
		SBOMPath:      sbomPath,
		SignaturePath: signaturePath,
		PublicKeyPath: publicKeyPath,
	})

	verified, err := verifier.VerifySignature()
	require.NoError(t, err, "Verification should succeed")
	assert.True(t, verified, "Signature should be valid")
}

func TestSBOMSigning_InvalidSignature(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping signing test in short mode")
	}

	tmpDir := t.TempDir()

	// Create test SBOM
	sbomPath := filepath.Join(tmpDir, "test.sbom.json")
	testSBOM := []byte(`{"bomFormat": "CycloneDX", "specVersion": "1.5"}`)
	err := os.WriteFile(sbomPath, testSBOM, 0644)
	require.NoError(t, err)

	// Create tampered signature
	signaturePath := filepath.Join(tmpDir, "test.sbom.json.sig")
	tamperedSig := []byte(`{"signature": "aW52YWxpZA==", "algorithm": "ECDSA-SHA256", "timestamp": "2024-01-01T00:00:00Z"}`)
	err = os.WriteFile(signaturePath, tamperedSig, 0644)
	require.NoError(t, err)

	// Generate keys (for public key)
	privateKeyPath := filepath.Join(tmpDir, "private.pem")
	publicKeyPath := filepath.Join(tmpDir, "public.pem")
	err = sbom.GenerateKeyPair(privateKeyPath, publicKeyPath)
	require.NoError(t, err)

	// Try to verify - should fail
	verifier := sbom.NewVerifier(sbom.VerificationOptions{
		SBOMPath:      sbomPath,
		SignaturePath: signaturePath,
		PublicKeyPath: publicKeyPath,
	})

	verified, err := verifier.VerifySignature()
	// Either the verification should fail with an error or return false
	if err == nil {
		assert.False(t, verified, "Verification should fail for tampered signature")
	}
}

func TestSBOMAttestation_Generation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping attestation test in short mode")
	}

	tmpDir := t.TempDir()

	// Create minimal go.mod for testing
	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := []byte(`module example.com/test

go 1.23

require (
	github.com/example/dep v1.0.0
)
`)
	err := os.WriteFile(goModPath, goModContent, 0644)
	require.NoError(t, err)

	// Create go.sum
	goSumPath := filepath.Join(tmpDir, "go.sum")
	goSumContent := []byte(`github.com/example/dep v1.0.0 h1:abc123
github.com/example/dep v1.0.0/go.mod h1:xyz789
`)
	err = os.WriteFile(goSumPath, goSumContent, 0644)
	require.NoError(t, err)

	// Generate attestation
	attestPath := filepath.Join(tmpDir, "provenance.json")

	// Change to tmpDir for go.mod detection
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create a dummy SBOM file for the attestation
	sbomPath := filepath.Join(tmpDir, "test.sbom.json")
	err = os.WriteFile(sbomPath, []byte(`{"bomFormat":"CycloneDX"}`), 0644)
	require.NoError(t, err)

	generator := sbom.NewProvenanceGenerator(sbom.ProvenanceOptions{
		SBOMPath:   sbomPath,
		ProjectDir: tmpDir,
		GoVersion:  "1.23",
	})
	provenance, err := generator.Generate()
	require.NoError(t, err, "Attestation generation should succeed")

	err = generator.WriteProvenance(provenance, attestPath)
	require.NoError(t, err)
	assert.FileExists(t, attestPath, "Attestation file should be created")

	// Verify it's valid JSON with expected fields
	data, err := os.ReadFile(attestPath)
	require.NoError(t, err)
	// SLSA v1.0 uses predicateType, not slsaVersion
	assert.Contains(t, string(data), "predicateType", "Should contain predicate type")
	assert.Contains(t, string(data), "buildDefinition", "Should contain build definition")
	assert.Contains(t, string(data), "runDetails", "Should contain run details")
}

func TestSBOMAttestation_WithInToto(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping attestation test in short mode")
	}

	tmpDir := t.TempDir()

	// Create minimal go.mod
	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := []byte(`module example.com/test

go 1.23
`)
	err := os.WriteFile(goModPath, goModContent, 0644)
	require.NoError(t, err)

	// Create go.sum
	goSumPath := filepath.Join(tmpDir, "go.sum")
	err = os.WriteFile(goSumPath, []byte(""), 0644)
	require.NoError(t, err)

	// Change to tmpDir
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create a dummy SBOM file
	sbomPath := filepath.Join(tmpDir, "test.sbom.json")
	err = os.WriteFile(sbomPath, []byte(`{"bomFormat":"CycloneDX"}`), 0644)
	require.NoError(t, err)

	// Generate provenance
	generator := sbom.NewProvenanceGenerator(sbom.ProvenanceOptions{
		SBOMPath:   sbomPath,
		ProjectDir: tmpDir,
		GoVersion:  "1.23",
	})
	provenance, err := generator.Generate()
	require.NoError(t, err)

	// Create in-toto attestation
	attestation, err := sbom.CreateInTotoAttestation(provenance, nil)
	require.NoError(t, err, "In-toto attestation creation should succeed")

	// Write to file
	attestPath := filepath.Join(tmpDir, "attestation.json")
	err = sbom.WriteInTotoAttestation(attestation, attestPath)
	require.NoError(t, err)
	assert.FileExists(t, attestPath, "In-toto attestation file should be created")

	// Verify contents - in-toto format
	data, err := os.ReadFile(attestPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "payloadType", "Should contain payloadType field")
	assert.Contains(t, string(data), "application/vnd.in-toto+json", "Should use in-toto payload type")
	assert.Contains(t, string(data), "payload", "Should contain base64-encoded payload")
	assert.Contains(t, string(data), "signatures", "Should contain signatures array")
}

func TestSBOMSigning_CosignAvailability(t *testing.T) {
	// This is a simple check - not a full integration test
	available := sbom.IsCosignAvailable()

	// Just verify the function runs without panic
	t.Logf("Cosign available: %v", available)

	// If cosign is not available, that's okay - this is a feature check
	if !available {
		t.Log("Cosign not available - keyless signing will not work")
	}
}
