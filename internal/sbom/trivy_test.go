package sbom

import (
	"testing"
)

func TestTrivyScanner_Name(t *testing.T) {
	scanner := NewTrivyScanner()
	if scanner.Name() != "trivy" {
		t.Errorf("Name() = %q, want %q", scanner.Name(), "trivy")
	}
}

func TestTrivyScanner_InstallationInstructions(t *testing.T) {
	scanner := NewTrivyScanner()
	instructions := scanner.InstallationInstructions()

	if instructions == "" {
		t.Error("InstallationInstructions() returned empty string")
	}

	// Should mention goenv tools install
	if !testContains(instructions, "goenv tools install trivy") {
		t.Error("InstallationInstructions() should mention 'goenv tools install trivy'")
	}

	// Should mention homebrew
	if !testContains(instructions, "brew install trivy") {
		t.Error("InstallationInstructions() should mention 'brew install trivy'")
	}

	// Should mention docker option
	if !testContains(instructions, "docker pull") {
		t.Error("InstallationInstructions() should mention docker option")
	}
}

func TestTrivyScanner_SupportsFormat(t *testing.T) {
	scanner := NewTrivyScanner()

	tests := []struct {
		format    string
		supported bool
	}{
		{"cyclonedx-json", true},
		{"cyclonedx", true},
		{"spdx-json", true},
		{"spdx", true},
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

func TestTrivyScanner_BuildArgs(t *testing.T) {
	scanner := NewTrivyScanner()

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
			wantArgs: []string{"sbom", "/path/to/sbom.json", "--format", "json"},
		},
		{
			name: "with output file",
			opts: &ScanOptions{
				SBOMPath:   "/path/to/sbom.json",
				OutputPath: "/path/to/results.json",
			},
			wantArgs: []string{"sbom", "/path/to/sbom.json", "--format", "json", "--output", "/path/to/results.json"},
		},
		{
			name: "with severity threshold",
			opts: &ScanOptions{
				SBOMPath:          "/path/to/sbom.json",
				SeverityThreshold: "high",
			},
			wantArgs: []string{"sbom", "/path/to/sbom.json", "--format", "json", "--severity", "HIGH"},
		},
		{
			name: "with offline mode",
			opts: &ScanOptions{
				SBOMPath: "/path/to/sbom.json",
				Offline:  true,
			},
			wantArgs: []string{"sbom", "/path/to/sbom.json", "--format", "json", "--skip-db-update"},
		},
		{
			name: "with only-fixed",
			opts: &ScanOptions{
				SBOMPath:  "/path/to/sbom.json",
				OnlyFixed: true,
			},
			wantArgs: []string{"sbom", "/path/to/sbom.json", "--format", "json", "--ignore-unfixed"},
		},
		{
			name: "with verbose",
			opts: &ScanOptions{
				SBOMPath: "/path/to/sbom.json",
				Verbose:  true,
			},
			wantArgs: []string{"sbom", "/path/to/sbom.json", "--format", "json", "--debug"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := scanner.buildTrivyArgs(tt.opts)

			// Check that expected args are present
			for _, want := range tt.wantArgs {
				if !testContainsString(args, want) {
					t.Errorf("buildTrivyArgs() missing expected arg %q, got %v", want, args)
				}
			}
		})
	}
}

func TestTrivyScanner_NormalizeSeverity(t *testing.T) {
	scanner := NewTrivyScanner()

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
		{"unknown", "Negligible"},
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

func TestTrivyScanner_MapPackageType(t *testing.T) {
	scanner := NewTrivyScanner()

	tests := []struct {
		trivyType string
		expected  string
	}{
		{"gomod", "go-module"},
		{"gobinary", "go-module"},
		{"go", "go-module"},
		{"npm", "npm"},
		{"pip", "pip"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.trivyType, func(t *testing.T) {
			result := scanner.mapPackageType(tt.trivyType)
			if result != tt.expected {
				t.Errorf("mapPackageType(%q) = %q, want %q",
					tt.trivyType, result, tt.expected)
			}
		})
	}
}
