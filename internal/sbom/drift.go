package sbom

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DriftDetector manages baseline SBOMs and detects drift
type DriftDetector struct {
	BaselineDir string // Directory storing baseline SBOMs
}

// DriftResult represents the outcome of drift detection
type DriftResult struct {
	HasDrift     bool             `json:"has_drift"`
	DriftSummary DriftSummary     `json:"drift_summary"`
	Changes      *DiffResult      `json:"changes,omitempty"`
	Baseline     BaselineMeta     `json:"baseline"`
	Current      SBOMMeta         `json:"current"`
	DetectedAt   time.Time        `json:"detected_at"`
	Violations   []DriftViolation `json:"violations,omitempty"`
}

// DriftSummary provides high-level drift statistics
type DriftSummary struct {
	TotalChanges         int    `json:"total_changes"`
	UnexpectedAdded      int    `json:"unexpected_added"`
	UnexpectedRemoved    int    `json:"unexpected_removed"`
	UnexpectedUpgrades   int    `json:"unexpected_upgrades"`
	UnexpectedDowngrades int    `json:"unexpected_downgrades"`
	LicenseChanges       int    `json:"license_changes"`
	SeverityLevel        string `json:"severity_level"` // none, low, medium, high
}

// BaselineMeta describes the baseline SBOM
type BaselineMeta struct {
	Path           string    `json:"path"`
	CreatedAt      time.Time `json:"created_at"`
	Version        string    `json:"version,omitempty"`
	ComponentCount int       `json:"component_count"`
	Description    string    `json:"description,omitempty"`
}

// DriftViolation represents a specific drift violation
type DriftViolation struct {
	Type      string `json:"type"` // added, removed, upgrade, downgrade, license_change
	Component string `json:"component"`
	OldValue  string `json:"old_value,omitempty"`
	NewValue  string `json:"new_value,omitempty"`
	Severity  string `json:"severity"` // low, medium, high
	Message   string `json:"message"`
}

// DriftOptions configures drift detection behavior
type DriftOptions struct {
	AllowedAdditions    []string // Component patterns allowed to be added
	AllowedRemovals     []string // Component patterns allowed to be removed
	AllowUpgrades       bool     // Whether version upgrades are allowed
	AllowDowngrades     bool     // Whether version downgrades are allowed (usually false)
	AllowLicenseChanges bool     // Whether license changes are allowed
	StrictMode          bool     // Fail on any drift
}

// NewDriftDetector creates a new drift detector
func NewDriftDetector(baselineDir string) (*DriftDetector, error) {
	if baselineDir == "" {
		return nil, fmt.Errorf("baseline directory cannot be empty")
	}

	// Create baseline directory if it doesn't exist
	if err := os.MkdirAll(baselineDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create baseline directory: %w", err)
	}

	return &DriftDetector{
		BaselineDir: baselineDir,
	}, nil
}

// SaveBaseline saves a current SBOM as the baseline
func (d *DriftDetector) SaveBaseline(sbomPath, name, version, description string) error {
	if name == "" {
		name = "default"
	}

	// Load and validate the SBOM
	sbom, err := loadSBOMForDiff(sbomPath)
	if err != nil {
		return fmt.Errorf("failed to load SBOM: %w", err)
	}

	// Create baseline metadata
	baseline := struct {
		BaselineMeta
		Components []Component `json:"components"`
	}{
		BaselineMeta: BaselineMeta{
			Path:           sbomPath,
			CreatedAt:      time.Now(),
			Version:        version,
			ComponentCount: len(sbom.Components),
			Description:    description,
		},
		Components: sbom.Components,
	}

	// Save to baseline directory
	baselineFile := filepath.Join(d.BaselineDir, fmt.Sprintf("%s.baseline.json", name))
	data, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal baseline: %w", err)
	}

	if err := os.WriteFile(baselineFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write baseline file: %w", err)
	}

	return nil
}

// DetectDrift compares a current SBOM against a baseline
func (d *DriftDetector) DetectDrift(currentPath, baselineName string, options DriftOptions) (*DriftResult, error) {
	if baselineName == "" {
		baselineName = "default"
	}

	// Load baseline
	baselineFile := filepath.Join(d.BaselineDir, fmt.Sprintf("%s.baseline.json", baselineName))
	baselineData, err := os.ReadFile(baselineFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read baseline file: %w", err)
	}

	var baseline struct {
		BaselineMeta
		Components []Component `json:"components"`
	}
	if err := json.Unmarshal(baselineData, &baseline); err != nil {
		return nil, fmt.Errorf("failed to parse baseline: %w", err)
	}

	// Load current SBOM
	current, err := loadSBOMForDiff(currentPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load current SBOM: %w", err)
	}

	// Create temporary files for diff comparison
	baselineTmpFile, err := d.createTempSBOM(baseline.Components)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp baseline: %w", err)
	}
	defer os.Remove(baselineTmpFile)

	// Perform diff
	diffResult, err := DiffSBOMs(baselineTmpFile, currentPath, &DiffOptions{
		IgnoreLicenses: options.AllowLicenseChanges, // If we allow changes, ignore them in diff
	})
	if err != nil {
		return nil, fmt.Errorf("failed to diff SBOMs: %w", err)
	}

	// Analyze drift violations
	violations := d.analyzeViolations(diffResult, options)

	// Calculate drift summary
	summary := d.calculateDriftSummary(diffResult, violations, options)

	result := &DriftResult{
		HasDrift:     len(violations) > 0,
		DriftSummary: summary,
		Changes:      diffResult,
		Baseline: BaselineMeta{
			Path:           baselineFile,
			CreatedAt:      baseline.CreatedAt,
			Version:        baseline.Version,
			ComponentCount: baseline.ComponentCount,
			Description:    baseline.Description,
		},
		Current: SBOMMeta{
			Path:           currentPath,
			ComponentCount: len(current.Components),
		},
		DetectedAt: time.Now(),
		Violations: violations,
	}

	return result, nil
}

// ListBaselines returns all available baselines
func (d *DriftDetector) ListBaselines() ([]BaselineMeta, error) {
	entries, err := os.ReadDir(d.BaselineDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read baseline directory: %w", err)
	}

	var baselines []BaselineMeta
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".baseline.json") {
			continue
		}

		baselinePath := filepath.Join(d.BaselineDir, entry.Name())
		data, err := os.ReadFile(baselinePath)
		if err != nil {
			continue // Skip unreadable files
		}

		var baseline struct {
			BaselineMeta
		}
		if err := json.Unmarshal(data, &baseline); err != nil {
			continue // Skip invalid files
		}

		// Store the baseline file path, not the original SBOM path
		baseline.BaselineMeta.Path = baselinePath
		baselines = append(baselines, baseline.BaselineMeta)
	}

	return baselines, nil
}

// DeleteBaseline removes a baseline
func (d *DriftDetector) DeleteBaseline(name string) error {
	if name == "" {
		return fmt.Errorf("baseline name cannot be empty")
	}

	baselineFile := filepath.Join(d.BaselineDir, fmt.Sprintf("%s.baseline.json", name))
	if err := os.Remove(baselineFile); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("baseline not found: %s", name)
		}
		return fmt.Errorf("failed to delete baseline: %w", err)
	}

	return nil
}

// analyzeViolations identifies drift violations based on options
func (d *DriftDetector) analyzeViolations(diff *DiffResult, options DriftOptions) []DriftViolation {
	var violations []DriftViolation

	// Check additions
	for _, comp := range diff.Added {
		if !d.isAllowedChange(comp.Name, comp.Group, options.AllowedAdditions) {
			violations = append(violations, DriftViolation{
				Type:      "added",
				Component: formatComponentName(comp.Name, comp.Group),
				NewValue:  comp.Version,
				Severity:  "medium",
				Message:   fmt.Sprintf("Unexpected dependency added: %s v%s", formatComponentName(comp.Name, comp.Group), comp.Version),
			})
		}
	}

	// Check removals
	for _, comp := range diff.Removed {
		if !d.isAllowedChange(comp.Name, comp.Group, options.AllowedRemovals) {
			violations = append(violations, DriftViolation{
				Type:      "removed",
				Component: formatComponentName(comp.Name, comp.Group),
				OldValue:  comp.Version,
				Severity:  "high",
				Message:   fmt.Sprintf("Unexpected dependency removed: %s v%s", formatComponentName(comp.Name, comp.Group), comp.Version),
			})
		}
	}

	// Check modifications
	for _, comp := range diff.Modified {
		severity := comp.Severity

		// Check upgrades
		if severity == "upgrade" && !options.AllowUpgrades {
			violations = append(violations, DriftViolation{
				Type:      "upgrade",
				Component: formatComponentName(comp.Name, comp.Group),
				OldValue:  comp.OldVersion,
				NewValue:  comp.NewVersion,
				Severity:  "low",
				Message:   fmt.Sprintf("Unexpected version upgrade: %s from v%s to v%s", formatComponentName(comp.Name, comp.Group), comp.OldVersion, comp.NewVersion),
			})
		}

		// Check downgrades
		if severity == "downgrade" && !options.AllowDowngrades {
			violations = append(violations, DriftViolation{
				Type:      "downgrade",
				Component: formatComponentName(comp.Name, comp.Group),
				OldValue:  comp.OldVersion,
				NewValue:  comp.NewVersion,
				Severity:  "high",
				Message:   fmt.Sprintf("Unexpected version downgrade: %s from v%s to v%s", formatComponentName(comp.Name, comp.Group), comp.OldVersion, comp.NewVersion),
			})
		}

		// Check license changes
		if comp.OldLicense != "" && comp.NewLicense != "" && comp.OldLicense != comp.NewLicense && !options.AllowLicenseChanges {
			violations = append(violations, DriftViolation{
				Type:      "license_change",
				Component: formatComponentName(comp.Name, comp.Group),
				OldValue:  comp.OldLicense,
				NewValue:  comp.NewLicense,
				Severity:  "medium",
				Message:   fmt.Sprintf("License changed: %s from %s to %s", formatComponentName(comp.Name, comp.Group), comp.OldLicense, comp.NewLicense),
			})
		}
	}

	// In strict mode, any change is a violation
	if options.StrictMode && (len(diff.Added) > 0 || len(diff.Removed) > 0 || len(diff.Modified) > 0) {
		if len(violations) == 0 {
			violations = append(violations, DriftViolation{
				Type:     "drift",
				Severity: "high",
				Message:  "Strict mode enabled: any change from baseline is not allowed",
			})
		}
	}

	return violations
}

// calculateDriftSummary computes drift summary statistics
func (d *DriftDetector) calculateDriftSummary(diff *DiffResult, violations []DriftViolation, options DriftOptions) DriftSummary {
	summary := DriftSummary{
		TotalChanges: diff.Summary.AddedCount + diff.Summary.RemovedCount + diff.Summary.ModifiedCount,
	}

	// Count unexpected changes
	for _, v := range violations {
		switch v.Type {
		case "added":
			summary.UnexpectedAdded++
		case "removed":
			summary.UnexpectedRemoved++
		case "upgrade":
			summary.UnexpectedUpgrades++
		case "downgrade":
			summary.UnexpectedDowngrades++
		case "license_change":
			summary.LicenseChanges++
		}
	}

	// Determine severity level
	summary.SeverityLevel = d.determineSeverityLevel(violations)

	return summary
}

// determineSeverityLevel calculates overall severity
func (d *DriftDetector) determineSeverityLevel(violations []DriftViolation) string {
	if len(violations) == 0 {
		return "none"
	}

	hasHigh := false
	hasMedium := false

	for _, v := range violations {
		switch v.Severity {
		case "high":
			hasHigh = true
		case "medium":
			hasMedium = true
		}
	}

	if hasHigh {
		return "high"
	}
	if hasMedium {
		return "medium"
	}
	return "low"
}

// isAllowedChange checks if a component change matches allowed patterns
func (d *DriftDetector) isAllowedChange(name, group string, allowedPatterns []string) bool {
	if len(allowedPatterns) == 0 {
		return false
	}

	fullName := formatComponentName(name, group)

	for _, pattern := range allowedPatterns {
		// Simple pattern matching (supports * wildcard)
		if matchPattern(pattern, fullName) {
			return true
		}
	}

	return false
}

// matchPattern performs simple wildcard pattern matching
func matchPattern(pattern, str string) bool {
	if pattern == "*" {
		return true
	}

	// Simple prefix/suffix matching
	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
		return strings.Contains(str, strings.Trim(pattern, "*"))
	}
	if strings.HasPrefix(pattern, "*") {
		return strings.HasSuffix(str, strings.TrimPrefix(pattern, "*"))
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(str, strings.TrimSuffix(pattern, "*"))
	}

	return pattern == str
}

// formatComponentName creates a full component name
func formatComponentName(name, group string) string {
	if group != "" {
		return fmt.Sprintf("%s/%s", group, name)
	}
	return name
}

// createTempSBOM creates a temporary SBOM file from components
func (d *DriftDetector) createTempSBOM(components []Component) (string, error) {
	tmpFile, err := os.CreateTemp("", "baseline-*.json")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	// Create proper CycloneDX format
	type cdxLicense struct {
		License struct {
			ID string `json:"id,omitempty"`
		} `json:"license,omitempty"`
	}

	type cdxComponent struct {
		Name       string       `json:"name"`
		Group      string       `json:"group,omitempty"`
		Version    string       `json:"version"`
		PackageURL string       `json:"purl,omitempty"`
		Licenses   []cdxLicense `json:"licenses,omitempty"`
	}

	type cdxSBOM struct {
		BomFormat   string         `json:"bomFormat"`
		SpecVersion string         `json:"specVersion"`
		Components  []cdxComponent `json:"components"`
	}

	cdx := cdxSBOM{
		BomFormat:   "CycloneDX",
		SpecVersion: "1.4",
		Components:  make([]cdxComponent, 0, len(components)),
	}

	for _, comp := range components {
		cdxComp := cdxComponent{
			Name:       comp.Name,
			Group:      comp.Group,
			Version:    comp.Version,
			PackageURL: comp.PackageURL,
		}

		if comp.License != "" {
			cdxComp.Licenses = []cdxLicense{
				{
					License: struct {
						ID string `json:"id,omitempty"`
					}{
						ID: comp.License,
					},
				},
			}
		}

		cdx.Components = append(cdx.Components, cdxComp)
	}

	data, err := json.Marshal(cdx)
	if err != nil {
		return "", err
	}

	if _, err := tmpFile.Write(data); err != nil {
		return "", err
	}

	return tmpFile.Name(), nil
}
