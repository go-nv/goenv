package sbom

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// ComplianceFramework represents a compliance standard
type ComplianceFramework string

const (
	FrameworkSOC2     ComplianceFramework = "soc2"
	FrameworkISO27001 ComplianceFramework = "iso27001"
	FrameworkSLSA     ComplianceFramework = "slsa"
	FrameworkSSDFv1   ComplianceFramework = "ssdf-v1.1"
	FrameworkCISA     ComplianceFramework = "cisa"
	FrameworkAll      ComplianceFramework = "all"
)

// ComplianceRequirement represents a single compliance requirement
type ComplianceRequirement struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Severity    string   `json:"severity"`
	Checks      []string `json:"checks"`
}

// ComplianceCheck represents the result of checking a requirement
type ComplianceCheck struct {
	RequirementID   string   `json:"requirement_id"`
	Status          string   `json:"status"` // pass, fail, partial, not-applicable
	Evidence        []string `json:"evidence,omitempty"`
	Issues          []string `json:"issues,omitempty"`
	Recommendations []string `json:"recommendations,omitempty"`
}

// ComplianceReport represents a full compliance report
type ComplianceReport struct {
	Framework     string                  `json:"framework"`
	GeneratedAt   time.Time               `json:"generated_at"`
	SBOMPath      string                  `json:"sbom_path"`
	OverallStatus string                  `json:"overall_status"`
	PassedCount   int                     `json:"passed_count"`
	FailedCount   int                     `json:"failed_count"`
	PartialCount  int                     `json:"partial_count"`
	NotApplicable int                     `json:"not_applicable_count"`
	Requirements  []ComplianceRequirement `json:"requirements"`
	Checks        []ComplianceCheck       `json:"checks"`
	Summary       string                  `json:"summary"`
}

// ComplianceReporter generates compliance reports
type ComplianceReporter struct {
	frameworks map[ComplianceFramework][]ComplianceRequirement
}

// NewComplianceReporter creates a new compliance reporter
func NewComplianceReporter() *ComplianceReporter {
	reporter := &ComplianceReporter{
		frameworks: make(map[ComplianceFramework][]ComplianceRequirement),
	}
	reporter.initializeFrameworks()
	return reporter
}

// GenerateReport generates a compliance report for a given SBOM
func (cr *ComplianceReporter) GenerateReport(sbomPath string, framework ComplianceFramework) (*ComplianceReport, error) {
	// Read SBOM file
	sbomData, err := os.ReadFile(sbomPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SBOM: %w", err)
	}

	var sbom map[string]interface{}
	if err := json.Unmarshal(sbomData, &sbom); err != nil {
		return nil, fmt.Errorf("failed to parse SBOM: %w", err)
	}

	// Get requirements for framework
	requirements := cr.getRequirements(framework)
	if len(requirements) == 0 {
		return nil, fmt.Errorf("unknown framework: %s", framework)
	}

	// Run compliance checks
	checks := cr.runChecks(sbom, requirements)

	// Calculate statistics
	passed, failed, partial, notApplicable := 0, 0, 0, 0
	for _, check := range checks {
		switch check.Status {
		case "pass":
			passed++
		case "fail":
			failed++
		case "partial":
			partial++
		case "not-applicable":
			notApplicable++
		}
	}

	// Determine overall status
	overallStatus := "compliant"
	if failed > 0 {
		overallStatus = "non-compliant"
	} else if partial > 0 {
		overallStatus = "partially-compliant"
	}

	report := &ComplianceReport{
		Framework:     string(framework),
		GeneratedAt:   time.Now(),
		SBOMPath:      sbomPath,
		OverallStatus: overallStatus,
		PassedCount:   passed,
		FailedCount:   failed,
		PartialCount:  partial,
		NotApplicable: notApplicable,
		Requirements:  requirements,
		Checks:        checks,
		Summary:       cr.generateSummary(overallStatus, passed, failed, partial, notApplicable),
	}

	return report, nil
}

// runChecks executes compliance checks against the SBOM
func (cr *ComplianceReporter) runChecks(sbom map[string]interface{}, requirements []ComplianceRequirement) []ComplianceCheck {
	checks := make([]ComplianceCheck, 0, len(requirements))

	for _, req := range requirements {
		check := ComplianceCheck{
			RequirementID:   req.ID,
			Evidence:        []string{},
			Issues:          []string{},
			Recommendations: []string{},
		}

		// Run each check type
		for _, checkType := range req.Checks {
			cr.executeCheck(&check, sbom, checkType)
		}

		// Determine overall status for this requirement
		if len(check.Issues) == 0 {
			check.Status = "pass"
		} else if len(check.Evidence) > 0 {
			check.Status = "partial"
		} else {
			check.Status = "fail"
		}

		checks = append(checks, check)
	}

	return checks
}

// executeCheck runs a specific check type
func (cr *ComplianceReporter) executeCheck(check *ComplianceCheck, sbom map[string]interface{}, checkType string) {
	switch checkType {
	case "has-sbom":
		check.Evidence = append(check.Evidence, "SBOM file exists and is valid")

	case "sbom-format":
		if format, ok := sbom["bomFormat"].(string); ok {
			check.Evidence = append(check.Evidence, fmt.Sprintf("SBOM format: %s", format))
		} else {
			check.Issues = append(check.Issues, "SBOM format not specified")
		}

	case "has-components":
		components := extractComponents(sbom)
		if len(components) > 0 {
			check.Evidence = append(check.Evidence, fmt.Sprintf("Found %d components", len(components)))
		} else {
			check.Issues = append(check.Issues, "No components found in SBOM")
			check.Recommendations = append(check.Recommendations, "Ensure all dependencies are included in SBOM")
		}

	case "component-licenses":
		components := extractComponents(sbom)
		licensedCount := 0
		unlicensedComponents := []string{}

		for _, comp := range components {
			compMap, ok := comp.(map[string]interface{})
			if !ok {
				continue
			}

			name := getComponentName(compMap)
			if hasLicense(compMap) {
				licensedCount++
			} else {
				unlicensedComponents = append(unlicensedComponents, name)
			}
		}

		if licensedCount > 0 {
			check.Evidence = append(check.Evidence, fmt.Sprintf("%d/%d components have license information", licensedCount, len(components)))
		}

		if len(unlicensedComponents) > 0 {
			check.Issues = append(check.Issues, fmt.Sprintf("%d components missing license info", len(unlicensedComponents)))
			check.Recommendations = append(check.Recommendations, "Add license information for all components")
		}

	case "component-versions":
		components := extractComponents(sbom)
		versionedCount := 0

		for _, comp := range components {
			if compMap, ok := comp.(map[string]interface{}); ok {
				if _, hasVersion := compMap["version"].(string); hasVersion {
					versionedCount++
				}
			}
		}

		if versionedCount == len(components) {
			check.Evidence = append(check.Evidence, fmt.Sprintf("All %d components have version information", versionedCount))
		} else {
			check.Issues = append(check.Issues, fmt.Sprintf("%d/%d components missing version info", len(components)-versionedCount, len(components)))
			check.Recommendations = append(check.Recommendations, "Ensure all components have version information")
		}

	case "build-metadata":
		metadata := extractMetadata(sbom)
		properties := extractProperties(metadata)

		buildProps := []string{"goenv:go_version", "goenv:build_context.goos", "goenv:build_context.goarch"}
		foundProps := 0

		for _, prop := range properties {
			if propName, ok := prop["name"].(string); ok {
				for _, buildProp := range buildProps {
					if propName == buildProp {
						foundProps++
						break
					}
				}
			}
		}

		if foundProps > 0 {
			check.Evidence = append(check.Evidence, fmt.Sprintf("Found %d/%d build metadata properties", foundProps, len(buildProps)))
		}

		if foundProps < len(buildProps) {
			check.Issues = append(check.Issues, "Incomplete build metadata")
			check.Recommendations = append(check.Recommendations, "Include complete build environment information")
		}

	case "supply-chain-transparency":
		properties := extractProperties(extractMetadata(sbom))
		hasReplaces := false
		hasVendored := false

		for _, prop := range properties {
			if propName, ok := prop["name"].(string); ok {
				if strings.Contains(propName, "replaces") {
					hasReplaces = true
				}
				if strings.Contains(propName, "vendored") {
					hasVendored = true
				}
			}
		}

		if hasReplaces || hasVendored {
			check.Evidence = append(check.Evidence, "Supply chain information present")
		} else {
			check.Evidence = append(check.Evidence, "Standard supply chain (no modifications)")
		}

	case "timestamp":
		metadata := extractMetadata(sbom)
		if timestamp, ok := metadata["timestamp"].(string); ok && timestamp != "" {
			check.Evidence = append(check.Evidence, fmt.Sprintf("SBOM generated at: %s", timestamp))
		} else {
			check.Issues = append(check.Issues, "SBOM timestamp missing")
			check.Recommendations = append(check.Recommendations, "Include generation timestamp in SBOM")
		}

	case "tool-information":
		metadata := extractMetadata(sbom)
		if tools, ok := metadata["tools"].([]interface{}); ok && len(tools) > 0 {
			check.Evidence = append(check.Evidence, fmt.Sprintf("Generated by %d tool(s)", len(tools)))
		} else {
			check.Issues = append(check.Issues, "Tool information missing")
			check.Recommendations = append(check.Recommendations, "Include tool metadata in SBOM")
		}

	case "vulnerability-tracking":
		// Check if vulnerability information is present
		components := extractComponents(sbom)
		hasVulnInfo := false

		for _, comp := range components {
			if compMap, ok := comp.(map[string]interface{}); ok {
				if props, ok := compMap["properties"].([]interface{}); ok {
					for _, prop := range props {
						if propMap, ok := prop.(map[string]interface{}); ok {
							if name, _ := propMap["name"].(string); strings.Contains(name, "vuln") || strings.Contains(name, "cve") {
								hasVulnInfo = true
								break
							}
						}
					}
				}
			}
		}

		if hasVulnInfo {
			check.Evidence = append(check.Evidence, "Vulnerability tracking information present")
		} else {
			check.Issues = append(check.Issues, "No vulnerability tracking information")
			check.Recommendations = append(check.Recommendations, "Integrate vulnerability scanning results")
		}

	case "provenance":
		// Check for provenance information
		metadata := extractMetadata(sbom)
		properties := extractProperties(metadata)

		hasGitInfo := false
		for _, prop := range properties {
			if propName, ok := prop["name"].(string); ok {
				if strings.Contains(propName, "git") || strings.Contains(propName, "vcs") || strings.Contains(propName, "commit") {
					hasGitInfo = true
					break
				}
			}
		}

		if hasGitInfo {
			check.Evidence = append(check.Evidence, "Provenance information available")
		} else {
			check.Issues = append(check.Issues, "No provenance information")
			check.Recommendations = append(check.Recommendations, "Include source repository and commit information")
		}
	}
}

// Helper functions
func getComponentName(comp map[string]interface{}) string {
	if name, ok := comp["name"].(string); ok {
		return name
	}
	return "unknown"
}

func hasLicense(comp map[string]interface{}) bool {
	if licenses, ok := comp["licenses"].([]interface{}); ok && len(licenses) > 0 {
		return true
	}
	if _, ok := comp["license"].(string); ok {
		return true
	}
	return false
}

// generateSummary creates a human-readable summary
func (cr *ComplianceReporter) generateSummary(status string, passed, failed, partial, notApplicable int) string {
	total := passed + failed + partial + notApplicable

	summary := fmt.Sprintf("Compliance Status: %s\n", strings.ToUpper(status))
	summary += fmt.Sprintf("Total Requirements: %d\n", total)
	summary += fmt.Sprintf("  ✓ Passed: %d\n", passed)

	if failed > 0 {
		summary += fmt.Sprintf("  ✗ Failed: %d\n", failed)
	}
	if partial > 0 {
		summary += fmt.Sprintf("  ⚠ Partial: %d\n", partial)
	}
	if notApplicable > 0 {
		summary += fmt.Sprintf("  - Not Applicable: %d\n", notApplicable)
	}

	if status == "compliant" {
		summary += "\nThe SBOM meets all requirements for this framework."
	} else if status == "partially-compliant" {
		summary += "\nThe SBOM meets most requirements but has some gaps."
	} else {
		summary += "\nThe SBOM does not meet the requirements for this framework."
	}

	return summary
}

// getRequirements returns requirements for a framework
func (cr *ComplianceReporter) getRequirements(framework ComplianceFramework) []ComplianceRequirement {
	if framework == FrameworkAll {
		// Combine all frameworks
		var all []ComplianceRequirement
		for _, reqs := range cr.frameworks {
			all = append(all, reqs...)
		}
		return all
	}
	return cr.frameworks[framework]
}

// initializeFrameworks sets up framework requirements
func (cr *ComplianceReporter) initializeFrameworks() {
	// SOC 2 Requirements
	cr.frameworks[FrameworkSOC2] = []ComplianceRequirement{
		{
			ID:          "SOC2-CC6.1",
			Name:        "Software Inventory",
			Description: "Maintain inventory of software components",
			Category:    "Change Management",
			Severity:    "high",
			Checks:      []string{"has-sbom", "has-components", "component-versions"},
		},
		{
			ID:          "SOC2-CC7.2",
			Name:        "Third-Party Management",
			Description: "Monitor third-party software dependencies",
			Category:    "System Monitoring",
			Severity:    "high",
			Checks:      []string{"component-licenses", "vulnerability-tracking"},
		},
		{
			ID:          "SOC2-CC8.1",
			Name:        "Change Tracking",
			Description: "Track changes to system components",
			Category:    "Change Management",
			Severity:    "medium",
			Checks:      []string{"timestamp", "tool-information"},
		},
	}

	// ISO 27001 Requirements
	cr.frameworks[FrameworkISO27001] = []ComplianceRequirement{
		{
			ID:          "ISO27001-A.8.9",
			Name:        "Configuration Management",
			Description: "Document and maintain configuration items",
			Category:    "Asset Management",
			Severity:    "high",
			Checks:      []string{"has-sbom", "has-components", "component-versions"},
		},
		{
			ID:          "ISO27001-A.12.6.1",
			Name:        "Technical Vulnerability Management",
			Description: "Identify and manage technical vulnerabilities",
			Category:    "Security Management",
			Severity:    "critical",
			Checks:      []string{"vulnerability-tracking", "component-licenses"},
		},
		{
			ID:          "ISO27001-A.14.2.1",
			Name:        "Secure Development Policy",
			Description: "Establish secure development practices",
			Category:    "Development",
			Severity:    "high",
			Checks:      []string{"build-metadata", "tool-information"},
		},
	}

	// SLSA Requirements
	cr.frameworks[FrameworkSLSA] = []ComplianceRequirement{
		{
			ID:          "SLSA-L1",
			Name:        "Build Scripted",
			Description: "Document build process",
			Category:    "Build",
			Severity:    "high",
			Checks:      []string{"has-sbom", "build-metadata", "tool-information"},
		},
		{
			ID:          "SLSA-L2",
			Name:        "Provenance",
			Description: "Provide build provenance",
			Category:    "Provenance",
			Severity:    "high",
			Checks:      []string{"provenance", "timestamp"},
		},
		{
			ID:          "SLSA-L3",
			Name:        "Supply Chain Transparency",
			Description: "Transparent supply chain",
			Category:    "Supply Chain",
			Severity:    "critical",
			Checks:      []string{"supply-chain-transparency", "has-components"},
		},
	}

	// SSDF v1.1 Requirements
	cr.frameworks[FrameworkSSDFv1] = []ComplianceRequirement{
		{
			ID:          "SSDF-PO.3.2",
			Name:        "SBOM Generation",
			Description: "Create and maintain SBOM",
			Category:    "Produce Well-Secured Software",
			Severity:    "high",
			Checks:      []string{"has-sbom", "sbom-format", "has-components"},
		},
		{
			ID:          "SSDF-PO.3.3",
			Name:        "Dependency Management",
			Description: "Archive and monitor dependencies",
			Category:    "Produce Well-Secured Software",
			Severity:    "high",
			Checks:      []string{"component-versions", "component-licenses", "vulnerability-tracking"},
		},
		{
			ID:          "SSDF-PS.1.1",
			Name:        "Build Environment",
			Description: "Secure build environment",
			Category:    "Protect Software",
			Severity:    "medium",
			Checks:      []string{"build-metadata", "tool-information"},
		},
		{
			ID:          "SSDF-RV.1.1",
			Name:        "Vulnerability Response",
			Description: "Identify and respond to vulnerabilities",
			Category:    "Respond to Vulnerabilities",
			Severity:    "critical",
			Checks:      []string{"vulnerability-tracking", "component-versions"},
		},
	}

	// CISA Requirements
	cr.frameworks[FrameworkCISA] = []ComplianceRequirement{
		{
			ID:          "CISA-SBOM-1",
			Name:        "SBOM Availability",
			Description: "SBOM must be available and accessible",
			Category:    "Availability",
			Severity:    "critical",
			Checks:      []string{"has-sbom", "sbom-format"},
		},
		{
			ID:          "CISA-SBOM-2",
			Name:        "Component Information",
			Description: "Complete component information required",
			Category:    "Completeness",
			Severity:    "high",
			Checks:      []string{"has-components", "component-versions", "component-licenses"},
		},
		{
			ID:          "CISA-SBOM-3",
			Name:        "Supply Chain Security",
			Description: "Transparent supply chain required",
			Category:    "Security",
			Severity:    "high",
			Checks:      []string{"supply-chain-transparency", "provenance"},
		},
	}
}

// FormatReportAsHTML generates an HTML report
func (cr *ComplianceReporter) FormatReportAsHTML(report *ComplianceReport) string {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<title>Compliance Report - %s</title>
<style>
body { font-family: Arial, sans-serif; margin: 20px; }
h1 { color: #333; }
.summary { background: #f0f0f0; padding: 15px; border-radius: 5px; margin: 20px 0; }
.status-compliant { color: green; font-weight: bold; }
.status-partial { color: orange; font-weight: bold; }
.status-non-compliant { color: red; font-weight: bold; }
table { width: 100%%; border-collapse: collapse; margin: 20px 0; }
th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
th { background-color: #4CAF50; color: white; }
.pass { background-color: #d4edda; }
.fail { background-color: #f8d7da; }
.partial { background-color: #fff3cd; }
.requirement { margin: 20px 0; padding: 15px; border: 1px solid #ddd; border-radius: 5px; }
</style>
</head>
<body>
<h1>Compliance Report: %s</h1>
<div class="summary">
<p><strong>Generated:</strong> %s</p>
<p><strong>SBOM:</strong> %s</p>
<p><strong>Status:</strong> <span class="status-%s">%s</span></p>
<p><strong>Passed:</strong> %d | <strong>Failed:</strong> %d | <strong>Partial:</strong> %d</p>
</div>
`, report.Framework, report.Framework, report.GeneratedAt.Format(time.RFC3339),
		report.SBOMPath, report.OverallStatus, strings.ToUpper(report.OverallStatus),
		report.PassedCount, report.FailedCount, report.PartialCount)

	html += "<h2>Requirements</h2>\n"

	for i, req := range report.Requirements {
		check := report.Checks[i]
		html += fmt.Sprintf(`<div class="requirement %s">
<h3>%s - %s</h3>
<p><strong>Category:</strong> %s | <strong>Severity:</strong> %s</p>
<p>%s</p>
<p><strong>Status:</strong> %s</p>
`, check.Status, req.ID, req.Name, req.Category, req.Severity, req.Description, check.Status)

		if len(check.Evidence) > 0 {
			html += "<p><strong>Evidence:</strong></p><ul>\n"
			for _, ev := range check.Evidence {
				html += fmt.Sprintf("<li>%s</li>\n", ev)
			}
			html += "</ul>\n"
		}

		if len(check.Issues) > 0 {
			html += "<p><strong>Issues:</strong></p><ul>\n"
			for _, issue := range check.Issues {
				html += fmt.Sprintf("<li>%s</li>\n", issue)
			}
			html += "</ul>\n"
		}

		if len(check.Recommendations) > 0 {
			html += "<p><strong>Recommendations:</strong></p><ul>\n"
			for _, rec := range check.Recommendations {
				html += fmt.Sprintf("<li>%s</li>\n", rec)
			}
			html += "</ul>\n"
		}

		html += "</div>\n"
	}

	html += "</body>\n</html>"
	return html
}
