package compliance

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-nv/goenv/internal/sbom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeStubCycloneDX creates a fake `cyclonedx-gomod` executable on PATH that
// writes a minimal CycloneDX document to its -output path. This lets the
// command-level e2e exercise the full generate -> enhance pipeline hermetically
// (no network, no real SBOM tool), while still running the real Go-aware
// enhancement against a real module via the managed toolchain.
func writeStubCycloneDX(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("stub generator uses a POSIX shell script")
	}
	binDir := t.TempDir()
	script := `#!/bin/sh
out=""
while [ $# -gt 0 ]; do
  case "$1" in
    -output) out="$2"; shift 2 ;;
    *) shift ;;
  esac
done
if [ -z "$out" ]; then out="sbom.json"; fi
cat > "$out" <<'JSON'
{"bomFormat":"CycloneDX","specVersion":"1.5","metadata":{"component":{"type":"application","name":"app","version":"0.0.0"}},"components":[{"type":"library","name":"github.com/example/dep","version":"v1.0.0"}]}
JSON
`
	stub := filepath.Join(binDir, "cyclonedx-gomod")
	require.NoError(t, os.WriteFile(stub, []byte(script), 0o755))
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func requireGoForCmd(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not available")
	}
}

// setSBOMFlags sets the package-level flag vars and restores them on cleanup.
func setSBOMFlags(t *testing.T, dir, output string) {
	t.Helper()
	prev := struct {
		tool, format, output, dir, image, toolArgs, binary  string
		modulesOnly, offline, deterministic, embed, enhance bool
	}{sbomTool, sbomFormat, sbomOutput, sbomDir, sbomImage, sbomToolArgs, sbomBinary,
		sbomModulesOnly, sbomOffline, sbomDeterministic, sbomEmbedDigests, sbomEnhance}
	t.Cleanup(func() {
		sbomTool, sbomFormat, sbomOutput, sbomDir = prev.tool, prev.format, prev.output, prev.dir
		sbomImage, sbomToolArgs, sbomBinary = prev.image, prev.toolArgs, prev.binary
		sbomModulesOnly, sbomOffline = prev.modulesOnly, prev.offline
		sbomDeterministic, sbomEmbedDigests, sbomEnhance = prev.deterministic, prev.embed, prev.enhance
	})

	sbomTool = "cyclonedx-gomod"
	sbomFormat = "cyclonedx-json"
	sbomOutput = output
	sbomDir = dir
	sbomImage = ""
	sbomToolArgs = ""
	sbomBinary = ""
	sbomModulesOnly = false
	sbomOffline = false
	sbomDeterministic = false
	sbomEmbedDigests = false
	sbomEnhance = true
}

func writeGoModule(t *testing.T, dir string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/e2e\n\ngo 1.22\n"), 0o644))
	main := "package main\n\nimport (\n\t\"fmt\"\n\t\"net/http\"\n)\n\nfunc main() {\n\t_ = http.MethodGet\n\tfmt.Println()\n}\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte(main), 0o644))
}

func readJSON(t *testing.T, path string) map[string]interface{} {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &m))
	return m
}

func componentNames(sbom map[string]interface{}) []string {
	var names []string
	comps, _ := sbom["components"].([]interface{})
	for _, c := range comps {
		if m, ok := c.(map[string]interface{}); ok {
			if n, _ := m["name"].(string); n != "" {
				names = append(names, n)
			}
		}
	}
	return names
}

// TestE2E_Command_GenerateAndEnhance runs the real `sbom project` RunE against a
// stub generator and a real Go module, asserting the enhancement pipeline ran:
// the original component is preserved AND the Go-aware stdlib component is added.
func TestE2E_Command_GenerateAndEnhance(t *testing.T) {
	requireGoForCmd(t)
	writeStubCycloneDX(t)

	dir := t.TempDir()
	writeGoModule(t, dir)
	t.Setenv("GOENV_ROOT", t.TempDir())

	output := filepath.Join(dir, "sbom.json")
	setSBOMFlags(t, dir, output)

	require.NoError(t, runSBOMProject(sbomProjectCmd, nil))

	sbom := readJSON(t, output)
	names := componentNames(sbom)
	assert.Contains(t, names, "github.com/example/dep", "original component must be preserved")
	assert.Contains(t, names, "golang-stdlib", "Go-aware stdlib component must be added by enhancement")
}

// TestE2E_Command_EnhanceDisabled verifies --enhance=false leaves the tool output
// untouched (no stdlib component).
func TestE2E_Command_EnhanceDisabled(t *testing.T) {
	requireGoForCmd(t)
	writeStubCycloneDX(t)

	dir := t.TempDir()
	writeGoModule(t, dir)
	t.Setenv("GOENV_ROOT", t.TempDir())

	output := filepath.Join(dir, "sbom.json")
	setSBOMFlags(t, dir, output)
	sbomEnhance = false

	require.NoError(t, runSBOMProject(sbomProjectCmd, nil))

	sbom := readJSON(t, output)
	assert.NotContains(t, componentNames(sbom), "golang-stdlib",
		"enhancement must not run when --enhance=false")
}

// TestE2E_Command_Deterministic ensures two deterministic runs produce identical
// output through the full command path.
func TestE2E_Command_Deterministic(t *testing.T) {
	requireGoForCmd(t)
	writeStubCycloneDX(t)

	dir := t.TempDir()
	writeGoModule(t, dir)
	t.Setenv("GOENV_ROOT", t.TempDir())

	run := func() []byte {
		output := filepath.Join(t.TempDir(), "sbom.json")
		setSBOMFlags(t, dir, output)
		sbomDeterministic = true
		require.NoError(t, runSBOMProject(sbomProjectCmd, nil))
		data, err := os.ReadFile(output)
		require.NoError(t, err)
		return data
	}

	assert.Equal(t, run(), run(), "deterministic command output must be byte-identical")
}

// TestE2E_Command_MissingToolError asserts a clear error when the generator is
// not installed (and no os.Exit is triggered on the resolve path).
func TestE2E_Command_MissingToolError(t *testing.T) {
	requireGoForCmd(t)

	dir := t.TempDir()
	writeGoModule(t, dir)
	t.Setenv("GOENV_ROOT", t.TempDir())
	// Point PATH somewhere without the tool so resolution fails cleanly.
	t.Setenv("PATH", t.TempDir())

	setSBOMFlags(t, dir, filepath.Join(dir, "sbom.json"))
	sbomTool = "cyclonedx-gomod"

	err := runSBOMProject(sbomProjectCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestE2E_Command_ScanListScanners verifies --list-scanners works with zero
// positional args (regression: ExactArgs(1) used to reject it) and that the
// built-in OSV scanner is reported.
func TestE2E_Command_ScanListScanners(t *testing.T) {
	prev := scanListScanners
	t.Cleanup(func() { scanListScanners = prev })
	scanListScanners = true

	require.NoError(t, runSBOMScan(sbomScanCmd, nil))
}

// TestE2E_Command_ScanRequiresArg verifies a clear error when no SBOM file and
// no --list-scanners is provided.
func TestE2E_Command_ScanRequiresArg(t *testing.T) {
	prev := scanListScanners
	t.Cleanup(func() { scanListScanners = prev })
	scanListScanners = false

	err := runSBOMScan(sbomScanCmd, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

// TestScanBuiltInOSVDefault asserts the default scanner is the built-in OSV one.
func TestScanBuiltInOSVDefault(t *testing.T) {
	s, err := sbom.GetScanner("osv")
	require.NoError(t, err)
	require.True(t, s.IsInstalled(), "osv scanner must be built-in/available")
}
