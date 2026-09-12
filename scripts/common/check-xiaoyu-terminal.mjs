import fs from 'node:fs'
import path from 'node:path'

const root = process.cwd()
const failures = []
const read = rel => fs.readFileSync(path.join(root, rel), 'utf8').replace(/^\uFEFF/, '')

try {
  for (const rel of [
    'rust/crates/xiaoyu-core/src/terminal.rs',
    'rust/crates/xiaoyu-core/src/pty_linux.rs',
    'rust/crates/xiaoyu-core/src/pty_windows.rs',
    'rust/crates/xiaoyu-core/src/lib.rs',
    'rust/crates/xiaoyu-core/src/bin/xiaoyu.rs',
    'rust/crates/xiaoyu-protocol/src/lib.rs',
    'internal/xiaoyu/runtime/service.go',
    'internal/xiaoyu/runtime/service_test.go',
  ]) {
    if (!fs.existsSync(path.join(root, rel))) failures.push(`缺少 Rust Terminal/PTY 源码：${rel}`)
  }

  const terminal = read('rust/crates/xiaoyu-core/src/terminal.rs')
  for (const token of [
    'TerminalManager',
    'MAX_TERMINALS',
    'MAX_OUTPUT_BYTES',
    'MAX_WRITE_BYTES',
    'stdio-pipe-v1',
    'linux-pty-v1',
    'windows-conpty-v1',
    'pub fn start',
    'pub fn write',
    'pub fn output',
    'pub fn resize',
    'pub fn close',
    'host_authorized',
    'terminal input requires Host authorization',
    'terminal resize requires Host authorization',
    'interactive_terminal_accepts_input_and_exposes_output',
    'linux_terminal_is_backed_by_a_real_tty',
    'linux_terminal_resize_updates_kernel_winsize',
    'windows_terminal_conpty_accepts_io_and_resize',
  ]) {
    if (!terminal.includes(token)) failures.push(`Rust Terminal/PTY 缺少 ${token}`)
  }

  const pty = read('rust/crates/xiaoyu-core/src/pty_linux.rs')
  for (const token of [
    'posix_openpt',
    'grantpt',
    'unlockpt',
    'ptsname_r',
    'setsid',
    'TIOCSCTTY',
    'TIOCSWINSZ',
    'FD_CLOEXEC',
    'TERM',
    'xterm-256color',
  ]) {
    if (!pty.includes(token)) failures.push(`Linux Native PTY backend 缺少 ${token}`)
  }
  const windowsPty = read('rust/crates/xiaoyu-core/src/pty_windows.rs')
  for (const token of [
    'CreatePseudoConsole',
    'ResizePseudoConsole',
    'ClosePseudoConsole',
    'PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE',
    'InitializeProcThreadAttributeList',
    'UpdateProcThreadAttribute',
    'DeleteProcThreadAttributeList',
    'CreateProcessW',
    'EXTENDED_STARTUPINFO_PRESENT',
    'CREATE_UNICODE_ENVIRONMENT',
    'CreatePipe',
  ]) {
    if (!windowsPty.includes(token)) failures.push(`Windows ConPTY backend 缺少 ${token}`)
  }
  if (!windowsPty.includes('PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE as usize')) failures.push('Windows ConPTY ABI 必须把 PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE 转为 UpdateProcThreadAttribute 所需的 usize')
  if (!windowsPty.includes('let mut pseudo_console: HPCON = 0;')) failures.push('Windows ConPTY ABI 必须按 windows-sys 0.61.2 的 isize HPCON 使用 0 初始化')
  if (!windowsPty.includes('normalize_windows_current_directory')) failures.push('Windows ConPTY 必须在 CreateProcessW 边界规范化 verbatim cwd')
  if (!windowsPty.includes('strip_prefix(r"\\\\?\\")')) failures.push('Windows ConPTY 必须移除本地盘符 cwd 的 \\?\ verbatim 前缀后再传给子进程')
  if (!windowsPty.includes('windows_current_directory_removes_verbatim_prefix_before_create_process')) failures.push('Windows ConPTY 缺少 cwd verbatim-prefix 单元测试')

  if (!terminal.includes('fn terminal_newline(backend: &str)')) failures.push('Terminal Runtime 缺少 backend-aware newline 语义')
  if (!terminal.includes('return b"\\r";')) failures.push('Windows ConPTY appendNewline 必须发送 CR/Enter 语义')
  if (!terminal.includes('windows_conpty_append_newline_uses_cr')) failures.push('Windows ConPTY 缺少 CR newline 单元测试')
  if (!terminal.includes('AGMP-CONPTY-CWD:%CD%')) failures.push('Windows ConPTY integration 必须验证 CreateProcessW current directory')
  const staleResizeUnlock = `        let process = process
            .as_mut()
            .ok_or_else(|| anyhow::anyhow!("terminal process is unavailable"))?;
        process.resize(request.rows, request.cols)?;
        drop(process);`
  if (terminal.includes(staleResizeUnlock)) failures.push('Terminal resize 禁止 drop 已 shadow 为引用的 process；必须用 MutexGuard 作用域释放锁')

  const protocol = read('rust/crates/xiaoyu-protocol/src/lib.rs')
  for (const token of [
    'TerminalState',
    'TerminalStartRequest',
    'TerminalSnapshot',
    'TerminalWriteRequest',
    'TerminalOutputRequest',
    'TerminalOutputResponse',
    'TerminalResizeRequest',
    'pub rows: u16',
    'pub cols: u16',
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
    '"terminal/resize"',
    '"terminal/close"',
  ]) {
    if (!cli.includes(method)) failures.push(`Rust JSON-RPC 缺少 ${method}`)
  }

  const core = read('rust/crates/xiaoyu-core/src/lib.rs')
  for (const capability of [
    'interactive-terminal-v2',
    'authorized-terminal-input',
    'authorized-terminal-resize',
    'bounded-terminal-output',
    'linux-native-pty-v1',
    'windows-conpty-v1',
  ]) {
    if (!core.includes(capability)) failures.push(`Rust Runtime capability 缺少 ${capability}`)
  }

  const service = read('internal/xiaoyu/runtime/service.go')
  for (const token of [
    'func (s *Service) StartTerminal(',
    'func (s *Service) WriteTerminal(',
    'func (s *Service) TerminalOutput(',
    'func (s *Service) ResizeTerminal(',
    'func (s *Service) CloseTerminal(',
    'XiaoYu Rust Terminal 必须先经过 Host 授权',
    'XiaoYu Rust Terminal 输入必须先经过 Host 授权',
    'XiaoYu Rust Terminal 调整尺寸必须先经过 Host 授权',
  ]) {
    if (!service.includes(token)) failures.push(`Go↔Rust Terminal Bridge 缺少 ${token}`)
  }

  const tests = read('internal/xiaoyu/runtime/service_test.go')
  if (!tests.includes('TestStartTerminalRejectsMissingHostAuthorizationBeforeRuntime')) failures.push('Go Terminal Bridge 缺少启动授权前置拒绝测试')
  if (!tests.includes('TestWriteTerminalRejectsMissingHostAuthorizationBeforeRuntime')) failures.push('Go Terminal Bridge 缺少输入授权前置拒绝测试')
  if (!tests.includes('TestResizeTerminalRejectsMissingHostAuthorizationBeforeRuntime')) failures.push('Go Terminal Bridge 缺少 resize 授权前置拒绝测试')

  const workflow = read('.github/workflows/safety.yml')
  if (!workflow.includes('check-xiaoyu-terminal.mjs')) failures.push('GitHub Actions 必须执行 XiaoYu Terminal/PTY Gate')
  if (!workflow.includes('cargo test --manifest-path rust/Cargo.toml -p xiaoyu-core --locked linux_terminal_')) failures.push('GitHub Actions 必须执行 Linux Native PTY integration tests')
  if (!workflow.includes('Windows Rust Runtime + ConPTY')) failures.push('GitHub Actions 必须提供独立 Windows Rust Runtime + ConPTY Job')
  if (!workflow.includes('cargo test --manifest-path rust/Cargo.toml -p xiaoyu-core --locked windows_terminal_')) failures.push('GitHub Actions 必须执行 Windows ConPTY integration tests')
  const continueAfterFormat = (workflow.match(/if: \$\{\{ !cancelled\(\) \}\}/g) || []).length
  if (continueAfterFormat < 6) failures.push('Rust fmt 失败后仍必须继续执行 Linux/Windows check、PTY integration 与 workspace tests，避免串行隐藏真实编译问题')
} catch (error) {
  failures.push(`XiaoYu Terminal/PTY Gate 检查失败：${error instanceof Error ? error.message : String(error)}`)
}

if (failures.length) {
  console.error('AGMP XiaoYu Terminal/PTY Gate FAIL')
  failures.forEach(item => console.error(` - ${item}`))
  process.exit(1)
}

console.log('AGMP XiaoYu Terminal/PTY Gate PASS (Linux PTY · Windows ConPTY cwd/CR · resize · per-input Host authorization)')
