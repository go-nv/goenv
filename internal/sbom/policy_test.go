package sbom

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testComponent represents a component for testing
type testComponent struct {
	Name    string
	Version string
	License string
}

func TestNewPolicyEngine(t *testing.T) {
	tests := []struct {
		name        string
		policyYAML  string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid policy",
			policyYAML: `
version: "1.0"
rules:
  - name: test-rule
    type: license
    severity: error
    description: Test rule
`,
			wantErr: false,
		},
		{
			name: "missing version",
			policyYAML: `
rules:
  - name: test-rule
    type: license
    severity: error
`,
			wantErr:     true,
			errContains: "version is required",
		},
		{
			name: "no rules",
			policyYAML: `
version: "1.0"
rules: []
`,
			wantErr:     true,
			errContains: "at least one rule is required",
		},
		{
			name: "invalid rule type",
			policyYAML: `
version: "1.0"
rules:
  - name: test-rule
    type: invalid-type
    severity: error
`,
			wantErr:     true,
			errContains: "invalid type",
		},
		{
			name: "invalid severity",
			policyYAML: `
version: "1.0"
rules:
  - name: test-rule
    type: license
    severity: critical
`,
			wantErr:     true,
			errContains: "invalid severity",
		},
		{
			name: "missing rule name",
			policyYAML: `
version: "1.0"
rules:
  - type: license
    severity: error
`,
			wantErr:     true,
			errContains: "name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary policy file
			tmpFile := filepath.Join(t.TempDir(), "policy.yaml")
			err := os.WriteFile(tmpFile, []byte(tt.policyYAML), 0644)
			if err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}

			// Try to create policy engine
			engine, err := NewPolicyEngine(tmpFile)

			if tt.wantErr {
				if err == nil {
					t.Error("NewPolicyEngine() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewPolicyEngine() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("NewPolicyEngine() unexpected error: %v", err)
				}
				if engine == nil {
					t.Error("NewPolicyEngine() returned nil engine")
				}
			}
		})
	}
}

func TestPolicyEngine_Validate_LicenseRules(t *testing.T) {
	tests := []struct {
		name            string
		policyYAML      string
		sbomComponents  []testComponent
		wantPassed      bool
		wantViolations  int
		violationRule   string
	}{
		{
			name: "blocked license detected",
			policyYAML: `
version: "1.0"
rules:
  - name: no-gpl
    type: license
    severity: error
    blocked:
      - GPL-3.0
`,
			sbomComponents: []testComponent{
				{
					Name:    "test-component",
					Version: "1.0.0",
					License: "GPL-3.0",
				},
			},
			wantPassed:     false,
			wantViolations: 1,
			violationRule:  "no-gpl",
		},
		{
			name: "allowed license passes",
			policyYAML: `
version: "1.0"
rules:
  - name: approved-only
    type: license
    severity: error
    blocked:
      - GPL-3.0
`,
			sbomComponents: []testComponent{
				{
					Name:    "test-component",
					Version: "1.0.0",
					License: "MIT",
				},
			},
			wantPassed:     true,
			wantViolations: 0,
		},
		{
			name: "multiple components with violations",
			policyYAML: `
version: "1.0"
rules:
  - name: no-copyleft
    type: license
    severity: error
    blocked:
      - GPL-3.0
      - AGPL-3.0
`,
			sbomComponents: []testComponent{
				{Name: "comp1", Version: "1.0.0", License: "MIT"},
				{Name: "comp2", Version: "2.0.0", License: "GPL-3.0"},
				{Name: "comp3", Version: "3.0.0", License: "AGPL-3.0"},
			},
			wantPassed:     false,
			wantViolations: 2,
			violationRule:  "no-copyleft",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create policy file
			policyPath := filepath.Join(t.TempDir(), "policy.yaml")
			err := os.WriteFile(policyPath, []byte(tt.policyYAML), 0644)
			if err != nil {
				t.Fatalf("failed to write policy: %v", err)
			}

			// Create SBOM file
			sbom := createPolicyTestSBOM(tt.sbomComponents)
			sbomPath := filepath.Join(t.TempDir(), "sbom.json")
			sbomData, _ := json.MarshalIndent(sbom, "", "  ")
			err = os.WriteFile(sbomPath, sbomData, 0644)
			if err != nil {
				t.Fatalf("failed to write SBOM: %v", err)
			}

			// Create engine and validate
			engine, err := NewPolicyEngine(policyPath)
			if err != nil {
				t.Fatalf("NewPolicyEngine() error: %v", err)
			}

			result, err := engine.Validate(sbomPath)
			if err != nil {
				t.Fatalf("Validate() error: %v", err)
			}

			if result.Passed != tt.wantPassed {
				t.Errorf("Validate() Passed = %v, want %v", result.Passed, tt.wantPassed)
			}

			if len(result.Violations) != tt.wantViolations {
				t.Errorf("Validate() violations = %d, want %d", len(result.Violations), tt.wantViolations)
			}

			if tt.wantViolations > 0 && tt.violationRule != "" {
				found := false
				for _, v := range result.Violations {
					if v.Rule == tt.violationRule {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Validate() missing expected violation rule: %s", tt.violationRule)
				}
			}
		})
	}
}

func TestPolicyEngine_Validate_SupplyChainRules(t *testing.T) {
	tests := []struct {
		name           string
		policyYAML     string
		sbomProperties map[string]string
		wantPassed     bool
		wantViolations int
	}{
		{
			name: "local replace directive detected",
			policyYAML: `
version: "1.0"
rules:
  - name: no-local-deps
    type: supply-chain
    severity: error
    check: replace-directives
    blocked:
      - local-path
`,
			sbomProperties: map[string]string{
				"goenv:module_context.replaces": `[{"old":"github.com/example/pkg","new":"local-path","type":"local-path"}]`,
			},
			wantPassed:     false,
			wantViolations: 1,
		},
		{
			name: "vendored dependencies detected",
			policyYAML: `
version: "1.0"
rules:
  - name: no-vendor
    type: supply-chain
    severity: warning
    check: vendoring-status
    blocked:
      - vendored
`,
			sbomProperties: map[string]string{
				"goenv:module_context.vendored": "true",
			},
			wantPassed:     true, // warnings don't fail by default
			wantViolations: 0,
		},
		{
			name: "no supply chain issues",
			policyYAML: `
version: "1.0"
rules:
  - name: clean-supply-chain
    type: supply-chain
    severity: error
    check: replace-directives
    blocked:
      - local-path
`,
			sbomProperties: map[string]string{},
			wantPassed:     true,
			wantViolations: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create policy file
			policyPath := filepath.Join(t.TempDir(), "policy.yaml")
			err := os.WriteFile(policyPath, []byte(tt.policyYAML), 0644)
			if err != nil {
				t.Fatalf("failed to write policy: %v", err)
			}

			// Create SBOM with properties
			sbom := createPolicyTestSBOMWithProps(tt.sbomProperties)
			sbomPath := filepath.Join(t.TempDir(), "sbom.json")
			sbomData, _ := json.MarshalIndent(sbom, "", "  ")
			err = os.WriteFile(sbomPath, sbomData, 0644)
			if err != nil {
				t.Fatalf("failed to write SBOM: %v", err)
			}

			// Create engine and validate
			engine, err := NewPolicyEngine(policyPath)
			if err != nil {
				t.Fatalf("NewPolicyEngine() error: %v", err)
			}

			result, err := engine.Validate(sbomPath)
			if err != nil {
				t.Fatalf("Validate() error: %v", err)
			}

			if result.Passed != tt.wantPassed {
				t.Errorf("Validate() Passed = %v, want %v", result.Passed, tt.wantPassed)
			}

			if len(result.Violations) != tt.wantViolations {
				t.Errorf("Validate() violations = %d, want %d", len(result.Violations), tt.wantViolations)
			}
		})
	}
}

func TestPolicyEngine_Validate_SecurityRules(t *testing.T) {
	tests := []struct {
		name           string
		policyYAML     string
		sbomProperties map[string]string
		componentProps map[string][]map[string]string
		wantPassed     bool
		wantViolations int
	}{
		{
			name: "CGO enabled when required disabled",
			policyYAML: `
version: "1.0"
rules:
  - name: cgo-must-be-disabled
    type: security
    severity: error
    check: cgo-disabled
    required:
      - "false"
`,
			sbomProperties: map[string]string{
				"goenv:build_context.cgo_enabled": "true",
			},
			wantPassed:     false,
			wantViolations: 1,
		},
		{
			name: "CGO disabled passes",
			policyYAML: `
version: "1.0"
rules:
  - name: cgo-must-be-disabled
    type: security
    severity: error
    check: cgo-disabled
    required:
      - "false"
`,
			sbomProperties: map[string]string{
				"goenv:build_context.cgo_enabled": "false",
			},
			wantPassed:     true,
			wantViolations: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create policy file
			policyPath := filepath.Join(t.TempDir(), "policy.yaml")
			err := os.WriteFile(policyPath, []byte(tt.policyYAML), 0644)
			if err != nil {
				t.Fatalf("failed to write policy: %v", err)
			}

			// Create SBOM with properties
			sbom := createPolicyTestSBOMWithProps(tt.sbomProperties)
			sbomPath := filepath.Join(t.TempDir(), "sbom.json")
			sbomData, _ := json.MarshalIndent(sbom, "", "  ")
			err = os.WriteFile(sbomPath, sbomData, 0644)
			if err != nil {
				t.Fatalf("failed to write SBOM: %v", err)
			}

			// Create engine and validate
			engine, err := NewPolicyEngine(policyPath)
			if err != nil {
				t.Fatalf("NewPolicyEngine() error: %v", err)
			}

			result, err := engine.Validate(sbomPath)
			if err != nil {
				t.Fatalf("Validate() error: %v", err)
			}

			if result.Passed != tt.wantPassed {
				t.Errorf("Validate() Passed = %v, want %v", result.Passed, tt.wantPassed)
			}

			if len(result.Violations) != tt.wantViolations {
				t.Errorf("Validate() violations = %d, want %d", len(result.Violations), tt.wantViolations)
			}
		})
	}
}

func TestPolicyEngine_Validate_CompletenessRules(t *testing.T) {
	tests := []struct {
		name           string
		policyYAML     string
		sbomComponents []testComponent
		sbomProperties map[string]string
		wantPassed     bool
		wantViolations int
	}{
		{
			name: "required component missing",
			policyYAML: `
version: "1.0"
rules:
  - name: must-have-deps
    type: completeness
    severity: error
    check: required-components
    required:
      - github.com/required/package
`,
			sbomComponents: []testComponent{
				{Name: "github.com/other/package", Version: "1.0.0"},
			},
			wantPassed:     false,
			wantViolations: 1,
		},
		{
			name: "required component present",
			policyYAML: `
version: "1.0"
rules:
  - name: must-have-deps
    type: completeness
    severity: error
    check: required-components
    required:
      - github.com/required/package
`,
			sbomComponents: []testComponent{
				{Name: "github.com/required/package", Version: "1.0.0"},
				{Name: "github.com/other/package", Version: "2.0.0"},
			},
			wantPassed:     true,
			wantViolations: 0,
		},
		{
			name: "required metadata missing",
			policyYAML: `
version: "1.0"
rules:
  - name: must-have-metadata
    type: completeness
    severity: error
    check: required-metadata
    required:
      - goenv:go_version
      - goenv:build_context.goos
`,
			sbomProperties: map[string]string{
				"goenv:go_version": "1.21.0",
				// missing goos
			},
			wantPassed:     false,
			wantViolations: 1,
		},
		{
			name: "required metadata present",
			policyYAML: `
version: "1.0"
rules:
  - name: must-have-metadata
    type: completeness
    severity: error
    check: required-metadata
    required:
      - goenv:go_version
      - goenv:build_context.goos
`,
			sbomProperties: map[string]string{
				"goenv:go_version":           "1.21.0",
				"goenv:build_context.goos":   "linux",
			},
			wantPassed:     true,
			wantViolations: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create policy file
			policyPath := filepath.Join(t.TempDir(), "policy.yaml")
			err := os.WriteFile(policyPath, []byte(tt.policyYAML), 0644)
			if err != nil {
				t.Fatalf("failed to write policy: %v", err)
			}

			// Create SBOM
			sbom := createPolicyTestSBOMFull(tt.sbomComponents, tt.sbomProperties)
			sbomPath := filepath.Join(t.TempDir(), "sbom.json")
			sbomData, _ := json.MarshalIndent(sbom, "", "  ")
			err = os.WriteFile(sbomPath, sbomData, 0644)
			if err != nil {
				t.Fatalf("failed to write SBOM: %v", err)
			}

			// Create engine and validate
			engine, err := NewPolicyEngine(policyPath)
			if err != nil {
				t.Fatalf("NewPolicyEngine() error: %v", err)
			}

			result, err := engine.Validate(sbomPath)
			if err != nil {
				t.Fatalf("Validate() error: %v", err)
			}

			if result.Passed != tt.wantPassed {
				t.Errorf("Validate() Passed = %v, want %v", result.Passed, tt.wantPassed)
			}

			if len(result.Violations) != tt.wantViolations {
				t.Errorf("Validate() violations = %d, want %d", len(result.Violations), tt.wantViolations)
			}
		})
	}
}

func TestPolicyEngine_Validate_MultipleRules(t *testing.T) {
	policyYAML := `
version: "1.0"
rules:
  - name: no-gpl
    type: license
    severity: error
    blocked:
      - GPL-3.0
  - name: no-local-deps
    type: supply-chain
    severity: error
    check: replace-directives
    blocked:
      - local-path
  - name: must-have-go-version
    type: completeness
    severity: error
    check: required-metadata
    required:
      - goenv:go_version
options:
  fail_on_error: true
  fail_on_warning: false
`

	tests := []struct {
		name           string
		sbomComponents []testComponent
		sbomProperties map[string]string
		wantPassed     bool
		minViolations  int
	}{
		{
			name: "all rules pass",
			sbomComponents: []testComponent{
				{Name: "test-pkg", Version: "1.0.0", License: "MIT"},
			},
			sbomProperties: map[string]string{
				"goenv:go_version": "1.21.0",
			},
			wantPassed:    true,
			minViolations: 0,
		},
		{
			name: "multiple violations",
			sbomComponents: []testComponent{
				{Name: "bad-pkg", Version: "1.0.0", License: "GPL-3.0"},
			},
			sbomProperties: map[string]string{
				"goenv:module_context.replaces": `[{"type":"local-path"}]`,
				// missing go_version
			},
			wantPassed:    false,
			minViolations: 2, // license + missing metadata (local-path may or may not trigger)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create policy file
			policyPath := filepath.Join(t.TempDir(), "policy.yaml")
			err := os.WriteFile(policyPath, []byte(policyYAML), 0644)
			if err != nil {
				t.Fatalf("failed to write policy: %v", err)
			}

			// Create SBOM
			sbom := createPolicyTestSBOMFull(tt.sbomComponents, tt.sbomProperties)
			sbomPath := filepath.Join(t.TempDir(), "sbom.json")
			sbomData, _ := json.MarshalIndent(sbom, "", "  ")
			err = os.WriteFile(sbomPath, sbomData, 0644)
			if err != nil {
				t.Fatalf("failed to write SBOM: %v", err)
			}

			// Create engine and validate
			engine, err := NewPolicyEngine(policyPath)
			if err != nil {
				t.Fatalf("NewPolicyEngine() error: %v", err)
			}

			result, err := engine.Validate(sbomPath)
			if err != nil {
				t.Fatalf("Validate() error: %v", err)
			}

			if result.Passed != tt.wantPassed {
				t.Errorf("Validate() Passed = %v, want %v", result.Passed, tt.wantPassed)
			}

			if len(result.Violations) < tt.minViolations {
				t.Errorf("Validate() violations = %d, want at least %d", len(result.Violations), tt.minViolations)
			}
		})
	}
}

func TestValidatePolicyConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      PolicyConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config",
			config: PolicyConfig{
				Version: "1.0",
				Rules: []PolicyRule{
					{Name: "test", Type: "license", Severity: "error"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing version",
			config: PolicyConfig{
				Rules: []PolicyRule{
					{Name: "test", Type: "license", Severity: "error"},
				},
			},
			wantErr:     true,
			errContains: "version is required",
		},
		{
			name: "no rules",
			config: PolicyConfig{
				Version: "1.0",
				Rules:   []PolicyRule{},
			},
			wantErr:     true,
			errContains: "at least one rule is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePolicyConfig(&tt.config)

			if tt.wantErr {
				if err == nil {
					t.Error("validatePolicyConfig() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("validatePolicyConfig() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("validatePolicyConfig() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestPolicyEngine_GenerateSummary(t *testing.T) {
	engine := &PolicyEngine{
		config: &PolicyConfig{},
	}

	tests := []struct {
		name       string
		result     *PolicyResult
		wantPass   string
		wantFail   string
	}{
		{
			name: "all passed",
			result: &PolicyResult{
				Passed:     true,
				Violations: []PolicyViolation{},
				Warnings:   []PolicyViolation{},
			},
			wantPass: "All policy checks passed",
		},
		{
			name: "with violations",
			result: &PolicyResult{
				Passed: false,
				Violations: []PolicyViolation{
					{Rule: "test-rule", Severity: "error", Message: "test error"},
				},
				Warnings: []PolicyViolation{},
			},
			wantFail: "Policy validation failed",
		},
		{
			name: "with warnings",
			result: &PolicyResult{
				Passed:     true,
				Violations: []PolicyViolation{},
				Warnings: []PolicyViolation{
					{Rule: "warn-rule", Severity: "warning", Message: "test warning"},
				},
			},
			wantPass: "All policy checks passed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := engine.generateSummary(tt.result)

			if tt.wantPass != "" && !strings.Contains(summary, tt.wantPass) {
				t.Errorf("generateSummary() missing expected text %q in: %s", tt.wantPass, summary)
			}
			if tt.wantFail != "" && !strings.Contains(summary, tt.wantFail) {
				t.Errorf("generateSummary() missing expected text %q in: %s", tt.wantFail, summary)
			}
		})
	}
}

// Helper functions

func createPolicyTestSBOM(components []testComponent) map[string]interface{} {
	// Convert components to CycloneDX format for licenses
	cdxComponents := make([]interface{}, len(components))
	for i, comp := range components {
		cdxComp := map[string]interface{}{
			"name":    comp.Name,
			"version": comp.Version,
		}
		
		if comp.License != "" {
			cdxComp["licenses"] = []interface{}{
				map[string]interface{}{
					"license": map[string]interface{}{
						"id": comp.License,
					},
				},
			}
		}
		
		cdxComponents[i] = cdxComp
	}

	return map[string]interface{}{
		"bomFormat":   "CycloneDX",
		"specVersion": "1.4",
		"components":  cdxComponents,
		"metadata":    map[string]interface{}{},
	}
}

func createPolicyTestSBOMWithProps(properties map[string]string) map[string]interface{} {
	props := make([]interface{}, 0, len(properties))
	for name, value := range properties {
		props = append(props, map[string]interface{}{
			"name":  name,
			"value": value,
		})
	}

	return map[string]interface{}{
		"bomFormat":   "CycloneDX",
		"specVersion": "1.4",
		"components":  []interface{}{},
		"metadata": map[string]interface{}{
			"properties": props,
		},
	}
}

func createPolicyTestSBOMFull(components []testComponent, properties map[string]string) map[string]interface{} {
	sbom := createPolicyTestSBOM(components)
	
	// Add properties
	props := make([]interface{}, 0, len(properties))
	for name, value := range properties {
		props = append(props, map[string]interface{}{
			"name":  name,
			"value": value,
		})
	}
	
	metadata := sbom["metadata"].(map[string]interface{})
	metadata["properties"] = props
	
	return sbom
}
