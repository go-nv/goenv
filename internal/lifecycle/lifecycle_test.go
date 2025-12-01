package lifecycle

import (
	"testing"
	"time"

	"github.com/go-nv/goenv/internal/utils"
	"github.com/stretchr/testify/assert"
)

// setupTestLifecycleData initializes lifecycle data with test-specific values
func setupTestLifecycleData() {
	lifecycleMutex.Lock()
	defer lifecycleMutex.Unlock()

	// Set up test data with dates relative to current time
	now := time.Now()

	versionLifecycle = map[string]VersionInfo{
		"1.25": {
			Version:      "1.25",
			ReleaseDate:  now.AddDate(0, -3, 0), // 3 months ago
			EOLDate:      now.AddDate(1, 0, 0),  // 1 year from now
			Status:       StatusCurrent,
			Recommended:  "",
			SecurityOnly: false,
		},
		"1.24": {
			Version:      "1.24",
			ReleaseDate:  now.AddDate(0, -9, 0),  // 9 months ago
			EOLDate:      now.AddDate(0, 6, 0),   // 6 months from now
			Status:       StatusCurrent,
			Recommended:  "",
			SecurityOnly: false,
		},
		"1.23": {
			Version:      "1.23",
			ReleaseDate:  now.AddDate(-1, -3, 0), // 15 months ago
			EOLDate:      now.AddDate(0, 2, 0),   // 2 months from now (near EOL)
			Status:       StatusNearEOL,
			Recommended:  "1.25",
			SecurityOnly: true,
		},
		"1.22": {
			Version:      "1.22",
			ReleaseDate:  now.AddDate(-2, 0, 0),  // 2 years ago
			EOLDate:      now.AddDate(0, -6, 0),  // 6 months ago (EOL)
			Status:       StatusEOL,
			Recommended:  "1.25",
			SecurityOnly: false,
		},
		"1.21": {
			Version:      "1.21",
			ReleaseDate:  now.AddDate(-2, -6, 0), // 2.5 years ago
			EOLDate:      now.AddDate(-1, 0, 0),  // 1 year ago (EOL)
			Status:       StatusEOL,
			Recommended:  "1.25",
			SecurityOnly: false,
		},
		"1.20": {
			Version:      "1.20",
			ReleaseDate:  now.AddDate(-3, 0, 0),  // 3 years ago
			EOLDate:      now.AddDate(-1, -6, 0), // 1.5 years ago (EOL)
			Status:       StatusEOL,
			Recommended:  "1.25",
			SecurityOnly: false,
		},
		"1.19": {
			Version:      "1.19",
			ReleaseDate:  now.AddDate(-3, -6, 0), // 3.5 years ago
			EOLDate:      now.AddDate(-2, 0, 0),  // 2 years ago (EOL)
			Status:       StatusEOL,
			Recommended:  "1.25",
			SecurityOnly: false,
		},
	}

	lifecycleInitialized = true
}

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
			assert.Equal(t, tt.expected, got, "utils.ExtractMajorMinor() = %v", tt.version)
		})
	}
}

func TestGetVersionInfo(t *testing.T) {
	setupTestLifecycleData()

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
			assert.Equal(t, tt.expectFound, found, "GetVersionInfo() found = %v", tt.version)
			if found && tt.expectEOL {
				assert.Equal(t, StatusEOL, info.Status, "GetVersionInfo() status = %v", tt.version)
			}
		})
	}
}

func TestIsSupported(t *testing.T) {
	setupTestLifecycleData()

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
			assert.Equal(t, tt.supported, got, "IsSupported() = %v", tt.version)
		})
	}
}

func TestIsEOL(t *testing.T) {
	setupTestLifecycleData()

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
			assert.Equal(t, tt.isEOL, got, "IsEOL() = %v", tt.version)
		})
	}
}

func TestIsNearEOL(t *testing.T) {
	setupTestLifecycleData()

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
			assert.Equal(t, tt.isNearEOL, got, "IsNearEOL() = %v", tt.version)
		})
	}
}

func TestGetRecommendedVersion(t *testing.T) {
	setupTestLifecycleData()

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
			assert.Equal(t, tt.want, got, "GetRecommendedVersion() = %v", tt.version)
		})
	}
}

func TestFormatWarning(t *testing.T) {
	setupTestLifecycleData()

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
				assert.Empty(t, got, "FormatWarning() =")
				return
			}

			for _, substr := range tt.expectContains {
				assert.Contains(t, got, substr, "FormatWarning() = %v %v %v", tt.version, got, substr)
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
			assert.Equal(t, tt.expected, got, "calculateStatus() =")
		})
	}
}

func TestVersionLifecycleData(t *testing.T) {
	setupTestLifecycleData()

	// Verify that lifecycle data is well-formed
	for version, info := range versionLifecycle {
		t.Run("version_"+version, func(t *testing.T) {
			assert.Equal(t, version, info.Version, "Version key doesn't match info.Version")

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
			assert.False(t, info.Status == StatusEOL && info.Recommended == "", "EOL version has no recommended upgrade")
		})
	}
}
