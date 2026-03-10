package sbom

import (
	"testing"
)

func TestGrypeScanner_Name(t *testing.T) {
	scanner := NewGrypeScanner()
	if scanner.Name() != "grype" {
		t.Errorf("Name() = %q, want %q", scanner.Name(), "grype")
	}
}

func TestGrypeScanner_InstallationInstructions(t *testing.T) {
	scanner := NewGrypeScanner()
	instructions := scanner.InstallationInstructions()

	if instructions == "" {
		t.Error("InstallationInstructions() returned empty string")
	}

	// Should mention goenv tools install
	if !testContains(instructions, "goenv tools install grype") {
		t.Error("InstallationInstructions() should mention 'goenv tools install grype'")
	}

	// Should mention homebrew
	if !testContains(instructions, "brew install grype") {
		t.Error("InstallationInstructions() should mention 'brew install grype'")
	}
}

func TestGrypeScanner_SupportsFormat(t *testing.T) {
	scanner := NewGrypeScanner()

	tests := []struct {
		format    string
		supported bool
	}{
		{"cyclonedx-json", true},
		{"cyclonedx-xml", true},
		{"spdx-json", true},
		{"spdx-tag-value", true},
		{"syft-json", true},
		{"unknown-format", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			result := scanner.SupportsFormat(tt.format)
			if result != tt.supported {
				t.Errorf("SupportsFormat(%q) = %v, want %v", tt.format, result, tt.supported)
			}
		})
	}
}

func TestGrypeScanner_BuildArgs(t *testing.T) {
	scanner := NewGrypeScanner()

	tests := []struct {
		name     string
		opts     *ScanOptions
		wantArgs []string
	}{
		{
			name: "basic scan",
			opts: &ScanOptions{
				SBOMPath: "/path/to/sbom.json",
			},
			wantArgs: []string{"sbom:/path/to/sbom.json", "-o", "json"},
		},
		{
			name: "with output file",
			opts: &ScanOptions{
				SBOMPath:   "/path/to/sbom.json",
				OutputPath: "/path/to/results.json",
			},
			wantArgs: []string{"sbom:/path/to/sbom.json", "-o", "json", "--file", "/path/to/results.json"},
		},
		{
			name: "with severity threshold",
			opts: &ScanOptions{
				SBOMPath:          "/path/to/sbom.json",
				SeverityThreshold: "high",
			},
			wantArgs: []string{"sbom:/path/to/sbom.json", "-o", "json", "--fail-on", "high"},
		},
		{
			name: "with offline mode",
			opts: &ScanOptions{
				SBOMPath: "/path/to/sbom.json",
				Offline:  true,
			},
			wantArgs: []string{"sbom:/path/to/sbom.json", "-o", "json", "--db-auto-update=false"},
		},
		{
			name: "with only-fixed",
			opts: &ScanOptions{
				SBOMPath:  "/path/to/sbom.json",
				OnlyFixed: true,
			},
			wantArgs: []string{"sbom:/path/to/sbom.json", "-o", "json", "--only-fixed"},
		},
		{
			name: "with verbose",
			opts: &ScanOptions{
				SBOMPath: "/path/to/sbom.json",
				Verbose:  true,
			},
			wantArgs: []string{"sbom:/path/to/sbom.json", "-o", "json", "-v"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := scanner.buildGrypeArgs(tt.opts)

			// Check that expected args are present
			for _, want := range tt.wantArgs {
				if !testContainsString(args, want) {
					t.Errorf("buildGrypeArgs() missing expected arg %q, got %v", want, args)
				}
			}
		})
	}
}

func TestGrypeScanner_NormalizeSeverity(t *testing.T) {
	scanner := NewGrypeScanner()

	tests := []struct {
		input    string
		expected string
	}{
		{"critical", "Critical"},
		{"CRITICAL", "Critical"},
		{"high", "High"},
		{"HIGH", "High"},
		{"medium", "Medium"},
		{"low", "Low"},
		{"negligible", "Negligible"},
		{"unknown", "Unknown"},
		{"", "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := scanner.normalizeSeverity(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeSeverity(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGrypeScanner_MapPackageType(t *testing.T) {
	scanner := NewGrypeScanner()

	tests := []struct {
		artifactType string
		language     string
		expected     string
	}{
		{"go-module", "go", "go-module"},
		{"unknown", "go", "go-module"},
		{"go-binary", "go", "go-module"},
		{"npm", "javascript", "npm"},
		{"", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.artifactType+"_"+tt.language, func(t *testing.T) {
			result := scanner.mapPackageType(tt.artifactType, tt.language)
			if result != tt.expected {
				t.Errorf("mapPackageType(%q, %q) = %q, want %q",
					tt.artifactType, tt.language, result, tt.expected)
			}
		})
	}
}

// Helper functions for testing
func testContains(s, substr string) bool {
	if len(s) == 0 || len(substr) == 0 {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func testContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
