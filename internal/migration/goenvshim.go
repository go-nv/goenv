package migration

import (
	"os"
	"path/filepath"
	"runtime"
)

// goenvShimNames are the file names a goenv shim can have inside the shims
// directory. Windows shims are generated as ".bat" files.
func goenvShimNames() []string {
	if runtime.GOOS == "windows" {
		return []string{"goenv.bat", "goenv.cmd", "goenv.exe", "goenv"}
	}
	return []string{"goenv"}
}

// FindGoenvShims returns the paths of any goenv shim found in shimsDir.
//
// rehash never creates a shim for goenv itself: the shims directory is at the
// front of PATH and every shim dispatches through "goenv exec", so a goenv
// shim makes goenv invoke itself until the process table fills up (issue #542).
// Anything matching here is therefore leftover state, not something goenv
// created on purpose.
//
// Only files that actually look like a shim are reported, so an unrelated file
// a user happens to have placed there is left alone.
func FindGoenvShims(shimsDir string) []string {
	var found []string

	for _, name := range goenvShimNames() {
		path := filepath.Join(shimsDir, name)

		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if !isGoenvShim(data) {
			continue
		}
		found = append(found, path)
	}

	return found
}

// isGoenvShim reports whether data looks like a goenv-generated shim, from
// either v2 (which baked in a libexec/goenv path) or v3 (which dispatches
// through "goenv exec").
func isGoenvShim(data []byte) bool {
	if V2ShimPattern.Match(data) {
		return true
	}
	return v3ShimPattern.Match(data)
}

// RemoveGoenvShims deletes every goenv shim found in shimsDir and returns the
// paths that were removed.
func RemoveGoenvShims(shimsDir string) ([]string, error) {
	var removed []string

	for _, path := range FindGoenvShims(shimsDir) {
		if err := os.Remove(path); err != nil {
			return removed, err
		}
		removed = append(removed, path)
	}

	return removed, nil
}
