import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'

const root = process.cwd()
const fail = []
const p = rel => path.join(root, rel)
const exists = rel => fs.existsSync(p(rel))
const read = rel => fs.readFileSync(p(rel), 'utf8').replace(/^\uFEFF/, '')
const need = rel => { if (!exists(rel)) fail.push(`缺少模块归属文件：${rel}`) }
const has = (rel, token) => { if (!read(rel).includes(token)) fail.push(`${rel} 缺少规则：${token}`) }

// 0.1.85：继续冻结真实边界，并要求所有进程创建统一经过 platform/runtime。
for (const rel of [
  'docs/development/MODULES.md',
  'docs/development/PROJECT-RULES.md',
  'docs/PROJECT-ARCHITECTURE.md',
  'AGENTS.md',
  'configs/modules.json',
  'configs/permissions.json',
  'internal/xiaoyu/contract/tool.go',
  'internal/xiaoyu/control/permission.go',
  'internal/xiaoyu/control/approval_store.go',
  'internal/xiaoyu/runtime/service.go',
  'internal/games/doc.go',
  'internal/server/doc.go',
  'internal/ops/doc.go',
  'internal/deploy/doc.go',
  'internal/system/doc.go',
  'internal/platform/files/doc.go',
  'internal/platform/runtime/doc.go',
  'internal/platform/runtime/model.go',
  'internal/platform/runtime/session.go',
  'internal/platform/runtime/command.go',
  'internal/platform/runtime/run.go',
  'internal/platform/runtime/output.go',
  'internal/platform/runtime/shell.go',
  'internal/ops/files/service.go',
  'internal/app/app_xiaoyu_tools.go',
  'internal/platform/runtime/manager.go',
  'frontend/src/shared/components/ModulePlaceholderView.vue',
  'frontend/src/features/xiaoyu/AIWorkbenchView.vue',
  'frontend/src/features/terminal/TerminalPanel.vue',
]) need(rel)

try {
  has('docs/development/PROJECT-RULES.md', '模块自治与 XiaoYu 调用规则')
  has('docs/development/PROJECT-RULES.md', '请求批准 / 帮我批准 / 完全访问权限')
  has('docs/development/MODULES.md', '一个业务模块一个明确归属')
  has('docs/development/MODULES.md', 'AI 不直接接管模块私有数据')
  has('AGENTS.md', '一个业务模块一个明确归属')
  has('internal/xiaoyu/contract/tool.go', '必须调用现有 Service')

  const permissions = JSON.parse(read('configs/permissions.json'))
  if (permissions.allowArbitraryShell !== true) fail.push('默认 allowArbitraryShell 必须为 true；XiaoYu 需要受审批保护的通用 Shell 后备能力。')
  for (const mode of ['ask', 'risk', 'full']) {
    if (!permissions.policies?.[mode]) fail.push(`permissions.policies 缺少 ${mode}`)
  }

  const allowedInternalRoots = new Set(['app', 'bridge', 'config', 'xiaoyu', 'games', 'server', 'ops', 'deploy', 'system', 'platform'])
  for (const item of fs.readdirSync(p('internal'), { withFileTypes: true })) {
    if (item.isDirectory() && !allowedInternalRoots.has(item.name)) fail.push(`internal 出现未授权一级领域：${item.name}`)
  }
  for (const forbidden of ['internal/service', 'internal/core', 'internal/games/dst/service']) {
    if (exists(forbidden)) fail.push(`禁止恢复旧分散目录：${forbidden}`)
  }

  // XiaoYu Go 侧保留 Domain Contract、Host Control、兼容 Run/Provider 与 Rust Runtime Bridge；新的通用 Agent Runtime 能力优先 Rust。
  const allowedXiaoYu = new Set(['contract', 'control', 'host', 'runtime'])
  for (const item of fs.readdirSync(p('internal/xiaoyu'), { withFileTypes: true })) {
    if (item.isDirectory() && !allowedXiaoYu.has(item.name)) fail.push(`XiaoYu Go Bridge 出现未授权子域：${item.name}；新 Agent Runtime 子域应优先 Rust`)
  }

  // 平台层禁止恢复 0.1.83 已清理的一函数一目录微包。
  for (const forbidden of [
    'internal/platform/files/apppath', 'internal/platform/files/atomic', 'internal/platform/files/fs', 'internal/platform/files/archive',
    'internal/platform/http/url', 'internal/platform/http/client', 'internal/platform/http/websocket',
    'internal/platform/net/ports', 'internal/platform/net/firewall', 'internal/platform/net/network',
    'internal/platform/os/info', 'internal/platform/os/paths',
    'internal/platform/telemetry/logging', 'internal/platform/telemetry/metrics',
    'internal/platform/runtime/process', 'internal/platform/runtime/terminal',
    'internal/platform/security/credential',
  ]) if (exists(forbidden)) fail.push(`禁止恢复平台微包：${forbidden}`)

  // 除一级领域聚合包外，禁止只有 doc.go 的空 Go 包。planned 能力放 modules.json / games.json。
  const domainRoots = new Set(['internal/xiaoyu','internal/games','internal/server','internal/ops','internal/deploy','internal/system','internal/platform/runtime','internal/platform/security'])
  const walkDirs = dir => {
    for (const item of fs.readdirSync(dir, { withFileTypes: true })) {
      if (!item.isDirectory()) continue
      const full = path.join(dir, item.name)
      const rel = path.relative(root, full).replaceAll('\\','/')
      const goFiles = fs.readdirSync(full, { withFileTypes: true }).filter(x => x.isFile() && x.name.endsWith('.go')).map(x => x.name)
      if (goFiles.length === 1 && goFiles[0] === 'doc.go' && !domainRoots.has(rel)) fail.push(`禁止 doc.go 空占位包：${rel}`)
      walkDirs(full)
    }
  }
  walkDirs(p('internal'))

  // 前端 skeleton 统一由 ModulePlaceholderView + modules.json 渲染，不再建立无引用 module.ts/FeaturePlaceholder 包装页。
  const frontendRoot = p('frontend/src')
  const walkFront = dir => {
    for (const item of fs.readdirSync(dir, { withFileTypes: true })) {
      const full = path.join(dir, item.name)
      if (item.isDirectory()) walkFront(full)
      else if (item.isFile() && item.name === 'module.ts') fail.push(`禁止无引用 feature module.ts 占位：${path.relative(root, full)}`)
    }
  }
  walkFront(frontendRoot)
  if (exists('frontend/src/shared/components/FeaturePlaceholder.vue')) fail.push('占位页面必须统一使用 ModulePlaceholderView，禁止恢复第二套 FeaturePlaceholder。')
  for (const oldDir of ['frontend/src/features/home','frontend/src/features/servers','frontend/src/features/platform']) {
    if (exists(oldDir)) fail.push(`前端业务命名已收口，禁止恢复：${oldDir}`)
  }


  // 0.1.85：进程创建/stdio 管理只能存在于 platform/runtime。其他模块只能调用 Session/Run/StartDetached。
  const execScanRoots = ['internal/xiaoyu','internal/games','internal/server','internal/ops','internal/deploy','internal/system','internal/platform/files','internal/platform/http','internal/platform/net','internal/platform/os','internal/platform/security','internal/platform/telemetry']
  const scanDirectProcess = dir => {
    if (!fs.existsSync(dir)) return
    for (const item of fs.readdirSync(dir, { withFileTypes: true })) {
      const full = path.join(dir, item.name)
      if (item.isDirectory()) scanDirectProcess(full)
      else if (item.isFile() && item.name.endsWith('.go')) {
        const body = fs.readFileSync(full, 'utf8')
        if (/exec\.Command(?:Context)?\s*\(/.test(body)) fail.push(`禁止绕过 platform/runtime 直接创建进程：${path.relative(root, full)}`)
        if (/\.(?:StdinPipe|StdoutPipe|StderrPipe)\s*\(/.test(body)) fail.push(`禁止在 platform/runtime 外直接持有 stdio pipe：${path.relative(root, full)}`)
      }
    }
  }
  for (const rel of execScanRoots) scanDirectProcess(p(rel))

  // 产品缩写只允许 AGMP；旧缩写和历史旧品牌不能重新进入现役源码。
  const legacyUpper = 'AI' + 'GMP'
  const legacyLower = legacyUpper.toLowerCase()
  const textExt = new Set(['.go','.rs','.ts','.tsx','.vue','.js','.mjs','.cjs','.json','.toml','.yaml','.yml','.ps1','.bat','.iss','.txt'])
  const scanActive = dir => {
    if (!fs.existsSync(dir)) return
    for (const item of fs.readdirSync(dir, { withFileTypes: true })) {
      const full = path.join(dir, item.name)
      if (item.isDirectory()) {
        if (!['node_modules','build','target','dist','.git'].includes(item.name)) scanActive(full)
      } else if (item.isFile() && textExt.has(path.extname(item.name).toLowerCase())) {
        const body = fs.readFileSync(full, 'utf8').replace(/^\uFEFF/, '')
        if (body.includes(legacyUpper) || body.includes(legacyLower)) fail.push(`发现旧产品缩写残留：${path.relative(root, full)}`)
        const oldBrand = 'Bon' + 'fire'
        const oldBrandZh = String.fromCodePoint(0x7bdd, 0x706b)
        if (body.includes(oldBrand) || body.includes(oldBrandZh)) fail.push(`发现旧产品品牌残留：${path.relative(root, full)}`)
      }
    }
  }
  for (const rel of ['internal','rust','frontend/src','configs','scripts/common','desktop','distribution']) scanActive(p(rel))

  const runtime = read('rust/crates/xiaoyu-core/src/lib.rs')
  const cli = read('rust/crates/xiaoyu-core/src/bin/xiaoyu.rs')
  for (const token of ['rust-agent-runtime-boundary', 'domain-provider-separation', 'tool-search-v1']) {
    if (!runtime.includes(token)) fail.push(`XiaoYu Rust Runtime 缺少职责标记：${token}`)
  }
  for (const forbidden of ['Command::Tool {', 'Command::Exec {']) {
    if (cli.includes(forbidden)) fail.push(`XiaoYu CLI 暂不允许暴露用户直连执行命令：${forbidden}`)
  }
  const hostTools = read('internal/app/app_xiaoyu_tools.go')
  for (const required of ['system.info','fs.list','fs.read','fs.write','fs.replace','shell.exec','process.run']) {
    if (!hostTools.includes(required)) fail.push(`AGMP Host Tool Registry 缺少：${required}`)
  }
  if (!read('internal/app/app_xiaoyu.go').includes('a.xiaoyuTools.Execute')) fail.push('AGMP Tool 调用必须经过 Host Registry Execute')

  const scanNames = dir => {
    for (const item of fs.readdirSync(dir, { withFileTypes: true })) {
      const full = path.join(dir, item.name)
      if (item.isDirectory()) scanNames(full)
      else if (item.isFile() && item.name.length > 48) fail.push(`文件名过长（>48）：${path.relative(root, full)}`)
    }
  }
  for (const rel of ['internal','frontend/src','scripts/common']) scanNames(p(rel))

  const settingsDir = p('internal/system/settings')
  if (exists('internal/system/settings')) {
    for (const name of fs.readdirSync(settingsDir)) {
      if (/license|filemanager|nodes?/i.test(name)) fail.push(`internal/system/settings 出现疑似跨模块文件：${name}`)
    }
  }
} catch (error) {
  fail.push(error instanceof Error ? error.message : String(error))
}

if (fail.length) {
  console.error('AGMP Module Gate FAIL')
  fail.forEach(item => console.error(` - ${item}`))
  process.exit(1)
}

console.log('AGMP Module Gate PASS (domain aggregation · Rust-first Agent Runtime · Go Domain Host)')
