package sbom

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// DiffFormatter formats diff results for display
type DiffFormatter interface {
	Format(result *DiffResult, w io.Writer) error
}

// JSONFormatter outputs diff as JSON
type JSONFormatter struct {
	Pretty bool
}

// Format writes the diff result as JSON
func (f *JSONFormatter) Format(result *DiffResult, w io.Writer) error {
	encoder := json.NewEncoder(w)
	if f.Pretty {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(result)
}

// TableFormatter outputs diff as a human-readable table
type TableFormatter struct {
	ShowUnchanged bool
	Color         bool
}

// Format writes the diff result as a formatted table
func (f *TableFormatter) Format(result *DiffResult, w io.Writer) error {
	// Print summary
	fmt.Fprintln(w, "SBOM Diff Summary")
	fmt.Fprintln(w, strings.Repeat("=", 60))
	fmt.Fprintf(w, "Old SBOM: %s (%d components)\n", result.Comparison.OldSBOM.Path, result.Comparison.OldSBOM.ComponentCount)
	fmt.Fprintf(w, "New SBOM: %s (%d components)\n", result.Comparison.NewSBOM.Path, result.Comparison.NewSBOM.ComponentCount)
	fmt.Fprintln(w)

	fmt.Fprintln(w, "Changes:")
	fmt.Fprintf(w, "  Added:      %d\n", result.Summary.AddedCount)
	fmt.Fprintf(w, "  Removed:    %d\n", result.Summary.RemovedCount)
	fmt.Fprintf(w, "  Modified:   %d\n", result.Summary.ModifiedCount)
	fmt.Fprintf(w, "  Unchanged:  %d\n", result.Summary.UnchangedCount)

	if result.Summary.VersionUpgrades > 0 || result.Summary.VersionDowngrades > 0 {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "Version Changes:\n")
		fmt.Fprintf(w, "  Upgrades:   %d\n", result.Summary.VersionUpgrades)
		fmt.Fprintf(w, "  Downgrades: %d\n", result.Summary.VersionDowngrades)
	}

	if result.Summary.LicenseChanges > 0 {
		fmt.Fprintf(w, "  License Changes: %d\n", result.Summary.LicenseChanges)
	}

	// Print added components
	if len(result.Added) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Added Components:")
		fmt.Fprintln(w, strings.Repeat("-", 60))
		for _, comp := range result.Added {
			prefix := f.colorize("  + ", "green")
			fmt.Fprintf(w, "%s%s", prefix, f.formatComponent(comp))
		}
	}

	// Print removed components
	if len(result.Removed) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Removed Components:")
		fmt.Fprintln(w, strings.Repeat("-", 60))
		for _, comp := range result.Removed {
			prefix := f.colorize("  - ", "red")
			fmt.Fprintf(w, "%s%s", prefix, f.formatComponent(comp))
		}
	}

	// Print modified components
	if len(result.Modified) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Modified Components:")
		fmt.Fprintln(w, strings.Repeat("-", 60))
		for _, comp := range result.Modified {
			prefix := f.colorize("  ~ ", "yellow")
			fmt.Fprintf(w, "%s%s\n", prefix, f.formatComponentName(comp))

			for _, change := range comp.Changes {
				fmt.Fprintf(w, "      %s\n", change)
			}
		}
	}

	// Print unchanged (if requested)
	if f.ShowUnchanged && len(result.Unchanged) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Unchanged Components:")
		fmt.Fprintln(w, strings.Repeat("-", 60))
		for _, comp := range result.Unchanged {
			fmt.Fprintf(w, "    %s\n", f.formatComponent(comp))
		}
	}

	return nil
}

func (f *TableFormatter) formatComponent(comp ComponentDiff) string {
	parts := []string{f.formatComponentName(comp)}

	if comp.Version != "" {
		parts = append(parts, fmt.Sprintf("v%s", comp.Version))
	}

	if comp.License != "" {
		parts = append(parts, fmt.Sprintf("(%s)", comp.License))
	}

	return strings.Join(parts, " ") + "\n"
}

func (f *TableFormatter) formatComponentName(comp ComponentDiff) string {
	if comp.Group != "" {
		return comp.Group + "/" + comp.Name
	}
	return comp.Name
}

func (f *TableFormatter) colorize(text, color string) string {
	if !f.Color {
		return text
	}

	colors := map[string]string{
		"red":    "\033[31m",
		"green":  "\033[32m",
		"yellow": "\033[33m",
		"reset":  "\033[0m",
	}

	if code, ok := colors[color]; ok {
		return code + text + colors["reset"]
	}
	return text
}

// GitHubFormatter outputs diff in GitHub Actions format
type GitHubFormatter struct{}

// Format writes the diff result for GitHub Actions
func (f *GitHubFormatter) Format(result *DiffResult, w io.Writer) error {
	// GitHub Actions annotations
	if result.Summary.RemovedCount > 0 {
		fmt.Fprintf(w, "::warning::SBOM Analysis: %d dependencies removed\n", result.Summary.RemovedCount)
	}

	if result.Summary.AddedCount > 0 {
		fmt.Fprintf(w, "::notice::SBOM Analysis: %d dependencies added\n", result.Summary.AddedCount)
	}

	if result.Summary.VersionDowngrades > 0 {
		fmt.Fprintf(w, "::warning::SBOM Analysis: %d version downgrades detected\n", result.Summary.VersionDowngrades)
	}

	// Output summary in collapsible section
	fmt.Fprintln(w, "<details>")
	fmt.Fprintln(w, "<summary>SBOM Diff Summary</summary>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "```")

	// Use table formatter for the body
	table := &TableFormatter{ShowUnchanged: false, Color: false}
	if err := table.Format(result, w); err != nil {
		return err
	}

	fmt.Fprintln(w, "```")
	fmt.Fprintln(w, "</details>")

	return nil
}

// MarkdownFormatter outputs diff as Markdown
type MarkdownFormatter struct {
	ShowUnchanged bool
}

// Format writes the diff result as Markdown
func (f *MarkdownFormatter) Format(result *DiffResult, w io.Writer) error {
	fmt.Fprintln(w, "# SBOM Diff Report")
	fmt.Fprintln(w)

	// Summary table
	fmt.Fprintln(w, "## Summary")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| Metric | Count |")
	fmt.Fprintln(w, "|--------|-------|")
	fmt.Fprintf(w, "| Added | %d |\n", result.Summary.AddedCount)
	fmt.Fprintf(w, "| Removed | %d |\n", result.Summary.RemovedCount)
	fmt.Fprintf(w, "| Modified | %d |\n", result.Summary.ModifiedCount)
	fmt.Fprintf(w, "| Unchanged | %d |\n", result.Summary.UnchangedCount)

	if result.Summary.VersionUpgrades > 0 || result.Summary.VersionDowngrades > 0 {
		fmt.Fprintf(w, "| Version Upgrades | %d |\n", result.Summary.VersionUpgrades)
		fmt.Fprintf(w, "| Version Downgrades | %d |\n", result.Summary.VersionDowngrades)
	}

	if result.Summary.LicenseChanges > 0 {
		fmt.Fprintf(w, "| License Changes | %d |\n", result.Summary.LicenseChanges)
	}
	fmt.Fprintln(w)

	// Added components
	if len(result.Added) > 0 {
		fmt.Fprintln(w, "## ➕ Added Components")
		fmt.Fprintln(w)
		for _, comp := range result.Added {
			fmt.Fprintf(w, "- **%s** `v%s`", f.formatComponentName(comp), comp.Version)
			if comp.License != "" {
				fmt.Fprintf(w, " (%s)", comp.License)
			}
			fmt.Fprintln(w)
		}
		fmt.Fprintln(w)
	}

	// Removed components
	if len(result.Removed) > 0 {
		fmt.Fprintln(w, "## ➖ Removed Components")
		fmt.Fprintln(w)
		for _, comp := range result.Removed {
			fmt.Fprintf(w, "- **%s** `v%s`", f.formatComponentName(comp), comp.Version)
			if comp.License != "" {
				fmt.Fprintf(w, " (%s)", comp.License)
			}
			fmt.Fprintln(w)
		}
		fmt.Fprintln(w)
	}

	// Modified components
	if len(result.Modified) > 0 {
		fmt.Fprintln(w, "## 🔄 Modified Components")
		fmt.Fprintln(w)
		for _, comp := range result.Modified {
			fmt.Fprintf(w, "- **%s**\n", f.formatComponentName(comp))
			for _, change := range comp.Changes {
				fmt.Fprintf(w, "  - %s\n", change)
			}
		}
		fmt.Fprintln(w)
	}

	return nil
}

func (f *MarkdownFormatter) formatComponentName(comp ComponentDiff) string {
	if comp.Group != "" {
		return comp.Group + "/" + comp.Name
	}
	return comp.Name
}

// GetFormatter returns the appropriate formatter for the given format
func GetFormatter(format string, showUnchanged bool, useColor bool) (DiffFormatter, error) {
	switch format {
	case "json":
		return &JSONFormatter{Pretty: true}, nil
	case "json-compact":
		return &JSONFormatter{Pretty: false}, nil
	case "table":
		return &TableFormatter{ShowUnchanged: showUnchanged, Color: useColor}, nil
	case "github":
		return &GitHubFormatter{}, nil
	case "markdown", "md":
		return &MarkdownFormatter{ShowUnchanged: showUnchanged}, nil
	default:
		return nil, fmt.Errorf("unsupported format: %s (supported: json, table, github, markdown)", format)
	}
}
