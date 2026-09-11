<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { backend } from '../../shared/api/backend'
import type {
  AuthCreatedInvitation, AuthCreatedMemberAuthorization, AuthEmailSecurityStatus,
  AuthInvitationView, AuthMemberAuthorizationView, AuthOrganization, AuthSecurityStatus, AuthUser,
} from '../../shared/types/backend'

const users = ref<AuthUser[]>([])
const invitations = ref<AuthInvitationView[]>([])
const coreGrants = ref<AuthMemberAuthorizationView[]>([])
const current = ref<AuthUser | null>(null)
const organization = ref<AuthOrganization | null>(null)
const emailStatus = ref<AuthEmailSecurityStatus | null>(null)
const securityStatus = ref<AuthSecurityStatus | null>(null)
const loading = ref(false)
const busy = ref(false)
const message = ref('')

const inviteTargetUsername = ref('')
const inviteTargetEmail = ref('')
const inviteRole = ref<'administrator' | 'operator'>('operator')
const inviteHours = ref(24)
const createdInvite = ref<AuthCreatedInvitation | null>(null)

const selectedMemberId = ref('')
const grantHours = ref(24)
const createdGrant = ref<AuthCreatedMemberAuthorization | null>(null)
const redeemCode = ref('')

const stepPassword = ref('')
const stepSecurityKey = ref('')
const emailInput = ref('')
const emailPassword = ref('')
const emailCode = ref('')
const securityPassword = ref('')
const newSecurityKey = ref('')

const profileName = ref('')
const canInvite = computed(() => current.value?.role === 'owner' || current.value?.role === 'administrator')
const isOwner = computed(() => current.value?.role === 'owner')
const selectableMembers = computed(() => users.value.filter(user => user.role !== 'owner'))
const invitationLink = computed(() => {
  if (!createdInvite.value?.token || !['http:', 'https:'].includes(window.location.protocol)) return ''
  return `${window.location.origin}${window.location.pathname}#/?invite=${encodeURIComponent(createdInvite.value.token)}`
})

function text(error: unknown) { return error instanceof Error ? error.message : String(error) }
function time(value?: number) { return value ? new Date(value * 1000).toLocaleString() : '—' }
function coreLabel(value: string) { return value === 'authorized' ? '核心已授权' : value === 'suspended' ? '核心已暂停' : '等待核心授权' }
function riskLabel(value: string) { return value === 'locked' ? '已锁定' : value === 'challenge' ? '待二验' : '正常' }

async function copy(value: string, event?: MouseEvent) {
  if (!value) return
  try { await navigator.clipboard.writeText(value) } catch {
    const area = document.createElement('textarea'); area.value = value; document.body.appendChild(area); area.select(); document.execCommand('copy'); area.remove()
  }
  const button = event?.currentTarget instanceof HTMLButtonElement ? event.currentTarget : null
  if (button) { const old = button.textContent || '复制'; button.textContent = '已复制'; setTimeout(() => { if (button.isConnected) button.textContent = old }, 1200) }
}

async function refresh() {
  loading.value = true; message.value = ''
  try {
    const [me, list, org, mail, sec] = await Promise.all([backend.currentUser(), backend.listUsers(), backend.authOrganization(), backend.myEmailSecurityStatus(), backend.mySecurityStatus()])
    current.value = me; users.value = list; organization.value = org; emailStatus.value = mail; securityStatus.value = sec; profileName.value = me.displayName
    if (canInvite.value) invitations.value = await backend.listUserInvitations()
    if (isOwner.value) coreGrants.value = await backend.listMemberCoreAuthorizations()
    if (!selectedMemberId.value && selectableMembers.value.length) selectedMemberId.value = selectableMembers.value[0].id
  } catch (error) { message.value = text(error) }
  finally { loading.value = false }
}

async function confirmStepUp() {
  busy.value = true; message.value = ''
  try { await backend.confirmMyCredentialStepUp({ password: stepPassword.value, securityKey: stepSecurityKey.value || undefined }); stepPassword.value = ''; stepSecurityKey.value = ''; message.value = '当前会话的高风险操作身份确认已完成，短时间内无需重复输入。' }
  catch (error) { message.value = text(error) }
  finally { busy.value = false }
}

async function createInvitation() {
  busy.value = true; message.value = ''; createdInvite.value = null
  try {
    createdInvite.value = await backend.createUserInvitation({ role: inviteRole.value, targetUsername: inviteTargetUsername.value.trim() || undefined, targetEmail: inviteTargetEmail.value.trim() || undefined, expiresInHours: inviteHours.value })
    invitations.value = await backend.listUserInvitations(); message.value = '邀请已签发。完整 Token 只在本次结果中显示，请通过可信渠道发送并核对实例指纹/校验码。'
  } catch (error) { message.value = text(error) }
  finally { busy.value = false }
}
async function revokeInvitation(id: string) { busy.value = true; try { await backend.revokeUserInvitation(id); invitations.value = await backend.listUserInvitations() } catch (error) { message.value = text(error) } finally { busy.value = false } }

async function createCoreGrant() {
  if (!selectedMemberId.value) return
  busy.value = true; message.value = ''; createdGrant.value = null
  try { createdGrant.value = await backend.createMemberCoreAuthorization({ userId: selectedMemberId.value, expiresInHours: grantHours.value }); coreGrants.value = await backend.listMemberCoreAuthorizations(); message.value = '成员核心授权码已签发。它只对指定成员、当前实例和当前组织有效，并且只能使用一次。' }
  catch (error) { message.value = text(error) }
  finally { busy.value = false }
}
async function redeemCoreGrant() { busy.value = true; message.value = ''; try { current.value = await backend.redeemMyCoreAuthorization({ token: redeemCode.value.trim() }); redeemCode.value = ''; await refresh(); message.value = '核心功能授权成功。仍会继续受官方 AGMP License、角色权限和风险策略约束。' } catch (error) { message.value = text(error) } finally { busy.value = false } }
async function revokeCore(userId: string) { busy.value = true; try { await backend.revokeMemberCoreAccess(userId); await refresh(); message.value = '该成员核心访问已暂停，旧 Session 已立即失效。' } catch (error) { message.value = text(error) } finally { busy.value = false } }
async function clearRisk(userId: string) { busy.value = true; try { await backend.clearMemberRisk(userId); await refresh(); message.value = '成员风控状态已由 Owner 解除；这不会冒充该成员完成后续二次验证。' } catch (error) { message.value = text(error) } finally { busy.value = false } }
async function removeMember(userId: string) { if (!confirm('确定把该成员移出当前组织吗？其现有会话会立即失效。')) return; busy.value = true; try { await backend.removeOrganizationMember(userId); await refresh(); message.value = '成员已移出组织。' } catch (error) { message.value = text(error) } finally { busy.value = false } }

async function bindEmail() { busy.value = true; message.value = ''; try { emailStatus.value = await backend.bindMyEmail({ email: emailInput.value.trim(), password: emailPassword.value }); emailPassword.value = ''; message.value = emailStatus.value.smtpConfigured ? '邮箱已绑定，下一步发送验证码完成验证。' : '邮箱已绑定，但管理员尚未配置 SMTP，因此暂时不能发送验证码。AGMP 其他功能不受影响。' } catch (error) { message.value = text(error) } finally { busy.value = false } }
async function requestEmailCode(purpose: 'bind' | 'risk') { busy.value = true; try { await backend.requestMyEmailVerification({ purpose }); message.value = '验证码已发送到当前绑定邮箱。' } catch (error) { message.value = text(error) } finally { busy.value = false } }
async function confirmEmailCode() { busy.value = true; try { emailStatus.value = await backend.confirmMyEmailVerification({ code: emailCode.value.trim() }); emailCode.value = ''; message.value = '邮箱验证已完成。' } catch (error) { message.value = text(error) } finally { busy.value = false } }
async function unbindEmail() { if (!emailPassword.value) { message.value = '解绑邮箱前请输入当前账号密码。'; return }; busy.value = true; try { emailStatus.value = await backend.unbindMyEmail(emailPassword.value); emailPassword.value = ''; emailInput.value = ''; message.value = '邮箱已解绑。密码找回和邮件二验将不可用，但 AGMP 与核心授权不受影响。' } catch (error) { message.value = text(error) } finally { busy.value = false } }
async function saveProfile() { if (!profileName.value.trim()) return; busy.value = true; try { current.value = await backend.updateMyDisplayName({ displayName: profileName.value.trim() }); await refresh(); message.value = '显示名称已更新。' } catch (error) { message.value = text(error) } finally { busy.value = false } }
async function rotateMySecurityKey() {
  if (!securityPassword.value) { message.value = '生成/轮换安全密钥前请输入当前账号密码。'; return }
  busy.value = true
  try { const material = await backend.rotateMySecurityKey({ password: securityPassword.value }); newSecurityKey.value = material.key; securityStatus.value = await backend.mySecurityStatus(); securityPassword.value = ''; message.value = '新安全密钥仅本次显示，请立即复制并离线保存。' }
  catch (error) { message.value = text(error) } finally { busy.value = false }
}
async function setMySecurityKeyVerification(enabled: boolean) {
  if (!securityPassword.value) { message.value = '切换登录安全密钥验证前请输入当前账号密码。'; return }
  busy.value = true
  try { securityStatus.value = await backend.setMySecurityKeyVerification({ password: securityPassword.value, enabled }); securityPassword.value = ''; message.value = enabled ? '登录安全密钥验证已开启。' : '登录安全密钥验证已关闭。' }
  catch (error) { message.value = text(error) } finally { busy.value = false }
}
async function logout() { await backend.logout(); window.dispatchEvent(new Event('agmp-auth-expired')) }

onMounted(() => void refresh())
</script>

<template>
  <section class="page page--wide">
    <div class="page-heading"><div><span class="eyebrow">ORGANIZATION & ACCESS</span><h1>用户与权限</h1><p>公开注册永久关闭；组织成员、核心授权、邮箱安全和官方 License 是互相独立的安全层。</p></div><button class="btn btn--secondary" @click="logout">退出登录</button></div>

    <div class="feature-grid feature-grid--two">
      <article class="panel feature-card"><span class="feature-title">当前组织</span><strong>{{ organization?.name || '读取中…' }}</strong><p class="feature-description">{{ organization?.id || '—' }} · 默认组 {{ organization?.defaultGroupId || 'management' }}</p></article>
      <article class="panel feature-card"><span class="feature-title">我的核心权限</span><strong>{{ coreLabel(current?.coreAccess || 'pending') }}</strong><p class="feature-description">加入组织不等于获得核心功能；非 Owner 成员需要 Owner 的一次性核心授权码，并继续受官方 License 控制。</p></article>
    </div>

    <article class="panel user-security-card auth-form">
      <div class="card-heading"><div><span class="eyebrow">STEP-UP</span><h3>高风险操作身份确认</h3></div><span class="status-pill">{{ riskLabel(current?.riskState || 'normal') }}</span></div>
      <div class="auth-notice">签发邀请、核心授权、移除成员等敏感操作会要求短时二次确认。未配置邮箱时使用当前密码；账号已有安全密钥时还必须提供密钥。已配置 SMTP + 已验证邮箱的账号也可以使用邮件验证。</div>
      <div class="user-form-grid"><label>当前密码<input v-model="stepPassword" type="password" autocomplete="current-password" /></label><label>安全密钥（如已配置）<input v-model="stepSecurityKey" type="password" autocomplete="off" /></label></div>
      <button class="btn btn--secondary" type="button" :disabled="busy || !stepPassword" @click="confirmStepUp">完成当前会话二次确认</button>
    </article>

    <article v-if="canInvite" class="panel user-security-card auth-form">
      <div class="card-heading"><div><span class="eyebrow">SIGNED INVITATION</span><h3>邀请新成员</h3></div><span class="status-pill">一次性</span></div>
      <div class="auth-notice">管理员只签发当前实例的邀请，成员自行设置密码。用户名和邮箱都可选择绑定到邀请；邮箱本身仍不是 AGMP 注册必填项。</div>
      <div class="user-form-grid"><label>绑定用户名（可选）<input v-model.trim="inviteTargetUsername" /></label><label>绑定邮箱（可选）<input v-model.trim="inviteTargetEmail" type="email" /></label><label>角色<select v-model="inviteRole"><option value="operator">Operator</option><option v-if="isOwner" value="administrator">Administrator</option></select></label><label>有效期<select v-model.number="inviteHours"><option :value="1">1 小时</option><option :value="24">24 小时</option><option :value="72">3 天</option><option :value="168">7 天</option></select></label></div>
      <button class="btn btn--primary" type="button" :disabled="busy" @click="createInvitation">签发成员邀请</button>
      <div v-if="createdInvite" class="security-key-reveal"><strong>新邀请 · 完整令牌仅本次显示</strong><p>实例指纹：{{ createdInvite.instanceFingerprint }} · 校验码：{{ createdInvite.verificationCode }}</p><input :value="invitationLink || createdInvite.token" readonly /><div class="auth-actions"><button v-if="invitationLink" class="btn btn--secondary" @click="copy(invitationLink, $event)">复制注册链接</button><button class="btn btn--secondary" @click="copy(createdInvite.token, $event)">复制完整令牌</button></div></div>
      <div v-for="item in invitations" :key="item.id" class="user-row"><div><strong>{{ item.targetUsername || '未绑定用户名' }} · {{ item.role }}</strong><small>{{ item.targetEmail || '未绑定邮箱' }} · {{ item.verificationCode }} · {{ item.status }} · {{ time(item.expiresAt) }}</small></div><button v-if="item.status === 'active'" class="btn btn--secondary" @click="revokeInvitation(item.id)">撤销</button></div>
    </article>

    <article v-if="isOwner" class="panel user-security-card auth-form">
      <div class="card-heading"><div><span class="eyebrow">CORE AUTHORIZATION</span><h3>成员核心功能授权</h3></div></div>
      <div class="auth-notice auth-notice--important">成员核心授权码和组织邀请是两件事。只有 Owner 可以签发核心授权；授权码不会替代 AGMP 官方 License，也不会跨实例/组织生效。</div>
      <div class="user-form-grid"><label>成员<select v-model="selectedMemberId"><option v-for="item in selectableMembers" :key="item.id" :value="item.id">{{ item.displayName || item.username }} · {{ coreLabel(item.coreAccess) }}</option></select></label><label>授权码有效期<select v-model.number="grantHours"><option :value="1">1 小时</option><option :value="24">24 小时</option><option :value="72">3 天</option><option :value="168">7 天</option></select></label></div>
      <button class="btn btn--primary" type="button" :disabled="busy || !selectedMemberId" @click="createCoreGrant">签发成员核心授权码</button>
      <div v-if="createdGrant" class="security-key-reveal"><strong>核心授权码 · 仅本次显示</strong><p>实例指纹：{{ createdGrant.instanceFingerprint }} · 校验码：{{ createdGrant.verificationCode }}</p><input :value="createdGrant.token" readonly /><button class="btn btn--secondary" @click="copy(createdGrant.token, $event)">复制授权码</button></div>
      <div v-for="item in coreGrants" :key="item.id" class="user-row"><div><strong>{{ item.username || item.userId }}</strong><small>{{ item.verificationCode }} · {{ item.status }} · {{ time(item.expiresAt) }}</small></div><button v-if="item.status === 'active'" class="btn btn--secondary" @click="backend.revokeMemberCoreAuthorization(item.id).then(refresh).catch(error => message = text(error))">撤销未使用授权码</button></div>
    </article>

    <article v-if="current?.role !== 'owner' && current?.coreAccess !== 'authorized'" class="panel user-security-card auth-form">
      <div class="card-heading"><div><span class="eyebrow">CORE ACCESS</span><h3>兑换核心授权码</h3></div></div>
      <label>Owner 提供的核心授权码<input v-model.trim="redeemCode" autocomplete="off" spellcheck="false" placeholder="AGMPU1.…" /></label><button class="btn btn--primary" type="button" :disabled="busy || !redeemCode" @click="redeemCoreGrant">兑换并启用核心权限</button>
    </article>

    <article class="panel user-security-card">
      <div class="card-heading"><div><span class="eyebrow">MEMBERS</span><h3>组织成员</h3></div><span class="version">{{ users.length }} 个</span></div>
      <div v-for="item in users" :key="item.id" class="user-row"><div><strong>{{ item.displayName || item.username }}</strong><small>@{{ item.username }} · {{ item.role }} · {{ coreLabel(item.coreAccess) }} · 风控 {{ riskLabel(item.riskState) }}</small></div><div v-if="isOwner && item.role !== 'owner'" class="auth-actions"><button v-if="item.coreAccess === 'authorized'" class="btn btn--secondary" @click="revokeCore(item.id)">暂停核心</button><button v-if="item.riskState !== 'normal'" class="btn btn--secondary" @click="clearRisk(item.id)">解除风控</button><button class="btn btn--secondary" @click="removeMember(item.id)">移出组织</button></div></div>
    </article>

    <article class="panel user-security-card auth-form">
      <div class="card-heading"><div><span class="eyebrow">OPTIONAL EMAIL</span><h3>我的邮箱与账号安全</h3></div><span class="status-pill">{{ emailStatus?.verified ? '邮箱已验证' : emailStatus?.bound ? '等待验证' : '未绑定' }}</span></div>
      <div class="auth-notice">邮箱是可选增强：用于密码找回、风险通知和邮件二次验证，不参与组织成员资格和核心授权。邮件功能还要求管理员先在“设置 → 邮箱服务”配置 SMTP。</div>
      <div v-if="!emailStatus?.smtpConfigured" class="auth-notice auth-notice--important">当前实例尚未配置邮箱服务。你仍可正常使用 AGMP；邮箱验证码/找回密码暂不可用。</div>
      <template v-if="!emailStatus?.bound"><div class="user-form-grid"><label>邮箱（可选）<input v-model.trim="emailInput" type="email" /></label><label>当前账号密码<input v-model="emailPassword" type="password" /></label></div><button class="btn btn--secondary" :disabled="busy || !emailInput || !emailPassword" @click="bindEmail">绑定邮箱</button></template>
      <template v-else><p>当前邮箱：<strong>{{ emailStatus.email }}</strong></p><div v-if="!emailStatus.verified" class="auth-actions"><button class="btn btn--secondary" :disabled="busy || !emailStatus.smtpConfigured" @click="requestEmailCode('bind')">发送验证验证码</button><input v-model.trim="emailCode" inputmode="numeric" maxlength="8" placeholder="8 位验证码" /><button class="btn btn--secondary" :disabled="busy || emailCode.length !== 8" @click="confirmEmailCode">确认验证</button></div><div class="user-form-grid"><label>解绑时输入当前密码<input v-model="emailPassword" type="password" /></label></div><button class="btn btn--secondary" :disabled="busy || !emailPassword" @click="unbindEmail">解绑邮箱</button></template>
    </article>

    <article class="panel user-security-card auth-form">
      <div class="card-heading"><div><span class="eyebrow">SECURITY KEY</span><h3>登录安全密钥</h3></div><span class="status-pill">{{ securityStatus?.verificationEnabled ? '登录验证已开启' : securityStatus?.configured ? '已配置' : '未配置' }}</span></div>
      <div class="auth-notice">安全密钥是可选增强。配置后可用于登录保护和高风险凭据二验；AGMP 后端只保存校验值，不保存明文。</div>
      <label>当前账号密码<input v-model="securityPassword" type="password" autocomplete="current-password" /></label>
      <div class="auth-actions"><button class="btn btn--secondary" :disabled="busy || !securityPassword" @click="rotateMySecurityKey">{{ securityStatus?.configured ? '轮换安全密钥' : '生成安全密钥' }}</button><button v-if="securityStatus?.configured && !securityStatus?.verificationEnabled" class="btn btn--secondary" :disabled="busy || !securityPassword" @click="setMySecurityKeyVerification(true)">开启登录密钥验证</button><button v-if="securityStatus?.verificationEnabled" class="btn btn--secondary" :disabled="busy || !securityPassword" @click="setMySecurityKeyVerification(false)">关闭登录密钥验证</button></div>
      <div v-if="newSecurityKey" class="security-key-reveal"><strong>新密钥 · 仅本次显示</strong><input :value="newSecurityKey" readonly /><button class="btn btn--secondary" @click="copy(newSecurityKey, $event)">复制密钥</button></div>
    </article>

    <article class="panel user-security-card auth-form"><div class="card-heading"><div><span class="eyebrow">PROFILE</span><h3>个人资料</h3></div></div><label>显示名称<input v-model.trim="profileName" maxlength="40" /></label><button class="btn btn--secondary" :disabled="busy || !profileName" @click="saveProfile">保存显示名称</button></article>

    <div v-if="message" class="auth-notice auth-notice--important">{{ message }}</div>
    <div v-if="loading" class="auth-notice">正在读取组织安全状态…</div>
  </section>
</template>

<style scoped>
.user-form-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px}.user-row{display:flex;align-items:center;justify-content:space-between;gap:14px;padding:12px 0;border-bottom:1px solid var(--border)}.user-row>div:first-child{display:grid;gap:4px}.user-row small{color:var(--text-secondary)}.user-security-card{display:grid;gap:14px;margin-top:16px}.security-key-reveal{display:grid;gap:9px}.security-key-reveal input{width:100%}@media(max-width:760px){.user-form-grid{grid-template-columns:1fr}.user-row{align-items:flex-start;flex-direction:column}}
</style>
