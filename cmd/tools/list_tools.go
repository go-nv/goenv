package tools

import (
	"encoding/json"
	"fmt"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var (
	listToolsJSON   bool
	listAllVersions bool
)

var listToolsCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed tools for current or all versions",
	Long: `List all Go tools installed in version's GOPATH/bin.

By default, lists tools for the current Go version.
Use --all to list tools across all installed Go versions.

Flags:
  --all     List tools across all Go versions
  --json    Output results in JSON format`,
	RunE: runList,
}

func init() {
	listToolsCmd.Flags().BoolVar(&listToolsJSON, "json", false, "Output in JSON format")
	listToolsCmd.Flags().BoolVar(&listAllVersions, "all", false, "List tools across all Go versions")
}

func runList(cmd *cobra.Command, args []string) error {
	cfg := config.Load()

	// Determine target versions
	var targetVersions []string
	if listAllVersions {
		versions, err := getInstalledVersions(cfg)
		if err != nil {
			return err
		}
		if len(versions) == 0 {
			return fmt.Errorf("no Go versions installed")
		}
		targetVersions = versions
	} else {
		mgr := manager.NewManager(cfg)
		currentVersion, _, err := mgr.GetCurrentVersion()
		if err != nil {
			return fmt.Errorf("no Go version set: %w", err)
		}
		if currentVersion == "system" {
			fmt.Fprintln(cmd.OutOrStdout(), "Cannot list tools for system Go")
			fmt.Fprintln(cmd.OutOrStdout(), "Use 'which <tool>' or check your system's $GOPATH/bin")
			return nil
		}
		if err := mgr.ValidateVersion(currentVersion); err != nil {
			return fmt.Errorf("go version %s not installed", currentVersion)
		}
		targetVersions = []string{currentVersion}
	}

	// Collect tools for all versions
	type versionTools struct {
		Version string   `json:"version"`
		Tools   []string `json:"tools"`
	}

	var allVersionTools []versionTools
	totalTools := 0

	for _, version := range targetVersions {
		tools, err := getToolsForVersion(cfg, version)
		if err != nil {
			return fmt.Errorf("failed to list tools for %s: %w", version, err)
		}

		allVersionTools = append(allVersionTools, versionTools{
			Version: version,
			Tools:   tools,
		})
		totalTools += len(tools)
	}

	// Handle JSON output
	if listToolsJSON {
		type jsonOutput struct {
			SchemaVersion string         `json:"schema_version"`
			Versions      []versionTools `json:"versions"`
		}
		output := jsonOutput{
			SchemaVersion: "1",
			Versions:      allVersionTools,
		}
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	// Human-readable output
	if listAllVersions {
		fmt.Fprintf(cmd.OutOrStdout(), "\n%s %s\n\n",
			utils.EmojiOr("🔧 ", ""),
			utils.BoldBlue("Tools Across All Versions"))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "\n%s %s\n\n",
			utils.EmojiOr("🔧 ", ""),
			utils.BoldBlue("Tools for Go "+targetVersions[0]))
	}

	for _, vt := range allVersionTools {
		if listAllVersions {
			fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n",
				utils.BoldCyan("Go "+vt.Version+":"),
				utils.Gray(fmt.Sprintf("(%d tool(s))", len(vt.Tools))))
		}

		if len(vt.Tools) == 0 {
			if listAllVersions {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s\n\n", utils.Gray("no tools installed"))
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "no tools installed yet")
				fmt.Fprintln(cmd.OutOrStdout())
				fmt.Fprintln(cmd.OutOrStdout(), "Install a tool with:")
				fmt.Fprintln(cmd.OutOrStdout(), "  goenv tools install <package>@<version>")
			}
			continue
		}

		for _, tool := range vt.Tools {
			fmt.Fprintf(cmd.OutOrStdout(), "  • %s\n", tool)
		}

		if listAllVersions {
			fmt.Fprintln(cmd.OutOrStdout())
		}
	}

	if !listAllVersions {
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintf(cmd.OutOrStdout(), "Total: %d tool(s)\n", totalTools)
	} else if totalTools > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n",
			utils.Gray(fmt.Sprintf("Total: %d tool(s) across %d version(s)", totalTools, len(targetVersions))))
	}

	return nil
}
