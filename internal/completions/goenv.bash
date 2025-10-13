#!/usr/bin/env bash
# Bash completion for goenv

_goenv() {
    COMPREPLY=()
    local cur="${COMP_WORDS[COMP_CWORD]}"
    local prev="${COMP_WORDS[COMP_CWORD-1]}"
    
    # Build the command line for goenv completions
    local words=("${COMP_WORDS[@]:1}")
    
    # Get completions from goenv
    local completions
    completions=$(goenv completions "${words[@]}" 2>/dev/null)
    
    # Generate completion matches
    COMPREPLY=( $(compgen -W "$completions" -- "$cur") )
    
    return 0
}

# Register the completion function
complete -F _goenv goenv
