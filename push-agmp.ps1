[CmdletBinding()]
param(
    [ValidateSet('menu','push','status','pull','commit','scan','log','open')]
    [string]$Action = 'menu',
    [string]$ProjectRoot = '',
    [string]$RepoUrl = 'https://github.com/yubboo/AI-Game-Manager-Panel.git',
    [string]$Branch = 'main',
    [string]$CommitMessage = '',
    [switch]$AllowMassDeletion
)

$ErrorActionPreference = 'Stop'

# Windows PowerShell 5.1 / PowerShell 7 都统一使用 UTF-8 控制台输出。
try {
    $utf8 = New-Object System.Text.UTF8Encoding($false)
    [Console]::InputEncoding = $utf8
    [Console]::OutputEncoding = $utf8
    $OutputEncoding = $utf8
} catch {
}

function Write-Title([string]$Text) {
    Write-Host ''
    Write-Host '====================================================================' -ForegroundColor DarkGray
    Write-Host ('  ' + $Text) -ForegroundColor White
    Write-Host '====================================================================' -ForegroundColor DarkGray
}

function Write-Step([string]$Text) { Write-Host ('[进行] ' + $Text) -ForegroundColor Cyan }
function Write-Ok([string]$Text) { Write-Host ('[完成] ' + $Text) -ForegroundColor Green }
function Write-Warn2([string]$Text) { Write-Host ('[注意] ' + $Text) -ForegroundColor Yellow }
function Stop-Fail([string]$Text) { Write-Host ('[失败] ' + $Text) -ForegroundColor Red; exit 1 }

function Invoke-Git([string[]]$Arguments, [switch]$AllowFailure) {
    & git @Arguments
    $code = $LASTEXITCODE
    if (-not $AllowFailure -and $code -ne 0) {
        Stop-Fail ('Git 命令失败：git ' + ($Arguments -join ' '))
    }
    return $code
}

if ([string]::IsNullOrWhiteSpace($ProjectRoot)) {
    $ProjectRoot = Split-Path -Parent $PSCommandPath
}
$ProjectRoot = [System.IO.Path]::GetFullPath($ProjectRoot)

if (-not (Test-Path -LiteralPath $ProjectRoot -PathType Container)) {
    Stop-Fail ('项目目录不存在：' + $ProjectRoot)
}
Set-Location -LiteralPath $ProjectRoot

if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
    Stop-Fail '未找到 Git。请先安装 Git for Windows。'
}

function Get-AGMPVersion {
    $packagePath = Join-Path $ProjectRoot 'frontend\package.json'
    if (Test-Path -LiteralPath $packagePath) {
        try {
            $pkg = Get-Content -LiteralPath $packagePath -Raw -Encoding UTF8 | ConvertFrom-Json
            if ($pkg.version) { return [string]$pkg.version }
        } catch {
        }
    }
    return 'unknown'
}

function Initialize-Repository {
    if (-not (Test-Path -LiteralPath (Join-Path $ProjectRoot '.git'))) {
        Write-Step '初始化 Git 仓库'
        Invoke-Git @('init') | Out-Null
    }

    Invoke-Git @('branch','-M',$Branch) | Out-Null

    $origin = & git remote get-url origin 2>$null
    if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($origin)) {
        Invoke-Git @('remote','add','origin',$RepoUrl) | Out-Null
        Write-Ok ('已设置 origin：' + $RepoUrl)
    } elseif ($origin.Trim() -ne $RepoUrl) {
        Write-Warn2 ('origin 原地址：' + $origin.Trim())
        Invoke-Git @('remote','set-url','origin',$RepoUrl) | Out-Null
        Write-Ok ('origin 已更新：' + $RepoUrl)
    }
}


function Assert-ProjectIntegrity {
    Write-Step '项目完整性检查'

    $requiredFiles = @(
        '.github/workflows/safety.yml',
        'go.mod',
        'go.sum',
        'main.go',
        'wails.json',
        'frontend/package.json',
        'frontend/src/app/router.ts',
        'frontend/src/features/xiaoyu/AIWorkbenchView.vue',
        'internal/xiaoyu/host/loop.go',
        'internal/xiaoyu/host/brain_model.go',
        'internal/app/app_xiaoyu_tools.go',
        'runtime/README.md',
        'rust/Cargo.toml',
        'rust/crates/xiaoyu-core/Cargo.toml',
        'rust/crates/xiaoyu-core/src/lib.rs',
        'rust/crates/xiaoyu-protocol/Cargo.toml',
        'scripts/common/check-github-safety.mjs',
        'scripts/common/check-duplicates.mjs',
        'scripts/common/check-source-tree.mjs',
        'scripts/common/check-naming.mjs',
        'scripts/common/check-project-layout.mjs',
        'scripts/common/check-xiaoyu-harness.mjs',
        'scripts/common/check-xiaoyu-agent-runtime.mjs',
        'scripts/windows/AIGameManagerPanel.ps1',
        'scripts/windows/tasks/Tasks.ps1',
        'docs/NAMING-CONVENTIONS.md',
        'AGMP-Sync.bat',
        'sync-agmp.ps1'
    )

    $missing = New-Object System.Collections.Generic.List[string]
    foreach ($rel in $requiredFiles) {
        $full = Join-Path $ProjectRoot ($rel -replace '/', '\')
        if (-not (Test-Path -LiteralPath $full -PathType Leaf)) {
            $missing.Add($rel)
        }
    }

    $requiredTrees = @(
        @{ Path = 'scripts/common'; MinimumFiles = 16 },
        @{ Path = 'scripts/windows'; MinimumFiles = 8 },
        @{ Path = 'rust/crates'; MinimumFiles = 5 },
        @{ Path = 'internal/xiaoyu'; MinimumFiles = 20 },
        @{ Path = 'frontend/src'; MinimumFiles = 25 }
    )

    foreach ($tree in $requiredTrees) {
        $full = Join-Path $ProjectRoot ($tree.Path -replace '/', '\')
        if (-not (Test-Path -LiteralPath $full -PathType Container)) {
            $missing.Add($tree.Path + '/ (目录缺失)')
            continue
        }
        $count = @(Get-ChildItem -LiteralPath $full -File -Recurse -ErrorAction SilentlyContinue).Count
        if ($count -lt [int]$tree.MinimumFiles) {
            $missing.Add(('{0}/ (文件数量异常：{1} < {2})' -f $tree.Path, $count, $tree.MinimumFiles))
        }
    }

    if ($missing.Count -gt 0) {
        Write-Host ''
        Write-Host '检测到源码工作副本不完整。本次禁止 Commit/Push：' -ForegroundColor Red
        foreach ($item in $missing) { Write-Host ('  - ' + $item) -ForegroundColor Red }
        Write-Host ''
        Write-Host '请重新完整解压/复制 AGMP 源码包到当前目录，并保留 .git 目录，然后重试。' -ForegroundColor Yellow
        Stop-Fail '项目完整性检查失败。'
    }

    Write-Ok '项目完整性检查通过。'
}

function Test-StagedDeletionSafety {
    $rows = @(& git diff --cached --name-status --diff-filter=D)
    if ($rows.Count -eq 0) { return }

    $criticalDeleted = New-Object System.Collections.Generic.List[string]
    $nonHistoryDeleted = New-Object System.Collections.Generic.List[string]

    foreach ($row in $rows) {
        if ([string]::IsNullOrWhiteSpace($row)) { continue }
        $parts = $row -split "`t", 2
        if ($parts.Count -lt 2) { continue }
        $file = $parts[1].Replace('\','/')

        $isLegacyHistory = (
            $file -match '^AGMP-\d+\.\d+\.\d+-(?:Release-Notes|Validation)\.md$' -or
            $file -match '^docs/(?:releases|prompts)/' -or
            $file -match '^docs/development/(?:VALIDATION|COMPLETION)-.*\.md$' -or
            $file -eq 'docs/BUILD-HOTFIX-0.1.37.md'
        )
        if ($isLegacyHistory) { continue }

        $nonHistoryDeleted.Add($file)
        if ($file -match '^(?:scripts/|rust/|\.github/|internal/xiaoyu/|frontend/src/|cmd/)') {
            $criticalDeleted.Add($file)
        }
    }

    if ($criticalDeleted.Count -gt 0) {
        Write-Host ''
        Write-Host '发现关键源码删除，默认拒绝推送：' -ForegroundColor Red
        $criticalDeleted | Select-Object -First 25 | ForEach-Object { Write-Host ('  - ' + $_) -ForegroundColor Red }
        if ($criticalDeleted.Count -gt 25) { Write-Host ('  ... 另有 ' + ($criticalDeleted.Count - 25) + ' 个') -ForegroundColor Red }
        Stop-Fail '关键源码删除保护已触发。若这是经过确认的大型重构，请先人工检查，不要直接一键推送。'
    }

    if ($nonHistoryDeleted.Count -gt 25 -and -not $AllowMassDeletion) {
        Write-Host ''
        Write-Host ('检测到 {0} 个非历史文件将被删除。' -f $nonHistoryDeleted.Count) -ForegroundColor Red
        $nonHistoryDeleted | Select-Object -First 25 | ForEach-Object { Write-Host ('  - ' + $_) -ForegroundColor Red }
        Write-Host ''
        Write-Host '为防止“覆盖源码时漏目录”导致整批源码被删，一键推送默认停止。' -ForegroundColor Yellow
        Write-Host '只有确认这是有意的大规模删除时，才可手动使用 -AllowMassDeletion。' -ForegroundColor Yellow
        Stop-Fail '大规模删除保护已触发。'
    }
}

function Remove-LegacyHistoryFiles {
    $history = Join-Path $ProjectRoot 'docs\PROJECT-HISTORY.md'
    if (-not (Test-Path -LiteralPath $history -PathType Leaf)) { return }

    $targets = New-Object System.Collections.Generic.List[string]
    Get-ChildItem -LiteralPath $ProjectRoot -File -Filter 'AGMP-*.md' -ErrorAction SilentlyContinue | ForEach-Object {
        if ($_.Name -match '^AGMP-\d+\.\d+\.\d+-(?:Release-Notes|Validation)\.md$') { $targets.Add($_.FullName) }
    }
    $dev = Join-Path $ProjectRoot 'docs\development'
    if (Test-Path -LiteralPath $dev) {
        Get-ChildItem -LiteralPath $dev -File -ErrorAction SilentlyContinue | ForEach-Object {
            if ($_.Name -match '^(?:VALIDATION|COMPLETION)-.*\.md$') { $targets.Add($_.FullName) }
        }
    }
    foreach ($dir in @('docs\releases','docs\prompts')) {
        $full = Join-Path $ProjectRoot $dir
        if (Test-Path -LiteralPath $full -PathType Container) { $targets.Add($full) }
    }
    $hotfix = Join-Path $ProjectRoot 'docs\BUILD-HOTFIX-0.1.37.md'
    if (Test-Path -LiteralPath $hotfix -PathType Leaf) { $targets.Add($hotfix) }

    if ($targets.Count -gt 0) {
        Write-Step '清理已经合并到 PROJECT-HISTORY.md 的旧历史文件'
        foreach ($target in ($targets | Select-Object -Unique)) {
            Remove-Item -LiteralPath $target -Recurse -Force
            Write-Host ('  - 删除旧历史：' + $target.Substring($ProjectRoot.Length).TrimStart('\')) -ForegroundColor DarkGray
        }
    }
}

function Remove-LegacyRenamedFiles {
    # Known historical paths that were renamed to shorter canonical names.
    # Leaving both old and new Go test files can redeclare the same functions.
    $legacy = @(
        'internal\bridge\httpapi\server_xiaoyu_models_test.go',
        'internal\bridge\httpapi\server_xiaoyu_models_release_test.go'
    )
    $removed = $false
    foreach ($rel in $legacy) {
        $full = Join-Path $ProjectRoot $rel
        if (Test-Path -LiteralPath $full -PathType Leaf) {
            if (-not $removed) {
                Write-Step '清理已重命名的旧源码文件'
                $removed = $true
            }
            Remove-Item -LiteralPath $full -Force
            Write-Host ('  - 删除旧路径：' + $rel) -ForegroundColor DarkGray
        }
    }
}

function Assert-SourceNotIgnored {
    # 这些目录名称同时也是运行数据名称，最容易被错误的 .gitignore 规则误伤。
    $critical = @(
        'internal/ops/logs/service.go',
        'frontend/src/features/logs/LogsView.vue',
        'frontend/src/features/logs/useLogHub.ts',
        'frontend/src/features/instances/InstancesView.vue'
    )
    foreach ($rel in $critical) {
        $full = Join-Path $ProjectRoot ($rel -replace '/', '\')
        if (-not (Test-Path -LiteralPath $full -PathType Leaf)) {
            Stop-Fail ('关键源码文件缺失：' + $rel)
        }
        & git check-ignore -q --no-index -- $rel
        if ($LASTEXITCODE -eq 0) {
            Stop-Fail ('.gitignore 错误忽略了源码：' + $rel + '。运行数据目录规则必须使用 /logs/、/instances/ 等根目录锚点。')
        }
    }
    Write-Ok '关键源码未被 .gitignore 误忽略。'
}

function Get-SafetyFiles([ValidateSet('tracked','staged','candidate')] [string]$Mode) {
    if ($Mode -eq 'staged') {
        return @(& git diff --cached --name-only --diff-filter=ACMR)
    }
    if ($Mode -eq 'candidate') {
        return @(& git ls-files --cached --others --exclude-standard)
    }
    return @(& git ls-files)
}

function Test-RepositorySafety([ValidateSet('tracked','staged','candidate')] [string]$Mode) {
    Write-Step ('安全检查：' + $Mode)
    Assert-ProjectIntegrity
    Assert-SourceNotIgnored

    $files = @(Get-SafetyFiles $Mode)
    $failures = New-Object System.Collections.Generic.List[string]
    $warnings = New-Object System.Collections.Generic.List[string]

    $forbiddenRoot = '^(?:data|log|logs|backups|instances|temp|exports|plugins|cache|build)/'
    $forbiddenExt = '\.(?:exe|test|zip|7z|rar|dmp|stackdump|tmp|bak|key|priv|seed|pem|p12|pfx|jks|keystore|cdk|lic|license|bflc)$'
    $forbiddenNames = '(^|/)(?:activation\.json|install\.id|device\.id|accounts\.json|bootstrap\.lock|credentials\.json|secrets\.json|cluster_token\.txt|server_token\.txt|adminlist\.txt|whitelist\.txt|blocklist\.txt)$'

    $secretPatterns = @(
        '(?<![A-Za-z0-9])sk-(?:proj-|ant-)?[A-Za-z0-9_-]{20,}',
        'github_pat_[A-Za-z0-9_]{20,}',
        'gh[pousr]_[A-Za-z0-9]{30,}',
        'AIza[0-9A-Za-z_-]{30,}',
        'xox[baprs]-[0-9A-Za-z-]{20,}',
        'AKIA[0-9A-Z]{16}',
        '-----BEGIN(?: RSA| EC| OPENSSH)? PRIVATE KEY-----'
    )
    $textExtensions = @('.go','.rs','.ts','.tsx','.js','.mjs','.cjs','.vue','.json','.yaml','.yml','.toml','.md','.txt','.ps1','.bat','.cmd','.sh','.html','.css','.ini','.conf','.xml','.iss')

    foreach ($file in $files) {
        if ([string]::IsNullOrWhiteSpace($file)) { continue }
        $rel = $file.Replace('\','/')
        $lower = $rel.ToLowerInvariant()

        if ($lower -eq '.env.example') { continue }
        if ($lower -eq '.env' -or $lower.StartsWith('.env.') -or $lower -match '(^|/)\.env($|\.)') {
            $failures.Add('环境变量文件禁止提交：' + $file); continue
        }
        if ($lower -match $forbiddenRoot) { $failures.Add('运行/构建数据禁止提交：' + $file); continue }
        if ($lower.StartsWith('runtime/') -and $lower -ne 'runtime/readme.md') { $failures.Add('runtime 运行数据禁止提交：' + $file); continue }
        if ($lower -match '(^|/)(?:node_modules|target|\.pnpm-store)/') { $failures.Add('依赖/缓存目录禁止提交：' + $file); continue }
        if ($lower -match $forbiddenExt) { $failures.Add('构建产物/密钥材料禁止提交：' + $file); continue }
        if ($lower -match $forbiddenNames) { $failures.Add('本机凭据/游戏私密状态禁止提交：' + $file); continue }

        $full = Join-Path $ProjectRoot ($file -replace '/', '\')
        if (-not (Test-Path -LiteralPath $full -PathType Leaf)) { continue }
        $item = Get-Item -LiteralPath $full
        if ($item.Length -ge 95MB) {
            $failures.Add(('文件超过/接近 GitHub 单文件限制：{0} ({1:N1} MB)' -f $file, ($item.Length / 1MB)))
            continue
        } elseif ($item.Length -ge 20MB) {
            $warnings.Add(('较大的仓库文件：{0} ({1:N1} MB)' -f $file, ($item.Length / 1MB)))
        }

        $ext = [System.IO.Path]::GetExtension($full).ToLowerInvariant()
        if ($textExtensions -contains $ext -and $item.Length -le 5MB) {
            try {
                $content = Get-Content -LiteralPath $full -Raw -Encoding UTF8
                foreach ($pattern in $secretPatterns) {
                    if ($content -match $pattern) {
                        $failures.Add('发现疑似真实凭据：' + $file)
                        break
                    }
                }
            } catch {
            }
        }
    }

    foreach ($warning in ($warnings | Select-Object -Unique)) { Write-Warn2 $warning }

    if (Get-Command node -ErrorAction SilentlyContinue) {
        $nodeGates = @(
            @{ Path = 'scripts\common\check-source-tree.mjs'; Name = 'Source Tree Gate' },
            @{ Path = 'scripts\common\check-naming.mjs'; Name = 'Naming Gate' },
            @{ Path = 'scripts\common\check-duplicates.mjs'; Name = 'Duplicate Source Gate' },
            @{ Path = 'scripts\common\check-github-safety.mjs'; Name = 'GitHub Safety Gate' }
        )
        foreach ($item in $nodeGates) {
            $gate = Join-Path $ProjectRoot $item.Path
            if (Test-Path -LiteralPath $gate) {
                & node $gate
                if ($LASTEXITCODE -ne 0) { $failures.Add(('项目 ' + $item.Name + ' 未通过。')) }
            }
        }
    }

    $uniqueFailures = @($failures | Select-Object -Unique)
    if ($uniqueFailures.Count -gt 0) {
        Write-Host ''
        foreach ($failure in $uniqueFailures) { Write-Host ('  - ' + $failure) -ForegroundColor Red }
        Stop-Fail '安全检查未通过，本次不会提交/推送。'
    }
    Write-Ok '安全检查通过。'
}

function Sync-Remote {
    Write-Step '同步 GitHub 远端'
    Invoke-Git @('fetch','origin',$Branch) | Out-Null
    & git show-ref --verify --quiet ('refs/remotes/origin/' + $Branch)
    if ($LASTEXITCODE -eq 0) {
        & git pull --rebase --autostash origin $Branch
        if ($LASTEXITCODE -ne 0) {
            Stop-Fail 'Pull/Rebase 发生冲突。脚本不会 force push；请解决冲突后重试。'
        }
    } else {
        Write-Warn2 ('远端 origin/' + $Branch + ' 尚不存在，跳过 Pull。')
    }
    Write-Ok '远端同步完成。'
}

function Stage-And-Commit([string]$Message) {
    Remove-LegacyRenamedFiles
    Remove-LegacyHistoryFiles
    Test-RepositorySafety 'candidate'
    Write-Step '暂存源码'
    Invoke-Git @('add','-A') | Out-Null
    Test-StagedDeletionSafety
    Test-RepositorySafety 'staged'

    $staged = @(& git diff --cached --name-only)
    if ($staged.Count -eq 0) {
        Write-Warn2 '没有新的项目变更需要提交。'
        return $false
    }

    Write-Host ('即将提交 {0} 个文件：' -f $staged.Count) -ForegroundColor Gray
    $staged | Select-Object -First 40 | ForEach-Object { Write-Host ('  + ' + $_) }
    if ($staged.Count -gt 40) { Write-Host ('  ... 另有 {0} 个文件' -f ($staged.Count - 40)) -ForegroundColor DarkGray }

    $name = & git config user.name
    $mail = & git config user.email
    if ([string]::IsNullOrWhiteSpace($name) -or [string]::IsNullOrWhiteSpace($mail)) {
        Stop-Fail 'Git user.name / user.email 尚未配置。'
    }

    if ([string]::IsNullOrWhiteSpace($Message)) {
        $version = Get-AGMPVersion
        if ($version -eq 'unknown') { $Message = 'AGMP 更新' } else { $Message = 'AGMP ' + $version + ' 更新' }
    }

    Write-Step ('创建提交：' + $Message)
    & git commit -m $Message
    if ($LASTEXITCODE -ne 0) { Stop-Fail 'git commit 失败。' }
    Write-Ok '提交完成。'
    return $true
}

function Show-Status {
    Write-Title 'AGMP GitHub 状态'
    Write-Host ('项目目录：' + $ProjectRoot)
    Write-Host ('项目版本：' + (Get-AGMPVersion))
    Write-Host ('远程仓库：' + $RepoUrl)
    Write-Host ''
    & git status -sb
    Write-Host ''
    Write-Host '最近提交：' -ForegroundColor Gray
    & git log --oneline --decorate -5
}

function Invoke-Action([string]$Name, [string]$Message = '') {
    switch ($Name) {
        'status' { Show-Status; return }
        'pull' { Sync-Remote; Show-Status; return }
        'scan' { Test-RepositorySafety 'tracked'; return }
        'log' { Write-Title 'AGMP 最近 Git 提交'; & git log --oneline --decorate --graph -15; return }
        'open' { Start-Process 'https://github.com/yubboo/AI-Game-Manager-Panel'; return }
        'commit' { [void](Stage-And-Commit $Message); Show-Status; return }
        'push' {
            Sync-Remote
            [void](Stage-And-Commit $Message)
            Test-RepositorySafety 'tracked'
            Write-Step '推送到 GitHub'
            & git push -u origin $Branch
            if ($LASTEXITCODE -ne 0) { Stop-Fail 'git push 失败。脚本不会 force push。' }
            Write-Title '[一键推送] 完成'
            Write-Host 'https://github.com/yubboo/AI-Game-Manager-Panel' -ForegroundColor Cyan
            return
        }
    }
}

function Show-Menu {
    while ($true) {
        Clear-Host
        Write-Title 'AGMP GitHub 一键推送助手'
        Write-Host ('项目：' + $ProjectRoot)
        Write-Host ('版本：' + (Get-AGMPVersion))
        Write-Host ''
        Write-Host '  1. [一键推送]  同步远端 > 安全检查 > 提交 > Push'
        Write-Host '  2. [查看状态]  查看修改、分支与最近提交'
        Write-Host '  3. [同步远端]  Fetch + Pull --rebase --autostash'
        Write-Host '  4. [仅提交]    安全检查 + Commit，不 Push'
        Write-Host '  5. [安全检查]  检查源码完整性、误删除、敏感文件、私钥、Token、大文件'
        Write-Host '  6. [提交历史]  查看最近 Git 提交'
        Write-Host '  7. [打开仓库]  打开 GitHub 项目页面'
        Write-Host '  8. [自定义推送] 输入本次 Commit Message 后推送'
        Write-Host ''
        Write-Host '  0. 退出'
        Write-Host ''
        Write-Host '原则：除敏感、本机运行数据、缓存和构建产物外，项目源码/公开配置/文档/CI/锁文件全部推送。' -ForegroundColor DarkGray
        $choice = Read-Host '请选择 [0-8]'
        try {
            switch ($choice) {
                '1' { Invoke-Action 'push' }
                '2' { Invoke-Action 'status' }
                '3' { Invoke-Action 'pull' }
                '4' { Invoke-Action 'commit' }
                '5' { Invoke-Action 'scan' }
                '6' { Invoke-Action 'log' }
                '7' { Invoke-Action 'open' }
                '8' { $msg = Read-Host '请输入 Commit Message'; Invoke-Action 'push' $msg }
                '0' { return }
                default { Write-Warn2 '无效选项。' }
            }
        } catch {
            Write-Host ('[异常] ' + $_.Exception.Message) -ForegroundColor Red
        }
        Write-Host ''
        Read-Host '按 Enter 返回菜单' | Out-Null
    }
}

Initialize-Repository
Assert-ProjectIntegrity
if ($Action -eq 'menu') { Show-Menu } else { Invoke-Action $Action $CommitMessage }
