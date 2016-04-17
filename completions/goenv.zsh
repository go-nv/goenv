if [[ ! -o interactive ]]; then
    return
fi

compctl -K _goenv goenv

_goenv() {
  local words completions
  read -cA words

  if [ "${#words}" -eq 2 ]; then
    completions="$(goenv commands)"
  else
    completions="$(goenv completions ${words[2,-2]})"
  fi

  reply=(${(ps:\n:)completions})
}
