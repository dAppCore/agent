# Install the Core CLI (Windows)
# Run: .\scripts\install-core.ps1
#
# REQUIREMENTS:
# - PowerShell 4.0 or later
# - Windows 10/11 or Windows Server 2016+
# - 100MB free disk space
#
# SECURITY CONTROLS:
# - Version pinning prevents supply chain attacks via tag manipulation
# - SHA256 hash verification ensures binary integrity
# - Path validation prevents LOCALAPPDATA redirection attacks
# - Symlink detection prevents directory traversal attacks
# - Restrictive ACLs on temp directories prevent local privilege escalation
# - GPG signature verification (when available) ensures code authenticity
#
# KNOWN LIMITATIONS:
# - Checksums are fetched from same origin as binaries (consider separate trust root)
# - No TLS certificate pinning (relies on system CA store)

$ErrorActionPreference = "Stop"

# Check PowerShell version (4.0+ required for Get-FileHash and other features)
if ($PSVersionTable.PSVersion.Major -lt 4) {
    Write-Host "[ERROR] PowerShell 4.0 or later is required. Current version: $($PSVersionTable.PSVersion)" -ForegroundColor Red
    exit 1
}

$Repo = "host-uk/core"
$Version = "v0.1.0"  # Pinned version - update when releasing new versions
$MinDiskSpaceMB = 100  # Minimum required disk space in MB

function Write-Info { Write-Host "[INFO] $args" -ForegroundColor Green }
function Write-Warn { Write-Host "[WARN] $args" -ForegroundColor Yellow }
function Write-Err { Write-Host "[ERROR] $args" -ForegroundColor Red; exit 1 }

function Test-Command($cmd) {
    return [bool](Get-Command $cmd -ErrorAction SilentlyContinue)
}

# Check available disk space
function Test-DiskSpace {
    param([string]$Path)

    try {
        # Get the drive from the path
        $drive = [System.IO.Path]::GetPathRoot($Path)
        if ([string]::IsNullOrEmpty($drive)) {
            $drive = [System.IO.Path]::GetPathRoot($env:LOCALAPPDATA)
        }

        $driveInfo = Get-PSDrive -Name $drive.Substring(0, 1) -ErrorAction Stop
        $freeSpaceMB = [math]::Round($driveInfo.Free / 1MB, 2)

        if ($freeSpaceMB -lt $MinDiskSpaceMB) {
            Write-Err "Insufficient disk space. Need at least ${MinDiskSpaceMB}MB, but only ${freeSpaceMB}MB available on drive $drive"
        }

        Write-Info "Disk space check passed (${freeSpaceMB}MB available)"
        return $true
    } catch {
        Write-Warn "Could not verify disk space: $($_.Exception.Message)"
        return $true  # Continue anyway if check fails
    }
}

# Validate and get secure install directory
function Get-SecureInstallDir {
    # Validate LOCALAPPDATA is within user profile
    $localAppData = $env:LOCALAPPDATA
    $userProfile = $env:USERPROFILE

    if ([string]::IsNullOrEmpty($localAppData)) {
        Write-Err "LOCALAPPDATA environment variable is not set"
    }

    if ([string]::IsNullOrEmpty($userProfile)) {
        Write-Err "USERPROFILE environment variable is not set"
    }

    # Resolve to absolute paths
    $resolvedLocalAppData = [System.IO.Path]::GetFullPath($localAppData)
    $resolvedUserProfile = [System.IO.Path]::GetFullPath($userProfile)

    # Ensure LOCALAPPDATA is under user profile (prevent redirection attacks)
    if (-not $resolvedLocalAppData.StartsWith($resolvedUserProfile, [System.StringComparison]::OrdinalIgnoreCase)) {
        Write-Err "LOCALAPPDATA ($resolvedLocalAppData) is not within user profile ($resolvedUserProfile). Possible path manipulation detected."
    }

    $installDir = Join-Path $resolvedLocalAppData "Programs\core"

    # Validate the resolved path doesn't contain suspicious patterns
    if ($installDir -match '\.\.' -or $installDir -match '[\x00-\x1f]') {
        Write-Err "Invalid characters detected in install path"
    }

    return $installDir
}

$InstallDir = Get-SecureInstallDir

# Verify directory is safe for writing (not a symlink, correct permissions)
function Test-SecureDirectory {
    param([string]$Path)

    if (-not (Test-Path $Path)) {
        return $true  # Directory doesn't exist yet, will be created
    }

    # Primary check: use fsutil for reliable reparse point detection (matches batch script)
    $fsutilResult = & fsutil reparsepoint query $Path 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Err "Directory '$Path' is a reparse point (symlink or junction). Possible symlink attack detected."
    }

    # Fallback: check .NET attributes
    $dirInfo = Get-Item $Path -Force
    if ($dirInfo.Attributes -band [System.IO.FileAttributes]::ReparsePoint) {
        Write-Err "Directory '$Path' is a symbolic link or junction. Possible symlink attack detected."
    }

    return $true
}

# Verify SHA256 hash of downloaded file
function Test-FileHash {
    param(
        [string]$FilePath,
        [string]$ExpectedHash
    )

    $actualHash = (Get-FileHash -Path $FilePath -Algorithm SHA256).Hash
    if ($actualHash -ne $ExpectedHash) {
        Remove-Item -Path $FilePath -Force -ErrorAction SilentlyContinue
        Write-Err "Hash verification failed! Expected: $ExpectedHash, Got: $actualHash. The downloaded file may be corrupted or tampered with."
    }
    Write-Info "Hash verification passed (SHA256: $($actualHash.Substring(0,16))...)"
}

# Create directory with atomic check-and-create to minimize TOCTOU window
function New-SecureDirectory {
    param([string]$Path)

    # Check parent directory for symlinks first
    $parent = Split-Path $Path -Parent
    if ($parent -and (Test-Path $parent)) {
        Test-SecureDirectory -Path $parent
    }

    # Create directory
    if (-not (Test-Path $Path)) {
        New-Item -ItemType Directory -Force -Path $Path | Out-Null
    }

    # Immediately verify it's not a symlink (minimize TOCTOU window)
    Test-SecureDirectory -Path $Path

    return $Path
}

# Set restrictive ACL on directory (owner-only access)
# This is REQUIRED for temp directories - failure is fatal
function Set-SecureDirectoryAcl {
    param(
        [string]$Path,
        [switch]$Required
    )

    $maxRetries = 3
    $retryCount = 0

    while ($retryCount -lt $maxRetries) {
        try {
            $acl = Get-Acl $Path

            # Disable inheritance and remove inherited rules
            $acl.SetAccessRuleProtection($true, $false)

            # Clear existing rules
            $acl.Access | ForEach-Object { $acl.RemoveAccessRule($_) } | Out-Null

            # Add full control for current user only
            $currentUser = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name
            $rule = New-Object System.Security.AccessControl.FileSystemAccessRule(
                $currentUser,
                [System.Security.AccessControl.FileSystemRights]::FullControl,
                [System.Security.AccessControl.InheritanceFlags]::ContainerInherit -bor [System.Security.AccessControl.InheritanceFlags]::ObjectInherit,
                [System.Security.AccessControl.PropagationFlags]::None,
                [System.Security.AccessControl.AccessControlType]::Allow
            )
            $acl.AddAccessRule($rule)

            Set-Acl -Path $Path -AclObject $acl
            Write-Info "Set restrictive permissions on $Path"
            return $true
        } catch {
            $retryCount++
            if ($retryCount -ge $maxRetries) {
                if ($Required) {
                    Write-Err "SECURITY: Failed to set restrictive ACL on '$Path' after $maxRetries attempts: $($_.Exception.Message)"
                } else {
                    Write-Warn "Could not set restrictive ACL on '$Path': $($_.Exception.Message)"
                    return $false
                }
            }
            Start-Sleep -Milliseconds 100
        }
    }
}

# Download pre-built binary with integrity verification
function Download-Binary {
    $arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
    $binaryUrl = "https://github.com/$Repo/releases/download/$Version/core-windows-$arch.exe"
    $checksumUrl = "https://github.com/$Repo/releases/download/$Version/checksums.txt"

    Write-Info "Attempting to download pre-built binary (version $Version)..."
    Write-Info "URL: $binaryUrl"

    # Track temp file for cleanup
    $tempExe = $null

    try {
        # Create and verify install directory
        New-SecureDirectory -Path $InstallDir

        # Use a temp file in the same directory (same filesystem for atomic move)
        $tempExe = Join-Path $InstallDir "core.exe.tmp.$([System.Guid]::NewGuid().ToString('N').Substring(0,8))"

        # Download the binary to temp location
        Invoke-WebRequest -Uri $binaryUrl -OutFile $tempExe -UseBasicParsing

        # Download and parse checksums
        Write-Info "Verifying download integrity..."
        $checksumContent = Invoke-WebRequest -Uri $checksumUrl -UseBasicParsing | Select-Object -ExpandProperty Content

        # Parse checksum file (format: "hash  filename")
        $expectedHash = $null
        $checksumLines = $checksumContent -split "`n"
        foreach ($line in $checksumLines) {
            if ($line -match "^([a-fA-F0-9]{64})\s+.*core-windows-$arch\.exe") {
                $expectedHash = $matches[1].ToUpper()
                break
            }
        }

        if ([string]::IsNullOrEmpty($expectedHash)) {
            Write-Err "Could not find checksum for core-windows-$arch.exe in checksums.txt"
        }

        # Verify hash BEFORE any move operation
        Test-FileHash -FilePath $tempExe -ExpectedHash $expectedHash

        # Re-verify directory hasn't been replaced with symlink (reduce TOCTOU window)
        Test-SecureDirectory -Path $InstallDir

        # Atomic move to final location (same filesystem)
        $finalPath = Join-Path $InstallDir "core.exe"
        Move-Item -Path $tempExe -Destination $finalPath -Force
        $tempExe = $null  # Clear so finally block doesn't try to delete

        Write-Info "Downloaded and verified: $finalPath"
        return $true
    } catch [System.Net.WebException] {
        Write-Warn "Network error during download: $($_.Exception.Message)"
        Write-Warn "Will attempt to build from source"
        return $false
    } catch [System.IO.IOException] {
        Write-Warn "File system error: $($_.Exception.Message)"
        Write-Warn "Will attempt to build from source"
        return $false
    } catch {
        Write-Warn "Download failed: $($_.Exception.Message)"
        Write-Warn "Will attempt to build from source"
        return $false
    } finally {
        # Clean up temp file if it exists (handles Ctrl+C and other interruptions)
        if ($tempExe -and (Test-Path $tempExe -ErrorAction SilentlyContinue)) {
            Remove-Item -Path $tempExe -Force -ErrorAction SilentlyContinue
        }
    }
}

# Verify GPG signature on git tag (if gpg is available)
function Test-GitTagSignature {
    param(
        [string]$RepoPath,
        [string]$Tag
    )

    if (-not (Test-Command gpg)) {
        Write-Warn "GPG not available - skipping tag signature verification"
        Write-Warn "For enhanced security, install GPG and import the project signing key"
        return $true  # Continue without verification
    }

    Push-Location $RepoPath
    try {
        # Attempt to verify the tag signature
        $result = git tag -v $Tag 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Info "GPG signature verified for tag $Tag"
            return $true
        } else {
            # Check if tag is unsigned vs signature invalid
            if ($result -match "error: no signature found") {
                Write-Warn "Tag $Tag is not signed - continuing without signature verification"
                return $true
            } else {
                Write-Err "GPG signature verification FAILED for tag $Tag. Possible tampering detected."
            }
        }
    } finally {
        Pop-Location
    }
}

# Build from source with security checks
function Build-FromSource {
    if (-not (Test-Command git)) {
        Write-Err "Git is required to build from source. Run '.\scripts\install-deps.ps1' first"
    }
    if (-not (Test-Command go)) {
        Write-Err "Go is required to build from source. Run '.\scripts\install-deps.ps1' first"
    }

    # Create secure temp directory with restrictive ACL
    $tmpdir = Join-Path ([System.IO.Path]::GetTempPath()) "core-build-$([System.Guid]::NewGuid().ToString('N'))"
    New-SecureDirectory -Path $tmpdir

    # ACL is REQUIRED for temp build directories (security critical)
    Set-SecureDirectoryAcl -Path $tmpdir -Required

    try {
        Write-Info "Cloning $Repo (version $Version)..."
        $cloneDir = Join-Path $tmpdir "Core"

        # Clone specific tag for reproducibility
        git clone --depth 1 --branch $Version "https://github.com/$Repo.git" $cloneDir
        if ($LASTEXITCODE -ne 0) {
            Write-Err "Failed to clone repository at version $Version"
        }

        # Verify GPG signature on tag (if available)
        Test-GitTagSignature -RepoPath $cloneDir -Tag $Version

        Write-Info "Building core CLI..."
        Push-Location $cloneDir
        try {
            go build -o core.exe .
            if ($LASTEXITCODE -ne 0) {
                Write-Err "Go build failed"
            }
        } finally {
            Pop-Location
        }

        # Create and verify install directory
        New-SecureDirectory -Path $InstallDir

        # Move built binary to install location
        Move-Item (Join-Path $cloneDir "core.exe") (Join-Path $InstallDir "core.exe") -Force

        Write-Info "Built and installed to $InstallDir\core.exe"
    } finally {
        if (Test-Path $tmpdir) {
            Remove-Item -Recurse -Force $tmpdir -ErrorAction SilentlyContinue
        }
    }
}

# Validate and add to PATH with precise matching
function Setup-Path {
    $userPath = [Environment]::GetEnvironmentVariable("PATH", "User")

    # Validate InstallDir is a real directory under user profile
    if (-not (Test-Path $InstallDir -PathType Container)) {
        Write-Warn "Install directory does not exist, skipping PATH setup"
        return
    }

    $resolvedInstallDir = [System.IO.Path]::GetFullPath($InstallDir)
    $resolvedUserProfile = [System.IO.Path]::GetFullPath($env:USERPROFILE)

    # Ensure we're only adding paths under user profile
    if (-not $resolvedInstallDir.StartsWith($resolvedUserProfile, [System.StringComparison]::OrdinalIgnoreCase)) {
        Write-Warn "Install directory is outside user profile, skipping PATH modification for security"
        return
    }

    # Use precise PATH entry matching (not substring)
    $pathEntries = $userPath -split ';' | ForEach-Object { $_.TrimEnd('\') }
    $normalizedInstallDir = $resolvedInstallDir.TrimEnd('\')

    $alreadyInPath = $pathEntries | Where-Object {
        $_.Equals($normalizedInstallDir, [System.StringComparison]::OrdinalIgnoreCase)
    }

    if (-not $alreadyInPath) {
        Write-Info "Adding $InstallDir to PATH..."
        # Trim trailing semicolons to prevent duplicates
        $cleanPath = $userPath.TrimEnd(';')
        [Environment]::SetEnvironmentVariable("PATH", "$cleanPath;$InstallDir", "User")
        $env:PATH = "$env:PATH;$InstallDir"
    }
}

# Verify installation
function Verify {
    Setup-Path

    if (Test-Command core) {
        Write-Info "Verifying installation..."
        & core --help | Select-Object -First 5
        Write-Host ""
        Write-Info "core CLI installed successfully!"
    } elseif (Test-Path "$InstallDir\core.exe") {
        Write-Info "core CLI installed to $InstallDir\core.exe"
        Write-Info "Restart your terminal to use 'core' command"
    } else {
        Write-Err "Installation failed"
    }
}

# Main
function Main {
    Write-Info "Installing Core CLI (version $Version)..."

    # Check disk space before starting
    Test-DiskSpace -Path $InstallDir

    # Try download first, fallback to build
    if (-not (Download-Binary)) {
        Build-FromSource
    }

    Verify
}

Main
