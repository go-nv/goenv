package lifecycle

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestOfflineMode verifies that lifecycle data works correctly in offline mode.
//
// It derives its expectations from the embedded data relative to the current
// date rather than hard-coding specific versions/statuses. An earlier version
// asserted "1.25 is current", which silently rotted into a failure once 1.25
// reached EOL and the embedded data was regenerated.
func TestOfflineMode(t *testing.T) {
	// Set offline mode
	os.Setenv("GOENV_OFFLINE", "1")
	defer os.Unsetenv("GOENV_OFFLINE")

	// Reset lifecycle initialization to force reload
	lifecycleMutex.Lock()
	lifecycleInitialized = false
	versionLifecycle = nil
	lifecycleMutex.Unlock()

	now := time.Now()

	// Pick representative versions from the embedded data by their dates so the
	// assertions remain correct as time passes and the data is regenerated.
	var eolVer, currentVer string
	for ver, info := range EmbeddedLifecycleData {
		if info.EOLDate.IsZero() {
			continue
		}
		switch {
		case now.After(info.EOLDate):
			// Prefer the most recently EOL'd version (unambiguously in the past).
			if eolVer == "" || info.EOLDate.After(EmbeddedLifecycleData[eolVer].EOLDate) {
				eolVer = ver
			}
		case info.EOLDate.After(now.AddDate(0, 6, 0)):
			// EOL comfortably in the future -> unambiguously current.
			if currentVer == "" || info.ReleaseDate.After(EmbeddedLifecycleData[currentVer].ReleaseDate) {
				currentVer = ver
			}
		}
	}

	if eolVer == "" {
		t.Fatal("embedded data should contain at least one EOL version")
	}

	// EOL versions must be reported as EOL from embedded data alone (offline),
	// and must recommend an upgrade target.
	info, found := GetVersionInfo(eolVer + ".0")
	assert.True(t, found, "should find EOL version %s in embedded data", eolVer)
	assert.Equal(t, eolVer, info.Version)
	assert.Equal(t, StatusEOL, info.Status, "%s (EOL %s) should be EOL offline",
		eolVer, EmbeddedLifecycleData[eolVer].EOLDate.Format("2006-01-02"))
	assert.NotEmpty(t, info.Recommended, "EOL version should recommend an upgrade")

	// A version whose EOL is comfortably in the future must be current.
	if currentVer != "" {
		info, found = GetVersionInfo(currentVer + ".0")
		assert.True(t, found, "should find current version %s in embedded data", currentVer)
		assert.Equal(t, StatusCurrent, info.Status, "%s (EOL %s) should be current offline",
			currentVer, EmbeddedLifecycleData[currentVer].EOLDate.Format("2006-01-02"))
	}
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
