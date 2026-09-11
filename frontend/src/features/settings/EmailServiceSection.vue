<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { backend } from '../../shared/api/backend'
import type { AuthSMTPSettings, AuthUser } from '../../shared/types/backend'

const loading = ref(false)
const saving = ref(false)
const message = ref('')
const currentUser = ref<AuthUser | null>(null)
const current = ref<AuthSMTPSettings>({ configured: false, host: '', port: 587, username: '', from: '', tlsMode: 'starttls', hasPassword: false })
const form = reactive({ host: '', port: 587, username: '', password: '', from: '', tlsMode: 'starttls', accountPassword: '', securityKey: '' })

const canManage = computed(() => currentUser.value?.role === 'owner' || currentUser.value?.role === 'administrator')

function loadForm(value: AuthSMTPSettings) {
  current.value = value
  form.host = value.host || ''
  form.port = value.port || 587
  form.username = value.username || ''
  form.password = ''
  form.from = value.from || ''
  form.tlsMode = value.tlsMode || 'starttls'
  form.accountPassword = ''
  form.securityKey = ''
}

async function refresh() {
  loading.value = true
  message.value = ''
  try {
    currentUser.value = await backend.currentUser()
    if (!canManage.value) {
      message.value = '只有 Owner / Administrator 可以查看或修改系统邮箱服务。'
      return
    }
    loadForm(await backend.smtpSettings())
  } catch (error) {
    message.value = error instanceof Error ? error.message : String(error)
  } finally {
    loading.value = false
  }
}

async function save() {
  if (!canManage.value) return
  message.value = ''
  if (!form.host.trim() || !form.from.trim() || !form.accountPassword) {
    message.value = '请填写 SMTP Host、发件邮箱，并输入当前管理员账号密码确认修改。'
    return
  }
  saving.value = true
  try {
    const value = await backend.saveSMTPSettings({
      host: form.host.trim(), port: Number(form.port), username: form.username.trim(), password: form.password || undefined,
      from: form.from.trim(), tlsMode: form.tlsMode, accountPassword: form.accountPassword, securityKey: form.securityKey || undefined,
    })
    loadForm(value)
    message.value = '邮箱服务已保存。用户现在可以自行选择绑定邮箱，用于密码找回、风险通知和邮件二次验证。'
  } catch (error) {
    message.value = error instanceof Error ? error.message : String(error)
  } finally {
    saving.value = false
  }
}

onMounted(() => void refresh())
</script>

<template>
  <section class="settings-section">
    <div class="settings-section__intro">
      <div>
        <strong>邮箱服务 / SMTP</strong>
        <span>由组织管理员配置 AGMP 的邮件发送通道。邮箱是可选安全增强，不是注册、成员授权或 XiaoYu 核心能力的前置条件。</span>
      </div>
      <button class="btn btn--ghost" type="button" :disabled="loading" @click="refresh">刷新</button>
    </div>

    <div :class="['feature-card', current.configured ? 'email-service-ready' : '']">
      <span class="feature-title">{{ current.configured ? '邮箱服务已配置' : '邮箱服务未配置' }}</span>
      <p class="feature-description">
        <template v-if="current.configured">当前邮件通道可供已绑定并验证邮箱的账号使用。SMTP 密码只保存在本机 Secret Vault，页面不会读取明文。</template>
        <template v-else>密码找回、邮件验证码和邮件风险通知暂不可用；邮箱服务未配置不影响 AGMP 登录、受邀加入组织、成员核心授权或普通功能使用。</template>
      </p>
    </div>

    <div v-if="!canManage && !loading" class="auth-notice">{{ message }}</div>

    <form v-else class="feature-grid feature-grid--two" @submit.prevent="save">
      <label class="feature-card field-block">
        <span class="feature-title">SMTP Host</span>
        <small>例如 smtp.example.com；本机邮件代理也可以使用 127.0.0.1。</small>
        <input v-model.trim="form.host" autocomplete="off" placeholder="smtp.example.com" />
      </label>
      <label class="feature-card field-block">
        <span class="feature-title">端口与 TLS</span>
        <div class="email-inline-fields">
          <input v-model.number="form.port" type="number" min="1" max="65535" />
          <select v-model="form.tlsMode"><option value="starttls">STARTTLS</option><option value="tls">TLS</option><option value="none">None（仅允许本机 SMTP）</option></select>
        </div>
      </label>
      <label class="feature-card field-block">
        <span class="feature-title">SMTP 用户名</span>
        <input v-model.trim="form.username" autocomplete="username" placeholder="可按服务商要求填写" />
      </label>
      <label class="feature-card field-block">
        <span class="feature-title">SMTP 密码</span>
        <small>{{ current.hasPassword ? '已有密码安全保存；留空保持原密码。' : '首次配置时按邮件服务商要求填写。' }}</small>
        <input v-model="form.password" type="password" autocomplete="new-password" placeholder="不会回显已保存密码" />
      </label>
      <label class="feature-card field-block">
        <span class="feature-title">发件邮箱</span>
        <input v-model.trim="form.from" type="email" autocomplete="email" placeholder="agmp@example.com" />
      </label>
      <div class="feature-card field-block">
        <span class="feature-title">管理员身份确认</span>
        <small>修改邮件安全通道属于敏感设置，必须再次输入当前账号密码；若账号配置了安全密钥，也必须同时提供。</small>
        <input v-model="form.accountPassword" type="password" autocomplete="current-password" placeholder="当前管理员账号密码" />
        <input v-model="form.securityKey" type="password" autocomplete="off" placeholder="登录安全密钥（如账号已配置）" />
      </div>
      <div class="settings-footer email-span-two">
        <span class="save-message" :class="{ active: message }">{{ message || '邮箱服务是实例级配置；不同 AGMP 实例互不共享 SMTP 凭据。' }}</span>
        <button class="btn btn--primary" type="submit" :disabled="saving || loading">{{ saving ? '保存中…' : '保存邮箱服务' }}</button>
      </div>
    </form>
  </section>
</template>

<style scoped>
.email-inline-fields{display:grid;grid-template-columns:minmax(100px,.45fr) minmax(180px,1fr);gap:10px}.email-span-two{grid-column:1/-1}.email-service-ready{border-color:color-mix(in srgb,var(--accent) 45%,var(--border))}@media(max-width:720px){.email-inline-fields{grid-template-columns:1fr}.email-span-two{grid-column:auto}}
</style>
