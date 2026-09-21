//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// writeBuildCache creates a build-cache directory inside an installed version,
// laid out exactly as `goenv exec` creates it: DIRECTLY in the version
// directory, named go-build-<GOOS>-<GOARCH>[-cgo]. It drops a sized payload
// file so the scanner reports a non-zero size.
//
// versionDir is the path returned by env.fakeVersion (…/versions/<version>).
func writeBuildCache(t *testing.T, versionDir, cacheName string, size int) string {
	t.Helper()

	dir := filepath.Join(versionDir, cacheName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create build cache %s: %v", dir, err)
	}

	payload := make([]byte, size)
	for i := range payload {
		payload[i] = 'x'
	}
	if err := os.WriteFile(filepath.Join(dir, "cache.a"), payload, 0o644); err != nil {
		t.Fatalf("failed to write build cache payload in %s: %v", dir, err)
	}
	return dir
}

// TestCacheClean_FindsAndCleansArchitectureSuffixedBuildCaches is the
// end-to-end regression for the "No caches found" bug.
//
// `goenv exec` points GOCACHE at versions/<version>/go-build-<GOOS>-<GOARCH>
// [-cgo] — directly in the version directory. When GOOS/GOARCH are unset (a
// native build) the markers are literally "host", producing names like
// go-build-host-host-cgo. The status/clean/info scanner used to look under
// versions/<version>/pkg/ and only recognise a bare "go-build" in the version
// directory, so every real (architecture-suffixed) cache was invisible: `goenv
// cache status` said "No caches found" and `goenv cache clean` refused to
// reclaim anything while gigabytes sat on disk.
//
// The unit tests at the time seeded the pkg/ layout, so they passed against the
// broken scanner. Only the real binary run against a real filesystem — this
// test — exercises the layout `goenv exec` actually writes.
func TestCacheClean_FindsAndCleansArchitectureSuffixedBuildCaches(t *testing.T) {
	e := newEnv(t)

	versionDir := e.fakeVersion("1.27.0")

	nativeCache := writeBuildCache(t, versionDir, "go-build-host-host-cgo", 16*1024)
	crossCache := writeBuildCache(t, versionDir, "go-build-linux-arm64", 8*1024)

	// status must SEE the caches that exist on disk.
	status := e.run("cache", "status")
	if !status.Succeeded() {
		e.Diagnose()
		t.Fatalf("cache status failed: exit=%d\n%s", status.ExitCode, status.Output())
	}
	requireNotContains(t, status.Output(), "No caches found",
		"cache status must discover architecture-suffixed caches in the version dir")
	requireContains(t, status.Output(), "1.27.0",
		"cache status must attribute the cache to its Go version")
	requireContains(t, status.Output(), "host-host",
		"cache status must show the architecture-suffixed cache name")

	// clean must REMOVE those build caches.
	clean := e.run("cache", "clean", "build", "--force")
	if !clean.Succeeded() {
		e.Diagnose()
		t.Fatalf("cache clean failed: exit=%d\n%s", clean.ExitCode, clean.Output())
	}
	requireContains(t, clean.Output(), "Removed",
		"cache clean must report the caches it removed")

	if fileExists(nativeCache) {
		e.Diagnose()
		t.Errorf("native build cache was not removed: %s", nativeCache)
	}
	if fileExists(crossCache) {
		e.Diagnose()
		t.Errorf("cross-compile build cache was not removed: %s", crossCache)
	}

	// A cache clean must never touch the Go installation itself.
	goBin := filepath.Join(versionDir, "bin", "go")
	if runtime.GOOS == "windows" {
		goBin += ".exe"
	}
	if !fileExists(goBin) {
		t.Errorf("cache clean must not delete the Go install: missing %s", goBin)
	}
}

// TestCacheClean_IsGlobalNotScopedToProjectVersion pins down the behaviour the
// command actually has: `goenv cache clean` operates on $GOENV_ROOT globally,
// across every installed version, and is completely independent of the current
// working directory and any project's pinned Go version.
//
// The working directory is turned into a project pinned to 1.27.0 (via
// .go-version and go.mod). Cleaning from there must still remove the cache of a
// DIFFERENT, non-pinned version (1.25.4) — proving the operation is global and
// not narrowed by the project you happen to run it from.
func TestCacheClean_IsGlobalNotScopedToProjectVersion(t *testing.T) {
	e := newEnv(t)

	pinnedDir := e.fakeVersion("1.27.0")
	otherDir := e.fakeVersion("1.25.4")

	pinnedCache := writeBuildCache(t, pinnedDir, "go-build-host-host-cgo", 8*1024)
	otherCache := writeBuildCache(t, otherDir, "go-build-host-host-cgo", 8*1024)

	// Make the working directory a project pinned to 1.27.0.
	e.writeFile(".go-version", "1.27.0\n")
	e.writeFile("go.mod", "module example.com/proj\n\ngo 1.27.0\n")

	clean := e.run("cache", "clean", "build", "--force")
	if !clean.Succeeded() {
		e.Diagnose()
		t.Fatalf("cache clean failed: exit=%d\n%s", clean.ExitCode, clean.Output())
	}

	if fileExists(pinnedCache) {
		e.Diagnose()
		t.Errorf("pinned version's cache should be cleaned: %s", pinnedCache)
	}
	if fileExists(otherCache) {
		e.Diagnose()
		t.Errorf("a non-pinned version's cache must also be cleaned "+
			"(clean is global, not scoped to the project's Go version): %s", otherCache)
	}
}

// TestCacheClean_CurrentScopesToProjectVersion is the inverse of the global
// test: with --current, the clean is scoped to the version the working
// directory resolves to (here via .go-version), and a different version's cache
// must be left untouched. This pins the behaviour of the project-scoped
// convenience flag, kept consistent with `goenv current`.
func TestCacheClean_CurrentScopesToProjectVersion(t *testing.T) {
	e := newEnv(t)

	currentDir := e.fakeVersion("1.27.0")
	otherDir := e.fakeVersion("1.25.4")

	currentCache := writeBuildCache(t, currentDir, "go-build-host-host-cgo", 8*1024)
	otherCache := writeBuildCache(t, otherDir, "go-build-host-host-cgo", 8*1024)

	// Pin the working directory to an exact, installed version.
	e.writeFile(".go-version", "1.27.0\n")

	clean := e.run("cache", "clean", "build", "--current", "--force")
	if !clean.Succeeded() {
		e.Diagnose()
		t.Fatalf("cache clean --current failed: exit=%d\n%s", clean.ExitCode, clean.Output())
	}
	requireContains(t, clean.Output(), "1.27.0",
		"clean --current must announce the version it scoped to")

	if fileExists(currentCache) {
		e.Diagnose()
		t.Errorf("the resolved version's cache should be cleaned: %s", currentCache)
	}
	if !fileExists(otherCache) {
		e.Diagnose()
		t.Errorf("--current must NOT clean a different version's cache: %s", otherCache)
	}
}

// TestCacheClean_FindsCachesUnderGoenvGocacheDir covers the GOENV_GOCACHE_DIR
// override end-to-end: `goenv exec` points GOCACHE at
// <custom>/<version>/go-build-*, off the GOENV_ROOT tree. status and clean must
// read the same override, or those caches are invisible and unreclaimable.
func TestCacheClean_FindsCachesUnderGoenvGocacheDir(t *testing.T) {
	e := newEnv(t)
	e.fakeVersion("1.27.0")

	// Lay a build cache exactly where `goenv exec` would with GOENV_GOCACHE_DIR set.
	customDir := filepath.Join(e.Home, "custom-gocache")
	cacheDir := filepath.Join(customDir, "1.27.0", "go-build-host-host-cgo")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("failed to create custom cache: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "cache.a"), make([]byte, 16*1024), 0o644); err != nil {
		t.Fatalf("failed to write custom cache payload: %v", err)
	}
	e.Set("GOENV_GOCACHE_DIR", customDir)

	status := e.run("cache", "status")
	if !status.Succeeded() {
		e.Diagnose()
		t.Fatalf("cache status failed: exit=%d\n%s", status.ExitCode, status.Output())
	}
	requireNotContains(t, status.Output(), "No caches found",
		"cache status must discover caches under GOENV_GOCACHE_DIR")
	requireContains(t, status.Output(), "1.27.0",
		"cache status must attribute the off-root cache to its version")

	clean := e.run("cache", "clean", "build", "--force")
	if !clean.Succeeded() {
		e.Diagnose()
		t.Fatalf("cache clean failed: exit=%d\n%s", clean.ExitCode, clean.Output())
	}
	requireContains(t, clean.Output(), "Removed", "clean must remove off-root caches")

	if fileExists(cacheDir) {
		e.Diagnose()
		t.Errorf("cache under GOENV_GOCACHE_DIR was not removed: %s", cacheDir)
	}
}
