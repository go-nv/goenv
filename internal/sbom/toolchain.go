package sbom

import (
	"bytes"
	"context"
	"debug/buildinfo"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
)

// defaultToolchainTimeout bounds how long any single `go` invocation may run.
// Module graph resolution can be slow on cold caches, so this is generous but
// still prevents SBOM generation from hanging a CI pipeline indefinitely.
const defaultToolchainTimeout = 90 * time.Second

// Toolchain provides authoritative Go build/module metadata by invoking the
// goenv-managed `go` binary. It is the source of truth for SBOM enhancement:
// values come from the real toolchain (go list, go mod edit, go env) rather
// than from fragile heuristics such as scraping environment variables or
// hand-parsing go.mod.
type Toolchain struct {
	goBin      string
	version    string
	projectDir string
	offline    bool
	timeout    time.Duration
}

// NewToolchain resolves the go binary for the currently selected goenv version
// and returns a Toolchain rooted at projectDir. It falls back to the `go` binary
// on PATH when no managed version resolves (e.g. `system`), so SBOM enhancement
// still works outside a fully configured goenv environment.
func NewToolchain(cfg *config.Config, mgr *manager.Manager, projectDir string, offline bool) *Toolchain {
	if projectDir == "" {
		projectDir = "."
	}

	t := &Toolchain{
		projectDir: projectDir,
		offline:    offline,
		timeout:    defaultToolchainTimeout,
	}

	// Prefer the goenv-managed toolchain so provenance reflects the exact Go
	// version goenv selected for this project.
	if mgr != nil && cfg != nil {
		if resolved, _, _, err := mgr.GetCurrentVersionResolved(); err == nil && resolved != "" && resolved != "system" {
			candidate := cfg.VersionGoBinary(resolved)
			if fileExists(candidate) {
				t.goBin = candidate
				t.version = resolved
			}
		}
	}

	// Fall back to whatever `go` is on PATH.
	if t.goBin == "" {
		if p, err := exec.LookPath("go"); err == nil {
			t.goBin = p
		}
	}

	return t
}

// Available reports whether a usable go binary was resolved.
func (t *Toolchain) Available() bool {
	return t != nil && t.goBin != ""
}

// GoBinary returns the resolved go binary path (may be empty when unavailable).
func (t *Toolchain) GoBinary() string {
	if t == nil {
		return ""
	}
	return t.goBin
}

// commandEnv builds the environment for toolchain invocations, honoring offline
// mode by disabling the module proxy so no network access occurs.
func (t *Toolchain) commandEnv() []string {
	env := os.Environ()
	if t.offline {
		env = append(env,
			"GOPROXY=off",
			"GOFLAGS=-mod=mod",
			"GOVCS=*:off",
		)
	}
	return env
}

// run executes `go args...` in the project directory and returns stdout.
func (t *Toolchain) run(args ...string) ([]byte, error) {
	if !t.Available() {
		return nil, fmt.Errorf("no go toolchain available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), t.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, t.goBin, args...)
	cmd.Dir = t.projectDir
	cmd.Env = t.commandEnv()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("go %s timed out after %s", strings.Join(args, " "), t.timeout)
		}
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return nil, fmt.Errorf("go %s failed: %s", strings.Join(args, " "), firstLine(msg))
		}
		return nil, fmt.Errorf("go %s failed: %w", strings.Join(args, " "), err)
	}

	return stdout.Bytes(), nil
}

// GoEnv returns the requested `go env` variables using the managed toolchain.
// Requesting specific keys keeps the output small and deterministic.
func (t *Toolchain) GoEnv(keys ...string) (map[string]string, error) {
	args := append([]string{"env", "-json"}, keys...)
	out, err := t.run(args...)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("failed to parse go env output: %w", err)
	}
	return result, nil
}

// StdlibPackages returns the standard-library import paths actually compiled
// into the project's packages, as determined by the toolchain. Because it uses
// `go list -deps`, it correctly honors the active GOOS/GOARCH and build tags and
// works regardless of whether the project directory is given as "." or an
// absolute path (unlike source-tree walking).
func (t *Toolchain) StdlibPackages() ([]string, error) {
	out, err := t.run("list", "-deps", "-f", "{{if .Standard}}{{.ImportPath}}{{end}}", "./...")
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var pkgs []string
	for _, line := range strings.Split(string(out), "\n") {
		pkg := strings.TrimSpace(line)
		if pkg == "" || seen[pkg] {
			continue
		}
		// Pseudo-packages that are not real importable stdlib units.
		if pkg == "C" || pkg == "unsafe" {
			continue
		}
		seen[pkg] = true
		pkgs = append(pkgs, pkg)
	}
	sort.Strings(pkgs)
	return pkgs, nil
}

// ToolchainModule mirrors the JSON emitted by `go list -m -json`.
type ToolchainModule struct {
	Path      string             `json:"Path"`
	Version   string             `json:"Version"`
	Main      bool               `json:"Main"`
	Indirect  bool               `json:"Indirect"`
	Dir       string             `json:"Dir"`
	GoVersion string             `json:"GoVersion"`
	Retracted []string           `json:"Retracted"`
	Replace   *ToolchainModule   `json:"Replace"`
	Error     *ToolchainModError `json:"Error"`
}

// ToolchainModError captures a per-module error from `go list -m`.
type ToolchainModError struct {
	Err string `json:"Err"`
}

// Modules returns the resolved module graph via `go list -m -json all`.
// This reflects the effective versions after MVS and replace directives.
func (t *Toolchain) Modules() ([]ToolchainModule, error) {
	out, err := t.run("list", "-m", "-json", "all")
	if err != nil {
		return nil, err
	}
	return decodeModuleStream(out)
}

// Retractions returns retracted-version information keyed by module path.
// It uses the authoritative `go list -m -retracted` query. In offline mode this
// requires network access it cannot perform, so callers should treat an error
// as "unknown" and fall back to go.mod retract directives.
func (t *Toolchain) Retractions() (map[string][]string, error) {
	if t.offline {
		return nil, fmt.Errorf("retraction lookup requires network access (offline mode)")
	}

	out, err := t.run("list", "-m", "-retracted", "-json", "all")
	if err != nil {
		return nil, err
	}

	mods, err := decodeModuleStream(out)
	if err != nil {
		return nil, err
	}

	result := make(map[string][]string)
	for _, m := range mods {
		if len(m.Retracted) > 0 {
			result[m.Path] = m.Retracted
		}
	}
	return result, nil
}

// decodeModuleStream parses a concatenated stream of JSON module objects as
// produced by `go list -m -json`.
func decodeModuleStream(data []byte) ([]ToolchainModule, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	var mods []ToolchainModule
	for {
		var m ToolchainModule
		if err := dec.Decode(&m); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to parse module JSON: %w", err)
		}
		mods = append(mods, m)
	}
	return mods, nil
}

// GoModFile mirrors the authoritative JSON emitted by `go mod edit -json`.
type GoModFile struct {
	Module  GoModModule    `json:"Module"`
	Go      string         `json:"Go"`
	Require []GoModRequire `json:"Require"`
	Exclude []GoModVersion `json:"Exclude"`
	Replace []GoModReplace `json:"Replace"`
	Retract []GoModRetract `json:"Retract"`
}

// GoModModule is the module stanza of go.mod.
type GoModModule struct {
	Path string `json:"Path"`
}

// GoModRequire is a require directive.
type GoModRequire struct {
	Path     string `json:"Path"`
	Version  string `json:"Version"`
	Indirect bool   `json:"Indirect"`
}

// GoModVersion is a module path/version pair.
type GoModVersion struct {
	Path    string `json:"Path"`
	Version string `json:"Version"`
}

// GoModReplace is a replace directive.
type GoModReplace struct {
	Old GoModVersion `json:"Old"`
	New GoModVersion `json:"New"`
}

// GoModRetract is a retract directive with optional rationale.
type GoModRetract struct {
	Low       string `json:"Low"`
	High      string `json:"High"`
	Rationale string `json:"Rationale"`
}

// ModEdit returns the structured go.mod via `go mod edit -json`. This is the
// authoritative parser (it uses the toolchain's own modfile implementation) and
// requires no network access, correctly handling block syntax, comments, and
// complex replace/retract directives that a hand-rolled parser would mishandle.
func (t *Toolchain) ModEdit() (*GoModFile, error) {
	out, err := t.run("mod", "edit", "-json")
	if err != nil {
		return nil, err
	}

	var mod GoModFile
	if err := json.Unmarshal(out, &mod); err != nil {
		return nil, fmt.Errorf("failed to parse go.mod JSON: %w", err)
	}
	return &mod, nil
}

// BinaryProvenance holds authoritative build metadata extracted from a compiled
// Go binary via debug/buildinfo. This is the highest-fidelity provenance source
// available: the settings are embedded by the compiler at build time and cannot
// be reconstructed from environment variables after the fact.
type BinaryProvenance struct {
	GoVersion    string             `json:"go_version"`
	Path         string             `json:"path,omitempty"`
	MainModule   string             `json:"main_module,omitempty"`
	MainVersion  string             `json:"main_version,omitempty"`
	Settings     map[string]string  `json:"settings,omitempty"`
	VCSRevision  string             `json:"vcs_revision,omitempty"`
	VCSTime      string             `json:"vcs_time,omitempty"`
	VCSModified  bool               `json:"vcs_modified,omitempty"`
	Dependencies []BinaryDependency `json:"dependencies,omitempty"`
}

// BinaryDependency is a module dependency recorded in a compiled binary.
type BinaryDependency struct {
	Path        string `json:"path"`
	Version     string `json:"version"`
	Sum         string `json:"sum,omitempty"`
	Replacement string `json:"replacement,omitempty"`
}

// ReadBinaryProvenance extracts embedded build information from a compiled Go
// binary. Returns an actionable error if the file is not a Go binary.
func ReadBinaryProvenance(path string) (*BinaryProvenance, error) {
	if !fileExists(path) {
		return nil, fmt.Errorf("binary not found: %s", path)
	}

	info, err := buildinfo.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read Go build info from %s (is it a Go binary?): %w", filepath.Base(path), err)
	}

	prov := &BinaryProvenance{
		GoVersion:   strings.TrimPrefix(info.GoVersion, "go"),
		Path:        info.Path,
		MainModule:  info.Main.Path,
		MainVersion: info.Main.Version,
		Settings:    make(map[string]string),
	}

	for _, s := range info.Settings {
		if s.Value == "" {
			continue
		}
		switch s.Key {
		case "vcs.revision":
			prov.VCSRevision = s.Value
		case "vcs.time":
			prov.VCSTime = s.Value
		case "vcs.modified":
			prov.VCSModified = s.Value == "true"
		default:
			prov.Settings[s.Key] = s.Value
		}
	}

	for _, dep := range info.Deps {
		if dep == nil {
			continue
		}
		bd := BinaryDependency{
			Path:    dep.Path,
			Version: dep.Version,
			Sum:     dep.Sum,
		}
		if dep.Replace != nil {
			bd.Replacement = dep.Replace.Path
			if dep.Replace.Version != "" {
				bd.Replacement += "@" + dep.Replace.Version
			}
			bd.Version = dep.Replace.Version
			if dep.Replace.Sum != "" {
				bd.Sum = dep.Replace.Sum
			}
		}
		prov.Dependencies = append(prov.Dependencies, bd)
	}

	return prov, nil
}

// firstLine returns the first non-empty line of s, trimmed.
func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return strings.TrimSpace(s)
}

// fileExists reports whether path exists and is a regular file.
func fileExists(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
