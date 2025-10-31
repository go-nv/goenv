package compliance

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var sbomCmd = &cobra.Command{
	Use:     "sbom",
	Short:   "Generate Software Bill of Materials for projects",
	GroupID: string(cmdpkg.GroupTools),
	Long: `Generate SBOMs using industry-standard tools (cyclonedx-gomod, syft) with goenv-managed toolchains.

This command is a thin wrapper that ensures SBOM generation uses the correct Go version
and is reproducible in CI environments.

Examples:
  # Generate CycloneDX SBOM for current project
  goenv sbom project --tool=cyclonedx-gomod --format=cyclonedx-json

  # Generate SPDX SBOM with syft
  goenv sbom project --tool=syft --format=spdx-json --output=sbom.spdx.json

  # Generate SBOM for container image
  goenv sbom project --tool=syft --image=ghcr.io/myapp:v1.0.0

Before using, install the required tool:
  goenv tools install cyclonedx-gomod@v1.6.0
  goenv tools install syft@v1.0.0`,
}

var sbomProjectCmd = &cobra.Command{
	Use:   "project",
	Short: "Generate SBOM for a Go project",
	Long: `Generate a Software Bill of Materials for a Go project using cyclonedx-gomod or syft.

This wrapper ensures:
- Reproducible builds with pinned Go and tool versions
- Correct cache isolation per Go version
- CI-friendly exit codes and output

Supported tools:
- cyclonedx-gomod: Native Go module SBOM generator (CycloneDX format)
- syft: Multi-language SBOM generator (supports containers)`,
	RunE: runSBOMProject,
}

var (
	sbomTool        string
	sbomFormat      string
	sbomOutput      string
	sbomDir         string
	sbomImage       string
	sbomModulesOnly bool
	sbomOffline     bool
	sbomToolArgs    string
)

func init() {
	sbomProjectCmd.Flags().StringVar(&sbomTool, "tool", "cyclonedx-gomod", "SBOM tool to use (cyclonedx-gomod, syft)")
	sbomProjectCmd.Flags().StringVar(&sbomFormat, "format", "cyclonedx-json", "Output format (cyclonedx-json, spdx-json)")
	sbomProjectCmd.Flags().StringVarP(&sbomOutput, "output", "o", "sbom.json", "Output file path")
	sbomProjectCmd.Flags().StringVar(&sbomDir, "dir", ".", "Project directory to scan")
	sbomProjectCmd.Flags().StringVar(&sbomImage, "image", "", "Container image to scan (syft only)")
	sbomProjectCmd.Flags().BoolVar(&sbomModulesOnly, "modules-only", false, "Only scan Go modules (cyclonedx-gomod)")
	sbomProjectCmd.Flags().BoolVar(&sbomOffline, "offline", false, "Offline mode - avoid network access")
	sbomProjectCmd.Flags().StringVar(&sbomToolArgs, "tool-args", "", "Additional arguments to pass to the tool")

	sbomCmd.AddCommand(sbomProjectCmd)
	cmdpkg.RootCmd.AddCommand(sbomCmd)
}

func runSBOMProject(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	// Validate flags
	if sbomImage != "" && sbomDir != "." {
		return fmt.Errorf("cannot specify both --image and --dir")
	}

	if sbomImage != "" && sbomTool != "syft" {
		return fmt.Errorf("--image is only supported with --tool=syft")
	}

	// Resolve tool path
	toolPath, err := resolveSBOMTool(cfg, sbomTool)
	if err != nil {
		return err
	}

	// Get current Go version for provenance
	goVersion, _, err := mgr.GetCurrentVersion()
	if err != nil {
		goVersion = "unknown"
	}

	// Print provenance header to stderr (safe for CI logs)
	fmt.Fprintf(cmd.ErrOrStderr(), "goenv: Generating SBOM with %s (Go %s, %s/%s)\n",
		sbomTool, goVersion, runtime.GOOS, runtime.GOARCH)

	// Build command based on tool
	var toolCmd *exec.Cmd
	switch sbomTool {
	case "cyclonedx-gomod":
		toolCmd, err = buildCycloneDXCommand(toolPath, cfg)
	case "syft":
		toolCmd, err = buildSyftCommand(toolPath, cfg)
	default:
		return fmt.Errorf("unsupported tool: %s (supported: cyclonedx-gomod, syft)", sbomTool)
	}

	if err != nil {
		return fmt.Errorf("failed to build command: %w", err)
	}

	// Set up environment
	toolCmd.Env = os.Environ()
	if sbomOffline {
		// Add offline flags if supported by tool
		// Most tools respect GOPROXY=off
		toolCmd.Env = append(toolCmd.Env, "GOPROXY=off")
	}

	// Connect output
	toolCmd.Stdout = cmd.OutOrStdout()
	toolCmd.Stderr = cmd.ErrOrStderr()

	// Run tool
	if cfg.Debug {
		fmt.Fprintf(cmd.ErrOrStderr(), "Debug: Running command: %s\n", strings.Join(toolCmd.Args, " "))
	}

	if err := toolCmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Preserve tool's exit code
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("tool execution failed: %w", err)
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "goenv: SBOM written to %s\n", sbomOutput)

	return nil
}

// resolveSBOMTool finds the tool binary in goenv-managed paths
func resolveSBOMTool(cfg *config.Config, tool string) (string, error) {
	// Check host-specific bin directory first using consolidated utility
	hostBin := cfg.HostBinDir()
	if toolPath, err := utils.FindExecutable(hostBin, tool); err == nil {
		return toolPath, nil
	}

	// Check if tool is in PATH (system-wide installation)
	if path, err := exec.LookPath(tool); err == nil {
		return path, nil
	}

	// Tool not found - provide actionable error
	return "", fmt.Errorf(`%s not found

To install:
  goenv tools install %s@latest

Or install system-wide with:
  go install <package-path>`, tool, tool)
}

// buildCycloneDXCommand builds the cyclonedx-gomod command
func buildCycloneDXCommand(toolPath string, cfg *config.Config) (*exec.Cmd, error) {
	args := []string{}

	// Output file
	args = append(args, "-output", sbomOutput)

	// Format
	if sbomFormat == "cyclonedx-json" {
		args = append(args, "-json")
	} else if sbomFormat != "cyclonedx-xml" {
		return nil, fmt.Errorf("cyclonedx-gomod only supports cyclonedx-json and cyclonedx-xml formats")
	}

	// Modules only
	if sbomModulesOnly {
		args = append(args, "-licenses", "-type", "library")
	}

	// Additional tool args
	if sbomToolArgs != "" {
		args = append(args, strings.Fields(sbomToolArgs)...)
	}

	cmdExec := exec.Command(toolPath, args...)
	cmdExec.Dir = sbomDir

	return cmdExec, nil
}

// buildSyftCommand builds the syft command
func buildSyftCommand(toolPath string, cfg *config.Config) (*exec.Cmd, error) {
	args := []string{}

	// Scan target (image or directory)
	target := sbomDir
	if sbomImage != "" {
		target = sbomImage
	}
	args = append(args, target)

	// Output format
	outputFormat := "cyclonedx-json"
	switch sbomFormat {
	case "cyclonedx-json":
		outputFormat = "cyclonedx-json"
	case "spdx-json":
		outputFormat = "spdx-json"
	case "syft-json":
		outputFormat = "json"
	default:
		return nil, fmt.Errorf("unsupported format for syft: %s", sbomFormat)
	}
	args = append(args, "-o", fmt.Sprintf("%s=%s", outputFormat, sbomOutput))

	// Quiet mode (reduce noise)
	args = append(args, "-q")

	// Additional tool args
	if sbomToolArgs != "" {
		args = append(args, strings.Fields(sbomToolArgs)...)
	}

	return exec.Command(toolPath, args...), nil
}
