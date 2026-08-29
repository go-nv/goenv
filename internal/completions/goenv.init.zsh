# Zsh completion for goenv, inlined by "goenv init -".
#
# This variant deliberately uses compctl rather than compdef. compdef is only
# defined after compinit has run, and "eval \"$(goenv init -)\"" frequently
# appears in .zshrc *before* compinit. compctl is a zsh builtin and is always
# available, so registration cannot fail based on .zshrc ordering.
#
# The compdef/#compdef variant is still what "goenv completion zsh" emits for
# installation into fpath.

if [[ -o interactive ]]; then
  _goenv_compctl() {
    local words completions
    read -cA words

    if [ "${#words}" -le 2 ]; then
      completions="$(goenv commands 2>/dev/null)"
    else
      completions="$(goenv completions ${words[2,-2]} 2>/dev/null)"
    fi

    reply=("${(ps:\n:)completions}")
  }

  compctl -K _goenv_compctl goenv
fi
