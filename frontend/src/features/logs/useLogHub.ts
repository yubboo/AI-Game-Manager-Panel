import { computed, onBeforeUnmount, reactive, ref } from 'vue'
import { backend } from '../../shared/api/backend'
import type {
  GlobalLogCatalogPage,
  GlobalLogCatalogRequest,
  GlobalLogExportResult,
  GlobalLogFile,
  GlobalLogLine,
  GlobalLogMutationResult,
} from '../../shared/types/backend'

export interface LogHubOptions {
  catalogPageSize: () => number
  readPageSize: () => number
  autoRefreshSeconds: () => number
}

function emptyCatalog(limit: number): GlobalLogCatalogPage {
  return {
    items: [],
    summary: { files: 0, lines: 0, bytes: 0, activeFiles: 0 },
    total: 0,
    offset: 0,
    limit,
    durationMs: 0,
  }
}

export function useLogHub(options: LogHubOptions) {
  const filters = reactive({
    query: '',
    source: 'all',
    kind: 'all',
    gameId: 'all',
    instanceId: '',
    shard: 'all',
    status: 'all',
    dateFrom: '',
    dateTo: '',
  })
  const contentFilters = reactive({ query: '', level: 'all', category: 'all' })
  const initialCatalogSize = Math.max(1, options.catalogPageSize() || 100)
  const catalog = ref<GlobalLogCatalogPage>(emptyCatalog(initialCatalogSize))
  const selected = ref<GlobalLogFile | null>(null)
  const lines = ref<GlobalLogLine[]>([])
  const direction = ref<'head' | 'tail'>('tail')
  const nextCursor = ref(0)
  const nextLine = ref(1)
  const eof = ref(true)
  const scanned = ref(0)
  const matched = ref(0)
  const readDurationMs = ref(0)
  const loadingCatalog = ref(false)
  const loadingContent = ref(false)
  const error = ref('')
  const notice = ref('')
  const autoRefresh = ref(false)
  let refreshTimer: ReturnType<typeof setTimeout> | null = null

  const pageIndex = computed(() => Math.floor(catalog.value.offset / Math.max(catalog.value.limit, 1)) + 1)
  const pageCount = computed(() => Math.max(1, Math.ceil(catalog.value.total / Math.max(catalog.value.limit, 1))))
  const hasPreviousPage = computed(() => catalog.value.offset > 0)
  const hasNextPage = computed(() => catalog.value.offset + catalog.value.items.length < catalog.value.total)

  function toUnixStart(value: string): number {
    if (!value) return 0
    const time = new Date(`${value}T00:00:00`).getTime()
    return Number.isFinite(time) ? Math.floor(time / 1000) : 0
  }

  function toUnixEnd(value: string): number {
    if (!value) return 0
    const time = new Date(`${value}T23:59:59`).getTime()
    return Number.isFinite(time) ? Math.floor(time / 1000) : 0
  }

  function catalogRequest(offset = catalog.value.offset): GlobalLogCatalogRequest {
    return {
      query: filters.query.trim(),
      source: filters.source,
      kind: filters.kind,
      gameId: filters.gameId,
      instanceId: filters.instanceId.trim(),
      shard: filters.shard,
      status: filters.status,
      dateFrom: toUnixStart(filters.dateFrom),
      dateTo: toUnixEnd(filters.dateTo),
      offset,
      limit: Math.max(1, options.catalogPageSize() || 100),
    }
  }

  function catalogSignature(value: GlobalLogCatalogPage): string {
    return [
      value.total,
      value.offset,
      value.limit,
      value.summary.files,
      value.summary.lines,
      value.summary.bytes,
      value.summary.activeFiles,
      ...value.items.map(item => `${item.id}:${item.byteSize}:${item.lineCount}:${item.modifiedAt}:${item.status}`),
    ].join('|')
  }

  async function refreshCatalog(resetPage = false, refreshTail = false) {
    if (loadingCatalog.value) return false
    loadingCatalog.value = true
    error.value = ''
    try {
      const offset = resetPage ? 0 : catalog.value.offset
      const value = await backend.globalLogCatalog(catalogRequest(offset))
      if (catalogSignature(value) !== catalogSignature(catalog.value)) catalog.value = value

      if (selected.value) {
        const latest = value.items.find(item => item.id === selected.value?.id)
        if (latest) selected.value = latest
        else if (resetPage || value.total === 0) {
          selected.value = null
          lines.value = []
        }
      }
      if (!selected.value && value.items.length > 0) await selectLog(value.items[0], false)
      else if (refreshTail && selected.value?.active && direction.value === 'tail') await loadSelected('tail')
      return true
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e)
      return false
    } finally {
      loadingCatalog.value = false
    }
  }

  async function applyFilters() {
    catalog.value.offset = 0
    await refreshCatalog(true)
  }

  async function clearFilters() {
    Object.assign(filters, { query: '', source: 'all', kind: 'all', gameId: 'all', instanceId: '', shard: 'all', status: 'all', dateFrom: '', dateTo: '' })
    await applyFilters()
  }

  async function selectLog(item: GlobalLogFile, clearNotice = true) {
    selected.value = item
    if (clearNotice) notice.value = ''
    await loadSelected(direction.value)
  }

  async function loadSelected(mode: 'head' | 'tail' | 'next' = direction.value) {
    if (!selected.value || loadingContent.value) return false
    loadingContent.value = true
    error.value = ''
    try {
      const continuation = mode === 'next'
      const value = await backend.readGlobalLog({
        id: selected.value.id,
        direction: mode,
        cursor: continuation ? nextCursor.value : 0,
        startLine: continuation ? nextLine.value : 1,
        limit: Math.max(1, options.readPageSize() || 600),
        query: contentFilters.query.trim(),
        level: contentFilters.level,
        category: contentFilters.category,
      })
      selected.value = value.file
      lines.value = continuation ? [...lines.value, ...value.lines] : value.lines
      nextCursor.value = value.nextCursor
      nextLine.value = value.nextLine
      eof.value = value.eof
      scanned.value = value.scanned
      matched.value = value.matched
      readDurationMs.value = value.durationMs
      return true
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e)
      return false
    } finally {
      loadingContent.value = false
    }
  }

  async function setDirection(value: 'head' | 'tail') {
    direction.value = value
    await loadSelected(value)
  }

  async function searchContent() {
    await loadSelected(direction.value)
  }

  async function previousPage() {
    if (!hasPreviousPage.value) return
    catalog.value.offset = Math.max(0, catalog.value.offset - Math.max(1, options.catalogPageSize() || 100))
    selected.value = null
    lines.value = []
    await refreshCatalog(false)
  }

  async function nextPage() {
    if (!hasNextPage.value) return
    catalog.value.offset += Math.max(1, options.catalogPageSize() || 100)
    selected.value = null
    lines.value = []
    await refreshCatalog(false)
  }

  async function exportSelected(): Promise<GlobalLogExportResult | null> {
    if (!selected.value) return null
    try {
      const value = await backend.exportGlobalLog(selected.value.id)
      notice.value = `日志已保存到 ${value.path}`
      return value
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e)
      return null
    }
  }

  async function exportFiltered(): Promise<GlobalLogExportResult | null> {
    try {
      const value = await backend.exportGlobalLogs(catalogRequest(0))
      notice.value = `筛选日志包已保存到 ${value.path}`
      return value
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e)
      return null
    }
  }

  async function deleteSelected(): Promise<GlobalLogMutationResult | null> {
    if (!selected.value) return null
    try {
      const value = await backend.deleteGlobalLog(selected.value.id)
      notice.value = `已删除 ${value.deleted} 个日志，释放 ${formatBytes(value.bytesFreed)}`
      selected.value = null
      lines.value = []
      await refreshCatalog(true)
      return value
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e)
      return null
    }
  }

  async function deleteFiltered(): Promise<GlobalLogMutationResult | null> {
    try {
      const value = await backend.deleteGlobalLogs({ filter: catalogRequest(0) })
      notice.value = `已删除 ${value.deleted} 个日志，跳过 ${value.skipped} 个活动日志，释放 ${formatBytes(value.bytesFreed)}`
      selected.value = null
      lines.value = []
      await refreshCatalog(true)
      return value
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e)
      return null
    }
  }

  async function clearHistory(): Promise<GlobalLogMutationResult | null> {
    try {
      const value = await backend.clearGlobalLogHistory()
      notice.value = `历史日志清理完成：删除 ${value.deleted} 个，跳过 ${value.skipped} 个活动日志。`
      selected.value = null
      lines.value = []
      await refreshCatalog(true)
      return value
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e)
      return null
    }
  }

  async function openFolder() {
    try {
      await backend.openGlobalLogFolder()
      notice.value = '已打开 AI Game Manager Panel 日志目录。手动删除文件后点击“刷新”即可同步真实数量。'
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e)
    }
  }

  function stopAutoRefresh() {
    if (refreshTimer) clearTimeout(refreshTimer)
    refreshTimer = null
  }

  function scheduleAutoRefresh() {
    stopAutoRefresh()
    if (!autoRefresh.value) return
    const delay = Math.max(2, options.autoRefreshSeconds() || 5) * 1000
    refreshTimer = setTimeout(async () => {
      if (!document.hidden) await refreshCatalog(false, true)
      scheduleAutoRefresh()
    }, delay)
  }

  function setAutoRefresh(value: boolean) {
    autoRefresh.value = value
    scheduleAutoRefresh()
  }

  onBeforeUnmount(stopAutoRefresh)

  return {
    filters,
    contentFilters,
    catalog,
    selected,
    lines,
    direction,
    eof,
    scanned,
    matched,
    readDurationMs,
    loadingCatalog,
    loadingContent,
    error,
    notice,
    autoRefresh,
    pageIndex,
    pageCount,
    hasPreviousPage,
    hasNextPage,
    refreshCatalog,
    applyFilters,
    clearFilters,
    selectLog,
    loadSelected,
    setDirection,
    searchContent,
    previousPage,
    nextPage,
    exportSelected,
    exportFiltered,
    deleteSelected,
    deleteFiltered,
    clearHistory,
    openFolder,
    setAutoRefresh,
  }
}

export function formatBytes(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let size = value
  let unit = 0
  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024
    unit++
  }
  return `${size >= 10 || unit === 0 ? size.toFixed(0) : size.toFixed(1)} ${units[unit]}`
}

export function formatLogTime(value: number): string {
  if (!value) return '—'
  return new Date(value * 1000).toLocaleString('zh-CN', { hour12: false })
}
