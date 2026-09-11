<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { backend } from '../../shared/api/backend'
import type { AuthBootstrapStatus, AuthInvitationPreview, AuthUser } from '../../shared/types/backend'

const emit = defineEmits<{ ready: [] }>()
type Stage = 'loading' | 'bootstrap' | 'invite' | 'login' | 'reset' | 'error'

const stage = ref<Stage>('loading')
const error = ref('')
const busy = ref(false)
const bootstrap = ref<AuthBootstrapStatus | null>(null)
const user = ref<AuthUser | null>(null)

const ownerUsername = ref('')
const ownerPassword = ref('')
const ownerPasswordConfirm = ref('')
const ownerSecurityKey = ref('')
const ownerRequireSecurityKey = ref(false)
const ownerKeySaved = ref(false)

const loginUsername = ref('')
const loginPassword = ref('')
const loginSecurityKey = ref('')
const inviteTokenInput = ref('')
const inviteToken = ref('')
const invitePreview = ref<AuthInvitationPreview | null>(null)
const inviteUsername = ref('')
const inviteDisplayName = ref('')
const inviteEmail = ref('')
const invitePassword = ref('')
const invitePasswordConfirm = ref('')

const resetUsername = ref('')
const resetCode = ref('')
const resetPassword = ref('')
const resetPasswordConfirm = ref('')
const resetRequested = ref(false)

const ownerFormComplete = computed(() => {
  const username = ownerUsername.value.trim()
  const passwordLength = ownerPassword.value.length
  const keyReady = !ownerRequireSecurityKey.value || (ownerSecurityKey.value.startsWith('BFK1-') && ownerKeySaved.value)
  return /^[A-Za-z0-9._-]{3,32}$/.test(username)
    && passwordLength >= 8 && passwordLength <= 128
    && ownerPassword.value === ownerPasswordConfirm.value
    && keyReady
})
const inviteFormComplete = computed(() => {
  const passwordLength = invitePassword.value.length
  return Boolean(invitePreview.value)
    && /^[A-Za-z0-9._-]{3,32}$/.test(inviteUsername.value.trim())
    && passwordLength >= 8 && passwordLength <= 128
    && invitePassword.value === invitePasswordConfirm.value
})

function messageOf(value: unknown) { return value instanceof Error ? value.message : String(value) }

function downloadKey(key: string, username: string) {
  if (!key) return
  const safeUser = (username || 'Owner').replace(/[^a-zA-Z0-9_.-]+/g, '_')
  const text = ['AI Game Manager Panel 登录安全密钥', `账号: ${username || 'Owner'}`, `密钥: ${key}`, '', '重要：如启用密钥登录/高风险验证，请离线保存。AGMP 不保存密钥明文。'].join('\r\n')
  const blob = new Blob(['\ufeff', text], { type: 'text/plain;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a'); anchor.href = url; anchor.download = `AI Game Manager Panel-${safeUser}-Security-Key.txt`
  document.body.appendChild(anchor); anchor.click(); anchor.remove(); URL.revokeObjectURL(url)
}
async function copyKey(key: string, event?: MouseEvent) {
  if (!key) return
  try { await navigator.clipboard.writeText(key) } catch {
    const textarea = document.createElement('textarea'); textarea.value = key; document.body.appendChild(textarea); textarea.select(); document.execCommand('copy'); textarea.remove()
  }
  const button = event?.currentTarget instanceof HTMLButtonElement ? event.currentTarget : null
  if (button) { const label = button.textContent || '复制密钥'; button.textContent = '已复制'; window.setTimeout(() => { if (button.isConnected) button.textContent = label }, 1200) }
}
async function generateOwnerSecurityKey() {
  error.value = ''; busy.value = true
  try { const material = await backend.generateSecurityKey(); ownerSecurityKey.value = material.key; ownerKeySaved.value = false; ownerRequireSecurityKey.value = false }
  catch (value) { error.value = messageOf(value) }
  finally { busy.value = false }
}

function invitationTokenFromLocation() {
  try {
    const hash = new URL(window.location.href).hash
    const q = hash.indexOf('?')
    if (q < 0) return ''
    return new URLSearchParams(hash.slice(q + 1)).get('invite')?.trim() || ''
  } catch { return '' }
}
function clearInvitationFromLocation() {
  try {
    const url = new URL(window.location.href); url.hash = '#/'
    window.history.replaceState(null, '', `${url.pathname}${url.search}${url.hash}`)
  } catch { /* desktop custom schemes may not expose History API */ }
}
async function inspectInvitation(value: string) {
  const token = value.trim(); if (!token) return false
  error.value = ''; busy.value = true
  try {
    const preview = await backend.inspectInvitation(token)
    invitePreview.value = preview; inviteToken.value = token; inviteTokenInput.value = token
    inviteUsername.value = preview.targetUsername || ''; inviteEmail.value = preview.targetEmail || ''
    inviteDisplayName.value = ''; invitePassword.value = ''; invitePasswordConfirm.value = ''
    backend.setSessionToken(''); stage.value = 'invite'; return true
  } catch (value) { invitePreview.value = null; inviteToken.value = ''; error.value = messageOf(value); return false }
  finally { busy.value = false }
}
async function inspectEnteredInvitation() {
  if (!inviteTokenInput.value.trim()) { error.value = '请粘贴管理员发给你的完整 AGMP 邀请令牌。'; return }
  await inspectInvitation(inviteTokenInput.value)
}
async function registerInvitedMember() {
  error.value = ''
  if (!inviteFormComplete.value) { error.value = '请填写完整用户名与密码，并确认两次密码一致。'; return }
  busy.value = true
  try {
    const session = await backend.registerInvitedUser({ invitationToken: inviteToken.value, username: inviteUsername.value.trim(), displayName: inviteDisplayName.value.trim(), password: invitePassword.value, email: inviteEmail.value.trim() || undefined })
    user.value = session.user; clearInvitationFromLocation(); emit('ready')
  } catch (value) { error.value = messageOf(value) }
  finally { busy.value = false }
}

async function initializeGate() {
  error.value = ''; stage.value = 'loading'
  try {
    const locationInvite = invitationTokenFromLocation()
    if (locationInvite) {
      bootstrap.value = await backend.authBootstrapStatus()
      if (await inspectInvitation(locationInvite)) return
      stage.value = 'login'
      return
    }

    // SILENT_SESSION_RESTORE：正常刷新优先用现有 Cookie / Wails Token 恢复当前用户。
    // 已登录用户只需要一次 currentUser 往返，不先展示登录页，也不先等待 Bootstrap 请求。
    if (backend.adapter() !== 'wails' || backend.getSessionToken()) {
      try {
        user.value = await backend.currentUser()
        emit('ready')
        return
      } catch {
        // 会话确实不存在或已过期时，才进入 Bootstrap / Login 判定。
      }
    }

    bootstrap.value = await backend.authBootstrapStatus()
    if (bootstrap.value.registrationOpen) { backend.setSessionToken(''); stage.value = 'bootstrap'; return }
    if (!bootstrap.value.ownerExists) { backend.setSessionToken(''); error.value = bootstrap.value.storeError || 'Bootstrap 已锁定，但无法确认最高管理员账号。请恢复受信任的 auth 数据。'; stage.value = 'error'; return }
    stage.value = 'login'
  } catch (value) { error.value = messageOf(value); stage.value = 'error' }
}

async function createOwner() {
  error.value = ''
  if (!ownerFormComplete.value) { error.value = ownerRequireSecurityKey.value ? '请填写账号密码；若启用安全密钥登录，请先生成并确认已保存密钥。' : '请填写完整管理员账号与密码。'; return }
  busy.value = true
  try {
    const keepKey = ownerKeySaved.value && ownerSecurityKey.value.startsWith('BFK1-')
    const session = await backend.createInitialAdministrator({ username: ownerUsername.value.trim(), displayName: '超级管理员', password: ownerPassword.value, securityKey: keepKey ? ownerSecurityKey.value : '', requireSecurityKey: ownerRequireSecurityKey.value })
    user.value = session.user; ownerSecurityKey.value = ''; ownerKeySaved.value = false; bootstrap.value = await backend.authBootstrapStatus(); emit('ready')
  } catch (value) { error.value = messageOf(value) }
  finally { busy.value = false }
}
async function login() {
  error.value = ''; busy.value = true
  try { const session = await backend.login({ username: loginUsername.value.trim(), password: loginPassword.value, securityKey: loginSecurityKey.value.trim() }); user.value = session.user; loginSecurityKey.value = ''; emit('ready') }
  catch (value) { error.value = messageOf(value) }
  finally { busy.value = false }
}
async function requestReset() {
  error.value = ''
  if (!resetUsername.value.trim()) { error.value = '请输入需要找回的用户名。'; return }
  busy.value = true
  try { await backend.requestPasswordReset({ username: resetUsername.value.trim() }); resetRequested.value = true }
  catch (value) { error.value = messageOf(value) }
  finally { busy.value = false }
}
async function confirmReset() {
  error.value = ''
  if (!resetCode.value.trim() || resetPassword.value.length < 8 || resetPassword.value !== resetPasswordConfirm.value) { error.value = '请填写 8 位验证码和至少 8 位的新密码，并确认两次密码一致。'; return }
  busy.value = true
  try { await backend.confirmPasswordReset({ username: resetUsername.value.trim(), code: resetCode.value.trim(), newPassword: resetPassword.value }); resetRequested.value = false; resetCode.value = ''; resetPassword.value = ''; resetPasswordConfirm.value = ''; loginUsername.value = resetUsername.value.trim(); stage.value = 'login'; error.value = '' }
  catch (value) { error.value = messageOf(value) }
  finally { busy.value = false }
}

onMounted(() => void initializeGate())
</script>

<template>
  <!-- SILENT_AUTH_LOADING：刷新恢复会话时不渲染登录卡，避免认证页闪现。 -->
  <main v-if="stage === 'loading'" class="auth-silent-restore" aria-hidden="true"></main>
  <main v-else class="auth-gate">
    <header class="auth-brand-floating"><div class="auth-brand"><div class="auth-flame">◆</div><div><span class="eyebrow">AI GAME MANAGER ACCESS</span><h1>AI游戏管理器面板</h1><p>实例身份、组织邀请与安全登录</p></div></div></header>
    <aside class="auth-stage-floating" aria-live="polite"><div class="auth-stage-summary">
      <template v-if="stage === 'bootstrap'"><span class="auth-stage-index">01</span><div><strong>创建最高管理员</strong><small>唯一一次公开注册入口</small></div></template>
      <template v-else-if="stage === 'invite'"><span class="auth-stage-index">02</span><div><strong>受邀成员注册</strong><small>仅当前实例签名邀请可用</small></div></template>
      <template v-else-if="stage === 'reset'"><span class="auth-stage-index">↺</span><div><strong>找回密码</strong><small>仅已绑定验证邮箱且管理员已配置邮件服务</small></div></template>
      <template v-else-if="stage === 'login'"><span class="auth-stage-index">01</span><div><strong>组织成员登录</strong><small>公开注册已关闭</small></div></template>
      <template v-else-if="stage === 'error'"><span class="auth-stage-index">!</span><div><strong>安全状态不可用</strong><small>系统采用失效关闭策略</small></div></template>
      <template v-else><span class="auth-stage-index">···</span><div><strong>正在检查启动状态</strong><small>实例身份与登录会话</small></div></template>
    </div></aside>

    <section class="auth-center-stage"><section :class="['auth-card', 'panel', `auth-card--${stage}`]">
      <div v-if="stage === 'error'" class="auth-form"><div class="auth-error">{{ error || '无法确认账号与 Bootstrap 状态。' }}</div><button class="btn btn--primary auth-submit" type="button" :disabled="busy" @click="initializeGate">重新检查</button></div>

      <form v-else-if="stage === 'bootstrap'" class="auth-form" @submit.prevent="createOwner">
        <div class="auth-notice auth-notice--important">首个 Owner 创建成功后，公开注册永久关闭。后续成员只能由管理员签发当前实例的邀请，再由成员自己注册。</div>
        <div v-if="bootstrap?.instanceFingerprint" class="auth-notice">当前实例安全指纹：<strong>{{ bootstrap.instanceFingerprint }}</strong><br />该指纹由首次安装生成，正常版本升级不会改变。</div>
        <label>管理员用户名<input v-model.trim="ownerUsername" autocomplete="username" required minlength="3" maxlength="32" placeholder="例如：admin" /></label>
        <label>密码<input v-model="ownerPassword" type="password" autocomplete="new-password" required minlength="8" maxlength="128" placeholder="至少 8 个字符" /></label>
        <label>确认密码<input v-model="ownerPasswordConfirm" type="password" autocomplete="new-password" required minlength="8" maxlength="128" placeholder="再次输入密码" /></label>
        <section class="auth-security-box">
          <div class="auth-security-box__heading"><div><strong>登录安全密钥</strong><small>可选增强，不是首次注册必填项</small></div><button class="btn btn--secondary" type="button" :disabled="busy" @click="generateOwnerSecurityKey">{{ ownerSecurityKey ? '重新生成' : '生成随机密钥' }}</button></div>
          <div v-if="!ownerSecurityKey" class="auth-notice">可以直接跳过。需要更高安全性时再生成、保存并选择启用。</div>
          <template v-else><label>一次性显示的安全密钥<input :value="ownerSecurityKey" readonly spellcheck="false" /></label><div class="auth-actions"><button class="btn btn--secondary" type="button" @click="copyKey(ownerSecurityKey, $event)">复制密钥</button><button class="btn btn--secondary" type="button" @click="downloadKey(ownerSecurityKey, ownerUsername)">下载密钥文件</button></div><label class="auth-checkbox"><input v-model="ownerKeySaved" type="checkbox" /><span>我已安全保存此密钥</span></label><label class="auth-checkbox"><input v-model="ownerRequireSecurityKey" type="checkbox" :disabled="!ownerKeySaved" /><span>登录时要求账号 + 密码 + 安全密钥</span></label></template>
        </section>
        <div class="auth-notice">邮箱不在这里强制绑定。登录后可在账号安全中自愿绑定；SMTP 由管理员在“设置中心 → 邮箱服务”统一配置。</div>
        <div v-if="error" class="auth-error">{{ error }}</div><button class="btn btn--primary auth-submit" type="submit" :disabled="busy || !ownerFormComplete">{{ busy ? '正在创建…' : '创建最高管理员并进入 AGMP' }}</button>
      </form>

      <form v-else-if="stage === 'invite'" class="auth-form" @submit.prevent="registerInvitedMember">
        <div class="auth-notice auth-notice--important">这是管理员签发的受邀注册，不是公开注册。请先通过可信渠道核对实例安全指纹与邀请校验码。</div>
        <div v-if="invitePreview" class="security-status-grid"><div><span>组织</span><strong>{{ invitePreview.organizationName }}</strong></div><div><span>角色</span><strong>{{ invitePreview.role }}</strong></div><div><span>实例安全指纹</span><strong>{{ invitePreview.instanceFingerprint }}</strong></div><div><span>邀请校验码</span><strong>{{ invitePreview.verificationCode }}</strong></div></div>
        <label>用户名<input v-model.trim="inviteUsername" autocomplete="username" :readonly="Boolean(invitePreview?.targetUsername)" required minlength="3" maxlength="32" /></label>
        <label>显示名称<input v-model.trim="inviteDisplayName" maxlength="40" placeholder="例如：值班运维" /></label>
        <label>邮箱（可选）<input v-model.trim="inviteEmail" type="email" :readonly="Boolean(invitePreview?.targetEmail)" placeholder="可留空；用于找回密码和安全验证" /></label>
        <label>密码<input v-model="invitePassword" type="password" autocomplete="new-password" required minlength="8" maxlength="128" /></label>
        <label>确认密码<input v-model="invitePasswordConfirm" type="password" autocomplete="new-password" required minlength="8" maxlength="128" /></label>
        <div class="auth-notice">完成注册后只是组织成员。XiaoYu/高级自动化等核心能力还需要 Owner 单独签发成员核心授权码。</div>
        <div v-if="error" class="auth-error">{{ error }}</div><button class="btn btn--primary auth-submit" type="submit" :disabled="busy || !inviteFormComplete">{{ busy ? '正在加入…' : '接受邀请并加入组织' }}</button><button class="btn btn--secondary" type="button" @click="stage = 'login'; error = ''">返回登录</button>
      </form>

      <form v-else-if="stage === 'reset'" class="auth-form" @submit.prevent="resetRequested ? confirmReset() : requestReset()">
        <div class="auth-notice">密码找回仅对“已绑定并验证邮箱 + 管理员已配置 SMTP”的账号生效。未绑定邮箱不影响正常使用，但不能使用邮箱找回密码。</div>
        <label>用户名<input v-model.trim="resetUsername" autocomplete="username" required /></label>
        <template v-if="resetRequested"><label>邮箱验证码<input v-model.trim="resetCode" inputmode="numeric" maxlength="8" /></label><label>新密码<input v-model="resetPassword" type="password" autocomplete="new-password" minlength="8" /></label><label>确认新密码<input v-model="resetPasswordConfirm" type="password" autocomplete="new-password" minlength="8" /></label></template>
        <div v-if="error" class="auth-error">{{ error }}</div><button class="btn btn--primary auth-submit" type="submit" :disabled="busy">{{ resetRequested ? '确认重置密码' : '发送找回验证码' }}</button><button class="btn btn--secondary" type="button" @click="stage = 'login'; error = ''">返回登录</button>
      </form>

      <form v-else-if="stage === 'login'" class="auth-form" @submit.prevent="login">
        <div class="auth-notice">公开注册已关闭。已有成员登录；陌生用户必须联系当前 AGMP 实例管理员获取邀请。</div>
        <div v-if="bootstrap?.instanceFingerprint" class="auth-notice">当前实例安全指纹：<strong>{{ bootstrap.instanceFingerprint }}</strong></div>
        <label>用户名<input v-model.trim="loginUsername" autocomplete="username" required /></label><label>密码<input v-model="loginPassword" type="password" autocomplete="current-password" required /></label><label>登录安全密钥（按需）<input v-model.trim="loginSecurityKey" type="password" autocomplete="off" placeholder="仅账号已启用时需要" /></label>
        <details class="auth-security-box"><summary>收到管理员邀请？</summary><label>完整邀请令牌<input v-model.trim="inviteTokenInput" autocomplete="off" spellcheck="false" placeholder="AGMPI1.…" /></label><button class="btn btn--secondary" type="button" :disabled="busy || !inviteTokenInput.trim()" @click="inspectEnteredInvitation">校验邀请</button></details>
        <div v-if="error" class="auth-error">{{ error }}</div><button class="btn btn--primary auth-submit" type="submit" :disabled="busy">{{ busy ? '正在登录…' : '登录 AGMP' }}</button><button class="btn btn--secondary" type="button" @click="resetUsername = loginUsername; resetRequested = false; error = ''; stage = 'reset'">忘记密码</button>
      </form>
    </section></section>
  </main>
</template>
