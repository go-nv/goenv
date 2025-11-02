// Package tools provides utilities and operations for managing Go tools across versions.
package tools

import (
	"strings"
)

// ExtractToolName extracts the binary name from a package path.
//
// Examples:
//
//	ExtractToolName("golang.org/x/tools/cmd/goimports@latest")  // "goimports"
//	ExtractToolName("github.com/go-delve/delve/cmd/dlv@v1.20.1") // "dlv"
//	ExtractToolName("golang.org/x/tools/cmd/goimports")          // "goimports"
func ExtractToolName(packagePath string) string {
	// Remove @version suffix
	if idx := strings.Index(packagePath, "@"); idx != -1 {
		packagePath = packagePath[:idx]
	}

	// Get last component
	parts := strings.Split(packagePath, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return packagePath
}

// ExtractToolNames extracts tool names from multiple package paths.
func ExtractToolNames(packages []string) []string {
	names := make([]string, 0, len(packages))
	for _, pkg := range packages {
		names = append(names, ExtractToolName(pkg))
	}
	return names
}

// NormalizePackagePath ensures a package path has a version specifier.
// Adds @latest if no version is specified.
//
// Examples:
//
//	NormalizePackagePath("golang.org/x/tools/cmd/goimports")        // "golang.org/x/tools/cmd/goimports@latest"
//	NormalizePackagePath("golang.org/x/tools/cmd/goimports@latest") // "golang.org/x/tools/cmd/goimports@latest"
func NormalizePackagePath(path string) string {
	if !strings.Contains(path, "@") {
		return path + "@latest"
	}
	return path
}

// NormalizePackagePaths normalizes multiple package paths.
func NormalizePackagePaths(paths []string) []string {
	normalized := make([]string, 0, len(paths))
	for _, path := range paths {
		normalized = append(normalized, NormalizePackagePath(path))
	}
	return normalized
}
