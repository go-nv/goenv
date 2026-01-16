package sbom

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/go-nv/goenv/internal/errors"
	"gopkg.in/yaml.v3"
)

// PolicyEngine validates SBOMs against defined policies
type PolicyEngine struct {
	config *PolicyConfig
}

// PolicyConfig defines the structure of policy configuration files
type PolicyConfig struct {
	Version string        `yaml:"version"`
	Rules   []PolicyRule  `yaml:"rules"`
	Options PolicyOptions `yaml:"options,omitempty"`
}

// PolicyRule defines a single validation rule
type PolicyRule struct {
	Name        string            `yaml:"name"`
	Type        string            `yaml:"type"` // supply-chain, security, completeness, license, custom
	Severity    string            `yaml:"severity"`
	Description string            `yaml:"description,omitempty"`
	Blocked     []string          `yaml:"blocked,omitempty"`
	Required    []string          `yaml:"required,omitempty"`
	Check       string            `yaml:"check,omitempty"`
	Condition   string            `yaml:"condition,omitempty"`
	Threshold   int               `yaml:"threshold,omitempty"`
	Patterns    []string          `yaml:"patterns,omitempty"`
	Metadata    map[string]string `yaml:"metadata,omitempty"`
}

// PolicyOptions configure how policies are enforced
type PolicyOptions struct {
	FailOnError   bool `yaml:"fail_on_error"`
	FailOnWarning bool `yaml:"fail_on_warning"`
	Verbose       bool `yaml:"verbose"`
}

// PolicyViolation represents a failed policy check
type PolicyViolation struct {
	Rule        string
	Severity    string
	Message     string
	Component   string
	Remediation string
}

// PolicyResult contains the results of policy validation
type PolicyResult struct {
	Passed     bool
	Violations []PolicyViolation
	Warnings   []PolicyViolation
	Summary    string
}

// NewPolicyEngine creates a new policy validation engine
func NewPolicyEngine(policyPath string) (*PolicyEngine, error) {
	data, err := os.ReadFile(policyPath)
	if err != nil {
		return nil, errors.FailedTo("read policy file", err)
	}

	var config PolicyConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, errors.FailedTo("parse policy YAML", err)
	}

	// Validate policy config
	if err := validatePolicyConfig(&config); err != nil {
		return nil, err
	}

	return &PolicyEngine{
		config: &config,
	}, nil
}

// validatePolicyConfig ensures policy configuration is valid
func validatePolicyConfig(config *PolicyConfig) error {
	if config.Version == "" {
		return fmt.Errorf("policy version is required")
	}

	if len(config.Rules) == 0 {
		return fmt.Errorf("at least one rule is required")
	}

	validTypes := map[string]bool{
		"supply-chain": true,
		"security":     true,
		"completeness": true,
		"license":      true,
		"custom":       true,
	}

	validSeverities := map[string]bool{
		"error":   true,
		"warning": true,
		"info":    true,
	}

	for i, rule := range config.Rules {
		if rule.Name == "" {
			return fmt.Errorf("rule %d: name is required", i)
		}
		if rule.Type == "" {
			return fmt.Errorf("rule %q: type is required", rule.Name)
		}
		if !validTypes[rule.Type] {
			return fmt.Errorf("rule %q: invalid type %q", rule.Name, rule.Type)
		}
		if rule.Severity == "" {
			return fmt.Errorf("rule %q: severity is required", rule.Name)
		}
		if !validSeverities[rule.Severity] {
			return fmt.Errorf("rule %q: invalid severity %q", rule.Name, rule.Severity)
		}
	}

	return nil
}

// Validate runs policy validation against an SBOM
func (pe *PolicyEngine) Validate(sbomPath string) (*PolicyResult, error) {
	// Parse SBOM
	sbom, err := parseSBOMFile(sbomPath)
	if err != nil {
		return nil, err
	}

	result := &PolicyResult{
		Passed:     true,
		Violations: []PolicyViolation{},
		Warnings:   []PolicyViolation{},
	}

	// Run each rule
	for _, rule := range pe.config.Rules {
		violations, err := pe.runRule(rule, sbom)
		if err != nil {
			return nil, fmt.Errorf("rule %q failed: %w", rule.Name, err)
		}

		for _, v := range violations {
			if v.Severity == "error" {
				result.Violations = append(result.Violations, v)
				result.Passed = false
			} else if v.Severity == "warning" {
				result.Warnings = append(result.Warnings, v)
				if pe.config.Options.FailOnWarning {
					result.Passed = false
				}
			}
		}
	}

	// Generate summary
	result.Summary = pe.generateSummary(result)

	return result, nil
}

// runRule executes a single policy rule
func (pe *PolicyEngine) runRule(rule PolicyRule, sbom map[string]interface{}) ([]PolicyViolation, error) {
	switch rule.Type {
	case "supply-chain":
		return pe.checkSupplyChain(rule, sbom)
	case "security":
		return pe.checkSecurity(rule, sbom)
	case "completeness":
		return pe.checkCompleteness(rule, sbom)
	case "license":
		return pe.checkLicense(rule, sbom)
	case "custom":
		return pe.checkCustom(rule, sbom)
	default:
		return nil, fmt.Errorf("unsupported rule type: %s", rule.Type)
	}
}

// checkSupplyChain validates supply chain security rules
func (pe *PolicyEngine) checkSupplyChain(rule PolicyRule, sbom map[string]interface{}) ([]PolicyViolation, error) {
	violations := []PolicyViolation{}

	// Check for blocked replace directives
	if rule.Check == "replace-directives" {
		metadata := extractMetadata(sbom)
		properties := extractProperties(metadata)

		for _, prop := range properties {
			name, ok := prop["name"].(string)
			if !ok {
				continue
			}

			if strings.HasPrefix(name, "goenv:module_context.replaces") {
				value, _ := prop["value"].(string)

				// Parse replace directives JSON
				if strings.Contains(value, "local-path") {
					for _, blocked := range rule.Blocked {
						if blocked == "local-path" {
							violations = append(violations, PolicyViolation{
								Rule:        rule.Name,
								Severity:    rule.Severity,
								Message:     "Local path replace directive detected",
								Component:   "module dependencies",
								Remediation: "Replace local dependencies with versioned module references",
							})
						}
					}
				}
			}
		}
	}

	// Check for vendored dependencies
	if rule.Check == "vendoring-status" {
		properties := extractProperties(extractMetadata(sbom))
		for _, prop := range properties {
			name, _ := prop["name"].(string)
			if name == "goenv:module_context.vendored" {
				value, _ := prop["value"].(string)

				for _, blocked := range rule.Blocked {
					if blocked == "vendored" && value == "true" {
						violations = append(violations, PolicyViolation{
							Rule:        rule.Name,
							Severity:    rule.Severity,
							Message:     "Vendored dependencies detected",
							Component:   "vendor directory",
							Remediation: "Remove vendor directory and use module proxy",
						})
					}
				}
			}
		}
	}

	return violations, nil
}

// checkSecurity validates security-related rules
func (pe *PolicyEngine) checkSecurity(rule PolicyRule, sbom map[string]interface{}) ([]PolicyViolation, error) {
	violations := []PolicyViolation{}

	// Check for retracted versions
	if rule.Check == "retracted-versions" {
		components := extractComponents(sbom)
		for _, comp := range components {
			compMap, ok := comp.(map[string]interface{})
			if !ok {
				continue
			}

			// Check component properties for retraction info
			if props, ok := compMap["properties"].([]interface{}); ok {
				for _, prop := range props {
					propMap, ok := prop.(map[string]interface{})
					if !ok {
						continue
					}

					name, _ := propMap["name"].(string)
					if strings.Contains(name, "retracted") {
						value, _ := propMap["value"].(string)
						if value == "true" {
							componentName, _ := compMap["name"].(string)
							violations = append(violations, PolicyViolation{
								Rule:        rule.Name,
								Severity:    rule.Severity,
								Message:     "Retracted module version in use",
								Component:   componentName,
								Remediation: "Update to non-retracted version",
							})
						}
					}
				}
			}
		}
	}

	// Check CGO status
	if rule.Check == "cgo-disabled" {
		properties := extractProperties(extractMetadata(sbom))
		for _, prop := range properties {
			name, _ := prop["name"].(string)
			if name == "goenv:build_context.cgo_enabled" {
				value, _ := prop["value"].(string)

				for _, required := range rule.Required {
					if required == "false" && value == "true" {
						violations = append(violations, PolicyViolation{
							Rule:        rule.Name,
							Severity:    rule.Severity,
							Message:     "CGO is enabled",
							Component:   "build configuration",
							Remediation: "Build with CGO_ENABLED=0 for better security",
						})
					}
				}
			}
		}
	}

	return violations, nil
}

// checkCompleteness validates SBOM completeness
func (pe *PolicyEngine) checkCompleteness(rule PolicyRule, sbom map[string]interface{}) ([]PolicyViolation, error) {
	violations := []PolicyViolation{}

	// Check for required components
	if rule.Check == "required-components" {
		components := extractComponents(sbom)
		componentNames := make(map[string]bool)

		for _, comp := range components {
			compMap, ok := comp.(map[string]interface{})
			if !ok {
				continue
			}
			name, _ := compMap["name"].(string)
			componentNames[name] = true
		}

		for _, required := range rule.Required {
			if !componentNames[required] {
				violations = append(violations, PolicyViolation{
					Rule:        rule.Name,
					Severity:    rule.Severity,
					Message:     fmt.Sprintf("Required component missing: %s", required),
					Component:   required,
					Remediation: "Ensure SBOM includes all required components",
				})
			}
		}
	}

	// Check for required metadata
	if rule.Check == "required-metadata" {
		properties := extractProperties(extractMetadata(sbom))
		propertyNames := make(map[string]bool)

		for _, prop := range properties {
			name, _ := prop["name"].(string)
			propertyNames[name] = true
		}

		for _, required := range rule.Required {
			if !propertyNames[required] {
				violations = append(violations, PolicyViolation{
					Rule:        rule.Name,
					Severity:    rule.Severity,
					Message:     fmt.Sprintf("Required metadata missing: %s", required),
					Component:   "metadata",
					Remediation: "Generate SBOM with --enhance flag",
				})
			}
		}
	}

	return violations, nil
}

// checkLicense validates license compliance
func (pe *PolicyEngine) checkLicense(rule PolicyRule, sbom map[string]interface{}) ([]PolicyViolation, error) {
	violations := []PolicyViolation{}

	components := extractComponents(sbom)
	for _, comp := range components {
		compMap, ok := comp.(map[string]interface{})
		if !ok {
			continue
		}

		// Check licenses field
		if licenses, ok := compMap["licenses"].([]interface{}); ok {
			for _, lic := range licenses {
				licMap, ok := lic.(map[string]interface{})
				if !ok {
					continue
				}

				licenseContent, ok := licMap["license"].(map[string]interface{})
				if !ok {
					continue
				}

				licenseID, _ := licenseContent["id"].(string)

				// Check against blocked licenses
				for _, blocked := range rule.Blocked {
					if licenseID == blocked {
						componentName, _ := compMap["name"].(string)
						violations = append(violations, PolicyViolation{
							Rule:        rule.Name,
							Severity:    rule.Severity,
							Message:     fmt.Sprintf("Blocked license detected: %s", licenseID),
							Component:   componentName,
							Remediation: fmt.Sprintf("Replace component with %s license", licenseID),
						})
					}
				}
			}
		}
	}

	return violations, nil
}

// checkCustom validates custom rules
func (pe *PolicyEngine) checkCustom(rule PolicyRule, sbom map[string]interface{}) ([]PolicyViolation, error) {
	// Custom rules are user-defined - would need scripting or expression evaluation
	// For now, return empty violations
	return []PolicyViolation{}, nil
}

// Helper functions

func extractComponents(sbom map[string]interface{}) []interface{} {
	if components, ok := sbom["components"].([]interface{}); ok {
		return components
	}
	return []interface{}{}
}

func extractMetadata(sbom map[string]interface{}) map[string]interface{} {
	if metadata, ok := sbom["metadata"].(map[string]interface{}); ok {
		return metadata
	}
	return map[string]interface{}{}
}

func extractProperties(metadata map[string]interface{}) []map[string]interface{} {
	properties := []map[string]interface{}{}
	if props, ok := metadata["properties"].([]interface{}); ok {
		for _, prop := range props {
			if propMap, ok := prop.(map[string]interface{}); ok {
				properties = append(properties, propMap)
			}
		}
	}
	return properties
}

func (pe *PolicyEngine) generateSummary(result *PolicyResult) string {
	var summary strings.Builder

	if result.Passed {
		summary.WriteString("✓ All policy checks passed\n")
	} else {
		summary.WriteString("✗ Policy validation failed\n")
	}

	if len(result.Violations) > 0 {
		summary.WriteString(fmt.Sprintf("\n%d violations found:\n", len(result.Violations)))
		for _, v := range result.Violations {
			summary.WriteString(fmt.Sprintf("  - [%s] %s: %s\n", v.Severity, v.Rule, v.Message))
		}
	}

	if len(result.Warnings) > 0 {
		summary.WriteString(fmt.Sprintf("\n%d warnings found:\n", len(result.Warnings)))
		for _, w := range result.Warnings {
			summary.WriteString(fmt.Sprintf("  - [%s] %s: %s\n", w.Severity, w.Rule, w.Message))
		}
	}

	return summary.String()
}

// parseSBOMFile reads and parses an SBOM JSON file
func parseSBOMFile(sbomPath string) (map[string]interface{}, error) {
	data, err := os.ReadFile(sbomPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SBOM file: %w", err)
	}

	var sbom map[string]interface{}
	if err := json.Unmarshal(data, &sbom); err != nil {
		return nil, fmt.Errorf("failed to parse SBOM JSON: %w", err)
	}

	return sbom, nil
}
