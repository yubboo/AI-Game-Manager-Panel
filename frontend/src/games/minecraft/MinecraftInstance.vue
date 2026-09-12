<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue'
import { backend } from '../../shared/api/backend'
import type { GameInstance, MinecraftLogLine, MinecraftProbeResult, MinecraftRuntimeSnapshot } from '../../shared/types/backend'

const props = defineProps<{ instance: GameInstance }>()
const emit = defineEmits<{ changed: [] }>()
const busy = ref(false), error = ref(''), runtime = ref<MinecraftRuntimeSnapshot | null>(null), probe = ref<MinecraftProbeResult | null>(null), logs = ref<MinecraftLogLine[]>([])
let timer: number | undefined
async function refresh() { try { const snapshot = await backend.minecraftStatus(props.instance.id); runtime.value = snapshot; const after = snapshot.logCursor > 120 ? snapshot.logCursor - 120 : 0; const b = await backend.minecraftLogs(props.instance.id, after, 120); logs.value = b.lines } catch {} }
async function action(kind:'start'|'stop'|'probe') { busy.value=true; error.value=''; try { if(kind==='start') runtime.value=await backend.startMinecraft(props.instance.id); if(kind==='stop') runtime.value=await backend.stopMinecraft(props.instance.id); if(kind==='probe') probe.value=await backend.probeMinecraft(props.instance.id); emit('changed'); await refresh() } catch(e){error.value=e instanceof Error?e.message:String(e)} finally{busy.value=false} }
void refresh(); timer=window.setInterval(refresh,3000); onBeforeUnmount(()=>{if(timer)window.clearInterval(timer)})
</script>
<template>
  <div class="mc-instance">
    <div class="mc-actions"><button class="btn" :disabled="busy" @click="action('start')">启动</button><button class="btn" :disabled="busy" @click="action('stop')">停止</button><button class="btn" :disabled="busy" @click="action('probe')">MC Ping</button></div>
    <p v-if="runtime"><small>Runtime：{{ runtime.state }}<span v-if="runtime.ready"> · Ready</span><span v-if="runtime.pid"> · PID {{ runtime.pid }}</span></small></p>
    <p v-if="probe"><small>Ping：{{ probe.version }} · {{ probe.playersOnline }}/{{ probe.playersMax }} · {{ probe.latencyMs }}ms</small></p>
    <div v-if="error" class="inline-alert inline-alert--danger">{{ error }}</div>
    <details v-if="logs.length"><summary>最近控制台（{{ logs.length }}）</summary><pre class="mc-console">{{ logs.map(line=>line.text).join('\n') }}</pre></details>
  </div>
</template>
<style scoped>.mc-instance{display:grid;gap:8px;margin-top:12px}.mc-actions{display:flex;gap:8px;flex-wrap:wrap}.mc-console{max-height:220px;overflow:auto;padding:10px;border:1px solid var(--border);border-radius:8px;font-size:12px;white-space:pre-wrap}</style>
