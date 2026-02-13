package sbom

import (
	"context"
	"testing"
	"time"
)

func TestGetScanner(t *testing.T) {
	tests := []struct {
		name        string
		scannerName string
		wantErr     bool
	}{
		{
			name:        "grype scanner",
			scannerName: "grype",
			wantErr:     false,
		},
		{
			name:        "trivy scanner",
			scannerName: "trivy",
			wantErr:     false,
		},
		{
			name:        "unknown scanner",
			scannerName: "unknown",
			wantErr:     true,
		},
		{
			name:        "empty scanner name",
			scannerName: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner, err := GetScanner(tt.scannerName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetScanner() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && scanner == nil {
				t.Error("GetScanner() returned nil scanner without error")
			}
			if !tt.wantErr && scanner.Name() != tt.scannerName {
				t.Errorf("GetScanner() returned scanner with name %s, want %s", scanner.Name(), tt.scannerName)
			}
		})
	}
}

func TestListAvailableScanners(t *testing.T) {
	scanners := ListAvailableScanners()

	if len(scanners) == 0 {
		t.Error("ListAvailableScanners() returned empty list")
	}

	// Should include at least grype and trivy
	foundGrype := false
	foundTrivy := false
	for _, s := range scanners {
		if s.Name() == "grype" {
			foundGrype = true
		}
		if s.Name() == "trivy" {
			foundTrivy = true
		}
	}

	if !foundGrype {
		t.Error("ListAvailableScanners() did not include grype")
	}
	if !foundTrivy {
		t.Error("ListAvailableScanners() did not include trivy")
	}
}

func TestParseSeverity(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected SeverityLevel
	}{
		{"critical lowercase", "critical", SeverityCritical},
		{"critical uppercase", "CRITICAL", SeverityCritical},
		{"critical mixed", "Critical", SeverityCritical},
		{"high lowercase", "high", SeverityHigh},
		{"high uppercase", "HIGH", SeverityHigh},
		{"high mixed", "High", SeverityHigh},
		{"medium lowercase", "medium", SeverityMedium},
		{"medium uppercase", "MEDIUM", SeverityMedium},
		{"medium mixed", "Medium", SeverityMedium},
		{"low lowercase", "low", SeverityLow},
		{"low uppercase", "LOW", SeverityLow},
		{"low mixed", "Low", SeverityLow},
		{"negligible lowercase", "negligible", SeverityNegligible},
		{"negligible uppercase", "NEGLIGIBLE", SeverityNegligible},
		{"negligible mixed", "Negligible", SeverityNegligible},
		{"unknown", "unknown", SeverityUnknown},
		{"invalid", "invalid", SeverityUnknown},
		{"empty", "", SeverityUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseSeverity(tt.input)
			if result != tt.expected {
				t.Errorf("ParseSeverity(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSeverityLevelString(t *testing.T) {
	tests := []struct {
		level    SeverityLevel
		expected string
	}{
		{SeverityCritical, "Critical"},
		{SeverityHigh, "High"},
		{SeverityMedium, "Medium"},
		{SeverityLow, "Low"},
		{SeverityNegligible, "Negligible"},
		{SeverityUnknown, "Unknown"},
		{SeverityLevel(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.level.String()
			if result != tt.expected {
				t.Errorf("SeverityLevel.String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFilterVulnerabilities(t *testing.T) {
	vulns := []Vulnerability{
		{ID: "CVE-1", Severity: "Critical", FixAvailable: true},
		{ID: "CVE-2", Severity: "High", FixAvailable: true},
		{ID: "CVE-3", Severity: "Medium", FixAvailable: false},
		{ID: "CVE-4", Severity: "Low", FixAvailable: true},
		{ID: "CVE-5", Severity: "Negligible", FixAvailable: false},
	}

	tests := []struct {
		name        string
		minSeverity string
		onlyFixed   bool
		wantCount   int
	}{
		{
			name:        "all vulnerabilities",
			minSeverity: "Negligible",
			onlyFixed:   false,
			wantCount:   5,
		},
		{
			name:        "high and above",
			minSeverity: "High",
			onlyFixed:   false,
			wantCount:   2,
		},
		{
			name:        "critical only",
			minSeverity: "Critical",
			onlyFixed:   false,
			wantCount:   1,
		},
		{
			name:        "only fixed",
			minSeverity: "Negligible",
			onlyFixed:   true,
			wantCount:   3,
		},
		{
			name:        "high with fix",
			minSeverity: "High",
			onlyFixed:   true,
			wantCount:   2,
		},
		{
			name:        "medium with fix",
			minSeverity: "Medium",
			onlyFixed:   true,
			wantCount:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := FilterVulnerabilities(vulns, tt.minSeverity, tt.onlyFixed)
			if len(filtered) != tt.wantCount {
				t.Errorf("FilterVulnerabilities() returned %d vulnerabilities, want %d", len(filtered), tt.wantCount)
			}
		})
	}
}

func TestScanError(t *testing.T) {
	t.Run("error with cause", func(t *testing.T) {
		cause := context.DeadlineExceeded
		err := NewScanError("grype", "timeout occurred", cause)

		if err.Scanner != "grype" {
			t.Errorf("Scanner = %q, want %q", err.Scanner, "grype")
		}
		if err.Message != "timeout occurred" {
			t.Errorf("Message = %q, want %q", err.Message, "timeout occurred")
		}
		if err.Cause != cause {
			t.Errorf("Cause = %v, want %v", err.Cause, cause)
		}

		errMsg := err.Error()
		if errMsg != "grype scanner error: timeout occurred: context deadline exceeded" {
			t.Errorf("Error() = %q, want it to contain scanner, message and cause", errMsg)
		}
	})

	t.Run("error without cause", func(t *testing.T) {
		err := NewScanError("trivy", "not found", nil)

		errMsg := err.Error()
		if errMsg != "trivy scanner error: not found" {
			t.Errorf("Error() = %q, want %q", errMsg, "trivy scanner error: not found")
		}
	})

	t.Run("unwrap", func(t *testing.T) {
		cause := context.Canceled
		err := NewScanError("scanner", "message", cause)

		unwrapped := err.Unwrap()
		if unwrapped != cause {
			t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
		}
	})
}

func TestVulnerabilitySummary(t *testing.T) {
	t.Run("calculate summary from vulnerabilities", func(t *testing.T) {
		result := &ScanResult{
			Scanner:        "test",
			ScannerVersion: "1.0.0",
			Timestamp:      time.Now(),
			Vulnerabilities: []Vulnerability{
				{Severity: "Critical", FixAvailable: true},
				{Severity: "Critical", FixAvailable: false},
				{Severity: "High", FixAvailable: true},
				{Severity: "High", FixAvailable: true},
				{Severity: "Medium", FixAvailable: false},
				{Severity: "Low", FixAvailable: true},
			},
			Summary: VulnerabilitySummary{
				Total:      6,
				Critical:   2,
				High:       2,
				Medium:     1,
				Low:        1,
				WithFix:    4,
				WithoutFix: 2,
			},
		}

		if result.Summary.Total != 6 {
			t.Errorf("Summary.Total = %d, want 6", result.Summary.Total)
		}
		if result.Summary.Critical != 2 {
			t.Errorf("Summary.Critical = %d, want 2", result.Summary.Critical)
		}
		if result.Summary.High != 2 {
			t.Errorf("Summary.High = %d, want 2", result.Summary.High)
		}
		if result.Summary.WithFix != 4 {
			t.Errorf("Summary.WithFix = %d, want 4", result.Summary.WithFix)
		}
	})
}

func TestScanOptions(t *testing.T) {
	t.Run("valid scan options", func(t *testing.T) {
		opts := &ScanOptions{
			SBOMPath:          "/path/to/sbom.json",
			Format:            "cyclonedx-json",
			OutputFormat:      "json",
			OutputPath:        "/path/to/results.json",
			SeverityThreshold: "high",
			FailOn:            "critical",
			OnlyFixed:         true,
			Offline:           false,
			Verbose:           true,
			AdditionalArgs:    []string{"--custom-arg"},
		}

		if opts.SBOMPath != "/path/to/sbom.json" {
			t.Errorf("SBOMPath = %q, want %q", opts.SBOMPath, "/path/to/sbom.json")
		}
		if opts.Format != "cyclonedx-json" {
			t.Errorf("Format = %q, want %q", opts.Format, "cyclonedx-json")
		}
		if !opts.OnlyFixed {
			t.Error("OnlyFixed = false, want true")
		}
	})
}
