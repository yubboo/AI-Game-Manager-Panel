import fs from 'node:fs'
import path from 'node:path'

const root = process.cwd()
const failures = []
const read = rel => fs.readFileSync(path.join(root, rel), 'utf8').replace(/^\uFEFF/, '')

try {
  for (const rel of [
    'internal/xiaoyu/control/capability_lease.go',
    'internal/xiaoyu/control/capability_lease_test.go',
    'internal/app/app_xiaoyu.go',
    'internal/app/app_xiaoyu_tool_context.go',
    'internal/app/app_xiaoyu_terminal.go',
    'internal/app/app_xiaoyu_terminal_test.go',
    'internal/xiaoyu/runtime/service.go',
    'rust/crates/xiaoyu-protocol/src/lib.rs',
    'rust/crates/xiaoyu-core/src/terminal.rs',
  ]) {
    if (!fs.existsSync(path.join(root, rel))) failures.push(`缺少 Capability Lease 源码：${rel}`)
  }

  const lease = read('internal/xiaoyu/control/capability_lease.go')
  for (const token of [
    'DefaultCapabilityLeaseTTL = 30 * time.Second',
    'type CapabilityLease struct',
    'type CapabilityLeaseStore struct',
    'leases map[string]CapabilityLease',
    'func NewCapabilityLeaseStore(',
    'func (s *CapabilityLeaseStore) Issue(',
    'func (s *CapabilityLeaseStore) Consume(',
    'ErrLeaseExpired',
    'ErrLeaseMismatch',
    'ErrLeaseConsumed',
    'RequestHash string',
    'ExpiresAt',
    'MaxUses',
  ]) {
    if (!lease.includes(token)) failures.push(`Capability Lease Store 缺少 ${token}`)
  }
  if (lease.includes('os.WriteFile') || lease.includes('AtomicReplace(')) failures.push('Capability Lease 必须是内存态；Host 重启必须自动撤销租约')

  const leaseTests = read('internal/xiaoyu/control/capability_lease_test.go')
  for (const test of [
    'TestCapabilityLeaseIsBoundSingleUseAndNotReplayable',
    'TestCapabilityLeaseExpiresAndDoesNotPersistAuthority',
    'TestCapabilityLeaseBindsRunAndPrincipal',
  ]) {
    if (!leaseTests.includes(test)) failures.push(`Capability Lease 缺少回归测试 ${test}`)
  }

  const app = read('internal/app/app_xiaoyu.go')
  for (const token of [
    'spec.Name == "shell.exec"',
    'approvedAgentLeaseHash(runID, command, cwd)',
    'a.xiaoyuLeases.Issue(',
    'tool/capability-lease-issued',
  ]) {
    if (!app.includes(token)) failures.push(`Host Tool 审批→租约签发缺少 ${token}`)
  }

  const terminal = read('internal/app/app_xiaoyu_terminal.go')
  for (const token of [
    'approvedAgentLeaseScope',
    'invocation.Lease == nil',
    'a.xiaoyuLeases.Consume(',
    'approvedAgentLeaseHash(invocation.RunID, command, cwd)',
    'CapabilityLeaseID: invocation.Lease.ID',
    'HostAuthorized:    true',
  ]) {
    if (!terminal.includes(token)) failures.push(`Native Terminal Lease Gate 缺少 ${token}`)
  }
  if (terminal.indexOf('a.xiaoyuLeases.Consume(') > terminal.indexOf('StartTerminal(ctx,')) failures.push('Capability Lease 必须在 Native Terminal 启动前原子消费')

  const appTests = read('internal/app/app_xiaoyu_terminal_test.go')
  for (const test of [
    'TestApprovedAgentShellUsesSingleUseLeaseAndNativeTerminal',
    'TestApprovedAgentShellRejectsMissingOrMismatchedLease',
  ]) {
    if (!appTests.includes(test)) failures.push(`Approved Agent Lease wiring 缺少测试 ${test}`)
  }

  const service = read('internal/xiaoyu/runtime/service.go')
  if (!service.includes('CapabilityLeaseID string')) failures.push('Go↔Rust Terminal Bridge 缺少 capabilityLeaseId')
  if (!service.includes('必须携带短时 Capability Lease')) failures.push('Go↔Rust Terminal Bridge 必须 fail-closed 拒绝缺少租约的 HostAuthorized start')

  const protocol = read('rust/crates/xiaoyu-protocol/src/lib.rs')
  if (!protocol.includes('pub capability_lease_id: String')) failures.push('xiaoyu.v1 TerminalStartRequest 缺少 capabilityLeaseId')
  const rustTerminal = read('rust/crates/xiaoyu-core/src/terminal.rs')
  if (!rustTerminal.includes('request.capability_lease_id.trim().is_empty()')) failures.push('Rust Native Terminal 必须拒绝空 Capability Lease ID')
  if (!rustTerminal.includes('terminal_start_requires_capability_lease')) failures.push('Rust Native Terminal 缺少 Capability Lease fail-closed 测试')

  const workflow = read('.github/workflows/safety.yml')
  if (!workflow.includes('check-xiaoyu-lease.mjs')) failures.push('GitHub Actions 必须执行 XiaoYu Capability Lease Gate')
} catch (error) {
  failures.push(`XiaoYu Capability Lease Gate 检查失败：${error instanceof Error ? error.message : String(error)}`)
}

if (failures.length) {
  console.error('AGMP XiaoYu Capability Lease Gate FAIL')
  failures.forEach(item => console.error(` - ${item}`))
  process.exit(1)
}

console.log('AGMP XiaoYu Capability Lease Gate PASS (ephemeral · exact fingerprint · Run/principal bound · single-use · Rust handoff)')
