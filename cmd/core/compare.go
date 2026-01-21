package core

import (
	"fmt"
	"time"

	cmdpkg "github.com/go-nv/goenv/cmd"
	"github.com/go-nv/goenv/internal/cmdutil"
	"github.com/go-nv/goenv/internal/lifecycle"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var compareCmd = &cobra.Command{
	Use:   "compare <version1> <version2>",
	Short: "Compare two Go versions side-by-side",
	Long: `Compare two Go versions to see their differences in:
- Release dates and age
- Support status (current, near EOL, or EOL)
- Installation status
- Size on disk (if installed)
- Major changes between versions

This helps you decide whether to upgrade or which version to choose.`,
	Example: `  # Compare two versions
  goenv compare 1.21.5 1.22.3

  # Compare current version with latest
  goenv compare $(goenv version-name) $(goenv install-list --stable | head -1)

  # Compare installed versions
  goenv compare 1.20.5 1.21.13`,
	Args: cobra.ExactArgs(2),
	RunE: runCompare,
}

func init() {
	cmdpkg.RootCmd.AddCommand(compareCmd)
}

func runCompare(cmd *cobra.Command, args []string) error {
	version1 := args[0]
	version2 := args[1]

	ctx := cmdutil.GetContexts(cmd)
	cfg := ctx.Config
	mgr := ctx.Manager

	// Resolve version specs
	resolvedV1, err1 := mgr.ResolveVersionSpec(version1)
	if err1 != nil {
		resolvedV1 = version1
	}

	resolvedV2, err2 := mgr.ResolveVersionSpec(version2)
	if err2 != nil {
		resolvedV2 = version2
	}

	// Get installation status
	installed1 := mgr.IsVersionInstalled(resolvedV1)
	installed2 := mgr.IsVersionInstalled(resolvedV2)

	// Get lifecycle information
	lifecycle1, hasLifecycle1 := lifecycle.GetVersionInfo(resolvedV1)
	lifecycle2, hasLifecycle2 := lifecycle.GetVersionInfo(resolvedV2)

	// Print comparison header
	fmt.Fprintf(cmd.OutOrStdout(), "\n%s Comparing Go Versions\n\n", utils.Emoji("⚖️  "))

	// Version names
	printComparisonRow(cmd, "Version",
		utils.BoldBlue(resolvedV1),
		utils.BoldBlue(resolvedV2))

	// Installation status
	status1 := formatInstallStatus(installed1)
	status2 := formatInstallStatus(installed2)
	printComparisonRow(cmd, "Installed", status1, status2)

	// Release dates
	if hasLifecycle1 && hasLifecycle2 {
		date1 := lifecycle1.ReleaseDate.Format("2006-01-02")
		date2 := lifecycle2.ReleaseDate.Format("2006-01-02")
		printComparisonRow(cmd, "Released", date1, date2)

		// Age comparison
		age1 := formatAge(lifecycle1.ReleaseDate)
		age2 := formatAge(lifecycle2.ReleaseDate)
		printComparisonRow(cmd, "Age", age1, age2)

		// Support status
		support1 := formatSupportStatus(lifecycle1.Status)
		support2 := formatSupportStatus(lifecycle2.Status)
		printComparisonRow(cmd, "Support", support1, support2)

		// EOL dates
		if lifecycle1.Status != lifecycle.StatusCurrent || lifecycle2.Status != lifecycle.StatusCurrent {
			eol1 := formatEOLDate(lifecycle1)
			eol2 := formatEOLDate(lifecycle2)
			printComparisonRow(cmd, "EOL Date", eol1, eol2)
		}
	}

	// Size comparison (if both installed)
	if installed1 && installed2 {
		size1, _ := calculateDirSize(cfg.VersionDir(resolvedV1))
		size2, _ := calculateDirSize(cfg.VersionDir(resolvedV2))

		printComparisonRow(cmd, "Size",
			formatSize(size1),
			formatSize(size2))

		// Size difference
		diff := size2 - size1
		if diff != 0 {
			diffStr := formatSizeDiff(diff)
			fmt.Fprintf(cmd.OutOrStdout(), "\n  %s Size difference: %s\n",
				utils.Emoji("📊"), diffStr)
		}
	}

	// Version difference analysis
	fmt.Fprintln(cmd.OutOrStdout())
	analyzeVersionDifference(cmd, resolvedV1, resolvedV2, hasLifecycle1, hasLifecycle2, lifecycle1, lifecycle2)

	// Recommendations
	fmt.Fprintln(cmd.OutOrStdout())
	provideRecommendations(cmd, resolvedV1, resolvedV2, installed1, installed2,
		hasLifecycle1, hasLifecycle2, lifecycle1, lifecycle2)

	return nil
}

// printComparisonRow prints a comparison row with aligned columns
func printComparisonRow(cmd *cobra.Command, label, value1, value2 string) {
	fmt.Fprintf(cmd.OutOrStdout(), "  %-12s %s  vs  %s\n", label+":", value1, value2)
}

// formatInstallStatus formats installation status with color
func formatInstallStatus(installed bool) string {
	if installed {
		return utils.Green("✓ Installed")
	}
	return utils.Gray("✗ Not installed")
}

// formatAge calculates and formats version age
func formatAge(releaseDate time.Time) string {
	age := time.Since(releaseDate)

	years := int(age.Hours() / 24 / 365)
	months := int(age.Hours()/24/30) % 12

	if years > 0 {
		if months > 0 {
			return fmt.Sprintf("%dy %dmo", years, months)
		}
		return fmt.Sprintf("%dy", years)
	}

	if months > 0 {
		return fmt.Sprintf("%dmo", months)
	}

	days := int(age.Hours() / 24)
	if days > 0 {
		return fmt.Sprintf("%dd", days)
	}

	return "New"
}

// formatSupportStatus formats support status with color
func formatSupportStatus(status lifecycle.SupportStatus) string {
	switch status {
	case lifecycle.StatusCurrent:
		return utils.Green("🟢 Current")
	case lifecycle.StatusNearEOL:
		return utils.Yellow("🟡 Near EOL")
	case lifecycle.StatusEOL:
		return utils.Red("🔴 EOL")
	default:
		return utils.Gray("❓ Unknown")
	}
}

// formatEOLDate formats EOL date
func formatEOLDate(info lifecycle.VersionInfo) string {
	if info.Status == lifecycle.StatusCurrent {
		return utils.Gray("N/A")
	}
	return info.EOLDate.Format("2006-01-02")
}

// formatSizeDiff formats size difference with sign
func formatSizeDiff(diff int64) string {
	if diff > 0 {
		return utils.Yellow(fmt.Sprintf("+%s (larger)", formatSize(diff)))
	} else if diff < 0 {
		return utils.Green(fmt.Sprintf("-%s (smaller)", formatSize(-diff)))
	}
	return "Same"
}

// analyzeVersionDifference provides analysis of version differences
func analyzeVersionDifference(cmd *cobra.Command, v1, v2 string,
	hasL1, hasL2 bool, l1, l2 lifecycle.VersionInfo) {

	fmt.Fprintf(cmd.OutOrStdout(), "%s Version Analysis\n", utils.Emoji("🔍"))

	// Parse versions
	major1, minor1, patch1 := parseVersion(v1)
	major2, minor2, patch2 := parseVersion(v2)

	// Major version difference
	if major1 != major2 {
		fmt.Fprintf(cmd.OutOrStdout(), "  %s Major version change (%d → %d) - Significant changes expected\n",
			utils.Emoji("⚠️  "), major1, major2)
		return
	}

	// Minor version difference
	minorDiff := minor2 - minor1
	if minorDiff > 0 {
		plural := ""
		if minorDiff > 1 {
			plural = "s"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  %s %d minor version%s newer (1.%d → 1.%d)\n",
			utils.Emoji("📈"), minorDiff, plural, minor1, minor2)

		if minorDiff >= 3 {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s Multiple minor versions - review release notes for breaking changes\n",
				utils.Emoji("💡"))
		}
	} else if minorDiff < 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "  %s Downgrade by %d minor version(s) (1.%d → 1.%d)\n",
			utils.Emoji("⬇️  "), -minorDiff, minor1, minor2)
	} else {
		// Same minor version, different patch
		patchDiff := patch2 - patch1
		if patchDiff > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s Patch upgrade (+%d) - bug fixes and security updates\n",
				utils.Emoji("🔧"), patchDiff)
		} else if patchDiff < 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s Patch downgrade (%d) - not recommended\n",
				utils.Emoji("⚠️  "), patchDiff)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s Same version\n", utils.Emoji("="))
		}
	}

	// Time difference
	if hasL1 && hasL2 {
		timeDiff := l2.ReleaseDate.Sub(l1.ReleaseDate)
		if timeDiff > 0 {
			months := int(timeDiff.Hours() / 24 / 30)
			if months > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s Released %d months apart\n",
					utils.Emoji("📅"), months)
			}
		}
	}
}

// provideRecommendations gives upgrade/downgrade recommendations
func provideRecommendations(cmd *cobra.Command, v1, v2 string,
	inst1, inst2 bool, hasL1, hasL2 bool, l1, l2 lifecycle.VersionInfo) {

	fmt.Fprintf(cmd.OutOrStdout(), "%s Recommendations\n", utils.Emoji("💡"))

	// Check if comparing same version
	if v1 == v2 {
		fmt.Fprintf(cmd.OutOrStdout(), "  • Both versions are the same\n")
		return
	}

	// Parse for comparison
	major1, minor1, patch1 := parseVersion(v1)
	major2, minor2, patch2 := parseVersion(v2)

	isNewer := (major2 > major1) ||
		(major2 == major1 && minor2 > minor1) ||
		(major2 == major1 && minor2 == minor1 && patch2 > patch1)

	// EOL warnings
	if hasL1 && l1.Status == lifecycle.StatusEOL {
		fmt.Fprintf(cmd.OutOrStdout(), "  • %s is EOL - upgrade recommended\n", utils.Yellow(v1))
	}

	if hasL2 && l2.Status == lifecycle.StatusEOL {
		fmt.Fprintf(cmd.OutOrStdout(), "  • %s is EOL - consider newer version\n", utils.Yellow(v2))
	}

	// Near EOL warnings
	if hasL1 && l1.Status == lifecycle.StatusNearEOL {
		fmt.Fprintf(cmd.OutOrStdout(), "  • %s approaching EOL - plan upgrade soon\n", utils.Yellow(v1))
	}

	if hasL2 && l2.Status == lifecycle.StatusNearEOL {
		fmt.Fprintf(cmd.OutOrStdout(), "  • %s approaching EOL - consider latest\n", utils.Yellow(v2))
	}

	// Upgrade recommendation
	if isNewer {
		if hasL2 && l2.Status == lifecycle.StatusCurrent {
			fmt.Fprintf(cmd.OutOrStdout(), "  • %s Upgrade to %s recommended (current, supported)\n",
				utils.Emoji("✅"), utils.Green(v2))
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "  • %s is newer than %s\n",
				utils.Green(v2), v1)
		}
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "  • %s Downgrade to %s not recommended\n",
			utils.Emoji("⚠️  "), utils.Yellow(v2))
	}

	// Installation suggestions
	if !inst2 && isNewer {
		fmt.Fprintf(cmd.OutOrStdout(), "  • Install %s: %s\n",
			v2, utils.Cyan(fmt.Sprintf("goenv install %s", v2)))
	}

	// Release notes
	fmt.Fprintf(cmd.OutOrStdout(), "\n  📖 Release notes:\n")
	fmt.Fprintf(cmd.OutOrStdout(), "     %s: %s\n", v1,
		utils.Cyan(fmt.Sprintf("https://go.dev/doc/go%s", utils.ExtractMajorMinor(v1))))
	fmt.Fprintf(cmd.OutOrStdout(), "     %s: %s\n", v2,
		utils.Cyan(fmt.Sprintf("https://go.dev/doc/go%s", utils.ExtractMajorMinor(v2))))
}

// parseVersion parses version string into major, minor, patch
func parseVersion(ver string) (major, minor, patch int) {
	major, minor, patch, err := utils.ParseVersionTuple(ver)
	if err != nil {
		// Return zeros for invalid versions - comparison will still work
		return 0, 0, 0
	}
	return major, minor, patch
}
