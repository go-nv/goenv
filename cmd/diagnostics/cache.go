package diagnostics

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/cgo"
	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/envdetect"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

// completeCacheCleanTypes provides shell completion for cache clean command
func completeCacheCleanTypes(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return []string{"build", "mod", "all"}, cobra.ShellCompDirectiveNoFileComp
}

var cacheCmd = &cobra.Command{
	Use:     "cache",
	Short:   "Manage goenv caches",
	GroupID: string(cmdpkg.GroupDiagnostics),
	Long: `Manage build and module caches for installed Go versions.

Subcommands:
  status    Show cache sizes and locations
  clean     Clean build or module caches to reclaim disk space
  migrate   Migrate old format caches to architecture-aware format

Use "goenv cache <command> --help" for more information about a command.`,
}

var cacheStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show cache sizes and locations",
	Long: `Display detailed information about build and module caches.

Shows:
  - Build cache sizes per version and architecture
  - Module cache sizes per version
  - Total cache usage
  - Cache locations

This helps understand disk usage and verify cache isolation is working.

Options:
  --json    Output machine-readable JSON for CI/automation
  --fast    Fast mode: skip file counting for better performance
            (useful for very large caches, displays ~ for file counts)`,
	RunE: runCacheStatus,
}

var cacheCleanCmd = &cobra.Command{
	Use:   "clean [type]",
	Short: "Clean build or module caches",
	Long: `Clean build or module caches to reclaim disk space.

When switching between Go versions, cached build artifacts can cause
"version mismatch" errors. This command helps fix those issues by
clearing the problematic caches.

Types:
  build    Clean build caches only (default)
  mod      Clean module caches only
  all      Clean both build and module caches

If no type is specified, defaults to 'build'.

Examples:
  goenv cache clean                                # Clean build caches (default)
  goenv cache clean build                          # Clean all build caches
  goenv cache clean mod                            # Clean all module caches
  goenv cache clean all                            # Clean everything

  # Clean specific version:
  goenv cache clean build --version 1.23.2

  # Clean old format caches only:
  goenv cache clean build --old-format

  # Prune caches by size (keep newest, delete oldest until under limit):
  goenv cache clean build --max-bytes 1GB          # Keep only 1GB of build caches
  goenv cache clean all --max-bytes 500MB          # Keep only 500MB total

  # Prune caches by age:
  goenv cache clean build --older-than 30d         # Delete caches older than 30 days
  goenv cache clean build --older-than 1w          # Delete caches older than 1 week
  goenv cache clean all --older-than 24h           # Delete caches older than 24 hours

  # Preview what would be deleted (dry-run):
  goenv cache clean build --dry-run                # Show what would be cleaned
  goenv cache clean all --older-than 30d --dry-run # Preview age-based cleanup

For diagnostic information about caches, use:
  goenv cache status                               # Show cache sizes and locations
  goenv doctor                                     # Check cache isolation settings`,
	Args:              cobra.MaximumNArgs(1),
	RunE:              runCacheClean,
	ValidArgsFunction: completeCacheCleanTypes,
}

var cacheMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate old format caches to architecture-aware format",
	Long: `Migrate old format build caches to new architecture-aware format.

This command helps users upgrading from older goenv versions by:
  - Detecting old format caches (go-build directories)
  - Moving them to architecture-specific directories (go-build-{GOOS}-{GOARCH})
  - Preventing cache conflicts between architectures
  - Cleaning up after migration

The migration is safe and can be run multiple times. Old format caches
will be moved to match the current system architecture.

Examples:
  goenv cache migrate          # Migrate all old format caches
  goenv cache migrate --force  # Skip confirmation prompt`,
	RunE: runCacheMigrate,
}

var cacheInfoCmd = &cobra.Command{
	Use:   "info [version]",
	Short: "Show CGO toolchain information for caches",
	Long: `Display CGO toolchain configuration for build caches.

This command shows which C compiler, flags, and other CGO-related
settings were used when creating each build cache. This helps diagnose
cache-related issues and understand why different caches exist.

Examples:
  goenv cache info           # Show info for all versions
  goenv cache info 1.23.2    # Show info for specific version
  goenv cache info --json    # Machine-readable output`,
	RunE: runCacheInfo,
}

var (
	cleanVersion   string
	cleanOldFormat bool
	cleanForce     bool
	cleanMaxBytes  string
	cleanOlderThan string
	cleanDryRun    bool
	cleanVerbose   bool
	migrateForce   bool
	statusJSON     bool
	statusFast     bool
	infoJSON       bool
)

func init() {
	cmdpkg.RootCmd.AddCommand(cacheCmd)
	cacheCmd.AddCommand(cacheStatusCmd)
	cacheCmd.AddCommand(cacheCleanCmd)
	cacheCmd.AddCommand(cacheMigrateCmd)
	cacheCmd.AddCommand(cacheInfoCmd)

	cacheStatusCmd.Flags().BoolVar(&statusJSON, "json", false, "Output machine-readable JSON")
	cacheStatusCmd.Flags().BoolVar(&statusFast, "fast", false, "Fast mode: skip file counting for better performance")

	cacheCleanCmd.Flags().StringVar(&cleanVersion, "version", "", "Clean caches for specific version only")
	cacheCleanCmd.Flags().BoolVar(&cleanOldFormat, "old-format", false, "Clean old format caches only")
	cacheCleanCmd.Flags().BoolVarP(&cleanForce, "force", "f", false, "Skip confirmation prompt")
	cacheCleanCmd.Flags().StringVar(&cleanMaxBytes, "max-bytes", "", "Keep only this much cache (e.g., 1GB, 500MB) - deletes oldest first")
	cacheCleanCmd.Flags().StringVar(&cleanOlderThan, "older-than", "", "Delete caches older than this duration (e.g., 30d, 1w, 24h)")
	cacheCleanCmd.Flags().BoolVarP(&cleanDryRun, "dry-run", "n", false, "Show what would be cleaned without actually cleaning")
	cacheCleanCmd.Flags().BoolVarP(&cleanVerbose, "verbose", "v", false, "Show detailed output")

	cacheMigrateCmd.Flags().BoolVarP(&migrateForce, "force", "f", false, "Skip confirmation prompt")

	cacheInfoCmd.Flags().BoolVar(&infoJSON, "json", false, "Output machine-readable JSON")

	helptext.SetCommandHelp(cacheCmd)
	helptext.SetCommandHelp(cacheStatusCmd)
	helptext.SetCommandHelp(cacheCleanCmd)
	helptext.SetCommandHelp(cacheInfoCmd)
	helptext.SetCommandHelp(cacheMigrateCmd)
}

// JSON schema for cache status output - stable API for CI/automation
type cacheStatusJSON struct {
	SchemaVersion string       `json:"schema_version"`
	Tool          toolInfo     `json:"tool"`
	Host          hostInfo     `json:"host"`
	Timestamp     string       `json:"timestamp"`
	Caches        []cacheEntry `json:"caches"`
	Totals        cacheTotals  `json:"totals"`
}

type toolInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Commit  string `json:"commit,omitempty"`
}

type hostInfo struct {
	GOOS      string `json:"goos"`
	GOARCH    string `json:"goarch"`
	Rosetta   bool   `json:"rosetta"`
	WSL       bool   `json:"wsl"`
	Container bool   `json:"container"`
}

type cacheEntry struct {
	Kind      string            `json:"kind"` // "build" or "mod"
	Path      string            `json:"path"`
	GoVersion string            `json:"go_version"`
	Target    *targetInfo       `json:"target,omitempty"` // nil for mod caches
	ABI       map[string]string `json:"abi,omitempty"`    // GOAMD64, GOARM, etc.
	SizeBytes int64             `json:"size_bytes"`
	Entries   int               `json:"entries"`
	Exists    bool              `json:"exists"`
	OldFormat bool              `json:"old_format"`
	Notes     []string          `json:"notes,omitempty"`
}

type targetInfo struct {
	GOOS   string `json:"goos"`
	GOARCH string `json:"goarch"`
}

type cacheTotals struct {
	SizeBytes int64 `json:"size_bytes"`
	Entries   int   `json:"entries"`
}

// Internal struct for rendering human-readable output
type cacheInfo struct {
	path      string
	sizeGB    float64
	sizeBytes int64
	files     int
	version   string
	arch      string // e.g., "darwin-arm64" or "host-host"
	modTime   time.Time
}

func runCacheStatus(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	// Get installed versions
	versions, err := mgr.ListInstalledVersions()
	if err != nil {
		return fmt.Errorf("cannot list installed versions: %w", err)
	}

	if len(versions) == 0 {
		if statusJSON {
			// Output minimal JSON for no versions
			result := cacheStatusJSON{
				SchemaVersion: "1",
				Tool: toolInfo{
					Name:    "goenv",
					Version: cmdpkg.AppVersion,
				},
				Host: hostInfo{
					GOOS:      runtime.GOOS,
					GOARCH:    runtime.GOARCH,
					Rosetta:   envdetect.IsRosetta(),
					WSL:       envdetect.IsWSL(),
					Container: envdetect.IsInContainer(),
				},
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Caches:    []cacheEntry{},
				Totals: cacheTotals{
					SizeBytes: 0,
					Entries:   0,
				},
			}
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetIndent("", "  ")
			return encoder.Encode(result)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "No Go versions installed.")
		return nil
	}

	// Auto-detect if user should use --fast mode for better performance
	if !statusFast && !statusJSON {
		shouldSuggestFast := false
		estimatedTotalFiles := 0

		// Quick sample: check first few caches to estimate total file count
		sampleCount := 0
		for _, version := range versions {
			if sampleCount >= 3 { // Sample first 3 versions
				break
			}

			versionPath := filepath.Join(cfg.VersionsDir(), version)
			entries, err := os.ReadDir(versionPath)
			if err != nil {
				continue
			}

			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				name := entry.Name()
				if strings.HasPrefix(name, "go-build-") || name == "go-build" {
					cachePath := filepath.Join(versionPath, name)
					// Quick sample with very short timeout
					_, sampleFiles := getDirSizeWithOptions(cachePath, false, 50*time.Millisecond)
					if sampleFiles > 0 {
						estimatedTotalFiles += sampleFiles
						sampleCount++
					}
					break // One cache per version for sampling
				}
			}
		}

		// If sample suggests >5000 files total, recommend --fast
		// Extrapolate: if 3 versions have N files, estimate total
		if sampleCount > 0 {
			estimatedTotal := (estimatedTotalFiles * len(versions)) / sampleCount
			if estimatedTotal > 5000 {
				shouldSuggestFast = true
			}
		}

		if shouldSuggestFast {
			fmt.Fprintf(cmd.OutOrStdout(),
				"%s Large cache detected (estimated %s+ files). "+
					"Use --fast for 5-10x faster scanning.\n\n",
				utils.Emoji("💡"), formatNumber(estimatedTotalFiles))
		}
	}

	// Collect all cache entries for JSON output
	var cacheEntries []cacheEntry
	var totalSize int64
	var totalEntries int

	// Collect build caches
	for _, version := range versions {
		versionPath := filepath.Join(cfg.VersionsDir(), version)

		// Check for architecture-specific caches
		entries, err := os.ReadDir(versionPath)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			name := entry.Name()
			// Look for go-build-* directories
			if strings.HasPrefix(name, "go-build-") || name == "go-build" {
				cachePath := filepath.Join(versionPath, name)
				size, files := getDirSizeWithOptions(cachePath, statusFast, 10*time.Second)

				// Parse ABI information from cache name
				goos, goarch, abi := parseABIFromCacheName(name)
				isOldFormat := name == "go-build"

				var target *targetInfo
				if !isOldFormat && goos != "" && goarch != "" {
					target = &targetInfo{
						GOOS:   goos,
						GOARCH: goarch,
					}
				}

				cacheEntries = append(cacheEntries, cacheEntry{
					Kind:      "build",
					Path:      cachePath,
					GoVersion: version,
					Target:    target,
					ABI:       abi,
					SizeBytes: size,
					Entries:   files,
					Exists:    true,
					OldFormat: isOldFormat,
				})

				totalSize += size
				totalEntries += files
			}
		}
	}

	// Collect module caches
	for _, version := range versions {
		versionPath := filepath.Join(cfg.VersionsDir(), version)
		modCachePath := filepath.Join(versionPath, "go-mod")

		if stat, err := os.Stat(modCachePath); err == nil && stat.IsDir() {
			size, files := getDirSizeWithOptions(modCachePath, statusFast, 10*time.Second)

			cacheEntries = append(cacheEntries, cacheEntry{
				Kind:      "mod",
				Path:      modCachePath,
				GoVersion: version,
				Target:    nil, // Module caches don't have target info
				SizeBytes: size,
				Entries:   files,
				Exists:    true,
				OldFormat: false,
			})

			totalSize += size
			totalEntries += files
		}
	}

	// If JSON output requested, marshal and output
	if statusJSON {
		result := cacheStatusJSON{
			SchemaVersion: "1",
			Tool: toolInfo{
				Name:    "goenv",
				Version: cmdpkg.AppVersion,
			},
			Host: hostInfo{
				GOOS:      runtime.GOOS,
				GOARCH:    runtime.GOARCH,
				Rosetta:   envdetect.IsRosetta(),
				WSL:       envdetect.IsWSL(),
				Container: envdetect.IsInContainer(),
			},
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Caches:    cacheEntries,
			Totals: cacheTotals{
				SizeBytes: totalSize,
				Entries:   totalEntries,
			},
		}

		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	}

	// Human-readable output
	fmt.Fprintf(cmd.OutOrStdout(), "%sCache Status\n", utils.Emoji("📊 "))
	fmt.Fprintln(cmd.OutOrStdout())

	// Display build caches
	var buildCaches []cacheInfo
	var totalBuildSize float64
	var totalBuildFiles int
	hasApproximateBuildFiles := false

	for _, entry := range cacheEntries {
		if entry.Kind == "build" {
			sizeGB := float64(entry.SizeBytes) / (1024 * 1024 * 1024)
			arch := "(old format)"
			if entry.Target != nil {
				arch = fmt.Sprintf("%s-%s", entry.Target.GOOS, entry.Target.GOARCH)
				if len(entry.ABI) > 0 {
					// Add ABI info to arch string
					for key, val := range entry.ABI {
						arch += fmt.Sprintf("-%s:%s", key, val)
					}
				}
			}

			buildCaches = append(buildCaches, cacheInfo{
				path:    entry.Path,
				sizeGB:  sizeGB,
				files:   entry.Entries,
				version: entry.GoVersion,
				arch:    arch,
			})

			totalBuildSize += sizeGB
			// Track if any entry has approximate count
			if entry.Entries < 0 {
				hasApproximateBuildFiles = true
			} else if !hasApproximateBuildFiles {
				totalBuildFiles += entry.Entries
			}
		}
	}

	// If any cache has approximate count, mark total as approximate
	if hasApproximateBuildFiles {
		totalBuildFiles = -1
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%sBuild Caches:\n", utils.Emoji("🔨 "))
	if len(buildCaches) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "  No build caches found")
	} else {
		// Sort by version, then arch
		sort.Slice(buildCaches, func(i, j int) bool {
			if buildCaches[i].version != buildCaches[j].version {
				return buildCaches[i].version < buildCaches[j].version
			}
			return buildCaches[i].arch < buildCaches[j].arch
		})

		for _, cache := range buildCaches {
			fmt.Fprintf(cmd.OutOrStdout(), "  Go %-8s (%s): %.2f GB, %s files\n",
				cache.version, cache.arch, cache.sizeGB, formatFileCount(cache.files))
		}
	}
	fmt.Fprintln(cmd.OutOrStdout())

	// Display module caches
	var modCaches []cacheInfo
	var totalModSize float64
	var totalModFiles int
	hasApproximateModFiles := false

	for _, entry := range cacheEntries {
		if entry.Kind == "mod" {
			sizeGB := float64(entry.SizeBytes) / (1024 * 1024 * 1024)

			modCaches = append(modCaches, cacheInfo{
				path:    entry.Path,
				sizeGB:  sizeGB,
				files:   entry.Entries,
				version: entry.GoVersion,
			})

			totalModSize += sizeGB
			// Track if any entry has approximate count
			if entry.Entries < 0 {
				hasApproximateModFiles = true
			} else if !hasApproximateModFiles {
				totalModFiles += entry.Entries
			}
		}
	}

	// If any cache has approximate count, mark total as approximate
	if hasApproximateModFiles {
		totalModFiles = -1
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%sModule Caches:\n", utils.Emoji("📦 "))
	if len(modCaches) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "  No module caches found")
	} else {
		sort.Slice(modCaches, func(i, j int) bool {
			return modCaches[i].version < modCaches[j].version
		})

		for _, cache := range modCaches {
			fmt.Fprintf(cmd.OutOrStdout(), "  Go %-8s: %.2f GB, %s modules/files\n",
				cache.version, cache.sizeGB, formatFileCount(cache.files))
		}
	}
	fmt.Fprintln(cmd.OutOrStdout())

	// Summary
	fmt.Fprintln(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintf(cmd.OutOrStdout(), "Total Build Cache:  %.2f GB (%s files)\n", totalBuildSize, formatFileCount(totalBuildFiles))
	fmt.Fprintf(cmd.OutOrStdout(), "Total Module Cache: %.2f GB (%s items)\n", totalModSize, formatFileCount(totalModFiles))
	fmt.Fprintf(cmd.OutOrStdout(), "Total:              %.2f GB\n", totalBuildSize+totalModSize)
	if totalBuildFiles < 0 || totalModFiles < 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Note: ~ indicates approximate file count (fast mode or large cache)\n")
	}
	fmt.Fprintln(cmd.OutOrStdout())

	// Show cache locations
	fmt.Fprintln(cmd.OutOrStdout(), "📍 Cache Locations:")
	fmt.Fprintf(cmd.OutOrStdout(), "  GOENV_ROOT: %s\n", cfg.Root)
	fmt.Fprintf(cmd.OutOrStdout(), "  Versions:   %s\n", cfg.VersionsDir())
	fmt.Fprintln(cmd.OutOrStdout())

	// Helpful tips
	fmt.Fprintf(cmd.OutOrStdout(), "%sTips:\n", utils.Emoji("💡 "))
	fmt.Fprintln(cmd.OutOrStdout(), "  • Clean build caches:  goenv cache clean build")
	fmt.Fprintln(cmd.OutOrStdout(), "  • Clean module caches: goenv cache clean mod")
	fmt.Fprintln(cmd.OutOrStdout(), "  • Clean all caches:    goenv cache clean all")
	for _, cache := range buildCaches {
		if cache.arch == "(old format)" {
			fmt.Fprintln(cmd.OutOrStdout(), "  • Old format caches detected - consider migrating them")
			break
		}
	}

	return nil
}

func runCacheClean(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	// Default to 'build' if no argument provided (matching old 'goenv clean' behavior)
	cleanType := "build"
	if len(args) > 0 {
		cleanType = args[0]
	}

	if cleanType != "build" && cleanType != "mod" && cleanType != "all" {
		return fmt.Errorf("invalid type: %s (must be 'build', 'mod', or 'all')", cleanType)
	}

	// Get installed versions
	versions, err := mgr.ListInstalledVersions()
	if err != nil {
		return fmt.Errorf("cannot list installed versions: %w", err)
	}

	if len(versions) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No Go versions installed.")
		return nil
	}

	// Filter versions if --version specified
	if cleanVersion != "" {
		found := false
		for _, v := range versions {
			if v == cleanVersion {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("version %s is not installed", cleanVersion)
		}
		versions = []string{cleanVersion}
	}

	// Collect caches to clean
	var cachesToClean []cacheInfo
	var totalSize float64
	var totalFiles int

	for _, version := range versions {
		versionPath := filepath.Join(cfg.VersionsDir(), version)

		// Collect build caches
		if cleanType == "build" || cleanType == "all" {
			entries, err := os.ReadDir(versionPath)
			if err != nil {
				continue
			}

			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}

				name := entry.Name()
				if strings.HasPrefix(name, "go-build-") || name == "go-build" {
					isOldFormat := name == "go-build"

					// Filter by old-format flag if specified
					if cleanOldFormat && !isOldFormat {
						continue
					}

					cachePath := filepath.Join(versionPath, name)
					size, files := getDirSize(cachePath)
					sizeGB := float64(size) / (1024 * 1024 * 1024)

					// Determine architecture label
					var arch string
					if isOldFormat {
						arch = "(old format)"
					} else {
						arch = strings.TrimPrefix(name, "go-build-")
					}

					// Get modification time
					modTime, err := getCacheModTime(cachePath)
					if err != nil {
						// If we can't get mod time, use zero time (will be oldest)
						modTime = time.Time{}
					}

					cachesToClean = append(cachesToClean, cacheInfo{
						path:      cachePath,
						sizeGB:    sizeGB,
						sizeBytes: size,
						files:     files,
						version:   version,
						arch:      arch,
						modTime:   modTime,
					})

					totalSize += sizeGB
					totalFiles += files
				}
			}
		}

		// Collect module caches
		if cleanType == "mod" || cleanType == "all" {
			modCachePath := filepath.Join(versionPath, "go-mod")
			if stat, err := os.Stat(modCachePath); err == nil && stat.IsDir() {
				size, files := getDirSize(modCachePath)
				sizeGB := float64(size) / (1024 * 1024 * 1024)

				// Get modification time
				modTime, err := getCacheModTime(modCachePath)
				if err != nil {
					// If we can't get mod time, use zero time (will be oldest)
					modTime = time.Time{}
				}

				cachesToClean = append(cachesToClean, cacheInfo{
					path:      modCachePath,
					sizeGB:    sizeGB,
					sizeBytes: size,
					files:     files,
					version:   version,
					arch:      "modules",
					modTime:   modTime,
				})

				totalSize += sizeGB
				totalFiles += files
			}
		}
	}

	if len(cachesToClean) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No caches found to clean.")
		return nil
	}

	// Apply pruning filters
	originalCount := len(cachesToClean)
	var prunedByAge, prunedBySize int

	// Filter by age first (--older-than)
	if cleanOlderThan != "" {
		maxAge, err := parseDurationWithUnits(cleanOlderThan)
		if err != nil {
			return fmt.Errorf("invalid --older-than value: %w", err)
		}

		cutoff := time.Now().Add(-maxAge)
		var filtered []cacheInfo
		for _, cache := range cachesToClean {
			if cache.modTime.Before(cutoff) {
				filtered = append(filtered, cache)
			}
		}

		prunedByAge = originalCount - len(filtered)
		cachesToClean = filtered
		totalSize = 0
		totalFiles = 0
		for _, cache := range cachesToClean {
			totalSize += cache.sizeGB
			totalFiles += cache.files
		}

		if len(cachesToClean) == 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "No caches older than %s found.\n", cleanOlderThan)
			return nil
		}
	}

	// Filter by size (--max-bytes) - keep newest, delete oldest
	if cleanMaxBytes != "" {
		maxBytes, err := parseByteSize(cleanMaxBytes)
		if err != nil {
			return fmt.Errorf("invalid --max-bytes value: %w", err)
		}

		// Sort by modification time (newest first)
		sort.Slice(cachesToClean, func(i, j int) bool {
			return cachesToClean[i].modTime.After(cachesToClean[j].modTime)
		})

		// Keep caches until we exceed maxBytes
		var kept []cacheInfo
		var keptSize int64
		var toDelete []cacheInfo

		for _, cache := range cachesToClean {
			if keptSize+cache.sizeBytes <= maxBytes {
				kept = append(kept, cache)
				keptSize += cache.sizeBytes
			} else {
				toDelete = append(toDelete, cache)
			}
		}

		// Only delete the excess caches
		cachesToClean = toDelete
		prunedBySize = len(kept)

		totalSize = 0
		totalFiles = 0
		for _, cache := range cachesToClean {
			totalSize += cache.sizeGB
			totalFiles += cache.files
		}

		if len(cachesToClean) == 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "All caches are within the %s limit (%.2f GB kept).\n",
				cleanMaxBytes, float64(keptSize)/(1024*1024*1024))
			return nil
		}
	}

	// Show pruning summary if filters were applied
	if cleanOlderThan != "" || cleanMaxBytes != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "%sPruning Summary:\n", utils.Emoji("🔍 "))
		if cleanOlderThan != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "   • Keeping %d cache(s) (newer than %s)\n", prunedByAge, cleanOlderThan)
			fmt.Fprintf(cmd.OutOrStdout(), "   • Deleting %d cache(s) (older than %s)\n", len(cachesToClean), cleanOlderThan)
		}
		if cleanMaxBytes != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "   • Keeping %d cache(s) (newest caches within %s limit)\n", prunedBySize, cleanMaxBytes)
			fmt.Fprintf(cmd.OutOrStdout(), "   • Deleting %d cache(s) (oldest caches exceeding limit)\n", len(cachesToClean))
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	// Show what will be cleaned
	fmt.Fprintf(cmd.OutOrStdout(), "%sCaches to clean:\n", utils.Emoji("🧹 "))
	fmt.Fprintln(cmd.OutOrStdout())

	sort.Slice(cachesToClean, func(i, j int) bool {
		if cachesToClean[i].version != cachesToClean[j].version {
			return cachesToClean[i].version < cachesToClean[j].version
		}
		return cachesToClean[i].arch < cachesToClean[j].arch
	})

	for _, cache := range cachesToClean {
		if cache.arch == "modules" {
			fmt.Fprintf(cmd.OutOrStdout(), "  Go %-8s [modules]:  %.2f GB (%s files)\n",
				cache.version, cache.sizeGB, formatNumber(cache.files))
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "  Go %-8s [%s]:  %.2f GB (%s files)\n",
				cache.version, cache.arch, cache.sizeGB, formatNumber(cache.files))
		}
	}
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "Total to clean: %.2f GB (%s files)\n", totalSize, formatNumber(totalFiles))
	fmt.Fprintln(cmd.OutOrStdout())

	// If dry-run, stop here
	if cleanDryRun {
		fmt.Fprintf(cmd.OutOrStdout(), "%sDry-run mode: No caches were actually deleted.\n", utils.Emoji("🔍 "))
		fmt.Fprintf(cmd.OutOrStdout(), "%sRun without --dry-run to perform the cleanup.\n", utils.Emoji("💡 "))
		return nil
	}

	// Confirm deletion using enhanced prompt with helpful non-interactive guidance
	confirmed := utils.PromptYesNo(utils.PromptConfig{
		Question: "Proceed with cleaning?",
		NonInteractiveError: fmt.Sprintf(
			"%sRunning in non-interactive mode (no TTY detected)",
			utils.Emoji("⚠️  ")),
		NonInteractiveHelp: []string{
			"",
			"This command requires confirmation. Options:",
			fmt.Sprintf("  1. Add --force flag: goenv cache clean %s --force", cleanType),
			fmt.Sprintf("  2. Use dry-run first: goenv cache clean %s --dry-run", cleanType),
			"  3. Set env var: GOENV_ASSUME_YES=1 goenv cache clean",
			"",
			"For CI/CD, we recommend: GOENV_ASSUME_YES=1",
		},
		AutoConfirm: cleanForce,
		Writer:      cmd.OutOrStdout(),
		ErrWriter:   cmd.ErrOrStderr(),
	})
	if !confirmed {
		fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
		return nil
	}

	// Clean caches
	fmt.Fprintln(cmd.OutOrStdout(), "Cleaning...")
	fmt.Fprintln(cmd.OutOrStdout())

	var cleaned int
	var cleanedSize float64
	var cleanedFiles int
	var failed int

	// Separate build caches from module caches
	var buildCaches []cacheInfo
	var modCachesByVersion = make(map[string]cacheInfo)

	for _, cache := range cachesToClean {
		if cache.arch == "modules" {
			modCachesByVersion[cache.version] = cache
		} else {
			buildCaches = append(buildCaches, cache)
		}
	}

	// Clean build caches (direct removal is fine - no read-only files)
	if len(buildCaches) > 0 && cleanVerbose {
		fmt.Fprintln(cmd.OutOrStdout(), "→ Cleaning build caches...")
	}
	for _, cache := range buildCaches {
		if cleanVerbose {
			fmt.Fprintf(cmd.OutOrStdout(), "  Removing %s...\n", cache.path)
		}
		err := os.RemoveAll(cache.path)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "  ✗ Failed to clean %s: %v\n", cache.path, err)
			failed++
		} else {
			if cleanVerbose {
				fmt.Fprintf(cmd.OutOrStdout(), "  ✓ Cleaned Go %s [%s] (%.2f GB, %s files)\n",
					cache.version, cache.arch, cache.sizeGB, formatNumber(cache.files))
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "  ✓ Cleaned Go %s [%s]\n", cache.version, cache.arch)
			}
			cleaned++
			cleanedSize += cache.sizeGB
			cleanedFiles += cache.files
		}
	}

	// Clean module caches (use 'go clean -modcache' to handle read-only files)
	if len(modCachesByVersion) > 0 && cleanVerbose {
		fmt.Fprintln(cmd.OutOrStdout(), "→ Cleaning module caches...")
	}
	for version, cache := range modCachesByVersion {
		if cleanVerbose {
			fmt.Fprintf(cmd.OutOrStdout(), "  Running 'go clean -modcache' for Go %s...\n", version)
		}
		// Use goenv exec to run 'go clean -modcache' with proper version and GOMODCACHE set
		// Explicitly set GOMODCACHE to ensure we clean the version-specific isolated cache
		// (cache.path is the modCachePath: GOENV_ROOT/versions/{version}/go-mod)
		cleanCmd := exec.Command("goenv", "exec", "go", "clean", "-modcache")
		cleanCmd.Env = append(os.Environ(),
			fmt.Sprintf("GOENV_VERSION=%s", version),
			fmt.Sprintf("GOMODCACHE=%s", cache.path))

		// Show output only in verbose mode
		if cleanVerbose {
			cleanCmd.Stdout = cmd.OutOrStdout()
			cleanCmd.Stderr = cmd.ErrOrStderr()
		} else {
			cleanCmd.Stdout = nil
			cleanCmd.Stderr = nil
		}

		if err := cleanCmd.Run(); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "  ✗ Failed to clean Go %s [modules]: %v\n", version, err)
			failed++
		} else {
			if cleanVerbose {
				fmt.Fprintf(cmd.OutOrStdout(), "  ✓ Cleaned Go %s [modules] (%.2f GB, %s files)\n",
					version, cache.sizeGB, formatNumber(cache.files))
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "  ✓ Cleaned Go %s [modules]\n", version)
			}
			cleaned++
			cleanedSize += cache.sizeGB
			cleanedFiles += cache.files
		}
	}

	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintf(cmd.OutOrStdout(), "%sCleaned %d cache(s)\n", utils.Emoji("✅ "), cleaned)
	fmt.Fprintf(cmd.OutOrStdout(), "%sReclaimed: %.2f GB (%s files)\n", utils.Emoji("💾 "), cleanedSize, formatNumber(cleanedFiles))
	if failed > 0 {
		fmt.Fprintf(cmd.ErrOrStderr(), "%sFailed: %d cache(s)\n", utils.Emoji("❌ "), failed)
	}

	return nil
}

func runCacheMigrate(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	// Get installed versions
	versions, err := mgr.ListInstalledVersions()
	if err != nil {
		return fmt.Errorf("cannot list installed versions: %w", err)
	}

	if len(versions) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No Go versions installed.")
		return nil
	}

	// Detect current system architecture
	// Use runtime.GOOS/GOARCH (not GOOS/GOARCH env vars which may be set for cross-compilation)
	// This ensures we migrate caches to the correct host architecture
	currentGOOS := runtime.GOOS
	currentGOARCH := runtime.GOARCH
	targetArch := fmt.Sprintf("%s-%s", currentGOOS, currentGOARCH)

	// Find old format caches
	type migrationTask struct {
		oldPath string
		newPath string
		version string
		sizeGB  float64
		files   int
	}

	var migrations []migrationTask

	for _, version := range versions {
		versionPath := filepath.Join(cfg.VersionsDir(), version)
		oldCachePath := filepath.Join(versionPath, "go-build")

		// Check if old format cache exists
		if stat, err := os.Stat(oldCachePath); err == nil && stat.IsDir() {
			size, files := getDirSize(oldCachePath)
			sizeGB := float64(size) / (1024 * 1024 * 1024)

			newCachePath := filepath.Join(versionPath, fmt.Sprintf("go-build-%s", targetArch))

			migrations = append(migrations, migrationTask{
				oldPath: oldCachePath,
				newPath: newCachePath,
				version: version,
				sizeGB:  sizeGB,
				files:   files,
			})
		}
	}

	if len(migrations) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "%sNo old format caches found. All caches are already using the new architecture-aware format.\n", utils.Emoji("✅ "))
		return nil
	}

	// Show what will be migrated
	fmt.Fprintf(cmd.OutOrStdout(), "%sCache Migration\n", utils.Emoji("🔄 "))
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "Target architecture: %s\n", targetArch)
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "Old format caches to migrate:")
	fmt.Fprintln(cmd.OutOrStdout())

	var totalSize float64
	var totalFiles int

	for _, m := range migrations {
		fmt.Fprintf(cmd.OutOrStdout(), "  Go %-8s: %.2f GB (%s files)\n",
			m.version, m.sizeGB, formatNumber(m.files))
		totalSize += m.sizeGB
		totalFiles += m.files
	}

	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "Total: %.2f GB (%s files)\n", totalSize, formatNumber(totalFiles))
	fmt.Fprintln(cmd.OutOrStdout())

	// Show what migration will do
	fmt.Fprintln(cmd.OutOrStdout(), "This will:")
	fmt.Fprintln(cmd.OutOrStdout(), "  1. Move old format caches to architecture-specific directories")
	fmt.Fprintln(cmd.OutOrStdout(), "  2. Preserve all cached build artifacts")
	fmt.Fprintln(cmd.OutOrStdout(), "  3. Enable proper cache isolation")
	fmt.Fprintln(cmd.OutOrStdout())

	// Confirm migration using enhanced prompt with helpful non-interactive guidance
	confirmed := utils.PromptYesNo(utils.PromptConfig{
		Question: "Proceed with migration?",
		NonInteractiveError: fmt.Sprintf(
			"%sRunning in non-interactive mode (no TTY detected)",
			utils.Emoji("⚠️  ")),
		NonInteractiveHelp: []string{
			"",
			"This command requires confirmation. Options:",
			"  1. Add --force flag: goenv cache migrate --force",
			"  2. Set env var: GOENV_ASSUME_YES=1 goenv cache migrate",
			"",
			"For CI/CD, we recommend: GOENV_ASSUME_YES=1",
		},
		AutoConfirm: migrateForce,
		Writer:      cmd.OutOrStdout(),
		ErrWriter:   cmd.ErrOrStderr(),
	})
	if !confirmed {
		fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
		return nil
	}

	// Perform migration
	fmt.Fprintln(cmd.OutOrStdout(), "Migrating...")
	fmt.Fprintln(cmd.OutOrStdout())

	var migrated int
	var migratedSize float64
	var migratedFiles int
	var failed int

	for _, m := range migrations {
		// Check if target already exists
		if stat, err := os.Stat(m.newPath); err == nil && stat.IsDir() {
			fmt.Fprintf(cmd.OutOrStdout(), "  %sGo %s: Target cache already exists, skipping\n", utils.Emoji("⚠️  "), m.version)
			continue
		}

		// Rename old cache to new cache
		err := os.Rename(m.oldPath, m.newPath)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "  ✗ Go %s: Failed to migrate: %v\n", m.version, err)
			failed++
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "  ✓ Go %s: Migrated to %s\n", m.version, targetArch)
			migrated++
			migratedSize += m.sizeGB
			migratedFiles += m.files
		}
	}

	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintf(cmd.OutOrStdout(), "%sMigrated %d cache(s)\n", utils.Emoji("✅ "), migrated)
	fmt.Fprintf(cmd.OutOrStdout(), "%sSize: %.2f GB (%s files)\n", utils.Emoji("📦 "), migratedSize, formatNumber(migratedFiles))
	if failed > 0 {
		fmt.Fprintf(cmd.ErrOrStderr(), "%sFailed: %d cache(s)\n", utils.Emoji("❌ "), failed)
	}
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "%sNext steps:\n", utils.Emoji("💡 "))
	fmt.Fprintln(cmd.OutOrStdout(), "  • Run 'goenv cache status' to verify the migration")
	fmt.Fprintln(cmd.OutOrStdout(), "  • Run 'goenv doctor' to check for any issues")

	return nil
}

// getDirSize calculates total size and file count of a directory recursively
// Uses filepath.WalkDir (Go 1.16+) which is more efficient than filepath.Walk
// because it avoids calling os.Stat on every entry.
func getDirSize(path string) (size int64, files int) {
	return getDirSizeWithOptions(path, false, 10*time.Second)
}

// getDirSizeWithOptions calculates directory size with performance options
// - fast: if true, skips file counting (returns -1 for files)
// - timeout: if walk exceeds this duration, returns approximate results
func getDirSizeWithOptions(path string, fast bool, timeout time.Duration) (size int64, files int) {
	startTime := time.Now()
	timedOut := false

	err := filepath.WalkDir(path, func(filePath string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Check timeout budget every 1000 files to avoid excessive time.Now() calls
		if !timedOut && files%1000 == 0 && time.Since(startTime) > timeout {
			timedOut = true
			// Don't return error - continue with what we have
		}

		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return nil // Skip if we can't get file info
			}
			size += info.Size()

			// In fast mode, skip counting files (or timeout)
			if !fast && !timedOut {
				files++
			}
		}
		return nil
	})

	if err != nil {
		return 0, 0
	}

	// Return -1 for files if in fast mode or timed out (indicates approximate)
	if fast || timedOut {
		return size, -1
	}

	return size, files
}

// formatNumber formats a number with thousand separators
func formatNumber(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}

	// Add commas
	var result []rune
	for i, digit := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, digit)
	}

	return string(result)
}

// formatFileCount formats file count, handling -1 for approximate/unknown counts
func formatFileCount(n int) string {
	if n < 0 {
		return "~"
	}
	return formatNumber(n)
}

// parseABIFromCacheName extracts GOOS, GOARCH, and ABI variants from cache directory name
// Examples:
//
//	"go-build-linux-amd64-v3" -> goos="linux", goarch="amd64", abi={"GOAMD64":"v3"}
//	"go-build-darwin-arm64" -> goos="darwin", goarch="arm64", abi=nil
//	"go-build" -> goos="", goarch="", abi=nil (old format)
func parseABIFromCacheName(cacheName string) (goos, goarch string, abi map[string]string) {
	if cacheName == "go-build" {
		return "", "", nil
	}

	// Remove "go-build-" prefix
	suffix := strings.TrimPrefix(cacheName, "go-build-")
	if suffix == cacheName {
		return "", "", nil // Invalid format
	}

	// Split by '-' to get components
	parts := strings.Split(suffix, "-")
	if len(parts) < 2 {
		return "", "", nil
	}

	goos = parts[0]
	goarch = parts[1]

	// Check for ABI variants in remaining parts
	if len(parts) > 2 {
		abi = make(map[string]string)
		remaining := parts[2:]

		// Process remaining parts for ABI variants, experiments, and CGO hash
		i := 0
		for i < len(remaining) {
			part := remaining[i]

			// Check for CGO hash (format: "cgo-<hash>")
			if part == "cgo" && i+1 < len(remaining) {
				abi["CGO_HASH"] = remaining[i+1]
				i += 2 // Skip both "cgo" and hash
				continue
			}

			// Check for GOEXPERIMENT (format: "exp-<experiments>")
			if part == "exp" && i+1 < len(remaining) {
				expValue := strings.ReplaceAll(remaining[i+1], "-", ",")
				abi["GOEXPERIMENT"] = expValue
				i += 2 // Skip both "exp" and value
				continue
			}

			// Check for architecture-specific ABI variants
			switch goarch {
			case "amd64":
				if strings.HasPrefix(part, "v") {
					abi["GOAMD64"] = part
				}
			case "arm":
				// Accept both "v6", "v7" and "6", "7" formats
				if strings.HasPrefix(part, "v") {
					abi["GOARM"] = strings.TrimPrefix(part, "v")
				} else if len(part) == 1 && part >= "5" && part <= "7" {
					// Accept bare digits 5-7 for ARM variants
					abi["GOARM"] = part
				}
			case "386":
				if part == "sse2" || part == "softfloat" {
					abi["GO386"] = part
				}
			case "mips", "mipsle":
				if part == "hardfloat" || part == "softfloat" {
					abi["GOMIPS"] = part
				}
			case "mips64", "mips64le":
				if part == "hardfloat" || part == "softfloat" {
					abi["GOMIPS64"] = part
				}
			case "ppc64", "ppc64le":
				if part == "power8" || part == "power9" || part == "power10" {
					abi["GOPPC64"] = part
				}
			case "riscv64":
				abi["GORISCV64"] = part
			case "wasm":
				abi["GOWASM"] = part
			}

			i++
		}
	}

	return goos, goarch, abi
}

// runCacheInfo shows CGO toolchain information for build caches
func runCacheInfo(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	// Determine which versions to show
	var versions []string
	if len(args) > 0 {
		versions = []string{args[0]}
	} else {
		var err error
		versions, err = mgr.ListInstalledVersions()
		if err != nil {
			return fmt.Errorf("cannot list installed versions: %w", err)
		}
	}

	if len(versions) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No Go versions installed.")
		return nil
	}

	// Collect cache information
	type cacheInfoResult struct {
		Version   string         `json:"version"`
		CacheDir  string         `json:"cache_dir"`
		BuildInfo *cgo.BuildInfo `json:"build_info,omitempty"`
		Error     string         `json:"error,omitempty"`
	}

	var results []cacheInfoResult

	for _, version := range versions {
		versionPath := filepath.Join(cfg.VersionsDir(), version)

		// Find all build cache directories
		entries, err := os.ReadDir(versionPath)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			name := entry.Name()
			// Look for go-build-* directories
			if !strings.HasPrefix(name, "go-build-") {
				continue
			}

			cachePath := filepath.Join(versionPath, name)
			result := cacheInfoResult{
				Version:  version,
				CacheDir: name,
			}

			// Try to read build.info
			buildInfo, err := cgo.ReadBuildInfo(cachePath)
			if err == nil {
				result.BuildInfo = buildInfo
			} else if !os.IsNotExist(err) {
				result.Error = fmt.Sprintf("failed to read build.info: %v", err)
			}

			results = append(results, result)
		}
	}

	// Output results
	if infoJSON {
		// JSON output
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(results); err != nil {
			return fmt.Errorf("failed to encode JSON: %w", err)
		}
	} else {
		// Human-readable output
		if len(results) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No build caches with CGO information found.")
			fmt.Fprintln(cmd.OutOrStdout())
			fmt.Fprintf(cmd.OutOrStdout(), "%sBuild caches with CGO enabled will automatically record toolchain information.\n", utils.Emoji("💡 "))
			return nil
		}

		fmt.Fprintf(cmd.OutOrStdout(), "%sCGO Toolchain Information\n", utils.Emoji("🔧 "))
		fmt.Fprintln(cmd.OutOrStdout())

		for _, result := range results {
			fmt.Fprintf(cmd.OutOrStdout(), "Version: %s\n", result.Version)
			fmt.Fprintf(cmd.OutOrStdout(), "Cache:   %s\n", result.CacheDir)

			if result.Error != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Error:   %s\n", result.Error)
			} else if result.BuildInfo != nil {
				info := result.BuildInfo
				fmt.Fprintf(cmd.OutOrStdout(), "Created: %s\n", info.Created.Format(time.RFC3339))

				if info.CC != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "CC:      %s\n", info.CC)
					if info.CCVersion != "" {
						fmt.Fprintf(cmd.OutOrStdout(), "         %s\n", info.CCVersion)
					}
				}

				if info.CXX != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "CXX:     %s\n", info.CXX)
					if info.CXXVersion != "" {
						fmt.Fprintf(cmd.OutOrStdout(), "         %s\n", info.CXXVersion)
					}
				}

				if info.CFLAGS != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "CFLAGS:  %s\n", info.CFLAGS)
				}

				if info.LDFLAGS != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "LDFLAGS: %s\n", info.LDFLAGS)
				}

				if info.PKGConfig != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "PKG_CONFIG: %s\n", info.PKGConfig)
				}

				if info.PKGConfigPath != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "PKG_CONFIG_PATH: %s\n", info.PKGConfigPath)
				}

				if info.ToolchainHash != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "Hash:    %s\n", info.ToolchainHash[:16]+"...")
				}
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "         (No CGO toolchain info - cache created without CGO)")
			}

			fmt.Fprintln(cmd.OutOrStdout())
		}
	}

	return nil
}

// parseByteSize parses byte size strings like "1GB", "500MB", "1.5GB" to bytes
func parseByteSize(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" {
		return 0, fmt.Errorf("empty byte size")
	}

	// Extract numeric part and unit
	var numStr string
	var unit string

	for i, c := range s {
		if (c >= '0' && c <= '9') || c == '.' {
			numStr += string(c)
		} else {
			unit = s[i:]
			break
		}
	}

	if numStr == "" {
		return 0, fmt.Errorf("invalid byte size format: %s", s)
	}

	// Parse the number
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number in byte size: %w", err)
	}

	// Parse the unit
	var multiplier float64
	switch unit {
	case "B", "":
		multiplier = 1
	case "KB", "K":
		multiplier = 1024
	case "MB", "M":
		multiplier = 1024 * 1024
	case "GB", "G":
		multiplier = 1024 * 1024 * 1024
	case "TB", "T":
		multiplier = 1024 * 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("unknown unit in byte size: %s (valid units: B, KB, MB, GB, TB)", unit)
	}

	return int64(num * multiplier), nil
}

// parseDurationWithUnits parses duration strings like "30d", "1w", "24h"
// Supports: d (days), w (weeks), h (hours), m (minutes), s (seconds)
func parseDurationWithUnits(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}

	// Try standard time.ParseDuration first (handles h, m, s, ms, us, ns)
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}

	// Handle custom units: d (days), w (weeks)
	var numStr string
	var unit string

	for i, c := range s {
		if (c >= '0' && c <= '9') || c == '.' {
			numStr += string(c)
		} else {
			unit = s[i:]
			break
		}
	}

	if numStr == "" {
		return 0, fmt.Errorf("invalid duration format: %s", s)
	}

	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number in duration: %w", err)
	}

	switch unit {
	case "d", "day", "days":
		return time.Duration(num * float64(24*time.Hour)), nil
	case "w", "week", "weeks":
		return time.Duration(num * float64(7*24*time.Hour)), nil
	default:
		return 0, fmt.Errorf("unknown unit in duration: %s (valid units: s, m, h, d, w)", unit)
	}
}

// getCacheModTime returns the last modification time of a cache directory
func getCacheModTime(path string) (time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}
