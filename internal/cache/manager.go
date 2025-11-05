// Package cache provides utilities and operations for managing Go build and module caches.
package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/go-nv/goenv/internal/cgo"
	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/platform"
	"github.com/go-nv/goenv/internal/utils"
)

// Manager handles cache operations for a goenv installation.
type Manager struct {
	config *config.Config
}

// NewManager creates a new cache manager with the given configuration.
func NewManager(cfg *config.Config) *Manager {
	return &Manager{config: cfg}
}

// CleanOptions specifies criteria for cleaning caches.
type CleanOptions struct {
	Kind      CacheKind     // Specific cache kind to clean ("build", "mod", or "" for all)
	Version   string        // Specific Go version, or "" for all versions
	OldFormat bool          // If true, only clean old-format caches
	MaxBytes  int64         // Keep newest caches under this size (0 = no limit)
	OlderThan time.Duration // Clean caches older than this (0 = no limit)
	DryRun    bool          // If true, don't actually delete anything
	Verbose   bool          // If true, print detailed progress
}

// CleanResult contains the results of a cache cleaning operation.
type CleanResult struct {
	CachesRemoved  int     // Number of caches that were removed
	BytesReclaimed int64   // Total bytes freed
	Errors         []error // Any errors encountered during cleaning
}

// MigrateOptions specifies criteria for migrating caches.
type MigrateOptions struct {
	TargetGOOS   string // Target operating system (e.g., "darwin", "linux")
	TargetGOARCH string // Target architecture (e.g., "arm64", "amd64")
	Force        bool   // Skip confirmation prompts
	DryRun       bool   // If true, don't actually migrate
	Verbose      bool   // If true, print detailed progress
}

// MigrateResult contains the results of a cache migration operation.
type MigrateResult struct {
	CachesMigrated int     // Number of caches that were migrated
	Errors         []error // Any errors encountered during migration
}

// FilterFunc is a predicate function for filtering cache info.
type FilterFunc func(CacheInfo) bool

// WithVersion returns a filter that matches a specific Go version.
func WithVersion(version string) FilterFunc {
	return func(info CacheInfo) bool {
		return info.GoVersion == version
	}
}

// WithKind returns a filter that matches a specific cache kind.
func WithKind(kind CacheKind) FilterFunc {
	return func(info CacheInfo) bool {
		return info.Kind == kind
	}
}

// WithOldFormat returns a filter that matches only old-format caches.
func WithOldFormat() FilterFunc {
	return func(info CacheInfo) bool {
		return info.OldFormat
	}
}

// WithMaxAge returns a filter that matches caches older than the specified duration.
func WithMaxAge(age time.Duration) FilterFunc {
	cutoff := time.Now().Add(-age)
	return func(info CacheInfo) bool {
		return info.ModTime.Before(cutoff)
	}
}

// WithMinSize returns a filter that matches caches larger than the specified size.
func WithMinSize(minBytes int64) FilterFunc {
	return func(info CacheInfo) bool {
		return info.SizeBytes >= minBytes
	}
}

// List returns all caches, optionally filtered by the provided filter functions.
// All filters must match (AND logic).
func (m *Manager) List(filters ...FilterFunc) ([]CacheInfo, error) {
	// Get all caches
	status, err := GetCacheStatus(m.config.Root, false)
	if err != nil {
		return nil, errors.FailedTo("get cache status", err)
	}

	// Combine all caches into a single slice
	allCaches := make([]CacheInfo, 0, len(status.BuildCaches)+len(status.ModCaches))
	allCaches = append(allCaches, status.BuildCaches...)
	allCaches = append(allCaches, status.ModCaches...)

	// Apply filters
	if len(filters) == 0 {
		return allCaches, nil
	}

	filtered := make([]CacheInfo, 0)
	for _, cache := range allCaches {
		matches := true
		for _, filter := range filters {
			if !filter(cache) {
				matches = false
				break
			}
		}
		if matches {
			filtered = append(filtered, cache)
		}
	}

	return filtered, nil
}

// GetStatus returns aggregate cache statistics for all installed Go versions.
//
// Parameters:
//   - fast: If true, skips file counting for better performance
//
// Returns comprehensive cache statistics including per-version breakdown.
func (m *Manager) GetStatus(fast bool) (*CacheStatus, error) {
	return GetCacheStatus(m.config.Root, fast)
}

// Clean removes caches based on the provided options.
//
// The cleaning process:
//  1. Lists all caches matching the criteria (version, kind, old-format)
//  2. Applies filter criteria (older-than, max-bytes)
//  3. Sorts caches by modification time (oldest first for deletion)
//  4. Removes selected caches (or reports what would be removed if dry-run)
//
// Returns a CleanResult with statistics about the operation.
func (m *Manager) Clean(opts CleanOptions) (*CleanResult, error) {
	result := &CleanResult{
		CachesRemoved:  0,
		BytesReclaimed: 0,
		Errors:         make([]error, 0),
	}

	// Build filter list
	filters := make([]FilterFunc, 0)

	// Filter by version if specified
	if opts.Version != "" {
		filters = append(filters, WithVersion(opts.Version))
	}

	// Filter by kind if specified
	if opts.Kind != "" {
		filters = append(filters, WithKind(opts.Kind))
	}

	// Filter by old format if specified
	if opts.OldFormat {
		filters = append(filters, WithOldFormat())
	}

	// Filter by age if specified
	if opts.OlderThan > 0 {
		filters = append(filters, WithMaxAge(opts.OlderThan))
	}

	// Get matching caches
	caches, err := m.List(filters...)
	if err != nil {
		return nil, errors.FailedTo("list caches", err)
	}

	if len(caches) == 0 {
		return result, nil
	}

	// Handle MaxBytes filtering (keep newest, delete oldest)
	if opts.MaxBytes > 0 {
		caches = m.filterByMaxBytes(caches, opts.MaxBytes)
	}

	// Sort by modification time (oldest first for deletion)
	slices.SortFunc(caches, func(a, b CacheInfo) int {
		if a.ModTime.Before(b.ModTime) {
			return -1
		} else if a.ModTime.After(b.ModTime) {
			return 1
		}
		return 0
	})

	// Delete caches
	for _, cache := range caches {
		if opts.DryRun {
			if opts.Verbose {
				fmt.Printf("[DRY RUN] Would remove: %s (%s)\n", cache.Path, FormatBytes(cache.SizeBytes))
			}
			result.CachesRemoved++
			result.BytesReclaimed += cache.SizeBytes
		} else {
			if err := os.RemoveAll(cache.Path); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("failed to remove %s: %w", cache.Path, err))
				continue
			}
			if opts.Verbose {
				fmt.Printf("Removed: %s (%s)\n", cache.Path, FormatBytes(cache.SizeBytes))
			}
			result.CachesRemoved++
			result.BytesReclaimed += cache.SizeBytes
		}
	}

	return result, nil
}

// filterByMaxBytes filters caches to only include those that should be deleted
// when keeping total size under maxBytes. Keeps newest caches, marks oldest for deletion.
func (m *Manager) filterByMaxBytes(caches []CacheInfo, maxBytes int64) []CacheInfo {
	// Sort by modification time (newest first)
	slices.SortFunc(caches, func(a, b CacheInfo) int {
		if a.ModTime.After(b.ModTime) {
			return -1
		} else if a.ModTime.Before(b.ModTime) {
			return 1
		}
		return 0
	})

	// Keep caches until we exceed maxBytes
	var keptSize int64
	toDelete := make([]CacheInfo, 0)

	for _, cache := range caches {
		if keptSize+cache.SizeBytes <= maxBytes {
			// Keep this cache
			keptSize += cache.SizeBytes
		} else {
			// Mark for deletion
			toDelete = append(toDelete, cache)
		}
	}

	return toDelete
}

// Migrate migrates old-format caches to new architecture-aware format.
//
// The migration process:
//  1. Identifies old-format "go-build" caches
//  2. Renames them to "go-build-<GOOS>-<GOARCH>" format
//  3. Preserves all cache contents
//  4. Updates metadata if present
//
// Returns a MigrateResult with statistics about the operation.
func (m *Manager) Migrate(opts MigrateOptions) (*MigrateResult, error) {
	result := &MigrateResult{
		CachesMigrated: 0,
		Errors:         make([]error, 0),
	}

	// Get all old-format build caches
	caches, err := m.List(WithKind(CacheKindBuild), WithOldFormat())
	if err != nil {
		return nil, errors.FailedTo("list old-format caches", err)
	}

	if len(caches) == 0 {
		return result, nil
	}

	// Determine target architecture if not specified
	targetGOOS := opts.TargetGOOS
	targetGOARCH := opts.TargetGOARCH

	if targetGOOS == "" {
		targetGOOS = platform.OS()
	}
	if targetGOARCH == "" {
		targetGOARCH = platform.Arch()
	}

	// Migrate each old-format cache
	for _, cache := range caches {
		oldPath := cache.Path
		dir := filepath.Dir(oldPath)

		// New cache name format
		newName := fmt.Sprintf("go-build-%s-%s", targetGOOS, targetGOARCH)
		newPath := filepath.Join(dir, newName)

		if opts.DryRun {
			if opts.Verbose {
				fmt.Printf("[DRY RUN] Would migrate: %s -> %s\n", oldPath, newPath)
			}
			result.CachesMigrated++
		} else {
			// Check if target already exists
			if utils.PathExists(newPath) {
				result.Errors = append(result.Errors,
					fmt.Errorf("target cache already exists: %s (skipping %s)", newPath, oldPath))
				continue
			}

			// Rename the directory
			if err := os.Rename(oldPath, newPath); err != nil {
				result.Errors = append(result.Errors,
					fmt.Errorf("failed to migrate %s: %w", oldPath, err))
				continue
			}

			if opts.Verbose {
				fmt.Printf("Migrated: %s -> %s\n", oldPath, newPath)
			}
			result.CachesMigrated++
		}
	}

	return result, nil
}

// GetCGOInfo returns CGO toolchain information for a specific version's build cache.
//
// If multiple build caches exist for the version, returns info from the first cache
// that contains CGO metadata. Returns nil if no CGO info is found.
func (m *Manager) GetCGOInfo(version string) (*CGOToolchainInfo, error) {
	// Get all build caches for this version
	caches, err := m.List(WithVersion(version), WithKind(CacheKindBuild))
	if err != nil {
		return nil, errors.FailedTo("list caches", err)
	}

	if len(caches) == 0 {
		return nil, fmt.Errorf("no build caches found for version %s", version)
	}

	// Find first cache with CGO info
	for _, cache := range caches {
		if cache.CGOInfo != nil {
			return cache.CGOInfo, nil
		}
	}

	return nil, fmt.Errorf("no CGO information found for version %s", version)
}

// GetVersionCaches returns all caches (build and module) for a specific Go version.
func (m *Manager) GetVersionCaches(version string) ([]CacheInfo, error) {
	return GetVersionCaches(m.config.Root, version, false)
}

// PruneByAge removes caches older than the specified age across all versions.
// This is a convenience wrapper around Clean with age filtering.
func (m *Manager) PruneByAge(age time.Duration, dryRun bool) (*CleanResult, error) {
	return m.Clean(CleanOptions{
		OlderThan: age,
		DryRun:    dryRun,
		Verbose:   false,
	})
}

// PruneBySize removes oldest caches to keep total size under maxBytes.
// This is a convenience wrapper around Clean with size filtering.
func (m *Manager) PruneBySize(maxBytes int64, dryRun bool) (*CleanResult, error) {
	return m.Clean(CleanOptions{
		MaxBytes: maxBytes,
		DryRun:   dryRun,
		Verbose:  false,
	})
}

// RemoveVersion removes all caches for a specific Go version.
// This is typically called when uninstalling a Go version.
func (m *Manager) RemoveVersion(version string) error {
	result, err := m.Clean(CleanOptions{
		Version: version,
		DryRun:  false,
		Verbose: false,
	})
	if err != nil {
		return err
	}

	if len(result.Errors) > 0 {
		// Return first error (others logged internally)
		return result.Errors[0]
	}

	return nil
}

// SummarizeByVersion returns a map of version -> total cache size.
// Useful for showing cache usage per Go version.
func (m *Manager) SummarizeByVersion() (map[string]int64, error) {
	status, err := m.GetStatus(true) // Use fast mode
	if err != nil {
		return nil, err
	}

	summary := make(map[string]int64)
	for version, versionCaches := range status.ByVersion {
		summary[version] = versionCaches.TotalSize
	}

	return summary, nil
}

// DetectOldFormatCaches returns all old-format caches that could be migrated.
func (m *Manager) DetectOldFormatCaches() ([]CacheInfo, error) {
	return m.List(WithKind(CacheKindBuild), WithOldFormat())
}

// ValidateCaches checks all caches for potential issues.
// Returns a slice of validation errors/warnings.
func (m *Manager) ValidateCaches() ([]string, error) {
	warnings := make([]string, 0)

	status, err := m.GetStatus(true)
	if err != nil {
		return nil, err
	}

	// Check for old-format caches
	oldFormatCount := 0
	for _, cache := range status.BuildCaches {
		if cache.OldFormat {
			oldFormatCount++
		}
	}

	if oldFormatCount > 0 {
		warnings = append(warnings,
			fmt.Sprintf("Found %d old-format build cache(s) - consider running 'goenv cache migrate'", oldFormatCount))
	}

	// Check for very large caches
	const largeThreshold = 5 * 1024 * 1024 * 1024 // 5 GB
	for _, cache := range append(status.BuildCaches, status.ModCaches...) {
		if cache.SizeBytes > largeThreshold {
			warnings = append(warnings,
				fmt.Sprintf("Large cache detected: %s (%s) - consider cleaning",
					cache.Path, FormatBytes(cache.SizeBytes)))
		}
	}

	// Check for very old caches
	oneYearAgo := time.Now().AddDate(-1, 0, 0)
	for _, cache := range append(status.BuildCaches, status.ModCaches...) {
		if !cache.ModTime.IsZero() && cache.ModTime.Before(oneYearAgo) {
			warnings = append(warnings,
				fmt.Sprintf("Old cache detected: %s (last modified %s)",
					cache.Path, cache.ModTime.Format("2006-01-02")))
		}
	}

	return warnings, nil
}

// BuildCacheSuffix constructs a simplified cache directory suffix.
//
// Simplified approach (pre-release simplification):
//   - Per-version isolation (prevents version mismatch errors)
//   - Per-architecture isolation (prevents exec format errors)
//   - Optional CGO marker (not hash - prevents proliferation)
//
// Format: go-build-{GOOS}-{GOARCH}[-cgo]
//
// Examples:
//   - go-build-darwin-arm64
//   - go-build-linux-amd64-cgo
//   - go-build-windows-amd64
//
// Parameters:
//   - goBinaryPath: Path to the Go binary (unused in simplified version)
//   - goos, goarch: Target OS and architecture
//   - env: Environment variables (for CGO_ENABLED check)
//
// Returns: Cache directory name like "go-build-darwin-arm64" or "go-build-linux-amd64-cgo"
func BuildCacheSuffix(goBinaryPath, goos, goarch string, env []string) string {
	// Start with OS-arch
	suffix := fmt.Sprintf("go-build-%s-%s", goos, goarch)

	// Add -cgo marker if CGO is enabled
	// This provides a basic separation between native and CGO builds
	// without the cache proliferation caused by hashing CGO toolchain details
	if cgo.IsCGOEnabled(env) {
		suffix += "-cgo"
	}

	return suffix
}

// CachePathForVersion returns the expected build cache path for a version and architecture.
// This is useful for programmatically constructing cache paths.
// Note: This returns a simplified path without ABI/CGO details. Use BuildCacheSuffix for full details.
func CachePathForVersion(goenvRoot, version, goos, goarch string) string {
	versionPath := filepath.Join(goenvRoot, "versions", version, "pkg")
	cacheName := fmt.Sprintf("go-build-%s-%s", goos, goarch)
	return filepath.Join(versionPath, cacheName)
}

// ModCachePathForVersion returns the module cache path for a version.
func ModCachePathForVersion(goenvRoot, version string) string {
	return filepath.Join(goenvRoot, "versions", version, "pkg", "mod")
}

// ParseCacheType converts a string like "build", "mod", "all" to appropriate filters.
// This is useful for CLI argument parsing.
func ParseCacheType(cacheType string) ([]FilterFunc, error) {
	cacheType = strings.ToLower(strings.TrimSpace(cacheType))

	switch cacheType {
	case "build":
		return []FilterFunc{WithKind(CacheKindBuild)}, nil
	case "mod", "module", "modules":
		return []FilterFunc{WithKind(CacheKindMod)}, nil
	case "all", "":
		return nil, nil // No filter = all
	default:
		return nil, fmt.Errorf("invalid cache type: %s (must be 'build', 'mod', or 'all')", cacheType)
	}
}
