import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'

const root = process.cwd()
const failures = []

const readAppSources = () => {
  const dir = path.join(root, 'internal/app')
  return fs.readdirSync(dir)
    .filter(name => /^app(?:_[a-z0-9]+)?\.go$/i.test(name) && !name.endsWith('_test.go'))
    .sort()
    .map(name => fs.readFileSync(path.join(dir, name), 'utf8'))
    .join('\n')
}

// Phase 1 只维护集中式架构文档，不再要求每个目录创建 README。
const requiredPaths = [
  'AGENTS.md',
  'docs/development/PROJECT-RULES.md',
  'docs/PROJECT-ARCHITECTURE.md',
  'docs/DEVELOPMENT-PLAN.md',
  'docs/PROJECT-HISTORY.md',
  'docs/development/RELEASE-KEY-MANAGEMENT.md',
  'configs/app.json',
  'configs/ui.json',
  'configs/ai.json',
  'configs/permissions.json',
  'configs/paths.json',
  'configs/logging.json',
  'configs/games.json',
  'configs/modules.json',
  'configs/release.json',
  'configs/server.json',
  'configs/update.json',
  'internal/config/types.go',
  'internal/config/loader.go',
  'internal/xiaoyu/contract/model.go',
  'internal/xiaoyu/contract/provider.go',
  'internal/xiaoyu/contract/tool.go',
  'internal/xiaoyu/contract/registry.go',
  'internal/xiaoyu/control/permission.go',
  'internal/xiaoyu/control/approval_store.go',
  'internal/xiaoyu/runtime/service.go',
  'internal/xiaoyu/runtime/rpc_worker.go',
  'internal/server/doc.go',
  'internal/ops/doc.go',
  'internal/deploy/doc.go',
  'internal/system/doc.go',
  'internal/platform/files/root.go',
  'internal/platform/net/model.go',
  'internal/platform/http/external_url.go',
  'internal/ops/logs/doc.go',
  'internal/ops/logs/model.go',
  'internal/ops/logs/service.go',
  'internal/ops/logs/catalog.go',
  'internal/ops/logs/read.go',
  'internal/ops/logs/mutation.go',
  'internal/ops/logs/recorder.go',
  'internal/deploy/environment/doc.go',
  'internal/deploy/environment/service.go',
  'internal/deploy/environment/service_test.go',
  'internal/system/auth/doc.go',
  'internal/system/auth/service.go',
  'internal/system/auth/service_test.go',
  'internal/system/license/service.go',
  'internal/system/license/machine_identity_windows.go',
  'internal/system/license/machine_identity_other.go',
  'internal/system/license/service_test.go',
  'internal/system/license/vendor_keys.go',
  'internal/system/license/vendor_public_keys.json',
  'internal/deploy/updater/service.go',
  'internal/deploy/updater/service_test.go',
  'cmd/aigame-manager-license-admin/main.go',
  'configs/license.json',
  'distribution/installer/windows/AIGameManagerPanel.iss',
  'distribution/installer/linux/ai-game-manager-panel',
  'distribution/installer/linux/ai-game-manager-panel-web',
  'distribution/installer/linux/ai-game-manager-panel.desktop',
  'scripts/build_linux.sh',
  'scripts/linux/lib/init.sh',
  'scripts/linux/actions/build-server.sh',
  'distribution/installer/linux/install.sh',
  'distribution/installer/linux/ai-game-manager-panel-ctl',
  'distribution/docker/Dockerfile',
  'distribution/docker/docker-compose.yml',
  'scripts/linux/actions/build.sh',
  'scripts/linux/actions/build-deb.sh',
  'AI-Game-Manager-Panel.bat',
  'AGMP-GitHub.bat',
  'push-agmp.ps1',
  'scripts/common/check-windows-helper.mjs',
  'scripts/common/check-windows-installer.mjs',
  'scripts/common/check-updater.mjs',
  'scripts/common/check-product-architecture.mjs',
  'scripts/common/check-modules.mjs',
  'scripts/windows/AIGameManagerPanel.ps1',
  'scripts/windows/Test-WindowsHelper.ps1',
  'scripts/windows/lib/Common.ps1',
  'scripts/windows/lib/Toolchain.ps1',
  'scripts/windows/lib/Dependencies.ps1',
  'scripts/windows/lib/Checks.ps1',
  'scripts/windows/lib/Wails.ps1',
  'scripts/windows/lib/Electron.ps1',
  'scripts/windows/tasks/Tasks.ps1',
  'desktop/electron/README.md',
  'desktop/electron/package.json',
  'desktop/electron/pnpm-workspace.yaml',
  'desktop/electron/tsconfig.json',
  'desktop/electron/electron-builder.yml',
  'desktop/electron/src/main.ts',
  'desktop/electron/src/preload.cts',
  'frontend/src/features/xiaoyu/AIWorkbenchView.vue',
  'frontend/src/features/xiaoyu/components/ApprovalMenu.vue',
  'docs/development/MODULES.md',
  'internal/xiaoyu/control/approval_store.go',
  'internal/xiaoyu/control/approval_store_test.go',
  'internal/xiaoyu/control/capability_lease.go',
  'internal/xiaoyu/control/capability_lease_test.go',
  'frontend/src/features/auth/AuthGate.vue',
  'frontend/src/features/users/UsersView.vue',
  'frontend/src/features/logs/LogsView.vue',
  'frontend/src/features/logs/useLogHub.ts',
  'frontend/src/features/settings/UpdateSection.vue',
  'frontend/src/features/deployment/DeploymentView.vue',
  'frontend/src/shared/components/ModulePlaceholderView.vue',
  'frontend/src/features/terminal/TerminalPanel.vue',
  'frontend/src/features/dashboard/DashboardView.vue',
  'frontend/src/features/instances/InstancesView.vue',
  'runtime/README.md',
  'distribution/licenses/README.md',
  'distribution/licenses/DSTCamp-MIT.txt',
  'scripts/tools/license/issue-bflc2.ps1',
  'scripts/tools/license/rotate-release-key.ps1',
  'scripts/tools/license/LicenseAdmin.ps1',
  'scripts/tools/license/AGMP-License-Admin.bat',
  'scripts/tools/license/acceptance-test.ps1',
  'scripts/tools/license/README.md',
  'scripts/common/check-release-key.mjs',
  'scripts/common/check-xiaoyu-core.mjs',
  'scripts/common/check-distribution-boundary.mjs',
  'scripts/common/check-multi-client-ai.mjs',
  'scripts/common/check-headless-xiaoyu.mjs',
  'scripts/common/check-xiaoyu-harness.mjs',
  'scripts/common/check-xiaoyu-model-center.mjs',
  'scripts/common/check-xiaoyu-agent-runtime.mjs',
  'scripts/common/check-xiaoyu-worker.mjs',
  'scripts/common/check-xiaoyu-lease.mjs',
  'internal/xiaoyu/host/kernel.go',
  'internal/xiaoyu/host/loop.go',
  'internal/xiaoyu/host/runs.go',
  'internal/xiaoyu/host/events.go',
  'internal/xiaoyu/host/models.go',
  'internal/xiaoyu/host/models_store.go',
  'internal/xiaoyu/host/models_http.go',
  'frontend/src/features/settings/ModelManagementSection.vue',
  'scripts/windows/lib/Rust.ps1',
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
  'internal/platform/runtime/command.go',
  'internal/platform/runtime/session.go',
  'internal/platform/runtime/run.go',
  'internal/platform/runtime/output.go',
  'internal/platform/runtime/shell.go',
  'internal/platform/runtime/manager.go',
  'internal/platform/runtime/process_windows.go',
  'internal/platform/runtime/process_unix.go',
  'internal/platform/runtime/process_other.go',
  'internal/xiaoyu/runtime/service_test.go',
  'internal/xiaoyu/contract/registry.go',
  'internal/app/app_xiaoyu_tools.go',
  'internal/ops/files/service.go',
  'internal/games/dst/runtime/manager.go',
  'internal/games/dst/runtime/model.go',
  'internal/games/dst/runtime/process.go',
  'internal/games/dst/runtime/process_test.go',
]

for (const rel of requiredPaths) {
  if (!fs.existsSync(path.join(root, rel))) failures.push(`缺少项目骨架：${rel}`)
}

for (const rel of ['internal/xiaoyu/runtime/process_windows.go','internal/xiaoyu/runtime/process_other.go']) {
  if (fs.existsSync(path.join(root, rel))) failures.push(`0.1.85 禁止 XiaoYu Runtime 恢复私有进程实现：${rel}`)
}

// 0.1.65 项目骨架收口：源码根目录只保留稳定边界，运行数据集中到 runtime/。
const canonicalTopLevelDirs = new Set([
  'cmd', 'configs', 'desktop', 'distribution', 'docs', 'frontend', 'internal', 'runtime', 'rust', 'scripts',
])
const generatedTopLevelDirs = new Set(['build', '.git', '.github'])
const legacyRuntimeDirs = new Set(['data', 'log', 'backups', 'instances', 'temp', 'exports', 'plugins', 'cache'])
const removedSourceDirs = ['installer', 'deploy', 'licenses', 'tools', 'update-log']
for (const name of removedSourceDirs) {
  if (fs.existsSync(path.join(root, name))) failures.push(`0.1.65 根目录已收口，禁止恢复旧源码目录：${name}/`)
}
for (const entry of fs.readdirSync(root, { withFileTypes: true })) {
  if (!entry.isDirectory()) continue
  const name = entry.name
  if (canonicalTopLevelDirs.has(name) || generatedTopLevelDirs.has(name) || legacyRuntimeDirs.has(name) || name === '.vscode' || name === '.idea') continue
  if (name === 'node_modules') continue
  failures.push(`0.1.65 根目录出现未归类目录：${name}/；请归入 distribution/docs/scripts/runtime 或既有核心边界`)
}
try {
  const paths = JSON.parse(fs.readFileSync(path.join(root, 'configs/paths.json'), 'utf8'))
  const expectedRuntimePaths = {
    dataDir: 'runtime/data', logDir: 'runtime/log', backupDir: 'runtime/backups', instanceDir: 'runtime/instances',
    tempDir: 'runtime/temp', exportDir: 'runtime/exports', pluginDir: 'runtime/plugins', cacheDir: 'runtime/cache',
  }
  for (const [key, expected] of Object.entries(expectedRuntimePaths)) {
    if (paths[key] !== expected) failures.push(`0.1.65 configs/paths.json ${key} 必须收口到 ${expected}`)
  }
  const appSource = readAppSources()
  for (const leaf of ['data', 'log', 'backups', 'instances', 'temp', 'exports', 'plugins', 'cache']) {
    if (!appSource.includes(`"runtime", "${leaf}"`)) failures.push(`0.1.65 Application 安全回退缺少 runtime/${leaf}`)
  }
} catch (error) {
  failures.push(`0.1.65 无法验证 runtime 路径收口：${error instanceof Error ? error.message : String(error)}`)
}

// 对照 GSM 源码功能建立完整模块 ID 门禁。以后可以新增，但这些基线模块不能无意中丢失。
const requiredModuleIds = [
  'ai', 'dashboard', 'deployment', 'instances', 'tasks', 'files', 'backups', 'terminal',
  'scheduler', 'environment', 'plugins', 'network', 'logs', 'nodes', 'users', 'game-config',
  'steamcmd', 'rcon', 'cloud-build', 'external-api', 'security', 'system', 'settings', 'developer',
  'wallpaper', 'weather', 'sponsor', 'auth', 'file-deploy', 'online-deploy', 'chunk-upload',
  'notifications', 'updater',
]

try {
  const moduleConfig = JSON.parse(fs.readFileSync(path.join(root, 'configs/modules.json'), 'utf8'))
  const modules = moduleConfig.modules ?? []
  const ids = new Set(modules.map(item => item.id))
  for (const id of requiredModuleIds) {
    if (!ids.has(id)) failures.push(`configs/modules.json 缺少基线模块：${id}`)
  }
  // 0.1.83 起未实现模块只存在于模块矩阵/路由占位，不再为每个规划项制造空 feature 目录。
  const realFeatureDirs = ['auth', 'dashboard', 'deployment', 'games', 'instances', 'logs', 'nodes', 'settings', 'terminal', 'users', 'xiaoyu']
  for (const dir of realFeatureDirs) {
    if (!fs.existsSync(path.join(root, 'frontend/src/features', dir))) failures.push(`缺少真实前端功能目录：frontend/src/features/${dir}/`)
  }

  // 这些路由来自本阶段实际审阅的 GSM3 源码。index 是路由聚合器，不作为业务模块。
  const gsmRouteBaseline = [
    'auth', 'backup', 'cloudBuild', 'config', 'developer', 'easytier', 'environment', 'externalApi',
    'fileDeploy', 'files', 'gameDeployment', 'gameconfig', 'games', 'instances', 'minecraft', 'moreGames',
    'network', 'onlineDeploy', 'pluginApi', 'plugins', 'rcon', 'scheduledTasks', 'security', 'settings',
    'sponsor', 'steamcmd', 'system', 'tasks', 'terminal', 'wallpaper', 'weather',
  ]
  const sourceRefs = new Set(modules.flatMap(item => item.sourceRefs ?? []))
  for (const route of gsmRouteBaseline) {
    if (!sourceRefs.has(`route:${route}`)) failures.push(`GSM 功能映射遗漏：route:${route}`)
  }
} catch (error) {
  failures.push(`无法读取 configs/modules.json：${error instanceof Error ? error.message : String(error)}`)
}


// Phase 2 的全局日志中心是唯一历史日志入口。DST 只能保留实时控制台与诊断，禁止重新出现第二套历史日志页面。
if (fs.existsSync(path.join(root, 'frontend/src/games/steam/dst/components/DstLogCenter.vue'))) {
  failures.push('检测到重复日志中心：frontend/src/games/steam/dst/components/DstLogCenter.vue；DST 历史日志必须跳转全局日志中心')
}

// 日志中心参数来自 configs/logging.json。关键分页/刷新参数必须为正数，目录必须是配置日志根目录下的安全相对目录。
try {
  const logging = JSON.parse(fs.readFileSync(path.join(root, 'configs/logging.json'), 'utf8'))
  for (const [name, value] of Object.entries({
    catalogPageSize: logging.catalogPageSize,
    readPageSize: logging.readPageSize,
    autoRefreshSeconds: logging.autoRefreshSeconds,
    flushIntervalMs: logging.flushIntervalMs,
  })) {
    if (!Number.isFinite(value) || value <= 0) failures.push(`configs/logging.json ${name} 必须为大于 0 的数字`)
  }
  const directories = logging.directories ?? {}
  for (const name of ['core', 'operations', 'audit', 'ai', 'steam', 'games', 'nodes']) {
    const value = String(directories[name] ?? '').trim()
    if (!value) failures.push(`configs/logging.json directories.${name} 不能为空`)
    if (path.isAbsolute(value) || value === '..' || value.startsWith(`..${path.sep}`) || value.includes('../') || value.includes('..\\')) {
      failures.push(`configs/logging.json directories.${name} 必须位于配置日志根目录以内`)
    }
  }
  const exportSubdir = String(logging.exportSubdir ?? '').trim()
  if (!exportSubdir || path.isAbsolute(exportSubdir) || exportSubdir.includes('..')) failures.push('configs/logging.json exportSubdir 必须是配置日志根目录下的安全相对目录')
} catch (error) {
  failures.push(`无法验证 configs/logging.json 日志中心参数：${error instanceof Error ? error.message : String(error)}`)
}

// 所有 JSON 配置都必须能被解析，避免把语法错误带进正式构建。
for (const name of ['app.json', 'ui.json', 'ai.json', 'permissions.json', 'paths.json', 'logging.json', 'games.json', 'modules.json', 'release.json', 'server.json', 'license.json']) {
  try {
    JSON.parse(fs.readFileSync(path.join(root, 'configs', name), 'utf8'))
  } catch (error) {
    failures.push(`configs/${name} 不是有效 JSON：${error instanceof Error ? error.message : String(error)}`)
  }
}

// 发布配置必须覆盖 Windows 安装包与 Linux 安装包，正式 Release 不能只剩免安装 EXE。
try {
  const release = JSON.parse(fs.readFileSync(path.join(root, 'configs/release.json'), 'utf8'))
  if (!release.windows?.setupFilename) failures.push('configs/release.json 缺少 windows.setupFilename')
  if (release.windows?.portableDir !== 'build/release/windows/wails/portable') failures.push('Windows portableDir 必须是 build/release/windows/wails/portable')
  if (release.windows?.installerDir !== 'build/release/windows/wails/installer') failures.push('Windows installerDir 必须是 build/release/windows/wails/installer')
  if (release.windows?.workDir !== 'build/work/windows-wails') failures.push('Windows workDir 必须是 build/work/windows-wails')
  if (!release.windows?.portableArchive) failures.push('configs/release.json 缺少 windows.portableArchive')
  if (release.electron?.framework !== 'Electron') failures.push('configs/release.json 缺少 Electron 桌面发布配置')
  if (release.electron?.transport !== 'loopback-http') failures.push('Electron Desktop 必须通过 loopback HTTP 复用 Go Core')
  if (release.electron?.sharedGoCore !== true || release.electron?.sharedVueFrontend !== true) failures.push('Electron Desktop 必须复用 Go Core 与 Vue 前端，不允许复制业务逻辑')
  if (release.electron?.workDir !== 'build/work/electron-builder') failures.push('Electron workDir 必须是 build/work/electron-builder')
  if (!release.linux?.portableArchive) failures.push('configs/release.json 缺少 linux.portableArchive')
  if (!release.linux?.debPackagePrefix) failures.push('configs/release.json 缺少 linux.debPackagePrefix')
  if (!String(release.linux?.runtimeRoot ?? '').includes('XDG_DATA_HOME')) failures.push('Linux 安装包运行目录必须使用用户可写 XDG 数据目录')
  const serverEdition = release.linux?.serverEdition
  const serverArch = new Set(serverEdition?.architectures ?? [])
  for (const arch of ['amd64', 'arm64']) {
    if (!serverArch.has(arch)) failures.push(`Linux Server Edition 缺少架构：${arch}`)
  }
  if (!serverEdition?.staticBinary) failures.push('Linux Server Edition 必须启用静态二进制构建')
} catch (error) {
  failures.push(`无法验证 configs/release.json：${error instanceof Error ? error.message : String(error)}`)
}

// Linux Server Edition 基线：默认不以 root 长期运行，并且必须同时声明 amd64/arm64。
try {
  const server = JSON.parse(fs.readFileSync(path.join(root, 'configs/server.json'), 'utf8'))
  if (server.runAsRoot === true) failures.push('configs/server.json 不允许默认 runAsRoot=true')
  if (!server.serviceUser) failures.push('configs/server.json 缺少 serviceUser')
  const arches = new Set(server.architectures ?? [])
  for (const arch of ['amd64', 'arm64']) if (!arches.has(arch)) failures.push(`configs/server.json 缺少架构：${arch}`)
  if (!server.docker?.runAsNonRoot) failures.push('Docker Server Edition 必须默认非 root 运行')
} catch (error) {
  failures.push(`无法验证 configs/server.json：${error instanceof Error ? error.message : String(error)}`)
}

// 0.1.45 Startup Security Gate：首个管理员只能 Bootstrap 一次，Web 业务 API 默认要求 Session。
try {
  const authService = fs.readFileSync(path.join(root, 'internal/system/auth/service.go'), 'utf8')
  const httpServer = fs.readFileSync(path.join(root, 'internal/bridge/httpapi/server.go'), 'utf8')
  const authGate = fs.readFileSync(path.join(root, 'frontend/src/features/auth/AuthGate.vue'), 'utf8')
  if (!authService.includes('bootstrap.lock')) failures.push('0.1.45 认证服务缺少 bootstrap.lock 失效关闭门禁')
  if (!authService.includes('CreateInitialOwner')) failures.push('0.1.45 认证服务缺少首个 Owner Bootstrap')
  if (!authService.toLowerCase().includes('pbkdf2')) failures.push('0.1.45 认证服务缺少 PBKDF2 密码派生')
  if (!httpServer.includes('requireSession')) failures.push('0.1.45 Web API 缺少 Bearer Session 中间件')
  if (!httpServer.includes('/api/v1/auth/bootstrap/owner')) failures.push('0.1.45 Web API 缺少首个 Owner Bootstrap 端点')
  if (!authGate.includes('registrationOpen')) failures.push('0.1.45 AuthGate 必须根据 registrationOpen 决定是否开放首个管理员创建')
  // 0.1.64 起运行环境已迁入“设置中心 → 运行环境与存储”，AuthGate 不再承载环境初始化。
  if (authGate.includes('environmentSetup') || authGate.includes('skipEnvironmentSetup')) failures.push('0.1.64 AuthGate 禁止重新承载运行环境初始化阶段')
} catch (error) {
  failures.push(`无法验证 0.1.45 Startup Security Gate：${error instanceof Error ? error.message : String(error)}`)
}

// 0.1.52 Dual Desktop Foundation：Wails 与 Electron 只允许拥有不同 Adapter，不得复制 Go 业务逻辑。
try {
  const electronMain = fs.readFileSync(path.join(root, 'desktop/electron/src/main.ts'), 'utf8')
  const electronPreload = fs.readFileSync(path.join(root, 'desktop/electron/src/preload.cts'), 'utf8')
  const electronPackage = JSON.parse(fs.readFileSync(path.join(root, 'desktop/electron/package.json'), 'utf8'))
  const backendApi = fs.readFileSync(path.join(root, 'frontend/src/shared/api/backend.ts'), 'utf8')
  const electronBuild = fs.readFileSync(path.join(root, 'scripts/windows/lib/Electron.ps1'), 'utf8')
  const menu = fs.readFileSync(path.join(root, 'scripts/windows/AIGameManagerPanel.ps1'), 'utf8')
  if (electronPackage.devDependencies?.electron !== '44.3.0') failures.push('Electron Desktop 必须固定 Electron 44.3.0')
  if (electronPackage.devDependencies?.['electron-builder'] !== '26.16.1') failures.push('Electron Builder 必须固定 26.16.1')
  if (!electronMain.includes('nodeIntegration: false')) failures.push('Electron 必须关闭 nodeIntegration')
  if (!electronMain.includes('contextIsolation: true')) failures.push('Electron 必须启用 contextIsolation')
  if (!electronMain.includes('sandbox: true')) failures.push('Electron Renderer 必须启用 sandbox')
  if (!electronMain.includes('setPermissionRequestHandler')) failures.push('Electron 缺少默认拒绝权限请求门禁')
  if (!electronMain.includes("AGMP_DESKTOP_FRAMEWORK: 'electron'")) failures.push('Electron Sidecar 缺少桌面框架环境标记')
  if (!electronMain.includes('AGMP_ROOT: runtimeRoot')) failures.push('Electron Production Core 必须使用用户可写 runtimeRoot')
  if (!electronMain.includes("path.join(process.resourcesPath, 'core'")) failures.push('Electron 必须使用打包的 AGMP-Core sidecar')
  if (!electronPreload.includes("framework: 'electron'")) failures.push('Electron preload 缺少只读框架标记')
  // 允许通过 contextBridge 暴露经过白名单约束的 invoke 能力，但禁止把 ipcRenderer 本体或通用 send/on 接口交给 Renderer。
  if (!electronPreload.includes('contextBridge.exposeInMainWorld')) failures.push('Electron preload 必须通过 contextBridge 暴露最小桥接能力')
  if (/\bipcRenderer\s*:/.test(electronPreload) || /\b(send|sendSync|on|once)\s*:\s*.*ipcRenderer/i.test(electronPreload)) failures.push('Electron preload 禁止向 Renderer 暴露通用 ipcRenderer / send / on 能力')
  for (const channel of ['agmp:select-directory', 'agmp:open-path']) if (!electronPreload.includes(channel)) failures.push(`Electron preload 缺少白名单 IPC：${channel}`)
  if (!backendApi.includes("export type BackendAdapter = 'wails' | 'electron' | 'http'")) failures.push('前端缺少 Wails/Electron/Web Adapter 分类')
  if (!backendApi.includes('hasElectronShell')) failures.push('前端缺少 Electron Shell 探测')
  if (backendApi.includes("if (mode() === 'desktop') return requireWails()")) failures.push('Electron 不得误走 Wails Bridge')
  if (!electronBuild.includes("'.\\cmd\\aigame-manager-web'")) failures.push('Electron Release 必须从同一 Go HTTP Core 构建 sidecar')
  if (!electronBuild.includes('build\\work\\electron-builder')) failures.push('Electron Release 必须使用 build/work 独立工作区')
  if (!menu.includes('10. 正式发布')) failures.push('主菜单必须提供 Wails/Electron/Linux/全平台正式发布入口')
} catch (error) {
  failures.push(`无法验证 Dual Desktop Foundation：${error instanceof Error ? error.message : String(error)}`)
}


// 0.1.53 First-Start Layout Alignment：整体必须视觉居中；品牌标题与当前步骤脱离内容卡片并左右对置。
try {
  const authGate = fs.readFileSync(path.join(root, 'frontend/src/features/auth/AuthGate.vue'), 'utf8')
  const usersView = fs.readFileSync(path.join(root, 'frontend/src/features/users/UsersView.vue'), 'utf8')
  const styles = fs.readFileSync(path.join(root, 'frontend/src/shared/styles/base.css'), 'utf8')
  const authService = fs.readFileSync(path.join(root, 'internal/system/auth/service.go'), 'utf8')
  const mainGo = fs.readFileSync(path.join(root, 'main.go'), 'utf8')
  if (authGate.includes('ownerDisplayName')) failures.push('0.1.53 首次 Owner 注册页禁止继续要求显示名称')
  if (!authGate.includes("displayName: '超级管理员'")) failures.push('0.1.53 首次 Owner 必须使用默认显示名称“超级管理员”')
  if (!authGate.includes('ownerFormComplete')) failures.push('0.1.53 首次 Owner 表单缺少完整性计算门禁')
  if (!authGate.includes(':disabled="busy || !ownerFormComplete"')) failures.push('0.1.53 创建最高管理员按钮必须在表单未完成时禁用')
  if (!authGate.includes('const ownerKeySaved = ref(false)') || !authGate.includes('const ownerRequireSecurityKey = ref(false)')) failures.push('0.1.53 两个首次安全选项必须默认未选中')
  if (authGate.includes('keyCopyMessage')) failures.push('0.1.53 首次密钥复制禁止通过响应式 keyCopyMessage 触发整页重绘')
  if (!authGate.includes('document.execCommand(\'copy\')')) failures.push('0.1.53 首次密钥复制缺少局部、无响应式重绘的复制路径')
  if (!authGate.includes('class="auth-brand-floating"') || !authGate.includes('class="auth-stage-floating"') || !authGate.includes('class="auth-center-stage"')) failures.push('0.1.64 首次启动必须使用左上品牌 / 右上步骤 / 中央表单三块独立区域')
  if (!authGate.includes('class="auth-stage-summary"')) failures.push('0.1.64 当前步骤必须位于右上独立浮动区')
  if (authGate.includes('class="auth-step"')) failures.push('0.1.64 内容卡片内禁止继续重复渲染步骤标题')
  if (!styles.includes('.auth-brand-floating') || !styles.includes('.auth-stage-floating') || !styles.includes('.auth-center-stage')) failures.push('0.1.64 首次启动缺少三块独立布局 CSS')
  if (!styles.includes('place-items:center')) failures.push('0.1.64 中央表单舞台必须独立视觉居中')
  if (!styles.includes('.auth-submit:disabled')) failures.push('0.1.53 首次创建按钮缺少明确灰色禁用态')
  if (!authService.includes('displayName = "超级管理员"')) failures.push('0.1.53 后端缺少 Owner 默认显示名称兜底')
  if (!authService.includes('UpdateMyDisplayName')) failures.push('0.1.53 后端缺少当前账号显示名称修改能力')
  if (!usersView.includes('updateMyDisplayName')) failures.push('0.1.53 用户与权限页缺少显示名称修改入口')
  if (!mainGo.includes('Width:            1280') || !mainGo.includes('Height:           840')) failures.push('0.1.53 Windows 默认窗口尺寸必须保持 1280x840')
} catch (error) {
  failures.push(`无法验证 0.1.53 First-Start Layout Alignment：${error instanceof Error ? error.message : String(error)}`)
}

// 0.1.50 Startup Security UX：安全密钥只能保存校验值，Step 02 必须允许持久化跳过。
try {
  const authService = fs.readFileSync(path.join(root, 'internal/system/auth/service.go'), 'utf8')
  const environmentService = fs.readFileSync(path.join(root, 'internal/deploy/environment/service.go'), 'utf8')
  const authGate = fs.readFileSync(path.join(root, 'frontend/src/features/auth/AuthGate.vue'), 'utf8')
  const usersView = fs.readFileSync(path.join(root, 'frontend/src/features/users/UsersView.vue'), 'utf8')
  const httpServer = fs.readFileSync(path.join(root, 'internal/bridge/httpapi/server.go'), 'utf8')
  if (!authService.includes('SecurityKeyHash')) failures.push('0.1.50 账号存储缺少安全密钥校验值字段')
  if (!authService.includes('sha256.Sum256')) failures.push('0.1.50 安全密钥必须保存 SHA-256 校验值')
  if (!authService.includes('ConstantTimeCompare')) failures.push('0.1.50 安全密钥比较必须使用常量时间比较')
  if (!authService.includes('RequireSecurityKey')) failures.push('0.1.50 账号缺少安全密钥登录策略')
  if (!httpServer.includes('/api/v1/auth/security-key/generate')) failures.push('0.1.50 Web API 缺少安全密钥生成端点')
  if (!httpServer.includes('/api/v1/auth/security-key/verification')) failures.push('0.1.50 Web API 缺少安全密钥验证策略端点')
  if (!authGate.includes('ownerKeySaved') || !authGate.includes('!ownerRequireSecurityKey.value ||')) failures.push('0.1.89 Owner 安全密钥必须保持可选；只有启用密钥登录时才要求保存')
  if (authGate.includes('environmentSetup') || authGate.includes('skipEnvironmentSetup')) failures.push('0.1.64 运行环境已迁入设置中心，AuthGate 禁止继续承载环境初始化步骤')
  if (!usersView.includes('rotateMySecurityKey')) failures.push('0.1.50 用户中心缺少安全密钥轮换入口')
  if (!usersView.includes('setMySecurityKeyVerification')) failures.push('0.1.50 用户中心缺少安全密钥验证开关')
  if (!environmentService.includes('Skipped')) failures.push('0.1.50 环境状态缺少持久化 skipped 标记')
  if (!environmentService.includes('func (s *Service) Skip')) failures.push('0.1.50 环境服务缺少持久化跳过方法')
} catch (error) {
  failures.push(`无法验证 0.1.50 Startup Security UX：${error instanceof Error ? error.message : String(error)}`)
}

// 0.1.72 Windows Installer UX：正式安装器必须自带中文文案、强制许可协议，并且不依赖外部 ChineseSimplified.isl。
try {
  const installerBuild = fs.readFileSync(path.join(root, 'scripts/windows/lib/Wails.ps1'), 'utf8')
  const installerIss = fs.readFileSync(path.join(root, 'distribution/installer/windows/AIGameManagerPanel.iss'), 'utf8')
  const eulaPath = path.join(root, 'distribution/installer/windows/EULA-zh-CN.txt')
  const releaseCfg = JSON.parse(fs.readFileSync(path.join(root, 'configs/release.json'), 'utf8'))
  if (!fs.existsSync(eulaPath)) failures.push('0.1.72 Windows Installer 缺少 EULA-zh-CN.txt')
  if (!installerIss.includes('LicenseFile=EULA-zh-CN.txt')) failures.push('0.1.72 Windows Installer 必须启用 Inno 强制许可协议页')
  if (!installerIss.includes('LicenseAccepted=我接受本协议') || !installerIss.includes('LicenseNotAccepted=我不同意本协议')) failures.push('0.1.72 许可协议接受/拒绝选项必须中文化')
  if (!installerIss.includes('您必须接受本协议才能继续安装')) failures.push('0.1.72 许可页必须明确不同意时无法继续安装')
  if (!installerIss.includes('Name: "chinesesimp"; MessagesFile: "compiler:Default.isl"')) failures.push('0.1.72 Windows Installer 必须以 Inno 内置 Default.isl 为基础，避免外部语言文件依赖')
  if (installerIss.includes('AGMP_USE_INNO_CHINESE') || installerBuild.includes('ChineseSimplified.isl')) failures.push('0.1.72 禁止恢复外部 ChineseSimplified.isl 条件依赖')
  if (!installerIss.includes('DisableWelcomePage=no') || !installerIss.includes('SetupLogging=yes') || !installerIss.includes('CloseApplications=yes')) failures.push('0.1.72 Windows Installer 缺少完整安装向导/日志/运行中程序处理')
  if (!installerIss.includes('UsePreviousAppDir=yes') || !installerIss.includes('UsePreviousTasks=yes')) failures.push('0.1.72 Windows Installer 升级安装必须复用既有目录与任务设置')
  if (!installerIss.includes('function InitializeUninstall(): Boolean;') || !installerIss.includes('运行数据、日志、备份和实例信息将默认保留')) failures.push('0.1.72 卸载流程缺少中文数据保留提示')
  if (!installerBuild.includes('EULA 为强制接受页') || !installerBuild.includes('正式安装器')) failures.push('0.1.72 Wails Release 缺少安装器协议/产物大小诊断')
  if (releaseCfg.windows?.installerLanguage !== 'zh-CN' || releaseCfg.windows?.licenseAgreementRequired !== true) failures.push('0.1.72 configs/release.json 未声明中文强制协议安装器')
} catch (error) {
  failures.push(`无法验证 0.1.72 Windows Installer UX：${error instanceof Error ? error.message : String(error)}`)
}


// 0.1.73 Windows Update：GitHub Releases 稳定通道 + SHA256 + 中文升级安装器 + runtime 数据保留。
try {
  const updateCfg = JSON.parse(fs.readFileSync(path.join(root, 'configs/update.json'), 'utf8'))
  const updaterService = fs.readFileSync(path.join(root, 'internal/deploy/updater/service.go'), 'utf8')
  const updateView = fs.readFileSync(path.join(root, 'frontend/src/features/settings/UpdateSection.vue'), 'utf8')
  const appView = fs.readFileSync(path.join(root, 'frontend/src/app/App.vue'), 'utf8')
  const topbar = fs.readFileSync(path.join(root, 'frontend/src/app/layout/AppTopbar.vue'), 'utf8')
  const installerIss = fs.readFileSync(path.join(root, 'distribution/installer/windows/AIGameManagerPanel.iss'), 'utf8')
  const wailsBuild = fs.readFileSync(path.join(root, 'scripts/windows/lib/Wails.ps1'), 'utf8')
  if (updateCfg.provider !== 'github-releases' || updateCfg.repository !== 'yubboo/AI-Game-Manager-Panel') failures.push('0.1.73 updater 必须使用官方 GitHub Releases 仓库')
  if (updateCfg.channel !== 'stable' || updateCfg.checkOnStartup !== true) failures.push('0.1.73 updater 必须启用 stable 启动检查')
  if (updateCfg.requireSha256 !== true || updateCfg.preserveRuntimeData !== true) failures.push('0.1.73 updater 必须要求 SHA256 并声明保留 runtime 数据')
  if (!updaterService.includes('/releases/latest') || !updaterService.includes('PrepareLatest') || !updaterService.includes('LaunchInstaller')) failures.push('0.1.73 Go updater 缺少 latest/check/download/install 链路')
  if (!updaterService.includes('安装器 SHA256 校验失败') || !updaterService.includes('github.com')) failures.push('0.1.73 updater 缺少安装器完整性/来源限制')
  if (!updateView.includes('下载并安装更新') || !updateView.includes('runtime')) failures.push('0.1.73 设置中心缺少更新安装与数据保留说明')
  if (!appView.includes('checkForUpdates(false)') || !topbar.includes('新版本 v')) failures.push('0.1.73 客户端缺少启动检查或顶部新版本提示')
  if (!installerIss.includes('DirExistsTitle=文件夹已存在') || !installerIss.includes('DirExists=文件夹：')) failures.push('0.1.73 Folder Exists 弹窗必须完整中文化')
  if (!installerIss.includes('GetInstalledVersion') || !installerIss.includes('{param:AGMPUPDATE|0}') || !installerIss.includes('本向导将升级到版本')) failures.push('0.1.73 安装器缺少旧版本/自动更新升级识别')
  if (!wailsBuild.includes('.sha256') || !wailsBuild.includes('Get-FileHash')) failures.push('0.1.73 Wails Release 必须生成 Setup SHA256 companion asset')
} catch (error) {
  failures.push(`无法验证 0.1.73 Windows Update：${error instanceof Error ? error.message : String(error)}`)
}

// 0.1.48 Release Test Data Isolation Hotfix：HTTP/Application 集成测试不得回退到真实用户数据目录。
try {
  const appSource = readAppSources()
  const appTest = fs.readFileSync(path.join(root, 'internal/app/app_test.go'), 'utf8')
  const httpTest = fs.readFileSync(path.join(root, 'internal/bridge/httpapi/server_test.go'), 'utf8')
  if (!appSource.includes('func NewWithOptions(options Options) *Application')) failures.push('0.1.48 Application 缺少显式运行目录注入入口 NewWithOptions')
  if (!appSource.includes('DataDir string')) failures.push('0.1.48 Application Options 缺少 DataDir 注入')
  if (!httpTest.includes('application.NewWithOptions')) failures.push('0.1.48 HTTP 集成测试必须使用 NewWithOptions 隔离数据目录')
  if (!httpTest.includes('DataDir: "data"')) failures.push('0.1.48 HTTP 集成测试缺少临时 dataDir 注入')
  if (httpTest.includes('application.New()')) failures.push('0.1.48 HTTP 集成测试禁止直接 application.New() 回退真实用户目录')
  if (httpTest.includes('XDG_CONFIG_HOME') || httpTest.includes('APPDATA') || httpTest.includes('LOCALAPPDATA')) failures.push('0.1.48 HTTP 集成测试禁止依赖平台用户配置环境变量')
  if (!appTest.includes('TestNewWithOptionsKeepsAuthDataInsideInjectedRoot')) failures.push('0.1.48 缺少认证数据目录隔离回归测试')
  if (!appTest.includes('bootstrap.lock')) failures.push('0.1.48 Application 隔离测试必须验证 bootstrap.lock 位于临时 dataDir')
} catch (error) {
  failures.push(`无法验证 0.1.48 Release Test Data Isolation Hotfix：${error instanceof Error ? error.message : String(error)}`)
}

// 0.1.47 Release Test Isolation Hotfix：环境初始化单测必须与真实机器 SteamCMD 安装状态解耦。
try {
  const environmentTest = fs.readFileSync(path.join(root, 'internal/deploy/environment/service_test.go'), 'utf8')
  if (!environmentTest.includes('InitializeRequest{SteamCMDPath: steamCMD}')) failures.push('0.1.47 环境初始化持久化测试必须显式使用临时 SteamCMD 路径')
  if (!environmentTest.includes('platformSteamCMDExecutableName')) failures.push('0.1.47 SteamCMD 自动探测测试必须使用平台相关可执行文件名')
  if (!environmentTest.includes('runtime.GOOS == "windows"')) failures.push('0.1.47 SteamCMD 测试缺少 Windows steamcmd.exe 分支')
  if (!environmentTest.includes('TestInitializationDoesNotRequireSteamCMD')) failures.push('0.1.89 基础 Runtime Manager 初始化不得强制要求 SteamCMD')
  if (environmentTest.includes('func executableName() string') && environmentTest.includes('return "steamcmd"')) failures.push('0.1.47 禁止恢复只适用于 Linux 的硬编码 SteamCMD 测试 helper')
} catch (error) {
  failures.push(`无法验证 0.1.47 Release Test Isolation Hotfix：${error instanceof Error ? error.message : String(error)}`)
}




// 0.1.63 Windows Dev Helper Final Refactor：唯一 BAT 启动器 + PowerShell Task Runner + UTF-8 Gate。
try {
  const rootBat = fs.readFileSync(path.join(root, 'AI-Game-Manager-Panel.bat'))
  if ([...rootBat].some(value => value > 0x7f)) failures.push('0.1.63 根 AI-Game-Manager-Panel.bat 必须保持 ASCII-safe')
  const rootText = rootBat.toString('ascii')
  if (!rootText.includes('chcp 65001') || !rootText.includes('scripts\\windows\\AIGameManagerPanel.ps1')) failures.push('0.1.63 根 AI-Game-Manager-Panel.bat 必须切换 UTF-8 并委托 AIGameManagerPanel.ps1')
  const helper = fs.readFileSync(path.join(root, 'scripts/windows/AIGameManagerPanel.ps1'), 'utf8')
  const tasks = fs.readFileSync(path.join(root, 'scripts/windows/tasks/Tasks.ps1'), 'utf8')
  const deps = fs.readFileSync(path.join(root, 'scripts/windows/lib/Dependencies.ps1'), 'utf8')
  const wails = fs.readFileSync(path.join(root, 'scripts/windows/lib/Wails.ps1'), 'utf8')
  const electron = fs.readFileSync(path.join(root, 'scripts/windows/lib/Electron.ps1'), 'utf8')
  const scriptBatFiles = []
  const walk = dir => { for (const e of fs.readdirSync(dir,{withFileTypes:true})) { const f=path.join(dir,e.name); if(e.isDirectory()) walk(f); else if(e.name.toLowerCase().endsWith('.bat')) scriptBatFiles.push(f) } }
  walk(path.join(root,'scripts/windows'))
  if (scriptBatFiles.length) failures.push(`0.1.63 scripts/windows 禁止残留 BAT：${scriptBatFiles.map(f=>path.relative(root,f)).join(', ')}`)
  for (const label of ['1. 初始化 / 修复基础开发环境','2. 开发模式','3. 项目检查','4. Wails Windows 候选构建','5. Electron Windows 候选构建','6. Linux Server 候选构建','7. 预览 / 启动构建产物','8. 状态、诊断与修复','9. 清理与重置','10. 正式发布']) if(!helper.includes(label)) failures.push(`0.1.89 主菜单缺少：${label}`)
  if (!helper.includes('Invoke-HelperSelfTest')) failures.push('0.1.63 主菜单缺少菜单 1-10 Dry-Run 自检')
  const initStart = tasks.indexOf('function Invoke-InitializeProject')
  const devStart = tasks.indexOf('function Invoke-DevelopmentMenu')
  const initBlock = initStart >= 0 && devStart > initStart ? tasks.slice(initStart, devStart) : ''
  if (!initBlock || initBlock.includes('Invoke-FrontendTypeCheck')) failures.push('0.1.63 初始化必须与代码质量检查解耦')
  if (!deps.includes("@('install','--frozen-lockfile','--store-dir',$store)")) failures.push('0.2.9 pnpm install 必须使用 frozen lockfile + AGMP 专用 store')
  if (deps.includes('--prefer-online')) failures.push('0.1.63 禁止使用未知 pnpm 参数 --prefer-online')
  if (!deps.includes("@('exec','install-electron','--no')")) failures.push('0.1.63 Electron Runtime 必须使用 install-electron')
  if (!deps.includes('electron_config_cache')) failures.push('0.1.63 Electron Runtime 缺少独立缓存')
  if (!wails.includes('build\\work\\windows-wails')) failures.push('0.1.67 Wails Release 必须使用 build/work/windows-wails')
  if (!wails.includes('BUILD-STATUS.txt')) failures.push('0.1.63 Wails Release 缺少状态文件')
  if (!electron.includes('build\\work\\electron-builder')) failures.push('0.1.67 Electron Release 必须使用 build/work/electron-builder')
  if (!fs.existsSync(path.join(root,'scripts/common/check-windows-helper.mjs'))) failures.push('0.1.63 缺少 Windows Helper/编码门禁')
} catch(error) { failures.push(`无法验证 0.1.63 Windows Helper：${error instanceof Error ? error.message : String(error)}`) }

// 0.1.67 Windows Release：最终产物只进 build/release；中间文件只进 build/work；按文件安全发布，避免旧 EXE 被占用导致整次 Release 失败。
try {
  const windowsBuild = fs.readFileSync(path.join(root, 'scripts/windows/lib/Wails.ps1'), 'utf8')
  const electronBuild = fs.readFileSync(path.join(root, 'scripts/windows/lib/Electron.ps1'), 'utf8')
  const common = fs.readFileSync(path.join(root, 'scripts/windows/lib/Common.ps1'), 'utf8')
  const installerIss = fs.readFileSync(path.join(root, 'distribution/installer/windows/AIGameManagerPanel.iss'), 'utf8')
  const builder = fs.readFileSync(path.join(root, 'desktop/electron/electron-builder.yml'), 'utf8')
  const releaseConfig = JSON.parse(fs.readFileSync(path.join(root, 'configs/release.json'), 'utf8'))
  if (!windowsBuild.includes(String.raw`build\work\windows-wails`)) failures.push('Wails Release 必须使用 build/work/windows-wails')
  if (!windowsBuild.includes('AI-Game-Manager-Panel-Windows-x64-Portable.zip')) failures.push('Wails Release 缺少 Portable 产物')
  if (!windowsBuild.includes('AI-Game-Manager-Panel-$script:AGMPVersion-Windows-x64-Setup.exe')) failures.push('Wails Release 缺少版本化 Setup 产物')
  if (!electronBuild.includes(String.raw`build\work\electron-builder`)) failures.push('Electron Release 必须使用 build/work/electron-builder')
  if (!electronBuild.includes(String.raw`build\release\windows\electron`)) failures.push('Electron Release 最终产物必须进入 build/release/windows/electron')
  if (!common.includes('function Publish-AGMPArtifact') || !common.includes('目标文件被其他进程占用')) failures.push('缺少 Windows 文件占用安全发布机制')
  if (!windowsBuild.includes('Publish-AGMPArtifact') || !electronBuild.includes('Publish-AGMPArtifact')) failures.push('Wails/Electron 必须按单文件使用安全发布函数')
  if (!installerIss.includes(String.raw`OutputDir=..\..\..\build\work\windows-wails\release-stage\installer`)) failures.push('Inno OutputDir 必须进入 build/work，不得直接写最终 Release')
  if (!builder.includes('output: ../../build/work/electron-builder')) failures.push('electron-builder output 必须进入 build/work')
  if (!builder.includes('asar: true') || !builder.includes('compression: maximum') || !builder.includes('electronLanguages:')) failures.push('Electron 包体积优化配置不完整')
  if (releaseConfig.windows?.releaseDir !== 'build/release/windows/wails') failures.push('configs/release.json Wails releaseDir 不正确')
  if (releaseConfig.electron?.releaseDir !== 'build/release/windows/electron') failures.push('configs/release.json Electron releaseDir 不正确')
} catch (error) {
  failures.push(`无法验证 0.1.67 Windows Release / Build Layout：${error instanceof Error ? error.message : String(error)}`)
}


// 0.1.67 GitHub Safety + Development License Mode：公开仓库禁止承载发行私钥/用户状态；开发构建不需要真实 CDK。
try {
  const ignore = fs.readFileSync(path.join(root, '.gitignore'), 'utf8')
  const checks = fs.readFileSync(path.join(root, 'scripts/windows/lib/Checks.ps1'), 'utf8')
  const wails = fs.readFileSync(path.join(root, 'scripts/windows/lib/Wails.ps1'), 'utf8')
  const tasks = fs.readFileSync(path.join(root, 'scripts/windows/tasks/Tasks.ps1'), 'utf8')
  const electronMain = fs.readFileSync(path.join(root, 'desktop/electron/src/main.ts'), 'utf8')
  const licenseService = fs.readFileSync(path.join(root, 'internal/system/license/service.go'), 'utf8')
  const devMode = fs.readFileSync(path.join(root, 'internal/system/license/buildmode_dev.go'), 'utf8')
  const releaseMode = fs.readFileSync(path.join(root, 'internal/system/license/buildmode_release.go'), 'utf8')
  for (const rule of ['*.key','*.priv','*.seed','*.pem','*.p12','*.pfx','*.cdk','**/activation.json','**/accounts.json','**/cluster_token.txt','runtime/*']) {
    if (!ignore.includes(rule)) failures.push(`0.1.67 .gitignore 缺少敏感数据规则：${rule}`)
  }
  if (!fs.existsSync(path.join(root, 'scripts/common/check-github-safety.mjs'))) failures.push('0.1.67 缺少 GitHub Safety Gate')
  if (!fs.existsSync(path.join(root, 'docs/development/GITHUB-SAFETY.md'))) failures.push('0.1.67 缺少 GitHub 提交安全规范')
  if (!fs.existsSync(path.join(root, '.github/workflows/safety.yml'))) failures.push('0.1.67 缺少 GitHub Actions Safety Gate')
  if (!checks.includes('Invoke-GitHubSafetyGate')) failures.push('0.1.67 Windows 项目检查未接入 GitHub Safety Gate')
  if (!devMode.includes('//go:build agmp_dev_license') || !devMode.includes('developmentEntitlementMode = true')) failures.push('0.1.67 缺少开发许可证 Build Tag')
  if (!releaseMode.includes('//go:build !agmp_dev_license') || !releaseMode.includes('developmentEntitlementMode = false')) failures.push('0.1.67 正式构建必须默认关闭开发许可证模式')
  if (!licenseService.includes('StateDevelopment') || !licenseService.includes('Features: []string{"*"}')) failures.push('0.1.67 License Service 缺少开发 entitlement 状态')
  if (!wails.includes("@('dev','-tags','agmp_dev_license')")) failures.push('0.1.67 Wails Dev 必须显式启用开发许可证 Build Tag')
  if (!tasks.includes("@('build','-tags','agmp_dev_license'")) failures.push('0.1.67 Web Dev 必须显式启用开发许可证 Build Tag')
  if (!electronMain.includes("['run', '-tags', 'agmp_dev_license'")) failures.push('0.1.67 Electron Dev Core 必须显式启用开发许可证 Build Tag')
  if (electronMain.includes('agmp_dev_license') && electronMain.includes("path.join(process.resourcesPath, 'core'")) {
    // Release still uses packaged sidecar; this assertion only documents separation.
  }
} catch (error) {
  failures.push(`无法验证 0.1.67 GitHub Safety / Development License Mode：${error instanceof Error ? error.message : String(error)}`)
}

// 0.1.67 全局品牌迁移：AI游戏管理器面板为产品名；“AGMP/AGMP”只允许留在 DST 专属模块、历史文档和兼容/安全规则中。
try {
  const appCfg = JSON.parse(fs.readFileSync(path.join(root, 'configs/app.json'), 'utf8'))
  const wailsCfg = JSON.parse(fs.readFileSync(path.join(root, 'wails.json'), 'utf8'))
  const electronPkg = JSON.parse(fs.readFileSync(path.join(root, 'desktop/electron/package.json'), 'utf8'))
  const builder = fs.readFileSync(path.join(root, 'desktop/electron/electron-builder.yml'), 'utf8')
  const goMod = fs.readFileSync(path.join(root, 'go.mod'), 'utf8')
  const sidebar = fs.readFileSync(path.join(root, 'frontend/src/app/layout/AppSidebar.vue'), 'utf8')
  if (appCfg.productName !== 'AI游戏管理器面板') failures.push('configs/app.json 全局产品名必须是 AI游戏管理器面板')
  if (wailsCfg.name !== 'AI-Game-Manager-Panel' || wailsCfg.info?.productName !== 'AI游戏管理器面板') failures.push('wails.json 全局品牌未完成迁移')
  if (electronPkg.name !== 'ai-game-manager-panel-electron') failures.push('Electron package name 未完成迁移')
  if (!builder.includes('productName: AI游戏管理器面板')) failures.push('electron-builder 产品名未完成迁移')
  if (!goMod.includes('module github.com/yubboo/AI-Game-Manager-Panel')) failures.push('Go module 必须切换到新 GitHub 仓库')
  if (!sidebar.includes('AI游戏管理器面板') || !sidebar.includes('codex-brand__mark">鱼<')) failures.push('前端 Sidebar 小鱼品牌未完成迁移')
} catch (error) {
  failures.push(`无法验证 0.1.67 品牌迁移：${error instanceof Error ? error.message : String(error)}`)
}

// 版本一致性以 Go Core 版本常量为唯一基准，不再依赖 Windows 脚本硬编码。
let version = null
try {
  const appGo = fs.readFileSync(path.join(root, 'internal/app/app.go'), 'utf8')
  version = appGo.match(/Version\s*=\s*"([^"]+)"/)?.[1]?.trim() ?? null
  if (!version) failures.push('无法从 internal/app/app.go 读取版本')
  if (version) {
    const packageJson = JSON.parse(fs.readFileSync(path.join(root, 'frontend/package.json'), 'utf8'))
    const electronPackage = JSON.parse(fs.readFileSync(path.join(root, 'desktop/electron/package.json'), 'utf8'))
    const wails = JSON.parse(fs.readFileSync(path.join(root, 'wails.json'), 'utf8'))
    const rustWorkspace = fs.readFileSync(path.join(root, 'rust/Cargo.toml'), 'utf8')
    const rustVersion = rustWorkspace.match(/\[workspace\.package\][\s\S]*?\nversion\s*=\s*"([^"]+)"/)?.[1]?.trim() ?? null
    const helper = fs.readFileSync(path.join(root, 'scripts/windows/AIGameManagerPanel.ps1'), 'utf8')
    const installer = fs.readFileSync(path.join(root, 'distribution/installer/windows/AIGameManagerPanel.iss'), 'utf8')
    if (packageJson.version !== version) failures.push(`frontend/package.json 版本 ${packageJson.version} != ${version}`)
    if (electronPackage.version !== version) failures.push(`desktop/electron/package.json 版本 ${electronPackage.version} != ${version}`)
    if (wails.info?.productVersion !== version) failures.push(`wails.json productVersion ${wails.info?.productVersion ?? 'missing'} != ${version}`)
    if (rustVersion !== version) failures.push(`rust/Cargo.toml workspace version ${rustVersion ?? 'missing'} != ${version}`)
    if (!helper.includes('Get-AGMPVersion')) failures.push('Windows Helper 必须动态读取 Go Core 版本，禁止重复硬编码版本')
    if (!installer.includes(`#define MyAppVersion "${version}"`)) failures.push(`Windows Installer 未同步版本 ${version}`)
    if (!installer.includes(`OutputBaseFilename=AI-Game-Manager-Panel-${version}-Windows-x64-Setup`)) failures.push(`Windows Installer 输出文件名未同步版本 ${version}`)
    for (const runtimeDir of ['data', 'log', 'backups', 'instances', 'temp', 'exports', 'plugins', 'cache']) {
      if (!installer.includes(`Name: "{app}\\runtime\\${runtimeDir}"`)) failures.push(`Windows Installer 未创建运行目录：${runtimeDir}`)
    }
    const history = fs.readFileSync(path.join(root, 'docs/PROJECT-HISTORY.md'), 'utf8')
    if (!history.includes(`## AI-Game-Manager-Panel ${version}`)) failures.push(`docs/PROJECT-HISTORY.md 缺少当前版本 ${version} 记录`)
  }
} catch(error) { failures.push(`无法验证版本一致性：${error instanceof Error ? error.message : String(error)}`) }



// 0.1.64 十项统一修复：Theme / Workbench / License / Environment / Wails / Electron。
try {
  const history = fs.readFileSync(path.join(root, 'docs/PROJECT-HISTORY.md'), 'utf8')
  if (!history.includes('## AI-Game-Manager-Panel 0.1.64')) failures.push('历史记录缺少 0.1.64 归档')
  const toolchain = fs.readFileSync(path.join(root, 'scripts/windows/lib/Toolchain.ps1'), 'utf8')
  if (/^\s*\$home\s*=/im.test(toolchain)) failures.push('0.1.64 PowerShell 禁止使用 $home 覆盖只读 $HOME')
  const showStart = toolchain.indexOf('function Show-AGMPToolchain')
  const showEnd = toolchain.indexOf('function Invoke-GoModules')
  const showBlock = showStart >= 0 && showEnd > showStart ? toolchain.slice(showStart, showEnd) : ''
  if (/return\s+\$tools\b/i.test(showBlock)) failures.push('0.1.64 Show-AGMPToolchain 禁止返回工具链 Hashtable')

  const electronMain = fs.readFileSync(path.join(root, 'desktop/electron/src/main.ts'), 'utf8')
  const electronBuilder = fs.readFileSync(path.join(root, 'desktop/electron/electron-builder.yml'), 'utf8')
  if (!electronMain.includes("label: '文件'") || !electronMain.includes("label: '编辑'") || !electronMain.includes("label: '帮助'")) failures.push('0.1.64 Electron 原生菜单必须中文化')
  if (!electronBuilder.includes('electronDist: node_modules/electron/dist')) failures.push('0.1.64 Electron Builder 必须复用本地 electronDist')

  const appVue = fs.readFileSync(path.join(root, 'frontend/src/app/App.vue'), 'utf8')
  const store = fs.readFileSync(path.join(root, 'frontend/src/shared/store/app.ts'), 'utf8')
  const css = fs.readFileSync(path.join(root, 'frontend/src/shared/styles/base.css'), 'utf8')
  if (!appVue.includes('AppRightSidebar') || !appVue.includes('BottomPanel')) failures.push('0.1.64 App Shell 缺少右侧栏或 Bottom Panel')
  if (!appVue.includes('gridTemplateColumns') || !appVue.includes("'minmax(0,1fr)'") || !appVue.includes("'0px'")) failures.push('0.1.100 工作台必须保持稳定五轨 Grid，并将收起侧轨动画到 0px')
  if (!store.includes('WORKBENCH_LAYOUT_KEY') || !store.includes('leftWidth') || !store.includes('rightWidth') || !store.includes('bottomHeight')) failures.push('0.1.64 工作台布局缺少持久化可拖拽尺寸')
  if (!store.includes("agmp.workbench.layout.v6") || !store.includes('WORKBENCH_INITIAL_LEFT_RATIO = 0.05') || !store.includes('WORKBENCH_INITIAL_CENTER_RATIO = 0.75') || !store.includes('WORKBENCH_INITIAL_RIGHT_RATIO = 0.20')) failures.push('0.1.100 工作台首次布局保持 5% / 75% / 20%，并迁移到 v6 Snap Pane 状态')
  if (!store.includes('LEFT_PANE_MIN = 220') || !store.includes('setLeftSidebarWidth(value: number, persist = true)') || !store.includes('setRightSidebarWidth(value: number, persist = true)') || !store.includes('commitWorkbenchLayout')) failures.push('0.1.95 Split Pane 必须具备左栏可读最小宽度、左右独立拖拽与拖拽结束持久化')
  const xiaoyuWorkbench = fs.readFileSync(path.join(root, 'frontend/src/features/xiaoyu/AIWorkbenchView.vue'), 'utf8')
  if (!xiaoyuWorkbench.includes('Enter 发送 · Shift+Enter 换行') || !xiaoyuWorkbench.includes('handleComposerKeydown') || !xiaoyuWorkbench.includes('xy-message--assistant') || !xiaoyuWorkbench.includes('xy-chat-scroll')) failures.push('0.1.95 小鱼必须使用真正对话流，Enter 发送、Shift+Enter 换行')

  if (!store.includes("theme === 'system'") || !store.includes("prefers-color-scheme")) failures.push('0.1.64 Theme Source 缺少 system 实时解析')
  for (const variable of ['--bg-app', '--bg-sidebar', '--bg-topbar', '--surface-1', '--text-primary', '--input-bg']) if (!css.includes(variable)) failures.push(`0.1.64 Theme Token 缺少 ${variable}`)
  for (const selector of ['.workbench-shell', '.workbench-main', '.workbench-content', '.codex-sidebar', '.right-sidebar', '.codex-topbar', '.bottom-panel']) {
    if (!css.includes(`:root[data-theme="light"] ${selector}`) && !css.includes(`:root[data-theme="light"]\n${selector}`)) failures.push(`0.1.64 Light Theme Gate 缺少 App Shell 覆盖：${selector}`)
  }
  if (!css.includes('background:var(--bg-app)') && !css.includes('background: var(--bg-app)')) failures.push('0.1.64 App Shell 背景必须使用 --bg-app Token')
  if (!css.includes('background:var(--bg-sidebar)') && !css.includes('background: var(--bg-sidebar)')) failures.push('0.1.64 Sidebars 必须使用 --bg-sidebar Token')
  if (!css.includes('background:var(--bg-topbar)') && !css.includes('background: var(--bg-topbar)')) failures.push('0.1.64 Topbar 必须使用 --bg-topbar Token')

  const ui = JSON.parse(fs.readFileSync(path.join(root, 'configs/ui.json'), 'utf8'))
  const navItems = (ui.navigation ?? []).flatMap(group => group.items ?? [])
  if (navItems.some(item => item.id === 'terminal' || item.to === '/terminal')) failures.push('0.1.64 左侧导航禁止残留 terminal')

  const licenseService = fs.readFileSync(path.join(root, 'internal/system/license/service.go'), 'utf8')
  const licenseTests = fs.readFileSync(path.join(root, 'internal/system/license/service_test.go'), 'utf8')
  for (const field of ['LicenseID', 'ActivationID', 'MachineCode', 'InstallID', 'Features', 'SeatLimit']) if (!licenseService.includes(field)) failures.push(`0.1.64 License Domain 缺少 ${field}`)
  if (!licenseService.includes('BFLC2.')) failures.push('0.1.64 License Certificate 必须使用 BFLC2')
  if (!licenseService.includes('HasFeature')) failures.push('0.1.64 License 缺少统一 Feature Entitlement')
  for (const testName of ['TestCertificateRejectsTamperedSignature', 'TestCertificateRejectsWrongMachineAndInstallID', 'TestExpiredCertificateAndUnbind', 'TestShortActivationCodeRequiresOnlineServer']) if (!licenseTests.includes(testName)) failures.push(`0.1.64 License 回归测试缺少 ${testName}`)

  const envService = fs.readFileSync(path.join(root, 'internal/deploy/environment/service.go'), 'utf8')
  const envTests = fs.readFileSync(path.join(root, 'internal/deploy/environment/service_test.go'), 'utf8')
  for (const field of ['SteamCMDRoot', 'GameLibraryRoot', 'InstanceConfigRoot', 'GameSaveRoot', 'CacheRoot']) if (!envService.includes(field)) failures.push(`0.1.64 StoragePaths 缺少 ${field}`)
  if (!envService.includes('defaultGameLibraryRoot')) failures.push('0.1.64 Environment 缺少平台默认 GameLibraryRoot')
  if (!envTests.includes('TestMigrateGameLibraryFailureKeepsCurrentRoot')) failures.push('0.1.64 Environment 缺少迁移失败保持旧目录回归测试')
} catch (error) {
  failures.push(`无法验证 0.1.64 十项统一修复：${error instanceof Error ? error.message : String(error)}`)
}



// 0.1.69 Offline License Issuer Key Lifecycle：私钥只在仓库外生成，客户端内置公开密钥环，正式 Release 必须有 active 发行公钥。
try {
  const vendorKeys = fs.readFileSync(path.join(root, 'internal/system/license/vendor_keys.go'), 'utf8')
  const ring = JSON.parse(fs.readFileSync(path.join(root, 'internal/system/license/vendor_public_keys.json'), 'utf8'))
  const licenseService = fs.readFileSync(path.join(root, 'internal/system/license/service.go'), 'utf8')
  const licenseTests = fs.readFileSync(path.join(root, 'internal/system/license/service_test.go'), 'utf8')
  const admin = fs.readFileSync(path.join(root, 'cmd/aigame-manager-license-admin/main.go'), 'utf8')
  const rotate = fs.readFileSync(path.join(root, 'scripts/tools/license/rotate-release-key.ps1'), 'utf8')
  const issue = fs.readFileSync(path.join(root, 'scripts/tools/license/issue-bflc2.ps1'), 'utf8')
  const checks = fs.readFileSync(path.join(root, 'scripts/windows/lib/Checks.ps1'), 'utf8')
  const linuxBuild = fs.readFileSync(path.join(root, 'scripts/linux/actions/build.sh'), 'utf8')
  const linuxServerBuild = fs.readFileSync(path.join(root, 'scripts/linux/actions/build-server.sh'), 'utf8')
  if (!vendorKeys.includes('//go:embed vendor_public_keys.json')) failures.push('0.1.69 客户端必须通过 go:embed 内置公开发行密钥环')
  if (!vendorKeys.includes('TrustedVendorPublicKeys') || !licenseService.includes('TrustedPublicKeys')) failures.push('0.1.69 必须保留历史可信公钥，密钥轮换不得让旧 BFLC2 立即失效')
  if (!Array.isArray(ring.keys) || ring.keys.length < 1) failures.push('0.1.69 公开发行密钥环至少保留历史验证公钥')
  if (!ring.keys.some(entry => entry.status === 'legacy' || entry.status === 'retired' || entry.status === 'active')) failures.push('0.1.69 发行公钥环缺少合法状态')
  if (!admin.includes('keygen --output-dir DIR') || !admin.includes('pathInside(root, absOutput)')) failures.push('0.1.69 keygen 必须显式写入仓库外目录并阻止私钥落入源码树')
  if (!admin.includes('expected-key-id') || !admin.includes('发行私钥与当前 active 公钥不匹配')) failures.push('0.1.69 BFLC2 签发必须校验私钥与 active 公钥一致')
  if (!licenseService.includes('IssuerKeyID') || !licenseTests.includes('TestCertificateCarriesIssuerKeyID')) failures.push('0.1.69 BFLC2 必须记录发行 KeyID 并有回归测试')
  if (!licenseTests.includes('TestRetiredTrustedKeyStillVerifiesHistoricalCertificate')) failures.push('0.1.69 缺少历史可信公钥兼容回归测试')
  if (!rotate.includes('vendor_public_keys.json') || !rotate.includes('agmp-release-private.key') || !rotate.includes('LOCALAPPDATA')) failures.push('0.1.69 缺少本机发行密钥轮换工作流')
  if (!issue.includes('activeKeyID') || !issue.includes('IssuedLicenses')) failures.push('0.1.69 BFLC2 签发工具必须自动匹配 active KeyID 并把证书保存在仓库外')
  if (!checks.includes('Invoke-ReleaseKeyGate') || !checks.includes("check-release-key.mjs")) failures.push('0.1.69 Windows 正式 Release 缺少发行密钥门禁')
  if (!linuxBuild.includes('check-release-key.mjs') || !linuxServerBuild.includes('check-release-key.mjs')) failures.push('0.1.69 Linux 正式 Release 缺少发行密钥门禁')
} catch (error) {
  failures.push(`无法验证 0.1.69 Offline License Issuer Key Lifecycle：${error instanceof Error ? error.message : String(error)}`)
}


// 0.1.70 BFLC2 Activation Closure：签发后回验、跨源码目录恢复 Active Key、证书文件导入、原子持久化与签发 KeyID 真正绑定。
try {
  const licenseService = fs.readFileSync(path.join(root, 'internal/system/license/service.go'), 'utf8')
  const licenseTests = fs.readFileSync(path.join(root, 'internal/system/license/service_test.go'), 'utf8')
  const admin = fs.readFileSync(path.join(root, 'cmd/aigame-manager-license-admin/main.go'), 'utf8')
  const consoleScript = fs.readFileSync(path.join(root, 'scripts/tools/license/LicenseAdmin.ps1'), 'utf8')
  const issue = fs.readFileSync(path.join(root, 'scripts/tools/license/issue-bflc2.ps1'), 'utf8')
  const sync = fs.readFileSync(path.join(root, 'scripts/tools/license/sync-release-public-key.ps1'), 'utf8')
  const verify = fs.readFileSync(path.join(root, 'scripts/tools/license/verify-bflc2.ps1'), 'utf8')
  const licenseView = fs.readFileSync(path.join(root, 'frontend/src/features/settings/LicenseSection.vue'), 'utf8')
  if (!licenseService.includes('writeActivationRecord') || !licenseService.includes('platformfiles.AtomicReplace')) failures.push('0.1.70 activation.json 必须使用同目录临时文件原子替换，避免激活记录半写损坏')
  if (!licenseService.includes('signingKeyID') || !(licenseService.includes('strings.EqualFold(declared, signingKeyID)') || licenseService.includes('matchesPublicKeyID(signingPublicKey, declared)'))) failures.push('0.1.70 BFLC2 IssuerKeyID 必须与实际完成验签的 Ed25519 公钥一致（允许同一公钥的历史 KeyID 兼容别名）')
  if (!licenseTests.includes('TestCertificateRejectsSpoofedIssuerKeyID')) failures.push('0.1.70 缺少 IssuerKeyID 伪造回归测试')
  if (!licenseTests.includes('TestActivationPersistsAcrossServiceRestart')) failures.push('0.1.70 缺少许可证重启持久化回归测试')
  if (!licenseService.includes('machineCodeDerivationNamespace') || !licenseTests.includes('TestMachineCodeDerivationProtocolIsStable')) failures.push('0.1.70 必须冻结机器码派生协议，禁止未来品牌改名导致已签发许可证失效')
  if (!admin.includes('sync-public-key') || !admin.includes('newestValidKeyMetadata')) failures.push('0.1.70 发行工具必须支持从仓库外 ReleaseKeys 恢复 Active 公钥到新源码目录')
  if (!admin.includes('verify --ring') && !admin.includes('case "verify"')) failures.push('0.1.70 发行工具缺少 BFLC2 独立回验命令')
  if (!consoleScript.includes('同步本机发行公钥') || !consoleScript.includes('验证 BFLC2')) failures.push('0.1.70 发行控制台缺少公钥恢复 / BFLC2 回验入口')
  if (!issue.includes('签发后回验') || !issue.includes('aigame-manager-license-admin verify')) failures.push('0.1.70 BFLC2 签发完成后必须自动用公开密钥环回验')
  if (!sync.includes('sync-public-key') || !verify.includes('certificate-file')) failures.push('0.1.70 发行工具 PowerShell 工作流不完整')
  if (!licenseView.includes('type="file"') || !licenseView.includes('importCertificateFile')) failures.push('0.1.70 授权页必须支持直接导入 .bflc 文件')
} catch (error) {
  failures.push(`无法验证 0.1.70 BFLC2 Activation Closure：${error instanceof Error ? error.message : String(error)}`)
}

// 0.1.68 Windows GUI 无控制台闪窗 + License Snapshot 稳定性门禁。
try {
  const licenseService = fs.readFileSync(path.join(root, 'internal/system/license/service.go'), 'utf8')
  const machineWindows = fs.readFileSync(path.join(root, 'internal/system/license/machine_identity_windows.go'), 'utf8')
  const licenseView = fs.readFileSync(path.join(root, 'frontend/src/features/settings/LicenseSection.vue'), 'utf8')
  const backendTypes = fs.readFileSync(path.join(root, 'frontend/src/shared/types/backend.ts'), 'utf8')
  if (licenseService.includes('exec.Command("reg"') || licenseService.includes('"os/exec"')) failures.push('0.1.68 许可证服务禁止通过 reg.exe 子进程读取 MachineGuid')
  if (!machineWindows.includes('syscall.RegOpenKeyEx') || !machineWindows.includes('syscall.RegQueryValueEx')) failures.push('0.1.68 Windows MachineGuid 必须直接使用 Registry API')
  if (!licenseService.includes('machineCodeOnce') || !licenseService.includes('installIDOnce')) failures.push('0.1.68 MachineCode / InstallID 必须缓存，禁止每次状态刷新重新计算')
  if (!licenseView.includes('computed(() => app.license)') || !licenseView.includes('if (!app.license) void refresh()')) failures.push('0.1.68 LicenseSection 必须复用全局许可证快照，禁止页面切换时清空重读')
  if (backendTypes.includes('AI Game Manager PanelSettings') || !backendTypes.includes('interface AGMPSettings')) failures.push('0.1.68 TypeScript 设置类型必须是合法标识符 AGMPSettings')
  if (!licenseView.includes('在线 License Server 尚未上线') || !licenseView.includes('离线许可证证书（BFLC2）')) failures.push('0.1.68 授权页必须明确区分在线 CDK 与离线 BFLC2')
} catch (error) {
  failures.push(`无法验证 0.1.68 Console Flash / License Snapshot：${error instanceof Error ? error.message : String(error)}`)
}


// 0.1.71 Release / Feature Gate Hardening：正式构建自动恢复 Active 公钥，后端 RequireFeature 成为安全边界，并提供完整许可证验收入口。
try {
  const licenseService = fs.readFileSync(path.join(root, 'internal/system/license/service.go'), 'utf8')
  const licenseTests = fs.readFileSync(path.join(root, 'internal/system/license/service_test.go'), 'utf8')
  const app = readAppSources()
  const checks = fs.readFileSync(path.join(root, 'scripts/windows/lib/Checks.ps1'), 'utf8')
  const consoleScript = fs.readFileSync(path.join(root, 'scripts/tools/license/LicenseAdmin.ps1'), 'utf8')
  const acceptance = fs.readFileSync(path.join(root, 'scripts/tools/license/acceptance-test.ps1'), 'utf8')
  if (!licenseService.includes('RequireFeature(feature string) error') || !licenseService.includes('ErrFeatureNotEntitled')) failures.push('0.1.71 License Service 必须提供后端 RequireFeature 强制授权入口')
  if (!app.includes('RequireLicenseFeature(feature string) error')) failures.push('0.1.71 Application 必须暴露统一 RequireLicenseFeature 后端授权边界')
  if (!licenseTests.includes('TestStandardAndProFeatureMatrix') || !licenseTests.includes('TestRequireFeatureBlocksUnlicensedAndAllowsLicensed') || !licenseTests.includes('TestExpiredAndUnboundLicenseBlockFeatures')) failures.push('0.1.71 缺少 Standard/Pro/未授权/过期/解绑 Feature Gate 回归矩阵')
  if (!checks.includes('Invoke-EnsureReleasePublicKey') || !checks.includes('sync-release-public-key.ps1')) failures.push('0.1.71 Windows 正式 Release 必须在 Active Key 缺失时自动尝试同步本机仓库外公开密钥')
  if (!consoleScript.includes('许可证完整验收') || !consoleScript.includes('acceptance-test.ps1')) failures.push('0.1.71 许可证发行控制台缺少一键完整验收入口')
  if (!acceptance.includes('StandardAndProFeatureMatrix') || !acceptance.includes('CertificateRejectsWrongMachineAndInstallID') || !acceptance.includes('agmp_dev_license')) failures.push('0.1.71 许可证验收脚本必须覆盖授权矩阵、设备绑定拒绝与开发模式')
} catch (error) {
  failures.push(`无法验证 0.1.71 Release / Feature Gate Hardening：${error instanceof Error ? error.message : String(error)}`)
}


// 0.1.74 Agent-first：小鱼核心 Runtime 是智能交互执行层，手动 UI 只作为兜底。
try {
  const ai = JSON.parse(fs.readFileSync(path.join(root, 'configs/ai.json'), 'utf8'))
  const rustWorkspace = fs.readFileSync(path.join(root, 'rust/Cargo.toml'), 'utf8')
  const rustRuntime = fs.readFileSync(path.join(root, 'rust/crates/xiaoyu-core/src/lib.rs'), 'utf8')
  const rustCli = fs.readFileSync(path.join(root, 'rust/crates/xiaoyu-core/src/bin/xiaoyu.rs'), 'utf8')
  const appSource = readAppSources()
  const aiView = fs.readFileSync(path.join(root, 'frontend/src/features/xiaoyu/AIWorkbenchView.vue'), 'utf8')
  if (ai.primaryInteraction !== 'agent' || ai.runtime?.engine !== 'rust' || ai.runtime?.protocol !== 'xiaoyu.v1') failures.push('0.1.74 AI 必须以 小鱼核心 Runtime 为主交互执行层')
  if (ai.allowManualFallback !== true) failures.push('0.1.74 Agent-first 必须保留手动兜底')
  for (const token of ['xiaoyu-protocol', 'xiaoyu-core', 'xiaoyu-core']) if (!rustWorkspace.includes(token)) failures.push(`0.1.74 Rust workspace 缺少 ${token}`)
  for (const token of ['rust-agent-runtime-boundary', 'domain-provider-separation', 'tool-search-v1', 'host-tool-contracts', 'risk-aware-planning']) if (!rustRuntime.includes(token)) failures.push(`0.2.9 XiaoYu Rust Runtime 边界缺少 ${token}`)
  for (const token of ['Doctor', 'Tools', 'Rpc']) if (!rustCli.includes(token)) failures.push(`0.1.85 Rust CLI 缺少 ${token}`)
  for (const forbidden of ['Command::Tool {', 'Command::Exec {', 'tool/call']) if (rustCli.includes(forbidden)) failures.push(`0.1.85 Rust CLI 禁止恢复本地执行入口：${forbidden}`)
  const hostTools = fs.readFileSync(path.join(root, 'internal/app/app_xiaoyu_tools.go'), 'utf8')
  for (const token of ['system.info', 'fs.list', 'fs.read', 'fs.write', 'fs.replace', 'shell.exec', 'process.run']) if (!hostTools.includes(token)) failures.push(`0.1.85 AGMP Host Tool Registry 缺少 ${token}`)
  if (!appSource.includes('a.xiaoyuTools.Execute')) failures.push('0.1.85 AGMP Tool 调用必须经过 Host Registry Execute')
  const rustHelper = fs.readFileSync(path.join(root, 'scripts/windows/lib/Rust.ps1'), 'utf8')
  for (const token of ['Ensure-AGMPMSVCBuildTools', 'Microsoft.VisualStudio.Component.VC.Tools.x86.x64', 'Microsoft.VisualStudio.Component.Windows11SDK.22621', 'Get-AGMPWindowsSDKLib', 'kernel32.lib', 'Read-AGMPMSVCStoragePlan', 'Import-AGMPVisualStudioEnvironment', 'Get-AGMPMSVCBootstrapper', 'vs_BuildTools-2022.exe', 'Get-AuthenticodeSignature']) if (!rustHelper.includes(token)) failures.push(`0.1.77 Windows Rust/MSVC 工具链缺少 ${token}`)
  if (/['"]--includeRecommended['"]/.test(rustHelper) || /['"]Microsoft\.VisualStudio\.Workload\.VCTools['"]/.test(rustHelper)) failures.push('0.1.77 Windows Rust/MSVC 必须使用 Rust 官方最小组件，不得安装完整 VCTools workload / Recommended 全家桶')
  const tasksSource = fs.readFileSync(path.join(root, 'scripts/windows/tasks/Tasks.ps1'), 'utf8').replace(/^\uFEFF/, '')
  const initStart078 = tasksSource.indexOf('function Invoke-InitializeProject')
  const initEnd078 = tasksSource.indexOf('function Invoke-DevelopmentMenu', initStart078)
  const initBlock078 = initStart078 >= 0 && initEnd078 > initStart078 ? tasksSource.slice(initStart078, initEnd078) : ''
  if (!initBlock078 || initBlock078.includes('Ensure-AGMPRustToolchain') || initBlock078.includes('Ensure-AGMPMSVCBuildTools')) failures.push('0.1.79 基础开发环境必须与 Rust/MSVC 开发工具链隔离')
  const release078 = JSON.parse(fs.readFileSync(path.join(root, 'configs/release.json'), 'utf8'))
  if (release078.windows?.userDistribution?.precompiledAgentIncluded !== true || release078.windows?.userDistribution?.runtimeCompilation !== false || release078.windows?.userDistribution?.requiresMSVCBuildTools !== false) failures.push('0.1.79 普通用户 Windows 发行包必须携带预编译 Agent 且禁止运行时编译/MSVC 前置')
  if (!appSource.includes('XiaoYuRunCommand') || !appSource.includes('XiaoYuCallTool') || !appSource.includes('requireXiaoYuMember(token)')) failures.push('0.1.89 Go Core 缺少 License + 组织成员核心授权的 XiaoYu 后端授权桥接')
  if (!aiView.includes('小鱼是主入口') || !aiView.includes('XIAOYU · AGMP INTELLIGENCE CORE')) failures.push('0.1.81 小鱼必须明确为主智能入口')
} catch (error) {
  failures.push(`无法验证 0.1.74 小鱼核心 Runtime：${error instanceof Error ? error.message : String(error)}`)
}

if (failures.length > 0) {
  console.error('[ERROR] 项目骨架检查失败：')
  for (const failure of failures) console.error(`  - ${failure}`)
  process.exit(1)
}

console.log(`[OK] 项目骨架、配置中心与版本一致性检查通过：AGMP ${version}`)
