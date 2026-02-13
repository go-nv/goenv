// Package tools provides utilities and operations for managing Go tools across versions.
package tools

import (
	"strings"
)

// commonTools maps common tool names to their full package paths
var commonTools = map[string]string{
	"gotestsum":     "gotest.tools/gotestsum",
	"gopls":         "golang.org/x/tools/gopls",
	"goimports":     "golang.org/x/tools/cmd/goimports",
	"golangci-lint": "github.com/golangci/golangci-lint/cmd/golangci-lint",
	"staticcheck":   "honnef.co/go/tools/cmd/staticcheck",
	"dlv":           "github.com/go-delve/delve/cmd/dlv",
	"gofumpt":       "mvdan.cc/gofumpt",
	"mockgen":       "go.uber.org/mock/mockgen",
	"goreleaser":    "github.com/goreleaser/goreleaser",
	"air":           "github.com/cosmtrek/air",
	"migrate":       "github.com/golang-migrate/migrate/v4/cmd/migrate",
	"swag":          "github.com/swaggo/swag/cmd/swag",
	"wire":          "github.com/google/wire/cmd/wire",
	"protoc-gen-go": "google.golang.org/protobuf/cmd/protoc-gen-go",
	"goose":         "github.com/pressly/goose/v3/cmd/goose",
	"cyclonedx":     "github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod",
	"syft":          "github.com/anchore/syft/cmd/syft",
	// Security scanners for SBOM vulnerability analysis (Phase 4A)
	"grype": "github.com/anchore/grype/cmd/grype",
	"trivy": "github.com/aquasecurity/trivy/cmd/trivy",
	// Commercial security scanners (Phase 4B)
	"snyk": "github.com/snyk/cli/cmd/snyk",
}

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
// Also expands common tool names to their full package paths.
//
// Examples:
//
//	NormalizePackagePath("golang.org/x/tools/cmd/goimports")        // "golang.org/x/tools/cmd/goimports@latest"
//	NormalizePackagePath("golang.org/x/tools/cmd/goimports@latest") // "golang.org/x/tools/cmd/goimports@latest"
//	NormalizePackagePath("gotestsum")                               // "gotest.tools/gotestsum@latest"
//	NormalizePackagePath("gopls@v0.20.0")                           // "golang.org/x/tools/gopls@v0.20.0"
func NormalizePackagePath(path string) string {
	// Extract version if present
	version := ""
	if idx := strings.Index(path, "@"); idx != -1 {
		version = path[idx:]
		path = path[:idx]
	}

	// Check if it's a common tool name
	if fullPath, exists := commonTools[path]; exists {
		path = fullPath
	}

	// Add @latest if no version specified
	if version == "" {
		version = "@latest"
	}

	return path + version
}

// NormalizePackagePaths normalizes multiple package paths.
func NormalizePackagePaths(paths []string) []string {
	normalized := make([]string, 0, len(paths))
	for _, path := range paths {
		normalized = append(normalized, NormalizePackagePath(path))
	}
	return normalized
}
