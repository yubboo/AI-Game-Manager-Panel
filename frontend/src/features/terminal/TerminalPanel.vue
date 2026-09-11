<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { backend } from '../../shared/api/backend'
import type { XiaoYuApprovalState, XiaoYuCommandResult, XiaoYuRuntimeStatus } from '../../shared/types/backend'

const status = ref<XiaoYuRuntimeStatus | null>(null)
const approval = ref<XiaoYuApprovalState | null>(null)
const command = ref('')
const cwd = ref('.')
const result = ref<XiaoYuCommandResult | null>(null)
const busy = ref(false)

async function refresh() {
  try {
    const [runtime, state] = await Promise.all([backend.xiaoyuRuntimeStatus(), backend.xiaoyuApprovalState()])
    status.value = runtime
    approval.value = state
  } catch {
    status.value = null
  }
}

onMounted(refresh)

async function run(approvalId = '') {
  if (!command.value.trim() || busy.value || !status.value?.ready) return
  busy.value = true
  result.value = null
  try {
    result.value = await backend.runXiaoYuCommand({
      command: command.value.trim(),
      workingDirectory: cwd.value.trim() || '.',
      ...(approvalId ? { approvalId } : {}),
    })
  } catch (error) {
    result.value = {
      command: command.value,
      cwd: cwd.value,
      exitCode: -1,
      stdout: '',
      stderr: error instanceof Error ? error.message : String(error),
      decision: 'deny',
    }
  } finally {
    busy.value = false
  }
}

async function approve() {
  const id = result.value?.approvalId
  if (!id || busy.value) return
  busy.value = true
  try {
    await backend.resolveXiaoYuApproval(id, 'approve')
  } catch (error) {
    result.value = { ...result.value!, pending: false, decision: 'deny', stderr: error instanceof Error ? error.message : String(error) }
    busy.value = false
    return
  }
  busy.value = false
  await run(id)
}

async function reject() {
  const id = result.value?.approvalId
  if (!id || busy.value) return
  busy.value = true
  try {
    await backend.resolveXiaoYuApproval(id, 'reject')
    result.value = { ...result.value!, pending: false, decision: 'deny', stderr: '用户已取消该命令。' }
  } catch (error) {
    result.value = { ...result.value!, pending: false, decision: 'deny', stderr: error instanceof Error ? error.message : String(error) }
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="terminal-panel">
    <div class="terminal-panel__toolbar">
      <label>工作目录 <input v-model="cwd" spellcheck="false" /></label>
      <div class="terminal-panel__state">
        <span :class="['dot', { ready: status?.ready }]" />
        <span>{{ status?.ready ? `小鱼在线 · ${status.version}` : '小鱼 Runtime 离线' }}</span>
        <span>·</span>
        <span>{{ approval?.modeLabel ?? '请求批准' }}</span>
        <button type="button" :disabled="busy" @click="refresh">刷新</button>
      </div>
    </div>

    <div class="terminal-panel__screen">
      <pre v-if="result">$ {{ result.command }}
{{ result.stdout }}{{ result.stderr }}
[exit={{ result.exitCode }} · decision={{ result.decision }}]</pre>
      <div v-else class="terminal-panel__empty">
        <strong>XiaoYu Control Terminal</strong>
        <span>人工命令与小鱼共用 AGMP 的身份、审批和审计边界；已有系统功能仍优先调用所属 Tool/Service，不用裸 Shell 绕过模块。</span>
      </div>
    </div>

    <div class="terminal-panel__command">
      <span>❯</span>
      <input
        v-model="command"
        :disabled="!status?.ready || busy"
        :placeholder="status?.ready ? '输入受控命令，例如：go version' : '小鱼 Runtime 就绪后可执行命令'"
        spellcheck="false"
        @keydown.enter.prevent="run()"
      />
      <button type="button" :disabled="busy || !status?.ready || !command.trim()" @click="run()">{{ busy ? '处理中' : '执行' }}</button>
      <button v-if="result?.pending" class="approve" type="button" :disabled="busy" @click="approve">批准并继续</button>
      <button v-if="result?.pending" type="button" :disabled="busy" @click="reject">取消</button>
    </div>
  </div>
</template>

<style scoped>
.terminal-panel{height:100%;display:flex;flex-direction:column;background:#0b0e0c;min-height:0}.terminal-panel__toolbar{display:flex;align-items:center;justify-content:space-between;gap:16px;padding:8px 12px;border-bottom:1px solid rgba(255,255,255,.06);background:#111512}.terminal-panel__toolbar label{font-size:11px;color:#7e8981;display:flex;align-items:center;gap:8px}.terminal-panel__toolbar input{width:min(360px,40vw);background:#090b09;border:1px solid rgba(255,255,255,.08);border-radius:6px;padding:5px 8px;color:#cbd5ce}.terminal-panel__state{display:flex;align-items:center;gap:6px;color:#758078;font-size:11px}.terminal-panel__state .dot{width:7px;height:7px;border-radius:50%;background:#9a6262}.terminal-panel__state .dot.ready{background:#68bd7d}.terminal-panel__state button,.terminal-panel__command button{border:1px solid rgba(255,255,255,.1);background:#171c18;color:#dce4de;border-radius:6px;padding:5px 9px}.terminal-panel__screen{flex:1;min-height:90px;overflow:auto;padding:12px 14px}.terminal-panel__screen pre{white-space:pre-wrap;word-break:break-word;color:#c7d2ca;font:12px/1.55 Consolas,'Cascadia Code',monospace;margin:0}.terminal-panel__empty{height:100%;display:flex;flex-direction:column;justify-content:center;align-items:center;text-align:center;gap:5px;color:#59645c}.terminal-panel__empty strong{color:#829188;font-size:12px}.terminal-panel__empty span{font-size:11px;max-width:720px;line-height:1.5}.terminal-panel__command{display:flex;align-items:center;gap:8px;border-top:1px solid rgba(255,255,255,.06);padding:8px 12px;background:#0e120f}.terminal-panel__command>span{color:#6eb284}.terminal-panel__command input{flex:1;min-width:80px;background:transparent;border:0;outline:0;color:#e1e8e3;font:12px Consolas,'Cascadia Code',monospace}.terminal-panel__command input:disabled{opacity:.55}.terminal-panel__command .approve{border-color:rgba(91,190,118,.35);color:#9bd2aa}@media(max-width:800px){.terminal-panel__toolbar{align-items:flex-start;flex-direction:column}.terminal-panel__toolbar input{width:60vw}.terminal-panel__state{flex-wrap:wrap}}
</style>
