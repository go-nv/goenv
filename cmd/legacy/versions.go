package legacy

import (
	"encoding/json"
	"fmt"
	"sort"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var versionsCmd = &cobra.Command{
	Use:    "versions",
	Short:  "List all installed Go versions",
	Long:   "Show all locally installed Go versions with the current version highlighted",
	RunE:   RunVersions,
	Hidden: true, // Legacy command - use 'goenv list' instead
}

var VersionsFlags struct {
	Bare        bool
	SkipAliases bool
	Complete    bool
	Json        bool
}

func init() {
	cmdpkg.RootCmd.AddCommand(versionsCmd)
	versionsCmd.Flags().BoolVarP(&VersionsFlags.Bare, "bare", "b", false, "Display bare version numbers only")
	versionsCmd.Flags().BoolVar(&VersionsFlags.SkipAliases, "skip-aliases", false, "Skip aliases")
	versionsCmd.Flags().BoolVar(&VersionsFlags.Json, "json", false, "Output in JSON format")
	versionsCmd.Flags().BoolVar(&VersionsFlags.Complete, "complete", false, "Internal flag for shell completions")
	_ = versionsCmd.Flags().MarkHidden("complete")
	helptext.SetCommandHelp(versionsCmd)
}

// simplifySource returns a simplified display name for the version source
func simplifySource(source string, cfg *config.Config) string {
	if source == "" {
		return ""
	}

	globalVersionFile := cfg.GlobalVersionFile()
	if source == globalVersionFile {
		return "global"
	}

	return source
}

func RunVersions(cmd *cobra.Command, args []string) error {
	// Handle completion mode
	if VersionsFlags.Complete {
		fmt.Fprintln(cmd.OutOrStdout(), "--bare")
		fmt.Fprintln(cmd.OutOrStdout(), "--skip-aliases")
		return nil
	}

	// Deprecation warning
	fmt.Fprintf(cmd.OutOrStderr(), "%sDeprecation warning: 'goenv versions' is a legacy command. Use 'goenv list' instead.\n", utils.Emoji("⚠️  "))
	fmt.Fprintf(cmd.OutOrStderr(), "  Modern command: goenv list\n")
	fmt.Fprintf(cmd.OutOrStderr(), "  See: goenv help list\n\n")

	// Handle invalid arguments (BATS test expects usage error)
	if len(args) > 0 {
		return fmt.Errorf("Usage: goenv versions [--bare] [--skip-aliases]")
	}

	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	versions, err := mgr.ListInstalledVersions()
	if err != nil {
		return fmt.Errorf("failed to list versions: %w", err)
	}

	hasSystemGo := mgr.HasSystemGo()

	// Handle case when no versions installed
	if len(versions) == 0 {
		if !hasSystemGo {
			if VersionsFlags.Bare || VersionsFlags.Json {
				// BATS test: bare mode prints empty output when no versions and no system go
				// JSON mode: return empty array
				if VersionsFlags.Json {
					type versionsOutput struct {
						SchemaVersion string        `json:"schema_version"`
						Versions      []interface{} `json:"versions"`
					}
					output := versionsOutput{
						SchemaVersion: "1",
						Versions:      []interface{}{},
					}
					encoder := json.NewEncoder(cmd.OutOrStdout())
					encoder.SetIndent("", "  ")
					return encoder.Encode(output)
				}
				return nil
			}
			return fmt.Errorf("Warning: no Go detected on the system")
		}

		// Only system version available
		if VersionsFlags.Bare {
			// BATS test: bare mode prints empty when only system available
			return nil
		}

		// Get current version to determine source
		_, versionSource, _ := mgr.GetCurrentVersion()

		// Show system version with source info
		displaySource := simplifySource(versionSource, cfg)
		if displaySource == "" {
			// Empty source means default behavior (no version file exists)
			fmt.Fprintf(cmd.OutOrStdout(), "* system\n")
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "* system (set by %s)\n", displaySource)
		}
		return nil
	}

	// Get current version to highlight it
	currentVersion, source, err := mgr.GetCurrentVersion()
	if err != nil {
		// If no version is set, default to empty (no highlighting)
		currentVersion = ""
	}

	// Handle JSON output
	if VersionsFlags.Json {
		type versionInfo struct {
			Version  string `json:"version"`
			Active   bool   `json:"active"`
			Source   string `json:"source,omitempty"`
			IsSystem bool   `json:"is_system,omitempty"`
		}

		type versionsOutput struct {
			SchemaVersion string        `json:"schema_version"`
			Versions      []versionInfo `json:"versions"`
		}

		var items []versionInfo

		// Add system version if available
		if hasSystemGo || currentVersion == "system" {
			displaySource := simplifySource(source, cfg)
			items = append(items, versionInfo{
				Version:  "system",
				Active:   currentVersion == "system",
				Source:   displaySource,
				IsSystem: true,
			})
		}

		// Add installed versions
		for _, v := range versions {
			displaySource := ""
			if v == currentVersion {
				displaySource = simplifySource(source, cfg)
			}
			items = append(items, versionInfo{
				Version: v,
				Active:  v == currentVersion,
				Source:  displaySource,
			})
		}

		output := versionsOutput{
			SchemaVersion: "1",
			Versions:      items,
		}

		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	// Show system version first (if available and not bare mode)
	// Only show system if: (1) system go exists, OR (2) it's the current version
	showSystem := (hasSystemGo || currentVersion == "system") && !VersionsFlags.Bare
	if showSystem {
		prefix := "  "
		suffix := ""

		if currentVersion == "system" {
			prefix = "* "
			displaySource := simplifySource(source, cfg)
			if displaySource != "" {
				// Only show source if a file actually set it
				suffix = fmt.Sprintf(" (set by %s)", displaySource)
			}
		}

		fmt.Fprintf(cmd.OutOrStdout(), "%ssystem%s\n", prefix, suffix)
	}

	// Display installed versions
	for _, version := range versions {
		if VersionsFlags.Bare {
			fmt.Fprintln(cmd.OutOrStdout(), version)
		} else {
			prefix := "  "
			suffix := ""

			if version == currentVersion {
				prefix = "* "
				displaySource := simplifySource(source, cfg)
				if displaySource != "" {
					suffix = fmt.Sprintf(" (set by %s)", displaySource)
				}
			}

			fmt.Fprintf(cmd.OutOrStdout(), "%s%s%s\n", prefix, version, suffix)
		}
	}

	// Display aliases (unless --skip-aliases or --bare)
	if !VersionsFlags.SkipAliases && !VersionsFlags.Bare {
		aliases, err := mgr.ListAliases()
		if err != nil {
			return fmt.Errorf("failed to list aliases: %w", err)
		}

		if len(aliases) > 0 {
			// Sort alias names for consistent output
			aliasNames := make([]string, 0, len(aliases))
			for name := range aliases {
				aliasNames = append(aliasNames, name)
			}
			sort.Strings(aliasNames)

			// Display aliases section
			fmt.Fprintln(cmd.OutOrStdout())
			fmt.Fprintln(cmd.OutOrStdout(), "Aliases:")
			for _, name := range aliasNames {
				targetVersion := aliases[name]
				prefix := "  "

				// Check if this alias points to the current version
				if targetVersion == currentVersion {
					prefix = "* "
				}

				fmt.Fprintf(cmd.OutOrStdout(), "%s%s -> %s\n", prefix, name, targetVersion)
			}
		}
	}

	return nil
}
