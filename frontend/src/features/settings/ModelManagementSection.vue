<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import AppIcon from '../../shared/components/AppIcon.vue'
import { backend } from '../../shared/api/backend'
import type {
  AuthUser,
  XiaoYuModelCatalog,
  XiaoYuModelConnectionRequest,
  XiaoYuModelConnectionResult,
  XiaoYuModelPreset,
  XiaoYuModelProfile,
  XiaoYuSaveModelRequest,
} from '../../shared/types/backend'

const router = useRouter()
const catalog = ref<XiaoYuModelCatalog | null>(null)
const currentUser = ref<AuthUser | null>(null)
const loading = ref(false)
const saving = ref(false)
const testing = ref(false)
const discovering = ref(false)
const message = ref('')
const testResult = ref<XiaoYuModelConnectionResult | null>(null)
const discoveredModels = ref<string[]>([])
const extraText = ref('{}')

const form = reactive<XiaoYuSaveModelRequest>({
  name: '我的模型', provider: 'openai', protocol: 'openai-responses', authMode: 'api-key', baseUrl: 'https://api.openai.com/v1', model: '', apiKey: '', enabled: true,
  contextWindow: 128000, maxOutputTokens: 8192, thinkingMode: 'default', reasoningEffort: 'default', extra: {},
})

const canManage = computed(() => currentUser.value?.role === 'owner' || currentUser.value?.role === 'administrator')
const selectedPreset = computed(() => catalog.value?.presets.find(item => item.id === form.provider))
const editing = computed(() => Boolean(form.id))
const isSubscription = computed(() => form.authMode === 'subscription')
const isAgentProvider = computed(() => selectedPreset.value?.kind === 'agent-provider')
const needsBaseUrl = computed(() => !isAgentProvider.value)
const canDiscoverModels = computed(() => !isAgentProvider.value)
const apiKeyHint = computed(() => {
  const profile = form.id ? catalog.value?.profiles.find(item => item.id === form.id) : undefined
  if (isSubscription.value) return '使用官方订阅/登录状态，不在 AGMP 中填写 Token'
  if (form.authMode === 'local') return '本地 Runtime 通常无需 API Key'
  if (profile?.hasApiKey) return '已安全保存；留空则保持原密钥'
  if (selectedPreset.value?.apiKeyOptional) return '该 Provider 可留空'
  return '请输入 API Key'
})

function presetBadge(preset: XiaoYuModelPreset) {
  if (preset.kind === 'agent-provider') return '订阅'
  if (preset.local) return '本地'
  if (preset.id === 'custom') return '自定义'
  return preset.name.slice(0, 2)
}

function capabilityLabels(capabilities?: XiaoYuModelProfile['capabilities']) {
  if (!capabilities) return []
  const labels: string[] = []
  labels.push(capabilities.native ? '原生 Adapter' : '兼容 Adapter')
  if (capabilities.reasoning) labels.push('Reasoning')
  if (capabilities.toolCalling) labels.push('Tool')
  if (capabilities.replay) labels.push('Replay')
  if (capabilities.vision) labels.push('Vision')
  if (capabilities.streaming) labels.push('Streaming')
  return labels
}

function choosePreset(preset: XiaoYuModelPreset) {
  form.provider = preset.id
  form.protocol = preset.protocol
  form.authMode = preset.defaultAuthMode
  form.baseUrl = preset.defaultBaseUrl
  if (!editing.value) form.name = preset.name
  if (preset.kind === 'agent-provider') form.model = ''
  form.apiKey = ''
  testResult.value = null
  discoveredModels.value = []
}

function applyProviderSelection() {
  const preset = catalog.value?.presets.find(item => item.id === form.provider)
  if (!preset) return
  form.protocol = preset.protocol
  form.authMode = preset.defaultAuthMode
  form.baseUrl = preset.defaultBaseUrl
  if (preset.kind === 'agent-provider') form.model = ''
  testResult.value = null
  discoveredModels.value = []
}

function resetForm() {
  Object.assign(form, {
    id: undefined, name: '我的模型', provider: 'openai', protocol: 'openai-responses', authMode: 'api-key', baseUrl: 'https://api.openai.com/v1', model: '', apiKey: '', enabled: true,
    contextWindow: 128000, maxOutputTokens: 8192, thinkingMode: 'default', reasoningEffort: 'default', extra: {},
  })
  extraText.value = '{}'
  testResult.value = null
  discoveredModels.value = []
}

function edit(profile: XiaoYuModelProfile) {
  Object.assign(form, {
    id: profile.id, name: profile.name, provider: profile.provider, protocol: profile.protocol, authMode: profile.authMode, baseUrl: profile.baseUrl, model: profile.model, apiKey: '', enabled: profile.enabled,
    contextWindow: profile.contextWindow, maxOutputTokens: profile.maxOutputTokens, thinkingMode: profile.thinkingMode, reasoningEffort: profile.reasoningEffort || 'default', extra: profile.extra ?? {},
  })
  extraText.value = JSON.stringify(profile.extra ?? {}, null, 2)
  testResult.value = null
  discoveredModels.value = []
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

function parseExtra(): Record<string, unknown> {
  const text = extraText.value.trim()
  if (!text) return {}
  const value = JSON.parse(text) as unknown
  if (!value || Array.isArray(value) || typeof value !== 'object') throw new Error('额外请求参数必须是 JSON 对象。')
  return value as Record<string, unknown>
}

function connectionRequest(): XiaoYuModelConnectionRequest {
  return { id: form.id, provider: form.provider, protocol: form.protocol, authMode: form.authMode, baseUrl: form.baseUrl, model: form.model, apiKey: form.apiKey || undefined, thinkingMode: form.thinkingMode, reasoningEffort: form.reasoningEffort, extra: parseExtra() }
}

async function refresh() {
  loading.value = true
  message.value = ''
  try {
    const [value, user] = await Promise.all([backend.xiaoyuModelCatalog(), backend.currentUser()])
    catalog.value = value
    currentUser.value = user
  } catch (error) {
    message.value = error instanceof Error ? error.message : String(error)
  } finally { loading.value = false }
}

async function save() {
  if (!canManage.value || saving.value) return
  saving.value = true
  message.value = ''
  try {
    const extra = parseExtra()
    const value = await backend.saveXiaoYuModel({ ...form, apiKey: form.apiKey?.trim() || undefined, extra })
    await refresh()
    edit(value)
    form.apiKey = ''
    message.value = value.isDefault ? '模型已保存，小鱼正在使用这个大脑。' : (value.brainEligible ? '模型已保存。你可以把它设为“小鱼默认大脑”。' : 'Provider 已保存并可测试授权状态；XiaoYu Brain Adapter 尚未启用。')
  } catch (error) { message.value = error instanceof Error ? error.message : String(error) }
  finally { saving.value = false }
}

async function testConnection() {
  if (!canManage.value || testing.value) return
  testing.value = true
  testResult.value = null
  message.value = ''
  try { testResult.value = await backend.testXiaoYuModel(connectionRequest()) }
  catch (error) {
    testResult.value = { ok: false, message: error instanceof Error ? error.message : String(error), endpoint: form.baseUrl, latencyMs: 0 }
  } finally { testing.value = false }
}

async function discover() {
  if (!canManage.value || discovering.value) return
  discovering.value = true
  message.value = ''
  try {
    const result = await backend.discoverXiaoYuModels(connectionRequest())
    discoveredModels.value = result.models ?? []
    testResult.value = result
    if (!form.model && discoveredModels.value.length) form.model = discoveredModels.value[0]
  } catch (error) {
    testResult.value = { ok: false, message: error instanceof Error ? error.message : String(error), endpoint: form.baseUrl, latencyMs: 0 }
  } finally { discovering.value = false }
}

async function setDefault(profile: XiaoYuModelProfile) {
  if (!canManage.value) return
  message.value = ''
  try { await backend.setXiaoYuDefaultModel(profile.id); await refresh(); message.value = `“${profile.name}”已成为小鱼默认大脑。` }
  catch (error) { message.value = error instanceof Error ? error.message : String(error) }
}

async function remove(profile: XiaoYuModelProfile) {
  if (!canManage.value || !window.confirm(`确定删除模型来源“${profile.name}”吗？若使用 API Key，对应 Secret 也会从本机 Vault 删除。`)) return
  try { await backend.deleteXiaoYuModel(profile.id); if (form.id === profile.id) resetForm(); await refresh(); message.value = '模型已删除。' }
  catch (error) { message.value = error instanceof Error ? error.message : String(error) }
}

onMounted(refresh)
</script>

<template>
  <section class="settings-section model-manager">
    <div class="settings-section__intro">
      <div class="settings-section__icon"><AppIcon name="ai" /></div>
      <div><strong>模型管理 · 小鱼大脑</strong><span>模型中心同时管理 API、套餐/订阅、官方 CLI Provider 与本地模型。无论模型来源是什么，真实操作都必须经过 AGMP Host、权限/审批和结构化 Tool。</span></div>
    </div>

    <div class="brain-banner" :class="{ ready: catalog?.brainReady }">
      <div><span class="brain-dot"></span><strong>{{ catalog?.brainReady ? '小鱼大脑已就绪' : '小鱼还没有可用的大脑' }}</strong><small>{{ catalog?.message ?? '正在读取模型配置……' }}</small></div>
      <button v-if="catalog?.brainReady" class="btn btn--secondary" type="button" @click="router.push('/')">去找小鱼</button>
    </div>

    <div class="settings-subsection">
      <div class="settings-subsection__heading"><strong>选择模型服务</strong><span>API Key、套餐/订阅、官方 CLI、本地 Runtime 与兼容接口统一由 Provider Contract 管理。</span></div>
      <div v-if="loading" class="model-empty">正在读取模型服务……</div>
      <div v-else class="provider-grid">
        <button v-for="preset in catalog?.presets ?? []" :key="preset.id" type="button" :class="['provider-card', { active: form.provider === preset.id }]" @click="choosePreset(preset)">
          <span class="provider-logo">{{ presetBadge(preset) }}</span>
          <span><strong>{{ preset.name }}</strong><small>{{ preset.description }}</small></span>
          <i v-if="form.provider === preset.id">✓</i>
        </button>
      </div>
    </div>

    <div class="model-editor-grid">
      <div class="model-editor-card">
        <div class="settings-subsection__heading"><strong>{{ editing ? '编辑模型' : '添加模型' }}</strong><span>API Key 存入 Secret Vault；订阅 Provider 只调用官方登录状态，不读取或复制厂商 Token 文件。</span></div>
        <div class="form-grid two">
          <label class="field-block"><span class="feature-title">模型名称</span><input v-model="form.name" :disabled="!canManage" placeholder="例如：小鱼主模型" /></label>
          <label class="field-block"><span class="feature-title">模型代码</span><div class="field-action"><input v-model="form.model" :disabled="!canManage || isAgentProvider" list="xiaoyu-model-list" :placeholder="isAgentProvider ? '由官方 Provider 管理' : '例如：gpt-5 / deepseek-chat'" /><button type="button" :disabled="!canManage || discovering || !canDiscoverModels" @click="discover">{{ discovering ? '拉取中' : '拉取' }}</button></div><datalist id="xiaoyu-model-list"><option v-for="item in discoveredModels" :key="item" :value="item" /></datalist></label>
          <label class="field-block"><span class="feature-title">接口协议</span><select v-model="form.protocol" :disabled="!canManage || isAgentProvider"><option value="openai-responses">OpenAI Responses（原生）</option><option value="deepseek">DeepSeek（原生）</option><option value="anthropic">Anthropic Messages（原生）</option><option value="gemini">Google Gemini（原生）</option><option value="openai-compatible">OpenAI Compatible</option><option value="codex-app-server">Codex App Server / CLI</option></select></label>
          <label class="field-block"><span class="feature-title">服务商</span><select v-model="form.provider" :disabled="!canManage" @change="applyProviderSelection"><option v-for="preset in catalog?.presets ?? []" :key="preset.id" :value="preset.id">{{ preset.name }}</option></select></label>
          <label class="field-block"><span class="feature-title">授权方式</span><select v-model="form.authMode" :disabled="!canManage"><option v-for="mode in selectedPreset?.authModes ?? ['api-key']" :key="mode" :value="mode">{{ mode === 'subscription' ? '套餐 / 订阅登录' : mode === 'local' ? '本地 Runtime' : 'API Key' }}</option></select></label>
          <label v-if="needsBaseUrl" class="field-block span-two"><span class="feature-title">Base URL</span><input v-model="form.baseUrl" :disabled="!canManage" placeholder="https://api.example.com/v1" /></label>
          <label v-if="form.authMode === 'api-key'" class="field-block"><span class="feature-title">API Key</span><input v-model="form.apiKey" :disabled="!canManage" type="password" autocomplete="new-password" :placeholder="apiKeyHint" /><small>{{ apiKeyHint }}</small></label>
          <div v-else-if="isSubscription" class="field-block subscription-note"><span class="feature-title">订阅账号</span><strong>使用官方 {{ selectedPreset?.name ?? 'CLI' }} 登录</strong><small>AGMP 只调用官方状态接口/CLI，不读取、不复制、不导出订阅 Token。当前 Codex Provider 可验证登录状态；安全 Brain Adapter 尚未启用。</small></div>
          <label class="field-block"><span class="feature-title">最大上下文</span><input v-model.number="form.contextWindow" :disabled="!canManage" type="number" min="1024" step="1024" /></label>
          <label class="field-block"><span class="feature-title">最大输出 Tokens</span><input v-model.number="form.maxOutputTokens" :disabled="!canManage" type="number" min="256" step="256" /></label>
          <label class="field-block"><span class="feature-title">思考模式</span><select v-model="form.thinkingMode" :disabled="!canManage"><option value="default">跟随模型默认</option><option value="on">开启</option><option value="off">关闭</option></select></label>
          <label class="field-block"><span class="feature-title">推理强度</span><select v-model="form.reasoningEffort" :disabled="!canManage"><option value="default">跟随模型默认</option><option value="minimal">Minimal</option><option value="low">Low</option><option value="medium">Medium</option><option value="high">High</option><option value="xhigh">Extra High</option><option value="max">Max</option></select></label>
          <label class="field-block span-two"><span class="feature-title">额外请求参数</span><textarea v-model="extraText" :disabled="!canManage" placeholder='例如：{"temperature":0.2,"reasoning_effort":"medium"}'></textarea><small>仅允许 JSON 对象；核心字段不能覆盖，Authorization、Token、Password、Secret 等敏感字段也会被后端拒绝。</small></label>
        </div>
        <div class="model-actions">
          <label class="checkbox-inline"><input v-model="form.enabled" :disabled="!canManage" type="checkbox" />启用这个模型</label>
          <div><button class="btn btn--secondary" type="button" @click="resetForm">新建</button><button class="btn btn--secondary" type="button" :disabled="!canManage || testing" @click="testConnection">{{ testing ? '测试中…' : '测试连接' }}</button><button class="btn btn--primary" type="button" :disabled="!canManage || saving" @click="save">{{ saving ? '保存中…' : '保存模型' }}</button></div>
        </div>
        <p v-if="!canManage" class="model-permission-note">当前账号只能查看模型配置；最高管理员或管理员才能修改大脑和密钥。</p>
      </div>

      <aside class="connection-card">
        <div class="settings-subsection__heading"><strong>连接测试</strong><span>API Provider 会执行真实 Tool Schema 探测；订阅 Provider 使用官方 CLI/登录接口验证账号状态。任何 API Key、OAuth Token 或 CLI 凭证都不会进入日志和 Trace。</span></div>
        <div class="connection-state" :class="{ ready: testResult?.ok, failed: testResult && !testResult.ok }"><span class="signal">◉</span><div><strong>{{ testResult ? (testResult.ok ? '连接正常' : '连接失败') : '等待测试' }}</strong><small>{{ testResult?.message ?? '保存前可以先测试当前配置' }}</small></div></div>
        <dl><div><dt>请求地址</dt><dd>{{ testResult?.endpoint || form.baseUrl || (isAgentProvider ? '官方 CLI' : '-') }}</dd></div><div><dt>耗时</dt><dd>{{ testResult?.latencyMs ? `${testResult.latencyMs} ms` : '-' }}</dd></div><div><dt>发现模型</dt><dd>{{ testResult?.models?.length ?? '-' }}</dd></div><div><dt>授权</dt><dd>{{ form.authMode === 'subscription' ? '套餐 / 订阅' : form.authMode === 'local' ? '本地 Runtime' : apiKeyHint }}</dd></div></dl>
      </aside>
    </div>

    <div class="settings-subsection models-list-section">
      <div class="settings-subsection__heading"><strong>我的模型</strong><span>可以保存多个模型，但只有一个“默认大脑”会作为 XiaoYu Autonomous Agent 的 Primary Brain。</span></div>
      <div v-if="!(catalog?.profiles.length)" class="model-empty">还没有模型。先从上方选择服务商并保存一个模型。</div>
      <div v-else class="model-table-wrap"><table class="model-table"><thead><tr><th>模型来源</th><th>服务商 / 协议</th><th>授权 / 接口</th><th>状态</th><th>小鱼大脑</th><th>操作</th></tr></thead><tbody><tr v-for="profile in catalog?.profiles ?? []" :key="profile.id"><td><strong>{{ profile.name }}</strong><small>{{ profile.model || (profile.providerKind === 'agent-provider' ? 'Provider Account' : '-') }}</small></td><td><span>{{ profile.provider }}</span><small>{{ profile.protocol }} · {{ profile.providerKind }}</small><div class="capability-row compact"><span v-for="item in capabilityLabels(profile.capabilities)" :key="item" class="capability-pill">{{ item }}</span></div></td><td><strong>{{ profile.authMode === 'subscription' ? '套餐 / 订阅' : profile.authMode === 'local' ? '本地 Runtime' : 'API Key' }}</strong><code>{{ profile.baseUrl || (profile.authMode === 'subscription' ? '官方 CLI / OAuth' : '-') }}</code></td><td><span :class="['model-status', { enabled: profile.enabled }]">{{ profile.enabled ? '启用' : '禁用' }}</span><small>{{ profile.lastTestAt ? (profile.lastTestOk ? '连接测试通过' : '最近测试失败') : (profile.hasCredential ? '授权已配置 · 未测试' : '授权未完成') }}</small></td><td><span v-if="profile.isDefault" class="default-pill">默认大脑</span><span v-else-if="!profile.brainEligible" class="capability-pill">Brain 适配中</span><button v-else type="button" class="link-button" :disabled="!canManage || !profile.enabled" @click="setDefault(profile)">设为默认</button></td><td><div class="table-actions"><button type="button" @click="edit(profile)">编辑</button><button type="button" :disabled="!canManage" @click="remove(profile)">删除</button></div></td></tr></tbody></table></div>
    </div>

    <div class="settings-footer"><span class="save-message" :class="{ active: message }">{{ message || '模型 Provider 可独立保存和测试；只有 Brain Eligible Provider 才能成为 XiaoYu 默认大脑。' }}</span><button class="btn btn--secondary" type="button" @click="refresh">刷新</button></div>
  </section>
</template>

<style scoped>
.brain-banner{display:flex;align-items:center;justify-content:space-between;gap:18px;margin-bottom:22px;padding:15px 17px;border:1px solid var(--border-default);border-radius:12px;background:var(--surface-2)}.brain-banner>div{display:grid;grid-template-columns:12px auto;column-gap:9px;align-items:center}.brain-banner small{grid-column:2;color:var(--text-muted);margin-top:3px}.brain-dot{width:9px;height:9px;border-radius:50%;background:var(--warning)}.brain-banner.ready .brain-dot{background:var(--success);box-shadow:0 0 10px color-mix(in srgb,var(--success) 60%,transparent)}
.provider-grid{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:11px}.provider-card{position:relative;display:flex;align-items:center;gap:11px;min-height:82px;padding:13px;text-align:left;border:1px solid var(--border-default);border-radius:11px;background:var(--surface-2);color:var(--text-primary)}.provider-card:hover{border-color:color-mix(in srgb,var(--accent) 45%,var(--border-default))}.provider-card.active{border-color:var(--accent);box-shadow:inset 0 0 0 1px color-mix(in srgb,var(--accent) 35%,transparent)}.provider-card>span:nth-child(2){display:grid;gap:4px;min-width:0}.provider-card strong{font-size:13px}.provider-card small{color:var(--text-muted);font-size:11px;line-height:1.35}.provider-card i{position:absolute;right:8px;top:8px;width:18px;height:18px;border-radius:50%;display:grid;place-items:center;background:var(--accent);color:white;font-size:11px;font-style:normal}.provider-logo{width:42px;height:42px;flex:0 0 42px;display:grid;place-items:center;border:1px solid var(--border-default);border-radius:10px;background:var(--surface-1);color:var(--accent);font-size:11px;font-weight:800}
.model-editor-grid{display:grid;grid-template-columns:minmax(0,1.7fr) minmax(300px,.8fr);gap:14px;margin-top:4px}.model-editor-card,.connection-card{padding:20px;border:1px solid var(--border-subtle);border-radius:12px;background:var(--surface-2)}.form-grid{display:grid;gap:12px}.form-grid.two{grid-template-columns:repeat(2,minmax(0,1fr))}.span-two{grid-column:1/-1}.field-action{display:flex;gap:8px}.field-action input{min-width:0;flex:1}.field-action button,.table-actions button{border:1px solid var(--border-default);border-radius:8px;background:var(--surface-1);color:var(--text-secondary);padding:0 11px}.model-actions{display:flex;justify-content:space-between;gap:14px;align-items:center;margin-top:16px}.model-actions>div{display:flex;gap:8px}.subscription-note{padding:10px 12px;border:1px solid var(--border-default);border-radius:9px;background:var(--surface-1)}.subscription-note strong,.subscription-note small{display:block;margin-top:5px}.subscription-note small{color:var(--text-muted);line-height:1.45}.model-permission-note{margin:12px 0 0;color:var(--warning);font-size:12px}
.connection-state{display:flex;gap:12px;align-items:center;padding:14px;border:1px solid var(--border-default);border-radius:10px;background:var(--surface-1)}.connection-state .signal{font-size:25px;color:var(--text-muted)}.connection-state.ready .signal{color:var(--success)}.connection-state.failed .signal{color:var(--danger)}.connection-state div{display:grid;gap:4px}.connection-state small{color:var(--text-muted)}.capability-row{display:flex;flex-wrap:wrap;gap:6px;margin-top:8px}.capability-row.compact{max-width:280px}.capability-pill{padding:3px 7px;border:1px solid var(--border-default);border-radius:999px;font-size:10px;color:var(--text-muted)}.connection-card dl{display:grid;gap:11px;margin:18px 0 0}.connection-card dl div{display:grid;grid-template-columns:78px minmax(0,1fr);gap:8px;font-size:12px}.connection-card dt{color:var(--text-muted)}.connection-card dd{margin:0;overflow-wrap:anywhere;color:var(--text-secondary)}
.models-list-section{margin-top:22px}.model-table-wrap{overflow:auto;border:1px solid var(--border-subtle);border-radius:11px}.model-table{width:100%;border-collapse:collapse;min-width:860px}.model-table th,.model-table td{padding:12px 13px;border-bottom:1px solid var(--border-subtle);text-align:left;vertical-align:middle}.model-table th{font-size:11px;color:var(--text-muted);font-weight:600;background:var(--surface-2)}.model-table td{font-size:12px}.model-table td strong,.model-table td small{display:block}.model-table td small{margin-top:3px;color:var(--text-muted)}.model-table code{display:block;max-width:270px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.model-status{color:var(--text-muted)}.model-status.enabled{color:var(--success)}.default-pill{display:inline-flex;padding:5px 8px;border-radius:999px;background:color-mix(in srgb,var(--success) 12%,var(--surface-1));color:var(--success);font-size:11px}.table-actions{display:flex;gap:7px}.table-actions button{padding:6px 8px}.model-empty{padding:24px;border:1px dashed var(--border-default);border-radius:10px;color:var(--text-muted);text-align:center;font-size:12px}
@media(max-width:1200px){.provider-grid{grid-template-columns:repeat(3,minmax(0,1fr))}.model-editor-grid{grid-template-columns:1fr}}@media(max-width:760px){.provider-grid{grid-template-columns:repeat(2,minmax(0,1fr))}.form-grid.two{grid-template-columns:1fr}.span-two{grid-column:auto}.model-actions{align-items:stretch;flex-direction:column}.model-actions>div{flex-wrap:wrap}}@media(max-width:520px){.provider-grid{grid-template-columns:1fr}}
</style>
