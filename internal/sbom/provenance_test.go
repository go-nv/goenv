package sbom

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProvenanceGenerator(t *testing.T) {
	tmpDir := t.TempDir()
	sbomPath := filepath.Join(tmpDir, "test.sbom.json")

	// Create test SBOM
	err := os.WriteFile(sbomPath, []byte(`{"bomFormat":"CycloneDX"}`), 0644)
	require.NoError(t, err)

	opts := ProvenanceOptions{
		SBOMPath:   sbomPath,
		ProjectDir: tmpDir,
		GoVersion:  "1.23.0",
	}

	generator := NewProvenanceGenerator(opts)
	require.NotNil(t, generator)
	assert.Equal(t, sbomPath, generator.options.SBOMPath)
	assert.Equal(t, tmpDir, generator.options.ProjectDir)
	assert.Equal(t, "1.23.0", generator.options.GoVersion)
}

func TestProvenanceGenerator_Generate(t *testing.T) {
	tmpDir := t.TempDir()

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

	// Create minimal go.mod
	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := []byte(`module example.com/test

go 1.23

require (
	github.com/example/dep v1.0.0
)
`)
	err = os.WriteFile(goModPath, goModContent, 0644)
	require.NoError(t, err)

	// Create go.sum
	goSumPath := filepath.Join(tmpDir, "go.sum")
	goSumContent := []byte(`github.com/example/dep v1.0.0 h1:abc123
github.com/example/dep v1.0.0/go.mod h1:xyz789
`)
	err = os.WriteFile(goSumPath, goSumContent, 0644)
	require.NoError(t, err)

	// Change to tmpDir for go.mod detection
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Generate provenance
	generator := NewProvenanceGenerator(ProvenanceOptions{
		SBOMPath:   sbomPath,
		ProjectDir: tmpDir,
		GoVersion:  "1.23.0",
	})

	provenance, err := generator.Generate()
	require.NoError(t, err, "Provenance generation should succeed")
	require.NotNil(t, provenance)

	// Verify structure
	assert.Equal(t, "https://in-toto.io/Statement/v1", provenance.Type)
	assert.Equal(t, "https://slsa.dev/provenance/v1", provenance.PredicateType)
	assert.NotEmpty(t, provenance.Subject)
	assert.NotNil(t, provenance.Predicate)

	// Verify subject
	assert.Equal(t, 1, len(provenance.Subject))
	assert.Equal(t, "test.sbom.json", provenance.Subject[0].Name)
	assert.NotEmpty(t, provenance.Subject[0].Digest["sha256"])

	// Verify build definition
	assert.NotNil(t, provenance.Predicate.BuildDefinition)
	assert.Contains(t, provenance.Predicate.BuildDefinition.BuildType, "goenv")
	assert.NotEmpty(t, provenance.Predicate.BuildDefinition.ExternalParameters)
	assert.Equal(t, "1.23.0", provenance.Predicate.BuildDefinition.ExternalParameters["go_version"])

	// Verify run details
	assert.NotNil(t, provenance.Predicate.RunDetails)
	assert.NotNil(t, provenance.Predicate.RunDetails.Builder)
	assert.Contains(t, provenance.Predicate.RunDetails.Builder.ID, "github.com/go-nv/goenv")
}

func TestProvenanceGenerator_GenerateNoGoMod(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test SBOM only (no go.mod)
	sbomPath := filepath.Join(tmpDir, "test.sbom.json")
	err := os.WriteFile(sbomPath, []byte(`{"bomFormat":"CycloneDX"}`), 0644)
	require.NoError(t, err)

	// Change to tmpDir
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Generate provenance - should still work without go.mod
	generator := NewProvenanceGenerator(ProvenanceOptions{
		SBOMPath:   sbomPath,
		ProjectDir: tmpDir,
		GoVersion:  "1.23.0",
	})

	provenance, err := generator.Generate()
	require.NoError(t, err, "Should work without go.mod")
	assert.NotNil(t, provenance)
}

func TestProvenanceGenerator_WriteProvenance(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test SBOM
	sbomPath := filepath.Join(tmpDir, "test.sbom.json")
	err := os.WriteFile(sbomPath, []byte(`{"bomFormat":"CycloneDX"}`), 0644)
	require.NoError(t, err)

	// Create go.mod
	goModPath := filepath.Join(tmpDir, "go.mod")
	err = os.WriteFile(goModPath, []byte("module test\ngo 1.23\n"), 0644)
	require.NoError(t, err)

	// Change to tmpDir
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Generate and write provenance
	generator := NewProvenanceGenerator(ProvenanceOptions{
		SBOMPath:   sbomPath,
		ProjectDir: tmpDir,
		GoVersion:  "1.23.0",
	})

	provenance, err := generator.Generate()
	require.NoError(t, err)

	outputPath := filepath.Join(tmpDir, "provenance.json")
	err = generator.WriteProvenance(provenance, outputPath)
	require.NoError(t, err)

	// Verify file exists
	assert.FileExists(t, outputPath)

	// Verify content is valid JSON
	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	var parsed ProvenanceStatement
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Equal(t, provenance.Type, parsed.Type)
	assert.Equal(t, provenance.PredicateType, parsed.PredicateType)
}

func TestComputeGoModDigest(t *testing.T) {
	tmpDir := t.TempDir()

	goModPath := filepath.Join(tmpDir, "go.mod")
	goModContent := []byte(`module example.com/test

go 1.23

require (
	github.com/example/dep v1.0.0
)
`)
	err := os.WriteFile(goModPath, goModContent, 0644)
	require.NoError(t, err)

	digest, err := ComputeGoModDigest(tmpDir)
	require.NoError(t, err, "Should compute digest for valid go.mod")
	assert.NotEmpty(t, digest)
	assert.Len(t, digest, 64) // SHA-256 hex string

	// Verify digest is consistent
	digest2, err := ComputeGoModDigest(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, digest, digest2, "Digest should be consistent")
}

func TestComputeGoModDigest_NoFile(t *testing.T) {
	tmpDir := t.TempDir()

	digest, err := ComputeGoModDigest(tmpDir)
	assert.Error(t, err, "Should error when go.mod doesn't exist")
	assert.Empty(t, digest)
}

func TestComputeGoSumDigest(t *testing.T) {
	tmpDir := t.TempDir()

	goSumPath := filepath.Join(tmpDir, "go.sum")
	goSumContent := []byte(`github.com/example/dep v1.0.0 h1:abc123
github.com/example/dep v1.0.0/go.mod h1:xyz789
`)
	err := os.WriteFile(goSumPath, goSumContent, 0644)
	require.NoError(t, err)

	digest, err := ComputeGoSumDigest(tmpDir)
	require.NoError(t, err, "Should compute digest for valid go.sum")
	assert.NotEmpty(t, digest)
	assert.Len(t, digest, 64) // SHA-256 hex string

	// Verify digest is consistent
	digest2, err := ComputeGoSumDigest(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, digest, digest2, "Digest should be consistent")
}

func TestComputeGoSumDigest_NoFile(t *testing.T) {
	tmpDir := t.TempDir()

	digest, err := ComputeGoSumDigest(tmpDir)
	assert.Error(t, err, "Should error when go.sum doesn't exist")
	assert.Empty(t, digest)
}

func TestCreateInTotoAttestation(t *testing.T) {
	// Create a minimal provenance statement
	statement := &ProvenanceStatement{
		Type:          "https://in-toto.io/Statement/v1",
		PredicateType: "https://slsa.dev/provenance/v1",
		Subject: []ResourceDescriptor{
			{
				Name: "test.sbom.json",
				Digest: map[string]string{
					"sha256": "abc123",
				},
			},
		},
		Predicate: Provenance{
			BuildDefinition: BuildDefinition{
				BuildType: "https://example.com/build/v1",
			},
			RunDetails: RunDetails{
				Builder: Builder{
					ID: "https://example.com",
				},
			},
		},
	}

	// Create attestation without signature
	attestation, err := CreateInTotoAttestation(statement, nil)
	require.NoError(t, err)
	require.NotNil(t, attestation)

	assert.Equal(t, "application/vnd.in-toto+json", attestation.PayloadType)
	assert.NotEmpty(t, attestation.Payload)
	assert.NotNil(t, attestation.Signatures)
	assert.Len(t, attestation.Signatures, 0, "Should have no signatures when nil passed")
}

func TestCreateInTotoAttestation_WithSignature(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate keys for signature
	privateKeyPath := filepath.Join(tmpDir, "private.pem")
	publicKeyPath := filepath.Join(tmpDir, "public.pem")
	err := GenerateKeyPair(privateKeyPath, publicKeyPath)
	require.NoError(t, err)

	// Create test SBOM
	sbomPath := filepath.Join(tmpDir, "test.sbom.json")
	err = os.WriteFile(sbomPath, []byte(`{"bomFormat":"CycloneDX"}`), 0644)
	require.NoError(t, err)

	// Sign it
	signer := NewSigner(SignatureOptions{
		KeyPath: privateKeyPath,
	})
	sig, err := signer.SignSBOM(sbomPath)
	require.NoError(t, err)

	// Create provenance statement
	statement := &ProvenanceStatement{
		Type:          "https://in-toto.io/Statement/v1",
		PredicateType: "https://slsa.dev/provenance/v1",
		Subject: []ResourceDescriptor{
			{
				Name: "test.sbom.json",
				Digest: map[string]string{
					"sha256": "abc123",
				},
			},
		},
		Predicate: Provenance{
			BuildDefinition: BuildDefinition{
				BuildType: "https://example.com/build/v1",
			},
			RunDetails: RunDetails{
				Builder: Builder{
					ID: "https://example.com",
				},
			},
		},
	}

	// Create attestation with signature
	attestation, err := CreateInTotoAttestation(statement, sig)
	require.NoError(t, err)
	require.NotNil(t, attestation)

	assert.Equal(t, "application/vnd.in-toto+json", attestation.PayloadType)
	assert.NotEmpty(t, attestation.Payload)
	assert.Len(t, attestation.Signatures, 1, "Should have one signature")
	assert.Equal(t, sig.KeyID, attestation.Signatures[0].KeyID)
	assert.Equal(t, sig.Value, attestation.Signatures[0].Sig)
}

func TestWriteInTotoAttestation(t *testing.T) {
	tmpDir := t.TempDir()

	attestation := &InTotoAttestation{
		PayloadType: "application/vnd.in-toto+json",
		Payload:     "base64-encoded-payload",
		Signatures:  []InTotoSignature{},
	}

	outputPath := filepath.Join(tmpDir, "attestation.json")
	err := WriteInTotoAttestation(attestation, outputPath)
	require.NoError(t, err)

	// Verify file exists
	assert.FileExists(t, outputPath)

	// Verify content
	data, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	var parsed InTotoAttestation
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Equal(t, attestation.PayloadType, parsed.PayloadType)
	assert.Equal(t, attestation.Payload, parsed.Payload)
}

func TestValidateProvenance(t *testing.T) {
	tests := []struct {
		name      string
		statement *ProvenanceStatement
		wantErr   bool
	}{
		{
			name: "valid provenance",
			statement: &ProvenanceStatement{
				Type:          "https://in-toto.io/Statement/v1",
				PredicateType: "https://slsa.dev/provenance/v1",
				Subject: []ResourceDescriptor{
					{
						Name: "test.sbom.json",
						Digest: map[string]string{
							"sha256": "abc123",
						},
					},
				},
				Predicate: Provenance{
					BuildDefinition: BuildDefinition{
						BuildType: "https://example.com/build/v1",
					},
					RunDetails: RunDetails{
						Builder: Builder{
							ID: "https://example.com",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing type",
			statement: &ProvenanceStatement{
				PredicateType: "https://slsa.dev/provenance/v1",
				Subject:       []ResourceDescriptor{},
			},
			wantErr: true,
		},
		{
			name: "missing predicate type",
			statement: &ProvenanceStatement{
				Type:    "https://in-toto.io/Statement/v1",
				Subject: []ResourceDescriptor{},
			},
			wantErr: true,
		},
		{
			name: "empty subject",
			statement: &ProvenanceStatement{
				Type:          "https://in-toto.io/Statement/v1",
				PredicateType: "https://slsa.dev/provenance/v1",
				Subject:       []ResourceDescriptor{},
			},
			wantErr: true,
		},
		{
			name: "nil predicate",
			statement: &ProvenanceStatement{
				Type:          "https://in-toto.io/Statement/v1",
				PredicateType: "https://slsa.dev/provenance/v1",
				Subject: []ResourceDescriptor{
					{Name: "test", Digest: map[string]string{"sha256": "abc"}},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProvenance(tt.statement)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProvenanceStatement_JSON(t *testing.T) {
	statement := &ProvenanceStatement{
		Type:          "https://in-toto.io/Statement/v1",
		PredicateType: "https://slsa.dev/provenance/v1",
		Subject: []ResourceDescriptor{
			{
				Name: "test.sbom.json",
				Digest: map[string]string{
					"sha256": "abc123",
				},
			},
		},
		Predicate: Provenance{
			BuildDefinition: BuildDefinition{
				BuildType: "https://example.com/build/v1",
				ExternalParameters: map[string]interface{}{
					"go_version": "1.23.0",
				},
				InternalParameters: map[string]interface{}{
					"cgo_enabled": false,
				},
			},
			RunDetails: RunDetails{
				Builder: Builder{
					ID: "https://example.com",
					Version: map[string]string{
						"goenv": "3.3.0",
					},
				},
			},
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(statement)
	require.NoError(t, err)
	assert.Contains(t, string(data), "in-toto.io/Statement")
	assert.Contains(t, string(data), "slsa.dev/provenance")

	// Unmarshal from JSON
	var parsed ProvenanceStatement
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Equal(t, statement.Type, parsed.Type)
	assert.Equal(t, statement.PredicateType, parsed.PredicateType)
	assert.Equal(t, len(statement.Subject), len(parsed.Subject))
}

func TestProvenanceOptions_Validation(t *testing.T) {
	tests := []struct {
		name    string
		opts    ProvenanceOptions
		wantErr bool
	}{
		{
			name: "valid options",
			opts: ProvenanceOptions{
				SBOMPath:   "/path/to/sbom.json",
				ProjectDir: "/path/to/project",
				GoVersion:  "1.23.0",
			},
			wantErr: false,
		},
		{
			name: "minimal options",
			opts: ProvenanceOptions{
				SBOMPath: "/path/to/sbom.json",
			},
			wantErr: false,
		},
		{
			name:    "empty options",
			opts:    ProvenanceOptions{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantErr {
				assert.Empty(t, tt.opts.SBOMPath)
			} else {
				assert.NotEmpty(t, tt.opts.SBOMPath)
			}
		})
	}
}
