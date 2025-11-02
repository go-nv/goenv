package manager

import (
	"path/filepath"
	"testing"

	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverVersion(t *testing.T) {
	var err error
	// Create temp directory
	tmpDir := t.TempDir()

	tests := []struct {
		name           string
		setupFiles     func(string)
		expectedVer    string
		expectedSource VersionSource
		expectNil      bool
		expectError    bool
	}{
		{
			name: "finds .go-version",
			setupFiles: func(dir string) {
				testutil.WriteTestFile(t, filepath.Join(dir, ".go-version"), []byte("1.24.1\n"), utils.PermFileDefault)
			},
			expectedVer:    "1.24.1",
			expectedSource: SourceGoVersion,
		},
		{
			name: "finds go.mod with go directive",
			setupFiles: func(dir string) {
				testutil.WriteTestFile(t, filepath.Join(dir, "go.mod"), []byte("module test\n\ngo 1.22\n"), utils.PermFileDefault)
			},
			expectedVer:    "1.22",
			expectedSource: SourceGoMod,
		},
		{
			name: "finds go.mod with toolchain (takes precedence)",
			setupFiles: func(dir string) {
				content := "module test\n\ngo 1.22\n\ntoolchain go1.22.5\n"
				testutil.WriteTestFile(t, filepath.Join(dir, "go.mod"), []byte(content), utils.PermFileDefault)
			},
			expectedVer:    "1.22.5",
			expectedSource: SourceGoMod,
		},
		{
			name: ".go-version takes precedence over go.mod",
			setupFiles: func(dir string) {
				testutil.WriteTestFile(t, filepath.Join(dir, ".go-version"), []byte("1.24.1\n"), utils.PermFileDefault)
				testutil.WriteTestFile(t, filepath.Join(dir, "go.mod"), []byte("module test\n\ngo 1.22\n"), utils.PermFileDefault)
			},
			expectedVer:    "1.24.1",
			expectedSource: SourceGoVersion,
		},
		{
			name: "returns nil when no version files",
			setupFiles: func(dir string) {
			},
			expectNil: true,
		},
		{
			name: "skips empty .go-version, checks go.mod",
			setupFiles: func(dir string) {
				testutil.WriteTestFile(t, filepath.Join(dir, ".go-version"), []byte("\n\n"), utils.PermFileDefault)
				testutil.WriteTestFile(t, filepath.Join(dir, "go.mod"), []byte("module test\n\ngo 1.22\n"), utils.PermFileDefault)
			},
			expectedVer:    "1.22",
			expectedSource: SourceGoMod,
		},
		{
			name: "skips comments in .go-version",
			setupFiles: func(dir string) {
				testutil.WriteTestFile(t, filepath.Join(dir, ".go-version"), []byte("# comment\n1.24.1\n"), utils.PermFileDefault)
			},
			expectedVer:    "1.24.1",
			expectedSource: SourceGoVersion,
		},
		{
			name: "handles Windows line endings (CRLF) in .go-version",
			setupFiles: func(dir string) {
				testutil.WriteTestFile(t, filepath.Join(dir, ".go-version"), []byte("1.24.1\r\n"), utils.PermFileDefault)
			},
			expectedVer:    "1.24.1",
			expectedSource: SourceGoVersion,
		},
		{
			name: "handles Windows line endings in go.mod with toolchain",
			setupFiles: func(dir string) {
				content := "module test\r\n\r\ngo 1.22\r\n\r\ntoolchain go1.22.5\r\n"
				testutil.WriteTestFile(t, filepath.Join(dir, "go.mod"), []byte(content), utils.PermFileDefault)
			},
			expectedVer:    "1.22.5",
			expectedSource: SourceGoMod,
		},
		{
			name: "go.mod toolchain overrides older .go-version",
			setupFiles: func(dir string) {
				testutil.WriteTestFile(t, filepath.Join(dir, ".go-version"), []byte("1.23\n"), utils.PermFileDefault)
				testutil.WriteTestFile(t, filepath.Join(dir, "go.mod"), []byte("module test\n\ngo 1.22\n\ntoolchain go1.24.1\n"), utils.PermFileDefault)
			},
			expectedVer:    "1.24.1",
			expectedSource: SourceGoMod,
		},
		{
			name: ".go-version used when equal to go.mod toolchain",
			setupFiles: func(dir string) {
				testutil.WriteTestFile(t, filepath.Join(dir, ".go-version"), []byte("1.24.1\n"), utils.PermFileDefault)
				testutil.WriteTestFile(t, filepath.Join(dir, "go.mod"), []byte("module test\n\ngo 1.22\n\ntoolchain go1.24.1\n"), utils.PermFileDefault)
			},
			expectedVer:    "1.24.1",
			expectedSource: SourceGoVersion, // User's explicit choice when versions match
		},
		{
			name: ".go-version used when newer than go.mod toolchain",
			setupFiles: func(dir string) {
				testutil.WriteTestFile(t, filepath.Join(dir, ".go-version"), []byte("1.25\n"), utils.PermFileDefault)
				testutil.WriteTestFile(t, filepath.Join(dir, "go.mod"), []byte("module test\n\ngo 1.22\n\ntoolchain go1.24.1\n"), utils.PermFileDefault)
			},
			expectedVer:    "1.25",
			expectedSource: SourceGoVersion, // User explicitly wants newer version
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test directory
			testDir := filepath.Join(tmpDir, tt.name)
			err = utils.EnsureDirWithContext(testDir, "create test directory")
			require.NoError(t, err)

			// Setup files
			if tt.setupFiles != nil {
				tt.setupFiles(testDir)
			}

			// Discover version
			result, err := DiscoverVersion(testDir)

			// Check error
			if tt.expectError {
				assert.Error(t, err, "expected error, got nil")
				return
			}
			require.NoError(t, err)

			// Check nil result
			if tt.expectNil {
				assert.Nil(t, result, "expected nil result")
				return
			}

			// Check result
			require.NotNil(t, result, "expected result, got nil")
			assert.Equal(t, tt.expectedVer, result.Version, "expected version")
			assert.Equal(t, tt.expectedSource, result.Source, "expected source")
		})
	}
}

func TestDiscoverVersionMismatch(t *testing.T) {
	var err error
	// Create temp directory
	tmpDir := t.TempDir()

	tests := []struct {
		name           string
		setupFiles     func(string)
		expectMismatch bool
		goVersionVer   string
		goModVer       string
	}{
		{
			name: "mismatch between .go-version and go.mod",
			setupFiles: func(dir string) {
				testutil.WriteTestFile(t, filepath.Join(dir, ".go-version"), []byte("1.24.1\n"), utils.PermFileDefault)
				testutil.WriteTestFile(t, filepath.Join(dir, "go.mod"), []byte("module test\n\ngo 1.22\n"), utils.PermFileDefault)
			},
			expectMismatch: true,
			goVersionVer:   "1.24.1",
			goModVer:       "1.22",
		},
		{
			name: "mismatch with toolchain in go.mod",
			setupFiles: func(dir string) {
				testutil.WriteTestFile(t, filepath.Join(dir, ".go-version"), []byte("1.24.1\n"), utils.PermFileDefault)
				content := "module test\n\ngo 1.22\n\ntoolchain go1.22.5\n"
				testutil.WriteTestFile(t, filepath.Join(dir, "go.mod"), []byte(content), utils.PermFileDefault)
			},
			expectMismatch: true,
			goVersionVer:   "1.24.1",
			goModVer:       "1.22.5", // toolchain takes precedence
		},
		{
			name: "no mismatch when versions match",
			setupFiles: func(dir string) {
				testutil.WriteTestFile(t, filepath.Join(dir, ".go-version"), []byte("1.22\n"), utils.PermFileDefault)
				testutil.WriteTestFile(t, filepath.Join(dir, "go.mod"), []byte("module test\n\ngo 1.22\n"), utils.PermFileDefault)
			},
			expectMismatch: false,
			goVersionVer:   "1.22",
			goModVer:       "1.22",
		},
		{
			name: "no mismatch when only .go-version exists",
			setupFiles: func(dir string) {
				testutil.WriteTestFile(t, filepath.Join(dir, ".go-version"), []byte("1.24.1\n"), utils.PermFileDefault)
			},
			expectMismatch: false,
			goVersionVer:   "1.24.1",
			goModVer:       "",
		},
		{
			name: "no mismatch when only go.mod exists",
			setupFiles: func(dir string) {
				testutil.WriteTestFile(t, filepath.Join(dir, "go.mod"), []byte("module test\n\ngo 1.22\n"), utils.PermFileDefault)
			},
			expectMismatch: false,
			goVersionVer:   "",
			goModVer:       "1.22",
		},
		{
			name: "no mismatch when neither exists",
			setupFiles: func(dir string) {
			},
			expectMismatch: false,
			goVersionVer:   "",
			goModVer:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test directory
			testDir := filepath.Join(tmpDir, tt.name)
			err = utils.EnsureDirWithContext(testDir, "create test directory")
			require.NoError(t, err)

			// Setup files
			if tt.setupFiles != nil {
				tt.setupFiles(testDir)
			}

			// Check for mismatch
			mismatch, goVersionVer, goModVer, err := DiscoverVersionMismatch(testDir)
			require.NoError(t, err)

			assert.Equal(t, tt.expectMismatch, mismatch, "expected mismatch=")
			assert.Equal(t, tt.goVersionVer, goVersionVer, "expected goVersionVer=")
			assert.Equal(t, tt.goModVer, goModVer, "expected goModVer=")
		})
	}
}

func TestParseVersionContent(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "simple version",
			content:  "1.24.1\n",
			expected: "1.24.1",
		},
		{
			name:     "version with whitespace",
			content:  "  1.24.1  \n",
			expected: "1.24.1",
		},
		{
			name:     "version with comment before",
			content:  "# comment\n1.24.1\n",
			expected: "1.24.1",
		},
		{
			name:     "version with empty lines",
			content:  "\n\n1.24.1\n",
			expected: "1.24.1",
		},
		{
			name:     "only comments",
			content:  "# comment\n# another comment\n",
			expected: "",
		},
		{
			name:     "empty content",
			content:  "",
			expected: "",
		},
		{
			name:     "only whitespace",
			content:  "  \n  \n",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseVersionContent(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}
