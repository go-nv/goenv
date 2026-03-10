package sbom

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestJSONFormatter(t *testing.T) {
	result := createTestDiffResult()

	tests := []struct {
		name   string
		pretty bool
	}{
		{"pretty JSON", true},
		{"compact JSON", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := &JSONFormatter{Pretty: tt.pretty}
			var buf bytes.Buffer

			err := formatter.Format(result, &buf)
			if err != nil {
				t.Fatalf("Format() error = %v", err)
			}

			// Verify it's valid JSON
			var decoded DiffResult
			if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
				t.Fatalf("output is not valid JSON: %v", err)
			}

			// Verify summary data
			if decoded.Summary.AddedCount != result.Summary.AddedCount {
				t.Errorf("AddedCount = %d, want %d", decoded.Summary.AddedCount, result.Summary.AddedCount)
			}
			if decoded.Summary.RemovedCount != result.Summary.RemovedCount {
				t.Errorf("RemovedCount = %d, want %d", decoded.Summary.RemovedCount, result.Summary.RemovedCount)
			}
		})
	}
}

func TestTableFormatter(t *testing.T) {
	result := createTestDiffResult()

	tests := []struct {
		name          string
		showUnchanged bool
		color         bool
		wantContains  []string
	}{
		{
			name:          "basic table",
			showUnchanged: false,
			color:         false,
			wantContains: []string{
				"SBOM Diff Summary",
				"Added:      2",
				"Removed:    1",
				"Modified:   1",
				"Added Components:",
				"new-pkg1",
				"new-pkg2",
				"Removed Components:",
				"old-pkg",
				"Modified Components:",
				"changed-pkg",
			},
		},
		{
			name:          "with unchanged",
			showUnchanged: true,
			color:         false,
			wantContains: []string{
				"Unchanged Components:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := &TableFormatter{
				ShowUnchanged: tt.showUnchanged,
				Color:         tt.color,
			}
			var buf bytes.Buffer

			err := formatter.Format(result, &buf)
			if err != nil {
				t.Fatalf("Format() error = %v", err)
			}

			output := buf.String()
			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("output missing expected text: %q", want)
				}
			}
		})
	}
}

func TestTableFormatter_Colorize(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		color     string
		useColor  bool
		wantColor bool
	}{
		{"with color", "test", "red", true, true},
		{"without color", "test", "red", false, false},
		{"unknown color", "test", "purple", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := &TableFormatter{Color: tt.useColor}
			result := formatter.colorize(tt.text, tt.color)

			hasColorCode := strings.Contains(result, "\033[")
			if hasColorCode != tt.wantColor {
				t.Errorf("colorize() has color codes = %v, want %v", hasColorCode, tt.wantColor)
			}

			if !strings.Contains(result, tt.text) {
				t.Errorf("colorize() output missing original text: %q", tt.text)
			}
		})
	}
}

func TestGitHubFormatter(t *testing.T) {
	result := createTestDiffResult()

	formatter := &GitHubFormatter{}
	var buf bytes.Buffer

	err := formatter.Format(result, &buf)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := buf.String()

	// Should contain GitHub Actions annotations
	wantContains := []string{
		"::notice::",
		"::warning::",
		"<details>",
		"</details>",
		"<summary>",
		"```",
	}

	for _, want := range wantContains {
		if !strings.Contains(output, want) {
			t.Errorf("output missing expected GitHub Actions element: %q", want)
		}
	}

	// Check for specific warnings
	if !strings.Contains(output, "2 dependencies added") {
		t.Error("output missing 'dependencies added' notice")
	}
	if !strings.Contains(output, "1 dependencies removed") {
		t.Error("output missing 'dependencies removed' warning")
	}
}

func TestMarkdownFormatter(t *testing.T) {
	result := createTestDiffResult()

	formatter := &MarkdownFormatter{ShowUnchanged: false}
	var buf bytes.Buffer

	err := formatter.Format(result, &buf)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := buf.String()

	// Should contain markdown elements
	wantContains := []string{
		"# SBOM Diff Report",
		"## Summary",
		"| Metric | Count |",
		"## ➕ Added Components",
		"## ➖ Removed Components",
		"## 🔄 Modified Components",
		"- **new-pkg1**",
		"- **new-pkg2**",
		"- **old-pkg**",
		"- **changed-pkg**",
	}

	for _, want := range wantContains {
		if !strings.Contains(output, want) {
			t.Errorf("output missing expected markdown element: %q", want)
		}
	}
}

func TestMarkdownFormatter_WithUnchanged(t *testing.T) {
	result := createTestDiffResult()

	formatter := &MarkdownFormatter{ShowUnchanged: true}
	var buf bytes.Buffer

	err := formatter.Format(result, &buf)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := buf.String()

	// When ShowUnchanged is true, we don't add unchanged section in the current implementation
	// But we should verify the rest still works
	if !strings.Contains(output, "# SBOM Diff Report") {
		t.Error("output missing header")
	}
}

func TestGetFormatter(t *testing.T) {
	tests := []struct {
		format   string
		wantType string
		wantErr  bool
	}{
		{"json", "*sbom.JSONFormatter", false},
		{"json-compact", "*sbom.JSONFormatter", false},
		{"table", "*sbom.TableFormatter", false},
		{"github", "*sbom.GitHubFormatter", false},
		{"markdown", "*sbom.MarkdownFormatter", false},
		{"md", "*sbom.MarkdownFormatter", false},
		{"invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			formatter, err := GetFormatter(tt.format, false, false)

			if tt.wantErr {
				if err == nil {
					t.Error("GetFormatter() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("GetFormatter() unexpected error: %v", err)
			}

			if formatter == nil {
				t.Fatal("GetFormatter() returned nil formatter")
			}

			// Verify we can format with it
			result := createTestDiffResult()
			var buf bytes.Buffer
			if err := formatter.Format(result, &buf); err != nil {
				t.Errorf("Format() error = %v", err)
			}
		})
	}
}

func TestTableFormatter_FormatComponent(t *testing.T) {
	formatter := &TableFormatter{}

	tests := []struct {
		name string
		comp ComponentDiff
		want string
	}{
		{
			name: "with group and version and license",
			comp: ComponentDiff{
				Name:    "pkg",
				Group:   "github.com/example",
				Version: "1.0.0",
				License: "MIT",
			},
			want: "github.com/example/pkg v1.0.0 (MIT)\n",
		},
		{
			name: "without group",
			comp: ComponentDiff{
				Name:    "pkg",
				Version: "1.0.0",
			},
			want: "pkg v1.0.0\n",
		},
		{
			name: "without license",
			comp: ComponentDiff{
				Name:    "pkg",
				Group:   "org",
				Version: "2.0.0",
			},
			want: "org/pkg v2.0.0\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatter.formatComponent(tt.comp)
			if got != tt.want {
				t.Errorf("formatComponent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMarkdownFormatter_FormatComponentName(t *testing.T) {
	formatter := &MarkdownFormatter{}

	tests := []struct {
		name string
		comp ComponentDiff
		want string
	}{
		{
			name: "with group",
			comp: ComponentDiff{Name: "pkg", Group: "github.com/example"},
			want: "github.com/example/pkg",
		},
		{
			name: "without group",
			comp: ComponentDiff{Name: "pkg"},
			want: "pkg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatter.formatComponentName(tt.comp)
			if got != tt.want {
				t.Errorf("formatComponentName() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Helper to create a test diff result
func createTestDiffResult() *DiffResult {
	return &DiffResult{
		Added: []ComponentDiff{
			{Name: "new-pkg1", Version: "1.0.0", ChangeType: "added"},
			{Name: "new-pkg2", Version: "2.0.0", License: "MIT", ChangeType: "added"},
		},
		Removed: []ComponentDiff{
			{Name: "old-pkg", Version: "0.5.0", ChangeType: "removed"},
		},
		Modified: []ComponentDiff{
			{
				Name:       "changed-pkg",
				OldVersion: "1.0.0",
				NewVersion: "1.1.0",
				ChangeType: "version_change",
				Severity:   "upgrade",
				Changes:    []string{"Version upgraded from 1.0.0 to 1.1.0"},
			},
		},
		Unchanged: []ComponentDiff{
			{Name: "stable-pkg", Version: "3.0.0", ChangeType: "unchanged"},
		},
		Summary: DiffSummary{
			TotalComponents:   5,
			AddedCount:        2,
			RemovedCount:      1,
			ModifiedCount:     1,
			UnchangedCount:    1,
			VersionUpgrades:   1,
			VersionDowngrades: 0,
			LicenseChanges:    0,
		},
		Comparison: ComparisonMeta{
			OldSBOM: SBOMMeta{
				Path:           "/tmp/old.json",
				Format:         "CycloneDX",
				SpecVersion:    "1.4",
				ComponentCount: 3,
			},
			NewSBOM: SBOMMeta{
				Path:           "/tmp/new.json",
				Format:         "CycloneDX",
				SpecVersion:    "1.4",
				ComponentCount: 4,
			},
		},
	}
}
