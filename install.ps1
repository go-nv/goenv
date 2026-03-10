# goenv installer script for Windows PowerShell
# Usage: iwr -useb https://raw.githubusercontent.com/go-nv/goenv/master/install.ps1 | iex

$ErrorActionPreference = "Stop"

# Configuration
$GOENV_ROOT = if ($env:GOENV_ROOT) { $env:GOENV_ROOT } else { "$HOME\.goenv" }
$GITHUB_REPO = "go-nv/goenv"
$INSTALL_DIR = "$GOENV_ROOT\bin"

# Colors
function Write-ColorOutput($ForegroundColor) {
    $fc = $host.UI.RawUI.ForegroundColor
    $host.UI.RawUI.ForegroundColor = $ForegroundColor
    if ($args) {
        Write-Output $args
    }
    $host.UI.RawUI.ForegroundColor = $fc
}

# Detect architecture
function Get-Architecture {
    $arch = $env:PROCESSOR_ARCHITECTURE
    switch ($arch) {
        "AMD64" { return "amd64" }
        "x86_64" { return "amd64" }
        "ARM64" { return "arm64" }
        default {
            Write-ColorOutput Red "Unsupported architecture: $arch"
            exit 1
        }
    }
}

# Get latest release version
function Get-LatestVersion {
    Write-ColorOutput Yellow "Fetching latest release..."
    
    try {
        $response = Invoke-RestMethod -Uri "https://api.github.com/repos/$GITHUB_REPO/releases/latest"
        $version = $response.tag_name
        
        if (-not $version) {
            throw "Failed to fetch latest version"
        }
        
        Write-ColorOutput Green "Latest version: $version"
        return $version
    }
    catch {
        Write-ColorOutput Red "Error fetching latest version: $_"
        exit 1
    }
}

# Download and install binary
function Install-Binary {
    param (
        [string]$Version,
        [string]$Arch
    )
    
    $versionNumber = $Version.TrimStart('v')
    $archiveName = "goenv_${versionNumber}_windows_${Arch}.zip"
    $downloadUrl = "https://github.com/$GITHUB_REPO/releases/download/$Version/$archiveName"
    $tmpDir = Join-Path $env:TEMP "goenv-install-$(Get-Random)"
    
    Write-ColorOutput Yellow "Downloading goenv..."
    Write-Output "URL: $downloadUrl"
    
    try {
        # Create temp directory
        New-Item -ItemType Directory -Path $tmpDir -Force | Out-Null
        
        # Download archive
        $archivePath = Join-Path $tmpDir $archiveName
        Invoke-WebRequest -Uri $downloadUrl -OutFile $archivePath -UseBasicParsing
        
        Write-ColorOutput Yellow "Extracting archive..."
        Expand-Archive -Path $archivePath -DestinationPath $tmpDir -Force
        
        Write-ColorOutput Yellow "Installing to $INSTALL_DIR..."
        New-Item -ItemType Directory -Path $INSTALL_DIR -Force | Out-Null
        
        # Copy binary
        $binaryPath = Join-Path $tmpDir "goenv.exe"
        if (Test-Path $binaryPath) {
            Copy-Item -Path $binaryPath -Destination "$INSTALL_DIR\goenv.exe" -Force
        } else {
            throw "Binary not found in archive"
        }
        
        # Copy completions if they exist
        $completionsPath = Join-Path $tmpDir "completions"
        if (Test-Path $completionsPath) {
            $targetCompletions = "$GOENV_ROOT\completions"
            New-Item -ItemType Directory -Path $targetCompletions -Force | Out-Null
            Copy-Item -Path "$completionsPath\*" -Destination $targetCompletions -Recurse -Force -ErrorAction SilentlyContinue
        }
        
        Write-ColorOutput Green "goenv installed successfully!"
    }
    catch {
        Write-ColorOutput Red "Installation failed: $_"
        exit 1
    }
    finally {
        # Cleanup
        if (Test-Path $tmpDir) {
            Remove-Item -Path $tmpDir -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
}

# Print setup instructions
function Show-Instructions {
    $profilePath = $PROFILE
    
    Write-Output ""
    Write-ColorOutput Green "=============================================="
    Write-ColorOutput Green "Installation complete!"
    Write-ColorOutput Green "=============================================="
    Write-Output ""
    Write-ColorOutput Yellow "Add the following to your PowerShell profile:"
    Write-Output "  $profilePath"
    Write-Output ""
    Write-Output "  `$env:GOENV_ROOT = \"`$HOME\.goenv\""
    Write-Output "  `$env:PATH = \"`$env:GOENV_ROOT\bin;`$env:PATH\""
    Write-Output "  & goenv init - | Invoke-Expression"
    Write-Output ""
    Write-ColorOutput Yellow "Quick setup command (copy and paste):"
    Write-Output ""
    Write-Output "  `$env:GOENV_ROOT = \"`$HOME\.goenv\""
    Write-Output "  `$env:PATH = \"`$env:GOENV_ROOT\bin;`$env:PATH\""
    Write-Output "  & goenv init - | Invoke-Expression"
    Write-Output ""
    Write-ColorOutput Yellow "Then reload your profile:"
    Write-Output "  . `$PROFILE"
    Write-Output ""
    Write-ColorOutput Yellow "Quick start:"
    Write-Output "  goenv install 1.22.0     # Install Go 1.22.0"
    Write-Output "  goenv global 1.22.0      # Set as default"
    Write-Output "  goenv versions           # List installed versions"
    Write-Output ""
    Write-ColorOutput Yellow "Enable tab completion (optional):"
    Write-Output "  goenv completion --install"
    Write-Output ""
    Write-ColorOutput Green "=============================================="
}

# Main installation flow
function Main {
    Write-ColorOutput Green "goenv installer for Windows"
    Write-Output ""
    
    $arch = Get-Architecture
    Write-ColorOutput Green "Detected architecture: windows_$arch"
    
    $version = Get-LatestVersion
    Install-Binary -Version $version -Arch $arch
    Show-Instructions
}

# Run main
Main
