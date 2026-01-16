package sbom

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewDriftDetector(t *testing.T) {
	tests := []struct {
		name        string
		baselineDir string
		wantErr     bool
	}{
		{"valid directory", t.TempDir(), false},
		{"empty directory", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector, err := NewDriftDetector(tt.baselineDir)
			if tt.wantErr {
				if err == nil {
					t.Error("NewDriftDetector() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("NewDriftDetector() unexpected error: %v", err)
			}

			if detector == nil {
				t.Fatal("NewDriftDetector() returned nil detector")
			}

			if detector.BaselineDir != tt.baselineDir {
				t.Errorf("BaselineDir = %q, want %q", detector.BaselineDir, tt.baselineDir)
			}

			// Verify directory was created
			if _, err := os.Stat(tt.baselineDir); os.IsNotExist(err) {
				t.Error("baseline directory was not created")
			}
		})
	}
}

func TestDriftDetector_SaveBaseline(t *testing.T) {
	baselineDir := t.TempDir()
	detector, err := NewDriftDetector(baselineDir)
	if err != nil {
		t.Fatalf("NewDriftDetector() error = %v", err)
	}

	// Create a test SBOM file
	sbomPath := filepath.Join(t.TempDir(), "test.json")
	sbom := createSimpleTestSBOM([]string{"pkg1", "pkg2"}, []string{"1.0.0", "2.0.0"})
	data, _ := json.Marshal(sbom)
	if err := os.WriteFile(sbomPath, data, 0644); err != nil {
		t.Fatalf("failed to create test SBOM: %v", err)
	}

	tests := []struct {
		name         string
		sbomPath     string
		baselineName string
		version      string
		description  string
		wantErr      bool
	}{
		{"valid save", sbomPath, "test", "v1.0.0", "test baseline", false},
		{"default name", sbomPath, "", "v1.0.0", "", false},
		{"invalid path", "/nonexistent/sbom.json", "test", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := detector.SaveBaseline(tt.sbomPath, tt.baselineName, tt.version, tt.description)
			if tt.wantErr {
				if err == nil {
					t.Error("SaveBaseline() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("SaveBaseline() unexpected error: %v", err)
			}

			// Verify baseline file was created
			name := tt.baselineName
			if name == "" {
				name = "default"
			}
			baselineFile := filepath.Join(baselineDir, name+".baseline.json")
			if _, err := os.Stat(baselineFile); os.IsNotExist(err) {
				t.Error("baseline file was not created")
			}

			// Verify baseline content
			data, err := os.ReadFile(baselineFile)
			if err != nil {
				t.Fatalf("failed to read baseline file: %v", err)
			}

			var baseline struct {
				BaselineMeta
				Components []Component
			}
			if err := json.Unmarshal(data, &baseline); err != nil {
				t.Fatalf("failed to unmarshal baseline: %v", err)
			}

			if baseline.Version != tt.version {
				t.Errorf("baseline version = %q, want %q", baseline.Version, tt.version)
			}
			if baseline.Description != tt.description {
				t.Errorf("baseline description = %q, want %q", baseline.Description, tt.description)
			}
			if len(baseline.Components) != 2 {
				t.Errorf("baseline has %d components, want 2", len(baseline.Components))
			}
		})
	}
}

func TestDriftDetector_DetectDrift(t *testing.T) {
	baselineDir := t.TempDir()
	detector, err := NewDriftDetector(baselineDir)
	if err != nil {
		t.Fatalf("NewDriftDetector() error = %v", err)
	}

	// Create baseline SBOM
	baselinePath := filepath.Join(t.TempDir(), "baseline.json")
	baselineSBOM := createSimpleTestSBOM(
		[]string{"pkg1", "pkg2", "pkg3"},
		[]string{"1.0.0", "2.0.0", "3.0.0"},
	)
	data, _ := json.Marshal(baselineSBOM)
	if err := os.WriteFile(baselinePath, data, 0644); err != nil {
		t.Fatalf("failed to create baseline SBOM: %v", err)
	}

	if err := detector.SaveBaseline(baselinePath, "test", "v1.0.0", "test"); err != nil {
		t.Fatalf("SaveBaseline() error = %v", err)
	}

	tests := []struct {
		name           string
		currentSBOM    *SimpleSBOM
		options        DriftOptions
		wantDrift      bool
		wantViolations int
		wantSeverity   string
	}{
		{
			name:           "no changes",
			currentSBOM:    createSimpleTestSBOM([]string{"pkg1", "pkg2", "pkg3"}, []string{"1.0.0", "2.0.0", "3.0.0"}),
			options:        DriftOptions{},
			wantDrift:      false,
			wantViolations: 0,
			wantSeverity:   "none",
		},
		{
			name:           "added component",
			currentSBOM:    createSimpleTestSBOM([]string{"pkg1", "pkg2", "pkg3", "pkg4"}, []string{"1.0.0", "2.0.0", "3.0.0", "4.0.0"}),
			options:        DriftOptions{},
			wantDrift:      true,
			wantViolations: 1,
			wantSeverity:   "medium",
		},
		{
			name:           "allowed addition",
			currentSBOM:    createSimpleTestSBOM([]string{"pkg1", "pkg2", "pkg3", "pkg4"}, []string{"1.0.0", "2.0.0", "3.0.0", "4.0.0"}),
			options:        DriftOptions{AllowedAdditions: []string{"pkg4"}},
			wantDrift:      false,
			wantViolations: 0,
			wantSeverity:   "none",
		},
		{
			name:           "removed component",
			currentSBOM:    createSimpleTestSBOM([]string{"pkg1", "pkg2"}, []string{"1.0.0", "2.0.0"}),
			options:        DriftOptions{},
			wantDrift:      true,
			wantViolations: 1,
			wantSeverity:   "high",
		},
		{
			name:           "allowed removal",
			currentSBOM:    createSimpleTestSBOM([]string{"pkg1", "pkg2"}, []string{"1.0.0", "2.0.0"}),
			options:        DriftOptions{AllowedRemovals: []string{"pkg3"}},
			wantDrift:      false,
			wantViolations: 0,
			wantSeverity:   "none",
		},
		{
			name:           "version upgrade disallowed",
			currentSBOM:    createSimpleTestSBOM([]string{"pkg1", "pkg2", "pkg3"}, []string{"1.1.0", "2.0.0", "3.0.0"}),
			options:        DriftOptions{AllowUpgrades: false},
			wantDrift:      true,
			wantViolations: 1,
			wantSeverity:   "low",
		},
		{
			name:           "version upgrade allowed",
			currentSBOM:    createSimpleTestSBOM([]string{"pkg1", "pkg2", "pkg3"}, []string{"1.1.0", "2.0.0", "3.0.0"}),
			options:        DriftOptions{AllowUpgrades: true},
			wantDrift:      false,
			wantViolations: 0,
			wantSeverity:   "none",
		},
		{
			name:           "version downgrade",
			currentSBOM:    createSimpleTestSBOM([]string{"pkg1", "pkg2", "pkg3"}, []string{"0.9.0", "2.0.0", "3.0.0"}),
			options:        DriftOptions{AllowDowngrades: false},
			wantDrift:      true,
			wantViolations: 1,
			wantSeverity:   "high",
		},
		{
			name:           "strict mode with changes",
			currentSBOM:    createSimpleTestSBOM([]string{"pkg1", "pkg2", "pkg3"}, []string{"1.1.0", "2.0.0", "3.0.0"}),
			options:        DriftOptions{StrictMode: true, AllowUpgrades: true},
			wantDrift:      true,
			wantViolations: 1,
			wantSeverity:   "high",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write current SBOM to file
			currentPath := filepath.Join(t.TempDir(), "current.json")
			data, _ := json.Marshal(tt.currentSBOM)
			if err := os.WriteFile(currentPath, data, 0644); err != nil {
				t.Fatalf("failed to write current SBOM: %v", err)
			}

			result, err := detector.DetectDrift(currentPath, "test", tt.options)
			if err != nil {
				t.Fatalf("DetectDrift() error = %v", err)
			}

			if result.HasDrift != tt.wantDrift {
				t.Errorf("HasDrift = %v, want %v", result.HasDrift, tt.wantDrift)
			}

			if len(result.Violations) != tt.wantViolations {
				t.Errorf("got %d violations, want %d", len(result.Violations), tt.wantViolations)
			}

			if result.DriftSummary.SeverityLevel != tt.wantSeverity {
				t.Errorf("severity = %q, want %q", result.DriftSummary.SeverityLevel, tt.wantSeverity)
			}
		})
	}
}

func TestDriftDetector_ListBaselines(t *testing.T) {
	baselineDir := t.TempDir()
	detector, err := NewDriftDetector(baselineDir)
	if err != nil {
		t.Fatalf("NewDriftDetector() error = %v", err)
	}

	// Initially empty
	baselines, err := detector.ListBaselines()
	if err != nil {
		t.Fatalf("ListBaselines() error = %v", err)
	}
	if len(baselines) != 0 {
		t.Errorf("expected 0 baselines, got %d", len(baselines))
	}

	// Create test SBOMs
	sbomPath1 := filepath.Join(t.TempDir(), "test1.json")
	sbom1 := createSimpleTestSBOM([]string{"pkg1"}, []string{"1.0.0"})
	data1, _ := json.Marshal(sbom1)
	os.WriteFile(sbomPath1, data1, 0644)

	sbomPath2 := filepath.Join(t.TempDir(), "test2.json")
	sbom2 := createSimpleTestSBOM([]string{"pkg2"}, []string{"2.0.0"})
	data2, _ := json.Marshal(sbom2)
	os.WriteFile(sbomPath2, data2, 0644)

	// Save baselines
	detector.SaveBaseline(sbomPath1, "baseline1", "v1.0.0", "first baseline")
	detector.SaveBaseline(sbomPath2, "baseline2", "v2.0.0", "second baseline")

	// List baselines
	baselines, err = detector.ListBaselines()
	if err != nil {
		t.Fatalf("ListBaselines() error = %v", err)
	}

	if len(baselines) != 2 {
		t.Fatalf("expected 2 baselines, got %d", len(baselines))
	}

	// Verify baseline metadata
	for _, baseline := range baselines {
		if baseline.ComponentCount == 0 {
			t.Error("baseline has 0 components")
		}
		if baseline.CreatedAt.IsZero() {
			t.Error("baseline CreatedAt is zero")
		}
		if !strings.HasSuffix(baseline.Path, ".baseline.json") {
			t.Errorf("baseline path doesn't end with .baseline.json: %s", baseline.Path)
		}
	}
}

func TestDriftDetector_DeleteBaseline(t *testing.T) {
	baselineDir := t.TempDir()
	detector, err := NewDriftDetector(baselineDir)
	if err != nil {
		t.Fatalf("NewDriftDetector() error = %v", err)
	}

	// Create a baseline
	sbomPath := filepath.Join(t.TempDir(), "test.json")
	sbom := createSimpleTestSBOM([]string{"pkg1"}, []string{"1.0.0"})
	data, _ := json.Marshal(sbom)
	os.WriteFile(sbomPath, data, 0644)
	detector.SaveBaseline(sbomPath, "test", "v1.0.0", "test")

	tests := []struct {
		name    string
		delName string
		wantErr bool
	}{
		{"valid delete", "test", false},
		{"empty name", "", true},
		{"non-existent", "nonexistent", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := detector.DeleteBaseline(tt.delName)
			if tt.wantErr {
				if err == nil {
					t.Error("DeleteBaseline() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("DeleteBaseline() unexpected error: %v", err)
			}

			// Verify file was deleted
			baselineFile := filepath.Join(baselineDir, tt.delName+".baseline.json")
			if _, err := os.Stat(baselineFile); !os.IsNotExist(err) {
				t.Error("baseline file still exists after deletion")
			}
		})
	}
}

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		pattern string
		str     string
		want    bool
	}{
		{"*", "anything", true},
		{"github.com/*", "github.com/user/repo", true},
		{"*/repo", "github.com/user/repo", true},
		{"*user*", "github.com/user/repo", true},
		{"exact", "exact", true},
		{"exact", "notexact", false},
		{"github.com/*", "gitlab.com/user/repo", false},
		{"*/specific", "github.com/different", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.str, func(t *testing.T) {
			got := matchPattern(tt.pattern, tt.str)
			if got != tt.want {
				t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.pattern, tt.str, got, tt.want)
			}
		})
	}
}

func TestFormatComponentName(t *testing.T) {
	tests := []struct {
		name  string
		group string
		want  string
	}{
		{"pkg", "github.com/user", "github.com/user/pkg"},
		{"pkg", "", "pkg"},
		{"module", "org.example", "org.example/module"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatComponentName(tt.name, tt.group)
			if got != tt.want {
				t.Errorf("formatComponentName(%q, %q) = %q, want %q", tt.name, tt.group, got, tt.want)
			}
		})
	}
}

func TestDriftSummary_SeverityCalculation(t *testing.T) {
	detector, _ := NewDriftDetector(t.TempDir())

	tests := []struct {
		name       string
		violations []DriftViolation
		want       string
	}{
		{
			name:       "no violations",
			violations: []DriftViolation{},
			want:       "none",
		},
		{
			name: "high severity",
			violations: []DriftViolation{
				{Severity: "high"},
			},
			want: "high",
		},
		{
			name: "medium severity",
			violations: []DriftViolation{
				{Severity: "medium"},
			},
			want: "medium",
		},
		{
			name: "low severity",
			violations: []DriftViolation{
				{Severity: "low"},
			},
			want: "low",
		},
		{
			name: "mixed severity - high wins",
			violations: []DriftViolation{
				{Severity: "low"},
				{Severity: "medium"},
				{Severity: "high"},
			},
			want: "high",
		},
		{
			name: "mixed severity - medium wins",
			violations: []DriftViolation{
				{Severity: "low"},
				{Severity: "medium"},
			},
			want: "medium",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detector.determineSeverityLevel(tt.violations)
			if got != tt.want {
				t.Errorf("determineSeverityLevel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDriftDetector_DetectDrift_InvalidBaseline(t *testing.T) {
	baselineDir := t.TempDir()
	detector, err := NewDriftDetector(baselineDir)
	if err != nil {
		t.Fatalf("NewDriftDetector() error = %v", err)
	}

	// Create current SBOM
	currentPath := filepath.Join(t.TempDir(), "current.json")
	currentSBOM := createSimpleTestSBOM([]string{"pkg1"}, []string{"1.0.0"})
	data, _ := json.Marshal(currentSBOM)
	os.WriteFile(currentPath, data, 0644)

	// Try to detect drift without baseline
	_, err = detector.DetectDrift(currentPath, "nonexistent", DriftOptions{})
	if err == nil {
		t.Error("DetectDrift() expected error for non-existent baseline, got nil")
	}
}

func TestDriftDetector_CreateTempSBOM(t *testing.T) {
	detector, _ := NewDriftDetector(t.TempDir())

	components := []Component{
		{Name: "pkg1", Version: "1.0.0"},
		{Name: "pkg2", Version: "2.0.0", Group: "github.com/test"},
	}

	tmpFile, err := detector.createTempSBOM(components)
	if err != nil {
		t.Fatalf("createTempSBOM() error = %v", err)
	}
	defer os.Remove(tmpFile)

	// Verify file exists and contains valid JSON
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read temp file: %v", err)
	}

	// Parse as raw JSON to check CycloneDX structure
	var cdxData map[string]interface{}
	if err := json.Unmarshal(data, &cdxData); err != nil {
		t.Fatalf("temp file contains invalid JSON: %v", err)
	}

	if cdxData["bomFormat"] != "CycloneDX" {
		t.Errorf("bomFormat = %q, want CycloneDX", cdxData["bomFormat"])
	}

	cdxComponents, ok := cdxData["components"].([]interface{})
	if !ok {
		t.Errorf("components field is not an array")
	} else if len(cdxComponents) != 2 {
		t.Errorf("components has %d items, want 2", len(cdxComponents))
	}
}

func TestDriftViolation_Types(t *testing.T) {
	baselineDir := t.TempDir()
	detector, _ := NewDriftDetector(baselineDir)

	// Create baseline with licenses - using proper CycloneDX format
	baselinePath := filepath.Join(t.TempDir(), "baseline.json")
	baselineCDX := map[string]interface{}{
		"bomFormat":   "CycloneDX",
		"specVersion": "1.4",
		"components": []map[string]interface{}{
			{
				"name":    "pkg1",
				"version": "1.0.0",
				"licenses": []map[string]interface{}{
					{
						"license": map[string]string{"id": "MIT"},
					},
				},
			},
		},
	}
	data, _ := json.Marshal(baselineCDX)
	os.WriteFile(baselinePath, data, 0644)
	detector.SaveBaseline(baselinePath, "test", "", "")

	// Test license change detection - using proper CycloneDX format
	currentPath := filepath.Join(t.TempDir(), "current.json")
	currentCDX := map[string]interface{}{
		"bomFormat":   "CycloneDX",
		"specVersion": "1.4",
		"components": []map[string]interface{}{
			{
				"name":    "pkg1",
				"version": "1.0.0",
				"licenses": []map[string]interface{}{
					{
						"license": map[string]string{"id": "Apache-2.0"},
					},
				},
			},
		},
	}
	data, _ = json.Marshal(currentCDX)
	os.WriteFile(currentPath, data, 0644)

	result, err := detector.DetectDrift(currentPath, "test", DriftOptions{
		AllowLicenseChanges: false, // Explicitly disallow license changes
	})
	if err != nil {
		t.Fatalf("DetectDrift() error = %v", err)
	}

	if !result.HasDrift {
		t.Error("expected drift for license change")
		t.Logf("Result: %+v", result)
		t.Logf("Changes: Added=%d, Removed=%d, Modified=%d",
			len(result.Changes.Added), len(result.Changes.Removed), len(result.Changes.Modified))
	}

	// Find license change violation
	var found bool
	for _, v := range result.Violations {
		t.Logf("Violation: Type=%s, Component=%s, Old=%s, New=%s", v.Type, v.Component, v.OldValue, v.NewValue)
		if v.Type == "license_change" {
			found = true
			if v.OldValue != "MIT" || v.NewValue != "Apache-2.0" {
				t.Errorf("license change violation has wrong values: %s -> %s", v.OldValue, v.NewValue)
			}
		}
	}

	if !found {
		t.Error("no license_change violation found")
	}
}

// Helper function to create test SBOMs with simple name/version arrays
func createSimpleTestSBOM(names []string, versions []string) *SimpleSBOM {
	components := make([]Component, len(names))
	for i := range names {
		components[i] = Component{
			Name:    names[i],
			Version: versions[i],
		}
	}
	return &SimpleSBOM{
		Format:      "CycloneDX",
		SpecVersion: "1.4",
		Components:  components,
	}
}

// Helper function to create test SBOMs from components
func createTestSBOMWithComponents(components []Component) *SimpleSBOM {
	return &SimpleSBOM{
		Format:      "CycloneDX",
		SpecVersion: "1.4",
		Components:  components,
	}
}

func TestDriftResult_JSON(t *testing.T) {
	result := &DriftResult{
		HasDrift: true,
		DriftSummary: DriftSummary{
			TotalChanges:       5,
			UnexpectedAdded:    2,
			UnexpectedRemoved:  1,
			UnexpectedUpgrades: 2,
			SeverityLevel:      "medium",
		},
		Baseline: BaselineMeta{
			Path:           "/path/to/baseline.json",
			CreatedAt:      time.Now(),
			Version:        "v1.0.0",
			ComponentCount: 10,
		},
		DetectedAt: time.Now(),
		Violations: []DriftViolation{
			{
				Type:      "added",
				Component: "new-pkg",
				NewValue:  "1.0.0",
				Severity:  "medium",
				Message:   "Unexpected dependency added",
			},
		},
	}

	// Test JSON marshaling
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal DriftResult: %v", err)
	}

	// Test JSON unmarshaling
	var decoded DriftResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal DriftResult: %v", err)
	}

	if decoded.HasDrift != result.HasDrift {
		t.Errorf("HasDrift = %v, want %v", decoded.HasDrift, result.HasDrift)
	}

	if decoded.DriftSummary.SeverityLevel != result.DriftSummary.SeverityLevel {
		t.Errorf("SeverityLevel = %q, want %q", decoded.DriftSummary.SeverityLevel, result.DriftSummary.SeverityLevel)
	}
}
