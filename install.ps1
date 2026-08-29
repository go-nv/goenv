# goenv installer script for Windows PowerShell
# Usage: iwr -useb https://raw.githubusercontent.com/go-nv/goenv/main/install.ps1 | iex

$ErrorActionPreference = "Stop"

# Ensure TLS 1.2 for GitHub API/downloads (older Windows defaults to TLS 1.0)
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

# Configuration
# Use if/else instead of ?? operator for PowerShell 5.1 compatibility
$defaultHome = if ($env:USERPROFILE) { $env:USERPROFILE } else { $HOME }
$GOENV_ROOT = if ($env:GOENV_ROOT) { $env:GOENV_ROOT } else { Join-Path $defaultHome ".goenv" }
$GITHUB_REPO = "go-nv/goenv"
$INSTALL_DIR = Join-Path $GOENV_ROOT "bin"

# API base URL. Overridable for GitHub Enterprise, mirrors, and for CI tests.
$GITHUB_API = if ($env:GOENV_GITHUB_API) { $env:GOENV_GITHUB_API } else { "https://api.github.com" }

# Colors — use Write-Host which works in non-interactive/piped contexts
function Write-ColorOutput {
  param(
    [Parameter(Position = 0)]
    [System.ConsoleColor]$ForegroundColor,
    [Parameter(Position = 1, ValueFromRemainingArguments)]
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

# Get the newest release that actually ships a Windows binary for this
# architecture.
#
# /releases/latest cannot be used: it returns the most recently published
# release across *all* branches, and the v2 maintenance branch publishes
# documentation-only releases that carry no binary assets at all. Installing
# from one of those fails with a 404 on the archive download (issue #582).
function Get-LatestVersion {
  param (
    [Parameter(Mandatory = $true)]
    [string]$Arch
  )

  Write-ColorOutput Yellow "Fetching latest release..."

  try {
    # Send an Authorization header when a token is available. Unauthenticated
    # GitHub API access is limited to 60 requests/hour per IP, which CI
    # runners routinely exhaust because their addresses are shared.
    $headers = @{}
    $token = if ($env:GOENV_GITHUB_TOKEN) { $env:GOENV_GITHUB_TOKEN }
    elseif ($env:GITHUB_TOKEN) { $env:GITHUB_TOKEN }
    elseif ($env:GH_TOKEN) { $env:GH_TOKEN }
    else { $null }
    if ($token) {
      $headers["Authorization"] = "Bearer $token"
    }

    # GitHub returns releases newest-first.
    $releases = Invoke-RestMethod -Uri "$GITHUB_API/repos/$GITHUB_REPO/releases?per_page=50" -Headers $headers
    foreach ($release in $releases) {
      if ($release.draft -or $release.prerelease) {
        continue
      }

      $versionNumber = $release.tag_name.TrimStart('v')
      $expectedAsset = "goenv_${versionNumber}_windows_${Arch}.zip"

      $hasAsset = $release.assets | Where-Object { $_.name -eq $expectedAsset }
      if ($hasAsset) {
        Write-ColorOutput Green "Latest version: $($release.tag_name)"
        return $release.tag_name
      }
    }

    throw "No release found containing a Windows $Arch binary (goenv_<version>_windows_${Arch}.zip)"
  }
  catch {
    $status = $null
    if ($_.Exception.Response) {
      $status = [int]$_.Exception.Response.StatusCode
    }

    if ($status -eq 403 -or $status -eq 429) {
      Write-ColorOutput Red "The GitHub releases API refused the request (HTTP $status)."
      Write-ColorOutput Red "This is usually the API rate limit (60 requests/hour for unauthenticated"
      Write-ColorOutput Red "callers, counted per IP address - shared CI runners hit it routinely)."
      Write-ColorOutput Red "Set GITHUB_TOKEN (or GOENV_GITHUB_TOKEN) to raise the limit."
    }
    else {
      Write-ColorOutput Red "Error fetching latest version: $_"
    }
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
    }
    else {
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
    # Match both forward and backslashes to support all v2 installations.
    # Use nested Join-Path for PowerShell 5.1 compatibility (no 3-arg support).
    $shimsPath = Join-Path $GOENV_ROOT "shims"
    $staleShim = Join-Path $shimsPath "goenv"
    if (Test-Path $staleShim) {
      $shimContent = Get-Content $staleShim -Raw -ErrorAction SilentlyContinue
      if ($shimContent -and $shimContent -match "libexec[\\/]goenv") {
        Write-ColorOutput Yellow "Removing stale v2 goenv shim..."
        $removed = Remove-Item -Path $staleShim -Force -ErrorAction SilentlyContinue -PassThru
        if ($removed) {
          Write-ColorOutput Green "Stale shim removed"
        }
        else {
          Write-ColorOutput Yellow "Warning: Failed to remove stale v2 goenv shim"
        }
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
    
  $version = Get-LatestVersion -Arch $arch
  Install-Binary -Version $version -Arch $arch
  Initialize-PowerShellProfile
  Show-Instructions
}

# Run main
Main
