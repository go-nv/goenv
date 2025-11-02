package diagnostics

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/cache"
	"github.com/go-nv/goenv/internal/cgo"
	"github.com/go-nv/goenv/internal/cmdutil"
	"github.com/go-nv/goenv/internal/envdetect"
	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/go-nv/goenv/internal/platform"
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
  status   Show cache sizes and locations
  clean    Clean build or module caches to reclaim disk space
  migrate  Migrate old format caches to architecture-aware format
  info     Show CGO toolchain information for caches

Note: Module caches are automatically shared across all Go versions
(at $GOENV_ROOT/shared/go-mod)

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

func runCacheStatus(cmd *cobra.Command, args []string) error {
	cfg, mgr := cmdutil.SetupContext()

	// Get installed versions (for validation)
	versions, err := mgr.ListInstalledVersions()
	if err != nil {
		return errors.FailedTo("list installed versions", err)
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
					GOOS:      platform.OS(),
					GOARCH:    platform.Arch(),
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
			return cmdutil.OutputJSON(cmd.OutOrStdout(), result)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "No Go versions installed.")
		return nil
	}

	// Create cache manager
	cacheMgr := cache.NewManager(cfg)

	// Get cache status using the cache package
	status, err := cacheMgr.GetStatus(statusFast)
	if err != nil {
		return errors.FailedTo("get cache status", err)
	}

	// Check if we should suggest --fast mode
	if !statusFast && !statusJSON && len(status.BuildCaches) > 3 {
		// Estimate if user should use --fast
		avgFiles := 0
		if len(status.BuildCaches) > 0 && status.TotalFiles > 0 {
			avgFiles = status.TotalFiles / len(status.BuildCaches)
			if avgFiles > 1000 {
				fmt.Fprintf(cmd.OutOrStdout(),
					"%s Large cache detected (%s+ files). "+
						"Use --fast for 5-10x faster scanning.\n\n",
					utils.Emoji("💡"), cache.FormatNumber(status.TotalFiles))
			}
		}
	}

	// Convert to JSON format if requested
	if statusJSON {
		result := cacheStatusJSON{
			SchemaVersion: "1",
			Tool: toolInfo{
				Name:    "goenv",
				Version: cmdpkg.AppVersion,
			},
			Host: hostInfo{
				GOOS:      platform.OS(),
				GOARCH:    platform.Arch(),
				Rosetta:   envdetect.IsRosetta(),
				WSL:       envdetect.IsWSL(),
				Container: envdetect.IsInContainer(),
			},
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Caches:    make([]cacheEntry, 0),
			Totals: cacheTotals{
				SizeBytes: status.TotalSize,
				Entries:   status.TotalFiles,
			},
		}

		// Convert cache.CacheInfo to cacheEntry
		for _, c := range append(status.BuildCaches, status.ModCaches...) {
			entry := cacheEntry{
				Kind:      c.Kind.String(),
				Path:      c.Path,
				GoVersion: c.GoVersion,
				SizeBytes: c.SizeBytes,
				Entries:   c.Files,
				Exists:    true,
				OldFormat: c.OldFormat,
			}

			if c.Target != nil {
				entry.Target = &targetInfo{
					GOOS:   c.Target.GOOS,
					GOARCH: c.Target.GOARCH,
				}
				entry.ABI = c.Target.ABI
			}

			result.Caches = append(result.Caches, entry)
		}

		return cmdutil.OutputJSON(cmd.OutOrStdout(), result)
	}

	// Human-readable output
	out := cmd.OutOrStdout()

	// Group by version for display
	if len(status.ByVersion) == 0 {
		fmt.Fprintln(out, "No caches found.")
		return nil
	}

	// Sort versions for consistent output
	sortedVersions := make([]string, 0, len(status.ByVersion))
	for v := range status.ByVersion {
		sortedVersions = append(sortedVersions, v)
	}
	slices.Sort(sortedVersions)

	for _, version := range sortedVersions {
		versionCaches := status.ByVersion[version]
		fmt.Fprintf(out, "%s Go %s │ (%s)\n",
			utils.Emoji("📦"),
			version,
			cache.FormatBytes(versionCaches.TotalSize))

		// Display build caches
		for _, c := range versionCaches.BuildCaches {
			archLabel := "unknown"
			if c.OldFormat {
				archLabel = "(old format)"
			} else if c.Target != nil {
				archLabel = fmt.Sprintf("%s-%s", c.Target.GOOS, c.Target.GOARCH)
				if len(c.Target.ABI) > 0 {
					// Add ABI details
					for k, v := range c.Target.ABI {
						archLabel += fmt.Sprintf(" %s=%s", strings.ToLower(strings.TrimPrefix(k, "GO")), v)
					}
				}
			}

			fileCount := cache.FormatFileCount(c.Files, c.Files < 0)
			fmt.Fprintf(out, "  %s Build %s: %s [%s] (%s files)\n",
				utils.Emoji("🔨"),
				archLabel,
				cache.FormatBytes(c.SizeBytes),
				filepath.Base(c.Path),
				fileCount)
		}

		// Display module cache
		if versionCaches.ModCache != nil {
			c := versionCaches.ModCache
			fileCount := cache.FormatFileCount(c.Files, c.Files < 0)
			fmt.Fprintf(out, "  %s Modules: %s [%s] (%s files)\n",
				utils.Emoji("📚"),
				cache.FormatBytes(c.SizeBytes),
				filepath.Base(c.Path),
				fileCount)
		}

		fmt.Fprintln(out)
	}

	// Display totals
	totalFileCount := cache.FormatFileCount(status.TotalFiles, status.TotalFiles < 0)
	fmt.Fprintf(out, "%s Total: %s (%s files)\n",
		utils.Emoji("💾"),
		cache.FormatBytes(status.TotalSize),
		totalFileCount)

	// Show tips
	if len(status.BuildCaches) > 0 {
		hasOldFormat := false
		for _, c := range status.BuildCaches {
			if c.OldFormat {
				hasOldFormat = true
				break
			}
		}

		if hasOldFormat {
			fmt.Fprintf(out, "\n%s Tip: Run 'goenv cache migrate' to convert old format caches to architecture-aware format.\n",
				utils.Emoji("💡"))
		}
	}

	if status.TotalSize > 5*1024*1024*1024 { // > 5GB
		fmt.Fprintf(out, "\n%s Tip: Run 'goenv cache clean' to free up disk space.\n",
			utils.Emoji("💡"))
	}

	return nil
}

func runCacheClean(cmd *cobra.Command, args []string) error {
	cfg, mgr := cmdutil.SetupContext()
	ctx := cmdutil.NewInteractiveContext(cmd)

	// Default to 'build' if no argument provided
	cleanType := "build"
	if len(args) > 0 {
		cleanType = args[0]
	}

	// Validate cache type
	var kind cache.CacheKind
	switch cleanType {
	case "build":
		kind = cache.CacheKindBuild
	case "mod", "module", "modules":
		kind = cache.CacheKindMod
	case "all":
		kind = "" // Empty means all
	default:
		return fmt.Errorf("invalid type: %s (must be 'build', 'mod', or 'all')", cleanType)
	}

	// Get installed versions (for validation)
	versions, err := mgr.ListInstalledVersions()
	if err != nil {
		return errors.FailedTo("list installed versions", err)
	}

	if len(versions) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No Go versions installed.")
		return nil
	}

	// Validate version flag if specified
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
	}

	// Parse flags
	var maxBytes int64
	if cleanMaxBytes != "" {
		parsed, err := cache.ParseByteSize(cleanMaxBytes)
		if err != nil {
			return fmt.Errorf("invalid --max-bytes value: %w", err)
		}
		maxBytes = parsed
	}

	var olderThan time.Duration
	if cleanOlderThan != "" {
		parsed, err := cache.ParseDuration(cleanOlderThan)
		if err != nil {
			return fmt.Errorf("invalid --older-than value: %w", err)
		}
		olderThan = parsed
	}

	// Create cache manager
	cacheMgr := cache.NewManager(cfg)

	// Build clean options
	opts := cache.CleanOptions{
		Kind:      kind,
		Version:   cleanVersion,
		OldFormat: cleanOldFormat,
		MaxBytes:  maxBytes,
		OlderThan: olderThan,
		DryRun:    cleanDryRun,
		Verbose:   cleanVerbose,
	}

	// Preview what will be cleaned
	previewCaches, err := cacheMgr.List()
	if err != nil {
		return errors.FailedTo("list caches", err)
	}

	// Apply filters to get actual list
	actualCaches := previewCaches
	if kind != "" {
		filtered := make([]cache.CacheInfo, 0)
		for _, c := range actualCaches {
			if c.Kind == kind {
				filtered = append(filtered, c)
			}
		}
		actualCaches = filtered
	}
	if cleanVersion != "" {
		filtered := make([]cache.CacheInfo, 0)
		for _, c := range actualCaches {
			if c.GoVersion == cleanVersion {
				filtered = append(filtered, c)
			}
		}
		actualCaches = filtered
	}
	if cleanOldFormat {
		filtered := make([]cache.CacheInfo, 0)
		for _, c := range actualCaches {
			if c.OldFormat {
				filtered = append(filtered, c)
			}
		}
		actualCaches = filtered
	}
	if olderThan > 0 {
		cutoff := time.Now().Add(-olderThan)
		filtered := make([]cache.CacheInfo, 0)
		for _, c := range actualCaches {
			if c.ModTime.Before(cutoff) {
				filtered = append(filtered, c)
			}
		}
		actualCaches = filtered
	}

	if len(actualCaches) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No caches found to clean.")
		return nil
	}

	// Show what will be cleaned
	totalSize := int64(0)
	for _, c := range actualCaches {
		totalSize += c.SizeBytes
	}

	out := cmd.OutOrStdout()
	if cleanDryRun {
		fmt.Fprintf(out, "%s Dry run - showing what would be cleaned:\n\n", utils.Emoji("🔍"))
	} else if !cleanForce {
		fmt.Fprintf(out, "%s About to clean %d cache(s), freeing %s:\n\n",
			utils.Emoji("⚠️"), len(actualCaches), cache.FormatBytes(totalSize))
	}

	if !cleanDryRun && !cleanForce && len(actualCaches) > 0 {
		// Ask for confirmation using InteractiveContext
		prompt := fmt.Sprintf("About to clean %d cache(s), freeing %s. Continue?", len(actualCaches), cache.FormatBytes(totalSize))
		if !ctx.Confirm(prompt, false) {
			fmt.Fprintln(out, "Cancelled.")
			return nil
		}
	}

	// Perform clean
	result, err := cacheMgr.Clean(opts)
	if err != nil {
		return errors.FailedTo("clean cache", err)
	}

	// Report results
	if cleanDryRun {
		fmt.Fprintf(out, "\n%s Would remove %d cache(s), freeing %s\n",
			utils.Emoji("✓"),
			result.CachesRemoved,
			cache.FormatBytes(result.BytesReclaimed))
	} else {
		fmt.Fprintf(out, "\n%s Removed %d cache(s), freed %s\n",
			utils.Emoji("✓"),
			result.CachesRemoved,
			cache.FormatBytes(result.BytesReclaimed))
	}

	if len(result.Errors) > 0 {
		fmt.Fprintf(out, "\n%s Encountered %d error(s):\n", utils.Emoji("⚠️"), len(result.Errors))
		for _, err := range result.Errors {
			fmt.Fprintf(out, "  - %v\n", err)
		}
	}

	return nil
}

func runCacheMigrate(cmd *cobra.Command, args []string) error {
	cfg, mgr := cmdutil.SetupContext()
	ctx := cmdutil.NewInteractiveContext(cmd)

	// Get installed versions
	versions, err := mgr.ListInstalledVersions()
	if err != nil {
		return errors.FailedTo("list installed versions", err)
	}

	if len(versions) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No Go versions installed.")
		return nil
	}

	// Create cache manager
	cacheMgr := cache.NewManager(cfg)

	// Detect old format caches
	oldCaches, err := cacheMgr.DetectOldFormatCaches()
	if err != nil {
		return errors.FailedTo("detect old format caches", err)
	}

	if len(oldCaches) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "%s No old format caches found. All caches are already architecture-aware.\n",
			utils.Emoji("✓"))
		return nil
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "%s Found %d old format cache(s) to migrate:\n\n", utils.Emoji("🔄"), len(oldCaches))

	for _, c := range oldCaches {
		fmt.Fprintf(out, "  %s: %s (Go %s)\n",
			filepath.Base(c.Path),
			cache.FormatBytes(c.SizeBytes),
			c.GoVersion)
	}

	targetArch := fmt.Sprintf("%s-%s", platform.OS(), platform.Arch())
	fmt.Fprintf(out, "\nTarget architecture: %s\n", targetArch)

	// Ask for confirmation unless --force
	if !migrateForce {
		prompt := "Migrate caches to target architecture?"
		if !ctx.Confirm(prompt, false) {
			fmt.Fprintln(out, "Cancelled.")
			return nil
		}
	}

	// Perform migration
	result, err := cacheMgr.Migrate(cache.MigrateOptions{
		TargetGOOS:   platform.OS(),
		TargetGOARCH: platform.Arch(),
		Force:        migrateForce,
		DryRun:       false,
		Verbose:      true,
	})
	if err != nil {
		return errors.CacheMigrationFailed(err)
	}

	// Report results
	fmt.Fprintf(out, "\n%s Migrated %d cache(s)\n",
		utils.Emoji("✓"),
		result.CachesMigrated)

	if len(result.Errors) > 0 {
		fmt.Fprintf(out, "\n%s Encountered %d error(s):\n", utils.Emoji("⚠️"), len(result.Errors))
		for _, err := range result.Errors {
			fmt.Fprintf(out, "  - %v\n", err)
		}
	}

	return nil
}

func runCacheInfo(cmd *cobra.Command, args []string) error {
	cfg, mgr := cmdutil.SetupContext()

	// Get installed versions
	versions, err := mgr.ListInstalledVersions()
	if err != nil {
		return errors.FailedTo("list installed versions", err)
	}

	if len(versions) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No Go versions installed.")
		return nil
	}

	// Filter to specific version if requested
	if len(args) > 0 {
		requestedVersion := args[0]
		found := false
		for _, v := range versions {
			if v == requestedVersion {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("version %s is not installed", requestedVersion)
		}
		versions = []string{requestedVersion}
	}

	// Create cache manager
	cacheMgr := cache.NewManager(cfg)

	out := cmd.OutOrStdout()

	// JSON output
	if infoJSON {
		type cgoInfoJSON struct {
			Version   string            `json:"version"`
			BuildInfo *cgo.BuildInfo    `json:"build_info,omitempty"`
			Caches    []cache.CacheInfo `json:"caches"`
		}

		results := make([]cgoInfoJSON, 0)

		for _, version := range versions {
			versionCaches, err := cacheMgr.GetVersionCaches(version)
			if err != nil {
				continue
			}

			info := cgoInfoJSON{
				Version: version,
				Caches:  versionCaches,
			}

			// Try to get CGO build info
			for _, c := range versionCaches {
				if c.Kind == cache.CacheKindBuild {
					buildInfo, err := cgo.ReadBuildInfo(c.Path)
					if err == nil && buildInfo.CC != "" {
						info.BuildInfo = buildInfo
						break
					}
				}
			}

			results = append(results, info)
		}

		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(results)
	}

	// Human-readable output
	for _, version := range versions {
		fmt.Fprintf(out, "%s Go %s\n\n", utils.Emoji("📦"), version)

		versionCaches, err := cacheMgr.GetVersionCaches(version)
		if err != nil {
			fmt.Fprintf(out, "  Error: %v\n\n", err)
			continue
		}

		if len(versionCaches) == 0 {
			fmt.Fprintln(out, "  No caches found.")
			continue
		}

		// Display each cache
		for _, c := range versionCaches {
			archLabel := "unknown"
			if c.Kind == cache.CacheKindBuild {
				if c.OldFormat {
					archLabel = "(old format)"
				} else if c.Target != nil {
					archLabel = fmt.Sprintf("%s-%s", c.Target.GOOS, c.Target.GOARCH)
				}

				fmt.Fprintf(out, "  %s Build cache [%s]:\n", utils.Emoji("🔨"), archLabel)
				fmt.Fprintf(out, "    Path: %s\n", c.Path)
				fmt.Fprintf(out, "    Size: %s\n", cache.FormatBytes(c.SizeBytes))

				// Try to get CGO info
				buildInfo, err := cgo.ReadBuildInfo(c.Path)
				if err == nil && buildInfo.CC != "" {
					fmt.Fprintln(out, "    CGO Toolchain:")
					if buildInfo.CC != "" {
						fmt.Fprintf(out, "      CC:      %s\n", buildInfo.CC)
						if buildInfo.CCVersion != "" {
							fmt.Fprintf(out, "               %s\n", buildInfo.CCVersion)
						}
					}
					if buildInfo.CXX != "" {
						fmt.Fprintf(out, "      CXX:     %s\n", buildInfo.CXX)
						if buildInfo.CXXVersion != "" {
							fmt.Fprintf(out, "               %s\n", buildInfo.CXXVersion)
						}
					}
					if buildInfo.CFLAGS != "" {
						fmt.Fprintf(out, "      CFLAGS:  %s\n", buildInfo.CFLAGS)
					}
					if buildInfo.LDFLAGS != "" {
						fmt.Fprintf(out, "      LDFLAGS: %s\n", buildInfo.LDFLAGS)
					}
					if buildInfo.ToolchainHash != "" {
						fmt.Fprintf(out, "      Hash:    %s...\n", buildInfo.ToolchainHash[:16])
					}
				} else {
					fmt.Fprintln(out, "    CGO: Not used (or no build.info file)")
				}
			} else {
				fmt.Fprintf(out, "  %s Module cache:\n", utils.Emoji("📚"))
				fmt.Fprintf(out, "    Path: %s\n", c.Path)
				fmt.Fprintf(out, "    Size: %s\n", cache.FormatBytes(c.SizeBytes))
			}

			fmt.Fprintln(out)
		}
	}

	return nil
}
