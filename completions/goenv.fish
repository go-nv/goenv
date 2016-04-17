function __fish_goenv_needs_command
  set cmd (commandline -opc)
  if [ (count $cmd) -eq 1 -a $cmd[1] = 'goenv' ]
    return 0
  end
  return 1
end

function __fish_goenv_using_command
  set cmd (commandline -opc)
  if [ (count $cmd) -gt 1 ]
    if [ $argv[1] = $cmd[2] ]
      return 0
    end
  end
  return 1
end

complete -f -c goenv -n '__fish_goenv_needs_command' -a '(goenv commands)'
for cmd in (goenv commands)
  complete -f -c goenv -n "__fish_goenv_using_command $cmd" -a "(goenv completions $cmd)"
end
