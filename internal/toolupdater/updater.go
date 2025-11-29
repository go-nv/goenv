// Package toolupdater provides automatic update functionality for Go tools.
// It integrates with the existing tools and defaulttools packages to enable
// smart tool version management across Go versions.
package toolupdater

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/defaulttools"
	"github.com/go-nv/goenv/internal/errors"
	toolspkg "github.com/go-nv/goenv/internal/tools"
	"github.com/go-nv/goenv/internal/utils"
)

// UpdateStrategy defines how tools should be updated
type UpdateStrategy string

const (
	// StrategyLatest always updates to the latest version
	StrategyLatest UpdateStrategy = "latest"

	// StrategyMinor updates within the same major version
	StrategyMinor UpdateStrategy = "minor"

	// StrategyPatch updates only patch versions
	StrategyPatch UpdateStrategy = "patch"

	// StrategyPin never updates (pinned version)
	StrategyPin UpdateStrategy = "pin"

	// StrategyAuto decides based on version stability
	StrategyAuto UpdateStrategy = "auto"
)

// UpdateOptions configures tool update behavior
type UpdateOptions struct {
	// Strategy determines how aggressively to update
	Strategy UpdateStrategy

	// GoVersion is the Go version to update tools for
	// If empty, updates for current version
	GoVersion string

	// ToolNames optionally filters which tools to update
	// If empty, checks all installed tools
	ToolNames []string

	// DryRun previews updates without executing
	DryRun bool

	// Force bypasses compatibility checks
	Force bool

	// Verbose enables detailed output
	Verbose bool

	// CheckOnly only checks for updates, doesn't install
	CheckOnly bool
}

// UpdateResult contains the results of an update operation
type UpdateResult struct {
	// Checked is the list of tools that were checked for updates
	Checked []UpdateCheck

	// Updated is the list of tools that were successfully updated
	Updated []string

	// Failed is the list of tools that failed to update
	Failed []string

	// Skipped is the list of tools skipped due to strategy/compatibility
	Skipped []string

	// Errors contains any errors encountered
	Errors []error
}

// UpdateCheck represents the result of checking a tool for updates
type UpdateCheck struct {
	ToolName        string
	CurrentVersion  string
	LatestVersion   string
	UpdateAvailable bool
	Compatible      bool
	Strategy        UpdateStrategy
	Reason          string // Why update was skipped/failed
}

// Updater manages tool updates
type Updater struct {
	cfg         *config.Config
	cache       *Cache
	toolsConfig *defaulttools.Config
}

// NewUpdater creates a new tool updater
func NewUpdater(cfg *config.Config) *Updater {
	// Load defaulttools config for version overrides and strategies
	toolsConfig, _ := defaulttools.LoadConfig(defaulttools.ConfigPath(cfg.Root))
	if toolsConfig == nil {
		toolsConfig = defaulttools.DefaultConfig()
	}

	return &Updater{
		cfg:         cfg,
		cache:       NewCache(cfg),
		toolsConfig: toolsConfig,
	}
}

// CheckForUpdates checks which tools have updates available
func (u *Updater) CheckForUpdates(opts UpdateOptions) (*UpdateResult, error) {
	result := &UpdateResult{
		Checked: []UpdateCheck{},
		Updated: []string{},
		Failed:  []string{},
		Skipped: []string{},
		Errors:  []error{},
	}

	// Determine which Go version to check
	goVersion := opts.GoVersion
	if goVersion == "" {
		// Use current version
		// We'll need to get this from manager, but for now use a placeholder
		goVersion = "current"
	}

	// Get list of installed tools
	tools, err := toolspkg.ListForVersion(u.cfg, goVersion)
	if err != nil {
		return nil, errors.FailedTo("list tools", err)
	}

	if len(tools) == 0 {
		return result, nil
	}

	// Filter tools if specific names requested
	toolsToCheck := tools
	if len(opts.ToolNames) > 0 {
		filtered := []toolspkg.Tool{}
		for _, tool := range tools {
			if utils.SliceContains(opts.ToolNames, tool.Name) {
				filtered = append(filtered, tool)
			}
		}
		toolsToCheck = filtered
	}

	// Check each tool for updates
	for _, tool := range toolsToCheck {
		check := u.checkToolUpdate(tool, opts)
		result.Checked = append(result.Checked, check)

		if !check.UpdateAvailable {
			continue
		}

		if opts.CheckOnly {
			// Just checking, don't update
			continue
		}

		if !check.Compatible && !opts.Force {
			result.Skipped = append(result.Skipped, tool.Name)
			continue
		}

		// Perform update
		if !opts.DryRun {
			if err := u.updateTool(tool, check.LatestVersion, goVersion, opts.Verbose); err != nil {
				result.Failed = append(result.Failed, tool.Name)
				result.Errors = append(result.Errors, fmt.Errorf("%s: %w", tool.Name, err))
			} else {
				result.Updated = append(result.Updated, tool.Name)
			}
		} else {
			// Dry run - would update
			result.Updated = append(result.Updated, tool.Name)
		}
	}

	return result, nil
}

// checkToolUpdate checks if a single tool has an update available
func (u *Updater) checkToolUpdate(tool toolspkg.Tool, opts UpdateOptions) UpdateCheck {
	check := UpdateCheck{
		ToolName:        tool.Name,
		CurrentVersion:  tool.Version,
		UpdateAvailable: false,
		Compatible:      true,
		Strategy:        opts.Strategy,
	}

	// Skip if no package path (can't query updates)
	if tool.PackagePath == "" {
		check.Reason = "no package path available"
		return check
	}

	// Determine target version using version overrides and compatibility checking
	targetVersion, reason := u.resolveTargetVersion(tool, opts.GoVersion)
	if targetVersion == "" {
		check.Reason = reason
		return check
	}

	check.LatestVersion = targetVersion

	// Compare versions
	comparison := toolspkg.CompareVersions(tool.Version, targetVersion)
	if comparison >= 0 {
		// Already at target or newer
		check.Reason = "already up to date"
		return check
	}

	check.UpdateAvailable = true

	// Check strategy compatibility
	if !u.shouldUpdate(tool.Version, targetVersion, opts.Strategy) {
		check.Reason = fmt.Sprintf("update blocked by strategy: %s", opts.Strategy)
		check.UpdateAvailable = false
		return check
	}

	// Check Go version compatibility (if not already handled by latest_compatible)
	if !opts.Force {
		compatible, reason := CheckCompatibility(tool.PackagePath, targetVersion, opts.GoVersion)
		check.Compatible = compatible
		if !compatible {
			check.Reason = reason
			check.UpdateAvailable = false
			return check
		}
	}

	return check
}

// resolveTargetVersion determines the target version for a tool update
// Handles version overrides and latest_compatible strategy
func (u *Updater) resolveTargetVersion(tool toolspkg.Tool, goVersion string) (string, string) {
	// Get tool config from defaulttools if it exists
	toolConfig := u.toolsConfig.GetToolByName(tool.Name)

	// Check for version override first
	if toolConfig != nil && len(toolConfig.VersionOverrides) > 0 {
		if override := u.findVersionOverride(toolConfig, goVersion); override != "" {
			if override == "@latest" {
				// Fall through to query latest
			} else {
				return override, ""
			}
		}
	}

	// Check version strategy (if configured in tool metadata)
	var versionStrategy string
	if toolConfig != nil {
		versionStrategy = toolConfig.VersionStrategy
	}

	if versionStrategy == "latest_compatible" {
		// Find latest compatible version
		version, err := u.findLatestCompatible(tool.PackagePath, goVersion)
		if err != nil {
			return "", fmt.Sprintf("failed to find compatible version: %v", err)
		}
		return version, ""
	}

	// Default: query latest version
	return u.queryLatestVersion(tool.PackagePath)
}

// findVersionOverride checks if there's a version override for the Go version
func (u *Updater) findVersionOverride(toolConfig *defaulttools.Tool, goVersion string) string {
	if toolConfig == nil || len(toolConfig.VersionOverrides) == 0 {
		return ""
	}

	// Try exact match first
	if override, ok := toolConfig.VersionOverrides[goVersion]; ok {
		return override
	}

	// Try pattern matches (e.g., "1.20+", "1.18-1.20")
	for pattern, override := range toolConfig.VersionOverrides {
		if matchesVersionPattern(goVersion, pattern) {
			return override
		}
	}

	return ""
}

// queryLatestVersion queries the latest version from cache or network
func (u *Updater) queryLatestVersion(packagePath string) (string, string) {
	// Check cache first
	cachedVersion, found := u.cache.GetLatestVersion(packagePath)
	if found {
		return cachedVersion, ""
	}

	// Query latest version
	latestVersion, err := toolspkg.GetLatestVersion(packagePath)
	if err != nil {
		return "", fmt.Sprintf("failed to query latest version: %v", err)
	}

	// Cache the result
	u.cache.SetLatestVersion(packagePath, latestVersion)
	return latestVersion, ""
}

// findLatestCompatible finds the latest version compatible with the Go version
func (u *Updater) findLatestCompatible(packagePath, goVersion string) (string, error) {
	// Get all available versions for the package
	versions, err := u.queryAvailableVersions(packagePath)
	if err != nil {
		return "", errors.FailedTo("query versions", err)
	}

	// Sort versions newest first
	// Try each version until we find a compatible one
	for _, version := range versions {
		compatible, _ := CheckCompatibility(packagePath, version, goVersion)
		if compatible {
			return version, nil
		}
	}

	return "", fmt.Errorf("no compatible version found for Go %s", goVersion)
}

// queryAvailableVersions queries all available versions for a package
func (u *Updater) queryAvailableVersions(packagePath string) ([]string, error) {
	// Use go list to get available versions
	output, err := utils.RunCommandOutput("go", "list", "-m", "-versions", packagePath)
	if err != nil {
		return nil, errors.FailedTo("list versions", err)
	}

	// Parse output: "packagePath v1.0.0 v1.1.0 v1.2.0"
	parts := strings.Fields(output)
	if len(parts) < 2 {
		return nil, fmt.Errorf("unexpected output format")
	}

	versions := parts[1:] // Skip package name
	// Reverse to get newest first
	for i := 0; i < len(versions)/2; i++ {
		versions[i], versions[len(versions)-1-i] = versions[len(versions)-1-i], versions[i]
	}

	return versions, nil
}

// matchesVersionPattern checks if a Go version matches a pattern
func matchesVersionPattern(goVersion, pattern string) bool {
	// Handle "X.Y+" pattern (X.Y or higher)
	if strings.HasSuffix(pattern, "+") {
		baseVersion := strings.TrimSuffix(pattern, "+")
		return utils.CompareGoVersions(goVersion, baseVersion) >= 0
	}

	// Handle "X.Y-X.Z" range pattern
	if strings.Contains(pattern, "-") {
		parts := strings.Split(pattern, "-")
		if len(parts) == 2 {
			return utils.CompareGoVersions(goVersion, parts[0]) >= 0 &&
				utils.CompareGoVersions(goVersion, parts[1]) <= 0
		}
	}

	// Exact match
	return goVersion == pattern
}

// shouldUpdate determines if an update should proceed based on strategy
func (u *Updater) shouldUpdate(currentVersion, latestVersion string, strategy UpdateStrategy) bool {
	switch strategy {
	case StrategyPin:
		return false
	case StrategyLatest:
		return true
	case StrategyAuto:
		// Auto strategy: update if it's a stable version
		return !isPreRelease(latestVersion)
	case StrategyMinor:
		// Only update within same major version (e.g., v1.2.3 -> v1.3.0 OK, v1.2.3 -> v2.0.0 NOT OK)
		currentMajor, _, _, err := toolspkg.ParseSemver(currentVersion)
		if err != nil {
			return true // If can't parse, allow update
		}
		latestMajor, _, _, err := toolspkg.ParseSemver(latestVersion)
		if err != nil {
			return true // If can't parse, allow update
		}
		return currentMajor == latestMajor
	case StrategyPatch:
		// Only update within same major.minor version (e.g., v1.2.3 -> v1.2.4 OK, v1.2.3 -> v1.3.0 NOT OK)
		currentMajor, currentMinor, _, err := toolspkg.ParseSemver(currentVersion)
		if err != nil {
			return true // If can't parse, allow update
		}
		latestMajor, latestMinor, _, err := toolspkg.ParseSemver(latestVersion)
		if err != nil {
			return true // If can't parse, allow update
		}
		return currentMajor == latestMajor && currentMinor == latestMinor
	default:
		return true
	}
}

// updateTool performs the actual tool update
func (u *Updater) updateTool(tool toolspkg.Tool, version, goVersion string, verbose bool) error {
	// Build package path with version
	packagePath := tool.PackagePath
	if version != "" {
		packagePath = packagePath + "@" + version
	} else {
		packagePath = packagePath + "@latest"
	}

	// Set up paths
	versionPath := filepath.Join(u.cfg.Root, "versions", goVersion)
	goRoot := versionPath
	goBin := filepath.Join(goRoot, "bin", "go")
	gopath := filepath.Join(versionPath, "gopath")

	// Check if Go binary exists
	if utils.FileNotExists(goBin) {
		return fmt.Errorf("go binary not found for version %s", goVersion)
	}

	// Ensure GOPATH exists
	if err := utils.EnsureDirWithContext(filepath.Join(gopath, "bin"), "create GOPATH"); err != nil {
		return err
	}

	// Run go install
	cmd := exec.Command(goBin, "install", packagePath)
	cmd.Env = append(os.Environ(),
		utils.EnvVarGoroot+"="+goRoot,
		utils.EnvVarGopath+"="+gopath,
	)

	// Set shared GOMODCACHE if not already set (matches exec.go behavior)
	if os.Getenv(utils.EnvVarGomodcache) == "" {
		sharedGomodcache := filepath.Join(u.cfg.Root, "shared", "go-mod")
		cmd.Env = append(cmd.Env, utils.EnvVarGomodcache+"="+sharedGomodcache)
	}

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go install failed: %w", err)
	}

	return nil
}

// UpdateAll updates all tools to their latest compatible versions
func (u *Updater) UpdateAll(goVersion string, strategy UpdateStrategy, dryRun, verbose bool) (*UpdateResult, error) {
	opts := UpdateOptions{
		Strategy:  strategy,
		GoVersion: goVersion,
		DryRun:    dryRun,
		Verbose:   verbose,
	}

	return u.CheckForUpdates(opts)
}

// UpdateSpecific updates specific tools by name
func (u *Updater) UpdateSpecific(toolNames []string, goVersion string, strategy UpdateStrategy, dryRun, verbose bool) (*UpdateResult, error) {
	opts := UpdateOptions{
		Strategy:  strategy,
		GoVersion: goVersion,
		ToolNames: toolNames,
		DryRun:    dryRun,
		Verbose:   verbose,
	}

	return u.CheckForUpdates(opts)
}

// CheckUpdatesOnly checks for updates without installing
func (u *Updater) CheckUpdatesOnly(goVersion string) (*UpdateResult, error) {
	opts := UpdateOptions{
		Strategy:  StrategyLatest,
		GoVersion: goVersion,
		CheckOnly: true,
	}

	return u.CheckForUpdates(opts)
}

// GetUpdateCache returns the cache for inspection
func (u *Updater) GetUpdateCache() *Cache {
	return u.cache
}

// ClearCache clears the update cache
func (u *Updater) ClearCache() error {
	return u.cache.Clear()
}

// Helper functions

func isPreRelease(version string) bool {
	// Simple check: if version contains -rc, -beta, -alpha, it's a pre-release
	if len(version) == 0 {
		return false
	}

	// Check for pre-release identifiers
	preReleaseMarkers := []string{"-rc", "-beta", "-alpha", "-pre"}
	for _, marker := range preReleaseMarkers {
		if strings.Contains(version, marker) {
			return true
		}
	}

	// Check for 0.x.x versions (considered pre-release)
	if strings.HasPrefix(version, "v0.") || strings.HasPrefix(version, "0.") {
		return true
	}

	return false
}

// GetLastCheckTime returns when the cache was last updated
func (u *Updater) GetLastCheckTime() time.Time {
	return u.cache.GetLastCheckTime()
}

// SetCacheTTL updates the cache TTL
func (u *Updater) SetCacheTTL(ttl time.Duration) {
	u.cache.SetTTL(ttl)
}
