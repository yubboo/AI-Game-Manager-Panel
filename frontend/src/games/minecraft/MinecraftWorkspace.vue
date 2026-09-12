<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { backend } from '../../shared/api/backend'
import type { MinecraftDeploymentResult, MinecraftPlan, MinecraftPlanRequest, MinecraftSoftware } from '../../shared/types/backend'

const emit = defineEmits<{ deployed: [] }>()
const request = reactive<MinecraftPlanRequest>({
  name: '我的 Minecraft 服务器', version: '', software: 'paper', memoryMb: 4096, port: 25565,
  onlineMode: true, whitelist: false, eulaAccepted: false, autoInstallJava: true, startAfterDeploy: true,
})
const planning = ref(false)
const deploying = ref(false)
const error = ref('')
const plan = ref<MinecraftPlan | null>(null)
const result = ref<MinecraftDeploymentResult | null>(null)
const canDeploy = computed(() => !!plan.value && request.eulaAccepted && !deploying.value)

function normalize() {
  request.name = request.name.trim() || 'Minecraft Server'
  request.version = request.version.trim()
  request.memoryMb = Math.min(131072, Math.max(512, Number(request.memoryMb) || 4096))
  request.port = Math.min(65535, Math.max(1, Number(request.port) || 25565))
}
async function buildPlan() {
  normalize(); planning.value = true; error.value = ''; result.value = null
  try { plan.value = await backend.minecraftPlan({ ...request }) }
  catch (e) { plan.value = null; error.value = e instanceof Error ? e.message : String(e) }
  finally { planning.value = false }
}
async function deploy() {
  normalize(); if (!request.eulaAccepted) { error.value = '请先确认接受 Minecraft EULA。'; return }
  deploying.value = true; error.value = ''; result.value = null
  try {
    const deployed = await backend.deployMinecraft({ ...request })
    result.value = deployed
    plan.value = deployed.plan
    emit('deployed')
  }
  catch (e) { error.value = e instanceof Error ? e.message : String(e) }
  finally { deploying.value = false }
}
function chooseSoftware(value: MinecraftSoftware) { request.software = value; plan.value = null; result.value = null }
</script>

<template>
  <div class="mc-grid">
    <article class="panel mc-card">
      <div class="mc-heading">
        <div><span class="eyebrow">MINECRAFT GAME PACK</span><h2>一句话与可视化共用部署内核</h2></div>
        <span class="chip">Vanilla / Paper / Fabric</span>
      </div>
      <p class="muted">版本与 Java 要求实时读取 Mojang；Paper/Fabric 再查各自官方上游。这里点击部署与 XiaoYu 的 <code>game.deploy</code> 最终创建同一种 GameInstance。</p>
      <div class="mc-form">
        <label>实例名称<input v-model="request.name" class="input" /></label>
        <label>Minecraft 版本<input v-model="request.version" class="input" placeholder="留空 = Mojang 最新稳定版" /></label>
        <label>服务端<select v-model="request.software" class="input" @change="chooseSoftware(request.software)"><option value="paper">Paper（推荐）</option><option value="vanilla">Vanilla</option><option value="fabric">Fabric</option></select></label>
        <label>内存 MB<input v-model.number="request.memoryMb" class="input" type="number" min="512" step="512" /></label>
        <label>端口<input v-model.number="request.port" class="input" type="number" min="1" max="65535" /></label>
        <label class="mc-check"><input v-model="request.onlineMode" type="checkbox" /> 正版验证 online-mode</label>
        <label class="mc-check"><input v-model="request.whitelist" type="checkbox" /> 启用白名单</label>
        <label class="mc-check"><input v-model="request.autoInstallJava" type="checkbox" /> 缺少 Java 时由 AGMP 安装受管 JRE</label>
        <label class="mc-check"><input v-model="request.startAfterDeploy" type="checkbox" /> 部署后启动并等待 Done + MC Ping 验证</label>
      </div>
      <label class="mc-eula"><input v-model="request.eulaAccepted" type="checkbox" /> 我确认接受 Minecraft EULA，允许 AGMP 写入 <code>eula=true</code></label>
      <div v-if="error" class="inline-alert inline-alert--danger">{{ error }}</div>
      <div class="mc-actions"><button class="btn" :disabled="planning || deploying" @click="buildPlan">{{ planning ? '实时查证中…' : '先生成部署计划' }}</button><button class="btn btn--primary" :disabled="!canDeploy" @click="deploy">{{ deploying ? '部署/验证中…' : '一键部署并验证' }}</button></div>
    </article>

    <article v-if="plan" class="panel mc-card">
      <span class="eyebrow">LIVE DEPLOYMENT PLAN</span><h2>{{ plan.versionFacts.version }} · {{ plan.versionFacts.artifact.software }}</h2>
      <div class="mc-facts"><div><small>Java</small><strong>{{ plan.versionFacts.javaMajor }}</strong></div><div><small>端口</small><strong>{{ plan.port }}</strong></div><div><small>内存</small><strong>{{ plan.memoryMb }} MB</strong></div><div><small>最新稳定版</small><strong>{{ plan.versionFacts.latestRelease }}</strong></div></div>
      <p><strong>事实源：</strong>{{ plan.versionFacts.sources.join(' · ') }}</p><p><strong>下载信任：</strong>{{ plan.versionFacts.artifact.trust }}</p><p><strong>实例目录：</strong><code>{{ plan.installPath }}</code></p>
      <ol class="mc-steps"><li v-for="step in plan.steps" :key="step">{{ step }}</li></ol>
    </article>

    <article v-if="result" class="panel mc-card mc-success">
      <span class="eyebrow">GAMEINSTANCE CREATED</span><h2>{{ result.instance.name }}</h2>
      <p>状态：<strong>{{ result.runtime.state }}</strong><span v-if="result.runtime.ready"> · Ready</span></p>
      <p v-if="result.probe">协议验证：{{ result.probe.version }} · {{ result.probe.playersOnline }}/{{ result.probe.playersMax }} 玩家 · {{ result.probe.latencyMs }}ms</p>
      <p>地址：<code>{{ result.instance.address || `127.0.0.1:${result.instance.port}` }}</code></p>
    </article>
  </div>
</template>

<style scoped>
.mc-grid{display:grid;gap:16px}.mc-card{display:grid;gap:14px}.mc-heading{display:flex;justify-content:space-between;gap:16px;align-items:flex-start}.mc-form{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px}.mc-form label{display:grid;gap:6px;font-size:13px}.mc-check,.mc-eula{display:flex!important;align-items:center;gap:8px}.mc-eula{padding:12px;border:1px solid var(--border);border-radius:10px}.mc-actions{display:flex;gap:10px;flex-wrap:wrap}.mc-facts{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:10px}.mc-facts>div{display:grid;gap:4px;padding:10px;border:1px solid var(--border);border-radius:10px}.mc-steps{margin:0;padding-left:20px;display:grid;gap:6px}.muted{opacity:.75}.mc-success{border-color:var(--success)}@media(max-width:720px){.mc-form,.mc-facts{grid-template-columns:1fr}.mc-heading{display:grid}}
</style>
