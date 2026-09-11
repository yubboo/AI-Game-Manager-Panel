function Invoke-ElectronDevelopment {
    Write-AGMPHeader "AI游戏管理器面板 $script:AGMPVersion - Electron 开发"
    Show-AGMPToolchain
    Set-AGMPNetworkEnvironment
    Invoke-GoModules
    Install-FrontendDependencies
    Install-ElectronDependencies
    Invoke-FrontendBuild
    Invoke-SyncWebAssets
    Invoke-ElectronTypeCheck
    $tools = Get-AGMPTools
    $agent=Build-AGMPXiaoYuCore -Profile debug
    $oldAgent=$env:AGMP_XIAOYU_RUNTIME
    $env:AGMP_XIAOYU_RUNTIME=$agent
    Write-Info '小鱼核心' $agent
    Write-Info '提示' '保持本窗口开启；Ctrl+C 停止。'
    try { Invoke-AGMPNative -FilePath $tools.Pnpm -Arguments @('run','dev') -WorkingDirectory (Join-Path $script:AGMPRoot 'desktop\electron') } finally { $env:AGMP_XIAOYU_RUNTIME=$oldAgent }
}

function Invoke-ElectronRelease {
    param([switch]$OfficialPublish)
    $modeTitle = if($OfficialPublish){'Electron Windows 正式发布'}else{'Electron Windows 候选构建'}
    Write-AGMPHeader "AI游戏管理器面板 $script:AGMPVersion - $modeTitle"
    if ($script:AGMPDryRun) {
        Invoke-GitHubSafetyGate
        Invoke-WindowsHelperGate
        Invoke-ProjectLayoutGate
        Invoke-ReleaseKeyGate -AllowUnconfigured
        $dryKeyText = if($OfficialPublish){'正式发布时将严格要求 active Release Key'}else{'候选构建无需 active Release Key'}
        Write-Info 'DRY-RUN' $dryKeyText
        Write-Info 'DRY-RUN' 'Build AGMP internal Go/AI Core components + Electron complete product + NSIS/Portable'
        return
    }
    if($OfficialPublish){
        Write-Warn '正式发布：必须通过发行公钥 Gate；最终产物写入 build\release。'
    }else{
        Write-Info '候选构建' '用于本机/测试机验收，不要求正式发行公钥；产物写入 build\candidate。'
    }
    Write-Info '许可证' '生产构建不启用 DEVELOPMENT 许可证；未签发 BFLC2 时程序会正常显示未激活。'
    Write-Info '生产环境' '最终 Electron 安装包包含 AGMP Core + 小鱼核心；普通用户无需开发工具链。'
    Show-AGMPToolchain
    Set-AGMPNetworkEnvironment
    Write-Step '[预检] Rust / Cargo / MSVC Windows 链接工具链（只检查，不自动安装）...'
    $null=Assert-AGMPRustWindowsToolchain
    Show-AGMPRustToolchain
    Show-AGMPMSVCBuildTools
    if($OfficialPublish){ Invoke-CoreReleaseGate } else { Invoke-CoreCandidateGate }
    Install-ElectronDependencies
    Invoke-ElectronTypeCheck
    Invoke-ElectronBuild
    $tools = Get-AGMPTools

    $coreDir = Join-Path $script:AGMPRoot 'build\work\electron-core'; Ensure-Directory $coreDir
    $coreExe = Join-Path $coreDir 'AI-Game-Manager-Core.exe'
    $agentExe = Join-Path $coreDir 'AI-Game-Manager-XiaoYu.exe'
    $stage = Join-Path $script:AGMPRoot 'build\work\electron-builder'
    $channelRoot = if($OfficialPublish){'build\release\windows\electron'}else{'build\candidate\windows\electron'}
    $release = Join-Path $script:AGMPRoot $channelRoot
    if(Test-Path -LiteralPath $stage){Remove-Item -LiteralPath $stage -Recurse -Force}

    Write-Step '[1/5] 构建 AGMP 内部 Go Core...'
    Invoke-AGMPNative -FilePath $tools.Go -Arguments @('build','-trimpath','-ldflags','-s -w','-o',$coreExe,'.\cmd\aigame-manager-web')
    if(-not (Test-FileMinimumSize $coreExe)){throw 'AI-Game-Manager-Core.exe 无效。'}

    Write-Step '[2/5] 构建 小鱼核心（Rust Runtime）...'
    $null=Build-AGMPXiaoYuCore -Profile release -Destination $agentExe
    if(-not (Test-FileMinimumSize $agentExe)){throw 'AI-Game-Manager-XiaoYu.exe 无效。'}

    Write-Step '[3/5] Electron TypeScript / Runtime 最终确认...'
    Install-ElectronDependencies
    Invoke-ElectronTypeCheck

    Write-Step '[4/5] electron-builder：NSIS + Portable...'
    Assert-ElectronPackagingRuntime
    Write-Info 'Electron' '启用 maximum 压缩，并只保留 zh-CN / en-US Electron 语言包。'
    Invoke-AGMPNative -FilePath $tools.Pnpm -Arguments @('run','dist:win') -WorkingDirectory (Join-Path $script:AGMPRoot 'desktop\electron')

    $installerName = "AI-Game-Manager-Panel-$script:AGMPVersion-x64-Setup.exe"
    $portableName = "AI-Game-Manager-Panel-$script:AGMPVersion-x64-Portable.exe"
    $stageInstaller = Join-Path $stage $installerName
    $stagePortable = Join-Path $stage $portableName
    if(-not (Test-FileMinimumSize $stageInstaller) -or -not (Test-FileMinimumSize $stagePortable)){throw 'Electron work 产物不完整。'}

    Write-Step '[5/5] 发布 Electron 构建（文件锁安全模式）...'
    $publishedInstaller = Publish-AGMPArtifact -Source $stageInstaller -Destination (Join-Path $release ('installer\'+$installerName))
    $publishedPortable = Publish-AGMPArtifact -Source $stagePortable -Destination (Join-Path $release ('portable\'+$portableName))
    Write-Ok "Electron Installer：$publishedInstaller"
    Write-Ok "Electron Portable：$publishedPortable"
    $directoryText = if($OfficialPublish){'build\release = 正式发布产物'}else{'build\candidate = 本地候选验收产物；正式发布请使用菜单 10'}
    Write-Info '目录' $directoryText
    Write-Ok "$modeTitle 完成。"
}

