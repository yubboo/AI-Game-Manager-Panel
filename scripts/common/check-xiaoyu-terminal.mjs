import fs from 'node:fs'
import path from 'node:path'

const root = process.cwd()
const failures = []
const read = rel => fs.readFileSync(path.join(root, rel), 'utf8').replace(/^\uFEFF/, '')

try {
  for (const rel of [
    'rust/crates/xiaoyu-core/src/terminal.rs',
    'rust/crates/xiaoyu-core/src/lib.rs',
    'rust/crates/xiaoyu-core/src/bin/xiaoyu.rs',
    'rust/crates/xiaoyu-protocol/src/lib.rs',
    'internal/xiaoyu/runtime/service.go',
    'internal/xiaoyu/runtime/service_test.go',
  ]) {
    if (!fs.existsSync(path.join(root, rel))) failures.push(`缺少 Rust Terminal Session 源码：${rel}`)
  }

  const terminal = read('rust/crates/xiaoyu-core/src/terminal.rs')
  for (const token of [
    'TerminalManager',
    'MAX_TERMINALS',
    'MAX_OUTPUT_BYTES',
    'MAX_WRITE_BYTES',
    'stdio-pipe-v1',
    'pub fn start',
    'pub fn write',
    'pub fn output',
    'pub fn close',
    'host_authorized',
    'terminal input requires Host authorization',
    'interactive_terminal_accepts_input_and_exposes_output',
  ]) {
    if (!terminal.includes(token)) failures.push(`Rust Terminal Session 缺少 ${token}`)
  }
  if (/portable[_-]pty|ConPTY|CreatePseudoConsole|openpty/.test(terminal)) {
    failures.push('0.2.12 不得把 stdio Terminal Session 伪装成已完成 Native PTY/ConPTY')
  }

  const protocol = read('rust/crates/xiaoyu-protocol/src/lib.rs')
  for (const token of [
    'TerminalState',
    'TerminalStartRequest',
    'TerminalSnapshot',
    'TerminalWriteRequest',
    'TerminalOutputRequest',
    'TerminalOutputResponse',
  ]) {
    if (!protocol.includes(token)) failures.push(`xiaoyu.v1 Terminal 协议缺少 ${token}`)
  }

  const cli = read('rust/crates/xiaoyu-core/src/bin/xiaoyu.rs')
  for (const method of [
    '"terminal/start"',
    '"terminal/get"',
    '"terminal/list"',
    '"terminal/write"',
    '"terminal/output"',
    '"terminal/close"',
  ]) {
    if (!cli.includes(method)) failures.push(`Rust JSON-RPC 缺少 ${method}`)
  }

  const core = read('rust/crates/xiaoyu-core/src/lib.rs')
  for (const capability of [
    'interactive-terminal-v1',
    'authorized-terminal-input',
    'bounded-terminal-output',
  ]) {
    if (!core.includes(capability)) failures.push(`Rust Runtime capability 缺少 ${capability}`)
  }

  const service = read('internal/xiaoyu/runtime/service.go')
  for (const token of [
    'func (s *Service) StartTerminal(',
    'func (s *Service) WriteTerminal(',
    'func (s *Service) TerminalOutput(',
    'func (s *Service) CloseTerminal(',
    'XiaoYu Rust Terminal 必须先经过 Host 授权',
    'XiaoYu Rust Terminal 输入必须先经过 Host 授权',
  ]) {
    if (!service.includes(token)) failures.push(`Go↔Rust Terminal Bridge 缺少 ${token}`)
  }

  const tests = read('internal/xiaoyu/runtime/service_test.go')
  if (!tests.includes('TestStartTerminalRejectsMissingHostAuthorizationBeforeRuntime')) failures.push('Go Terminal Bridge 缺少启动授权前置拒绝测试')
  if (!tests.includes('TestWriteTerminalRejectsMissingHostAuthorizationBeforeRuntime')) failures.push('Go Terminal Bridge 缺少输入授权前置拒绝测试')

  const workflow = read('.github/workflows/safety.yml')
  if (!workflow.includes('check-xiaoyu-terminal.mjs')) failures.push('GitHub Actions 必须执行 XiaoYu Terminal Session Gate')
} catch (error) {
  failures.push(`XiaoYu Terminal Session Gate 检查失败：${error instanceof Error ? error.message : String(error)}`)
}

if (failures.length) {
  console.error('AGMP XiaoYu Terminal Session Gate FAIL')
  failures.forEach(item => console.error(` - ${item}`))
  process.exit(1)
}

console.log('AGMP XiaoYu Terminal Session Gate PASS (interactive stdio session · per-input Host authorization · bounded output · PTY claim reserved)')
