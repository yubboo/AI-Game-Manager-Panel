[CmdletBinding()]
param(
    [string]$Destination = 'H:\一键部署\AI-Game-Manager-Panel'
)

$ErrorActionPreference = 'Stop'
$Source = Split-Path -Parent $PSCommandPath
$Destination = [System.IO.Path]::GetFullPath($Destination)

function Write-Step([string]$Text) { Write-Host ('[进行] ' + $Text) -ForegroundColor Cyan }
function Write-Ok([string]$Text) { Write-Host ('[完成] ' + $Text) -ForegroundColor Green }
function Write-Warn2([string]$Text) { Write-Host ('[注意] ' + $Text) -ForegroundColor Yellow }
function Stop-Fail([string]$Text) { Write-Host ('[失败] ' + $Text) -ForegroundColor Red; exit 1 }

try {
    $utf8 = New-Object System.Text.UTF8Encoding($false)
    [Console]::InputEncoding = $utf8
    [Console]::OutputEncoding = $utf8
    $OutputEncoding = $utf8
} catch {}

Write-Host '====================================================================' -ForegroundColor DarkGray
Write-Host '  AGMP 源码同步助手' -ForegroundColor White
Write-Host ('  来源：' + $Source)
Write-Host ('  目标：' + $Destination)
Write-Host '====================================================================' -ForegroundColor DarkGray

if ($Source.TrimEnd('\\') -ieq $Destination.TrimEnd('\\')) {
    Write-Warn2 '当前脚本已经位于 GitHub 工作副本目标目录，无需再次同步。'
    Write-Host ''
    Write-Host '如果你是要推送当前项目，请运行 AGMP-GitHub.bat。' -ForegroundColor Cyan
    Write-Host 'AGMP-Sync.bat 应从“新版本完整解压目录”运行，用来同步到 H:\一键部署\AI-Game-Manager-Panel。' -ForegroundColor DarkGray
    exit 0
}

$required = @(
    'go.mod',
    'frontend\package.json',
    'internal\xiaoyu\host\loop.go',
    'rust\Cargo.toml',
    'scripts\common\check-source-tree.mjs',
    'scripts\windows\AIGameManagerPanel.ps1'
)
foreach ($rel in $required) {
    if (-not (Test-Path -LiteralPath (Join-Path $Source $rel) -PathType Leaf)) {
        Stop-Fail ('源码包不完整，缺少：' + $rel)
    }
}

if (-not (Test-Path -LiteralPath $Destination -PathType Container)) {
    Write-Step '创建目标目录'
    New-Item -ItemType Directory -Path $Destination -Force | Out-Null
}

if (-not (Get-Command robocopy.exe -ErrorAction SilentlyContinue)) {
    Stop-Fail '未找到 robocopy.exe。Windows 10/11 正常应自带该工具。'
}

Write-Step '同步源码文件（保留目标 .git 与本机运行数据）'
$excludeDirs = @(
    (Join-Path $Source '.git'),
    (Join-Path $Source 'build'),
    (Join-Path $Source 'node_modules')
)
$args = @(
    $Source,
    $Destination,
    '/E',
    '/COPY:DAT',
    '/DCOPY:DAT',
    '/R:2',
    '/W:1',
    '/XJ',
    '/NFL',
    '/NDL',
    '/NP',
    '/XD'
) + $excludeDirs
& robocopy.exe @args
$code = $LASTEXITCODE
if ($code -ge 8) {
    Stop-Fail ('robocopy 同步失败，退出代码：' + $code)
}
Write-Ok '源码复制完成。'

# 目标是 Git 工作副本时，只删除“Git 已跟踪但新版源码已不存在”的旧源码。
# 未跟踪的 runtime / instances / backups / logs 等本机数据不会被删除。
if ((Test-Path -LiteralPath (Join-Path $Destination '.git')) -and (Get-Command git.exe -ErrorAction SilentlyContinue)) {
    Write-Step '清理新版已移除的旧 Git 跟踪文件'
    $tracked = @(& git.exe -C $Destination ls-files)
    foreach ($rel in $tracked) {
        if ([string]::IsNullOrWhiteSpace($rel)) { continue }
        $sourceFile = Join-Path $Source ($rel -replace '/', '\\')
        $targetFile = Join-Path $Destination ($rel -replace '/', '\\')
        if (-not (Test-Path -LiteralPath $sourceFile) -and (Test-Path -LiteralPath $targetFile -PathType Leaf)) {
            Remove-Item -LiteralPath $targetFile -Force
            Write-Host ('  - 移除旧源码：' + $rel) -ForegroundColor DarkGray
        }
    }
}

Write-Step '验证目标关键源码'
foreach ($rel in $required) {
    if (-not (Test-Path -LiteralPath (Join-Path $Destination $rel) -PathType Leaf)) {
        Stop-Fail ('同步后仍缺少关键文件：' + $rel)
    }
}
Write-Ok '目标源码完整。'
Write-Host ''
Write-Host '下一步：在目标目录运行 AGMP-GitHub.bat -> 1 一键推送。' -ForegroundColor Cyan
