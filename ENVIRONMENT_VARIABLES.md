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
`GOENV_DISABLE_GOPATH` | `0` | Disables management of `GOPATH`.<br> Set this to `1`  if you want to use a `GOPATH` that you export. It's recommend that you use this (as set to `0`) to avoid mixing multiple versions of golang packages at `GOPATH` when using different versions of golang. See https://github.com/syndbg/goenv/issues/72#issuecomment-478011438
`GOENV_GOPATH_PREFIX` | `$HOME/go` | `GOPATH` prefix that's exported when `GOENV_DISABLE_GOPATH` is not `1`.<br> E.g in practice it can be `$HOME/go/1.12.0` if you currently use `1.12.0` version of go.
