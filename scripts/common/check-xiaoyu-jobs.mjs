import fs from 'node:fs'
import path from 'node:path'

const root = process.cwd()
const failures = []
const read = rel => fs.readFileSync(path.join(root, rel), 'utf8').replace(/^\uFEFF/, '')
const need = (rel, token, message = `${rel} 缺少 ${token}`) => {
  if (!read(rel).includes(token)) failures.push(message)
}

try {
  for (const rel of [
    'rust/crates/xiaoyu-core/src/session.rs',
    'rust/crates/xiaoyu-core/src/jobs.rs',
    'rust/crates/xiaoyu-protocol/src/lib.rs',
  ]) {
    if (!fs.existsSync(path.join(root, rel))) failures.push(`缺少 Rust Session/Job Runtime 源码：${rel}`)
  }

  const jobs = read('rust/crates/xiaoyu-core/src/jobs.rs')
  for (const token of [
    'MAX_JOBS',
    'MAX_OUTPUT_BYTES',
    'host_authorized',
    'Command::new',
    'resolve_cwd',
    'pub fn cancel',
    'pub fn output',
    'spawn_monitor',
    'running_job_can_be_cancelled',
    'read(&mut chunk)',
  ]) {
    if (!jobs.includes(token)) failures.push(`Rust Job Runtime 缺少 ${token}`)
  }
  if (!jobs.includes('if !request.host_authorized')) failures.push('Rust Job Runtime 必须拒绝未经 Host 授权的启动请求')
  if (!jobs.includes('output.truncated')) failures.push('Rust Job Runtime 必须暴露有界输出截断状态')

  const session = read('rust/crates/xiaoyu-core/src/session.rs')
  for (const token of ['SessionManager', 'pub fn create', 'pub fn get', 'pub fn list', 'pub fn close', 'candidate.starts_with(&root)']) {
    if (!session.includes(token)) failures.push(`Rust Session Registry 缺少 ${token}`)
  }

  const protocol = read('rust/crates/xiaoyu-protocol/src/lib.rs')
  for (const token of ['JobState', 'JobStartRequest', 'JobSnapshot', 'JobOutputRequest', 'JobOutputResponse', 'host_authorized']) {
    if (!protocol.includes(token)) failures.push(`xiaoyu.v1 Job 协议缺少 ${token}`)
  }

  const cli = read('rust/crates/xiaoyu-core/src/bin/xiaoyu.rs')
  for (const method of ['"session/get"', '"session/list"', '"session/close"', '"jobs/start"', '"jobs/get"', '"jobs/list"', '"jobs/output"', '"jobs/cancel"']) {
    if (!cli.includes(method)) failures.push(`Rust JSON-RPC 缺少 ${method}`)
  }

  const core = read('rust/crates/xiaoyu-core/src/lib.rs')
  for (const capability of ['session-registry-v1', 'long-running-jobs-v1', 'cancellable-jobs', 'bounded-job-output']) {
    if (!core.includes(capability)) failures.push(`Rust Runtime capability 缺少 ${capability}`)
  }
  if (!core.includes('model-visible permission bypass')) failures.push('Job Runtime 必须明确不是模型绕过 Host Approval 的通道')

  const ownership = read('docs/architecture/LANGUAGE-OWNERSHIP.md')
  if (!ownership.includes('Session / Job') || !ownership.includes('Rust')) failures.push('Language Ownership 文档必须把 Session / Job 明确归属 Rust')

  const workflow = read('.github/workflows/safety.yml')
  if (!workflow.includes('check-xiaoyu-jobs.mjs')) failures.push('GitHub Actions 必须执行 XiaoYu Session/Job Gate')
} catch (error) {
  failures.push(`XiaoYu Session/Job Gate 检查失败：${error instanceof Error ? error.message : String(error)}`)
}

if (failures.length) {
  console.error('AGMP XiaoYu Session/Job Gate FAIL')
  failures.forEach(item => console.error(` - ${item}`))
  process.exit(1)
}

console.log('AGMP XiaoYu Session/Job Gate PASS (Rust sessions · long jobs · cancellation · bounded output · Host authorization)')
