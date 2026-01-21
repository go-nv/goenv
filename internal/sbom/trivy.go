package sbom

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// TrivyScanner implements the Scanner interface for Aqua Security Trivy
type TrivyScanner struct{}

// NewTrivyScanner creates a new Trivy scanner instance
func NewTrivyScanner() *TrivyScanner {
	return &TrivyScanner{}
}

// Name returns the scanner name
func (t *TrivyScanner) Name() string {
	return "trivy"
}

// Version returns the Trivy version
func (t *TrivyScanner) Version() (string, error) {
	cmd := exec.Command("trivy", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", NewScanError("trivy", "failed to get version", err)
	}

	// Parse version from output (format: "Version: 0.48.0")
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Version:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1], nil
			}
		}
	}

	return strings.TrimSpace(string(output)), nil
}

// IsInstalled checks if Trivy is available
func (t *TrivyScanner) IsInstalled() bool {
	_, err := exec.LookPath("trivy")
	return err == nil
}

// InstallationInstructions returns help text for installing Trivy
func (t *TrivyScanner) InstallationInstructions() string {
	return `Trivy installation options:

1. Using goenv tools (recommended):
   goenv tools install trivy

2. Using Homebrew (macOS/Linux):
   brew install trivy

3. Using script:
   curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin

4. Using Docker:
   docker pull aquasec/trivy:latest

5. Download binary:
   https://github.com/aquasecurity/trivy/releases

For more information: https://github.com/aquasecurity/trivy`
}

// SupportsFormat checks if Trivy supports the given SBOM format
func (t *TrivyScanner) SupportsFormat(format string) bool {
	supported := map[string]bool{
		"cyclonedx-json": true,
		"cyclonedx":      true,
		"spdx-json":      true,
		"spdx":           true,
		"syft-json":      true,
	}
	return supported[format]
}

// Scan performs vulnerability scanning using Trivy
func (t *TrivyScanner) Scan(ctx context.Context, opts *ScanOptions) (*ScanResult, error) {
	startTime := time.Now()

	// Validate options
	if opts.SBOMPath == "" {
		return nil, NewScanError("trivy", "SBOM path is required", nil)
	}

	if !t.IsInstalled() {
		return nil, NewScanError("trivy", "trivy is not installed", nil)
	}

	// Check SBOM file exists
	if _, err := os.Stat(opts.SBOMPath); err != nil {
		return nil, NewScanError("trivy", fmt.Sprintf("SBOM file not found: %s", opts.SBOMPath), err)
	}

	// Build trivy command
	args := t.buildTrivyArgs(opts)

	cmd := exec.CommandContext(ctx, "trivy", args...)
	cmd.Env = append(os.Environ(), t.buildTrivyEnv(opts)...)

	// Capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Trivy exits with non-zero if vulnerabilities found, but still produces JSON output
		if _, ok := err.(*exec.ExitError); !ok {
			return nil, NewScanError("trivy", "failed to run trivy", err)
		}
		// Continue processing output even with non-zero exit
	}

	// Parse results
	result, err := t.parseTrivyOutput(output, opts)
	if err != nil {
		return nil, err
	}

	// Get scanner version
	version, _ := t.Version()
	result.Scanner = t.Name()
	result.ScannerVersion = version
	result.Timestamp = time.Now()
	result.SBOMPath = opts.SBOMPath
	result.SBOMFormat = opts.Format
	result.Metadata.ScanDuration = time.Since(startTime)

	return result, nil
}

// buildTrivyArgs constructs command-line arguments for Trivy
func (t *TrivyScanner) buildTrivyArgs(opts *ScanOptions) []string {
	args := []string{
		"sbom",
		opts.SBOMPath,
	}

	// Output format (default to JSON for parsing)
	outputFormat := "json"
	if opts.OutputFormat != "" {
		outputFormat = opts.OutputFormat
	}
	args = append(args, "--format", outputFormat)

	// Output file
	if opts.OutputPath != "" {
		args = append(args, "--output", opts.OutputPath)
	}

	// Severity threshold
	if opts.SeverityThreshold != "" {
		args = append(args, "--severity", strings.ToUpper(opts.SeverityThreshold))
	}

	// Exit code behavior
	if opts.FailOn != "" {
		args = append(args, "--exit-code", "1")
	}

	// Offline mode
	if opts.Offline {
		args = append(args, "--skip-db-update")
	}

	// Only show fixed vulnerabilities (using ignore-unfixed)
	if opts.OnlyFixed {
		args = append(args, "--ignore-unfixed")
	}

	// Verbose output
	if opts.Verbose {
		args = append(args, "--debug")
	}

	// Quiet mode for cleaner output
	if !opts.Verbose {
		args = append(args, "--quiet")
	}

	// Additional arguments
	args = append(args, opts.AdditionalArgs...)

	return args
}

// buildTrivyEnv constructs environment variables for Trivy
func (t *TrivyScanner) buildTrivyEnv(opts *ScanOptions) []string {
	env := []string{}

	// Disable automatic DB updates in offline mode
	if opts.Offline {
		env = append(env, "TRIVY_SKIP_DB_UPDATE=true")
	}

	return env
}

// parseTrivyOutput parses Trivy JSON output into ScanResult
func (t *TrivyScanner) parseTrivyOutput(data []byte, opts *ScanOptions) (*ScanResult, error) {
	var trivyResult struct {
		SchemaVersion int    `json:"SchemaVersion"`
		ArtifactName  string `json:"ArtifactName"`
		ArtifactType  string `json:"ArtifactType"`
		Metadata      struct {
			ImageConfig struct {
				Architecture string `json:"architecture"`
				OS           string `json:"os"`
			} `json:"ImageConfig"`
		} `json:"Metadata"`
		Results []struct {
			Target          string `json:"Target"`
			Class           string `json:"Class"`
			Type            string `json:"Type"`
			Vulnerabilities []struct {
				VulnerabilityID  string `json:"VulnerabilityID"`
				PkgName          string `json:"PkgName"`
				PkgID            string `json:"PkgID"`
				InstalledVersion string `json:"InstalledVersion"`
				FixedVersion     string `json:"FixedVersion"`
				Status           string `json:"Status"`
				Layer            struct {
					Digest string `json:"Digest"`
					DiffID string `json:"DiffID"`
				} `json:"Layer"`
				SeveritySource string `json:"SeveritySource"`
				PrimaryURL     string `json:"PrimaryURL"`
				DataSource     struct {
					ID   string `json:"ID"`
					Name string `json:"Name"`
					URL  string `json:"URL"`
				} `json:"DataSource"`
				Title       string   `json:"Title"`
				Description string   `json:"Description"`
				Severity    string   `json:"Severity"`
				CweIDs      []string `json:"CweIDs"`
				CVSS        map[string]struct {
					V2Vector string  `json:"V2Vector"`
					V3Vector string  `json:"V3Vector"`
					V2Score  float64 `json:"V2Score"`
					V3Score  float64 `json:"V3Score"`
				} `json:"CVSS"`
				References       []string `json:"References"`
				PublishedDate    string   `json:"PublishedDate"`
				LastModifiedDate string   `json:"LastModifiedDate"`
			} `json:"Vulnerabilities"`
		} `json:"Results"`
	}

	if err := json.Unmarshal(data, &trivyResult); err != nil {
		return nil, NewScanError("trivy", "failed to parse JSON output", err)
	}

	result := &ScanResult{
		Vulnerabilities: make([]Vulnerability, 0),
		Summary:         VulnerabilitySummary{},
		Metadata:        ScanMetadata{},
	}

	// Parse vulnerabilities from all results
	for _, res := range trivyResult.Results {
		for _, tv := range res.Vulnerabilities {
			vuln := Vulnerability{
				ID:             tv.VulnerabilityID,
				PackageName:    tv.PkgName,
				PackageVersion: tv.InstalledVersion,
				PackageType:    t.mapPackageType(res.Type),
				Severity:       t.normalizeSeverity(tv.Severity),
				Description:    tv.Description,
				URLs:           tv.References,
				PublishedAt:    tv.PublishedDate,
				ModifiedAt:     tv.LastModifiedDate,
				FixedInVersion: tv.FixedVersion,
				FixAvailable:   tv.FixedVersion != "",
				Metadata: map[string]interface{}{
					"title":      tv.Title,
					"dataSource": tv.DataSource,
					"cweIDs":     tv.CweIDs,
					"target":     res.Target,
					"status":     tv.Status,
				},
			}

			// Extract CVSS score (prefer V3 over V2)
			if len(tv.CVSS) > 0 {
				for _, cvss := range tv.CVSS {
					if cvss.V3Score > 0 {
						vuln.CVSS = cvss.V3Score
						break
					} else if cvss.V2Score > 0 {
						vuln.CVSS = cvss.V2Score
					}
				}
			}

			// Add primary URL if available
			if tv.PrimaryURL != "" {
				vuln.URLs = append([]string{tv.PrimaryURL}, vuln.URLs...)
			}

			result.Vulnerabilities = append(result.Vulnerabilities, vuln)

			// Update summary
			result.Summary.Total++
			switch ParseSeverity(vuln.Severity) {
			case SeverityCritical:
				result.Summary.Critical++
			case SeverityHigh:
				result.Summary.High++
			case SeverityMedium:
				result.Summary.Medium++
			case SeverityLow:
				result.Summary.Low++
			case SeverityNegligible:
				result.Summary.Negligible++
			default:
				result.Summary.Unknown++
			}

			if vuln.FixAvailable {
				result.Summary.WithFix++
			} else {
				result.Summary.WithoutFix++
			}
		}
	}

	return result, nil
}

// mapPackageType maps Trivy package types to our internal types
func (t *TrivyScanner) mapPackageType(trivyType string) string {
	switch trivyType {
	case "gomod", "gobinary":
		return "go-module"
	case "go":
		return "go-module"
	default:
		return trivyType
	}
}

// normalizeSeverity normalizes severity strings to standard values
func (t *TrivyScanner) normalizeSeverity(severity string) string {
	severity = strings.ToLower(severity)
	switch severity {
	case "critical":
		return "Critical"
	case "high":
		return "High"
	case "medium":
		return "Medium"
	case "low":
		return "Low"
	case "unknown":
		return "Negligible"
	default:
		return "Unknown"
	}
}

// WriteTrivyConfig writes a Trivy configuration file for optimized Go scanning
func WriteTrivyConfig(path string) error {
	config := `# Trivy configuration for Go projects
format: json
output: trivy-results.json

# Vulnerability settings
severity:
  - UNKNOWN
  - LOW
  - MEDIUM
  - HIGH
  - CRITICAL

# Scan settings
scan:
  skip-dirs:
    - vendor
    - node_modules
  skip-files:
    - "*.test"

# Output settings
output:
  format: json
  
# Cache settings
cache:
  dir: /tmp/trivy-cache
  
# Database settings
db:
  skip-update: false
  
# Vulnerability settings
vulnerability:
  type: os,library
  
# Ignore unfixed vulnerabilities
ignore-unfixed: false

# Exit code
exit-code: 0
`

	if err := os.WriteFile(path, []byte(config), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetTrivyDatabaseInfo returns information about the Trivy vulnerability database
func GetTrivyDatabaseInfo() (map[string]interface{}, error) {
	cmd := exec.Command("trivy", "image", "--download-db-only", "--skip-update")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to check database: %w", err)
	}

	// Get cache directory
	cacheDir := filepath.Join(os.TempDir(), "trivy")
	if xdgCache := os.Getenv("XDG_CACHE_HOME"); xdgCache != "" {
		cacheDir = filepath.Join(xdgCache, "trivy")
	}

	info := map[string]interface{}{
		"cacheDir": cacheDir,
	}

	// Check if database exists
	dbPath := filepath.Join(cacheDir, "db", "trivy.db")
	if stat, err := os.Stat(dbPath); err == nil {
		info["lastUpdated"] = stat.ModTime()
		info["size"] = stat.Size()
	}

	return info, nil
}
