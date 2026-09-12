import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { backend } from '../api/backend'
import type { AppInfo, PlatformConfig, AGMPSettings, DSTDedicatedSnapshot, DSTEnvironment, DSTWorkspaceSnapshot, GameWorkspaceSnapshot, LicenseStatus, SteamAppInventory, SteamEnvironment, UpdateStatus } from '../types/backend'


let systemThemeQuery: MediaQueryList | null = null
let systemThemeListener: ((event: MediaQueryListEvent) => void) | null = null


const WORKBENCH_LAYOUT_KEY = 'agmp.workbench.layout.v6'
const WORKBENCH_INITIAL_LEFT_RATIO = 0.05
const WORKBENCH_INITIAL_CENTER_RATIO = 0.75
const WORKBENCH_INITIAL_RIGHT_RATIO = 0.20
const UI_THEME_CHOICE_KEY = 'agmp.ui.theme-choice'

type WorkbenchLayoutState = {
  leftOpen: boolean
  rightOpen: boolean
  bottomOpen: boolean
  leftWidth: number
  rightWidth: number
  bottomHeight: number
}

const defaultWorkbenchLayout: WorkbenchLayoutState = {
  leftOpen: true,
  rightOpen: true,
  bottomOpen: false,
  leftWidth: 0,
  rightWidth: 0,
  bottomHeight: 280,
}

function readWorkbenchLayout(): WorkbenchLayoutState {
  try {
    const raw = window.localStorage.getItem(WORKBENCH_LAYOUT_KEY)
    if (!raw) return { ...defaultWorkbenchLayout }
    const parsed = JSON.parse(raw) as Partial<WorkbenchLayoutState>
    return {
      leftOpen: parsed.leftOpen ?? true,
      rightOpen: parsed.rightOpen ?? true,
      bottomOpen: parsed.bottomOpen ?? false,
      leftWidth: Number.isFinite(parsed.leftWidth) ? Number(parsed.leftWidth) : 0,
      rightWidth: Number.isFinite(parsed.rightWidth) ? Number(parsed.rightWidth) : 0,
      bottomHeight: Number.isFinite(parsed.bottomHeight) ? Number(parsed.bottomHeight) : 280,
    }
  } catch {
    return { ...defaultWorkbenchLayout }
  }
}

function resolveTheme(theme: AGMPSettings['theme']): 'light' | 'dark' {
  if (theme === 'system') return window.matchMedia?.('(prefers-color-scheme: light)').matches ? 'light' : 'dark'
  return theme
}

function readPersistedThemeChoice(): AGMPSettings['theme'] {
  try {
    const value = window.localStorage?.getItem(UI_THEME_CHOICE_KEY)
    if (value === 'light' || value === 'dark' || value === 'system') return value
  } catch { /* hardened WebViews may disable localStorage */ }
  return 'dark'
}

export const useAppStore = defineStore('app', () => {
  const backendMode = backend.mode()
  const backendAdapter = backend.adapter()
  const info = ref<AppInfo | null>(null)
  const platformConfig = ref<PlatformConfig | null>(null)
  const platformConfigError = ref('')
  const settings = ref<AGMPSettings | null>(null)
  const resolvedTheme = ref<'light' | 'dark'>(resolveTheme(readPersistedThemeChoice()))
  const license = ref<LicenseStatus | null>(null)
  const licenseLoading = ref(false)
  const update = ref<UpdateStatus | null>(null)
  const updateLoading = ref(false)
  const updateError = ref('')
  const steam = ref<SteamEnvironment | null>(null)
  const steamApps = ref<SteamAppInventory | null>(null)
  const steamLoading = ref(false)
  const steamAppsLoading = ref(false)
  const steamError = ref('')
  const steamAppsError = ref('')
  const steamLastScannedAt = ref(0)
  const steamLastDurationMs = ref(0)
  const dstEnvironment = ref<DSTEnvironment | null>(null)
  const dstEnvironmentLoading = ref(false)
  const dstEnvironmentError = ref('')
  const dstDedicated = ref<DSTDedicatedSnapshot | null>(null)
  const dstDedicatedLoading = ref(false)
  const dstDedicatedError = ref('')
  const gameWorkspace = ref<GameWorkspaceSnapshot | null>(null)
  const gameWorkspaceLoading = ref(false)
  const gameWorkspaceError = ref('')
  const backendReady = ref(false)
  const backendMessage = ref('尚未检测')
  const workbenchLayout = ref<WorkbenchLayoutState>(readWorkbenchLayout())

  function persistWorkbenchLayout() {
    try { window.localStorage.setItem(WORKBENCH_LAYOUT_KEY, JSON.stringify(workbenchLayout.value)) } catch { /* 浏览器隐私模式下忽略持久化失败。 */ }
  }

  const LEFT_PANE_MIN = 220
  const LEFT_PANE_MAX = 520
  const RIGHT_PANE_MIN = 260
  const RIGHT_PANE_MAX = 760
  const PANE_RESIZER_BUDGET = 18

  function layoutBounds(viewportWidth = window.innerWidth) {
    const compact = viewportWidth < 1180
    const layoutWidth = Math.max(0, viewportWidth - PANE_RESIZER_BUDGET)
    const centerMin = compact ? Math.max(460, Math.round(layoutWidth * 0.48)) : Math.max(640, Math.round(layoutWidth * 0.42))
    const leftMin = compact ? 168 : LEFT_PANE_MIN
    const rightMin = compact ? 230 : RIGHT_PANE_MIN
    return { layoutWidth, centerMin, leftMin, rightMin }
  }

  function clamp(value: number, min: number, max: number) {
    return Math.min(Math.max(value, min), Math.max(min, max))
  }

  function normalizeWorkbenchLayout(viewportWidth = window.innerWidth) {
    // 0.1.95：5/75/20 只定义“首次目标比例”，不是持续强制比例。
    // 左右栏各自拥有独立 min/max；拖动任意一侧时绝不重算另一侧。
    // 只有窗口本身发生 resize 时，才在必要时收缩右侧来守住中央最小工作宽度。
    const { layoutWidth, centerMin, leftMin, rightMin } = layoutBounds(viewportWidth)
    const defaultLeft = clamp(Math.round(layoutWidth * WORKBENCH_INITIAL_LEFT_RATIO), leftMin, LEFT_PANE_MAX)
    const defaultRight = clamp(Math.round(layoutWidth * WORKBENCH_INITIAL_RIGHT_RATIO), rightMin, RIGHT_PANE_MAX)

    if (workbenchLayout.value.leftWidth <= 0) workbenchLayout.value.leftWidth = defaultLeft
    if (workbenchLayout.value.rightWidth <= 0) workbenchLayout.value.rightWidth = defaultRight

    workbenchLayout.value.leftWidth = clamp(workbenchLayout.value.leftWidth, leftMin, LEFT_PANE_MAX)
    workbenchLayout.value.rightWidth = clamp(workbenchLayout.value.rightWidth, rightMin, RIGHT_PANE_MAX)

    const left = workbenchLayout.value.leftOpen ? workbenchLayout.value.leftWidth : 0
    const right = workbenchLayout.value.rightOpen ? workbenchLayout.value.rightWidth : 0
    const overflow = left + right + centerMin - layoutWidth
    if (overflow > 0 && workbenchLayout.value.rightOpen) {
      workbenchLayout.value.rightWidth = clamp(workbenchLayout.value.rightWidth - overflow, rightMin, RIGHT_PANE_MAX)
    }

    workbenchLayout.value.bottomHeight = Math.min(Math.max(180, workbenchLayout.value.bottomHeight), Math.max(220, window.innerHeight * 0.62))
    persistWorkbenchLayout()
  }

  function setLeftSidebarOpen(value: boolean) { workbenchLayout.value.leftOpen = value; normalizeWorkbenchLayout() }
  function setRightSidebarOpen(value: boolean) { workbenchLayout.value.rightOpen = value; normalizeWorkbenchLayout() }
  function setBottomPanelOpen(value: boolean) { workbenchLayout.value.bottomOpen = value; persistWorkbenchLayout() }
  function toggleLeftSidebar() { setLeftSidebarOpen(!workbenchLayout.value.leftOpen) }
  function toggleRightSidebar() { setRightSidebarOpen(!workbenchLayout.value.rightOpen) }
  function toggleBottomPanel() { setBottomPanelOpen(!workbenchLayout.value.bottomOpen) }

  function leftSidebarDragBounds() {
    const { layoutWidth, centerMin, leftMin } = layoutBounds()
    const right = workbenchLayout.value.rightOpen ? workbenchLayout.value.rightWidth : 0
    const maxByCenter = layoutWidth - centerMin - right
    return {
      min: leftMin,
      max: Math.max(leftMin, Math.min(LEFT_PANE_MAX, maxByCenter)),
      // Codex-style snap: reach the hard minimum, then a tiny extra pull collapses the pane.
      collapseAt: Math.max(72, leftMin - 8),
      // Hysteresis avoids flicker while the pointer is hovering around the snap boundary.
      reopenAt: leftMin + 30,
    }
  }

  function setLeftSidebarWidth(value: number, persist = true) {
    const bounds = leftSidebarDragBounds()
    workbenchLayout.value.leftWidth = clamp(value, bounds.min, bounds.max)
    if (persist) persistWorkbenchLayout()
  }

  function dragLeftSidebarWidth(value: number) {
    const bounds = leftSidebarDragBounds()
    if (workbenchLayout.value.leftOpen && value <= bounds.collapseAt) {
      workbenchLayout.value.leftOpen = false
      return 'collapsed' as const
    }
    if (!workbenchLayout.value.leftOpen) {
      if (value < bounds.reopenAt) return 'held-collapsed' as const
      workbenchLayout.value.leftOpen = true
      workbenchLayout.value.leftWidth = clamp(value, bounds.min, bounds.max)
      return 'reopened' as const
    }
    workbenchLayout.value.leftWidth = clamp(value, bounds.min, bounds.max)
    return 'resized' as const
  }

  function setRightSidebarWidth(value: number, persist = true) {
    const { layoutWidth, centerMin, rightMin } = layoutBounds()
    const left = workbenchLayout.value.leftOpen ? workbenchLayout.value.leftWidth : 0
    const maxByCenter = layoutWidth - centerMin - left
    workbenchLayout.value.rightWidth = clamp(value, rightMin, Math.min(RIGHT_PANE_MAX, maxByCenter))
    if (persist) persistWorkbenchLayout()
  }

  function setBottomPanelHeight(value: number, persist = true) {
    workbenchLayout.value.bottomHeight = Math.min(Math.max(180, value), Math.max(220, window.innerHeight * 0.62))
    if (persist) persistWorkbenchLayout()
  }

  function commitWorkbenchLayout() { persistWorkbenchLayout() }


  function applyTheme(theme: AGMPSettings['theme']) {
    try { window.localStorage?.setItem(UI_THEME_CHOICE_KEY, theme) } catch { /* best effort */ }
    if (systemThemeQuery && systemThemeListener) systemThemeQuery.removeEventListener('change', systemThemeListener)
    systemThemeQuery = null
    systemThemeListener = null

    const updateResolvedTheme = () => {
      resolvedTheme.value = resolveTheme(theme)
      document.documentElement.dataset.themeChoice = theme
      document.documentElement.dataset.theme = resolvedTheme.value
      document.body.dataset.theme = resolvedTheme.value
      document.documentElement.classList.toggle('theme-light', resolvedTheme.value === 'light')
      document.documentElement.classList.toggle('theme-dark', resolvedTheme.value === 'dark')
      document.documentElement.style.colorScheme = resolvedTheme.value
    }

    updateResolvedTheme()
    if (theme === 'system' && window.matchMedia) {
      systemThemeQuery = window.matchMedia('(prefers-color-scheme: light)')
      systemThemeListener = () => updateResolvedTheme()
      systemThemeQuery.addEventListener('change', systemThemeListener)
    }
  }

  function disposeThemeListener() {
    if (systemThemeQuery && systemThemeListener) systemThemeQuery.removeEventListener('change', systemThemeListener)
    systemThemeQuery = null
    systemThemeListener = null
  }

  async function loadLicense() {
    if (licenseLoading.value) return false
    licenseLoading.value = true
    try {
      license.value = await backend.licenseStatus()
      return true
    } catch {
      return false
    } finally {
      licenseLoading.value = false
    }
  }

  const licensedFeatures = computed(() => new Set(license.value?.features ?? []))
  function hasLicenseFeature(feature: string) {
    if (!feature) return true
    if (!license.value?.valid) return false
    return licensedFeatures.value.has('*') || licensedFeatures.value.has(feature)
  }

  async function checkForUpdates(force = false) {
    if (updateLoading.value) return false
    updateLoading.value = true
    updateError.value = ''
    try {
      update.value = await backend.checkForUpdates(force)
      return true
    } catch (error) {
      updateError.value = error instanceof Error ? error.message : String(error)
      return false
    } finally {
      updateLoading.value = false
    }
  }

  async function loadInfo() {
    try {
      info.value = await backend.appInfo()
      backendReady.value = true
    } catch {
      backendReady.value = false
    }
  }


  async function loadPlatformConfig() {
    platformConfigError.value = ''
    try {
      platformConfig.value = await backend.platformConfig()
      const width = platformConfig.value.ui.sidebarWidth
      const compact = platformConfig.value.ui.compactSidebarWidth
      if (width > 0) document.documentElement.style.setProperty('--sidebar-width', `${width}px`)
      if (compact > 0) document.documentElement.style.setProperty('--sidebar-compact-width', `${compact}px`)
      return true
    } catch (error) {
      platformConfigError.value = error instanceof Error ? error.message : String(error)
      return false
    }
  }

  async function loadSettings() {
    try {
      settings.value = await backend.settings()
      applyTheme(settings.value.theme)
    } catch {
      // 刷新期间读取设置失败也沿用上次主题，禁止突然回落到 dark 造成整页闪屏。
      applyTheme(readPersistedThemeChoice())
    }
  }

  async function refreshSteam() {
    // Stale-while-revalidate: keep the last successful data rendered while a
    // fresh snapshot is collected. This avoids unmounting large parts of the
    // page and eliminates the visible WebView2 repaint/flicker.
    if (steamLoading.value) return false

    steamLoading.value = true
    steamAppsLoading.value = true
    steamError.value = ''
    steamAppsError.value = ''

    try {
      const snapshot = await backend.steamSnapshot()
      // Apply environment + app inventory together in the same microtask.
      steam.value = snapshot.environment
      steamApps.value = snapshot.inventory
      steamLastScannedAt.value = snapshot.scannedAt
      steamLastDurationMs.value = snapshot.durationMs
      return true
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error)
      // Preserve previous successful data on transient refresh errors.
      steamError.value = message
      steamAppsError.value = message
      return false
    } finally {
      steamLoading.value = false
      steamAppsLoading.value = false
    }
  }


  async function refreshDSTEnvironment() {
    if (dstEnvironmentLoading.value) return false
    dstEnvironmentLoading.value = true
    dstEnvironmentError.value = ''
    try {
      dstEnvironment.value = await backend.dstEnvironment()
      return true
    } catch (error) {
      dstEnvironmentError.value = error instanceof Error ? error.message : String(error)
      return false
    } finally {
      dstEnvironmentLoading.value = false
    }
  }

  async function refreshDSTDedicated() {
    if (dstDedicatedLoading.value) return false
    dstDedicatedLoading.value = true
    dstDedicatedError.value = ''
    try {
      dstDedicated.value = await backend.dstDedicated()
      return true
    } catch (error) {
      dstDedicatedError.value = error instanceof Error ? error.message : String(error)
      return false
    } finally {
      dstDedicatedLoading.value = false
    }
  }

  async function refreshDSTWorkspace() {
    if (gameWorkspaceLoading.value || dstDedicatedLoading.value) return false
    gameWorkspaceLoading.value = true
    dstDedicatedLoading.value = true
    gameWorkspaceError.value = ''
    dstDedicatedError.value = ''
    try {
      const snapshot: DSTWorkspaceSnapshot = await backend.dstWorkspace()
      applyGameWorkspaceSnapshot(snapshot.workspace)
      dstDedicated.value = snapshot.dedicated
      dstEnvironment.value = snapshot.environment
      return true
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error)
      gameWorkspaceError.value = message
      dstDedicatedError.value = message
      return false
    } finally {
      gameWorkspaceLoading.value = false
      dstDedicatedLoading.value = false
    }
  }

  function applyGameWorkspaceSnapshot(snapshot: GameWorkspaceSnapshot) {
    gameWorkspace.value = snapshot
    steam.value = snapshot.steam
    steamLastScannedAt.value = snapshot.scannedAt
    steamLastDurationMs.value = snapshot.durationMs
  }

  async function refreshGameWorkspace(id: string) {
    if (gameWorkspaceLoading.value) return false
    gameWorkspaceLoading.value = true
    gameWorkspaceError.value = ''
    try {
      const snapshot = await backend.gameWorkspace(id)
      applyGameWorkspaceSnapshot(snapshot)
      return true
    } catch (error) {
      gameWorkspaceError.value = error instanceof Error ? error.message : String(error)
      return false
    } finally {
      gameWorkspaceLoading.value = false
    }
  }

  async function ping() {
    try {
      backendMessage.value = await backend.ping()
      backendReady.value = true
    } catch (error) {
      backendMessage.value = error instanceof Error ? error.message : String(error)
      backendReady.value = false
    }
  }

  return {
    backendMode,
    backendAdapter,
    info,
    platformConfig,
    platformConfigError,
    settings,
    resolvedTheme,
    license,
    licenseLoading,
    update,
    updateLoading,
    updateError,
    steam,
    steamApps,
    steamLoading,
    steamAppsLoading,
    steamError,
    steamAppsError,
    steamLastScannedAt,
    steamLastDurationMs,
    dstEnvironment,
    dstEnvironmentLoading,
    dstEnvironmentError,
    dstDedicated,
    dstDedicatedLoading,
    dstDedicatedError,
    gameWorkspace,
    gameWorkspaceLoading,
    gameWorkspaceError,
    backendReady,
    backendMessage,
    workbenchLayout,
    normalizeWorkbenchLayout,
    setLeftSidebarOpen,
    setRightSidebarOpen,
    setBottomPanelOpen,
    toggleLeftSidebar,
    toggleRightSidebar,
    toggleBottomPanel,
    setLeftSidebarWidth,
    dragLeftSidebarWidth,
    setRightSidebarWidth,
    setBottomPanelHeight,
    commitWorkbenchLayout,
    loadInfo,
    loadPlatformConfig,
    loadSettings,
    loadLicense,
    checkForUpdates,
    refreshSteam,
    refreshDSTEnvironment,
    refreshDSTDedicated,
    refreshDSTWorkspace,
    refreshGameWorkspace,
    applyTheme,
    disposeThemeListener,
    hasLicenseFeature,
    ping,
  }
})
