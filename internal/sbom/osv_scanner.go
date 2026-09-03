package sbom

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// OSVScanner is a built-in vulnerability scanner backed by the OSV / Go
// vulnerability database. Unlike grype/trivy/snyk it requires no external binary
// to be installed, and it is the only backend that scans the Go standard library
// and toolchain components that goenv's SBOM enhancement emits — catching stdlib
// CVEs that generic SBOM scanners miss entirely.
type OSVScanner struct {
	client      OSVClient
	concurrency int
	// enrichSeverity resolves missing severities via a vuln's GHSA alias.
	enrichSeverity bool
}

// NewOSVScanner returns an OSVScanner using the public OSV.dev API.
func NewOSVScanner() *OSVScanner {
	return &OSVScanner{
		client:         NewHTTPOSVClient(),
		concurrency:    8,
		enrichSeverity: true,
	}
}

// Name implements Scanner.
func (s *OSVScanner) Name() string { return "osv" }

// Version implements Scanner. The OSV database is queried live, so there is no
// local tool version; report the protocol/database identity instead.
func (s *OSVScanner) Version() (string, error) { return "osv.dev/v1 (Go vuln DB)", nil }

// IsInstalled implements Scanner. The OSV scanner is built into goenv, so it is
// always available (subject to network access at scan time).
func (s *OSVScanner) IsInstalled() bool { return true }

// InstallationInstructions implements Scanner.
func (s *OSVScanner) InstallationInstructions() string {
	return `The OSV scanner is built into goenv - no installation required.

It queries the OSV.dev vulnerability database (which mirrors the official Go
vulnerability database at https://vuln.go.dev) and requires network access.

For reachability-aware scanning of a project or binary, also consider:
  goenv tools install govulncheck@latest
  goenv exec govulncheck ./...
  goenv exec govulncheck -mode=binary ./bin/app`
}

// SupportsFormat implements Scanner.
func (s *OSVScanner) SupportsFormat(format string) bool {
	return format == "cyclonedx-json" || format == ""
}

// Scan implements Scanner by extracting Go components from a CycloneDX SBOM and
// querying the OSV database for each.
func (s *OSVScanner) Scan(ctx context.Context, opts *ScanOptions) (*ScanResult, error) {
	start := time.Now()

	if opts == nil || opts.SBOMPath == "" {
		return nil, NewScanError("osv", "SBOM path is required", nil)
	}
	if opts.Offline {
		return nil, NewScanError("osv", "offline mode is not supported: OSV scanning requires network access to the vulnerability database", nil)
	}
	if opts.Format != "" && !s.SupportsFormat(opts.Format) {
		return nil, NewScanError("osv", fmt.Sprintf("unsupported format: %s (osv supports cyclonedx-json)", opts.Format), nil)
	}

	data, err := os.ReadFile(opts.SBOMPath)
	if err != nil {
		return nil, NewScanError("osv", fmt.Sprintf("failed to read SBOM: %s", opts.SBOMPath), err)
	}

	var doc cycloneDXDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, NewScanError("osv", "failed to parse CycloneDX SBOM", err)
	}

	targets := extractOSVTargets(&doc)
	if len(targets) == 0 {
		return nil, NewScanError("osv", "no Go components found in SBOM (is it an enhanced CycloneDX SBOM?)", nil)
	}

	vulns, err := s.scanTargets(ctx, targets)
	if err != nil {
		return nil, err
	}

	// Filtering.
	vulns = filterVulnerabilities(vulns, opts)

	result := &ScanResult{
		Scanner:         s.Name(),
		SBOMPath:        opts.SBOMPath,
		SBOMFormat:      "cyclonedx-json",
		Vulnerabilities: vulns,
		Summary:         summarize(vulns),
		Timestamp:       time.Now(),
		Metadata: ScanMetadata{
			DBVersion: "osv.dev",
			// OSV is a live database with no version stamp of its own, so a scan
			// is a point-in-time snapshot. Record when it was queried so the
			// result is honestly reproducible ("vulnerabilities as of this time").
			DBUpdatedAt:  start,
			ScanDuration: time.Since(start),
			GoVersion:    doc.goVersion(),
		},
	}
	version, _ := s.Version()
	result.ScannerVersion = version
	return result, nil
}

// osvTarget is a single package+version to query.
type osvTarget struct {
	name    string
	version string
	pkgType string // "stdlib" or "go-module"
}

// scanTargets queries OSV for each target concurrently and maps results to
// Vulnerability records, applying version-aware fix detection and severity.
func (s *OSVScanner) scanTargets(ctx context.Context, targets []osvTarget) ([]Vulnerability, error) {
	concurrency := s.concurrency
	if concurrency < 1 {
		concurrency = 1
	}

	type job struct {
		target osvTarget
	}
	jobs := make(chan job)
	var (
		mu       sync.Mutex
		all      []Vulnerability
		firstErr error
		wg       sync.WaitGroup
	)

	worker := func() {
		defer wg.Done()
		for j := range jobs {
			resp, err := s.client.Query(ctx, OSVQuery{
				Package: OSVPackage{Name: j.target.name, Ecosystem: osvEcosystem},
				Version: j.target.version,
			})
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				continue
			}
			for i := range resp.Vulns {
				v := &resp.Vulns[i]
				vuln := toVulnerability(v, j.target)
				mu.Lock()
				all = append(all, vuln)
				mu.Unlock()
			}
		}
	}

	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go worker()
	}
	for _, t := range targets {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return nil, ctx.Err()
		case jobs <- job{target: t}:
		}
	}
	close(jobs)
	wg.Wait()

	if firstErr != nil && len(all) == 0 {
		return nil, NewScanError("osv", "vulnerability query failed", firstErr)
	}

	all = dedupeVulnerabilities(all)

	if s.enrichSeverity {
		s.enrichUnknownSeverities(ctx, all)
	}
	sortVulnerabilities(all)
	return all, nil
}

// enrichUnknownSeverities resolves "Unknown" severities by looking up a vuln's
// GHSA alias (which carries CVSS data) with bounded concurrency and a cache.
func (s *OSVScanner) enrichUnknownSeverities(ctx context.Context, vulns []Vulnerability) {
	type idx struct {
		i    int
		ghsa string
	}
	var pending []idx
	for i := range vulns {
		if vulns[i].Severity != "Unknown" && vulns[i].Severity != "" {
			continue
		}
		if g, _ := vulns[i].Metadata["ghsa"].(string); g != "" {
			pending = append(pending, idx{i: i, ghsa: g})
		}
	}
	if len(pending) == 0 {
		return
	}

	concurrency := s.concurrency
	if concurrency < 1 {
		concurrency = 1
	}
	var (
		mu    sync.Mutex
		cache = map[string]string{}
		wg    sync.WaitGroup
		ch    = make(chan idx)
	)
	worker := func() {
		defer wg.Done()
		for p := range ch {
			mu.Lock()
			sev, cached := cache[p.ghsa]
			mu.Unlock()
			if !cached {
				if v, err := s.client.QueryID(ctx, p.ghsa); err == nil {
					sev = severityFromVuln(v)
				} else {
					sev = "Unknown"
				}
				mu.Lock()
				cache[p.ghsa] = sev
				mu.Unlock()
			}
			if sev != "" && sev != "Unknown" {
				mu.Lock()
				vulns[p.i].Severity = sev
				mu.Unlock()
			}
		}
	}
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go worker()
	}
	for _, p := range pending {
		select {
		case <-ctx.Done():
			close(ch)
			wg.Wait()
			return
		case ch <- p:
		}
	}
	close(ch)
	wg.Wait()
}

// toVulnerability converts an OSV vuln (for a specific target) to the internal
// Vulnerability model.
func toVulnerability(v *OSVVuln, target osvTarget) Vulnerability {
	fixed := applicableFixedVersion(v, target.version)
	description := v.Summary
	if description == "" {
		description = firstSentence(v.Details)
	}
	vuln := Vulnerability{
		ID:             v.ID,
		PackageName:    target.name,
		PackageVersion: target.version,
		PackageType:    target.pkgType,
		Severity:       severityFromVuln(v),
		Description:    description,
		URLs:           vulnReferenceURLs(v),
		PublishedAt:    v.Published,
		ModifiedAt:     v.Modified,
		FixedInVersion: fixed,
		FixAvailable:   fixed != "",
		Metadata:       map[string]interface{}{},
	}
	if cve := primaryAlias(v); cve != "" {
		vuln.Metadata["primary_alias"] = cve
	}
	if g := ghsaAlias(v); g != "" {
		vuln.Metadata["ghsa"] = g
	}
	if len(v.Aliases) > 0 {
		vuln.Metadata["aliases"] = v.Aliases
	}
	return vuln
}

// --- CycloneDX extraction -------------------------------------------------

type cycloneDXDoc struct {
	Metadata   cycloneDXMetadata    `json:"metadata"`
	Components []cycloneDXComponent `json:"components"`
}

type cycloneDXMetadata struct {
	Properties []cycloneDXProperty `json:"properties"`
	Component  *cycloneDXComponent `json:"component"`
}

type cycloneDXProperty struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type cycloneDXComponent struct {
	Type       string              `json:"type"`
	Name       string              `json:"name"`
	Version    string              `json:"version"`
	Purl       string              `json:"purl"`
	Properties []cycloneDXProperty `json:"properties"`
}

// goVersion extracts the goenv-recorded Go toolchain version from metadata.
func (d *cycloneDXDoc) goVersion() string {
	for _, p := range d.Metadata.Properties {
		if p.Name == "goenv:go_version" {
			return p.Value
		}
	}
	return ""
}

// extractOSVTargets builds the de-duplicated set of OSV queries from an SBOM.
func extractOSVTargets(doc *cycloneDXDoc) []osvTarget {
	seen := make(map[string]bool)
	var targets []osvTarget
	add := func(name, version, pkgType string) {
		name = strings.TrimSpace(name)
		version = strings.TrimSpace(version)
		if name == "" || version == "" {
			return
		}
		key := name + "@" + version
		if seen[key] {
			return
		}
		seen[key] = true
		targets = append(targets, osvTarget{name: name, version: version, pkgType: pkgType})
	}

	goVer := doc.goVersion()

	for _, c := range doc.Components {
		switch {
		case c.Name == "golang-stdlib":
			// The stdlib component's version is the Go toolchain version.
			v := c.Version
			if v == "" {
				v = goVer
			}
			add("stdlib", v, "stdlib")
			add("toolchain", v, "stdlib")
		case strings.HasPrefix(c.Purl, "pkg:golang/"):
			name, version := parseGoPurl(c.Purl)
			if version == "" {
				version = c.Version
			}
			add(name, version, "go-module")
		case looksLikeModulePath(c.Name) && c.Version != "":
			add(c.Name, c.Version, "go-module")
		}
	}

	// If the SBOM lacked an explicit stdlib component but recorded a Go version,
	// still scan the toolchain/stdlib (defense in depth).
	if goVer != "" && !seen["stdlib@"+goVer] {
		add("stdlib", goVer, "stdlib")
		add("toolchain", goVer, "stdlib")
	}

	return targets
}

// parseGoPurl extracts the module path and version from a Go purl such as
// "pkg:golang/github.com/foo/bar@v1.2.3".
func parseGoPurl(purl string) (string, string) {
	rest := strings.TrimPrefix(purl, "pkg:golang/")
	// Strip qualifiers/subpath.
	if i := strings.IndexAny(rest, "?#"); i != -1 {
		rest = rest[:i]
	}
	name, version := rest, ""
	if at := strings.LastIndex(rest, "@"); at != -1 {
		name = rest[:at]
		version = rest[at+1:]
	}
	if decoded, err := url.PathUnescape(name); err == nil {
		name = decoded
	}
	if decoded, err := url.PathUnescape(version); err == nil {
		version = decoded
	}
	return name, version
}

// looksLikeModulePath heuristically detects a Go module path (domain-qualified).
func looksLikeModulePath(name string) bool {
	if name == "" || name == "golang-stdlib" {
		return false
	}
	slash := strings.Index(name, "/")
	if slash <= 0 {
		return false
	}
	return strings.Contains(name[:slash], ".")
}

// --- result post-processing ----------------------------------------------

func filterVulnerabilities(vulns []Vulnerability, opts *ScanOptions) []Vulnerability {
	if opts == nil {
		return vulns
	}
	threshold := severityRank(opts.SeverityThreshold)
	var out []Vulnerability
	for _, v := range vulns {
		if opts.OnlyFixed && !v.FixAvailable {
			continue
		}
		if threshold > 0 {
			r := severityRank(v.Severity)
			// Keep Unknown (rank 0) so unrated vulns are never silently hidden.
			if r > 0 && r < threshold {
				continue
			}
		}
		out = append(out, v)
	}
	return out
}

func severityRank(s string) int {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium", "moderate":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

func summarize(vulns []Vulnerability) VulnerabilitySummary {
	var sum VulnerabilitySummary
	for _, v := range vulns {
		sum.Total++
		switch strings.ToLower(v.Severity) {
		case "critical":
			sum.Critical++
		case "high":
			sum.High++
		case "medium", "moderate":
			sum.Medium++
		case "low":
			sum.Low++
		case "negligible":
			sum.Negligible++
		default:
			sum.Unknown++
		}
		if v.FixAvailable {
			sum.WithFix++
		} else {
			sum.WithoutFix++
		}
	}
	return sum
}

// dedupeVulnerabilities collapses duplicate (id, package, version) findings that
// can arise when both stdlib and toolchain queries surface the same advisory.
func dedupeVulnerabilities(vulns []Vulnerability) []Vulnerability {
	seen := make(map[string]bool)
	var out []Vulnerability
	for _, v := range vulns {
		key := v.ID + "|" + v.PackageName + "|" + v.PackageVersion
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, v)
	}
	return out
}

// sortVulnerabilities orders by severity (desc) then ID for stable output.
func sortVulnerabilities(vulns []Vulnerability) {
	sort.SliceStable(vulns, func(i, j int) bool {
		ri, rj := severityRank(vulns[i].Severity), severityRank(vulns[j].Severity)
		if ri != rj {
			return ri > rj
		}
		return vulns[i].ID < vulns[j].ID
	})
}

// firstSentence returns a trimmed first sentence/line of a details blob.
func firstSentence(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if i := strings.IndexAny(s, ".\n"); i != -1 {
		return strings.TrimSpace(s[:i+1])
	}
	if len(s) > 200 {
		return s[:200]
	}
	return s
}
