Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Initialize-AGMPConsole {
    try { & chcp.com 65001 *> $null } catch {}
    $utf8 = New-Object System.Text.UTF8Encoding($false)
    [Console]::InputEncoding = $utf8
    [Console]::OutputEncoding = $utf8
    $global:OutputEncoding = $utf8
}

function Get-AGMPVersion {
    $appGo = Join-Path $script:AGMPRoot 'internal\app\app.go'
    $content = [IO.File]::ReadAllText($appGo)
    $match = [regex]::Match($content, 'Version\s*=\s*"([^"]+)"')
    if (-not $match.Success) { throw '无法从 internal/app/app.go 读取 AGMP 版本。' }
    return $match.Groups[1].Value
}

function Write-AGMPHeader([string]$Title) {
    Write-Host ''
    Write-Host '====================================================================' -ForegroundColor Cyan
    Write-Host ('  ' + $Title) -ForegroundColor Cyan
    Write-Host '====================================================================' -ForegroundColor Cyan
}
function Write-Step([string]$Text) { Write-Host $Text -ForegroundColor Cyan }
function Write-Info([string]$Tag,[string]$Text) { Write-Host ('['+$Tag+'] '+$Text) -ForegroundColor Gray }
function Write-Ok([string]$Text) { Write-Host ('[完成] '+$Text) -ForegroundColor Green }
function Write-Warn([string]$Text) { Write-Host ('[警告] '+$Text) -ForegroundColor Yellow }
function Write-Fail([string]$Text) { Write-Host ('[失败] '+$Text) -ForegroundColor Red }

function Ensure-Directory([string]$Path) {
    if ($script:AGMPDryRun) { return }
    if (-not (Test-Path -LiteralPath $Path -PathType Container)) {
        New-Item -ItemType Directory -Force -Path $Path | Out-Null
    }
}

function Get-AGMPToolPath([string[]]$Names) {
    foreach ($name in $Names) {
        $cmd = Get-Command $name -ErrorAction SilentlyContinue
        if ($cmd -and $cmd.Source) { return $cmd.Source }
    }
    return $null
}

function Invoke-AGMPNative {
    param(
        [Parameter(Mandatory=$true)][string]$FilePath,
        [string[]]$Arguments = @(),
        [string]$WorkingDirectory = $script:AGMPRoot,
        [switch]$AllowFailure,
        [switch]$Quiet,
        [ref]$ExitCode
    )
    if ($script:AGMPDryRun) {
        Write-Info 'DRY-RUN' ((Split-Path -Leaf $FilePath) + ' ' + ($Arguments -join ' '))
        if ($ExitCode) { $ExitCode.Value = 0 }
        return
    }
    if (-not (Test-Path -LiteralPath $FilePath -PathType Leaf) -and -not (Get-Command $FilePath -ErrorAction SilentlyContinue)) {
        throw "找不到可执行程序：$FilePath"
    }
    Push-Location $WorkingDirectory
    try {
        if ($Quiet) { & $FilePath @Arguments *> $null } else { & $FilePath @Arguments }
        $code = $LASTEXITCODE
        if ($null -eq $code) { $code = 0 }
    } finally {
        Pop-Location
    }
    if ($ExitCode) { $ExitCode.Value = [int]$code }
    if (-not $AllowFailure -and $code -ne 0) {
        throw "命令执行失败（退出代码 $code）：$(Split-Path -Leaf $FilePath) $($Arguments -join ' ')"
    }
}

function Get-LocalAppDataRoot {
    if ($env:LOCALAPPDATA) { return $env:LOCALAPPDATA }
    if ($env:TEMP) { return $env:TEMP }
    return $script:AGMPRoot
}

function Set-AGMPNetworkEnvironment {
    $local = 'localhost,127.0.0.1,::1'
    foreach ($name in @('NO_PROXY','no_proxy')) {
        $current = [Environment]::GetEnvironmentVariable($name, 'Process')
        if ([string]::IsNullOrWhiteSpace($current)) {
            [Environment]::SetEnvironmentVariable($name, $local, 'Process')
        } elseif ($current -notmatch '(^|,)localhost(,|$)') {
            [Environment]::SetEnvironmentVariable($name, ($current.TrimEnd(',') + ',' + $local), 'Process')
        }
    }
}

function Test-FileMinimumSize([string]$Path,[long]$MinimumBytes=1024) {
    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) { return $false }
    return (Get-Item -LiteralPath $Path).Length -ge $MinimumBytes
}

function Write-StatusFile([string]$Name,[hashtable]$Values) {
    $dir = Join-Path $script:AGMPRoot 'build'
    Ensure-Directory $dir
    $lines = foreach ($key in $Values.Keys) { "$key=$($Values[$key])" }
    $path = Join-Path $dir $Name
    [IO.File]::WriteAllLines($path, $lines, (New-Object Text.UTF8Encoding($false)))
    return $path
}

function Publish-AGMPArtifact {
    param(
        [Parameter(Mandatory=$true)][string]$Source,
        [Parameter(Mandatory=$true)][string]$Destination,
        [int]$RetryCount = 12,
        [int]$DelayMilliseconds = 250
    )
    if (-not (Test-FileMinimumSize $Source)) { throw "待发布产物无效：$Source" }
    $parent = Split-Path -Parent $Destination
    Ensure-Directory $parent
    $temp = "$Destination.incoming-$PID-$(Get-Date -Format 'yyyyMMddHHmmssfff')"
    Copy-Item -LiteralPath $Source -Destination $temp -Force
    $lastError = $null
    for ($i = 1; $i -le $RetryCount; $i++) {
        try {
            if (Test-Path -LiteralPath $Destination) { Remove-Item -LiteralPath $Destination -Force -ErrorAction Stop }
            Move-Item -LiteralPath $temp -Destination $Destination -Force -ErrorAction Stop
            if ((Get-Item -LiteralPath $Source).Length -ne (Get-Item -LiteralPath $Destination).Length) {
                throw "发布后文件大小校验失败：$Destination"
            }
            return $Destination
        } catch {
            $lastError = $_.Exception.Message
            if ($i -lt $RetryCount) { Start-Sleep -Milliseconds $DelayMilliseconds }
        }
    }
    # Windows 上用户可能正在运行旧 Portable，或 Defender/浏览器短暂持有旧文件。
    # 不再因为一个旧文件锁而让整个 Release 失败；保留本次新产物并给出唯一文件名。
    $dir = Split-Path -Parent $Destination
    $name = [IO.Path]::GetFileNameWithoutExtension($Destination)
    $ext = [IO.Path]::GetExtension($Destination)
    $fallback = Join-Path $dir ("$name-Rebuild-$(Get-Date -Format 'yyyyMMdd-HHmmss')$ext")
    Move-Item -LiteralPath $temp -Destination $fallback -Force
    Write-Warn "目标文件被其他进程占用，已避免覆盖并发布为：$fallback"
    Write-Warn "原始 Windows 错误：$lastError"
    return $fallback
}
