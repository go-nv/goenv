<#
.SYNOPSIS
    Cross-platform build wrapper for Windows (PowerShell)

.DESCRIPTION
    This script delegates to the unified Go-based build tool.
    All build logic is now consolidated in scripts/build-tool/main.go

.PARAMETER Task
    The build task to run: build, test, clean, install, uninstall, dev-deps, cross-build, version, etc.

.PARAMETER Prefix
    Installation prefix for install command

.EXAMPLE
    .\build.ps1 build
    .\build.ps1 test
    .\build.ps1 install -Prefix "C:\tools\goenv"

.NOTES
    Requires Go to be installed and in PATH
#>

param(
    [Parameter(Position=0)]
    [string]$Task = 'build',
    
    [Parameter()]
    [string]$Prefix = ""
)

# Build arguments for the Go build tool
$args = @()
$args += "-task=$Task"

if ($Prefix -ne "") {
    $args += "-prefix=$Prefix"
}

# Delegate to the unified Go-based build tool
& go run scripts/build-tool/main.go @args
exit $LASTEXITCODE
