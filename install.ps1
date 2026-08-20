# AtomGit CLI (ag) - Windows installer
# Repository: https://atomgit.com/hust-open-atom-club/atomgit-cli
#
# Usage (PowerShell):
#   irm https://atomgit.com/hust-open-atom-club/atomgit-cli/releases/download/latest/install.ps1 | iex
#   $env:AG_VERSION = "vX.Y.Z"; .\install.ps1
#
# After installation, run `ag auth login` to authenticate with OAuth.
# Default token file: %USERPROFILE%\.config\ag-cli\token.json
#
# If execution policy blocks the script, run:
#   Set-ExecutionPolicy -Scope CurrentUser RemoteSigned

#Requires -Version 5.1

$ErrorActionPreference = "Stop"

# Release builds replace this placeholder with the current tag. Set AG_VERSION
# when running the source template directly.
$BundledTag = '__AG_RELEASE_TAG__'

function Die([string]$Msg) {
    $host.UI.WriteErrorLine("install.ps1: $Msg")
    exit 1
}

$RepoOwner = if ($env:AG_REPO_OWNER) { $env:AG_REPO_OWNER } else { "hust-open-atom-club" }
$RepoName = if ($env:AG_REPO_NAME) { $env:AG_REPO_NAME } else { "atomgit-cli" }
$BaseUrl = "https://atomgit.com/$RepoOwner/$RepoName"

$Version = if ($env:AG_VERSION) { $env:AG_VERSION }
elseif ($env:AG_DEFAULT_VERSION) { $env:AG_DEFAULT_VERSION }
else { $BundledTag }

if ($Version -eq '__AG_RELEASE_TAG__') {
    Die 'The source installer is not bound to a release. Set AG_VERSION=vX.Y.Z or use install.ps1 from an AtomGit Release.'
}

function Get-GoArch {
    $proc = [Environment]::GetEnvironmentVariable("PROCESSOR_ARCHITECTURE")
    $wow = [Environment]::GetEnvironmentVariable("PROCESSOR_ARCHITEW6432")
    if ($proc -eq "ARM64") { return "arm64" }
    if ($proc -eq "AMD64" -or $wow -eq "AMD64") { return "amd64" }
    Die "unsupported CPU architecture: $proc (need amd64 or arm64 Windows build)"
}

function Get-InstallDir {
    if ($env:AG_INSTALL_DIR) { return $env:AG_INSTALL_DIR }
    $prefix = $env:AG_PREFIX
    if (-not $prefix) {
        return Join-Path $env:USERPROFILE ".local\bin"
    }
    return Join-Path $prefix "bin"
}

$arch = Get-GoArch
$asset = "ag_windows_${arch}.zip"
$url = "$BaseUrl/releases/download/$Version/$asset"

$tmp = Join-Path $env:TEMP ("ag-install-" + [guid]::NewGuid().ToString())
New-Item -ItemType Directory -Path $tmp | Out-Null
try {
    Write-Host "Downloading $url ..."
    try {
        Invoke-WebRequest -Uri $url -OutFile (Join-Path $tmp $asset) -UseBasicParsing
    }
    catch {
        Die "Failed to download the prebuilt archive. Verify that $Version is published with an asset named $asset. Error: $_"
    }

    $extract = Join-Path $tmp "extract"
    New-Item -ItemType Directory -Path $extract | Out-Null
    Expand-Archive -Path (Join-Path $tmp $asset) -DestinationPath $extract -Force

    $exe = Join-Path $extract "ag.exe"
    if (-not (Test-Path -LiteralPath $exe)) {
        Die "ag.exe was not found in the archive"
    }

    $dest = Get-InstallDir
    if (-not (Test-Path -LiteralPath $dest)) {
        New-Item -ItemType Directory -Path $dest -Force | Out-Null
    }

    $target = Join-Path $dest "ag.exe"
    Copy-Item -LiteralPath $exe -Destination $target -Force
    Write-Host "Installing to $target"

    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -notlike "*$dest*") {
        $newPath = if ($userPath) { "$userPath;$dest" } else { $dest }
        [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
        $env:Path += ";$dest"
        Write-Host "Added directory to the current user PATH: $dest"
        Write-Host "Reopen the terminal if ag is not available in this window."
    }

    & $target --help | Out-Null
    Write-Host "ag installed: $target"
}
finally {
    Remove-Item -LiteralPath $tmp -Recurse -Force -ErrorAction SilentlyContinue
}
