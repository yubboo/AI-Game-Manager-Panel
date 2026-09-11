<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import AppIcon from '../../shared/components/AppIcon.vue'
import { backend } from '../../shared/api/backend'
import { useAppStore } from '../../shared/store/app'

const app = useAppStore()
const license = computed(() => app.license)
const certificate = ref('')
const certificateFileName = ref('')
const busy = ref(false)
const message = ref('')

const stateLabel = computed(() => {
  const state = license.value?.state || 'UNLICENSED'
  const map: Record<string, string> = {
    UNLICENSED: '未激活', ACTIVE: '已激活', EXPIRING: '即将到期', EXPIRED: '已过期',
    DEVICE_MISMATCH: '设备不匹配', REVOKED: '已吊销', SEAT_LIMIT: '设备席位已满',
    OFFLINE_GRACE: '离线宽限期', SERVER_UNREACHABLE: '许可证服务器不可达', DEVELOPMENT: '源码开发模式',
  }
  return map[state] || state
})

const stateTone = computed(() => license.value?.valid ? 'is-valid' : (license.value?.activated ? 'is-warning' : 'is-muted'))
const expiresText = computed(() => !license.value?.expiresAt ? '永久' : new Date(license.value.expiresAt * 1000).toLocaleDateString('zh-CN'))

async function refresh() {
  message.value = ''
  const ok = await app.loadLicense()
  if (!ok) message.value = '许可证状态刷新失败，请查看日志中心中的 Core 错误。'
}

async function activate() {
  busy.value = true
  message.value = ''
  try {
    app.license = await backend.activateLicense({ certificate: certificate.value.trim() })
    certificate.value = ''
    certificateFileName.value = ''
    message.value = '许可证已激活并持久化绑定当前机器码与安装 ID；完全退出并重启后仍应保持有效。'
  } catch (error) {
    message.value = error instanceof Error ? error.message : String(error)
  } finally {
    busy.value = false
  }
}

async function importCertificateFile(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  message.value = ''
  if (file.size > 256 * 1024) {
    message.value = 'BFLC2 文件异常过大，已拒绝读取。'
    input.value = ''
    return
  }
  try {
    const value = (await file.text()).trim()
    if (!value.startsWith('BFLC2.')) throw new Error('所选文件不是 BFLC2 离线许可证证书。')
    certificate.value = value
    certificateFileName.value = file.name
    message.value = `已读取 ${file.name}，点击“导入 BFLC2 并激活”完成本机验签与持久化。`
  } catch (error) {
    certificate.value = ''
    certificateFileName.value = ''
    message.value = error instanceof Error ? error.message : String(error)
  } finally {
    input.value = ''
  }
}

async function unbind() {
  if (!window.confirm('确定解绑当前设备的本地许可证吗？这不会删除账号，也不会清除服务器数据。')) return
  busy.value = true
  try {
    app.license = await backend.unbindLicense()
    message.value = '当前设备本地许可证已解绑。'
  } catch (error) {
    message.value = error instanceof Error ? error.message : String(error)
  } finally {
    busy.value = false
  }
}

async function copyValue(value: string, label: string) {
  if (!value) return
  try { await navigator.clipboard.writeText(value); message.value = `${label}已复制。` }
  catch { message.value = `${label}复制失败，请手动选择复制。` }
}

onMounted(() => { if (!app.license) void refresh() })
</script>

<template>
  <section class="settings-section settings-section--license">
    <div class="settings-section__intro">
      <div class="settings-section__icon"><AppIcon name="settings" /></div>
      <div><strong>授权与许可证</strong><span>账号负责身份认证；许可证负责当前设备可以使用哪些高级功能。未激活不会阻断登录、设置和基础诊断。</span></div>
    </div>

    <div class="license-overview-grid">
      <article class="feature-card">
        <span class="feature-title">授权状态</span>
        <p class="feature-description">当前 AI Game Manager Panel Core 的许可证状态，Wails、Electron 与本机 Web 共用同一份结果。</p>
        <strong :class="['license-state', stateTone]">{{ stateLabel }}</strong>
      </article>
      <article class="feature-card">
        <span class="feature-title">许可证版本</span>
        <p class="feature-description">许可证决定 Edition 与 Feature Entitlement，不在 Vue 页面里写死版本权限。</p>
        <strong>{{ license?.edition || '—' }}</strong>
      </article>
      <article class="feature-card">
        <span class="feature-title">设备使用量</span>
        <p class="feature-description">本地证书记录当前激活席位；未来云端 License Server 将提供远程设备列表与解绑。</p>
        <strong>{{ license?.activeSeats || 0 }} / {{ license?.seatLimit || 1 }}</strong>
      </article>
      <article class="feature-card">
        <span class="feature-title">有效期</span>
        <p class="feature-description">永久许可证显示“永久”；有期限许可证到期后高级 Feature Gate 会停止放行。</p>
        <strong>{{ expiresText }}</strong>
      </article>
    </div>

    <div class="settings-subsection">
      <div class="settings-subsection__heading"><strong>设备与唯一标识</strong><span>机器码识别“哪台电脑”，安装 ID 识别“这台电脑上的哪次 AI Game Manager Panel 安装”。0.1.68 起这两个值在 Core 生命周期内缓存，页面切换或刷新状态不会重新生成。</span></div>
      <div class="feature-grid feature-grid--two">
        <article class="feature-card feature-card--value">
          <span class="feature-title">机器码</span>
          <p class="feature-description">由稳定系统标识经过 AI Game Manager Panel 专用 SHA-256 派生，不展示原始 MachineGuid。</p>
          <div class="feature-value-row"><code>{{ license?.machineCode || '读取中…' }}</code><button class="btn btn--secondary" @click="copyValue(license?.machineCode || '', '机器码')">复制</button></div>
        </article>
        <article class="feature-card feature-card--value">
          <span class="feature-title">安装 ID</span>
          <p class="feature-description">AI Game Manager Panel 首次运行生成的 BFID，用于区分同一设备上的不同安装实例。签发许可证时必须复制“准备激活的这一份程序”显示的 BFID；源码开发目录与正式安装版可能拥有不同 BFID。</p>
          <div class="feature-value-row"><code>{{ license?.installId || license?.identificationCode || '读取中…' }}</code><button class="btn btn--secondary" @click="copyValue(license?.installId || license?.identificationCode || '', '安装 ID')">复制</button></div>
        </article>
      </div>
    </div>

    <div class="settings-subsection">
      <div class="settings-subsection__heading"><strong>发行公钥与证书签名</strong><span>这里只显示公开 KeyID / 指纹，不包含发行私钥。0.1.70 起客户端保留可信公钥环，轮换新密钥时历史公钥仍可验证旧 BFLC2。</span></div>
      <div class="feature-grid feature-grid--two">
        <article class="feature-card feature-card--value">
          <span class="feature-title">当前 Active 发行 KeyID</span>
          <p class="feature-description">正式 Release 必须配置 Active 发行公钥；源码开发模式允许暂未初始化。</p>
          <div class="feature-value-row"><code>{{ license?.vendorKeyId || '尚未配置发行密钥' }}</code><button class="btn btn--secondary" :disabled="!license?.vendorKeyId" @click="copyValue(license?.vendorKeyId || '', '发行 KeyID')">复制</button></div>
        </article>
        <article class="feature-card feature-card--value">
          <span class="feature-title">发行公钥指纹</span>
          <p class="feature-description">用于核对当前客户端信任的 Active Ed25519 公钥；可信公钥总数：{{ license?.trustedKeyCount || 0 }}。</p>
          <div class="feature-value-row"><code>{{ license?.vendorKeyFingerprint || '等待项目所有者本机初始化' }}</code><button class="btn btn--secondary" :disabled="!license?.vendorKeyFingerprint" @click="copyValue(license?.vendorKeyFingerprint || '', '发行公钥指纹')">复制</button></div>
        </article>
      </div>
      <div v-if="license?.issuerKeyId" class="feature-card"><span class="feature-title">当前许可证签发 KeyID</span><p class="feature-description">当前已导入 BFLC2 实际由此 KeyID 签发；它可以是 Active 或历史可信发行公钥。</p><code>{{ license.issuerKeyId }}</code></div>
    </div>

    <div v-if="license?.state !== 'DEVELOPMENT'" class="settings-subsection">
      <div class="settings-subsection__heading"><strong>激活 AI Game Manager Panel</strong><span>当前版本的离线授权是 BFLC2 许可证证书，不是短 CDK。短 CDK 属于未来在线 License Server 的兑换码，目前尚未启用。</span></div>
      <div class="license-activation-grid">
        <label class="field-block"><span class="feature-title">在线激活码（CDK）</span><small>暂未开放。License Server 上线后才会启用短 CDK 在线兑换。</small><input value="在线 License Server 尚未上线" disabled autocomplete="off" spellcheck="false" /></label>
        <label class="field-block"><span class="feature-title">离线许可证证书（BFLC2）</span><small>可以直接选择发行工具生成的 .bflc 文件，也可以粘贴完整 BFLC2。客户端只读取证书文本，不读取任何发行私钥。</small><input type="file" accept=".bflc,text/plain" @change="importCertificateFile" /><small v-if="certificateFileName">已选择：{{ certificateFileName }}</small><textarea v-model.trim="certificate" rows="4" autocomplete="off" spellcheck="false" placeholder="BFLC2.&lt;payload&gt;.&lt;signature&gt;"></textarea></label>
      </div>
      <div class="settings-actions-row">
        <span class="save-message" :class="{ active: message }">{{ message || license?.message || '未授权不会阻断账号登录。' }}</span>
        <div class="inline-actions">
          <button class="btn btn--secondary" type="button" :disabled="busy" @click="refresh">刷新状态</button>
          <button v-if="license?.activated" class="btn btn--secondary" type="button" :disabled="busy" @click="unbind">解绑当前设备</button>
          <button class="btn btn--primary" type="button" :disabled="busy || !certificate" @click="activate">{{ busy ? '处理中…' : '导入 BFLC2 并激活' }}</button>
        </div>
      </div>
    </div>

    <div v-if="license?.state === 'DEVELOPMENT'" class="settings-subsection">
      <div class="settings-subsection__heading"><strong>源码开发许可证模式</strong><span>当前进程使用 agmp_dev_license Build Tag，仅用于本地源码开发；正式 Release 不包含此模式，也不需要把万能 CDK 或发行私钥放进仓库。</span></div>
      <div class="feature-card"><span class="feature-title">开发权限</span><p class="feature-description">开发构建暂时放行全部 Feature Entitlement，方便调试 AI、计划任务、插件和远程节点。正式构建会恢复许可证校验。</p><strong>Development · 全功能调试</strong></div>
    </div>

    <div class="settings-subsection">
      <div class="settings-subsection__heading"><strong>授权关系说明</strong><span>身份认证和许可证相互独立：账号证明你是谁，许可证决定高级功能权限。</span></div>
      <div class="feature-grid feature-grid--two">
        <article class="feature-card"><span class="feature-title">LicenseID / ActivationID</span><p class="feature-description">LicenseID 是许可证唯一编号；ActivationID 是当前许可证与设备/安装实例的一次唯一绑定记录。</p><code>{{ license?.licenseId || '尚未签发' }} · {{ license?.activationId || '尚未绑定' }}</code></article>
        <article class="feature-card"><span class="feature-title">本地与云端</span><p class="feature-description">当前版本完成本地 Ed25519 离线证书和本地解绑。在线 CDK、远程设备列表、席位与远程解绑属于 License Server 阶段，未上线时不会伪装可用。</p><strong>本地证书可用 · 云端接口预留</strong></article>
      </div>
    </div>

    <div v-if="license?.valid" class="settings-subsection">
      <div class="settings-subsection__heading"><strong>已授权功能</strong><span>高级功能统一由 Go Core 的 Feature Entitlement 判定。</span></div>
      <div class="entitlement-list"><code v-for="feature in license.features" :key="feature">{{ feature }}</code></div>
      <div class="license-id-grid"><span>LicenseID <code>{{ license.licenseId || '—' }}</code></span><span>ActivationID <code>{{ license.activationId || '—' }}</code></span></div>
    </div>
  </section>
</template>
