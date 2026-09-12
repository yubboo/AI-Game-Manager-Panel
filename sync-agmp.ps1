[CmdletBinding()]
param([string]$Destination = 'H:\一键部署\AI-Game-Manager-Panel')
$ErrorActionPreference = 'Stop'
$Source = Split-Path -Parent $PSCommandPath
$Destination = [System.IO.Path]::GetFullPath($Destination)
function Step($t){Write-Host ('[进行] '+$t) -ForegroundColor Cyan}
function Ok($t){Write-Host ('[完成] '+$t) -ForegroundColor Green}
function Fail($t){Write-Host ('[失败] '+$t) -ForegroundColor Red; exit 1}
try {$utf8=New-Object System.Text.UTF8Encoding($false);[Console]::InputEncoding=$utf8;[Console]::OutputEncoding=$utf8;$OutputEncoding=$utf8}catch{}
Write-Host '====================================================================' -ForegroundColor DarkGray
Write-Host '  AGMP 0.4.0 TS + Rust 源码同步助手' -ForegroundColor White
Write-Host ('  来源：'+$Source)
Write-Host ('  目标：'+$Destination)
Write-Host '====================================================================' -ForegroundColor DarkGray
$required=@('package.json','apps\server\src\main.ts','packages\agent\src\index.ts','packages\minecraft\src\index.ts','crates\Cargo.toml','crates\native-runtime\src\lib.rs','frontend\package.json','desktop\electron\src\main.ts','desktop\electron\src\preload.cts','desktop\electron\tsconfig.json','scripts\gates\check-no-go.mjs')
foreach($rel in $required){if(-not(Test-Path -LiteralPath (Join-Path $Source $rel) -PathType Leaf)){Fail ('源码包不完整，缺少：'+$rel)}}
if($Source.TrimEnd('\') -ieq $Destination.TrimEnd('\')){Ok '当前已经是目标工作副本。';exit 0}
if(-not(Test-Path -LiteralPath $Destination)){New-Item -ItemType Directory -Path $Destination -Force|Out-Null}
if(-not(Get-Command robocopy.exe -ErrorAction SilentlyContinue)){Fail '未找到 robocopy.exe'}
Step '复制新架构源码（保留目标 .git 与本机 runtime）'
$exclude=@((Join-Path $Source '.git'),(Join-Path $Source 'node_modules'),(Join-Path $Source '.tmp-test'),(Join-Path $Source 'build'),(Join-Path $Source 'frontend\node_modules'),(Join-Path $Source 'frontend\dist'),(Join-Path $Source 'desktop\electron\node_modules'),(Join-Path $Source 'dist'),(Join-Path $Source 'target'),(Join-Path $Source 'crates\target'),(Join-Path $Source 'runtime\data'),(Join-Path $Source 'runtime\workspace'),(Join-Path $Source 'runtime\native'))
$log=Join-Path ([IO.Path]::GetTempPath()) ('agmp-sync-'+[guid]::NewGuid().ToString('N')+'.log')
& robocopy.exe $Source $Destination /E /COPY:DAT /DCOPY:DAT /R:2 /W:1 /XJ /NFL /NDL /NP /XD $exclude ('/UNILOG:'+$log)|Out-Null
$code=$LASTEXITCODE
if($code -ge 8){if(Test-Path $log){Get-Content $log -Encoding Unicode|Select-Object -Last 80};Fail ('robocopy 失败：'+$code)}
Remove-Item $log -Force -ErrorAction SilentlyContinue
Ok '复制完成。'
Step '清理目标工作副本中的可丢弃构建缓存'
foreach($generated in @('build\work','.tmp-test')){
  $p=Join-Path $Destination $generated
  if(Test-Path -LiteralPath $p){Remove-Item -LiteralPath $p -Recurse -Force -ErrorAction SilentlyContinue}
}
Ok '构建缓存清理完成。'
if((Test-Path (Join-Path $Destination '.git')) -and (Get-Command git.exe -ErrorAction SilentlyContinue)){
  Step '删除 Git 已跟踪但新 0.4.0 已移除的旧源码（Go/Wails 也会在这里清掉）'
  $tracked=@(& git.exe -C $Destination ls-files)
  foreach($rel in $tracked){if([string]::IsNullOrWhiteSpace($rel)){continue};$src=Join-Path $Source ($rel -replace '/','\');$dst=Join-Path $Destination ($rel -replace '/','\');if(-not(Test-Path -LiteralPath $src) -and (Test-Path -LiteralPath $dst -PathType Leaf)){Remove-Item -LiteralPath $dst -Force;Write-Host ('  - 移除旧源码：'+$rel) -ForegroundColor DarkGray}}
}
foreach($legacy in @('go.mod','go.sum','main.go','wails.json')){$p=Join-Path $Destination $legacy;if(Test-Path $p){Remove-Item $p -Force}}
foreach($legacyDir in @('cmd','internal','rust')){$p=Join-Path $Destination $legacyDir;if(Test-Path $p){Remove-Item $p -Recurse -Force}}
foreach($rel in $required){if(-not(Test-Path -LiteralPath (Join-Path $Destination $rel) -PathType Leaf)){Fail ('同步后缺少：'+$rel)}}
Ok '0.4.0 新架构源码同步完成。'
Write-Host ''
Write-Host '下一步：在目标目录运行 AGMP-GitHub.bat -> 1 一键推送。' -ForegroundColor Cyan
