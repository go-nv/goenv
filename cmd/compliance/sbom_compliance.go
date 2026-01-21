package compliance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-nv/goenv/internal/sbom"
	"github.com/spf13/cobra"
)

var (
	complianceFramework string
	complianceFormat    string
	complianceOutput    string
)

// sbomComplianceCmd represents the compliance command
var sbomComplianceCmd = &cobra.Command{
	Use:   "compliance",
	Short: "Generate compliance reports for SBOM",
	Long: `Generate compliance reports for various standards and frameworks.

Supported frameworks:
  - soc2: SOC 2 compliance checks
  - iso27001: ISO 27001 requirements
  - slsa: SLSA (Supply-chain Levels for Software Artifacts)
  - ssdf-v1.1: NIST Secure Software Development Framework
  - cisa: CISA SBOM requirements
  - all: Check against all frameworks

Output formats:
  - json: Machine-readable JSON format
  - html: Human-readable HTML report
  - text: Plain text summary (default)

Examples:
  # Generate SOC 2 compliance report
  goenv sbom compliance report sbom.json --framework soc2

  # Generate HTML report for ISO 27001
  goenv sbom compliance report sbom.json --framework iso27001 --format html --output report.html

  # Check all frameworks and save as JSON
  goenv sbom compliance report sbom.json --framework all --format json --output compliance.json

  # List available frameworks
  goenv sbom compliance frameworks`,
}

// sbomComplianceReportCmd generates compliance reports
var sbomComplianceReportCmd = &cobra.Command{
	Use:   "report <sbom-file>",
	Short: "Generate compliance report for SBOM",
	Args:  cobra.ExactArgs(1),
	RunE:  runComplianceReport,
}

// sbomComplianceFrameworksCmd lists available frameworks
var sbomComplianceFrameworksCmd = &cobra.Command{
	Use:   "frameworks",
	Short: "List available compliance frameworks",
	Run:   runComplianceFrameworks,
}

func init() {
	sbomCmd.AddCommand(sbomComplianceCmd)
	sbomComplianceCmd.AddCommand(sbomComplianceReportCmd)
	sbomComplianceCmd.AddCommand(sbomComplianceFrameworksCmd)

	sbomComplianceReportCmd.Flags().StringVarP(&complianceFramework, "framework", "f", "soc2", "Compliance framework (soc2, iso27001, slsa, ssdf-v1.1, cisa, all)")
	sbomComplianceReportCmd.Flags().StringVar(&complianceFormat, "format", "text", "Output format (json, html, text)")
	sbomComplianceReportCmd.Flags().StringVarP(&complianceOutput, "output", "o", "", "Output file (default: stdout)")
}

func runComplianceReport(cmd *cobra.Command, args []string) error {
	sbomPath := args[0]

	// Check if SBOM file exists
	if _, err := os.Stat(sbomPath); os.IsNotExist(err) {
		return fmt.Errorf("SBOM file not found: %s", sbomPath)
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(sbomPath)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	// Validate framework
	framework := sbom.ComplianceFramework(complianceFramework)
	validFrameworks := []string{"soc2", "iso27001", "slsa", "ssdf-v1.1", "cisa", "all"}
	isValid := false
	for _, f := range validFrameworks {
		if complianceFramework == f {
			isValid = true
			break
		}
	}
	if !isValid {
		return fmt.Errorf("invalid framework: %s (valid: %v)", complianceFramework, validFrameworks)
	}

	// Generate report
	reporter := sbom.NewComplianceReporter()
	report, err := reporter.GenerateReport(absPath, framework)
	if err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	// Format output
	var output string
	switch complianceFormat {
	case "json":
		jsonData, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		output = string(jsonData)

	case "html":
		output = reporter.FormatReportAsHTML(report)

	case "text":
		output = formatTextReport(report)

	default:
		return fmt.Errorf("invalid format: %s (valid: json, html, text)", complianceFormat)
	}

	// Write output
	if complianceOutput != "" {
		if err := os.WriteFile(complianceOutput, []byte(output), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Printf("✓ Compliance report written to %s\n", complianceOutput)
	} else {
		fmt.Println(output)
	}

	// Exit with non-zero if non-compliant
	if report.OverallStatus == "non-compliant" {
		os.Exit(1)
	}

	return nil
}

func runComplianceFrameworks(cmd *cobra.Command, args []string) {
	frameworks := []struct {
		id   string
		name string
		desc string
	}{
		{"soc2", "SOC 2", "Service Organization Control 2 compliance"},
		{"iso27001", "ISO 27001", "Information security management standard"},
		{"slsa", "SLSA", "Supply-chain Levels for Software Artifacts"},
		{"ssdf-v1.1", "SSDF v1.1", "NIST Secure Software Development Framework"},
		{"cisa", "CISA", "CISA SBOM requirements and guidance"},
		{"all", "All Frameworks", "Check against all supported frameworks"},
	}

	fmt.Println("Available Compliance Frameworks:")
	for _, f := range frameworks {
		fmt.Printf("  %-12s  %-15s  %s\n", f.id, f.name, f.desc)
	}
	fmt.Println("\nUse --framework <id> to select a framework")
}

func formatTextReport(report *sbom.ComplianceReport) string {
	output := "═══════════════════════════════════════════════════════════════\n"
	output += fmt.Sprintf("  COMPLIANCE REPORT: %s\n", report.Framework)
	output += "═══════════════════════════════════════════════════════════════\n\n"
	output += fmt.Sprintf("Generated: %s\n", report.GeneratedAt.Format("2006-01-02 15:04:05"))
	output += fmt.Sprintf("SBOM File: %s\n\n", report.SBOMPath)

	// Overall status
	statusSymbol := "✓"
	if report.OverallStatus == "non-compliant" {
		statusSymbol = "✗"
	} else if report.OverallStatus == "partially-compliant" {
		statusSymbol = "⚠"
	}
	output += fmt.Sprintf("Overall Status: %s %s\n\n", statusSymbol, report.OverallStatus)

	// Statistics
	output += "Requirements Summary:\n"
	output += fmt.Sprintf("  Total:   %d\n", report.PassedCount+report.FailedCount+report.PartialCount+report.NotApplicable)
	output += fmt.Sprintf("  Passed:  %d ✓\n", report.PassedCount)
	if report.FailedCount > 0 {
		output += fmt.Sprintf("  Failed:  %d ✗\n", report.FailedCount)
	}
	if report.PartialCount > 0 {
		output += fmt.Sprintf("  Partial: %d ⚠\n", report.PartialCount)
	}
	if report.NotApplicable > 0 {
		output += fmt.Sprintf("  N/A:     %d\n", report.NotApplicable)
	}
	output += "\n"

	// Requirements details
	output += "Requirements Details:\n"
	output += "───────────────────────────────────────────────────────────────\n\n"

	for i, req := range report.Requirements {
		check := report.Checks[i]

		// Status symbol
		symbol := "✓"
		if check.Status == "fail" {
			symbol = "✗"
		} else if check.Status == "partial" {
			symbol = "⚠"
		} else if check.Status == "not-applicable" {
			symbol = "-"
		}

		output += fmt.Sprintf("%s [%s] %s - %s\n", symbol, req.ID, req.Name, check.Status)
		output += fmt.Sprintf("  Category: %s | Severity: %s\n", req.Category, req.Severity)
		output += fmt.Sprintf("  %s\n", req.Description)

		if len(check.Evidence) > 0 {
			output += "  Evidence:\n"
			for _, ev := range check.Evidence {
				output += fmt.Sprintf("    • %s\n", ev)
			}
		}

		if len(check.Issues) > 0 {
			output += "  Issues:\n"
			for _, issue := range check.Issues {
				output += fmt.Sprintf("    ! %s\n", issue)
			}
		}

		if len(check.Recommendations) > 0 {
			output += "  Recommendations:\n"
			for _, rec := range check.Recommendations {
				output += fmt.Sprintf("    → %s\n", rec)
			}
		}

		output += "\n"
	}

	output += "═══════════════════════════════════════════════════════════════\n"
	output += report.Summary + "\n"
	output += "═══════════════════════════════════════════════════════════════\n"

	return output
}
