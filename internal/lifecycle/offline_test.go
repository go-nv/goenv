package lifecycle

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestOfflineMode verifies that lifecycle data works correctly in offline mode
func TestOfflineMode(t *testing.T) {
	// Set offline mode
	os.Setenv("GOENV_OFFLINE", "1")
	defer os.Unsetenv("GOENV_OFFLINE")

	// Reset lifecycle initialization to force reload
	lifecycleMutex.Lock()
	lifecycleInitialized = false
	versionLifecycle = nil
	lifecycleMutex.Unlock()

	// Get version info for an EOL version (should work with embedded data)
	info, found := GetVersionInfo("1.22.5")
	assert.True(t, found, "Should find version 1.22 in embedded data")
	assert.Equal(t, "1.22", info.Version)

	// Status should be calculated correctly from embedded dates
	// 1.22 EOL date is 2025-02-11, which is in the past (today is 2025-12-01)
	assert.Equal(t, StatusEOL, info.Status, "1.22 should be EOL based on embedded dates")

	// Should have a recommended version
	assert.NotEmpty(t, info.Recommended, "EOL version should have recommended upgrade")

	// Test a current version
	info, found = GetVersionInfo("1.25.0")
	assert.True(t, found, "Should find version 1.25 in embedded data")
	assert.Equal(t, StatusCurrent, info.Status, "1.25 should be current based on embedded dates")

	// Test near-EOL version
	// 1.23 has EOL date of 2025-08-12, which is in the past, so it's EOL
	info, found = GetVersionInfo("1.23.0")
	assert.True(t, found, "Should find version 1.23 in embedded data")
	assert.Equal(t, StatusEOL, info.Status, "1.23 should be EOL (past 2025-08-12)")
}

// TestEmbeddedDataHasDates verifies embedded data has necessary date information
func TestEmbeddedDataHasDates(t *testing.T) {
	// Check that embedded data has dates
	for version, info := range EmbeddedLifecycleData {
		t.Run("version_"+version, func(t *testing.T) {
			assert.False(t, info.ReleaseDate.IsZero(), "Version %s should have release date", version)
			assert.False(t, info.EOLDate.IsZero(), "Version %s should have EOL date", version)
		})
	}
}
