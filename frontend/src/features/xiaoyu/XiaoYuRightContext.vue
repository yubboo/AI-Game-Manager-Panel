<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { backend } from '../../shared/api/backend'
import { useAppStore } from '../../shared/store/app'
import type { AuthUser, XiaoYuApprovalState, XiaoYuCapabilitySnapshot, XiaoYuDSHBundle, XiaoYuHarnessSnapshot, XiaoYuRuntimeStatus, XiaoYuToolSpec, XiaoYuTraceEvent } from '../../shared/types/backend'

const app = useAppStore()
const router = useRouter()
const runtime = ref<XiaoYuRuntimeStatus | null>(null)
const approvalState = ref<XiaoYuApprovalState | null>(null)
const harness = ref<XiaoYuHarnessSnapshot | null>(null)
const capabilities = ref<XiaoYuCapabilitySnapshot | null>(null)
const tools = ref<XiaoYuToolSpec[]>([])
const trace = ref<XiaoYuTraceEvent[]>([])
const dshBundles = ref<XiaoYuDSHBundle[]>([])
const currentUser = ref<AuthUser | null>(null)
const busy = ref(false)
const message = ref('')
let timer = 0

const isOwner = computed(() => currentUser.value?.role === 'owner')
const brainReady = computed(() => Boolean(harness.value?.brain?.ready))

function riskLabel(value: string) {
  return ({ read: '读取', operate: '操作', modify: '修改', destructive: '高风险', system: '系统' } as Record<string, string>)[value] ?? value
}
function activeDSH(bundle: XiaoYuDSHBundle) {
  return capabilities.value?.plugins?.find(item => item.manifest.source === 'deepseek-harness' && item.manifest.name === bundle.packageName && item.state === 'active')
}

async function refresh(showError = true) {
  if (busy.value) return
  busy.value = true
  try {
    const user = currentUser.value ?? await backend.currentUser()
    currentUser.value = user
    const [runtimeValue, approvalValue, harnessValue, traceValue] = await Promise.all([
      backend.xiaoyuRuntimeStatus(),
      backend.xiaoyuApprovalState(),
      backend.xiaoyuHarnessStatus(),
      backend.xiaoyuTrace(0, 20),
    ])
    runtime.value = runtimeValue
    approvalState.value = approvalValue
    harness.value = harnessValue
    capabilities.value = harnessValue.capabilities
    tools.value = runtimeValue.ready ? ((await backend.xiaoyuTools()) ?? []) : []
    trace.value = (traceValue ?? []).slice(-6).reverse()
    dshBundles.value = isOwner.value ? ((await backend.xiaoyuDSHPlugins()) ?? []) : []
    message.value = ''
  } catch (error) {
    if (showError) message.value = error instanceof Error ? error.message : String(error)
  } finally {
    busy.value = false
  }
}

async function mountDSH(bundle: XiaoYuDSHBundle, allowXiaoYu: boolean) {
  if (!isOwner.value || busy.value) return
  busy.value = true
  try {
    await backend.mountXiaoYuDSHPlugin({ directory: bundle.directory, trusted: true, xiaoyu: allowXiaoYu })
    message.value = `${bundle.packageName} 已加载${allowXiaoYu ? '并允许小鱼调用' : ''}。`
  } catch (error) {
    message.value = error instanceof Error ? error.message : String(error)
  } finally {
    busy.value = false
    await refresh(false)
  }
}

async function unmountPlugin(id: string) {
  if (!isOwner.value || busy.value) return
  busy.value = true
  try { await backend.unmountXiaoYuPlugin(id); message.value = '插件已卸载。' }
  catch (error) { message.value = error instanceof Error ? error.message : String(error) }
  finally { busy.value = false; await refresh(false) }
}

function openSettings(section: string) { void router.push({ path: '/settings', query: { section } }) }

onMounted(() => {
  void refresh()
  timer = window.setInterval(() => { void refresh(false) }, 5000)
})
onBeforeUnmount(() => { if (timer) window.clearInterval(timer) })
</script>

<template>
  <div class="xy-context">
    <section class="right-sidebar__section xy-context__head">
      <div class="xy-context__title"><div><span class="right-sidebar__label">小鱼上下文</span><strong>运行状态</strong></div><button type="button" :disabled="busy" @click="refresh()">{{ busy ? '刷新中' : '刷新' }}</button></div>
      <p v-if="message" class="xy-context__message">{{ message }}</p>
      <div class="xy-status-grid">
        <div><span>核心</span><strong :class="runtime?.ready ? 'status-success' : 'status-warning'">{{ runtime?.ready ? 'READY' : 'OFFLINE' }}</strong></div>
        <div><span>大脑</span><strong :class="brainReady ? 'status-success' : 'status-warning'">{{ brainReady ? (harness?.brain?.name || 'READY') : '未就绪' }}</strong></div>
        <div><span>Tool</span><strong>{{ tools.length }}</strong></div>
        <div><span>任务</span><strong>{{ harness?.runs?.filter(item => !['completed', 'failed', 'cancelled'].includes(item.status)).length ?? 0 }}</strong></div>
      </div>
      <small>{{ harness?.brain?.model || runtime?.message || '正在读取 XiaoYu Runtime…' }}</small>
    </section>


    <section class="right-sidebar__section">
      <span class="right-sidebar__label">小鱼设置</span>
      <button class="right-sidebar__action" type="button" @click="openSettings('models')"><span class="xy-mini-icon">AI</span><span><strong>模型管理</strong><small>Provider、默认大脑与连接测试</small></span></button>
      <button class="right-sidebar__action" type="button" @click="openSettings('intelligence')"><span class="xy-mini-icon">M</span><span><strong>记忆、技能与专家</strong><small>Memory / Skills / Experts</small></span></button>
      <button class="right-sidebar__action" type="button" @click="app.setBottomPanelOpen(true)"><span class="xy-mini-icon">›_</span><span><strong>底部终端</strong><small>终端 / 日志 / 任务输出</small></span></button>
    </section>

    <details class="xy-details">
      <summary>能力与 Tool <span>{{ capabilities?.tools?.filter(item => item.xiaoyu).length ?? tools.length }}</span></summary>
      <div class="xy-tool-list"><div v-for="tool in tools" :key="tool.name"><code>{{ tool.name }}</code><span>{{ riskLabel(tool.risk) }}</span></div><div v-if="!tools.length" class="xy-empty">暂无可用 Tool。</div></div>
    </details>

    <details v-if="isOwner && dshBundles.length" class="xy-details">
      <summary>Harness 插件 <span>{{ capabilities?.plugins?.length ?? 0 }}</span></summary>
      <article v-for="bundle in dshBundles" :key="bundle.directory" class="xy-plugin">
        <div><strong>{{ bundle.packageName }}</strong><small>{{ bundle.version || '未标版本' }}</small></div>
        <div v-if="!activeDSH(bundle)" class="xy-actions"><button type="button" :disabled="busy" @click="mountDSH(bundle, false)">加载</button><button class="approve" type="button" :disabled="busy" @click="mountDSH(bundle, true)">允许小鱼</button></div>
        <div v-else class="xy-actions"><span class="status-success">已挂载</span><button type="button" :disabled="busy" @click="unmountPlugin(activeDSH(bundle)!.manifest.id)">卸载</button></div>
      </article>
    </details>

    <details class="xy-details">
      <summary>最近 Trace <span>{{ trace.length }}</span></summary>
      <div class="xy-trace"><div v-for="item in trace" :key="item.sequence"><code>#{{ item.sequence }} {{ item.type }}</code><span>{{ item.summary }}</span></div><div v-if="!trace.length" class="xy-empty">暂无 Trace。</div></div>
    </details>
  </div>
</template>

<style scoped>
.xy-context{display:grid}.xy-context__head{padding-top:4px}.xy-context__title{display:flex;align-items:flex-start;justify-content:space-between;gap:10px}.xy-context__title>div{display:grid;gap:5px}.xy-context__title button,.xy-actions button{border:1px solid var(--border-default);border-radius:7px;background:var(--surface-2);color:var(--text-secondary);padding:6px 8px;font-size:10px}.xy-context__message{padding:8px;border-radius:8px;background:var(--surface-2);color:var(--text-secondary)!important}.xy-status-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:7px}.xy-status-grid>div{display:grid;gap:3px;padding:8px;border:1px solid var(--border-subtle);border-radius:8px;background:var(--surface-1)}.xy-status-grid span{font-size:9px;color:var(--text-muted)}.xy-status-grid strong{font-size:11px;overflow-wrap:anywhere}.xy-empty{padding:8px 2px;color:var(--text-muted);font-size:11px}.xy-actions{display:flex;align-items:center;gap:6px;flex-wrap:wrap}.xy-actions button.approve{color:var(--success,#62b87a)}.xy-mini-icon{width:28px;height:28px;display:grid!important;place-items:center;border:1px solid var(--border-subtle);border-radius:7px;color:var(--text-muted);font:700 9px/1 monospace}.xy-details{border-bottom:1px solid var(--border-subtle);padding:12px 2px}.xy-details summary{display:flex;justify-content:space-between;gap:12px;cursor:pointer;color:var(--text-secondary);font-size:11px;font-weight:700;list-style:none}.xy-details summary::-webkit-details-marker{display:none}.xy-details summary span{color:var(--text-muted);font-weight:500}.xy-tool-list,.xy-trace{display:grid;gap:5px;margin-top:9px}.xy-tool-list>div,.xy-trace>div{display:flex;justify-content:space-between;gap:8px;padding:6px 7px;border-radius:7px;background:var(--surface-1);font-size:9px}.xy-tool-list code{min-width:0;overflow-wrap:anywhere}.xy-tool-list span{color:var(--text-muted)}.xy-trace>div{display:grid}.xy-trace span{color:var(--text-muted);line-height:1.4}.xy-plugin{display:grid;gap:7px;margin-top:8px;padding:8px;border-radius:8px;background:var(--surface-1)}.xy-plugin>div:first-child{display:flex;justify-content:space-between;gap:8px}.xy-plugin strong{font-size:10px}.xy-plugin small{font-size:9px;color:var(--text-muted)}
</style>
