## Environment variables

You can configure how `goenv` operates with the following settings:

name | default | description
-----|---------|------------
`GOENV_VERSION` | | Specifies the Go version to be used.<br>Also see `goenv help shell`.
`GOENV_ROOT` | `~/.goenv` | Defines the directory under which Go versions and shims reside.<br> Current value shown by `goenv root`.
`GOENV_DEBUG` | | Outputs debug information.<br>Also as: `goenv --debug <subcommand>`
`GOENV_HOOK_PATH` | | Colon-separated list of paths searched for goenv hooks.
`GOENV_DIR` | `$PWD` | Directory to start searching for `.go-version` files.
`GOENV_DISABLE_GOROOT` | `0` | Disables management of `GOROOT`.<br> Set this to `1` if you want to use a `GOROOT` that you export.
`GOENV_DISABLE_GOPATH` | `0` | Disables management of `GOPATH`.<br> Set this to `1`  if you want to use a `GOPATH` that you export. It's recommend that you use this (as set to `0`) to avoid mixing multiple versions of golang packages at `GOPATH` when using different versions of golang. See https://github.com/go-nv/goenv/issues/72#issuecomment-478011438
`GOENV_GOPATH_PREFIX` | `$HOME/go` | `GOPATH` prefix that's exported when `GOENV_DISABLE_GOPATH` is not `1`.<br> E.g in practice it can be `$HOME/go/1.12.0` if you currently use `1.12.0` version of go.
`GOENV_APPEND_GOPATH` | | If `GOPATH` is set, it will be appended to the computed `GOPATH`.
`GOENV_PREPEND_GOPATH` | | If `GOPATH` is set, it will be prepended to the computed `GOPATH`.
`GOENV_GOMOD_VERSION_ENABLE` | | if `GOENV_GOMOD_VERSION_ENABLE` is set to 1, it will try to use the project's `go.mod` file to get the version.
`GOENV_AUTO_INSTALL` | | if `GOENV_AUTO_INSTALL` is set to 1, it will automatically run install if no command arguments specified (just run `goenv`!)
`GOENV_AUTO_INSTALL_FLAGS` | | (Note: only works if `GOENV_AUTO_INSTALL` is set to 1) Appends flags to the auto install command (see `goenv install --help` for all available flags)
`GOENV_RC_FILE` | `$HOME/.goenvrc` | If `GOENV_RC_FILE` is set, it will be modified accordingly.
`GOENV_PATH_ORDER` | | If `GOENV_PATH_ORDER` is set to `front`, `$GOENV_ROOT/shims` will be prepended to the existing `PATH`.Set `GOENV_PATH_ORDER` to a configuration file named by `GOENV_RC_FILE`(e.g. `~/.goenvrc`), for example `GOENV_PATH_ORDER=front` in `~/.goenvrc`.
`GOENV_DISABLE_PATH_WARNING` | `0` | Disables the warning about system Go taking precedence over goenv when shims are at the end of PATH. Set to `1` to disable the warning.
`GOENV_AUTOMATICALLY_DETECT_VERSION` | `0` | If set to `1`, goenv will automatically update `GOROOT` and `GOPATH` when you change directories. This enables automatic version switching based on local `.go-version` files without needing to reload your shell configuration. **Note:** This feature adds a hook that runs on every directory change, which may have a slight performance impact.
