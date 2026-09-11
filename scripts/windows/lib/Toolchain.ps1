function Get-AGMPTools {
    param([switch]$RequireWsl)
    if ($script:AGMPDryRun) { return @{ Go='go'; Node='node'; Pnpm='pnpm'; Wsl='wsl' } }
    $go = Get-AGMPToolPath @('go.exe','go')
    $node = Get-AGMPToolPath @('node.exe','node')
    $pnpm = Get-AGMPToolPath @('pnpm.cmd','pnpm.exe','pnpm')
    if (-not $go) { throw '未检测到 Go。' }
    if (-not $node) { throw '未检测到 Node.js。' }
    if (-not $pnpm) { throw '未检测到 pnpm。请先安装 pnpm。' }
    $wsl = Get-AGMPToolPath @('wsl.exe','wsl')
    if ($RequireWsl -and -not $wsl) { throw '未检测到 WSL。Linux Server Release 需要 WSL 或 Linux 环境。' }
    return @{ Go=$go; Node=$node; Pnpm=$pnpm; Wsl=$wsl }
}

function Show-AGMPToolchain {
    $tools = Get-AGMPTools
    if ($script:AGMPDryRun) {
        Write-Info '环境' 'Go：DRY-RUN'
        Write-Info '环境' 'Node.js：DRY-RUN'
        Write-Info '环境' 'pnpm：DRY-RUN'
        return
    }
    $goVer = (& $tools.Go version 2>&1 | Out-String).Trim()
    $nodeVer = (& $tools.Node --version 2>&1 | Out-String).Trim()
    $pnpmVer = (& $tools.Pnpm --version 2>&1 | Out-String).Trim()
    Write-Info '环境' "Go：$goVer"
    Write-Info '环境' "Node.js：$nodeVer"
    Write-Info '环境' "pnpm：$pnpmVer"
    return
}

function Invoke-GoModules {
    $tools = Get-AGMPTools
    $code = 0
    Invoke-AGMPNative -FilePath $tools.Go -Arguments @('mod','tidy') -AllowFailure -ExitCode ([ref]$code)
    if ($code -eq 0) { return }
    Write-Warn '当前 Go Proxy 执行 go mod tidy 失败，仅对本次 AGMP 进程回退 goproxy.cn。'
    $old = $env:GOPROXY
    try {
        $env:GOPROXY = 'https://goproxy.cn,direct'
        Invoke-AGMPNative -FilePath $tools.Go -Arguments @('mod','tidy')
    } finally {
        $env:GOPROXY = $old
    }
}

function Get-AGMPWailsCliPath {
    $found = Get-AGMPToolPath @('wails.exe','wails')
    if ($found) { return $found }
    $wailsInstallDir = Join-Path (Get-LocalAppDataRoot) 'AI-Game-Manager-Panel\DevTools\Wails\v2.15.0'
    $exe = Join-Path $wailsInstallDir 'wails.exe'
    if (Test-Path -LiteralPath $exe -PathType Leaf) { return $exe }
    return $null
}

function Install-AGMPWailsCli {
    $found = Get-AGMPWailsCliPath
    if ($found) { Write-Ok "Wails CLI 已就绪：$found"; return $found }
    $wailsInstallDir = Join-Path (Get-LocalAppDataRoot) 'AI-Game-Manager-Panel\DevTools\Wails\v2.15.0'
    $exe = Join-Path $wailsInstallDir 'wails.exe'
    if ($script:AGMPDryRun) { Write-Info 'DRY-RUN' "准备 Wails v2.15.0：$exe"; return $exe }
    Ensure-Directory $wailsInstallDir
    Write-Info 'Wails' "首次准备 Wails v2.15.0：$wailsInstallDir"
    $tools = Get-AGMPTools
    $oldGobIn = $env:GOBIN
    $oldProxy = $env:GOPROXY
    try {
        $env:GOBIN = $wailsInstallDir
        $code = 0
        Invoke-AGMPNative -FilePath $tools.Go -Arguments @('install','github.com/wailsapp/wails/v2/cmd/wails@v2.15.0') -AllowFailure -ExitCode ([ref]$code)
        if ($code -ne 0) {
            Write-Warn 'Wails 下载失败，回退 goproxy.cn 重试一次。'
            $env:GOPROXY = 'https://goproxy.cn,direct'
            Invoke-AGMPNative -FilePath $tools.Go -Arguments @('install','github.com/wailsapp/wails/v2/cmd/wails@v2.15.0')
        }
    } finally {
        $env:GOBIN = $oldGobIn
        $env:GOPROXY = $oldProxy
    }
    if (-not (Test-Path -LiteralPath $exe -PathType Leaf)) { throw "Wails CLI 安装完成后仍未生成：$exe" }
    Write-Ok "Wails CLI 已准备：$exe"
    return $exe
}

function Assert-AGMPWailsCli {
    if ($script:AGMPDryRun) { return 'wails.exe' }
    $found = Get-AGMPWailsCliPath
    if (-not $found) {
        throw 'Wails CLI 尚未准备。请先运行菜单 1“初始化/修复基础开发环境”；构建/开发菜单不会在中途偷偷安装开发工具。'
    }
    return $found
}

# 兼容旧调用名，但行为已经改为只检查，不再自动安装。
function Get-AGMPWailsCli {
    $value = Assert-AGMPWailsCli
    return $value
}

function Get-InnoCompiler {
    if ($script:AGMPDryRun) { return $null }
    $candidates = New-Object Collections.Generic.List[string]
    if ($env:AGMP_ISCC) { $candidates.Add($env:AGMP_ISCC) }
    $cmd = Get-Command 'ISCC.exe' -ErrorAction SilentlyContinue
    if ($cmd -and $cmd.Source) { $candidates.Add($cmd.Source) }
    foreach ($candidate in @(
        (Join-Path ${env:ProgramFiles(x86)} 'Inno Setup 6\ISCC.exe'),
        (Join-Path $env:ProgramFiles 'Inno Setup 6\ISCC.exe'),
        (Join-Path $env:LOCALAPPDATA 'Programs\Inno Setup 6\ISCC.exe')
    )) { if ($candidate) { $candidates.Add($candidate) } }
    foreach ($candidate in $candidates) {
        if ($candidate -and (Test-Path -LiteralPath $candidate -PathType Leaf)) { return (Resolve-Path -LiteralPath $candidate).Path }
    }
    return $null
}

function Ensure-InnoCompiler {
    $iscc = Get-InnoCompiler
    if ($iscc) { return $iscc }
    if ($script:AGMPDryRun) { return 'C:\Program Files (x86)\Inno Setup 6\ISCC.exe' }
    Write-Warn '未检测到 Inno Setup 6。'
    $winget = Get-AGMPToolPath @('winget.exe','winget')
    if (-not $winget) { throw '缺少 Inno Setup 6，且未检测到 winget。请安装 Inno Setup 6 后重试。' }
    $answer = Read-Host '是否使用 winget 安装 Inno Setup 6？[Y/N]'
    if ($answer -notmatch '^(?i:y|yes)$') { throw '已取消 Inno Setup 安装。' }
    Invoke-AGMPNative -FilePath $winget -Arguments @('install','--id','JRSoftware.InnoSetup','-e','--silent','--accept-package-agreements','--accept-source-agreements','--disable-interactivity')
    Start-Sleep -Milliseconds 300
    $iscc = Get-InnoCompiler
    if (-not $iscc) { throw 'Inno Setup 安装结束后仍未找到 ISCC.exe。' }
    return $iscc
}


function Assert-InnoCompiler {
    if ($script:AGMPDryRun) { return 'C:\Program Files (x86)\Inno Setup 6\ISCC.exe' }
    $iscc = Get-InnoCompiler
    if (-not $iscc) {
        throw 'Inno Setup 6 尚未准备。请在菜单 8“状态、诊断与修复”中选择“准备 Inno Setup”，构建菜单不会中途自动安装。'
    }
    return $iscc
}
