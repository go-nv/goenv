package tools

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSecurityToolAliasesResolve verifies that the SBOM/secops tool short names
// documented in help and used by `goenv sbom` resolve to installable module
// paths. This is a direct regression guard for the reported bug where
// `goenv tools install cyclonedx-gomod@v1.6.0` failed with
// "missing dot in first path element".
func TestSecurityToolAliasesResolve(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// The exact command from the docs that used to fail.
		{"cyclonedx-gomod@v1.6.0", "github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@v1.6.0"},
		{"cyclonedx-gomod", "github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest"},
		{"cyclonedx", "github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest"},
		{"syft@latest", "github.com/anchore/syft/cmd/syft@latest"},
		{"grype", "github.com/anchore/grype/cmd/grype@latest"},
		{"trivy@v0.50.0", "github.com/aquasecurity/trivy/cmd/trivy@v0.50.0"},
		{"cosign", "github.com/sigstore/cosign/v2/cmd/cosign@latest"},
		{"govulncheck", "golang.org/x/vuln/cmd/govulncheck@latest"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, NormalizePackagePath(tt.input))
		})
	}
}

// TestKnownToolPath verifies the public accessor used for install hints.
func TestKnownToolPath(t *testing.T) {
	p, ok := KnownToolPath("cyclonedx-gomod")
	require.True(t, ok, "cyclonedx-gomod must be a known tool")
	assert.Equal(t, "github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod", p)

	_, ok = KnownToolPath("definitely-not-a-tool")
	assert.False(t, ok)
}

// TestAllToolAliasesAreInstallable is an enterprise-grade invariant: every alias
// in the registry must expand to a module path whose first element is a domain
// (contains a dot). `go install <name>` rejects anything else with
// "missing dot in first path element", so this guards the whole table at once.
func TestAllToolAliasesAreInstallable(t *testing.T) {
	for name, full := range commonTools {
		t.Run(name, func(t *testing.T) {
			normalized := NormalizePackagePath(name)
			// Strip the @version we know NormalizePackagePath appends.
			path := strings.TrimSuffix(normalized, "@latest")
			require.Equal(t, full, path, "alias %q should expand to its full path", name)

			first := path
			if idx := strings.Index(path, "/"); idx != -1 {
				first = path[:idx]
			}
			assert.Contains(t, first, ".",
				"alias %q expands to %q whose first path element %q lacks a dot; `go install` will reject it",
				name, path, first)

			// The produced binary name should match the alias for tools whose
			// alias is intentionally the binary name (sanity for UX consistency).
			if name == "cyclonedx-gomod" || name == "syft" || name == "grype" || name == "trivy" || name == "cosign" || name == "govulncheck" {
				assert.Equal(t, name, ExtractToolName(path),
					"binary name for %q should equal the alias", name)
			}
		})
	}
}
