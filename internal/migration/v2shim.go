package migration

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// V2ShimPattern matches v2's libexec/goenv path with both forward slashes and backslashes.
// The pattern ensures we match "libexec/goenv" or "libexec\goenv" where "goenv" is the final
// path component (followed by quote, space, or directory separator, but not a hyphen or letter).
var V2ShimPattern = regexp.MustCompile(`libexec[\\/]goenv(["'\s/\\]|$)`)

// v3ShimPattern matches the marker line that goenv writes into every shim it
// generates (see internal/shims.ShimMarker). A shim named "goenv" containing
// this is just as recursive as a v2 leftover, so it must be recognised too.
//
// This deliberately does NOT match on "goenv exec". The compiled goenv binary
// embeds the shim templates and therefore contains that byte sequence itself,
// so a substring test would classify the goenv binary as a shim and delete it.
// Detection for a destructive operation must positively identify the target,
// never merely fail to rule it out.
//
// The optional \r keeps this working against shims stored with CRLF endings,
// which is the normal state of a checked-out or edited file on Windows.
var v3ShimPattern = regexp.MustCompile(`(?m)^(#|REM) goenv-shim v1\r?$`)

// RemoveStaleV2Shim removes the stale goenv shim left over from v2 installations.
// v2's goenv-rehash bakes the Homebrew Cellar path into shims at creation time
// (e.g. exec "/opt/homebrew/Cellar/goenv/2.2.38_1/libexec/goenv" on macOS/Linux
// or "C:\path\to\libexec\goenv" on Windows). After upgrading to v3, the old shim
// may still point to a deleted path, shadowing the real v3 binary.
//
// We only remove the shim if it contains "libexec/goenv" or "libexec\goenv" —
// the v2 fingerprint — to avoid deleting anything unexpected.
//
// Returns true if the shim was removed, false otherwise.
// If removal fails, it returns an error that can be logged as a warning.
func RemoveStaleV2Shim(shimsDir string) (bool, error) {
	goenvShim := filepath.Join(shimsDir, "goenv")

	// Read the shim file
	data, err := os.ReadFile(goenvShim)
	if err != nil {
		// File doesn't exist or can't be read - nothing to remove
		return false, nil
	}

	// Check if it contains the v2 fingerprint (supports both / and \)
	if !V2ShimPattern.Match(data) {
		// Not a v2 shim - leave it alone
		return false, nil
	}

	// Remove the stale shim
	if err := os.Remove(goenvShim); err != nil {
		return false, fmt.Errorf("failed to remove stale v2 goenv shim %q: %w", goenvShim, err)
	}

	return true, nil
}
