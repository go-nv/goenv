package manager

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanProjects(t *testing.T) {
	// Create test directory structure
	tmpDir := t.TempDir()

	// Project 1: .go-version
	proj1 := filepath.Join(tmpDir, "proj1")
	if err := os.MkdirAll(proj1, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proj1, ".go-version"), []byte("1.21.5\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Project 2: go.mod
	proj2 := filepath.Join(tmpDir, "proj2")
	if err := os.MkdirAll(proj2, 0755); err != nil {
		t.Fatal(err)
	}
	gomodContent := "module test\n\ngo 1.22.3\n"
	if err := os.WriteFile(filepath.Join(proj2, "go.mod"), []byte(gomodContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Project 3: go.mod with toolchain
	proj3 := filepath.Join(tmpDir, "proj3")
	if err := os.MkdirAll(proj3, 0755); err != nil {
		t.Fatal(err)
	}
	gomodWithToolchain := "module test3\n\ngo 1.21.0\n\ntoolchain go1.23.2\n"
	if err := os.WriteFile(filepath.Join(proj3, "go.mod"), []byte(gomodWithToolchain), 0644); err != nil {
		t.Fatal(err)
	}

	// Scan with no depth limit
	projects, err := ScanProjects(tmpDir, 0)
	if err != nil {
		t.Fatal(err)
	}

	// Verify we found all projects
	if len(projects) != 3 {
		t.Errorf("Expected 3 projects, got %d", len(projects))
	}

	// Build map for easier verification
	versionMap := make(map[string]string) // path -> version
	sourceMap := make(map[string]string)  // path -> source

	for _, proj := range projects {
		versionMap[proj.Path] = proj.Version
		sourceMap[proj.Path] = proj.Source
	}

	// Verify Project 1
	if ver, ok := versionMap[proj1]; !ok || ver != "1.21.5" {
		t.Errorf("Project 1: expected version 1.21.5, got %v", ver)
	}
	if src, ok := sourceMap[proj1]; !ok || src != ".go-version" {
		t.Errorf("Project 1: expected source .go-version, got %v", src)
	}

	// Verify Project 2
	if ver, ok := versionMap[proj2]; !ok || ver != "1.22.3" {
		t.Errorf("Project 2: expected version 1.22.3, got %v", ver)
	}
	if src, ok := sourceMap[proj2]; !ok || src != "go.mod" {
		t.Errorf("Project 2: expected source go.mod, got %v", src)
	}

	// Verify Project 3 (toolchain should take precedence)
	if ver, ok := versionMap[proj3]; !ok || ver != "1.23.2" {
		t.Errorf("Project 3: expected version 1.23.2 (toolchain), got %v", ver)
	}
}

func TestScanProjectsWithDepthLimit(t *testing.T) {
	tmpDir := t.TempDir()

	// Project at depth 1
	proj1 := filepath.Join(tmpDir, "proj1")
	os.MkdirAll(proj1, 0755)
	os.WriteFile(filepath.Join(proj1, ".go-version"), []byte("1.21.5\n"), 0644)

	// Project at depth 3
	proj2 := filepath.Join(tmpDir, "a", "b", "proj2")
	os.MkdirAll(proj2, 0755)
	os.WriteFile(filepath.Join(proj2, ".go-version"), []byte("1.22.3\n"), 0644)

	// Project at depth 5
	proj3 := filepath.Join(tmpDir, "a", "b", "c", "d", "proj3")
	os.MkdirAll(proj3, 0755)
	os.WriteFile(filepath.Join(proj3, ".go-version"), []byte("1.23.2\n"), 0644)

	// Scan with depth 3 - should find proj1 and proj2, but not proj3
	projects, err := ScanProjects(tmpDir, 3)
	if err != nil {
		t.Fatal(err)
	}

	if len(projects) != 2 {
		t.Errorf("With depth 3, expected 2 projects, got %d", len(projects))
	}

	// Verify we didn't find the deep project
	for _, proj := range projects {
		if proj.Path == proj3 {
			t.Error("Should not have found project at depth 5 with maxDepth=3")
		}
	}

	// Scan with depth 5 - should find all
	projects, err = ScanProjects(tmpDir, 5)
	if err != nil {
		t.Fatal(err)
	}

	if len(projects) != 3 {
		t.Errorf("With depth 5, expected 3 projects, got %d", len(projects))
	}
}

func TestScanProjectsSkipsCommonDirs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .go-version in node_modules (should be skipped)
	nmDir := filepath.Join(tmpDir, "node_modules", "something")
	os.MkdirAll(nmDir, 0755)
	os.WriteFile(filepath.Join(nmDir, ".go-version"), []byte("1.21.5\n"), 0644)

	// Create .go-version in vendor (should be skipped)
	vendorDir := filepath.Join(tmpDir, "vendor", "github.com", "something")
	os.MkdirAll(vendorDir, 0755)
	os.WriteFile(filepath.Join(vendorDir, ".go-version"), []byte("1.22.3\n"), 0644)

	// Create .go-version in .git (should be skipped)
	gitDir := filepath.Join(tmpDir, ".git", "something")
	os.MkdirAll(gitDir, 0755)
	os.WriteFile(filepath.Join(gitDir, ".go-version"), []byte("1.23.2\n"), 0644)

	// Create valid project at root
	os.WriteFile(filepath.Join(tmpDir, ".go-version"), []byte("1.20.5\n"), 0644)

	// Scan
	projects, err := ScanProjects(tmpDir, 0)
	if err != nil {
		t.Fatal(err)
	}

	// Should only find the root project
	if len(projects) != 1 {
		t.Errorf("Expected 1 project (skipping node_modules, vendor, .git), got %d", len(projects))
	}

	if len(projects) > 0 && projects[0].Version != "1.20.5" {
		t.Errorf("Expected to find root project with version 1.20.5, got %s", projects[0].Version)
	}
}

func TestScanProjectsEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	// Empty directory
	projects, err := ScanProjects(tmpDir, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(projects) != 0 {
		t.Errorf("Expected 0 projects in empty directory, got %d", len(projects))
	}
}

func TestScanProjectsWithComments(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .go-version with comments
	versionFile := filepath.Join(tmpDir, ".go-version")
	content := `# This is a comment
# Another comment
1.21.5
# Trailing comment
`
	os.WriteFile(versionFile, []byte(content), 0644)

	// Scan
	projects, err := ScanProjects(tmpDir, 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(projects) != 1 {
		t.Fatalf("Expected 1 project, got %d", len(projects))
	}

	if projects[0].Version != "1.21.5" {
		t.Errorf("Expected version 1.21.5, got %s", projects[0].Version)
	}
}

func TestReadVersionFile(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "simple version",
			content:  "1.21.5\n",
			expected: "1.21.5",
		},
		{
			name:     "version with whitespace",
			content:  "  1.21.5  \n",
			expected: "1.21.5",
		},
		{
			name:     "version with comments",
			content:  "# Comment\n1.21.5\n",
			expected: "1.21.5",
		},
		{
			name:     "empty lines before version",
			content:  "\n\n1.21.5\n",
			expected: "1.21.5",
		},
		{
			name:     "only comments",
			content:  "# Only comments\n# Nothing else\n",
			expected: "",
		},
		{
			name:     "empty file",
			content:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := filepath.Join(t.TempDir(), ".go-version")
			if err := os.WriteFile(tmpFile, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			version, err := readVersionFile(tmpFile)
			if err != nil {
				t.Fatal(err)
			}

			if version != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, version)
			}
		})
	}
}

func TestScanProjectsNonexistentDir(t *testing.T) {
	// Scan nonexistent directory - should return empty slice without panic
	// The function is designed to be resilient and continue scanning even with errors
	projects, _ := ScanProjects("/nonexistent/path/that/should/not/exist", 0)

	// Should return empty slice (not nil)
	if projects == nil {
		t.Error("Expected non-nil slice even for nonexistent directory")
	}
	if len(projects) != 0 {
		t.Errorf("Expected 0 projects for nonexistent directory, got %d", len(projects))
	}
}
