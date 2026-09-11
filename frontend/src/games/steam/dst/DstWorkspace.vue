<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import StatusPill from '../../../shared/components/StatusPill.vue'
import { backend } from '../../../shared/api/backend'
import { useSteamMaintenance } from '../shared/useSteamMaintenance'
import TokenManager from './components/TokenManager.vue'
import ClusterImporter from './components/ClusterImporter.vue'
import PreflightPanel from './components/PreflightPanel.vue'
import NetworkConfig from './components/NetworkConfig.vue'
import { useDstPreflight } from './composables/useDstPreflight'
import { classifyDSTConsoleLine } from './console/format'
import type {
  DSTDedicatedSnapshot,
  DSTEnvironment,
  DSTProcessLogLine,
  DSTProcessSnapshot,
  DSTClusterRuntimeSnapshot,
  GameAppState,
  GameWorkspaceGameState,
  DSTClusterImportResult,
} from '../../../shared/types/backend'

const props = defineProps<{
  game: GameWorkspaceGameState | null
  dedicated: DSTDedicatedSnapshot | null
  loading: boolean
  error?: string
  scannedAt?: number
  durationMs?: number
  steamDetected: boolean
  steamInstallPath?: string
  warnings?: string[]
  environment?: DSTEnvironment | null
}>()

const emit = defineEmits<{ refresh: [] }>()
const router = useRouter()

type WorkspaceSection = 'overview' | 'logs' | 'rooms' | 'saves' | 'world' | 'network' | 'mods' | 'tokens' | 'players' | 'backups' | 'install'

const section = ref<WorkspaceSection>('overview')
const manualServerPath = ref('')
const extraArgs = ref('')
const preferenceMessage = ref('')
const preferenceSaving = ref(false)
const selectedClusterPath = ref('')
const { result: preflightResult, loading: preflightLoading, error: preflightError, refresh: refreshPreflight } = useDstPreflight(selectedClusterPath)
const activeConsoleShard = ref<'Master' | 'Caves'>('Master')
const masterRuntimeState = ref<DSTProcessSnapshot | null>(null)
const cavesRuntimeState = ref<DSTProcessSnapshot | null>(null)
const clusterRuntimeState = ref<DSTClusterRuntimeSnapshot | null>(null)
const runtimeSnapshotKey = ref('')
const runtimeState = computed(() => activeConsoleShard.value === 'Caves' ? cavesRuntimeState.value : masterRuntimeState.value)
const runtimeLoading = ref(false)
const portRepairLoading = ref(false)
const runtimeError = ref('')
const runtimeMessage = ref('')
const runtimeLogs = ref<DSTProcessLogLine[]>([])
const runtimeLogCursor = ref(0)
const runtimeLogDropped = ref(false)
const runtimeLogSessionId = ref('')
const consoleCommand = ref('')
const consoleViewport = ref<HTMLElement | null>(null)
const consoleAutoScroll = ref(true)
const consoleViewMode = ref<'structured' | 'raw'>('structured')
const consoleFilter = ref('important')
const consoleSearch = ref('')
let runtimeTimer: number | undefined


const sections: Array<{ id: WorkspaceSection; label: string; hint: string }> = [
  { id: 'overview', label: '概览与控制台', hint: '运行与实时日志' },
  { id: 'logs', label: '日志中心', hint: '历史 / 搜索 / 下载' },
  { id: 'rooms', label: '房间管理', hint: 'Cluster / Master / Caves' },
  { id: 'saves', label: '存档管理', hint: '导入 / 复制 / 恢复' },
  { id: 'world', label: '世界设置', hint: '地面与洞穴' },
  { id: 'network', label: '网络与端口', hint: '自动修复 / 高级' },
  { id: 'mods', label: 'Mod 管理', hint: 'Workshop / 配置' },
  { id: 'tokens', label: '令牌管理', hint: 'Token / 权限' },
  { id: 'players', label: '玩家管理', hint: '在线 / 黑白名单' },
  { id: 'backups', label: '备份', hint: '快照与恢复' },
  { id: 'install', label: '安装与更新', hint: 'Steam / 文件校验' },
]

watch(() => props.dedicated, (value) => {
  if (!value || preferenceSaving.value) return
  manualServerPath.value = value.manualPath ?? ''
  extraArgs.value = value.extraArgs ?? ''
}, { immediate: true })

function installed(state?: GameAppState) {
  return !!state?.installed && !!state.installPathExists
}

const gameAppId = computed(() => props.game?.gameApp?.appId ?? props.game?.catalog.steam?.gameAppId ?? 0)
const serverAppId = computed(() => props.game?.serverApp?.appId ?? props.game?.catalog.steam?.serverAppId ?? 0)
const gameInstalled = computed(() => installed(props.game?.gameApp))
const serverManifestInstalled = computed(() => installed(props.game?.serverApp))
const serverInstalled = computed(() => !!props.dedicated?.installation.valid)
const ready = computed(() => serverInstalled.value && !!props.dedicated?.confDir.valid)
const shardActive = (state: DSTProcessSnapshot | null) => !!state && ['starting', 'running', 'stopping'].includes(state.status)
const runtimeActive = computed(() => shardActive(masterRuntimeState.value) || shardActive(cavesRuntimeState.value))
const steamAvailable = computed(() => props.steamDetected && !!props.steamInstallPath)
const {
  task: maintenanceTask,
  error: maintenanceError,
  busy: maintenanceBusy,
  stateLabel: maintenanceStateLabel,
  title: maintenanceTitle,
  progressText: maintenanceProgressText,
  progressSourceLabel: maintenanceProgressSourceLabel,
  startInstall: installSteamApp,
  startValidation: validateSteamApp,
} = useSteamMaintenance({ onCompleted: () => emit('refresh') })

const serverClusters = computed(() => (props.environment?.clusters ?? []).filter((cluster) =>
  cluster.distribution === 'steam'
    && cluster.source === 'server'
    && cluster.shards.some((shard) => shard.name.toLowerCase() === 'master'),
))

const selectedCluster = computed(() => serverClusters.value.find((cluster) => cluster.path === selectedClusterPath.value) ?? null)
const tokenPreflight = computed(() => preflightResult.value?.checks.find((check) => check.code === 'token') ?? null)
const firstRunFileReady = computed(() => gameInstalled.value && serverInstalled.value)
const firstRunClusterReady = computed(() => serverClusters.value.length > 0)
const firstRunTokenReady = computed(() => tokenPreflight.value?.severity === 'ok')
const portPreflightBlocked = computed(() => (preflightResult.value?.checks ?? []).some((check) =>
  ['ports_explicit', 'ports_config'].includes(check.code) && check.severity === 'blocker',
))

watch(serverClusters, (clusters) => {
  if (clusters.length === 0) {
    selectedClusterPath.value = ''
    masterRuntimeState.value = null
    cavesRuntimeState.value = null
    clusterRuntimeState.value = null
    runtimeSnapshotKey.value = ''
    runtimeLogs.value = []
    runtimeLogCursor.value = 0
    runtimeLogSessionId.value = ''
    return
  }
  if (!clusters.some((cluster) => cluster.path === selectedClusterPath.value)) {
    selectedClusterPath.value = clusters[0]!.path
    return
  }
}, { immediate: true })

watch(selectedClusterPath, () => {
  masterRuntimeState.value = null
  cavesRuntimeState.value = null
  clusterRuntimeState.value = null
  runtimeSnapshotKey.value = ''
  runtimeLogs.value = []
  runtimeLogCursor.value = 0
  runtimeLogSessionId.value = ''
  runtimeLogDropped.value = false
  runtimeError.value = ''
  if (selectedClusterPath.value) void refreshRuntime(true)
})

watch(() => runtimeLogs.value.length, async () => {
  if (!consoleAutoScroll.value) return
  await nextTick()
  const element = consoleViewport.value
  if (element) element.scrollTop = element.scrollHeight
})

const runtimeStatusLabel = computed(() => {
  const state = runtimeState.value
  if (!state) return '未启动'
  if (state.status === 'starting') return '正在启动'
  if (state.status === 'stopping') return '正在停止'
  if (state.status === 'crashed') return '异常退出'
  if (state.status === 'stopped') return '已停止'
  if (state.worldReady) return '世界已就绪'
  return '世界加载中'
})

const runtimeTone = computed(() => {
  const state = runtimeState.value
  if (!state || state.status === 'stopped') return 'muted' as const
  if (state.status === 'crashed') return 'danger' as const
  if (state.status === 'stopping' || state.status === 'starting' || !state.worldReady) return 'warning' as const
  return 'success' as const
})

function shardStatusLabel(state: DSTProcessSnapshot | null) {
  if (!state) return '未启动'
  if (state.status === 'starting') return '正在启动'
  if (state.status === 'stopping') return '正在停止'
  if (state.status === 'crashed') return '异常退出'
  if (state.status === 'stopped') return '已停止'
  return state.worldReady ? '世界已就绪' : '世界加载中'
}

function shardTone(state: DSTProcessSnapshot | null) {
  if (!state || state.status === 'stopped') return 'muted' as const
  if (state.status === 'crashed') return 'danger' as const
  if (state.status === 'stopping' || state.status === 'starting' || !state.worldReady) return 'warning' as const
  return 'success' as const
}

const clusterStatusLabel = computed(() => {
  switch (clusterRuntimeState.value?.overall) {
    case 'running': return selectedHasCaves.value ? '地面 + 洞穴运行中' : '地面运行中'
    case 'starting': return '服务器启动中'
    case 'partial': return '部分 Shard 运行'
    case 'stopping': return '服务器停止中'
    case 'crashed': return '服务器异常'
    default: return '服务器已停止'
  }
})

const clusterTone = computed(() => {
  switch (clusterRuntimeState.value?.overall) {
    case 'running': return 'success' as const
    case 'crashed': return 'danger' as const
    case 'starting':
    case 'partial':
    case 'stopping': return 'warning' as const
    default: return 'muted' as const
  }
})

const selectedHasCaves = computed(() => !!selectedCluster.value?.shards.some((shard) => shard.name.toLowerCase() === 'caves'))
const portCleanupAllowed = computed(() => !runtimeActive.value || clusterRuntimeState.value?.overall === 'stopping')
const portBlocked = computed(() => portCleanupAllowed.value && !!clusterRuntimeState.value?.ports && !clusterRuntimeState.value.ports.ready)
const canStartCluster = computed(() => ready.value && !!selectedCluster.value && !!preflightResult.value?.ready && !runtimeActive.value && !portBlocked.value)
const canStopCluster = computed(() => runtimeActive.value)
const canStartActiveShard = computed(() => {
  if (!ready.value || !selectedCluster.value || !preflightResult.value?.ready) return false
  if (activeConsoleShard.value === 'Caves' && !selectedHasCaves.value) return false
  const state = runtimeState.value
  if (state && ['starting', 'running', 'stopping'].includes(state.status)) return false
  if (activeConsoleShard.value === 'Caves' && !shardActive(masterRuntimeState.value)) return false
  return true
})
const canStopActiveShard = computed(() => shardActive(runtimeState.value) && !(activeConsoleShard.value === 'Master' && shardActive(cavesRuntimeState.value)))
const canSendCommand = computed(() => !!runtimeState.value && ['starting', 'running'].includes(runtimeState.value.status))
const consoleCategoryOptions = [
  { value: 'all', label: '全部' },
  { value: 'important', label: '重要事件' },
  { value: 'player', label: '玩家' },
  { value: 'world', label: '世界' },
  { value: 'auth', label: '认证' },
  { value: 'steam', label: 'Steam' },
  { value: 'warning', label: '警告' },
  { value: 'error', label: '错误' },
]
const structuredConsoleLines = computed(() => {
  const query = consoleSearch.value.trim().toLowerCase()
  return runtimeLogs.value
    .map(classifyDSTConsoleLine)
    .filter((line) => {
      if (consoleFilter.value === 'important' && !line.important) return false
      if (consoleFilter.value !== 'all' && consoleFilter.value !== 'important' && line.category !== consoleFilter.value) return false
      if (!query) return true
      return `${line.title} ${line.detail} ${line.text} ${line.label}`.toLowerCase().includes(query)
    })
})

function applyRuntimeSnapshot(cluster: DSTClusterRuntimeSnapshot) {
  // Backend returns a fresh object on every poll. Avoid invalidating the entire
  // workspace when nothing changed; WebView2 otherwise repaints the large
  // console surface even for identical status snapshots.
  const key = JSON.stringify(cluster)
  if (key === runtimeSnapshotKey.value) return
  runtimeSnapshotKey.value = key
  clusterRuntimeState.value = cluster
  masterRuntimeState.value = cluster.master.found ? cluster.master.process : null
  cavesRuntimeState.value = cluster.caves.found ? cluster.caves.process : null
}

async function refreshRuntime(fetchLogs = true) {
  if (!selectedClusterPath.value || runtimeLoading.value) return
  runtimeLoading.value = true
  try {
    const cluster = await backend.dstClusterStatus({ clusterPath: selectedClusterPath.value })
    applyRuntimeSnapshot(cluster)
    const active = activeConsoleShard.value === 'Caves' ? cluster.caves : cluster.master
    if (active.found) {
      const nextSessionId = active.process.logSessionId ?? ''
      if (nextSessionId !== runtimeLogSessionId.value) {
        runtimeLogSessionId.value = nextSessionId
        runtimeLogs.value = []
        runtimeLogCursor.value = 0
        runtimeLogDropped.value = false
      }
    }
    const hasNewRuntimeLogs = active.found && (runtimeLogs.value.length === 0 || active.process.logCursor > runtimeLogCursor.value)
    if (active.found && fetchLogs && hasNewRuntimeLogs) {
      const after = runtimeLogs.value.length === 0 && runtimeLogCursor.value === 0 && active.process.logCursor > 500
        ? Math.max(0, active.process.logCursor - 500)
        : runtimeLogCursor.value
      const batch = await backend.dstProcessLogs({
        clusterPath: selectedClusterPath.value,
        shardName: activeConsoleShard.value,
        after,
        limit: 500,
      })
      if (batch.dropped) runtimeLogDropped.value = true
      if (batch.lines.length) {
        runtimeLogs.value.push(...batch.lines)
        if (runtimeLogs.value.length > 2000) runtimeLogs.value.splice(0, runtimeLogs.value.length - 2000)
      }
      runtimeLogCursor.value = batch.nextCursor
    }
    runtimeError.value = ''
  } catch (error) {
    runtimeError.value = error instanceof Error ? error.message : String(error)
  } finally {
    runtimeLoading.value = false
  }
}

async function startCluster() {
  if (!selectedClusterPath.value || !canStartCluster.value) return
  runtimeLoading.value = true
  runtimeError.value = ''
  runtimeMessage.value = ''
  runtimeLogs.value = []
  runtimeLogCursor.value = 0
  runtimeLogSessionId.value = ''
  runtimeLogDropped.value = false
  activeConsoleShard.value = 'Master'
  try {
    applyRuntimeSnapshot(await backend.startDSTCluster({ clusterPath: selectedClusterPath.value, ugcDirectory: '' }))
    section.value = 'overview'
  } catch (error) {
    runtimeError.value = error instanceof Error ? error.message : String(error)
  } finally {
    runtimeLoading.value = false
    await refreshRuntime(true)
  }
}

async function stopCluster() {
  if (!selectedClusterPath.value || !canStopCluster.value) return
  runtimeLoading.value = true
  runtimeError.value = ''
  runtimeMessage.value = ''
  try {
    applyRuntimeSnapshot(await backend.stopDSTCluster({ clusterPath: selectedClusterPath.value }))
  } catch (error) {
    runtimeError.value = error instanceof Error ? error.message : String(error)
  } finally {
    runtimeLoading.value = false
    await refreshRuntime(true)
  }
}

async function startActiveShard() {
  if (!selectedClusterPath.value || !canStartActiveShard.value) return
  runtimeLoading.value = true
  runtimeError.value = ''
  runtimeLogs.value = []
  runtimeLogCursor.value = 0
  runtimeLogSessionId.value = ''
  try {
    const state = await backend.startDSTShard({ clusterPath: selectedClusterPath.value, shardName: activeConsoleShard.value, ugcDirectory: '' })
    if (activeConsoleShard.value === 'Caves') cavesRuntimeState.value = state
    else masterRuntimeState.value = state
  } catch (error) {
    runtimeError.value = error instanceof Error ? error.message : String(error)
  } finally {
    runtimeLoading.value = false
    await refreshRuntime(true)
  }
}

async function stopActiveShard() {
  if (!selectedClusterPath.value || !canStopActiveShard.value) return
  runtimeLoading.value = true
  runtimeError.value = ''
  try {
    const state = await backend.stopDSTProcess({ clusterPath: selectedClusterPath.value, shardName: activeConsoleShard.value })
    if (activeConsoleShard.value === 'Caves') cavesRuntimeState.value = state
    else masterRuntimeState.value = state
  } catch (error) {
    runtimeError.value = error instanceof Error ? error.message : String(error)
  } finally {
    runtimeLoading.value = false
  }
}

async function repairRecommendedPorts() {
  if (!selectedClusterPath.value || runtimeActive.value) return
  portRepairLoading.value = true
  runtimeError.value = ''
  runtimeMessage.value = ''
  try {
    const config = await backend.dstPortConfiguration({ clusterPath: selectedClusterPath.value })
    const result = await backend.configureDSTPorts({
      clusterPath: selectedClusterPath.value,
      useRecommended: true,
      settings: config.recommended,
    })
    if (result.report.ready) {
      runtimeMessage.value = 'Master / Caves 推荐端口已自动补齐，启动前检查已重新执行。'
    } else {
      runtimeError.value = `推荐端口已写入，但当前仍有端口被占用：${result.report.blockers.join('；')}`
    }
    await refreshPreflight()
    await refreshRuntime(true)
  } catch (error) {
    runtimeError.value = error instanceof Error ? error.message : String(error)
  } finally {
    portRepairLoading.value = false
  }
}

async function cleanupPorts(force = false) {
  if (!selectedClusterPath.value || !portCleanupAllowed.value) return
  if (force && !window.confirm('只会强制结束占用当前端口的 DST Dedicated Server 进程，不会结束其他程序；若你手动运行了其他 DST 专服实例，也可能受到影响。继续吗？')) return
  runtimeLoading.value = true
  runtimeError.value = ''
  runtimeMessage.value = ''
  try {
    const result = await backend.cleanupDSTPorts({ clusterPath: selectedClusterPath.value, force })
    if (result.terminatedPids.length) {
      runtimeMessage.value = `已清理 DST 残留进程 PID：${result.terminatedPids.join(', ')}`
    } else if (result.skippedPids.length) {
      runtimeError.value = `仍有未自动结束的端口占用 PID：${result.skippedPids.join(', ')}`
    }
  } catch (error) {
    runtimeError.value = error instanceof Error ? error.message : String(error)
  } finally {
    runtimeLoading.value = false
    await refreshRuntime(false)
  }
}

async function sendConsoleCommand(commandOverride?: string) {
  const command = (commandOverride ?? consoleCommand.value).trim()
  if (!selectedClusterPath.value || !command || !canSendCommand.value) return
  try {
    const state = await backend.sendDSTCommand({ clusterPath: selectedClusterPath.value, shardName: activeConsoleShard.value, command })
    if (activeConsoleShard.value === 'Caves') cavesRuntimeState.value = state
    else masterRuntimeState.value = state
    if (!commandOverride) consoleCommand.value = ''
  } catch (error) {
    runtimeError.value = error instanceof Error ? error.message : String(error)
  }
}

function clearConsoleView() {
  runtimeLogs.value = []
  runtimeLogCursor.value = runtimeState.value?.logCursor ?? runtimeLogCursor.value
  runtimeLogDropped.value = false
}

function openLogCenter() {
  void router.push({
    path: '/logs',
    query: {
      source: 'dst',
      game: 'steam.dst',
      instance: selectedClusterPath.value || undefined,
      shard: activeConsoleShard.value,
    },
  })
}

function selectSection(id: WorkspaceSection) {
  if (id === 'logs') {
    openLogCenter()
    return
  }
  section.value = id
}

async function exportCurrentRuntimeLog() {
  const id = runtimeState.value?.logSessionId
  if (!id) return
  runtimeMessage.value = ''
  try {
    const result = await backend.exportDSTLog(id)
    runtimeMessage.value = backend.mode() === 'desktop'
      ? `日志已保存到 AGMP 项目 log 目录：${result.path}`
      : `日志已下载：${result.name}`
  } catch (error) {
    runtimeError.value = error instanceof Error ? error.message : String(error)
  }
}

watch(selectedHasCaves, (hasCaves) => {
  if (!hasCaves && activeConsoleShard.value === 'Caves') activeConsoleShard.value = 'Master'
})

watch(activeConsoleShard, () => {
  runtimeLogs.value = []
  runtimeLogCursor.value = 0
  runtimeLogSessionId.value = ''
  runtimeLogDropped.value = false
  consoleCommand.value = ''
  if (selectedClusterPath.value) void refreshRuntime(true)
})

function stopRuntimePolling() {
  if (runtimeTimer !== undefined) {
    window.clearTimeout(runtimeTimer)
    runtimeTimer = undefined
  }
}

function runtimePollDelay() {
  if (typeof document !== 'undefined' && document.visibilityState !== 'visible') return 5000
  return runtimeActive.value ? 1200 : 3000
}

function scheduleRuntimePolling(delay = runtimePollDelay()) {
  stopRuntimePolling()
  if (section.value !== 'overview' || !selectedClusterPath.value) return
  runtimeTimer = window.setTimeout(async () => {
    runtimeTimer = undefined
    if (section.value !== 'overview' || !selectedClusterPath.value) return
    await refreshRuntime(true)
    scheduleRuntimePolling()
  }, delay)
}

watch(section, () => scheduleRuntimePolling(0))
watch(selectedClusterPath, () => scheduleRuntimePolling(250))
watch(runtimeActive, () => scheduleRuntimePolling())

onMounted(() => scheduleRuntimePolling(0))

onUnmounted(stopRuntimePolling)

const lastScanText = computed(() => {
  if (!props.scannedAt) return ''
  return new Date(props.scannedAt * 1000).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
})

const dedicatedSource = computed(() => {
  switch (props.dedicated?.installation.source) {
    case 'manual': return '手动确认路径'
    case 'steam': return 'Steam Library 自动发现'
    default: return '尚未发现'
  }
})

function formatBytes(bytes: number) {
  if (!bytes) return '未知'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let value = bytes
  let index = 0
  while (value >= 1024 && index < units.length - 1) { value /= 1024; index += 1 }
  return `${value.toFixed(index >= 3 ? 1 : 0)} ${units[index]}`
}

function formatUpdated(timestamp: number) {
  if (!timestamp) return '未知'
  const date = new Date(timestamp * 1000)
  if (Number.isNaN(date.getTime())) return '未知'
  return date.toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

function handleTokenChanged() {
  void refreshPreflight()
}

function handleClusterImported(result: DSTClusterImportResult) {
  selectedClusterPath.value = result.path
  emit('refresh')
}

async function saveDedicatedPreferences() {
  preferenceMessage.value = ''
  preferenceSaving.value = true
  try {
    await backend.saveDSTDedicatedPreferences({
      dedicatedServerPath: manualServerPath.value.trim(),
      dedicatedServerExtraArgs: extraArgs.value.trim(),
    })
    preferenceMessage.value = '已保存，正在重新验证 Dedicated Server…'
    emit('refresh')
  } catch (error) {
    preferenceMessage.value = error instanceof Error ? error.message : String(error)
  } finally {
    preferenceSaving.value = false
  }
}

async function clearManualPath() {
  manualServerPath.value = ''
  await saveDedicatedPreferences()
}

const placeholderCopy: Partial<Record<WorkspaceSection, { title: string; description: string }>> = {
  rooms: { title: '房间管理', description: '这里将管理 Cluster / Master / Caves、房间名称、密码、人数和运行状态。' },
  world: { title: '世界设置', description: '这里将提供地面与洞穴世界设置，并保持与 Klei 原生配置兼容。' },
  mods: { title: 'Mod 管理', description: '这里将管理 Steam Workshop Mod、服务器 Mod 配置与更新。' },
  players: { title: '玩家管理', description: '这里将展示在线/历史玩家，并逐步接入管理员、白名单、黑名单与封禁管理。' },
  backups: { title: '备份', description: '这里将提供服务器级快照、自动备份、恢复与保留策略。' },
}

const activePlaceholder = computed(() => placeholderCopy[section.value] ?? null)
</script>

<template>
  <section class="dst-workspace-shell">
    <aside class="dst-workspace-nav panel">
      <div class="dst-game-identity">
        <div class="dst-game-identity__mark">饥</div>
        <div>
          <span class="eyebrow">STEAM.DST</span>
          <h2>饥荒联机版</h2>
          <small>Don't Starve Together</small>
        </div>
      </div>

      <div class="dst-env-summary">
        <StatusPill v-if="loading" tone="warning" label="环境检测中" />
        <StatusPill v-else-if="error" tone="danger" label="检测失败" />
        <StatusPill v-else-if="ready" tone="success" label="环境就绪" />
        <StatusPill v-else tone="warning" label="需要准备" />
        <button class="btn btn--secondary btn--compact" :disabled="loading" @click="emit('refresh')">{{ loading ? '检测中…' : '重新检测' }}</button>
        <small v-if="lastScanText">{{ lastScanText }} · {{ durationMs ?? 0 }} ms</small>
      </div>

      <nav class="dst-section-nav" aria-label="饥荒联机版管理功能">
        <button v-for="item in sections" :key="item.id" :class="{ active: section === item.id }" @click="selectSection(item.id)">
          <strong>{{ item.label }}</strong><small>{{ item.hint }}</small>
        </button>
      </nav>

      <div class="dst-nav-footer">
        <div><span>游戏本体</span><strong>{{ gameAppId || '—' }}</strong></div>
        <div><span>专用服务器</span><strong>{{ serverAppId || '—' }}</strong></div>
      </div>
    </aside>

    <main class="dst-workspace-main">
      <div v-if="error" class="notice notice--error">DST 环境检测失败：{{ error }}</div>
      <div v-else-if="warnings?.length" class="notice"><strong>检测提示：</strong>{{ warnings[0] }}<span v-if="warnings.length > 1">（另有 {{ warnings.length - 1 }} 条）</span></div>

      <template v-if="section === 'overview'">
        <section class="panel dst-first-run-flow">
          <div class="dst-first-run-flow__head"><div><span class="eyebrow">FIRST SERVER</span><h3>首次开服流程</h3><p>按顺序准备 Steam 文件、SERVER 世界、Klei 令牌，全部通过 Preflight 后才允许启动。</p></div><StatusPill :tone="preflightResult?.ready ? 'success' : 'warning'" :label="preflightResult?.ready ? '准备完成' : '仍需准备'" /></div>
          <div class="dst-first-run-flow__steps">
            <button :class="{ done: firstRunFileReady }" @click="section = 'install'"><span>01</span><div><strong>Steam 文件</strong><small>{{ firstRunFileReady ? '游戏本体与专服已就绪' : '安装或校验 322330 / 343050' }}</small></div><b>{{ firstRunFileReady ? '✓' : '→' }}</b></button>
            <button :class="{ done: firstRunClusterReady }" @click="section = 'saves'"><span>02</span><div><strong>服务器世界</strong><small>{{ firstRunClusterReady ? `已发现 ${serverClusters.length} 个 SERVER Cluster` : '导入客户端世界或现有 Cluster' }}</small></div><b>{{ firstRunClusterReady ? '✓' : '→' }}</b></button>
            <button :class="{ done: firstRunTokenReady }" @click="section = 'tokens'"><span>03</span><div><strong>Klei 服务器令牌</strong><small>{{ firstRunTokenReady ? '当前 Cluster 已配置 Token' : '上传或手动输入 cluster_token' }}</small></div><b>{{ firstRunTokenReady ? '✓' : '→' }}</b></button>
            <button :class="{ done: preflightResult?.ready }" @click="refreshPreflight"><span>04</span><div><strong>启动前检查</strong><small>{{ preflightResult?.ready ? '所有阻止项已通过' : `${preflightResult?.blockers ?? '—'} 个阻止项` }}</small></div><b>{{ preflightResult?.ready ? '✓' : '↻' }}</b></button>
          </div>
        </section>

        <header class="dst-workbar panel">
          <div class="dst-workbar__cluster">
            <span class="eyebrow">ACTIVE SERVER</span>
            <label for="dst-master-cluster">当前 SERVER Cluster</label>
            <select id="dst-master-cluster" v-model="selectedClusterPath" :disabled="runtimeActive">
              <option value="" disabled>请选择一个可开服 Cluster</option>
              <option v-for="cluster in serverClusters" :key="cluster.path" :value="cluster.path">{{ cluster.name }} · {{ cluster.path }}</option>
            </select>
          </div>
          <div class="dst-workbar__status">
            <StatusPill :tone="clusterTone" :label="clusterStatusLabel" />
            <small v-if="masterRuntimeState">Master PID {{ masterRuntimeState.pid || '—' }} · {{ masterRuntimeState.worldReady ? 'Ready' : '加载中' }}<template v-if="selectedHasCaves"> · Caves {{ cavesRuntimeState?.pid ? `PID ${cavesRuntimeState.pid}` : '未启动' }}</template></small>
            <small v-else>当前 Cluster 尚未由 AGMP 启动</small>
          </div>
          <div class="dst-workbar__actions">
            <button class="btn btn--primary" :disabled="runtimeLoading || !canStartCluster" @click="startCluster">{{ runtimeLoading && !runtimeActive ? '处理中…' : (selectedHasCaves ? '一键启动地面 + 洞穴' : '启动地面服务器') }}</button>
            <button class="btn btn--secondary" :disabled="runtimeLoading || !canStopCluster" @click="stopCluster">全部优雅停止</button>
          </div>
        </header>

        <div v-if="serverClusters.length === 0" class="notice notice--warning">尚未发现 Steam SERVER 类型且包含 Master/server.ini 的 Cluster。本地玩家存档（LOCAL）不会被当成 Dedicated Server Cluster 启动。</div>
        <div v-if="runtimeError" class="notice notice--error">{{ runtimeError }}</div>
        <div v-if="runtimeMessage" class="notice notice--success">{{ runtimeMessage }}</div>
        <div v-if="portBlocked" class="notice notice--error dst-port-blocker">
          <div>
            <strong>启动端口检查未通过</strong>
            <span v-for="message in clusterRuntimeState?.ports.blockers ?? []" :key="message">{{ message }}</span>
          </div>
          <div class="dst-port-blocker__actions">
            <button class="btn btn--secondary btn--compact" :disabled="runtimeLoading" @click="cleanupPorts(false)">清理 AGMP 托管残留</button>
            <button class="btn btn--danger btn--compact" :disabled="runtimeLoading" @click="cleanupPorts(true)">强制清理 DST 残留</button>
          </div>
        </div>
        <div v-if="portPreflightBlocked" class="notice notice--warning dst-network-quickfix">
          <div><strong>洞穴联动需要补齐端口配置</strong><span>普通用户无需手工编辑 ini，AGMP 可以直接生成一套互不冲突的推荐端口。</span></div>
          <div class="dst-network-quickfix__actions">
            <button class="btn btn--primary btn--compact" :disabled="portRepairLoading || runtimeActive" @click="repairRecommendedPorts">{{ portRepairLoading ? '正在修复…' : '一键修复推荐端口' }}</button>
            <button class="btn btn--secondary btn--compact" @click="section = 'network'">自定义端口</button>
          </div>
        </div>
        <PreflightPanel :result="preflightResult" :loading="preflightLoading" :error="preflightError" @refresh="refreshPreflight" @configure-token="section = 'tokens'" @configure-network="section = 'network'" @import-world="section = 'saves'" />
        <div v-if="runtimeState?.logPersistenceError" class="notice notice--warning">实时控制台仍可使用，但本次持久化日志初始化失败：{{ runtimeState.logPersistenceError }}</div>

        <section class="dst-runtime-cards">
          <article class="panel dst-runtime-card">
            <div class="dst-runtime-card__top"><span>MASTER · 地面</span><StatusPill :tone="shardTone(masterRuntimeState)" :label="shardStatusLabel(masterRuntimeState)" /></div>
            <strong>{{ selectedCluster?.name || '未选择 Cluster' }}</strong>
            <small>{{ masterRuntimeState?.pid ? `PID ${masterRuntimeState.pid}` : '等待启动' }}</small>
            <div class="dst-runtime-card__meta"><span>世界 Ready</span><b>{{ masterRuntimeState?.worldReady ? '是' : '否' }}</b></div>
          </article>
          <article class="panel dst-runtime-card" :class="{ 'dst-runtime-card--planned': !selectedHasCaves }">
            <div class="dst-runtime-card__top"><span>CAVES · 洞穴</span><StatusPill :tone="selectedHasCaves ? shardTone(cavesRuntimeState) : 'muted'" :label="selectedHasCaves ? shardStatusLabel(cavesRuntimeState) : '未配置'" /></div>
            <strong>{{ selectedCluster?.name || '未选择 Cluster' }}</strong>
            <small v-if="selectedHasCaves">{{ cavesRuntimeState?.pid ? `PID ${cavesRuntimeState.pid}` : '等待与 Master 联动启动' }}</small>
            <small v-else>当前 Cluster 没有 Caves/server.ini，将按地面单 Shard 运行。</small>
            <div class="dst-runtime-card__meta"><span>Shard 联动</span><b>{{ clusterRuntimeState?.shardLink === 'ready' ? '已就绪' : clusterRuntimeState?.shardLink === 'waiting' ? '连接中' : clusterRuntimeState?.shardLink === 'disconnected' ? '未连接' : selectedHasCaves ? '等待启动' : '不适用' }}</b></div>
          </article>
          <article class="panel dst-runtime-card">
            <div class="dst-runtime-card__top"><span>DEDICATED SERVER</span><StatusPill :tone="serverInstalled ? 'success' : 'warning'" :label="serverInstalled ? `${dedicated?.installation.bitness}-bit` : '不可用'" /></div>
            <strong>AppID {{ serverAppId || '—' }}</strong>
            <small>{{ dedicated?.installation.executable || '未检测到有效 EXE' }}</small>
            <div class="dst-runtime-card__meta"><span>Klei</span><b>{{ dedicated?.confDir.valid ? '可用' : '异常' }}</b></div>
          </article>
        </section>

        <section class="panel dst-live-console">
          <div class="dst-live-console__head">
            <div><span class="eyebrow">LIVE CONSOLE</span><h3>{{ activeConsoleShard === 'Master' ? 'Master · 地面' : 'Caves · 洞穴' }} 实时控制台</h3></div>
            <div class="dst-live-console__head-actions">
              <button class="btn btn--secondary btn--compact" @click="openLogCenter">日志中心</button>
              <button class="btn btn--secondary btn--compact" :disabled="!runtimeState?.logSessionId" @click="exportCurrentRuntimeLog">下载当前日志</button>
              <button class="btn btn--secondary btn--compact" :disabled="runtimeLogs.length === 0" @click="clearConsoleView">清空显示</button>
              <button class="btn btn--secondary btn--compact" :class="{ active: consoleAutoScroll }" @click="consoleAutoScroll = !consoleAutoScroll">自动滚动 {{ consoleAutoScroll ? '开' : '关' }}</button>
              <button class="btn btn--secondary btn--compact" :disabled="runtimeLoading || !selectedClusterPath" @click="refreshRuntime(true)">刷新状态</button>
              <StatusPill :tone="runtimeTone" :label="runtimeStatusLabel" />
            </div>
          </div>
          <div class="dst-console-shard-tabs" role="tablist" aria-label="DST Shard 控制台">
            <button type="button" role="tab" :aria-selected="activeConsoleShard === 'Master'" :class="{ active: activeConsoleShard === 'Master' }" @click="activeConsoleShard = 'Master'">
              <span>Master</span><strong>地面</strong><i :class="`is-${shardTone(masterRuntimeState)}`"></i>
            </button>
            <button type="button" role="tab" :aria-selected="activeConsoleShard === 'Caves'" :class="{ active: activeConsoleShard === 'Caves' }" :disabled="!selectedHasCaves" @click="activeConsoleShard = 'Caves'">
              <span>Caves</span><strong>洞穴</strong><i :class="`is-${selectedHasCaves ? shardTone(cavesRuntimeState) : 'muted'}`"></i>
            </button>
          </div>
          <div v-if="runtimeLogDropped" class="notice notice--warning">日志增长速度超过浏览器读取窗口，较早的实时日志已从内存缓冲区淘汰。</div>
          <div class="dst-console-viewbar">
            <div class="dst-console-mode-switch">
              <button type="button" :class="{ active: consoleViewMode === 'structured' }" @click="consoleViewMode = 'structured'">结构化</button>
              <button type="button" :class="{ active: consoleViewMode === 'raw' }" @click="consoleViewMode = 'raw'">原始日志</button>
            </div>
            <template v-if="consoleViewMode === 'structured'">
              <select v-model="consoleFilter" aria-label="控制台分类筛选">
                <option v-for="option in consoleCategoryOptions" :key="option.value" :value="option.value">{{ option.label }}</option>
              </select>
              <input v-model="consoleSearch" type="search" placeholder="搜索玩家、端口、错误、事件…" />
              <small>{{ structuredConsoleLines.length }} / {{ runtimeLogs.length }} 条</small>
            </template>
            <small v-else>原始 stdout / stderr · {{ runtimeLogs.length }} 条</small>
          </div>
          <div ref="consoleViewport" class="dst-console-output dst-console-output--integrated" :class="{ 'dst-console-output--structured': consoleViewMode === 'structured' }" aria-live="polite">
            <div v-if="runtimeLogs.length === 0" class="dst-console-empty">{{ runtimeState ? '等待服务器输出…' : `启动 ${activeConsoleShard === 'Master' ? 'Master 地面' : 'Caves 洞穴'} 后，stdout / stderr 会在这里实时显示。` }}</div>
            <template v-else-if="consoleViewMode === 'structured'">
              <div v-if="structuredConsoleLines.length === 0" class="dst-console-empty">当前筛选条件下没有日志事件。</div>
              <div v-for="line in structuredConsoleLines" :key="line.sequence" class="dst-console-event" :class="`dst-console-event--${line.category}`">
                <span class="dst-console-event__time">{{ line.clock || line.sequence.toString().padStart(5, '0') }}</span>
                <span class="dst-console-event__badge">{{ line.label }}</span>
                <div class="dst-console-event__body"><strong>{{ line.title }}</strong><small v-if="line.detail">{{ line.detail }}</small></div>
              </div>
            </template>
            <template v-else>
              <div v-for="line in runtimeLogs" :key="line.sequence" class="dst-console-line"><span>{{ line.sequence.toString().padStart(5, '0') }}</span><code>{{ line.text }}</code></div>
            </template>
          </div>
          <form class="dst-console-command" @submit.prevent="sendConsoleCommand()">
            <input v-model="consoleCommand" type="text" :disabled="!canSendCommand" placeholder="输入 DST 控制台命令，例如 c_save()" />
            <button class="btn btn--primary" type="submit" :disabled="!canSendCommand || !consoleCommand.trim()">发送</button>
          </form>
          <div class="dst-console-quick-actions">
            <button class="btn btn--secondary btn--compact" :disabled="!canSendCommand" @click="sendConsoleCommand('c_save()')">保存世界</button>
            <button class="btn btn--secondary btn--compact" disabled>公告 · 待迁移</button>
            <button class="btn btn--secondary btn--compact" disabled>玩家列表 · 待迁移</button>
            <button class="btn btn--secondary btn--compact" disabled>回档 · 待迁移</button>
            <button v-if="canStartActiveShard" class="btn btn--secondary btn--compact" :disabled="runtimeLoading" @click="startActiveShard">启动当前 Shard</button>
            <button class="btn btn--danger btn--compact" :disabled="!canStopActiveShard" @click="stopActiveShard">停止当前 Shard</button>
          </div>
        </section>
      </template>

      <template v-else-if="section === 'network'">
        <NetworkConfig :cluster-path="selectedClusterPath" :runtime-active="runtimeActive" @configured="refreshPreflight(); refreshRuntime(true)" />
      </template>

      <template v-else-if="section === 'tokens'">
        <TokenManager :clusters="serverClusters" :cluster-path="selectedClusterPath" :runtime-active="runtimeActive" @select="selectedClusterPath = $event" @changed="handleTokenChanged" @refresh="emit('refresh')" />
      </template>

      <template v-else-if="section === 'saves'">
        <ClusterImporter :environment="environment" :runtime-active="runtimeActive" @imported="handleClusterImported" @refresh="emit('refresh')" />
      </template>

      <template v-else-if="section === 'install'">
        <article class="panel workspace-panel workspace-panel--wide">
          <div class="workspace-panel__heading">
            <div><span class="eyebrow">INSTALL & VERIFY</span><h3>安装、更新与文件完整性</h3><p>游戏本体与 Steam Library 专用服务器都交给用户自己的 Steam 官方客户端安装/校验。AGMP 不接收 Steam 密码，只监控真实 Steam 状态。</p></div>
            <StatusPill :tone="serverInstalled ? 'success' : 'warning'" :label="serverInstalled ? 'Dedicated Server 有效' : '需要处理'" />
          </div>

          <div v-if="!gameInstalled && !serverManifestInstalled" class="notice notice--warning dst-install-order">
            当前电脑尚未安装饥荒联机版游戏本体和 Dedicated Server。请先使用用户自己的 Steam 安装游戏本体（AppID {{ gameAppId }}），完成后再安装专用服务器（AppID {{ serverAppId }}）。账号登录、许可和安装目录全部由 Steam 官方客户端处理。
          </div>

          <div class="dst-prerequisite-grid dst-prerequisite-grid--install">
            <section class="game-install-card">
              <div class="game-install-card__heading"><div><span>Steam Client</span><strong>饥荒联机版游戏本体</strong></div><StatusPill :tone="gameInstalled ? 'success' : 'warning'" :label="gameInstalled ? '已安装' : '未安装'" /></div>
              <div class="game-install-card__appid"><span>完整 AppID</span><code>{{ gameAppId }}</code></div>
              <div class="game-install-card__details"><code>{{ game?.gameApp?.installPath || '未检测到' }}</code><small>Build {{ game?.gameApp?.buildId || '未知' }} · {{ formatBytes(game?.gameApp?.sizeOnDisk ?? 0) }} · {{ formatUpdated(game?.gameApp?.lastUpdated ?? 0) }}</small></div>
              <button v-if="gameInstalled" class="btn btn--primary" :disabled="maintenanceBusy" @click="validateSteamApp(gameAppId)">使用 Steam 校验游戏本体</button>
              <button v-else class="btn btn--primary" :disabled="!steamAvailable || maintenanceBusy" @click="installSteamApp(gameAppId)">使用 Steam 安装游戏本体</button>
              <small v-if="!gameInstalled && !steamAvailable" class="dst-runtime-message">未检测到 Steam 官方客户端，请先安装并登录 Steam。</small>
            </section>

            <section class="game-install-card">
              <div class="game-install-card__heading"><div><span>Dedicated Server</span><strong>饥荒联机版专用服务器</strong></div><StatusPill :tone="serverInstalled ? 'success' : 'warning'" :label="serverInstalled ? `${dedicated?.installation.bitness}-bit` : '无效/未安装'" /></div>
              <div class="game-install-card__appid"><span>完整 AppID</span><code>{{ serverAppId }}</code></div>
              <div class="game-install-card__details"><code>{{ dedicated?.installation.rootDir || '未检测到' }}</code><small>Bin：{{ dedicated?.installation.binDir || '—' }}</small><small>EXE：{{ dedicated?.installation.executable || '—' }}</small></div>
              <button v-if="serverManifestInstalled" class="btn btn--primary" :disabled="runtimeActive || maintenanceBusy" @click="validateSteamApp(serverAppId)">校验专用服务器完整性</button>
              <button v-else class="btn btn--primary" :disabled="!steamAvailable || !gameInstalled || runtimeActive || maintenanceBusy" @click="installSteamApp(serverAppId)">使用 Steam 安装专用服务器</button>
              <small v-if="runtimeActive" class="dst-runtime-message">Master/Caves 运行时禁止修改专服文件，请先正常停止服务器。</small>
              <small v-else-if="!serverManifestInstalled && !steamAvailable" class="dst-runtime-message">未检测到 Steam 官方客户端，无法发起专用服务器安装。</small>
              <small v-else-if="!serverManifestInstalled && !gameInstalled" class="dst-runtime-message">请先完成游戏本体 AppID {{ gameAppId }} 的 Steam 安装，再安装专用服务器。</small>
            </section>
          </div>

          <section v-if="maintenanceTask || maintenanceError" class="dst-maintenance panel">
            <div class="dst-maintenance__head">
              <div><span class="eyebrow">STEAM MAINTENANCE</span><h4>{{ maintenanceTitle }}</h4></div>
              <StatusPill v-if="maintenanceTask" :tone="maintenanceTask.state === 'completed' && maintenanceTask.completionConfirmed ? 'success' : maintenanceTask.state === 'failed' || maintenanceTask.state === 'timeout' ? 'danger' : 'warning'" :label="maintenanceStateLabel" />
            </div>
            <div v-if="maintenanceError" class="notice notice--error">{{ maintenanceError }}</div>
            <template v-if="maintenanceTask">
              <div class="dst-maintenance__meta">
                <span>AppID {{ maintenanceTask.appId }}</span>
                <span>阶段 {{ maintenanceTask.phase }}</span>
                <strong>{{ maintenanceProgressText }}</strong>
                <code>{{ maintenanceTask.uri }}</code>
              </div>
              <div class="dst-maintenance__progress" :class="{ 'is-indeterminate': maintenanceTask.progressMode !== 'determinate' }">
                <i v-if="maintenanceTask.progressMode === 'determinate'" :style="{ width: `${maintenanceTask.progress}%` }"></i>
                <i v-else></i>
              </div>
              <div v-if="maintenanceTask.bytesTotal > 0" class="dst-maintenance__bytes">
                <span v-if="maintenanceTask.progressEstimated">约</span><span>{{ formatBytes(maintenanceTask.bytesDone) }}</span><span>/</span><span>{{ formatBytes(maintenanceTask.bytesTotal) }}</span>
              </div>
              <p>{{ maintenanceTask.message }}</p>
              <div class="dst-maintenance__facts">
                <div><span>真实安装目录</span><code>{{ maintenanceTask.installPath || '等待 Steam 写入安装目录' }}</code></div>
                <div><span>Steam Library</span><code>{{ maintenanceTask.libraryPath || '—' }}</code></div>
                <div><span>AppManifest</span><code>{{ maintenanceTask.manifestPath || '—' }}</code></div>
                <div><span>Steam content_log</span><code>{{ maintenanceTask.contentLogPath || '—' }}</code></div>
                <div><span>StateFlags</span><code>{{ maintenanceTask.stateFlags }}</code></div>
                <div><span>进度来源</span><code>{{ maintenanceProgressSourceLabel }}</code></div>
              </div>
              <div v-if="maintenanceTask.validationFiles > 0 || maintenanceTask.mismatchedFiles > 0" class="dst-maintenance__result">
                <strong>Steam 文件扫描结果</strong>
                <span>扫描 {{ maintenanceTask.validationFiles }} 个文件 · {{ formatBytes(maintenanceTask.validationBytes) }}</span>
                <span :class="{ 'is-warning': maintenanceTask.mismatchedFiles > 0 }">不匹配 {{ maintenanceTask.mismatchedFiles }} 个 · {{ formatBytes(maintenanceTask.mismatchedBytes) }}</span>
              </div>
              <small v-if="maintenanceTask.state === 'completed' && maintenanceTask.completionConfirmed">完成依据：{{ maintenanceTask.completionSource }}</small>
              <small v-else>校验以“本次请求”的独立 Steam 会话为准：优先读取本次 content_log Validating；若 Steam 未及时暴露开始事件，则用连续真实磁盘读取作为开始兜底。StateFlags=4 不能单独宣布完成，I/O 也只能驱动近似进度。估算最高 99%，只有本次会话出现 Steam 终态事件并再次确认 AppManifest 稳定后才会显示 100% / 校验完成。重复校验不会复用上一轮终态。</small>
            </template>
          </section>

          <section class="dst-runtime-settings">
            <div class="dst-runtime-settings__heading"><div><span class="eyebrow">PATH OVERRIDE</span><h4>专用服务器手动路径</h4></div><StatusPill :tone="dedicated?.installation.source === 'manual' ? 'success' : 'muted'" :label="dedicatedSource" /></div>
            <p>保持 DSTCamp 原行为：有效手动路径优先于 Steam Library 自动发现；无效路径自动回退。</p>
            <div class="dst-runtime-field"><label for="dst-server-path">Dedicated Server 根目录</label><input id="dst-server-path" v-model="manualServerPath" type="text" placeholder="例如 D:\steam\steam\steamapps\common\Don't Starve Together Dedicated Server" /></div>
            <div class="dst-runtime-field"><label for="dst-extra-args">额外启动参数</label><input id="dst-extra-args" v-model="extraArgs" type="text" placeholder="例如 -foo 1 -bar &quot;hello world&quot;" /></div>
            <div class="dst-runtime-settings__actions"><button class="btn btn--secondary" :disabled="preferenceSaving" @click="clearManualPath">清除手动路径</button><button class="btn btn--primary" :disabled="preferenceSaving" @click="saveDedicatedPreferences">{{ preferenceSaving ? '保存中…' : '保存并重新检测' }}</button></div>
            <small v-if="preferenceMessage" class="dst-runtime-message">{{ preferenceMessage }}</small>
          </section>

          <section class="dst-runtime-settings">
            <div class="dst-runtime-settings__heading"><div><span class="eyebrow">CONF_DIR</span><h4>Klei 启动目录</h4></div><StatusPill :tone="dedicated?.confDir.valid ? 'success' : 'warning'" :label="dedicated?.confDir.valid ? (dedicated?.confDir.default ? '默认' : '自定义') : '不可用'" /></div>
            <div class="dst-runtime-kv"><span>Windows 文档</span><code>{{ dedicated?.confDir.documentsDir || '—' }}</code></div>
            <div class="dst-runtime-kv"><span>Klei 根目录</span><code>{{ dedicated?.confDir.kleiRoot || '—' }}</code></div>
            <div class="dst-runtime-kv"><span>-conf_dir</span><code>{{ dedicated?.confDir.valid ? (dedicated?.confDir.argument || '（默认路径，不传该参数）') : (dedicated?.confDir.error || '—') }}</code></div>
          </section>
        </article>
      </template>

      <article v-else class="panel workspace-placeholder workspace-placeholder--centered">
        <span class="eyebrow">GAME WORKSPACE</span>
        <h3>{{ activePlaceholder?.title }}</h3>
        <p>{{ activePlaceholder?.description }}</p>
        <StatusPill tone="muted" label="框架已预留 · 功能后续按 DSTCamp 等价迁移" />
      </article>
    </main>
  </section>
</template>
