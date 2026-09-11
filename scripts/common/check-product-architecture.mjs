import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'

const root = process.cwd()
const failures = []
const read = rel => fs.readFileSync(path.join(root, rel), 'utf8').replace(/^\uFEFF/, '')
const ok = (v,m) => { if (!v) failures.push(m) }
try {
  const rules = read('AGENTS.md')
  const projectRules = read('docs/development/PROJECT-RULES.md')
  const arch = read('docs/PROJECT-ARCHITECTURE.md')
  const release = JSON.parse(read('configs/release.json'))
  const ai = JSON.parse(read('configs/ai.json'))
  const product = release.productArchitecture ?? {}
  const envs = release.environments ?? {}
  ok(product.model === 'two-brains-one-body', '产品模型必须是 two-brains-one-body')
  ok(product.agentRole === 'xiaoyu-super-brain-assistant', 'XiaoYu 必须定义为 AGMP 内置智能大脑/助手')
  ok(product.standaloneAgentProduct === false, '禁止把 Agent 定义为独立产品')
  ok(product.sharedCapabilityLayer === true, 'AI 与人工必须共享能力层')
  for (const key of ['web','windowsWails','windowsElectron','windowsRustNative','linuxServerWeb','dockerWeb']) ok(product.deliveryTargets?.[key]?.kind === 'complete-product-target', `运行端 ${key} 必须是完整 AGMP 产品目标`)
  ok(product.humanOverride === 'always-wins', '双大脑架构必须保证 Human override 永远优先')
  ok(product.webAIFeatureParity === 'required', 'Web/Linux 不允许阉割 XiaoYu AI 能力')
  for (const key of ['web','windowsWails','windowsElectron','linuxServerWeb','dockerWeb']) ok(product.deliveryTargets?.[key]?.xiaoyu === true, `运行端 ${key} 必须具备 XiaoYu`) 
  ok(envs.development?.usesDeveloperHelper === true, '开发环境必须使用开发助手')
  ok(envs.production?.usesDeveloperHelper === false && envs.production?.runtimeCompilation === false && envs.production?.requiresDeveloperToolchain === false, '生产环境禁止开发助手/现场编译/开发工具链')
  ok(ai.productModel === 'embedded-xiaoyu-intelligence-core' && ai.runtime?.userFacingProduct === false, 'AI Runtime 必须是内置核心实现而非用户产品')
  const history = read('docs/PROJECT-HISTORY.md')
  ok(history.includes('## AI-Game-Manager-Panel 0.1.79'), 'PROJECT-HISTORY 必须保留 0.1.79 历史记录')
  for (const token of ['AI Game Manager Panel 是唯一产品主体','小鱼是 AGMP 内置的核心大脑','两个大脑，同一副身体','开发环境与生产环境硬边界']) ok(rules.includes(token), `AGENTS.md 缺少架构规则：${token}`)
  for (const token of ['AI Game Manager Panel（AGMP）是唯一产品本体','小鱼是 AGMP 内置的智能核心','两个大脑，同一副身体','开发环境与生产环境严格分离','0.1.83 完成核心目录定型后冻结骨架']) ok(projectRules.includes(token), `PROJECT-RULES.md 缺少强制规则：${token}`)
  ok(arch.includes('Web / Wails / Electron / Rust Native') && arch.includes('同一产品，不同运行端'), '项目架构文档缺少统一多端产品模型')

  const wails = read('scripts/windows/lib/Wails.ps1')
  const tasks = read('scripts/windows/tasks/Tasks.ps1')
  const iss = read('distribution/installer/windows/AIGameManagerPanel.iss')
  ok(!wails.includes("build\\release\\windows\\wails\\agent"), 'Wails Release 禁止单独发布 Agent 产物')
  ok(!wails.includes('$publishedAgent'), 'Wails Release 禁止 standalone Agent publish')
  ok(!wails.includes('$publishedDesktop') && !wails.includes('$publishedWeb'), 'build/release 只允许完整 Wails 产品包，不发布裸 Desktop/Web 内部构建物')
  ok(wails.includes("$internalAI = Join-Path $work 'internal\\xiaoyu'"), 'Wails Portable 必须把小鱼核心收口到 internal/xiaoyu')
  ok(!wails.includes("Join-Path $work 'AI-Game-Manager-XiaoYu.exe'"), 'Wails Portable 禁止把 Agent 放在产品根目录')
  ok(iss.includes('DestDir: "{app}\\internal\\xiaoyu"'), 'Wails Setup 必须把小鱼核心作为内部组件安装')
  ok(!/Name:\s*"[^"]*Agent/i.test(iss), '安装器禁止给 Agent 创建用户快捷方式')
  ok(!tasks.includes("@('Rust Agent','build\\release"), '状态页禁止把 Agent 作为独立用户 Release 展示')
} catch (e) { failures.push(`产品架构 Gate 检查失败：${e instanceof Error ? e.message : String(e)}`) }
if (failures.length) {
  console.error('AGMP Product Architecture Gate FAIL')
  failures.forEach(x => console.error(` - ${x}`))
  process.exit(1)
}
console.log('AGMP Product Architecture Gate PASS (single product · Human + XiaoYu · one body · multi-client · dev/prod isolated)')
