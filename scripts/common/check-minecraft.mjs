import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'

const root = process.cwd()
const failures = []
const read = rel => fs.readFileSync(path.join(root, rel), 'utf8').replace(/^\uFEFF/, '')
const req = (text, token, message) => { if (!text.includes(token)) failures.push(message) }
const file = rel => { if (!fs.existsSync(path.join(root, rel))) failures.push(`缺少 Minecraft Vertical Slice 文件：${rel}`) }

try {
  for (const rel of [
    'internal/games/minecraft/model.go','internal/games/minecraft/facts.go','internal/games/minecraft/facts_test.go',
    'internal/games/minecraft/service.go','internal/games/minecraft/service_test.go','internal/games/minecraft/deploy_test.go','internal/games/minecraft/runtime.go','internal/games/minecraft/runtime_test.go','internal/games/minecraft/probe.go','internal/games/minecraft/probe_test.go',
    'internal/server/instance/store.go','internal/server/instance/store_test.go','internal/app/app_minecraft.go',
    'frontend/src/games/minecraft/MinecraftWorkspace.vue','frontend/src/games/minecraft/MinecraftInstance.vue',
  ]) file(rel)

  const facts = read('internal/games/minecraft/facts.go')
  for (const token of ['piston-meta.mojang.com','fill.papermc.io/v3','meta.fabricmc.net','javaVersion','sha256','SoftwareVanilla','SoftwarePaper','SoftwareFabric']) req(facts, token, `Minecraft 实时事实解析缺少：${token}`)
  const service = read('internal/games/minecraft/service.go')
  for (const token of ['EULAAccepted','OnlineMode == nil','InstallJava','ResolveRuntime','server.properties','eula=true','agmp-minecraft.json','server.jar','StartAfterDeploy','ensurePortAvailable','waitReady','Probe(','Instance{']) req(service, token, `Minecraft 部署闭环缺少：${token}`)
  req(service, '当前版本拒绝静默覆盖', 'Minecraft 部署必须 fail-closed，不能覆盖已有实例')
  req(service, 'failDeploymentValidation', 'Minecraft 自动部署验证失败后必须进入统一进程收尾路径')
  req(service, 'os.RemoveAll(plan.InstallPath)', 'GameInstance 持久化失败必须回滚新安装目录')
  const deployTests = read('internal/games/minecraft/deploy_test.go')
  for (const token of ['TestDeployPersistsVerifiedInstanceUsingManagedRuntime','TestDeployRollsBackInstallWhenInstancePersistenceFails','TestDeployStopsProcessWhenPingValidationFails']) req(deployTests, token, `Minecraft 真实部署 integration test 缺少：${token}`)
  const serviceTests = read('internal/games/minecraft/service_test.go')
  for (const token of ['TestPlanRejectsMissingOnlineModeBeforeFactLookup','TestPlanRejectsOfflineWithoutWhitelistBeforeFactLookup','TestEnsurePortAvailableRejectsOccupiedPort']) req(serviceTests, token, `Minecraft fail-closed 回归测试缺少：${token}`)
  const runtime = read('internal/games/minecraft/runtime.go')
  for (const token of ['platformruntime.NewSession','Done (','SendLine("stop")','logLimit']) req(runtime, token, `Minecraft Runtime 缺少：${token}`)
  const runtimeTests = read('internal/games/minecraft/runtime_test.go')
  req(runtimeTests, 'TestRuntimeReadyRequiresDoneAndHelpMarker', 'Minecraft Ready marker 缺少严格回归测试')
  const probe = read('internal/games/minecraft/probe.go')
  for (const token of ['writePacket','readVarInt','PlayersOnline','net.Dialer']) req(probe, token, `Minecraft 协议 Ping 缺少：${token}`)
  req(read('internal/games/minecraft/probe_test.go'), 'TestProbeReadsMinecraftStatusProtocol', 'Minecraft 协议 Ping 缺少真实 socket 回归测试')

  const tools = read('internal/app/app_xiaoyu_tools.go')
  for (const token of ['game.deploy.plan','game.deploy','minecraft.java','OriginAgent','"onlineMode"','"eulaAccepted"']) req(tools, token, `XiaoYu 一句话开服接线缺少：${token}`)
  req(tools, '"required": []string{"gameId", "name", "software", "onlineMode", "whitelist", "eulaAccepted", "autoInstallJava", "startAfterDeploy"}', 'XiaoYu game.deploy schema 必须显式要求认证模式/EULA/启动意图，禁止 bool 零值静默决策')
  const http = read('internal/bridge/httpapi/server.go')
  for (const token of ['/api/v1/minecraft/plan','/api/v1/minecraft/deploy','/api/v1/minecraft/instances/{id}/start','/probe']) req(http, token, `Web Minecraft API 缺少：${token}`)
  const wails = read('internal/bridge/wails/app.go')
  for (const token of ['GetMinecraftPlan','DeployMinecraft','StartMinecraft','StopMinecraft','ProbeMinecraft']) req(wails, token, `Desktop Minecraft API 缺少：${token}`)
  const ui = read('frontend/src/games/minecraft/MinecraftWorkspace.vue')
  for (const token of ['一句话与可视化共用部署内核','实时查证','Minecraft EULA','一键部署并验证','backend.deployMinecraft']) req(ui, token, `Minecraft 可视化部署 UI 缺少：${token}`)
  const instances = read('frontend/src/games/minecraft/MinecraftInstance.vue')
  for (const token of ['startMinecraft','stopMinecraft','probeMinecraft','最近控制台']) req(instances, token, `Minecraft 可视化管理缺少：${token}`)

  const games = JSON.parse(read('configs/games.json'))
  const mc = games.templates?.find(item => item.id === 'minecraft.java')
  if (!mc || mc.state !== 'supported') failures.push('Minecraft Vertical Slice 完成后 Game Pack 必须为 supported')
  for (const os of ['windows','linux']) if (!mc?.supportedOs?.includes(os)) failures.push(`Minecraft Game Pack 缺少原生 OS：${os}`)
  for (const source of ['mojang','papermc','fabric']) if (!mc?.factSources?.includes(source)) failures.push(`Minecraft Game Pack 缺少事实源：${source}`)
} catch (error) { failures.push(error instanceof Error ? error.message : String(error)) }

if (failures.length) {
  console.error('AGMP Minecraft Vertical Slice Gate FAIL')
  failures.forEach(item => console.error(` - ${item}`))
  process.exit(1)
}
console.log('AGMP Minecraft Vertical Slice Gate PASS (live facts · Java · verified download · port preflight · explicit auth/EULA · GameInstance · Done + MC Ping · visual/XiaoYu shared deploy)')
