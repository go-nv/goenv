package sbom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// defaultOSVBaseURL is the public OSV.dev query API. The Go vulnerability
// database (https://vuln.go.dev) is mirrored into OSV under the "Go" ecosystem,
// so a single client covers stdlib, the toolchain, and third-party modules.
const defaultOSVBaseURL = "https://api.osv.dev"

// osvEcosystem is the OSV ecosystem name for Go packages.
const osvEcosystem = "Go"

// OSVQuery is a single OSV package+version query.
type OSVQuery struct {
	Package OSVPackage `json:"package"`
	Version string     `json:"version,omitempty"`
}

// OSVPackage identifies a package within an ecosystem.
type OSVPackage struct {
	Name      string `json:"name"`
	Ecosystem string `json:"ecosystem"`
}

// OSVResponse is the result of a single /v1/query call.
type OSVResponse struct {
	Vulns []OSVVuln `json:"vulns"`
}

// OSVVuln is a subset of the OSV schema (https://ossf.github.io/osv-schema/)
// sufficient for vulnerability reporting.
type OSVVuln struct {
	ID               string                 `json:"id"`
	Aliases          []string               `json:"aliases,omitempty"`
	Summary          string                 `json:"summary,omitempty"`
	Details          string                 `json:"details,omitempty"`
	Affected         []OSVAffected          `json:"affected,omitempty"`
	Severity         []OSVSeverity          `json:"severity,omitempty"`
	References       []OSVReference         `json:"references,omitempty"`
	Published        string                 `json:"published,omitempty"`
	Modified         string                 `json:"modified,omitempty"`
	DatabaseSpecific map[string]interface{} `json:"database_specific,omitempty"`
}

// OSVAffected describes the versions of a package affected by a vulnerability.
type OSVAffected struct {
	Package OSVPackage `json:"package"`
	Ranges  []OSVRange `json:"ranges,omitempty"`
}

// OSVRange is a range of affected versions expressed as introduced/fixed events.
type OSVRange struct {
	Type   string          `json:"type"`
	Events []OSVRangeEvent `json:"events,omitempty"`
}

// OSVRangeEvent is a single introduced/fixed/last_affected event.
type OSVRangeEvent struct {
	Introduced   string `json:"introduced,omitempty"`
	Fixed        string `json:"fixed,omitempty"`
	LastAffected string `json:"last_affected,omitempty"`
}

// OSVSeverity is a machine-readable severity score (usually a CVSS vector).
type OSVSeverity struct {
	Type  string `json:"type"`
	Score string `json:"score"`
}

// OSVReference is a URL reference for a vulnerability.
type OSVReference struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

// OSVClient queries a vulnerability database using the OSV protocol.
type OSVClient interface {
	// Query returns vulnerabilities affecting a specific package version.
	Query(ctx context.Context, q OSVQuery) (*OSVResponse, error)
	// QueryID hydrates a single vulnerability by its ID (used to resolve
	// severity from a CVE/GHSA alias when the primary entry omits it).
	QueryID(ctx context.Context, id string) (*OSVVuln, error)
}

// HTTPOSVClient is the production OSVClient backed by the OSV.dev HTTP API.
type HTTPOSVClient struct {
	BaseURL    string
	HTTPClient *http.Client
	UserAgent  string
	// MaxRetries bounds transient-failure retries (network/5xx).
	MaxRetries int
}

// NewHTTPOSVClient returns an HTTPOSVClient with production defaults.
func NewHTTPOSVClient() *HTTPOSVClient {
	return &HTTPOSVClient{
		BaseURL:    defaultOSVBaseURL,
		HTTPClient: &http.Client{Timeout: 20 * time.Second},
		UserAgent:  "goenv-sbom-osv",
		MaxRetries: 2,
	}
}

func (c *HTTPOSVClient) baseURL() string {
	if c.BaseURL != "" {
		return strings.TrimRight(c.BaseURL, "/")
	}
	return defaultOSVBaseURL
}

func (c *HTTPOSVClient) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 20 * time.Second}
}

// Query implements OSVClient.
func (c *HTTPOSVClient) Query(ctx context.Context, q OSVQuery) (*OSVResponse, error) {
	body, err := json.Marshal(q)
	if err != nil {
		return nil, err
	}
	var resp OSVResponse
	if err := c.doJSON(ctx, "POST", "/v1/query", body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// QueryID implements OSVClient.
func (c *HTTPOSVClient) QueryID(ctx context.Context, id string) (*OSVVuln, error) {
	var v OSVVuln
	if err := c.doJSON(ctx, "GET", "/v1/vulns/"+id, nil, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// doJSON performs an HTTP request with bounded retries and decodes JSON.
func (c *HTTPOSVClient) doJSON(ctx context.Context, method, path string, body []byte, out interface{}) error {
	url := c.baseURL() + path

	var lastErr error
	retries := c.MaxRetries
	if retries < 0 {
		retries = 0
	}
	for attempt := 0; attempt <= retries; attempt++ {
		if attempt > 0 {
			// Exponential backoff, capped, and cancellable.
			backoff := time.Duration(attempt*attempt) * 250 * time.Millisecond
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		var reader io.Reader
		if body != nil {
			reader = bytes.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, url, reader)
		if err != nil {
			return err
		}
		req.Header.Set("Accept", "application/json")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if c.UserAgent != "" {
			req.Header.Set("User-Agent", c.UserAgent)
		}

		resp, err := c.httpClient().Do(req)
		if err != nil {
			lastErr = err
			continue // transient; retry
		}

		data, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("OSV API returned status %d", resp.StatusCode)
			continue // server-side; retry
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("OSV API returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
		}
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("failed to decode OSV response: %w", err)
		}
		return nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("OSV request failed")
	}
	return lastErr
}

// --- Severity handling ---------------------------------------------------

// severityFromVuln extracts a normalized severity (Critical/High/Medium/Low/
// Unknown) from an OSV vulnerability. It prefers the qualitative
// database_specific.severity string (present on GHSA entries) and falls back to
// computing a CVSS v3 base score from the vector.
func severityFromVuln(v *OSVVuln) string {
	if s := databaseSpecificSeverity(v); s != "" {
		return normalizeSeverity(s)
	}
	for _, sev := range v.Severity {
		if strings.HasPrefix(sev.Type, "CVSS_V3") || strings.HasPrefix(sev.Type, "CVSS_V4") {
			if score, ok := cvssBaseScore(sev.Score); ok {
				return severityBand(score)
			}
		}
	}
	return "Unknown"
}

// databaseSpecificSeverity reads a qualitative severity string if present.
func databaseSpecificSeverity(v *OSVVuln) string {
	if v.DatabaseSpecific == nil {
		return ""
	}
	if s, ok := v.DatabaseSpecific["severity"].(string); ok {
		return s
	}
	return ""
}

// normalizeSeverity maps assorted severity strings to canonical bands.
func normalizeSeverity(s string) string {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "CRITICAL":
		return "Critical"
	case "HIGH":
		return "High"
	case "MODERATE", "MEDIUM":
		return "Medium"
	case "LOW":
		return "Low"
	case "NEGLIGIBLE":
		return "Negligible"
	default:
		return "Unknown"
	}
}

// severityBand maps a CVSS base score to a qualitative severity band.
func severityBand(score float64) string {
	switch {
	case score >= 9.0:
		return "Critical"
	case score >= 7.0:
		return "High"
	case score >= 4.0:
		return "Medium"
	case score > 0.0:
		return "Low"
	default:
		return "Negligible"
	}
}

// cvssBaseScore computes a CVSS v3.x base score from a vector string. It returns
// (score, true) on success. Only base metrics are used (temporal/environmental
// are ignored), which is standard for advisory severity banding.
func cvssBaseScore(vector string) (float64, bool) {
	metrics := parseCVSSVector(vector)
	av, ok1 := cvssWeight(cvssAV, metrics["AV"])
	ac, ok2 := cvssWeight(cvssAC, metrics["AC"])
	ui, ok4 := cvssWeight(cvssUI, metrics["UI"])
	cImp, ok5 := cvssWeight(cvssCIA, metrics["C"])
	iImp, ok6 := cvssWeight(cvssCIA, metrics["I"])
	aImp, ok7 := cvssWeight(cvssCIA, metrics["A"])
	scope, ok8 := metrics["S"]
	if !(ok1 && ok2 && ok4 && ok5 && ok6 && ok7 && ok8) {
		return 0, false
	}
	changed := scope == "C"

	// Privileges Required weighting depends on Scope.
	prTable := cvssPRUnchanged
	if changed {
		prTable = cvssPRChanged
	}
	pr, ok3 := cvssWeight(prTable, metrics["PR"])
	if !ok3 {
		return 0, false
	}

	iss := 1 - (1-cImp)*(1-iImp)*(1-aImp)
	var impact float64
	if changed {
		impact = 7.52*(iss-0.029) - 3.25*math.Pow(iss-0.02, 15)
	} else {
		impact = 6.42 * iss
	}
	if impact <= 0 {
		return 0, true
	}
	exploitability := 8.22 * av * ac * pr * ui
	var score float64
	if changed {
		score = cvssRoundup(math.Min(1.08*(impact+exploitability), 10))
	} else {
		score = cvssRoundup(math.Min(impact+exploitability, 10))
	}
	return score, true
}

// parseCVSSVector parses "CVSS:3.1/AV:N/AC:L/..." into a metric map.
func parseCVSSVector(vector string) map[string]string {
	out := make(map[string]string)
	for _, part := range strings.Split(vector, "/") {
		kv := strings.SplitN(part, ":", 2)
		if len(kv) == 2 {
			out[kv[0]] = kv[1]
		}
	}
	return out
}

var (
	cvssAV          = map[string]float64{"N": 0.85, "A": 0.62, "L": 0.55, "P": 0.2}
	cvssAC          = map[string]float64{"L": 0.77, "H": 0.44}
	cvssUI          = map[string]float64{"N": 0.85, "R": 0.62}
	cvssCIA         = map[string]float64{"H": 0.56, "L": 0.22, "N": 0.0}
	cvssPRUnchanged = map[string]float64{"N": 0.85, "L": 0.62, "H": 0.27}
	cvssPRChanged   = map[string]float64{"N": 0.85, "L": 0.68, "H": 0.5}
)

func cvssWeight(table map[string]float64, key string) (float64, bool) {
	v, ok := table[key]
	return v, ok
}

// cvssRoundup implements the CVSS v3.1 Roundup function.
func cvssRoundup(x float64) float64 {
	scaled := int(math.Round(x * 100000))
	if scaled%10000 == 0 {
		return float64(scaled) / 100000
	}
	return (math.Floor(float64(scaled)/10000) + 1) / 10
}

// --- Go version range logic ----------------------------------------------

// applicableFixedVersion returns the version in which the vulnerability was
// fixed for the release branch of scannedVersion, or "" if no fix applies to
// that branch (e.g. an EOL branch that never received a backport).
func applicableFixedVersion(v *OSVVuln, scannedVersion string) string {
	scanned, ok := parseGoVersion(scannedVersion)
	if !ok {
		return ""
	}
	var best string
	var bestVer goVersion
	for _, aff := range v.Affected {
		for _, r := range aff.Ranges {
			if r.Type != "SEMVER" && r.Type != "ECOSYSTEM" {
				continue
			}
			introduced := goVersion{}
			for _, e := range r.Events {
				if e.Introduced != "" {
					iv, ok := parseGoVersion(e.Introduced)
					if ok {
						introduced = iv
					}
					continue
				}
				if e.Fixed != "" {
					fv, ok := parseGoVersion(e.Fixed)
					if !ok {
						continue
					}
					// The scanned version is covered by [introduced, fixed) and
					// shares the same major.minor as the fix (branch-aware), or
					// the range has no minor boundary. Choose the smallest such
					// fix greater than the scanned version.
					if compareGoVersion(scanned, fv) < 0 && compareGoVersion(scanned, introduced) >= 0 {
						if best == "" || compareGoVersion(fv, bestVer) < 0 {
							best = e.Fixed
							bestVer = fv
						}
					}
				}
			}
		}
	}
	return best
}

type goVersion struct {
	major, minor, patch int
}

// parseGoVersion parses "1.21.3", "1.21", "1.21.0-0", "v1.2.3" into components.
func parseGoVersion(s string) (goVersion, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	s = strings.TrimPrefix(s, "go")
	// Drop any pre-release suffix (e.g. "1.21.0-0" -> "1.21.0").
	if idx := strings.IndexAny(s, "-+"); idx != -1 {
		s = s[:idx]
	}
	if s == "" {
		return goVersion{}, false
	}
	parts := strings.Split(s, ".")
	var gv goVersion
	var err error
	if len(parts) >= 1 {
		if gv.major, err = strconv.Atoi(parts[0]); err != nil {
			return goVersion{}, false
		}
	}
	if len(parts) >= 2 {
		if gv.minor, err = strconv.Atoi(parts[1]); err != nil {
			return goVersion{}, false
		}
	}
	if len(parts) >= 3 {
		if gv.patch, err = strconv.Atoi(parts[2]); err != nil {
			return goVersion{}, false
		}
	}
	return gv, true
}

// compareGoVersion returns -1, 0, or 1.
func compareGoVersion(a, b goVersion) int {
	if a.major != b.major {
		return sign(a.major - b.major)
	}
	if a.minor != b.minor {
		return sign(a.minor - b.minor)
	}
	return sign(a.patch - b.patch)
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	default:
		return 0
	}
}

// primaryAlias returns the most useful external alias (prefer CVE, then GHSA).
func primaryAlias(v *OSVVuln) string {
	var ghsa string
	for _, a := range v.Aliases {
		if strings.HasPrefix(a, "CVE-") {
			return a
		}
		if strings.HasPrefix(a, "GHSA-") && ghsa == "" {
			ghsa = a
		}
	}
	return ghsa
}

// ghsaAlias returns the first GHSA alias, used for severity enrichment.
func ghsaAlias(v *OSVVuln) string {
	for _, a := range v.Aliases {
		if strings.HasPrefix(a, "GHSA-") {
			return a
		}
	}
	return ""
}

// vulnReferenceURLs returns a compact set of reference URLs.
func vulnReferenceURLs(v *OSVVuln) []string {
	var urls []string
	for _, r := range v.References {
		if r.URL != "" {
			urls = append(urls, r.URL)
		}
		if len(urls) >= 5 {
			break
		}
	}
	return urls
}
