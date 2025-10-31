package tools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/utils"
)

// normalizePackagePaths ensures all package paths have a version specifier
func normalizePackagePaths(paths []string) []string {
	var normalized []string
	for _, path := range paths {
		// Add @latest if no version specified
		if !strings.Contains(path, "@") {
			path += "@latest"
		}
		normalized = append(normalized, path)
	}
	return normalized
}

// getInstalledVersions returns a list of all installed Go versions
func getInstalledVersions(cfg *config.Config) ([]string, error) {
	versionsDir := filepath.Join(cfg.Root, "versions")
	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read versions directory: %w", err)
	}

	var versions []string
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != "system" {
			versions = append(versions, entry.Name())
		}
	}

	return versions, nil
}

// extractToolNames extracts tool names from package paths
func extractToolNames(packages []string) []string {
	var names []string
	for _, pkg := range packages {
		names = append(names, extractToolName(pkg))
	}
	return names
}

// installToolForVersion installs a tool for a specific Go version
func installToolForVersion(cfg *config.Config, version, packagePath string, verbose bool) error {
	versionPath := filepath.Join(cfg.Root, "versions", version)
	goRoot := filepath.Join(versionPath, "go")
	goBin := filepath.Join(goRoot, "bin", "go")
	gopath := filepath.Join(versionPath, "gopath")

	// Check if Go binary exists
	if _, err := os.Stat(goBin); os.IsNotExist(err) {
		return fmt.Errorf("go binary not found for version %s", version)
	}

	// Run go install
	cmd := exec.Command(goBin, "install", packagePath)
	cmd.Env = append(os.Environ(),
		"GOROOT="+goRoot,
		"GOPATH="+gopath,
	)

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	return cmd.Run()
}

// getToolsForVersion lists tools installed for a specific version
func getToolsForVersion(cfg *config.Config, version string) ([]string, error) {
	binPath := filepath.Join(cfg.Root, "versions", version, "gopath", "bin")

	entries, err := os.ReadDir(binPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var tools []string
	for _, entry := range entries {
		if !entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			// Skip platform-specific variants (e.g., tool.exe when we already have tool)
			name := entry.Name()
			// Remove extensions for deduplication
			baseName := name
			for _, ext := range utils.WindowsExecutableExtensions() {
				baseName = strings.TrimSuffix(baseName, ext)
			}
			baseName = strings.TrimSuffix(baseName, ".darwin")

			// Only add the base name
			if baseName == name || !contains(tools, baseName) {
				tools = append(tools, baseName)
			}
		}
	}

	return tools, nil
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
