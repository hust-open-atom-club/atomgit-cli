# AtomGit CLI (ag) — Windows 一键安装
# 仓库: https://atomgit.com/hust-open-atom-club/atomgit-cli
#
# 用法（在 PowerShell 中）:
#   irm https://atomgit.com/hust-open-atom-club/atomgit-cli/releases/download/latest/install.ps1 | iex
#   $env:AG_VERSION = "vX.Y.Z"; .\install.ps1
#
# 安装完成后执行 `ag auth login` 完成 OAuth 认证。
# 令牌文件默认路径: %USERPROFILE%\.config\ag-cli\token.json
#
# 若提示执行策略，可先: Set-ExecutionPolicy -Scope CurrentUser RemoteSigned

#Requires -Version 5.1

$ErrorActionPreference = "Stop"

# Release 构建会将占位符替换为当前 tag；直接运行源码模板时需设置 AG_VERSION。
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
    Die '源码安装器模板尚未绑定发布版本；请设置 AG_VERSION=vX.Y.Z，或使用 AtomGit Release 中的 install.ps1'
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
        Die "无法下载预编译包（请确认已发布 $Version 且附件名为 $asset）。错误: $_"
    }

    $extract = Join-Path $tmp "extract"
    New-Item -ItemType Directory -Path $extract | Out-Null
    Expand-Archive -Path (Join-Path $tmp $asset) -DestinationPath $extract -Force

    $exe = Join-Path $extract "ag.exe"
    if (-not (Test-Path -LiteralPath $exe)) {
        Die "压缩包内未找到 ag.exe"
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
        Write-Host "已将目录加入当前用户 PATH: $dest"
        Write-Host "若本窗口仍找不到 ag，请重新打开终端。"
    }

    & $target --help | Out-Null
    Write-Host "ag 已安装: $target"
}
finally {
    Remove-Item -LiteralPath $tmp -Recurse -Force -ErrorAction SilentlyContinue
}
