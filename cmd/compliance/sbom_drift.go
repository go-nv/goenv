package compliance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-nv/goenv/internal/sbom"
	"github.com/spf13/cobra"
)

var (
	driftBaselineDir     string
	driftBaselineName    string
	driftFormat          string
	driftFailOnDrift     bool
	driftAllowUpgrades   bool
	driftAllowDowngrades bool
	driftAllowLicense    bool
	driftAllowAdded      []string
	driftAllowRemoved    []string
	driftStrictMode      bool
	driftOutput          string
)

// sbomDriftCmd manages baseline SBOMs and detects drift
var sbomDriftCmd = &cobra.Command{
	Use:   "drift",
	Short: "Manage baseline SBOMs and detect drift",
	Long: `Manage baseline SBOMs and detect drift in dependencies.

Drift detection helps track unexpected changes by comparing current SBOMs
against established baselines. This is useful for:

- Detecting unexpected dependency changes in CI/CD
- Validating dependency updates before deployment
- Tracking supply chain drift over time
- Ensuring reproducibility across environments`,
}

// sbomDriftSaveCmd saves a baseline SBOM
var sbomDriftSaveCmd = &cobra.Command{
	Use:   "save <sbom-file>",
	Short: "Save a baseline SBOM",
	Long: `Save a current SBOM as a baseline for future drift detection.

Example:
  goenv sbom drift save sbom.json --name production --version v1.0.0`,
	Args: cobra.ExactArgs(1),
	RunE: runDriftSave,
}

// sbomDriftDetectCmd detects drift against a baseline
var sbomDriftDetectCmd = &cobra.Command{
	Use:   "detect <sbom-file>",
	Short: "Detect drift against a baseline",
	Long: `Detect drift by comparing a current SBOM against a baseline.

Example:
  # Basic drift detection
  goenv sbom drift detect current.json --name production

  # Allow version upgrades
  goenv sbom drift detect current.json --allow-upgrades

  # Allow specific additions
  goenv sbom drift detect current.json --allow-added "github.com/new/*"

  # Fail CI if drift detected
  goenv sbom drift detect current.json --fail-on-drift

  # Strict mode (no changes allowed)
  goenv sbom drift detect current.json --strict`,
	Args: cobra.ExactArgs(1),
	RunE: runDriftDetect,
}

// sbomDriftListCmd lists available baselines
var sbomDriftListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available baselines",
	Long:  `List all available baseline SBOMs.`,
	RunE:  runDriftList,
}

// sbomDriftDeleteCmd deletes a baseline
var sbomDriftDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a baseline SBOM",
	Long:  `Delete a baseline SBOM by name.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runDriftDelete,
}

func init() {
	// Add drift subcommands
	sbomDriftCmd.AddCommand(sbomDriftSaveCmd)
	sbomDriftCmd.AddCommand(sbomDriftDetectCmd)
	sbomDriftCmd.AddCommand(sbomDriftListCmd)
	sbomDriftCmd.AddCommand(sbomDriftDeleteCmd)

	// Common flags
	sbomDriftCmd.PersistentFlags().StringVar(&driftBaselineDir, "baseline-dir", ".goenv/baselines", "Directory to store baselines")

	// Save flags
	sbomDriftSaveCmd.Flags().StringVar(&driftBaselineName, "name", "default", "Baseline name")
	sbomDriftSaveCmd.Flags().String("version", "", "Baseline version")
	sbomDriftSaveCmd.Flags().String("description", "", "Baseline description")

	// Detect flags
	sbomDriftDetectCmd.Flags().StringVar(&driftBaselineName, "name", "default", "Baseline name to compare against")
	sbomDriftDetectCmd.Flags().StringVar(&driftFormat, "format", "table", "Output format: table, json")
	sbomDriftDetectCmd.Flags().BoolVar(&driftFailOnDrift, "fail-on-drift", false, "Exit with error if drift detected")
	sbomDriftDetectCmd.Flags().BoolVar(&driftAllowUpgrades, "allow-upgrades", false, "Allow version upgrades")
	sbomDriftDetectCmd.Flags().BoolVar(&driftAllowDowngrades, "allow-downgrades", false, "Allow version downgrades")
	sbomDriftDetectCmd.Flags().BoolVar(&driftAllowLicense, "allow-license-changes", false, "Allow license changes")
	sbomDriftDetectCmd.Flags().StringSliceVar(&driftAllowAdded, "allow-added", nil, "Component patterns allowed to be added (supports *)")
	sbomDriftDetectCmd.Flags().StringSliceVar(&driftAllowRemoved, "allow-removed", nil, "Component patterns allowed to be removed (supports *)")
	sbomDriftDetectCmd.Flags().BoolVar(&driftStrictMode, "strict", false, "Strict mode: no changes allowed")
	sbomDriftDetectCmd.Flags().StringVarP(&driftOutput, "output", "o", "", "Write output to file instead of stdout")

	// Add to parent command
	sbomCmd.AddCommand(sbomDriftCmd)
}

func runDriftSave(cmd *cobra.Command, args []string) error {
	sbomFile := args[0]

	// Validate file exists
	if _, err := os.Stat(sbomFile); os.IsNotExist(err) {
		return fmt.Errorf("SBOM file not found: %s", sbomFile)
	}

	// Get flags
	version, _ := cmd.Flags().GetString("version")
	description, _ := cmd.Flags().GetString("description")

	// Create drift detector
	detector, err := sbom.NewDriftDetector(driftBaselineDir)
	if err != nil {
		return fmt.Errorf("failed to create drift detector: %w", err)
	}

	// Save baseline
	if err := detector.SaveBaseline(sbomFile, driftBaselineName, version, description); err != nil {
		return fmt.Errorf("failed to save baseline: %w", err)
	}

	fmt.Printf("✓ Baseline saved: %s\n", driftBaselineName)
	fmt.Printf("  Location: %s\n", filepath.Join(driftBaselineDir, fmt.Sprintf("%s.baseline.json", driftBaselineName)))
	if version != "" {
		fmt.Printf("  Version: %s\n", version)
	}
	if description != "" {
		fmt.Printf("  Description: %s\n", description)
	}

	return nil
}

func runDriftDetect(cmd *cobra.Command, args []string) error {
	sbomFile := args[0]

	// Validate file exists
	if _, err := os.Stat(sbomFile); os.IsNotExist(err) {
		return fmt.Errorf("SBOM file not found: %s", sbomFile)
	}

	// Create drift detector
	detector, err := sbom.NewDriftDetector(driftBaselineDir)
	if err != nil {
		return fmt.Errorf("failed to create drift detector: %w", err)
	}

	// Configure options
	options := sbom.DriftOptions{
		AllowedAdditions:    driftAllowAdded,
		AllowedRemovals:     driftAllowRemoved,
		AllowUpgrades:       driftAllowUpgrades,
		AllowDowngrades:     driftAllowDowngrades,
		AllowLicenseChanges: driftAllowLicense,
		StrictMode:          driftStrictMode,
	}

	// Detect drift
	result, err := detector.DetectDrift(sbomFile, driftBaselineName, options)
	if err != nil {
		return fmt.Errorf("failed to detect drift: %w", err)
	}

	// Format and output result
	var output []byte
	switch driftFormat {
	case "json":
		output, err = json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal result: %w", err)
		}
	case "table":
		output = []byte(formatDriftTable(result))
	default:
		return fmt.Errorf("unknown format: %s", driftFormat)
	}

	// Write output
	if driftOutput != "" {
		if err := os.WriteFile(driftOutput, output, 0644); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
		fmt.Printf("Drift report written to: %s\n", driftOutput)
	} else {
		fmt.Println(string(output))
	}

	// Exit with error if drift detected and --fail-on-drift is set
	if driftFailOnDrift && result.HasDrift {
		return fmt.Errorf("drift detected")
	}

	return nil
}

func runDriftList(cmd *cobra.Command, args []string) error {
	detector, err := sbom.NewDriftDetector(driftBaselineDir)
	if err != nil {
		return fmt.Errorf("failed to create drift detector: %w", err)
	}

	baselines, err := detector.ListBaselines()
	if err != nil {
		return fmt.Errorf("failed to list baselines: %w", err)
	}

	if len(baselines) == 0 {
		fmt.Println("No baselines found")
		fmt.Printf("Create one with: goenv sbom drift save <sbom-file> --name <name>\n")
		return nil
	}

	fmt.Printf("Available baselines (%d):\n\n", len(baselines))
	for _, baseline := range baselines {
		// Extract name from baseline file path
		name := strings.TrimSuffix(filepath.Base(baseline.Path), ".baseline.json")
		fmt.Printf("  %s\n", name)
		if baseline.Version != "" {
			fmt.Printf("    Version: %s\n", baseline.Version)
		}
		fmt.Printf("    Created: %s\n", baseline.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("    Components: %d\n", baseline.ComponentCount)
		if baseline.Description != "" {
			fmt.Printf("    Description: %s\n", baseline.Description)
		}
		fmt.Println()
	}

	return nil
}

func runDriftDelete(cmd *cobra.Command, args []string) error {
	name := args[0]

	detector, err := sbom.NewDriftDetector(driftBaselineDir)
	if err != nil {
		return fmt.Errorf("failed to create drift detector: %w", err)
	}

	if err := detector.DeleteBaseline(name); err != nil {
		return err
	}

	fmt.Printf("✓ Baseline deleted: %s\n", name)
	return nil
}

func formatDriftTable(result *sbom.DriftResult) string {
	var sb strings.Builder

	// Header
	sb.WriteString("═════════════════════════════════════════════════════════\n")
	sb.WriteString("                  SBOM Drift Report\n")
	sb.WriteString("═════════════════════════════════════════════════════════\n\n")

	// Status
	if result.HasDrift {
		sb.WriteString(fmt.Sprintf("Status:         ⚠️  DRIFT DETECTED (Severity: %s)\n", strings.ToUpper(result.DriftSummary.SeverityLevel)))
	} else {
		sb.WriteString("Status:         ✓ NO DRIFT\n")
	}
	sb.WriteString(fmt.Sprintf("Detected:       %s\n", result.DetectedAt.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("Baseline:       %s (created %s)\n", filepath.Base(result.Baseline.Path), result.Baseline.CreatedAt.Format("2006-01-02")))
	if result.Baseline.Version != "" {
		sb.WriteString(fmt.Sprintf("Version:        %s\n", result.Baseline.Version))
	}
	sb.WriteString("\n")

	// Summary
	summary := result.DriftSummary
	sb.WriteString("Summary:\n")
	sb.WriteString(fmt.Sprintf("  Total Changes:          %d\n", summary.TotalChanges))
	if summary.UnexpectedAdded > 0 {
		sb.WriteString(fmt.Sprintf("  Unexpected Added:       %d\n", summary.UnexpectedAdded))
	}
	if summary.UnexpectedRemoved > 0 {
		sb.WriteString(fmt.Sprintf("  Unexpected Removed:     %d\n", summary.UnexpectedRemoved))
	}
	if summary.UnexpectedUpgrades > 0 {
		sb.WriteString(fmt.Sprintf("  Unexpected Upgrades:    %d\n", summary.UnexpectedUpgrades))
	}
	if summary.UnexpectedDowngrades > 0 {
		sb.WriteString(fmt.Sprintf("  Unexpected Downgrades:  %d\n", summary.UnexpectedDowngrades))
	}
	if summary.LicenseChanges > 0 {
		sb.WriteString(fmt.Sprintf("  License Changes:        %d\n", summary.LicenseChanges))
	}
	sb.WriteString("\n")

	// Violations
	if len(result.Violations) > 0 {
		sb.WriteString(fmt.Sprintf("Violations (%d):\n\n", len(result.Violations)))
		for i, v := range result.Violations {
			severityIcon := "•"
			switch v.Severity {
			case "high":
				severityIcon = "🔴"
			case "medium":
				severityIcon = "🟡"
			case "low":
				severityIcon = "🔵"
			}

			sb.WriteString(fmt.Sprintf("  %s [%s] %s\n", severityIcon, strings.ToUpper(v.Severity), v.Message))
			if v.Component != "" {
				sb.WriteString(fmt.Sprintf("     Component: %s\n", v.Component))
				if v.OldValue != "" && v.NewValue != "" {
					sb.WriteString(fmt.Sprintf("     Change: %s → %s\n", v.OldValue, v.NewValue))
				}
			}
			if i < len(result.Violations)-1 {
				sb.WriteString("\n")
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("═════════════════════════════════════════════════════════\n")

	return sb.String()
}
