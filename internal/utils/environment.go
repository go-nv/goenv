package utils

import (
	"os"
	"strconv"
)

type GoenvEnvVar string
type GoenvBoolEnv = GoenvEnvVar

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
	GoenvEnvVarInstallRoot      GoenvEnvVar = "GOENV_INSTALL_ROOT"
	// #endregion

	// #region Bool Env Vars
	GoenvEnvVarDisableGoroot GoenvBoolEnv = "GOENV_DISABLE_GOROOT"
	GoenvEnvVarDisableGopath GoenvBoolEnv = "GOENV_DISABLE_GOPATH"
	GoenvEnvVarGopathPrefix  GoenvBoolEnv = "GOENV_GOPATH_PREFIX"
	GoenvEnvVarAppendGopath  GoenvBoolEnv = "GOENV_APPEND_GOPATH"
	GoenvEnvVarPrependGopath GoenvBoolEnv = "GOENV_PREPEND_GOPATH"
	GoenvEnvVarAutoInstall   GoenvBoolEnv = "GOENV_AUTO_INSTALL"
	GoenvEnvVarNoAutoRehash  GoenvBoolEnv = "GOENV_NO_AUTO_REHASH"
	GoenvEnvVarOffline       GoenvBoolEnv = "GOENV_OFFLINE"
	GoenvEnvVarAutoRehash    GoenvBoolEnv = "GOENV_AUTO_REHASH"
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
func (g GoenvBoolEnv) IsTrue() bool {
	val, ok := g.Value()
	if !ok {
		return false
	}

	isSet, err := strconv.ParseBool(val)

	return err == nil && isSet
}
