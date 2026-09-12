import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'

const root = process.cwd()
const failures = []
const read = rel => fs.readFileSync(path.join(root, rel), 'utf8').replace(/^\uFEFF/, '')
const req = (text, token, message) => { if (!text.includes(token)) failures.push(message) }
const file = rel => { if (!fs.existsSync(path.join(root, rel))) failures.push(`缺少合同文件：${rel}`) }

try {
  for (const rel of [
    'internal/platform/contract/runtime.go',
    'internal/platform/contract/runtime_test.go',
    'internal/games/common/pack.go',
    'internal/server/instance/model.go',
    'internal/app/app_instances.go',
    'internal/app/app_games.go',
    'internal/xiaoyu/host/model_auth.go',
    'frontend/src/features/nodes/NodesView.vue',
    'frontend/src/features/instances/InstancesView.vue',
  ]) file(rel)

  const platform = read('internal/platform/contract/runtime.go')
  for (const token of ['SurfaceWeb','SurfaceDesktop','SurfaceRuntime','web-client','windows-desktop','macos-desktop','linux-runtime','ExecutesOnNode: false','State           string']) req(platform, token, `Platform Contract 缺少：${token}`)
  req(platform, 'State: "planned"', 'Platform Contract 必须诚实标记尚未完成的原生端')

  const pack = read('internal/games/common/pack.go')
  for (const token of ['type Pack struct','SupportedOS','Capabilities','UIPanels','InstallStrategy','FactSources','Executable() bool']) req(pack, token, `Game Pack Contract 缺少：${token}`)

  const instance = read('internal/server/instance/model.go')
  for (const token of ['type Instance struct','OriginVisual','OriginAgent','NodeOS','RuntimeState','Capabilities']) req(instance, token, `GameInstance Contract 缺少：${token}`)

  const appGames = read('internal/app/app_games.go')
  req(appGames, 'func (a *Application) GamePacks()', 'Application 必须公开共享 GamePacks')
  const appInstances = read('internal/app/app_instances.go')
  req(appInstances, 'func (a *Application) GameInstances()', 'Application 必须公开共享 GameInstances')

  const tools = read('internal/app/app_xiaoyu_tools.go')
  for (const token of ['game.pack.list','game.instance.list','a.GamePacks()','a.GameInstances()']) req(tools, token, `XiaoYu 与可视化资源合同未共用：${token}`)

  const http = read('internal/bridge/httpapi/server.go')
  for (const token of ['/api/v1/platform/runtime','/api/v1/game-packs','/api/v1/instances']) req(http, token, `Web Contract API 缺少：${token}`)
  const wails = read('internal/bridge/wails/app.go')
  for (const token of ['GetPlatformRuntimeContract','GetGamePacks','GetGameInstances']) req(wails, token, `Desktop Contract API 缺少：${token}`)

  const nodes = read('frontend/src/features/nodes/NodesView.vue')
  req(nodes, '控制端和执行端分离', '节点 UI 必须明确控制端与执行端分离')
  if (nodes.includes('本机 Windows')) failures.push('节点 UI 禁止硬编码“本机 Windows”')
  const instances = read('frontend/src/features/instances/InstancesView.vue')
  req(instances, 'GameInstance', '服务器 UI 必须采用共享 GameInstance 合同')

  const games = JSON.parse(read('configs/games.json'))
  const dst = games.templates?.find(item => item.id === 'steam.dst')
  if (!dst || dst.state !== 'supported' || !Array.isArray(dst.capabilities) || !Array.isArray(dst.uiPanels) || !Array.isArray(dst.supportedOs)) failures.push('DST Game Pack 必须声明真实支持状态/OS/能力/UI Panels')
  const mc = games.templates?.find(item => item.id === 'minecraft.java')
  if (!mc || mc.state !== 'supported' || !mc.supportedOs?.includes('windows') || !mc.supportedOs?.includes('linux')) failures.push('0.3.0 Minecraft Game Pack 必须在真实 Vertical Slice 落地后标记 supported，并声明 Windows/Linux 原生 Runtime')
} catch (error) {
  failures.push(error instanceof Error ? error.message : String(error))
}

if (failures.length) {
  console.error('AGMP Product Contracts Gate FAIL')
  failures.forEach(item => console.error(` - ${item}`))
  process.exit(1)
}
console.log('AGMP Product Contracts Gate PASS (Platform · Model Provider · Game Pack · GameInstance · shared visual/XiaoYu control)')
