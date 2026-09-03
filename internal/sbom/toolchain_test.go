package sbom

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/go-nv/goenv/internal/config"
)

func TestParseBuildTags(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"-tags=netgo,osusergo", []string{"netgo", "osusergo"}},
		{"-tags netgo,osusergo", []string{"netgo", "osusergo"}},
		{"-mod=mod -tags=integration", []string{"integration"}},
		{"-tags=a b", []string{"a"}},
		{"", nil},
		{"-v -race", nil},
	}
	for _, c := range cases {
		got := parseBuildTags(c.in)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("parseBuildTags(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestClassifyReplaces(t *testing.T) {
	in := []GoModReplace{
		{Old: GoModVersion{Path: "github.com/a/b"}, New: GoModVersion{Path: "../local"}},
		{Old: GoModVersion{Path: "github.com/a/b", Version: "v1.0.0"}, New: GoModVersion{Path: "github.com/fork/b", Version: "v1.1.0"}},
		{Old: GoModVersion{Path: "github.com/a/b"}, New: GoModVersion{Path: "github.com/a/b", Version: "v1.2.0"}},
	}
	got := classifyReplaces(in)
	if len(got) != 3 {
		t.Fatalf("expected 3 directives, got %d", len(got))
	}
	if got[0].Type != "local-path" || got[0].RiskLevel != "high" {
		t.Errorf("local path: got type=%s risk=%s", got[0].Type, got[0].RiskLevel)
	}
	if got[1].Type != "fork" || got[1].RiskLevel != "medium" {
		t.Errorf("fork: got type=%s risk=%s", got[1].Type, got[1].RiskLevel)
	}
	if got[2].Type != "version" || got[2].RiskLevel != "low" {
		t.Errorf("version: got type=%s risk=%s", got[2].Type, got[2].RiskLevel)
	}
}

func TestVersionRetracted(t *testing.T) {
	ranges := []GoModRetract{
		{Low: "v1.0.0", High: "v1.0.0"},
		{Low: "v1.2.0", High: "v1.2.5"},
	}
	if !versionRetracted("v1.0.0", ranges) {
		t.Error("v1.0.0 should be retracted (exact endpoint)")
	}
	if !versionRetracted("v1.2.5", ranges) {
		t.Error("v1.2.5 should be retracted (high endpoint)")
	}
	if versionRetracted("v1.1.0", ranges) {
		t.Error("v1.1.0 should not match endpoints")
	}
	if versionRetracted("", ranges) {
		t.Error("empty version should never match")
	}
}

// TestDiscoverStdlibImports_RelativeDot guards against the regression where
// filepath.Walk(".") skipped the entire tree because the root's name (".")
// starts with a dot, silently producing an empty stdlib set for the default
// --dir value.
func TestDiscoverStdlibImports_RelativeDot(t *testing.T) {
	dir := t.TempDir()
	main := "package main\n\nimport (\n\t\"crypto/tls\"\n\t\"fmt\"\n\t\"net/http\"\n)\n\nfunc main() {\n\t_ = tls.VersionTLS13\n\t_ = http.MethodGet\n\tfmt.Println()\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(main), 0644); err != nil {
		t.Fatal(err)
	}

	e := &Enhancer{config: &config.Config{Root: dir}}

	oldWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	imports, err := e.discoverStdlibImports(".")
	if err != nil {
		t.Fatalf("discoverStdlibImports: %v", err)
	}
	if len(imports) == 0 {
		t.Fatal("expected stdlib imports for '.', got none (filepath.Walk '.' skip regression)")
	}

	want := map[string]bool{"crypto/tls": false, "fmt": false, "net/http": false}
	for _, imp := range imports {
		if _, ok := want[imp]; ok {
			want[imp] = true
		}
	}
	for pkg, found := range want {
		if !found {
			t.Errorf("expected stdlib import %q to be discovered", pkg)
		}
	}
}

// TestReadBinaryProvenance validates the headline differentiator: extracting
// authoritative build provenance from a compiled Go artifact via
// debug/buildinfo. It builds a real binary and asserts the embedded settings
// (which env-var scraping could never recover) are captured.
func TestReadBinaryProvenance(t *testing.T) {
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go toolchain not available")
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/prov\n\ngo 1.22\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	bin := filepath.Join(dir, "app")
	build := exec.Command(goBin, "build", "-trimpath", "-ldflags", "-s -w", "-o", bin, ".")
	build.Dir = dir
	build.Env = append(os.Environ(), "CGO_ENABLED=0", "GOFLAGS=")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}

	prov, err := ReadBinaryProvenance(bin)
	if err != nil {
		t.Fatalf("ReadBinaryProvenance: %v", err)
	}

	if prov.GoVersion == "" {
		t.Error("expected embedded Go version")
	}
	if prov.Settings["GOARCH"] == "" {
		t.Error("expected GOARCH build setting from binary")
	}
	if got := prov.Settings["CGO_ENABLED"]; got != "0" {
		t.Errorf("expected CGO_ENABLED=0 from binary, got %q", got)
	}
	// Go embeds -trimpath (not -ldflags) in build info; assert a reliably
	// recorded build flag is captured.
	if got := prov.Settings["-trimpath"]; got != "true" {
		t.Errorf("expected -trimpath=true build setting, got %q", got)
	}
}

func TestReadBinaryProvenance_NotABinary(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "notgo.txt")
	if err := os.WriteFile(p, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadBinaryProvenance(p); err == nil {
		t.Error("expected error reading provenance from non-Go file")
	}
	if _, err := ReadBinaryProvenance(filepath.Join(dir, "missing")); err == nil {
		t.Error("expected error for missing file")
	}
}
