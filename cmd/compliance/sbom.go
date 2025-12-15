package compliance

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/cmdutil"
	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/platform"
	"github.com/go-nv/goenv/internal/resolver"
	"github.com/spf13/cobra"
)

var sbomCmd = &cobra.Command{
	Use:     "sbom",
	Short:   "Generate Software Bill of Materials for projects",
	GroupID: string(cmdpkg.GroupTools),
	Long: `Generate SBOMs using industry-standard tools (cyclonedx-gomod, syft) with goenv-managed toolchains.

CURRENT STATE (v3.0): This is a convenience wrapper that runs SBOM tools with the 
correct Go version and environment. It does NOT generate SBOMs itself or add features
beyond what the underlying tools provide.

ROADMAP: Future versions will add validation, policy enforcement, signing, vulnerability
scanning, and compliance reporting. See docs/roadmap/SBOM_ROADMAP.md for details.

ALTERNATIVE: Advanced users can run SBOM tools directly:
  goenv exec cyclonedx-gomod -json -output sbom.json

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

WHAT THIS DOES:
- Runs SBOM tools with the correct Go version and environment
- Provides unified CLI across different SBOM tools
- Ensures reproducibility in CI/CD pipelines

WHAT THIS DOES NOT DO (yet):
- Validate SBOM format or completeness (planned: v3.1)
- Sign or attest SBOMs (planned: v3.2)
- Scan for vulnerabilities (planned: v3.5)
- Enforce policies (planned: v3.1)

See docs/roadmap/SBOM_ROADMAP.md for planned features.

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
	ctx := cmdutil.GetContexts(cmd)
	cfg := ctx.Config
	mgr := ctx.Manager

	// Validate flags
	if sbomImage != "" && sbomDir != "." {
		return fmt.Errorf("cannot specify both --image and --dir")
	}

	if sbomImage != "" && sbomTool != "syft" {
		return fmt.Errorf("--image is only supported with --tool=syft")
	}

	// Get current Go version for provenance and tool resolution
	goVersion, versionSource, err := mgr.GetCurrentVersion()
	if err != nil {
		goVersion = "unknown"
	}

	// Resolve tool path using version context
	toolPath, err := resolveSBOMTool(cfg, sbomTool, goVersion, versionSource)
	if err != nil {
		return err
	}

	// Print provenance header to stderr (safe for CI logs)
	fmt.Fprintf(cmd.ErrOrStderr(), "goenv: Generating SBOM with %s (Go %s, %s/%s)\n",
		sbomTool, goVersion, platform.OS(), platform.Arch())

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
		return errors.FailedTo("build command", err)
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
		return errors.FailedTo("execute tool", err)
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "goenv: SBOM written to %s\n", sbomOutput)

	return nil
}

// resolveSBOMTool finds the tool binary using version-aware resolution
func resolveSBOMTool(cfg *config.Config, tool, version, versionSource string) (string, error) {
	// Use resolver to respect local vs global context
	r := resolver.New(cfg)

	if version != "unknown" && version != "" {
		if toolPath, err := r.ResolveBinary(tool, version, versionSource); err == nil {
			return toolPath, nil
		}
	}

	// Fallback: Check if tool is in PATH (system-wide installation)
	if path, err := exec.LookPath(tool); err == nil {
		return path, nil
	}

	// Tool not found - provide actionable error
	return "", fmt.Errorf(`%s not found

To install for current version:
  goenv tools install %s@latest

Or install system-wide with:
  go install <package-path>`, tool, tool)
}

// buildCycloneDXCommand builds the cyclonedx-gomod command
func buildCycloneDXCommand(toolPath string, cfg *config.Config) (*exec.Cmd, error) {
	args := []string{}

	// Use the "mod" subcommand for module SBOMs (default behavior)
	// "mod" generates SBOMs for modules, including all packages
	// "app" would be for application binaries, "bin" for pre-built binaries
	args = append(args, "mod")

	// Output file - new format uses -output (with single dash)
	args = append(args, "-output", sbomOutput)

	// Format - newer versions use -output-format flag
	if sbomFormat == "cyclonedx-json" {
		args = append(args, "-json")
	} else if sbomFormat != "cyclonedx-xml" {
		return nil, fmt.Errorf("cyclonedx-gomod only supports cyclonedx-json and cyclonedx-xml formats")
	}

	// Modules only (include licenses and set type)
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
