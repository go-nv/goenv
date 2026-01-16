package sbom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// SnykScanner implements the Scanner interface for Snyk
type SnykScanner struct {
	apiToken string
	orgID    string
}

// NewSnykScanner creates a new Snyk scanner instance
func NewSnykScanner() *SnykScanner {
	return &SnykScanner{
		apiToken: os.Getenv("SNYK_TOKEN"),
		orgID:    os.Getenv("SNYK_ORG_ID"),
	}
}

// Name returns the scanner name
func (s *SnykScanner) Name() string {
	return "snyk"
}

// Version returns the Snyk CLI version
func (s *SnykScanner) Version() (string, error) {
	cmd := exec.Command("snyk", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get snyk version: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// IsInstalled checks if Snyk CLI is available
func (s *SnykScanner) IsInstalled() bool {
	_, err := exec.LookPath("snyk")
	return err == nil && s.apiToken != ""
}

// InstallationInstructions returns help for installing Snyk
func (s *SnykScanner) InstallationInstructions() string {
	return `Snyk CLI Installation:

1. Using goenv (recommended):
   goenv tools install snyk

2. Using npm:
   npm install -g snyk

3. Using Homebrew (macOS):
   brew install snyk/tap/snyk

4. Using binary:
   Download from https://github.com/snyk/cli/releases

5. Authenticate:
   snyk auth
   # Or set environment variables:
   export SNYK_TOKEN="your-api-token"
   export SNYK_ORG_ID="your-org-id"  # Optional but recommended

Get your API token from: https://app.snyk.io/account

Note: Snyk requires authentication to function.
`
}

// SupportsFormat checks if Snyk supports the given SBOM format
func (s *SnykScanner) SupportsFormat(format string) bool {
	supportedFormats := map[string]bool{
		"cyclonedx-json": true,
		"cyclonedx":      true,
		"spdx-json":      true,
		"spdx":           true,
	}
	return supportedFormats[format]
}

// Scan performs vulnerability scanning using Snyk
func (s *SnykScanner) Scan(ctx context.Context, opts *ScanOptions) (*ScanResult, error) {
	startTime := time.Now()

	// Validate authentication
	if s.apiToken == "" {
		return nil, NewScanError("snyk", "SNYK_TOKEN environment variable not set", nil)
	}

	// Read SBOM file
	sbomData, err := os.ReadFile(opts.SBOMPath)
	if err != nil {
		return nil, NewScanError("snyk", fmt.Sprintf("failed to read SBOM file: %s", opts.SBOMPath), err)
	}

	// Determine SBOM format
	sbomFormat := opts.Format
	if sbomFormat == "" {
		sbomFormat = detectSBOMFormat(sbomData)
	}

	// Test SBOM using Snyk CLI (for local testing)
	cliResult, cliErr := s.scanWithCLI(ctx, opts)

	// Also try API-based scanning for better results
	apiResult, apiErr := s.scanWithAPI(ctx, sbomData, sbomFormat, opts)

	// Prefer API result if available, fall back to CLI
	var result *ScanResult
	var scanErr error

	if apiResult != nil {
		result = apiResult
	} else if cliResult != nil {
		result = cliResult
	} else {
		// Both failed
		if apiErr != nil {
			scanErr = apiErr
		} else {
			scanErr = cliErr
		}
		return nil, NewScanError("snyk", "both CLI and API scanning failed", scanErr)
	}

	result.Metadata.ScanDuration = time.Since(startTime)
	return result, nil
}

// scanWithCLI uses Snyk CLI for scanning
func (s *SnykScanner) scanWithCLI(ctx context.Context, opts *ScanOptions) (*ScanResult, error) {
	args := []string{"sbom", "test"}

	// Add file path
	args = append(args, "--file", opts.SBOMPath)

	// Add JSON output
	args = append(args, "--json")

	// Add severity threshold if specified
	if opts.SeverityThreshold != "" {
		args = append(args, "--severity-threshold", opts.SeverityThreshold)
	}

	// Add org ID if available
	if s.orgID != "" {
		args = append(args, "--org", s.orgID)
	}

	// Additional args
	if len(opts.AdditionalArgs) > 0 {
		args = append(args, opts.AdditionalArgs...)
	}

	cmd := exec.CommandContext(ctx, "snyk", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("SNYK_TOKEN=%s", s.apiToken))

	output, err := cmd.CombinedOutput()

	// Snyk CLI may exit non-zero even on successful scan with vulnerabilities
	// Parse the output regardless
	if len(output) == 0 {
		return nil, NewScanError("snyk", "snyk CLI returned no output", err)
	}

	return s.parseSnykCLIOutput(output, opts)
}

// scanWithAPI uses Snyk REST API for scanning
func (s *SnykScanner) scanWithAPI(ctx context.Context, sbomData []byte, format string, opts *ScanOptions) (*ScanResult, error) {
	// Snyk API endpoint for SBOM testing
	apiURL := "https://api.snyk.io/rest/orgs/" + s.orgID + "/sbom_tests"
	if s.orgID == "" {
		// Try to get org ID from API if not set
		orgs, err := s.getOrganizations(ctx)
		if err != nil || len(orgs) == 0 {
			return nil, fmt.Errorf("SNYK_ORG_ID not set and could not fetch organizations")
		}
		s.orgID = orgs[0]
		apiURL = "https://api.snyk.io/rest/orgs/" + s.orgID + "/sbom_tests"
	}

	// Create request payload
	payload := map[string]interface{}{
		"data": map[string]interface{}{
			"type": "sbom_test",
			"attributes": map[string]interface{}{
				"sbom": map[string]interface{}{
					"format": format,
					"data":   string(sbomData),
				},
			},
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal API payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create API request: %w", err)
	}

	req.Header.Set("Authorization", "token "+s.apiToken)
	req.Header.Set("Content-Type", "application/vnd.api+json")
	req.Header.Set("Accept", "application/vnd.api+json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read API response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API returned error %d: %s", resp.StatusCode, string(body))
	}

	return s.parseSnykAPIOutput(body, opts)
}

// getOrganizations fetches user's Snyk organizations
func (s *SnykScanner) getOrganizations(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.snyk.io/rest/self/orgs", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "token "+s.apiToken)
	req.Header.Set("Accept", "application/vnd.api+json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var orgIDs []string
	for _, org := range result.Data {
		orgIDs = append(orgIDs, org.ID)
	}

	return orgIDs, nil
}

// parseSnykCLIOutput parses Snyk CLI JSON output
func (s *SnykScanner) parseSnykCLIOutput(output []byte, opts *ScanOptions) (*ScanResult, error) {
	var cliOutput struct {
		Vulnerabilities []struct {
			ID          string   `json:"id"`
			Title       string   `json:"title"`
			Severity    string   `json:"severity"`
			PackageName string   `json:"packageName"`
			Version     string   `json:"version"`
			From        []string `json:"from"`
			FixedIn     []string `json:"fixedIn"`
			CVSSScore   float64  `json:"cvssScore"`
			Description string   `json:"description"`
			References  []struct {
				URL string `json:"url"`
			} `json:"references"`
		} `json:"vulnerabilities"`
		Summary struct {
			Total    int `json:"total"`
			High     int `json:"high"`
			Medium   int `json:"medium"`
			Low      int `json:"low"`
			Critical int `json:"critical"`
		} `json:"summary"`
	}

	if err := json.Unmarshal(output, &cliOutput); err != nil {
		return nil, fmt.Errorf("failed to parse Snyk CLI output: %w", err)
	}

	version, _ := s.Version()
	result := &ScanResult{
		Scanner:         "snyk",
		ScannerVersion:  version,
		Timestamp:       time.Now(),
		SBOMPath:        opts.SBOMPath,
		SBOMFormat:      opts.Format,
		Vulnerabilities: make([]Vulnerability, 0),
	}

	// Convert vulnerabilities
	for _, v := range cliOutput.Vulnerabilities {
		vuln := Vulnerability{
			ID:             v.ID,
			PackageName:    v.PackageName,
			PackageVersion: v.Version,
			PackageType:    "go-module", // Default for Go projects
			Severity:       s.normalizeSeverity(v.Severity),
			CVSS:           v.CVSSScore,
			Description:    v.Description,
			VulnerablePath: v.From,
			FixAvailable:   len(v.FixedIn) > 0,
		}

		if len(v.FixedIn) > 0 {
			vuln.FixedInVersion = v.FixedIn[0]
		}

		for _, ref := range v.References {
			vuln.URLs = append(vuln.URLs, ref.URL)
		}

		result.Vulnerabilities = append(result.Vulnerabilities, vuln)
	}

	// Calculate summary
	result.Summary = s.calculateSummary(result.Vulnerabilities)

	return result, nil
}

// parseSnykAPIOutput parses Snyk API JSON response
func (s *SnykScanner) parseSnykAPIOutput(output []byte, opts *ScanOptions) (*ScanResult, error) {
	var apiResponse struct {
		Data struct {
			Attributes struct {
				Issues []struct {
					ID       string `json:"id"`
					Title    string `json:"title"`
					Severity string `json:"severity"`
					Package  struct {
						Name    string `json:"name"`
						Version string `json:"version"`
					} `json:"package"`
					FixInfo struct {
						IsFixable      bool     `json:"isFixable"`
						FixedInVersion []string `json:"fixedInVersions"`
					} `json:"fixInfo"`
					CVSSScore   float64  `json:"cvssScore"`
					Description string   `json:"description"`
					References  []string `json:"references"`
				} `json:"issues"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(output, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse Snyk API output: %w", err)
	}

	version, _ := s.Version()
	result := &ScanResult{
		Scanner:         "snyk",
		ScannerVersion:  version,
		Timestamp:       time.Now(),
		SBOMPath:        opts.SBOMPath,
		SBOMFormat:      opts.Format,
		Vulnerabilities: make([]Vulnerability, 0),
	}

	// Convert issues to vulnerabilities
	for _, issue := range apiResponse.Data.Attributes.Issues {
		vuln := Vulnerability{
			ID:             issue.ID,
			PackageName:    issue.Package.Name,
			PackageVersion: issue.Package.Version,
			PackageType:    "go-module",
			Severity:       s.normalizeSeverity(issue.Severity),
			CVSS:           issue.CVSSScore,
			Description:    issue.Description,
			URLs:           issue.References,
			FixAvailable:   issue.FixInfo.IsFixable,
		}

		if len(issue.FixInfo.FixedInVersion) > 0 {
			vuln.FixedInVersion = issue.FixInfo.FixedInVersion[0]
		}

		result.Vulnerabilities = append(result.Vulnerabilities, vuln)
	}

	// Calculate summary
	result.Summary = s.calculateSummary(result.Vulnerabilities)

	return result, nil
}

// normalizeSeverity converts Snyk severity to standard format
func (s *SnykScanner) normalizeSeverity(severity string) string {
	severity = toLower(severity)
	switch severity {
	case "critical":
		return "Critical"
	case "high":
		return "High"
	case "medium":
		return "Medium"
	case "low":
		return "Low"
	default:
		return "Unknown"
	}
}

// calculateSummary computes vulnerability summary statistics
func (s *SnykScanner) calculateSummary(vulns []Vulnerability) VulnerabilitySummary {
	summary := VulnerabilitySummary{
		Total: len(vulns),
	}

	for _, v := range vulns {
		switch v.Severity {
		case "Critical":
			summary.Critical++
		case "High":
			summary.High++
		case "Medium":
			summary.Medium++
		case "Low":
			summary.Low++
		case "Negligible":
			summary.Negligible++
		default:
			summary.Unknown++
		}

		if v.FixAvailable {
			summary.WithFix++
		} else {
			summary.WithoutFix++
		}
	}

	return summary
}

// detectSBOMFormat attempts to detect SBOM format from content
func detectSBOMFormat(data []byte) string {
	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return "unknown"
	}

	// Check for CycloneDX markers
	if bomFormat, ok := jsonData["bomFormat"]; ok && bomFormat == "CycloneDX" {
		return "cyclonedx-json"
	}

	// Check for SPDX markers
	if spdxVersion, ok := jsonData["spdxVersion"]; ok && spdxVersion != nil {
		return "spdx-json"
	}

	return "unknown"
}
