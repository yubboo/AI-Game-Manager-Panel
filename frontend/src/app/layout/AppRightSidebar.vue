<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppIcon from '../../shared/components/AppIcon.vue'
import { useAppStore } from '../../shared/store/app'
import XiaoYuRightContext from '../../features/xiaoyu/XiaoYuRightContext.vue'

const app = useAppStore()
const route = useRoute()
const router = useRouter()

const title = computed(() => String(route.meta.title ?? '工作区'))
const group = computed(() => String(route.meta.group ?? '本机'))
const isSettings = computed(() => route.path === '/settings')
const isXiaoYu = computed(() => route.path === '/')
const activeSettingsSection = computed(() => String(route.query.section || 'general'))
const settingsSections = [
  { id: 'general', label: '外观与语言', description: '主题、语言和界面偏好' },
  { id: 'models', label: '模型管理', description: '配置小鱼的大模型大脑' },
  { id: 'intelligence', label: '记忆、技能与专家', description: 'Memory / Skills / Experts' },
  { id: 'email', label: '邮箱服务', description: 'SMTP、找回密码与风险验证' },
  { id: 'license', label: '授权与许可证', description: 'License、机器码与安装 ID' },
  { id: 'environment', label: '运行环境与存储', description: 'SteamCMD、游戏库与路径' },
  { id: 'updates', label: '更新与升级', description: '检查版本与下载安装器' },
  { id: 'diagnostics', label: '开发诊断', description: '调试与开发状态' },
]
const licenseText = computed(() => {
  if (!app.license) return '未读取'
  if (app.license.valid) return app.license.edition ? `${app.license.edition} · 已授权` : '已授权'
  return '未授权'
})

function openSettings(section?: string) {
  void router.push({ path: '/settings', query: section && section !== 'general' ? { section } : undefined })
}
</script>

<template>
  <aside class="right-sidebar" aria-label="上下文与二级菜单">
    <header class="right-sidebar__header">
      <div><span class="eyebrow">CONTEXT</span><strong>{{ isSettings ? '设置分类' : (isXiaoYu ? '小鱼上下文' : '辅助面板') }}</strong></div>
      <button class="pane-icon-button" type="button" title="隐藏右侧栏" aria-label="隐藏右侧栏" @click="app.setRightSidebarOpen(false)">×</button>
    </header>

    <div class="right-sidebar__scroll">
      <template v-if="isSettings">
        <section class="right-sidebar__section right-sidebar__section--compact">
          <span class="right-sidebar__label">设置中心</span>
          <strong>{{ settingsSections.find(item => item.id === activeSettingsSection)?.label || '外观与语言' }}</strong>
          <small>二级菜单固定在右侧，不再挤占中央主内容区。</small>
        </section>
        <nav class="context-nav" aria-label="设置二级菜单">
          <button v-for="section in settingsSections" :key="section.id" type="button" :class="['context-nav__item', { active: activeSettingsSection === section.id }]" :aria-current="activeSettingsSection === section.id ? 'page' : undefined" @click="openSettings(section.id)">
            <strong>{{ section.label }}</strong><span>{{ section.description }}</span>
          </button>
        </nav>
        <section class="right-sidebar__section right-sidebar__status-grid">
          <div><span>Go Core</span><strong :class="app.backendReady ? 'status-success' : 'status-warning'">{{ app.backendReady ? '已连接' : '未连接' }}</strong></div>
          <div><span>许可证</span><strong :class="app.license?.valid ? 'status-success' : 'status-warning'">{{ licenseText }}</strong></div>
        </section>
      </template>

      <XiaoYuRightContext v-else-if="isXiaoYu" />

      <template v-else>
        <section class="right-sidebar__section right-sidebar__section--compact"><span class="right-sidebar__label">当前页面</span><strong>{{ title }}</strong><small>{{ group }} · {{ app.backendAdapter }}</small></section>
        <section class="right-sidebar__section right-sidebar__status-grid">
          <div><span>Go Core</span><strong :class="app.backendReady ? 'status-success' : 'status-warning'">{{ app.backendReady ? '已连接' : '未连接' }}</strong></div>
          <div><span>许可证</span><strong :class="app.license?.valid ? 'status-success' : 'status-warning'">{{ licenseText }}</strong></div>
        </section>
        <section class="right-sidebar__section">
          <span class="right-sidebar__label">相关操作</span>
          <button class="right-sidebar__action" type="button" @click="openSettings('environment')"><AppIcon name="environment" /><span><strong>运行环境与存储</strong><small>SteamCMD、游戏库与路径</small></span></button>
          <button class="right-sidebar__action" type="button" @click="app.setBottomPanelOpen(true)"><AppIcon name="terminal" /><span><strong>底部终端</strong><small>终端 / 日志 / 任务输出</small></span></button>
        </section>
      </template>
    </div>
  </aside>
</template>
