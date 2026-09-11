import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'

const root = process.cwd()
const failures = []
const read = rel => fs.readFileSync(path.join(root, rel), 'utf8').replace(/^\uFEFF/, '')
const requireFile = rel => {
  if (!fs.existsSync(path.join(root, rel))) failures.push(`缺少小鱼核心文件：${rel}`)
}
const requireText = (label, text, token) => {
  if (!text.includes(token)) failures.push(`${label} 缺少：${token}`)
}

const required = [
  'rust/Cargo.toml',
  'rust/rust-toolchain.toml',
  'rust/README.md',
  'rust/crates/xiaoyu-protocol/Cargo.toml',
  'rust/crates/xiaoyu-protocol/src/lib.rs',
  'rust/crates/xiaoyu-core/Cargo.toml',
  'rust/crates/xiaoyu-core/src/lib.rs',
  'rust/crates/xiaoyu-core/Cargo.toml',
  'rust/crates/xiaoyu-core/src/bin/xiaoyu.rs',
  'internal/xiaoyu/runtime/service.go',
  'internal/platform/runtime/run.go',
  'internal/platform/runtime/output.go',
  'internal/platform/runtime/shell.go',
  'internal/ops/files/service.go',
  'internal/app/app_xiaoyu_tools.go',
  'internal/platform/runtime/process_windows.go',
  'internal/xiaoyu/runtime/service_test.go',
  'internal/xiaoyu/contract/model.go',
  'internal/xiaoyu/contract/tool.go',
  'internal/xiaoyu/contract/registry.go',
  'internal/xiaoyu/control/permission.go',
  'internal/xiaoyu/control/approval_store.go',
  'scripts/windows/lib/Rust.ps1',
]
required.forEach(requireFile)

try {
  const workspace = read('rust/Cargo.toml')
  const protocol = read('rust/crates/xiaoyu-protocol/src/lib.rs')
  const runtime = read('rust/crates/xiaoyu-core/src/lib.rs')
  const cli = read('rust/crates/xiaoyu-core/src/bin/xiaoyu.rs')
  const goBridge = read('internal/xiaoyu/runtime/service.go')
  const sharedWindows = read('internal/platform/runtime/process_windows.go')
  const sharedRun = read('internal/platform/runtime/run.go')
  const app = read('internal/app/app_xiaoyu.go')
  const appTools = read('internal/app/app_xiaoyu_tools.go')
  const goToolRegistry = read('internal/xiaoyu/contract/registry.go')
  const workspaceFiles = read('internal/ops/files/service.go')
  const sharedOutput = read('internal/platform/runtime/output.go')
  const rustHelper = read('scripts/windows/lib/Rust.ps1')
  const tasks = read('scripts/windows/tasks/Tasks.ps1')
  const wails = read('scripts/windows/lib/Wails.ps1')
  const electron = read('scripts/windows/lib/Electron.ps1')
  const installer = read('distribution/installer/windows/AIGameManagerPanel.iss')
  const aiView = read('frontend/src/features/xiaoyu/AIWorkbenchView.vue')
  const terminal = read('frontend/src/features/terminal/TerminalPanel.vue')
  const aiConfig = JSON.parse(read('configs/ai.json'))
  const workflow = read('.github/workflows/safety.yml')

  requireText('Rust workspace', workspace, 'xiaoyu-protocol')
  requireText('Rust workspace', workspace, 'xiaoyu-core')
  requireText('Rust workspace', workspace, 'xiaoyu-core')
  requireText('Agent protocol', protocol, 'xiaoyu.v1')
  requireText('Agent protocol', protocol, '#[serde(rename_all = "camelCase")]')
  for (const token of ['RiskLevel', 'ApprovalMode', 'ApprovalDecision', 'ToolSpec', 'JsonRpcRequest', 'JsonRpcResponse']) requireText('Agent protocol', protocol, token)
  for (const token of ['brain-only-boundary', 'host-tool-contracts', 'risk-aware-planning', 'session-foundation']) requireText('小鱼核心', runtime, token)
  for (const forbidden of ['std::process', 'std::fs', 'Command::new', 'process.run', 'fs.read', 'fs.list']) {
    if (runtime.includes(forbidden)) failures.push(`小鱼 Brain 禁止直接拥有 OS/领域执行能力：${forbidden}`)
  }
  for (const token of ['Doctor', 'Tools', 'Rpc', 'tools/list', 'session/create', 'policy/preview']) requireText('XiaoYu CLI', cli, token)
  for (const forbidden of ['Command::Tool {', 'Command::Exec {', 'tool/call']) {
    if (cli.includes(forbidden)) failures.push(`XiaoYu CLI 禁止恢复本地执行入口：${forbidden}`)
  }
  for (const token of ['system.info', 'fs.list', 'fs.read', 'fs.write', 'fs.replace', 'shell.exec', 'process.run', 'RegisterTool', 'runApprovedShell']) requireText('AGMP Host Tool Registry', appTools, token)
  for (const token of ['ErrToolExists', 'ToolHandler', 'Execute(', 'MustRegisterHandler']) requireText('Go Tool Registry', goToolRegistry, token)
  for (const token of ['ErrOutsideWorkspace', 'EvalSymlinks', 'DefaultReadLimit']) requireText('Workspace Files', workspaceFiles, token)
  for (const token of ['Sequence', 'Subscribe', 'History']) requireText('Shared Runtime Output', sharedOutput, token)
  requireText('Go Agent bridge', goBridge, 'ProtocolVersion = "xiaoyu.v1"')
  requireText('Shared Runtime Windows', sharedWindows, 'createNoWindow')
  requireText('Shared Runtime one-shot', sharedRun, 'func Run(')
  requireText('Go Agent bridge', goBridge, 'platformruntime.Run')
  requireText('Application', app, 'XiaoYuRunCommand')
  requireText('Application', app, 'XiaoYuCallTool')
  requireText('Application Tool Dispatch', app, 'a.xiaoyuTools.Execute')
  requireText('Application', app, 'requireXiaoYuMember(token)')
  for (const token of [
    'Ensure-AGMPMSVCBuildTools',
        'Microsoft.VisualStudio.Component.VC.Tools.x86.x64',
    'Microsoft.VisualStudio.Component.Windows11SDK.22621',
    'Import-AGMPVisualStudioEnvironment',
    'Get-AGMPMSVCBootstrapper',
    'Get-AGMPWindowsSDKLib',
    'kernel32.lib',
    'Read-AGMPMSVCStoragePlan',
    'Get-AGMPDriveFreeSpaceGB',
    'PackageCache',
    'SharedPath',
    'vs_BuildTools-2022.exe',
    'Get-AuthenticodeSignature',
    'link.exe',
    '2.3 GB ~ 60 GB',
    '至少预留 8 GB',
  ]) requireText('Windows Rust Toolchain', rustHelper, token)
  if (/['"]--includeRecommended['"]/.test(rustHelper)) failures.push('Windows Rust Toolchain 禁止把 --includeRecommended 作为安装参数。')
  if (/['"]Microsoft\.VisualStudio\.Workload\.VCTools['"]/.test(rustHelper)) failures.push('Windows Rust Toolchain 禁止安装完整 VCTools workload；Rust 官方最小前置只需要 MSVC x64/x86 + Windows SDK。')
  const initStart = tasks.indexOf('function Invoke-InitializeProject')
  const initEnd = tasks.indexOf('function Invoke-DevelopmentMenu', initStart)
  const initBlock = initStart >= 0 && initEnd > initStart ? tasks.slice(initStart, initEnd) : ''
  if (!initBlock) failures.push('无法定位基础开发环境初始化函数。')
  if (initBlock.includes('Ensure-AGMPRustToolchain') || initBlock.includes('Ensure-AGMPMSVCBuildTools')) failures.push('0.1.79 基础开发环境禁止强制安装 Rust/MSVC。')
  requireText('Windows 开发菜单', tasks, '小鱼核心 CLI            Rust Runtime / Doctor / Tool Registry')
  requireText('Wails Build', wails, 'Assert-AGMPRustWindowsToolchain')
  requireText('Wails Build', wails, 'Build-AGMPXiaoYuCore')
  requireText('Wails Build', wails, 'AI-Game-Manager-XiaoYu.exe')
  requireText('Electron Release', electron, 'Build-AGMPXiaoYuCore')
  requireText('Installer', installer, 'AI-Game-Manager-XiaoYu.exe')
  requireText('小鱼工作台', aiView, 'XIAOYU · AGMP INTELLIGENCE CORE')
  requireText('小鱼工作台', aiView, '小鱼是主入口')
  requireText('Terminal', terminal, 'XiaoYu Control Terminal')

  if (aiConfig.primaryInteraction !== 'agent') failures.push('configs/ai.json primaryInteraction 必须为 agent')
  if (aiConfig.runtime?.engine !== 'rust') failures.push('configs/ai.json runtime.engine 必须为 rust')
  if (aiConfig.runtime?.protocol !== 'xiaoyu.v1') failures.push('configs/ai.json runtime.protocol 必须为 xiaoyu.v1')
  if (aiConfig.runtime?.transport !== 'json-rpc-stdio') failures.push('configs/ai.json runtime.transport 必须为 json-rpc-stdio')
  if (aiConfig.allowManualFallback !== true) failures.push('Agent-first 仍必须保留手动兜底')
  requireText('GitHub Actions', workflow, 'cargo check --manifest-path rust/Cargo.toml --workspace')
  requireText('GitHub Actions', workflow, 'cargo test --manifest-path rust/Cargo.toml --workspace')
} catch (error) {
  failures.push(`小鱼核心 Gate 检查失败：${error instanceof Error ? error.message : String(error)}`)
}

if (failures.length) {
  console.error('AGMP 小鱼核心 Gate FAIL')
  failures.forEach(item => console.error(` - ${item}`))
  process.exit(1)
}
console.log('AGMP 小鱼核心 Gate PASS (Rust Brain-only · Host Tool Registry · Shared Runtime · Approval)')
