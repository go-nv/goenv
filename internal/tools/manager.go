// Package tools provides utilities and operations for managing Go tools across versions.
package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/utils"
)

// Manager handles tool operations across Go versions.
type Manager struct {
	cfg        *config.Config
	versionMgr VersionManager
}

// NewManager creates a new tools Manager.
func NewManager(cfg *config.Config, versionMgr VersionManager) *Manager {
	return &Manager{
		cfg:        cfg,
		versionMgr: versionMgr,
	}
}

// InstallOptions configures tool installation behavior.
type InstallOptions struct {
	// Packages is the list of package paths to install (e.g., "golang.org/x/tools/cmd/goimports@latest").
	Packages []string

	// Versions is the list of Go versions to install tools for.
	// If empty, installs for current version only.
	Versions []string

	// DryRun previews the installation without executing it.
	DryRun bool

	// Verbose enables detailed output during installation.
	Verbose bool
}

// Install installs Go tools for specified versions.
// Uses 'go install' with proper GOPATH/GOROOT isolation.
func (m *Manager) Install(opts InstallOptions) (*InstallResult, error) {
	if len(opts.Packages) == 0 {
		return nil, fmt.Errorf("no packages specified")
	}

	// Normalize package paths
	packages := NormalizePackagePaths(opts.Packages)

	// Determine target versions
	versions := opts.Versions
	if len(versions) == 0 {
		// Use current version
		current, err := m.versionMgr.ListInstalledVersions()
		if err != nil {
			return nil, errors.FailedTo("get installed versions", err)
		}
		if len(current) == 0 {
			return nil, errors.NoVersionsInstalled()
		}
		versions = current[:1] // Just use first one for now
	}

	result := &InstallResult{
		Installed: []string{},
		Failed:    []string{},
		Errors:    []error{},
	}

	// Preview mode - don't actually install
	if opts.DryRun {
		for _, version := range versions {
			for _, pkg := range packages {
				result.Installed = append(result.Installed, fmt.Sprintf("%s@%s", ExtractToolName(pkg), version))
			}
		}
		return result, nil
	}

	// Execute installations
	var installedTools = make(map[string]bool)
	var newTools []Tool

	for _, version := range versions {
		for _, pkg := range packages {
			toolName := ExtractToolName(pkg)

			if tool, err := m.InstallSingleTool(version, pkg, opts.Verbose); err != nil {
				result.Failed = append(result.Failed, fmt.Sprintf("%s@%s", toolName, version))
				result.Errors = append(result.Errors, fmt.Errorf("%s (using Go %s): %w", pkg, version, err))
			} else {
				result.Installed = append(result.Installed, fmt.Sprintf("%s@%s", toolName, version))
				if !installedTools[toolName] {
					installedTools[toolName] = true
					newTools = append(newTools, tool)
				}
			}
		}
	}

	if len(newTools) > 0 {
		cfgPath := ConfigPath(m.cfg.Root)

		// Load existing config
		cfg, err := LoadConfig(cfgPath)
		if err != nil {
			return nil, errors.FailedTo("load config", err)
		}

		for _, tool := range newTools {
			// Check if tool already exists in config and update it

			idx := utils.FindIndex(cfg.Tools, func(t Tool) bool {
				return t.Name == tool.Name
			})

			if idx >= 0 {
				// Update existing tool entry
				cfg.Tools[idx] = tool
			} else {
				// Add new tool entry
				cfg.Tools = append(cfg.Tools, tool)
			}
		}

		SaveConfig(cfgPath, cfg)
	}

	return result, nil
}

// InstallSingleTool installs a single tool package for a specific Go version.
// This is useful for commands that need per-tool progress feedback.
// For batch installation, use Install() instead.
func (m *Manager) InstallSingleTool(goVersion, packagePath string, verbose bool) (Tool, error) {
	// Normalize package path to ensure it has a version
	packagePath = NormalizePackagePath(packagePath)

	// Split package path from version for the Tool struct
	// packagePath format: "golang.org/x/tools/gopls@v0.20.0"
	parts := strings.Split(packagePath, "@")
	pkg := parts[0]
	ver := "@latest" // Default to latest if not specified
	if len(parts) > 1 {
		ver = "@" + parts[1]
	}

	cfgPath := ConfigPath(m.cfg.Root)

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		return Tool{}, errors.FailedTo("load config", err)
	}

	newTool := Tool{
		Name:           ExtractToolName(packagePath),
		Package:        pkg,
		Version:        ver,
		UpdateStrategy: "auto", // Use stable versions
	}

	// Create a temporary config with a single tool to reuse the gold standard installation logic
	cfg.Tools = []Tool{newTool}

	// Use the gold standard InstallTools function with host-specific GOPATH
	return newTool, InstallTools(cfg, goVersion, m.cfg.Root, m.cfg.VersionDir(goVersion), verbose)
}

// UninstallOptions configures tool uninstallation behavior.
type UninstallOptions struct {
	// ToolNames is the list of tool names to uninstall.
	ToolNames []string

	// Versions is the list of Go versions to uninstall tools from.
	// If empty, uninstalls from current version only.
	Versions []string

	// DryRun previews the uninstallation without executing it.
	DryRun bool

	// Force skips confirmation prompts.
	Force bool
}

// Uninstall removes Go tools from specified versions.
func (m *Manager) Uninstall(opts UninstallOptions) (*UninstallResult, error) {
	if len(opts.ToolNames) == 0 {
		return nil, fmt.Errorf("no tools specified")
	}

	// Determine target versions
	versions := opts.Versions
	if len(versions) == 0 {
		// Use current version
		current, err := m.versionMgr.ListInstalledVersions()
		if err != nil {
			return nil, errors.FailedTo("get installed versions", err)
		}
		if len(current) == 0 {
			return nil, errors.NoVersionsInstalled()
		}
		versions = current[:1]
	}

	result := &UninstallResult{
		Removed: []string{},
		Failed:  []string{},
		Errors:  []error{},
	}

	// Preview mode
	if opts.DryRun {
		for _, version := range versions {
			for _, toolName := range opts.ToolNames {
				if IsInstalled(m.cfg, version, toolName) {
					result.Removed = append(result.Removed, fmt.Sprintf("%s@%s", toolName, version))
				}
			}
		}
		return result, nil
	}

	// Execute uninstalls
	for _, version := range versions {
		for _, toolName := range opts.ToolNames {
			if err := m.UninstallSingleTool(version, toolName); err != nil {
				result.Failed = append(result.Failed, fmt.Sprintf("%s@%s", toolName, version))
				result.Errors = append(result.Errors, fmt.Errorf("%s@%s: %w", toolName, version, err))
			} else {
				result.Removed = append(result.Removed, fmt.Sprintf("%s@%s", toolName, version))
			}
		}
	}

	return result, nil
}

// UninstallSingleTool removes a tool binary from a specific Go version.
// This is useful for commands that need per-tool progress feedback.
// For batch uninstallation, use Uninstall() instead.
func (m *Manager) UninstallSingleTool(version, toolName string) error {
	binPath := filepath.Join(m.cfg.Root, "versions", version, "gopath", "bin")

	// Find and remove all platform variants
	candidates := []string{
		filepath.Join(binPath, toolName),
		filepath.Join(binPath, toolName+".exe"),
		filepath.Join(binPath, toolName+".darwin"),
	}

	found := false
	for _, candidate := range candidates {
		if utils.PathExists(candidate) {
			if err := os.Remove(candidate); err != nil {
				return fmt.Errorf("failed to remove %s: %w", candidate, err)
			}
			found = true
		}
	}

	if !found {
		return fmt.Errorf("tool not found")
	}

	return nil
}

// CheckToolUpdates checks if tools have newer versions available.
// It populates the LatestVersion and IsOutdated fields for tools with PackagePath.
// Returns a new slice with updated Tool structs.
func (m *Manager) CheckToolUpdates(tools []ToolMetadata) []ToolMetadata {
	result := make([]ToolMetadata, 0, len(tools))

	for _, tool := range tools {
		// Skip tools without package path
		if tool.PackagePath == "" {
			result = append(result, tool)
			continue
		}

		// Query latest version
		latestVersion, err := GetLatestVersion(tool.PackagePath)
		if err != nil {
			// Can't check, keep as-is
			result = append(result, tool)
			continue
		}

		// Update tool with version info
		tool.LatestVersion = latestVersion
		tool.IsOutdated = CompareVersions(tool.Version, latestVersion) < 0
		result = append(result, tool)
	}

	return result
}

// GetStatus returns comprehensive tool installation status across all versions.
func (m *Manager) GetStatus() (*ToolStatus, error) {
	allTools, err := ListAll(m.cfg, m.versionMgr)
	if err != nil {
		return nil, errors.FailedTo("list tools", err)
	}

	status := &ToolStatus{
		ByVersion:   allTools,
		AllTools:    []ToolMetadata{},
		Outdated:    []ToolMetadata{},
		TotalCount:  0,
		UniqueTools: 0,
	}

	// Flatten all tools and check for updates
	uniqueNames := make(map[string]bool)
	for version, tools := range allTools {
		// Check updates for this version's tools
		updatedTools := m.CheckToolUpdates(tools)
		status.ByVersion[version] = updatedTools

		for _, tool := range updatedTools {
			status.AllTools = append(status.AllTools, tool)
			uniqueNames[tool.Name] = true
			status.TotalCount++

			if tool.IsOutdated {
				status.Outdated = append(status.Outdated, tool)
			}
		}
	}

	status.UniqueTools = len(uniqueNames)

	return status, nil
}

// SyncOptions configures tool synchronization behavior.
type SyncOptions struct {
	// SourceVersion is the Go version to copy tools from.
	SourceVersion string

	// TargetVersion is the Go version to copy tools to.
	TargetVersion string

	// ToolNames optionally filters which tools to sync.
	// If empty, syncs all tools.
	ToolNames []string

	// DryRun previews the sync without executing it.
	DryRun bool

	// Verbose enables detailed output during sync.
	Verbose bool
}

// Sync copies tools from one Go version to another.
// Reinstalls tools from source using 'go install'.
func (m *Manager) Sync(opts SyncOptions) (*InstallResult, error) {
	if opts.SourceVersion == "" || opts.TargetVersion == "" {
		return nil, fmt.Errorf("source and target versions required")
	}

	if opts.SourceVersion == opts.TargetVersion {
		return nil, fmt.Errorf("source and target versions are the same")
	}

	// List tools in source version
	sourceTools, err := ListForVersion(m.cfg, opts.SourceVersion)
	if err != nil {
		return nil, errors.FailedTo("list source tools", err)
	}

	if len(sourceTools) == 0 {
		return nil, fmt.Errorf("no tools found in source version %s", opts.SourceVersion)
	}

	// Filter tools if specific names requested
	var toolsToSync []ToolMetadata
	if len(opts.ToolNames) > 0 {
		for _, tool := range sourceTools {
			if slices.Contains(opts.ToolNames, tool.Name) {
				toolsToSync = append(toolsToSync, tool)
			}
		}
	} else {
		toolsToSync = sourceTools
	}

	if len(toolsToSync) == 0 {
		return nil, fmt.Errorf("no matching tools to sync")
	}

	// Extract package paths
	// Note: We don't have package paths in Tool struct from binary inspection
	// For now, assume @latest. Commands can use tooldetect package for metadata.
	packages := make([]string, len(toolsToSync))
	for i, tool := range toolsToSync {
		if tool.PackagePath != "" {
			packages[i] = tool.PackagePath
		} else {
			// Fallback: just the tool name + @latest
			packages[i] = tool.Name + "@latest"
		}
	}

	// Install to target version
	installOpts := InstallOptions{
		Packages: packages,
		Versions: []string{opts.TargetVersion},
		DryRun:   opts.DryRun,
		Verbose:  opts.Verbose,
	}

	return m.Install(installOpts)
}

// ListTools returns all tools installed for a specific version.
// This is a convenience wrapper around detection.ListForVersion.
func (m *Manager) ListTools(version string) ([]ToolMetadata, error) {
	return ListForVersion(m.cfg, version)
}

// ListAllTools returns all tools across all installed Go versions.
// This is a convenience wrapper around detection.ListAll.
func (m *Manager) ListAllTools() (map[string][]ToolMetadata, error) {
	return ListAll(m.cfg, m.versionMgr)
}

// IsToolInstalled checks if a tool is installed for a specific version.
// This is a convenience wrapper around detection.IsInstalled.
func (m *Manager) IsToolInstalled(version, toolName string) bool {
	return IsInstalled(m.cfg, version, toolName)
}
