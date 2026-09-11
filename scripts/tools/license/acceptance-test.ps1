﻿$ErrorActionPreference = 'Stop'
$Root = (Resolve-Path (Join-Path $PSScriptRoot '..\..\..')).Path

function Run-Native([string]$Exe, [string[]]$Arguments) {
    & $Exe @Arguments
    if ($LASTEXITCODE -ne 0) { throw ("命令失败（exit {0}）：{1} {2}" -f $LASTEXITCODE, $Exe, ($Arguments -join ' ')) }
}

Write-Host '====================================================================' -ForegroundColor Cyan
Write-Host '     AI Game Manager Panel 0.2.0 - License Acceptance Gate' -ForegroundColor Cyan
Write-Host '====================================================================' -ForegroundColor Cyan
Push-Location $Root
try {
    $node = (Get-Command node -ErrorAction Stop).Source
    $go = (Get-Command go -ErrorAction Stop).Source

    Write-Host '[1/6] GitHub Safety Gate...' -ForegroundColor Yellow
    Run-Native $node @('scripts/common/check-github-safety.mjs')

    Write-Host '[2/6] Project Layout Gate...' -ForegroundColor Yellow
    Run-Native $node @('scripts/common/check-project-layout.mjs')

    Write-Host '[3/6] Active Release Key Gate...' -ForegroundColor Yellow
    $ringPath = Join-Path $Root 'internal\service\license\vendor_public_keys.json'
    $ring = Get-Content -LiteralPath $ringPath -Raw -Encoding UTF8 | ConvertFrom-Json
    if ([string]::IsNullOrWhiteSpace([string]$ring.activeKeyId)) {
        $keyRoot = Join-Path $env:LOCALAPPDATA 'AI-Game-Manager-Panel\ReleaseKeys'
        if (Test-Path -LiteralPath $keyRoot) {
            Write-Host '  当前源码未配置 Active Key，自动同步本机仓库外公开密钥...' -ForegroundColor DarkYellow
            & (Join-Path $PSScriptRoot 'sync-release-public-key.ps1')
            if ($LASTEXITCODE -ne 0) { throw '发行公钥自动同步失败。' }
        }
    }
    Run-Native $node @('scripts/common/check-release-key.mjs')

    Write-Host '[4/6] Release License Matrix...' -ForegroundColor Yellow
    Run-Native $go @('test','./internal/system/license','-run','Test(RequireFeatureBlocksUnlicensedAndAllowsLicensed|StandardAndProFeatureMatrix|ExpiredAndUnboundLicenseBlockFeatures|CertificateRejectsWrongMachineAndInstallID|CertificateRejectsTamperedSignature|ActivationPersistsAcrossServiceRestart|CertificateRejectsSpoofedIssuerKeyID)$','-count=1')

    Write-Host '[5/6] Development License Mode...' -ForegroundColor Yellow
    Run-Native $go @('test','-tags','agmp_dev_license','./internal/system/license','-count=1')

    Write-Host '[6/6] Windows License Cross Build...' -ForegroundColor Yellow
    $oldGOOS=$env:GOOS; $oldGOARCH=$env:GOARCH; $oldCGO=$env:CGO_ENABLED
    try {
        $env:GOOS='windows'; $env:GOARCH='amd64'; $env:CGO_ENABLED='0'
        Run-Native $go @('test','./internal/system/license','-run','^$')
    } finally {
        $env:GOOS=$oldGOOS; $env:GOARCH=$oldGOARCH; $env:CGO_ENABLED=$oldCGO
    }

    Write-Host ''
    Write-Host '[PASS] 许可证发行密钥、Standard/Pro Feature Gate、异常拒绝、开发模式与 Windows 交叉编译全部通过。' -ForegroundColor Green
    Write-Host '下一步实机：正式 Release -> 启动 -> 验证 Pro 页面 -> 完全退出 -> 重启 -> 授权仍有效。' -ForegroundColor Gray
} finally {
    Pop-Location
}
