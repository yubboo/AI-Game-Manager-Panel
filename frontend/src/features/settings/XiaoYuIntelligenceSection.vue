<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { backend } from '../../shared/api/backend'
import type {
  AuthUser,
  XiaoYuExpertDefinition,
  XiaoYuExpertSaveRequest,
  XiaoYuIntelligenceCatalog,
  XiaoYuIntelligenceVisibility,
  XiaoYuMemoryKind,
  XiaoYuMemoryRecord,
  XiaoYuMemorySaveRequest,
  XiaoYuSkillDefinition,
  XiaoYuSkillSaveRequest,
} from '../../shared/types/backend'

type Tab = 'memory' | 'skills' | 'experts'
const active = ref<Tab>('memory')
const catalog = ref<XiaoYuIntelligenceCatalog>({ memories: [], skills: [], experts: [] })
const currentUser = ref<AuthUser | null>(null)
const busy = ref(false)
const message = ref('')
const supervisor = computed(() => currentUser.value?.role === 'owner' || currentUser.value?.role === 'administrator')

const memoryForm = reactive<XiaoYuMemorySaveRequest>({
  kind: 'user', scope: { type: 'user', id: '' }, content: '', sensitivity: 'normal', confidence: 1, visibility: 'private', enabled: true,
})
const skillForm = reactive<XiaoYuSkillSaveRequest>({ name: '', description: '', prompt: '', enabled: true, visibility: 'private' })
const expertForm = reactive<XiaoYuExpertSaveRequest>({ name: '', description: '', prompt: '', enabled: true, visibility: 'private' })
const skillText = reactive({ tags: '', games: '', tools: '', checklist: '', validators: '' })
const expertText = reactive({ domains: '', games: '', knowledge: '', skills: '', tools: '', checklist: '', validators: '', recovery: '' })

function list(value: string) { return value.split(/\r?\n|,/).map(item => item.trim()).filter(Boolean) }
function joined(values?: string[]) { return (values || []).join('\n') }
function errorText(value: unknown) { return value instanceof Error ? value.message : String(value) }
function timeLabel(value?: string) { if (!value) return '—'; const date = new Date(value); return Number.isNaN(date.getTime()) ? value : date.toLocaleString() }
function kindLabel(kind: XiaoYuMemoryKind) {
  return ({ session: '会话记忆', task: '任务记忆', user: '用户长期记忆', server: '服务器记忆', instance: '实例记忆', experience: '经验记忆' } as Record<XiaoYuMemoryKind, string>)[kind] || kind
}
function visibilityLabel(value: XiaoYuIntelligenceVisibility) {
  return value === 'organization' ? '当前组织共享' : value === 'group' ? '当前管理组共享' : '私人'
}
function normalizeVisibility(value?: XiaoYuIntelligenceVisibility): XiaoYuIntelligenceVisibility {
  if (!supervisor.value || (value !== 'group' && value !== 'organization')) return 'private'
  return value
}

async function refresh() {
  busy.value = true
  try {
    const [intelligence, user] = await Promise.all([backend.xiaoyuIntelligenceCatalog(), backend.currentUser()])
    catalog.value = {
      ...intelligence,
      memories: intelligence.memories ?? [],
      skills: intelligence.skills ?? [],
      experts: intelligence.experts ?? [],
    }
    currentUser.value = user
    if (!memoryForm.scope.id && memoryForm.kind === 'user') memoryForm.scope.id = user.id
  } catch (error) {
    message.value = errorText(error)
  } finally {
    busy.value = false
  }
}

function resetMemory() {
  Object.assign(memoryForm, { id: undefined, kind: 'user', scope: { type: 'user', id: currentUser.value?.id || '' }, content: '', source: '', sensitivity: 'normal', confidence: 1, visibility: 'private', enabled: true, expiresAt: undefined })
}
function editMemory(item: XiaoYuMemoryRecord) {
  Object.assign(memoryForm, { id: item.id, kind: item.kind, scope: { ...item.scope }, content: item.content, source: item.source, sensitivity: item.sensitivity, confidence: item.confidence, visibility: item.visibility, enabled: item.enabled, expiresAt: item.expiresAt })
}
async function saveMemory() {
  busy.value = true; message.value = ''
  try {
    await backend.saveXiaoYuMemory({ ...memoryForm, scope: { ...memoryForm.scope }, visibility: normalizeVisibility(memoryForm.visibility) })
    message.value = 'Memory 已保存。敏感 Memory 只保存在当前 AGMP，本身不会自动发送给模型。'
    resetMemory(); await refresh()
  } catch (error) { message.value = errorText(error) } finally { busy.value = false }
}
async function removeMemory(item: XiaoYuMemoryRecord) {
  if (!window.confirm('确定删除这条 XiaoYu Memory 吗？')) return
  try { await backend.deleteXiaoYuIntelligence('memory', item.id); await refresh(); message.value = 'Memory 已删除。' } catch (error) { message.value = errorText(error) }
}

function resetSkill() {
  Object.assign(skillForm, { id: undefined, name: '', description: '', prompt: '', enabled: true, visibility: 'private' })
  Object.assign(skillText, { tags: '', games: '', tools: '', checklist: '', validators: '' })
}
function editSkill(item: XiaoYuSkillDefinition) {
  if (item.builtin) return
  Object.assign(skillForm, { id: item.id, name: item.name, description: item.description || '', prompt: item.prompt, enabled: item.enabled, visibility: item.visibility })
  Object.assign(skillText, { tags: joined(item.tags), games: joined(item.gameIds), tools: joined(item.toolAllowlist), checklist: joined(item.checklist), validators: joined(item.validators) })
}
async function saveSkill() {
  busy.value = true; message.value = ''
  try {
    await backend.saveXiaoYuSkill({ ...skillForm, visibility: normalizeVisibility(skillForm.visibility), tags: list(skillText.tags), gameIds: list(skillText.games), toolAllowlist: list(skillText.tools), checklist: list(skillText.checklist), validators: list(skillText.validators) })
    message.value = 'Skill 已保存。Skill 只增强小鱼规划，不能扩大 Host Tool 权限。'
    resetSkill(); await refresh()
  } catch (error) { message.value = errorText(error) } finally { busy.value = false }
}
async function removeSkill(item: XiaoYuSkillDefinition) {
  if (item.builtin || !window.confirm(`确定删除自定义 Skill「${item.name}」吗？`)) return
  try { await backend.deleteXiaoYuIntelligence('skill', item.id); await refresh(); message.value = 'Skill 已删除。' } catch (error) { message.value = errorText(error) }
}

function resetExpert() {
  Object.assign(expertForm, { id: undefined, name: '', description: '', prompt: '', enabled: true, visibility: 'private' })
  Object.assign(expertText, { domains: '', games: '', knowledge: '', skills: '', tools: '', checklist: '', validators: '', recovery: '' })
}
function editExpert(item: XiaoYuExpertDefinition) {
  if (item.builtin) return
  Object.assign(expertForm, { id: item.id, name: item.name, description: item.description || '', prompt: item.prompt, enabled: item.enabled, visibility: item.visibility })
  Object.assign(expertText, { domains: joined(item.domains), games: joined(item.gameIds), knowledge: joined(item.knowledge), skills: joined(item.skillIds), tools: joined(item.toolAllowlist), checklist: joined(item.checklist), validators: joined(item.validators), recovery: joined(item.recoveryRules) })
}
async function saveExpert() {
  busy.value = true; message.value = ''
  try {
    await backend.saveXiaoYuExpert({ ...expertForm, visibility: normalizeVisibility(expertForm.visibility), domains: list(expertText.domains), gameIds: list(expertText.games), knowledge: list(expertText.knowledge), skillIds: list(expertText.skills), toolAllowlist: list(expertText.tools), checklist: list(expertText.checklist), validators: list(expertText.validators), recoveryRules: list(expertText.recovery) })
    message.value = 'Expert 已保存。Expert 是调教/知识层，不拥有自己的系统执行器。'
    resetExpert(); await refresh()
  } catch (error) { message.value = errorText(error) } finally { busy.value = false }
}
async function removeExpert(item: XiaoYuExpertDefinition) {
  if (item.builtin || !window.confirm(`确定删除自定义 Expert「${item.name}」吗？`)) return
  try { await backend.deleteXiaoYuIntelligence('expert', item.id); await refresh(); message.value = 'Expert 已删除。' } catch (error) { message.value = errorText(error) }
}

onMounted(() => { void refresh() })
</script>

<template>
  <section class="settings-section intelligence-center">
    <div class="settings-section__intro intelligence-intro">
      <div>
        <strong>小鱼 · Memory / Skills / Experts</strong>
        <span>构建当前 AGMP 私有部署实例里的长期记忆、技能与领域专家。Organization 共享仅限当前组织成员，不会进入 AGMP 公网或其他安装实例。</span>
      </div>
      <button class="btn btn--secondary" type="button" :disabled="busy" @click="refresh">刷新</button>
    </div>

    <div class="intelligence-guard">
      <strong>安全边界</strong>
      <span>Private 只有本人可见；Group 只在当前管理组共享；Organization 只在当前组织共享。普通成员不能发布共享 Skill / Expert / Memory，Sensitive Memory 默认永不注入模型。</span>
    </div>

    <div class="intelligence-summary">
      <div><strong>{{ catalog.memories.length }}</strong><span>可见 Memory</span></div>
      <div><strong>{{ catalog.skills.length }}</strong><span>Skills</span></div>
      <div><strong>{{ catalog.experts.length }}</strong><span>Experts</span></div>
    </div>

    <div class="intelligence-tabs" role="tablist">
      <button type="button" :class="{ active: active === 'memory' }" @click="active = 'memory'">Memory 记忆</button>
      <button type="button" :class="{ active: active === 'skills' }" @click="active = 'skills'">Skills 技能</button>
      <button type="button" :class="{ active: active === 'experts' }" @click="active = 'experts'">Experts 专家</button>
    </div>

    <div v-if="active === 'memory'" class="intelligence-layout">
      <div class="intelligence-list">
        <article v-for="item in catalog.memories" :key="item.id" class="feature-card intelligence-item">
          <div class="item-head"><div><strong>{{ kindLabel(item.kind) }}</strong><small>{{ visibilityLabel(item.visibility) }} · {{ item.scope.type }} / {{ item.scope.id }}</small></div><span class="status-pill">{{ item.sensitivity === 'sensitive' ? '敏感·不注入模型' : '普通' }}</span></div>
          <p>{{ item.content }}</p>
          <small>来源 {{ item.source }} · 可信度 {{ item.confidence }} · {{ timeLabel(item.updatedAt) }}</small>
          <div class="row-actions"><button class="btn btn--secondary" type="button" @click="editMemory(item)">编辑</button><button class="btn btn--secondary" type="button" @click="removeMemory(item)">删除</button></div>
        </article>
        <div v-if="!catalog.memories.length" class="empty-box">当前没有可见 Memory。任务 Trace 不会自动冒充长期记忆。</div>
      </div>
      <form class="feature-card intelligence-form" @submit.prevent="saveMemory">
        <strong>{{ memoryForm.id ? '编辑 Memory' : '新增 Memory' }}</strong>
        <label>类型<select v-model="memoryForm.kind"><option value="user">用户长期记忆</option><option value="session">会话记忆</option><option value="task">任务记忆</option><option value="server">服务器记忆</option><option value="instance">实例记忆</option><option value="experience">经验记忆</option></select></label>
        <label>作用域 ID<input v-model.trim="memoryForm.scope.id" maxlength="256" placeholder="用户/经验私人记忆会由服务端绑定当前账号；Task 填 Run ID" /></label>
        <label>内容<textarea v-model="memoryForm.content" rows="6" maxlength="16384" placeholder="保存真正对未来任务有用的事实、偏好或经验。不要保存 API Key、Token、密码。" /></label>
        <div class="two-cols"><label>敏感级别<select v-model="memoryForm.sensitivity"><option value="normal">普通 · 可进入匹配上下文</option><option value="sensitive">敏感 · 只本地管理，不发模型</option></select></label><label>可信度<input v-model.number="memoryForm.confidence" type="number" min="0" max="1" step="0.05" /></label></div>
        <label v-if="supervisor">共享范围<select v-model="memoryForm.visibility"><option value="private">Private · 仅本人</option><option v-if="!['user','session','task'].includes(memoryForm.kind)" value="group">Group · 当前管理组</option><option v-if="!['user','session','task'].includes(memoryForm.kind)" value="organization">Organization · 当前组织</option></select></label>
        <label v-else>共享范围<input value="Private · 普通成员只能保存本人私人记忆" readonly /></label>
        <label class="check-line"><input v-model="memoryForm.enabled" type="checkbox" />启用这条 Memory</label>
        <div class="row-actions"><button v-if="memoryForm.id" class="btn btn--secondary" type="button" @click="resetMemory">取消编辑</button><button class="btn btn--primary" :disabled="busy || !memoryForm.content.trim()">保存 Memory</button></div>
      </form>
    </div>

    <div v-else-if="active === 'skills'" class="intelligence-layout">
      <div class="intelligence-list">
        <article v-for="item in catalog.skills" :key="item.id" class="feature-card intelligence-item">
          <div class="item-head"><div><strong>{{ item.name }}</strong><small>{{ item.builtin ? 'AGMP 内置 Skill' : visibilityLabel(item.visibility) }}</small></div><span class="status-pill">{{ item.enabled ? '启用' : '停用' }}</span></div>
          <p>{{ item.description || item.prompt }}</p><small v-if="item.toolAllowlist?.length">建议 Tool：{{ item.toolAllowlist.join(' · ') }}</small>
          <div v-if="!item.builtin" class="row-actions"><button class="btn btn--secondary" type="button" @click="editSkill(item)">编辑</button><button class="btn btn--secondary" type="button" @click="removeSkill(item)">删除</button></div>
        </article>
      </div>
      <form class="feature-card intelligence-form" @submit.prevent="saveSkill">
        <strong>{{ skillForm.id ? '编辑自定义 Skill' : '创建自定义 Skill' }}</strong>
        <label>名称<input v-model.trim="skillForm.name" maxlength="160" placeholder="例如：我的 Paper 升级流程" /></label>
        <label>说明<input v-model.trim="skillForm.description" maxlength="1000" placeholder="什么时候应该使用这个 Skill" /></label>
        <label>增强 Prompt<textarea v-model="skillForm.prompt" rows="6" maxlength="32768" placeholder="专业流程与限制。Prompt 不能获得额外 Tool 权限。" /></label>
        <label>标签<textarea v-model="skillText.tags" rows="2" placeholder="minecraft, update" /></label>
        <label>适用 Game ID<textarea v-model="skillText.games" rows="2" placeholder="minecraft" /></label>
        <label>建议 Tool<textarea v-model="skillText.tools" rows="3" placeholder="每行一个 Tool 名称" /></label>
        <label>Checklist<textarea v-model="skillText.checklist" rows="4" placeholder="每行一个检查步骤" /></label>
        <label>Validators<textarea v-model="skillText.validators" rows="3" placeholder="每行一个结果验证条件" /></label>
        <label v-if="supervisor">共享范围<select v-model="skillForm.visibility"><option value="private">Private</option><option value="group">Group</option><option value="organization">Organization</option></select></label>
        <label v-else>共享范围<input value="Private · 普通成员不能发布组织 Skill" readonly /></label>
        <label class="check-line"><input v-model="skillForm.enabled" type="checkbox" />启用 Skill</label>
        <div class="row-actions"><button v-if="skillForm.id" class="btn btn--secondary" type="button" @click="resetSkill">取消编辑</button><button class="btn btn--primary" :disabled="busy || !skillForm.name.trim() || !skillForm.prompt.trim()">保存 Skill</button></div>
      </form>
    </div>

    <div v-else class="intelligence-layout">
      <div class="intelligence-list">
        <article v-for="item in catalog.experts" :key="item.id" class="feature-card intelligence-item">
          <div class="item-head"><div><strong>{{ item.name }}</strong><small>{{ item.builtin ? 'AGMP 内置领域专家' : visibilityLabel(item.visibility) }}</small></div><span class="status-pill">{{ item.enabled ? '启用' : '停用' }}</span></div>
          <p>{{ item.description || item.prompt }}</p>
          <details><summary>查看专家调教内容</summary><p class="prompt-preview">{{ item.prompt }}</p><ul><li v-for="line in item.checklist || []" :key="line">{{ line }}</li></ul></details>
          <div v-if="!item.builtin" class="row-actions"><button class="btn btn--secondary" type="button" @click="editExpert(item)">编辑</button><button class="btn btn--secondary" type="button" @click="removeExpert(item)">删除</button></div>
        </article>
      </div>
      <form class="feature-card intelligence-form" @submit.prevent="saveExpert">
        <strong>{{ expertForm.id ? '编辑自定义 Expert' : '创建自定义 Expert' }}</strong>
        <label>专家名称<input v-model.trim="expertForm.name" maxlength="160" placeholder="例如：公司 Minecraft 发布专家" /></label>
        <label>说明<input v-model.trim="expertForm.description" maxlength="1000" placeholder="该专家负责什么领域" /></label>
        <label>专家 Prompt<textarea v-model="expertForm.prompt" rows="7" maxlength="32768" placeholder="定义专业判断规则。自定义 Prompt 永远不能覆盖 AGMP Host 安全边界。" /></label>
        <label>领域<textarea v-model="expertText.domains" rows="2" placeholder="Minecraft, Paper, Java" /></label>
        <label>适用 Game ID<textarea v-model="expertText.games" rows="2" placeholder="minecraft" /></label>
        <label>稳定知识点<textarea v-model="expertText.knowledge" rows="5" placeholder="每行一个知识点；动态状态必须通过 Tool Observation 获取" /></label>
        <label>关联 Skill ID<textarea v-model="expertText.skills" rows="3" placeholder="builtin.skill.safe-change" /></label>
        <label>建议 Tool<textarea v-model="expertText.tools" rows="3" /></label>
        <label>Checklist<textarea v-model="expertText.checklist" rows="4" /></label>
        <label>Validators<textarea v-model="expertText.validators" rows="3" /></label>
        <label>Recovery Rules<textarea v-model="expertText.recovery" rows="3" /></label>
        <label v-if="supervisor">共享范围<select v-model="expertForm.visibility"><option value="private">Private</option><option value="group">Group</option><option value="organization">Organization</option></select></label>
        <label v-else>共享范围<input value="Private · 普通成员不能发布组织 Expert" readonly /></label>
        <label class="check-line"><input v-model="expertForm.enabled" type="checkbox" />启用 Expert</label>
        <div class="row-actions"><button v-if="expertForm.id" class="btn btn--secondary" type="button" @click="resetExpert">取消编辑</button><button class="btn btn--primary" :disabled="busy || !expertForm.name.trim() || !expertForm.prompt.trim()">保存 Expert</button></div>
      </form>
    </div>

    <p class="intelligence-message">{{ message || 'Memory 是上下文，不是权限；Skill / Expert 是调教层，不是第二套执行器。' }}</p>
  </section>
</template>

<style scoped>
.intelligence-center { display: grid; gap: 18px; }
.intelligence-intro { justify-content: space-between; }
.intelligence-guard { display: grid; gap: 5px; padding: 14px 16px; border: 1px solid var(--border); border-radius: 14px; background: var(--surface-2); }
.intelligence-guard span, .intelligence-item small, .intelligence-message { color: var(--muted); }
.intelligence-summary { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; }
.intelligence-summary > div { display: grid; gap: 3px; padding: 15px; border: 1px solid var(--border); border-radius: 14px; background: var(--surface-2); }
.intelligence-summary strong { font-size: 24px; }.intelligence-summary span { color: var(--muted); font-size: 12px; }
.intelligence-tabs { display: flex; gap: 8px; flex-wrap: wrap; padding-bottom: 10px; border-bottom: 1px solid var(--border); }
.intelligence-tabs button { padding: 9px 14px; border: 0; border-radius: 10px; background: transparent; color: var(--muted); cursor: pointer; }
.intelligence-tabs button.active { background: var(--accent-soft); color: var(--text); }
.intelligence-layout { display: grid; grid-template-columns: minmax(0, 1.15fr) minmax(320px, .85fr); gap: 16px; align-items: start; }
.intelligence-list, .intelligence-item, .intelligence-form { display: grid; gap: 12px; }.intelligence-item p { margin: 0; white-space: pre-wrap; }
.item-head { display: flex; gap: 12px; justify-content: space-between; align-items: flex-start; }.item-head > div { display: grid; gap: 4px; }
.intelligence-form { position: sticky; top: 16px; }.intelligence-form label { display: grid; gap: 6px; color: var(--muted); font-size: 13px; }
.intelligence-form input, .intelligence-form select, .intelligence-form textarea { width: 100%; box-sizing: border-box; }
.two-cols { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 10px; }.check-line { display: flex !important; flex-direction: row; align-items: center; gap: 8px !important; }.check-line input { width: auto; }
.row-actions { display: flex; flex-wrap: wrap; gap: 8px; justify-content: flex-end; }.empty-box { padding: 24px; text-align: center; color: var(--muted); border: 1px dashed var(--border); border-radius: 14px; }
.prompt-preview { color: var(--muted); font-size: 12px; white-space: pre-wrap; }.intelligence-message { margin: 0; font-size: 13px; }
@media (max-width: 980px) { .intelligence-layout { grid-template-columns: 1fr; }.intelligence-form { position: static; } }
@media (max-width: 640px) { .intelligence-summary, .two-cols { grid-template-columns: 1fr; } }
</style>
