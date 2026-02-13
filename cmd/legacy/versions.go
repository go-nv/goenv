package legacy

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/cmdutil"
	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var versionsCmd = &cobra.Command{
	Use:   "versions",
	Short: "List all installed Go versions (deprecated: use 'goenv list')",
	Long: `Show all locally installed Go versions with the current version highlighted.

Usage: goenv versions [--bare] [--skip-aliases] [--used]

Lists all Go versions found in '$GOENV_ROOT/versions/*'.

Options:
  --bare          List version numbers only (one per line)
  --skip-aliases  Skip showing system Go as an alias
  --used          Scan current directory tree for version usage
  --depth N       Maximum scan depth (default: 3, used with --used)
  --json          Output in JSON format

Examples:
  # List installed versions
  goenv versions

  # Check which versions are used by your projects
  cd ~/work
  goenv versions --used

  # Quick scan (only immediate subdirectories)
  goenv versions --used --depth 1

  # Deep scan (search deeper)
  goenv versions --used --depth 5`,
	RunE:    RunVersions,
	GroupID: string(cmdpkg.GroupLegacy),
}

var VersionsFlags struct {
	Bare        bool
	SkipAliases bool
	Complete    bool
	Json        bool
	Used        bool
	Depth       int
}

func init() {
	cmdpkg.RootCmd.AddCommand(versionsCmd)
	versionsCmd.Flags().BoolVarP(&VersionsFlags.Bare, "bare", "b", false, "Display bare version numbers only")
	versionsCmd.Flags().BoolVar(&VersionsFlags.SkipAliases, "skip-aliases", false, "Skip aliases")
	versionsCmd.Flags().BoolVar(&VersionsFlags.Json, "json", false, "Output in JSON format")
	versionsCmd.Flags().BoolVar(&VersionsFlags.Used, "used", false, "Scan current directory tree for version usage")
	versionsCmd.Flags().IntVar(&VersionsFlags.Depth, "depth", 3, "Maximum scan depth (used with --used)")
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

// findLatestVersion finds the highest semantic version from a list of versions
func findLatestVersion(versions []string) string {
	if len(versions) == 0 {
		return ""
	}

	latest := versions[0]
	for _, v := range versions[1:] {
		// Skip system and special versions
		if v == manager.SystemVersion || latest == manager.SystemVersion {
			continue
		}

		// Compare versions using semantic version logic
		if utils.CompareGoVersions(v, latest) > 0 {
			latest = v
		}
	}

	return latest
}

func RunVersions(cmd *cobra.Command, args []string) error {
	// Handle completion mode
	if VersionsFlags.Complete {
		fmt.Fprintln(cmd.OutOrStdout(), "--bare")
		fmt.Fprintln(cmd.OutOrStdout(), "--skip-aliases")
		fmt.Fprintln(cmd.OutOrStdout(), "--used")
		return nil
	}

	// Deprecation warning
	fmt.Fprintf(cmd.OutOrStderr(), "%sDeprecation warning: 'goenv versions' is a legacy command. Use 'goenv list' instead.\n", utils.Emoji("⚠️  "))
	fmt.Fprintf(cmd.OutOrStderr(), "  Modern command: goenv list\n")
	fmt.Fprintf(cmd.OutOrStderr(), "  See: goenv help list\n\n")

	// Handle invalid arguments (BATS test expects usage error)
	if err := cmdutil.ValidateMaxArgs(args, 0, "no arguments"); err != nil {
		return fmt.Errorf("usage: goenv versions [--bare] [--skip-aliases] [--used]")
	}

	ctx := cmdutil.GetContexts(cmd)
	cfg := ctx.Config
	mgr := ctx.Manager

	// If --used flag is set, show usage analysis
	if VersionsFlags.Used {
		return runVersionsWithUsage(cmd, mgr, cfg)
	}

	versions, err := mgr.ListInstalledVersions()
	if err != nil {
		return errors.FailedTo("list versions", err)
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
					return cmdutil.OutputJSON(cmd.OutOrStdout(), output)
				}
				return nil
			}

			// Provide helpful message when no versions are installed
			fmt.Fprintf(cmd.OutOrStderr(), "%s%s\n\n", utils.Emoji("ℹ️  "), utils.Yellow("no Go versions installed yet"))
			fmt.Fprintf(cmd.OutOrStderr(), "%s\n", utils.BoldBlue("Get started by installing a version:"))
			fmt.Fprintf(cmd.OutOrStderr(), "  %s        %s Install latest Go\n", utils.Cyan("goenv install"), utils.Gray("→"))
			fmt.Fprintf(cmd.OutOrStderr(), "  %s %s Install specific version\n", utils.Cyan("goenv install 1.21.5"), utils.Gray("→"))
			fmt.Fprintf(cmd.OutOrStderr(), "  goenv install -l     %s List available versions\n\n", utils.Emoji("→"))
			fmt.Fprintf(cmd.OutOrStderr(), "After installing, set it as your default:\n")
			fmt.Fprintf(cmd.OutOrStderr(), "  goenv global <version>\n")

			return errors.NoVersionsInstalled()
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
		if hasSystemGo || currentVersion == manager.SystemVersion {
			displaySource := simplifySource(source, cfg)
			items = append(items, versionInfo{
				Version:  "system",
				Active:   currentVersion == manager.SystemVersion,
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

		return cmdutil.OutputJSON(cmd.OutOrStdout(), output)
	}

	// Show system version first (if available and not bare mode)
	// Only show system if: (1) system go exists, OR (2) it's the current version
	showSystem := (hasSystemGo || currentVersion == manager.SystemVersion) && !VersionsFlags.Bare
	if showSystem {
		prefix := "  "
		suffix := ""

		if currentVersion == manager.SystemVersion {
			prefix = "* "
			displaySource := simplifySource(source, cfg)
			if displaySource != "" {
				// Only show source if a file actually set it
				suffix = fmt.Sprintf(" (set by %s)", displaySource)
			}
		}

		fmt.Fprintf(cmd.OutOrStdout(), "%ssystem%s\n", prefix, suffix)
	}

	// Find the latest version (highest semantic version)
	latestVersion := findLatestVersion(versions)

	// Display installed versions
	for _, version := range versions {
		if VersionsFlags.Bare {
			fmt.Fprintln(cmd.OutOrStdout(), version)
		} else {
			prefix := "  "
			suffix := ""
			isCurrent := version == currentVersion
			isLatest := version == latestVersion

			if isCurrent {
				prefix = utils.Green("* ")
				displaySource := simplifySource(source, cfg)
				if displaySource != "" {
					suffix = fmt.Sprintf(" %s", utils.Gray(fmt.Sprintf("(set by %s)", displaySource)))
				}
			}

			// Add latest indicator for the newest installed version
			if isLatest && !isCurrent {
				suffix = fmt.Sprintf(" %s", utils.Gray("(latest installed)"))
			} else if isLatest && isCurrent {
				// Both current and latest
				if suffix != "" {
					// Append to existing suffix
					suffix = suffix + utils.Gray(" [latest]")
				} else {
					suffix = fmt.Sprintf(" %s", utils.Gray("[latest]"))
				}
			}

			// Format version with color if it's current
			versionDisplay := version
			if isCurrent {
				versionDisplay = utils.Cyan(version)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "%s%s%s\n", prefix, versionDisplay, suffix)
		}
	}

	// Display aliases (unless --skip-aliases or --bare)
	if !VersionsFlags.SkipAliases && !VersionsFlags.Bare {
		aliases, err := mgr.ListAliases()
		if err != nil {
			return errors.FailedTo("list aliases", err)
		}

		if len(aliases) > 0 {
			// Sort alias names for consistent output
			aliasNames := make([]string, 0, len(aliases))
			for name := range aliases {
				aliasNames = append(aliasNames, name)
			}
			slices.Sort(aliasNames)

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

// runVersionsWithUsage displays versions with project usage information
func runVersionsWithUsage(cmd *cobra.Command, mgr *manager.Manager, cfg *config.Config) error {
	cwd, err := os.Getwd()
	if err != nil {
		return errors.FailedTo("get current directory", err)
	}

	// Get installed versions
	versions, err := mgr.ListInstalledVersions()
	if err != nil {
		return errors.FailedTo("list versions", err)
	}

	if len(versions) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "%sno Go versions installed.\n", utils.Emoji("ℹ️  "))
		return nil
	}

	// Show scanning message
	fmt.Fprintf(cmd.OutOrStdout(), "%sScanning for Go projects in %s...\n\n", utils.Emoji("🔍 "), cwd)

	// Scan for projects
	projects, err := manager.ScanProjects(cwd, VersionsFlags.Depth)
	if err != nil {
		return errors.FailedTo("scan directories", err)
	}

	// Build usage map: version -> []projects
	usageMap := make(map[string][]manager.ProjectInfo)
	for _, proj := range projects {
		usageMap[proj.Version] = append(usageMap[proj.Version], proj)
	}

	// Get current version for highlighting
	currentVersion, _, _ := mgr.GetCurrentVersion()

	// Display versions with usage
	fmt.Fprintf(cmd.OutOrStdout(), "%sInstalled versions:\n", utils.Emoji("📊 "))

	for _, ver := range versions {
		prefix := "  "
		if ver == currentVersion {
			prefix = utils.Green("* ")
		}

		projs := usageMap[ver]
		status := ""
		if len(projs) > 0 {
			status = fmt.Sprintf(" - %s Used by %d project(s)", utils.Emoji("✓"), len(projs))
		} else {
			status = fmt.Sprintf(" - %s Not found (may be safe to remove)", utils.Emoji("⚠️  "))
		}

		versionDisplay := ver
		if ver == currentVersion {
			versionDisplay = utils.Cyan(ver)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "%s%s%s\n", prefix, versionDisplay, status)
	}

	// Show project details if any found
	if len(projects) > 0 {
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintf(cmd.OutOrStdout(), "%sProjects found:\n", utils.Emoji("📁 "))

		// Sort versions for consistent display
		sortedVersions := make([]string, 0, len(usageMap))
		for ver := range usageMap {
			sortedVersions = append(sortedVersions, ver)
		}
		slices.SortFunc(sortedVersions, func(a, b string) int {
			return utils.CompareGoVersions(a, b)
		})

		// Display projects grouped by version
		for _, ver := range sortedVersions {
			projs := usageMap[ver]
			fmt.Fprintf(cmd.OutOrStdout(), "  Go %s:\n", ver)
			for _, proj := range projs {
				relPath, err := filepath.Rel(cwd, proj.Path)
				if err != nil {
					relPath = proj.Path
				}
				// Ensure we show it as a directory
				if !filepath.IsAbs(relPath) && relPath != "." {
					relPath = relPath + "/"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "    %s %s (%s)\n", utils.Emoji("•"), relPath, proj.Source)
			}
		}
	} else {
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintf(cmd.OutOrStdout(), "%sNo projects found in current directory tree\n", utils.Emoji("ℹ️  "))
	}

	// Show tips
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "%sTips:\n", utils.Emoji("💡 "))
	fmt.Fprintln(cmd.OutOrStdout(), "  • Navigate to your projects directory before running this command")
	fmt.Fprintln(cmd.OutOrStdout(), "  • This only scans current directory tree, not entire system")
	fmt.Fprintf(cmd.OutOrStdout(), "  • Use --depth to control scan depth (current: %d)\n", VersionsFlags.Depth)
	if len(usageMap) < len(versions) {
		fmt.Fprintln(cmd.OutOrStdout(), "  • Versions marked 'Not found' may be used elsewhere or safe to remove")
	}

	return nil
}
