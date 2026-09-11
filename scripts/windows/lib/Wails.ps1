function Invoke-WailsDevelopment {
    Write-AGMPHeader "AI游戏管理器面板 $script:AGMPVersion - Wails 开发"
    Show-AGMPToolchain
    Set-AGMPNetworkEnvironment
    Invoke-GoModules
    Install-FrontendDependencies
    Invoke-FrontendTypeCheck
    $agent=Build-AGMPXiaoYuCore -Profile debug
    $oldAgent=$env:AGMP_XIAOYU_RUNTIME
    $env:AGMP_XIAOYU_RUNTIME=$agent
    $wails = Assert-AGMPWailsCli
    Write-Info '小鱼核心' $agent
    Write-Info 'Wails' "CLI：$wails"
    Write-Step '[预检] Wails Bridge Go 编译...'
    $tools = Get-AGMPTools
    Invoke-AGMPNative -FilePath $tools.Go -Arguments @('test','-tags','agmp_dev_license','./internal/bridge/wails')
    Write-Info '提示' '保持本窗口开启；Ctrl+C 停止。'
    try { Invoke-AGMPNative -FilePath $wails -Arguments @('dev','-tags','agmp_dev_license') } finally { $env:AGMP_XIAOYU_RUNTIME=$oldAgent }
}

function Build-WailsPortable([string]$DesktopExe,[string]$WebExe,[string]$AgentExe,[string]$DestinationZip) {
    $workRoot = Join-Path $script:AGMPRoot 'build\work\windows-wails\portable'
    $work = Join-Path $workRoot 'AI-Game-Manager-Panel'
    if (Test-Path -LiteralPath $workRoot) { Remove-Item -LiteralPath $workRoot -Recurse -Force }
    Ensure-Directory $work
    Copy-Item -LiteralPath $DesktopExe -Destination (Join-Path $work 'AI-Game-Manager-Panel.exe') -Force
    # Agent/Web 二进制属于 AGMP 内部实现，不放在产品根目录制造“多个独立程序”的用户心智。
    $internalAI = Join-Path $work 'internal\xiaoyu'
    $internalCore = Join-Path $work 'internal\core'
    Ensure-Directory $internalAI
    Ensure-Directory $internalCore
    Copy-Item -LiteralPath $AgentExe -Destination (Join-Path $internalAI 'AI-Game-Manager-XiaoYu.exe') -Force
    Copy-Item -LiteralPath $WebExe -Destination (Join-Path $internalCore 'AI-Game-Manager-Web.exe') -Force
    Copy-Item -LiteralPath (Join-Path $script:AGMPRoot 'README.md') -Destination $work -Force
    Copy-Item -LiteralPath (Join-Path $script:AGMPRoot 'configs') -Destination (Join-Path $work 'configs') -Recurse -Force
    Copy-Item -LiteralPath (Join-Path $script:AGMPRoot 'distribution\licenses') -Destination (Join-Path $work 'licenses') -Recurse -Force
    Ensure-Directory (Join-Path $work 'runtime')
    Ensure-Directory (Split-Path -Parent $DestinationZip)
    if (Test-Path -LiteralPath $DestinationZip) { Remove-Item -LiteralPath $DestinationZip -Force }
    Compress-Archive -Path (Join-Path $work '*') -DestinationPath $DestinationZip -CompressionLevel Optimal -Force
}

function Build-WailsInstaller([string]$Iscc,[string]$StageInstaller) {
    $iss = Join-Path $script:AGMPRoot 'distribution\installer\windows\AIGameManagerPanel.iss'
    $eula = Join-Path $script:AGMPRoot 'distribution\installer\windows\EULA-zh-CN.txt'
    if (-not (Test-Path -LiteralPath $eula -PathType Leaf)) { throw '缺少 Windows 安装许可协议：EULA-zh-CN.txt' }
    Write-Info '安装器' '简体中文界面由项目安装脚本内置，不再依赖额外 Inno 中文语言包。'
    Write-Info '协议' 'EULA 为强制接受页：用户不同意协议时不能继续安装。'
    Invoke-AGMPNative -FilePath $Iscc -Arguments @($iss)
    if (-not (Test-FileMinimumSize $StageInstaller)) { throw 'Inno Setup 返回后未生成有效安装器。' }
}

function Invoke-WailsRelease {
    param([switch]$OfficialPublish)
    $modeTitle = if($OfficialPublish){'Wails Windows 正式发布'}else{'Wails Windows 候选构建'}
    Write-AGMPHeader "AI游戏管理器面板 $script:AGMPVersion - $modeTitle"
    if ($script:AGMPDryRun) {
        Invoke-GitHubSafetyGate
        Invoke-WindowsHelperGate
        Invoke-ProjectLayoutGate
        Invoke-ReleaseKeyGate -AllowUnconfigured
        $null = Assert-AGMPWailsCli
        $null = Assert-InnoCompiler
        $dryKeyText = if($OfficialPublish){'正式发布时将严格要求 active Release Key'}else{'候选构建无需 active Release Key'}
        Write-Info 'DRY-RUN' $dryKeyText
        Write-Info 'DRY-RUN' '小鱼核心 + Wails 完整产品 + Portable + Inno Setup + lock-safe publish'
        return
    }
    if($OfficialPublish){
        Write-Warn '正式发布：必须通过发行公钥 Gate；最终产物写入 build\release。'
    }else{
        Write-Info '候选构建' '用于本机/测试机验收，不要求正式发行公钥；产物写入 build\candidate，不会冒充正式 Release。'
    }
    Write-Info '许可证' '生产构建不启用 DEVELOPMENT 许可证；未签发 BFLC2 时程序会正常显示未激活。'
    Write-Info '发布边界' '这里只构建完整 AGMP 产品；普通用户安装成品后无需 Rust/MSVC/Go/Node。'
    Show-AGMPToolchain
    Set-AGMPNetworkEnvironment
    Write-Step '[预检] Rust / Cargo / MSVC / Wails / Inno（只检查，不自动安装）...'
    $null=Assert-AGMPRustWindowsToolchain
    $wails = Assert-AGMPWailsCli
    $iscc = Assert-InnoCompiler
    Show-AGMPRustToolchain
    Show-AGMPMSVCBuildTools
    if($OfficialPublish){ Invoke-CoreReleaseGate } else { Invoke-CoreCandidateGate }
    $tools = Get-AGMPTools

    $workRoot = Join-Path $script:AGMPRoot 'build\work\windows-wails'
    $binWork = Join-Path $workRoot 'bin'
    $releaseStage = Join-Path $workRoot 'release-stage'
    $channelRoot = if($OfficialPublish){'build\release\windows\wails'}else{'build\candidate\windows\wails'}
    $release = Join-Path $script:AGMPRoot $channelRoot
    foreach ($p in @($binWork,$releaseStage)) { if(Test-Path -LiteralPath $p){Remove-Item -LiteralPath $p -Recurse -Force}; Ensure-Directory $p }

    $webWork = Join-Path $binWork 'AI-Game-Manager-Web.exe'
    $agentWork = Join-Path $binWork 'AI-Game-Manager-XiaoYu.exe'
    $desktopWork = Join-Path $binWork 'AI-Game-Manager-Panel.exe'
    $portableStage = Join-Path $releaseStage 'portable\AI-Game-Manager-Panel-Windows-x64-Portable.zip'
    $installerName = "AI-Game-Manager-Panel-$script:AGMPVersion-Windows-x64-Setup.exe"
    $installerStage = Join-Path $releaseStage ('installer\' + $installerName)
    Ensure-Directory (Split-Path -Parent $portableStage); Ensure-Directory (Split-Path -Parent $installerStage)

    Write-Step '[1/7] 构建 Web EXE...'
    Invoke-AGMPNative -FilePath $tools.Go -Arguments @('build','-trimpath','-ldflags','-s -w','-o',$webWork,'.\cmd\aigame-manager-web')
    if(-not (Test-FileMinimumSize $webWork)){throw 'Web EXE 无效。'}

    Write-Step '[2/7] 构建 小鱼核心（Rust Runtime）...'
    $null=Build-AGMPXiaoYuCore -Profile release -Destination $agentWork
    if(-not (Test-FileMinimumSize $agentWork)){throw '小鱼核心 Runtime 无效。'}

    Write-Step '[3/7] 构建 Wails Desktop...'
    Invoke-AGMPNative -FilePath $wails -Arguments @('build','-s','-o','AI-Game-Manager-Panel.exe')
    $wailsDefault = Join-Path $script:AGMPRoot 'build\bin\AI-Game-Manager-Panel.exe'
    if(-not (Test-FileMinimumSize $wailsDefault)){throw 'Wails Desktop EXE 无效。'}
    Copy-Item -LiteralPath $wailsDefault -Destination $desktopWork -Force
    Remove-Item -LiteralPath $wailsDefault -Force -ErrorAction SilentlyContinue
    $legacyBin = Join-Path $script:AGMPRoot 'build\bin'
    if((Test-Path $legacyBin) -and -not (Get-ChildItem -LiteralPath $legacyBin -Force | Select-Object -First 1)){Remove-Item -LiteralPath $legacyBin -Force}

    Write-Step '[4/7] 生成 Portable...'
    Build-WailsPortable -DesktopExe $desktopWork -WebExe $webWork -AgentExe $agentWork -DestinationZip $portableStage
    Write-Step '[5/7] 生成 Inno Setup...'
    Build-WailsInstaller -Iscc $iscc -StageInstaller $installerStage
    Write-Step '[6/7] 校验候选产物...'
    if(-not (Test-FileMinimumSize $portableStage) -or -not (Test-FileMinimumSize $installerStage)){throw 'Work 产物不完整。'}

    $publishLabel = if($OfficialPublish){'发布正式 Release'}else{'发布候选 Build'}
    Write-Step "[7/7] $publishLabel（文件锁安全模式）..."
    $publishedPortable = Publish-AGMPArtifact -Source $portableStage -Destination (Join-Path $release 'portable\AI-Game-Manager-Panel-Windows-x64-Portable.zip')
    $publishedSetup = Publish-AGMPArtifact -Source $installerStage -Destination (Join-Path $release ('installer\' + $installerName))
    $setupHash = (Get-FileHash -LiteralPath $publishedSetup -Algorithm SHA256).Hash.ToLowerInvariant()
    $setupHashFile = "$publishedSetup.sha256"
    [IO.File]::WriteAllText($setupHashFile, "$setupHash  $([IO.Path]::GetFileName($publishedSetup))`n", (New-Object Text.UTF8Encoding($false)))
    $setupMB = [Math]::Round((Get-Item -LiteralPath $publishedSetup).Length / 1MB, 2)
    $installerTag = if($OfficialPublish){'正式安装器'}else{'候选安装器'}
    Write-Info $installerTag ("$publishedSetup ($setupMB MB)")
    Write-Info 'SHA256' ("$setupHashFile · $setupHash")
    if($OfficialPublish){ Write-Info 'GitHub Release' '正式发布时上传 Setup.exe 与同名 .sha256；客户端会校验后升级。' }
    $statusName = if($OfficialPublish){'BUILD-STATUS.txt'}else{'BUILD-CANDIDATE-STATUS.txt'}
    $statusChannel = if($OfficialPublish){'RELEASE'}else{'CANDIDATE'}
    Write-StatusFile $statusName @{Status='SUCCESS';Channel=$statusChannel;Version=$script:AGMPVersion;Product='AI-Game-Manager-Panel';Portable=$publishedPortable;Setup=$publishedSetup;SetupSHA256=$setupHashFile;StandaloneAgent='DISABLED'}
    $directoryText = if($OfficialPublish){'build\release = 正式发布产物'}else{'build\candidate = 本地候选验收产物；正式发布请使用菜单 10'}
    Write-Info '目录' $directoryText
    Write-Ok "$modeTitle 完成。"
}

