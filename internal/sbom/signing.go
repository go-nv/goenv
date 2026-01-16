package sbom

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// SignatureOptions contains configuration for SBOM signing
type SignatureOptions struct {
	// KeyPath is the path to private key file (for key-based signing)
	KeyPath string
	// KeyPassword is the password for encrypted private keys
	KeyPassword string
	// Keyless enables keyless signing via Sigstore/Fulcio
	Keyless bool
	// OIDCIssuer for keyless signing
	OIDCIssuer string
	// OIDCClientID for keyless signing
	OIDCClientID string
	// OutputPath is where to write the signature
	OutputPath string
	// Algorithm specifies the signing algorithm (ecdsa-p256, rsa-2048)
	Algorithm string
}

// VerificationOptions contains configuration for signature verification
type VerificationOptions struct {
	// SBOMPath is the path to the SBOM file
	SBOMPath string
	// SignaturePath is the path to the signature file
	SignaturePath string
	// PublicKeyPath is the path to public key file (for key-based verification)
	PublicKeyPath string
	// CertPath is the path to certificate for keyless verification
	CertPath string
	// UseCosign enables verification via cosign CLI
	UseCosign bool
}

// Signature represents a cryptographic signature for an SBOM
type Signature struct {
	// Algorithm used for signing (e.g., "ecdsa-p256-sha256")
	Algorithm string `json:"algorithm"`
	// Value is the base64-encoded signature
	Value string `json:"value"`
	// KeyID identifies the signing key
	KeyID string `json:"keyID,omitempty"`
	// Certificate for keyless signing
	Certificate string `json:"certificate,omitempty"`
	// Timestamp when signature was created
	Timestamp time.Time `json:"timestamp"`
	// SignedBy contains identity information
	SignedBy *SignerIdentity `json:"signedBy,omitempty"`
}

// SignerIdentity contains information about who signed the SBOM
type SignerIdentity struct {
	// Email of the signer (for keyless signing)
	Email string `json:"email,omitempty"`
	// Issuer (e.g., "https://accounts.google.com")
	Issuer string `json:"issuer,omitempty"`
	// Subject (often same as email)
	Subject string `json:"subject,omitempty"`
}

// Signer provides SBOM signing capabilities
type Signer struct {
	options SignatureOptions
}

// NewSigner creates a new SBOM signer with the given options
func NewSigner(opts SignatureOptions) *Signer {
	return &Signer{
		options: opts,
	}
}

// SignSBOM signs an SBOM file and returns the signature
func (s *Signer) SignSBOM(sbomPath string) (*Signature, error) {
	// Read SBOM file
	sbomData, err := os.ReadFile(sbomPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SBOM file %s: %w", sbomPath, err)
	}

	// Normalize SBOM for consistent signing
	normalizedData, err := normalizeSBOMForSigning(sbomData)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize SBOM: %w", err)
	}

	// Choose signing method
	if s.options.Keyless {
		return s.signKeyless(normalizedData, sbomPath)
	}

	return s.signWithKey(normalizedData)
}

// signWithKey performs key-based signing
func (s *Signer) signWithKey(data []byte) (*Signature, error) {
	if s.options.KeyPath == "" {
		return nil, fmt.Errorf("key path required for key-based signing")
	}

	// Read private key
	keyData, err := os.ReadFile(s.options.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file %s: %w", s.options.KeyPath, err)
	}

	// Parse private key
	privateKey, err := parsePrivateKey(keyData, s.options.KeyPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Compute digest
	digest := sha256.Sum256(data)

	// Sign based on key type
	var signatureBytes []byte
	var algorithm string

	switch key := privateKey.(type) {
	case *ecdsa.PrivateKey:
		signatureBytes, err = ecdsa.SignASN1(rand.Reader, key, digest[:])
		if err != nil {
			return nil, fmt.Errorf("failed to sign with ECDSA: %w", err)
		}
		algorithm = "ecdsa-p256-sha256"

	default:
		return nil, fmt.Errorf("unsupported key type: %T", privateKey)
	}

	// Generate key ID
	keyID := generateKeyID(privateKey)

	signature := &Signature{
		Algorithm: algorithm,
		Value:     base64.StdEncoding.EncodeToString(signatureBytes),
		KeyID:     keyID,
		Timestamp: time.Now().UTC(),
	}

	return signature, nil
}

// signKeyless performs keyless signing via cosign/Sigstore
func (s *Signer) signKeyless(data []byte, sbomPath string) (*Signature, error) {
	// Check if cosign is available
	cosignPath, err := exec.LookPath("cosign")
	if err != nil {
		return nil, fmt.Errorf("cosign not found in PATH; install it for keyless signing: %w", err)
	}

	// Create temporary directory for outputs
	tempDir, err := os.MkdirTemp("", "goenv-sbom-sign-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	sigPath := filepath.Join(tempDir, "sbom.sig")
	certPath := filepath.Join(tempDir, "sbom.cert")

	// Build cosign command
	args := []string{
		"sign-blob",
		"--output-signature", sigPath,
		"--output-certificate", certPath,
	}

	// Add OIDC parameters if specified
	if s.options.OIDCIssuer != "" {
		args = append(args, "--oidc-issuer", s.options.OIDCIssuer)
	}
	if s.options.OIDCClientID != "" {
		args = append(args, "--oidc-client-id", s.options.OIDCClientID)
	}

	args = append(args, sbomPath)

	// Execute cosign
	cmd := exec.Command(cosignPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("cosign sign failed: %w\nStderr: %s", err, stderr.String())
	}

	// Read signature
	sigData, err := os.ReadFile(sigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read signature: %w", err)
	}

	// Read certificate
	certData, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate: %w", err)
	}

	signature := &Signature{
		Algorithm:   "cosign-keyless",
		Value:       base64.StdEncoding.EncodeToString(sigData),
		Certificate: base64.StdEncoding.EncodeToString(certData),
		Timestamp:   time.Now().UTC(),
	}

	// Try to extract identity from certificate
	identity, err := extractIdentityFromCert(certData)
	if err == nil {
		signature.SignedBy = identity
	}

	return signature, nil
}

// WriteSignature writes a signature to a file
func (s *Signer) WriteSignature(sig *Signature, outputPath string) error {
	data, err := json.MarshalIndent(sig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal signature: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write signature file %s: %w", outputPath, err)
	}

	return nil
}

// Verifier provides SBOM signature verification
type Verifier struct {
	options VerificationOptions
}

// NewVerifier creates a new signature verifier
func NewVerifier(opts VerificationOptions) *Verifier {
	return &Verifier{
		options: opts,
	}
}

// VerifySignature verifies an SBOM signature
func (v *Verifier) VerifySignature() (bool, error) {
	// If using cosign, delegate to cosign CLI
	if v.options.UseCosign {
		return v.verifyCosign()
	}

	// Otherwise, do key-based verification
	return v.verifyWithKey()
}

// verifyCosign uses cosign CLI to verify signature
func (v *Verifier) verifyCosign() (bool, error) {
	cosignPath, err := exec.LookPath("cosign")
	if err != nil {
		return false, fmt.Errorf("cosign not found in PATH: %w", err)
	}

	args := []string{
		"verify-blob",
		"--signature", v.options.SignaturePath,
	}

	// Add certificate if provided
	if v.options.CertPath != "" {
		args = append(args, "--certificate", v.options.CertPath)
	}

	// Add public key if provided
	if v.options.PublicKeyPath != "" {
		args = append(args, "--key", v.options.PublicKeyPath)
	}

	args = append(args, v.options.SBOMPath)

	cmd := exec.Command(cosignPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("verification failed: %w\nStderr: %s", err, stderr.String())
	}

	return true, nil
}

// verifyWithKey performs key-based signature verification
func (v *Verifier) verifyWithKey() (bool, error) {
	if v.options.PublicKeyPath == "" {
		return false, fmt.Errorf("public key path required for key-based verification")
	}

	// Read SBOM
	sbomData, err := os.ReadFile(v.options.SBOMPath)
	if err != nil {
		return false, fmt.Errorf("failed to read SBOM file %s: %w", v.options.SBOMPath, err)
	}

	// Normalize SBOM
	normalizedData, err := normalizeSBOMForSigning(sbomData)
	if err != nil {
		return false, fmt.Errorf("failed to normalize SBOM: %w", err)
	}

	// Read signature
	sigData, err := os.ReadFile(v.options.SignaturePath)
	if err != nil {
		return false, fmt.Errorf("failed to read signature file %s: %w", v.options.SignaturePath, err)
	}

	var signature Signature
	if err := json.Unmarshal(sigData, &signature); err != nil {
		return false, fmt.Errorf("failed to parse signature: %w", err)
	}

	// Read public key
	pubKeyData, err := os.ReadFile(v.options.PublicKeyPath)
	if err != nil {
		return false, fmt.Errorf("failed to read public key file %s: %w", v.options.PublicKeyPath, err)
	}

	publicKey, err := parsePublicKey(pubKeyData)
	if err != nil {
		return false, fmt.Errorf("failed to parse public key: %w", err)
	}

	// Decode signature
	sigBytes, err := base64.StdEncoding.DecodeString(signature.Value)
	if err != nil {
		return false, fmt.Errorf("failed to decode signature: %w", err)
	}

	// Compute digest
	digest := sha256.Sum256(normalizedData)

	// Verify based on key type
	switch key := publicKey.(type) {
	case *ecdsa.PublicKey:
		valid := ecdsa.VerifyASN1(key, digest[:], sigBytes)
		return valid, nil

	default:
		return false, fmt.Errorf("unsupported key type: %T", publicKey)
	}
}

// Helper functions

// normalizeSBOMForSigning normalizes SBOM data for consistent signing
func normalizeSBOMForSigning(data []byte) ([]byte, error) {
	// Parse as JSON
	var sbom map[string]interface{}
	if err := json.Unmarshal(data, &sbom); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Remove signature metadata if present
	delete(sbom, "signatures")
	delete(sbom, "signature")

	// Remove generation timestamps for reproducibility
	if metadata, ok := sbom["metadata"].(map[string]interface{}); ok {
		delete(metadata, "timestamp")
		// Keep other metadata intact
	}

	// Re-serialize with consistent formatting
	normalized, err := json.Marshal(sbom)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize: %w", err)
	}

	return normalized, nil
}

// parsePrivateKey parses a PEM-encoded private key
func parsePrivateKey(keyData []byte, password string) (crypto.PrivateKey, error) {
	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	var keyBytes []byte
	var err error

	// Handle encrypted keys
	if password != "" && x509.IsEncryptedPEMBlock(block) {
		keyBytes, err = x509.DecryptPEMBlock(block, []byte(password))
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt key: %w", err)
		}
	} else {
		keyBytes = block.Bytes
	}

	// Try parsing as various key types
	if key, err := x509.ParseECPrivateKey(keyBytes); err == nil {
		return key, nil
	}

	if key, err := x509.ParsePKCS8PrivateKey(keyBytes); err == nil {
		return key, nil
	}

	return nil, fmt.Errorf("unsupported private key format")
}

// parsePublicKey parses a PEM-encoded public key
func parsePublicKey(keyData []byte) (crypto.PublicKey, error) {
	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	// Try parsing as public key
	if key, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		return key, nil
	}

	// Try parsing as certificate
	if cert, err := x509.ParseCertificate(block.Bytes); err == nil {
		return cert.PublicKey, nil
	}

	return nil, fmt.Errorf("unsupported public key format")
}

// generateKeyID generates a key identifier from a private key
func generateKeyID(privateKey crypto.PrivateKey) string {
	// Extract public key
	var publicKey crypto.PublicKey
	switch key := privateKey.(type) {
	case *ecdsa.PrivateKey:
		publicKey = &key.PublicKey
	default:
		return "unknown"
	}

	// Marshal public key
	pubBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "unknown"
	}

	// Compute digest
	digest := sha256.Sum256(pubBytes)
	return fmt.Sprintf("sha256:%x", digest[:8]) // First 8 bytes
}

// extractIdentityFromCert extracts signer identity from a certificate
func extractIdentityFromCert(certData []byte) (*SignerIdentity, error) {
	cert, err := x509.ParseCertificate(certData)
	if err != nil {
		return nil, err
	}

	identity := &SignerIdentity{}

	// Try to extract email from subject
	if len(cert.EmailAddresses) > 0 {
		identity.Email = cert.EmailAddresses[0]
		identity.Subject = cert.EmailAddresses[0]
	}

	// Try to extract issuer
	if len(cert.Issuer.Organization) > 0 {
		identity.Issuer = cert.Issuer.Organization[0]
	}

	return identity, nil
}

// GenerateKeyPair generates a new ECDSA key pair for SBOM signing
func GenerateKeyPair(privateKeyPath, publicKeyPath string) error {
	// Generate ECDSA P-256 key pair
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Marshal private key
	privKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}

	// Write private key
	privKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privKeyBytes,
	})
	if err := os.WriteFile(privateKeyPath, privKeyPEM, 0600); err != nil {
		return fmt.Errorf("failed to write private key file %s: %w", privateKeyPath, err)
	}

	// Marshal public key
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to marshal public key: %w", err)
	}

	// Write public key
	pubKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})
	if err := os.WriteFile(publicKeyPath, pubKeyPEM, 0644); err != nil {
		return fmt.Errorf("failed to write public key file %s: %w", publicKeyPath, err)
	}

	return nil
}

// IsCosignAvailable checks if cosign is installed and available
func IsCosignAvailable() bool {
	_, err := exec.LookPath("cosign")
	return err == nil
}
