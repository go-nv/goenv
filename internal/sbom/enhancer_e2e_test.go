package sbom

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newEnhancerAt builds an Enhancer with a real toolchain bound to dir. The
// config/manager use an isolated temp root so tests never touch the developer's
// real goenv installation.
func newEnhancerAt(t *testing.T, dir string, offline bool) *Enhancer {
	t.Helper()
	cfg := &config.Config{Root: t.TempDir()}
	mgr := manager.NewManager(cfg, nil)
	return &Enhancer{
		config:    cfg,
		manager:   mgr,
		toolchain: NewToolchain(nil, nil, dir, offline),
	}
}

// writeBaseSBOM writes a minimal CycloneDX document with the given components.
func writeBaseSBOM(t *testing.T, path string, components []map[string]interface{}) {
	t.Helper()
	comps := make([]interface{}, len(components))
	for i, c := range components {
		comps[i] = c
	}
	doc := map[string]interface{}{
		"bomFormat":   "CycloneDX",
		"specVersion": "1.5",
		"metadata": map[string]interface{}{
			"component": map[string]interface{}{"type": "application", "name": "app", "version": "0.0.0"},
		},
		"components": comps,
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o644))
}

// readSBOM parses an on-disk SBOM back into a generic map.
func readSBOM(t *testing.T, path string) map[string]interface{} {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &m))
	return m
}

// findComponent returns the component with the given name, or nil.
func findComponent(sbom map[string]interface{}, name string) map[string]interface{} {
	comps, _ := sbom["components"].([]interface{})
	for _, c := range comps {
		m, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		if n, _ := m["name"].(string); n == name {
			return m
		}
	}
	return nil
}

// metadataProps returns metadata.properties as a name->value map.
func metadataProps(sbom map[string]interface{}) map[string]string {
	out := make(map[string]string)
	meta, _ := sbom["metadata"].(map[string]interface{})
	props, _ := meta["properties"].([]interface{})
	for _, p := range props {
		m, ok := p.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := m["name"].(string)
		value, _ := m["value"].(string)
		out[name] = value
	}
	return out
}

func TestE2E_Enhance_AddsStdlibAndBuildContext(t *testing.T) {
	requireGo(t)
	dir := t.TempDir()
	writeModule(t, dir, "module example.com/enh\n\ngo 1.22\n", stdlibMain)

	sbomPath := filepath.Join(dir, "sbom.json")
	writeBaseSBOM(t, sbomPath, nil)

	e := newEnhancerAt(t, dir, false)
	require.NoError(t, e.EnhanceCycloneDX(sbomPath, EnhanceOptions{ProjectDir: dir}))

	sbom := readSBOM(t, sbomPath)

	// Stdlib component must be present with real packages (the default-dir
	// regression guard, exercised through the full pipeline).
	stdlib := findComponent(sbom, "golang-stdlib")
	require.NotNil(t, stdlib, "golang-stdlib component must be added")
	assert.NotEmpty(t, stdlib["version"])

	props := metadataProps(sbom)
	assert.NotEmpty(t, props["goenv:go_version"])
	assert.NotEmpty(t, props["goenv:platform"])
	assert.NotEmpty(t, props["goenv:build_context.goos"])
	assert.NotEmpty(t, props["goenv:build_context.goarch"])
	assert.Contains(t, props, "goenv:build_context.cgo_enabled")
	assert.NotEmpty(t, props["goenv:toolchain_path"])
}

func TestE2E_Enhance_MarksReplacedComponents(t *testing.T) {
	requireGo(t)
	dir := t.TempDir()
	goMod := `module example.com/repl

go 1.22

require github.com/example/lib v1.2.3

replace github.com/example/lib => ../local-fork
`
	writeModule(t, dir, goMod, "package main\n\nfunc main() {}\n")

	sbomPath := filepath.Join(dir, "sbom.json")
	writeBaseSBOM(t, sbomPath, []map[string]interface{}{
		{"type": "library", "name": "github.com/example/lib", "version": "v1.2.3"},
	})

	e := newEnhancerAt(t, dir, false)
	require.NoError(t, e.EnhanceCycloneDX(sbomPath, EnhanceOptions{ProjectDir: dir}))

	sbom := readSBOM(t, sbomPath)
	comp := findComponent(sbom, "github.com/example/lib")
	require.NotNil(t, comp)

	goenvData, ok := comp["goenv"].(map[string]interface{})
	require.True(t, ok, "replaced component must carry goenv metadata")
	assert.Equal(t, true, goenvData["replaced"])

	rd, ok := goenvData["replace_directive"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "local-path", rd["type"])
	assert.Equal(t, "high", rd["risk_level"])
}

func TestE2E_Enhance_MarksRetractedMainModule(t *testing.T) {
	requireGo(t)
	dir := t.TempDir()
	goMod := `module example.com/retracted

go 1.22

retract v1.0.0 // known bad
`
	writeModule(t, dir, goMod, "package main\n\nfunc main() {}\n")

	sbomPath := filepath.Join(dir, "sbom.json")
	writeBaseSBOM(t, sbomPath, []map[string]interface{}{
		{"type": "application", "name": "example.com/retracted", "version": "v1.0.0"},
	})

	// Offline so we exercise the deterministic modfile fallback path.
	e := newEnhancerAt(t, dir, true)
	require.NoError(t, e.EnhanceCycloneDX(sbomPath, EnhanceOptions{ProjectDir: dir, OfflineMode: true}))

	sbom := readSBOM(t, sbomPath)
	comp := findComponent(sbom, "example.com/retracted")
	require.NotNil(t, comp)
	goenvData, ok := comp["goenv"].(map[string]interface{})
	require.True(t, ok, "retracted component must carry goenv metadata")
	assert.Equal(t, true, goenvData["retracted"])
	assert.NotEmpty(t, goenvData["retraction_reason"])
}

func TestE2E_Enhance_Deterministic_Reproducible(t *testing.T) {
	requireGo(t)
	dir := t.TempDir()
	writeModule(t, dir, "module example.com/det\n\ngo 1.22\n", stdlibMain)

	makeOne := func() string {
		p := filepath.Join(t.TempDir(), "sbom.json")
		writeBaseSBOM(t, p, []map[string]interface{}{
			{"type": "library", "name": "zeta", "version": "1"},
			{"type": "library", "name": "alpha", "version": "2"},
		})
		e := newEnhancerAt(t, dir, false)
		require.NoError(t, e.EnhanceCycloneDX(p, EnhanceOptions{ProjectDir: dir, Deterministic: true}))
		return p
	}

	p1 := makeOne()
	p2 := makeOne()

	h1, err := ComputeSBOMDigest(p1, "sha256")
	require.NoError(t, err)
	h2, err := ComputeSBOMDigest(p2, "sha256")
	require.NoError(t, err)
	assert.Equal(t, h1, h2, "deterministic enhancement must be reproducible across runs")

	// Components must be sorted in deterministic mode.
	sbom := readSBOM(t, p1)
	comps, _ := sbom["components"].([]interface{})
	require.GreaterOrEqual(t, len(comps), 2)
	first, _ := comps[0].(map[string]interface{})
	assert.Equal(t, "alpha", first["name"], "components should be lexicographically sorted")
}

func TestE2E_Enhance_BinaryProvenanceInjected(t *testing.T) {
	goBin := requireGo(t)
	dir := t.TempDir()
	writeModule(t, dir, "module example.com/binprov\n\ngo 1.22\n", "package main\n\nfunc main() {}\n")

	bin := filepath.Join(dir, "app")
	build := exec.Command(goBin, "build", "-o", bin, ".")
	build.Dir = dir
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	out, err := build.CombinedOutput()
	require.NoError(t, err, "go build failed: %s", out)

	sbomPath := filepath.Join(dir, "sbom.json")
	writeBaseSBOM(t, sbomPath, nil)

	e := newEnhancerAt(t, dir, false)
	require.NoError(t, e.EnhanceCycloneDX(sbomPath, EnhanceOptions{ProjectDir: dir, BinaryPath: bin}))

	sbom := readSBOM(t, sbomPath)
	props := metadataProps(sbom)
	// CGO from the binary must be authoritative (false, since we built with 0).
	assert.Equal(t, "false", props["goenv:build_context.cgo_enabled"])
	assert.NotEmpty(t, props["goenv:go_version"])
}

func TestE2E_Enhance_BinaryProvenanceMissingBinaryFails(t *testing.T) {
	requireGo(t)
	dir := t.TempDir()
	writeModule(t, dir, "module example.com/missing\n\ngo 1.22\n", stdlibMain)

	sbomPath := filepath.Join(dir, "sbom.json")
	writeBaseSBOM(t, sbomPath, nil)

	e := newEnhancerAt(t, dir, false)
	err := e.EnhanceCycloneDX(sbomPath, EnhanceOptions{ProjectDir: dir, BinaryPath: filepath.Join(dir, "does-not-exist")})
	assert.Error(t, err, "explicit --binary that does not exist must fail loudly")
}

func TestE2E_Enhance_EmbedDigests(t *testing.T) {
	requireGo(t)
	dir := t.TempDir()
	writeModule(t, dir, "module example.com/dig\n\ngo 1.22\n", stdlibMain)

	sbomPath := filepath.Join(dir, "sbom.json")
	writeBaseSBOM(t, sbomPath, nil)

	e := newEnhancerAt(t, dir, false)
	require.NoError(t, e.EnhanceCycloneDX(sbomPath, EnhanceOptions{ProjectDir: dir, EmbedDigests: true}))

	props := metadataProps(readSBOM(t, sbomPath))
	digest := props["goenv:module_context.go_mod_digest"]
	require.NotEmpty(t, digest, "go.mod digest should be embedded")
	assert.Contains(t, digest, "sha256:")
}
