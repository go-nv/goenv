package sbom

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewComplianceReporter(t *testing.T) {
	reporter := NewComplianceReporter()

	if reporter == nil {
		t.Fatal("NewComplianceReporter() returned nil")
	}

	if reporter.frameworks == nil {
		t.Error("frameworks map not initialized")
	}

	// Check that frameworks are loaded
	expectedFrameworks := []ComplianceFramework{
		FrameworkSOC2,
		FrameworkISO27001,
		FrameworkSLSA,
		FrameworkSSDFv1,
		FrameworkCISA,
	}

	for _, fw := range expectedFrameworks {
		if len(reporter.frameworks[fw]) == 0 {
			t.Errorf("framework %s not initialized", fw)
		}
	}
}

func TestComplianceReporter_GenerateReport(t *testing.T) {
	tests := []struct {
		name       string
		sbomData   map[string]interface{}
		framework  ComplianceFramework
		wantStatus string
		minPassed  int
		wantErr    bool
	}{
		{
			name:       "partially compliant SBOM - SOC2",
			sbomData:   createComplianceTestSBOM(true, true, true),
			framework:  FrameworkSOC2,
			wantStatus: "partially-compliant",
			minPassed:  2,
			wantErr:    false,
		},
		{
			name: "non-compliant SBOM - missing components",
			sbomData: map[string]interface{}{
				"bomFormat":   "CycloneDX",
				"specVersion": "1.4",
				"components":  []interface{}{},
				"metadata":    map[string]interface{}{},
			},
			framework:  FrameworkSOC2,
			wantStatus: "non-compliant",
			minPassed:  0,
			wantErr:    false,
		},
		{
			name:       "partially compliant SBOM - ISO27001",
			sbomData:   createComplianceTestSBOM(true, true, true),
			framework:  FrameworkISO27001,
			wantStatus: "partially-compliant",
			minPassed:  2,
			wantErr:    false,
		},
		{
			name:       "partially compliant SBOM - SLSA",
			sbomData:   createComplianceTestSBOM(true, true, true),
			framework:  FrameworkSLSA,
			wantStatus: "partially-compliant",
			minPassed:  2,
			wantErr:    false,
		},
		{
			name:       "partially compliant SBOM - SSDF",
			sbomData:   createComplianceTestSBOM(true, true, true),
			framework:  FrameworkSSDFv1,
			wantStatus: "partially-compliant",
			minPassed:  2,
			wantErr:    false,
		},
		{
			name:       "partially compliant SBOM - CISA",
			sbomData:   createComplianceTestSBOM(true, true, true),
			framework:  FrameworkCISA,
			wantStatus: "partially-compliant",
			minPassed:  2,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp SBOM file
			tmpFile := filepath.Join(t.TempDir(), "sbom.json")
			sbomData, _ := json.MarshalIndent(tt.sbomData, "", "  ")
			err := os.WriteFile(tmpFile, sbomData, 0644)
			if err != nil {
				t.Fatalf("failed to write test SBOM: %v", err)
			}

			// Generate report
			reporter := NewComplianceReporter()
			report, err := reporter.GenerateReport(tmpFile, tt.framework)

			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateReport() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Validate report
			if report == nil {
				t.Fatal("GenerateReport() returned nil report")
			}

			if report.Framework != string(tt.framework) {
				t.Errorf("Framework = %s, want %s", report.Framework, tt.framework)
			}

			if report.OverallStatus != tt.wantStatus {
				t.Errorf("OverallStatus = %s, want %s", report.OverallStatus, tt.wantStatus)
			}

			if report.PassedCount < tt.minPassed {
				t.Errorf("PassedCount = %d, want at least %d", report.PassedCount, tt.minPassed)
			}

			if report.SBOMPath != tmpFile {
				t.Errorf("SBOMPath = %s, want %s", report.SBOMPath, tmpFile)
			}

			if report.GeneratedAt.IsZero() {
				t.Error("GeneratedAt is zero")
			}

			if len(report.Requirements) == 0 {
				t.Error("Requirements is empty")
			}

			if len(report.Checks) != len(report.Requirements) {
				t.Errorf("Checks count = %d, want %d", len(report.Checks), len(report.Requirements))
			}

			if report.Summary == "" {
				t.Error("Summary is empty")
			}
		})
	}
}

func TestComplianceReporter_GenerateReport_InvalidFile(t *testing.T) {
	reporter := NewComplianceReporter()

	// Test non-existent file
	_, err := reporter.GenerateReport("/nonexistent/sbom.json", FrameworkSOC2)
	if err == nil {
		t.Error("GenerateReport() expected error for non-existent file")
	}

	// Test invalid JSON
	tmpFile := filepath.Join(t.TempDir(), "invalid.json")
	err = os.WriteFile(tmpFile, []byte("invalid json"), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err = reporter.GenerateReport(tmpFile, FrameworkSOC2)
	if err == nil {
		t.Error("GenerateReport() expected error for invalid JSON")
	}
}

func TestComplianceReporter_GenerateReport_UnknownFramework(t *testing.T) {
	reporter := NewComplianceReporter()

	tmpFile := filepath.Join(t.TempDir(), "sbom.json")
	sbomData := createComplianceTestSBOM(true, true, true)
	jsonData, _ := json.Marshal(sbomData)
	err := os.WriteFile(tmpFile, jsonData, 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err = reporter.GenerateReport(tmpFile, ComplianceFramework("unknown"))
	if err == nil {
		t.Error("GenerateReport() expected error for unknown framework")
	}
}

func TestComplianceReporter_FrameworkAll(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "sbom.json")
	sbomData := createComplianceTestSBOM(true, true, true)
	jsonData, _ := json.Marshal(sbomData)
	err := os.WriteFile(tmpFile, jsonData, 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	reporter := NewComplianceReporter()
	report, err := reporter.GenerateReport(tmpFile, FrameworkAll)

	if err != nil {
		t.Fatalf("GenerateReport() error = %v", err)
	}

	// Should have requirements from all frameworks
	minRequirements := 10 // At least 10 total requirements across all frameworks
	if len(report.Requirements) < minRequirements {
		t.Errorf("Requirements count = %d, want at least %d", len(report.Requirements), minRequirements)
	}
}

func TestComplianceCheck_HasSBOM(t *testing.T) {
	reporter := NewComplianceReporter()
	sbom := createComplianceTestSBOM(true, false, false)

	check := ComplianceCheck{
		Evidence:        []string{},
		Issues:          []string{},
		Recommendations: []string{},
	}

	reporter.executeCheck(&check, sbom, "has-sbom")

	if len(check.Evidence) == 0 {
		t.Error("has-sbom check should add evidence")
	}

	if !strings.Contains(check.Evidence[0], "SBOM") {
		t.Error("has-sbom evidence should mention SBOM")
	}
}

func TestComplianceCheck_Components(t *testing.T) {
	tests := []struct {
		name          string
		hasComponents bool
		wantEvidence  bool
		wantIssues    bool
	}{
		{
			name:          "with components",
			hasComponents: true,
			wantEvidence:  true,
			wantIssues:    false,
		},
		{
			name:          "no components",
			hasComponents: false,
			wantEvidence:  false,
			wantIssues:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reporter := NewComplianceReporter()
			sbom := createComplianceTestSBOM(tt.hasComponents, false, false)

			check := ComplianceCheck{
				Evidence:        []string{},
				Issues:          []string{},
				Recommendations: []string{},
			}

			reporter.executeCheck(&check, sbom, "has-components")

			if tt.wantEvidence && len(check.Evidence) == 0 {
				t.Error("expected evidence for has-components check")
			}

			if tt.wantIssues && len(check.Issues) == 0 {
				t.Error("expected issues for has-components check")
			}

			if tt.wantIssues && len(check.Recommendations) == 0 {
				t.Error("expected recommendations when issues found")
			}
		})
	}
}

func TestComplianceCheck_Licenses(t *testing.T) {
	tests := []struct {
		name         string
		hasLicenses  bool
		wantEvidence bool
		wantIssues   bool
	}{
		{
			name:         "with licenses",
			hasLicenses:  true,
			wantEvidence: true,
			wantIssues:   false,
		},
		{
			name:         "no licenses",
			hasLicenses:  false,
			wantEvidence: false,
			wantIssues:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reporter := NewComplianceReporter()
			sbom := createComplianceTestSBOM(true, tt.hasLicenses, false)

			check := ComplianceCheck{
				Evidence:        []string{},
				Issues:          []string{},
				Recommendations: []string{},
			}

			reporter.executeCheck(&check, sbom, "component-licenses")

			if tt.wantEvidence && len(check.Evidence) == 0 {
				t.Error("expected evidence for license check")
			}

			if tt.wantIssues && len(check.Issues) == 0 {
				t.Error("expected issues for license check")
			}
		})
	}
}

func TestComplianceCheck_BuildMetadata(t *testing.T) {
	tests := []struct {
		name         string
		hasMetadata  bool
		wantEvidence bool
		wantIssues   bool
	}{
		{
			name:         "with metadata",
			hasMetadata:  true,
			wantEvidence: true,
			wantIssues:   false,
		},
		{
			name:         "no metadata",
			hasMetadata:  false,
			wantEvidence: false,
			wantIssues:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reporter := NewComplianceReporter()
			sbom := createComplianceTestSBOM(true, false, tt.hasMetadata)

			check := ComplianceCheck{
				Evidence:        []string{},
				Issues:          []string{},
				Recommendations: []string{},
			}

			reporter.executeCheck(&check, sbom, "build-metadata")

			if tt.wantEvidence && len(check.Evidence) == 0 {
				t.Error("expected evidence for build metadata check")
			}

			if tt.wantIssues && len(check.Issues) == 0 {
				t.Error("expected issues for build metadata check")
			}
		})
	}
}

func TestComplianceCheck_AllTypes(t *testing.T) {
	checkTypes := []string{
		"has-sbom",
		"sbom-format",
		"has-components",
		"component-licenses",
		"component-versions",
		"build-metadata",
		"supply-chain-transparency",
		"timestamp",
		"tool-information",
		"vulnerability-tracking",
		"provenance",
	}

	reporter := NewComplianceReporter()
	sbom := createComplianceTestSBOM(true, true, true)

	for _, checkType := range checkTypes {
		t.Run(checkType, func(t *testing.T) {
			check := ComplianceCheck{
				Evidence:        []string{},
				Issues:          []string{},
				Recommendations: []string{},
			}

			// Should not panic
			reporter.executeCheck(&check, sbom, checkType)

			// Should have at least evidence or issues
			if len(check.Evidence) == 0 && len(check.Issues) == 0 {
				t.Errorf("check %s produced no evidence or issues", checkType)
			}
		})
	}
}

func TestComplianceReporter_FormatReportAsHTML(t *testing.T) {
	reporter := NewComplianceReporter()

	report := &ComplianceReport{
		Framework:     "test-framework",
		GeneratedAt:   time.Now(),
		SBOMPath:      "/path/to/sbom.json",
		OverallStatus: "compliant",
		PassedCount:   3,
		FailedCount:   0,
		PartialCount:  0,
		NotApplicable: 0,
		Requirements: []ComplianceRequirement{
			{
				ID:          "TEST-1",
				Name:        "Test Requirement",
				Description: "Test description",
				Category:    "Test",
				Severity:    "high",
			},
		},
		Checks: []ComplianceCheck{
			{
				RequirementID:   "TEST-1",
				Status:          "pass",
				Evidence:        []string{"Test evidence"},
				Issues:          []string{},
				Recommendations: []string{},
			},
		},
		Summary: "Test summary",
	}

	html := reporter.FormatReportAsHTML(report)

	// Check for key HTML elements
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("HTML output missing DOCTYPE")
	}

	if !strings.Contains(html, "<html>") {
		t.Error("HTML output missing html tag")
	}

	if !strings.Contains(html, "test-framework") {
		t.Error("HTML output missing framework name")
	}

	if !strings.Contains(html, "TEST-1") {
		t.Error("HTML output missing requirement ID")
	}

	if !strings.Contains(html, "Test evidence") {
		t.Error("HTML output missing evidence")
	}

	if !strings.Contains(html, "compliant") {
		t.Error("HTML output missing status")
	}
}

func TestComplianceReport_Statistics(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "sbom.json")

	// Create SBOM with mixed compliance
	sbomData := map[string]interface{}{
		"bomFormat":   "CycloneDX",
		"specVersion": "1.4",
		"components": []interface{}{
			map[string]interface{}{
				"name":    "test-component",
				"version": "1.0.0",
				// Missing license
			},
		},
		"metadata": map[string]interface{}{
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}

	jsonData, _ := json.Marshal(sbomData)
	err := os.WriteFile(tmpFile, jsonData, 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	reporter := NewComplianceReporter()
	report, err := reporter.GenerateReport(tmpFile, FrameworkSOC2)

	if err != nil {
		t.Fatalf("GenerateReport() error = %v", err)
	}

	// Verify statistics
	total := report.PassedCount + report.FailedCount + report.PartialCount + report.NotApplicable
	if total != len(report.Requirements) {
		t.Errorf("statistics don't add up: %d + %d + %d + %d != %d",
			report.PassedCount, report.FailedCount, report.PartialCount,
			report.NotApplicable, len(report.Requirements))
	}

	// Should have some passed and some failed/partial
	if report.PassedCount == 0 {
		t.Error("expected at least some passed checks")
	}
}

func TestGetComponentName(t *testing.T) {
	tests := []struct {
		name string
		comp map[string]interface{}
		want string
	}{
		{
			name: "with name",
			comp: map[string]interface{}{"name": "test-component"},
			want: "test-component",
		},
		{
			name: "without name",
			comp: map[string]interface{}{},
			want: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getComponentName(tt.comp)
			if got != tt.want {
				t.Errorf("getComponentName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasLicense(t *testing.T) {
	tests := []struct {
		name string
		comp map[string]interface{}
		want bool
	}{
		{
			name: "with licenses array",
			comp: map[string]interface{}{
				"licenses": []interface{}{
					map[string]interface{}{"license": "MIT"},
				},
			},
			want: true,
		},
		{
			name: "with license string",
			comp: map[string]interface{}{
				"license": "MIT",
			},
			want: true,
		},
		{
			name: "without license",
			comp: map[string]interface{}{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasLicense(tt.comp)
			if got != tt.want {
				t.Errorf("hasLicense() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function to create test SBOM data
func createComplianceTestSBOM(hasComponents, hasLicenses, hasMetadata bool) map[string]interface{} {
	components := []interface{}{}

	if hasComponents {
		comp := map[string]interface{}{
			"name":    "test-component",
			"version": "1.0.0",
		}

		if hasLicenses {
			comp["licenses"] = []interface{}{
				map[string]interface{}{
					"license": map[string]interface{}{
						"id": "MIT",
					},
				},
			}
		}

		components = append(components, comp)
	}

	metadata := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"tools": []interface{}{
			map[string]interface{}{
				"name": "goenv",
			},
		},
	}

	if hasMetadata {
		metadata["properties"] = []interface{}{
			map[string]interface{}{
				"name":  "goenv:go_version",
				"value": "1.21.0",
			},
			map[string]interface{}{
				"name":  "goenv:build_context.goos",
				"value": "linux",
			},
			map[string]interface{}{
				"name":  "goenv:build_context.goarch",
				"value": "amd64",
			},
		}
	}

	return map[string]interface{}{
		"bomFormat":   "CycloneDX",
		"specVersion": "1.4",
		"components":  components,
		"metadata":    metadata,
	}
}
