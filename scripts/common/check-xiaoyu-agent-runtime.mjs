import fs from 'node:fs'
import path from 'node:path'

const root = process.cwd()
const failures = []
const read = rel => fs.readFileSync(path.join(root, rel), 'utf8').replace(/^\uFEFF/, '')
const need = (rel, token, message = `${rel} 缺少 ${token}`) => {
  if (!read(rel).includes(token)) failures.push(message)
}

try {
  const permissions = JSON.parse(read('configs/permissions.json'))
  const ai = JSON.parse(read('configs/ai.json'))
  if (permissions.allowArbitraryShell !== true) failures.push('XiaoYu Agent Runtime 默认必须具备可由审批策略控制的 shell.exec 能力')
  for (const mode of ['ask', 'risk', 'full']) if (!permissions.policies?.[mode]) failures.push(`缺少审批模式 ${mode}`)
  if ((ai.agentLoop?.maxSteps ?? 0) < 64 || (ai.agentLoop?.maxToolCalls ?? 0) < 48) failures.push('Agent 自主循环预算过小，会把强模型人为截断')

  const tools = read('internal/app/app_xiaoyu_tools.go')
  for (const token of ['Name: "shell.exec"', 'Name: "fs.write"', 'Name: "fs.replace"', 'Name: "fs.remove"', 'Name: "environment.remove_runtime"']) {
    if (!tools.includes(token)) failures.push(`XiaoYu 通用/闭环能力缺失：${token}`)
  }
  need('internal/xiaoyu/host/loop.go', 'toolGuidanceTier', 'Agent Loop 必须区分 domain/general/control/fallback guidance tier')
  need('internal/xiaoyu/host/loop.go', 'observationsForModel', 'Agent Loop 必须对模型上下文做 Observation 投影/压缩')
  need('internal/xiaoyu/host/intelligence.go', 'rankMemories', 'Memory 必须按目标相关性进入模型 Context')

  const rust = read('rust/crates/xiaoyu-core/src/lib.rs')
  for (const token of ['Guided Autonomy', '不是能力边界', 'shell.exec', 'ToolAllowlist', '不要为了“安全”主动装傻', 'Goal State']) {
    if (!rust.includes(token)) failures.push(`XiaoYu Brain 缺少 Guided Autonomy 规则：${token}`)
  }

  const model = read('internal/xiaoyu/host/models.go')
  const adapter = read('internal/xiaoyu/host/brain_model.go')
  for (const token of ['ThinkingToolChoiceCompatible', 'ReasoningReplay', 'NativeToolKinds']) if (!model.includes(token)) failures.push(`Provider capability matrix 缺少 ${token}`)
  if (!adapter.includes('!thinkingActive || ResolveModelCapabilities(p).ThinkingToolChoiceCompatible')) failures.push('DeepSeek thinking/tool_choice 必须由 capability negotiation 决定')

  need('docs/development/XIAOYU-AGENT-RUNTIME.md', 'Model Intelligence', '缺少 XiaoYu Agent Runtime 架构说明')
} catch (error) {
  failures.push(`XiaoYu Agent Runtime Gate 检查失败：${error instanceof Error ? error.message : String(error)}`)
}

if (failures.length) {
  console.error('AGMP XiaoYu Agent Runtime Gate FAIL')
  for (const failure of failures) console.error(` - ${failure}`)
  process.exit(1)
}
console.log('AGMP XiaoYu Agent Runtime Gate PASS (guided autonomy · general hands · approval authority · context compaction · native capability preservation)')
