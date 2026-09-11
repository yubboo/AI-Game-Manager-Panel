<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppIcon from '../../shared/components/AppIcon.vue'
import { useAppStore } from '../../shared/store/app'

const route = useRoute()
const router = useRouter()
const app = useAppStore()
const title = computed(() => String(route.meta.title ?? 'AI游戏管理器面板'))
const group = computed(() => String(route.meta.group ?? '本机'))
function openUpdater() { void router.push({ path: '/settings', query: { section: 'updates' } }) }
</script>

<template>
  <header class="topbar codex-topbar">
    <div class="topbar__leading">
      <button v-if="!app.workbenchLayout.leftOpen" class="pane-icon-button" type="button" title="显示左侧栏" aria-label="显示左侧栏" @click="app.setLeftSidebarOpen(true)">☰</button>
      <div class="codex-breadcrumb"><span>{{ group }}</span><b>/</b><strong>{{ title }}</strong></div>
    </div>
    <div class="topbar__actions codex-topbar__actions">
      <button v-if="app.update?.updateAvailable" class="btn btn--primary topbar-update-button" type="button" @click="openUpdater">
        新版本 v{{ app.update.latestVersion }}
      </button>
      <button class="pane-icon-button" type="button" :class="{ active: app.workbenchLayout.bottomOpen }" title="显示/隐藏底部终端" aria-label="显示或隐藏底部终端" @click="app.toggleBottomPanel"><AppIcon name="terminal" /></button>
      <button class="pane-icon-button" type="button" :class="{ active: app.workbenchLayout.rightOpen }" title="显示/隐藏右侧栏" aria-label="显示或隐藏右侧栏" @click="app.toggleRightSidebar">▥</button>
      <span class="core-dot" :class="{ online: app.backendReady }"></span>
      <span>{{ app.backendReady ? 'Core 就绪' : 'Core 未连接' }}</span>
    </div>
  </header>
</template>
