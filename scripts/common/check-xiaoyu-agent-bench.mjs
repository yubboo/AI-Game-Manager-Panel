import fs from 'node:fs'
import path from 'node:path'

const root = process.cwd()
const failures = []
const read = rel => fs.readFileSync(path.join(root, rel), 'utf8').replace(/^\uFEFF/, '')
const need = (rel, token, message = `${rel} 缺少 ${token}`) => {
  if (!read(rel).includes(token)) failures.push(message)
}

try {
  const bench = 'internal/xiaoyu/host/bench_test.go'
  for (const test of [
    'TestAgentBenchRecoversFromDomainFailureWithFallback',
    'TestAgentBenchBlocksPrematureCompletionUntilVerified',
    'TestAgentBenchApprovalResumesExactToolCall',
  ]) need(bench, test, `XiaoYu Agent Bench 缺少场景：${test}`)

  need(bench, 'environment.remove_runtime', 'Agent Bench 必须覆盖 Domain Tool 失败后的自主恢复')
  need(bench, 'shell.exec', 'Agent Bench 必须覆盖通用 fallback 能力')
  need(bench, '完成判定暂缓', 'Agent Bench 必须覆盖 mutation 后验证门禁')
  need(bench, 'bench-approval', 'Agent Bench 必须覆盖审批挂起与原调用恢复')

  const workflow = read('.github/workflows/safety.yml')
  need('.github/workflows/safety.yml', 'XiaoYu Agent Bench', 'GitHub Actions 必须单独运行 XiaoYu Agent Bench')
  if (!workflow.includes("-run '^TestAgentBench'")) failures.push('CI Agent Bench 必须只筛选 TestAgentBench* 场景，便于单独诊断')
} catch (error) {
  failures.push(`XiaoYu Agent Bench Gate 检查失败：${error instanceof Error ? error.message : String(error)}`)
}

if (failures.length) {
  console.error('AGMP XiaoYu Agent Bench Gate FAIL')
  for (const failure of failures) console.error(` - ${failure}`)
  process.exit(1)
}
console.log('AGMP XiaoYu Agent Bench Gate PASS (recovery · verification · approval-resume)')
