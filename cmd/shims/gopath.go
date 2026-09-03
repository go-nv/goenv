package shims

import (
	"path/filepath"
	"strings"

	"github.com/go-nv/goenv/internal/pathutil"
)

// normalizeGopathEntries expands the entries of an inherited GOPATH so a value
// the shell left unexpanded — most commonly a quoted GOPATH="~/go" that stores a
// literal '~' (Go: "GOPATH entry cannot start with shell metacharacter '~'") —
// is not handed to `go` verbatim. It expands each entry (~/, ~, $HOME, %VAR% on
// Windows) and drops only empty entries.
//
// It deliberately does NOT drop or otherwise "fix" entries that are still
// invalid after expansion (e.g. a relative path or a mistyped "~foo"): those are
// left in place so `go` reports its own clear error, rather than goenv silently
// substituting a different — and possibly default — GOPATH. It also does not drop
// goenv-managed paths, so it is safe to apply unconditionally, including for the
// system version and GOENV_DISABLE_GOPATH, where a user's own
// "$GOPATH_PREFIX/<version>" entry must be preserved. Callers that actually
// prepend a managed path use sanitizeInheritedGopath to de-duplicate.
func normalizeGopathEntries(raw string) []string {
	if raw == "" {
		return nil
	}
	var out []string
	for _, entry := range filepath.SplitList(raw) {
		p := pathutil.ExpandPath(entry)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

// sanitizeInheritedGopath is normalizeGopathEntries plus removal of goenv-managed
// per-version paths ($GOPATH_PREFIX/<version>), so that goenv re-prepending its
// own managed path does not accumulate duplicates (a duplicated GOPATH makes
// `go install` a silent no-op). Use it ONLY when a managed GOPATH is actually
// being prepended; goPathPattern is cfg.GopathPrefix() (e.g. $HOME/go).
// $GOPATH_PREFIX itself and every other entry are kept.
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
