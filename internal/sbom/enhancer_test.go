package sbom

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
)

// sanitizeTestName replaces characters invalid in Windows paths
func sanitizeTestName(name string) string {
	replacer := strings.NewReplacer(
		":", "_",
		"<", "_",
		">", "_",
		"\"", "_",
		"/", "_",
		"\\", "_",
		"|", "_",
		"?", "_",
		"*", "_",
	)
	return replacer.Replace(name)
}

func TestComputeSBOMDigest(t *testing.T) {
	tempDir := t.TempDir()
	sbomPath := filepath.Join(tempDir, "sbom.json")

	sbom := map[string]interface{}{
		"bomFormat":   "CycloneDX",
		"specVersion": "1.5",
		"version":     1,
		"metadata": map[string]interface{}{
			"timestamp": "2024-01-15T12:00:00Z",
		},
		"components": []interface{}{
			map[string]interface{}{"name": "test", "version": "v1.0.0"},
		},
	}

	data, _ := json.MarshalIndent(sbom, "", "  ")
	os.WriteFile(sbomPath, data, 0644)

	hash, err := ComputeSBOMDigest(sbomPath, "sha256")
	if err != nil {
		t.Fatalf("ComputeSBOMDigest failed: %v", err)
	}

	if len(hash) != 64 {
		t.Errorf("Expected hash length 64, got %d", len(hash))
	}

	hash2, err := ComputeSBOMDigest(sbomPath, "sha256")
	if err != nil {
		t.Fatalf("Second ComputeSBOMDigest failed: %v", err)
	}

	if hash != hash2 {
		t.Errorf("Hash not deterministic: %s != %s", hash, hash2)
	}
}

func TestVerifyReproducible(t *testing.T) {
	tempDir := t.TempDir()

	sbom := map[string]interface{}{
		"bomFormat":   "CycloneDX",
		"specVersion": "1.5",
		"version":     1,
		"metadata": map[string]interface{}{
			"timestamp": "2024-01-15T12:00:00Z",
		},
		"components": []interface{}{
			map[string]interface{}{"name": "test", "version": "v1.0.0"},
		},
	}

	sbom1Path := filepath.Join(tempDir, "sbom1.json")
	sbom2Path := filepath.Join(tempDir, "sbom2.json")

	data, _ := json.MarshalIndent(sbom, "", "  ")
	os.WriteFile(sbom1Path, data, 0644)
	os.WriteFile(sbom2Path, data, 0644)

	match, diff, err := VerifyReproducible(sbom1Path, sbom2Path)
	if err != nil {
		t.Fatalf("VerifyReproducible failed: %v", err)
	}

	if !match {
		t.Errorf("SBOMs should match but don't. Diff: %s", diff)
	}

	sbom["components"] = []interface{}{
		map[string]interface{}{"name": "different", "version": "v2.0.0"},
	}
	data, _ = json.MarshalIndent(sbom, "", "  ")
	os.WriteFile(sbom2Path, data, 0644)

	match, diff, err = VerifyReproducible(sbom1Path, sbom2Path)
	if err != nil {
		t.Fatalf("VerifyReproducible failed: %v", err)
	}

	if match {
		t.Error("SBOMs should not match but do")
	}

	if diff == "" {
		t.Error("Expected diff output but got empty string")
	}
}

func TestNormalizeSBOM(t *testing.T) {
	sbom := map[string]interface{}{
		"bomFormat": "CycloneDX",
		"metadata": map[string]interface{}{
			"timestamp": "2024-01-15T12:00:00Z",
			"tool":      "goenv",
		},
		"components": []interface{}{
			map[string]interface{}{"name": "zeta"},
			map[string]interface{}{"name": "alpha"},
		},
	}

	normalized := normalizeSBOM(sbom)

	metadata, ok := normalized["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("Normalized metadata not a map")
	}

	if _, hasTimestamp := metadata["timestamp"]; hasTimestamp {
		t.Error("Timestamp should be removed from normalized SBOM")
	}

	if tool, ok := metadata["tool"].(string); !ok || tool != "goenv" {
		t.Error("Other metadata fields should be preserved")
	}

	components, ok := normalized["components"].([]interface{})
	if !ok {
		t.Fatal("Normalized components not an array")
	}

	if len(components) != 2 {
		t.Fatalf("Expected 2 components, got %d", len(components))
	}

	firstComp := components[0].(map[string]interface{})
	firstName, _ := firstComp["name"].(string)
	if firstName != "alpha" {
		t.Errorf("First component should be 'alpha', got '%s'", firstName)
	}
}

func TestGenerateDeterministicUUID(t *testing.T) {
	uuid1 := generateDeterministicUUID("test-content")
	uuid2 := generateDeterministicUUID("test-content")

	if uuid1 != uuid2 {
		t.Errorf("UUIDs should match for same content: %s != %s", uuid1, uuid2)
	}

	uuid3 := generateDeterministicUUID("different-content")
	if uuid1 == uuid3 {
		t.Error("UUIDs should differ for different content")
	}

	if len(uuid1) < 32 {
		t.Errorf("UUID too short: %s", uuid1)
	}
}

func TestStdlibDetection(t *testing.T) {
	tempDir := t.TempDir()

	// Create a simple Go file with stdlib imports
	goFile := `package main

import (
	"fmt"
	"os"
	"encoding/json"
	"github.com/external/package"
)

func main() {
	fmt.Println("test")
}
`
	if err := os.WriteFile(filepath.Join(tempDir, "main.go"), []byte(goFile), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create test SBOM
	sbomPath := filepath.Join(tempDir, "sbom.json")
	sbom := map[string]interface{}{
		"bomFormat":   "CycloneDX",
		"specVersion": "1.5",
		"version":     1,
		"components":  []interface{}{},
	}
	data, _ := json.MarshalIndent(sbom, "", "  ")
	os.WriteFile(sbomPath, data, 0644)

	// Test stdlib import discovery directly
	enhancer := &Enhancer{
		config:  &config.Config{Root: tempDir},
		manager: &manager.Manager{},
	}

	stdlibImports, err := enhancer.discoverStdlibImports(tempDir)
	if err != nil {
		t.Fatalf("discoverStdlibImports failed: %v", err)
	}

	if len(stdlibImports) == 0 {
		t.Fatal("Expected stdlib imports but got none")
	}

	// Check expected stdlib packages
	expected := []string{"fmt", "os", "encoding/json"}
	for _, exp := range expected {
		found := false
		for _, imp := range stdlibImports {
			if imp == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected stdlib package %s not found", exp)
		}
	}

	// Ensure external package is NOT detected as stdlib
	for _, imp := range stdlibImports {
		if strings.Contains(imp, "github.com") {
			t.Errorf("External package %s incorrectly identified as stdlib", imp)
		}
	}
}

func TestBuildConstraintAnalysis(t *testing.T) {
	tempDir := t.TempDir()

	// Create Go files with various build constraints
	tests := []struct {
		name       string
		filename   string
		content    string
		activeTags []string
		shouldSee  []string
	}{
		{
			name:       "go:build with linux",
			filename:   "linux.go",
			content:    "//go:build linux\n\npackage main\n",
			activeTags: []string{"linux"},
			shouldSee:  []string{"linux"},
		},
		{
			name:       "go:build with AND",
			filename:   "linux_cgo.go",
			content:    "//go:build linux && cgo\n\npackage main\n",
			activeTags: []string{"linux", "cgo"},
			shouldSee:  []string{"linux && cgo"},
		},
		{
			name:       "legacy +build",
			filename:   "darwin.go",
			content:    "// +build darwin\n\npackage main\n",
			activeTags: []string{"darwin"},
			shouldSee:  []string{"// +build darwin"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir := filepath.Join(tempDir, sanitizeTestName(tt.name))
			os.MkdirAll(testDir, 0755)

			if err := os.WriteFile(filepath.Join(testDir, tt.filename), []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			enhancer := &Enhancer{
				config:  &config.Config{Root: testDir},
				manager: &manager.Manager{},
			}

			constraints, excluded, err := enhancer.analyzeBuildConstraints(testDir, tt.activeTags)
			if err != nil {
				t.Fatalf("analyzeBuildConstraints failed: %v", err)
			}

			// Check that expected constraints are found
			for _, expected := range tt.shouldSee {
				found := false
				for _, constraint := range constraints {
					if strings.Contains(constraint, expected) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected constraint %q not found in: %v", expected, constraints)
				}
			}

			_ = excluded // We're primarily testing constraint detection here
		})
	}
}

func TestEvaluateConstraint(t *testing.T) {
	enhancer := &Enhancer{
		config:  &config.Config{},
		manager: &manager.Manager{},
	}

	tests := []struct {
		name       string
		constraint string
		tags       map[string]bool
		expected   bool
	}{
		{
			name:       "simple match",
			constraint: "linux",
			tags:       map[string]bool{"linux": true},
			expected:   true,
		},
		{
			name:       "simple no match",
			constraint: "windows",
			tags:       map[string]bool{"linux": true},
			expected:   false,
		},
		{
			name:       "AND both true",
			constraint: "linux && cgo",
			tags:       map[string]bool{"linux": true, "cgo": true},
			expected:   true,
		},
		{
			name:       "AND one false",
			constraint: "linux && cgo",
			tags:       map[string]bool{"linux": true, "cgo": false},
			expected:   false,
		},
		{
			name:       "OR one true",
			constraint: "linux || darwin",
			tags:       map[string]bool{"linux": true, "darwin": false},
			expected:   true,
		},
		{
			name:       "OR both false",
			constraint: "linux || darwin",
			tags:       map[string]bool{"windows": true},
			expected:   false,
		},
		{
			name:       "NOT true",
			constraint: "!cgo",
			tags:       map[string]bool{"cgo": false},
			expected:   true,
		},
		{
			name:       "NOT false",
			constraint: "!cgo",
			tags:       map[string]bool{"cgo": true},
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := enhancer.evaluateConstraint(tt.constraint, tt.tags)
			if result != tt.expected {
				t.Errorf("evaluateConstraint(%q, %v) = %v, expected %v",
					tt.constraint, tt.tags, result, tt.expected)
			}
		})
	}
}

func TestEvaluateLegacyConstraint(t *testing.T) {
	enhancer := &Enhancer{
		config:  &config.Config{},
		manager: &manager.Manager{},
	}

	tests := []struct {
		name       string
		constraint string
		tags       map[string]bool
		expected   bool
	}{
		{
			name:       "simple match",
			constraint: "linux",
			tags:       map[string]bool{"linux": true},
			expected:   true,
		},
		{
			name:       "AND with comma",
			constraint: "linux,cgo",
			tags:       map[string]bool{"linux": true, "cgo": true},
			expected:   true,
		},
		{
			name:       "OR with space",
			constraint: "linux darwin",
			tags:       map[string]bool{"darwin": true},
			expected:   true,
		},
		{
			name:       "NOT",
			constraint: "!cgo",
			tags:       map[string]bool{"cgo": false},
			expected:   true,
		},
		{
			name:       "complex: (linux AND NOT cgo) OR darwin",
			constraint: "linux,!cgo darwin",
			tags:       map[string]bool{"linux": true, "cgo": false},
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := enhancer.evaluateLegacyConstraint(tt.constraint, tt.tags)
			if result != tt.expected {
				t.Errorf("evaluateLegacyConstraint(%q, %v) = %v, expected %v",
					tt.constraint, tt.tags, result, tt.expected)
			}
		})
	}
}

func TestRetractedVersionDetection(t *testing.T) {
	tempDir := t.TempDir()

	// Create a go.mod with retract directives
	goMod := `module testmodule

go 1.23

require (
	github.com/example/lib v1.2.3
)

retract (
	v1.0.0 // Bug in this version
	v1.1.0 // Security issue
)
`
	modPath := filepath.Join(tempDir, "go.mod")
	if err := os.WriteFile(modPath, []byte(goMod), 0644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	enhancer := &Enhancer{
		config:  &config.Config{Root: tempDir},
		manager: &manager.Manager{},
	}

	// Test checkLocalRetraction
	if !enhancer.checkLocalRetraction([]byte(goMod), "v1.0.0") {
		t.Error("Expected v1.0.0 to be retracted")
	}

	if !enhancer.checkLocalRetraction([]byte(goMod), "v1.1.0") {
		t.Error("Expected v1.1.0 to be retracted")
	}

	if enhancer.checkLocalRetraction([]byte(goMod), "v1.2.3") {
		t.Error("Expected v1.2.3 NOT to be retracted")
	}

	// Test markRetractedVersions
	// Note: markRetractedVersions only marks components that appear in go.mod requires
	// For this test, we'll just verify the function runs without error
	components := []interface{}{
		map[string]interface{}{
			"name":    "github.com/example/lib",
			"version": "v1.2.3",
		},
	}

	err := enhancer.markRetractedVersions(components, tempDir)
	if err != nil {
		t.Fatalf("markRetractedVersions failed: %v", err)
	}

	// The component should have been processed (requires map includes it)
	// In a real scenario with actual retraction from module proxy, it would be marked
}

func TestParseReplaceDirectives(t *testing.T) {
	tempDir := t.TempDir()

	goMod := `module testmodule

go 1.23

require (
	github.com/example/lib v1.2.3
)

replace github.com/example/lib => ../local-fork
replace github.com/other/lib v1.0.0 => github.com/fork/lib v1.1.0
`
	modPath := filepath.Join(tempDir, "go.mod")
	if err := os.WriteFile(modPath, []byte(goMod), 0644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	enhancer := &Enhancer{
		config:  &config.Config{Root: tempDir},
		manager: &manager.Manager{},
	}

	directives, err := enhancer.parseReplaceDirectives(tempDir)
	if err != nil {
		t.Fatalf("parseReplaceDirectives failed: %v", err)
	}

	if len(directives) != 2 {
		t.Fatalf("Expected 2 replace directives, got %d", len(directives))
	}

	// Check first directive (local path)
	if directives[0].Type != "local-path" {
		t.Errorf("Expected type 'local-path', got %s", directives[0].Type)
	}
	if directives[0].RiskLevel != "high" {
		t.Errorf("Expected risk level 'high', got %s", directives[0].RiskLevel)
	}

	// Check second directive (fork)
	if directives[1].Type != "fork" {
		t.Errorf("Expected type 'fork', got %s", directives[1].Type)
	}
	if directives[1].RiskLevel != "medium" {
		t.Errorf("Expected risk level 'medium', got %s", directives[1].RiskLevel)
	}
}
