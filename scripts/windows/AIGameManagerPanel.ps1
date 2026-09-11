param(
    [switch]$SelfTest,
    [switch]$DryRun,
    [int]$Task = -1
)
$ErrorActionPreference='Stop'
$script:AGMPRoot=(Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$script:AGMPDryRun=$DryRun.IsPresent
if($Task -lt -1 -or $Task -gt 10){ throw 'Task 必须是 -1 或 0 到 10。' }
. (Join-Path $PSScriptRoot 'lib\Common.ps1')
Initialize-AGMPConsole
. (Join-Path $PSScriptRoot 'lib\Toolchain.ps1')
. (Join-Path $PSScriptRoot 'lib\Rust.ps1')
. (Join-Path $PSScriptRoot 'lib\Dependencies.ps1')
. (Join-Path $PSScriptRoot 'lib\Checks.ps1')
. (Join-Path $PSScriptRoot 'lib\Wails.ps1')
. (Join-Path $PSScriptRoot 'lib\Electron.ps1')
. (Join-Path $PSScriptRoot 'tasks\Tasks.ps1')
$script:AGMPVersion=Get-AGMPVersion
Set-AGMPNetworkEnvironment

function Show-MainMenu {
    Clear-Host
    Write-Host '====================================================================' -ForegroundColor Cyan
    Write-Host ("                     AI游戏管理器面板开发助手  $script:AGMPVersion") -ForegroundColor Cyan
    Write-Host '                 AI Game Manager Panel Developer / Publisher Helper' -ForegroundColor Gray
    Write-Host '                   PowerShell Task Runner' -ForegroundColor Gray
    Write-Host '====================================================================' -ForegroundColor Cyan
    Write-Host ''
    Write-Host ' [环境准备]' -ForegroundColor Yellow
    Write-Host '   1. 初始化 / 修复基础开发环境 Go + Node + pnpm + Frontend + Wails · 不安装 Rust/MSVC'
    Write-Host ''
    Write-Host ' [日常开发]' -ForegroundColor Yellow
    Write-Host '   2. 开发模式                Wails / Electron / Web / 小鱼核心 · 无需 CDK/BFLC2'
    Write-Host '   3. 项目检查                快速检查 / 完整检查'
    Write-Host ''
    Write-Host ' [候选构建 / 本地验收]' -ForegroundColor Yellow
    Write-Host '   4. Wails Windows 候选构建  Setup + Portable · 不要求 Release Key'
    Write-Host '   5. Electron Windows 候选构建 NSIS + Portable · 不要求 Release Key'
    Write-Host '   6. Linux Server 候选构建   WSL · amd64 / arm64 / all'
    Write-Host '   7. 预览 / 启动构建产物     Wails / Electron / Web / 输出目录'
    Write-Host ''
    Write-Host ' [项目维护]' -ForegroundColor Yellow
    Write-Host '   8. 状态、诊断与修复        工具链 / Rust/MSVC / Inno / Release Key / Gates'
    Write-Host '   9. 清理与重置              work / candidate / Frontend / Electron / 缓存'
    Write-Host ''
    Write-Host ' [正式发布 · 仅发布者]' -ForegroundColor Yellow
    Write-Host '  10. 正式发布                Wails / Electron / Linux / 全平台 · 严格 Release Key'
    Write-Host ''
    Write-Host '   0. 退出'
    Write-Host ''
    Write-Host '--------------------------------------------------------------------' -ForegroundColor DarkGray
    Write-Host ' 重要：这是源码开发/发布助手。普通用户不要运行本脚本，只需安装正式 Setup.exe；无需 Rust/MSVC/Go/Node。' -ForegroundColor Yellow
    Write-Host ' 提示：菜单 4/5/6 是候选构建，不要求发行公钥；只有菜单 10 正式发布会严格校验 Release Key。' -ForegroundColor DarkGray
    Write-Host '====================================================================' -ForegroundColor Cyan
}

function Invoke-MenuTask([int]$Number) {
    switch($Number){
      0{return}
      1{Invoke-InitializeProject}
      2{Invoke-DevelopmentMenu}
      3{Invoke-CheckMenu}
      4{Invoke-WailsRelease}
      5{Invoke-ElectronRelease}
      6{Invoke-LinuxServerRelease}
      7{Invoke-PreviewMenu}
      8{Invoke-StatusAndDiagnostics}
      9{Invoke-MaintenanceMenu}
      10{Invoke-OneClickRelease}
      default{throw "无效菜单：$Number"}
    }
}

function Invoke-HelperSelfTest {
    Write-AGMPHeader "AI游戏管理器面板 $script:AGMPVersion - 菜单 1-10 Dry-Run 自检"
    $previousDryRun=$script:AGMPDryRun
    try {
      $script:AGMPDryRun=$true
      & (Join-Path $PSScriptRoot 'Test-WindowsHelper.ps1')
      foreach($n in 1..10){
        Write-Step "[菜单 $n/10] Dry-Run..."
        Invoke-MenuTask $n
      }
      Write-Ok '菜单 1-10 所有入口 Dry-Run 通过。'
    } finally {
      $script:AGMPDryRun=$previousDryRun
    }
}

try {
    if($SelfTest){Invoke-HelperSelfTest;exit 0}
    if($Task -ge 0){Invoke-MenuTask $Task;exit 0}
    while($true){
      Show-MainMenu
      $raw=Read-Host '请选择 [0-10]'
      $choice=0
      if(-not [int]::TryParse($raw,[ref]$choice) -or $choice -lt 0 -or $choice -gt 10){Write-Warn '请输入 0 到 10。';Start-Sleep -Milliseconds 700;continue}
      if($choice -eq 0){exit 0}
      try{Invoke-MenuTask $choice;Write-Host '';Write-Ok '本次操作完成。'}catch{Write-Host '';Write-Fail $_.Exception.Message}
      Write-Host '';[void](Read-Host '按 Enter 返回菜单')
    }
}catch{
    Write-Host '';Write-Fail $_.Exception.Message
    exit 1
}
