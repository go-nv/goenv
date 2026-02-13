package sbom

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateKeyPair(t *testing.T) {
	tmpDir := t.TempDir()
	privateKeyPath := filepath.Join(tmpDir, "private.pem")
	publicKeyPath := filepath.Join(tmpDir, "public.pem")

	err := GenerateKeyPair(privateKeyPath, publicKeyPath)
	require.NoError(t, err, "Key generation should succeed")

	// Verify files exist
	assert.FileExists(t, privateKeyPath)
	assert.FileExists(t, publicKeyPath)

	// Verify private key is valid
	privData, err := os.ReadFile(privateKeyPath)
	require.NoError(t, err)
	privBlock, _ := pem.Decode(privData)
	require.NotNil(t, privBlock, "Private key should be valid PEM")
	assert.Equal(t, "EC PRIVATE KEY", privBlock.Type)

	privKey, err := x509.ParseECPrivateKey(privBlock.Bytes)
	require.NoError(t, err, "Private key should parse correctly")
	assert.Equal(t, "P-256", privKey.Curve.Params().Name)

	// Verify public key is valid
	pubData, err := os.ReadFile(publicKeyPath)
	require.NoError(t, err)
	pubBlock, _ := pem.Decode(pubData)
	require.NotNil(t, pubBlock, "Public key should be valid PEM")
	assert.Equal(t, "PUBLIC KEY", pubBlock.Type)

	pubKeyInterface, err := x509.ParsePKIXPublicKey(pubBlock.Bytes)
	require.NoError(t, err, "Public key should parse correctly")

	pubKey, ok := pubKeyInterface.(*ecdsa.PublicKey)
	require.True(t, ok, "Public key should be ECDSA")
	assert.Equal(t, "P-256", pubKey.Curve.Params().Name)

	// Verify keys match
	assert.Equal(t, privKey.PublicKey.X, pubKey.X)
	assert.Equal(t, privKey.PublicKey.Y, pubKey.Y)
}

func TestGenerateKeyPair_InvalidPaths(t *testing.T) {
	tests := []struct {
		name           string
		privateKeyPath string
		publicKeyPath  string
		expectError    bool
	}{
		{
			name:           "empty private key path",
			privateKeyPath: "",
			publicKeyPath:  "/tmp/public.pem",
			expectError:    true,
		},
		{
			name:           "empty public key path",
			privateKeyPath: "/tmp/private.pem",
			publicKeyPath:  "",
			expectError:    true,
		},
		{
			name:           "invalid directory",
			privateKeyPath: "/nonexistent/dir/private.pem",
			publicKeyPath:  "/nonexistent/dir/public.pem",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := GenerateKeyPair(tt.privateKeyPath, tt.publicKeyPath)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSigner_KeyBased(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate test keys
	privateKeyPath := filepath.Join(tmpDir, "private.pem")
	publicKeyPath := filepath.Join(tmpDir, "public.pem")
	err := GenerateKeyPair(privateKeyPath, publicKeyPath)
	require.NoError(t, err)

	// Create test SBOM
	sbomPath := filepath.Join(tmpDir, "test.sbom.json")
	testSBOM := map[string]interface{}{
		"bomFormat":   "CycloneDX",
		"specVersion": "1.5",
		"version":     1,
	}
	data, err := json.MarshalIndent(testSBOM, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(sbomPath, data, 0644)
	require.NoError(t, err)

	// Sign the SBOM
	signer := NewSigner(SignatureOptions{
		KeyPath: privateKeyPath,
	})

	sig, err := signer.SignSBOM(sbomPath)
	require.NoError(t, err, "Signing should succeed")
	assert.NotEmpty(t, sig.Value)
	assert.Equal(t, "ecdsa-p256-sha256", sig.Algorithm)
	assert.NotEmpty(t, sig.KeyID)
	assert.NotEmpty(t, sig.Timestamp)

	// Write signature to file
	sigPath := filepath.Join(tmpDir, "test.sbom.json.sig")
	err = signer.WriteSignature(sig, sigPath)
	require.NoError(t, err)
	assert.FileExists(t, sigPath)
}

func TestSigner_InvalidKey(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test SBOM
	sbomPath := filepath.Join(tmpDir, "test.sbom.json")
	err := os.WriteFile(sbomPath, []byte(`{"bomFormat":"CycloneDX"}`), 0644)
	require.NoError(t, err)

	// Try to sign with non-existent key
	signer := NewSigner(SignatureOptions{
		KeyPath: "/nonexistent/key.pem",
	})

	_, err = signer.SignSBOM(sbomPath)
	assert.Error(t, err, "Should fail with invalid key path")
}

func TestSigner_InvalidSBOM(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate test keys
	privateKeyPath := filepath.Join(tmpDir, "private.pem")
	publicKeyPath := filepath.Join(tmpDir, "public.pem")
	err := GenerateKeyPair(privateKeyPath, publicKeyPath)
	require.NoError(t, err)

	// Try to sign non-existent SBOM
	signer := NewSigner(SignatureOptions{
		KeyPath: privateKeyPath,
	})

	_, err = signer.SignSBOM("/nonexistent/sbom.json")
	assert.Error(t, err, "Should fail with invalid SBOM path")
}

func TestVerifier_KeyBased(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate test keys
	privateKeyPath := filepath.Join(tmpDir, "private.pem")
	publicKeyPath := filepath.Join(tmpDir, "public.pem")
	err := GenerateKeyPair(privateKeyPath, publicKeyPath)
	require.NoError(t, err)

	// Create and sign test SBOM
	sbomPath := filepath.Join(tmpDir, "test.sbom.json")
	testSBOM := map[string]interface{}{
		"bomFormat":   "CycloneDX",
		"specVersion": "1.5",
		"version":     1,
	}
	data, err := json.MarshalIndent(testSBOM, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(sbomPath, data, 0644)
	require.NoError(t, err)

	signer := NewSigner(SignatureOptions{
		KeyPath: privateKeyPath,
	})
	sig, err := signer.SignSBOM(sbomPath)
	require.NoError(t, err)

	sigPath := filepath.Join(tmpDir, "test.sbom.json.sig")
	err = signer.WriteSignature(sig, sigPath)
	require.NoError(t, err)

	// Verify the signature
	verifier := NewVerifier(VerificationOptions{
		SBOMPath:      sbomPath,
		SignaturePath: sigPath,
		PublicKeyPath: publicKeyPath,
	})

	verified, err := verifier.VerifySignature()
	require.NoError(t, err, "Verification should succeed")
	assert.True(t, verified, "Signature should be valid")
}

func TestVerifier_TamperedSBOM(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate test keys
	privateKeyPath := filepath.Join(tmpDir, "private.pem")
	publicKeyPath := filepath.Join(tmpDir, "public.pem")
	err := GenerateKeyPair(privateKeyPath, publicKeyPath)
	require.NoError(t, err)

	// Create and sign test SBOM
	sbomPath := filepath.Join(tmpDir, "test.sbom.json")
	originalSBOM := map[string]interface{}{
		"bomFormat":   "CycloneDX",
		"specVersion": "1.5",
		"version":     1,
	}
	data, err := json.MarshalIndent(originalSBOM, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(sbomPath, data, 0644)
	require.NoError(t, err)

	signer := NewSigner(SignatureOptions{
		KeyPath: privateKeyPath,
	})
	sig, err := signer.SignSBOM(sbomPath)
	require.NoError(t, err)

	sigPath := filepath.Join(tmpDir, "test.sbom.json.sig")
	err = signer.WriteSignature(sig, sigPath)
	require.NoError(t, err)

	// Tamper with the SBOM
	tamperedSBOM := map[string]interface{}{
		"bomFormat":   "CycloneDX",
		"specVersion": "1.5",
		"version":     2, // Changed version
	}
	data, err = json.MarshalIndent(tamperedSBOM, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(sbomPath, data, 0644)
	require.NoError(t, err)

	// Verify should fail
	verifier := NewVerifier(VerificationOptions{
		SBOMPath:      sbomPath,
		SignaturePath: sigPath,
		PublicKeyPath: publicKeyPath,
	})

	verified, err := verifier.VerifySignature()
	// Either should return error or return false
	if err == nil {
		assert.False(t, verified, "Signature should be invalid for tampered SBOM")
	}
}

func TestVerifier_InvalidSignatureFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate test keys
	privateKeyPath := filepath.Join(tmpDir, "private.pem")
	publicKeyPath := filepath.Join(tmpDir, "public.pem")
	err := GenerateKeyPair(privateKeyPath, publicKeyPath)
	require.NoError(t, err)

	// Create test SBOM
	sbomPath := filepath.Join(tmpDir, "test.sbom.json")
	err = os.WriteFile(sbomPath, []byte(`{"bomFormat":"CycloneDX"}`), 0644)
	require.NoError(t, err)

	// Create invalid signature file
	sigPath := filepath.Join(tmpDir, "test.sbom.json.sig")
	err = os.WriteFile(sigPath, []byte(`{"invalid":"json"}`), 0644)
	require.NoError(t, err)

	// Verify should fail
	verifier := NewVerifier(VerificationOptions{
		SBOMPath:      sbomPath,
		SignaturePath: sigPath,
		PublicKeyPath: publicKeyPath,
	})

	verified, err := verifier.VerifySignature()
	// Should either error or return false for invalid signature
	if err == nil {
		assert.False(t, verified, "Should return false with invalid signature file")
	}
}

func TestVerifier_WrongPublicKey(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate first key pair
	privateKeyPath1 := filepath.Join(tmpDir, "private1.pem")
	publicKeyPath1 := filepath.Join(tmpDir, "public1.pem")
	err := GenerateKeyPair(privateKeyPath1, publicKeyPath1)
	require.NoError(t, err)

	// Generate second key pair
	privateKeyPath2 := filepath.Join(tmpDir, "private2.pem")
	publicKeyPath2 := filepath.Join(tmpDir, "public2.pem")
	err = GenerateKeyPair(privateKeyPath2, publicKeyPath2)
	require.NoError(t, err)

	// Create and sign with first key
	sbomPath := filepath.Join(tmpDir, "test.sbom.json")
	err = os.WriteFile(sbomPath, []byte(`{"bomFormat":"CycloneDX"}`), 0644)
	require.NoError(t, err)

	signer := NewSigner(SignatureOptions{
		KeyPath: privateKeyPath1,
	})
	sig, err := signer.SignSBOM(sbomPath)
	require.NoError(t, err)

	sigPath := filepath.Join(tmpDir, "test.sbom.json.sig")
	err = signer.WriteSignature(sig, sigPath)
	require.NoError(t, err)

	// Try to verify with wrong public key
	verifier := NewVerifier(VerificationOptions{
		SBOMPath:      sbomPath,
		SignaturePath: sigPath,
		PublicKeyPath: publicKeyPath2, // Wrong key!
	})

	verified, err := verifier.VerifySignature()
	// Should either error or return false
	if err == nil {
		assert.False(t, verified, "Verification should fail with wrong public key")
	}
}

func TestIsCosignAvailable(t *testing.T) {
	// This test just verifies the function runs without panic
	available := IsCosignAvailable()
	t.Logf("Cosign available: %v", available)

	// No assertion - we just want to ensure it doesn't panic
	// The result depends on the test environment
}

func TestNormalizeSBOMForSigning(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "valid JSON",
			input:       `{"bomFormat":"CycloneDX","version":1}`,
			expectError: false,
		},
		{
			name:        "JSON with whitespace variations",
			input:       `{"bomFormat": "CycloneDX" , "version" :1 }`,
			expectError: false,
		},
		{
			name:        "invalid JSON",
			input:       `{invalid json}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normalizeSBOMForSigning([]byte(tt.input))

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, result)

				// Verify it's valid JSON
				var parsed map[string]interface{}
				err = json.Unmarshal(result, &parsed)
				assert.NoError(t, err, "Normalized output should be valid JSON")
			}
		})
	}
}

func TestSignature_JSON(t *testing.T) {
	timestamp := time.Date(2025, 12, 8, 12, 0, 0, 0, time.UTC)

	sig := &Signature{
		Value:     "test-signature-value",
		Algorithm: "ECDSA-SHA256",
		KeyID:     "test-key-id",
		Timestamp: timestamp,
	}

	// Marshal to JSON
	data, err := json.Marshal(sig)
	require.NoError(t, err)
	assert.Contains(t, string(data), "test-signature-value")
	assert.Contains(t, string(data), "ECDSA-SHA256")

	// Unmarshal from JSON
	var parsed Signature
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Equal(t, sig.Value, parsed.Value)
	assert.Equal(t, sig.Algorithm, parsed.Algorithm)
	assert.Equal(t, sig.KeyID, parsed.KeyID)
	assert.Equal(t, sig.Timestamp.Unix(), parsed.Timestamp.Unix())
}

func TestSignatureOptions_Validation(t *testing.T) {
	tests := []struct {
		name    string
		opts    SignatureOptions
		wantErr bool
	}{
		{
			name: "valid key-based",
			opts: SignatureOptions{
				KeyPath: "/path/to/key.pem",
			},
			wantErr: false,
		},
		{
			name: "valid keyless",
			opts: SignatureOptions{
				Keyless:    true,
				OIDCIssuer: "https://oauth2.sigstore.dev/auth",
			},
			wantErr: false,
		},
		{
			name:    "empty options",
			opts:    SignatureOptions{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test validates that our options struct has expected fields
			// The actual validation happens in the sign command
			if tt.wantErr {
				assert.True(t, tt.opts.KeyPath == "" && !tt.opts.Keyless)
			} else {
				assert.True(t, tt.opts.KeyPath != "" || tt.opts.Keyless)
			}
		})
	}
}
