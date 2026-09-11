param([string]$KeyRoot = '')
$ErrorActionPreference = 'Stop'
$Root = (Resolve-Path (Join-Path $PSScriptRoot '..\..\..')).Path

function Get-DefaultKeyRoot {
    if (-not [string]::IsNullOrWhiteSpace($env:AGMP_RELEASE_KEY_DIR)) { return $env:AGMP_RELEASE_KEY_DIR }
    if (-not [string]::IsNullOrWhiteSpace($env:LOCALAPPDATA)) { return (Join-Path $env:LOCALAPPDATA 'AI-Game-Manager-Panel\ReleaseKeys') }
    return (Join-Path $HOME '.agmp\release-keys')
}

if ([string]::IsNullOrWhiteSpace($KeyRoot)) { $KeyRoot = Get-DefaultKeyRoot }
if (-not (Test-Path -LiteralPath $KeyRoot)) { throw "未找到本机发行密钥目录：$KeyRoot" }
$go = Get-Command go -ErrorAction SilentlyContinue
if (-not $go) { throw '未找到 Go。请先运行项目初始化。' }
$goExe = $go.Source
$ringPath = Join-Path $Root 'internal\system\license\vendor_public_keys.json'

Write-Host '====================================================================' -ForegroundColor Cyan
Write-Host '          AI Game Manager Panel - 同步本机发行公钥' -ForegroundColor Cyan
Write-Host '====================================================================' -ForegroundColor Cyan
Write-Host "本机发行密钥目录：$KeyRoot"
Write-Host "目标公开密钥环：$ringPath"
Write-Host '只读取 key-metadata.json / 公钥信息；不会把发行私钥复制进源码。' -ForegroundColor Yellow
Write-Host ''

Push-Location $Root
try {
    & $goExe run ./cmd/aigame-manager-license-admin sync-public-key --ring $ringPath --key-root $KeyRoot
    if ($LASTEXITCODE -ne 0) { throw "同步发行公钥失败，退出码 $LASTEXITCODE" }
    $node = Get-Command node -ErrorAction SilentlyContinue
    if ($node) {
        & $node.Source scripts/common/check-release-key.mjs
        if ($LASTEXITCODE -ne 0) { throw 'Release Key Gate 未通过。' }
        & $node.Source scripts/common/check-github-safety.mjs
        if ($LASTEXITCODE -ne 0) { throw 'GitHub Safety Gate 未通过。' }
    }
    Write-Host ''
    Write-Host '[完成] 本机发行公钥已同步到当前源码。私钥仍只存在仓库外。' -ForegroundColor Green
} finally { Pop-Location }
