param([string]$CertificateFile = '')
$ErrorActionPreference = 'Stop'
$Root = (Resolve-Path (Join-Path $PSScriptRoot '..\..\..')).Path
$ringPath = Join-Path $Root 'internal\service\license\vendor_public_keys.json'
$go = Get-Command go -ErrorAction SilentlyContinue
if (-not $go) { throw '未找到 Go。请先运行项目初始化。' }
$goExe = $go.Source

Write-Host '====================================================================' -ForegroundColor Cyan
Write-Host '          AI Game Manager Panel - BFLC2 离线验证' -ForegroundColor Cyan
Write-Host '====================================================================' -ForegroundColor Cyan
$Machine = Read-Host '目标机器码（BFM-...）'
$InstallID = Read-Host '目标安装 ID（BFID-...）'
if ([string]::IsNullOrWhiteSpace($CertificateFile)) { $CertificateFile = Read-Host 'BFLC2 文件路径（可拖入 .bflc 文件）' }
$CertificateFile = $CertificateFile.Trim('"')
if (-not (Test-Path -LiteralPath $CertificateFile)) { throw "BFLC2 文件不存在：$CertificateFile" }

Push-Location $Root
try {
    & $goExe run ./cmd/aigame-manager-license-admin verify --ring $ringPath --certificate-file $CertificateFile --machine $Machine --install-id $InstallID
    if ($LASTEXITCODE -ne 0) { throw "BFLC2 验证失败，退出码 $LASTEXITCODE" }
    Write-Host ''
    Write-Host '[PASS] 证书签名、机器码、安装 ID 与有效期均验证通过。' -ForegroundColor Green
} finally { Pop-Location }
