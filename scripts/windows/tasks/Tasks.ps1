function Invoke-InitializeProject {
    Write-AGMPHeader "AI游戏管理器面板 $script:AGMPVersion - 初始化 / 修复基础开发环境"
    Write-Step '[1/5] 检查项目结构...'; Invoke-ProjectLayoutGate
    Write-Step '[2/5] 检查 Go / Node.js / pnpm...'; Show-AGMPToolchain
    Set-AGMPNetworkEnvironment
    Write-Step '[3/5] Go Modules...'; Invoke-GoModules
    Write-Step '[4/5] Frontend Dependencies...'; Install-FrontendDependencies
    Write-Step '[5/5] Wails CLI...'; $null=Install-AGMPWailsCli
    Write-Ok '基础开发环境初始化 / 修复完成。'
    Write-Info '边界' '菜单 1 不会安装 Rust、Cargo、MSVC 或 Visual Studio Build Tools。'
    Write-Info '可选工具链' '需要开发小鱼 Rust Core 或制作 Windows 候选/正式包时，请在菜单 8 中显式准备 Rust + MSVC。'
    Write-Info '生产环境' '普通用户只安装正式 AGMP 成品，无需 Go / Node / pnpm / Rust / MSVC / Wails。'
}

function Invoke-DevelopmentMenu {
    if ($script:AGMPDryRun) { Invoke-WailsDevelopment; Invoke-ElectronDevelopment; Invoke-WebDevelopment; Invoke-RustAgentDevelopment; return }
    Write-Host ''
    Write-Host '  开发模式自动启用 DEVELOPMENT 许可证状态，无需 CDK / BFLC2 / Release Key。' -ForegroundColor DarkGray
    Write-Host '  1. Wails 桌面开发          推荐 Windows 主开发入口'
    Write-Host '  2. Electron 兼容桌面开发  用于兼容性验证'
    Write-Host '  3. Web 一体服务开发        Go Web + 已构建前端 + 小鱼核心'
    Write-Host '  4. 小鱼核心 CLI            Rust Runtime / Doctor / Tool Registry'
    Write-Host '  0. 返回'
    switch (Read-Host '请选择 [0-4]') {
      '1'{Invoke-WailsDevelopment}
      '2'{Invoke-ElectronDevelopment}
      '3'{Invoke-WebDevelopment}
      '4'{Invoke-RustAgentDevelopment}
      default{return}
    }
}

function Invoke-WebDevelopment {
    Write-AGMPHeader "AI游戏管理器面板 $script:AGMPVersion - Web 一体服务开发"
    Show-AGMPToolchain
    Set-AGMPNetworkEnvironment
    Invoke-GoModules
    Install-FrontendDependencies
    Invoke-FrontendTypeCheck
    $tools = Get-AGMPTools
    $webExe = Join-Path $script:AGMPRoot 'build\work\dev\AI-Game-Manager-Web.exe'; Ensure-Directory (Split-Path -Parent $webExe)
    if ($script:AGMPDryRun) { Write-Info 'DRY-RUN' '构建并启动 Go Web + Frontend + 小鱼核心'; return }
    Invoke-FrontendBuild
    Invoke-SyncWebAssets
    $agent=Build-AGMPXiaoYuCore -Profile debug
    $oldAgent=$env:AGMP_XIAOYU_RUNTIME
    $env:AGMP_XIAOYU_RUNTIME=$agent
    Write-Step '[预检] Web Go 编译...'
    Invoke-AGMPNative -FilePath $tools.Go -Arguments @('build','-tags','agmp_dev_license','-trimpath','-o',$webExe,'.\cmd\aigame-manager-web')
    Write-Info 'Web' 'Go 后端与生产前端资源已经同步，本窗口直接承载完整 Web 开发实例。'
    Write-Info '小鱼核心' $agent
    Write-Info '地址' 'http://127.0.0.1:17890'
    Write-Info '提示' '保持本窗口开启；Ctrl+C 停止服务。'
    try { & $webExe } finally { $env:AGMP_XIAOYU_RUNTIME=$oldAgent }
}

function Invoke-CheckMenu {
    if ($script:AGMPDryRun) { Invoke-QuickProjectCheck; Invoke-FullProjectCheck; return }
    Write-Host ''
    Write-Host '  1. 快速检查              日常开发 Gate + TypeScript + Go Core'
    Write-Host '  2. 完整构建检查          不要求正式发行密钥'
    Write-Host '  3. 开发助手菜单自检      1-10 Dry-Run，不执行真实构建'
    Write-Host '  0. 返回'
    switch (Read-Host '请选择 [0-3]') {
      '1'{Invoke-QuickProjectCheck}
      '2'{Invoke-FullProjectCheck}
      '3'{Invoke-HelperSelfTest}
      default{return}
    }
}

function Invoke-LinuxServerRelease {
    param([switch]$OfficialPublish)
    $modeTitle = if($OfficialPublish){'Linux Server 正式发布'}else{'Linux Server 候选构建'}
    Write-AGMPHeader "AI游戏管理器面板 $script:AGMPVersion - $modeTitle"
    if($script:AGMPDryRun){
        Invoke-ReleaseKeyGate -AllowUnconfigured
        Write-Info 'DRY-RUN' 'WSL -> scripts/build_linux.sh [amd64|arm64|all]'
        return
    }
    $target='amd64'
    Write-Host ''
    Write-Host '  1. amd64 / x86_64          常见云服务器 / WSL'
    Write-Host '  2. arm64 / aarch64         ARM 云服务器'
    Write-Host '  3. all                      仅已配置 Rust 交叉工具链时使用'
    Write-Host '  0. 返回'
    switch(Read-Host '请选择 Linux 目标 [0-3]'){
      '1'{$target='amd64'}
      '2'{$target='arm64'}
      '3'{$target='all'}
      default{return}
    }
    if($OfficialPublish){ Invoke-EnsureReleasePublicKey; Invoke-ReleaseKeyGate } else { Invoke-ReleaseKeyGate -AllowUnconfigured }
    $tools = Get-AGMPTools -RequireWsl
    $wslRoot = (& $tools.Wsl wslpath -a $script:AGMPRoot 2>&1 | Out-String).Trim()
    if([string]::IsNullOrWhiteSpace($wslRoot)){throw '无法转换项目路径为 WSL 路径。'}
    $channel=if($OfficialPublish){'release'}else{'candidate'}
    # Bash 双引号转义，避免 Windows 项目路径中的空格破坏 WSL 命令。
    $safeRoot=$wslRoot.Replace('\','\\').Replace('"','\"').Replace('$','\$').Replace('`','\`')
    $command='cd "' + $safeRoot + '" && AGMP_BUILD_CHANNEL=' + $channel + ' bash scripts/build_linux.sh ' + $target
    Write-Info 'WSL' "目标：$target · Channel：$channel"
    Invoke-AGMPNative -FilePath $tools.Wsl -Arguments @('bash','-lc',$command)
    Write-Ok "$modeTitle 完成。"
}

function Get-AGMPExistingArtifact([string[]]$RelativeCandidates) {
    foreach($rel in $RelativeCandidates){
        $path=Join-Path $script:AGMPRoot $rel
        if(Test-Path -LiteralPath $path -PathType Leaf){return $path}
    }
    return $null
}

function Invoke-PreviewMenu {
    if($script:AGMPDryRun){Write-Info 'DRY-RUN' '预览 Wails / Electron / Web / 打开候选与正式产物目录';return}
    Write-Host ''
    Write-Host '  1. 启动 Wails Desktop 工作产物'
    Write-Host '  2. 启动 Electron Portable   优先候选，其次正式版'
    Write-Host '  3. 启动 Web 工作产物'
    Write-Host '  4. 打开 build\candidate'
    Write-Host '  5. 打开 build\release'
    Write-Host '  0. 返回'
    $choice=Read-Host '请选择 [0-5]'
    switch($choice){
      '1'{
          $p=Join-Path $script:AGMPRoot 'build\work\windows-wails\bin\AI-Game-Manager-Panel.exe'
          $agent=Join-Path $script:AGMPRoot 'build\work\windows-wails\bin\AI-Game-Manager-XiaoYu.exe'
          if(!(Test-Path $p) -or !(Test-Path $agent)){throw '尚未生成 Wails 工作产物。请先运行菜单 4。'}
          $oldAgent=$env:AGMP_XIAOYU_RUNTIME; $env:AGMP_XIAOYU_RUNTIME=$agent
          try { & $p } finally { $env:AGMP_XIAOYU_RUNTIME=$oldAgent }
      }
      '2'{
          $p=Get-AGMPExistingArtifact @(
            "build\candidate\windows\electron\portable\AI-Game-Manager-Panel-$script:AGMPVersion-x64-Portable.exe",
            "build\release\windows\electron\portable\AI-Game-Manager-Panel-$script:AGMPVersion-x64-Portable.exe"
          )
          if(-not $p){throw '尚未生成 Electron Portable。请先运行菜单 5 或菜单 10。'}
          & $p
      }
      '3'{
          $p=Get-AGMPExistingArtifact @('build\work\dev\AI-Game-Manager-Web.exe','build\work\windows-wails\bin\AI-Game-Manager-Web.exe')
          $agent=Get-AGMPExistingArtifact @('build\work\rust-target\debug\xiaoyu.exe','build\work\windows-wails\bin\AI-Game-Manager-XiaoYu.exe')
          if(-not $p -or -not $agent){throw '尚未生成 Web + 小鱼核心工作产物。请先运行菜单 2 → 3 或菜单 4。'}
          $oldAgent=$env:AGMP_XIAOYU_RUNTIME; $env:AGMP_XIAOYU_RUNTIME=$agent
          Write-Info '地址' 'http://127.0.0.1:17890'
          try { & $p } finally { $env:AGMP_XIAOYU_RUNTIME=$oldAgent }
      }
      '4'{ $p=Join-Path $script:AGMPRoot 'build\candidate'; Ensure-Directory $p; Start-Process explorer.exe -ArgumentList @($p) }
      '5'{ $p=Join-Path $script:AGMPRoot 'build\release'; Ensure-Directory $p; Start-Process explorer.exe -ArgumentList @($p) }
      default{return}
    }
}

function Show-AGMPStatus {
    Write-AGMPHeader "AI游戏管理器面板 $script:AGMPVersion - 状态与诊断"
    Show-AGMPToolchain
    Show-AGMPRustToolchain
    Show-AGMPMSVCBuildTools
    Write-Info '根目录' $script:AGMPRoot
    Write-Info 'pnpm Store' (Get-AGMPPnpmStore)
    Write-Info 'Electron Cache' (Get-AGMPElectronCache)
    Write-Host ''
    Write-Host '  build 目录：' -ForegroundColor Yellow
    Write-Host '    build\candidate\ 候选验收产物：不要求正式发行密钥'
    Write-Host '    build\release\   正式发布产物：只有菜单 10 可以写入'
    Write-Host '    build\work\      临时构建文件：可安全清理'
    Write-Host ''
    Write-Host '  发行密钥：' -ForegroundColor Yellow
    Invoke-ReleaseKeyGate -AllowUnconfigured
    Write-Info '许可证发行工具' 'scripts\tools\license\AGMP-License-Admin.bat'
    $wails = Get-AGMPWailsCliPath; if($wails){Write-Info 'Wails CLI' $wails}else{Write-Warn 'Wails CLI 尚未准备；请运行菜单 1。'}
    $inno = Get-InnoCompiler; if($inno){Write-Info 'Inno Setup' $inno}else{Write-Warn 'Inno Setup 未检测到；需要 Wails 安装包时请在菜单 8 显式准备。'}
    Invoke-WindowsHelperGate
}

function Invoke-ReleaseKeyMaintenance {
    Write-AGMPHeader "AI游戏管理器面板 $script:AGMPVersion - 发行密钥状态"
    Invoke-ReleaseKeyGate -AllowUnconfigured
    if(Test-AGMPActiveReleasePublicKey){ Write-Ok '当前源码已经配置 active 发行公钥。'; return }
    $tool=Join-Path $script:AGMPRoot 'scripts\tools\license\AGMP-License-Admin.bat'
    Write-Warn '当前没有 active 发行公钥；这不影响开发和候选构建，只阻止菜单 10 正式发布。'
    if($script:AGMPDryRun){return}
    Write-Host '  1. 尝试从仓库外 ReleaseKeys 同步已有公钥'
    Write-Host '  2. 打开 License Admin 工具（首次初始化/轮换由你手动确认）'
    Write-Host '  0. 返回'
    switch(Read-Host '请选择 [0-2]'){
      '1'{Invoke-EnsureReleasePublicKey;Invoke-ReleaseKeyGate}
      '2'{if(!(Test-Path $tool)){throw "缺少许可证管理工具：$tool"};& $tool}
      default{return}
    }
}

function Invoke-StatusAndDiagnostics {
    if($script:AGMPDryRun){Show-AGMPStatus;return}
    Write-Host ''
    Write-Host '  1. 查看完整状态与 Gates'
    Write-Host '  2. 修复基础开发环境        Go Modules / Frontend / Wails CLI'
    Write-Host '  3. 准备 Rust + MSVC         仅小鱼源码开发 / Windows 构建'
    Write-Host '  4. 准备 Inno Setup          仅 Wails 安装包构建'
    Write-Host '  5. 发行密钥状态 / 同步'
    Write-Host '  6. Windows Helper 结构自检'
    Write-Host '  0. 返回'
    switch(Read-Host '请选择 [0-6]'){
      '1'{Show-AGMPStatus}
      '2'{Invoke-InitializeProject}
      '3'{$null=Ensure-AGMPRustWindowsToolchain;Write-Ok 'Rust + MSVC 已就绪。'}
      '4'{$null=Ensure-InnoCompiler;Write-Ok 'Inno Setup 已就绪。'}
      '5'{Invoke-ReleaseKeyMaintenance}
      '6'{& (Join-Path $script:AGMPRoot 'scripts\windows\Test-WindowsHelper.ps1') -Detailed}
      default{return}
    }
}

function Invoke-MaintenanceMenu {
    if($script:AGMPDryRun){Write-Info 'DRY-RUN' '清理 work / candidate / Frontend / Electron / AGMP 专用依赖缓存';return}
    Write-Host ''
    Write-Host '  1. 清理 build\work              保留 candidate / release'
    Write-Host '  2. 清理 build\candidate         保留正式 release'
    Write-Host '  3. 重置 Frontend 依赖'
    Write-Host '  4. 重置 Electron 依赖'
    Write-Host '  5. 重置前端/Electron + AGMP 专用 pnpm Store'
    Write-Host '  0. 返回'
    $choice=Read-Host '请选择 [0-5]'
    switch($choice){
      '1'{ $p=Join-Path $script:AGMPRoot 'build\work';if(Test-Path $p){Remove-Item $p -Recurse -Force};Write-Ok 'build\work 已清理。' }
      '2'{ $p=Join-Path $script:AGMPRoot 'build\candidate';if(Test-Path $p){Remove-Item $p -Recurse -Force};Write-Ok 'build\candidate 已清理；正式 release 未动。' }
      '3'{ $p=Join-Path $script:AGMPRoot 'frontend\node_modules';if(Test-Path $p){Remove-Item $p -Recurse -Force};Install-FrontendDependencies }
      '4'{ $p=Join-Path $script:AGMPRoot 'desktop\electron\node_modules';if(Test-Path $p){Remove-Item $p -Recurse -Force};Install-ElectronDependencies }
      '5'{ foreach($rel in @('frontend\node_modules','desktop\electron\node_modules')){$p=Join-Path $script:AGMPRoot $rel;if(Test-Path $p){Remove-Item $p -Recurse -Force}};$store=Get-AGMPPnpmStore;if(Test-Path $store){Remove-Item $store -Recurse -Force};Install-FrontendDependencies;Install-ElectronDependencies }
      default{return}
    }
}

function Invoke-OneClickRelease {
    if($script:AGMPDryRun){
      Invoke-WailsRelease -OfficialPublish
      Invoke-ElectronRelease -OfficialPublish
      Invoke-LinuxServerRelease -OfficialPublish
      return
    }
    Write-AGMPHeader "AI游戏管理器面板 $script:AGMPVersion - 正式发布"
    Write-Warn '只有本菜单执行正式 Release Key Gate，并写入 build\release。菜单 4/5/6 只是候选构建。'
    Write-Host '  1. Wails Windows 正式发布     推荐主发行版'
    Write-Host '  2. Electron Windows 正式发布  兼容发行版'
    Write-Host '  3. Linux Server 正式发布'
    Write-Host '  4. Windows 全部               Wails → Electron'
    Write-Host '  5. 全平台                     Wails → Electron → Linux'
    Write-Host '  0. 返回'
    switch(Read-Host '请选择 [0-5]'){
      '1'{Invoke-WailsRelease -OfficialPublish}
      '2'{Invoke-ElectronRelease -OfficialPublish}
      '3'{Invoke-LinuxServerRelease -OfficialPublish}
      '4'{Invoke-WailsRelease -OfficialPublish;Invoke-ElectronRelease -OfficialPublish}
      '5'{Invoke-WailsRelease -OfficialPublish;Invoke-ElectronRelease -OfficialPublish;Invoke-LinuxServerRelease -OfficialPublish}
      default{return}
    }
}
