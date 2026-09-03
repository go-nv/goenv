package sbom

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCVSSBaseScore(t *testing.T) {
	cases := []struct {
		vector string
		want   float64
		band   string
	}{
		{"CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", 7.5, "High"},
		{"CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H", 10.0, "Critical"},
		{"CVSS:3.1/AV:N/AC:H/PR:N/UI:N/S:U/C:L/I:N/A:N", 3.7, "Low"},
		{"CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H", 7.8, "High"},
	}
	for _, c := range cases {
		got, ok := cvssBaseScore(c.vector)
		require.True(t, ok, "vector should parse: %s", c.vector)
		assert.InDelta(t, c.want, got, 0.05, "score for %s", c.vector)
		assert.Equal(t, c.band, severityBand(got), "band for %s", c.vector)
	}
}

func TestCVSSBaseScore_Invalid(t *testing.T) {
	_, ok := cvssBaseScore("not-a-vector")
	assert.False(t, ok)
	_, ok = cvssBaseScore("CVSS:3.1/AV:N")
	assert.False(t, ok, "incomplete vector must fail")
}

func TestSeverityFromVuln(t *testing.T) {
	// database_specific.severity wins.
	v := &OSVVuln{DatabaseSpecific: map[string]interface{}{"severity": "HIGH"}}
	assert.Equal(t, "High", severityFromVuln(v))

	// CVSS vector fallback.
	v = &OSVVuln{Severity: []OSVSeverity{{Type: "CVSS_V3", Score: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H"}}}
	assert.Equal(t, "Critical", severityFromVuln(v))

	// Nothing available.
	assert.Equal(t, "Unknown", severityFromVuln(&OSVVuln{}))
}

func TestNormalizeSeverity(t *testing.T) {
	assert.Equal(t, "Critical", normalizeSeverity("critical"))
	assert.Equal(t, "High", normalizeSeverity(" HIGH "))
	assert.Equal(t, "Medium", normalizeSeverity("MODERATE"))
	assert.Equal(t, "Medium", normalizeSeverity("medium"))
	assert.Equal(t, "Low", normalizeSeverity("Low"))
	assert.Equal(t, "Unknown", normalizeSeverity("weird"))
}

func TestParseGoVersion(t *testing.T) {
	cases := map[string]goVersion{
		"1.21.3":   {1, 21, 3},
		"1.21":     {1, 21, 0},
		"1.21.0-0": {1, 21, 0},
		"v1.2.3":   {1, 2, 3},
		"go1.20.5": {1, 20, 5},
	}
	for in, want := range cases {
		got, ok := parseGoVersion(in)
		require.True(t, ok, "should parse %s", in)
		assert.Equal(t, want, got, "parse %s", in)
	}
	if _, ok := parseGoVersion(""); ok {
		t.Error("empty should not parse")
	}
}

func TestCompareGoVersion(t *testing.T) {
	lt := func(a, b string) {
		av, _ := parseGoVersion(a)
		bv, _ := parseGoVersion(b)
		assert.Equal(t, -1, compareGoVersion(av, bv), "%s < %s", a, b)
		assert.Equal(t, 1, compareGoVersion(bv, av), "%s > %s", b, a)
	}
	lt("1.20.10", "1.21.0")
	lt("1.21.0", "1.21.3")
	lt("1.21.3", "1.22.0")
	eq, _ := parseGoVersion("1.21.3")
	assert.Equal(t, 0, compareGoVersion(eq, eq))
}

// TestApplicableFixedVersion uses the real multi-branch range shape of
// GO-2023-2102 (fixed in 1.20.10 and 1.21.3).
func TestApplicableFixedVersion(t *testing.T) {
	v := &OSVVuln{
		Affected: []OSVAffected{{
			Package: OSVPackage{Name: "stdlib", Ecosystem: "Go"},
			Ranges: []OSVRange{{
				Type: "SEMVER",
				Events: []OSVRangeEvent{
					{Introduced: "0"},
					{Fixed: "1.20.10"},
					{Introduced: "1.21.0-0"},
					{Fixed: "1.21.3"},
				},
			}},
		}},
	}
	assert.Equal(t, "1.21.3", applicableFixedVersion(v, "1.21.0"))
	assert.Equal(t, "1.20.10", applicableFixedVersion(v, "1.20.5"))
	// Already patched on the 1.21 branch -> no fix applies.
	assert.Equal(t, "", applicableFixedVersion(v, "1.21.5"))
	// Newer minor not covered -> no fix.
	assert.Equal(t, "", applicableFixedVersion(v, "1.22.0"))
}

func TestParseGoPurl(t *testing.T) {
	name, version := parseGoPurl("pkg:golang/github.com/foo/bar@v1.2.3")
	assert.Equal(t, "github.com/foo/bar", name)
	assert.Equal(t, "v1.2.3", version)

	name, version = parseGoPurl("pkg:golang/golang.org/x/net@v0.17.0?type=module")
	assert.Equal(t, "golang.org/x/net", name)
	assert.Equal(t, "v0.17.0", version)

	name, version = parseGoPurl("pkg:golang/stdlib")
	assert.Equal(t, "stdlib", name)
	assert.Equal(t, "", version)
}

func TestLooksLikeModulePath(t *testing.T) {
	assert.True(t, looksLikeModulePath("github.com/foo/bar"))
	assert.True(t, looksLikeModulePath("golang.org/x/net"))
	assert.False(t, looksLikeModulePath("fmt"))
	assert.False(t, looksLikeModulePath("golang-stdlib"))
	assert.False(t, looksLikeModulePath("localpkg"))
}

func TestExtractOSVTargets(t *testing.T) {
	doc := &cycloneDXDoc{
		Metadata: cycloneDXMetadata{
			Properties: []cycloneDXProperty{{Name: "goenv:go_version", Value: "1.21.0"}},
		},
		Components: []cycloneDXComponent{
			{Type: "library", Name: "golang-stdlib", Version: "1.21.0"},
			{Type: "library", Name: "github.com/foo/bar", Version: "v1.0.0", Purl: "pkg:golang/github.com/foo/bar@v1.0.0"},
			{Type: "library", Name: "golang.org/x/net", Version: "v0.17.0"},
		},
	}
	targets := extractOSVTargets(doc)

	byKey := map[string]osvTarget{}
	for _, tt := range targets {
		byKey[tt.name+"@"+tt.version] = tt
	}
	assert.Contains(t, byKey, "stdlib@1.21.0")
	assert.Contains(t, byKey, "toolchain@1.21.0")
	assert.Contains(t, byKey, "github.com/foo/bar@v1.0.0")
	assert.Contains(t, byKey, "golang.org/x/net@v0.17.0")
	assert.Equal(t, "stdlib", byKey["stdlib@1.21.0"].pkgType)
	assert.Equal(t, "go-module", byKey["github.com/foo/bar@v1.0.0"].pkgType)
}

func TestFilterAndSummarize(t *testing.T) {
	vulns := []Vulnerability{
		{ID: "A", Severity: "Critical", FixAvailable: true},
		{ID: "B", Severity: "High", FixAvailable: false},
		{ID: "C", Severity: "Low", FixAvailable: true},
		{ID: "D", Severity: "Unknown", FixAvailable: false},
	}

	// Severity threshold high: keep Critical/High, keep Unknown (fail-safe), drop Low.
	got := filterVulnerabilities(vulns, &ScanOptions{SeverityThreshold: "high"})
	ids := map[string]bool{}
	for _, v := range got {
		ids[v.ID] = true
	}
	assert.True(t, ids["A"] && ids["B"] && ids["D"])
	assert.False(t, ids["C"], "Low should be filtered out by high threshold")

	// OnlyFixed keeps A and C.
	got = filterVulnerabilities(vulns, &ScanOptions{OnlyFixed: true})
	assert.Len(t, got, 2)

	sum := summarize(vulns)
	assert.Equal(t, 4, sum.Total)
	assert.Equal(t, 1, sum.Critical)
	assert.Equal(t, 1, sum.High)
	assert.Equal(t, 1, sum.Low)
	assert.Equal(t, 1, sum.Unknown)
	assert.Equal(t, 2, sum.WithFix)
}

func TestDedupeVulnerabilities(t *testing.T) {
	in := []Vulnerability{
		{ID: "GO-1", PackageName: "stdlib", PackageVersion: "1.21.0"},
		{ID: "GO-1", PackageName: "stdlib", PackageVersion: "1.21.0"},
		{ID: "GO-1", PackageName: "toolchain", PackageVersion: "1.21.0"},
	}
	out := dedupeVulnerabilities(in)
	assert.Len(t, out, 2, "same id different package kept; exact dup removed")
}

// TestHTTPOSVClient_Query exercises the real HTTP client against a local
// httptest server (no network).
func TestHTTPOSVClient_Query(t *testing.T) {
	var gotBody OSVQuery
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/query", r.URL.Path)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(OSVResponse{Vulns: []OSVVuln{{ID: "GO-2023-2102"}}})
	}))
	defer srv.Close()

	c := &HTTPOSVClient{BaseURL: srv.URL, HTTPClient: srv.Client(), UserAgent: "test"}
	resp, err := c.Query(context.Background(), OSVQuery{Package: OSVPackage{Name: "stdlib", Ecosystem: "Go"}, Version: "1.21.0"})
	require.NoError(t, err)
	require.Len(t, resp.Vulns, 1)
	assert.Equal(t, "GO-2023-2102", resp.Vulns[0].ID)
	assert.Equal(t, "stdlib", gotBody.Package.Name)
	assert.Equal(t, "1.21.0", gotBody.Version)
}

// TestHTTPOSVClient_RetriesOn500 verifies transient 5xx responses are retried.
func TestHTTPOSVClient_RetriesOn500(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(OSVResponse{})
	}))
	defer srv.Close()

	c := &HTTPOSVClient{BaseURL: srv.URL, HTTPClient: srv.Client(), MaxRetries: 3}
	_, err := c.Query(context.Background(), OSVQuery{Version: "1.0.0"})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, calls, 2, "should have retried after 500")
}

func TestCVSSRoundup(t *testing.T) {
	assert.Equal(t, 4.0, cvssRoundup(4.0))
	assert.InDelta(t, 4.1, cvssRoundup(4.02), 0.001)
	assert.InDelta(t, 7.5, cvssRoundup(7.48), 0.001)
	assert.False(t, math.IsNaN(cvssRoundup(0)))
}
