function Invoke-WindowsHelperGate {
    $tools = Get-AGMPTools
    Invoke-AGMPNative -FilePath $tools.Node -Arguments @('scripts/common/check-windows-helper.mjs')
}
function Invoke-ProjectLayoutGate {
    $tools = Get-AGMPTools
    Invoke-AGMPNative -FilePath $tools.Node -Arguments @('scripts/common/check-project-layout.mjs')
    Invoke-AGMPNative -FilePath $tools.Node -Arguments @('scripts/common/check-product-architecture.mjs')
    Invoke-AGMPNative -FilePath $tools.Node -Arguments @('scripts/common/check-modules.mjs')
    Invoke-AGMPNative -FilePath $tools.Node -Arguments @('scripts/common/check-multi-client-ai.mjs')
    Invoke-AGMPNative -FilePath $tools.Node -Arguments @('scripts/common/check-headless-xiaoyu.mjs')
    Invoke-AGMPNative -FilePath $tools.Node -Arguments @('scripts/common/check-organization-security.mjs')
    Invoke-AGMPNative -FilePath $tools.Node -Arguments @('scripts/common/check-bridge-auth-boundary.mjs')
    Invoke-AGMPNative -FilePath $tools.Node -Arguments @('scripts/common/check-web-session-security.mjs')
    Invoke-AGMPNative -FilePath $tools.Node -Arguments @('scripts/common/check-environment-manager.mjs')
}
function Invoke-GitHubSafetyGate {
    $tools = Get-AGMPTools
    Invoke-AGMPNative -FilePath $tools.Node -Arguments @('scripts/common/check-github-safety.mjs')
}
function Invoke-UpdaterGate {
    $tools = Get-AGMPTools
    Invoke-AGMPNative -FilePath $tools.Node -Arguments @('scripts/common/check-updater.mjs')
}
function Invoke-RustAgentGate {
    $tools = Get-AGMPTools
    Invoke-AGMPNative -FilePath $tools.Node -Arguments @('scripts/common/check-xiaoyu-core.mjs')
    Invoke-AGMPNative -FilePath $tools.Node -Arguments @('scripts/common/check-xiaoyu-harness.mjs')
    Invoke-AGMPNative -FilePath $tools.Node -Arguments @('scripts/common/check-xiaoyu-model-center.mjs')
    Invoke-AGMPNative -FilePath $tools.Node -Arguments @('scripts/common/check-xiaoyu-intelligence.mjs')
    Invoke-AGMPNative -FilePath $tools.Node -Arguments @('scripts/common/check-xiaoyu-agent-runtime.mjs')
}
function Invoke-WindowsInstallerGate {
    $tools = Get-AGMPTools
    Invoke-AGMPNative -FilePath $tools.Node -Arguments @('scripts/common/check-windows-installer.mjs')
    Invoke-AGMPNative -FilePath $tools.Node -Arguments @('scripts/common/check-distribution-boundary.mjs')
}
function Invoke-ReleaseKeyGate {
    param([switch]$AllowUnconfigured)
    $tools = Get-AGMPTools
    $args = @('scripts/common/check-release-key.mjs')
    if ($AllowUnconfigured) { $args += '--allow-unconfigured' }
    Invoke-AGMPNative -FilePath $tools.Node -Arguments $args
}
function Test-AGMPActiveReleasePublicKey {
    $ringPath = Join-Path $script:AGMPRoot 'internal\system\license\vendor_public_keys.json'
    if (-not (Test-Path -LiteralPath $ringPath -PathType Leaf)) { return $false }
    try {
        $ring = Get-Content -LiteralPath $ringPath -Raw -Encoding UTF8 | ConvertFrom-Json
        return -not [string]::IsNullOrWhiteSpace([string]$ring.activeKeyId)
    } catch { return $false }
}

function Invoke-EnsureReleasePublicKey {
    if (Test-AGMPActiveReleasePublicKey) { return }
    $keyRoot = Join-Path $env:LOCALAPPDATA 'AI-Game-Manager-Panel\ReleaseKeys'
    if (-not (Test-Path -LiteralPath $keyRoot -PathType Container)) {
        throw "正式发布尚未准备发行密钥。未找到：$keyRoot。请先运行 scripts\tools\license\AGMP-License-Admin.bat 初始化/管理发行密钥；候选构建请使用菜单 4/5/6。"
    }
    $metadata = @(Get-ChildItem -LiteralPath $keyRoot -Directory -ErrorAction SilentlyContinue | Where-Object { Test-Path -LiteralPath (Join-Path $_.FullName 'key-metadata.json') })
    if ($metadata.Count -eq 0) {
        throw "正式发布尚未准备有效 key-metadata.json：$keyRoot。请先运行 scripts\tools\license\AGMP-License-Admin.bat；候选构建不需要正式发行密钥。"
    }
    Write-Step '[Release Key] 检测到仓库外发行公钥元数据，正在同步公开部分...'
    & (Join-Path $script:AGMPRoot 'scripts\tools\license\sync-release-public-key.ps1')
    if ($LASTEXITCODE -ne 0) { throw '自动同步本机发行公钥失败。私钥不会复制进源码；请检查 License Admin 状态。' }
    if (-not (Test-AGMPActiveReleasePublicKey)) { throw '发行公钥同步完成后仍没有 activeKeyId，拒绝正式发布。' }
}

function Invoke-SyncWebAssets {
    $tools = Get-AGMPTools
    Invoke-AGMPNative -FilePath $tools.Node -Arguments @('scripts/common/sync-web-assets.mjs')
}

function Invoke-QuickProjectCheck {
    Write-AGMPHeader "AI游戏管理器面板 $script:AGMPVersion - 快速项目检查"
    Write-Step '[1/10] GitHub 安全门禁...'
    Invoke-GitHubSafetyGate
    Write-Step '[2/10] Windows 脚本 / 编码 / 菜单门禁...'
    Invoke-WindowsHelperGate
    Write-Step '[3/10] 项目结构与版本门禁...'
    Invoke-ProjectLayoutGate
    Write-Step '[4/10] Windows 安装器专项门禁...'; Invoke-WindowsInstallerGate
    Write-Step '[5/10] Windows Updater 专项门禁...'; Invoke-UpdaterGate
    Write-Step '[6/10] 小鱼核心 / Harness 专项门禁...'; Invoke-RustAgentGate
    Write-Step '[7/10] 发行密钥状态（开发检查允许未初始化）...'
    Invoke-ReleaseKeyGate -AllowUnconfigured
    Write-Step '[8/10] Frontend TypeScript...'
    Install-FrontendDependencies
    Invoke-FrontendTypeCheck
    Write-Step '[9/10] Go Core 测试...'
    $tools = Get-AGMPTools
    Invoke-AGMPNative -FilePath $tools.Go -Arguments @('test','./internal/...')
    Write-Step '[10/10] Frontend 相对导入 / Windows Helper 再确认...'
    Invoke-WindowsHelperGate
    Write-Ok '快速项目检查通过（不阻断独立 Electron 开发）。'
}

function Invoke-CoreBuildGate {
    param([switch]$RequireReleaseKey)
    Write-Step '[Core 1/13] GitHub 安全门禁...'; Invoke-GitHubSafetyGate
    Write-Step '[Core 2/13] Windows Helper 门禁...'; Invoke-WindowsHelperGate
    Write-Step '[Core 3/13] 项目结构 / 版本门禁...'; Invoke-ProjectLayoutGate
    Write-Step '[Core 4/13] Windows 安装器专项门禁...'; Invoke-WindowsInstallerGate
    Write-Step '[Core 5/13] Windows Updater 专项门禁...'; Invoke-UpdaterGate
    Write-Step '[Core 6/13] 小鱼核心 / Harness 专项门禁...'; Invoke-RustAgentGate
    if ($RequireReleaseKey) {
        Write-Step '[Core 7/13] 正式发行密钥门禁...'
        Invoke-EnsureReleasePublicKey
        Invoke-ReleaseKeyGate
    } else {
        Write-Step '[Core 7/13] 发行密钥状态（候选构建允许未配置）...'
        Invoke-ReleaseKeyGate -AllowUnconfigured
    }
    Write-Step '[Core 8/13] Go Modules...'; Invoke-GoModules
    Write-Step '[Core 9/13] Frontend Production Build...'; Install-FrontendDependencies; Invoke-FrontendBuild
    Write-Step '[Core 10/13] 同步 Web Assets...'; Invoke-SyncWebAssets
    $tools = Get-AGMPTools
    Write-Step '[Core 11/13] go test ./...'; Invoke-AGMPNative -FilePath $tools.Go -Arguments @('test','./...')
    Write-Step '[Core 12/13] go vet ./...'; Invoke-AGMPNative -FilePath $tools.Go -Arguments @('vet','./...')
    Write-Step '[Core 13/13] Web 编译门禁...'
    $checkDir = Join-Path $script:AGMPRoot 'build\work\check'; Ensure-Directory $checkDir
    Invoke-AGMPNative -FilePath $tools.Go -Arguments @('build','-trimpath','-o',(Join-Path $checkDir 'AI-Game-Manager-Web-check.exe'),'.\cmd\aigame-manager-web')
}

function Invoke-CoreCandidateGate {
    Invoke-CoreBuildGate
}

function Invoke-CoreReleaseGate {
    Invoke-CoreBuildGate -RequireReleaseKey
}

function Invoke-FullProjectCheck {
    Write-AGMPHeader "AI游戏管理器面板 $script:AGMPVersion - 完整项目检查"
    Write-Info '说明' '完整检查验证源码是否可构建，但不要求正式发行密钥。正式发行密钥只在菜单 10 校验。'
    Invoke-CoreCandidateGate
    Write-Step '[Desktop 1/2] Electron npm / Runtime...'; Install-ElectronDependencies
    Write-Step '[Desktop 2/2] Electron TypeScript + Build...'; Invoke-ElectronTypeCheck; Invoke-ElectronBuild
    Write-Ok '完整项目检查通过。'
}
