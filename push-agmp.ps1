[CmdletBinding()]
param()
$ErrorActionPreference='Stop'
$Root=Split-Path -Parent $PSCommandPath
function Step($t){Write-Host ('[进行] '+$t) -ForegroundColor Cyan}
function Ok($t){Write-Host ('[完成] '+$t) -ForegroundColor Green}
function Fail($t){Write-Host ('[失败] '+$t) -ForegroundColor Red;exit 1}
try{$utf8=New-Object System.Text.UTF8Encoding($false);[Console]::InputEncoding=$utf8;[Console]::OutputEncoding=$utf8;$OutputEncoding=$utf8}catch{}
Set-Location $Root
Write-Host '====================================================================' -ForegroundColor DarkGray
Write-Host '  AGMP GitHub 一键推送助手' -ForegroundColor White
Write-Host '  架构：TypeScript Agent + Rust Native · Go=0' -ForegroundColor DarkGray
Write-Host '====================================================================' -ForegroundColor DarkGray
Write-Host '  1. [一键推送]  安全检查 > 提交 > Push'
Write-Host '  2. [查看状态]  git status / 最近提交'
Write-Host '  3. [仅安全检查]'
Write-Host '  0. 退出'
$choice=Read-Host '请选择'
if($choice -eq '0'){exit 0}
if(-not(Get-Command git.exe -ErrorAction SilentlyContinue)){Fail '未找到 git.exe'}
if(-not(Test-Path '.git')){Fail '当前目录不是 Git 工作副本；请先运行 AGMP-Sync.bat 同步到 H:\一键部署\AI-Game-Manager-Panel'}
if($choice -eq '2'){& git.exe status --short --branch;& git.exe log -5 --oneline;exit $LASTEXITCODE}
function Safety{
  $generated=Join-Path $Root 'build\work'
  if(Test-Path -LiteralPath $generated){
    Step '清理可丢弃 build/work 构建缓存'
    Remove-Item -LiteralPath $generated -Recurse -Force -ErrorAction SilentlyContinue
  }
  Step '运行 0.4.0 Architecture / No-Go / Naming / GitHub Safety Gates'
  foreach($script in @('scripts/gates/check-architecture.mjs','scripts/gates/check-no-go.mjs','scripts/gates/check-ui-freeze.mjs','scripts/common/check-naming.mjs','scripts/common/check-github-safety.mjs')){& node.exe $script;if($LASTEXITCODE -ne 0){Fail ('Gate 失败：'+$script)}}
  $goFiles=@(Get-ChildItem -Recurse -File -Filter '*.go' -ErrorAction SilentlyContinue | Where-Object {$_.FullName -notmatch '\\node_modules\\|\\runtime\\|\\target\\'})
  if($goFiles.Count -gt 0){Fail ('发现 Go 源码：'+$goFiles[0].FullName)}
  Ok '安全检查通过。'
}
Safety
if($choice -eq '3'){exit 0}
if($choice -ne '1'){Fail '无效选项'}
Step '同步远端 main'
& git.exe fetch origin;if($LASTEXITCODE -ne 0){Fail 'git fetch 失败'}
& git.exe pull --rebase --autostash origin main;if($LASTEXITCODE -ne 0){Fail 'git pull --rebase 失败'}
Safety
& git.exe add -A
$status=& git.exe status --porcelain
if(-not $status){Ok '没有需要提交的修改。';exit 0}
$message=Read-Host '提交说明（回车使用 AGMP 0.4.0 TS + Rust architecture reset）'
if([string]::IsNullOrWhiteSpace($message)){$message='AGMP 0.4.0 TS + Rust architecture reset'}
& git.exe commit -m $message;if($LASTEXITCODE -ne 0){Fail 'git commit 失败'}
Step 'Push origin main'
& git.exe push origin main;if($LASTEXITCODE -ne 0){Fail 'git push 失败'}
Ok '推送完成。告诉 ChatGPT“推送了”，继续检查 GitHub Actions。'
