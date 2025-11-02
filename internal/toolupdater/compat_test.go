package toolupdater

import (
	"os"
	"testing"

	"github.com/go-nv/goenv/internal/utils"
)

func TestParseGoVersion(t *testing.T) {
	tests := []struct {
		name      string
		version   string
		wantMajor int
		wantMinor int
		wantErr   bool
	}{
		{
			name:      "standard version",
			version:   "1.21.5",
			wantMajor: 1,
			wantMinor: 21,
			wantErr:   false,
		},
		{
			name:      "version with go prefix",
			version:   "go1.20.2",
			wantMajor: 1,
			wantMinor: 20,
			wantErr:   false,
		},
		{
			name:      "short version",
			version:   "1.21",
			wantMajor: 1,
			wantMinor: 21,
			wantErr:   false,
		},
		{
			name:      "rc version",
			version:   "1.22rc1",
			wantMajor: 1,
			wantMinor: 22,
			wantErr:   false,
		},
		{
			name:      "beta version",
			version:   "1.23beta2",
			wantMajor: 1,
			wantMinor: 23,
			wantErr:   false,
		},
		{
			name:    "invalid format",
			version: "invalid",
			wantErr: true,
		},
		{
			name:    "empty string",
			version: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			major, minor, err := parseGoVersion(tt.version)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if major != tt.wantMajor {
				t.Errorf("Expected major %d, got %d", tt.wantMajor, major)
			}

			if minor != tt.wantMinor {
				t.Errorf("Expected minor %d, got %d", tt.wantMinor, minor)
			}
		})
	}
}

func TestCompareGoVersions(t *testing.T) {
	tests := []struct {
		name string
		v1   string
		v2   string
		want int // -1, 0, or 1
	}{
		{
			name: "equal versions",
			v1:   "1.21.5",
			v2:   "1.21.5",
			want: 0,
		},
		{
			name: "v1 less than v2 (minor)",
			v1:   "1.20.0",
			v2:   "1.21.0",
			want: -1,
		},
		{
			name: "v1 greater than v2 (minor)",
			v1:   "1.22.0",
			v2:   "1.21.0",
			want: 1,
		},
		{
			name: "v1 less than v2 (major)",
			v1:   "1.21.0",
			v2:   "2.0.0",
			want: -1,
		},
		{
			name: "v1 greater than v2 (major)",
			v1:   "2.0.0",
			v2:   "1.21.0",
			want: 1,
		},
		{
			name: "with go prefix",
			v1:   "go1.21.0",
			v2:   "go1.21.0",
			want: 0,
		},
		{
			name: "different formats same version",
			v1:   "1.21",
			v2:   "1.21.0",
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := utils.CompareGoVersions(tt.v1, tt.v2)

			if got != tt.want {
				t.Errorf("utils.CompareGoVersions(%s, %s) = %d, want %d", tt.v1, tt.v2, got, tt.want)
			}
		})
	}
}

func TestIsGoVersionCompatible(t *testing.T) {
	tests := []struct {
		name string
		v1   string
		v2   string
		want bool
	}{
		{
			name: "same major.minor",
			v1:   "1.21.5",
			v2:   "1.21.3",
			want: true,
		},
		{
			name: "different minor",
			v1:   "1.21.0",
			v2:   "1.22.0",
			want: false,
		},
		{
			name: "different major",
			v1:   "1.21.0",
			v2:   "2.0.0",
			want: false,
		},
		{
			name: "with go prefix",
			v1:   "go1.21.5",
			v2:   "go1.21.0",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsGoVersionCompatible(tt.v1, tt.v2)

			if got != tt.want {
				t.Errorf("IsGoVersionCompatible(%s, %s) = %v, want %v", tt.v1, tt.v2, got, tt.want)
			}
		})
	}
}

func TestCheckGoVersionRequirement(t *testing.T) {
	tests := []struct {
		name       string
		required   string
		current    string
		wantCompat bool
		wantReason string
	}{
		{
			name:       "current meets requirement",
			required:   "1.20",
			current:    "1.21.5",
			wantCompat: true,
			wantReason: "",
		},
		{
			name:       "current equals requirement",
			required:   "1.21",
			current:    "1.21.0",
			wantCompat: true,
			wantReason: "",
		},
		{
			name:       "current below requirement (minor)",
			required:   "1.22",
			current:    "1.21.5",
			wantCompat: false,
		},
		{
			name:       "current below requirement (major)",
			required:   "2.0",
			current:    "1.21.5",
			wantCompat: false,
		},
		{
			name:       "with go prefixes",
			required:   "go1.20",
			current:    "go1.21",
			wantCompat: true,
			wantReason: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCompat, gotReason := checkGoVersionRequirement(tt.required, tt.current)

			if gotCompat != tt.wantCompat {
				t.Errorf("checkGoVersionRequirement(%s, %s) compatibility = %v, want %v",
					tt.required, tt.current, gotCompat, tt.wantCompat)
			}

			if !tt.wantCompat && gotReason == "" {
				t.Error("Expected reason for incompatibility but got empty string")
			}

			if tt.wantCompat && gotReason != "" {
				t.Errorf("Expected no reason for compatible versions but got: %s", gotReason)
			}
		})
	}
}

func TestValidateToolVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    bool
	}{
		{
			name:    "valid semver",
			version: "v1.2.3",
			want:    true,
		},
		{
			name:    "valid semver with patch",
			version: "v0.14.2",
			want:    true,
		},
		{
			name:    "latest",
			version: "latest",
			want:    true,
		},
		{
			name:    "empty",
			version: "",
			want:    true,
		},
		{
			name:    "without v prefix",
			version: "1.2.3",
			want:    false,
		},
		{
			name:    "invalid format",
			version: "v1.x.3",
			want:    false,
		},
		{
			name:    "just v",
			version: "v",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateToolVersion(tt.version)

			if got != tt.want {
				t.Errorf("ValidateToolVersion(%s) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestCheckCompatibility(t *testing.T) {
	tests := []struct {
		name        string
		packagePath string
		toolVersion string
		goVersion   string
		wantCompat  bool
		skipCI      bool // Some tests require network/go command
	}{
		{
			name:        "empty requirements always compatible",
			packagePath: "",
			toolVersion: "",
			goVersion:   "1.21.5",
			wantCompat:  true,
		},
		{
			name:        "non-queryable package assumes compatible",
			packagePath: "nonexistent.example.com/tool",
			toolVersion: "v1.0.0",
			goVersion:   "1.21.5",
			wantCompat:  true, // Fail open
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipCI && isCI() {
				t.Skip("Skipping network test in CI")
			}

			gotCompat, gotReason := CheckCompatibility(tt.packagePath, tt.toolVersion, tt.goVersion)

			if gotCompat != tt.wantCompat {
				t.Errorf("CheckCompatibility() compatibility = %v, want %v (reason: %s)",
					gotCompat, tt.wantCompat, gotReason)
			}
		})
	}
}

func TestCheckCompatibilityDetailed(t *testing.T) {
	tests := []struct {
		name        string
		packagePath string
		toolVersion string
		goVersion   string
		wantCompat  bool
		skipCI      bool
	}{
		{
			name:        "non-existent package",
			packagePath: "nonexistent.example.com/tool",
			toolVersion: "v1.0.0",
			goVersion:   "1.21.5",
			wantCompat:  true, // Fail open with warning in reason
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipCI && isCI() {
				t.Skip("Skipping network test in CI")
			}

			result, err := CheckCompatibilityDetailed(tt.packagePath, tt.toolVersion, tt.goVersion)
			if err != nil {
				t.Fatalf("CheckCompatibilityDetailed failed: %v", err)
			}

			if result.Compatible != tt.wantCompat {
				t.Errorf("CheckCompatibilityDetailed() compatibility = %v, want %v (reason: %s)",
					result.Compatible, tt.wantCompat, result.Reason)
			}
		})
	}
}

func TestGetCompatibleGoVersions(t *testing.T) {
	installedVersions := []string{"1.20.0", "1.21.5", "1.22.0"}

	tests := []struct {
		name            string
		packagePath     string
		toolVersion     string
		wantMinVersions int // Minimum number of compatible versions expected
		skipCI          bool
	}{
		{
			name:            "non-existent package returns all versions",
			packagePath:     "nonexistent.example.com/tool",
			toolVersion:     "v1.0.0",
			wantMinVersions: 3, // All installed versions
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipCI && isCI() {
				t.Skip("Skipping network test in CI")
			}

			compatible, err := GetCompatibleGoVersions(tt.packagePath, tt.toolVersion, installedVersions)
			if err != nil {
				t.Fatalf("GetCompatibleGoVersions failed: %v", err)
			}

			if len(compatible) < tt.wantMinVersions {
				t.Errorf("Expected at least %d compatible versions, got %d",
					tt.wantMinVersions, len(compatible))
			}
		})
	}
}

func TestSuggestGoVersionForTool(t *testing.T) {
	installedVersions := []string{"1.20.0", "1.21.5", "1.22.0"}

	tests := []struct {
		name        string
		packagePath string
		toolVersion string
		wantVersion string // Expected suggested version (or "" to skip exact check)
		skipCI      bool
	}{
		{
			name:        "suggests latest compatible version",
			packagePath: "nonexistent.example.com/tool",
			toolVersion: "v1.0.0",
			wantVersion: "1.22.0", // Latest of the installed versions
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipCI && isCI() {
				t.Skip("Skipping network test in CI")
			}

			suggested, err := SuggestGoVersionForTool(tt.packagePath, tt.toolVersion, installedVersions)
			if err != nil {
				t.Fatalf("SuggestGoVersionForTool failed: %v", err)
			}

			if tt.wantVersion != "" && suggested != tt.wantVersion {
				t.Errorf("Expected suggested version %s, got %s", tt.wantVersion, suggested)
			}

			// Verify suggested version is in installed list
			found := false
			for _, v := range installedVersions {
				if v == suggested {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Suggested version %s not in installed versions", suggested)
			}
		})
	}
}

func TestSuggestGoVersionForToolNoCompatible(t *testing.T) {
	// Empty installed versions should return error
	_, err := SuggestGoVersionForTool("some/package", "v1.0.0", []string{})
	if err == nil {
		t.Error("Expected error for no installed versions")
	}
}

// Helper function to detect CI environment
func isCI() bool {
	ciEnvVars := []string{
		"CI",
		"CONTINUOUS_INTEGRATION",
		"GITHUB_ACTIONS",
		"GITLAB_CI",
		"CIRCLECI",
		"TRAVIS",
	}

	for _, envVar := range ciEnvVars {
		if os.Getenv(envVar) != "" {
			return true
		}
	}
	return false
}
