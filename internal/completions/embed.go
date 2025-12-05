package completions

import _ "embed"

//go:embed goenv.bash
var Bash string

//go:embed goenv.zsh
var Zsh string

//go:embed goenv.fish
var Fish string

//go:embed goenv.ps1
var PowerShell string
