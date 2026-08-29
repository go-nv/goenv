package migration

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/go-nv/goenv/internal/shims"
)

// goenvShimNames are the file names a goenv shim can have inside the shims
// directory, for the current platform.
func goenvShimNames() []string {
	return goenvShimNamesFor(runtime.GOOS)
}

// goenvShimNamesFor returns the candidate shim names for a given GOOS.
//
// Taking the OS as a parameter rather than reading runtime.GOOS directly means
// the Windows naming rules are exercised on every CI runner, not only the
// Windows one — platform-specific logic that can only be tested on that
// platform is logic that effectively is not tested.
func goenvShimNamesFor(goos string) []string {
	if goos == "windows" {
		// rehash generates the shims.WindowsShimExtension form; the others cover
		// shims left by other tools or earlier versions. "goenv.exe" is included
		// so a stale copy is detected, but content checks still have to confirm
		// it is a script — see FindGoenvShims.
		return []string{"goenv" + shims.WindowsShimExtension, "goenv.cmd", "goenv.exe", "goenv"}
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
// Identification is positive and structural: a file must carry a goenv shim
// marker (or the v2 fingerprint) to be reported. A compiled goenv binary placed
// or symlinked into the shims directory must never be mistaken for a shim,
// because callers delete what this returns.
func FindGoenvShims(shimsDir string) []string {
	var found []string

	for _, name := range goenvShimNames() {
		path := filepath.Join(shimsDir, name)

		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() {
			// Missing, a directory, or a symlink — never a shim we generated.
			continue
		}

		// Shims are small text scripts. Reading a bounded prefix keeps a large
		// binary from being slurped into memory just to be rejected.
		data, err := readPrefix(path, shimScanLimit)
		if err != nil {
			continue
		}
		if isBinaryContent(data) {
			continue
		}
		if !isGoenvShim(data) {
			continue
		}
		found = append(found, path)
	}

	return found
}

// shimScanLimit bounds how much of a candidate file is inspected. Generated
// shims are well under 2 KiB.
const shimScanLimit = 8 << 10

// readPrefix reads at most limit bytes from path.
func readPrefix(path string, limit int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return io.ReadAll(io.LimitReader(file, limit))
}

// isBinaryContent reports whether data looks like a compiled artifact rather
// than a shell script. A NUL byte does not occur in the shims goenv generates.
func isBinaryContent(data []byte) bool {
	return bytes.IndexByte(data, 0) >= 0
}

// isGoenvShim reports whether data looks like a goenv-generated shim, from
// either v2 (which baked in a libexec/goenv path) or v3 (which carries an
// explicit shim marker).
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
