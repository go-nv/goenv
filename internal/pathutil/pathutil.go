package pathutil

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/go-nv/goenv/internal/utils"
)

// ExpandPath expands environment variables and a leading tilde in a path.
// Handles: $HOME, ${HOME} (all platforms); %USERPROFILE% / %VAR% and ~\ (Windows);
// and ~/ , ~ (all platforms). Go's standard library does not expand these shell
// metacharacters, and a value that reaches `go` with a literal one (e.g. GOPATH)
// is rejected.
func ExpandPath(path string) string {
	if path == "" {
		return path
	}

	// Expand Windows %VAR% first (os.ExpandEnv only understands $VAR/${VAR}); gated
	// to Windows so a literal '%' in a POSIX path is left untouched. Then expand
	// the POSIX-style variables.
	if runtime.GOOS == "windows" {
		path = expandWindowsEnv(path)
	}
	path = os.ExpandEnv(path)

	// Expand a leading tilde, accepting either separator so "~\go" works on Windows
	// in addition to "~/go".
	if path == "~" {
		if homeDir, err := os.UserHomeDir(); err == nil {
			return homeDir
		}
		return path
	}
	if strings.HasPrefix(path, "~/") || (runtime.GOOS == "windows" && strings.HasPrefix(path, `~\`)) {
		if homeDir, err := os.UserHomeDir(); err == nil {
			return filepath.Join(homeDir, path[2:])
		}
	}

	return path
}

// expandWindowsEnv replaces %NAME% references with their environment values,
// leaving unknown names untouched. os.ExpandEnv does not understand this form.
func expandWindowsEnv(path string) string {
	return winEnvVarPattern.ReplaceAllStringFunc(path, func(match string) string {
		name := match[1 : len(match)-1]
		if val, ok := os.LookupEnv(name); ok {
			return val
		}
		return match
	})
}

var winEnvVarPattern = regexp.MustCompile(`%([A-Za-z_][A-Za-z0-9_()]*)%`)

// FindExecutable finds an executable file, handling Windows executable extensions.
// On Windows, it checks for .exe, .bat, .cmd, and .com files.
// On Unix, it returns the path as-is.
// Returns the full path if the executable exists, or an error if not found.
func FindExecutable(basePath string) (string, error) {
	if !utils.IsWindows() {
		// On Unix, just check if the file exists
		if !utils.PathExists(basePath) {
			return "", os.ErrNotExist
		}
		return basePath, nil
	}

	// On Windows, try common executable extensions from utils
	for _, ext := range utils.WindowsExecutableExtensions() {
		extPath := basePath + ext
		if utils.PathExists(extPath) {
			return extPath, nil
		}
	}

	// None found, return error for .exe (expected in production)
	return "", os.ErrNotExist
}

// ResolveBinary searches for a binary in a list of directories in order.
// Returns the full path to the binary if found, or an error if not found.
// This is a low-level helper - higher level logic should be in resolver package.
func ResolveBinary(command string, dirs []string) (string, error) {
	for _, dir := range dirs {
		if path, err := utils.FindExecutable(dir, command); err == nil {
			return path, nil
		}
	}
	return "", os.ErrNotExist
}

// FilterFromPath removes all occurrences of a directory from a PATH-style string.
func FilterFromPath(pathEnv, dirToRemove string) string {
	cleanDir := filepath.Clean(dirToRemove)
	parts := strings.Split(pathEnv, string(os.PathListSeparator))
	var filtered []string
	for _, p := range parts {
		if filepath.Clean(p) != cleanDir {
			filtered = append(filtered, p)
		}
	}
	return strings.Join(filtered, string(os.PathListSeparator))
}
