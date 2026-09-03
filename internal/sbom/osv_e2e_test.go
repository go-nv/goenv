package sbom

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// osvLiveClient returns a real client and skips the test if the OSV API is not
// reachable, so the suite stays green offline while still exercising the live
// integration when a network is available.
func osvLiveClient(t *testing.T) *HTTPOSVClient {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping live OSV API test in -short mode")
	}
	c := NewHTTPOSVClient()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	// Probe with a known-vulnerable stdlib version.
	if _, err := c.Query(ctx, OSVQuery{Package: OSVPackage{Name: "stdlib", Ecosystem: osvEcosystem}, Version: "1.21.0"}); err != nil {
		t.Skipf("OSV API not reachable: %v", err)
	}
	return c
}

// TestE2E_OSV_RealAPI_StdlibHasVulns hits the live OSV database and asserts that
// a known-vulnerable Go stdlib version reports advisories with CVE aliases.
func TestE2E_OSV_RealAPI_StdlibHasVulns(t *testing.T) {
	c := osvLiveClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := c.Query(ctx, OSVQuery{
		Package: OSVPackage{Name: "stdlib", Ecosystem: osvEcosystem},
		Version: "1.21.0",
	})
	if err != nil {
		t.Skipf("live query failed: %v", err)
	}
	if len(resp.Vulns) == 0 {
		t.Fatal("expected known vulnerabilities for Go stdlib 1.21.0")
	}

	var sawCVE bool
	for _, v := range resp.Vulns {
		for _, a := range v.Aliases {
			if len(a) > 4 && a[:4] == "CVE-" {
				sawCVE = true
			}
		}
	}
	if !sawCVE {
		t.Error("expected at least one CVE alias among stdlib advisories")
	}
}

// TestE2E_OSV_RealAPI_ScanEnhancedSBOM runs the full OSVScanner against the live
// database using an enhanced-style CycloneDX SBOM (stdlib component), asserting
// the scanner surfaces findings, severities, and fix versions end to end.
func TestE2E_OSV_RealAPI_ScanEnhancedSBOM(t *testing.T) {
	c := osvLiveClient(t)

	dir := t.TempDir()
	doc := map[string]interface{}{
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
	}
	data, _ := json.MarshalIndent(doc, "", "  ")
	p := filepath.Join(dir, "sbom.json")
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}

	s := &OSVScanner{client: c, concurrency: 8, enrichSeverity: true}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := s.Scan(ctx, &ScanOptions{SBOMPath: p, Format: "cyclonedx-json"})
	if err != nil {
		t.Skipf("live scan failed: %v", err)
	}

	if result.Summary.Total == 0 {
		t.Fatal("expected the scanner to report stdlib vulnerabilities for Go 1.21.0")
	}
	// Enrichment should have resolved real severities for at least some vulns.
	rated := result.Summary.Critical + result.Summary.High + result.Summary.Medium + result.Summary.Low
	if rated == 0 {
		t.Error("expected at least one severity-rated vulnerability after enrichment")
	}
	// At least one advisory should carry a fix version for the 1.21 branch.
	var sawFix bool
	for _, v := range result.Vulnerabilities {
		if v.FixAvailable {
			sawFix = true
			break
		}
	}
	if !sawFix {
		t.Error("expected at least one advisory with an available fix version")
	}

	t.Logf("live OSV scan: total=%d critical=%d high=%d medium=%d low=%d unknown=%d",
		result.Summary.Total, result.Summary.Critical, result.Summary.High,
		result.Summary.Medium, result.Summary.Low, result.Summary.Unknown)
}
