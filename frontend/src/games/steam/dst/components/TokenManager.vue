<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import StatusPill from '../../../../shared/components/StatusPill.vue'
import { backend } from '../../../../shared/api/backend'
import type {
  DSTCluster,
  DSTTokenStatus,
  DSTKleiPackageImportMode,
  DSTKleiPackagePreview,
} from '../../../../shared/types/backend'

const props = defineProps<{
  clusters: DSTCluster[]
  clusterPath: string
  runtimeActive?: boolean
}>()

const emit = defineEmits<{ select: [path: string]; changed: []; refresh: [] }>()

const status = ref<DSTTokenStatus | null>(null)
const loading = ref(false)
const message = ref('')
const secret = ref('')
const showSecret = ref(false)
const filePath = ref('')
const packagePreview = ref<DSTKleiPackagePreview | null>(null)
const packageName = ref('')
const packageBase64 = ref('')
const packageMode = ref<DSTKleiPackageImportMode>('token_only')

const selected = computed(() => props.clusters.find(cluster => cluster.path === props.clusterPath) ?? null)
const statusTone = computed(() => status.value?.configured ? 'success' as const : status.value?.state === 'empty' ? 'danger' as const : 'warning' as const)
const statusLabel = computed(() => status.value?.configured ? '已配置' : status.value?.state === 'empty' ? '文件为空' : '未配置')

function selectCluster(event: Event) {
  const target = event.target as HTMLSelectElement
  clearPackageMemory()
  emit('select', target.value)
}

async function openKleiPage() {
  message.value = ''
  try {
    await backend.openDSTTokenPage()
  } catch (error) {
    message.value = error instanceof Error ? error.message : String(error)
  }
}

async function refreshStatus() {
  if (!props.clusterPath) {
    status.value = null
    return
  }
  loading.value = true
  message.value = ''
  try {
    status.value = await backend.dstTokenStatus(props.clusterPath)
  } catch (error) {
    message.value = error instanceof Error ? error.message : String(error)
  } finally {
    loading.value = false
  }
}

watch(() => props.clusterPath, () => void refreshStatus(), { immediate: true })
onUnmounted(clearPackageMemory)

async function saveValue(value: string) {
  const token = value.trim()
  if (!props.clusterPath || !token) return
  loading.value = true
  message.value = ''
  try {
    status.value = await backend.saveDSTToken({ clusterPath: props.clusterPath, value: token })
    message.value = '令牌已安全保存到当前 Cluster。AGMP 不会在页面、日志、localStorage 或诊断包中回显令牌正文。'
    emit('changed')
    emit('refresh')
  } catch (error) {
    message.value = error instanceof Error ? error.message : String(error)
  } finally {
    secret.value = ''
    loading.value = false
  }
}

async function saveManual() {
  await saveValue(secret.value)
}

async function uploadFile(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  clearPackageMemory()
  message.value = ''
  try {
    if (file.name.toLowerCase().endsWith('.zip')) {
      await inspectPackage(file)
      return
    }
    if (file.size > 16 * 1024) throw new Error('令牌文件异常过大，请确认选择的是 cluster_token.txt。')
    const text = await file.text()
    await saveValue(text)
  } catch (error) {
    message.value = error instanceof Error ? error.message : String(error)
  } finally {
    input.value = ''
  }
}

async function inspectPackage(file: File) {
  if (!props.clusterPath) throw new Error('请先选择要应用令牌的服务器世界。')
  if (file.size > 8 * 1024 * 1024) throw new Error('Klei 配置包超过 8 MiB 安全上限，请确认选择的是官方“下载设置”ZIP。')
  loading.value = true
  try {
    const base64 = await fileToBase64(file)
    const preview = await backend.inspectDSTKleiPackage({ archiveName: file.name, archiveBase64: base64 })
    packageName.value = file.name
    packageBase64.value = base64
    packagePreview.value = preview
    packageMode.value = 'token_only'
    message.value = '已安全识别 Klei 官方配置包。默认只导入服务器令牌，不覆盖当前世界配置。'
  } finally {
    loading.value = false
  }
}

async function applyPackage() {
  if (!props.clusterPath || !packagePreview.value || !packageBase64.value) return
  loading.value = true
  message.value = ''
  try {
    const result = await backend.importDSTKleiPackage({
      clusterPath: props.clusterPath,
      archiveName: packageName.value,
      archiveBase64: packageBase64.value,
      mode: packageMode.value,
    })
    await refreshStatus()
    const details = result.mode === 'full_config'
      ? `已导入令牌和 ${result.configFilesCopied} 个官方配置文件${result.backupPath ? `；旧配置已备份到 ${result.backupPath}` : ''}。`
      : '已仅导入服务器令牌；当前 cluster.ini、Master/Caves 配置和存档均未修改。'
    message.value = `${details}${result.warnings.length ? ` ${result.warnings.join('；')}` : ''}`
    clearPackageMemory()
    emit('changed')
    emit('refresh')
  } catch (error) {
    message.value = error instanceof Error ? error.message : String(error)
  } finally {
    loading.value = false
  }
}

function clearPackageMemory() {
  packagePreview.value = null
  packageName.value = ''
  packageBase64.value = ''
  packageMode.value = 'token_only'
}

async function fileToBase64(file: File) {
  const buffer = new Uint8Array(await file.arrayBuffer())
  const chunks: string[] = []
  const step = 0x8000
  for (let offset = 0; offset < buffer.length; offset += step) {
    chunks.push(String.fromCharCode(...buffer.subarray(offset, Math.min(offset + step, buffer.length))))
  }
  return btoa(chunks.join(''))
}

async function importPath() {
  if (!props.clusterPath || !filePath.value.trim()) return
  loading.value = true
  message.value = ''
  try {
    status.value = await backend.importDSTToken({ clusterPath: props.clusterPath, sourcePath: filePath.value.trim() })
    filePath.value = ''
    message.value = '已从本机文件路径导入令牌，源文件未被修改。'
    emit('changed')
    emit('refresh')
  } catch (error) {
    message.value = error instanceof Error ? error.message : String(error)
  } finally {
    loading.value = false
  }
}

function formatTime(timestamp?: number) {
  if (!timestamp) return '—'
  return new Date(timestamp * 1000).toLocaleString('zh-CN')
}

function formatBytes(bytes: number) {
  if (!bytes) return '0 B'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KiB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MiB`
}
</script>

<template>
  <article class="panel workspace-panel workspace-panel--wide dst-token-manager">
    <div class="workspace-panel__heading">
      <div><span class="eyebrow">KLEI SERVER TOKEN</span><h3>服务器令牌</h3><p>Dedicated Server 必须使用 Klei 服务器令牌完成注册。AGMP 只负责安全写入当前 Cluster，不接管 Steam/Klei 登录凭据。</p></div>
      <StatusPill :tone="statusTone" :label="loading ? '处理中' : statusLabel" />
    </div>

    <div v-if="clusters.length === 0" class="notice notice--warning">当前没有可配置令牌的 Steam SERVER Cluster。请先到“存档管理”导入一个世界。</div>
    <template v-else>
      <div class="dst-token-cluster-row">
        <label for="dst-token-cluster">应用到服务器世界</label>
        <select id="dst-token-cluster" :value="clusterPath" :disabled="runtimeActive" @change="selectCluster">
          <option v-for="cluster in clusters" :key="cluster.path" :value="cluster.path">{{ cluster.name }} · {{ cluster.path }}</option>
        </select>
      </div>

      <section class="dst-token-status-card">
        <div><span>状态</span><strong>{{ status?.message || '正在读取…' }}</strong></div>
        <div><span>保存位置</span><code>{{ status?.path || `${selected?.path ?? ''}\\cluster_token.txt` }}</code></div>
        <div><span>最近修改</span><strong>{{ formatTime(status?.modifiedAt) }}</strong></div>
        <div><span>隐私策略</span><strong>不回显正文 · 不持久化到前端 · 不进入诊断包</strong></div>
      </section>

      <div class="dst-token-methods">
        <section class="dst-token-method">
          <span class="eyebrow">METHOD 1</span><h4>从 Klei 官方获取</h4>
          <p>进入 Klei 官方服务器配置页完成登录和“下载设置”。官网通常会下载一个类似 MyDediServer.zip 的配置包。</p>
          <button class="btn btn--primary" type="button" @click="openKleiPage">打开 Klei 官方服务器页面</button>
        </section>

        <section class="dst-token-method dst-token-method--recommended">
          <span class="eyebrow">METHOD 2 · RECOMMENDED</span><h4>导入 Klei 官方 ZIP / Token</h4>
          <p>直接选择 Klei “下载设置”得到的 ZIP，AGMP 自动识别 cluster_token.txt、cluster.ini、Master/Caves 配置；也继续支持单独的 TXT Token。</p>
          <label class="btn btn--primary dst-token-upload" :class="{ disabled: loading || runtimeActive }">
            选择 Klei ZIP / Token<input type="file" accept=".zip,.txt,application/zip,text/plain" :disabled="loading || runtimeActive" @change="uploadFile" />
          </label>
        </section>

        <section class="dst-token-method">
          <span class="eyebrow">METHOD 3</span><h4>手动粘贴令牌</h4>
          <p>输入框默认遮挡内容，保存成功后立即清空。</p>
          <div class="dst-token-secret-row"><input v-model="secret" :type="showSecret ? 'text' : 'password'" autocomplete="off" spellcheck="false" placeholder="粘贴 Klei server token" :disabled="loading || runtimeActive" /><button class="btn btn--secondary btn--compact" type="button" @click="showSecret = !showSecret">{{ showSecret ? '隐藏' : '显示' }}</button></div>
          <button class="btn btn--primary" :disabled="loading || runtimeActive || !secret.trim()" @click="saveManual">保存并应用</button>
        </section>
      </div>

      <section v-if="packagePreview" class="dst-klei-package-preview">
        <div class="dst-klei-package-preview__head">
          <div><span class="eyebrow">KLEI PACKAGE DETECTED</span><h4>{{ packagePreview.archiveName }}</h4><p>ZIP 仅在本次导入期间保存在内存，完成或切换 Cluster 后立即释放；Token 正文不会进入预览。</p></div>
          <StatusPill tone="success" label="官方结构已识别" />
        </div>
        <div class="dst-klei-package-facts">
          <div><span>服务器名称</span><strong>{{ packagePreview.serverName || '未提供' }}</strong></div>
          <div><span>最大玩家</span><strong>{{ packagePreview.maxPlayers || '未提供' }}</strong></div>
          <div><span>游戏模式</span><strong>{{ packagePreview.gameMode || '未提供' }}</strong></div>
          <div><span>服务器密码</span><strong>{{ packagePreview.passworded ? '已配置（不显示）' : '未配置' }}</strong></div>
          <div><span>Token</span><strong>{{ packagePreview.hasToken ? '✓ 已检测' : '✕ 缺失' }}</strong></div>
          <div><span>cluster.ini</span><strong>{{ packagePreview.hasClusterIni ? '✓ 已检测' : '—' }}</strong></div>
          <div><span>Master</span><strong>{{ packagePreview.hasMasterServerIni ? '✓ server.ini' : '—' }}</strong></div>
          <div><span>Caves</span><strong>{{ packagePreview.hasCavesServerIni ? '✓ server.ini' : '未包含' }}</strong></div>
        </div>
        <div class="dst-klei-package-mode">
          <label :class="{ active: packageMode === 'token_only' }"><input v-model="packageMode" type="radio" value="token_only" /><span><strong>仅导入服务器令牌（推荐）</strong><small>不修改当前世界、cluster.ini、Master/Caves 配置，最适合已有存档。</small></span></label>
          <label :class="{ active: packageMode === 'full_config' }"><input v-model="packageMode" type="radio" value="full_config" :disabled="!packagePreview.hasClusterIni || !packagePreview.hasMasterServerIni" /><span><strong>导入完整 Klei 配置</strong><small>更新 cluster.ini 与官方 Master/Caves server.ini；覆盖前自动备份现有配置，不动世界存档；备份仅保存在本机 Cluster 内。</small></span></label>
        </div>
        <div class="dst-klei-package-actions"><small>{{ packagePreview.entryCount }} 个 ZIP 条目 · 解压约 {{ formatBytes(packagePreview.uncompressedBytes) }}</small><button class="btn btn--secondary" type="button" @click="clearPackageMemory">取消</button><button class="btn btn--primary" type="button" :disabled="loading || runtimeActive" @click="applyPackage">导入并应用</button></div>
      </section>

      <details class="dst-token-advanced">
        <summary>高级：从本机已有令牌文件路径导入</summary>
        <div class="dst-token-secret-row"><input v-model="filePath" type="text" placeholder="例如 D:\\备份\\cluster_token.txt" :disabled="loading || runtimeActive" /><button class="btn btn--secondary" :disabled="loading || runtimeActive || !filePath.trim()" @click="importPath">导入</button></div>
      </details>
      <div v-if="runtimeActive" class="notice notice--warning">服务器运行期间不允许替换当前 Cluster 的令牌或配置，请先正常停服。</div>
      <div v-if="message" class="notice" :class="{ 'notice--error': message.includes('失败') || message.includes('错误') || message.includes('无效') }">{{ message }}</div>
    </template>
  </article>
</template>
