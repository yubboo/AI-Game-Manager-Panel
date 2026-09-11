function Get-AGMPCargo {
    return Get-AGMPToolPath @('cargo.exe','cargo')
}

function Get-AGMPRustc {
    return Get-AGMPToolPath @('rustc.exe','rustc')
}

function Assert-AGMPRustToolchain {
    if ($script:AGMPDryRun) { return @{Cargo='cargo';Rustc='rustc'} }
    $cargo=Get-AGMPCargo
    $rustc=Get-AGMPRustc
    if(-not $cargo -or -not $rustc){
        throw 'Rust/Cargo 尚未准备。请在菜单 8“状态、诊断与修复”中显式准备 Rust/MSVC；开发/构建菜单不会自动安装工具链。'
    }
    return @{Cargo=$cargo;Rustc=$rustc}
}

function Refresh-AGMPProcessPath {
    $machine=[Environment]::GetEnvironmentVariable('Path','Machine')
    $user=[Environment]::GetEnvironmentVariable('Path','User')
    $env:Path=(@($machine,$user) -join ';')
}

function Test-AGMPSHA256File {
    param([Parameter(Mandatory=$true)][string]$File,[Parameter(Mandatory=$true)][string]$HashFile)
    if (-not (Test-Path -LiteralPath $File -PathType Leaf) -or -not (Test-Path -LiteralPath $HashFile -PathType Leaf)) { return $false }
    try {
        $text=[IO.File]::ReadAllText($HashFile).Trim()
        $match=[regex]::Match($text,'(?i)\b[0-9a-f]{64}\b')
        if(-not $match.Success){return $false}
        $expected=$match.Value.ToLowerInvariant()
        $actual=(Get-FileHash -LiteralPath $File -Algorithm SHA256).Hash.ToLowerInvariant()
        return $actual -eq $expected
    }catch{return $false}
}

function Get-AGMPRustupBootstrapper {
    $cache=Get-AGMPInstallerCache
    $exe=Join-Path $cache 'rustup-init-x86_64-pc-windows-msvc.exe'
    $sha=Join-Path $cache 'rustup-init-x86_64-pc-windows-msvc.exe.sha256'
    if(Test-AGMPSHA256File -File $exe -HashFile $sha){
        Write-Info '缓存' "复用已验证的 rustup-init：$exe"
        return $exe
    }
    Remove-Item -LiteralPath $exe,$sha -Force -ErrorAction SilentlyContinue
    $base='https://static.rust-lang.org/rustup/dist/x86_64-pc-windows-msvc/rustup-init.exe'
    Write-Info 'Rust' '使用 Rust 官方 rustup-init（minimal profile），不调用 winget，不开启第二个安装控制台。'
    try {
        Invoke-WebRequest -Uri $base -OutFile $exe -UseBasicParsing
        Invoke-WebRequest -Uri ($base + '.sha256') -OutFile $sha -UseBasicParsing
    }catch{
        $curl=Get-AGMPToolPath @('curl.exe','curl')
        if(-not $curl){throw "Rust 官方 rustup-init 下载失败：$($_.Exception.Message)"}
        Invoke-AGMPNative -FilePath $curl -Arguments @('-fL','--retry','3','--connect-timeout','20','-o',$exe,$base)
        Invoke-AGMPNative -FilePath $curl -Arguments @('-fL','--retry','3','--connect-timeout','20','-o',$sha,($base + '.sha256'))
    }
    if(-not (Test-AGMPSHA256File -File $exe -HashFile $sha)){
        Remove-Item -LiteralPath $exe,$sha -Force -ErrorAction SilentlyContinue
        throw 'rustup-init SHA256 校验失败，已拒绝执行并删除下载文件。'
    }
    Write-Ok "Rust 官方 rustup-init 下载并校验完成：$exe"
    return $exe
}

function Ensure-AGMPRustToolchain {
    $cargo=Get-AGMPCargo
    $rustc=Get-AGMPRustc
    if($cargo -and $rustc){ return @{Cargo=$cargo;Rustc=$rustc} }
    if($script:AGMPDryRun){ return @{Cargo='cargo';Rustc='rustc'} }

    Write-Warn 'Rust/Cargo 尚未就绪。这里只在你显式选择“菜单 8 → 准备 Rust + MSVC”时安装；普通开发菜单和构建菜单不会偷偷安装。'
    $answer=Read-Host '是否使用 Rust 官方 rustup-init 安装 stable minimal？[Y/N]'
    if($answer -notmatch '^(?i:y|yes)$'){ throw '已取消 Rust 工具链准备。' }
    $rustup=Get-AGMPRustupBootstrapper
    Invoke-AGMPNative -FilePath $rustup -Arguments @('-y','--profile','minimal','--default-host','x86_64-pc-windows-msvc','--default-toolchain','stable')
    Refresh-AGMPProcessPath
    $cargo=Get-AGMPCargo
    $rustc=Get-AGMPRustc
    if(-not $cargo -or -not $rustc){ throw 'rustup-init 完成后仍未找到 cargo/rustc。请关闭并重新打开 AI-Game-Manager-Panel.bat 后重试。' }
    return @{Cargo=$cargo;Rustc=$rustc}
}

function Show-AGMPRustToolchain {
    $cargo=Get-AGMPCargo
    $rustc=Get-AGMPRustc
    if($script:AGMPDryRun){ Write-Info '环境' 'Rust：DRY-RUN'; return }
    if(-not $cargo -or -not $rustc){ Write-Info '可选工具链' 'Rust/Cargo：未安装（仅 小鱼核心 的 Rust 开发/源码发布需要；普通用户无需）'; return }
    $rustVer=(& $rustc --version 2>&1 | Out-String).Trim()
    $cargoVer=(& $cargo --version 2>&1 | Out-String).Trim()
    Write-Info '环境' "Rust：$rustVer"
    Write-Info '环境' "Cargo：$cargoVer"
}


function Get-AGMPVsWhere {
    $candidates = New-Object Collections.Generic.List[string]
    $cmd = Get-Command 'vswhere.exe' -ErrorAction SilentlyContinue
    if ($cmd -and $cmd.Source) { $candidates.Add($cmd.Source) }
    $pf86 = [Environment]::GetEnvironmentVariable('ProgramFiles(x86)')
    $pf = [Environment]::GetEnvironmentVariable('ProgramFiles')
    foreach ($candidate in @(
        $(if($pf86){ Join-Path $pf86 'Microsoft Visual Studio\Installer\vswhere.exe' }),
        $(if($pf){ Join-Path $pf 'Microsoft Visual Studio\Installer\vswhere.exe' })
    )) {
        if ($candidate) { $candidates.Add($candidate) }
    }
    foreach ($candidate in $candidates) {
        if ($candidate -and (Test-Path -LiteralPath $candidate -PathType Leaf)) {
            return (Resolve-Path -LiteralPath $candidate).Path
        }
    }
    return $null
}

function Get-AGMPVisualStudioInstallPath {
    $vswhere = Get-AGMPVsWhere
    if (-not $vswhere) { return $null }
    $path = (& $vswhere -latest -products * -requires Microsoft.VisualStudio.Component.VC.Tools.x86.x64 -property installationPath 2>$null | Out-String).Trim()
    if ([string]::IsNullOrWhiteSpace($path)) { return $null }
    return $path
}

function Import-AGMPBatchEnvironment {
    param(
      [Parameter(Mandatory=$true)][string]$BatchFile,
      [string]$Arguments=''
    )
    if (-not (Test-Path -LiteralPath $BatchFile -PathType Leaf)) { return $false }
    $cmd = Get-AGMPToolPath @('cmd.exe','cmd')
    if (-not $cmd) { return $false }
    $command = '"' + $BatchFile + '" ' + $Arguments + ' >nul && set'
    $lines = & $cmd /d /s /c $command 2>$null
    if ($LASTEXITCODE -ne 0) { return $false }
    foreach ($line in $lines) {
        if (-not $line -or $line.StartsWith('=')) { continue }
        $idx = $line.IndexOf('=')
        if ($idx -le 0) { continue }
        $name = $line.Substring(0,$idx)
        $value = $line.Substring($idx+1)
        [Environment]::SetEnvironmentVariable($name,$value,'Process')
    }
    return $true
}

function Import-AGMPVisualStudioEnvironment {
    $install = Get-AGMPVisualStudioInstallPath
    if (-not $install) { return $false }
    $vsDevCmd = Join-Path $install 'Common7\Tools\VsDevCmd.bat'
    if (Test-Path -LiteralPath $vsDevCmd -PathType Leaf) {
        if (Import-AGMPBatchEnvironment -BatchFile $vsDevCmd -Arguments '-arch=x64 -host_arch=x64') { return $true }
    }
    $vcvars = Join-Path $install 'VC\Auxiliary\Build\vcvars64.bat'
    if (Test-Path -LiteralPath $vcvars -PathType Leaf) {
        if (Import-AGMPBatchEnvironment -BatchFile $vcvars) { return $true }
    }
    return $false
}

function Get-AGMPMSVCLinker {
    return Get-AGMPToolPath @('link.exe')
}

function Get-AGMPWindowsSDKLib {
    $roots = New-Object Collections.Generic.List[string]
    if ($env:WindowsSdkDir) { $roots.Add((Join-Path $env:WindowsSdkDir 'Lib')) }
    $pf86 = [Environment]::GetEnvironmentVariable('ProgramFiles(x86)')
    if ($pf86) { $roots.Add((Join-Path $pf86 'Windows Kits\10\Lib')) }
    foreach ($root in $roots) {
        if (-not (Test-Path -LiteralPath $root -PathType Container)) { continue }
        $match = Get-ChildItem -LiteralPath $root -Filter kernel32.lib -File -Recurse -ErrorAction SilentlyContinue |
            Where-Object { $_.FullName -match '(?i)\\um\\x64\\kernel32\.lib$' } |
            Sort-Object FullName -Descending |
            Select-Object -First 1
        if ($match) { return $match.FullName }
    }
    return $null
}

function Get-AGMPInstallerCache {
    param([string]$BaseDirectory='')
    if ($BaseDirectory) {
        $dir = Join-Path $BaseDirectory 'Installer'
    } else {
        $dir = Join-Path (Get-LocalAppDataRoot) 'AI-Game-Manager-Panel\DevTools\Installers'
    }
    Ensure-Directory $dir
    return $dir
}

function Get-AGMPDriveFreeSpaceGB {
    param([Parameter(Mandatory=$true)][string]$Path)
    try {
        $full = [System.IO.Path]::GetFullPath($Path)
        $root = [System.IO.Path]::GetPathRoot($full)
        if (-not $root) { return $null }
        $driveName = $root.TrimEnd('\\').TrimEnd(':')
        $drive = Get-PSDrive -Name $driveName -ErrorAction SilentlyContinue
        if (-not $drive) { return $null }
        return [Math]::Round(($drive.Free / 1GB), 1)
    } catch { return $null }
}

function Get-AGMPMSVCStoragePreferencePath {
    $dir = Join-Path (Get-LocalAppDataRoot) 'AI-Game-Manager-Panel\DevTools'
    Ensure-Directory $dir
    return (Join-Path $dir 'msvc-storage-root.txt')
}

function Get-AGMPSavedMSVCStorageRoot {
    $path = Get-AGMPMSVCStoragePreferencePath
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) { return $null }
    try {
        $value = ([IO.File]::ReadAllText($path)).Trim()
        if ($value -and [System.IO.Path]::IsPathRooted($value)) { return $value }
    } catch {}
    return $null
}

function Save-AGMPMSVCStorageRoot {
    param([Parameter(Mandatory=$true)][string]$Root)
    $path = Get-AGMPMSVCStoragePreferencePath
    [IO.File]::WriteAllText($path, $Root, (New-Object Text.UTF8Encoding($false)))
}

function New-AGMPCustomMSVCStoragePlan {
    param([Parameter(Mandatory=$true)][string]$Root)
    Ensure-Directory $Root
    return @{
        Custom = $true
        Root = $Root
        InstallPath = (Join-Path $Root 'BuildTools')
        PackageCache = (Join-Path $Root 'PackageCache')
        SharedPath = (Join-Path $Root 'Shared')
        BootstrapperCache = (Get-AGMPInstallerCache -BaseDirectory $Root)
    }
}

function Show-AGMPMSVCInstallSummary {
    param([hashtable]$Plan)
    Write-Host ''
    Write-Info 'MSVC' 'AGMP 只安装 Rust Windows 链接所需的精简 C++ 组件：'
    Write-Host '  必须：MSVC x64/x86 编译/链接工具（包含 link.exe）'
    Write-Host '  必须：Windows 11 SDK 10.0.22621'
    Write-Host '  自动依赖：这两个组件自身需要的 Build Tools 基础文件（由微软安装器解析）'
    Write-Host '  不装：完整 C++ Workload、CMake、vcpkg、AddressSanitizer、ATL、MFC、测试适配器、ARM/ARM64 工具链'
    Write-Host ''
    Write-Info '容量' '微软官方对 Build Tools 2022 给出的磁盘范围是 2.3 GB ~ 60 GB（取决于所选组件）。'
    Write-Info '容量' 'Rust 官方的“最小安装”只要求上面两个组件。组件版本会变化，因此微软没有给这两个组件组合一个固定下载 GB 数。'
    Write-Info '估算' '当前精简方案通常仍需数 GB；建议目标盘至少预留 8 GB，系统盘 C: 至少预留 2 GB 给 Windows SDK/共享系统组件。'
    Write-Host ''
    if ($Plan.Custom) {
        Write-Info '位置' "Build Tools：$($Plan.InstallPath)"
        Write-Info '位置' "组件下载缓存：$($Plan.PackageCache)"
        Write-Info '位置' "共享组件：$($Plan.SharedPath)"
        Write-Info '位置' "微软启动安装器缓存：$($Plan.BootstrapperCache)"
        $free = Get-AGMPDriveFreeSpaceGB $Plan.Root
        if ($null -ne $free) { Write-Info '空间' "目标盘剩余：$free GB" }
    } else {
        Write-Info '位置' "Build Tools：Microsoft 默认位置 $($Plan.InstallPath)"
        Write-Info '位置' '组件下载缓存/共享组件：使用微软当前默认位置；已安装过 Visual Studio 时这些全局位置可能已经固定。'
        Write-Info '位置' "微软启动安装器缓存：$($Plan.BootstrapperCache)"
        $free = Get-AGMPDriveFreeSpaceGB $Plan.BootstrapperCache
        if ($null -ne $free) { Write-Info '空间' "系统盘/缓存盘剩余：$free GB" }
    }
    Write-Host ''
}

function Read-AGMPMSVCStoragePlan {
    $defaultBootstrapperCache = Get-AGMPInstallerCache
    $savedRoot = Get-AGMPSavedMSVCStorageRoot
    Write-Host ''
    Write-Host '请选择 Visual Studio Build Tools 存储方式：'
    Write-Host '  1. 使用微软默认安装位置（通常 C:）'
    Write-Host '  2. 自定义目录/磁盘（C: 空间不足时推荐）'
    if ($savedRoot) { Write-Host "  3. 复用上次自定义位置：$savedRoot" }
    Write-Host '  0. 取消'
    $range = if ($savedRoot) { '[0-3]' } else { '[0-2]' }
    $choice = Read-Host "请选择 $range"
    if ($choice -eq '0') { throw '已取消 MSVC Build Tools 准备。' }

    if ($choice -eq '3' -and $savedRoot) {
        $plan = New-AGMPCustomMSVCStoragePlan -Root $savedRoot
        Show-AGMPMSVCInstallSummary $plan
        return $plan
    }

    if ($choice -eq '2') {
        $root = (Read-Host '请输入自定义根目录，例如 D:\AGMP-DevTools\MSVC').Trim().Trim('"')
        if ([string]::IsNullOrWhiteSpace($root) -or -not [System.IO.Path]::IsPathRooted($root)) {
            throw '自定义目录必须是绝对路径，例如 D:\AGMP-DevTools\MSVC。'
        }
        if ($script:AGMPRoot -and $root.StartsWith($script:AGMPRoot,[System.StringComparison]::OrdinalIgnoreCase)) {
            Write-Warn '不建议把 MSVC 工具链安装到 AGMP 源码目录内部；源码清理/换版本时可能误删。'
            $confirm = Read-Host '仍然使用该目录？[Y/N]'
            if ($confirm -notmatch '^(?i:y|yes)$') { return Read-AGMPMSVCStoragePlan }
        }
        $plan = New-AGMPCustomMSVCStoragePlan -Root $root
        Show-AGMPMSVCInstallSummary $plan
        $free = Get-AGMPDriveFreeSpaceGB $root
        if ($null -ne $free -and $free -lt 8) {
            Write-Warn "目标盘只剩 $free GB，低于 AGMP 建议的 8 GB 预留空间。"
            $ok = Read-Host '仍然继续？[Y/N]'
            if ($ok -notmatch '^(?i:y|yes)$') { return Read-AGMPMSVCStoragePlan }
        }
        Save-AGMPMSVCStorageRoot -Root $root
        Write-Info '位置' '已记住该自定义目录；下次初始化可直接选择“复用上次自定义位置”，不会因为换源码版本而丢失下载缓存。'
        return $plan
    }

    $pf86 = [Environment]::GetEnvironmentVariable('ProgramFiles(x86)')
    $defaultInstall = if ($pf86) { Join-Path $pf86 'Microsoft Visual Studio\2022\BuildTools' } else { 'C:\Program Files (x86)\Microsoft Visual Studio\2022\BuildTools' }
    $plan = @{
        Custom = $false
        Root = ''
        InstallPath = $defaultInstall
        PackageCache = ''
        SharedPath = ''
        BootstrapperCache = $defaultBootstrapperCache
    }
    Show-AGMPMSVCInstallSummary $plan
    return $plan
}
function Test-AGMPMicrosoftSignedExecutable {
    param([Parameter(Mandatory=$true)][string]$Path)
    if (-not (Test-FileMinimumSize $Path 262144)) { return $false }
    try {
        $signature = Get-AuthenticodeSignature -FilePath $Path
        if (-not $signature.SignerCertificate) { return $false }
        $subject = [string]$signature.SignerCertificate.Subject
        if ($subject -notmatch '(?i)Microsoft Corporation') { return $false }
        if ($signature.Status -ne [System.Management.Automation.SignatureStatus]::Valid) {
            Write-Warn "安装器存在 Microsoft 签名，但 Windows 当前验证状态为 $($signature.Status)。为避免执行损坏或被替换的文件，将重新下载。"
            return $false
        }
        return $true
    } catch {
        Write-Warn "无法验证安装器签名：$($_.Exception.Message)"
        return $false
    }
}

function Get-AGMPVisualStudioBuildToolsInstallPath {
    $vswhere = Get-AGMPVsWhere
    if (-not $vswhere) { return $null }
    $path = (& $vswhere -latest -products Microsoft.VisualStudio.Product.BuildTools -property installationPath 2>$null | Out-String).Trim()
    if ([string]::IsNullOrWhiteSpace($path)) { return $null }
    return $path
}

function Get-AGMPVisualStudioInstallerSetup {
    $pf86 = [Environment]::GetEnvironmentVariable('ProgramFiles(x86)')
    $pf = [Environment]::GetEnvironmentVariable('ProgramFiles')
    foreach ($candidate in @(
        $(if($pf86){ Join-Path $pf86 'Microsoft Visual Studio\Installer\setup.exe' }),
        $(if($pf){ Join-Path $pf 'Microsoft Visual Studio\Installer\setup.exe' })
    )) {
        if ($candidate -and (Test-Path -LiteralPath $candidate -PathType Leaf)) { return $candidate }
    }
    return $null
}

function Get-AGMPMSVCBootstrapper {
    param([string]$InstallerCache='')
    $cache = if ($InstallerCache) { $InstallerCache } else { Get-AGMPInstallerCache }
    Ensure-Directory $cache
    $cached = Join-Path $cache 'vs_BuildTools-2022.exe'

    if (Test-AGMPMicrosoftSignedExecutable $cached) {
        Write-Info '缓存' "复用已下载的 Visual Studio Build Tools 安装器：$cached"
        return $cached
    }
    if (Test-Path -LiteralPath $cached) {
        Write-Warn '缓存的 Build Tools 安装器无效或签名验证失败，已删除并准备重新下载。'
        Remove-Item -LiteralPath $cached -Force -ErrorAction SilentlyContinue
    }

    $downloads = if ($env:USERPROFILE) { Join-Path $env:USERPROFILE 'Downloads\vs_BuildTools.exe' } else { $null }
    if ($downloads -and (Test-AGMPMicrosoftSignedExecutable $downloads)) {
        Copy-Item -LiteralPath $downloads -Destination $cached -Force
        Write-Info '缓存' "检测到下载目录已有微软 Build Tools 安装器，已复用：$downloads"
        Write-Info '缓存' "已复制到用户选择的 AGMP 缓存：$cached"
        return $cached
    }

    $url = 'https://aka.ms/vs/17/release/vs_buildtools.exe'
    $part = "$cached.part"
    Remove-Item -LiteralPath $part -Force -ErrorAction SilentlyContinue
    Write-Info '下载' '正在获取微软 Visual Studio 2022 Build Tools 启动安装器（微软文档称 bootstrapper 约 1 MB；真正的数 GB 组件由它后续下载）。'
    Write-Info '下载' "来源：$url"
    Write-Info '下载' "保存位置：$cached"

    $downloaded = $false
    try {
        Invoke-WebRequest -Uri $url -OutFile $part -UseBasicParsing
        $downloaded = $true
    } catch {
        Write-Warn "PowerShell 下载失败：$($_.Exception.Message)"
        $curl = Get-AGMPToolPath @('curl.exe','curl')
        if ($curl) {
            Write-Info '下载' '尝试使用 curl 继续下载...'
            Invoke-AGMPNative -FilePath $curl -Arguments @('-fL','--retry','3','--connect-timeout','20','-o',$part,$url)
            $downloaded = $true
        }
    }
    if (-not $downloaded -or -not (Test-FileMinimumSize $part 262144)) {
        Remove-Item -LiteralPath $part -Force -ErrorAction SilentlyContinue
        throw 'Visual Studio Build Tools 启动安装器下载失败。请检查网络后重试；已经成功缓存的安装器下次会直接复用。'
    }
    if (-not (Test-AGMPMicrosoftSignedExecutable $part)) {
        Remove-Item -LiteralPath $part -Force -ErrorAction SilentlyContinue
        throw '下载的 Visual Studio Build Tools 安装器未通过 Microsoft Authenticode 签名验证，已拒绝执行并删除。'
    }
    Move-Item -LiteralPath $part -Destination $cached -Force
    Write-Ok "Build Tools 启动安装器下载并验证完成：$cached"
    return $cached
}

function Invoke-AGMPMSVCWorkloadInstall {
    param([hashtable]$Plan)
    $existingBuildTools = Get-AGMPVisualStudioBuildToolsInstallPath
    $setup = Get-AGMPVisualStudioInstallerSetup
    $minimalComponents = @(
      '--add','Microsoft.VisualStudio.Component.VC.Tools.x86.x64',
      '--add','Microsoft.VisualStudio.Component.Windows11SDK.22621'
    )

    if ($existingBuildTools -and $setup) {
        Write-Info 'MSVC' "检测到现有 Build Tools：$existingBuildTools"
        Write-Info 'MSVC' '只补齐 Rust 官方最小组件：MSVC x64/x86 + Windows 11 SDK 22621。'
        $args = @('modify','--installPath',$existingBuildTools) + $minimalComponents + @('--passive','--norestart')
        Invoke-AGMPNative -FilePath $setup -Arguments $args
        return
    }

    if (-not $Plan) { $Plan = Read-AGMPMSVCStoragePlan }
    $bootstrapper = Get-AGMPMSVCBootstrapper -InstallerCache $Plan.BootstrapperCache
    Write-Info 'MSVC' '正在安装精简 Visual Studio 2022 Build Tools：MSVC x64/x86 + Windows SDK + 必要依赖。'
    Write-Info 'MSVC' '按 Rust 官方最小组件方案安装；不添加完整 C++ Workload，也不添加 Recommended/Optional 全家桶。'
    $args = @('--wait','--passive','--norestart') + $minimalComponents
    if ($Plan.Custom) {
        Ensure-Directory $Plan.InstallPath
        Ensure-Directory $Plan.PackageCache
        Ensure-Directory $Plan.SharedPath
        $args += @('--installPath',$Plan.InstallPath)
        # cache/shared 只能在这台机器第一次初始化 Visual Studio 全局安装路径时设定。
        $anyVs = $null
        $vswhere = Get-AGMPVsWhere
        if ($vswhere) {
            $anyVs = (& $vswhere -latest -products * -property installationPath 2>$null | Out-String).Trim()
        }
        if ([string]::IsNullOrWhiteSpace($anyVs)) {
            $args += @('--path',("cache={0}" -f $Plan.PackageCache),'--path',("shared={0}" -f $Plan.SharedPath))
        } else {
            Write-Warn '检测到这台电脑以前安装过 Visual Studio 产品。Microsoft 的 cache/shared 全局路径可能已锁定，本次仅保证 Build Tools 主安装目录使用你的自定义位置。'
        }
    }
    Invoke-AGMPNative -FilePath $bootstrapper -Arguments $args
}

function Assert-AGMPMSVCBuildTools {
    if ($script:AGMPDryRun) { return 'link.exe' }
    $linker = Get-AGMPMSVCLinker
    $sdkLib = Get-AGMPWindowsSDKLib
    if (-not $linker -or -not $sdkLib) {
        $null = Import-AGMPVisualStudioEnvironment
        $linker = Get-AGMPMSVCLinker
        $sdkLib = Get-AGMPWindowsSDKLib
    }
    if (-not $linker -or -not $sdkLib) {
        throw 'MSVC x64 Linker / Windows SDK 尚未就绪。请在菜单 8“状态、诊断与修复”中显式准备 Rust/MSVC；不会在开发/构建中途重复安装 Visual Studio。'
    }
    return $linker
}

function Ensure-AGMPMSVCBuildTools {
    $linker = Get-AGMPMSVCLinker
    $sdkLib = Get-AGMPWindowsSDKLib
    if ($linker -and $sdkLib) { return $linker }
    if ($script:AGMPDryRun) { return 'link.exe' }

    if (Import-AGMPVisualStudioEnvironment) {
        $linker = Get-AGMPMSVCLinker
        $sdkLib = Get-AGMPWindowsSDKLib
        if ($linker -and $sdkLib) {
            Write-Info 'MSVC' '已检测到现有 MSVC + Windows SDK，仅加载开发者环境，不重复安装。'
            return $linker
        }
    }

    if (-not $linker) { Write-Warn 'Rust 已安装，但缺少 Windows MSVC 链接器 link.exe。' }
    if (-not $sdkLib) { Write-Warn '未检测到可供 x64 Rust 链接使用的 Windows SDK（kernel32.lib）。' }
    Write-Info 'MSVC' 'Rust x86_64-pc-windows-msvc 最小前置：MSVC x64/x86 Build Tools + Windows 11 SDK。'
    $existingBuildTools = Get-AGMPVisualStudioBuildToolsInstallPath
    $plan = $null
    if ($existingBuildTools) {
        Write-Info 'MSVC' "已检测到 Build Tools，但当前缺少/未启用 Rust 所需 C++ 组件：$existingBuildTools"
        Write-Info 'MSVC' '现有 Visual Studio 的安装/缓存位置已经确定，本次只补必需组件，不重复安装整个 Build Tools。'
        $answer = Read-Host '是否只补齐 MSVC x64/x86 + Windows 11 SDK 22621？[Y/N]'
    } else {
        Write-Info 'MSVC' '未检测到可用的 Visual C++ Build Tools。无需 winget，AGMP 可以直接使用微软官方安装器。'
        Write-Host ''
        Write-Info '容量' '微软官方：Build Tools 2022 根据组件不同需要约 2.3 GB ~ 60 GB 磁盘空间。'
        Write-Info 'AGMP' 'Rust 官方最小方案只需两个组件：MSVC x64/x86 + Windows 11 SDK 22621；AGMP 不安装完整 C++ Workload。'
        Write-Info '建议' '请至少准备 8 GB 目标盘空间；即使自定义到其他盘，C: 仍建议保留至少 2 GB 给系统/SDK 共享组件。'
        $answer = Read-Host '是否继续准备精简 MSVC Build Tools？[Y/N]'
        if ($answer -match '^(?i:y|yes)$') { $plan = Read-AGMPMSVCStoragePlan }
    }
    if ($answer -notmatch '^(?i:y|yes)$') {
        throw '已取消 MSVC Build Tools 准备。小鱼核心 的 Rust Runtime 无法在 Windows 上链接生成内部组件。'
    }

    Invoke-AGMPMSVCWorkloadInstall -Plan $plan
    Start-Sleep -Seconds 2
    $null = Import-AGMPVisualStudioEnvironment
    $linker = Get-AGMPMSVCLinker
    $sdkLib = Get-AGMPWindowsSDKLib
    if (-not $linker -or -not $sdkLib) {
        throw 'MSVC 安装/修改已完成，但当前进程仍未同时检测到 link.exe 与 Windows SDK。请重新打开 AI-Game-Manager-Panel.bat 后重试；若仍失败，请在 Visual Studio Installer 的“单个组件”中确认 MSVC x64/x86 与 Windows 11 SDK 22621 已安装。'
    }
    Write-Ok "MSVC 链接器已就绪：$linker"
    Write-Ok "Windows SDK 已就绪：$sdkLib"
    return $linker
}

function Show-AGMPMSVCBuildTools {
    if ($script:AGMPDryRun) { Write-Info '环境' 'MSVC Linker：DRY-RUN'; return }
    $linker = Get-AGMPMSVCLinker
    if (-not $linker) {
        $null = Import-AGMPVisualStudioEnvironment
        $linker = Get-AGMPMSVCLinker
    }
    $sdkLib = Get-AGMPWindowsSDKLib
    if ($linker) { Write-Info '环境' "MSVC Linker：$linker" } else { Write-Warn 'MSVC Linker：未检测到。' }
    if ($sdkLib) { Write-Info '环境' "Windows SDK：$sdkLib" } else { Write-Warn 'Windows SDK：未检测到 x64 kernel32.lib。' }
    $install = Get-AGMPVisualStudioInstallPath
    if ($install) { Write-Info '环境' "Visual Studio Build Tools：$install" }
    if (-not $linker -or -not $sdkLib) { Write-Info '可选工具链' 'MSVC：未就绪（仅 小鱼核心 的 Rust 开发/源码发布需要；普通用户无需）。' }
}

function Assert-AGMPRustWindowsToolchain {
    $tools = Assert-AGMPRustToolchain
    $linker = Assert-AGMPMSVCBuildTools
    return @{Cargo=$tools.Cargo;Rustc=$tools.Rustc;Linker=$linker}
}

function Ensure-AGMPRustWindowsToolchain {
    $tools = Ensure-AGMPRustToolchain
    $linker = Ensure-AGMPMSVCBuildTools
    return @{Cargo=$tools.Cargo;Rustc=$tools.Rustc;Linker=$linker}
}

function Build-AGMPXiaoYuCore {
    param(
      [ValidateSet('debug','release')][string]$Profile='release',
      [string]$Destination=''
    )
    $tools=Assert-AGMPRustWindowsToolchain
    $manifest=Join-Path $script:AGMPRoot 'rust\Cargo.toml'
    $target=Join-Path $script:AGMPRoot 'build\work\rust-target'
    Ensure-Directory $target
    $args=@('build','--manifest-path',$manifest,'-p','xiaoyu-core','--target-dir',$target)
    if($Profile -eq 'release'){ $args += '--release' }
    Write-Info '小鱼核心' "构建 $Profile · xiaoyu.v1"
    Invoke-AGMPNative -FilePath $tools.Cargo -Arguments $args
    $source=Join-Path $target "$Profile\xiaoyu.exe"
    if($script:AGMPDryRun){
      Write-Info 'DRY-RUN' "预期小鱼核心 Runtime：$source"
      if($Destination){ return $Destination }
      return $source
    }
    if(-not (Test-FileMinimumSize $source)){ throw "小鱼核心 Runtime 构建后未找到有效文件：$source" }
    if($Destination){
      Ensure-Directory (Split-Path -Parent $Destination)
      Copy-Item -LiteralPath $source -Destination $Destination -Force
      return $Destination
    }
    return $source
}

function Invoke-RustAgentDevelopment {
    Write-AGMPHeader "AI游戏管理器面板 $script:AGMPVersion - 小鱼核心 / CLI"
    Show-AGMPToolchain
    Show-AGMPRustToolchain
    Show-AGMPMSVCBuildTools
    $agent=Build-AGMPXiaoYuCore -Profile debug
    Write-Info '小鱼核心 Runtime' $agent
    Write-Host ''
    Write-Host '  1. Doctor / Runtime 状态'
    Write-Host '  2. Tool Registry'
    Write-Host '  3. JSON-RPC stdio（高级调试）'
    Write-Host '  0. 返回'
    if($script:AGMPDryRun){ return }
    switch(Read-Host '请选择 [0-3]'){
      '1'{ Invoke-AGMPNative -FilePath $agent -Arguments @('--root',$script:AGMPRoot,'doctor') }
      '2'{ Invoke-AGMPNative -FilePath $agent -Arguments @('--root',$script:AGMPRoot,'tools') }
      '3'{ Write-Info '提示' '每行一个 JSON-RPC 2.0 请求；Ctrl+C 退出。'; & $agent --root $script:AGMPRoot rpc }
      default{return}
    }
}
