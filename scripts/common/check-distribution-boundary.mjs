import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'

const root = process.cwd()
const failures = []
const read = rel => fs.readFileSync(path.join(root, rel), 'utf8').replace(/^\uFEFF/, '')
const requireTrue = (value, message) => { if (!value) failures.push(message) }

try {
  const release = JSON.parse(read('configs/release.json'))
  const win = release.windows ?? {}
  const user = win.userDistribution ?? {}
  const publisher = win.publisherBuild ?? {}
  requireTrue(user.precompiledAgentIncluded === true && user.embeddedAICore === true, '普通用户发行包必须内置预编译 小鱼核心')
  requireTrue(user.runtimeCompilation === false, '普通用户安装阶段禁止现场编译 Runtime')
  for (const key of ['requiresRust','requiresCargo','requiresMSVCBuildTools','requiresGo','requiresNode']) {
    requireTrue(user[key] === false, `普通用户发行边界 ${key} 必须为 false`)
  }
  requireTrue(publisher.fromSource === true && publisher.buildsRustAgent === true, '发布者源码构建必须明确编译 小鱼核心')
  requireTrue(user.agentExposedToUser === false && user.standaloneAgentArtifact === false, '生产发行禁止把 Agent 暴露为独立用户产品')
  requireTrue(publisher.publishesStandaloneAgent === false, '发行构建禁止单独发布 Agent Asset')
  requireTrue(publisher.requiresRust === true && publisher.requiresCargo === true && publisher.requiresMSVCBuildTools === true,
    '发布者源码构建必须明确 Rust/Cargo/MSVC 前置')

  const iss = read('distribution/installer/windows/AIGameManagerPanel.iss')
  const files = iss.match(/\[Files\]([\s\S]*?)(?=\n\[[A-Za-z]+\])/i)?.[1] ?? ''
  requireTrue(files.includes('AI-Game-Manager-XiaoYu.exe'), 'Wails Setup 必须打包预编译 AI Core Runtime')
  requireTrue(files.includes('DestDir: "{app}\\internal\\xiaoyu"'), 'Wails Setup 必须将 AI Runtime 放入 AGMP 内部组件目录')
  requireTrue(iss.includes('普通用户安装器只包含完整 AGMP 的预编译运行产物'), 'Installer 缺少开发/生产边界声明')
  const forbidden = [
    /Source:\s*"[^"]*\\rust(?:\\|\")/i,
    /Cargo\.toml/i,
    /rustup(?:\.exe)?/i,
    /cargo\.exe/i,
    /rustc\.exe/i,
    /link\.exe/i,
    /vs_BuildTools/i,
    /Visual Studio Build Tools/i,
    /Source:\s*"[^"]*\\scripts\\windows/i,
  ]
  for (const rx of forbidden) if (rx.test(files)) failures.push(`普通用户 Wails Setup 禁止包含开发工具：${rx}`)

  const electron = read('desktop/electron/electron-builder.yml')
  requireTrue(electron.includes('AI-Game-Manager-XiaoYu.exe'), 'Electron 完整产品必须携带预编译 AI Core 内部组件')
  const electronForbidden = [/from:\s*\.\.\/\.\.\/rust\b/i, /vs_BuildTools/i, /rustup\.exe/i, /cargo\.exe/i, /rustc\.exe/i]
  for (const rx of electronForbidden) if (rx.test(electron)) failures.push(`普通用户 Electron 包禁止包含开发工具：${rx}`)

  const tasks = read('scripts/windows/tasks/Tasks.ps1')
  const initStart = tasks.indexOf('function Invoke-InitializeProject')
  const initEnd = tasks.indexOf('function Invoke-DevelopmentMenu', initStart)
  const init = initStart >= 0 && initEnd > initStart ? tasks.slice(initStart, initEnd) : ''
  requireTrue(Boolean(init), '无法定位基础开发环境初始化函数')
  requireTrue(!init.includes('Ensure-AGMPRustToolchain'), '基础开发环境初始化禁止自动安装 Rust')
  requireTrue(!init.includes('Ensure-AGMPMSVCBuildTools'), '基础开发环境初始化禁止自动安装 MSVC')
  requireTrue(init.includes('不会安装 Rust、Cargo、MSVC'), '基础开发初始化必须明确不安装 Rust/MSVC')

  const helper = read('scripts/windows/AIGameManagerPanel.ps1')
  requireTrue(helper.includes('普通用户不要运行本脚本'), '开发助手必须明确普通用户不要运行')
  requireTrue(helper.includes('无需 Rust/MSVC/Go/Node'), '开发助手必须明确普通用户无需编译工具链')

  const wails = read('scripts/windows/lib/Wails.ps1')
  requireTrue(wails.indexOf('Build-AGMPXiaoYuCore') < wails.indexOf('Build-WailsInstaller'), 'Wails Release 必须先编译 Agent，再生成 Setup')
} catch (error) {
  failures.push(`发行边界 Gate 检查失败：${error instanceof Error ? error.message : String(error)}`)
}

if (failures.length) {
  console.error('AGMP Distribution Boundary Gate FAIL')
  failures.forEach(item => console.error(` - ${item}`))
  process.exit(1)
}
console.log('AGMP Distribution Boundary Gate PASS (dev/publisher isolated · production ships complete AGMP with embedded AI Core)')
