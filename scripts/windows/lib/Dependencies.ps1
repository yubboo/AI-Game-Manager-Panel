function Get-AGMPPnpmStore {
    $path = Join-Path (Get-LocalAppDataRoot) 'AI-Game-Manager-Panel\DevTools\pnpm-store'
    Ensure-Directory $path
    return $path
}
function Get-AGMPElectronCache {
    $path = Join-Path (Get-LocalAppDataRoot) 'AI-Game-Manager-Panel\DevTools\electron-cache'
    Ensure-Directory $path
    return $path
}

function Reset-AGMPDependencyArea([string]$ProjectDir,[string]$StoreDir,[string]$Label) {
    $modules = Join-Path $ProjectDir 'node_modules'
    if (Test-Path -LiteralPath $modules) {
        Write-Warn "$Label node_modules 异常，正在删除后重建。"
        Remove-Item -LiteralPath $modules -Recurse -Force -ErrorAction Stop
    }
    if (Test-Path -LiteralPath $StoreDir) {
        $stamp = Get-Date -Format 'yyyyMMdd-HHmmss'
        $backup = "$StoreDir.corrupt-$stamp"
        Write-Warn "$Label 专用 pnpm Store 可能异常，隔离为：$backup"
        Move-Item -LiteralPath $StoreDir -Destination $backup -Force
    }
    Ensure-Directory $StoreDir
}

function Install-PnpmProject {
    param([string]$ProjectDir,[string]$Label)
    $tools = Get-AGMPTools
    $store = Get-AGMPPnpmStore
    Write-Info '缓存' "$Label pnpm Store：$store"
    Write-Info '依赖' "$Label：安装 / 同步中..."
    $args = @('install','--frozen-lockfile','--store-dir',$store)
    $code = 0
    Invoke-AGMPNative -FilePath $tools.Pnpm -Arguments $args -WorkingDirectory $ProjectDir -AllowFailure -ExitCode ([ref]$code)
    if ($code -eq 0) { Write-Ok "$Label 依赖已就绪。"; return }
    Write-Warn "$Label 首次安装失败，执行一次安全自愈：只重置 AGMP 当前项目 node_modules 与 AGMP 专用 Store。"
    Reset-AGMPDependencyArea -ProjectDir $ProjectDir -StoreDir $store -Label $Label
    Invoke-AGMPNative -FilePath $tools.Pnpm -Arguments $args -WorkingDirectory $ProjectDir
    Write-Ok "$Label 依赖自愈后已就绪。"
}

function Install-FrontendDependencies {
    Install-PnpmProject -ProjectDir (Join-Path $script:AGMPRoot 'frontend') -Label 'Frontend'
}

function Test-ElectronRuntime {
    $exe = Join-Path $script:AGMPRoot 'desktop\electron\node_modules\electron\dist\electron.exe'
    return (Test-Path -LiteralPath $exe -PathType Leaf)
}

function Install-ElectronRuntime {
    if (Test-ElectronRuntime) { Write-Ok 'Electron Runtime 已存在，跳过下载。'; return }
    if ($script:AGMPDryRun) { Write-Info 'DRY-RUN' '准备 Electron 44.3.0 Runtime'; return }
    $tools = Get-AGMPTools
    $project = Join-Path $script:AGMPRoot 'desktop\electron'
    $cache = Get-AGMPElectronCache
    $oldCache = $env:electron_config_cache
    $oldMirror = $env:ELECTRON_MIRROR
    try {
        $env:electron_config_cache = $cache
        if ($env:AGMP_ELECTRON_MIRROR) { $env:ELECTRON_MIRROR = $env:AGMP_ELECTRON_MIRROR } else { $env:ELECTRON_MIRROR = 'https://npmmirror.com/mirrors/electron/' }
        Write-Info 'Electron' 'Runtime 未安装，正在准备 Electron 44.3.0...'
        $code = 0
        Invoke-AGMPNative -FilePath $tools.Pnpm -Arguments @('exec','install-electron','--no') -WorkingDirectory $project -AllowFailure -ExitCode ([ref]$code)
        if ($code -ne 0) {
            Write-Warn 'Electron 镜像下载失败，回退官方 GitHub Release 再试一次。'
            Remove-Item Env:ELECTRON_MIRROR -ErrorAction SilentlyContinue
            Invoke-AGMPNative -FilePath $tools.Pnpm -Arguments @('exec','install-electron','--no') -WorkingDirectory $project
        }
    } finally {
        if ($null -eq $oldCache) { Remove-Item Env:electron_config_cache -ErrorAction SilentlyContinue } else { $env:electron_config_cache = $oldCache }
        if ($null -eq $oldMirror) { Remove-Item Env:ELECTRON_MIRROR -ErrorAction SilentlyContinue } else { $env:ELECTRON_MIRROR = $oldMirror }
    }
    if (-not (Test-ElectronRuntime)) { throw 'Electron Runtime 下载完成后仍未找到 electron.exe。' }
    Write-Ok 'Electron Runtime 已就绪。'
}


function Assert-ElectronPackagingRuntime {
    $project = Join-Path $script:AGMPRoot 'desktop\electron'
    $runtimeDir = Join-Path $project 'node_modules\electron\dist'
    $runtimeExe = Join-Path $runtimeDir 'electron.exe'
    $builderConfig = Join-Path $project 'electron-builder.yml'
    if ($script:AGMPDryRun) {
        Write-Info 'DRY-RUN' '验证 Electron local electronDist + electron-builder 配置'
        return
    }
    if (-not (Test-Path -LiteralPath $runtimeExe -PathType Leaf)) {
        throw "Electron 本地 Runtime 不完整：$runtimeExe"
    }
    if (-not (Test-Path -LiteralPath $builderConfig -PathType Leaf)) {
        throw "缺少 electron-builder 配置：$builderConfig"
    }
    $builderText = [IO.File]::ReadAllText($builderConfig)
    if ($builderText -notmatch '(?m)^electronDist:\s*node_modules/electron/dist\s*$') {
        throw 'electron-builder 必须复用已准备的 node_modules/electron/dist，禁止再次走 Electron distribution 下载链。'
    }
    Write-Info 'Electron' "Builder Runtime：$runtimeDir"
}

function Install-ElectronDependencies {
    $project = Join-Path $script:AGMPRoot 'desktop\electron'
    Install-PnpmProject -ProjectDir $project -Label 'Electron npm 包'
    Install-ElectronRuntime
    Assert-ElectronPackagingRuntime
}

function Invoke-FrontendTypeCheck {
    $tools = Get-AGMPTools
    Invoke-AGMPNative -FilePath $tools.Pnpm -Arguments @('run','typecheck') -WorkingDirectory (Join-Path $script:AGMPRoot 'frontend')
}
function Invoke-FrontendBuild {
    $tools = Get-AGMPTools
    Invoke-AGMPNative -FilePath $tools.Pnpm -Arguments @('run','build') -WorkingDirectory (Join-Path $script:AGMPRoot 'frontend')
    if (-not $script:AGMPDryRun -and -not (Test-Path -LiteralPath (Join-Path $script:AGMPRoot 'frontend\dist\index.html'))) { throw 'Frontend Build 未生成 dist/index.html。' }
}
function Invoke-ElectronTypeCheck {
    $tools = Get-AGMPTools
    Invoke-AGMPNative -FilePath $tools.Pnpm -Arguments @('run','typecheck') -WorkingDirectory (Join-Path $script:AGMPRoot 'desktop\electron')
}
function Invoke-ElectronBuild {
    $tools = Get-AGMPTools
    Invoke-AGMPNative -FilePath $tools.Pnpm -Arguments @('run','build') -WorkingDirectory (Join-Path $script:AGMPRoot 'desktop\electron')
}
