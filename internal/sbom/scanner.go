package sbom

import (
	"context"
	"fmt"
	"time"
)

// Scanner provides vulnerability scanning capabilities for SBOMs
type Scanner interface {
	// Name returns the scanner name (e.g., "grype", "trivy")
	Name() string

	// Version returns the scanner version
	Version() (string, error)

	// IsInstalled checks if the scanner tool is available
	IsInstalled() bool

	// InstallationInstructions returns help text for installing the scanner
	InstallationInstructions() string

	// Scan performs vulnerability scanning on an SBOM file
	Scan(ctx context.Context, opts *ScanOptions) (*ScanResult, error)

	// SupportsFormat checks if the scanner supports the given SBOM format
	SupportsFormat(format string) bool
}

// ScanOptions configures vulnerability scanning behavior
type ScanOptions struct {
	// SBOMPath is the path to the SBOM file to scan
	SBOMPath string

	// Format specifies the SBOM format (cyclonedx-json, spdx-json, etc.)
	Format string

	// OutputFormat specifies how to format scan results (json, table, sarif)
	OutputFormat string

	// OutputPath is where to write scan results (empty = stdout)
	OutputPath string

	// SeverityThreshold filters results to this severity and above (low, medium, high, critical)
	SeverityThreshold string

	// FailOn determines when to exit with non-zero code (any, high, critical, none)
	FailOn string

	// OnlyFixed shows only vulnerabilities with available fixes
	OnlyFixed bool

	// Offline mode - avoid network access for vulnerability database updates
	Offline bool

	// Verbose enables detailed output
	Verbose bool

	// AdditionalArgs are extra arguments to pass to the scanner
	AdditionalArgs []string
}

// ScanResult contains the results of vulnerability scanning
type ScanResult struct {
	// Scanner name and version
	Scanner        string    `json:"scanner"`
	ScannerVersion string    `json:"scannerVersion"`
	Timestamp      time.Time `json:"timestamp"`

	// Input information
	SBOMPath   string `json:"sbomPath"`
	SBOMFormat string `json:"sbomFormat"`

	// Scan results
	Vulnerabilities []Vulnerability      `json:"vulnerabilities"`
	Summary         VulnerabilitySummary `json:"summary"`

	// Metadata
	Metadata ScanMetadata `json:"metadata"`
}

// Vulnerability represents a single security vulnerability
type Vulnerability struct {
	// Vulnerability identifier (CVE-2023-39325, GHSA-xxxx-yyyy-zzzz)
	ID string `json:"id"`

	// Package information
	PackageName    string `json:"packageName"`
	PackageVersion string `json:"packageVersion"`
	PackageType    string `json:"packageType"` // go-module, stdlib, etc.

	// Vulnerability details
	Severity    string   `json:"severity"` // Critical, High, Medium, Low, Negligible
	CVSS        float64  `json:"cvss,omitempty"`
	Description string   `json:"description"`
	URLs        []string `json:"urls,omitempty"`
	PublishedAt string   `json:"publishedAt,omitempty"`
	ModifiedAt  string   `json:"modifiedAt,omitempty"`

	// Fix information
	FixedInVersion string `json:"fixedInVersion,omitempty"`
	FixAvailable   bool   `json:"fixAvailable"`

	// Path information
	VulnerablePath []string `json:"vulnerablePath,omitempty"` // Dependency path to vulnerable package

	// Scanner-specific metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// VulnerabilitySummary provides aggregate statistics
type VulnerabilitySummary struct {
	Total      int `json:"total"`
	Critical   int `json:"critical"`
	High       int `json:"high"`
	Medium     int `json:"medium"`
	Low        int `json:"low"`
	Negligible int `json:"negligible"`
	Unknown    int `json:"unknown"`

	// Fix availability
	WithFix    int `json:"withFix"`
	WithoutFix int `json:"withoutFix"`
}

// ScanMetadata contains additional scan information
type ScanMetadata struct {
	// Database information
	DBVersion   string    `json:"dbVersion,omitempty"`
	DBUpdatedAt time.Time `json:"dbUpdatedAt,omitempty"`

	// Scan timing
	ScanDuration time.Duration `json:"scanDuration"`

	// Go-specific context (from enhanced SBOM)
	GoVersion  string   `json:"goVersion,omitempty"`
	CGOEnabled bool     `json:"cgoEnabled,omitempty"`
	BuildTags  []string `json:"buildTags,omitempty"`
	Vendored   bool     `json:"vendored,omitempty"`
}

// ScanError represents an error during scanning
type ScanError struct {
	Scanner string
	Message string
	Cause   error
}

func (e *ScanError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s scanner error: %s: %v", e.Scanner, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s scanner error: %s", e.Scanner, e.Message)
}

func (e *ScanError) Unwrap() error {
	return e.Cause
}

// NewScanError creates a new scanner error
func NewScanError(scanner, message string, cause error) *ScanError {
	return &ScanError{
		Scanner: scanner,
		Message: message,
		Cause:   cause,
	}
}

// GetScanner returns a scanner implementation by name
func GetScanner(name string) (Scanner, error) {
	switch name {
	case "grype":
		return NewGrypeScanner(), nil
	case "trivy":
		return NewTrivyScanner(), nil
	case "snyk":
		return NewSnykScanner(), nil
	case "veracode":
		return NewVeracodeScanner(), nil
	default:
		return nil, fmt.Errorf("unknown scanner: %s (supported: grype, trivy, snyk, veracode)", name)
	}
}

// ListAvailableScanners returns all available scanner implementations
func ListAvailableScanners() []Scanner {
	return []Scanner{
		NewGrypeScanner(),
		NewTrivyScanner(),
		NewSnykScanner(),
		NewVeracodeScanner(),
	}
}

// FindInstalledScanners returns scanners that are currently installed
func FindInstalledScanners() []Scanner {
	var installed []Scanner
	for _, scanner := range ListAvailableScanners() {
		if scanner.IsInstalled() {
			installed = append(installed, scanner)
		}
	}
	return installed
}

// SeverityLevel represents vulnerability severity
type SeverityLevel int

const (
	SeverityUnknown SeverityLevel = iota
	SeverityNegligible
	SeverityLow
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

// ParseSeverity converts a severity string to SeverityLevel
func ParseSeverity(s string) SeverityLevel {
	switch toLower(s) {
	case "critical":
		return SeverityCritical
	case "high":
		return SeverityHigh
	case "medium":
		return SeverityMedium
	case "low":
		return SeverityLow
	case "negligible":
		return SeverityNegligible
	default:
		return SeverityUnknown
	}
}

// toLower is a simple lowercase helper to avoid importing strings
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

// String returns the string representation of SeverityLevel
func (s SeverityLevel) String() string {
	switch s {
	case SeverityCritical:
		return "Critical"
	case SeverityHigh:
		return "High"
	case SeverityMedium:
		return "Medium"
	case SeverityLow:
		return "Low"
	case SeverityNegligible:
		return "Negligible"
	default:
		return "Unknown"
	}
}

// FilterVulnerabilities filters vulnerabilities based on criteria
func FilterVulnerabilities(vulns []Vulnerability, minSeverity string, onlyFixed bool) []Vulnerability {
	minLevel := ParseSeverity(minSeverity)
	var filtered []Vulnerability

	for _, v := range vulns {
		// Check severity threshold
		if ParseSeverity(v.Severity) < minLevel {
			continue
		}

		// Check fix availability
		if onlyFixed && !v.FixAvailable {
			continue
		}

		filtered = append(filtered, v)
	}

	return filtered
}
