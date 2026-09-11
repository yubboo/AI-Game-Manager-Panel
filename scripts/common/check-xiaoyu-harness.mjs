import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'

const root = process.cwd()
const failures = []
const read = rel => fs.readFileSync(path.join(root, rel), 'utf8').replace(/^\uFEFF/, '')
const requireToken = (text, token, message) => { if (!text.includes(token)) failures.push(message) }

try {
  for (const rel of [
    'internal/xiaoyu/host/kernel.go','internal/xiaoyu/host/brain.go','internal/xiaoyu/host/loop.go',
    'internal/xiaoyu/host/runs.go','internal/xiaoyu/host/events.go','internal/xiaoyu/host/dsh.go',
    'internal/app/app_xiaoyu_harness.go','rust/crates/xiaoyu-core/src/lib.rs'
  ]) if (!fs.existsSync(path.join(root, rel))) failures.push(`XiaoYu Harness 缺少：${rel}`)

  const loop = read('internal/xiaoyu/host/loop.go')
  for (const token of ['MaxSteps','MaxToolCalls','MaxFailures','RepeatLimit','RunWaitingApproval','RunWaitingUser','RunPaused','ControlOwner','AgentPhase','PhaseUnderstanding','PhasePlanning','PhaseExecuting','PhaseVerifying','PhaseRecovering','DecisionSummary','CapabilityContext','UIRoute','WithStateSink','ThreadContext','WithThreadContext']) requireToken(loop, token, `Agent Loop 缺少安全/状态能力：${token}`)
  const runs = read('internal/xiaoyu/host/runs.go')
  for (const token of ['AdvanceAsync','Takeover','ReleaseToXiaoYu','context.WithCancel','ErrRunHumanOwned','ErrRunTerminal','publishActive','Steer(','ThreadContext(','用户实时纠正']) requireToken(runs, token, `RunManager 缺少服务器生命周期/人工接管能力：${token}`)
  const events = read('internal/xiaoyu/host/events.go')
  for (const token of ['maxHistory','Subscribe','Sequence','subscribers']) requireToken(events, token, `XiaoYu Trace/Event 缺少：${token}`)
  const kernel = read('internal/xiaoyu/host/kernel.go')
  const hostTypes = read('internal/xiaoyu/host/types.go')
  requireToken(hostTypes, 'PluginAPIVersion = "xiaoyu.plugin.v1"', 'Plugin Kernel 缺少稳定 API Version')
  for (const token of ['CapabilitySnapshot','Mount','Unmount']) requireToken(kernel, token, `Plugin Kernel 缺少：${token}`)
  const dsh = read('internal/xiaoyu/host/dsh.go')
  requireToken(dsh, 'deepseek-harness', '缺少 DeepSeek Harness compatibility bridge')
  for (const token of ['--permission','--allow-fs-read=','ReplaceEnvironment: true','filepath.EvalSymlinks','pathWithin']) requireToken(dsh, token, `DSH 兼容桥缺少第三方插件隔离：${token}`)
  const workbench = read('frontend/src/features/xiaoyu/AIWorkbenchView.vue')
  requireToken(workbench, 'XIAOYU_WORKBENCH_NO_STRETCH_CALLOUT', 'XiaoYu 工作台缺少配置提示防拉伸回归标记')
  requireToken(workbench, '.ai-workspace.agent-workspace{', 'XiaoYu 工作台必须覆盖历史三行 Grid')
  requireToken(workbench, 'display:flex;flex-direction:column', 'XiaoYu 工作台必须使用纵向 Flex，禁止配置提示占据 1fr')
  for (const token of ['xy-chat-scroll','XiaoYuAttachmentRequest','chooseImages','visionReady','xy-composer-dock','handleComposerKeydown','handleComposerFocus','terminalRunStatuses','event.shiftKey','Enter 发送 · Shift+Enter 换行','UI_NAVIGATION_BRIDGE','xy-agent-activity','decisionSummary','uiRoute: router.currentRoute.value.fullPath','settings/changed','Codex-style steering']) requireToken(workbench, token, `XiaoYu 对话工作台缺少：${token}`)
  const toolHost = read('internal/app/app_xiaoyu_tools.go')
  for (const token of ['Name: "ui.navigate"','Type: "ui/navigate"','settings.models','settings.environment','CapabilityUI','Name: "settings.get"','Name: "settings.theme.set"','Risk: xiaoyucontract.RiskModify','previousTheme','Type: "settings/changed"']) requireToken(toolHost, token, `XiaoYu UI Control 缺少：${token}`)
  const appShell = read('frontend/src/app/App.vue')
  const appStore = read('frontend/src/shared/store/app.ts')
  for (const token of ['setPointerCapture','pointercancel',"window.addEventListener('pointermove'",'commitWorkbenchLayout','dragLeftSidebarWidth','is-pane-snapping','is-left-collapsed']) requireToken(appShell, token, `Workbench Splitter 缺少稳定拖拽/吸附收起能力：${token}`)
  for (const token of ['LEFT_PANE_MIN = 220','LEFT_PANE_MAX = 520','RIGHT_PANE_MIN = 260','setLeftSidebarWidth','dragLeftSidebarWidth','collapseAt','reopenAt','setRightSidebarWidth',"WORKBENCH_LAYOUT_KEY = 'agmp.workbench.layout.v6'"]) requireToken(appStore, token, `Workbench 独立栏宽/吸附边界缺少：${token}`)
  const app = read('internal/app/app_xiaoyu_harness.go')
  for (const token of ['hostReceiptValidator','normalizeXiaoYuAttachments','AppendAttachments','XiaoYuStartRun','XiaoYuTakeoverRun','XiaoYuEventStream','xiaoyuAdvanceAfterSteer','WithThreadContext']) requireToken(app, token, `Application Harness 缺少：${token}`)
  const rust = read('rust/crates/xiaoyu-core/src/lib.rs')
  for (const token of ['prepare_brain','resolve_brain','rust-agent-runtime-boundary','ui.navigate','agent-kernel-v2','Goal-first','不要输出隐藏思维链','public_action_summary','settings.*','frame.thread.recentTurns','用户实时纠正','approvalId']) requireToken(rust, token, `Rust Agent Runtime 缺少：${token}`)
  const permissions = JSON.parse(read('configs/permissions.json'))
  if (permissions?.policies?.risk?.read !== 'allow' || permissions?.policies?.risk?.operate !== 'allow' || permissions?.policies?.risk?.modify !== 'allow' || permissions?.policies?.risk?.destructive !== 'confirm' || permissions?.policies?.risk?.system !== 'confirm') failures.push('帮我批准必须自动放行 read/operate/modify，仅 destructive/system 进入真实审批')
  const approvalMenu = read('frontend/src/features/xiaoyu/components/ApprovalMenu.vue')
  for (const token of ["document.addEventListener('pointerdown'", "document.addEventListener('keydown'", "event.key === 'Escape'", 'root.value?.contains(target)']) requireToken(approvalMenu, token, `审批模式菜单缺少点击外部/Escape 关闭：${token}`)
} catch (error) { failures.push(error instanceof Error ? error.message : String(error)) }

if (failures.length) {
  console.error('AGMP XiaoYu Harness Gate FAIL')
  failures.forEach(item => console.error(` - ${item}`))
  process.exit(1)
}
console.log('AGMP XiaoYu Harness Gate PASS (steerable thread · terminal turn boundary · three approval modes · outside-click close · Rust Agent Runtime)')
