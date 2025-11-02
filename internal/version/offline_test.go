package version

import (
	"testing"

	"github.com/go-nv/goenv/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOfflineMode(t *testing.T) {
	// Set offline mode
	t.Setenv(utils.GoenvEnvVarOffline.String(), "1")

	// Create a fetcher
	fetcher := NewFetcher()

	// Fetch versions - should use embedded data
	versions, err := fetcher.FetchAllVersions()
	require.NoError(t, err, "FetchAllVersions failed in offline mode")

	// Verify we got versions
	require.NotEmpty(t, versions, "Expected versions in offline mode, got none")

	// Verify we're getting embedded versions (should match EmbeddedVersions length)
	assert.Len(t, versions, len(EmbeddedVersions), "Expected embedded versions")

	// Verify first version matches
	if len(versions) > 0 && len(EmbeddedVersions) > 0 {
		assert.Equal(t, EmbeddedVersions[0].Version, versions[0], "First version mismatch")
	}
}

func TestOfflineModeFetchWithFallback(t *testing.T) {
	// Set offline mode
	t.Setenv(utils.GoenvEnvVarOffline.String(), "1")

	// Create a fetcher
	fetcher := NewFetcher()

	// Use FetchWithFallback - should skip network and use embedded
	releases, err := fetcher.FetchWithFallback("/tmp/test-goenv-offline")
	require.NoError(t, err, "FetchWithFallback failed in offline mode")

	// Verify we got releases
	require.NotEmpty(t, releases, "Expected releases in offline mode, got none")

	// Verify we're getting embedded versions
	assert.Len(t, releases, len(EmbeddedVersions), "Expected embedded releases")
}

func TestNormalModeStillWorks(t *testing.T) {
	// Ensure offline mode is NOT set
	t.Setenv(utils.GoenvEnvVarOffline.String(), "")

	// Create a fetcher without cache (will try network, may fail, that's OK)
	fetcher := NewFetcher()

	// This test just ensures the code doesn't panic when offline mode is not set
	// We don't test network behavior here as it would make tests flaky
	_, err := fetcher.FetchAllVersions()

	// We expect either success (if network available) or a network error
	// We just want to ensure no panic occurs
	if err != nil {
		t.Logf("Network fetch failed (expected in isolated test environment): %v", err)
	}
}
