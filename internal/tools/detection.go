// Package tools provides utilities and operations for managing Go tools across versions.
package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/utils"
)

// VersionManager provides access to Go version management operations.
// This interface prevents circular dependencies with internal/manager.
type VersionManager interface {
	// ListInstalledVersions returns all installed Go versions.
	ListInstalledVersions() ([]string, error)
}

// ListForVersion lists all tools installed for a specific Go version.
// Returns Tool structs with full metadata including binary path and modification time.
// If extractMetadata is true, it will also extract package path and version info
// using 'go version -m' (which is slower but provides complete information).
func ListForVersion(cfg *config.Config, version string) ([]ToolMetadata, error) {
	return listForVersionWithOptions(cfg, version, true)
}

// ListForVersionBasic lists tools without extracting package metadata (faster).
func ListForVersionBasic(cfg *config.Config, version string) ([]ToolMetadata, error) {
	return listForVersionWithOptions(cfg, version, false)
}

// listForVersionWithOptions is the internal implementation with metadata extraction control.
func listForVersionWithOptions(cfg *config.Config, version string, extractMetadata bool) ([]ToolMetadata, error) {
	binPath := filepath.Join(cfg.Root, "versions", version, "gopath", "bin")

	entries, err := os.ReadDir(binPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.FailedTo("read tools directory", err)
	}

	var tools []ToolMetadata
	seen := make(map[string]bool) // Deduplicate across platform variants

	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		name := entry.Name()

		// Remove platform-specific extensions for deduplication
		baseName := name
		for _, ext := range utils.WindowsExecutableExtensions() {
			baseName = strings.TrimSuffix(baseName, ext)
		}
		baseName = strings.TrimSuffix(baseName, ".darwin")

		// Skip if we've already seen this tool (e.g., both "tool" and "tool.exe")
		if seen[baseName] {
			continue
		}
		seen[baseName] = true

		// Get file info for metadata
		fullPath := filepath.Join(binPath, name)
		modTime := utils.GetFileModTime(fullPath)
		if modTime.IsZero() {
			continue // Skip files we can't stat
		}

		tool := ToolMetadata{
			Name:       baseName,
			BinaryPath: fullPath,
			GoVersion:  version,
			ModTime:    modTime,
		}

		// Optionally extract package path and version
		if extractMetadata {
			packagePath, toolVersion, err := ExtractToolInfo(fullPath)
			if err == nil {
				tool.PackagePath = packagePath
				tool.Version = toolVersion
			}
		}

		tools = append(tools, tool)
	}

	return tools, nil
}

// ListAll lists all tools across all installed Go versions.
// Returns a map of version -> tools and handles errors gracefully.
func ListAll(cfg *config.Config, mgr VersionManager) (map[string][]ToolMetadata, error) {
	versions, err := mgr.ListInstalledVersions()
	if err != nil {
		return nil, errors.FailedTo("list installed versions", err)
	}

	result := make(map[string][]ToolMetadata)

	for _, version := range versions {
		tools, err := ListForVersion(cfg, version)
		if err != nil {
			// Log error but continue with other versions
			continue
		}
		if len(tools) > 0 {
			result[version] = tools
		}
	}

	return result, nil
}

// IsInstalled checks if a specific tool is installed for a given Go version.
func IsInstalled(cfg *config.Config, version, toolName string) bool {
	binPath := filepath.Join(cfg.Root, "versions", version, "gopath", "bin")

	// Check for the tool with various possible extensions
	candidates := []string{
		filepath.Join(binPath, toolName),
		filepath.Join(binPath, toolName+".exe"),
		filepath.Join(binPath, toolName+".darwin"),
	}

	for _, candidate := range candidates {
		if utils.FileExists(candidate) {
			return true
		}
	}

	return false
}

// GetToolInfo retrieves detailed information about a specific tool installation.
// Returns nil if the tool is not found.
func GetToolInfo(cfg *config.Config, version, toolName string) (*ToolMetadata, error) {
	tools, err := ListForVersion(cfg, version)
	if err != nil {
		return nil, err
	}

	for i := range tools {
		if tools[i].Name == toolName {
			return &tools[i], nil
		}
	}

	return nil, nil // Not found, but not an error
}

// CollectUniqueTools returns a deduplicated list of all unique tool names
// across all installed Go versions.
func CollectUniqueTools(cfg *config.Config, mgr VersionManager) ([]string, error) {
	allTools, err := ListAll(cfg, mgr)
	if err != nil {
		return nil, err
	}

	uniqueNames := make(map[string]bool)
	for _, tools := range allTools {
		for _, tool := range tools {
			uniqueNames[tool.Name] = true
		}
	}

	// Convert to sorted slice
	names := make([]string, 0, len(uniqueNames))
	for name := range uniqueNames {
		names = append(names, name)
	}

	return names, nil
}

// ExtractToolInfo extracts package path and version from a Go binary using 'go version -m'.
// Returns the package path and version, or an error if the information cannot be extracted.
func ExtractToolInfo(binaryPath string) (packagePath string, version string, err error) {
	output, err := utils.RunCommandOutput("go", "version", "-m", binaryPath)
	if err != nil {
		return "", "", errors.FailedTo("run go version -m", err)
	}

	// Parse output
	lines := utils.SplitLines(output)
	for _, line := range lines {

		// Look for "path" line (main module path)
		if strings.HasPrefix(line, "path") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				packagePath = parts[1]
			}
		}

		// Look for "mod" line (main module version)
		if strings.HasPrefix(line, "mod") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				version = parts[2]
			}
		}
	}

	if packagePath == "" {
		return "", "", fmt.Errorf("could not extract package path")
	}

	// If version not found, mark as unknown
	if version == "" {
		version = "unknown"
	}

	return packagePath, version, nil
}

// IsGoTool checks if a binary is a Go tool (vs system binary) by attempting
// to extract tool information from it.
func IsGoTool(binaryPath string) bool {
	packagePath, _, err := ExtractToolInfo(binaryPath)
	if err != nil {
		return false
	}
	return packagePath != ""
}

// FilterGoTools filters a list of binary paths to only include Go tools.
func FilterGoTools(binaries []string) []string {
	var goTools []string
	for _, binary := range binaries {
		if IsGoTool(binary) {
			goTools = append(goTools, binary)
		}
	}
	return goTools
}
