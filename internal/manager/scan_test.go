package manager

import (
	"path/filepath"
	"testing"

	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanProjects(t *testing.T) {
	var err error
	// Create test directory structure
	tmpDir := t.TempDir()

	// Project 1: .go-version
	proj1 := filepath.Join(tmpDir, "proj1")
	err = utils.EnsureDirWithContext(proj1, "create test directory")
	require.NoError(t, err)
	testutil.WriteTestFile(t, filepath.Join(proj1, ".go-version"), []byte("1.21.5\n"), utils.PermFileDefault)

	// Project 2: go.mod
	proj2 := filepath.Join(tmpDir, "proj2")
	err = utils.EnsureDirWithContext(proj2, "create test directory")
	require.NoError(t, err)
	gomodContent := "module test\n\ngo 1.22.3\n"
	testutil.WriteTestFile(t, filepath.Join(proj2, "go.mod"), []byte(gomodContent), utils.PermFileDefault)

	// Project 3: go.mod with toolchain
	proj3 := filepath.Join(tmpDir, "proj3")
	err = utils.EnsureDirWithContext(proj3, "create test directory")
	require.NoError(t, err)
	gomodWithToolchain := "module test3\n\ngo 1.21.0\n\ntoolchain go1.23.2\n"
	testutil.WriteTestFile(t, filepath.Join(proj3, "go.mod"), []byte(gomodWithToolchain), utils.PermFileDefault)

	// Scan with no depth limit
	projects, err := ScanProjects(tmpDir, 0)
	require.NoError(t, err)

	// Verify we found all projects
	assert.Len(t, projects, 3, "Expected 3 projects")

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
	var err error
	tmpDir := t.TempDir()

	// Project at depth 1
	proj1 := filepath.Join(tmpDir, "proj1")
	_ = utils.EnsureDirWithContext(proj1, "create test directory")
	testutil.WriteTestFile(t, filepath.Join(proj1, ".go-version"), []byte("1.21.5\n"), utils.PermFileDefault)

	// Project at depth 3
	proj2 := filepath.Join(tmpDir, "a", "b", "proj2")
	_ = utils.EnsureDirWithContext(proj2, "create test directory")
	testutil.WriteTestFile(t, filepath.Join(proj2, ".go-version"), []byte("1.22.3\n"), utils.PermFileDefault)

	// Project at depth 5
	proj3 := filepath.Join(tmpDir, "a", "b", "c", "d", "proj3")
	_ = utils.EnsureDirWithContext(proj3, "create test directory")
	testutil.WriteTestFile(t, filepath.Join(proj3, ".go-version"), []byte("1.23.2\n"), utils.PermFileDefault)

	// Scan with depth 3 - should find proj1 and proj2, but not proj3
	projects, err := ScanProjects(tmpDir, 3)
	require.NoError(t, err)

	assert.Len(t, projects, 2, "With depth 3, expected 2 projects")

	// Verify we didn't find the deep project
	for _, proj := range projects {
		assert.NotEqual(t, proj3, proj.Path, "Should not have found project at depth 5 with maxDepth=3")
	}

	// Scan with depth 5 - should find all
	projects, err = ScanProjects(tmpDir, 5)
	require.NoError(t, err)

	assert.Len(t, projects, 3, "With depth 5, expected 3 projects")
}

func TestScanProjectsSkipsCommonDirs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .go-version in node_modules (should be skipped)
	nmDir := filepath.Join(tmpDir, "node_modules", "something")
	_ = utils.EnsureDirWithContext(nmDir, "create test directory")
	testutil.WriteTestFile(t, filepath.Join(nmDir, ".go-version"), []byte("1.21.5\n"), utils.PermFileDefault)

	// Create .go-version in vendor (should be skipped)
	vendorDir := filepath.Join(tmpDir, "vendor", "github.com", "something")
	_ = utils.EnsureDirWithContext(vendorDir, "create test directory")
	testutil.WriteTestFile(t, filepath.Join(vendorDir, ".go-version"), []byte("1.22.3\n"), utils.PermFileDefault)

	// Create .go-version in .git (should be skipped)
	gitDir := filepath.Join(tmpDir, ".git", "something")
	_ = utils.EnsureDirWithContext(gitDir, "create test directory")
	testutil.WriteTestFile(t, filepath.Join(gitDir, ".go-version"), []byte("1.23.2\n"), utils.PermFileDefault)

	// Create valid project at root
	testutil.WriteTestFile(t, filepath.Join(tmpDir, ".go-version"), []byte("1.20.5\n"), utils.PermFileDefault)

	// Scan
	projects, err := ScanProjects(tmpDir, 0)
	require.NoError(t, err)

	// Should only find the root project
	assert.Len(t, projects, 1, "Expected 1 project (skipping node_modules, vendor, .git)")

	assert.False(t, len(projects) > 0 && projects[0].Version != "1.20.5", "Expected to find root project with version 1.20.5")
}

func TestScanProjectsEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	// Empty directory
	projects, err := ScanProjects(tmpDir, 0)
	require.NoError(t, err)

	assert.Len(t, projects, 0, "Expected 0 projects in empty directory")
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
	testutil.WriteTestFile(t, versionFile, []byte(content), utils.PermFileDefault)

	// Scan
	projects, err := ScanProjects(tmpDir, 0)
	require.NoError(t, err)

	if len(projects) != 1 {
		t.Fatalf("Expected 1 project, got %d", len(projects))
	}

	assert.Equal(t, "1.21.5", projects[0].Version, "Expected version 1.21.5")
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
			testutil.WriteTestFile(t, tmpFile, []byte(tt.content), utils.PermFileDefault)

			version, err := readVersionFile(tmpFile)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, version)
		})
	}
}

func TestScanProjectsNonexistentDir(t *testing.T) {
	// Scan nonexistent directory - should return empty slice without panic
	// The function is designed to be resilient and continue scanning even with errors
	projects, _ := ScanProjects("/nonexistent/path/that/should/not/exist", 0)

	// Should return empty slice (not nil)
	assert.NotNil(t, projects, "Expected non-nil slice even for nonexistent directory")
	assert.Len(t, projects, 0, "Expected 0 projects for nonexistent directory")
}
