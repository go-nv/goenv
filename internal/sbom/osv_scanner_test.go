package sbom

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeOSVClient is an in-memory OSVClient for hermetic scanner tests.
type fakeOSVClient struct {
	mu       sync.Mutex
	queries  []OSVQuery
	byName   map[string][]OSVVuln // keyed by package name
	byID     map[string]OSVVuln
	queryErr error
}

func (f *fakeOSVClient) Query(ctx context.Context, q OSVQuery) (*OSVResponse, error) {
	f.mu.Lock()
	f.queries = append(f.queries, q)
	f.mu.Unlock()
	if f.queryErr != nil {
		return nil, f.queryErr
	}
	return &OSVResponse{Vulns: f.byName[q.Package.Name]}, nil
}

func (f *fakeOSVClient) QueryID(ctx context.Context, id string) (*OSVVuln, error) {
	if v, ok := f.byID[id]; ok {
		return &v, nil
	}
	return &OSVVuln{ID: id}, nil
}

func writeCycloneDX(t *testing.T, dir string, doc map[string]interface{}) string {
	t.Helper()
	data, err := json.MarshalIndent(doc, "", "  ")
	require.NoError(t, err)
	p := filepath.Join(dir, "sbom.json")
	require.NoError(t, os.WriteFile(p, data, 0o644))
	return p
}

func TestOSVScanner_Basics(t *testing.T) {
	s := NewOSVScanner()
	assert.Equal(t, "osv", s.Name())
	assert.True(t, s.IsInstalled(), "OSV scanner is built-in and always available")
	assert.True(t, s.SupportsFormat("cyclonedx-json"))
	assert.False(t, s.SupportsFormat("spdx-json"))
	v, err := s.Version()
	require.NoError(t, err)
	assert.NotEmpty(t, v)
}

func TestOSVScanner_OfflineIsRejected(t *testing.T) {
	s := NewOSVScanner()
	_, err := s.Scan(context.Background(), &ScanOptions{SBOMPath: "x.json", Offline: true})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "offline")
}

func TestOSVScanner_NoComponents(t *testing.T) {
	dir := t.TempDir()
	p := writeCycloneDX(t, dir, map[string]interface{}{
		"bomFormat": "CycloneDX", "specVersion": "1.5",
		"components": []interface{}{},
	})
	s := &OSVScanner{client: &fakeOSVClient{byName: map[string][]OSVVuln{}}, concurrency: 2}
	_, err := s.Scan(context.Background(), &ScanOptions{SBOMPath: p})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no Go components")
}

func TestOSVScanner_ScansStdlibComponent(t *testing.T) {
	dir := t.TempDir()
	p := writeCycloneDX(t, dir, map[string]interface{}{
		"bomFormat":   "CycloneDX",
		"specVersion": "1.5",
		"metadata": map[string]interface{}{
			"properties": []interface{}{
				map[string]interface{}{"name": "goenv:go_version", "value": "1.21.0"},
			},
		},
		"components": []interface{}{
			map[string]interface{}{"type": "library", "name": "golang-stdlib", "version": "1.21.0"},
		},
	})

	fake := &fakeOSVClient{
		byName: map[string][]OSVVuln{
			"stdlib": {{
				ID:               "GO-2023-2102",
				Aliases:          []string{"CVE-2023-39325", "GHSA-4374-p667-p6c8"},
				Summary:          "HTTP/2 rapid reset",
				DatabaseSpecific: map[string]interface{}{"severity": "HIGH"},
				Affected: []OSVAffected{{
					Package: OSVPackage{Name: "stdlib", Ecosystem: "Go"},
					Ranges: []OSVRange{{Type: "SEMVER", Events: []OSVRangeEvent{
						{Introduced: "1.21.0-0"}, {Fixed: "1.21.3"},
					}}},
				}},
			}},
		},
	}

	s := &OSVScanner{client: fake, concurrency: 4, enrichSeverity: true}
	result, err := s.Scan(context.Background(), &ScanOptions{SBOMPath: p})
	require.NoError(t, err)

	require.Len(t, result.Vulnerabilities, 1)
	v := result.Vulnerabilities[0]
	assert.Equal(t, "GO-2023-2102", v.ID)
	assert.Equal(t, "stdlib", v.PackageName)
	assert.Equal(t, "1.21.0", v.PackageVersion)
	assert.Equal(t, "High", v.Severity)
	assert.Equal(t, "1.21.3", v.FixedInVersion)
	assert.True(t, v.FixAvailable)
	assert.Equal(t, "CVE-2023-39325", v.Metadata["primary_alias"])
	assert.Equal(t, 1, result.Summary.High)

	// Both stdlib and toolchain must have been queried.
	names := map[string]bool{}
	for _, q := range fake.queries {
		names[q.Package.Name] = true
	}
	assert.True(t, names["stdlib"] && names["toolchain"])
}

func TestOSVScanner_SeverityEnrichmentViaGHSA(t *testing.T) {
	dir := t.TempDir()
	p := writeCycloneDX(t, dir, map[string]interface{}{
		"bomFormat":   "CycloneDX",
		"specVersion": "1.5",
		"components": []interface{}{
			map[string]interface{}{"type": "library", "name": "golang-stdlib", "version": "1.21.0"},
		},
	})

	fake := &fakeOSVClient{
		byName: map[string][]OSVVuln{
			// GO entry lacks severity; only a GHSA alias is present.
			"stdlib": {{
				ID:      "GO-2023-2102",
				Aliases: []string{"CVE-2023-39325", "GHSA-4374-p667-p6c8"},
			}},
		},
		byID: map[string]OSVVuln{
			"GHSA-4374-p667-p6c8": {
				ID:               "GHSA-4374-p667-p6c8",
				DatabaseSpecific: map[string]interface{}{"severity": "HIGH"},
			},
		},
	}

	s := &OSVScanner{client: fake, concurrency: 4, enrichSeverity: true}
	result, err := s.Scan(context.Background(), &ScanOptions{SBOMPath: p})
	require.NoError(t, err)
	require.NotEmpty(t, result.Vulnerabilities)
	assert.Equal(t, "High", result.Vulnerabilities[0].Severity,
		"severity should be enriched from the GHSA alias")
}

func TestOSVScanner_ModuleComponents(t *testing.T) {
	dir := t.TempDir()
	p := writeCycloneDX(t, dir, map[string]interface{}{
		"bomFormat":   "CycloneDX",
		"specVersion": "1.5",
		"components": []interface{}{
			map[string]interface{}{"type": "library", "name": "golang.org/x/net", "version": "v0.10.0",
				"purl": "pkg:golang/golang.org/x/net@v0.10.0"},
		},
	})
	fake := &fakeOSVClient{
		byName: map[string][]OSVVuln{
			"golang.org/x/net": {{ID: "GO-2023-9999", DatabaseSpecific: map[string]interface{}{"severity": "MODERATE"}}},
		},
	}
	s := &OSVScanner{client: fake, concurrency: 2}
	result, err := s.Scan(context.Background(), &ScanOptions{SBOMPath: p})
	require.NoError(t, err)
	require.Len(t, result.Vulnerabilities, 1)
	assert.Equal(t, "golang.org/x/net", result.Vulnerabilities[0].PackageName)
	assert.Equal(t, "Medium", result.Vulnerabilities[0].Severity)
}

func TestOSVScanner_SeverityThresholdFilter(t *testing.T) {
	dir := t.TempDir()
	p := writeCycloneDX(t, dir, map[string]interface{}{
		"bomFormat": "CycloneDX", "specVersion": "1.5",
		"components": []interface{}{
			map[string]interface{}{"type": "library", "name": "golang-stdlib", "version": "1.21.0"},
		},
	})
	fake := &fakeOSVClient{byName: map[string][]OSVVuln{
		"stdlib": {
			{ID: "GO-CRIT", DatabaseSpecific: map[string]interface{}{"severity": "CRITICAL"}},
			{ID: "GO-LOW", DatabaseSpecific: map[string]interface{}{"severity": "LOW"}},
		},
	}}
	s := &OSVScanner{client: fake, concurrency: 2}
	result, err := s.Scan(context.Background(), &ScanOptions{SBOMPath: p, SeverityThreshold: "high"})
	require.NoError(t, err)
	for _, v := range result.Vulnerabilities {
		assert.NotEqual(t, "GO-LOW", v.ID, "low severity should be filtered by high threshold")
	}
}

func TestOSVScanner_QueryErrorSurfacesWhenNoResults(t *testing.T) {
	dir := t.TempDir()
	p := writeCycloneDX(t, dir, map[string]interface{}{
		"bomFormat": "CycloneDX", "specVersion": "1.5",
		"components": []interface{}{
			map[string]interface{}{"type": "library", "name": "golang-stdlib", "version": "1.21.0"},
		},
	})
	fake := &fakeOSVClient{queryErr: assertAnErr{}}
	s := &OSVScanner{client: fake, concurrency: 2}
	_, err := s.Scan(context.Background(), &ScanOptions{SBOMPath: p})
	require.Error(t, err)
}

type assertAnErr struct{}

func (assertAnErr) Error() string { return "boom" }

func TestGetScanner_IncludesOSV(t *testing.T) {
	s, err := GetScanner("osv")
	require.NoError(t, err)
	assert.Equal(t, "osv", s.Name())

	names := map[string]bool{}
	for _, sc := range ListAvailableScanners() {
		names[sc.Name()] = true
	}
	assert.True(t, names["osv"], "osv must be listed among available scanners")
}
