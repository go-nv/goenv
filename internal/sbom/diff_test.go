package sbom

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDiffSBOMs(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name           string
		oldComponents  []Component
		newComponents  []Component
		wantAdded      int
		wantRemoved    int
		wantModified   int
		wantUpgrades   int
		wantDowngrades int
	}{
		{
			name: "identical SBOMs",
			oldComponents: []Component{
				{Name: "pkg1", Version: "1.0.0", License: "MIT"},
				{Name: "pkg2", Version: "2.0.0", License: "Apache-2.0"},
			},
			newComponents: []Component{
				{Name: "pkg1", Version: "1.0.0", License: "MIT"},
				{Name: "pkg2", Version: "2.0.0", License: "Apache-2.0"},
			},
			wantAdded:    0,
			wantRemoved:  0,
			wantModified: 0,
		},
		{
			name: "added component",
			oldComponents: []Component{
				{Name: "pkg1", Version: "1.0.0"},
			},
			newComponents: []Component{
				{Name: "pkg1", Version: "1.0.0"},
				{Name: "pkg2", Version: "2.0.0"},
			},
			wantAdded:    1,
			wantRemoved:  0,
			wantModified: 0,
		},
		{
			name: "removed component",
			oldComponents: []Component{
				{Name: "pkg1", Version: "1.0.0"},
				{Name: "pkg2", Version: "2.0.0"},
			},
			newComponents: []Component{
				{Name: "pkg1", Version: "1.0.0"},
			},
			wantAdded:    0,
			wantRemoved:  1,
			wantModified: 0,
		},
		{
			name: "version upgrade",
			oldComponents: []Component{
				{Name: "pkg1", Version: "1.0.0"},
			},
			newComponents: []Component{
				{Name: "pkg1", Version: "1.1.0"},
			},
			wantAdded:      0,
			wantRemoved:    0,
			wantModified:   1,
			wantUpgrades:   1,
			wantDowngrades: 0,
		},
		{
			name: "version downgrade",
			oldComponents: []Component{
				{Name: "pkg1", Version: "2.0.0"},
			},
			newComponents: []Component{
				{Name: "pkg1", Version: "1.0.0"},
			},
			wantAdded:      0,
			wantRemoved:    0,
			wantModified:   1,
			wantUpgrades:   0,
			wantDowngrades: 1,
		},
		{
			name: "license change",
			oldComponents: []Component{
				{Name: "pkg1", Version: "1.0.0", License: "MIT"},
			},
			newComponents: []Component{
				{Name: "pkg1", Version: "1.0.0", License: "Apache-2.0"},
			},
			wantAdded:    0,
			wantRemoved:  0,
			wantModified: 1,
		},
		{
			name: "complex changes",
			oldComponents: []Component{
				{Name: "pkg1", Version: "1.0.0", License: "MIT"},
				{Name: "pkg2", Version: "2.0.0", License: "Apache-2.0"},
				{Name: "pkg3", Version: "3.0.0"},
			},
			newComponents: []Component{
				{Name: "pkg1", Version: "1.1.0", License: "MIT"},     // upgraded
				{Name: "pkg2", Version: "2.0.0", License: "GPL-3.0"}, // license changed
				{Name: "pkg4", Version: "4.0.0"},                     // added
			},
			wantAdded:      1,
			wantRemoved:    1,
			wantModified:   2,
			wantUpgrades:   1,
			wantDowngrades: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test SBOM files
			oldPath := filepath.Join(tmpDir, "old.json")
			newPath := filepath.Join(tmpDir, "new.json")

			createTestSBOM(t, oldPath, tt.oldComponents)
			createTestSBOM(t, newPath, tt.newComponents)

			// Run diff
			opts := &DiffOptions{}
			result, err := DiffSBOMs(oldPath, newPath, opts)
			if err != nil {
				t.Fatalf("DiffSBOMs() error = %v", err)
			}

			// Verify counts
			if got := result.Summary.AddedCount; got != tt.wantAdded {
				t.Errorf("AddedCount = %d, want %d", got, tt.wantAdded)
			}
			if got := result.Summary.RemovedCount; got != tt.wantRemoved {
				t.Errorf("RemovedCount = %d, want %d", got, tt.wantRemoved)
			}
			if got := result.Summary.ModifiedCount; got != tt.wantModified {
				t.Errorf("ModifiedCount = %d, want %d", got, tt.wantModified)
			}
			if got := result.Summary.VersionUpgrades; got != tt.wantUpgrades {
				t.Errorf("VersionUpgrades = %d, want %d", got, tt.wantUpgrades)
			}
			if got := result.Summary.VersionDowngrades; got != tt.wantDowngrades {
				t.Errorf("VersionDowngrades = %d, want %d", got, tt.wantDowngrades)
			}
		})
	}
}

func TestDetermineVersionChangeSeverity(t *testing.T) {
	tests := []struct {
		oldVer string
		newVer string
		want   string
	}{
		{"1.0.0", "1.0.0", "unchanged"},
		{"1.0.0", "1.1.0", "upgrade"},
		{"1.1.0", "1.0.0", "downgrade"},
		{"v1.0.0", "v1.1.0", "upgrade"},
		{"v2.0.0", "v1.0.0", "downgrade"},
		{"0.1.0", "0.2.0", "upgrade"},
	}

	for _, tt := range tests {
		t.Run(tt.oldVer+"->"+tt.newVer, func(t *testing.T) {
			got := determineVersionChangeSeverity(tt.oldVer, tt.newVer)
			if got != tt.want {
				t.Errorf("determineVersionChangeSeverity(%q, %q) = %q, want %q",
					tt.oldVer, tt.newVer, got, tt.want)
			}
		})
	}
}

func TestComponentKey(t *testing.T) {
	tests := []struct {
		group string
		name  string
		want  string
	}{
		{"github.com/example", "pkg", "github.com/example/pkg"},
		{"", "pkg", "pkg"},
		{"org.apache", "commons", "org.apache/commons"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := componentKey(tt.group, tt.name)
			if got != tt.want {
				t.Errorf("componentKey(%q, %q) = %q, want %q",
					tt.group, tt.name, got, tt.want)
			}
		})
	}
}

func TestLoadSBOMForDiff(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		components  []Component
		wantCount   int
		wantFormat  string
		wantSpecVer string
	}{
		{
			name: "basic SBOM",
			components: []Component{
				{Name: "pkg1", Version: "1.0.0", Group: "github.com/example"},
				{Name: "pkg2", Version: "2.0.0", License: "MIT"},
			},
			wantCount:   2,
			wantFormat:  "CycloneDX",
			wantSpecVer: "1.4",
		},
		{
			name:        "empty SBOM",
			components:  []Component{},
			wantCount:   0,
			wantFormat:  "CycloneDX",
			wantSpecVer: "1.4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(tmpDir, "test.json")
			createTestSBOM(t, path, tt.components)

			sbom, err := loadSBOMForDiff(path)
			if err != nil {
				t.Fatalf("loadSBOMForDiff() error = %v", err)
			}

			if got := len(sbom.Components); got != tt.wantCount {
				t.Errorf("component count = %d, want %d", got, tt.wantCount)
			}
			if sbom.Format != tt.wantFormat {
				t.Errorf("format = %q, want %q", sbom.Format, tt.wantFormat)
			}
			if sbom.SpecVersion != tt.wantSpecVer {
				t.Errorf("specVersion = %q, want %q", sbom.SpecVersion, tt.wantSpecVer)
			}
		})
	}
}

func TestLoadSBOMForDiff_InvalidFile(t *testing.T) {
	_, err := loadSBOMForDiff("/nonexistent/file.json")
	if err == nil {
		t.Error("loadSBOMForDiff() expected error for nonexistent file")
	}
}

func TestLoadSBOMForDiff_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "invalid.json")

	if err := os.WriteFile(path, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err := loadSBOMForDiff(path)
	if err == nil {
		t.Error("loadSBOMForDiff() expected error for invalid JSON")
	}
}

func TestCompareComponents(t *testing.T) {
	tests := []struct {
		name        string
		old         *Component
		new         *Component
		opts        *DiffOptions
		wantDiff    bool
		wantType    string
		wantChanges int
	}{
		{
			name:     "no changes",
			old:      &Component{Name: "pkg", Version: "1.0.0", License: "MIT"},
			new:      &Component{Name: "pkg", Version: "1.0.0", License: "MIT"},
			opts:     &DiffOptions{},
			wantDiff: false,
		},
		{
			name:        "version change",
			old:         &Component{Name: "pkg", Version: "1.0.0"},
			new:         &Component{Name: "pkg", Version: "1.1.0"},
			opts:        &DiffOptions{},
			wantDiff:    true,
			wantType:    "version_change",
			wantChanges: 1,
		},
		{
			name:        "license change",
			old:         &Component{Name: "pkg", Version: "1.0.0", License: "MIT"},
			new:         &Component{Name: "pkg", Version: "1.0.0", License: "Apache-2.0"},
			opts:        &DiffOptions{},
			wantDiff:    true,
			wantType:    "license_change",
			wantChanges: 1,
		},
		{
			name:     "license change ignored",
			old:      &Component{Name: "pkg", Version: "1.0.0", License: "MIT"},
			new:      &Component{Name: "pkg", Version: "1.0.0", License: "Apache-2.0"},
			opts:     &DiffOptions{IgnoreLicenses: true},
			wantDiff: false,
		},
		{
			name:        "version and license change",
			old:         &Component{Name: "pkg", Version: "1.0.0", License: "MIT"},
			new:         &Component{Name: "pkg", Version: "1.1.0", License: "Apache-2.0"},
			opts:        &DiffOptions{},
			wantDiff:    true,
			wantType:    "version_change",
			wantChanges: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := compareComponents(tt.old, tt.new, tt.opts)

			if tt.wantDiff && diff == nil {
				t.Error("compareComponents() expected diff, got nil")
				return
			}
			if !tt.wantDiff && diff != nil {
				t.Errorf("compareComponents() expected no diff, got %+v", diff)
				return
			}

			if !tt.wantDiff {
				return
			}

			if diff.ChangeType != tt.wantType {
				t.Errorf("ChangeType = %q, want %q", diff.ChangeType, tt.wantType)
			}
			if got := len(diff.Changes); got != tt.wantChanges {
				t.Errorf("Changes count = %d, want %d", got, tt.wantChanges)
			}
		})
	}
}

func TestCalculateDiffSummary(t *testing.T) {
	result := &DiffResult{
		Added:   []ComponentDiff{{Name: "pkg1"}, {Name: "pkg2"}},
		Removed: []ComponentDiff{{Name: "pkg3"}},
		Modified: []ComponentDiff{
			{Name: "pkg4", Severity: "upgrade", OldLicense: "MIT", NewLicense: "Apache-2.0"},
			{Name: "pkg5", Severity: "downgrade"},
			{Name: "pkg6", OldLicense: "GPL", NewLicense: "MIT"},
		},
		Unchanged: []ComponentDiff{{Name: "pkg7"}, {Name: "pkg8"}},
	}

	summary := calculateDiffSummary(result)

	if summary.AddedCount != 2 {
		t.Errorf("AddedCount = %d, want 2", summary.AddedCount)
	}
	if summary.RemovedCount != 1 {
		t.Errorf("RemovedCount = %d, want 1", summary.RemovedCount)
	}
	if summary.ModifiedCount != 3 {
		t.Errorf("ModifiedCount = %d, want 3", summary.ModifiedCount)
	}
	if summary.UnchangedCount != 2 {
		t.Errorf("UnchangedCount = %d, want 2", summary.UnchangedCount)
	}
	if summary.TotalComponents != 8 {
		t.Errorf("TotalComponents = %d, want 8", summary.TotalComponents)
	}
	if summary.VersionUpgrades != 1 {
		t.Errorf("VersionUpgrades = %d, want 1", summary.VersionUpgrades)
	}
	if summary.VersionDowngrades != 1 {
		t.Errorf("VersionDowngrades = %d, want 1", summary.VersionDowngrades)
	}
	if summary.LicenseChanges != 2 {
		t.Errorf("LicenseChanges = %d, want 2", summary.LicenseChanges)
	}
}

// Helper function to create test SBOM files
func createTestSBOM(t *testing.T, path string, components []Component) {
	t.Helper()

	type cdxComponent struct {
		Name       string `json:"name"`
		Group      string `json:"group,omitempty"`
		Version    string `json:"version"`
		PackageURL string `json:"purl,omitempty"`
		Licenses   []struct {
			License struct {
				ID   string `json:"id,omitempty"`
				Name string `json:"name,omitempty"`
			} `json:"license,omitempty"`
		} `json:"licenses,omitempty"`
	}

	type cdxSBOM struct {
		BomFormat   string         `json:"bomFormat"`
		SpecVersion string         `json:"specVersion"`
		Components  []cdxComponent `json:"components"`
	}

	sbom := cdxSBOM{
		BomFormat:   "CycloneDX",
		SpecVersion: "1.4",
		Components:  make([]cdxComponent, 0, len(components)),
	}

	for _, comp := range components {
		cdxComp := cdxComponent{
			Name:       comp.Name,
			Group:      comp.Group,
			Version:    comp.Version,
			PackageURL: comp.PackageURL,
		}

		if comp.License != "" {
			cdxComp.Licenses = []struct {
				License struct {
					ID   string `json:"id,omitempty"`
					Name string `json:"name,omitempty"`
				} `json:"license,omitempty"`
			}{
				{
					License: struct {
						ID   string `json:"id,omitempty"`
						Name string `json:"name,omitempty"`
					}{
						ID: comp.License,
					},
				},
			}
		}

		sbom.Components = append(sbom.Components, cdxComp)
	}

	data, err := json.MarshalIndent(sbom, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal SBOM: %v", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write SBOM file: %v", err)
	}
}
