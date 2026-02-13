package manager

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/go-nv/goenv/internal/config"
)

// ProjectInfo represents a discovered Go project
type ProjectInfo struct {
	Path    string // Absolute path to project directory
	Version string // Go version required
	Source  string // config.VersionFileName or config.GoModFileName
}

// Directories to skip during scanning for performance
var skipDirs = map[string]bool{
	"node_modules": true,
	".git":         true,
	"vendor":       true,
	".svn":         true,
	".hg":          true,
	"__pycache__":  true,
	"venv":         true,
	".env":         true,
	"dist":         true,
	"build":        true,
	"target":       true,
	".idea":        true,
	".vscode":      true,
	".next":        true,
	".cache":       true,
}

// ScanProjects finds all Go projects in a directory tree.
// It looks for .go-version and go.mod files up to the specified depth.
//
// Parameters:
//   - rootDir: The directory to start scanning from
//   - maxDepth: Maximum directory depth to scan (0 = unlimited)
//
// Returns a slice of ProjectInfo for each discovered project.
// Returns an empty slice (not nil) even if scanning fails.
func ScanProjects(rootDir string, maxDepth int) ([]ProjectInfo, error) {
	projects := make([]ProjectInfo, 0)

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip errors and continue scanning
			return nil
		}

		// Handle directories
		if info.IsDir() {
			// Skip hidden directories (except root)
			if path != rootDir && strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}

			// Skip common non-project directories
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}

			// Check depth limit
			if maxDepth > 0 {
				relPath, err := filepath.Rel(rootDir, path)
				if err == nil {
					depth := strings.Count(relPath, string(os.PathSeparator))
					if depth >= maxDepth {
						return filepath.SkipDir
					}
				}
			}

			return nil
		}

		// Check for .go-version file
		if info.Name() == config.VersionFileName {
			version, err := readVersionFile(path)
			if err == nil && version != "" {
				projects = append(projects, ProjectInfo{
					Path:    filepath.Dir(path),
					Version: version,
					Source:  config.VersionFileName,
				})
			}
			return nil
		}

		// Check for go.mod file
		if info.Name() == config.GoModFileName {
			version, err := ParseGoModVersion(path)
			if err == nil && version != "" {
				projects = append(projects, ProjectInfo{
					Path:    filepath.Dir(path),
					Version: version,
					Source:  config.GoModFileName,
				})
			}
			return nil
		}

		return nil
	})

	return projects, err
}

// readVersionFile reads a .go-version file and returns the version string.
// It returns the first non-empty, non-comment line.
func readVersionFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines and comments
		if line != "" && !strings.HasPrefix(line, "#") {
			return line, nil
		}
	}

	return "", nil
}
