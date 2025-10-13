#compdef goenv
# Zsh completion for goenv

_goenv() {
    local -a completions
    local -a words
    
    # Get the command line words
    words=(${(z)LBUFFER})
    
    # Remove 'goenv' from the beginning
    if [[ ${#words[@]} -gt 0 && "${words[1]}" == "goenv" ]]; then
        words=("${words[@]:1}")
    fi
    
    # Get completions from goenv
    completions=("${(@f)$(goenv completions ${words[@]} 2>/dev/null)}")
    
    # Provide completions
    _describe 'goenv' completions
}

# Register the completion function
compdef _goenv goenv
