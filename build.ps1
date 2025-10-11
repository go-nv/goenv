<#
.SYNOPSIS
    Build script for goenv on Windows (PowerShell)

.DESCRIPTION
    This script provides build functionality for Windows developers.
    It's the PowerShell equivalent of the Unix Makefile.

.PARAMETER Task
    The build task to run: build, test, clean, install, uninstall, dev-deps, cross-build, version

.EXAMPLE
    .\build.ps1 build
    .\build.ps1 test
    .\build.ps1 -Task clean

.NOTES
    Requires Go to be installed and in PATH
#>

param(
    [Parameter(Position=0)]
    [ValidateSet('build', 'test', 'clean', 'install', 'uninstall', 'dev-deps', 'cross-build', 'version', 'help')]
    [string]$Task = 'build'
)

# Build variables
$BinaryName = "goenv.exe"
$Version = if (Test-Path "APP_VERSION") { Get-Content "APP_VERSION" -Raw | ForEach-Object { $_.Trim() } } else { "dev" }
$CommitSha = if (Get-Command git -ErrorAction SilentlyContinue) {
    git rev-parse --short HEAD 2>$null
    if (-not $?) { "unknown" }
} else { "unknown" }
$BuildTime = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
$LDFlags = "-ldflags `"-X main.version=$Version -X main.commit=$CommitSha -X main.buildTime=$BuildTime`""

# Default installation prefix
$env:PREFIX = if ($env:PREFIX) { $env:PREFIX } else { "$env:LOCALAPPDATA\goenv" }

function Show-Help {
    Write-Host @"
goenv Build Script for Windows

Usage: .\build.ps1 [TASK]

Tasks:
  build         Build the goenv binary (default)
  test          Run all tests
  clean         Remove built binaries and clean build artifacts
  install       Install goenv to PREFIX (default: %LOCALAPPDATA%\goenv)
  uninstall     Uninstall goenv from PREFIX
  dev-deps      Download and tidy Go module dependencies
  cross-build   Build for multiple platforms (Linux, macOS, FreeBSD, Windows)
  version       Show version information
  help          Show this help message

Environment Variables:
  PREFIX        Installation prefix (default: %LOCALAPPDATA%\goenv)

Examples:
  .\build.ps1 build
  .\build.ps1 test
  `$env:PREFIX = "C:\tools\goenv"; .\build.ps1 install

"@
}

function Build {
    Write-Host "Building $BinaryName..." -ForegroundColor Cyan
    $cmd = "go build $LDFlags -o $BinaryName ."
    Write-Host "Running: $cmd" -ForegroundColor Gray
    Invoke-Expression $cmd
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Build successful: $BinaryName" -ForegroundColor Green

        # Create bin directory for backward compatibility
        if (-not (Test-Path "bin")) {
            New-Item -ItemType Directory -Path "bin" | Out-Null
        }
        Copy-Item $BinaryName "bin\goenv.exe" -Force
        Write-Host "Copied to bin\goenv.exe for backward compatibility" -ForegroundColor Green
    } else {
        Write-Host "Build failed!" -ForegroundColor Red
        exit 1
    }
}

function Test {
    Write-Host "Running tests..." -ForegroundColor Cyan
    go test -v ./...
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Tests passed!" -ForegroundColor Green
    } else {
        Write-Host "Tests failed!" -ForegroundColor Red
        exit 1
    }
}

function Clean {
    Write-Host "Cleaning build artifacts..." -ForegroundColor Cyan

    if (Test-Path $BinaryName) {
        Remove-Item $BinaryName -Force
        Write-Host "Removed $BinaryName" -ForegroundColor Gray
    }

    if (Test-Path "bin") {
        Remove-Item "bin" -Recurse -Force
        Write-Host "Removed bin directory" -ForegroundColor Gray
    }

    if (Test-Path "dist") {
        Remove-Item "dist" -Recurse -Force
        Write-Host "Removed dist directory" -ForegroundColor Gray
    }

    go clean
    Write-Host "Clean complete!" -ForegroundColor Green
}

function Install {
    Write-Host "Installing goenv to $env:PREFIX..." -ForegroundColor Cyan

    # Build first
    Build

    # Create installation directories
    if (-not (Test-Path "$env:PREFIX\bin")) {
        New-Item -ItemType Directory -Path "$env:PREFIX\bin" -Force | Out-Null
    }

    # Copy binary
    Copy-Item $BinaryName "$env:PREFIX\bin\goenv.exe" -Force
    Write-Host "Installed goenv.exe to $env:PREFIX\bin" -ForegroundColor Green

    # Install shell completions
    if (Test-Path "completions") {
        $completionsPath = "$env:PREFIX\share\goenv\completions"
        if (-not (Test-Path $completionsPath)) {
            New-Item -ItemType Directory -Path $completionsPath -Force | Out-Null
        }
        Copy-Item "completions\*" $completionsPath -Recurse -Force -ErrorAction SilentlyContinue
        Write-Host "Installed completions to $completionsPath" -ForegroundColor Green
    }

    Write-Host ""
    Write-Host "Installation complete!" -ForegroundColor Green
    Write-Host "Add $env:PREFIX\bin to your PATH to use goenv" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "PowerShell:" -ForegroundColor Cyan
    Write-Host "  `$env:PATH = `"$env:PREFIX\bin;`$env:PATH`"" -ForegroundColor Gray
    Write-Host ""
}

function Uninstall {
    Write-Host "Uninstalling goenv from $env:PREFIX..." -ForegroundColor Cyan

    if (Test-Path "$env:PREFIX\bin\goenv.exe") {
        Remove-Item "$env:PREFIX\bin\goenv.exe" -Force
        Write-Host "Removed goenv.exe" -ForegroundColor Gray
    }

    if (Test-Path "$env:PREFIX\share\goenv") {
        Remove-Item "$env:PREFIX\share\goenv" -Recurse -Force
        Write-Host "Removed completions" -ForegroundColor Gray
    }

    Write-Host "Uninstall complete!" -ForegroundColor Green
}

function Dev-Deps {
    Write-Host "Downloading Go module dependencies..." -ForegroundColor Cyan
    go mod download

    Write-Host "Tidying Go modules..." -ForegroundColor Cyan
    go mod tidy

    Write-Host "Dependencies updated!" -ForegroundColor Green
}

function Cross-Build {
    Write-Host "Building for multiple platforms..." -ForegroundColor Cyan

    if (-not (Test-Path "dist")) {
        New-Item -ItemType Directory -Path "dist" | Out-Null
    }

    $platforms = @(
        @{OS="linux"; Arch="amd64"},
        @{OS="linux"; Arch="arm64"},
        @{OS="darwin"; Arch="amd64"},
        @{OS="darwin"; Arch="arm64"},
        @{OS="windows"; Arch="amd64"},
        @{OS="windows"; Arch="386"},
        @{OS="freebsd"; Arch="amd64"}
    )

    foreach ($platform in $platforms) {
        $goos = $platform.OS
        $goarch = $platform.Arch
        $output = "dist\goenv-$goos-$goarch"
        if ($goos -eq "windows") {
            $output += ".exe"
        }

        Write-Host "Building for $goos/$goarch..." -ForegroundColor Gray
        $env:GOOS = $goos
        $env:GOARCH = $goarch
        $cmd = "go build $LDFlags -o $output ."
        Invoke-Expression $cmd

        if ($LASTEXITCODE -eq 0) {
            Write-Host "  ✓ $output" -ForegroundColor Green
        } else {
            Write-Host "  ✗ Failed to build for $goos/$goarch" -ForegroundColor Red
        }
    }

    # Reset environment
    Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
    Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue

    Write-Host ""
    Write-Host "Cross-compilation complete! Binaries in dist/" -ForegroundColor Green
}

function Show-Version {
    Write-Host "goenv Build Information" -ForegroundColor Cyan
    Write-Host "  Version:    $Version" -ForegroundColor Gray
    Write-Host "  Commit:     $CommitSha" -ForegroundColor Gray
    Write-Host "  Build Time: $BuildTime" -ForegroundColor Gray
}

# Main execution
switch ($Task) {
    'build'       { Build }
    'test'        { Test }
    'clean'       { Clean }
    'install'     { Install }
    'uninstall'   { Uninstall }
    'dev-deps'    { Dev-Deps }
    'cross-build' { Cross-Build }
    'version'     { Show-Version }
    'help'        { Show-Help }
    default       { Show-Help }
}
