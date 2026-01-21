package compliance

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/cmdutil"
	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/platform"
	"github.com/go-nv/goenv/internal/resolver"
	"github.com/go-nv/goenv/internal/sbom"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var sbomCmd = &cobra.Command{
	Use:     "sbom",
	Short:   "Generate Software Bill of Materials for projects",
	GroupID: string(cmdpkg.GroupTools),
	Long: `Generate SBOMs using industry-standard tools (cyclonedx-gomod, syft) with goenv-managed toolchains.

CURRENT STATE (v3.0): This is a convenience wrapper that runs SBOM tools with the 
correct Go version and environment. It does NOT generate SBOMs itself or add features
beyond what the underlying tools provide.

ROADMAP: Future versions will add validation, policy enforcement, signing, vulnerability
scanning, and compliance reporting. See docs/roadmap/SBOM_ROADMAP.md for details.

ALTERNATIVE: Advanced users can run SBOM tools directly:
  goenv exec cyclonedx-gomod -json -output sbom.json

Examples:
  # Generate CycloneDX SBOM for current project
  goenv sbom project --tool=cyclonedx-gomod --format=cyclonedx-json

  # Generate SPDX SBOM with syft
  goenv sbom project --tool=syft --format=spdx-json --output=sbom.spdx.json

  # Generate SBOM for container image
  goenv sbom project --tool=syft --image=ghcr.io/myapp:v1.0.0

Before using, install the required tool:
  goenv tools install cyclonedx-gomod@v1.6.0
  goenv tools install syft@v1.0.0`,
}

var sbomProjectCmd = &cobra.Command{
	Use:   "project",
	Short: "Generate SBOM for a Go project",
	Long: `Generate a Software Bill of Materials for a Go project using cyclonedx-gomod or syft.

WHAT THIS DOES:
- Runs SBOM tools with the correct Go version and environment
- Provides unified CLI across different SBOM tools
- Ensures reproducibility in CI/CD pipelines

WHAT THIS DOES NOT DO (yet):
- Validate SBOM format or completeness (planned: v3.1)
- Sign or attest SBOMs (planned: v3.2)
- Scan for vulnerabilities (planned: v3.5)
- Enforce policies (planned: v3.1)

See docs/roadmap/SBOM_ROADMAP.md for planned features.

Supported tools:
- cyclonedx-gomod: Native Go module SBOM generator (CycloneDX format)
- syft: Multi-language SBOM generator (supports containers)`,
	RunE: runSBOMProject,
}

var (
	sbomTool          string
	sbomFormat        string
	sbomOutput        string
	sbomDir           string
	sbomImage         string
	sbomModulesOnly   bool
	sbomOffline       bool
	sbomToolArgs      string
	sbomDeterministic bool
	sbomEmbedDigests  bool
	sbomEnhance       bool
)

func init() {
	sbomProjectCmd.Flags().StringVar(&sbomTool, "tool", "cyclonedx-gomod", "SBOM tool to use (cyclonedx-gomod, syft)")
	sbomProjectCmd.Flags().StringVar(&sbomFormat, "format", "cyclonedx-json", "Output format (cyclonedx-json, spdx-json)")
	sbomProjectCmd.Flags().StringVarP(&sbomOutput, "output", "o", "sbom.json", "Output file path")
	sbomProjectCmd.Flags().StringVar(&sbomDir, "dir", ".", "Project directory to scan")
	sbomProjectCmd.Flags().StringVar(&sbomImage, "image", "", "Container image to scan (syft only)")
	sbomProjectCmd.Flags().BoolVar(&sbomModulesOnly, "modules-only", false, "Only scan Go modules (cyclonedx-gomod)")
	sbomProjectCmd.Flags().BoolVar(&sbomOffline, "offline", false, "Offline mode - avoid network access")
	sbomProjectCmd.Flags().StringVar(&sbomToolArgs, "tool-args", "", "Additional arguments to pass to the tool")
	sbomProjectCmd.Flags().BoolVar(&sbomEnhance, "enhance", true, "Add Go-aware metadata to SBOM (default true)")
	sbomProjectCmd.Flags().BoolVar(&sbomDeterministic, "deterministic", false, "Generate deterministic/reproducible SBOM")
	sbomProjectCmd.Flags().BoolVar(&sbomEmbedDigests, "embed-digests", false, "Embed go.mod/go.sum digests for reproducibility")

	sbomCmd.AddCommand(sbomProjectCmd)
	sbomCmd.AddCommand(sbomHashCmd)
	sbomCmd.AddCommand(sbomVerifyCmd)
	sbomCmd.AddCommand(sbomValidateCmd)
	sbomCmd.AddCommand(sbomSignCmd)
	sbomCmd.AddCommand(sbomVerifySignatureCmd)
	sbomCmd.AddCommand(sbomAttestCmd)
	sbomCmd.AddCommand(sbomScanCmd)
	cmdpkg.RootCmd.AddCommand(sbomCmd)
}

var sbomHashCmd = &cobra.Command{
	Use:   "hash <sbom-file>",
	Short: "Compute digest of an SBOM file",
	Long: `Compute a cryptographic hash of an SBOM file for reproducibility verification.

This command normalizes the SBOM (sorting components, normalizing whitespace) before
computing the digest to ensure consistent hashing across different generation runs.

The digest can be used to verify that two SBOMs have identical semantic content,
even if they were generated at different times or with different metadata timestamps.

Examples:
  # Compute hash of an SBOM
  goenv sbom hash sbom.json

  # Compute hash with specific algorithm
  goenv sbom hash sbom.json --algorithm=sha512`,
	Args: cobra.ExactArgs(1),
	RunE: runSBOMHash,
}

var sbomVerifyCmd = &cobra.Command{
	Use:   "verify-reproducible <sbom1> <sbom2>",
	Short: "Verify two SBOMs have identical reproducible content",
	Long: `Compare two SBOM files to verify they have identical semantic content.

This command normalizes both SBOMs (removing timestamps, sorting components) and
compares their content digests to verify reproducibility. Exit code 0 indicates
the SBOMs are identical, non-zero indicates differences.

This is useful for:
- Verifying deterministic SBOM generation in CI/CD
- Detecting unexpected changes in dependencies
- Validating reproducible builds

Examples:
  # Compare two SBOMs
  goenv sbom verify-reproducible sbom1.json sbom2.json

  # Verify with detailed diff output
  goenv sbom verify-reproducible sbom1.json sbom2.json --diff`,
	Args: cobra.ExactArgs(2),
	RunE: runSBOMVerify,
}

var sbomValidateCmd = &cobra.Command{
	Use:   "validate <sbom-file>",
	Short: "Validate SBOM against policy rules",
	Long: `Validate an SBOM file against defined policy rules.

Policy files are YAML documents that define validation rules for:
- Supply chain security (replace directives, vendoring)
- Security requirements (CGO status, retracted versions)
- Completeness checks (required components, metadata)
- License compliance (allowed/blocked licenses)

Examples:
  # Validate with default policy
  goenv sbom validate sbom.json --policy=.goenv-policy.yaml

  # Validate and fail on warnings
  goenv sbom validate sbom.json --policy=policy.yaml --fail-on-warning

  # Validate with verbose output
  goenv sbom validate sbom.json --policy=policy.yaml --verbose`,
	Args: cobra.ExactArgs(1),
	RunE: runSBOMValidate,
}

var sbomSignCmd = &cobra.Command{
	Use:   "sign <sbom-file>",
	Short: "Sign an SBOM with cryptographic signature",
	Long: `Sign an SBOM file to create a cryptographic signature for integrity verification.

Supports two signing methods:
1. Key-based signing: Uses a private key file (ECDSA recommended)
2. Keyless signing: Uses Sigstore/Fulcio for identity-based signing

Key-based signing is suitable for CI/CD pipelines with managed keys.
Keyless signing is ideal for developer workflows and OIDC-enabled environments.

Examples:
  # Sign with private key
  goenv sbom sign sbom.json --key=private.pem --output=sbom.json.sig

  # Sign with keyless (Sigstore)
  goenv sbom sign sbom.json --keyless --output=sbom.json.sig

  # Generate a new key pair first
  goenv sbom generate-keys --private=private.pem --public=public.pem`,
	Args: cobra.ExactArgs(1),
	RunE: runSBOMSign,
}

var sbomVerifySignatureCmd = &cobra.Command{
	Use:   "verify-signature <sbom-file>",
	Short: "Verify SBOM cryptographic signature",
	Long: `Verify the cryptographic signature of an SBOM file.

Verification ensures:
- The SBOM has not been tampered with
- The SBOM was signed by a trusted key/identity
- The signature is valid and not expired

Examples:
  # Verify with public key
  goenv sbom verify-signature sbom.json --signature=sbom.json.sig --key=public.pem

  # Verify keyless signature
  goenv sbom verify-signature sbom.json --signature=sbom.json.sig --certificate=sbom.cert

  # Verify using cosign
  goenv sbom verify-signature sbom.json --signature=sbom.json.sig --use-cosign`,
	Args: cobra.ExactArgs(1),
	RunE: runSBOMVerifySignature,
}

var sbomAttestCmd = &cobra.Command{
	Use:   "attest <sbom-file>",
	Short: "Generate SLSA provenance attestation for SBOM",
	Long: `Generate a SLSA (Supply-chain Levels for Software Artifacts) provenance 
attestation for an SBOM file. This creates a verifiable record of how the SBOM 
was generated, including Go version, build context, and dependencies.

The provenance can be used for:
- SLSA Level 3 compliance
- Supply chain security verification
- Reproducible build validation
- Audit trail documentation

Examples:
  # Generate provenance attestation
  goenv sbom attest sbom.json --output=sbom.provenance.json

  # Generate and sign provenance
  goenv sbom attest sbom.json --output=sbom.provenance.json --sign --key=private.pem

  # Generate in-toto attestation bundle
  goenv sbom attest sbom.json --output=sbom.att.json --in-toto --sign --key=private.pem`,
	Args: cobra.ExactArgs(1),
	RunE: runSBOMAttest,
}

var (
	// Hash and verify flags
	hashAlgorithm   string
	verifyDiff      bool
	policyFile      string
	failOnWarning   bool
	verboseValidate bool

	// Signing flags
	signKeyPath      string
	signKeyPassword  string
	signKeyless      bool
	signOIDCIssuer   string
	signOIDCClientID string
	signOutput       string

	// Verification flags
	verifySignaturePath string
	verifyPublicKey     string
	verifyCertificate   string
	verifyUseCosign     bool

	// Attestation flags
	attestOutput       string
	attestSign         bool
	attestKeyPath      string
	attestInToto       bool
	attestInvocationID string
)

func init() {
	// Hash command flags
	sbomHashCmd.Flags().StringVar(&hashAlgorithm, "algorithm", "sha256", "Hash algorithm (sha256, sha512)")

	// Verify reproducibility flags
	sbomVerifyCmd.Flags().BoolVar(&verifyDiff, "diff", false, "Show detailed differences if SBOMs don't match")

	// Validate command flags
	sbomValidateCmd.Flags().StringVarP(&policyFile, "policy", "p", ".goenv-policy.yaml", "Path to policy configuration file")
	sbomValidateCmd.Flags().BoolVar(&failOnWarning, "fail-on-warning", false, "Treat warnings as failures")
	sbomValidateCmd.Flags().BoolVar(&verboseValidate, "verbose", false, "Show detailed validation output")

	// Sign command flags
	sbomSignCmd.Flags().StringVar(&signKeyPath, "key", "", "Path to private key file for signing")
	sbomSignCmd.Flags().StringVar(&signKeyPassword, "key-password", "", "Password for encrypted private key")
	sbomSignCmd.Flags().BoolVar(&signKeyless, "keyless", false, "Use keyless signing via Sigstore/Fulcio")
	sbomSignCmd.Flags().StringVar(&signOIDCIssuer, "oidc-issuer", "", "OIDC issuer for keyless signing")
	sbomSignCmd.Flags().StringVar(&signOIDCClientID, "oidc-client-id", "", "OIDC client ID for keyless signing")
	sbomSignCmd.Flags().StringVarP(&signOutput, "output", "o", "", "Output path for signature (default: <sbom-file>.sig)")

	// Verify signature command flags
	sbomVerifySignatureCmd.Flags().StringVarP(&verifySignaturePath, "signature", "s", "", "Path to signature file (required)")
	sbomVerifySignatureCmd.Flags().StringVar(&verifyPublicKey, "key", "", "Path to public key file")
	sbomVerifySignatureCmd.Flags().StringVar(&verifyCertificate, "certificate", "", "Path to certificate file for keyless verification")
	sbomVerifySignatureCmd.Flags().BoolVar(&verifyUseCosign, "use-cosign", false, "Use cosign CLI for verification")
	sbomVerifySignatureCmd.MarkFlagRequired("signature")

	// Attest command flags
	sbomAttestCmd.Flags().StringVarP(&attestOutput, "output", "o", "", "Output path for attestation (default: <sbom-file>.provenance.json)")
	sbomAttestCmd.Flags().BoolVar(&attestSign, "sign", false, "Sign the attestation after generation")
	sbomAttestCmd.Flags().StringVar(&attestKeyPath, "key", "", "Path to private key for signing attestation")
	sbomAttestCmd.Flags().BoolVar(&attestInToto, "in-toto", false, "Generate in-toto attestation bundle format")
	sbomAttestCmd.Flags().StringVar(&attestInvocationID, "invocation-id", "", "Unique invocation ID for this build")
}

func runSBOMHash(cmd *cobra.Command, args []string) error {
	sbomPath := args[0]

	// Verify file exists
	if !utils.FileExists(sbomPath) {
		return fmt.Errorf("SBOM file not found: %s", sbomPath)
	}

	// Compute hash using the enhancer's deterministic logic
	ctx := cmdutil.GetContexts(cmd)
	cfg := ctx.Config

	hash, err := sbom.ComputeSBOMDigest(sbomPath, hashAlgorithm)
	if err != nil {
		return errors.FailedTo("compute SBOM digest", err)
	}

	// Output hash in format: <algorithm>:<hex-digest>
	if cfg.Debug {
		fmt.Fprintf(cmd.OutOrStdout(), "%s:%s  %s\n", hashAlgorithm, hash, sbomPath)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "%s:%s\n", hashAlgorithm, hash)
	}

	return nil
}

func runSBOMVerify(cmd *cobra.Command, args []string) error {
	sbom1Path := args[0]
	sbom2Path := args[1]

	// Verify both files exist
	if !utils.FileExists(sbom1Path) {
		return fmt.Errorf("SBOM file not found: %s", sbom1Path)
	}
	if !utils.FileExists(sbom2Path) {
		return fmt.Errorf("SBOM file not found: %s", sbom2Path)
	}

	// Compare SBOMs
	ctx := cmdutil.GetContexts(cmd)

	cfg := ctx.Config
	match, diff, err := sbom.VerifyReproducible(sbom1Path, sbom2Path)
	if err != nil {
		return errors.FailedTo("verify reproducibility", err)
	}

	if match {
		fmt.Fprintf(cmd.OutOrStdout(), "✓ SBOMs are reproducibly identical\n")
		if cfg.Debug {
			fmt.Fprintf(cmd.ErrOrStderr(), "Debug: %s == %s\n", sbom1Path, sbom2Path)
		}
		return nil
	}

	// SBOMs don't match
	fmt.Fprintf(cmd.ErrOrStderr(), "✗ SBOMs differ\n")
	if verifyDiff {
		fmt.Fprintf(cmd.OutOrStdout(), "\nDifferences:\n%s\n", diff)
	}

	return fmt.Errorf("SBOMs are not reproducibly identical")
}

func runSBOMValidate(cmd *cobra.Command, args []string) error {
	sbomPath := args[0]

	// Verify SBOM file exists
	if !utils.FileExists(sbomPath) {
		return fmt.Errorf("SBOM file not found: %s", sbomPath)
	}

	// Verify policy file exists
	if !utils.FileExists(policyFile) {
		return fmt.Errorf("policy file not found: %s (use --policy to specify)", policyFile)
	}

	ctx := cmdutil.GetContexts(cmd)
	cfg := ctx.Config

	// Load policy engine
	engine, err := sbom.NewPolicyEngine(policyFile)
	if err != nil {
		return errors.FailedTo("load policy", err)
	}

	if cfg.Debug || verboseValidate {
		fmt.Fprintf(cmd.ErrOrStderr(), "goenv: Validating %s against policy %s\n", sbomPath, policyFile)
	}

	// Run validation
	result, err := engine.Validate(sbomPath)
	if err != nil {
		return errors.FailedTo("validate SBOM", err)
	}

	// Output results
	if verboseValidate {
		fmt.Fprint(cmd.OutOrStdout(), result.Summary)

		// Show detailed violations
		if len(result.Violations) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "\nViolations:\n")
			for i, v := range result.Violations {
				fmt.Fprintf(cmd.OutOrStdout(), "\n%d. %s\n", i+1, v.Rule)
				fmt.Fprintf(cmd.OutOrStdout(), "   Severity: %s\n", v.Severity)
				fmt.Fprintf(cmd.OutOrStdout(), "   Component: %s\n", v.Component)
				fmt.Fprintf(cmd.OutOrStdout(), "   Message: %s\n", v.Message)
				if v.Remediation != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "   Remediation: %s\n", v.Remediation)
				}
			}
		}

		if len(result.Warnings) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "\nWarnings:\n")
			for i, w := range result.Warnings {
				fmt.Fprintf(cmd.OutOrStdout(), "\n%d. %s\n", i+1, w.Rule)
				fmt.Fprintf(cmd.OutOrStdout(), "   Severity: %s\n", w.Severity)
				fmt.Fprintf(cmd.OutOrStdout(), "   Component: %s\n", w.Component)
				fmt.Fprintf(cmd.OutOrStdout(), "   Message: %s\n", w.Message)
				if w.Remediation != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "   Remediation: %s\n", w.Remediation)
				}
			}
		}
	} else {
		// Concise output
		if result.Passed {
			fmt.Fprintf(cmd.OutOrStdout(), "✓ SBOM validation passed\n")
		} else {
			fmt.Fprintf(cmd.ErrOrStderr(), "✗ SBOM validation failed\n")
			fmt.Fprintf(cmd.ErrOrStderr(), "  %d violations, %d warnings\n",
				len(result.Violations), len(result.Warnings))
			fmt.Fprintf(cmd.ErrOrStderr(), "  Run with --verbose for details\n")
		}
	}

	// Return error if validation failed
	if !result.Passed {
		if failOnWarning && len(result.Warnings) > 0 {
			return fmt.Errorf("validation failed with %d violations and %d warnings",
				len(result.Violations), len(result.Warnings))
		}
		return fmt.Errorf("validation failed with %d violations", len(result.Violations))
	}

	return nil
}

func runSBOMSign(cmd *cobra.Command, args []string) error {
	sbomPath := args[0]

	// Verify SBOM file exists
	if !utils.FileExists(sbomPath) {
		return fmt.Errorf("SBOM file not found: %s", sbomPath)
	}

	// Determine output path
	outputPath := signOutput
	if outputPath == "" {
		outputPath = sbomPath + ".sig"
	}

	ctx := cmdutil.GetContexts(cmd)
	cfg := ctx.Config

	// Validate signing options
	if !signKeyless && signKeyPath == "" {
		return fmt.Errorf("either --key or --keyless must be specified")
	}

	if signKeyless && signKeyPath != "" {
		return fmt.Errorf("cannot specify both --key and --keyless")
	}

	// Check for cosign if using keyless
	if signKeyless && !sbom.IsCosignAvailable() {
		return fmt.Errorf("keyless signing requires cosign to be installed\n" +
			"Install it from: https://docs.sigstore.dev/cosign/installation/")
	}

	if cfg.Debug {
		if signKeyless {
			fmt.Fprintf(cmd.ErrOrStderr(), "goenv: Signing %s with keyless signing (Sigstore)\n", sbomPath)
		} else {
			fmt.Fprintf(cmd.ErrOrStderr(), "goenv: Signing %s with key %s\n", sbomPath, signKeyPath)
		}
	}

	// Create signer
	signer := sbom.NewSigner(sbom.SignatureOptions{
		KeyPath:      signKeyPath,
		KeyPassword:  signKeyPassword,
		Keyless:      signKeyless,
		OIDCIssuer:   signOIDCIssuer,
		OIDCClientID: signOIDCClientID,
		OutputPath:   outputPath,
	})

	// Sign the SBOM
	signature, err := signer.SignSBOM(sbomPath)
	if err != nil {
		return errors.FailedTo("sign SBOM", err)
	}

	// Write signature
	if err := signer.WriteSignature(signature, outputPath); err != nil {
		return errors.FailedTo("write signature", err)
	}

	// Success output
	fmt.Fprintf(cmd.OutOrStdout(), "✓ SBOM signed successfully\n")
	fmt.Fprintf(cmd.OutOrStdout(), "  SBOM: %s\n", sbomPath)
	fmt.Fprintf(cmd.OutOrStdout(), "  Signature: %s\n", outputPath)
	fmt.Fprintf(cmd.OutOrStdout(), "  Algorithm: %s\n", signature.Algorithm)

	if signature.SignedBy != nil && signature.SignedBy.Email != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "  Signed by: %s\n", signature.SignedBy.Email)
	} else if signature.KeyID != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "  Key ID: %s\n", signature.KeyID)
	}

	if cfg.Debug {
		fmt.Fprintf(cmd.ErrOrStderr(), "Debug: Timestamp: %s\n", signature.Timestamp.Format(time.RFC3339))
	}

	return nil
}

func runSBOMVerifySignature(cmd *cobra.Command, args []string) error {
	sbomPath := args[0]

	// Verify files exist
	if !utils.FileExists(sbomPath) {
		return fmt.Errorf("SBOM file not found: %s", sbomPath)
	}

	if !utils.FileExists(verifySignaturePath) {
		return fmt.Errorf("signature file not found: %s", verifySignaturePath)
	}

	ctx := cmdutil.GetContexts(cmd)
	cfg := ctx.Config

	// Validate verification options
	if !verifyUseCosign && verifyPublicKey == "" && verifyCertificate == "" {
		return fmt.Errorf("either --key, --certificate, or --use-cosign must be specified")
	}

	if verifyUseCosign && !sbom.IsCosignAvailable() {
		return fmt.Errorf("--use-cosign requires cosign to be installed\n" +
			"Install it from: https://docs.sigstore.dev/cosign/installation/")
	}

	if cfg.Debug {
		fmt.Fprintf(cmd.ErrOrStderr(), "goenv: Verifying signature for %s\n", sbomPath)
		fmt.Fprintf(cmd.ErrOrStderr(), "goenv: Signature file: %s\n", verifySignaturePath)
	}

	// Create verifier
	verifier := sbom.NewVerifier(sbom.VerificationOptions{
		SBOMPath:      sbomPath,
		SignaturePath: verifySignaturePath,
		PublicKeyPath: verifyPublicKey,
		CertPath:      verifyCertificate,
		UseCosign:     verifyUseCosign,
	})

	// Verify signature
	valid, err := verifier.VerifySignature()
	if err != nil {
		return errors.FailedTo("verify signature", err)
	}

	if valid {
		fmt.Fprintf(cmd.OutOrStdout(), "✓ Signature verification passed\n")
		fmt.Fprintf(cmd.OutOrStdout(), "  SBOM: %s\n", sbomPath)
		fmt.Fprintf(cmd.OutOrStdout(), "  Signature: %s\n", verifySignaturePath)

		// Try to read signature metadata
		if sigData, err := os.ReadFile(verifySignaturePath); err == nil {
			var sig sbom.Signature
			if err := json.Unmarshal(sigData, &sig); err == nil {
				fmt.Fprintf(cmd.OutOrStdout(), "  Algorithm: %s\n", sig.Algorithm)
				if sig.SignedBy != nil && sig.SignedBy.Email != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "  Signed by: %s\n", sig.SignedBy.Email)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "  Signed at: %s\n", sig.Timestamp.Format(time.RFC3339))
			}
		}

		return nil
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "✗ Signature verification failed\n")
	return fmt.Errorf("signature is invalid or does not match SBOM")
}

func runSBOMAttest(cmd *cobra.Command, args []string) error {
	sbomPath := args[0]

	// Verify SBOM file exists
	if !utils.FileExists(sbomPath) {
		return fmt.Errorf("SBOM file not found: %s", sbomPath)
	}

	ctx := cmdutil.GetContexts(cmd)
	cfg, mgr := ctx.Config, ctx.Manager

	// Determine output path
	outputPath := attestOutput
	if outputPath == "" {
		if attestInToto {
			outputPath = sbomPath + ".att.json"
		} else {
			outputPath = sbomPath + ".provenance.json"
		}
	}

	// Get current Go version
	goVersion, _, err := mgr.GetCurrentVersion()
	if err != nil {
		goVersion = "unknown"
	}

	// Get project directory
	projectDir := "."
	if sbomDir != "" {
		projectDir = sbomDir
	}

	// Compute go.mod and go.sum digests
	goModDigest, _ := sbom.ComputeGoModDigest(projectDir)
	goSumDigest, _ := sbom.ComputeGoSumDigest(projectDir)

	// Generate invocation ID if not provided
	invocationID := attestInvocationID
	if invocationID == "" {
		invocationID = fmt.Sprintf("%s-%d", goVersion, time.Now().Unix())
	}

	if cfg.Debug {
		fmt.Fprintf(cmd.ErrOrStderr(), "goenv: Generating SLSA provenance for %s\n", sbomPath)
		fmt.Fprintf(cmd.ErrOrStderr(), "goenv: Go version: %s\n", goVersion)
		fmt.Fprintf(cmd.ErrOrStderr(), "goenv: Invocation ID: %s\n", invocationID)
	}

	// Create provenance generator
	generator := sbom.NewProvenanceGenerator(sbom.ProvenanceOptions{
		SBOMPath:        sbomPath,
		GoVersion:       goVersion,
		GoModDigest:     goModDigest,
		GoSumDigest:     goSumDigest,
		BuildTags:       []string{}, // TODO: Extract from build context
		CGOEnabled:      false,      // TODO: Extract from build context
		GOOS:            runtime.GOOS,
		GOARCH:          runtime.GOARCH,
		LDFlags:         "",
		Vendored:        utils.FileExists(filepath.Join(projectDir, "vendor")),
		ModuleProxy:     os.Getenv("GOPROXY"),
		SBOMTool:        sbomTool,
		SBOMToolVersion: "latest", // TODO: Get actual version
		ProjectDir:      projectDir,
		InvocationID:    invocationID,
	})

	// Generate provenance
	statement, err := generator.Generate()
	if err != nil {
		return errors.FailedTo("generate provenance", err)
	}

	// Validate provenance
	if err := sbom.ValidateProvenance(statement); err != nil {
		return errors.FailedTo("validate provenance", err)
	}

	// If in-toto format requested and signing enabled
	if attestInToto && attestSign {
		if attestKeyPath == "" {
			return fmt.Errorf("--key is required when using --sign with --in-toto")
		}

		// Sign the provenance first
		signer := sbom.NewSigner(sbom.SignatureOptions{
			KeyPath: attestKeyPath,
		})

		// Serialize statement for signing
		statementData, err := json.Marshal(statement)
		if err != nil {
			return errors.FailedTo("marshal provenance", err)
		}

		// Create temp file for signing
		tempFile, err := os.CreateTemp("", "provenance-*.json")
		if err != nil {
			return errors.FailedTo("create temp file", err)
		}
		defer os.Remove(tempFile.Name())

		if err := os.WriteFile(tempFile.Name(), statementData, 0644); err != nil {
			return errors.FailedTo("write temp file", err)
		}

		signature, err := signer.SignSBOM(tempFile.Name())
		if err != nil {
			return errors.FailedTo("sign provenance", err)
		}

		// Create in-toto attestation
		attestation, err := sbom.CreateInTotoAttestation(statement, signature)
		if err != nil {
			return errors.FailedTo("create in-toto attestation", err)
		}

		// Write in-toto attestation
		if err := sbom.WriteInTotoAttestation(attestation, outputPath); err != nil {
			return errors.FailedTo("write attestation", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "✓ In-toto attestation generated and signed\n")
		fmt.Fprintf(cmd.OutOrStdout(), "  SBOM: %s\n", sbomPath)
		fmt.Fprintf(cmd.OutOrStdout(), "  Attestation: %s\n", outputPath)
		fmt.Fprintf(cmd.OutOrStdout(), "  Format: in-toto\n")
		fmt.Fprintf(cmd.OutOrStdout(), "  Signed: Yes\n")

	} else {
		// Write provenance statement
		if err := generator.WriteProvenance(statement, outputPath); err != nil {
			return errors.FailedTo("write provenance", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "✓ SLSA provenance generated\n")
		fmt.Fprintf(cmd.OutOrStdout(), "  SBOM: %s\n", sbomPath)
		fmt.Fprintf(cmd.OutOrStdout(), "  Provenance: %s\n", outputPath)
		fmt.Fprintf(cmd.OutOrStdout(), "  Format: SLSA v1.0\n")
		fmt.Fprintf(cmd.OutOrStdout(), "  Builder: goenv\n")
		fmt.Fprintf(cmd.OutOrStdout(), "  Go version: %s\n", goVersion)

		if attestSign && !attestInToto {
			fmt.Fprintf(cmd.OutOrStdout(), "\nTo sign the provenance, use:\n")
			fmt.Fprintf(cmd.OutOrStdout(), "  goenv sbom sign %s --key=%s\n", outputPath, attestKeyPath)
		}
	}

	if cfg.Debug {
		fmt.Fprintf(cmd.ErrOrStderr(), "Debug: Invocation ID: %s\n", invocationID)
		fmt.Fprintf(cmd.ErrOrStderr(), "Debug: go.mod digest: %s\n", goModDigest)
		fmt.Fprintf(cmd.ErrOrStderr(), "Debug: go.sum digest: %s\n", goSumDigest)
	}

	return nil
}

func runSBOMProject(cmd *cobra.Command, args []string) error {
	ctx := cmdutil.GetContexts(cmd)
	cfg := ctx.Config
	mgr := ctx.Manager
	env := ctx.Environment

	// Validate flags
	if sbomImage != "" && sbomDir != "." {
		return fmt.Errorf("cannot specify both --image and --dir")
	}

	if sbomImage != "" && sbomTool != "syft" {
		return fmt.Errorf("--image is only supported with --tool=syft")
	}

	// Get current Go version for provenance and tool resolution
	goVersion, versionSource, err := mgr.GetCurrentVersion()
	if err != nil {
		goVersion = "unknown"
	}

	// Resolve tool path using version context
	toolPath, err := resolveSBOMTool(cfg, env, sbomTool, goVersion, versionSource)
	if err != nil {
		return err
	}

	// Print provenance header to stderr (safe for CI logs)
	fmt.Fprintf(cmd.ErrOrStderr(), "goenv: Generating SBOM with %s (Go %s, %s/%s)\n",
		sbomTool, goVersion, platform.OS(), platform.Arch())

	// Build command based on tool
	var toolCmd *exec.Cmd
	switch sbomTool {
	case "cyclonedx-gomod":
		toolCmd, err = buildCycloneDXCommand(toolPath, cfg)
	case "syft":
		toolCmd, err = buildSyftCommand(toolPath, cfg)
	default:
		return fmt.Errorf("unsupported tool: %s (supported: cyclonedx-gomod, syft)", sbomTool)
	}

	if err != nil {
		return errors.FailedTo("build command", err)
	}

	// Set up environment
	toolCmd.Env = os.Environ()
	if sbomOffline {
		// Add offline flags if supported by tool
		// Most tools respect GOPROXY=off
		toolCmd.Env = append(toolCmd.Env, "GOPROXY=off")
	}

	// Connect output
	toolCmd.Stdout = cmd.OutOrStdout()
	toolCmd.Stderr = cmd.ErrOrStderr()

	// Run tool
	if cfg.Debug {
		fmt.Fprintf(cmd.ErrOrStderr(), "Debug: Running command: %s\n", strings.Join(toolCmd.Args, " "))
	}

	if err := toolCmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Preserve tool's exit code
			os.Exit(exitErr.ExitCode())
		}
		return errors.FailedTo("execute tool", err)
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "goenv: SBOM written to %s\n", sbomOutput)

	// Enhance SBOM with Go-aware metadata if enabled
	if sbomEnhance && (sbomTool == "cyclonedx-gomod" || sbomFormat == "cyclonedx-json") {
		if err := enhanceSBOM(cfg, mgr, cmd); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "goenv: Warning: Failed to enhance SBOM: %v\n", err)
			// Don't fail - enhancement is optional
		} else {
			fmt.Fprintf(cmd.ErrOrStderr(), "goenv: SBOM enhanced with Go-aware metadata\n")
		}
	}

	return nil
}

// enhanceSBOM adds Go-specific metadata to the generated SBOM
func enhanceSBOM(cfg *config.Config, mgr *manager.Manager, cmd *cobra.Command) error {
	// Import the enhancer package
	enhancer := sbom.NewEnhancer(cfg, mgr)

	opts := sbom.EnhanceOptions{
		ProjectDir:    sbomDir,
		Deterministic: sbomDeterministic,
		OfflineMode:   sbomOffline,
		EmbedDigests:  sbomEmbedDigests || sbomDeterministic, // Always embed if deterministic
	}

	return enhancer.EnhanceCycloneDX(sbomOutput, opts)
}

// resolveSBOMTool finds the tool binary in goenv-managed paths
func resolveSBOMTool(cfg *config.Config, env *utils.GoenvEnvironment, tool, version, versionSource string) (string, error) {
	// Use resolver to respect local vs global context
	sbomTools := map[string]string{
		"cyclonedx-gomod": "github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod",
		"syft":            "github.com/anchore/syft/cmd/syft",
	}

	r := resolver.New(cfg, env)

	if version != "unknown" && version != "" {
		if toolPath, err := r.ResolveBinary(tool, version, versionSource); err == nil {
			return toolPath, nil
		}
	}

	// Fallback: Check if tool is in PATH (system-wide installation)
	if path, err := exec.LookPath(tool); err == nil {
		return path, nil
	}

	goTool, ok := sbomTools[tool]
	if !ok {
		return "", fmt.Errorf("unsupported SBOM tool: %s", tool)
	}

	goTool = fmt.Sprintf("%s@latest", goTool)

	fmt.Printf("goenv: %s not found in goenv-managed paths or system PATH. Attempting to install...\n", tool)

	cmd := exec.Command("goenv", "tools", "install", goTool)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err == nil {
		// Retry finding the tool after installation; it will be rehashed into the shims automatically
		if toolPath, err := r.ResolveBinary(tool, version, versionSource); err == nil {
			return toolPath, nil
		}
	} else {
		return "", fmt.Errorf("goenv: Failed to install %s: %w", tool, err)
	}

	// Tool not found - provide actionable error
	return "", fmt.Errorf(`%s not found

To install:
  goenv tools install %s

Or install system-wide with:
  go install %s`, tool, goTool, goTool)
}

// buildCycloneDXCommand builds the cyclonedx-gomod command
func buildCycloneDXCommand(toolPath string, cfg *config.Config) (*exec.Cmd, error) {
	args := []string{}

	// Use the "mod" subcommand for module SBOMs (default behavior)
	// "mod" generates SBOMs for modules, including all packages
	// "app" would be for application binaries, "bin" for pre-built binaries
	args = append(args, "mod")

	// Output file - new format uses -output (with single dash)
	args = append(args, "-output", sbomOutput)

	// Format - newer versions use -output-format flag
	if sbomFormat == "cyclonedx-json" {
		args = append(args, "-json")
	} else if sbomFormat != "cyclonedx-xml" {
		return nil, fmt.Errorf("cyclonedx-gomod only supports cyclonedx-json and cyclonedx-xml formats")
	}

	// Modules only (include licenses and set type)
	if sbomModulesOnly {
		args = append(args, "-licenses", "-type", "library")
	}

	// Additional tool args
	if sbomToolArgs != "" {
		args = append(args, strings.Fields(sbomToolArgs)...)
	}

	cmdExec := exec.Command(toolPath, args...)
	cmdExec.Dir = sbomDir

	return cmdExec, nil
}

// buildSyftCommand builds the syft command
func buildSyftCommand(toolPath string, cfg *config.Config) (*exec.Cmd, error) {
	args := []string{}

	// Scan target (image or directory)
	target := sbomDir
	if sbomImage != "" {
		target = sbomImage
	}
	args = append(args, target)

	// Output format
	outputFormat := "cyclonedx-json"
	switch sbomFormat {
	case "cyclonedx-json":
		outputFormat = "cyclonedx-json"
	case "spdx-json":
		outputFormat = "spdx-json"
	case "syft-json":
		outputFormat = "json"
	default:
		return nil, fmt.Errorf("unsupported format for syft: %s", sbomFormat)
	}
	args = append(args, "-o", fmt.Sprintf("%s=%s", outputFormat, sbomOutput))

	// Quiet mode (reduce noise)
	args = append(args, "-q")

	// Additional tool args
	if sbomToolArgs != "" {
		args = append(args, strings.Fields(sbomToolArgs)...)
	}

	return exec.Command(toolPath, args...), nil
}

var sbomScanCmd = &cobra.Command{
	Use:   "scan <sbom-file>",
	Short: "Scan SBOM for vulnerabilities using security scanners",
	Long: `Scan an SBOM file for known vulnerabilities using security scanners.

Supported scanners:

Open Source (Phase 4A):
- Grype (Anchore): Fast, offline vulnerability scanning
- Trivy (Aqua Security): Kubernetes-native, container scanning

Commercial/Enterprise (Phase 4B):
- Snyk: Developer-first security with prioritized fixes
- Veracode: Enterprise compliance and policy enforcement

The scan command reads an SBOM file (CycloneDX or SPDX format) and checks all
components against vulnerability databases to identify security issues.

Results include:
- Vulnerability ID (CVE-2023-xxxxx, GHSA-xxxx-yyyy-zzzz)
- Affected package and version
- Severity level (Critical, High, Medium, Low)
- Fix information (available version with patch)
- CVSS scores and descriptions

Examples:
  # Scan with Grype (default)
  goenv sbom scan sbom.json

  # Scan with Trivy
  goenv sbom scan sbom.json --scanner=trivy

  # Scan with Snyk (requires SNYK_TOKEN)
  goenv sbom scan sbom.json --scanner=snyk

  # Scan with Veracode (requires API credentials)
  goenv sbom scan sbom.json --scanner=veracode

  # Show only high and critical vulnerabilities
  goenv sbom scan sbom.json --severity=high

  # Show only vulnerabilities with available fixes
  goenv sbom scan sbom.json --only-fixed

  # Save results to file
  goenv sbom scan sbom.json --output=scan-results.json

  # Fail build if any vulnerabilities found
  goenv sbom scan sbom.json --fail-on=any

Phase 4A/4B: Scanner Integration (v3.4+)
Supports both open-source and commercial scanners for comprehensive vulnerability detection.`,
	Args: cobra.ExactArgs(1),
	RunE: runSBOMScan,
}

var (
	scanScanner      string
	scanFormat       string
	scanOutputFormat string
	scanOutput       string
	scanSeverity     string
	scanFailOn       string
	scanOnlyFixed    bool
	scanOffline      bool
	scanVerbose      bool
	scanListScanners bool
)

func init() {
	sbomScanCmd.Flags().StringVar(&scanScanner, "scanner", "grype", "Scanner to use (grype, trivy)")
	sbomScanCmd.Flags().StringVar(&scanFormat, "format", "cyclonedx-json", "SBOM format (cyclonedx-json, spdx-json)")
	sbomScanCmd.Flags().StringVar(&scanOutputFormat, "output-format", "json", "Output format (json, table, sarif)")
	sbomScanCmd.Flags().StringVarP(&scanOutput, "output", "o", "", "Output file (default: stdout)")
	sbomScanCmd.Flags().StringVar(&scanSeverity, "severity", "", "Minimum severity to report (low, medium, high, critical)")
	sbomScanCmd.Flags().StringVar(&scanFailOn, "fail-on", "", "Exit with error if vulnerabilities found (any, high, critical)")
	sbomScanCmd.Flags().BoolVar(&scanOnlyFixed, "only-fixed", false, "Show only vulnerabilities with available fixes")
	sbomScanCmd.Flags().BoolVar(&scanOffline, "offline", false, "Offline mode - skip vulnerability database updates")
	sbomScanCmd.Flags().BoolVar(&scanVerbose, "verbose", false, "Verbose output")
	sbomScanCmd.Flags().BoolVar(&scanListScanners, "list-scanners", false, "List available scanners and exit")
}

func runSBOMScan(cmd *cobra.Command, args []string) error {
	// Handle --list-scanners flag
	if scanListScanners {
		return listScanners()
	}

	sbomPath := args[0]

	// Get scanner
	scanner, err := sbom.GetScanner(scanScanner)
	if err != nil {
		return err
	}

	// Check if scanner is installed
	if !scanner.IsInstalled() {
		fmt.Fprintf(os.Stderr, "Error: %s is not installed\n\n", scanner.Name())
		fmt.Fprintf(os.Stderr, "%s\n", scanner.InstallationInstructions())
		return fmt.Errorf("%s not found", scanner.Name())
	}

	// Check if scanner supports the format
	if !scanner.SupportsFormat(scanFormat) {
		return fmt.Errorf("%s does not support format: %s", scanner.Name(), scanFormat)
	}

	// Prepare scan options
	opts := &sbom.ScanOptions{
		SBOMPath:          sbomPath,
		Format:            scanFormat,
		OutputFormat:      scanOutputFormat,
		OutputPath:        scanOutput,
		SeverityThreshold: scanSeverity,
		FailOn:            scanFailOn,
		OnlyFixed:         scanOnlyFixed,
		Offline:           scanOffline,
		Verbose:           scanVerbose,
	}

	// Run scan
	fmt.Printf("Scanning %s with %s...\n", sbomPath, scanner.Name())

	ctx := cmd.Context()
	result, err := scanner.Scan(ctx, opts)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	// Display results
	if scanOutput == "" {
		// Print to stdout
		return displayScanResults(result, scanOutputFormat)
	}

	fmt.Printf("✅ Scan complete: %d vulnerabilities found\n", result.Summary.Total)
	fmt.Printf("   Critical: %d, High: %d, Medium: %d, Low: %d\n",
		result.Summary.Critical, result.Summary.High,
		result.Summary.Medium, result.Summary.Low)
	fmt.Printf("   Results saved to: %s\n", scanOutput)

	// Apply fail-on logic
	return checkFailOnCondition(result, scanFailOn)
}

func listScanners() error {
	fmt.Println("Available vulnerability scanners:")
	fmt.Println()

	scanners := sbom.ListAvailableScanners()
	for _, scanner := range scanners {
		installed := "❌ Not installed"
		if scanner.IsInstalled() {
			version, _ := scanner.Version()
			installed = fmt.Sprintf("✅ Installed (v%s)", version)
		}

		fmt.Printf("  %s - %s\n", scanner.Name(), installed)
	}

	fmt.Println()
	fmt.Println("To install a scanner:")
	fmt.Println("  goenv tools install grype")
	fmt.Println("  goenv tools install trivy")

	return nil
}

func displayScanResults(result *sbom.ScanResult, format string) error {
	switch format {
	case "json":
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal results: %w", err)
		}
		fmt.Println(string(data))

	case "table":
		displayTableResults(result)

	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}

	return nil
}

func displayTableResults(result *sbom.ScanResult) {
	fmt.Printf("\n🔍 Scan Results (%s v%s)\n", result.Scanner, result.ScannerVersion)
	fmt.Println(strings.Repeat("=", 80))

	fmt.Printf("\n📊 Summary:\n")
	fmt.Printf("   Total: %d vulnerabilities\n", result.Summary.Total)
	fmt.Printf("   Critical: %d | High: %d | Medium: %d | Low: %d\n",
		result.Summary.Critical, result.Summary.High,
		result.Summary.Medium, result.Summary.Low)
	fmt.Printf("   With Fix: %d | Without Fix: %d\n",
		result.Summary.WithFix, result.Summary.WithoutFix)

	if len(result.Vulnerabilities) == 0 {
		fmt.Printf("\n✅ No vulnerabilities found!\n")
		return
	}

	fmt.Printf("\n🚨 Vulnerabilities:\n")
	fmt.Println()

	for i, vuln := range result.Vulnerabilities {
		// Severity indicator
		indicator := getSeverityIndicator(vuln.Severity)

		fmt.Printf("%d. %s %s [%s]\n", i+1, indicator, vuln.ID, vuln.Severity)
		fmt.Printf("   Package: %s@%s\n", vuln.PackageName, vuln.PackageVersion)

		if vuln.FixAvailable {
			fmt.Printf("   ✅ Fix: Upgrade to %s\n", vuln.FixedInVersion)
		} else {
			fmt.Printf("   ⚠️  No fix available\n")
		}

		if vuln.CVSS > 0 {
			fmt.Printf("   CVSS: %.1f\n", vuln.CVSS)
		}

		if vuln.Description != "" {
			// Truncate long descriptions
			desc := vuln.Description
			if len(desc) > 100 {
				desc = desc[:97] + "..."
			}
			fmt.Printf("   %s\n", desc)
		}

		if len(vuln.URLs) > 0 {
			fmt.Printf("   🔗 %s\n", vuln.URLs[0])
		}

		fmt.Println()
	}
}

func getSeverityIndicator(severity string) string {
	switch severity {
	case "Critical":
		return "🔴"
	case "High":
		return "🟠"
	case "Medium":
		return "🟡"
	case "Low":
		return "🔵"
	default:
		return "⚪"
	}
}

func checkFailOnCondition(result *sbom.ScanResult, failOn string) error {
	if failOn == "" {
		return nil
	}

	switch failOn {
	case "any":
		if result.Summary.Total > 0 {
			return fmt.Errorf("found %d vulnerabilities (--fail-on=any)", result.Summary.Total)
		}
	case "critical":
		if result.Summary.Critical > 0 {
			return fmt.Errorf("found %d critical vulnerabilities", result.Summary.Critical)
		}
	case "high":
		if result.Summary.Critical > 0 || result.Summary.High > 0 {
			total := result.Summary.Critical + result.Summary.High
			return fmt.Errorf("found %d high/critical vulnerabilities", total)
		}
	}

	return nil
}
