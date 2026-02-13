package shell

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/cmdutil"
	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/shellutil"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var promptCmd = &cobra.Command{
	Use:     "prompt",
	Short:   "Display active Go version for shell prompt",
	GroupID: string(cmdpkg.GroupShell),
	Long: `Outputs the active Go version formatted for shell prompts.

This command is optimized for prompt integration and includes caching
to minimize performance impact. It respects GOENV_PROMPT_* environment
variables for customization.

Examples:
  # Basic usage
  export PS1='$(goenv prompt) $ '

  # With formatting
  export PS1='$(goenv prompt --prefix "[" --suffix "]") $ '

  # Show only in Go projects
  export PS1='$(goenv prompt --go-project-only) $ '

Environment Variables:
  GOENV_PROMPT_FORMAT      - Format string (e.g., "go:%s")
  GOENV_PROMPT_PREFIX      - Prefix string (e.g., "(")
  GOENV_PROMPT_SUFFIX      - Suffix string (e.g., ")")
  GOENV_PROMPT_NO_SYSTEM   - Don't show system Go (set to "1")
  GOENV_PROMPT_PROJECT_ONLY - Show only in Go projects (set to "1")
  GOENV_PROMPT_CACHE_TTL   - Cache duration in seconds (default: 5)
  GOENV_DISABLE_PROMPT     - Disable prompt output entirely (set to "1")`,
	RunE: runPrompt,
}

var promptFlags struct {
	format      string
	prefix      string
	suffix      string
	noSystem    bool
	projectOnly bool
	cacheTTL    int
	short       bool
	icon        string
}

func init() {
	promptCmd.Flags().StringVar(&promptFlags.format, "format", "", "Format string with %s placeholder")
	promptCmd.Flags().StringVar(&promptFlags.prefix, "prefix", "", "Prefix before version")
	promptCmd.Flags().StringVar(&promptFlags.suffix, "suffix", "", "Suffix after version")
	promptCmd.Flags().BoolVar(&promptFlags.noSystem, "no-system", false, "Don't show system Go")
	promptCmd.Flags().BoolVar(&promptFlags.projectOnly, "go-project-only", false, "Show only in Go projects")
	promptCmd.Flags().IntVar(&promptFlags.cacheTTL, "cache-ttl", 5, "Cache TTL in seconds")
	promptCmd.Flags().BoolVar(&promptFlags.short, "short", false, "Show short version (e.g., 1.23 instead of 1.23.2)")
	promptCmd.Flags().StringVar(&promptFlags.icon, "icon", "", "Icon/emoji to prepend (e.g., 🐹)")

	// Register as direct subcommand of root
	cmdpkg.RootCmd.AddCommand(promptCmd)
}

func runPrompt(cmd *cobra.Command, args []string) error {
	// Check if prompt is globally disabled
	if utils.GoenvEnvVarDisablePrompt.IsTrue() {
		return nil
	}

	ctx := cmdutil.GetContexts(cmd)
	mgr := ctx.Manager

	// Get active version (with caching)
	version, err := getActiveVersionCached(mgr, promptFlags.cacheTTL)
	if err != nil {
		// Don't error in prompts, just return empty
		// Prompts should never cause shell errors
		return nil
	}

	// Apply filters
	if shouldHideVersion(version) {
		return nil
	}

	// Apply formatting
	output := formatPromptVersion(version)

	// Don't add newline - prompts expect no trailing newline
	fmt.Fprint(cmd.OutOrStdout(), output)

	return nil
}

// shouldHideVersion checks if version should be hidden based on filters
func shouldHideVersion(version string) bool {
	// Check --no-system flag or env var
	if (promptFlags.noSystem || utils.GoenvEnvVarPromptNoSystem.IsTrue()) && version == manager.SystemVersion {
		return true
	}

	// Check --go-project-only flag or env var
	if promptFlags.projectOnly || utils.GoenvEnvVarPromptProjectOnly.IsTrue() {
		if !isGoProject() {
			return true
		}
	}

	return false
}

// getActiveVersionCached returns the active version with simple file-based caching
func getActiveVersionCached(mgr *manager.Manager, ttl int) (string, error) {
	// Cache key is based on:
	// 1. Current working directory (version files are directory-specific)
	// 2. GOENV_VERSION env var
	cacheKey := generateCacheKey()
	cachePath := getCachePath(mgr.Config(), cacheKey)

	// Check cache validity
	if isCacheValid(cachePath, ttl) {
		if version, err := readCache(cachePath); err == nil {
			return version, nil
		}
	}

	// Cache miss or expired - query version
	// Use GetCurrentVersionResolved to handle partial versions
	resolvedVersion, _, _, err := mgr.GetCurrentVersionResolved()
	if err != nil {
		return "", err
	}

	// Update cache (ignore errors - caching is optional)
	_ = writeCache(cachePath, resolvedVersion)

	return resolvedVersion, nil
}

// generateCacheKey creates a unique cache key for the current context
func generateCacheKey() string {
	var parts []string

	// Include CWD
	if cwd, err := os.Getwd(); err == nil {
		parts = append(parts, cwd)
	}

	// Include GOENV_VERSION if set
	if goenvVersion := utils.GoenvEnvVarVersion.UnsafeValue(); goenvVersion != "" {
		parts = append(parts, utils.GoenvEnvVarVersion.String()+"="+goenvVersion)
	}

	// Hash the parts
	return hashString(strings.Join(parts, "|"))
}

// hashString creates a SHA256 hash of a string (first 16 chars)
func hashString(s string) string {
	fullHash := utils.SHA256String(s)
	return fullHash[:16] // Use first 16 hex chars
}

// getCachePath returns the path to the cache file
func getCachePath(cfg *config.Config, key string) string {
	cacheDir := filepath.Join(cfg.Root, "cache", "prompt")
	_ = utils.EnsureDirWithContext(cacheDir, "create cache directory") // Ignore error - will fail later if can't create
	return filepath.Join(cacheDir, key)
}

// isCacheValid checks if cache file exists and is within TTL
func isCacheValid(path string, ttl int) bool {
	modTime := utils.GetFileModTime(path)
	if modTime.IsZero() {
		return false
	}

	age := time.Since(modTime).Seconds()
	return age < float64(ttl)
}

// readCache reads version from cache file
func readCache(path string) (string, error) {
	data, err := utils.ReadFileWithContext(path, "read file")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// writeCache writes version to cache file
func writeCache(path string, version string) error {
	return utils.WriteFileWithContext(path, []byte(version), utils.PermFileDefault, "write file")
}

// isGoProject checks if current directory is a Go project
func isGoProject() bool {
	// Check for .go-version
	if utils.PathExists(config.VersionFileName) {
		return true
	}

	// Check for go.mod
	if utils.PathExists(config.GoModFileName) {
		return true
	}

	// Check for .tool-versions with go entry
	// Use manager API for consistent parsing
	if utils.PathExists(config.ToolVersionsFileName) {
		cfg := config.Load()
		env := utils.EnvironmentFromContextOrLoad(context.Background())
		mgr := manager.NewManager(cfg, env)
		_ = cfg // unused but required by SetupContext

		// Try to read version from .tool-versions
		if version, err := mgr.ReadVersionFile(config.ToolVersionsFileName); err == nil && version != "" {
			return true
		}
	}

	// Check for any .go files
	matches, err := filepath.Glob("*.go")
	if err == nil && len(matches) > 0 {
		return true
	}

	return false
}

// formatPromptVersion applies formatting to the version string
func formatPromptVersion(version string) string {
	// Apply short format if requested
	if promptFlags.short {
		version = formatVersionShort(version)
	}

	// Start with icon if provided
	icon := promptFlags.icon
	if icon == "" {
		icon = utils.GoenvEnvVarPromptIcon.UnsafeValue()
	}
	var output string
	if icon != "" {
		output = icon + " "
	}

	// Get prefix
	prefix := promptFlags.prefix
	if prefix == "" {
		prefix = utils.GoenvEnvVarPromptPrefix.UnsafeValue()
	}

	// Apply format string
	format := promptFlags.format
	if format == "" {
		format = utils.GoenvEnvVarPromptFormat.UnsafeValue()
	}
	if format != "" {
		version = fmt.Sprintf(format, version)
	}

	// Get suffix
	suffix := promptFlags.suffix
	if suffix == "" {
		suffix = utils.GoenvEnvVarPromptSuffix.UnsafeValue()
	}

	return output + prefix + version + suffix
}

// formatVersionShort returns the short version (major.minor)
func formatVersionShort(version string) string {
	// Don't shorten "system"
	if version == manager.SystemVersion {
		return version
	}

	parts := strings.Split(version, ".")
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return version
}

// ============================================================================
// Prompt Configuration Subcommand
// ============================================================================

var promptConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure prompt integration interactively",
	Long: `Interactive configuration for shell prompt integration.

This wizard helps you set up your shell prompt to display the active Go version.
It will detect your shell and offer to automatically add prompt integration
to your shell profile.

Examples:
  goenv prompt config              # Interactive setup wizard
  goenv prompt config --help       # Show this help`,
	RunE: runPromptConfig,
}

func init() {
	// Register config as subcommand of prompt
	promptCmd.AddCommand(promptConfigCmd)
}

func runPromptConfig(cmd *cobra.Command, args []string) error {
	// Create interactive context
	ctx := cmdutil.NewInteractiveContext(cmd)

	// Detect shell
	shell := shellutil.DetectShell()

	// Welcome message
	ctx.Printf("\n%sPrompt Integration Setup\n\n", "🎨 ")
	ctx.Printf("This wizard will help you configure your shell prompt to display\n")
	ctx.Printf("the active Go version managed by goenv.\n\n")
	ctx.Printf("Detected shell: %s\n\n", shell)

	// Show shell-specific instructions
	showPromptSetupInstructions(ctx, shell)

	// Offer automatic setup
	if shell == shellutil.ShellTypeCmd {
		ctx.Printf("\nAutomatic setup is not available for cmd.exe.\n")
		ctx.Printf("Please manually configure your prompt using the instructions above.\n")
		return nil
	}

	// Ask if user wants automatic setup
	if !ctx.Confirm("\nWould you like to add prompt integration to your shell profile automatically?", true) {
		ctx.Printf("\nNo changes made. You can manually add the code shown above to your shell profile.\n")
		return nil
	}

	// Apply automatic setup
	if err := applyPromptSetup(ctx, shell); err != nil {
		return err
	}

	// Success message
	ctx.Printf("\n%sPrompt integration added successfully!\n", "✅ ")
	ctx.Printf("\nTo activate the changes, run:\n")
	ctx.Printf("  %s\n\n", getReloadCommand(shell))

	return nil
}

// showPromptSetupInstructions displays shell-specific setup instructions
func showPromptSetupInstructions(ctx *cmdutil.InteractiveContext, shell shellutil.ShellType) {
	switch shell {
	case shellutil.ShellTypeBash:
		showBashPromptSetup(ctx)
	case shellutil.ShellTypeZsh:
		showZshPromptSetup(ctx)
	case shellutil.ShellTypeFish:
		showFishPromptSetup(ctx)
	case shellutil.ShellTypePowerShell:
		showPowerShellPromptSetup(ctx)
	case shellutil.ShellTypeCmd:
		showCmdPromptSetup(ctx)
	default:
		ctx.Printf("Automatic setup is not available for your shell.\n")
		ctx.Printf("You can manually add:\n")
		ctx.Printf("  export PS1='$(goenv prompt) '\"$PS1\"\n")
	}
}

func showBashPromptSetup(ctx *cmdutil.InteractiveContext) {
	ctx.Printf("For Bash, add this to ~/.bashrc:\n\n")
	ctx.Printf("  # Show Go version in prompt\n")
	ctx.Printf("  export PS1='$(goenv prompt --prefix \"(\" --suffix \") \") '\"$PS1\"\n\n")
	ctx.Printf("Or for a custom format with colors:\n")
	ctx.Printf("  export PS1='\\[\\033[36m\\]$(goenv prompt --format \"go:%%s\")\\[\\033[0m\\] '\"$PS1\"\n")
}

func showZshPromptSetup(ctx *cmdutil.InteractiveContext) {
	ctx.Printf("For Zsh, add this to ~/.zshrc:\n\n")
	ctx.Printf("  # Show Go version in prompt\n")
	ctx.Printf("  setopt PROMPT_SUBST\n")
	ctx.Printf("  export PS1='$(goenv prompt --prefix \"(\" --suffix \") \") %%~ %%# '\n\n")
	ctx.Printf("Or use in your existing prompt:\n")
	ctx.Printf("  GOENV_VERSION='$(goenv prompt 2>/dev/null)'\n")
	ctx.Printf("  PS1='${GOENV_VERSION:+($GOENV_VERSION) }'\"$PS1\"\n")
}

func showFishPromptSetup(ctx *cmdutil.InteractiveContext) {
	ctx.Printf("For Fish, add this to ~/.config/fish/functions/fish_prompt.fish:\n\n")
	ctx.Printf("  function fish_prompt\n")
	ctx.Printf("    # Show Go version\n")
	ctx.Printf("    set -l goenv_version (goenv prompt 2>/dev/null)\n")
	ctx.Printf("    if test -n \"$goenv_version\"\n")
	ctx.Printf("      set_color cyan\n")
	ctx.Printf("      echo -n \"($goenv_version) \"\n")
	ctx.Printf("      set_color normal\n")
	ctx.Printf("    end\n\n")
	ctx.Printf("    # ... rest of your prompt\n")
	ctx.Printf("  end\n")
}

func showPowerShellPromptSetup(ctx *cmdutil.InteractiveContext) {
	ctx.Printf("For PowerShell, add this to your $PROFILE:\n\n")
	ctx.Printf("  function prompt {\n")
	ctx.Printf("    $goenvVersion = goenv prompt 2>$null\n")
	ctx.Printf("    if ($goenvVersion) {\n")
	ctx.Printf("      Write-Host \"($goenvVersion) \" -NoNewline -ForegroundColor Cyan\n")
	ctx.Printf("    }\n")
	ctx.Printf("    \"PS $($PWD.Path)> \"\n")
	ctx.Printf("  }\n")
}

func showCmdPromptSetup(ctx *cmdutil.InteractiveContext) {
	ctx.Printf("Note: cmd.exe has very limited prompt customization capabilities.\n\n")
	ctx.Printf("You can manually check your Go version with:\n")
	ctx.Printf("  goenv prompt\n\n")
	ctx.Printf("For better prompt support, consider using PowerShell instead.\n")
}

// applyPromptSetup automatically adds prompt integration to shell profile
func applyPromptSetup(ctx *cmdutil.InteractiveContext, shell shellutil.ShellType) error {
	// Get profile path
	profilePath := shellutil.GetProfilePath(shell)
	if profilePath == "" {
		return fmt.Errorf("could not determine shell profile path for %s", shell)
	}

	// Read existing profile (or create if not exists)
	content, err := os.ReadFile(profilePath)
	if err != nil && !os.IsNotExist(err) {
		return errors.FailedTo("read profile file", err)
	}

	// Check if already configured
	contentStr := string(content)
	if strings.Contains(contentStr, "goenv prompt") {
		ctx.ErrorPrintf("\n%sPrompt integration already exists in %s\n", "⚠️  ", profilePath)
		return fmt.Errorf("prompt integration already configured")
	}

	// Generate prompt setup code
	promptSetup := generatePromptSetup(shell)

	// Append to profile
	newContent := contentStr
	if len(contentStr) > 0 && !strings.HasSuffix(contentStr, "\n") {
		newContent += "\n"
	}
	newContent += "\n" + promptSetup + "\n"

	// Write back to profile
	if err := utils.WriteFileWithContext(profilePath, []byte(newContent), utils.PermFileDefault, "write to profile file"); err != nil {
		return err
	}

	ctx.Printf("\n%sAdded prompt integration to %s\n", "✓ ", profilePath)

	return nil
}

// generatePromptSetup generates the shell-specific prompt configuration code
func generatePromptSetup(shell shellutil.ShellType) string {
	var builder strings.Builder

	builder.WriteString("# goenv prompt integration\n")

	switch shell {
	case shellutil.ShellTypeBash:
		builder.WriteString("# Show active Go version in prompt\n")
		builder.WriteString("export PS1='$(goenv prompt --prefix \"(\" --suffix \") \") '\"$PS1\"\n")

	case shellutil.ShellTypeZsh:
		builder.WriteString("# Show active Go version in prompt\n")
		builder.WriteString("setopt PROMPT_SUBST\n")
		builder.WriteString("export PS1='$(goenv prompt --prefix \"(\" --suffix \") \") '\"$PS1\"\n")

	case shellutil.ShellTypeFish:
		builder.WriteString("# This should be added to ~/.config/fish/functions/fish_prompt.fish\n")
		builder.WriteString("# Add these lines inside your fish_prompt function:\n")
		builder.WriteString("#   set -l goenv_version (goenv prompt 2>/dev/null)\n")
		builder.WriteString("#   test -n \"$goenv_version\"; and echo -n \"($goenv_version) \"\n")

	case shellutil.ShellTypePowerShell:
		builder.WriteString("# Show active Go version in prompt\n")
		builder.WriteString("function global:prompt {\n")
		builder.WriteString("  $goenvVersion = goenv prompt 2>$null\n")
		builder.WriteString("  if ($goenvVersion) {\n")
		builder.WriteString("    Write-Host \"($goenvVersion) \" -NoNewline -ForegroundColor Cyan\n")
		builder.WriteString("  }\n")
		builder.WriteString("  \"PS $($PWD.Path)> \"\n")
		builder.WriteString("}\n")
	}

	return builder.String()
}

// getReloadCommand returns the shell-specific command to reload the profile
func getReloadCommand(shell shellutil.ShellType) string {
	switch shell {
	case shellutil.ShellTypeBash:
		return "source ~/.bashrc"
	case shellutil.ShellTypeZsh:
		return "source ~/.zshrc"
	case shellutil.ShellTypeFish:
		return "source ~/.config/fish/config.fish"
	case shellutil.ShellTypePowerShell:
		return ". $PROFILE"
	default:
		return "restart your shell"
	}
}
