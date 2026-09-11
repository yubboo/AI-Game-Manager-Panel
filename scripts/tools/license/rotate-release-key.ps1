param(
    [string]$KeyRoot = ''
)
$ErrorActionPreference = 'Stop'
$Root = (Resolve-Path (Join-Path $PSScriptRoot '..\..\..')).Path

function Get-DefaultKeyRoot {
    if (-not [string]::IsNullOrWhiteSpace($env:AGMP_RELEASE_KEY_DIR)) { return $env:AGMP_RELEASE_KEY_DIR }
    if (-not [string]::IsNullOrWhiteSpace($env:LOCALAPPDATA)) { return (Join-Path $env:LOCALAPPDATA 'AI-Game-Manager-Panel\ReleaseKeys') }
    return (Join-Path $HOME '.agmp\release-keys')
}

function Get-FullPath([string]$PathValue) {
    return [System.IO.Path]::GetFullPath($PathValue)
}

function Test-Inside([string]$Parent, [string]$Child) {
    $separators = [char[]]@([System.IO.Path]::DirectorySeparatorChar, [System.IO.Path]::AltDirectorySeparatorChar)
    $parentFull = (Get-FullPath $Parent).TrimEnd($separators) + [System.IO.Path]::DirectorySeparatorChar
    $childFull = (Get-FullPath $Child).TrimEnd($separators) + [System.IO.Path]::DirectorySeparatorChar
    return $childFull.StartsWith($parentFull, [System.StringComparison]::OrdinalIgnoreCase)
}

function Write-Utf8NoBom([string]$PathValue, [string]$Text) {
    $enc = New-Object System.Text.UTF8Encoding($false)
    [System.IO.File]::WriteAllText($PathValue, $Text, $enc)
}

if ([string]::IsNullOrWhiteSpace($KeyRoot)) { $KeyRoot = Get-DefaultKeyRoot }
$KeyRoot = Get-FullPath $KeyRoot
if (Test-Inside $Root $KeyRoot) {
    throw "发行密钥目录不能位于源码仓库内部：$KeyRoot"
}

$go = Get-Command go -ErrorAction SilentlyContinue
if (-not $go) { throw '未找到 Go。请先运行 AI-Game-Manager-Panel.bat -> 1 初始化开发环境。' }
$goExe = $go.Source

# 0.1.70：防止解压新源码后因为 vendor_public_keys.json 尚未同步，误生成第二把发行私钥。
$ringPath = Join-Path $Root 'internal\service\license\vendor_public_keys.json'
$currentActive = ''
if (Test-Path -LiteralPath $ringPath) {
    try {
        $existingRing = Get-Content -LiteralPath $ringPath -Raw -Encoding UTF8 | ConvertFrom-Json
        $currentActive = [string]$existingRing.activeKeyId
    } catch { }
}
$existingMetadata = @()
if (Test-Path -LiteralPath $KeyRoot) {
    $existingMetadata = @(Get-ChildItem -LiteralPath $KeyRoot -Directory -ErrorAction SilentlyContinue | Where-Object { Test-Path -LiteralPath (Join-Path $_.FullName 'key-metadata.json') })
}
if ($existingMetadata.Count -gt 0 -and [string]::IsNullOrWhiteSpace($currentActive)) {
    Write-Host '[保护] 检测到本机已经存在发行密钥，但当前新源码尚未同步 Active 公钥。' -ForegroundColor Yellow
    Write-Host '请返回发行控制台选择 4“同步本机发行公钥”，不要重新生成第二把密钥。' -ForegroundColor Yellow
    throw '已阻止误生成新发行密钥。先同步现有公钥；只有明确轮换时再执行初始化/轮换。'
}
if (-not [string]::IsNullOrWhiteSpace($currentActive)) {
    Write-Host "当前 Active KeyID：$currentActive" -ForegroundColor Yellow
    $confirm = Read-Host '这是密钥轮换操作。若确实要生成全新发行密钥请输入 ROTATE，否则直接回车取消'
    if ($confirm -cne 'ROTATE') { throw '已取消发行密钥轮换。' }
}

$stamp = Get-Date -Format 'yyyyMMdd-HHmmss'
$random = [Guid]::NewGuid().ToString('N').Substring(0,8)
$keyDir = Join-Path $KeyRoot ("$stamp-$random")
New-Item -ItemType Directory -Path $keyDir -Force | Out-Null

Write-Host '====================================================================' -ForegroundColor Cyan
Write-Host '          AI Game Manager Panel - 发行密钥初始化 / 轮换' -ForegroundColor Cyan
Write-Host '====================================================================' -ForegroundColor Cyan
Write-Host "源码目录：$Root" -ForegroundColor DarkGray
Write-Host "私钥目录：$keyDir" -ForegroundColor Yellow
Write-Host '私钥只会生成到源码仓库外；源码中只更新公开密钥环。' -ForegroundColor DarkYellow
Write-Host ''

Push-Location $Root
try {
    & $goExe run ./cmd/aigame-manager-license-admin keygen --output-dir $keyDir
    if ($LASTEXITCODE -ne 0) { throw "发行密钥生成失败，退出码 $LASTEXITCODE" }
} finally {
    Pop-Location
}

$metadataPath = Join-Path $keyDir 'key-metadata.json'
$privatePath = Join-Path $keyDir 'agmp-release-private.key'
if (-not (Test-Path -LiteralPath $metadataPath)) { throw "缺少密钥元数据：$metadataPath" }
if (-not (Test-Path -LiteralPath $privatePath)) { throw "缺少发行私钥：$privatePath" }
$meta = Get-Content -LiteralPath $metadataPath -Raw -Encoding UTF8 | ConvertFrom-Json

# Windows 上进一步收紧私钥 ACL；失败时仅警告，Go 仍使用 0600 语义写入。
$icacls = Get-Command icacls.exe -ErrorAction SilentlyContinue
if ($icacls) {
    try {
        $identity = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name
        $icaclsExe = $icacls.Source
        & $icaclsExe $privatePath '/inheritance:r' "/grant:r" "${identity}:(F)" | Out-Null
        if ($LASTEXITCODE -ne 0) { Write-Warning '无法自动收紧私钥 ACL，请手动确认该文件仅当前 Windows 账号可访问。' }
    } catch {
        Write-Warning "无法自动收紧私钥 ACL：$($_.Exception.Message)"
    }
}

if (-not (Test-Path -LiteralPath $ringPath)) { throw "缺少发行公钥环：$ringPath" }
$ring = Get-Content -LiteralPath $ringPath -Raw -Encoding UTF8 | ConvertFrom-Json
$keys = New-Object System.Collections.Generic.List[object]
foreach ($entry in @($ring.keys)) {
    $status = [string]$entry.status
    if ($status -ieq 'active') { $status = 'retired' }
    $keys.Add([ordered]@{
        keyId = [string]$entry.keyId
        fingerprint = [string]$entry.fingerprint
        publicKey = [string]$entry.publicKey
        status = $status
        createdAt = [long]$entry.createdAt
        note = [string]$entry.note
    })
}
$keys.Add([ordered]@{
    keyId = [string]$meta.keyId
    fingerprint = [string]$meta.fingerprint
    publicKey = [string]$meta.publicKey
    status = 'active'
    createdAt = [long]$meta.createdAt
    note = '由项目所有者本机通过 AGMP 0.1.70+ 发行密钥轮换工具生成。私钥保存在源码仓库外。'
})
$newRing = [ordered]@{
    schemaVersion = 1
    activeKeyId = [string]$meta.keyId
    keys = $keys
}
$json = $newRing | ConvertTo-Json -Depth 8
Write-Utf8NoBom $ringPath ($json + [Environment]::NewLine)

Push-Location $Root
try {
    $node = Get-Command node -ErrorAction SilentlyContinue
    if ($node) {
        $nodeExe = $node.Source
        & $nodeExe scripts/common/check-release-key.mjs
        if ($LASTEXITCODE -ne 0) { throw "发行密钥门禁失败，退出码 $LASTEXITCODE" }
        & $nodeExe scripts/common/check-github-safety.mjs
        if ($LASTEXITCODE -ne 0) { throw "GitHub Safety Gate 失败，退出码 $LASTEXITCODE" }
    } else {
        Write-Warning '未找到 Node.js，暂未自动执行发行密钥/GitHub Safety Gate。'
    }
} finally {
    Pop-Location
}

Write-Host ''
Write-Host '[完成] 新发行密钥已启用。' -ForegroundColor Green
Write-Host "KeyID：$($meta.keyId)" -ForegroundColor Green
Write-Host "指纹：$($meta.fingerprint)" -ForegroundColor Green
Write-Host "私钥：$privatePath" -ForegroundColor Yellow
Write-Host "公开密钥环：$ringPath" -ForegroundColor Gray
Write-Host ''
Write-Host '接下来：' -ForegroundColor Cyan
Write-Host '  1. 备份整个私钥目录到你自己的离线安全介质。' -ForegroundColor Gray
Write-Host '  2. GitHub 只提交 vendor_public_keys.json；绝对不要提交私钥目录。' -ForegroundColor Gray
Write-Host '  3. 使用 AGMP-License-Admin.bat -> 3 签发 BFLC2。' -ForegroundColor Gray
Write-Host '  4. 正式 Release 会强制检查 active 发行密钥，未配置会拒绝构建。' -ForegroundColor Gray
