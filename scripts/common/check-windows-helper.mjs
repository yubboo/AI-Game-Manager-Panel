import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'

const root = process.cwd()
const failures = []
const fail = (message) => failures.push(message)

function walk(dir, predicate = () => true) {
  const result = []
  if (!fs.existsSync(dir)) return result
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name)
    if (entry.isDirectory()) result.push(...walk(full, predicate))
    else if (predicate(full)) result.push(full)
  }
  return result
}

const required = [
  'AI-Game-Manager-Panel.bat',
  'AGMP-GitHub.bat',
  'push-agmp.ps1',
  'AGMP-Sync.bat',
  'sync-agmp.ps1',
  'scripts/windows/AIGameManagerPanel.ps1',
  'scripts/windows/Test-WindowsHelper.ps1',
  'scripts/windows/lib/Common.ps1',
  'scripts/windows/lib/Toolchain.ps1',
  'scripts/windows/lib/Rust.ps1',
  'scripts/windows/lib/Dependencies.ps1',
  'scripts/windows/lib/Checks.ps1',
  'scripts/windows/lib/Wails.ps1',
  'scripts/windows/lib/Electron.ps1',
  'scripts/windows/tasks/Tasks.ps1',
  'scripts/common/check-distribution-boundary.mjs',
]
for (const rel of required) if (!fs.existsSync(path.join(root, rel))) fail(`缺少 Windows Helper 文件：${rel}`)

try {
  const bat = fs.readFileSync(path.join(root, 'AI-Game-Manager-Panel.bat'))
  if ([...bat].some(value => value > 0x7f)) fail('根 AI-Game-Manager-Panel.bat 必须保持 ASCII-safe，禁止直接写中文。')
  const text = bat.toString('ascii')
  if (!text.includes('chcp 65001')) fail('根 AI-Game-Manager-Panel.bat 必须先切换 UTF-8 code page。')
  if (!text.includes('scripts\\windows\\AIGameManagerPanel.ps1')) fail('根 AI-Game-Manager-Panel.bat 必须只委托 scripts/windows/AIGameManagerPanel.ps1。')
  if (!text.includes('%*')) fail('根 BAT 必须把 -Task/-SelfTest 等快捷参数原样转发给 PowerShell Helper。')
  const withoutCrlf = text.replace(/\r\n/g, '')
  if (withoutCrlf.includes('\n') || withoutCrlf.includes('\r')) fail('根 AI-Game-Manager-Panel.bat 必须使用 CRLF。')
} catch (error) {
  fail(`无法读取 AI-Game-Manager-Panel.bat：${error instanceof Error ? error.message : String(error)}`)
}


try {
  const githubBat = fs.readFileSync(path.join(root, 'AGMP-GitHub.bat'))
  if ([...githubBat].some(value => value > 0x7f)) fail('AGMP-GitHub.bat 必须保持纯 ASCII，所有中文菜单必须由 PowerShell 输出。')
  if (githubBat.length >= 3 && githubBat[0] === 0xef && githubBat[1] === 0xbb && githubBat[2] === 0xbf) fail('AGMP-GitHub.bat 禁止 UTF-8 BOM；否则 CMD 会出现 锘緻echo。')
  const githubBatText = githubBat.toString('ascii')
  if (!githubBatText.includes('%~dp0push-agmp.ps1')) fail('AGMP-GitHub.bat 必须通过脚本自身目录定位 push-agmp.ps1。')
  if (!githubBatText.includes('-ExecutionPolicy Bypass')) fail('AGMP-GitHub.bat 必须兼容本地 PowerShell ExecutionPolicy。')
  if (!githubBatText.includes('set "RC=%ERRORLEVEL%"') || !githubBatText.includes('pause')) fail('AGMP-GitHub.bat 必须在 PowerShell 异常退出时保留窗口，禁止错误一闪而过。')
  const githubBatWithoutCrlf = githubBatText.replace(/\r\n/g, '')
  if (githubBatWithoutCrlf.includes('\n') || githubBatWithoutCrlf.includes('\r')) fail('AGMP-GitHub.bat 必须使用 CRLF。')

  const pushPs1 = fs.readFileSync(path.join(root, 'push-agmp.ps1'))
  if (pushPs1.length < 3 || pushPs1[0] !== 0xef || pushPs1[1] !== 0xbb || pushPs1[2] !== 0xbf) fail('push-agmp.ps1 必须 UTF-8 BOM，兼容 Windows PowerShell 5.1。')
  const pushText = pushPs1.toString('utf8').replace(/^\uFEFF/, '')
  if (pushText.includes('\uFFFD')) fail('push-agmp.ps1 包含乱码 replacement char。')
  const pushWithoutCrlf = pushText.replace(/\r\n/g, '')
  if (pushWithoutCrlf.includes('\n') || pushWithoutCrlf.includes('\r')) fail('push-agmp.ps1 必须使用 CRLF。')
  for (const token of ['Assert-SourceNotIgnored', 'Test-RepositorySafety', "Path = 'rust/crates'; MinimumFiles = 5", 'pull --rebase', 'git push']) {
    if (!pushText.includes(token)) fail(`push-agmp.ps1 缺少 GitHub 工作台关键能力：${token}`)
  }
} catch (error) {
  fail(`无法验证 GitHub 推送工作台编码：${error instanceof Error ? error.message : String(error)}`)
}

try {
  const syncBat = fs.readFileSync(path.join(root, 'AGMP-Sync.bat'))
  if ([...syncBat].some(value => value > 0x7f)) fail('AGMP-Sync.bat 必须保持纯 ASCII。')
  if (syncBat.length >= 3 && syncBat[0] === 0xef && syncBat[1] === 0xbb && syncBat[2] === 0xbf) fail('AGMP-Sync.bat 禁止 UTF-8 BOM。')
  const syncBatText = syncBat.toString('ascii')
  if (!syncBatText.includes('%~dp0sync-agmp.ps1')) fail('AGMP-Sync.bat 必须通过脚本自身目录定位 sync-agmp.ps1。')
  if (!syncBatText.includes('set "RC=%ERRORLEVEL%"') || !syncBatText.includes('pause')) fail('AGMP-Sync.bat 必须保留同步结果窗口。')
  const syncBatWithoutCrlf = syncBatText.replace(/\r\n/g, '')
  if (syncBatWithoutCrlf.includes('\n') || syncBatWithoutCrlf.includes('\r')) fail('AGMP-Sync.bat 必须使用 CRLF。')

  const syncPs1 = fs.readFileSync(path.join(root, 'sync-agmp.ps1'))
  if (syncPs1.length < 3 || syncPs1[0] !== 0xef || syncPs1[1] !== 0xbb || syncPs1[2] !== 0xbf) fail('sync-agmp.ps1 必须 UTF-8 BOM。')
  const syncText = syncPs1.toString('utf8').replace(/^\uFEFF/, '')
  const syncWithoutCrlf = syncText.replace(/\r\n/g, '')
  if (syncWithoutCrlf.includes('\n') || syncWithoutCrlf.includes('\r')) fail('sync-agmp.ps1 必须使用 CRLF。')
  if (!syncText.includes('robocopy.exe') || !syncText.includes("git.exe -C $Destination ls-files")) fail('sync-agmp.ps1 缺少稳定源码同步/旧跟踪文件清理能力。')
} catch (error) {
  fail(`无法验证源码同步助手编码：${error instanceof Error ? error.message : String(error)}`)
}

const windowsRoot = path.join(root, 'scripts/windows')
const batFiles = walk(windowsRoot, file => file.toLowerCase().endsWith('.bat'))
if (batFiles.length) fail(`scripts/windows 禁止残留 BAT；发现：${batFiles.map(file => path.relative(root, file)).join(', ')}`)

for (const file of walk(windowsRoot, file => file.toLowerCase().endsWith('.ps1'))) {
  const bytes = fs.readFileSync(file)
  const rel = path.relative(root, file)
  if (bytes.length < 3 || bytes[0] !== 0xef || bytes[1] !== 0xbb || bytes[2] !== 0xbf) fail(`PS1 必须 UTF-8 BOM：${rel}`)
  const text = bytes.toString('utf8').replace(/^\uFEFF/, '')
  if (text.includes('\uFFFD')) fail(`PS1 包含乱码 replacement char：${rel}`)
  const withoutCrlf = text.replace(/\r\n/g, '')
  if (withoutCrlf.includes('\n') || withoutCrlf.includes('\r')) fail(`PS1 必须使用 CRLF：${rel}`)
  if (!rel.endsWith('Test-WindowsHelper.ps1')) {
    if (/\bInvoke-Expression\b/i.test(text)) fail(`禁止 Invoke-Expression：${rel}`)
    if (/--prefer-online/i.test(text)) fail(`禁止未知 pnpm 参数 --prefer-online：${rel}`)
    if (/shell\s*:\s*true/i.test(text)) fail(`禁止 shell:true：${rel}`)
  }
  if (text.includes('\u0000')) fail(`脚本包含 NUL：${rel}`)
}

// 0.1.70：许可证发行工具同样面向 Windows PowerShell 5.1，必须遵守 UTF-8 BOM + CRLF。
const licenseToolsRoot = path.join(root, 'scripts/tools/license')
for (const file of walk(licenseToolsRoot, file => file.toLowerCase().endsWith('.ps1'))) {
  const bytes = fs.readFileSync(file)
  const rel = path.relative(root, file)
  if (bytes.length < 3 || bytes[0] !== 0xef || bytes[1] !== 0xbb || bytes[2] !== 0xbf) fail(`许可证 PS1 必须 UTF-8 BOM：${rel}`)
  const text = bytes.toString('utf8').replace(/^\uFEFF/, '')
  if (text.includes('\uFFFD')) fail(`许可证 PS1 包含乱码 replacement char：${rel}`)
  const withoutCrlf = text.replace(/\r\n/g, '')
  if (withoutCrlf.includes('\n') || withoutCrlf.includes('\r')) fail(`许可证 PS1 必须使用 CRLF：${rel}`)
  if (/\bInvoke-Expression\b/i.test(text)) fail(`许可证脚本禁止 Invoke-Expression：${rel}`)
  if (text.includes('\u0000')) fail(`许可证脚本包含 NUL：${rel}`)
}

try {
  const menu = fs.readFileSync(path.join(root, 'scripts/windows/AIGameManagerPanel.ps1'), 'utf8')
  for (const label of [
    '1. 初始化 / 修复基础开发环境', '2. 开发模式', '3. 项目检查', '4. Wails Windows 候选构建',
    '5. Electron Windows 候选构建', '6. Linux Server 候选构建', '7. 预览 / 启动构建产物',
    '8. 状态、诊断与修复', '9. 清理与重置', '10. 正式发布',
  ]) if (!menu.includes(label)) fail(`主菜单缺少：${label}`)
  for (let n = 1; n <= 10; n++) if (!menu.includes(`${n}{Invoke-`)) fail(`菜单 ${n} 缺少任务映射。`)
  if (!menu.includes('Invoke-HelperSelfTest')) fail('缺少菜单 1-10 Dry-Run 自检入口。')
  if (!menu.includes('普通用户不要运行本脚本') || !menu.includes('无需 Rust/MSVC/Go/Node')) fail('0.1.79 开发助手必须明确开发/普通用户边界。')
  if (!menu.includes('菜单 4/5/6 是候选构建') || !menu.includes('只有菜单 10 正式发布')) fail('0.1.95 必须明确候选构建与正式发布边界。')
} catch (error) {
  fail(`无法验证主菜单：${error instanceof Error ? error.message : String(error)}`)
}

// Resolve all relative imports in frontend TS/Vue script files. This catches missing views before vue-tsc.
const frontendRoot = path.join(root, 'frontend/src')
const sourceFiles = walk(frontendRoot, file => /\.(?:ts|tsx|vue|js|mjs)$/.test(file))
const importPattern = /(?:from\s+|import\s*\(\s*)['"](\.{1,2}\/[^'"]+)['"]/g
const extensions = ['', '.ts', '.tsx', '.vue', '.js', '.mjs', '.json']
for (const file of sourceFiles) {
  const text = fs.readFileSync(file, 'utf8')
  for (const match of text.matchAll(importPattern)) {
    const spec = match[1]
    const base = path.resolve(path.dirname(file), spec)
    const candidates = [
      ...extensions.map(ext => base + ext),
      ...extensions.filter(Boolean).map(ext => path.join(base, 'index' + ext)),
    ]
    if (!candidates.some(candidate => fs.existsSync(candidate))) {
      fail(`Frontend 相对导入不存在：${path.relative(root, file)} -> ${spec}`)
    }
  }
}



// 0.1.64: PowerShell automatic/read-only variable collision and desktop release regression gates.
try {
  const protectedVariablePattern = /^\s*\$(home|host|pid|psversiontable|psscriptroot|pscommandpath)\s*=/gim
  for (const script of walk(windowsRoot, file => file.toLowerCase().endsWith('.ps1'))) {
    const source = fs.readFileSync(script, 'utf8').replace(/^\uFEFF/, '')
    const collisions = [...source.matchAll(protectedVariablePattern)].map(match => match[1])
    if (collisions.length) fail(`PowerShell 自动/只读变量发生可写赋值冲突：${path.relative(root, script)} -> ${[...new Set(collisions)].join(', ')}`)
  }

  const toolchain = fs.readFileSync(path.join(root, 'scripts/windows/lib/Toolchain.ps1'), 'utf8')
  const showStart = toolchain.indexOf('function Show-AGMPToolchain')
  const showEnd = toolchain.indexOf('function Invoke-GoModules')
  const showBlock = showStart >= 0 && showEnd > showStart ? toolchain.slice(showStart, showEnd) : ''
  if (!showBlock) fail('无法定位 Show-AGMPToolchain。')
  if (/return\s+\$tools\b/i.test(showBlock) || /Write-Output\s+\$tools\b/i.test(showBlock)) fail('Show-AGMPToolchain 禁止把工具链对象泄漏到控制台。')
  if (!showBlock.includes("Write-Info '环境'")) fail('Show-AGMPToolchain 必须使用中文 [环境] 输出。')
} catch (error) {
  fail(`无法验证 PowerShell 0.1.64 回归门禁：${error instanceof Error ? error.message : String(error)}`)
}

try {
  const common = fs.readFileSync(path.join(root, 'scripts/windows/lib/Common.ps1'), 'utf8').replace(/^\uFEFF/, '')
  const toolchain = fs.readFileSync(path.join(root, 'scripts/windows/lib/Toolchain.ps1'), 'utf8').replace(/^\uFEFF/, '')
  const rust = fs.readFileSync(path.join(root, 'scripts/windows/lib/Rust.ps1'), 'utf8').replace(/^\uFEFF/, '')
  const checks = fs.readFileSync(path.join(root, 'scripts/windows/lib/Checks.ps1'), 'utf8').replace(/^\uFEFF/, '')
  const wails = fs.readFileSync(path.join(root, 'scripts/windows/lib/Wails.ps1'), 'utf8').replace(/^\uFEFF/, '')
  const electron = fs.readFileSync(path.join(root, 'scripts/windows/lib/Electron.ps1'), 'utf8').replace(/^\uFEFF/, '')
  const tasks = fs.readFileSync(path.join(root, 'scripts/windows/tasks/Tasks.ps1'), 'utf8').replace(/^\uFEFF/, '')
  const syncKey = fs.readFileSync(path.join(root, 'scripts/tools/license/sync-release-public-key.ps1'), 'utf8').replace(/^\uFEFF/, '')
  const wailsBridge = fs.readFileSync(path.join(root, 'internal/bridge/wails/app.go'), 'utf8')

  if (!common.includes('function Ensure-Directory') || !common.includes('if ($script:AGMPDryRun) { return }')) fail('Dry-Run 自检禁止创建目录或写入构建工作区。')
  for (const fn of ['Install-AGMPWailsCli', 'Assert-AGMPWailsCli', 'Assert-InnoCompiler']) {
    if (!toolchain.includes(`function ${fn}`)) fail(`Windows Helper 缺少工具链边界函数：${fn}`)
  }
  for (const fn of ['Assert-AGMPRustToolchain', 'Assert-AGMPMSVCBuildTools', 'Assert-AGMPRustWindowsToolchain']) {
    if (!rust.includes(`function ${fn}`)) fail(`Windows Helper 缺少 Rust/MSVC 只检查函数：${fn}`)
  }
  if (rust.includes('Rustlang.Rustup') || /winget[^\n]*Rust/i.test(rust)) fail('Rust 工具链准备禁止使用 winget；必须使用官方 rustup-init 并在当前控制台内执行。')
  if (!rust.includes('static.rust-lang.org/rustup/dist/x86_64-pc-windows-msvc/rustup-init.exe') || !rust.includes('Test-AGMPSHA256File')) fail('Rust bootstrap 必须使用官方 rustup-init + SHA256 校验。')
  const buildCoreStart = rust.indexOf('function Build-AGMPXiaoYuCore')
  const agentDevStart = rust.indexOf('function Invoke-RustAgentDevelopment')
  const buildCore = buildCoreStart >= 0 && agentDevStart > buildCoreStart ? rust.slice(buildCoreStart, agentDevStart) : ''
  if (!buildCore.includes('Assert-AGMPRustWindowsToolchain') || buildCore.includes('Ensure-AGMPRustWindowsToolchain')) fail('Build-AGMPXiaoYuCore 必须只检查 Rust/MSVC，禁止构建中途自动安装。')
  if (!buildCore.includes('if($script:AGMPDryRun)') || !buildCore.includes('预期小鱼核心 Runtime')) fail('Build-AGMPXiaoYuCore 的 Dry-Run 必须在真实产物校验前返回，菜单自检不得依赖已存在的 xiaoyu.exe。')
  if (!checks.includes('Invoke-CoreCandidateGate') || !checks.includes('Invoke-CoreReleaseGate')) fail('缺少 Candidate / Release 两套 Core Gate。')
  if (!tasks.includes('Invoke-WailsRelease -OfficialPublish') || !tasks.includes('Invoke-ElectronRelease -OfficialPublish') || !tasks.includes('Invoke-LinuxServerRelease -OfficialPublish')) fail('菜单 10 Dry-Run/正式发布必须覆盖 Wails、Electron、Linux OfficialPublish。')
  if (!wails.includes('Invoke-CoreCandidateGate') || !wails.includes('Invoke-CoreReleaseGate')) fail('Wails 构建必须区分候选与正式 Gate。')
  if (!tasks.includes("$channel=if($OfficialPublish){'release'}else{'candidate'}")) fail('Linux 构建必须区分 candidate/release 输出 Channel。')
  if (!electron.includes('Invoke-CoreCandidateGate') || !electron.includes('Invoke-CoreReleaseGate')) fail('Electron 构建必须区分候选与正式 Gate。')
  if (!syncKey.includes("internal\\system\\license\\vendor_public_keys.json") || syncKey.includes("internal\\service\\license")) fail('发行公钥同步目标必须是 internal/system/license/vendor_public_keys.json。')
  if (/\bdstruntime\./.test(wailsBridge)) fail('Wails Bridge 残留未定义 dstruntime 别名；DST Runtime 必须统一使用 dstruntimecore。')
  if (/\btoken\.(?:Status|SaveRequest|ImportRequest)/.test(wailsBridge)) fail('Wails Bridge 的 token 参数遮蔽了 DST token 包；必须使用 dsttoken 别名。')
  if (!wails.includes("@('test','-tags','agmp_dev_license','./internal/bridge/wails')")) fail('Wails 开发启动前必须先做 Bridge Go 编译预检。')
} catch (error) {
  fail(`无法验证 0.1.95 Candidate/Release/Toolchain 边界：${error instanceof Error ? error.message : String(error)}`)
}

try {
  const electronMain = fs.readFileSync(path.join(root, 'desktop/electron/src/main.ts'), 'utf8')
  const builder = fs.readFileSync(path.join(root, 'desktop/electron/electron-builder.yml'), 'utf8')
  const deps = fs.readFileSync(path.join(root, 'scripts/windows/lib/Dependencies.ps1'), 'utf8')
  for (const label of ['文件', '编辑', '视图', '窗口', '帮助', '撤销', '重做', '剪切', '复制', '粘贴', '全选']) {
    if (!electronMain.includes(`label: '${label}'`)) fail(`Electron 原生菜单缺少中文标签：${label}`)
  }
  if (electronMain.includes("shell.openExternal('https://github.com/')")) fail('Electron 帮助菜单禁止跳转占位 GitHub 地址。')
  if (!builder.includes('electronDist: node_modules/electron/dist')) fail('Electron Builder 必须复用本地 node_modules/electron/dist。')
  if (!deps.includes('Assert-ElectronPackagingRuntime')) fail('Electron Release 缺少本地 Runtime/electronDist 前置验证。')
} catch (error) {
  fail(`无法验证 Electron 0.1.64 回归门禁：${error instanceof Error ? error.message : String(error)}`)
}

try {
  const ui = JSON.parse(fs.readFileSync(path.join(root, 'configs/ui.json'), 'utf8'))
  const navItems = (ui.navigation ?? []).flatMap(group => group.items ?? [])
  if (navItems.some(item => item.id === 'terminal' || item.to === '/terminal')) fail('终端入口必须从左侧导航移到 Bottom Panel。')
  for (const rel of ['frontend/src/app/layout/AppRightSidebar.vue', 'frontend/src/app/layout/BottomPanel.vue']) {
    if (!fs.existsSync(path.join(root, rel))) fail(`缺少三栏工作台组件：${rel}`)
  }
  const appVue = fs.readFileSync(path.join(root, 'frontend/src/app/App.vue'), 'utf8')
  for (const token of ['pane-resizer--left', 'pane-resizer--right', 'pane-resizer--bottom', 'AppRightSidebar', 'BottomPanel']) {
    if (!appVue.includes(token)) fail(`三栏/底部工作台缺少：${token}`)
  }
} catch (error) {
  fail(`无法验证工作台 0.1.64 回归门禁：${error instanceof Error ? error.message : String(error)}`)
}

if (failures.length) {
  console.error('[ERROR] Windows Helper / 编码 / Frontend Import Gate 失败：')
  for (const failure of failures) console.error(`  - ${failure}`)
  process.exit(1)
}
console.log('[OK] Windows Helper、UTF-8 编码、菜单 1-10 与 Frontend Import Gate 通过。')
