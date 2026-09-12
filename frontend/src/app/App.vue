<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import AppSidebar from './layout/AppSidebar.vue'
import AppRightSidebar from './layout/AppRightSidebar.vue'
import AppTopbar from './layout/AppTopbar.vue'
import BottomPanel from './layout/BottomPanel.vue'
import AuthGate from '../features/auth/AuthGate.vue'
import LicenseFeatureGate from '../shared/components/LicenseFeatureGate.vue'
import { useAppStore } from '../shared/store/app'

const app = useAppStore()
const route = useRoute()
const ready = ref(false)
const dragKind = ref<'left' | 'right' | 'bottom' | null>(null)
const dragStartX = ref(0)
const dragStartY = ref(0)
const dragStartSize = ref(0)
const paneSnapAnimating = ref(false)
let paneSnapTimer: number | null = null
let dragPointerId: number | null = null
let dragHandle: HTMLElement | null = null
const requiredLicenseFeature = computed(() => String(route.meta.licenseFeature ?? ''))
const featureAllowed = computed(() => !requiredLicenseFeature.value || app.hasLicenseFeature(requiredLicenseFeature.value))

const workbenchStyle = computed(() => ({
  '--left-pane-width': `${app.workbenchLayout.leftWidth}px`,
  '--right-pane-width': `${app.workbenchLayout.rightWidth}px`,
  '--bottom-pane-height': `${app.workbenchLayout.bottomHeight}px`,
  // Keep a stable five-track grid so collapse/expand can animate instead of destroying tracks.
  gridTemplateColumns: [
    app.workbenchLayout.leftOpen ? `${app.workbenchLayout.leftWidth}px` : '0px',
    app.workbenchLayout.leftOpen ? 'var(--pane-resizer-size)' : '0px',
    'minmax(0,1fr)',
    app.workbenchLayout.rightOpen ? 'var(--pane-resizer-size)' : '0px',
    app.workbenchLayout.rightOpen ? `${app.workbenchLayout.rightWidth}px` : '0px',
  ].join(' '),
}))

function triggerPaneSnapAnimation() {
  paneSnapAnimating.value = true
  if (paneSnapTimer !== null) window.clearTimeout(paneSnapTimer)
  paneSnapTimer = window.setTimeout(() => {
    paneSnapAnimating.value = false
    paneSnapTimer = null
  }, 210)
}

function enterApp() {
  // SHELL_FIRST_HYDRATION：会话恢复成功后立即恢复三栏外壳，再后台补齐非认证数据。
  // 刷新不再让 AuthGate 等待 Info / Config / Settings / License 四个请求后才切回主界面。
  app.normalizeWorkbenchLayout()
  if (window.innerWidth < 980) app.setRightSidebarOpen(false)
  ready.value = true
  void hydrateApp()
}

async function hydrateApp() {
  await Promise.allSettled([app.loadInfo(), app.loadPlatformRuntime(), app.loadPlatformConfig(), app.loadGamePacks(), app.loadGameInstances(), app.loadSettings(), app.loadLicense()])
  app.normalizeWorkbenchLayout()
  if (app.platformConfig?.update?.checkOnStartup !== false) void app.checkForUpdates(false)
}

function authExpired() { ready.value = false }

function beginResize(kind: 'left' | 'right' | 'bottom', event: PointerEvent) {
  event.preventDefault()
  endResize()
  dragKind.value = kind
  dragPointerId = event.pointerId
  dragHandle = event.currentTarget as HTMLElement
  try { dragHandle.setPointerCapture(event.pointerId) } catch { /* window listeners remain the fallback */ }
  dragStartX.value = event.clientX
  dragStartY.value = event.clientY
  dragStartSize.value = kind === 'left' ? app.workbenchLayout.leftWidth : kind === 'right' ? app.workbenchLayout.rightWidth : app.workbenchLayout.bottomHeight
  document.body.classList.add(kind === 'bottom' ? 'is-resizing-row' : 'is-resizing-column')
  window.addEventListener('pointermove', resizePane, { passive: false })
  window.addEventListener('pointerup', endResize)
  window.addEventListener('pointercancel', endResize)
}

function resizePane(event: PointerEvent) {
  if (dragPointerId !== null && event.pointerId !== dragPointerId) return
  event.preventDefault()
  if (dragKind.value === 'left') {
    const result = app.dragLeftSidebarWidth(dragStartSize.value + event.clientX - dragStartX.value)
    if (result === 'collapsed' || result === 'reopened') triggerPaneSnapAnimation()
  }
  if (dragKind.value === 'right') app.setRightSidebarWidth(dragStartSize.value - (event.clientX - dragStartX.value), false)
  if (dragKind.value === 'bottom') app.setBottomPanelHeight(dragStartSize.value - (event.clientY - dragStartY.value), false)
}

function endResize(event?: PointerEvent) {
  if (event && dragPointerId !== null && event.pointerId !== dragPointerId) return
  if (dragPointerId !== null && dragHandle) {
    try { if (dragHandle.hasPointerCapture(dragPointerId)) dragHandle.releasePointerCapture(dragPointerId) } catch { /* already released */ }
  }
  const hadDrag = dragKind.value !== null
  dragKind.value = null
  dragPointerId = null
  dragHandle = null
  document.body.classList.remove('is-resizing-column', 'is-resizing-row')
  window.removeEventListener('pointermove', resizePane)
  window.removeEventListener('pointerup', endResize)
  window.removeEventListener('pointercancel', endResize)
  if (hadDrag) app.commitWorkbenchLayout()
}

function handleViewportResize() {
  app.normalizeWorkbenchLayout()
  if (window.innerWidth < 980 && app.workbenchLayout.rightOpen) app.setRightSidebarOpen(false)
}

watch(() => route.query.panel, panel => { if (panel === 'terminal') app.setBottomPanelOpen(true) }, { immediate: true })

onMounted(() => {
  window.addEventListener('agmp-auth-expired', authExpired)
  window.addEventListener('resize', handleViewportResize)
})
onBeforeUnmount(() => {
  window.removeEventListener('agmp-auth-expired', authExpired)
  window.removeEventListener('resize', handleViewportResize)
  endResize()
  if (paneSnapTimer !== null) window.clearTimeout(paneSnapTimer)
  paneSnapTimer = null
  app.disposeThemeListener()
})
</script>

<template>
  <AuthGate v-if="!ready" @ready="enterApp" />
  <div
    v-else
    class="workbench-shell"
    :class="{ 'is-pane-snapping': paneSnapAnimating, 'is-left-collapsed': !app.workbenchLayout.leftOpen, 'is-right-collapsed': !app.workbenchLayout.rightOpen }"
    :style="workbenchStyle"
  >
    <AppSidebar :aria-hidden="!app.workbenchLayout.leftOpen" :inert="!app.workbenchLayout.leftOpen" />
    <button class="pane-resizer pane-resizer--left" type="button" aria-label="拖拽调整左侧栏宽度" :aria-hidden="!app.workbenchLayout.leftOpen" :tabindex="app.workbenchLayout.leftOpen ? 0 : -1" @pointerdown="app.workbenchLayout.leftOpen && beginResize('left', $event)"></button>

    <main class="workbench-main">
      <AppTopbar />
      <div class="content-scroll workbench-content">
        <LicenseFeatureGate v-if="requiredLicenseFeature && !featureAllowed" :feature="requiredLicenseFeature" />
        <RouterView v-else />
      </div>
      <button v-if="app.workbenchLayout.bottomOpen" class="pane-resizer pane-resizer--bottom" type="button" aria-label="拖拽调整底部面板高度" @pointerdown="beginResize('bottom', $event)"></button>
      <BottomPanel v-if="app.workbenchLayout.bottomOpen" />
    </main>

    <button class="pane-resizer pane-resizer--right" type="button" aria-label="拖拽调整右侧栏宽度" :aria-hidden="!app.workbenchLayout.rightOpen" :tabindex="app.workbenchLayout.rightOpen ? 0 : -1" @pointerdown="app.workbenchLayout.rightOpen && beginResize('right', $event)"></button>
    <AppRightSidebar :aria-hidden="!app.workbenchLayout.rightOpen" :inert="!app.workbenchLayout.rightOpen" />
  </div>
</template>
