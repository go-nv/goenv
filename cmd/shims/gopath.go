package shims

import (
	"path/filepath"
	"strings"

	"github.com/go-nv/goenv/internal/pathutil"
)

// normalizeGopathEntries expands and validates the entries of an inherited
// GOPATH so goenv never re-exports (or hands `go`) a value it will reject.
//
// goenv carries the user's own GOPATH through to `go` on every platform, so a
// value the shell left unnormalized — most commonly a quoted GOPATH="~/go" that
// stores a literal '~' (Go: "GOPATH entry cannot start with shell metacharacter
// '~'") — breaks every build. For each entry this expands shell sugar the shell
// did not (~/, ~, $HOME, ${HOME}, %USERPROFILE%) and drops empty entries and any
// that are still not absolute (Go requires absolute GOPATH entries).
//
// It does NOT drop goenv-managed paths: it is safe to apply unconditionally,
// including for the system version or when GOENV_DISABLE_GOPATH is set, where a
// user's own "$GOPATH_PREFIX/<version>" entry must be preserved. Callers that
// actually prepend a managed path use sanitizeInheritedGopath to de-duplicate.
func normalizeGopathEntries(raw string) []string {
	if raw == "" {
		return nil
	}
	var out []string
	for _, entry := range filepath.SplitList(raw) {
		p := pathutil.ExpandPath(entry)
		if p == "" || !filepath.IsAbs(p) {
			continue
		}
		out = append(out, p)
	}
	return out
}

// sanitizeInheritedGopath is normalizeGopathEntries plus de-duplication of
// goenv-managed per-version paths ($GOPATH_PREFIX/<version>), so that re-entering
// a managed shell does not accumulate duplicates. Use it ONLY when a managed
// GOPATH is actually being prepended; goPathPattern is cfg.GopathPrefix() (e.g.
// $HOME/go). $GOPATH_PREFIX itself and any custom absolute paths are kept.
func sanitizeInheritedGopath(raw, goPathPattern string) []string {
	managedPrefix := goPathPattern + string(filepath.Separator)
	var out []string
	for _, p := range normalizeGopathEntries(raw) {
		if strings.HasPrefix(p, managedPrefix) && p != goPathPattern {
			continue
		}
		out = append(out, p)
	}
	return out
}
