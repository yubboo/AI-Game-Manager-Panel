[CmdletBinding()]
param(
    [string]$RepoUrl = "https://github.com/yubboo/AI-Game-Manager-Panel.git",
    [string]$Branch = "main",
    [string]$Version = "0.2.2",
    [string]$CommitMessage = "",
    [switch]$NoPush
)

$ErrorActionPreference = "Stop"

function Write-Step($msg) {
    Write-Host ""
    Write-Host "==> $msg" -ForegroundColor Cyan
}

function Write-Ok($msg) {
    Write-Host "[OK] $msg" -ForegroundColor Green
}

function Write-Warn2($msg) {
    Write-Host "[WARN] $msg" -ForegroundColor Yellow
}

function Fail($msg) {
    Write-Host "[FAIL] $msg" -ForegroundColor Red
    exit 1
}

# 默认以脚本所在目录作为项目根目录。
$ProjectRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $ProjectRoot

Write-Host "============================================================" -ForegroundColor DarkGray
Write-Host "  AGMP GitHub Push Helper" -ForegroundColor White
Write-Host "  Project: $ProjectRoot"
Write-Host "  Remote : $RepoUrl"
Write-Host "  Branch : $Branch"
Write-Host "============================================================" -ForegroundColor DarkGray

Write-Step "检查 Git"
if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
    Fail "未找到 Git。请先安装 Git for Windows，然后重新运行本脚本。"
}
$gitVersion = git --version
Write-Ok $gitVersion

# 防止把下载 ZIP / 构建包当源码提交。
Write-Step "检查不应提交的文件"
$zipFiles = Get-ChildItem -Path $ProjectRoot -File -Recurse -Filter "*.zip" -ErrorAction SilentlyContinue |
    Where-Object { $_.FullName -notmatch '\\\.git\\' }

if ($zipFiles.Count -gt 0) {
    Write-Warn2 "发现 ZIP 文件；脚本会在暂存后自动取消暂存这些文件："
    $zipFiles | ForEach-Object { Write-Host "  - $($_.FullName.Substring($ProjectRoot.Length + 1))" }
}

# 常见敏感文件只做阻止，不自动删除。
$sensitivePatterns = @(
    ".env",
    ".env.local",
    ".env.production",
    ".env.development",
    "*.pem",
    "*.pfx",
    "*.p12",
    "*.key"
)

$sensitiveFiles = @()
foreach ($pattern in $sensitivePatterns) {
    $sensitiveFiles += Get-ChildItem -Path $ProjectRoot -File -Recurse -Filter $pattern -ErrorAction SilentlyContinue |
        Where-Object { $_.FullName -notmatch '\\\.git\\' -and $_.FullName -notmatch '\\node_modules\\' }
}

if ($sensitiveFiles.Count -gt 0) {
    Write-Warn2 "发现可能包含凭据的文件。脚本不会自动提交这些文件："
    $sensitiveFiles | Sort-Object FullName -Unique | ForEach-Object {
        Write-Host "  - $($_.FullName.Substring($ProjectRoot.Length + 1))"
    }
}

Write-Step "初始化 / 检查 Git 仓库"
if (-not (Test-Path (Join-Path $ProjectRoot ".git"))) {
    git init | Out-Host
    if ($LASTEXITCODE -ne 0) { Fail "git init 失败。" }
    Write-Ok "已初始化 Git 仓库。"
} else {
    Write-Ok "当前目录已经是 Git 仓库。"
}

Write-Step "设置主分支为 $Branch"
git branch -M $Branch
if ($LASTEXITCODE -ne 0) { Fail "无法切换/重命名主分支。" }
Write-Ok "当前主分支：$Branch"

Write-Step "配置远程仓库"
$origin = git remote get-url origin 2>$null
if ($LASTEXITCODE -eq 0 -and $origin) {
    if ($origin.Trim() -ne $RepoUrl) {
        Write-Warn2 "origin 当前为：$origin"
        git remote set-url origin $RepoUrl
        if ($LASTEXITCODE -ne 0) { Fail "更新 origin 失败。" }
        Write-Ok "origin 已更新为：$RepoUrl"
    } else {
        Write-Ok "origin 已正确配置。"
    }
} else {
    git remote add origin $RepoUrl
    if ($LASTEXITCODE -ne 0) { Fail "添加 origin 失败。" }
    Write-Ok "已添加 origin：$RepoUrl"
}

Write-Step "暂存源码"
git add -A
if ($LASTEXITCODE -ne 0) { Fail "git add 失败。" }

# 取消暂存 ZIP。
$staged = @(git diff --cached --name-only)
foreach ($file in $staged) {
    if ($file -match '\.zip$') {
        git restore --staged -- "$file" 2>$null
        if ($LASTEXITCODE -ne 0) {
            git reset -- "$file" 2>$null | Out-Null
        }
        Write-Warn2 "已取消暂存：$file"
    }
}

# 取消暂存明显的凭据文件。
$staged = @(git diff --cached --name-only)
$blocked = @()
foreach ($file in $staged) {
    $name = [System.IO.Path]::GetFileName($file)
    if (
        $name -eq ".env" -or
        $name -like ".env.*" -or
        $name -like "*.pem" -or
        $name -like "*.pfx" -or
        $name -like "*.p12" -or
        $name -like "*.key"
    ) {
        $blocked += $file
        git restore --staged -- "$file" 2>$null
        if ($LASTEXITCODE -ne 0) {
            git reset -- "$file" 2>$null | Out-Null
        }
    }
}

if ($blocked.Count -gt 0) {
    Write-Warn2 "以下敏感文件已自动取消暂存："
    $blocked | ForEach-Object { Write-Host "  - $_" }
}

Write-Step "显示即将提交的文件"
$staged = @(git diff --cached --name-only)
if ($staged.Count -eq 0) {
    Write-Warn2 "当前没有新的文件需要提交。"
} else {
    $staged | ForEach-Object { Write-Host "  + $_" }
}

if ([string]::IsNullOrWhiteSpace($CommitMessage)) {
    $CommitMessage = "AGMP $Version"
}

if ($staged.Count -gt 0) {
    Write-Step "创建提交：$CommitMessage"

    # 如果 Git 身份未配置，给出清晰提示。
    $userName = git config user.name
    $userEmail = git config user.email

    if ([string]::IsNullOrWhiteSpace($userName) -or [string]::IsNullOrWhiteSpace($userEmail)) {
        Write-Warn2 "Git 尚未配置 user.name / user.email。"
        Write-Host ""
        Write-Host "请先执行，例如：" -ForegroundColor Yellow
        Write-Host '  git config --global user.name "你的 GitHub 用户名"'
        Write-Host '  git config --global user.email "你的 GitHub 邮箱"'
        Write-Host ""
        Fail "配置 Git 身份后重新运行本脚本。"
    }

    git commit -m $CommitMessage
    if ($LASTEXITCODE -ne 0) { Fail "git commit 失败。" }

    Write-Ok "提交完成。"
}

Write-Step "当前状态"
git status --short | Out-Host

if ($NoPush) {
    Write-Warn2 "已指定 -NoPush，本次不推送。"
    exit 0
}

Write-Step "推送到 GitHub"
Write-Host "如果 GitHub 要求登录，请按照 Git Credential Manager / 浏览器提示完成授权。" -ForegroundColor DarkGray

git push -u origin $Branch
if ($LASTEXITCODE -ne 0) {
    Write-Host ""
    Write-Warn2 "推送失败。常见原因："
    Write-Host "  1. 尚未登录 GitHub / Git Credential Manager"
    Write-Host "  2. GitHub 账号没有该仓库写入权限"
    Write-Host "  3. 网络或代理阻止 GitHub"
    Write-Host "  4. 远程 main 已有提交，需要先拉取并处理历史"
    Fail "git push 未成功。"
}

Write-Host ""
Write-Host "============================================================" -ForegroundColor DarkGray
Write-Host "  推送完成。" -ForegroundColor Green
Write-Host "  https://github.com/yubboo/AI-Game-Manager-Panel" -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor DarkGray
