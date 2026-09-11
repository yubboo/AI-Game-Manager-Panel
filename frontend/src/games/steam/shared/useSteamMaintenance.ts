import { computed, onUnmounted, ref } from 'vue'
import { backend } from '../../../shared/api/backend'
import type { SteamMaintenanceTask } from '../../../shared/types/backend'

export interface SteamMaintenanceOptions {
  onCompleted?: (task: SteamMaintenanceTask) => void
  pollIntervalMs?: number
}

// Shared Steam maintenance controller for every Steam game workspace.
// Game-specific UIs only provide an AppID and any game-side safety/availability
// rules. Session creation, duplicate-request protection, polling and task labels
// all stay here so DST/Palworld/Valheim/etc. do not fork validation behavior.
export function useSteamMaintenance(options: SteamMaintenanceOptions = {}) {
  const task = ref<SteamMaintenanceTask | null>(null)
  const error = ref('')
  const starting = ref(false)
  let timer: number | undefined

  const busy = computed(() => starting.value || (!!task.value && ['requested', 'monitoring'].includes(task.value.state)))

  const stateLabel = computed(() => {
    const value = task.value
    const operation = value?.operation === 'install' ? '安装' : '校验'
    if (value?.state === 'requested') return `等待${operation}`
    if (value?.state === 'monitoring') return `Steam ${operation}中`
    if (value?.state === 'completed') return `${operation}完成`
    if (value?.state === 'timeout') return '等待超时'
    if (value?.state === 'failed') return `${operation}失败`
    return value?.state || '等待'
  })

  const title = computed(() => task.value?.operation === 'install' ? 'Steam 官方安装任务' : 'Steam 官方文件校验')

  const progressText = computed(() => {
    const value = task.value
    if (!value) return ''
    if (value.progressMode === 'determinate') {
      return `${value.progressEstimated ? '约 ' : ''}${Math.max(0, Math.min(100, value.progress))}%`
    }
    if (value.phase === 'validating') return 'Steam 扫描中'
    if (value.phase === 'finalizing') return 'Steam 收尾中'
    return '实时监控中'
  })

  const progressSourceLabel = computed(() => {
    const source = task.value?.progressSource || ''
    if (source === 'steam-appmanifest-bytesdownloaded') return 'AppManifest 下载字节'
    if (source === 'steam-appmanifest-bytesstaged') return 'AppManifest 暂存字节'
    if (source === 'steam-appmanifest-stateflags+content-log') return 'AppManifest StateFlags + content_log'
    if (source === 'steam-windows-process-io-estimate') return 'Steam Validating + Windows 进程实际读取量（估算）'
    if (source === 'steam-terminal-gate') return 'Steam 终态门禁（99% 等待确认）'
    if (source === 'steam-content-log-validating') return 'Steam content_log 验证阶段'
    if (source === 'steam-terminal-confirmed') return 'Steam 终态确认'
    return source || 'Steam content_log + AppManifest'
  })

  function stopPolling() {
    if (timer !== undefined) {
      window.clearInterval(timer)
      timer = undefined
    }
  }

  async function refresh() {
    const id = task.value?.id
    if (!id) return
    try {
      task.value = await backend.steamMaintenanceTask(id)
      if (['completed', 'failed', 'timeout'].includes(task.value.state)) {
        stopPolling()
        if (task.value.state === 'completed') options.onCompleted?.(task.value)
      }
    } catch (cause) {
      error.value = cause instanceof Error ? cause.message : String(cause)
      stopPolling()
    }
  }

  function beginPolling() {
    stopPolling()
    const interval = Math.max(500, options.pollIntervalMs ?? 1000)
    timer = window.setInterval(() => void refresh(), interval)
  }

  async function startInstall(appId: number) {
    if (busy.value) return task.value
    starting.value = true
    error.value = ''
    stopPolling()
    try {
      task.value = await backend.startSteamInstall({ appId })
      beginPolling()
      return task.value
    } catch (cause) {
      error.value = cause instanceof Error ? cause.message : String(cause)
      return null
    } finally {
      starting.value = false
    }
  }

  async function startValidation(appId: number) {
    if (busy.value) return task.value
    starting.value = true
    error.value = ''
    stopPolling()
    try {
      task.value = await backend.startSteamValidation({ appId })
      beginPolling()
      return task.value
    } catch (cause) {
      error.value = cause instanceof Error ? cause.message : String(cause)
      return null
    } finally {
      starting.value = false
    }
  }

  onUnmounted(stopPolling)

  return {
    task,
    error,
    starting,
    busy,
    stateLabel,
    title,
    progressText,
    progressSourceLabel,
    refresh,
    startInstall,
    startValidation,
    stopPolling,
  }
}
