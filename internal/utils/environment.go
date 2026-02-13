package utils

import (
	"context"
	"os"
	"strings"

	"github.com/sethvargo/go-envconfig"
)

type GoenvEnvVar string
type GoenvBoolEnv = GoenvEnvVar

// Common system environment variable names
const (
	EnvVarPath        = "PATH"
	EnvVarHome        = "HOME"
	EnvVarShell       = "SHELL"
	EnvVarTerm        = "TERM"
	EnvVarGopath      = "GOPATH"
	EnvVarGoroot      = "GOROOT"
	EnvVarShlvl       = "SHLVL"
	EnvVarUser        = "USER"
	EnvVarUserProfile = "USERPROFILE" // Windows
	EnvVarProgramData = "ProgramData" // Windows

	// Go toolchain environment variables
	EnvVarGocache     = "GOCACHE"
	EnvVarGomodcache  = "GOMODCACHE"
	EnvVarGotoolchain = "GOTOOLCHAIN"
	EnvVarGoos        = "GOOS"
	EnvVarGoarch      = "GOARCH"

	// CI/CD environment variable names
	EnvVarGitHubActions = "GITHUB_ACTIONS"
	EnvVarGitHubToken   = "GITHUB_TOKEN"
	EnvVarGitLabCI      = "GITLAB_CI"
	EnvVarCircleCI      = "CIRCLECI"
	EnvVarTravisCI      = "TRAVIS"
	EnvVarCI            = "CI"

	// Container/Virtualization detection
	EnvVarWSLDistroName           = "WSL_DISTRO_NAME"
	EnvVarKubernetesServiceHost   = "KUBERNETES_SERVICE_HOST"
	EnvVarContainer               = "container" // lowercase
	EnvVarContainerUpper          = "CONTAINER" // uppercase
	EnvVarBuildkitSandboxHostname = "BUILDKIT_SANDBOX_HOSTNAME"

	// Go installation/build variables
	EnvVarGoBuildMirrorURL = "GO_BUILD_MIRROR_URL"

	// Test control environment variables
	EnvVarSkipNetworkTests = "SKIP_NETWORK_TESTS"

	// Display/UI environment variables
	EnvVarNoColor = "NO_COLOR"

	// PowerShell environment
	EnvVarPSModulePath = "PSModulePath"

	// Shell version detection
	EnvVarZshVersion  = "ZSH_VERSION"
	EnvVarFishVersion = "FISH_VERSION"
	EnvVarBashVersion = "BASH_VERSION"

	// Shell profile
	EnvVarProfile = "PROFILE"
	EnvVarComspec = "COMSPEC" // Windows command processor

	// Windows-specific environment variables
	EnvVarLocalAppData    = "LOCALAPPDATA"
	EnvVarSystemRoot      = "SystemRoot"
	EnvVarProgramFilesArm = "ProgramFiles(Arm)"
	EnvVarScoop           = "SCOOP"

	// macOS-specific environment variables
	EnvVarMacOSXDeploymentTarget = "MACOSX_DEPLOYMENT_TARGET"

	// MinGW/MSYS environment
	EnvVarMSYSTEM     = "MSYSTEM"      // MinGW system type (MINGW64, MINGW32, etc.)
	EnvVarMINGWPrefix = "MINGW_PREFIX" // MinGW installation prefix
)

const (
	// #region String Env Vars
	GoenvEnvVarVersion          GoenvEnvVar = "GOENV_VERSION"
	GoenvEnvVarRoot             GoenvEnvVar = "GOENV_ROOT"
	GoenvEnvVarDebug            GoenvEnvVar = "GOENV_DEBUG"
	GoenvEnvVarDir              GoenvEnvVar = "GOENV_DIR"
	GoenvEnvVarShell            GoenvEnvVar = "GOENV_SHELL"
	GoenvEnvVarAutoInstallFlags GoenvEnvVar = "GOENV_AUTO_INSTALL_FLAGS"
	GoenvEnvVarRcFile           GoenvEnvVar = "GOENV_RC_FILE"
	GoenvEnvVarPathOrder        GoenvEnvVar = "GOENV_PATH_ORDER"
	GoenvEnvVarShimDebug        GoenvEnvVar = "GOENV_SHIM_DEBUG"
	GoenvEnvVarHooksConfig      GoenvEnvVar = "GOENV_HOOKS_CONFIG"
	GoenvEnvVarHooksLog         GoenvEnvVar = "GOENV_HOOKS_LOG"
	GoenvEnvVarInstallRoot      GoenvEnvVar = "GOENV_INSTALL_ROOT"
	GoenvEnvVarInstallTimeout   GoenvEnvVar = "GOENV_INSTALL_TIMEOUT"
	GoenvEnvVarInstallRetries   GoenvEnvVar = "GOENV_INSTALL_RETRIES"
	GoenvEnvVarInstallResume    GoenvEnvVar = "GOENV_INSTALL_RESUME"
	GoenvEnvVarGocacheDir       GoenvEnvVar = "GOENV_GOCACHE_DIR"
	GoenvEnvVarFileArg          GoenvEnvVar = "GOENV_FILE_ARG"
	GoenvEnvVarPromptPrefix     GoenvEnvVar = "GOENV_PROMPT_PREFIX"
	GoenvEnvVarPromptSuffix     GoenvEnvVar = "GOENV_PROMPT_SUFFIX"
	GoenvEnvVarPromptFormat     GoenvEnvVar = "GOENV_PROMPT_FORMAT"
	GoenvEnvVarPromptIcon       GoenvEnvVar = "GOENV_PROMPT_ICON"
	GoenvEnvVarVersionOrigin    GoenvEnvVar = "GOENV_VERSION_ORIGIN"
	// #endregion

	// #region Bool Env Vars
	GoenvEnvVarDisableGoroot       GoenvBoolEnv = "GOENV_DISABLE_GOROOT"
	GoenvEnvVarDisableGopath       GoenvBoolEnv = "GOENV_DISABLE_GOPATH"
	GoenvEnvVarAutoInstall         GoenvBoolEnv = "GOENV_AUTO_INSTALL"
	GoenvEnvVarNoAutoRehash        GoenvBoolEnv = "GOENV_NO_AUTO_REHASH"
	GoenvEnvVarOffline             GoenvBoolEnv = "GOENV_OFFLINE"
	GoenvEnvVarAutoRehash          GoenvBoolEnv = "GOENV_AUTO_REHASH"
	GoenvEnvVarDisableGocache      GoenvBoolEnv = "GOENV_DISABLE_GOCACHE"
	GoenvEnvVarDisableGomod        GoenvBoolEnv = "GOENV_DISABLE_GOMOD"
	GoenvEnvVarDisablePrompt       GoenvBoolEnv = "GOENV_DISABLE_PROMPT"
	GoenvEnvVarDisablePromptHelper GoenvBoolEnv = "GOENV_DISABLE_PROMPT_HELPER"
	GoenvEnvVarPromptNoSystem      GoenvBoolEnv = "GOENV_PROMPT_NO_SYSTEM"
	GoenvEnvVarPromptProjectOnly   GoenvBoolEnv = "GOENV_PROMPT_PROJECT_ONLY"
	GoenvEnvVarCacheBgRefresh      GoenvBoolEnv = "GOENV_CACHE_BG_REFRESH"
	GoenvEnvVarAssumeYes           GoenvBoolEnv = "GOENV_ASSUME_YES"
	// #endregion
)

// String returns the string representation of the environment variable.
func (g GoenvEnvVar) String() string {
	return string(g)
}

func (g GoenvEnvVar) Set(val string) error {
	return os.Setenv(g.String(), val)
}

func (g GoenvEnvVar) Unsetenv() error {
	return os.Unsetenv(g.String())
}

// UnsafeValue retrieves the value of the environment variable without checking if it's set.
func (g GoenvEnvVar) UnsafeValue() string {
	return os.Getenv(g.String())
}

// Value retrieves the value of the environment variable and a boolean indicating if it is set.
func (g GoenvEnvVar) Value() (string, bool) {
	return os.LookupEnv(g.String())
}

// IsSet checks if the environment variable is set.
func (g GoenvEnvVar) IsSet() bool {
	_, ok := g.Value()
	return ok
}

// IsEnvVarTrue checks if the environment variable is set to a truthy value.
// Only accepts exact lowercase strings: "1", "true", "yes" for true
// and "0", "false", "no" for false (returns false)
// Any other value (including capitalized versions) is considered invalid and returns false
func (g GoenvBoolEnv) IsTrue() bool {
	val, ok := g.Value()
	if !ok {
		return false
	}

	// Only accept exact lowercase strings
	switch val {
	case "1", "true", "yes":
		return true
	case "0", "false", "no":
		return false
	default:
		// Invalid value - treat as false (will fall back to prompting)
		return false
	}
}

// GetEnvValue retrieves an environment variable value from an env slice
// The env slice should be in the format ["KEY=value", "KEY2=value2", ...]
// as returned by os.Environ()
func GetEnvValue(env []string, key string) string {
	prefix := key + "="
	for _, envVar := range env {
		if strings.HasPrefix(envVar, prefix) {
			return strings.TrimPrefix(envVar, prefix)
		}
	}
	return ""
}

// IsMinGW detects if running in MinGW/MSYS environment (Git Bash, MSYS2)
// MinGW is a Unix-like environment on Windows that uses Unix-style paths
// but runs Windows binaries.
func IsMinGW() bool {
	if !IsWindows() {
		return false
	}

	// Check MinGW/MSYS environment variables
	msystem := os.Getenv(EnvVarMSYSTEM)
	mingwPrefix := os.Getenv(EnvVarMINGWPrefix)
	if msystem != "" || mingwPrefix != "" {
		return true
	}

	// Check if SHELL points to bash/sh but we're on Windows
	shell := os.Getenv(EnvVarShell)
	return strings.Contains(shell, "bash") || strings.Contains(shell, "/sh")
}

// GoenvEnvironment holds all GOENV_* environment variables.
// Uses go-envconfig with PrefixLookuper to automatically map GOENV_* vars.
// Field names map to env vars like: VERSION -> GOENV_VERSION
type GoenvEnvironment struct {
	// String Env Vars
	Version          string `env:"VERSION"`
	Root             string `env:"ROOT"`
	Debug            string `env:"DEBUG"`
	Dir              string `env:"DIR"`
	Shell            string `env:"SHELL"`
	AutoInstallFlags string `env:"AUTO_INSTALL_FLAGS"`
	RcFile           string `env:"RC_FILE"`
	PathOrder        string `env:"PATH_ORDER"`
	ShimDebug        string `env:"SHIM_DEBUG"`
	HooksConfig      string `env:"HOOKS_CONFIG"`
	HooksLog         string `env:"HOOKS_LOG"`
	InstallRoot      string `env:"INSTALL_ROOT"`
	InstallTimeout   string `env:"INSTALL_TIMEOUT"`
	InstallRetries   string `env:"INSTALL_RETRIES"`
	InstallResume    string `env:"INSTALL_RESUME"`
	GocacheDir       string `env:"GOCACHE_DIR"`
	FileArg          string `env:"FILE_ARG"`
	PromptPrefix     string `env:"PROMPT_PREFIX"`
	PromptSuffix     string `env:"PROMPT_SUFFIX"`
	PromptFormat     string `env:"PROMPT_FORMAT"`
	PromptIcon       string `env:"PROMPT_ICON"`
	VersionOrigin    string `env:"VERSION_ORIGIN"`

	// Bool Env Vars
	DisableGoroot       bool `env:"DISABLE_GOROOT,default=false"`
	DisableGopath       bool `env:"DISABLE_GOPATH,default=false"`
	AutoInstall         bool `env:"AUTO_INSTALL,default=false"`
	NoAutoRehash        bool `env:"NO_AUTO_REHASH,default=false"`
	Offline             bool `env:"OFFLINE,default=false"`
	AutoRehash          bool `env:"AUTO_REHASH,default=false"`
	DisableGocache      bool `env:"DISABLE_GOCACHE,default=false"`
	DisableGomod        bool `env:"DISABLE_GOMOD,default=false"`
	DisablePrompt       bool `env:"DISABLE_PROMPT,default=false"`
	DisablePromptHelper bool `env:"DISABLE_PROMPT_HELPER,default=false"`
	PromptNoSystem      bool `env:"PROMPT_NO_SYSTEM,default=false"`
	PromptProjectOnly   bool `env:"PROMPT_PROJECT_ONLY,default=false"`
	CacheBgRefresh      bool `env:"CACHE_BG_REFRESH,default=false"`
	AssumeYes           bool `env:"ASSUME_YES,default=false"`
}

// LoadEnvironment parses all GOENV_* environment variables using go-envconfig.
// It uses PrefixLookuper to automatically prepend "GOENV_" to all field names.
func LoadEnvironment(ctx context.Context) (*GoenvEnvironment, error) {
	var env GoenvEnvironment
	if err := envconfig.ProcessWith(ctx, &envconfig.Config{
		Target:   &env,
		Lookuper: envconfig.PrefixLookuper("GOENV_", envconfig.OsLookuper()),
	}); err != nil {
		return nil, err
	}
	return &env, nil
}

// Context key type for type-safe context storage
type environmentContextKey string

const EnvironmentContextKey environmentContextKey = "goenv.environment"

// EnvironmentToContext stores the environment in the context.
func EnvironmentToContext(ctx context.Context, env *GoenvEnvironment) context.Context {
	return context.WithValue(ctx, EnvironmentContextKey, env)
}

// EnvironmentFromContext retrieves the environment from the context.
// Returns nil if not found or if context is nil.
func EnvironmentFromContext(ctx context.Context) *GoenvEnvironment {
	if ctx == nil {
		return nil
	}
	if env, ok := ctx.Value(EnvironmentContextKey).(*GoenvEnvironment); ok {
		return env
	}
	return nil
}

// EnvironmentFromContextOrLoad retrieves the environment from context,
// or loads it if not found. This is a convenience function for code that
// may not have environment in context (e.g., tests, utility functions).
// Returns a default environment if loading fails.
func EnvironmentFromContextOrLoad(ctx context.Context) *GoenvEnvironment {
	// Try to get from context first
	if env := EnvironmentFromContext(ctx); env != nil {
		return env
	}

	// Fallback: load from environment variables
	if ctx == nil {
		ctx = context.Background()
	}
	env, err := LoadEnvironment(ctx)
	if err != nil {
		// Return empty environment as last resort
		return &GoenvEnvironment{}
	}
	return env
}

// Helper methods for GoenvEnvironment - String values
// These provide nil-safe access to environment variables

func (e *GoenvEnvironment) GetVersion() string {
	if e == nil {
		return ""
	}
	return e.Version
}

func (e *GoenvEnvironment) GetRoot() string {
	if e == nil {
		return ""
	}
	return e.Root
}

func (e *GoenvEnvironment) GetDebug() string {
	if e == nil {
		return ""
	}
	return e.Debug
}

func (e *GoenvEnvironment) GetDir() string {
	if e == nil {
		return ""
	}
	return e.Dir
}

func (e *GoenvEnvironment) GetShell() string {
	if e == nil {
		return ""
	}
	return e.Shell
}

func (e *GoenvEnvironment) GetAutoInstallFlags() string {
	if e == nil {
		return ""
	}
	return e.AutoInstallFlags
}

func (e *GoenvEnvironment) GetRcFile() string {
	if e == nil {
		return ""
	}
	return e.RcFile
}

func (e *GoenvEnvironment) GetPathOrder() string {
	if e == nil {
		return ""
	}
	return e.PathOrder
}

func (e *GoenvEnvironment) GetShimDebug() string {
	if e == nil {
		return ""
	}
	return e.ShimDebug
}

func (e *GoenvEnvironment) GetHooksConfig() string {
	if e == nil {
		return ""
	}
	return e.HooksConfig
}

func (e *GoenvEnvironment) GetHooksLog() string {
	if e == nil {
		return ""
	}
	return e.HooksLog
}

func (e *GoenvEnvironment) GetInstallRoot() string {
	if e == nil {
		return ""
	}
	return e.InstallRoot
}

func (e *GoenvEnvironment) GetInstallTimeout() string {
	if e == nil {
		return ""
	}
	return e.InstallTimeout
}

func (e *GoenvEnvironment) GetInstallRetries() string {
	if e == nil {
		return ""
	}
	return e.InstallRetries
}

func (e *GoenvEnvironment) GetInstallResume() string {
	if e == nil {
		return ""
	}
	return e.InstallResume
}

func (e *GoenvEnvironment) GetGocacheDir() string {
	if e == nil {
		return ""
	}
	return e.GocacheDir
}

func (e *GoenvEnvironment) GetFileArg() string {
	if e == nil {
		return ""
	}
	return e.FileArg
}

func (e *GoenvEnvironment) GetPromptPrefix() string {
	if e == nil {
		return ""
	}
	return e.PromptPrefix
}

func (e *GoenvEnvironment) GetPromptSuffix() string {
	if e == nil {
		return ""
	}
	return e.PromptSuffix
}

func (e *GoenvEnvironment) GetPromptFormat() string {
	if e == nil {
		return ""
	}
	return e.PromptFormat
}

func (e *GoenvEnvironment) GetPromptIcon() string {
	if e == nil {
		return ""
	}
	return e.PromptIcon
}

func (e *GoenvEnvironment) GetVersionOrigin() string {
	if e == nil {
		return ""
	}
	return e.VersionOrigin
}

// Helper methods for GoenvEnvironment - Boolean values
// These provide nil-safe access to boolean environment variables

func (e *GoenvEnvironment) HasDisableGoroot() bool {
	if e == nil {
		return false
	}
	return e.DisableGoroot
}

func (e *GoenvEnvironment) HasDisableGopath() bool {
	if e == nil {
		return false
	}
	return e.DisableGopath
}

func (e *GoenvEnvironment) HasAutoInstall() bool {
	if e == nil {
		return false
	}
	return e.AutoInstall
}

func (e *GoenvEnvironment) HasNoAutoRehash() bool {
	if e == nil {
		return false
	}
	return e.NoAutoRehash
}

func (e *GoenvEnvironment) HasOffline() bool {
	if e == nil {
		return false
	}
	return e.Offline
}

func (e *GoenvEnvironment) HasAutoRehash() bool {
	if e == nil {
		return false
	}
	return e.AutoRehash
}

func (e *GoenvEnvironment) HasDisableGocache() bool {
	if e == nil {
		return false
	}
	return e.DisableGocache
}

func (e *GoenvEnvironment) HasDisableGomod() bool {
	if e == nil {
		return false
	}
	return e.DisableGomod
}

func (e *GoenvEnvironment) HasDisablePrompt() bool {
	if e == nil {
		return false
	}
	return e.DisablePrompt
}

func (e *GoenvEnvironment) HasDisablePromptHelper() bool {
	if e == nil {
		return false
	}
	return e.DisablePromptHelper
}

func (e *GoenvEnvironment) HasPromptNoSystem() bool {
	if e == nil {
		return false
	}
	return e.PromptNoSystem
}

func (e *GoenvEnvironment) HasPromptProjectOnly() bool {
	if e == nil {
		return false
	}
	return e.PromptProjectOnly
}

func (e *GoenvEnvironment) HasCacheBgRefresh() bool {
	if e == nil {
		return false
	}
	return e.CacheBgRefresh
}

func (e *GoenvEnvironment) HasAssumeYes() bool {
	if e == nil {
		return false
	}
	return e.AssumeYes
}
