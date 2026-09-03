package shims

import (
	"path/filepath"
	"strings"

	"github.com/go-nv/goenv/internal/pathutil"
)

// sanitizeInheritedGopath normalizes the entries of an inherited GOPATH so goenv
// never re-exports (or hands `go`) a value it will reject.
//
// goenv both re-exports GOPATH from its shell integration and sets it for every
// shimmed process, always prepending a per-version path. The user's own GOPATH
// is therefore carried through to `go` on every platform, so a value the shell
// left unnormalized — most commonly a quoted GOPATH="~/go" that stores a literal
// '~' (Go: "GOPATH entry cannot start with shell metacharacter '~'") — breaks
// every build. For each entry this:
//
//   - expands shell sugar the shell did not (~/, ~, $HOME, ${HOME}, %USERPROFILE%),
//   - drops empty entries and any that are still not absolute (Go requires
//     absolute GOPATH entries), and
//   - drops goenv-managed per-version paths ($GOPATH_PREFIX/<version>) so
//     re-entering a managed shell does not duplicate them.
//
// goPathPattern is cfg.GopathPrefix() (e.g. $HOME/go). The returned slice is the
// cleaned, ready-to-join list of the user's own entries.
func sanitizeInheritedGopath(raw, goPathPattern string) []string {
	if raw == "" {
		return nil
	}

	managedPrefix := goPathPattern + string(filepath.Separator)
	var out []string
	for _, entry := range filepath.SplitList(raw) {
		p := pathutil.ExpandPath(entry)
		if p == "" || !filepath.IsAbs(p) {
			continue
		}
		// Skip $GOPATH_PREFIX/<version> paths, but keep $GOPATH_PREFIX itself and
		// any other custom absolute paths.
		if strings.HasPrefix(p, managedPrefix) && p != goPathPattern {
			continue
		}
		out = append(out, p)
	}
	return out
}
