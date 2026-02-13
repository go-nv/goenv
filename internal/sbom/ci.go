package sbom

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CIPlatform represents different CI/CD platforms
type CIPlatform string

const (
	PlatformGitHubActions  CIPlatform = "github-actions"
	PlatformGitLabCI       CIPlatform = "gitlab-ci"
	PlatformCircleCI       CIPlatform = "circleci"
	PlatformJenkins        CIPlatform = "jenkins"
	PlatformAzurePipelines CIPlatform = "azure-pipelines"
	PlatformUnknown        CIPlatform = "unknown"
)

// CIChecker provides CI/CD-specific SBOM validation
type CIChecker struct {
	ProjectRoot string
	Platform    CIPlatform
}

// CICheckResult contains results of CI checks
type CICheckResult struct {
	Passed          bool                   `json:"passed"`
	SBOMExists      bool                   `json:"sbom_exists"`
	SBOMPath        string                 `json:"sbom_path,omitempty"`
	IsStale         bool                   `json:"is_stale"`
	StaleReason     string                 `json:"stale_reason,omitempty"`
	GoModModified   time.Time              `json:"go_mod_modified,omitempty"`
	SBOMModified    time.Time              `json:"sbom_modified,omitempty"`
	Recommendations []string               `json:"recommendations,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// CIScanResult contains results of vulnerability scanning
type CIScanResult struct {
	Scanner         string                 `json:"scanner"`
	ScanTime        time.Time              `json:"scan_time"`
	Passed          bool                   `json:"passed"`
	Summary         *VulnerabilitySummary  `json:"summary"`
	Vulnerabilities []Vulnerability        `json:"vulnerabilities"`
	FailureReason   string                 `json:"failure_reason,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// NewCIChecker creates a new CI checker
func NewCIChecker(projectRoot string) *CIChecker {
	if projectRoot == "" {
		cwd, _ := os.Getwd()
		projectRoot = cwd
	}

	return &CIChecker{
		ProjectRoot: projectRoot,
		Platform:    DetectCIPlatform(),
	}
}

// DetectCIPlatform detects the current CI/CD platform
func DetectCIPlatform() CIPlatform {
	// GitHub Actions
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		return PlatformGitHubActions
	}

	// GitLab CI
	if os.Getenv("GITLAB_CI") == "true" {
		return PlatformGitLabCI
	}

	// CircleCI
	if os.Getenv("CIRCLECI") == "true" {
		return PlatformCircleCI
	}

	// Jenkins
	if os.Getenv("JENKINS_HOME") != "" {
		return PlatformJenkins
	}

	// Azure Pipelines
	if os.Getenv("TF_BUILD") == "True" {
		return PlatformAzurePipelines
	}

	return PlatformUnknown
}

// CheckSBOM validates SBOM existence and staleness
func (c *CIChecker) CheckSBOM(sbomPath string, maxAge time.Duration) (*CICheckResult, error) {
	result := &CICheckResult{
		Metadata: make(map[string]interface{}),
	}

	// Find SBOM if not specified
	if sbomPath == "" {
		candidates := []string{
			"sbom.json",
			"sbom.cyclonedx.json",
			"sbom.spdx.json",
			"bom.json",
		}

		for _, candidate := range candidates {
			fullPath := filepath.Join(c.ProjectRoot, candidate)
			if _, err := os.Stat(fullPath); err == nil {
				sbomPath = fullPath
				break
			}
		}
	} else if !filepath.IsAbs(sbomPath) {
		sbomPath = filepath.Join(c.ProjectRoot, sbomPath)
	}

	// Check if SBOM exists
	sbomInfo, err := os.Stat(sbomPath)
	if os.IsNotExist(err) {
		result.SBOMExists = false
		result.Passed = false
		result.Recommendations = append(result.Recommendations,
			"Generate SBOM with: goenv sbom project")
		return result, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to stat SBOM: %w", err)
	}

	result.SBOMExists = true
	result.SBOMPath = sbomPath
	result.SBOMModified = sbomInfo.ModTime()

	// Check for go.mod and go.sum
	goModPath := filepath.Join(c.ProjectRoot, "go.mod")
	goSumPath := filepath.Join(c.ProjectRoot, "go.sum")

	goModInfo, goModErr := os.Stat(goModPath)
	goSumInfo, goSumErr := os.Stat(goSumPath)

	// Check if SBOM is stale
	if goModErr == nil {
		result.GoModModified = goModInfo.ModTime()
		if goModInfo.ModTime().After(sbomInfo.ModTime()) {
			result.IsStale = true
			result.StaleReason = "go.mod modified after SBOM generation"
			result.Recommendations = append(result.Recommendations,
				"Regenerate SBOM with: goenv sbom project")
		}
	}

	if goSumErr == nil && goSumInfo.ModTime().After(sbomInfo.ModTime()) {
		result.IsStale = true
		if result.StaleReason == "" {
			result.StaleReason = "go.sum modified after SBOM generation"
		} else {
			result.StaleReason += "; go.sum also modified"
		}
		if len(result.Recommendations) == 0 {
			result.Recommendations = append(result.Recommendations,
				"Regenerate SBOM with: goenv sbom project")
		}
	}

	// Check age if specified
	if maxAge > 0 {
		age := time.Since(sbomInfo.ModTime())
		if age > maxAge {
			result.IsStale = true
			if result.StaleReason == "" {
				result.StaleReason = fmt.Sprintf("SBOM is %v old (max age: %v)", age.Round(time.Hour), maxAge)
			} else {
				result.StaleReason += fmt.Sprintf("; also older than %v", maxAge)
			}
			if len(result.Recommendations) == 0 {
				result.Recommendations = append(result.Recommendations,
					"Regenerate SBOM with: goenv sbom project")
			}
		}
	}

	// Set passed status
	result.Passed = result.SBOMExists && !result.IsStale

	// Add CI platform metadata
	result.Metadata["ci_platform"] = string(c.Platform)
	result.Metadata["project_root"] = c.ProjectRoot

	return result, nil
}

// FormatCIOutput formats check results for CI platform
func (c *CIChecker) FormatCIOutput(result *CICheckResult) string {
	var buf strings.Builder

	switch c.Platform {
	case PlatformGitHubActions:
		if !result.Passed {
			if !result.SBOMExists {
				buf.WriteString("::error file=go.mod,title=SBOM Missing::SBOM file not found. Generate with: goenv sbom project\n")
			} else if result.IsStale {
				buf.WriteString(fmt.Sprintf("::error file=%s,title=SBOM Stale::%s\n",
					filepath.Base(result.SBOMPath), result.StaleReason))
			}
		} else {
			buf.WriteString(fmt.Sprintf("::notice file=%s,title=SBOM Valid::SBOM is up-to-date\n",
				filepath.Base(result.SBOMPath)))
		}

	case PlatformGitLabCI:
		if !result.Passed {
			buf.WriteString(fmt.Sprintf("❌ SBOM check failed: %s\n", result.StaleReason))
		} else {
			buf.WriteString("✅ SBOM check passed\n")
		}

	default:
		// Generic output
		if result.Passed {
			buf.WriteString("✅ SBOM Check: PASSED\n")
			buf.WriteString(fmt.Sprintf("   SBOM: %s\n", result.SBOMPath))
		} else {
			buf.WriteString("❌ SBOM Check: FAILED\n")
			if !result.SBOMExists {
				buf.WriteString("   SBOM file not found\n")
			} else {
				buf.WriteString(fmt.Sprintf("   Reason: %s\n", result.StaleReason))
			}
			if len(result.Recommendations) > 0 {
				buf.WriteString("   Recommendations:\n")
				for _, rec := range result.Recommendations {
					buf.WriteString(fmt.Sprintf("     - %s\n", rec))
				}
			}
		}
	}

	return buf.String()
}

// RunScanner runs a vulnerability scanner and formats output
func (c *CIChecker) RunScanner(sbomPath, scannerName string, options ScanOptions) (*CIScanResult, error) {
	result := &CIScanResult{
		Scanner:  scannerName,
		ScanTime: time.Now(),
		Metadata: make(map[string]interface{}),
	}

	// Get scanner
	scanner, err := GetScanner(scannerName)
	if err != nil {
		return nil, fmt.Errorf("failed to get scanner: %w", err)
	}

	// Run scan
	ctx := context.Background()
	scanResult, err := scanner.Scan(ctx, &options)
	if err != nil {
		result.Passed = false
		result.FailureReason = err.Error()
		return result, err
	}

	// Populate result
	result.Summary = &scanResult.Summary
	result.Vulnerabilities = scanResult.Vulnerabilities
	result.Metadata["scan_options"] = options
	result.Metadata["ci_platform"] = string(c.Platform)

	// Determine pass/fail based on threshold
	result.Passed = c.evaluateScanResult(scanResult, options.FailOn)

	return result, nil
}

// evaluateScanResult determines if scan passes based on thresholds
func (c *CIChecker) evaluateScanResult(scanResult *ScanResult, failOn string) bool {
	if scanResult.Summary.Total == 0 {
		return true
	}

	// If failOn threshold is set, fail if any vulnerabilities at or above that level
	threshold := strings.ToLower(failOn)

	switch threshold {
	case "critical":
		return scanResult.Summary.Critical == 0
	case "high":
		return scanResult.Summary.Critical == 0 && scanResult.Summary.High == 0
	case "medium":
		return scanResult.Summary.Critical == 0 && scanResult.Summary.High == 0 && scanResult.Summary.Medium == 0
	case "low":
		return scanResult.Summary.Critical == 0 && scanResult.Summary.High == 0 &&
			scanResult.Summary.Medium == 0 && scanResult.Summary.Low == 0
	default:
		// No threshold or "none" - always pass
		return true
	}
}

// FormatScanOutput formats scan results for CI platform
func (c *CIChecker) FormatScanOutput(result *CIScanResult) string {
	var buf strings.Builder

	switch c.Platform {
	case PlatformGitHubActions:
		if !result.Passed {
			buf.WriteString(fmt.Sprintf("::error title=Vulnerabilities Found::Found %d vulnerabilities\n",
				result.Summary.Total))
		}

		// Add annotations for each vulnerability
		for _, vuln := range result.Vulnerabilities {
			title := fmt.Sprintf("%s: %s", vuln.ID, vuln.PackageName)
			msg := vuln.Description
			if vuln.FixedInVersion != "" {
				msg += fmt.Sprintf(" (Fix: upgrade to %s)", vuln.FixedInVersion)
			}

			if vuln.Severity == "critical" || vuln.Severity == "high" {
				buf.WriteString(fmt.Sprintf("::error title=%s::%s\n", title, msg))
			} else if vuln.Severity == "medium" {
				buf.WriteString(fmt.Sprintf("::warning title=%s::%s\n", title, msg))
			} else {
				buf.WriteString(fmt.Sprintf("::notice title=%s::%s\n", title, msg))
			}
		}

	case PlatformGitLabCI:
		buf.WriteString("=== Vulnerability Scan Results ===\n")
		buf.WriteString(fmt.Sprintf("Scanner: %s\n", result.Scanner))
		buf.WriteString(fmt.Sprintf("Total: %d vulnerabilities\n", result.Summary.Total))
		buf.WriteString(fmt.Sprintf("  Critical: %d\n", result.Summary.Critical))
		buf.WriteString(fmt.Sprintf("  High: %d\n", result.Summary.High))
		buf.WriteString(fmt.Sprintf("  Medium: %d\n", result.Summary.Medium))
		buf.WriteString(fmt.Sprintf("  Low: %d\n", result.Summary.Low))

		if !result.Passed {
			buf.WriteString("\n❌ Scan FAILED: Vulnerabilities exceed threshold\n")
		} else {
			buf.WriteString("\n✅ Scan PASSED\n")
		}

	default:
		// Generic output
		buf.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		buf.WriteString("           Vulnerability Scan Results\n")
		buf.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		buf.WriteString(fmt.Sprintf("Scanner:     %s\n", result.Scanner))
		buf.WriteString(fmt.Sprintf("Scan Time:   %s\n", result.ScanTime.Format(time.RFC3339)))
		buf.WriteString(fmt.Sprintf("Total Found: %d vulnerabilities\n", result.Summary.Total))
		buf.WriteString("\nBy Severity:\n")
		buf.WriteString(fmt.Sprintf("  🔴 Critical: %d\n", result.Summary.Critical))
		buf.WriteString(fmt.Sprintf("  🟠 High:     %d\n", result.Summary.High))
		buf.WriteString(fmt.Sprintf("  🟡 Medium:   %d\n", result.Summary.Medium))
		buf.WriteString(fmt.Sprintf("  🔵 Low:      %d\n", result.Summary.Low))
		buf.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

		if result.Passed {
			buf.WriteString("✅ Result: PASSED\n")
		} else {
			buf.WriteString("❌ Result: FAILED\n")
		}
	}

	return buf.String()
}

// WriteScanResultToFile writes scan result to a file
func (c *CIChecker) WriteScanResultToFile(result *CIScanResult, outputPath string) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write result: %w", err)
	}

	return nil
}

// ExportToGitHubSARIF exports vulnerabilities to GitHub SARIF format
func (c *CIChecker) ExportToGitHubSARIF(result *CIScanResult, outputPath string) error {
	// SARIF format for GitHub Code Scanning
	sarif := map[string]interface{}{
		"version": "2.1.0",
		"$schema": "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		"runs": []map[string]interface{}{
			{
				"tool": map[string]interface{}{
					"driver": map[string]interface{}{
						"name":           result.Scanner,
						"informationUri": "https://github.com/go-nv/goenv",
						"version":        "1.0.0",
					},
				},
				"results": c.convertToSARIFResults(result.Vulnerabilities),
			},
		},
	}

	data, err := json.MarshalIndent(sarif, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal SARIF: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write SARIF: %w", err)
	}

	return nil
}

// convertToSARIFResults converts vulnerabilities to SARIF format
func (c *CIChecker) convertToSARIFResults(vulns []Vulnerability) []map[string]interface{} {
	results := make([]map[string]interface{}, 0, len(vulns))

	for _, vuln := range vulns {
		level := "warning"
		if vuln.Severity == "critical" {
			level = "error"
		} else if vuln.Severity == "low" || vuln.Severity == "negligible" {
			level = "note"
		}

		result := map[string]interface{}{
			"ruleId": vuln.ID,
			"level":  level,
			"message": map[string]interface{}{
				"text": fmt.Sprintf("%s in %s %s", vuln.Description, vuln.PackageName, vuln.PackageVersion),
			},
			"locations": []map[string]interface{}{
				{
					"physicalLocation": map[string]interface{}{
						"artifactLocation": map[string]interface{}{
							"uri": "go.mod",
						},
					},
				},
			},
		}

		if vuln.FixedInVersion != "" {
			result["fixes"] = []map[string]interface{}{
				{
					"description": map[string]interface{}{
						"text": fmt.Sprintf("Upgrade to version %s", vuln.FixedInVersion),
					},
				},
			}
		}

		results = append(results, result)
	}

	return results
}
