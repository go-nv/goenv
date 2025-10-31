package tools

import (
	"fmt"
	"sort"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

func newStatusCommand() *cobra.Command {
	var statusJSON bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show tool installation consistency across versions",
		Long: `Display an overview of tool installation across all Go versions.

This command shows which tools are installed in which versions,
helping you maintain consistency across your Go installations.

Examples:
  goenv tools status                # Show tool installation status
  goenv tools status --json         # JSON output for automation`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(cmd, statusJSON)
		},
	}

	cmd.Flags().BoolVar(&statusJSON, "json", false, "Output in JSON format")

	return cmd
}

type toolStatus struct {
	Name             string          `json:"name"`
	TotalVersions    int             `json:"total_versions"`
	InstalledIn      int             `json:"installed_in"`
	VersionPresence  map[string]bool `json:"version_presence"`
	ConsistencyScore float64         `json:"consistency_score"`
}

func runStatus(cmd *cobra.Command, jsonOutput bool) error {
	cfg := config.Load()

	// Get all installed Go versions
	versions, err := getInstalledVersions(cfg)
	if err != nil {
		return err
	}

	if len(versions) == 0 {
		return fmt.Errorf("no Go versions installed")
	}

	// Collect all tools across all versions
	toolsByVersion := make(map[string][]string)
	allToolNames := make(map[string]bool)

	for _, version := range versions {
		tools, err := getToolsForVersion(cfg, version)
		if err != nil {
			continue
		}
		toolsByVersion[version] = tools
		for _, tool := range tools {
			allToolNames[tool] = true
		}
	}

	// Calculate status for each tool
	var toolStatuses []toolStatus

	for toolName := range allToolNames {
		presence := make(map[string]bool)
		installedCount := 0

		for _, version := range versions {
			isInstalled := contains(toolsByVersion[version], toolName)
			presence[version] = isInstalled
			if isInstalled {
				installedCount++
			}
		}

		consistency := float64(installedCount) / float64(len(versions)) * 100

		toolStatuses = append(toolStatuses, toolStatus{
			Name:             toolName,
			TotalVersions:    len(versions),
			InstalledIn:      installedCount,
			VersionPresence:  presence,
			ConsistencyScore: consistency,
		})
	}

	// Sort by consistency (least consistent first)
	sort.Slice(toolStatuses, func(i, j int) bool {
		if toolStatuses[i].ConsistencyScore == toolStatuses[j].ConsistencyScore {
			return toolStatuses[i].Name < toolStatuses[j].Name
		}
		return toolStatuses[i].ConsistencyScore < toolStatuses[j].ConsistencyScore
	})

	// Handle JSON output
	if jsonOutput {
		type jsonOutput struct {
			SchemaVersion string       `json:"schema_version"`
			GoVersions    []string     `json:"go_versions"`
			Tools         []toolStatus `json:"tools"`
		}
		output := jsonOutput{
			SchemaVersion: "1",
			GoVersions:    versions,
			Tools:         toolStatuses,
		}
		return utils.PrintJSON(cmd.OutOrStdout(), output)
	}

	// Human-readable output
	if len(toolStatuses) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "\n%s No tools installed\n",
			utils.EmojiOr("ℹ️  ", ""))
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\n%s %s\n\n",
		utils.EmojiOr("📊 ", ""),
		utils.BoldBlue("Tool Installation Status"))

	fmt.Fprintf(cmd.OutOrStdout(), "Go versions: %s\n\n",
		utils.Cyan(fmt.Sprintf("%d installed", len(versions))))

	// Categorize tools
	var fullyInstalled, partiallyInstalled, singleVersion []toolStatus

	for _, status := range toolStatuses {
		if status.InstalledIn == len(versions) {
			fullyInstalled = append(fullyInstalled, status)
		} else if status.InstalledIn == 1 {
			singleVersion = append(singleVersion, status)
		} else {
			partiallyInstalled = append(partiallyInstalled, status)
		}
	}

	// Show fully installed tools
	if len(fullyInstalled) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n",
			utils.EmojiOr("✅ ", ""),
			utils.BoldGreen(fmt.Sprintf("Consistent Tools (%d/%d versions)", len(versions), len(versions))))

		for _, status := range fullyInstalled {
			fmt.Fprintf(cmd.OutOrStdout(), "  • %s\n", utils.Green(status.Name))
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	// Show partially installed tools
	if len(partiallyInstalled) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n",
			utils.EmojiOr("⚠️  ", ""),
			utils.BoldYellow("Partially Installed Tools"))

		for _, status := range partiallyInstalled {
			fmt.Fprintf(cmd.OutOrStdout(), "  • %s %s %s\n",
				utils.Yellow(status.Name),
				utils.Gray("→"),
				utils.Gray(fmt.Sprintf("%d/%d versions (%.0f%%)",
					status.InstalledIn, status.TotalVersions, status.ConsistencyScore)))

			// Show which versions are missing it
			var missingVersions []string
			for _, version := range versions {
				if !status.VersionPresence[version] {
					missingVersions = append(missingVersions, version)
				}
			}

			if len(missingVersions) <= 3 {
				fmt.Fprintf(cmd.OutOrStdout(), "    %s %s\n",
					utils.Gray("Missing in:"),
					utils.Red(fmt.Sprintf("%v", missingVersions)))
			}
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	// Show single-version tools
	if len(singleVersion) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n",
			utils.EmojiOr("ℹ️  ", ""),
			utils.BoldBlue("Version-Specific Tools"))

		for _, status := range singleVersion {
			var installedVersion string
			for version, present := range status.VersionPresence {
				if present {
					installedVersion = version
					break
				}
			}
			fmt.Fprintf(cmd.OutOrStdout(), "  • %s %s %s\n",
				utils.Cyan(status.Name),
				utils.Gray("→"),
				utils.Gray("only in "+installedVersion))
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	// Summary and recommendations
	fmt.Fprintf(cmd.OutOrStdout(), "%sSummary:\n", utils.Emoji("📝 "))
	fmt.Fprintf(cmd.OutOrStdout(), "  Total tools: %s\n", utils.Cyan(fmt.Sprintf("%d", len(toolStatuses))))
	fmt.Fprintf(cmd.OutOrStdout(), "  Consistent:  %s\n", utils.Green(fmt.Sprintf("%d", len(fullyInstalled))))
	fmt.Fprintf(cmd.OutOrStdout(), "  Partial:     %s\n", utils.Yellow(fmt.Sprintf("%d", len(partiallyInstalled))))
	fmt.Fprintf(cmd.OutOrStdout(), "  Specific:    %s\n", utils.Blue(fmt.Sprintf("%d", len(singleVersion))))

	if len(partiallyInstalled) > 0 {
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintf(cmd.OutOrStdout(), "%sRecommendations:\n", utils.Emoji("💡 "))
		fmt.Fprintln(cmd.OutOrStdout(), "  • Use 'goenv tools install <tool> --all' to install across all versions")
		fmt.Fprintln(cmd.OutOrStdout(), "  • Use 'goenv tools sync <from> <to>' to copy tools between versions")
	}

	return nil
}
