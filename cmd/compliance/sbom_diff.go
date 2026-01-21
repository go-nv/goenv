package compliance

import (
	"fmt"
	"os"

	"github.com/go-nv/goenv/internal/sbom"
	"github.com/spf13/cobra"
)

var (
	diffFormat        string
	diffShowUnchanged bool
	diffShowOnly      string
	diffFailOn        string
	diffColor         bool
	diffOutput        string
)

var sbomDiffCmd = &cobra.Command{
	Use:   "diff <old-sbom> <new-sbom>",
	Short: "Compare two SBOMs and show differences",
	Long: `Compare two SBOM files and display the differences.

This command analyzes changes between two SBOMs, including:
  - Added dependencies
  - Removed dependencies
  - Version changes (upgrades/downgrades)
  - License changes

Useful for:
  - Release change tracking
  - Security impact analysis
  - Dependency drift detection
  - CI/CD validation

Phase 5: Automation & Compliance (v3.5)

Examples:
  # Basic comparison
  goenv sbom diff sbom-v1.0.0.json sbom-v1.1.0.json

  # JSON output for automation
  goenv sbom diff old.json new.json --format=json

  # GitHub Actions format
  goenv sbom diff old.json new.json --format=github

  # Show only additions
  goenv sbom diff old.json new.json --show=added

  # Fail if dependencies were removed
  goenv sbom diff old.json new.json --fail-on=removed

  # Save to file
  goenv sbom diff old.json new.json -o diff-report.md --format=markdown`,
	Args: cobra.ExactArgs(2),
	RunE: runSBOMDiff,
}

func init() {
	sbomDiffCmd.Flags().StringVarP(&diffFormat, "format", "f", "table",
		"Output format: table, json, github, markdown")
	sbomDiffCmd.Flags().BoolVarP(&diffShowUnchanged, "show-unchanged", "u", false,
		"Show unchanged components (table format only)")
	sbomDiffCmd.Flags().StringVar(&diffShowOnly, "show", "all",
		"Show only specific changes: all, added, removed, modified")
	sbomDiffCmd.Flags().StringVar(&diffFailOn, "fail-on", "",
		"Exit with error if condition met: added, removed, modified, downgrade, license-change")
	sbomDiffCmd.Flags().BoolVar(&diffColor, "color", true,
		"Use colored output (table format only, auto-detected for TTY)")
	sbomDiffCmd.Flags().StringVarP(&diffOutput, "output", "o", "",
		"Write output to file instead of stdout")

	sbomCmd.AddCommand(sbomDiffCmd)
}

func runSBOMDiff(cmd *cobra.Command, args []string) error {
	oldPath := args[0]
	newPath := args[1]

	// Validate input files exist
	if _, err := os.Stat(oldPath); err != nil {
		return fmt.Errorf("old SBOM file not found: %s", oldPath)
	}
	if _, err := os.Stat(newPath); err != nil {
		return fmt.Errorf("new SBOM file not found: %s", newPath)
	}

	// Prepare diff options
	opts := &sbom.DiffOptions{
		ShowUnchanged: diffShowUnchanged,
	}

	// Perform diff
	result, err := sbom.DiffSBOMs(oldPath, newPath, opts)
	if err != nil {
		return fmt.Errorf("failed to diff SBOMs: %w", err)
	}

	// Filter results if requested
	if diffShowOnly != "all" {
		result = filterDiffResult(result, diffShowOnly)
	}

	// Auto-detect color support if not explicitly set
	useColor := diffColor
	if !cmd.Flags().Changed("color") {
		useColor = isTerminal()
	}

	// Get formatter
	formatter, err := sbom.GetFormatter(diffFormat, diffShowUnchanged, useColor)
	if err != nil {
		return err
	}

	// Determine output destination
	var output *os.File
	if diffOutput != "" {
		f, err := os.Create(diffOutput)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer f.Close()
		output = f
	} else {
		output = os.Stdout
	}

	// Format and write output
	if err := formatter.Format(result, output); err != nil {
		return fmt.Errorf("failed to format diff: %w", err)
	}

	// Check fail conditions
	if diffFailOn != "" {
		if shouldFail(result, diffFailOn) {
			if diffOutput == "" {
				fmt.Fprintf(os.Stderr, "\n")
			}
			return fmt.Errorf("diff check failed: condition '%s' met", diffFailOn)
		}
	}

	return nil
}

func filterDiffResult(result *sbom.DiffResult, showOnly string) *sbom.DiffResult {
	filtered := &sbom.DiffResult{
		Summary:    result.Summary,
		Comparison: result.Comparison,
	}

	switch showOnly {
	case "added":
		filtered.Added = result.Added
	case "removed":
		filtered.Removed = result.Removed
	case "modified":
		filtered.Modified = result.Modified
	default:
		// Return original
		return result
	}

	return filtered
}

func shouldFail(result *sbom.DiffResult, condition string) bool {
	switch condition {
	case "added":
		return result.Summary.AddedCount > 0
	case "removed":
		return result.Summary.RemovedCount > 0
	case "modified":
		return result.Summary.ModifiedCount > 0
	case "downgrade":
		return result.Summary.VersionDowngrades > 0
	case "license-change":
		return result.Summary.LicenseChanges > 0
	case "any":
		return result.Summary.AddedCount > 0 ||
			result.Summary.RemovedCount > 0 ||
			result.Summary.ModifiedCount > 0
	default:
		return false
	}
}

func isTerminal() bool {
	// Check if stdout is a terminal
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
