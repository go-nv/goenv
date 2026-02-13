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

// GrypeScanner implements the Scanner interface for Anchore Grype
type GrypeScanner struct{}

// NewGrypeScanner creates a new Grype scanner instance
func NewGrypeScanner() *GrypeScanner {
	return &GrypeScanner{}
}

// Name returns the scanner name
func (g *GrypeScanner) Name() string {
	return "grype"
}

// Version returns the Grype version
func (g *GrypeScanner) Version() (string, error) {
	cmd := exec.Command("grype", "version")
	output, err := cmd.Output()
	if err != nil {
		return "", NewScanError("grype", "failed to get version", err)
	}

	// Parse version from output (format: "grype 0.74.0")
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Version:") || strings.HasPrefix(line, "Application:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[len(parts)-1], nil
			}
		}
	}

	return strings.TrimSpace(string(output)), nil
}

// IsInstalled checks if Grype is available
func (g *GrypeScanner) IsInstalled() bool {
	_, err := exec.LookPath("grype")
	return err == nil
}

// InstallationInstructions returns help text for installing Grype
func (g *GrypeScanner) InstallationInstructions() string {
	return `Grype installation options:

1. Using goenv tools (recommended):
   goenv tools install grype

2. Using Homebrew (macOS/Linux):
   brew install grype

3. Using script:
   curl -sSfL https://raw.githubusercontent.com/anchore/grype/main/install.sh | sh -s -- -b /usr/local/bin

4. Download binary:
   https://github.com/anchore/grype/releases

For more information: https://github.com/anchore/grype`
}

// SupportsFormat checks if Grype supports the given SBOM format
func (g *GrypeScanner) SupportsFormat(format string) bool {
	supported := map[string]bool{
		"cyclonedx-json": true,
		"cyclonedx-xml":  true,
		"spdx-json":      true,
		"spdx-tag-value": true,
		"syft-json":      true,
	}
	return supported[format]
}

// Scan performs vulnerability scanning using Grype
func (g *GrypeScanner) Scan(ctx context.Context, opts *ScanOptions) (*ScanResult, error) {
	startTime := time.Now()

	// Validate options
	if opts.SBOMPath == "" {
		return nil, NewScanError("grype", "SBOM path is required", nil)
	}

	if !g.IsInstalled() {
		return nil, NewScanError("grype", "grype is not installed", nil)
	}

	// Check SBOM file exists
	if _, err := os.Stat(opts.SBOMPath); err != nil {
		return nil, NewScanError("grype", fmt.Sprintf("SBOM file not found: %s", opts.SBOMPath), err)
	}

	// Build grype command
	args := g.buildGrypeArgs(opts)

	cmd := exec.CommandContext(ctx, "grype", args...)
	cmd.Env = append(os.Environ(), g.buildGrypeEnv(opts)...)

	// Capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Grype exits with non-zero if vulnerabilities found, but still produces JSON output
		if _, ok := err.(*exec.ExitError); !ok {
			return nil, NewScanError("grype", "failed to run grype", err)
		}
		// Continue processing output even with non-zero exit
	}

	// Parse results
	result, err := g.parseGrypeOutput(output, opts)
	if err != nil {
		return nil, err
	}

	// Get scanner version
	version, _ := g.Version()
	result.Scanner = g.Name()
	result.ScannerVersion = version
	result.Timestamp = time.Now()
	result.SBOMPath = opts.SBOMPath
	result.SBOMFormat = opts.Format
	result.Metadata.ScanDuration = time.Since(startTime)

	return result, nil
}

// buildGrypeArgs constructs command-line arguments for Grype
func (g *GrypeScanner) buildGrypeArgs(opts *ScanOptions) []string {
	args := []string{
		fmt.Sprintf("sbom:%s", opts.SBOMPath),
	}

	// Output format (default to JSON for parsing)
	outputFormat := "json"
	if opts.OutputFormat != "" {
		outputFormat = opts.OutputFormat
	}
	args = append(args, "-o", outputFormat)

	// Output file
	if opts.OutputPath != "" {
		args = append(args, "--file", opts.OutputPath)
	}

	// Severity threshold
	if opts.SeverityThreshold != "" {
		args = append(args, "--fail-on", opts.SeverityThreshold)
	} else if opts.FailOn != "" {
		args = append(args, "--fail-on", opts.FailOn)
	}

	// Offline mode
	if opts.Offline {
		args = append(args, "--db-auto-update=false")
	}

	// Only show fixed vulnerabilities
	if opts.OnlyFixed {
		args = append(args, "--only-fixed")
	}

	// Verbose output
	if opts.Verbose {
		args = append(args, "-v")
	}

	// Additional arguments
	args = append(args, opts.AdditionalArgs...)

	return args
}

// buildGrypeEnv constructs environment variables for Grype
func (g *GrypeScanner) buildGrypeEnv(opts *ScanOptions) []string {
	env := []string{}

	// Disable automatic DB updates in offline mode
	if opts.Offline {
		env = append(env, "GRYPE_DB_AUTO_UPDATE=false")
	}

	return env
}

// parseGrypeOutput parses Grype JSON output into ScanResult
func (g *GrypeScanner) parseGrypeOutput(data []byte, opts *ScanOptions) (*ScanResult, error) {
	var grypeResult struct {
		Matches []struct {
			Vulnerability struct {
				ID          string   `json:"id"`
				DataSource  string   `json:"dataSource"`
				Severity    string   `json:"severity"`
				Description string   `json:"description"`
				URLs        []string `json:"urls"`
				CVSS        []struct {
					Version string `json:"version"`
					Vector  string `json:"vector"`
					Metrics struct {
						BaseScore float64 `json:"baseScore"`
					} `json:"metrics"`
				} `json:"cvss"`
				Fix struct {
					Versions []string `json:"versions"`
					State    string   `json:"state"`
				} `json:"fix"`
			} `json:"vulnerability"`
			RelatedVulnerabilities []struct {
				ID         string `json:"id"`
				DataSource string `json:"dataSource"`
			} `json:"relatedVulnerabilities"`
			MatchDetails []struct {
				Type       string                 `json:"type"`
				Matcher    string                 `json:"matcher"`
				SearchedBy map[string]interface{} `json:"searchedBy"`
				Found      map[string]interface{} `json:"found"`
			} `json:"matchDetails"`
			Artifact struct {
				Name      string `json:"name"`
				Version   string `json:"version"`
				Type      string `json:"type"`
				Locations []struct {
					Path string `json:"path"`
				} `json:"locations"`
				Language string `json:"language"`
				PURL     string `json:"purl"`
			} `json:"artifact"`
		} `json:"matches"`
		Source struct {
			Type   string `json:"type"`
			Target string `json:"target"`
		} `json:"source"`
		Descriptor struct {
			Name    string `json:"name"`
			Version string `json:"version"`
			DB      struct {
				Built         string `json:"built"`
				SchemaVersion int    `json:"schemaVersion"`
				Location      string `json:"location"`
				Checksum      string `json:"checksum"`
			} `json:"db"`
		} `json:"descriptor"`
	}

	if err := json.Unmarshal(data, &grypeResult); err != nil {
		return nil, NewScanError("grype", "failed to parse JSON output", err)
	}

	result := &ScanResult{
		Vulnerabilities: make([]Vulnerability, 0, len(grypeResult.Matches)),
		Summary:         VulnerabilitySummary{},
		Metadata:        ScanMetadata{},
	}

	// Parse vulnerabilities
	for _, match := range grypeResult.Matches {
		vuln := Vulnerability{
			ID:             match.Vulnerability.ID,
			PackageName:    match.Artifact.Name,
			PackageVersion: match.Artifact.Version,
			PackageType:    g.mapPackageType(match.Artifact.Type, match.Artifact.Language),
			Severity:       g.normalizeSeverity(match.Vulnerability.Severity),
			Description:    match.Vulnerability.Description,
			URLs:           match.Vulnerability.URLs,
			FixAvailable:   len(match.Vulnerability.Fix.Versions) > 0,
			Metadata: map[string]interface{}{
				"dataSource":   match.Vulnerability.DataSource,
				"matchDetails": match.MatchDetails,
			},
		}

		// Extract CVSS score
		if len(match.Vulnerability.CVSS) > 0 {
			vuln.CVSS = match.Vulnerability.CVSS[0].Metrics.BaseScore
		}

		// Extract fixed version
		if len(match.Vulnerability.Fix.Versions) > 0 {
			vuln.FixedInVersion = match.Vulnerability.Fix.Versions[0]
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

	// Parse database metadata
	if grypeResult.Descriptor.DB.Built != "" {
		result.Metadata.DBVersion = grypeResult.Descriptor.DB.Built
		if t, err := time.Parse(time.RFC3339, grypeResult.Descriptor.DB.Built); err == nil {
			result.Metadata.DBUpdatedAt = t
		}
	}

	return result, nil
}

// mapPackageType maps Grype package types to our internal types
func (g *GrypeScanner) mapPackageType(artifactType, language string) string {
	if language == "go" {
		return "go-module"
	}
	if strings.Contains(artifactType, "go") {
		return "go-module"
	}
	return artifactType
}

// normalizeSeverity normalizes severity strings to standard values
func (g *GrypeScanner) normalizeSeverity(severity string) string {
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
	case "negligible":
		return "Negligible"
	default:
		return "Unknown"
	}
}

// WriteGrypeConfig writes a Grype configuration file for optimized Go scanning
func WriteGrypeConfig(path string) error {
	config := map[string]interface{}{
		"# Grype configuration for Go projects": nil,
		"fail-on-severity":                      "unknown",
		"output":                                []string{"json"},
		"quiet":                                 false,
		"db": map[string]interface{}{
			"cache-dir":                 filepath.Join(os.TempDir(), "grype-db"),
			"auto-update":               true,
			"validate-by-hash-on-start": false,
		},
		"log": map[string]interface{}{
			"structured": false,
			"level":      "error",
		},
		"check-for-app-update": false,
		"only-fixed":           false,
		"platform":             "",
		"search": map[string]interface{}{
			"scope":              "all-layers",
			"unindexed-archives": false,
		},
		"registry": map[string]interface{}{
			"insecure-skip-tls-verify": false,
			"insecure-use-http":        false,
		},
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
