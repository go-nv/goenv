package compliance

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/go-nv/goenv/internal/sbom"
	"github.com/spf13/cobra"
)

var (
	ciCheckSBOMPath string
	ciCheckMaxAge   string
	ciCheckJSON     bool

	ciScanScanner     string
	ciScanSeverity    string
	ciScanOutputJSON  string
	ciScanOutputSARIF string
	ciScanFailOn      string
)

var sbomCICmd = &cobra.Command{
	Use:   "ci",
	Short: "CI/CD pipeline integration commands",
	Long: `Commands designed for CI/CD pipeline integration.

These commands are optimized for continuous integration environments,
providing machine-readable output and proper exit codes for pipeline control.

Supported CI/CD platforms:
  - GitHub Actions
  - GitLab CI
  - CircleCI
  - Jenkins
  - Azure Pipelines

Examples:
  # Check if SBOM exists and is up-to-date
  goenv sbom ci check

  # Check with custom SBOM path
  goenv sbom ci check --sbom sbom.cyclonedx.json

  # Run vulnerability scan
  goenv sbom ci scan --scanner grype

  # Scan with severity threshold
  goenv sbom ci scan --scanner trivy --fail-on high`,
}

var sbomCICheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check SBOM existence and staleness",
	Long: `Check if SBOM exists and is up-to-date in CI/CD pipeline.

This command:
- Verifies SBOM file exists
- Checks if SBOM is stale (older than go.mod/go.sum)
- Optionally checks maximum age
- Outputs CI platform-specific annotations
- Returns exit code 1 if checks fail

The command automatically detects the CI/CD platform and formats
output accordingly (GitHub Actions annotations, GitLab CI format, etc.)`,
	RunE: runCICheck,
}

var sbomCIScanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Run vulnerability scan in CI/CD",
	Long: `Run vulnerability scanner and format output for CI/CD.

This command:
- Runs vulnerability scanner (Grype, Trivy, Snyk, etc.)
- Formats output for CI platform (annotations, reports)
- Exports results in JSON or SARIF format
- Returns exit code based on severity threshold
- Integrates with GitHub Security tab (SARIF upload)

The scanner must be installed and available in PATH.`,
	RunE: runCIScan,
}

func init() {
	// Add CI subcommands
	sbomCICmd.AddCommand(sbomCICheckCmd)
	sbomCICmd.AddCommand(sbomCIScanCmd)

	// Check flags
	sbomCICheckCmd.Flags().StringVar(&ciCheckSBOMPath, "sbom", "", "Path to SBOM file (auto-detected if not specified)")
	sbomCICheckCmd.Flags().StringVar(&ciCheckMaxAge, "max-age", "", "Maximum age for SBOM (e.g., '24h', '7d')")
	sbomCICheckCmd.Flags().BoolVar(&ciCheckJSON, "json", false, "Output results as JSON")

	// Scan flags
	sbomCIScanCmd.Flags().StringVar(&ciCheckSBOMPath, "sbom", "", "Path to SBOM file (auto-detected if not specified)")
	sbomCIScanCmd.Flags().StringVar(&ciScanScanner, "scanner", "grype", "Scanner to use (grype, trivy, snyk, veracode)")
	sbomCIScanCmd.Flags().StringVar(&ciScanSeverity, "fail-on", "high", "Fail on severity level (critical, high, medium, low)")
	sbomCIScanCmd.Flags().StringVar(&ciScanOutputJSON, "output-json", "", "Write JSON results to file")
	sbomCIScanCmd.Flags().StringVar(&ciScanOutputSARIF, "output-sarif", "", "Write SARIF results to file (for GitHub Code Scanning)")

	// Add to parent
	sbomCmd.AddCommand(sbomCICmd)
}

func runCICheck(cmd *cobra.Command, args []string) error {
	// Create CI checker
	checker := sbom.NewCIChecker("")

	// Parse max age if specified
	var maxAge time.Duration
	if ciCheckMaxAge != "" {
		var err error
		maxAge, err = time.ParseDuration(ciCheckMaxAge)
		if err != nil {
			return fmt.Errorf("invalid max-age format: %w (use format like '24h', '7d')", err)
		}
	}

	// Run check
	result, err := checker.CheckSBOM(ciCheckSBOMPath, maxAge)
	if err != nil {
		return fmt.Errorf("SBOM check failed: %w", err)
	}

	// Output results
	if ciCheckJSON {
		// JSON output
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))
	} else {
		// CI platform-specific output
		output := checker.FormatCIOutput(result)
		fmt.Print(output)

		// Also print summary for human readers
		if !result.Passed {
			fmt.Println()
			if !result.SBOMExists {
				fmt.Println("SBOM file not found. Please generate one with:")
				fmt.Println("  goenv sbom generate")
			} else if result.IsStale {
				fmt.Printf("SBOM is stale: %s\n", result.StaleReason)
				fmt.Println("Please regenerate with:")
				fmt.Println("  goenv sbom generate")
			}
		}
	}

	// Exit with error code if check failed
	if !result.Passed {
		os.Exit(1)
	}

	return nil
}

func runCIScan(cmd *cobra.Command, args []string) error {
	// Create CI checker
	checker := sbom.NewCIChecker("")

	// Detect CI platform
	platform := sbom.DetectCIPlatform()
	fmt.Printf("Detected CI platform: %s\n", platform)

	// Find or check SBOM
	sbomPath := ciCheckSBOMPath
	if sbomPath == "" {
		// Auto-detect
		candidates := []string{
			"sbom.json",
			"sbom.cyclonedx.json",
			"sbom.spdx.json",
		}

		for _, candidate := range candidates {
			if _, err := os.Stat(candidate); err == nil {
				sbomPath = candidate
				break
			}
		}

		if sbomPath == "" {
			return fmt.Errorf("SBOM file not found. Generate with: goenv sbom generate")
		}
	}

	fmt.Printf("Scanning SBOM: %s\n", sbomPath)
	fmt.Printf("Scanner: %s\n", ciScanScanner)
	fmt.Println()

	// Configure scan options
	scanOptions := sbom.ScanOptions{
		SBOMPath: sbomPath,
		FailOn:   ciScanSeverity,
		Format:   "json",
		Verbose:  false,
	}

	// Run scan
	result, err := checker.RunScanner(sbomPath, ciScanScanner, scanOptions)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	// Output formatted results
	output := checker.FormatScanOutput(result)
	fmt.Print(output)

	// Write JSON output if requested
	if ciScanOutputJSON != "" {
		fmt.Printf("\nWriting JSON results to: %s\n", ciScanOutputJSON)
		if err := checker.WriteScanResultToFile(result, ciScanOutputJSON); err != nil {
			return fmt.Errorf("failed to write JSON output: %w", err)
		}
	}

	// Write SARIF output if requested (for GitHub Code Scanning)
	if ciScanOutputSARIF != "" {
		fmt.Printf("Writing SARIF results to: %s\n", ciScanOutputSARIF)
		if err := checker.ExportToGitHubSARIF(result, ciScanOutputSARIF); err != nil {
			return fmt.Errorf("failed to write SARIF output: %w", err)
		}

		if platform == sbom.PlatformGitHubActions {
			fmt.Println("\nTo upload to GitHub Security tab, add to your workflow:")
			fmt.Println("  - uses: github/codeql-action/upload-sarif@v2")
			fmt.Println("    with:")
			fmt.Printf("      sarif_file: %s\n", ciScanOutputSARIF)
		}
	}

	// Exit with error code if scan failed
	if !result.Passed {
		fmt.Println("\n❌ Scan failed due to vulnerabilities exceeding threshold")
		os.Exit(1)
	}

	return nil
}
