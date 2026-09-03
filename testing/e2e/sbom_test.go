//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// --- goenv sbom scan: built-in OSV scanner -----------------------------------

// TestSBOMScan_ListScannersShowsBuiltInOSV verifies that the real binary reports
// the built-in OSV scanner as available with no installation. This is the outer
// proof that the scanner is wired into the shipped command, not just the
// package: --list-scanners also regressed once (it was rejected by ExactArgs(1))
// and only a real invocation catches that.
func TestSBOMScan_ListScannersShowsBuiltInOSV(t *testing.T) {
	e := newEnv(t)

	res := e.run("sbom", "scan", "--list-scanners")

	if !res.Succeeded() {
		e.Diagnose()
		t.Fatalf("goenv sbom scan --list-scanners failed (exit %d):\n%s", res.ExitCode, res.Output())
	}
	requireContains(t, res.Output(), "osv", "the built-in OSV scanner must be listed")
	requireContains(t, res.Output(), "built in", "OSV must be advertised as needing no installation")
}

// TestSBOMScan_RequiresArgument verifies a clear, non-crashing error when no
// SBOM file and no --list-scanners flag is supplied.
func TestSBOMScan_RequiresArgument(t *testing.T) {
	e := newEnv(t)

	res := e.run("sbom", "scan")

	if res.Succeeded() {
		t.Fatalf("expected 'goenv sbom scan' with no args to fail, got success:\n%s", res.Output())
	}
	if res.TimedOut {
		t.Fatal("goenv sbom scan hung instead of returning an argument error")
	}
	requireContains(t, res.Output(), "required", "the error should explain that an SBOM file is required")
}

// TestSBOMScan_OSVDetectsStdlibVulnerabilities is the headline end-to-end proof:
// the shipped binary, using only its built-in OSV scanner, detects standard
// library CVEs for a pinned Go version — the class of finding that generic SBOM
// scanners miss because they omit the stdlib component entirely.
//
// Network-gated: it queries the live Go vulnerability database.
func TestSBOMScan_OSVDetectsStdlibVulnerabilities(t *testing.T) {
	skipWithoutNetwork(t)

	e := newEnv(t).AllowNetwork()

	// An enhanced-style CycloneDX SBOM pinned to a Go version with known,
	// long-fixed stdlib advisories.
	e.writeFile("sbom.json", enhancedStdlibSBOM("1.21.0"))

	res := e.run("sbom", "scan", "sbom.json", "--scanner=osv", "--output-format=table")

	if res.TimedOut {
		t.Fatal("goenv sbom scan timed out contacting the OSV database")
	}
	// The scan exits zero without --fail-on even when vulnerabilities are found.
	if !res.Succeeded() {
		e.Diagnose()
		t.Fatalf("goenv sbom scan failed (exit %d):\n%s", res.ExitCode, res.Output())
	}
	requireContains(t, res.Output(), "stdlib@1.21.0", "the stdlib component must be scanned")
	requireContains(t, res.Output(), "GO-", "at least one Go advisory ID should be reported")
	requireContains(t, res.Output(), "Upgrade to", "a fixed version should be suggested")
}

// TestSBOMScan_OSVFailOnHighExitsNonZero verifies the CI gate: --fail-on=high
// must produce a non-zero exit when high/critical stdlib advisories are found.
func TestSBOMScan_OSVFailOnHighExitsNonZero(t *testing.T) {
	skipWithoutNetwork(t)

	e := newEnv(t).AllowNetwork()
	e.writeFile("sbom.json", enhancedStdlibSBOM("1.21.0"))

	res := e.run("sbom", "scan", "sbom.json", "--scanner=osv", "--fail-on=high")

	if res.TimedOut {
		t.Fatal("goenv sbom scan timed out contacting the OSV database")
	}
	if res.Succeeded() {
		t.Fatalf("expected non-zero exit from --fail-on=high with known stdlib vulns:\n%s", res.Output())
	}
}

// --- goenv sbom project: Go-aware enhancement --------------------------------

// TestSBOMProject_EnhancesWithStdlibComponent runs the real `sbom project`
// pipeline against a real Go module, using a stub generator so the test needs no
// external SBOM tool or network. It proves the shipped binary augments the
// generator's output with the Go standard library component.
func TestSBOMProject_EnhancesWithStdlibComponent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("stub generator is a POSIX shell script")
	}
	if _, err := lookPath("go"); err != nil {
		t.Skip("a real go toolchain is required to enhance the SBOM")
	}

	e := newEnv(t)

	// A minimal, buildable, stdlib-only module.
	e.writeFile("go.mod", "module example.com/e2e\n\ngo 1.21\n")
	e.writeFile("main.go", "package main\n\nimport (\n\t\"fmt\"\n\t\"net/http\"\n)\n\nfunc main() {\n\t_ = http.MethodGet\n\tfmt.Println()\n}\n")

	// A fake cyclonedx-gomod on PATH that writes a base CycloneDX document.
	binDir := filepath.Join(e.Home, "fakebin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir fakebin: %v", err)
	}
	writeStubGenerator(t, filepath.Join(binDir, "cyclonedx-gomod"))
	e.Set("PATH", binDir+string(filepath.ListSeparator)+sandboxPath())

	res := e.run("sbom", "project", "--tool=cyclonedx-gomod", "--output=sbom.json")

	if !res.Succeeded() {
		e.Diagnose()
		t.Fatalf("goenv sbom project failed (exit %d):\n%s", res.ExitCode, res.Output())
	}
	requireContains(t, res.Output(), "enhanced", "the command should report Go-aware enhancement")

	data, err := os.ReadFile(e.workPath("sbom.json"))
	if err != nil {
		e.Diagnose()
		t.Fatalf("reading generated SBOM: %v", err)
	}
	requireContains(t, string(data), "golang-stdlib", "enhancement must add the Go stdlib component")
	requireContains(t, string(data), "github.com/example/dep", "the original component must be preserved")
}

// TestSBOMProject_GoenvItself is the ultimate real-world check: it generates and
// then scans an SBOM of the goenv repository itself — a real Go module with real
// third-party dependencies (cobra, pflag, …) and heavy standard-library use —
// entirely through the shipped binary.
//
// Network-gated: it installs cyclonedx-gomod with the host toolchain and lets Go
// resolve goenv's module graph. It reuses the host module cache so goenv's
// already-downloaded dependencies are not fetched again.
func TestSBOMProject_GoenvItself(t *testing.T) {
	skipWithoutNetwork(t)
	if runtime.GOOS == "windows" {
		t.Skip("this integration test makes POSIX tooling assumptions")
	}
	goBin, err := lookPath("go")
	if err != nil {
		t.Skip("a real go toolchain is required")
	}

	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("locating repo root: %v", err)
	}

	// Host toolchain identity: used to label the stdlib component with a real
	// version and to share the module cache (so goenv's deps aren't re-fetched).
	goVersion := strings.TrimPrefix(goEnvValue(t, goBin, "GOVERSION"), "go")
	hostModCache := goEnvValue(t, goBin, "GOMODCACHE")
	// On a goenv-using machine `go` is itself a shim under ~/.goenv, which the
	// sandbox strips from PATH. Expose the real toolchain binary via GOROOT/bin
	// so the SBOM tool and enhancer can invoke `go`.
	goRootBin := filepath.Join(goEnvValue(t, goBin, "GOROOT"), "bin")

	// Install cyclonedx-gomod into a stable, reusable bin so repeated runs don't
	// recompile it. CI runners are ephemeral and install once.
	toolBin := filepath.Join(os.TempDir(), "goenv-e2e-tools")
	if err := os.MkdirAll(toolBin, 0o755); err != nil {
		t.Fatalf("mkdir tool bin: %v", err)
	}
	if _, err := os.Stat(filepath.Join(toolBin, "cyclonedx-gomod")); err != nil {
		installCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		install := exec.CommandContext(installCtx, goBin, "install",
			"github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest")
		install.Env = append(os.Environ(), "GOBIN="+toolBin)
		if out, err := install.CombinedOutput(); err != nil {
			t.Skipf("could not install cyclonedx-gomod: %v\n%s", err, out)
		}
	}

	e := newEnv(t).AllowNetwork()
	e.Set("PATH", strings.Join([]string{toolBin, goRootBin, sandboxPath()}, string(filepath.ListSeparator)))
	// A real version label; the enhancer's toolchain still falls back to the host
	// go on PATH because this version is not "installed" under the sandbox root.
	e.Set("GOENV_VERSION", goVersion)
	e.Set("GOMODCACHE", hostModCache)

	out := e.workPath("goenv.cdx.json")
	res := e.runWithTimeout(4*time.Minute, "sbom", "project",
		"--dir", repoRoot, "--tool=cyclonedx-gomod", "--output", out)
	if !res.Succeeded() {
		e.Diagnose()
		t.Fatalf("goenv sbom project on goenv failed (exit %d):\n%s", res.ExitCode, res.Output())
	}

	data, err := os.ReadFile(out)
	if err != nil {
		e.Diagnose()
		t.Fatalf("reading goenv SBOM: %v", err)
	}
	sbomText := string(data)
	requireContains(t, sbomText, "github.com/spf13/cobra", "goenv's real cobra dependency must appear in its SBOM")
	requireContains(t, sbomText, "golang-stdlib", "the Go stdlib component must be added by enhancement")
	requireContains(t, sbomText, "goenv:go_version", "Go-aware provenance must be recorded")

	// Scan the goenv SBOM with the built-in OSV scanner. goenv's dependencies and
	// a current toolchain are usually clean, so assert the scan *completes*
	// (stdlib CVE detection itself is covered by the pinned-version test above).
	scan := e.runWithTimeout(2*time.Minute, "sbom", "scan", out, "--scanner=osv", "--output-format=table")
	if scan.TimedOut {
		t.Fatal("scanning goenv's SBOM timed out contacting the OSV database")
	}
	if !scan.Succeeded() {
		e.Diagnose()
		t.Fatalf("goenv sbom scan on goenv's own SBOM failed (exit %d):\n%s", scan.ExitCode, scan.Output())
	}
	requireContains(t, scan.Output(), "Scan Results", "the scanner should render a results report")
}

// goEnvValue returns a single `go env` value using the host toolchain, skipping
// the test if it cannot be read.
func goEnvValue(t *testing.T, goBin, key string) string {
	t.Helper()
	out, err := exec.Command(goBin, "env", key).Output()
	if err != nil {
		t.Skipf("go env %s failed: %v", key, err)
	}
	return strings.TrimSpace(string(out))
}

// --- goenv sbom project: --binary provenance, determinism, safe failures -----

// TestSBOMProject_BinaryProvenance proves the headline differentiator through the
// shipped binary: an SBOM generated with --binary carries authoritative VCS
// provenance (the commit the artifact was built from), which generic SBOM tools
// running against a source tree do not capture.
func TestSBOMProject_BinaryProvenance(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX stub generator and git assumptions")
	}
	goBin, err := lookPath("go")
	if err != nil {
		t.Skip("a real go toolchain is required to build the fixture binary")
	}
	if _, err := lookPath("git"); err != nil {
		t.Skip("git is required to embed VCS provenance")
	}

	e := newEnv(t)
	e.writeFile("go.mod", "module example.com/prov\n\ngo 1.21\n")
	e.writeFile("main.go", "package main\n\nfunc main() {}\n")

	// A git repo so the compiler embeds vcs.revision into the binary.
	runGitIn(t, e.Dir, "init")
	runGitIn(t, e.Dir, "add", "-A")
	runGitIn(t, e.Dir, "commit", "-m", "init")

	// Build the fixture with the host toolchain (outside the sandbox).
	build := exec.Command(goBin, "build", "-o", "app", ".")
	build.Dir = e.Dir
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building fixture binary failed: %v\n%s", err, out)
	}

	binDir := filepath.Join(e.Home, "fakebin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir fakebin: %v", err)
	}
	writeStubGenerator(t, filepath.Join(binDir, "cyclonedx-gomod"))
	e.Set("PATH", binDir+string(filepath.ListSeparator)+sandboxPath())

	res := e.run("sbom", "project", "--tool=cyclonedx-gomod", "--binary", "app", "--output", "sbom.json")
	if !res.Succeeded() {
		e.Diagnose()
		t.Fatalf("goenv sbom project --binary failed (exit %d):\n%s", res.ExitCode, res.Output())
	}

	data, err := os.ReadFile(e.workPath("sbom.json"))
	if err != nil {
		t.Fatalf("reading SBOM: %v", err)
	}
	requireContains(t, string(data), "goenv:binary.vcs_revision", "the SBOM must record the VCS commit from the binary")
}

// TestSBOMProject_DeterministicReproducible proves that --deterministic yields
// byte-identical SBOMs across runs, and that `goenv sbom hash` reports a stable
// digest — the foundation for signable, auditable SBOMs.
func TestSBOMProject_DeterministicReproducible(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX stub generator")
	}

	e := newEnv(t)
	e.writeFile("go.mod", "module example.com/det\n\ngo 1.21\n")
	e.writeFile("main.go", "package main\n\nimport \"net/http\"\n\nfunc main() { _ = http.MethodGet }\n")

	binDir := filepath.Join(e.Home, "fakebin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir fakebin: %v", err)
	}
	writeStubGenerator(t, filepath.Join(binDir, "cyclonedx-gomod"))
	e.Set("PATH", binDir+string(filepath.ListSeparator)+sandboxPath())

	if r := e.run("sbom", "project", "--tool=cyclonedx-gomod", "--deterministic", "--output", "a.json"); !r.Succeeded() {
		e.Diagnose()
		t.Fatalf("first deterministic run failed:\n%s", r.Output())
	}
	if r := e.run("sbom", "project", "--tool=cyclonedx-gomod", "--deterministic", "--output", "b.json"); !r.Succeeded() {
		t.Fatalf("second deterministic run failed:\n%s", r.Output())
	}

	a, err := os.ReadFile(e.workPath("a.json"))
	if err != nil {
		t.Fatalf("read a.json: %v", err)
	}
	b, err := os.ReadFile(e.workPath("b.json"))
	if err != nil {
		t.Fatalf("read b.json: %v", err)
	}
	if !bytes.Equal(a, b) {
		t.Fatalf("deterministic SBOMs differ between runs:\n--- a ---\n%s\n--- b ---\n%s", a, b)
	}

	h := e.run("sbom", "hash", "a.json")
	if !h.Succeeded() {
		t.Fatalf("goenv sbom hash failed:\n%s", h.Output())
	}
	if !containsHex64(h.Output()) {
		t.Fatalf("expected a 64-hex sha256 digest in hash output:\n%s", h.Output())
	}
}

// TestSBOMProject_MissingToolOfflineIsClean verifies the hardened tool
// resolution: when the generator is absent in offline mode (the sandbox default)
// the command fails clearly and never attempts a network install.
func TestSBOMProject_MissingToolOfflineIsClean(t *testing.T) {
	e := newEnv(t) // sandbox sets GOENV_OFFLINE=1
	e.writeFile("go.mod", "module example.com/missing\n\ngo 1.21\n")
	e.writeFile("main.go", "package main\n\nfunc main() {}\n")

	res := e.run("sbom", "project", "--tool=cyclonedx-gomod", "--output", "sbom.json")
	if res.Succeeded() {
		t.Fatalf("expected failure when the SBOM tool is missing:\n%s", res.Output())
	}
	requireContains(t, res.Output(), "not found", "a missing tool should produce a clear 'not found' error")
	requireNotContains(t, res.Output(), "installing", "offline mode must never attempt a network install")
}

// TestSBOMScan_OSVOfflineRejected verifies the built-in OSV scanner refuses to
// run in offline mode instead of silently producing an empty result.
func TestSBOMScan_OSVOfflineRejected(t *testing.T) {
	e := newEnv(t)
	e.writeFile("sbom.json", enhancedStdlibSBOM("1.21.0"))

	res := e.run("sbom", "scan", "sbom.json", "--scanner=osv", "--offline")
	if res.Succeeded() {
		t.Fatalf("expected offline OSV scan to fail:\n%s", res.Output())
	}
	requireContains(t, res.Output(), "offline", "the error should explain OSV needs network access")
}

// TestSBOMScan_GrypeNotInstalledGivesHelpfulError verifies that selecting an
// uninstalled external scanner produces actionable guidance, not a crash.
func TestSBOMScan_GrypeNotInstalledGivesHelpfulError(t *testing.T) {
	if _, err := lookPath("grype"); err == nil {
		t.Skip("grype is installed on this host; cannot exercise the not-installed path")
	}
	e := newEnv(t)
	e.writeFile("sbom.json", enhancedStdlibSBOM("1.21.0"))

	res := e.run("sbom", "scan", "sbom.json", "--scanner=grype")
	if res.Succeeded() {
		t.Fatalf("expected failure when grype is not installed:\n%s", res.Output())
	}
	requireContains(t, res.Output(), "not installed", "should say the scanner is not installed")
	requireContains(t, res.Output(), "install", "should offer installation guidance")
}

// TestSBOMProject_OfflineOverridesYes verifies a security-relevant interaction:
// offline mode refuses the auto-install even when --yes is given, so a
// reproducible/air-gapped build never reaches out to the network.
func TestSBOMProject_OfflineOverridesYes(t *testing.T) {
	e := newEnv(t) // sandbox sets GOENV_OFFLINE=1
	e.writeFile("go.mod", "module example.com/offyes\n\ngo 1.21\n")
	e.writeFile("main.go", "package main\n\nfunc main() {}\n")

	res := e.run("sbom", "project", "--tool=cyclonedx-gomod", "--yes", "--output", "sbom.json")
	if res.Succeeded() {
		t.Fatalf("expected failure when tool missing offline, even with --yes:\n%s", res.Output())
	}
	requireContains(t, res.Output(), "not found", "offline+--yes should still fail cleanly")
	requireNotContains(t, res.Output(), "installing", "offline must refuse install even with --yes")
}

// TestSBOMScan_OSVOnlyFixedFilter exercises result filtering end-to-end against
// the live database: --only-fixed should still surface stdlib advisories (which
// have fixes) with upgrade guidance. Network-gated.
func TestSBOMScan_OSVOnlyFixedFilter(t *testing.T) {
	skipWithoutNetwork(t)

	e := newEnv(t).AllowNetwork()
	e.writeFile("sbom.json", enhancedStdlibSBOM("1.21.0"))

	res := e.run("sbom", "scan", "sbom.json", "--scanner=osv", "--only-fixed", "--output-format=table")
	if res.TimedOut {
		t.Fatal("goenv sbom scan --only-fixed timed out contacting OSV")
	}
	if !res.Succeeded() {
		e.Diagnose()
		t.Fatalf("goenv sbom scan --only-fixed failed (exit %d):\n%s", res.ExitCode, res.Output())
	}
	requireContains(t, res.Output(), "Scan Results", "the scanner should render a results report")
	requireContains(t, res.Output(), "Upgrade to", "only-fixed results should include fix versions")
}

// TestSBOMProject_RecordsProvenance verifies that the shipped binary records the
// provenance needed to reproduce a result: which goenv produced the SBOM and
// which generator (name + version) built the base document.
func TestSBOMProject_RecordsProvenance(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX stub generator")
	}
	e := newEnv(t)
	e.writeFile("go.mod", "module example.com/prov\n\ngo 1.21\n")
	e.writeFile("main.go", "package main\n\nfunc main() {}\n")

	binDir := filepath.Join(e.Home, "fakebin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir fakebin: %v", err)
	}
	writeStubGenerator(t, filepath.Join(binDir, "cyclonedx-gomod"))
	e.Set("PATH", binDir+string(filepath.ListSeparator)+sandboxPath())

	res := e.run("sbom", "project", "--tool=cyclonedx-gomod", "--output", "sbom.json")
	if !res.Succeeded() {
		e.Diagnose()
		t.Fatalf("goenv sbom project failed (exit %d):\n%s", res.ExitCode, res.Output())
	}

	data, err := os.ReadFile(e.workPath("sbom.json"))
	if err != nil {
		t.Fatalf("reading SBOM: %v", err)
	}
	sbomText := string(data)
	requireContains(t, sbomText, "goenv:generator_version", "the goenv version that produced the SBOM must be recorded")
	requireContains(t, sbomText, "goenv:sbom_tool_version", "the SBOM generator version must be recorded")
	requireContains(t, sbomText, "v1.6.0", "the recorded tool version should come from the generator's own metadata.tools")
}

// TestSBOMProject_AutoInstallsGeneratorWithConsent exercises the full
// auto-install happy path through the shipped binary: with a real managed Go
// version and --yes consent, a missing SBOM generator is installed on demand and
// the SBOM is produced.
//
// Network-gated and toolchain-heavy (it installs a Go version and compiles
// cyclonedx-gomod), so it runs on the weekly network job, not the per-PR path.
// It is intentionally named so the PR job's -run filter does not select it.
func TestSBOMProject_AutoInstallsGeneratorWithConsent(t *testing.T) {
	skipWithoutNetwork(t)
	if runtime.GOOS == "windows" {
		t.Skip("auto-install path is exercised on POSIX runners")
	}

	const version = "1.23.6"
	e := newEnv(t).AllowNetwork()

	// Go's module cache and extracted toolchain store files read-only, which
	// makes t.TempDir()'s RemoveAll cleanup fail with "permission denied". This
	// cleanup is registered after newEnv, so (LIFO) it runs before the TempDir
	// removal and makes the tree deletable.
	t.Cleanup(func() { makeTreeWritable(e.Root); makeTreeWritable(e.Home) })

	// Enforce that our version pinning never needs a toolchain switch: with
	// GOTOOLCHAIN=local, Go refuses to download a different toolchain, so if the
	// pinned generator and installed Go ever stopped lining up this test fails
	// loudly and fast instead of silently pulling one. It also mirrors goenv's
	// own recommended setting (`goenv doctor` advises GOTOOLCHAIN=local).
	e.Set("GOTOOLCHAIN", "local")

	// Keep the install lean: an empty (but enabled) default-tools list means
	// `goenv install` won't pull the shipped dev tools (gopls, etc., which would
	// drag in a newer toolchain), while explicit `goenv tools install` still
	// works — disabling the config entirely would also block the explicit install.
	e.writeGoenvFile("default-tools.yaml", "enabled: true\ntools: []\n")

	if r := e.runWithTimeout(10*time.Minute, "install", version); !r.Succeeded() {
		t.Fatalf("goenv install %s failed:\n%s", version, r.Output())
	}

	e.writeFile("go.mod", "module example.com/autoinstall\n\ngo 1.21\n")
	e.writeFile("main.go", "package main\n\nimport \"net/http\"\n\nfunc main() { _ = http.MethodGet }\n")
	e.writeFile(".go-version", version+"\n")

	// Mimic `goenv init`: put the shims dir on PATH so the freshly-installed
	// generator (and the managed `go` it shells out to) resolve through shims,
	// exactly as they would for a real user. The generator is deliberately absent
	// so --yes drives the auto-install.
	e.Set("PATH", e.path("shims")+string(filepath.ListSeparator)+sandboxPath())

	// Pin the generator to a version whose go.mod requires go 1.20, so it builds
	// cleanly on 1.23.6 with no toolchain switch (enforced by GOTOOLCHAIN=local
	// above). This keeps the happy path deterministic and exercises --tool-version
	// end to end.
	res := e.runWithTimeout(5*time.Minute,
		"sbom", "project", "--tool=cyclonedx-gomod", "--tool-version=v1.6.0", "--yes", "--output", "sbom.json")
	if !res.Succeeded() {
		e.Diagnose()
		t.Fatalf("auto-install SBOM run failed (exit %d):\n%s", res.ExitCode, res.Output())
	}
	requireContains(t, res.Output(), "installing", "--yes consent should trigger the generator install")
	requireContains(t, res.Output(), "v1.6.0", "the pinned generator version should be installed")

	data, err := os.ReadFile(e.workPath("sbom.json"))
	if err != nil {
		t.Fatalf("reading SBOM: %v", err)
	}
	requireContains(t, string(data), "golang-stdlib", "the auto-installed generator must produce an enhanced SBOM")
}

// runGitIn runs a git command in dir with a deterministic identity.
func runGitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
}

// makeTreeWritable adds the owner-write bit to every entry under root so a
// subsequent RemoveAll can delete Go's read-only module cache and toolchain.
func makeTreeWritable(root string) {
	_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		_ = os.Chmod(p, info.Mode()|0o200)
		return nil
	})
}

// containsHex64 reports whether s contains a 64-character hexadecimal run
// (a sha256 digest).
func containsHex64(s string) bool {
	run := 0
	for _, r := range s {
		isHex := (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
		if isHex {
			run++
			if run >= 64 {
				return true
			}
		} else {
			run = 0
		}
	}
	return false
}

// enhancedStdlibSBOM returns a minimal enhanced-style CycloneDX SBOM whose only
// component is the Go standard library pinned to goVersion.
func enhancedStdlibSBOM(goVersion string) string {
	return `{
  "bomFormat": "CycloneDX",
  "specVersion": "1.5",
  "metadata": {
    "properties": [
      {"name": "goenv:go_version", "value": "` + goVersion + `"}
    ]
  },
  "components": [
    {"type": "library", "name": "golang-stdlib", "version": "` + goVersion + `"}
  ]
}
`
}

// writeStubGenerator writes a POSIX shell stub that emulates cyclonedx-gomod by
// writing a fixed CycloneDX document to the path following its -output flag.
func writeStubGenerator(t *testing.T, path string) {
	t.Helper()
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
{"bomFormat":"CycloneDX","specVersion":"1.5","metadata":{"component":{"type":"application","name":"app","version":"0.0.0"},"tools":{"components":[{"type":"application","name":"cyclonedx-gomod","version":"v1.6.0"}]}},"components":[{"type":"library","name":"github.com/example/dep","version":"v1.0.0"}]}
JSON
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write stub generator: %v", err)
	}
}
