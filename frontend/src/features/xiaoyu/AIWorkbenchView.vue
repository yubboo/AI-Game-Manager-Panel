<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { backend } from '../../shared/api/backend'
import { useAppStore } from '../../shared/store/app'
import type { XiaoYuApprovalRequest, XiaoYuApprovalState, XiaoYuAttachmentRequest, XiaoYuHarnessSnapshot, XiaoYuRunState, XiaoYuRuntimeStatus, XiaoYuTraceEvent, AuthUser } from '../../shared/types/backend'
import ApprovalMenu from './components/ApprovalMenu.vue'

const app = useAppStore()
const router = useRouter()
const workbenchSessionStorageKey = 'agmp.xiaoyu.workbench.session'
function resolveWorkbenchSessionId() {
  try {
    const existing = window.sessionStorage?.getItem(workbenchSessionStorageKey)?.trim()
    if (existing) return existing
    const created = globalThis.crypto?.randomUUID?.() || `xy-session-${Date.now()}-${Math.random().toString(36).slice(2)}`
    window.sessionStorage?.setItem(workbenchSessionStorageKey, created)
    return created
  } catch {
    return `xy-session-${Date.now()}-${Math.random().toString(36).slice(2)}`
  }
}
const workbenchSessionId = resolveWorkbenchSessionId()
const prompt = ref('')
const messageScroller = ref<HTMLElement | null>(null)
const composerTextarea = ref<HTMLTextAreaElement | null>(null)
const imageInput = ref<HTMLInputElement | null>(null)
type PendingImage = XiaoYuAttachmentRequest & { id: string; size: number; previewUrl: string }
const pendingImages = ref<PendingImage[]>([])

type ChatMessage = { id: string; role: 'user' | 'assistant'; text: string; runId?: string }
const chatStorageKey = `agmp.xiaoyu.chat.${workbenchSessionId}`
function loadChatMessages(): ChatMessage[] {
  try {
    const parsed = JSON.parse(window.sessionStorage?.getItem(chatStorageKey) || '[]') as ChatMessage[]
    return Array.isArray(parsed) ? parsed.filter(item => item && (item.role === 'user' || item.role === 'assistant') && typeof item.text === 'string') : []
  } catch { return [] }
}
const messages = ref<ChatMessage[]>(loadChatMessages())
let activeAssistantMessageId = ''
function persistChatMessages() {
  try { window.sessionStorage?.setItem(chatStorageKey, JSON.stringify(messages.value.slice(-80))) } catch { /* best effort */ }
}
function createMessageId(prefix: string) { return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}` }
async function scrollConversationToBottom() { await nextTick(); const el = messageScroller.value; if (el) el.scrollTop = el.scrollHeight }
function appendMessage(message: ChatMessage) { messages.value.push(message); persistChatMessages(); void scrollConversationToBottom() }
function syncAssistantReply(run: XiaoYuRunState) {
  const text = run.message?.trim() || runNotice(run)
  let target = activeAssistantMessageId ? messages.value.find(item => item.id === activeAssistantMessageId) : undefined
  if (!target || target.role !== 'assistant' || target.runId !== run.id) {
    target = [...messages.value].reverse().find(item => item.role === 'assistant' && item.runId === run.id)
  }
  if (target) target.text = text
  else {
    target = { id: createMessageId('assistant'), role: 'assistant', text, runId: run.id }
    messages.value.push(target)
  }
  activeAssistantMessageId = target.id
  persistChatMessages()
  void scrollConversationToBottom()
}
const notice = ref('')
const runtimeStatus = ref<XiaoYuRuntimeStatus | null>(null)
const approvalState = ref<XiaoYuApprovalState | null>(null)
const pendingApprovals = ref<XiaoYuApprovalRequest[]>([])
const approvalSaving = ref(false)
const currentUser = ref<AuthUser | null>(null)
const harnessStatus = ref<XiaoYuHarnessSnapshot | null>(null)
const currentRun = ref<XiaoYuRunState | null>(null)
const runBusy = ref(false)
const trace = ref<XiaoYuTraceEvent[]>([])
let stopEventStream: (() => void) | null = null
let refreshRunTimer = 0
let lastTraceSequence = 0

const platform = computed(() => app.platformConfig)
const permissions = computed(() => platform.value?.permissions)
const workbench = computed(() => platform.value?.ui.workbench)
const suggestions = computed(() => workbench.value?.suggestions ?? [])
const runtimeReady = computed(() => Boolean(runtimeStatus.value?.ready))
const approvalMode = computed(() => approvalState.value?.mode ?? 'ask')
const canSetMode = computed(() => currentUser.value?.role === 'owner')
const canApprove = computed(() => currentUser.value?.role === 'owner' || currentUser.value?.role === 'administrator')
const terminalRunStatuses = new Set(['completed', 'failed', 'cancelled'])
const isTerminalRun = (run?: XiaoYuRunState | null) => Boolean(run && terminalRunStatuses.has(run.status))
const currentApproval = computed(() => {
  if (!currentRun.value || currentRun.value.status !== 'waiting_approval') return null
  const id = currentRun.value.approvalId?.trim()
  if (!id) return null
  return pendingApprovals.value.find(item => item.id === id) ?? null
})
const brainReady = computed(() => Boolean(harnessStatus.value?.brain?.ready))
const visionReady = computed(() => Boolean(harnessStatus.value?.brain?.capabilities?.vision))
const canSubmitMessage = computed(() => Boolean(brainReady.value && !runBusy.value && (prompt.value.trim() || pendingImages.value.length)))
const runActive = computed(() => Boolean(currentRun.value && !isTerminalRun(currentRun.value)))

onMounted(async () => {
  try { currentUser.value = await backend.currentUser() } catch { currentUser.value = null }
  await Promise.all([refreshRuntime(), refreshApproval(), refreshHarness()])
  lastTraceSequence = trace.value.reduce((max, item) => Math.max(max, item.sequence), 0)
  stopEventStream = backend.xiaoyuSubscribeEvents(lastTraceSequence, handleXiaoYuEvent, error => {
    // Event stream failure does not stop the server-owned Run. Normal refreshes
    // remain available and the user can reopen this page to reconnect.
    notice.value = error.message
  })
})

onBeforeUnmount(() => {
  if (refreshRunTimer) window.clearTimeout(refreshRunTimer)
  stopEventStream?.()
  stopEventStream = null
})

function handleXiaoYuEvent(event: XiaoYuTraceEvent) {
  lastTraceSequence = Math.max(lastTraceSequence, event.sequence)
  trace.value = [event, ...trace.value.filter(item => item.sequence !== event.sequence)].slice(0, 8)

  // UI_NAVIGATION_BRIDGE：XiaoYu 不模拟鼠标点击 DOM。Brain 调用受限 ui.navigate Tool，
  // Host 只广播白名单站内目标；只有发起这个 Run 的浏览器/WebView session 才执行路由切换。
  if (event.type === 'ui/navigate') {
    const sessionId = typeof event.data?.sessionId === 'string' ? event.data.sessionId : ''
    const path = typeof event.data?.path === 'string' ? event.data.path : ''
    if (sessionId === workbenchSessionId && path.startsWith('/') && !path.startsWith('//') && !path.includes('://')) {
      void router.push(path)
    }
    return
  }

  if (event.type === 'settings/changed') {
    const sessionId = typeof event.data?.sessionId === 'string' ? event.data.sessionId : ''
    const theme = typeof event.data?.theme === 'string' ? event.data.theme : ''
    if ((!sessionId || sessionId === workbenchSessionId) && (theme === 'light' || theme === 'dark' || theme === 'system')) {
      app.applyTheme(theme)
    }
    return
  }

  if (event.runId && currentRun.value && event.runId !== currentRun.value.id) return
  if (event.type.includes('approval') || event.type === 'tool/pending') void refreshApproval()
  if (refreshRunTimer) window.clearTimeout(refreshRunTimer)
  refreshRunTimer = window.setTimeout(() => {
    void refreshHarness()
  }, 180)
}

async function refreshRuntime() {
  try {
    runtimeStatus.value = await backend.xiaoyuRuntimeStatus()
  } catch (error) {
    notice.value = error instanceof Error ? error.message : String(error)
  }
}

async function refreshApproval() {
  try {
    approvalState.value = await backend.xiaoyuApprovalState()
    pendingApprovals.value = canApprove.value ? ((await backend.xiaoyuPendingApprovals()) ?? []) : []
  } catch (error) {
    notice.value = error instanceof Error ? error.message : String(error)
  }
}

function riskLabel(value: string) {
  return ({ read: '读取', operate: '普通操作', modify: '修改', destructive: '高风险', system: '系统' } as Record<string, string>)[value] ?? value
}

async function resolveInlineApproval(decision: 'approve' | 'reject') {
  const item = currentApproval.value
  if (!item || !canApprove.value || runBusy.value) return
  runBusy.value = true
  try {
    await backend.resolveXiaoYuApproval(item.id, decision)
    pendingApprovals.value = pendingApprovals.value.filter(value => value.id !== item.id)
    if (currentRun.value?.status === 'waiting_approval' && currentRun.value.approvalId === item.id) {
      currentRun.value = await backend.xiaoyuContinueRun(currentRun.value.id)
      syncAssistantReply(currentRun.value)
    }
    notice.value = decision === 'approve' ? '已批准，小鱼正在继续任务。' : '已拒绝，小鱼会依据拒绝结果重新规划。'
    await Promise.all([refreshApproval(), refreshHarness()])
  } catch (error) {
    notice.value = error instanceof Error ? error.message : String(error)
  } finally {
    runBusy.value = false
  }
}

async function refreshHarness() {
  try {
    const [status, events] = await Promise.all([backend.xiaoyuHarnessStatus(), backend.xiaoyuTrace(0, 30)])
    harnessStatus.value = status
    trace.value = events.slice(-8).reverse()
    if (!currentRun.value && status.runs.length) {
      currentRun.value = status.runs.find(item => !['completed', 'failed', 'cancelled'].includes(item.status)) ?? status.runs[0]
    } else if (currentRun.value) {
      currentRun.value = status.runs.find(item => item.id === currentRun.value?.id) ?? currentRun.value
    }
    if (currentRun.value) {
      if (!messages.value.some(item => item.runId === currentRun.value?.id)) {
        messages.value.push({ id: createMessageId('user'), role: 'user', text: currentRun.value.goal, runId: currentRun.value.id })
      }
      syncAssistantReply(currentRun.value)
      if (isTerminalRun(currentRun.value)) {
        // Codex-style turn boundary: a terminal Run is immutable. Merely
        // focusing/clicking the composer must never reactivate it.
        activeAssistantMessageId = ''
        if (notice.value.includes('正在继续任务') || notice.value.includes('重新规划')) notice.value = ''
      }
    }
  } catch (error) {
    notice.value = error instanceof Error ? error.message : String(error)
  }
}

async function setApprovalMode(mode: string) {
  if (!canSetMode.value || approvalSaving.value || mode === approvalMode.value) return
  approvalSaving.value = true
  try {
    approvalState.value = await backend.setXiaoYuApprovalMode(mode)
    notice.value = `小鱼权限已切换为“${approvalState.value.modeLabel}”。`
  } catch (error) {
    notice.value = error instanceof Error ? error.message : String(error)
  } finally {
    approvalSaving.value = false
  }
}

function readFileAsBase64(file: File): Promise<string> { return new Promise((resolve, reject) => { const reader = new FileReader(); reader.onerror = () => reject(reader.error || new Error('读取图片失败')); reader.onload = () => { const value = typeof reader.result === 'string' ? reader.result : ''; const comma = value.indexOf(','); if (comma < 0) reject(new Error('图片编码失败')); else resolve(value.slice(comma + 1)) }; reader.readAsDataURL(file) }) }
async function chooseImages(event: Event) { const input = event.target as HTMLInputElement; const files = [...(input.files ?? [])]; input.value = ''; if (!visionReady.value) { notice.value = '当前默认模型未声明 Vision 能力。'; return } let total = pendingImages.value.reduce((n, i) => n + i.size, 0); for (const file of files.slice(0, Math.max(0, 4 - pendingImages.value.length))) { if (!['image/png','image/jpeg','image/webp','image/gif'].includes(file.type) || file.size > 5*1024*1024) { notice.value = `图片不支持或超过 5MiB：${file.name}`; continue } if (total + file.size > 12*1024*1024) { notice.value = '单条消息图片总大小不能超过 12MiB。'; break } pendingImages.value.push({ id: `${file.name}-${file.size}-${file.lastModified}`, name: file.name, mediaType: file.type, dataBase64: await readFileAsBase64(file), size: file.size, previewUrl: URL.createObjectURL(file) }); total += file.size } }
function removePendingImage(id: string) { const index = pendingImages.value.findIndex(i => i.id === id); if (index >= 0) { URL.revokeObjectURL(pendingImages.value[index].previewUrl); pendingImages.value.splice(index, 1) } }
function clearPendingImages() { for (const item of pendingImages.value) URL.revokeObjectURL(item.previewUrl); pendingImages.value = [] }
function attachmentRequests(): XiaoYuAttachmentRequest[] { return pendingImages.value.map(({ name, mediaType, dataBase64 }) => ({ name, mediaType, dataBase64 })) }

function useSuggestion(value: string) {
  prompt.value = value
  notice.value = ''
  void nextTick(() => { composerTextarea.value?.focus(); resizeComposer() })
}

function resizeComposer() {
  const el = composerTextarea.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = `${Math.min(160, Math.max(54, el.scrollHeight))}px`
}

function handleComposerFocus() {
  // Focus is a pure UI action. It must never continue a completed/failed/
  // cancelled Run. Clear only stale transient notices from the previous turn.
  if (isTerminalRun(currentRun.value) && notice.value) notice.value = ''
}

function handleComposerKeydown(event: KeyboardEvent) {
  if (event.key !== 'Enter' || event.shiftKey || event.isComposing) return
  event.preventDefault()
  void submit()
}

async function submit() {
  const typedGoal = prompt.value.trim()
  if ((!typedGoal && !pendingImages.value.length) || runBusy.value) return
  const goal = typedGoal || (runActive.value ? '请查看我刚附加的图片，并结合当前任务继续处理。' : '请查看我附加的图片，并帮助我完成相关任务。')
  const attachments = attachmentRequests()
  if (!brainReady.value) {
    notice.value = harnessStatus.value?.brain?.message || '请先到系统设置 → 模型管理配置小鱼的大模型大脑。'
    return
  }
  if (currentRun.value?.status === 'paused' || currentRun.value?.controlOwner === 'human') {
    notice.value = '当前任务已由人工接管；请先点击“交还小鱼”，再继续给小鱼新的指令。'
    return
  }
  runBusy.value = true
  const currentIsLive = Boolean(currentRun.value && !isTerminalRun(currentRun.value))
  const continuingRunId = currentIsLive ? currentRun.value!.id : ''
  appendMessage({ id: createMessageId('user'), role: 'user', text: pendingImages.value.length ? `${goal}\n[图片 × ${pendingImages.value.length}]` : goal, ...(continuingRunId ? { runId: continuingRunId } : {}) })
  try {
    if (currentIsLive && currentRun.value) {
      // Codex-style steering: running/waiting/approval turns all accept a new
      // user instruction. Host cancels the current reasoning turn at a safe
      // boundary, invalidates stale approval, then replans the same Run.
      currentRun.value = await backend.xiaoyuContinueRun(currentRun.value.id, { message: goal, attachments })
      notice.value = currentRun.value.status === 'waiting_approval'
        ? '已收到你的纠正；旧审批会失效，小鱼将按新指令重新规划。'
        : ''
    } else {
      currentRun.value = await backend.xiaoyuStartRun({ goal, context: { sessionId: workbenchSessionId, uiRoute: router.currentRoute.value.fullPath }, attachments })
      const lastUser = [...messages.value].reverse().find(item => item.role === 'user' && !item.runId)
      if (lastUser) lastUser.runId = currentRun.value.id
    }
    prompt.value = ''
    clearPendingImages()
    resizeComposer()
    activeAssistantMessageId = createMessageId('assistant')
    messages.value.push({ id: activeAssistantMessageId, role: 'assistant', text: runNotice(currentRun.value), runId: currentRun.value.id })
    syncAssistantReply(currentRun.value)
    await refreshHarness()
  } catch (error) {
    notice.value = error instanceof Error ? error.message : String(error)
  } finally {
    runBusy.value = false
  }
}

async function cancelRun() {
  if (!currentRun.value || runBusy.value) return
  runBusy.value = true
  try {
    currentRun.value = await backend.xiaoyuCancelRun(currentRun.value.id)
    notice.value = ''
    syncAssistantReply(currentRun.value)
    await refreshHarness()
  } catch (error) {
    notice.value = error instanceof Error ? error.message : String(error)
  } finally {
    runBusy.value = false
  }
}

async function pauseRun() {
  if (!currentRun.value || runBusy.value) return
  runBusy.value = true
  try {
    currentRun.value = await backend.xiaoyuPauseRun(currentRun.value.id, { reason: '用户从工作台暂停 XiaoYu' })
    notice.value = ''
    syncAssistantReply(currentRun.value)
  } catch (error) {
    notice.value = error instanceof Error ? error.message : String(error)
  } finally { runBusy.value = false }
}

async function takeoverRun() {
  if (!currentRun.value || runBusy.value) return
  runBusy.value = true
  try {
    currentRun.value = await backend.xiaoyuTakeoverRun(currentRun.value.id, { reason: '用户主动接管当前任务' })
    notice.value = ''
    syncAssistantReply(currentRun.value)
  } catch (error) {
    notice.value = error instanceof Error ? error.message : String(error)
  } finally { runBusy.value = false }
}

async function resumeRun() {
  if (!currentRun.value || runBusy.value) return
  runBusy.value = true
  try {
    currentRun.value = await backend.xiaoyuResumeRun(currentRun.value.id)
    notice.value = ''
    syncAssistantReply(currentRun.value)
  } catch (error) {
    notice.value = error instanceof Error ? error.message : String(error)
  } finally { runBusy.value = false }
}

function runNotice(run: XiaoYuRunState) {
  if (run.status === 'completed') return run.message || '小鱼已经完成这个目标。'
  if (run.status === 'waiting_approval') return run.message || '小鱼执行到高风险步骤，正在等待你的批准。'
  if (run.status === 'waiting_user') return run.message || '小鱼需要你补充信息后再继续。'
  if (run.status === 'failed') return run.message || '这个自主任务没有完成，可以查看 Observation 和 Trace 定位原因。'
  if (run.status === 'paused') return run.message || '任务已暂停，等待人工处理或交还小鱼。'
  if (run.status === 'cancelled') return '任务已取消。'
  return run.decisionSummary?.trim() || `小鱼正在处理：第 ${run.step} 步，已调用 ${run.toolCalls} 次 Tool。`
}

function runPhaseLabel(phase?: string) {
  return ({ understanding: '理解目标', planning: '规划中', executing: '执行中', verifying: '验证中', recovering: '恢复中', waiting_approval: '等待批准', waiting_user: '等待补充', paused: '已暂停', completed: '已完成', failed: '失败', cancelled: '已取消' } as Record<string, string>)[phase || ''] ?? (phase || '处理中')
}

function runStatusLabel(status: string) {
  return ({ created: '已创建', running: '执行中', paused: '已暂停', waiting_approval: '等待批准', waiting_user: '等待补充', completed: '已完成', failed: '失败', cancelled: '已取消' } as Record<string, string>)[status] ?? status
}

</script>

<template>
  <section class="ai-shell-page xy-chat-page">
    <div class="ai-workspace agent-workspace xy-chat-workspace">
      <header class="xy-chat-header" aria-label="小鱼是主入口">
        <div>
          <span class="ai-kicker">XIAOYU · AGMP INTELLIGENCE CORE</span>
          <strong>小鱼</strong>
        </div>
        <div class="agent-runtime-badge" :class="{ ready: runtimeReady }">
          <span class="dot"></span>
          {{ runtimeReady ? `运行正常 · ${runtimeStatus?.version}` : '核心未就绪' }}
        </div>
      </header>

      <main class="xy-chat-surface">
        <div ref="messageScroller" class="xy-chat-scroll">
          <div class="xy-chat-thread">
            <div v-if="!messages.length" class="xy-chat-welcome">
              <div class="xy-chat-avatar">鱼</div>
              <h1>今天想一起做什么？</h1>
              <p>告诉小鱼你的目标。她会读取状态、分析问题、调用受控工具，并在高风险操作前征求你的决定。</p>
              <div class="ai-suggestions xy-chat-suggestions">
                <button v-for="item in suggestions" :key="item" type="button" @click="useSuggestion(item)">{{ item }}</button>
              </div>
            </div>

            <div v-for="message in messages" :key="message.id" class="xy-message" :class="`xy-message--${message.role}`">
              <div v-if="message.role === 'assistant'" class="xy-message__avatar">鱼</div>
              <div class="xy-message__body">
                <div class="xy-message__text">{{ message.text }}</div>
                <template v-if="message.role === 'assistant' && currentRun && message.runId === currentRun.id">
                  <div v-if="currentRun.decisionSummary && runActive" class="xy-agent-activity" :class="`phase-${currentRun.phase || 'planning'}`">
                    <span class="xy-agent-activity__dot"></span>
                    <strong>{{ runPhaseLabel(currentRun.phase) }}</strong>
                    <span>{{ currentRun.decisionSummary }}</span>
                  </div>
                  <div class="xy-run-meta">
                    <span :class="`run-${currentRun.status}`">{{ runStatusLabel(currentRun.status) }}</span>
                    <span>{{ runPhaseLabel(currentRun.phase) }}</span>
                    <span>步骤 {{ currentRun.step }}</span>
                    <span>Tool {{ currentRun.toolCalls }}</span>
                    <span v-if="currentRun.failures">失败 {{ currentRun.failures }}</span>
                  </div>
                  <details v-if="currentRun.observations?.length" class="xy-run-details">
                    <summary>查看执行详情</summary>
                    <div v-for="(item, index) in currentRun.observations.slice(-6)" :key="`${item.step}-${index}`" class="xy-run-observation">
                      <code>#{{ item.step }}{{ item.tool ? ` · ${item.tool}` : '' }}</code>
                      <span>{{ item.error || item.summary }}</span>
                    </div>
                  </details>
                  <div v-if="runActive" class="run-control-actions xy-run-actions">
                    <button v-if="currentRun.status !== 'paused' && currentRun.controlOwner !== 'human'" type="button" :disabled="runBusy" @click="pauseRun">暂停</button>
                    <button v-if="currentRun.controlOwner !== 'human'" type="button" :disabled="runBusy" @click="takeoverRun">人工接管</button>
                    <button v-if="currentRun.status === 'paused' || currentRun.controlOwner !== 'xiaoyu'" type="button" class="run-resume" :disabled="runBusy" @click="resumeRun">交还小鱼</button>
                    <button type="button" class="run-cancel" :disabled="runBusy" @click="cancelRun">取消任务</button>
                  </div>
                </template>
              </div>
            </div>
            <div v-if="runBusy" class="xy-message xy-message--assistant xy-message--thinking">
              <div class="xy-message__avatar">鱼</div>
              <div class="xy-thinking"><i></i><i></i><i></i><span>小鱼正在处理</span></div>
            </div>
          </div>
        </div>

        <div class="xy-composer-dock">
          <article v-if="currentApproval" class="xy-inline-approval" aria-live="polite">
            <div class="xy-inline-approval__main">
              <div class="xy-inline-approval__eyebrow"><span>等待你的决定</span><b>{{ riskLabel(currentApproval.risk) }}</b></div>
              <strong>{{ currentApproval.subject }}</strong>
              <p>{{ currentApproval.summary }}</p>
            </div>
            <div class="xy-inline-approval__actions">
              <button type="button" class="approve" :disabled="runBusy" @click="resolveInlineApproval('approve')">批准并继续</button>
              <button type="button" :disabled="runBusy" @click="resolveInlineApproval('reject')">拒绝</button>
            </div>
          </article>
          <div v-if="!brainReady" class="brain-config-callout xy-brain-callout">
            <div><strong>还没有可用的大模型大脑</strong><span>先配置 Provider / API / 本地模型，保存后小鱼会自动接入。</span></div>
            <button type="button" @click="router.push('/settings?section=models')">去配置</button>
          </div>
          <div v-if="notice" class="ai-notice xy-chat-notice">{{ notice }}</div>
          <div class="ai-composer xy-chat-composer" :class="{ disabled: !brainReady }">
            <input ref="imageInput" class="xy-image-input" type="file" accept="image/png,image/jpeg,image/webp,image/gif" multiple @change="chooseImages" />
            <div v-if="pendingImages.length" class="xy-image-strip"><div v-for="item in pendingImages" :key="item.id" class="xy-image-chip"><img :src="item.previewUrl" :alt="item.name || '图片'" /><span>{{ item.name || '图片' }}</span><button type="button" @click="removePendingImage(item.id)">×</button></div></div>
            <textarea
              ref="composerTextarea"
              v-model="prompt"
              rows="1"
              :disabled="!brainReady || runBusy"
              :placeholder="brainReady ? (workbench?.placeholder ?? '给小鱼发消息……') : '请先配置小鱼的大模型大脑'"
              @focus="handleComposerFocus"
              @input="resizeComposer"
              @keydown="handleComposerKeydown"
            ></textarea>
            <div class="ai-composer__footer xy-composer-footer">
              <div class="ai-composer__left">
                <button class="ai-add" type="button" :disabled="!brainReady || runBusy || !visionReady" :title="visionReady ? '附加图片 / 截图' : '当前模型未声明 Vision 能力'" @click="imageInput?.click()">＋</button>
                <ApprovalMenu
                  :model-value="approvalMode"
                  :modes="permissions?.approvalModes ?? []"
                  :disabled="!canSetMode"
                  :saving="approvalSaving"
                  @update:model-value="setApprovalMode"
                />
              </div>
              <div class="ai-composer__right">
                <span class="xy-key-hint">Enter 发送 · Shift+Enter 换行</span>
                <span class="xy-provider">{{ brainReady ? `${harnessStatus?.brain?.name || 'Brain'}${harnessStatus?.brain?.model ? ` · ${harnessStatus.brain.model}` : ''}` : 'Brain 未接入' }}</span>
                <button class="ai-send xy-send" type="button" :disabled="!canSubmitMessage" aria-label="发送给小鱼" title="发送 (Enter)" @click="submit">↑</button>
              </div>
            </div>
          </div>
          <div class="ai-manual-row xy-manual-row">
            <span>手动入口：</span>
            <button type="button" @click="router.push('/deployment')">可视化部署</button>
            <button type="button" @click="app.setBottomPanelOpen(true)">底部终端</button>
          </div>
        </div>
      </main>
    </div>
  </section>
</template>

<style scoped>
/* XIAOYU_WORKBENCH_NO_STRETCH_CALLOUT: 0.1.95 聊天工作台继续覆盖历史三行 Grid，任何提示都不得占据 1fr 主行。 */
.ai-workspace.agent-workspace{display:flex;flex-direction:column}
.xy-chat-page{height:100%;min-height:0;overflow:hidden}.xy-chat-workspace{width:100%;max-width:1180px;height:100%;min-height:0;margin:0 auto;padding:0 32px 18px;display:flex;flex-direction:column}.xy-chat-header{height:58px;flex:0 0 58px;display:flex;align-items:center;justify-content:space-between;gap:18px;border-bottom:1px solid var(--border-subtle)}.xy-chat-header>div:first-child{display:flex;align-items:baseline;gap:10px}.xy-chat-header strong{font-size:15px}.xy-chat-header .ai-kicker{font-size:9px}.agent-runtime-badge{display:flex;align-items:center;gap:7px;padding:6px 10px;border:1px solid var(--border-subtle);border-radius:999px;font-size:10px;color:var(--text-muted);background:var(--surface-1)}.agent-runtime-badge .dot{width:7px;height:7px;border-radius:50%;background:#8b5d5d}.agent-runtime-badge.ready{color:#9bcaa7}.agent-runtime-badge.ready .dot{background:#65c77e;box-shadow:0 0 10px rgba(101,199,126,.42)}
.xy-chat-surface{flex:1;min-height:0;display:flex;flex-direction:column}.xy-chat-scroll{flex:1;min-height:0;overflow:auto;overscroll-behavior:contain;scrollbar-gutter:stable;padding:24px 0 18px}.xy-chat-thread{width:min(860px,100%);min-height:100%;margin:0 auto;display:flex;flex-direction:column;gap:22px}.xy-chat-welcome{margin:auto;padding:36px 16px 54px;text-align:center}.xy-chat-avatar,.xy-message__avatar{display:grid;place-items:center;border:1px solid var(--border-default);background:var(--surface-2);color:var(--text-secondary);font-weight:700}.xy-chat-avatar{width:44px;height:44px;margin:0 auto 18px;border-radius:14px;font-size:15px}.xy-chat-welcome h1{margin:0;color:var(--text-primary);font-size:30px;letter-spacing:-.035em}.xy-chat-welcome p{max-width:620px;margin:12px auto 22px;color:var(--text-muted);font-size:12px;line-height:1.7}.xy-chat-suggestions{gap:9px}.xy-chat-suggestions button{font-size:11px;min-height:36px;padding:0 14px}
.xy-message{width:100%;display:flex;align-items:flex-start;gap:11px}.xy-message--user{justify-content:flex-end}.xy-message__avatar{width:28px;height:28px;flex:0 0 28px;border-radius:9px;font-size:10px}.xy-message__body{min-width:0;max-width:min(760px,calc(100% - 40px))}.xy-message--user .xy-message__body{max-width:min(680px,82%);padding:10px 13px;border:1px solid var(--border-default);border-radius:14px 14px 4px 14px;background:var(--surface-2)}.xy-message--assistant .xy-message__body{padding:3px 0 0}.xy-message__text{white-space:pre-wrap;overflow-wrap:anywhere;color:var(--text-primary);font-size:13px;line-height:1.72}.xy-message--assistant .xy-message__text{color:var(--text-secondary)}.xy-agent-activity{display:grid;grid-template-columns:auto auto minmax(0,1fr);align-items:center;gap:7px;margin-top:10px;padding:8px 10px;border:1px solid var(--border-subtle);border-radius:10px;background:color-mix(in srgb,var(--surface-2) 65%,transparent);font-size:10px;color:var(--text-muted)}.xy-agent-activity__dot{width:6px;height:6px;border-radius:50%;background:#7e98b6;box-shadow:0 0 9px color-mix(in srgb,#7e98b6 45%,transparent)}.xy-agent-activity strong{color:var(--text-secondary);font-weight:650}.xy-agent-activity.phase-executing .xy-agent-activity__dot{background:#d9bd7a}.xy-agent-activity.phase-verifying .xy-agent-activity__dot{background:#65c77e}.xy-agent-activity.phase-recovering{border-color:color-mix(in srgb,#d99085 32%,var(--border-subtle));background:color-mix(in srgb,#d99085 6%,var(--surface-1))}.xy-agent-activity.phase-recovering .xy-agent-activity__dot{background:#d99085}.xy-run-meta{display:flex;flex-wrap:wrap;gap:7px 10px;margin-top:9px;color:var(--text-faint);font-size:9px}.xy-run-meta>span:first-child{padding:2px 6px;border:1px solid var(--border-subtle);border-radius:999px}.xy-run-meta .run-completed{color:#9bcaa7}.xy-run-meta .run-waiting_approval,.xy-run-meta .run-waiting_user{color:#d9bd7a}.xy-run-meta .run-failed{color:#d99085}.xy-run-details{margin-top:9px;color:var(--text-muted);font-size:10px}.xy-run-details summary{width:max-content;cursor:pointer;color:var(--text-faint)}.xy-run-observation{display:grid;grid-template-columns:auto minmax(0,1fr);gap:8px;margin-top:7px;padding-left:2px}.xy-run-observation code{color:var(--text-faint)}.xy-run-actions{margin-top:10px}.run-control-actions{display:flex;gap:7px;flex-wrap:wrap}.run-control-actions button{border:1px solid var(--border-subtle);background:var(--surface-1);color:var(--text-secondary);border-radius:8px;padding:6px 9px;font-size:10px}.run-control-actions .run-resume{color:#9bcaa7}.run-control-actions .run-cancel{color:#d8aaa2}.xy-thinking{height:30px;display:flex;align-items:center;gap:5px;color:var(--text-muted);font-size:10px}.xy-thinking i{width:5px;height:5px;border-radius:50%;background:var(--text-faint);animation:xyThink 1.1s infinite ease-in-out}.xy-thinking i:nth-child(2){animation-delay:.12s}.xy-thinking i:nth-child(3){animation-delay:.24s}.xy-thinking span{margin-left:5px}@keyframes xyThink{0%,70%,100%{opacity:.3;transform:translateY(0)}35%{opacity:1;transform:translateY(-2px)}}
.xy-composer-dock{width:min(900px,100%);flex:0 0 auto;margin:0 auto}.xy-inline-approval{display:flex;align-items:center;justify-content:space-between;gap:14px;margin:0 0 9px;padding:11px 12px;border:1px solid color-mix(in srgb,#d9bd7a 44%,var(--border-default));border-radius:12px;background:color-mix(in srgb,#d9bd7a 8%,var(--surface-1));box-shadow:0 10px 28px rgba(0,0,0,.12)}.xy-inline-approval__main{min-width:0;display:grid;gap:4px}.xy-inline-approval__eyebrow{display:flex;align-items:center;gap:8px;font-size:9px;color:var(--text-muted)}.xy-inline-approval__eyebrow b{padding:2px 6px;border-radius:999px;background:color-mix(in srgb,#d9bd7a 12%,transparent);color:#d9bd7a}.xy-inline-approval strong{font-size:12px;color:var(--text-primary)}.xy-inline-approval p{margin:0;color:var(--text-muted);font-size:10px;line-height:1.45}.xy-inline-approval__actions{flex:0 0 auto;display:flex;gap:7px}.xy-inline-approval__actions button{border:1px solid var(--border-default);border-radius:8px;background:var(--surface-2);color:var(--text-secondary);padding:7px 10px;font-size:10px}.xy-inline-approval__actions button.approve{border-color:color-mix(in srgb,#65c77e 42%,var(--border-default));color:#9bcaa7}.xy-brain-callout{display:flex;align-items:center;justify-content:space-between;gap:14px;margin:0 0 9px;padding:9px 11px;border:1px solid color-mix(in srgb,#d9bd7a 28%,var(--border-subtle));border-radius:10px;background:color-mix(in srgb,#d9bd7a 5%,var(--surface-1))}.xy-brain-callout>div{min-width:0;display:flex;align-items:baseline;gap:8px}.xy-brain-callout strong{font-size:11px;white-space:nowrap}.xy-brain-callout span{overflow:hidden;text-overflow:ellipsis;white-space:nowrap;color:var(--text-muted);font-size:10px}.xy-brain-callout button{flex:0 0 auto;border:1px solid var(--border-default);border-radius:7px;padding:6px 9px;background:var(--surface-2);color:var(--text-primary);font-size:10px}.xy-chat-notice{margin-bottom:8px}.xy-chat-composer{min-height:0;border-radius:15px;box-shadow:0 14px 42px rgba(0,0,0,.16)}.xy-chat-composer.disabled{opacity:.72}.xy-image-input{display:none}.xy-image-strip{display:flex;gap:8px;overflow-x:auto;padding:10px 12px 0}.xy-image-chip{display:grid;grid-template-columns:42px minmax(0,110px) 20px;align-items:center;gap:7px;padding:5px 6px;border:1px solid var(--border-subtle);border-radius:10px;background:var(--surface-2)}.xy-image-chip img{width:42px;height:42px;border-radius:7px;object-fit:cover}.xy-image-chip span{overflow:hidden;text-overflow:ellipsis;white-space:nowrap;color:var(--text-muted);font-size:9px}.xy-image-chip button{border:0;background:transparent;color:var(--text-muted);font-size:16px}.xy-chat-composer textarea{display:block;width:100%;height:54px;min-height:54px;max-height:160px;resize:none;overflow-y:auto;padding:15px 16px 6px;font-size:13px;line-height:1.55}.xy-composer-footer{min-height:44px;padding:5px 8px 7px}.xy-composer-footer .ai-composer__right{min-width:0}.xy-key-hint{color:var(--text-faint)!important;font-size:9px!important;white-space:nowrap}.xy-provider{max-width:210px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.xy-send{width:32px;height:32px;display:grid;place-items:center;padding:0!important;border-radius:9px!important;font-size:17px!important;font-weight:700}.xy-send:disabled{opacity:.35}.xy-manual-row{justify-content:flex-end;margin-top:5px;font-size:9px}.xy-manual-row span{color:var(--text-faint)}
@media(max-width:900px){.xy-inline-approval{align-items:stretch;flex-direction:column}.xy-inline-approval__actions{justify-content:flex-end}.xy-chat-workspace{padding:0 16px 12px}.xy-chat-header{height:50px;flex-basis:50px}.xy-key-hint,.xy-provider{display:none}.xy-message--user .xy-message__body{max-width:90%}.xy-chat-welcome h1{font-size:25px}.xy-brain-callout span{display:none}}
</style>
