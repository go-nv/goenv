# goenv installer script for Windows PowerShell
# Usage: iwr -useb https://raw.githubusercontent.com/go-nv/goenv/main/install.ps1 | iex

$ErrorActionPreference = "Stop"

# Ensure TLS 1.2 for GitHub API/downloads (older Windows defaults to TLS 1.0)
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

# Configuration
$GOENV_ROOT = if ($env:GOENV_ROOT) { $env:GOENV_ROOT } else { Join-Path (if ($env:USERPROFILE) { $env:USERPROFILE } else { $HOME }) ".goenv" }
$GITHUB_REPO = "go-nv/goenv"
$INSTALL_DIR = Join-Path $GOENV_ROOT "bin"

# Colors — use Write-Host which works in non-interactive/piped contexts
function Write-ColorOutput {
    param(
        [Parameter(Position=0)]
        [System.ConsoleColor]$ForegroundColor,
        [Parameter(Position=1, ValueFromRemainingArguments)]
        [string[]]$Message
    )
    Write-Host ($Message -join ' ') -ForegroundColor $ForegroundColor
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
            Copy-Item -Path $binaryPath -Destination (Join-Path $INSTALL_DIR "goenv.exe") -Force
        } else {
            throw "Binary not found in archive"
        }
        
        # Copy completions if they exist
        $completionsPath = Join-Path $tmpDir "completions"
        if (Test-Path $completionsPath) {
            $targetCompletions = Join-Path $GOENV_ROOT "completions"
            New-Item -ItemType Directory -Path $targetCompletions -Force | Out-Null
            Copy-Item -Path "$completionsPath\*" -Destination $targetCompletions -Recurse -Force -ErrorAction SilentlyContinue
        }
        
        Write-ColorOutput Green "goenv installed successfully!"
        
        # Remove stale goenv shim from v2 installations.
        # v2's goenv-rehash bakes the Cellar/libexec path into shims at creation time.
        # Only remove if it contains "libexec/goenv" — the v2 fingerprint.
        $staleShim = Join-Path (Join-Path $GOENV_ROOT "shims") "goenv"
        if (Test-Path $staleShim) {
            $shimContent = Get-Content $staleShim -Raw -ErrorAction SilentlyContinue
            if ($shimContent -and $shimContent -match "libexec/goenv") {
                Write-ColorOutput Yellow "Removing stale v2 goenv shim..."
                Remove-Item -Path $staleShim -Force -ErrorAction SilentlyContinue
                Write-ColorOutput Green "Stale shim removed"
            }
        }
    }
    catch {
        Write-ColorOutput Red "Installation failed: $_"
        exit 1
    }
    finally {
        # Cleanup temp directory
        if (Test-Path $tmpDir) {
            Remove-Item -Path $tmpDir -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
}

# Auto-configure PowerShell profile
function Initialize-PowerShellProfile {
    $profilePath = $PROFILE
    
    # Create profile directory if it doesn't exist
    $profileDir = Split-Path -Parent $profilePath
    if (-not (Test-Path $profileDir)) {
        New-Item -ItemType Directory -Path $profileDir -Force | Out-Null
    }
    
    # Create profile file if it doesn't exist
    if (-not (Test-Path $profilePath)) {
        New-Item -ItemType File -Path $profilePath -Force | Out-Null
    }
    
    # Check if goenv is already configured
    $profileContent = Get-Content $profilePath -Raw -ErrorAction SilentlyContinue
    if ($profileContent -and $profileContent -match "goenv init") {
        Write-ColorOutput Green "goenv is already configured in $profilePath"
        return
    }
    
    # Add goenv configuration with comment marker
    Write-ColorOutput Yellow "Adding goenv configuration to $profilePath..."
    
    $goenvConfig = @"

# goenv - Go version manager (auto-configured by installer)
`$env:GOENV_ROOT = "`$HOME\.goenv"
`$env:PATH = "`$env:GOENV_ROOT\bin;`$env:PATH"
& goenv init - | Invoke-Expression
"@
    
    Add-Content -Path $profilePath -Value $goenvConfig
    Write-ColorOutput Green "PowerShell profile configured successfully!"
}

# Print setup completion message
function Show-Instructions {
    $profilePath = $PROFILE
    
    Write-Output ""
    Write-ColorOutput Green "=============================================="
    Write-ColorOutput Green "Installation complete!"
    Write-ColorOutput Green "=============================================="
    Write-Output ""
    Write-ColorOutput Yellow "To start using goenv, reload your profile:"
    Write-Output "  . `$PROFILE"
    Write-Output ""
    Write-ColorOutput Yellow "Or restart your PowerShell session"
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
    Initialize-PowerShellProfile
    Show-Instructions
}

# Run main
Main
