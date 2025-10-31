package lifecycle

import (
	"strings"
	"testing"
	"time"

	"github.com/go-nv/goenv/internal/utils"
)

func TestExtractMajorMinor(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{"simple version", "1.21.5", "1.21"},
		{"major.minor only", "1.22", "1.22"},
		{"with go prefix", "go1.21.5", "1.21"},
		{"with v prefix", "v1.21.5", "1.21"},
		{"with rc suffix", "1.23.0-rc1", "1.23"},
		{"with beta suffix", "1.24.0-beta1", "1.24"},
		{"invalid single digit", "1", ""},
		{"invalid non-numeric", "abc", ""},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := utils.ExtractMajorMinor(tt.version)
			if got != tt.expected {
				t.Errorf("utils.ExtractMajorMinor(%q) = %q, want %q", tt.version, got, tt.expected)
			}
		})
	}
}

func TestGetVersionInfo(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		expectFound bool
		expectEOL   bool
	}{
		{"current version 1.25", "1.25.3", true, false},
		{"current version 1.24", "1.24.1", true, false},
		{"near EOL version 1.23", "1.23.2", true, false},
		{"EOL version 1.22", "1.22.8", true, true},
		{"EOL version 1.21", "1.21.5", true, true},
		{"EOL version 1.20", "1.20.12", true, true},
		{"unknown future version", "1.99.0", false, false},
		{"invalid version", "invalid", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, found := GetVersionInfo(tt.version)
			if found != tt.expectFound {
				t.Errorf("GetVersionInfo(%q) found = %v, want %v", tt.version, found, tt.expectFound)
			}
			if found && tt.expectEOL {
				if info.Status != StatusEOL {
					t.Errorf("GetVersionInfo(%q) status = %v, want StatusEOL", tt.version, info.Status)
				}
			}
		})
	}
}

func TestIsSupported(t *testing.T) {
	tests := []struct {
		name      string
		version   string
		supported bool
	}{
		{"current version 1.25", "1.25.3", true},
		{"current version 1.24", "1.24.1", true},
		{"near EOL 1.23 (still supported with security updates)", "1.23.2", true},
		{"EOL version 1.22", "1.22.8", false},
		{"old EOL version 1.20", "1.20.5", false},
		{"unknown future version (assumed supported)", "1.99.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsSupported(tt.version)
			if got != tt.supported {
				t.Errorf("IsSupported(%q) = %v, want %v", tt.version, got, tt.supported)
			}
		})
	}
}

func TestIsEOL(t *testing.T) {
	tests := []struct {
		name    string
		version string
		isEOL   bool
	}{
		{"current version 1.25", "1.25.3", false},
		{"current version 1.24", "1.24.1", false},
		{"near EOL 1.23 (not yet EOL)", "1.23.2", false},
		{"EOL version 1.22", "1.22.8", true},
		{"old EOL version 1.19", "1.19.5", true},
		{"unknown version", "1.99.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsEOL(tt.version)
			if got != tt.isEOL {
				t.Errorf("IsEOL(%q) = %v, want %v", tt.version, got, tt.isEOL)
			}
		})
	}
}

func TestIsNearEOL(t *testing.T) {
	tests := []struct {
		name      string
		version   string
		isNearEOL bool
	}{
		{"current version", "1.25.3", false},
		{"near EOL version", "1.23.2", true},
		{"already EOL", "1.22.8", false},
		{"unknown version", "1.99.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNearEOL(tt.version)
			if got != tt.isNearEOL {
				t.Errorf("IsNearEOL(%q) = %v, want %v", tt.version, got, tt.isNearEOL)
			}
		})
	}
}

func TestGetRecommendedVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
	}{
		{"EOL version 1.22", "1.22.8", "1.25"},
		{"near EOL version 1.23", "1.23.2", "1.25"},
		{"current version 1.25 (has recommended upgrade)", "1.25.3", ""},
		{"current version 1.24 (no recommendation)", "1.24.1", ""},
		{"unknown version", "1.99.0", "latest stable version"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetRecommendedVersion(tt.version)
			if got != tt.want {
				t.Errorf("GetRecommendedVersion(%q) = %q, want %q", tt.version, got, tt.want)
			}
		})
	}
}

func TestFormatWarning(t *testing.T) {
	tests := []struct {
		name           string
		version        string
		expectContains []string
		expectEmpty    bool
	}{
		{
			name:           "EOL version warning",
			version:        "1.22.8",
			expectContains: []string{"1.22", "no longer supported", "EOL:", "1.25"},
		},
		{
			name:           "near EOL version warning",
			version:        "1.23.2",
			expectContains: []string{"1.23", "support ends soon", "1.25"},
		},
		{
			name:        "current version - no warning",
			version:     "1.25.3",
			expectEmpty: true,
		},
		{
			name:        "unknown version - no warning",
			version:     "1.99.0",
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatWarning(tt.version)

			if tt.expectEmpty {
				if got != "" {
					t.Errorf("FormatWarning(%q) = %q, want empty string", tt.version, got)
				}
				return
			}

			for _, substr := range tt.expectContains {
				if !strings.Contains(got, substr) {
					t.Errorf("FormatWarning(%q) = %q, want to contain %q", tt.version, got, substr)
				}
			}
		})
	}
}

func TestCalculateStatus(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		info     VersionInfo
		expected SupportStatus
	}{
		{
			name: "EOL in the past",
			info: VersionInfo{
				EOLDate: now.AddDate(0, -6, 0), // 6 months ago
			},
			expected: StatusEOL,
		},
		{
			name: "EOL in 1 month (near EOL)",
			info: VersionInfo{
				EOLDate: now.AddDate(0, 1, 0),
			},
			expected: StatusNearEOL,
		},
		{
			name: "EOL in 6 months (current)",
			info: VersionInfo{
				EOLDate: now.AddDate(0, 6, 0),
			},
			expected: StatusCurrent,
		},
		{
			name: "EOL in 1 year (current)",
			info: VersionInfo{
				EOLDate: now.AddDate(1, 0, 0),
			},
			expected: StatusCurrent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateStatus(tt.info)
			if got != tt.expected {
				t.Errorf("calculateStatus() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestVersionLifecycleData(t *testing.T) {
	// Verify that lifecycle data is well-formed
	for version, info := range versionLifecycle {
		t.Run("version_"+version, func(t *testing.T) {
			if info.Version != version {
				t.Errorf("Version key %q doesn't match info.Version %q", version, info.Version)
			}

			if info.ReleaseDate.IsZero() {
				t.Errorf("Version %q has zero ReleaseDate", version)
			}

			if info.EOLDate.IsZero() {
				t.Errorf("Version %q has zero EOLDate", version)
			}

			if info.EOLDate.Before(info.ReleaseDate) {
				t.Errorf("Version %q has EOLDate before ReleaseDate", version)
			}

			// EOL versions should have a recommended version
			if info.Status == StatusEOL && info.Recommended == "" {
				t.Errorf("EOL version %q has no recommended upgrade", version)
			}
		})
	}
}
