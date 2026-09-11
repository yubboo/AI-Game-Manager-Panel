param([switch]$Detailed)
$ErrorActionPreference='Stop'
$root=(Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$failures=New-Object Collections.Generic.List[string]
function Fail([string]$m){$failures.Add($m)}
try{
  $bat=[IO.File]::ReadAllBytes((Join-Path $root 'AI-Game-Manager-Panel.bat'))
  if($bat | Where-Object {$_ -gt 127}){Fail 'AI-Game-Manager-Panel.bat 不是 ASCII-safe。'}
  $text=[Text.Encoding]::ASCII.GetString($bat)
  if($text -notmatch 'chcp 65001' -or $text -notmatch 'scripts\\windows\\AIGameManagerPanel.ps1'){Fail 'AI-Game-Manager-Panel.bat 未正确委托 AIGameManagerPanel.ps1。'}
  if(($text -replace "`r`n",'').Contains("`n")){Fail 'AI-Game-Manager-Panel.bat 不是 CRLF。'}
  $psFiles=Get-ChildItem (Join-Path $root 'scripts\windows') -Filter *.ps1 -Recurse
  foreach($f in $psFiles){
    $bytes=[IO.File]::ReadAllBytes($f.FullName)
    if($bytes.Length -lt 3 -or $bytes[0]-ne 0xEF -or $bytes[1]-ne 0xBB -or $bytes[2]-ne 0xBF){Fail "PS1 缺少 UTF-8 BOM：$($f.FullName)"}
    $s=[IO.File]::ReadAllText($f.FullName)
    if($s.Contains([char]0xFFFD)){Fail "PS1 包含 Unicode replacement char：$($f.FullName)"}
    $forbiddenInvoke=('Invoke' + '-Expression')
    $forbiddenPrefer=('--prefer' + '-online')
    if($s -match ('(?i)' + [regex]::Escape($forbiddenInvoke))){Fail "禁止危险 PowerShell 动态执行：$($f.FullName)"}
    if($s -match [regex]::Escape($forbiddenPrefer)){Fail "禁止未知 pnpm 在线偏好参数：$($f.FullName)"}
    $automaticAssignments='(?im)^\s*\$(home|host|pid|psversiontable|psscriptroot|pscommandpath)\s*='
    if($s -match $automaticAssignments){Fail "PowerShell 自动/只读变量发生可写赋值冲突：$($f.FullName)"}
  }

  $toolchain=[IO.File]::ReadAllText((Join-Path $root 'scripts\windows\lib\Toolchain.ps1'))
  $showStart=$toolchain.IndexOf('function Show-AGMPToolchain')
  $showEnd=$toolchain.IndexOf('function Invoke-GoModules')
  if($showStart -lt 0 -or $showEnd -le $showStart){Fail '无法定位 Show-AGMPToolchain。'} else {
    $showBlock=$toolchain.Substring($showStart,$showEnd-$showStart)
    if($showBlock -match '(?i)return\s+\$tools\b'){Fail 'Show-AGMPToolchain 禁止返回工具链 Hashtable。'}
  }
  $main=[IO.File]::ReadAllText((Join-Path $root 'scripts\windows\AIGameManagerPanel.ps1'))
  $tasks=[IO.File]::ReadAllText((Join-Path $root 'scripts\windows\tasks\Tasks.ps1'))
  $rust=[IO.File]::ReadAllText((Join-Path $root 'scripts\windows\lib\Rust.ps1'))
  $wails=[IO.File]::ReadAllText((Join-Path $root 'scripts\windows\lib\Wails.ps1'))
  $checks=[IO.File]::ReadAllText((Join-Path $root 'scripts\windows\lib\Checks.ps1'))
  $bridge=[IO.File]::ReadAllText((Join-Path $root 'internal\bridge\wails\app.go'))
  if(-not $main.Contains('菜单 4/5/6 是候选构建') -or -not $main.Contains('只有菜单 10 正式发布')){Fail '主菜单没有明确 Candidate / Official Publish 边界。'}
  if(-not $checks.Contains('Invoke-CoreCandidateGate') -or -not $checks.Contains('Invoke-CoreReleaseGate')){Fail '缺少 Candidate / Release 两套 Core Gate。'}
  if(-not $tasks.Contains('Invoke-LinuxServerRelease -OfficialPublish')){Fail '菜单 10 自检/正式发布没有覆盖 Linux OfficialPublish。'}
  if(-not $rust.Contains('if($script:AGMPDryRun)') -or -not $rust.Contains('预期小鱼核心 Runtime')){Fail '小鱼核心 Dry-Run 仍可能依赖真实 xiaoyu.exe。'}
  if(-not $wails.Contains("@('test','-tags','agmp_dev_license','./internal/bridge/wails')")){Fail 'Wails 开发启动前缺少 Bridge Go 编译预检。'}
  if($bridge -match '\bdstruntime\.' -or $bridge -match '\btoken\.(Status|SaveRequest|ImportRequest)'){Fail 'Wails Bridge 存在 DST import alias/shadowing 回归。'}
  if(-not $text.Contains('%*')){Fail '根 BAT 没有转发快捷参数。'}

  $electronBuilder=[IO.File]::ReadAllText((Join-Path $root 'desktop\electron\electron-builder.yml'))
  if($electronBuilder -notmatch '(?m)^electronDist:\s*node_modules/electron/dist\s*$'){Fail 'Electron Builder 缺少本地 electronDist。'}
  $electronMain=[IO.File]::ReadAllText((Join-Path $root 'desktop\electron\src\main.ts'))
  foreach($label in @("label: '文件'","label: '编辑'","label: '视图'","label: '窗口'","label: '帮助'")){if(-not $electronMain.Contains($label)){Fail "Electron 原生菜单缺少中文项：$label"}}

  $batInside=Get-ChildItem (Join-Path $root 'scripts\windows') -Filter *.bat -Recurse -ErrorAction SilentlyContinue
  if($batInside.Count -gt 0){Fail 'scripts/windows 下禁止残留 BAT；唯一 BAT 入口只能是根 AI-Game-Manager-Panel.bat。'}
}catch{Fail $_.Exception.Message}
if($failures.Count -gt 0){foreach($f in $failures){Write-Host "[FAIL] $f" -ForegroundColor Red};throw 'Windows Helper 编码与结构门禁失败。'}
Write-Host '[PASS] Windows Helper 编码与结构门禁通过。' -ForegroundColor Green
