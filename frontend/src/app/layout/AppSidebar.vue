<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import AppIcon from '../../shared/components/AppIcon.vue'
import { useAppStore } from '../../shared/store/app'
import type { AppIconName } from '../../shared/types/ui'

const app = useAppStore()

const groups = computed(() => app.platformConfig?.ui.navigation ?? [])

function iconName(value: string): AppIconName {
  return value as AppIconName
}
</script>

<template>
  <aside class="sidebar codex-sidebar">
    <div class="brand codex-brand">
      <div class="brand-mark codex-brand__mark">鱼</div>
      <div class="brand-copy">
        <div><strong>AI游戏管理器面板</strong></div>
        <small>小鱼 · 游戏服务器智能伙伴</small>
      </div>
      <button class="sidebar-collapse" type="button" title="隐藏左侧栏" aria-label="隐藏左侧栏" @click="app.setLeftSidebarOpen(false)">«</button>
    </div>

    <div class="sidebar-scroll codex-nav-scroll">
      <template v-if="groups.length">
        <div v-for="group in groups" :key="group.id" class="nav-group codex-nav-group">
          <div v-if="group.title" class="nav-group__title">{{ group.title }}</div>
          <nav class="nav-list">
            <RouterLink v-for="item in group.items" :key="item.id" :to="item.to" class="nav-item codex-nav-item">
              <AppIcon :name="iconName(item.icon)" />
              <span>{{ item.label }}</span>
            </RouterLink>
          </nav>
        </div>
      </template>
      <div v-else class="sidebar-loading">正在载入 configs/ui.json…</div>
    </div>

    <div class="codex-project-card">
      <span>当前项目</span>
      <strong><i :class="{ online: app.backendReady }"></i>本机游戏服务器</strong>
      <small>0.3.1 · Minecraft Web Build Hotfix</small>
    </div>

    <div class="sidebar-footer codex-sidebar-footer">
      <span><i :class="{ online: app.backendReady }"></i>{{ app.backendReady ? 'Core 正常' : 'Core 未连接' }}</span>
      <span>v{{ app.info?.version ?? '0.3.1' }}</span>
    </div>
  </aside>
</template>
