package sbom

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// DiffResult represents the differences between two SBOMs
type DiffResult struct {
	Added      []ComponentDiff `json:"added"`
	Removed    []ComponentDiff `json:"removed"`
	Modified   []ComponentDiff `json:"modified"`
	Unchanged  []ComponentDiff `json:"unchanged,omitempty"`
	Summary    DiffSummary     `json:"summary"`
	Comparison ComparisonMeta  `json:"comparison"`
}

// ComponentDiff represents a change in a component
type ComponentDiff struct {
	Name       string   `json:"name"`
	Group      string   `json:"group,omitempty"`
	OldVersion string   `json:"old_version,omitempty"`
	NewVersion string   `json:"new_version,omitempty"`
	Version    string   `json:"version,omitempty"` // For added/removed
	ChangeType string   `json:"change_type"`       // "added", "removed", "version_change", "license_change", "unchanged"
	OldLicense string   `json:"old_license,omitempty"`
	NewLicense string   `json:"new_license,omitempty"`
	License    string   `json:"license,omitempty"`  // For added/removed
	Severity   string   `json:"severity,omitempty"` // For version changes (upgrade/downgrade)
	PackageURL string   `json:"purl,omitempty"`
	Changes    []string `json:"changes,omitempty"` // Human-readable change descriptions
}

// DiffSummary provides high-level statistics about the diff
type DiffSummary struct {
	TotalComponents   int `json:"total_components"`
	AddedCount        int `json:"added_count"`
	RemovedCount      int `json:"removed_count"`
	ModifiedCount     int `json:"modified_count"`
	UnchangedCount    int `json:"unchanged_count"`
	VersionUpgrades   int `json:"version_upgrades"`
	VersionDowngrades int `json:"version_downgrades"`
	LicenseChanges    int `json:"license_changes"`
}

// ComparisonMeta contains metadata about the comparison
type ComparisonMeta struct {
	OldSBOM SBOMMeta `json:"old_sbom"`
	NewSBOM SBOMMeta `json:"new_sbom"`
}

// SBOMMeta contains metadata about an SBOM
type SBOMMeta struct {
	Path           string `json:"path"`
	Format         string `json:"format"`
	SpecVersion    string `json:"spec_version,omitempty"`
	ComponentCount int    `json:"component_count"`
	Generated      string `json:"generated,omitempty"`
}

// DiffOptions controls the diff behavior
type DiffOptions struct {
	ShowUnchanged    bool
	IgnoreLicenses   bool
	FilterChangeType string // "added", "removed", "modified", "all"
}

// DiffSBOMs compares two SBOMs and returns the differences
func DiffSBOMs(oldPath, newPath string, opts *DiffOptions) (*DiffResult, error) {
	if opts == nil {
		opts = &DiffOptions{}
	}

	// Load both SBOMs
	oldSBOM, err := loadSBOMForDiff(oldPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load old SBOM: %w", err)
	}

	newSBOM, err := loadSBOMForDiff(newPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load new SBOM: %w", err)
	}

	// Build component maps for comparison
	oldComponents := buildComponentMap(oldSBOM)
	newComponents := buildComponentMap(newSBOM)

	result := &DiffResult{
		Added:    []ComponentDiff{},
		Removed:  []ComponentDiff{},
		Modified: []ComponentDiff{},
		Comparison: ComparisonMeta{
			OldSBOM: SBOMMeta{
				Path:           oldPath,
				Format:         oldSBOM.Format,
				SpecVersion:    oldSBOM.SpecVersion,
				ComponentCount: len(oldComponents),
			},
			NewSBOM: SBOMMeta{
				Path:           newPath,
				Format:         newSBOM.Format,
				SpecVersion:    newSBOM.SpecVersion,
				ComponentCount: len(newComponents),
			},
		},
	}

	// Find added and modified components
	for key, newComp := range newComponents {
		if oldComp, exists := oldComponents[key]; exists {
			// Component exists in both - check for changes
			diff := compareComponents(oldComp, newComp, opts)
			if diff != nil {
				result.Modified = append(result.Modified, *diff)
			} else if opts.ShowUnchanged {
				result.Unchanged = append(result.Unchanged, ComponentDiff{
					Name:       newComp.Name,
					Group:      newComp.Group,
					Version:    newComp.Version,
					License:    newComp.License,
					ChangeType: "unchanged",
				})
			}
		} else {
			// Component is new
			result.Added = append(result.Added, ComponentDiff{
				Name:       newComp.Name,
				Group:      newComp.Group,
				Version:    newComp.Version,
				License:    newComp.License,
				PackageURL: newComp.PackageURL,
				ChangeType: "added",
			})
		}
	}

	// Find removed components
	for key, oldComp := range oldComponents {
		if _, exists := newComponents[key]; !exists {
			result.Removed = append(result.Removed, ComponentDiff{
				Name:       oldComp.Name,
				Group:      oldComp.Group,
				Version:    oldComp.Version,
				License:    oldComp.License,
				PackageURL: oldComp.PackageURL,
				ChangeType: "removed",
			})
		}
	}

	// Sort results for consistent output
	sortComponentDiffs(result.Added)
	sortComponentDiffs(result.Removed)
	sortComponentDiffs(result.Modified)
	sortComponentDiffs(result.Unchanged)

	// Calculate summary
	result.Summary = calculateDiffSummary(result)

	return result, nil
}

// compareComponents compares two components and returns a diff if they differ
func compareComponents(old, new *Component, opts *DiffOptions) *ComponentDiff {
	var changes []string
	diff := &ComponentDiff{
		Name:  new.Name,
		Group: new.Group,
	}

	hasChanges := false

	// Check version changes
	if old.Version != new.Version {
		hasChanges = true
		diff.OldVersion = old.Version
		diff.NewVersion = new.Version
		diff.ChangeType = "version_change"

		// Determine if upgrade or downgrade
		severity := determineVersionChangeSeverity(old.Version, new.Version)
		diff.Severity = severity

		if severity == "upgrade" {
			changes = append(changes, fmt.Sprintf("Version upgraded from %s to %s", old.Version, new.Version))
		} else if severity == "downgrade" {
			changes = append(changes, fmt.Sprintf("Version downgraded from %s to %s", old.Version, new.Version))
		} else {
			changes = append(changes, fmt.Sprintf("Version changed from %s to %s", old.Version, new.Version))
		}
	}

	// Check license changes (if not ignored)
	if !opts.IgnoreLicenses && old.License != new.License {
		hasChanges = true
		diff.OldLicense = old.License
		diff.NewLicense = new.License
		if diff.ChangeType == "" {
			diff.ChangeType = "license_change"
		}
		changes = append(changes, fmt.Sprintf("License changed from %s to %s", old.License, new.License))
	}

	if !hasChanges {
		return nil
	}

	diff.Changes = changes
	diff.PackageURL = new.PackageURL
	return diff
}

// determineVersionChangeSeverity analyzes version change direction
func determineVersionChangeSeverity(oldVer, newVer string) string {
	// Simple comparison - could be enhanced with semver parsing
	oldVer = strings.TrimPrefix(oldVer, "v")
	newVer = strings.TrimPrefix(newVer, "v")

	if oldVer == newVer {
		return "unchanged"
	}

	// Try to determine upgrade vs downgrade
	if strings.Compare(newVer, oldVer) > 0 {
		return "upgrade"
	} else if strings.Compare(newVer, oldVer) < 0 {
		return "downgrade"
	}

	return "changed"
}

// calculateDiffSummary computes summary statistics
func calculateDiffSummary(result *DiffResult) DiffSummary {
	summary := DiffSummary{
		AddedCount:     len(result.Added),
		RemovedCount:   len(result.Removed),
		ModifiedCount:  len(result.Modified),
		UnchangedCount: len(result.Unchanged),
	}

	// Count version changes and license changes
	for _, diff := range result.Modified {
		if diff.ChangeType == "version_change" || diff.Severity != "" {
			if diff.Severity == "upgrade" {
				summary.VersionUpgrades++
			} else if diff.Severity == "downgrade" {
				summary.VersionDowngrades++
			}
		}
		if diff.OldLicense != "" && diff.NewLicense != "" && diff.OldLicense != diff.NewLicense {
			summary.LicenseChanges++
		}
	}

	summary.TotalComponents = summary.AddedCount + summary.RemovedCount +
		summary.ModifiedCount + summary.UnchangedCount

	return summary
}

// sortComponentDiffs sorts component diffs by name for consistent output
func sortComponentDiffs(diffs []ComponentDiff) {
	sort.Slice(diffs, func(i, j int) bool {
		if diffs[i].Group != diffs[j].Group {
			return diffs[i].Group < diffs[j].Group
		}
		return diffs[i].Name < diffs[j].Name
	})
}

// Component represents a simplified component for diffing
type Component struct {
	Name       string
	Group      string
	Version    string
	License    string
	PackageURL string
}

// SimpleSBOM represents a simplified SBOM structure for diffing
type SimpleSBOM struct {
	Format      string
	SpecVersion string
	Components  []Component
}

// loadSBOMForDiff loads an SBOM file and extracts components for diffing
func loadSBOMForDiff(path string) (*SimpleSBOM, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Try to detect format and parse
	// For now, assume CycloneDX JSON format
	var cdx struct {
		BomFormat   string `json:"bomFormat"`
		SpecVersion string `json:"specVersion"`
		Components  []struct {
			Name       string `json:"name"`
			Group      string `json:"group,omitempty"`
			Version    string `json:"version"`
			PackageURL string `json:"purl,omitempty"`
			Licenses   []struct {
				License struct {
					ID   string `json:"id,omitempty"`
					Name string `json:"name,omitempty"`
				} `json:"license,omitempty"`
			} `json:"licenses,omitempty"`
		} `json:"components"`
	}

	if err := json.Unmarshal(data, &cdx); err != nil {
		return nil, fmt.Errorf("failed to parse SBOM: %w", err)
	}

	sbom := &SimpleSBOM{
		Format:      cdx.BomFormat,
		SpecVersion: cdx.SpecVersion,
		Components:  make([]Component, 0, len(cdx.Components)),
	}

	for _, comp := range cdx.Components {
		license := ""
		if len(comp.Licenses) > 0 && comp.Licenses[0].License.ID != "" {
			license = comp.Licenses[0].License.ID
		} else if len(comp.Licenses) > 0 && comp.Licenses[0].License.Name != "" {
			license = comp.Licenses[0].License.Name
		}

		sbom.Components = append(sbom.Components, Component{
			Name:       comp.Name,
			Group:      comp.Group,
			Version:    comp.Version,
			License:    license,
			PackageURL: comp.PackageURL,
		})
	}

	return sbom, nil
}

// buildComponentMap creates a map of components keyed by name+group for fast lookup
func buildComponentMap(sbom *SimpleSBOM) map[string]*Component {
	m := make(map[string]*Component)
	for i := range sbom.Components {
		comp := &sbom.Components[i]
		key := componentKey(comp.Group, comp.Name)
		m[key] = comp
	}
	return m
}

// componentKey creates a unique key for a component
func componentKey(group, name string) string {
	if group != "" {
		return group + "/" + name
	}
	return name
}
