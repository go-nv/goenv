package manager

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverVersion(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "goenv-test-discovery-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name           string
		setupFiles     func(string) error
		expectedVer    string
		expectedSource VersionSource
		expectNil      bool
		expectError    bool
	}{
		{
			name: "finds .go-version",
			setupFiles: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, ".go-version"), []byte("1.24.1\n"), 0644)
			},
			expectedVer:    "1.24.1",
			expectedSource: SourceGoVersion,
		},
		{
			name: "finds go.mod with go directive",
			setupFiles: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n\ngo 1.22\n"), 0644)
			},
			expectedVer:    "1.22",
			expectedSource: SourceGoMod,
		},
		{
			name: "finds go.mod with toolchain (takes precedence)",
			setupFiles: func(dir string) error {
				content := "module test\n\ngo 1.22\n\ntoolchain go1.22.5\n"
				return os.WriteFile(filepath.Join(dir, "go.mod"), []byte(content), 0644)
			},
			expectedVer:    "1.22.5",
			expectedSource: SourceGoMod,
		},
		{
			name: ".go-version takes precedence over go.mod",
			setupFiles: func(dir string) error {
				if err := os.WriteFile(filepath.Join(dir, ".go-version"), []byte("1.24.1\n"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n\ngo 1.22\n"), 0644)
			},
			expectedVer:    "1.24.1",
			expectedSource: SourceGoVersion,
		},
		{
			name: "returns nil when no version files",
			setupFiles: func(dir string) error {
				return nil // no files
			},
			expectNil: true,
		},
		{
			name: "skips empty .go-version, checks go.mod",
			setupFiles: func(dir string) error {
				if err := os.WriteFile(filepath.Join(dir, ".go-version"), []byte("\n\n"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n\ngo 1.22\n"), 0644)
			},
			expectedVer:    "1.22",
			expectedSource: SourceGoMod,
		},
		{
			name: "skips comments in .go-version",
			setupFiles: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, ".go-version"), []byte("# comment\n1.24.1\n"), 0644)
			},
			expectedVer:    "1.24.1",
			expectedSource: SourceGoVersion,
		},
		{
			name: "handles Windows line endings (CRLF) in .go-version",
			setupFiles: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, ".go-version"), []byte("1.24.1\r\n"), 0644)
			},
			expectedVer:    "1.24.1",
			expectedSource: SourceGoVersion,
		},
		{
			name: "handles Windows line endings in go.mod with toolchain",
			setupFiles: func(dir string) error {
				content := "module test\r\n\r\ngo 1.22\r\n\r\ntoolchain go1.22.5\r\n"
				return os.WriteFile(filepath.Join(dir, "go.mod"), []byte(content), 0644)
			},
			expectedVer:    "1.22.5",
			expectedSource: SourceGoMod,
		},
		{
			name: "go.mod toolchain overrides older .go-version",
			setupFiles: func(dir string) error {
				if err := os.WriteFile(filepath.Join(dir, ".go-version"), []byte("1.23\n"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n\ngo 1.22\n\ntoolchain go1.24.1\n"), 0644)
			},
			expectedVer:    "1.24.1",
			expectedSource: SourceGoMod,
		},
		{
			name: ".go-version used when equal to go.mod toolchain",
			setupFiles: func(dir string) error {
				if err := os.WriteFile(filepath.Join(dir, ".go-version"), []byte("1.24.1\n"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n\ngo 1.22\n\ntoolchain go1.24.1\n"), 0644)
			},
			expectedVer:    "1.24.1",
			expectedSource: SourceGoVersion, // User's explicit choice when versions match
		},
		{
			name: ".go-version used when newer than go.mod toolchain",
			setupFiles: func(dir string) error {
				if err := os.WriteFile(filepath.Join(dir, ".go-version"), []byte("1.25\n"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n\ngo 1.22\n\ntoolchain go1.24.1\n"), 0644)
			},
			expectedVer:    "1.25",
			expectedSource: SourceGoVersion, // User explicitly wants newer version
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test directory
			testDir := filepath.Join(tmpDir, tt.name)
			if err := os.MkdirAll(testDir, 0755); err != nil {
				t.Fatal(err)
			}

			// Setup files
			if tt.setupFiles != nil {
				if err := tt.setupFiles(testDir); err != nil {
					t.Fatal(err)
				}
			}

			// Discover version
			result, err := DiscoverVersion(testDir)

			// Check error
			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check nil result
			if tt.expectNil {
				if result != nil {
					t.Errorf("expected nil result, got %+v", result)
				}
				return
			}

			// Check result
			if result == nil {
				t.Fatal("expected result, got nil")
			}
			if result.Version != tt.expectedVer {
				t.Errorf("expected version %q, got %q", tt.expectedVer, result.Version)
			}
			if result.Source != tt.expectedSource {
				t.Errorf("expected source %q, got %q", tt.expectedSource, result.Source)
			}
		})
	}
}

func TestDiscoverVersionMismatch(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "goenv-test-mismatch-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name           string
		setupFiles     func(string) error
		expectMismatch bool
		goVersionVer   string
		goModVer       string
	}{
		{
			name: "mismatch between .go-version and go.mod",
			setupFiles: func(dir string) error {
				if err := os.WriteFile(filepath.Join(dir, ".go-version"), []byte("1.24.1\n"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n\ngo 1.22\n"), 0644)
			},
			expectMismatch: true,
			goVersionVer:   "1.24.1",
			goModVer:       "1.22",
		},
		{
			name: "mismatch with toolchain in go.mod",
			setupFiles: func(dir string) error {
				if err := os.WriteFile(filepath.Join(dir, ".go-version"), []byte("1.24.1\n"), 0644); err != nil {
					return err
				}
				content := "module test\n\ngo 1.22\n\ntoolchain go1.22.5\n"
				return os.WriteFile(filepath.Join(dir, "go.mod"), []byte(content), 0644)
			},
			expectMismatch: true,
			goVersionVer:   "1.24.1",
			goModVer:       "1.22.5", // toolchain takes precedence
		},
		{
			name: "no mismatch when versions match",
			setupFiles: func(dir string) error {
				if err := os.WriteFile(filepath.Join(dir, ".go-version"), []byte("1.22\n"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n\ngo 1.22\n"), 0644)
			},
			expectMismatch: false,
			goVersionVer:   "1.22",
			goModVer:       "1.22",
		},
		{
			name: "no mismatch when only .go-version exists",
			setupFiles: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, ".go-version"), []byte("1.24.1\n"), 0644)
			},
			expectMismatch: false,
			goVersionVer:   "1.24.1",
			goModVer:       "",
		},
		{
			name: "no mismatch when only go.mod exists",
			setupFiles: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n\ngo 1.22\n"), 0644)
			},
			expectMismatch: false,
			goVersionVer:   "",
			goModVer:       "1.22",
		},
		{
			name: "no mismatch when neither exists",
			setupFiles: func(dir string) error {
				return nil
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
			if err := os.MkdirAll(testDir, 0755); err != nil {
				t.Fatal(err)
			}

			// Setup files
			if tt.setupFiles != nil {
				if err := tt.setupFiles(testDir); err != nil {
					t.Fatal(err)
				}
			}

			// Check for mismatch
			mismatch, goVersionVer, goModVer, err := DiscoverVersionMismatch(testDir)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if mismatch != tt.expectMismatch {
				t.Errorf("expected mismatch=%v, got %v", tt.expectMismatch, mismatch)
			}
			if goVersionVer != tt.goVersionVer {
				t.Errorf("expected goVersionVer=%q, got %q", tt.goVersionVer, goVersionVer)
			}
			if goModVer != tt.goModVer {
				t.Errorf("expected goModVer=%q, got %q", tt.goModVer, goModVer)
			}
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
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
