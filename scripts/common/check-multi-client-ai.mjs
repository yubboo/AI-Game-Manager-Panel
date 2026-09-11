import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'

const root = process.cwd()
const failures = []
const read = rel => fs.readFileSync(path.join(root, rel), 'utf8').replace(/^\uFEFF/, '')
const need = (source, token, message) => { if (!source.includes(token)) failures.push(message) }

try {
  const http = read('internal/bridge/httpapi/server.go')
  const wails = read('internal/bridge/wails/app.go')
  const frontend = read('frontend/src/shared/api/backend.ts')
  const types = read('frontend/src/shared/types/backend.ts')
  const harness = read('internal/app/app_xiaoyu_harness.go')
  const runs = read('internal/xiaoyu/host/runs.go')
  const events = read('internal/xiaoyu/host/events.go')
  const hostTools = read('internal/app/app_xiaoyu_tools.go')
  const rust = read('rust/crates/xiaoyu-core/src/lib.rs')
  const release = JSON.parse(read('configs/release.json'))

  const controls = ['StartXiaoYuRun','ContinueXiaoYuRun','GetXiaoYuRunState','CancelXiaoYuRun','PauseXiaoYuRun','TakeoverXiaoYuRun','ResumeXiaoYuRun']
  for (const name of controls) {
    need(wails, name, `Wails 缺少 XiaoYu 能力：${name}`)
  }
  const httpRoutes = ['/api/v1/xiaoyu/runtime','/api/v1/xiaoyu/tools','/api/v1/xiaoyu/capabilities','/api/v1/xiaoyu/harness','/api/v1/xiaoyu/models','/api/v1/xiaoyu/runs','/api/v1/xiaoyu/events']
  for (const route of httpRoutes) need(http, route, `Web HTTP 缺少 XiaoYu 能力：${route}`)
  for (const action of ['continue','cancel','pause','takeover','resume']) need(http, `/{id}/${action}`, `Web HTTP 缺少 Run 控制：${action}`)
  for (const fn of ['xiaoyuStartRun','xiaoyuContinueRun','xiaoyuCancelRun','xiaoyuPauseRun','xiaoyuTakeoverRun','xiaoyuResumeRun','xiaoyuSubscribeEvents']) need(frontend, fn, `统一前端 Adapter 缺少：${fn}`)
  need(types, "'paused'", '前端 Run 状态必须支持 paused')
  need(types, "'xiaoyu' | 'human' | 'shared'", '前端必须暴露双大脑 ControlOwner')
  need(types, 'initiatorId?: string', '前端 Run 状态必须携带发起人边界')
  need(runs, 'CreateFor', 'RunManager 必须记录服务端 Run 发起人')
  need(runs, 'InitiatorID', 'RunState 必须记录发起人 ID')
  need(harness, 'canControlXiaoYuRun', '多用户 Web 必须限制 Run 控制权')
  need(harness, 'SubscribeFiltered', 'XiaoYu Event Stream 必须按账号过滤')
  need(events, 'sanitizeTraceEvent', 'XiaoYu Trace 必须在最终写入边界统一脱敏')
  need(hostTools, 'a.startDSTCluster(ctx, request)', 'XiaoYu DST 启动必须复用人与 AI 共用的领域动作')
  if (hostTools.includes('a.dstRuntime.StartCluster(ctx')) failures.push('XiaoYu Tool 禁止绕过统一领域动作直接调用 DST Runtime')
  need(harness, 'AdvanceAsync(a.Context()', 'Web/Desktop 自主 Run 必须绑定 AGMP Application 生命周期，而不是浏览器请求生命周期')
  need(harness, 'XiaoYuTakeoverRun', 'Application 缺少人工接管能力')
  need(harness, 'XiaoYuEventStream', 'Application 缺少服务器端 XiaoYu Event Stream')
  need(rust, 'rust-agent-runtime-boundary', 'Rust XiaoYu Core 必须保持 Agent Runtime 边界')

  const product = release.productArchitecture ?? {}
  if (product.model !== 'two-brains-one-body') failures.push('产品模型必须冻结为 two-brains-one-body')
  if (product.humanOverride !== 'always-wins') failures.push('人工接管必须永远优先于 XiaoYu')
  if (product.webAIFeatureParity !== 'required') failures.push('Web AI Feature Parity 必须为 required')
  for (const key of ['web','windowsWails','windowsElectron','linuxServerWeb','dockerWeb']) {
    const target = product.deliveryTargets?.[key]
    if (!target || target.kind !== 'complete-product-target') failures.push(`缺少完整产品目标：${key}`)
    if (target?.xiaoyu !== true) failures.push(`${key} 必须携带完整 XiaoYu 能力`)
  }

  const forbiddenRoots = [path.join(root,'internal','xiaoyu'), path.join(root,'rust','crates','xiaoyu-core')]
  for (const base of forbiddenRoots) {
    const stack = [base]
    while (stack.length) {
      const current = stack.pop()
      for (const entry of fs.readdirSync(current, { withFileTypes: true })) {
        const full = path.join(current, entry.name)
        if (entry.isDirectory()) stack.push(full)
        else if (/\.(go|rs)$/.test(entry.name)) {
          const text = fs.readFileSync(full, 'utf8')
          if (/desktop\/electron|internal\/bridge\/wails|wailsbridge/.test(text)) failures.push(`XiaoYu Core/Host 禁止依赖桌面 Adapter：${path.relative(root, full)}`)
        }
      }
    }
  }
} catch (error) {
  failures.push(error instanceof Error ? error.message : String(error))
}

if (failures.length) {
  console.error('AGMP Multi-Client AI Gate FAIL')
  failures.forEach(item => console.error(` - ${item}`))
  process.exit(1)
}
console.log('AGMP Multi-Client AI Gate PASS (two brains · one body · Web/Wails/Linux AI parity · human override)')
