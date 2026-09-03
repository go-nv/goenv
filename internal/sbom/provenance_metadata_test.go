package sbom

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractBaseToolInfo(t *testing.T) {
	// CycloneDX 1.5 object form: metadata.tools.components[]
	obj := map[string]interface{}{
		"metadata": map[string]interface{}{
			"tools": map[string]interface{}{
				"components": []interface{}{
					map[string]interface{}{"name": "cyclonedx-gomod", "version": "v1.6.0"},
				},
			},
		},
	}
	name, version := extractBaseToolInfo(obj)
	assert.Equal(t, "cyclonedx-gomod", name)
	assert.Equal(t, "v1.6.0", version)

	// Legacy array form: metadata.tools[]
	arr := map[string]interface{}{
		"metadata": map[string]interface{}{
			"tools": []interface{}{
				map[string]interface{}{"name": "syft", "version": "1.0.0", "vendor": "anchore"},
			},
		},
	}
	name, version = extractBaseToolInfo(arr)
	assert.Equal(t, "syft", name)
	assert.Equal(t, "1.0.0", version)

	// Missing tools -> empty.
	name, version = extractBaseToolInfo(map[string]interface{}{"metadata": map[string]interface{}{}})
	assert.Empty(t, name)
	assert.Empty(t, version)
}

// TestEnhancer_RecordsProvenanceRoundTrip verifies the enhancer records the
// goenv generator version and the SBOM tool + version (from the base document's
// own metadata.tools), and that ReadGoenvProvenance reads them back.
func TestEnhancer_RecordsProvenanceRoundTrip(t *testing.T) {
	requireGo(t)
	dir := t.TempDir()
	writeModule(t, dir, "module example.com/prov\n\ngo 1.22\n", stdlibMain)

	base := map[string]interface{}{
		"bomFormat":   "CycloneDX",
		"specVersion": "1.5",
		"metadata": map[string]interface{}{
			"tools": map[string]interface{}{
				"components": []interface{}{
					map[string]interface{}{"name": "cyclonedx-gomod", "version": "v1.6.0"},
				},
			},
		},
		"components": []interface{}{},
	}
	p := filepath.Join(dir, "sbom.json")
	data, err := json.MarshalIndent(base, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(p, data, 0o644))

	e := newEnhancerAt(t, dir, false)
	require.NoError(t, e.EnhanceCycloneDX(p, EnhanceOptions{ProjectDir: dir, GeneratorVersion: "3.9.9-test"}))

	prov, err := ReadGoenvProvenance(p)
	require.NoError(t, err)
	assert.Equal(t, "3.9.9-test", prov.GeneratorVersion, "goenv generator version must be recorded")
	assert.Equal(t, "cyclonedx-gomod", prov.SBOMTool, "SBOM tool name must be recorded from metadata.tools")
	assert.Equal(t, "v1.6.0", prov.SBOMToolVersion, "SBOM tool version must be recorded from metadata.tools")
}

func TestReadGoenvProvenance_ParsesProperties(t *testing.T) {
	dir := t.TempDir()
	doc := map[string]interface{}{
		"metadata": map[string]interface{}{
			"properties": []interface{}{
				map[string]interface{}{"name": "goenv:go_version", "value": "1.23.6"},
				map[string]interface{}{"name": "goenv:generator_version", "value": "3.2.0"},
				map[string]interface{}{"name": "goenv:sbom_tool", "value": "cyclonedx-gomod"},
				map[string]interface{}{"name": "goenv:sbom_tool_version", "value": "v1.6.0"},
				map[string]interface{}{"name": "goenv:build_context.cgo_enabled", "value": "true"},
				map[string]interface{}{"name": "goenv:build_context.tags", "value": `["netgo","osusergo"]`},
				map[string]interface{}{"name": "goenv:module_context.vendored", "value": "true"},
			},
		},
	}
	p := filepath.Join(dir, "sbom.json")
	data, _ := json.Marshal(doc)
	require.NoError(t, os.WriteFile(p, data, 0o644))

	prov, err := ReadGoenvProvenance(p)
	require.NoError(t, err)
	assert.Equal(t, "1.23.6", prov.GoVersion)
	assert.Equal(t, "3.2.0", prov.GeneratorVersion)
	assert.Equal(t, "v1.6.0", prov.SBOMToolVersion)
	assert.True(t, prov.CGOEnabled)
	assert.Equal(t, []string{"netgo", "osusergo"}, prov.BuildTags)
	assert.True(t, prov.Vendored)
}

// TestOSVScanner_RecordsQueryTime verifies the scan stamps when the live OSV
// database was queried, so a result is an honest point-in-time snapshot.
func TestOSVScanner_RecordsQueryTime(t *testing.T) {
	dir := t.TempDir()
	p := writeCycloneDX(t, dir, map[string]interface{}{
		"bomFormat": "CycloneDX", "specVersion": "1.5",
		"components": []interface{}{
			map[string]interface{}{"type": "library", "name": "golang-stdlib", "version": "1.21.0"},
		},
	})
	fake := &fakeOSVClient{byName: map[string][]OSVVuln{}}
	s := &OSVScanner{client: fake, concurrency: 2}

	result, err := s.Scan(context.Background(), &ScanOptions{SBOMPath: p})
	require.NoError(t, err)
	assert.False(t, result.Metadata.DBUpdatedAt.IsZero(), "the query time must be recorded for reproducibility")
	assert.Equal(t, "osv.dev", result.Metadata.DBVersion)
}

// --- provenance policy rule --------------------------------------------------

func provenanceSBOM(props map[string]string) map[string]interface{} {
	var arr []interface{}
	for name, value := range props {
		arr = append(arr, map[string]interface{}{"name": name, "value": value})
	}
	return map[string]interface{}{
		"metadata": map[string]interface{}{"properties": arr},
	}
}

func TestCheckProvenance_ToolchainPin(t *testing.T) {
	pe := &PolicyEngine{}

	sbom := provenanceSBOM(map[string]string{"goenv:go_version": "1.23.6"})

	// Matches a pin -> no violation.
	v, err := pe.checkProvenance(PolicyRule{Name: "pin-go", Type: "provenance", Check: "toolchain-version", Severity: "error", Required: []string{"1.23.6", "1.23.7"}}, sbom)
	require.NoError(t, err)
	assert.Empty(t, v)

	// Does not match the pin -> violation.
	v, err = pe.checkProvenance(PolicyRule{Name: "pin-go", Type: "provenance", Check: "toolchain-version", Severity: "error", Required: []string{"1.24.0"}}, sbom)
	require.NoError(t, err)
	require.Len(t, v, 1)
	assert.Contains(t, v[0].Message, "not one of the pinned versions")

	// Presence-only (no Required) with the property present -> no violation.
	v, err = pe.checkProvenance(PolicyRule{Name: "have-go", Type: "provenance", Check: "toolchain-version", Severity: "warning"}, sbom)
	require.NoError(t, err)
	assert.Empty(t, v)
}

func TestCheckProvenance_MissingProperty(t *testing.T) {
	pe := &PolicyEngine{}
	sbom := provenanceSBOM(map[string]string{}) // no goenv properties

	v, err := pe.checkProvenance(PolicyRule{Name: "have-tool", Type: "provenance", Check: "tool-version", Severity: "error"}, sbom)
	require.NoError(t, err)
	require.Len(t, v, 1)
	assert.Contains(t, v[0].Message, "not recorded")
}

func TestCheckProvenance_UnsupportedCheck(t *testing.T) {
	pe := &PolicyEngine{}
	_, err := pe.checkProvenance(PolicyRule{Name: "x", Type: "provenance", Check: "bogus", Severity: "error"}, provenanceSBOM(nil))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported provenance check")
}

// TestPolicyEngine_ProvenanceRuleEndToEnd validates that a provenance rule is
// accepted by the config validator and enforced through the public API.
func TestPolicyEngine_ProvenanceRuleEndToEnd(t *testing.T) {
	dir := t.TempDir()
	policy := `
version: "1"
rules:
  - name: pin-toolchain
    type: provenance
    check: toolchain-version
    severity: error
    required:
      - "1.23.6"
`
	policyPath := filepath.Join(dir, "policy.yaml")
	require.NoError(t, os.WriteFile(policyPath, []byte(policy), 0o644))

	pe, err := NewPolicyEngine(policyPath)
	require.NoError(t, err, "provenance rule type must be accepted by the validator")

	// An SBOM built with the pinned version passes.
	pass := provenanceSBOM(map[string]string{"goenv:go_version": "1.23.6"})
	violations, err := pe.runRule(pe.config.Rules[0], pass)
	require.NoError(t, err)
	assert.Empty(t, violations)

	// An SBOM built with a different version is flagged.
	fail := provenanceSBOM(map[string]string{"goenv:go_version": "1.22.0"})
	violations, err = pe.runRule(pe.config.Rules[0], fail)
	require.NoError(t, err)
	assert.Len(t, violations, 1)
}
