$ErrorActionPreference = 'Stop'
$Root = (Resolve-Path (Join-Path $PSScriptRoot '..\..\..')).Path

function Show-Header {
    Clear-Host
    Write-Host '====================================================================' -ForegroundColor Cyan
    Write-Host '          AI Game Manager Panel - 许可证发行控制台' -ForegroundColor Cyan
    Write-Host '====================================================================' -ForegroundColor Cyan
    Write-Host '  1. 初始化 / 轮换发行密钥   私钥仅保存在仓库外，本地生成' -ForegroundColor Yellow
    Write-Host '  2. 查看发行密钥状态        Active KeyID / 指纹 / 历史可信公钥'
    Write-Host '  3. 签发 BFLC2 离线许可证   绑定机器码 + 安装 ID'
    Write-Host '  4. 同步本机发行公钥        新源码目录恢复 Active Key' -ForegroundColor Green
    Write-Host '  5. 验证 BFLC2 许可证       签名 + 机器码 + 安装 ID + 有效期'
    Write-Host '  6. 许可证完整验收           Release Key + Standard/Pro + 异常拒绝' -ForegroundColor Green
    Write-Host '  0. 退出'
    Write-Host '--------------------------------------------------------------------' -ForegroundColor DarkGray
    Write-Host '  短 CDK 属于未来在线 License Server；当前离线授权使用 BFLC2。' -ForegroundColor DarkGray
    Write-Host '====================================================================' -ForegroundColor Cyan
}

function Show-KeyStatus {
    Push-Location $Root
    try {
        $node = Get-Command node -ErrorAction SilentlyContinue
        if ($node) { $nodeExe = $node.Source; & $nodeExe scripts/common/check-release-key.mjs --allow-unconfigured }
        $ringPath = Join-Path $Root 'internal\service\license\vendor_public_keys.json'
        $ring = Get-Content -LiteralPath $ringPath -Raw -Encoding UTF8 | ConvertFrom-Json
        Write-Host ''
        $activeDisplay = [string]$ring.activeKeyId
        if ([string]::IsNullOrWhiteSpace($activeDisplay)) { $activeDisplay = '未配置' }
        Write-Host ("Active KeyID：" + $activeDisplay) -ForegroundColor Yellow
        if ([string]::IsNullOrWhiteSpace([string]$ring.activeKeyId)) {
            Write-Host '提示：若你已经在旧源码版本生成过发行密钥，请使用菜单 4 同步本机发行公钥，不要重新生成。' -ForegroundColor DarkYellow
        }
        Write-Host ("可信公钥数量：" + @($ring.keys).Count)
        foreach ($entry in @($ring.keys)) {
            $tone = if ([string]$entry.status -ieq 'active') { 'Green' } elseif ([string]$entry.status -ieq 'legacy') { 'DarkYellow' } else { 'Gray' }
            Write-Host ("  [{0}] {1}" -f ([string]$entry.status).ToUpperInvariant(), [string]$entry.keyId) -ForegroundColor $tone
            Write-Host ("      {0}" -f [string]$entry.fingerprint) -ForegroundColor DarkGray
        }
    } finally { Pop-Location }
}

while ($true) {
    Show-Header
    $choice = Read-Host '请选择 [0-6]'
    try {
        switch ($choice) {
            '1' { & (Join-Path $PSScriptRoot 'rotate-release-key.ps1') }
            '2' { Show-KeyStatus }
            '3' { & (Join-Path $PSScriptRoot 'issue-bflc2.ps1') }
            '4' { & (Join-Path $PSScriptRoot 'sync-release-public-key.ps1') }
            '5' { & (Join-Path $PSScriptRoot 'verify-bflc2.ps1') }
            '6' { & (Join-Path $PSScriptRoot 'acceptance-test.ps1') }
            '0' { exit 0 }
            default { Write-Warning '请输入 0 到 6。' }
        }
    } catch {
        Write-Host ''
        Write-Host ("[失败] " + $_.Exception.Message) -ForegroundColor Red
    }
    Write-Host ''
    [void](Read-Host '按 Enter 返回许可证发行控制台')
}
