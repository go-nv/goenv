package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	cmdpkg "github.com/go-nv/goenv/cmd"
	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/lifecycle"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info <version>",
	Short: "Show detailed information about a Go version",
	Long: `Display comprehensive information about a specific Go version, including:
- Installation status
- Release date and lifecycle information
- Installation path
- Size on disk
- Support status (current, near EOL, or EOL)
- Release notes link`,
	Example: `  # Show info for a specific version
  goenv info 1.21.5

  # Show info for current version
  goenv info $(goenv version-name)

  # Show info for latest installed
  goenv info $(goenv versions --bare | tail -1)`,
	Args: cobra.ExactArgs(1),
	RunE: runInfo,
}

var infoFlags struct {
	json bool
}

func init() {
	cmdpkg.RootCmd.AddCommand(infoCmd)
	infoCmd.Flags().BoolVar(&infoFlags.json, "json", false, "Output in JSON format")
}

func runInfo(cmd *cobra.Command, args []string) error {
	version := args[0]
	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	// Resolve version spec (handles aliases, "latest", etc.)
	resolvedVersion, err := mgr.ResolveVersionSpec(version)
	if err != nil {
		// If resolution fails, try to show info anyway for the literal version
		resolvedVersion = version
	}

	// Check if version is installed
	isInstalled := mgr.IsVersionInstalled(resolvedVersion)
	var installPath string
	var sizeOnDisk int64

	if isInstalled {
		installPath = filepath.Join(cfg.VersionsDir(), resolvedVersion)
		// Calculate size
		sizeOnDisk, _ = calculateDirSize(installPath)
	}

	// Get lifecycle information
	lifecycleInfo, hasLifecycleInfo := lifecycle.GetVersionInfo(resolvedVersion)

	// Format output
	if infoFlags.json {
		return outputJSON(resolvedVersion, isInstalled, installPath, sizeOnDisk, lifecycleInfo, hasLifecycleInfo)
	}

	return outputHuman(cmd, resolvedVersion, isInstalled, installPath, sizeOnDisk, lifecycleInfo, hasLifecycleInfo)
}

func outputHuman(cmd *cobra.Command, version string, installed bool, path string, size int64, info lifecycle.VersionInfo, hasInfo bool) error {
	// Header
	fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", utils.Emoji("ℹ️  "), utils.BoldBlue(fmt.Sprintf("Go %s", version)))
	fmt.Fprintln(cmd.OutOrStdout())

	// Installation Status
	if installed {
		fmt.Fprintf(cmd.OutOrStdout(), "  %s Status:       %s\n", utils.Emoji("✅"), utils.Green("Installed"))
		fmt.Fprintf(cmd.OutOrStdout(), "  📁 Install path: %s\n", utils.Cyan(path))
		if size > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "  💾 Size on disk: %s\n", formatSize(size))
		}
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "  %s Status:       %s\n", utils.Emoji("❌"), utils.Yellow("Not installed"))
		fmt.Fprintf(cmd.OutOrStdout(), "  💡 Install with: %s\n", utils.Cyan(fmt.Sprintf("goenv install %s", version)))
	}

	// Lifecycle Information
	if hasInfo {
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintf(cmd.OutOrStdout(), "  📅 Released:     %s\n", info.ReleaseDate.Format("2006-01-02"))

		// Support status
		switch info.Status {
		case lifecycle.StatusCurrent:
			fmt.Fprintf(cmd.OutOrStdout(), "  🟢 Support:      %s\n", utils.Green("Current (fully supported)"))
		case lifecycle.StatusNearEOL:
			fmt.Fprintf(cmd.OutOrStdout(), "  🟡 Support:      %s\n", utils.Yellow(fmt.Sprintf("Near EOL (ends %s)", info.EOLDate.Format("2006-01-02"))))
			if info.SecurityOnly {
				fmt.Fprintf(cmd.OutOrStdout(), "                   %s\n", utils.Gray("Security updates only"))
			}
		case lifecycle.StatusEOL:
			fmt.Fprintf(cmd.OutOrStdout(), "  🔴 Support:      %s\n", utils.Red(fmt.Sprintf("End of Life (ended %s)", info.EOLDate.Format("2006-01-02"))))
			if info.Recommended != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "  ⚠️  Recommended:  %s\n", utils.Cyan(fmt.Sprintf("Upgrade to %s", info.Recommended)))
			}
		case lifecycle.StatusUnknown:
			fmt.Fprintf(cmd.OutOrStdout(), "  ❓ Support:      %s\n", utils.Gray("Unknown (possibly newer version)"))
		}
	}

	// Release Notes
	fmt.Fprintln(cmd.OutOrStdout())
	majorMinor := utils.ExtractMajorMinor(version)
	if majorMinor != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "  📖 Release notes: %s\n", utils.Cyan(fmt.Sprintf("https://go.dev/doc/go%s", majorMinor)))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "  📖 Release notes: %s\n", utils.Cyan("https://go.dev/doc/devel/release"))
	}

	// Download page
	fmt.Fprintf(cmd.OutOrStdout(), "  📦 Downloads:     %s\n", utils.Cyan("https://go.dev/dl/"))

	return nil
}

func outputJSON(version string, installed bool, path string, size int64, info lifecycle.VersionInfo, hasInfo bool) error {
	type output struct {
		Version     string `json:"version"`
		Installed   bool   `json:"installed"`
		InstallPath string `json:"install_path,omitempty"`
		SizeBytes   int64  `json:"size_bytes,omitempty"`
		SizeHuman   string `json:"size_human,omitempty"`
		ReleaseDate string `json:"release_date,omitempty"`
		EOLDate     string `json:"eol_date,omitempty"`
		Status      string `json:"status,omitempty"`
		Recommended string `json:"recommended,omitempty"`
		ReleaseURL  string `json:"release_url"`
		DownloadURL string `json:"download_url"`
	}

	out := output{
		Version:     version,
		Installed:   installed,
		ReleaseURL:  fmt.Sprintf("https://go.dev/doc/go%s", utils.ExtractMajorMinor(version)),
		DownloadURL: "https://go.dev/dl/",
	}

	if installed {
		out.InstallPath = path
		out.SizeBytes = size
		out.SizeHuman = formatSize(size)
	}

	if hasInfo {
		out.ReleaseDate = info.ReleaseDate.Format("2006-01-02")
		out.EOLDate = info.EOLDate.Format("2006-01-02")

		switch info.Status {
		case lifecycle.StatusCurrent:
			out.Status = "current"
		case lifecycle.StatusNearEOL:
			out.Status = "near_eol"
		case lifecycle.StatusEOL:
			out.Status = "eol"
		case lifecycle.StatusUnknown:
			out.Status = "unknown"
		}

		out.Recommended = info.Recommended
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(out)
}

// calculateDirSize recursively calculates the total size of a directory
// Deprecated: Use utils.CalculateDirectorySize instead
func calculateDirSize(path string) (int64, error) {
	return utils.CalculateDirectorySize(path)
}

// formatSize formats bytes into human-readable format
// Deprecated: Use utils.FormatBytes instead
func formatSize(bytes int64) string {
	return utils.FormatBytes(bytes)
}
