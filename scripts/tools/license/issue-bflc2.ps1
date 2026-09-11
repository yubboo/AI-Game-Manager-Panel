param(
    [string]$PrivateKey = '',
    [string]$KeyRoot = ''
)
$ErrorActionPreference = 'Stop'
$Root = (Resolve-Path (Join-Path $PSScriptRoot '..\..\..')).Path

function Get-DefaultKeyRoot {
    if (-not [string]::IsNullOrWhiteSpace($env:AGMP_RELEASE_KEY_DIR)) { return $env:AGMP_RELEASE_KEY_DIR }
    if (-not [string]::IsNullOrWhiteSpace($env:LOCALAPPDATA)) { return (Join-Path $env:LOCALAPPDATA 'AI-Game-Manager-Panel\ReleaseKeys') }
    return (Join-Path $HOME '.agmp\release-keys')
}

function Find-PrivateKeyForActiveKey([string]$RootDir, [string]$ActiveKeyID) {
    if (-not (Test-Path -LiteralPath $RootDir)) { return $null }
    foreach ($dir in Get-ChildItem -LiteralPath $RootDir -Directory -ErrorAction SilentlyContinue | Sort-Object LastWriteTime -Descending) {
        $metadata = Join-Path $dir.FullName 'key-metadata.json'
        if (-not (Test-Path -LiteralPath $metadata)) { continue }
        try {
            $meta = Get-Content -LiteralPath $metadata -Raw -Encoding UTF8 | ConvertFrom-Json
            if ([string]$meta.keyId -ieq $ActiveKeyID) {
                $candidate = Join-Path $dir.FullName 'agmp-release-private.key'
                if (Test-Path -LiteralPath $candidate) { return $candidate }
            }
        } catch { }
    }
    return $null
}

$ringPath = Join-Path $Root 'internal\service\license\vendor_public_keys.json'
$ring = Get-Content -LiteralPath $ringPath -Raw -Encoding UTF8 | ConvertFrom-Json
$activeKeyID = [string]$ring.activeKeyId
if ([string]::IsNullOrWhiteSpace($activeKeyID)) {
    throw '当前源码尚未配置 active 发行密钥。若本机已经生成过发行密钥，请先运行 AGMP-License-Admin.bat -> 4 同步本机发行公钥；只有首次初始化或明确轮换时才使用菜单 1。'
}
if ([string]::IsNullOrWhiteSpace($KeyRoot)) { $KeyRoot = Get-DefaultKeyRoot }
if ([string]::IsNullOrWhiteSpace($PrivateKey)) { $PrivateKey = Find-PrivateKeyForActiveKey $KeyRoot $activeKeyID }
if ([string]::IsNullOrWhiteSpace($PrivateKey)) { $PrivateKey = Read-Host "未自动找到 $activeKeyID 的私钥，请输入私钥文件路径" }
if (-not (Test-Path -LiteralPath $PrivateKey)) { throw "私钥文件不存在：$PrivateKey" }

$go = Get-Command go -ErrorAction SilentlyContinue
if (-not $go) { throw '未找到 Go。请先运行项目初始化。' }
$goExe = $go.Source

Push-Location $Root
try {
    $keyInfoRaw = (& $goExe run ./cmd/aigame-manager-license-admin key-info --private-key $PrivateKey | Out-String).Trim()
    if ($LASTEXITCODE -ne 0) { throw '无法读取发行私钥信息。' }
    $keyInfo = $keyInfoRaw | ConvertFrom-Json
    if ([string]$keyInfo.keyId -ine $activeKeyID) {
        throw "私钥与当前 active 公钥不匹配。当前=$activeKeyID，私钥=$($keyInfo.keyId)"
    }

    Write-Host '====================================================================' -ForegroundColor Cyan
    Write-Host '          AI Game Manager Panel - BFLC2 离线许可证签发' -ForegroundColor Cyan
    Write-Host '====================================================================' -ForegroundColor Cyan
    Write-Host "发行 KeyID：$activeKeyID" -ForegroundColor Green
    Write-Host "发行指纹：$($keyInfo.fingerprint)" -ForegroundColor DarkGray
    Write-Host ''

    Write-Host '注意：BFID 绑定具体安装实例。请从真正准备激活的 Release 程序中复制 BFM/BFID，不要拿另一份源码开发目录的 BFID。' -ForegroundColor Yellow
    $Machine = Read-Host '用户机器码（BFM-...）'
    $InstallID = Read-Host '用户安装 ID（BFID-...）'
    if ([string]::IsNullOrWhiteSpace($Machine) -or [string]::IsNullOrWhiteSpace($InstallID)) { throw '机器码和安装 ID 不能为空。' }

    $License = Read-Host 'LicenseID（留空自动生成）'
    $Edition = Read-Host '授权版本 standard / pro（默认 standard）'
    if ([string]::IsNullOrWhiteSpace($Edition)) { $Edition = 'standard' }
    if ($Edition -notin @('standard','pro')) { throw '授权版本目前只支持 standard 或 pro。' }
    $Features = Read-Host '功能权限（逗号分隔；留空按 Edition 默认）'
    $SeatsText = Read-Host '设备席位数（默认 1）'
    if ([string]::IsNullOrWhiteSpace($SeatsText)) { $SeatsText = '1' }
    $Seats = 1
    if (-not [int]::TryParse($SeatsText, [ref]$Seats) -or $Seats -lt 1) { throw '设备席位数必须是 >= 1 的整数。' }
    $DaysText = Read-Host '有效天数（0=永久，默认 0）'
    if ([string]::IsNullOrWhiteSpace($DaysText)) { $DaysText = '0' }
    $Days = 0
    if (-not [int]::TryParse($DaysText, [ref]$Days) -or $Days -lt 0) { throw '有效天数必须是 >= 0 的整数。' }
    $Expires = 0L
    if ($Days -gt 0) { $Expires = [DateTimeOffset]::UtcNow.AddDays($Days).ToUnixTimeSeconds() }

    $issuedRoot = Join-Path $KeyRoot 'IssuedLicenses'
    $issueArgs = @('run','./cmd/aigame-manager-license-admin','issue','--private-key',$PrivateKey,'--expected-key-id',$activeKeyID,'--machine',$Machine,'--install-id',$InstallID,'--edition',$Edition,'--seats',"$Seats",'--expires',"$Expires",'--out-dir',$issuedRoot)
    if (-not [string]::IsNullOrWhiteSpace($License)) { $issueArgs += @('--license',$License) }
    if (-not [string]::IsNullOrWhiteSpace($Features)) { $issueArgs += @('--features',$Features) }
    $issueOutput = (& $goExe @issueArgs | Out-String).Trim()
    if ($LASTEXITCODE -ne 0) { throw "许可证签发失败，退出码 $LASTEXITCODE" }
    Write-Host $issueOutput

    $certificateFile = ''
    foreach ($line in ($issueOutput -split "`r?`n")) {
        if ($line -match '^CertificateFile:\s*(.+)$') { $certificateFile = $Matches[1].Trim() }
    }
    if ([string]::IsNullOrWhiteSpace($certificateFile) -or -not (Test-Path -LiteralPath $certificateFile)) {
        throw '签发完成但无法定位生成的 BFLC2 文件，已停止交付。'
    }

    Write-Host ''
    Write-Host '[验证] 正在使用当前公开密钥环回验刚签发的 BFLC2...' -ForegroundColor Cyan
    & $goExe run ./cmd/aigame-manager-license-admin verify --ring $ringPath --certificate-file $certificateFile --machine $Machine --install-id $InstallID
    if ($LASTEXITCODE -ne 0) { throw 'BFLC2 已生成，但签发后回验失败；请勿交付该证书。' }

    Write-Host ''
    Write-Host "[完成] BFLC2 已保存并通过签发后回验：$certificateFile" -ForegroundColor Green
    Write-Host '下一步：在对应设备的 设置中心 -> 授权与许可证 中导入该 .bflc 文件或完整证书。' -ForegroundColor Yellow
    Write-Host '不要把实际 BFLC2 提交到公开 GitHub。' -ForegroundColor Yellow
} finally {
    Pop-Location
}
