package version

import (
	"testing"

	"github.com/go-nv/goenv/internal/utils"
)

func TestOfflineMode(t *testing.T) {
	// Set offline mode
	t.Setenv(utils.GoenvEnvVarOffline.String(), "1")

	// Create a fetcher
	fetcher := NewFetcher()

	// Fetch versions - should use embedded data
	versions, err := fetcher.FetchAllVersions()
	if err != nil {
		t.Fatalf("FetchAllVersions failed in offline mode: %v", err)
	}

	// Verify we got versions
	if len(versions) == 0 {
		t.Fatal("Expected versions in offline mode, got none")
	}

	// Verify we're getting embedded versions (should match EmbeddedVersions length)
	if len(versions) != len(EmbeddedVersions) {
		t.Errorf("Expected %d embedded versions, got %d", len(EmbeddedVersions), len(versions))
	}

	// Verify first version matches
	if len(versions) > 0 && len(EmbeddedVersions) > 0 {
		if versions[0] != EmbeddedVersions[0].Version {
			t.Errorf("First version mismatch: got %s, want %s", versions[0], EmbeddedVersions[0].Version)
		}
	}
}

func TestOfflineModeFetchWithFallback(t *testing.T) {
	// Set offline mode
	t.Setenv(utils.GoenvEnvVarOffline.String(), "1")

	// Create a fetcher
	fetcher := NewFetcher()

	// Use FetchWithFallback - should skip network and use embedded
	releases, err := fetcher.FetchWithFallback("/tmp/test-goenv-offline")
	if err != nil {
		t.Fatalf("FetchWithFallback failed in offline mode: %v", err)
	}

	// Verify we got releases
	if len(releases) == 0 {
		t.Fatal("Expected releases in offline mode, got none")
	}

	// Verify we're getting embedded versions
	if len(releases) != len(EmbeddedVersions) {
		t.Errorf("Expected %d embedded releases, got %d", len(EmbeddedVersions), len(releases))
	}
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
