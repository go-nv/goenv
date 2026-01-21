package sbom

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// SLSA Provenance v1.0 specification structures
// https://slsa.dev/spec/v1.0/provenance

// ProvenanceStatement represents a complete SLSA provenance attestation
type ProvenanceStatement struct {
	Type          string               `json:"_type"`
	PredicateType string               `json:"predicateType"`
	Subject       []ResourceDescriptor `json:"subject"`
	Predicate     Provenance           `json:"predicate"`
}

// ResourceDescriptor describes a software artifact
type ResourceDescriptor struct {
	Name   string            `json:"name,omitempty"`
	URI    string            `json:"uri,omitempty"`
	Digest map[string]string `json:"digest"`
}

// Provenance is the SLSA provenance predicate
type Provenance struct {
	BuildDefinition BuildDefinition `json:"buildDefinition"`
	RunDetails      RunDetails      `json:"runDetails"`
}

// BuildDefinition describes how the build was performed
type BuildDefinition struct {
	BuildType            string                 `json:"buildType"`
	ExternalParameters   map[string]interface{} `json:"externalParameters"`
	InternalParameters   map[string]interface{} `json:"internalParameters,omitempty"`
	ResolvedDependencies []ResourceDescriptor   `json:"resolvedDependencies,omitempty"`
}

// RunDetails provides runtime information about the build
type RunDetails struct {
	Builder    Builder              `json:"builder"`
	Metadata   BuildMetadata        `json:"metadata"`
	Byproducts []ResourceDescriptor `json:"byproducts,omitempty"`
}

// Builder identifies who/what performed the build
type Builder struct {
	ID      string            `json:"id"`
	Version map[string]string `json:"version,omitempty"`
}

// BuildMetadata contains metadata about the build execution
type BuildMetadata struct {
	InvocationID string    `json:"invocationId,omitempty"`
	StartedOn    time.Time `json:"startedOn,omitempty"`
	FinishedOn   time.Time `json:"finishedOn,omitempty"`
	Reproducible bool      `json:"reproducible,omitempty"`
}

// ProvenanceOptions configures SLSA provenance generation
type ProvenanceOptions struct {
	// SBOMPath is the path to the SBOM file
	SBOMPath string
	// GoVersion is the Go version used
	GoVersion string
	// GoModDigest is the SHA256 of go.mod
	GoModDigest string
	// GoSumDigest is the SHA256 of go.sum
	GoSumDigest string
	// BuildTags are the build tags used
	BuildTags []string
	// CGOEnabled indicates if CGO was enabled
	CGOEnabled bool
	// GOOS is the target OS
	GOOS string
	// GOARCH is the target architecture
	GOARCH string
	// LDFlags are the linker flags used
	LDFlags string
	// Vendored indicates if dependencies are vendored
	Vendored bool
	// ModuleProxy is the module proxy URL
	ModuleProxy string
	// SBOMTool is the tool used to generate the SBOM
	SBOMTool string
	// SBOMToolVersion is the version of the SBOM tool
	SBOMToolVersion string
	// ProjectDir is the project directory
	ProjectDir string
	// InvocationID is a unique identifier for this build
	InvocationID string
}

// ProvenanceGenerator generates SLSA provenance attestations for SBOMs
type ProvenanceGenerator struct {
	options ProvenanceOptions
}

// NewProvenanceGenerator creates a new provenance generator
func NewProvenanceGenerator(opts ProvenanceOptions) *ProvenanceGenerator {
	return &ProvenanceGenerator{
		options: opts,
	}
}

// Generate creates a SLSA provenance attestation for an SBOM
func (g *ProvenanceGenerator) Generate() (*ProvenanceStatement, error) {
	// Compute SBOM digest
	sbomDigest, err := computeFileDigest(g.options.SBOMPath)
	if err != nil {
		return nil, fmt.Errorf("failed to compute SBOM digest: %w", err)
	}

	// Create subject (the SBOM itself)
	subject := []ResourceDescriptor{
		{
			Name: filepath.Base(g.options.SBOMPath),
			URI:  fmt.Sprintf("file://%s", g.options.SBOMPath),
			Digest: map[string]string{
				"sha256": sbomDigest,
			},
		},
	}

	// Build external parameters (user-controlled inputs)
	externalParams := map[string]interface{}{
		"go_version": g.options.GoVersion,
		"goos":       g.options.GOOS,
		"goarch":     g.options.GOARCH,
	}

	if len(g.options.BuildTags) > 0 {
		externalParams["build_tags"] = g.options.BuildTags
	}

	if g.options.LDFlags != "" {
		externalParams["ldflags"] = g.options.LDFlags
	}

	// Build internal parameters (builder-controlled)
	internalParams := map[string]interface{}{
		"cgo_enabled":  g.options.CGOEnabled,
		"vendored":     g.options.Vendored,
		"module_proxy": g.options.ModuleProxy,
		"sbom_tool":    g.options.SBOMTool,
		"tool_version": g.options.SBOMToolVersion,
	}

	// Add resolved dependencies (go.mod/go.sum)
	resolvedDeps := []ResourceDescriptor{}
	if g.options.GoModDigest != "" {
		resolvedDeps = append(resolvedDeps, ResourceDescriptor{
			Name: "go.mod",
			URI:  fmt.Sprintf("file://%s", filepath.Join(g.options.ProjectDir, "go.mod")),
			Digest: map[string]string{
				"sha256": g.options.GoModDigest,
			},
		})
	}

	if g.options.GoSumDigest != "" {
		resolvedDeps = append(resolvedDeps, ResourceDescriptor{
			Name: "go.sum",
			URI:  fmt.Sprintf("file://%s", filepath.Join(g.options.ProjectDir, "go.sum")),
			Digest: map[string]string{
				"sha256": g.options.GoSumDigest,
			},
		})
	}

	// Get goenv version
	goenvVersion := getGoenvVersion()

	// Create the provenance statement
	statement := &ProvenanceStatement{
		Type:          "https://in-toto.io/Statement/v1",
		PredicateType: "https://slsa.dev/provenance/v1",
		Subject:       subject,
		Predicate: Provenance{
			BuildDefinition: BuildDefinition{
				BuildType:            "https://github.com/go-nv/goenv/SBOMBuild/v1",
				ExternalParameters:   externalParams,
				InternalParameters:   internalParams,
				ResolvedDependencies: resolvedDeps,
			},
			RunDetails: RunDetails{
				Builder: Builder{
					ID: "https://github.com/go-nv/goenv",
					Version: map[string]string{
						"goenv": goenvVersion,
					},
				},
				Metadata: BuildMetadata{
					InvocationID: g.options.InvocationID,
					StartedOn:    time.Now().UTC(),
					FinishedOn:   time.Now().UTC(),
					Reproducible: true, // goenv aims for reproducible SBOMs
				},
			},
		},
	}

	return statement, nil
}

// WriteProvenance writes a provenance statement to a file
func (g *ProvenanceGenerator) WriteProvenance(statement *ProvenanceStatement, outputPath string) error {
	data, err := json.MarshalIndent(statement, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal provenance: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write provenance file %s: %w", outputPath, err)
	}

	return nil
}

// GenerateInTotoAttestation creates an in-toto attestation bundle
// This combines the provenance with additional metadata
type InTotoAttestation struct {
	PayloadType string            `json:"payloadType"`
	Payload     string            `json:"payload"` // Base64-encoded ProvenanceStatement
	Signatures  []InTotoSignature `json:"signatures"`
}

// InTotoSignature represents a signature in the in-toto format
type InTotoSignature struct {
	KeyID string `json:"keyid"`
	Sig   string `json:"sig"`
}

// CreateInTotoAttestation creates an in-toto attestation from a provenance statement
func CreateInTotoAttestation(statement *ProvenanceStatement, signature *Signature) (*InTotoAttestation, error) {
	// Serialize the statement
	payload, err := json.Marshal(statement)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal statement: %w", err)
	}

	// Base64 encode the payload (in-toto spec requirement)
	encodedPayload := base64.StdEncoding.EncodeToString(payload)

	attestation := &InTotoAttestation{
		PayloadType: "application/vnd.in-toto+json",
		Payload:     encodedPayload,
		Signatures:  []InTotoSignature{},
	}

	// Add signature if provided
	if signature != nil {
		attestation.Signatures = append(attestation.Signatures, InTotoSignature{
			KeyID: signature.KeyID,
			Sig:   signature.Value,
		})
	}

	return attestation, nil
}

// WriteInTotoAttestation writes an in-toto attestation to a file
func WriteInTotoAttestation(attestation *InTotoAttestation, outputPath string) error {
	data, err := json.MarshalIndent(attestation, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal attestation: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write attestation file %s: %w", outputPath, err)
	}

	return nil
}

// Helper functions

// computeFileDigest computes the SHA256 digest of a file
func computeFileDigest(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// getGoenvVersion returns the current goenv version
func getGoenvVersion() string {
	// Try to get version from git
	cmd := exec.Command("git", "describe", "--tags", "--always")
	if output, err := cmd.Output(); err == nil {
		return string(output)
	}

	// Fallback to runtime info
	return fmt.Sprintf("goenv-%s-%s", runtime.GOOS, runtime.GOARCH)
}

// ComputeGoModDigest computes the SHA256 digest of go.mod
func ComputeGoModDigest(projectDir string) (string, error) {
	goModPath := filepath.Join(projectDir, "go.mod")
	return computeFileDigest(goModPath)
}

// ComputeGoSumDigest computes the SHA256 digest of go.sum
func ComputeGoSumDigest(projectDir string) (string, error) {
	goSumPath := filepath.Join(projectDir, "go.sum")
	return computeFileDigest(goSumPath)
}

// ValidateProvenance validates a SLSA provenance statement
func ValidateProvenance(statement *ProvenanceStatement) error {
	if statement.Type != "https://in-toto.io/Statement/v1" {
		return fmt.Errorf("invalid statement type: %s", statement.Type)
	}

	if statement.PredicateType != "https://slsa.dev/provenance/v1" {
		return fmt.Errorf("invalid predicate type: %s", statement.PredicateType)
	}

	if len(statement.Subject) == 0 {
		return fmt.Errorf("provenance must have at least one subject")
	}

	if statement.Predicate.BuildDefinition.BuildType == "" {
		return fmt.Errorf("build type is required")
	}

	if statement.Predicate.RunDetails.Builder.ID == "" {
		return fmt.Errorf("builder ID is required")
	}

	return nil
}
