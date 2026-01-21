package tools

import (
	"fmt"

	"github.com/go-nv/goenv/internal/cmdutil"
	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/tools"
	"github.com/go-nv/goenv/internal/toolupdater"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var updateToolsCmd = &cobra.Command{
	Use:   "update [tool...]",
	Short: "Update installed Go tools to latest versions",
	Long: `Updates Go tools to their latest compatible versions.

By default, updates tools in the current Go version's GOPATH.
Use --all to update tools across all installed Go versions.

Update strategies can be configured in ~/.goenv/default-tools.yaml:
  - latest: Always update to latest version (default)
  - minor:  Update within same major version
  - patch:  Only patch updates
  - pin:    Never update (pinned version)
  - auto:   Smart update based on stability

Examples:
  goenv tools update                    # Update all tools using config strategies
  goenv tools update gopls              # Update only gopls
  goenv tools update --all              # Update across all Go versions
  goenv tools update --check            # Check for updates without installing
  goenv tools update --strategy latest  # Override strategy for this run
  goenv tools update --dry-run          # Preview what would be updated
  goenv tools update --force            # Bypass compatibility checks`,
	RunE: runUpdateTools,
}

var updateToolsFlags struct {
	check    bool
	dryRun   bool
	all      bool
	strategy string
	force    bool
	verbose  bool
}

func init() {
	// Now registered as subcommand in tools.go
	updateToolsCmd.Flags().BoolVarP(&updateToolsFlags.check, "check", "c", false, "Check for updates without installing")
	updateToolsCmd.Flags().BoolVarP(&updateToolsFlags.dryRun, "dry-run", "n", false, "Show what would be updated without actually updating")
	updateToolsCmd.Flags().BoolVar(&updateToolsFlags.all, "all", false, "Update tools across all installed Go versions")
	updateToolsCmd.Flags().StringVar(&updateToolsFlags.strategy, "strategy", "", "Override update strategy (latest, minor, patch, pin, auto)")
	updateToolsCmd.Flags().BoolVar(&updateToolsFlags.force, "force", false, "Force update bypassing compatibility checks")
	updateToolsCmd.Flags().BoolVarP(&updateToolsFlags.verbose, "verbose", "v", false, "Show detailed output")
}

func runUpdateTools(cmd *cobra.Command, args []string) error {
	ctx := cmdutil.GetContexts(cmd)
	cfg := ctx.Config
	mgr := ctx.Manager

	// Load config to get update strategies
	configPath := tools.ConfigPath(cfg.Root)
	toolConfig, err := tools.LoadConfig(configPath)
	if err != nil {
		// Config load error is not fatal, use defaults
		if updateToolsFlags.verbose {
			fmt.Fprintf(cmd.OutOrStderr(), "Warning: Failed to load config: %v\n", err)
		}
		toolConfig = tools.DefaultConfig()
	}

	// Determine update strategy
	strategy := toolupdater.StrategyAuto // Default
	if updateToolsFlags.strategy != "" {
		strategy = toolupdater.UpdateStrategy(updateToolsFlags.strategy)
	} else if toolConfig.UpdateStrategy != "" {
		strategy = toolupdater.UpdateStrategy(toolConfig.UpdateStrategy)
	}

	// Create updater
	updater := toolupdater.NewUpdater(cfg)

	// Determine target versions
	var targetVersions []string
	if updateToolsFlags.all {
		versions, err := mgr.ListInstalledVersions()
		if err != nil {
			return err
		}
		if len(versions) == 0 {
			return errors.NoVersionsInstalled()
		}
		targetVersions = versions
	} else {
		goVersion, _, _, err := mgr.GetCurrentVersionResolved()
		if err != nil {
			return errors.FailedTo("determine Go version", err)
		}
		if goVersion == manager.SystemVersion {
			return fmt.Errorf("cannot update tools for system Go version")
		}
		if err := mgr.ValidateVersion(goVersion); err != nil {
			return errors.VersionNotInstalled(goVersion, "")
		}
		targetVersions = []string{goVersion}
	}

	// Process all target versions
	totalUpdated := 0
	totalFailed := 0
	totalSkipped := 0

	for versionIdx, goVersion := range targetVersions {
		if len(targetVersions) > 1 {
			if versionIdx > 0 {
				fmt.Fprintln(cmd.OutOrStdout())
			}
			fmt.Fprintf(cmd.OutOrStdout(), "\n%s %s\n\n",
				utils.BoldCyan("Go "+goVersion+":"),
				utils.Gray("("+cfg.Root+"/versions/"+goVersion+")"))
		}

		// Prepare update options
		opts := toolupdater.UpdateOptions{
			Strategy:  strategy,
			GoVersion: goVersion,
			ToolNames: args, // Filter to specific tools if provided
			DryRun:    updateToolsFlags.dryRun,
			Force:     updateToolsFlags.force,
			Verbose:   updateToolsFlags.verbose,
			CheckOnly: updateToolsFlags.check,
		}

		// Run update check
		result, err := updater.CheckForUpdates(opts)
		if err != nil {
			return errors.FailedTo("check tool updates", err)
		}

		// Display results
		if err := displayUpdateResults(cmd, result, goVersion); err != nil {
			return err
		}

		totalUpdated += len(result.Updated)
		totalFailed += len(result.Failed)
		totalSkipped += len(result.Skipped)

		// Show errors if any
		if len(result.Errors) > 0 && updateToolsFlags.verbose {
			fmt.Fprintln(cmd.OutOrStderr(), "\nErrors:")
			for _, err := range result.Errors {
				fmt.Fprintf(cmd.OutOrStderr(), "  %s %v\n", utils.Red("✗"), err)
			}
		}
	}

	// Summary for --all mode
	if len(targetVersions) > 1 {
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintf(cmd.OutOrStdout(), "%s Summary: ", utils.Emoji("📊 "))
		if totalUpdated > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "Updated %d, ", totalUpdated)
		}
		if totalSkipped > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "Skipped %d, ", totalSkipped)
		}
		if totalFailed > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "Failed %d", totalFailed)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "All up to date")
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	// Show rehash hint if tools were updated
	if totalUpdated > 0 && !updateToolsFlags.dryRun && !updateToolsFlags.check {
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintf(cmd.OutOrStdout(), "%s Run 'goenv rehash' to update shims\n", utils.Emoji("💡 "))
	}

	if totalFailed > 0 {
		return fmt.Errorf("%d tool(s) failed to update", totalFailed)
	}

	return nil
}

func displayUpdateResults(cmd *cobra.Command, result *toolupdater.UpdateResult, goVersion string) error {
	if len(result.Checked) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "%s No tools found for Go %s\n",
			utils.EmojiOr("ℹ️  ", ""), goVersion)
		fmt.Fprintln(cmd.OutOrStdout(), "\nRun 'goenv tools default init' to set up automatic tool installation.")
		return nil
	}

	// Check mode: just show available updates
	if updateToolsFlags.check {
		fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n\n",
			utils.EmojiOr("🔍 ", ""),
			utils.BoldBlue("Checking for updates..."))

		availableUpdates := 0
		for _, check := range result.Checked {
			if check.UpdateAvailable {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s %s: %s → %s\n",
					utils.Yellow("⬆"),
					utils.BoldWhite(check.ToolName),
					utils.Gray(check.CurrentVersion),
					utils.Green(check.LatestVersion))
				availableUpdates++
			} else if updateToolsFlags.verbose {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s %s: %s (up to date)\n",
					utils.Green("✓"),
					utils.Gray(check.ToolName),
					utils.Gray(check.CurrentVersion))
			}
		}

		if availableUpdates == 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "\n%s All tools are up to date!\n",
				utils.EmojiOr("✅ ", ""))
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "\n%s %d update(s) available\n",
				utils.EmojiOr("💡 ", ""), availableUpdates)
			fmt.Fprintln(cmd.OutOrStdout(), "Run 'goenv tools update' to install updates")
		}
		return nil
	}

	// Dry run mode: show what would be updated
	if updateToolsFlags.dryRun {
		fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n\n",
			utils.EmojiOr("🔍 ", ""),
			utils.BoldBlue("Dry run - would update:"))

		if len(result.Updated) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "  (no updates needed)")
		} else {
			for _, toolName := range result.Updated {
				// Find the check info
				for _, check := range result.Checked {
					if check.ToolName == toolName {
						fmt.Fprintf(cmd.OutOrStdout(), "  %s %s: %s → %s\n",
							utils.Yellow("⬆"),
							utils.BoldWhite(check.ToolName),
							utils.Gray(check.CurrentVersion),
							utils.Green(check.LatestVersion))
						break
					}
				}
			}
		}

		if len(result.Skipped) > 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "\nWould skip:")
			for _, toolName := range result.Skipped {
				// Find the check info
				for _, check := range result.Checked {
					if check.ToolName == toolName {
						fmt.Fprintf(cmd.OutOrStdout(), "  %s %s: %s\n",
							utils.Gray("○"),
							utils.Gray(check.ToolName),
							utils.Gray(check.Reason))
						break
					}
				}
			}
		}
		return nil
	}

	// Actual update: show results
	fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n\n",
		utils.EmojiOr("🔄 ", ""),
		utils.BoldBlue("Updating tools..."))

	if len(result.Updated) > 0 {
		for _, toolName := range result.Updated {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s %s\n",
				utils.Green("✓"),
				utils.BoldWhite(toolName))
		}
	}

	if len(result.Failed) > 0 {
		fmt.Fprintln(cmd.OutOrStderr(), "\nFailed:")
		for _, toolName := range result.Failed {
			fmt.Fprintf(cmd.OutOrStderr(), "  %s %s\n",
				utils.Red("✗"),
				utils.Yellow(toolName))
		}
	}

	if len(result.Skipped) > 0 && updateToolsFlags.verbose {
		fmt.Fprintln(cmd.OutOrStdout(), "\nSkipped:")
		for _, toolName := range result.Skipped {
			for _, check := range result.Checked {
				if check.ToolName == toolName {
					fmt.Fprintf(cmd.OutOrStdout(), "  %s %s: %s\n",
						utils.Gray("○"),
						utils.Gray(check.ToolName),
						utils.Gray(check.Reason))
					break
				}
			}
		}
	}

	if len(result.Updated) == 0 && len(result.Failed) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "%s All tools are up to date!\n",
			utils.EmojiOr("✅ ", ""))
	}

	return nil
}
