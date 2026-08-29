package completions

import _ "embed"

//go:embed goenv.bash
var Bash string

//go:embed goenv.zsh
var Zsh string

// ZshInit is the zsh completion variant inlined by "goenv init -". It uses the
// compctl builtin so registration works regardless of whether compinit has run
// yet, unlike Zsh which relies on compdef and must be installed into fpath.
//
//go:embed goenv.init.zsh
var ZshInit string

//go:embed goenv.fish
var Fish string

//go:embed goenv.ps1
var PowerShell string
