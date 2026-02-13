# Fish completion for goenv
# This script is embedded in the goenv binary
# Install with: goenv completion fish --install
# Or manually: goenv completion fish >> ~/.config/fish/config.fish

function __goenv_completions
    # Get the current command line tokens
    set -l cmd (commandline -opc)
    
    # Remove 'goenv' from the beginning if present
    if test (count $cmd) -gt 0
        set cmd $cmd[2..-1]
    end
    
    # Get completions from goenv
    goenv completions $cmd 2>/dev/null
end

# Register completion for goenv
complete -c goenv -f -a '(__goenv_completions)'
