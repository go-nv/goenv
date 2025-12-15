package tools

import (
	"fmt"

	"github.com/go-nv/goenv/internal/cmdutil"
	"github.com/go-nv/goenv/internal/errors"
	toolspkg "github.com/go-nv/goenv/internal/tools"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

func newOutdatedCommand() *cobra.Command {
	var outdatedJSON bool

	cmd := &cobra.Command{
		Use:   "outdated",
		Short: "Show outdated tools across all Go versions",
		Long: `Check which tools are outdated across all installed Go versions.

This command checks all tools in all Go versions and reports which ones
have newer versions available.

Examples:
  goenv tools outdated                # Show all outdated tools
  goenv tools outdated --json         # JSON output for automation`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOutdated(cmd, outdatedJSON)
		},
	}

	cmd.Flags().BoolVar(&outdatedJSON, "json", false, "Output in JSON format")

	return cmd
}

type outdatedTool struct {
	Name           string `json:"name"`
	GoVersion      string `json:"go_version"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	PackagePath    string `json:"package_path"`
}

func runOutdated(cmd *cobra.Command, jsonOutput bool) error {
	ctx := cmdutil.GetContexts(cmd)
	cfg := ctx.Config
	mgr := ctx.Manager

	// Get all installed Go versions
	versions, err := mgr.ListInstalledVersions()
	if err != nil {
		return err
	}

	if len(versions) == 0 {
		return errors.NoVersionsInstalled()
	}

	// Collect outdated tools
	var allOutdated []outdatedTool

	for _, version := range versions {
		tools, err := toolspkg.ListForVersion(cfg, version)
		if err != nil {
			continue // Skip versions where we can't list tools
		}

		for _, tool := range tools {
			if tool.PackagePath == "" {
				continue // Skip tools without package path
			}

			// Query latest version
			latestVersion, err := toolspkg.GetLatestVersion(tool.PackagePath)
			if err != nil {
				continue // Skip tools we can't check
			}

			// Compare versions
			if toolspkg.CompareVersions(tool.Version, latestVersion) < 0 {
				allOutdated = append(allOutdated, outdatedTool{
					Name:           tool.Name,
					GoVersion:      version,
					CurrentVersion: tool.Version,
					LatestVersion:  latestVersion,
					PackagePath:    tool.PackagePath,
				})
			}
		}
	}

	// Handle JSON output
	if jsonOutput {
		type jsonOutput struct {
			SchemaVersion string         `json:"schema_version"`
			OutdatedTools []outdatedTool `json:"outdated_tools"`
		}
		output := jsonOutput{
			SchemaVersion: "1",
			OutdatedTools: allOutdated,
		}
		return utils.PrintJSON(cmd.OutOrStdout(), output)
	}

	// Human-readable output
	if len(allOutdated) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "\n%s %s\n",
			utils.EmojiOr("✅ ", ""),
			utils.BoldGreen("All tools are up to date!"))
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\n%s %s\n\n",
		utils.EmojiOr("📊 ", ""),
		utils.BoldBlue("Outdated Tools"))

	// Group by Go version
	versionMap := make(map[string][]outdatedTool)
	for _, tool := range allOutdated {
		versionMap[tool.GoVersion] = append(versionMap[tool.GoVersion], tool)
	}

	for _, version := range versions {
		outdatedForVersion := versionMap[version]
		if len(outdatedForVersion) == 0 {
			continue
		}

		fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n",
			utils.BoldCyan("Go "+version+":"),
			utils.Gray(fmt.Sprintf("(%d outdated)", len(outdatedForVersion))))

		for _, tool := range outdatedForVersion {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s %s %s %s %s %s\n",
				utils.EmojiOr("⬆️  ", "→"),
				utils.Cyan(tool.Name),
				utils.Gray(tool.CurrentVersion),
				utils.Gray("→"),
				utils.Green(tool.LatestVersion),
				utils.Gray("available"))
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n",
		utils.Gray(fmt.Sprintf("Total: %d outdated tool(s) across %d version(s)", len(allOutdated), len(versionMap))))

	fmt.Fprintf(cmd.OutOrStdout(), "%sTo update:\n", utils.Emoji("💡 "))
	fmt.Fprintln(cmd.OutOrStdout(), "  goenv tools update                  # Update current version")
	fmt.Fprintln(cmd.OutOrStdout(), "  goenv tools update --all            # Update all versions")

	return nil
}
