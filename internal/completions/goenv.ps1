# PowerShell completion for goenv
# Installation: Add to your PowerShell $PROFILE
#   . $env:GOENV_ROOT\completions\goenv.ps1

Register-ArgumentCompleter -CommandName goenv -ScriptBlock {
    param(
        $wordToComplete,
        $commandAst,
        $cursorPosition
    )
    
    # Get all words from the command line
    $words = $commandAst.ToString() -split '\s+' | Where-Object { $_ -ne '' }
    
    # Remove 'goenv' from the beginning
    if ($words.Count -gt 0 -and $words[0] -eq 'goenv') {
        $words = $words[1..($words.Count - 1)]
    }
    
    # Call goenv completions
    try {
        $completions = & goenv completions $words 2>$null
        
        # Return completion results
        $completions | ForEach-Object {
            [System.Management.Automation.CompletionResult]::new(
                $_,
                $_,
                'ParameterValue',
                $_
            )
        }
    }
    catch {
        # Silently fail if goenv command errors
    }
}
