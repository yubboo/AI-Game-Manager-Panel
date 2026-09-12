import fs from 'node:fs'
import path from 'node:path'

const root = process.cwd()
const failures = []
const read = rel => fs.readFileSync(path.join(root, rel), 'utf8').replace(/^\uFEFF/, '')

const required = [
  'internal/xiaoyu/runtime/rpc_worker.go',
  'internal/xiaoyu/runtime/service.go',
  'internal/xiaoyu/runtime/service_test.go',
  'internal/app/app.go',
  'rust/crates/xiaoyu-core/src/lib.rs',
]
for (const rel of required) {
  if (!fs.existsSync(path.join(root, rel))) failures.push(`缺少 Persistent Runtime Worker 源码：${rel}`)
}

try {
  const worker = read('internal/xiaoyu/runtime/rpc_worker.go')
  for (const token of [
    'type rpcWorker struct',
    'startRPCWorker',
    'platformruntime.NewSession',
    'session.Subscribe(workerOutputBuffer)',
    'w.session.SendLine(string(raw))',
    'ctx.Done()',
    'w.session.Kill()',
    'platformruntime.OutputStderr',
    'workerStderrLimit',
    'type tailWriter struct',
  ]) {
    if (!worker.includes(token)) failures.push(`Persistent Runtime Worker 缺少 ${token}`)
  }
  if (/exec\.Command|\.(?:StdinPipe|StdoutPipe|StderrPipe)\s*\(/.test(worker)) {
    failures.push('XiaoYu Worker 不得绕过 platform/runtime 直接持有 OS process/stdio')
  }

  const service = read('internal/xiaoyu/runtime/service.go')
  for (const token of [
    'rpcMu      sync.Mutex',
    'worker     *rpcWorker',
    'func (s *Service) Start(',
    'func (s *Service) Close()',
    'startRPCWorker(path, s.root)',
    'func (s *Service) CreateSession(',
    'func (s *Service) StartJob(',
    'HostAuthorized bool',
    'XiaoYu Rust Job 必须先经过 Host 授权',
  ]) {
    if (!service.includes(token)) failures.push(`Go↔Rust Worker Bridge 缺少 ${token}`)
  }
  const runRpcStart = service.indexOf('func (s *Service) runRPC(')
  const runRpcEnd = service.indexOf('// Domain Tool dispatch')
  const runRpc = service.slice(runRpcStart, runRpcEnd)
  if (runRpc.includes('platformruntime.Run(')) failures.push('0.2.11 runRPC 不得继续为每次请求启动一次 Rust 进程')

  const app = read('internal/app/app.go')
  if (!app.includes('a.xiaoyuRuntime.Start(workerCtx)')) failures.push('Application Startup 必须预热 Persistent Rust Runtime Worker')
  if (!app.includes('a.xiaoyuRuntime.Close()')) failures.push('Application Shutdown 必须关闭 Persistent Rust Runtime Worker')

  const core = read('rust/crates/xiaoyu-core/src/lib.rs')
  if (!core.includes('persistent-rpc-worker-v1')) failures.push('Rust Runtime status 缺少 persistent-rpc-worker-v1 capability')

  const tests = read('internal/xiaoyu/runtime/service_test.go')
  if (!tests.includes('TestTailWriterKeepsBoundedSuffix')) failures.push('Persistent Worker 缺少 bounded stderr 单测')
  if (!tests.includes('TestStartJobRejectsMissingHostAuthorizationBeforeRuntime')) failures.push('Go Bridge 缺少 Host authorization 前置拒绝测试')

  const sync = read('sync-agmp.ps1')
  if (!sync.includes('/UNILOG:')) failures.push('源码同步必须使用 Robocopy Unicode Log，禁止中文路径控制台乱码')
  if (!sync.includes('Out-Null')) failures.push('源码同步必须抑制 Robocopy 原生 OEM 控制台输出')
} catch (error) {
  failures.push(`Persistent Worker Gate 检查失败：${error instanceof Error ? error.message : String(error)}`)
}

if (failures.length) {
  console.error('AGMP XiaoYu Persistent Worker Gate FAIL')
  failures.forEach(item => console.error(` - ${item}`))
  process.exit(1)
}

console.log('AGMP XiaoYu Persistent Worker Gate PASS (platform/runtime supervision · stateful sessions/jobs · timeout kill · bounded stderr · Unicode sync)')
