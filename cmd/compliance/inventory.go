package compliance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/spf13/cobra"
)

var inventoryCmd = &cobra.Command{
	Use:     "inventory",
	Short:   "List installed Go versions and tools",
	GroupID: "system",
	Long: `List Go versions and tools installed by goenv for audit and compliance purposes.

This is NOT an SBOM generator - it's a simple inventory of what goenv has installed.
For project SBOMs, use 'goenv sbom project' with cyclonedx-gomod or syft.

Examples:
  # List installed Go versions
  goenv inventory go

  # Output as JSON
  goenv inventory go --json

  # Include SHA256 checksums
  goenv inventory go --checksums`,
}

var inventoryGoCmd = &cobra.Command{
	Use:   "go",
	Short: "List installed Go versions",
	Long:  `List Go versions installed by goenv with paths, installation dates, and optional checksums.`,
	RunE:  runInventoryGo,
}

var (
	inventoryJSON      bool
	inventoryChecksums bool
)

func init() {
	inventoryGoCmd.Flags().BoolVar(&inventoryJSON, "json", false, "Output as JSON")
	inventoryGoCmd.Flags().BoolVar(&inventoryChecksums, "checksums", false, "Include SHA256 checksums of go binaries")

	inventoryCmd.AddCommand(inventoryGoCmd)
	cmdpkg.RootCmd.AddCommand(inventoryCmd)
}

func runInventoryGo(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	versions, err := mgr.ListInstalledVersions()
	if err != nil {
		return fmt.Errorf("failed to list versions: %w", err)
	}

	if len(versions) == 0 {
		if inventoryJSON {
			fmt.Fprintln(cmd.OutOrStdout(), "[]")
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "No Go versions installed.")
		}
		return nil
	}

	// Collect inventory data
	inventory := make([]goInstallation, 0, len(versions))
	for _, version := range versions {
		install := collectGoInstallation(cfg, version, inventoryChecksums)
		inventory = append(inventory, install)
	}

	// Output
	if inventoryJSON {
		return outputInventoryJSON(cmd, inventory)
	}
	return outputInventoryText(cmd, inventory)
}

type goInstallation struct {
	Version     string    `json:"version"`
	Path        string    `json:"path"`
	BinaryPath  string    `json:"binary_path"`
	InstalledAt time.Time `json:"installed_at,omitempty"`
	SHA256      string    `json:"sha256,omitempty"`
	OS          string    `json:"os"`
	Arch        string    `json:"arch"`
}

func collectGoInstallation(cfg *config.Config, version string, includeChecksum bool) goInstallation {
	versionPath := filepath.Join(cfg.VersionsDir(), version)
	goBinary := filepath.Join(versionPath, "bin", "go")
	if runtime.GOOS == "windows" {
		goBinary += ".exe"
	}

	install := goInstallation{
		Version:    version,
		Path:       versionPath,
		BinaryPath: goBinary,
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
	}

	// Get installation time from directory modification time
	if info, err := os.Stat(versionPath); err == nil {
		install.InstalledAt = info.ModTime()
	}

	// Compute checksum if requested
	if includeChecksum {
		if checksum, err := computeSHA256(goBinary); err == nil {
			install.SHA256 = checksum
		}
	}

	return install
}

func outputInventoryJSON(cmd *cobra.Command, inventory []goInstallation) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(inventory)
}

func outputInventoryText(cmd *cobra.Command, inventory []goInstallation) error {
	fmt.Fprintln(cmd.OutOrStdout(), "═══════════════════════════════════════════════════════════════")
	fmt.Fprintln(cmd.OutOrStdout(), "               GOENV GO VERSION INVENTORY")
	fmt.Fprintln(cmd.OutOrStdout(), "═══════════════════════════════════════════════════════════════")
	fmt.Fprintln(cmd.OutOrStdout())

	for i, install := range inventory {
		fmt.Fprintf(cmd.OutOrStdout(), "%d. Go %s\n", i+1, install.Version)
		fmt.Fprintf(cmd.OutOrStdout(), "   Path:      %s\n", install.Path)
		fmt.Fprintf(cmd.OutOrStdout(), "   Binary:    %s\n", install.BinaryPath)
		fmt.Fprintf(cmd.OutOrStdout(), "   Platform:  %s/%s\n", install.OS, install.Arch)

		if !install.InstalledAt.IsZero() {
			fmt.Fprintf(cmd.OutOrStdout(), "   Installed: %s\n", install.InstalledAt.Format("2006-01-02 15:04:05"))
		}

		if install.SHA256 != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "   SHA256:    %s\n", install.SHA256)
		}

		fmt.Fprintln(cmd.OutOrStdout())
	}

	fmt.Fprintln(cmd.OutOrStdout(), "───────────────────────────────────────────────────────────────")
	fmt.Fprintf(cmd.OutOrStdout(), "Total: %d Go version(s) installed\n", len(inventory))
	fmt.Fprintln(cmd.OutOrStdout(), "═══════════════════════════════════════════════════════════════")

	return nil
}

func computeSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
