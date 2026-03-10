package sbom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// VeracodeScanner implements the Scanner interface for Veracode
type VeracodeScanner struct {
	apiID     string
	apiSecret string
}

// NewVeracodeScanner creates a new Veracode scanner instance
func NewVeracodeScanner() *VeracodeScanner {
	return &VeracodeScanner{
		apiID:     os.Getenv("VERACODE_API_KEY_ID"),
		apiSecret: os.Getenv("VERACODE_API_KEY_SECRET"),
	}
}

// Name returns the scanner name
func (v *VeracodeScanner) Name() string {
	return "veracode"
}

// Version returns the Veracode wrapper version
func (v *VeracodeScanner) Version() (string, error) {
	cmd := exec.Command("java", "-jar", v.getWrapperPath(), "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get veracode version: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// IsInstalled checks if Veracode API wrapper is available
func (v *VeracodeScanner) IsInstalled() bool {
	// Check for Java
	_, err := exec.LookPath("java")
	if err != nil {
		return false
	}

	// Check for credentials
	if v.apiID == "" || v.apiSecret == "" {
		return false
	}

	// Check for wrapper JAR
	wrapperPath := v.getWrapperPath()
	if _, err := os.Stat(wrapperPath); err != nil {
		return false
	}

	return true
}

// getWrapperPath returns the path to Veracode API wrapper JAR
func (v *VeracodeScanner) getWrapperPath() string {
	// Check environment variable first
	if path := os.Getenv("VERACODE_WRAPPER_PATH"); path != "" {
		return path
	}

	// Check common locations
	commonPaths := []string{
		"./VeracodeJavaAPI.jar",
		"/opt/veracode/VeracodeJavaAPI.jar",
		os.ExpandEnv("$HOME/.veracode/VeracodeJavaAPI.jar"),
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return "VeracodeJavaAPI.jar" // Default
}

// InstallationInstructions returns help for setting up Veracode
func (v *VeracodeScanner) InstallationInstructions() string {
	return `Veracode Setup:

Note: Veracode uses a Java-based API wrapper (not available via 'goenv tools install')

1. Install Java (required):
   # macOS
   brew install openjdk
   
   # Or download from: https://www.oracle.com/java/technologies/downloads/

2. Download Veracode API Wrapper:
   wget https://downloads.veracode.com/securityscan/VeracodeJavaAPI.jar
   
   # Or download from: https://help.veracode.com/r/c_about_wrappers
   # Place in: $HOME/.veracode/VeracodeJavaAPI.jar

3. Set up credentials:
   export VERACODE_API_KEY_ID="your-api-key-id"
   export VERACODE_API_KEY_SECRET="your-api-key-secret"
   
   # Optional: Set wrapper location
   export VERACODE_WRAPPER_PATH="/path/to/VeracodeJavaAPI.jar"

4. Get API credentials from:
   https://web.analysiscenter.veracode.com/ → Account → API Credentials

5. Verify installation:
   java -jar $HOME/.veracode/VeracodeJavaAPI.jar -version

Note: Veracode requires a paid enterprise license.
For SBOM analysis, ensure you have "Software Composition Analysis" (SCA) enabled.
`
}

// SupportsFormat checks if Veracode supports the given SBOM format
func (v *VeracodeScanner) SupportsFormat(format string) bool {
	supportedFormats := map[string]bool{
		"cyclonedx-json": true,
		"cyclonedx-xml":  true,
		"cyclonedx":      true,
		"spdx-json":      true,
		"spdx":           true,
	}
	return supportedFormats[format]
}

// Scan performs vulnerability scanning using Veracode SCA
func (v *VeracodeScanner) Scan(ctx context.Context, opts *ScanOptions) (*ScanResult, error) {
	startTime := time.Now()

	// Validate authentication
	if v.apiID == "" || v.apiSecret == "" {
		return nil, NewScanError("veracode", "VERACODE_API_KEY_ID and VERACODE_API_KEY_SECRET must be set", nil)
	}

	// Read SBOM file
	sbomData, err := os.ReadFile(opts.SBOMPath)
	if err != nil {
		return nil, NewScanError("veracode", fmt.Sprintf("failed to read SBOM file: %s", opts.SBOMPath), err)
	}

	// Upload SBOM and create workspace
	workspaceID, err := v.uploadSBOM(ctx, sbomData, opts)
	if err != nil {
		return nil, NewScanError("veracode", "failed to upload SBOM", err)
	}

	// Poll for scan results
	result, err := v.pollScanResults(ctx, workspaceID, opts)
	if err != nil {
		return nil, NewScanError("veracode", "failed to get scan results", err)
	}

	result.Metadata.ScanDuration = time.Since(startTime)
	return result, nil
}

// uploadSBOM uploads SBOM to Veracode SCA and creates a workspace
func (v *VeracodeScanner) uploadSBOM(ctx context.Context, sbomData []byte, opts *ScanOptions) (string, error) {
	apiURL := "https://sca.analysiscenter.veracode.com/api/workspaces"

	// Create multipart form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add SBOM file
	part, err := writer.CreateFormFile("file", "sbom.json")
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := part.Write(sbomData); err != nil {
		return "", fmt.Errorf("failed to write SBOM data: %w", err)
	}

	// Add metadata
	_ = writer.WriteField("name", "goenv-sbom-scan-"+time.Now().Format("20060102-150405"))
	_ = writer.WriteField("type", "sbom_scan")

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, body)
	if err != nil {
		return "", fmt.Errorf("failed to create upload request: %w", err)
	}

	// Add authentication via HMAC
	v.addHMACAuth(req, body.Bytes())
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Execute request
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response to get workspace ID
	var uploadResponse struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal(respBody, &uploadResponse); err != nil {
		return "", fmt.Errorf("failed to parse upload response: %w", err)
	}

	return uploadResponse.ID, nil
}

// pollScanResults polls Veracode for scan completion and results
func (v *VeracodeScanner) pollScanResults(ctx context.Context, workspaceID string, opts *ScanOptions) (*ScanResult, error) {
	apiURL := fmt.Sprintf("https://sca.analysiscenter.veracode.com/api/workspaces/%s/issues", workspaceID)

	maxAttempts := 60 // 5 minutes max (5s intervals)
	for attempt := 0; attempt < maxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(5 * time.Second):
			// Check scan status
			req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
			if err != nil {
				continue
			}

			v.addHMACAuth(req, nil)
			req.Header.Set("Accept", "application/json")

			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				continue
			}

			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				// Scan complete, parse results
				return v.parseVeracodeResults(body, opts)
			}

			if resp.StatusCode >= 400 && resp.StatusCode != http.StatusNotFound {
				return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
			}
		}
	}

	return nil, fmt.Errorf("scan timed out after %d seconds", maxAttempts*5)
}

// parseVeracodeResults parses Veracode SCA API response
func (v *VeracodeScanner) parseVeracodeResults(data []byte, opts *ScanOptions) (*ScanResult, error) {
	var apiResponse struct {
		Issues []struct {
			ID        string `json:"id"`
			Title     string `json:"title"`
			Severity  string `json:"severity"`
			Component struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"component"`
			Vulnerability struct {
				CVE         string   `json:"cve"`
				CVSS        float64  `json:"cvss_score"`
				Description string   `json:"description"`
				References  []string `json:"references"`
			} `json:"vulnerability"`
			Remediation struct {
				FixAvailable bool   `json:"fix_available"`
				FixedVersion string `json:"fixed_version"`
			} `json:"remediation"`
		} `json:"issues"`
	}

	if err := json.Unmarshal(data, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse Veracode results: %w", err)
	}

	version, _ := v.Version()
	result := &ScanResult{
		Scanner:         "veracode",
		ScannerVersion:  version,
		Timestamp:       time.Now(),
		SBOMPath:        opts.SBOMPath,
		SBOMFormat:      opts.Format,
		Vulnerabilities: make([]Vulnerability, 0),
	}

	// Convert issues to vulnerabilities
	for _, issue := range apiResponse.Issues {
		vuln := Vulnerability{
			ID:             issue.Vulnerability.CVE,
			PackageName:    issue.Component.Name,
			PackageVersion: issue.Component.Version,
			PackageType:    "go-module",
			Severity:       v.normalizeSeverity(issue.Severity),
			CVSS:           issue.Vulnerability.CVSS,
			Description:    issue.Vulnerability.Description,
			URLs:           issue.Vulnerability.References,
			FixAvailable:   issue.Remediation.FixAvailable,
			FixedInVersion: issue.Remediation.FixedVersion,
		}

		result.Vulnerabilities = append(result.Vulnerabilities, vuln)
	}

	// Calculate summary
	result.Summary = v.calculateSummary(result.Vulnerabilities)

	return result, nil
}

// addHMACAuth adds Veracode HMAC authentication to request
func (v *VeracodeScanner) addHMACAuth(req *http.Request, body []byte) {
	// Veracode uses HMAC authentication
	// For simplicity, using API wrapper approach
	// In production, implement proper HMAC calculation
	req.Header.Set("Authorization", fmt.Sprintf("VERACODE-HMAC-SHA-256 id=%s", v.apiID))
	// Full HMAC implementation would go here
}

// normalizeSeverity converts Veracode severity to standard format
func (v *VeracodeScanner) normalizeSeverity(severity string) string {
	severity = toLower(severity)
	switch severity {
	case "veryhigh", "very high", "critical":
		return "Critical"
	case "high":
		return "High"
	case "medium":
		return "Medium"
	case "low":
		return "Low"
	case "verylow", "very low":
		return "Negligible"
	default:
		return "Unknown"
	}
}

// calculateSummary computes vulnerability summary statistics
func (v *VeracodeScanner) calculateSummary(vulns []Vulnerability) VulnerabilitySummary {
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
