package tooldetect

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Tool represents a detected Go tool
type Tool struct {
	Name        string // Binary name (e.g., "gopls")
	Path        string // Full path to binary
	PackagePath string // Go package path (e.g., "golang.org/x/tools/gopls")
	Version     string // Current version (e.g., "v0.14.1")
}

// BuildInfo represents parsed output from 'go version -m'
type BuildInfo struct {
	Path    string
	Main    Module
	Deps    []Module
	GoValue string `json:"GoVersion"`
}

// Module represents a Go module
type Module struct {
	Path    string
	Version string
	Sum     string
}

// ListInstalledTools returns all Go tools installed for a specific Go version
func ListInstalledTools(goenvRoot, goVersion string) ([]Tool, error) {
	// Determine GOPATH/bin directory for this version
	versionPath := filepath.Join(goenvRoot, "versions", goVersion)
	gopathBin := filepath.Join(versionPath, "gopath", "bin")

	// Check if directory exists
	if _, err := os.Stat(gopathBin); os.IsNotExist(err) {
		return []Tool{}, nil // No tools installed yet
	}

	// List all binaries
	entries, err := os.ReadDir(gopathBin)
	if err != nil {
		return nil, fmt.Errorf("failed to read GOPATH/bin: %w", err)
	}

	var tools []Tool

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Skip non-executables on Windows
		if runtime.GOOS == "windows" && !strings.HasSuffix(name, ".exe") {
			continue
		}

		// Remove .exe suffix for display name
		displayName := strings.TrimSuffix(name, ".exe")

		binaryPath := filepath.Join(gopathBin, name)

		// Extract package path and version
		packagePath, version, err := ExtractToolInfo(binaryPath)
		if err != nil {
			// If we can't extract info, still include it but with limited data
			tools = append(tools, Tool{
				Name:        displayName,
				Path:        binaryPath,
				PackagePath: "",
				Version:     "unknown",
			})
			continue
		}

		tools = append(tools, Tool{
			Name:        displayName,
			Path:        binaryPath,
			PackagePath: packagePath,
			Version:     version,
		})
	}

	return tools, nil
}

// ExtractToolInfo extracts package path and version from a Go binary using 'go version -m'
func ExtractToolInfo(binaryPath string) (packagePath string, version string, err error) {
	cmd := exec.Command("go", "version", "-m", binaryPath)
	output, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to run go version -m: %w", err)
	}

	// Parse output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

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

// GetLatestVersion queries the Go module proxy for the latest version of a package
func GetLatestVersion(packagePath string) (string, error) {
	// Use go list to get the latest version
	cmd := exec.Command("go", "list", "-m", "-versions", "-json", packagePath+"@latest")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to query latest version: %w", err)
	}

	// Parse JSON output
	var info struct {
		Version string
	}

	if err := json.Unmarshal(output, &info); err != nil {
		return "", fmt.Errorf("failed to parse version info: %w", err)
	}

	return info.Version, nil
}

// CompareVersions compares two version strings (semantic versioning)
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
func CompareVersions(v1, v2 string) int {
	// Remove 'v' prefix if present
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	// Handle special cases
	if v1 == v2 {
		return 0
	}
	if v1 == "unknown" {
		return -1
	}
	if v2 == "unknown" {
		return 1
	}

	// Simple string comparison for now
	// Could be enhanced with proper semver parsing
	if v1 < v2 {
		return -1
	} else if v1 > v2 {
		return 1
	}
	return 0
}

// IsGoTool checks if a binary is a Go tool (vs system binary)
func IsGoTool(binaryPath string) bool {
	// Try to extract tool info
	packagePath, _, err := ExtractToolInfo(binaryPath)
	if err != nil {
		return false
	}

	// Go tools have package paths
	return packagePath != ""
}

// FilterGoTools filters a list of binaries to only include Go tools
func FilterGoTools(binaries []string) []string {
	var goTools []string

	for _, binary := range binaries {
		if IsGoTool(binary) {
			goTools = append(goTools, binary)
		}
	}

	return goTools
}
