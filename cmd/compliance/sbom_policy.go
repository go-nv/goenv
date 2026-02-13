package compliance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-nv/goenv/internal/sbom"
	"github.com/spf13/cobra"
)

var (
	policyFilePath   string
	policyJSONOutput bool
	policyFailOnWarn bool
	policyGenPath    string
)

// sbomPolicyCmd represents the policy command
var sbomPolicyCmd = &cobra.Command{
	Use:   "policy",
	Short: "SBOM policy validation and enforcement",
	Long: `Validate SBOMs against defined compliance policies.

Policies are defined in YAML files and can enforce rules for:
- License compliance (allowed/denied licenses)
- Vulnerability thresholds (max critical/high/medium)
- Dependency restrictions (allowed/blocked dependencies)
- Metadata requirements (supplier, author, formats)

Examples:
  # Validate SBOM against policy
  goenv sbom policy validate sbom.json --policy .sbom-policy.yaml
  
  # Check SBOM and fail on violations
  goenv sbom policy check sbom.json --policy .sbom-policy.yaml --fail-on-warning
  
  # Generate a default policy template
  goenv sbom policy generate --output sbom-policy.yaml
  
  # Validate with JSON output for CI/CD
  goenv sbom policy validate sbom.json --json`,
}

// sbomPolicyValidateCmd validates an SBOM against a policy
var sbomPolicyValidateCmd = &cobra.Command{
	Use:   "validate <sbom-file>",
	Short: "Validate SBOM against policy rules",
	Long: `Validate an SBOM file against defined policy rules.

This command checks the SBOM for compliance with organizational
policies including license restrictions, vulnerability thresholds,
dependency rules, and metadata requirements.

Policy files are YAML documents that define validation rules.
If no policy is specified, the command searches for common
policy file names in the current directory.

Exit codes:
  0 - All policy checks passed
  1 - Policy violations found
  2 - Error reading SBOM or policy`,
	Args: cobra.ExactArgs(1),
	RunE: runPolicyValidate,
}

// sbomPolicyCheckCmd is an alias for validate with stricter defaults
var sbomPolicyCheckCmd = &cobra.Command{
	Use:   "check <sbom-file>",
	Short: "Check SBOM compliance (strict mode)",
	Long: `Check SBOM compliance with strict policy enforcement.

This is an alias for 'validate' but with stricter defaults:
- Fails on warnings by default
- Exits with non-zero on any violation
- Designed for CI/CD pipelines

Use this command in automated workflows where you want
to enforce strict compliance.`,
	Args: cobra.ExactArgs(1),
	RunE: runPolicyCheck,
}

// sbomPolicyGenerateCmd generates a default policy template
var sbomPolicyGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate default policy template",
	Long: `Generate a default SBOM policy configuration file.

This creates a sample policy YAML file with common rules
that you can customize for your organization's requirements.

The generated policy includes:
- Common license allowlists/denylists
- Reasonable vulnerability thresholds
- Basic metadata requirements
- Example dependency rules

Customize this template to match your organization's
security and compliance requirements.`,
	RunE: runPolicyGenerate,
}

// sbomPolicyReportCmd generates a policy compliance report
var sbomPolicyReportCmd = &cobra.Command{
	Use:   "report <sbom-file>",
	Short: "Generate policy compliance report",
	Long: `Generate a detailed compliance report from policy validation.

Creates a comprehensive report showing:
- Policy validation results
- Violations by severity
- Component-level details
- Remediation recommendations
- Summary statistics

Output formats:
  text  - Human-readable report (default)
  json  - Machine-readable JSON
  html  - HTML report (if supported)`,
	Args: cobra.ExactArgs(1),
	RunE: runPolicyReport,
}

func init() {
	// Add policy command to sbom
	sbomCmd.AddCommand(sbomPolicyCmd)

	// Add subcommands
	sbomPolicyCmd.AddCommand(sbomPolicyValidateCmd)
	sbomPolicyCmd.AddCommand(sbomPolicyCheckCmd)
	sbomPolicyCmd.AddCommand(sbomPolicyGenerateCmd)
	sbomPolicyCmd.AddCommand(sbomPolicyReportCmd)

	// Validate flags
	sbomPolicyValidateCmd.Flags().StringVar(&policyFilePath, "policy", "", "Path to policy file (auto-detects if not specified)")
	sbomPolicyValidateCmd.Flags().BoolVar(&policyJSONOutput, "json", false, "Output results as JSON")
	sbomPolicyValidateCmd.Flags().BoolVar(&policyFailOnWarn, "fail-on-warning", false, "Exit with error on warnings")

	// Check flags (same as validate)
	sbomPolicyCheckCmd.Flags().StringVar(&policyFilePath, "policy", "", "Path to policy file (auto-detects if not specified)")
	sbomPolicyCheckCmd.Flags().BoolVar(&policyJSONOutput, "json", false, "Output results as JSON")
	// Note: fail-on-warning is true by default for check

	// Generate flags
	sbomPolicyGenerateCmd.Flags().StringVarP(&policyGenPath, "output", "o", "sbom-policy.yaml", "Output path for generated policy")

	// Report flags
	sbomPolicyReportCmd.Flags().StringVar(&policyFilePath, "policy", "", "Path to policy file (auto-detects if not specified)")
	sbomPolicyReportCmd.Flags().BoolVar(&policyJSONOutput, "json", false, "Output report as JSON")
}

func runPolicyValidate(cmd *cobra.Command, args []string) error {
	sbomPath := args[0]

	// Load policy engine
	engine, err := loadPolicyEngine(policyFilePath)
	if err != nil {
		return fmt.Errorf("failed to load policy: %w", err)
	}

	fmt.Printf("Validating SBOM: %s\n", sbomPath)
	if policyFilePath != "" {
		fmt.Printf("Policy: %s\n", policyFilePath)
	}
	fmt.Println()

	// Run validation
	result, err := engine.Validate(sbomPath)
	if err != nil {
		return fmt.Errorf("policy validation failed: %w", err)
	}

	// Output results
	if policyJSONOutput {
		return outputPolicyJSON(result)
	}

	// Text output
	fmt.Println(result.Summary)

	if !result.Passed {
		fmt.Println("\n❌ Policy validation FAILED")
		os.Exit(1)
	}

	if policyFailOnWarn && len(result.Warnings) > 0 {
		fmt.Println("\n⚠️  Warnings treated as failures (--fail-on-warning)")
		os.Exit(1)
	}

	fmt.Println("\n✅ Policy validation PASSED")
	return nil
}

func runPolicyCheck(cmd *cobra.Command, args []string) error {
	// Check is just validate with stricter defaults
	policyFailOnWarn = true
	return runPolicyValidate(cmd, args)
}

func runPolicyGenerate(cmd *cobra.Command, args []string) error {
	// Check if file already exists
	if _, err := os.Stat(policyGenPath); err == nil {
		return fmt.Errorf("policy file already exists: %s (use --output to specify different path)", policyGenPath)
	}

	fmt.Printf("Generating default policy: %s\n", policyGenPath)

	// Generate default policy using the existing PolicyConfig structure
	defaultPolicy := createDefaultPolicy()

	// Write to file with comments
	if err := writePolicyWithComments(policyGenPath, defaultPolicy); err != nil {
		return fmt.Errorf("failed to write policy: %w", err)
	}

	fmt.Println("✅ Policy template created successfully")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Review and customize the policy rules")
	fmt.Println("  2. Test with: goenv sbom policy validate sbom.json")
	fmt.Println("  3. Integrate into CI/CD pipeline")

	return nil
}

func runPolicyReport(cmd *cobra.Command, args []string) error {
	sbomPath := args[0]

	// Load policy engine
	engine, err := loadPolicyEngine(policyFilePath)
	if err != nil {
		return fmt.Errorf("failed to load policy: %w", err)
	}

	// Run validation
	result, err := engine.Validate(sbomPath)
	if err != nil {
		return fmt.Errorf("policy validation failed: %w", err)
	}

	// Generate detailed report
	if policyJSONOutput {
		return outputPolicyJSON(result)
	}

	// Text report
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("         SBOM Policy Compliance Report")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("SBOM: %s\n", sbomPath)
	if policyFilePath != "" {
		fmt.Printf("Policy: %s\n", policyFilePath)
	}
	fmt.Println()

	// Overall status
	if result.Passed {
		fmt.Println("Status: ✅ COMPLIANT")
	} else {
		fmt.Println("Status: ❌ NON-COMPLIANT")
	}
	fmt.Println()

	// Summary statistics
	fmt.Println("Summary:")
	fmt.Printf("  Total Violations: %d\n", len(result.Violations))
	fmt.Printf("  Warnings: %d\n", len(result.Warnings))
	fmt.Println()

	// Violations by severity
	if len(result.Violations) > 0 {
		fmt.Println("Violations:")
		for _, v := range result.Violations {
			fmt.Printf("\n  🔴 [%s] %s\n", strings.ToUpper(v.Severity), v.Rule)
			fmt.Printf("     Component: %s\n", v.Component)
			fmt.Printf("     Message: %s\n", v.Message)
			if v.Remediation != "" {
				fmt.Printf("     Fix: %s\n", v.Remediation)
			}
		}
		fmt.Println()
	}

	// Warnings
	if len(result.Warnings) > 0 {
		fmt.Println("Warnings:")
		for _, w := range result.Warnings {
			fmt.Printf("\n  🟡 [%s] %s\n", strings.ToUpper(w.Severity), w.Rule)
			fmt.Printf("     Component: %s\n", w.Component)
			fmt.Printf("     Message: %s\n", w.Message)
			if w.Remediation != "" {
				fmt.Printf("     Recommendation: %s\n", w.Remediation)
			}
		}
		fmt.Println()
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if !result.Passed {
		os.Exit(1)
	}

	return nil
}

// Helper functions

func loadPolicyEngine(policyPath string) (*sbom.PolicyEngine, error) {
	if policyPath != "" {
		// Use specified policy file
		return sbom.NewPolicyEngine(policyPath)
	}

	// Auto-detect policy file in current directory
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	policyNames := []string{
		".sbom-policy.yaml",
		".sbom-policy.yml",
		"sbom-policy.yaml",
		"sbom-policy.yml",
		".goenv-policy.yaml",
		".goenv-policy.yml",
	}

	for _, name := range policyNames {
		path := filepath.Join(cwd, name)
		if _, err := os.Stat(path); err == nil {
			policyFilePath = path // Set for output
			return sbom.NewPolicyEngine(path)
		}
	}

	return nil, fmt.Errorf("no policy file found (searched: %v). Use --policy to specify or generate one with 'goenv sbom policy generate'", policyNames)
}

func outputPolicyJSON(result *sbom.PolicyResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Println(string(data))
	return nil
}

func createDefaultPolicy() string {
	return `# SBOM Policy Configuration
# Version: 1.0
# This file defines compliance rules for SBOM validation

version: "1.0"

# Supply chain security rules
rules:
  # Prevent local path dependencies
  - name: no-local-dependencies
    type: supply-chain
    severity: error
    description: Local path replace directives introduce supply chain risk
    check: replace-directives
    blocked:
      - local-path

  # Prevent vendored dependencies
  - name: no-vendor-directory
    type: supply-chain
    severity: warning
    description: Vendored dependencies should use module proxy
    check: vendoring-status
    blocked:
      - vendored

  # Security rules
  - name: no-retracted-versions
    type: security
    severity: error
    description: Retracted versions have known issues
    check: retracted-versions

  - name: cgo-disabled-production
    type: security
    severity: warning
    description: CGO increases attack surface
    check: cgo-disabled
    required:
      - "false"

  # License compliance
  - name: approved-licenses-only
    type: license
    severity: error
    description: Only approved licenses are allowed
    blocked:
      - GPL-3.0
      - AGPL-3.0
      - SSPL-1.0

  # Completeness checks
  - name: required-metadata
    type: completeness
    severity: warning
    description: SBOM should include comprehensive metadata
    check: required-metadata
    required:
      - goenv:go_version
      - goenv:build_context.goos
      - goenv:build_context.goarch

# Policy enforcement options
options:
  fail_on_error: true
  fail_on_warning: false
  verbose: false
`
}

func writePolicyWithComments(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
