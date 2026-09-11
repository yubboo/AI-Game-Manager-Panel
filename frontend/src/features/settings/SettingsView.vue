<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import AppIcon from '../../shared/components/AppIcon.vue'
import { backend } from '../../shared/api/backend'
import { useAppStore } from '../../shared/store/app'
import type { AGMPSettings } from '../../shared/types/backend'
import LicenseSection from './LicenseSection.vue'
import EnvironmentStorageSection from './EnvironmentStorageSection.vue'
import UpdateSection from './UpdateSection.vue'
import ModelManagementSection from './ModelManagementSection.vue'
import XiaoYuIntelligenceSection from './XiaoYuIntelligenceSection.vue'
import EmailServiceSection from './EmailServiceSection.vue'

const app = useAppStore()
const route = useRoute()
const form = reactive<AGMPSettings>({ theme: 'dark', language: 'zh-CN', debug: false })
const message = ref('')
const saving = ref(false)

type SectionID = 'general' | 'models' | 'intelligence' | 'email' | 'license' | 'environment' | 'updates' | 'diagnostics'
const validSections = new Set<SectionID>(['general', 'models', 'intelligence', 'email', 'license', 'environment', 'updates', 'diagnostics'])

const activeSection = computed<SectionID>(() => {
  const value = String(route.query.section || 'general') as SectionID
  return validSections.has(value) ? value : 'general'
})

onMounted(async () => {
  try {
    Object.assign(form, await backend.settings())
    app.applyTheme(form.theme)
  } catch (e) {
    message.value = e instanceof Error ? e.message : String(e)
  }
})

watch(() => form.theme, theme => app.applyTheme(theme))

async function save() {
  message.value = ''
  saving.value = true
  try {
    await backend.saveSettings({ ...form })
    if (app.settings) Object.assign(app.settings, form)
    else app.settings = { ...form }
    app.applyTheme(form.theme)
    message.value = '设置已保存并全局应用。'
  } catch (e) {
    message.value = e instanceof Error ? e.message : String(e)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <section class="page settings-page page--wide">
    <div class="page-heading settings-page-heading">
      <div><span class="eyebrow">AI GAME MANAGER PREFERENCES</span><h1>设置中心</h1><p>统一管理 AI Game Manager Panel 的全局主题、许可证、运行环境与存储路径。游戏专属参数仍留在各自实例中。</p></div>
    </div>

    <div class="settings-layout">
      <article class="panel settings-card">
        <section v-if="activeSection === 'general'" class="settings-section">
          <div class="settings-section__intro"><div class="settings-section__icon"><AppIcon name="settings" /></div><div><strong>外观与语言</strong><span>主题是全局状态源，Sidebar、Topbar、页面、卡片、表单、弹窗和状态区域必须同步变化。</span></div></div>
          <div class="feature-grid feature-grid--two">
            <label class="feature-card field-block">
              <span class="feature-title">主题</span>
              <small>“跟随系统”会实时监听 Windows / Linux 桌面的浅色与深色变化，不需要刷新页面。</small>
              <select v-model="form.theme"><option value="dark">AI游戏管理器面板深色</option><option value="light">明亮模式</option><option value="system">跟随系统</option></select>
            </label>
            <label class="feature-card field-block">
              <span class="feature-title">语言</span>
              <small>当前版本统一提供简体中文；后续国际化仍由同一设置源管理。</small>
              <select v-model="form.language"><option value="zh-CN">简体中文</option></select>
            </label>
          </div>
          <div class="theme-preview-grid">
            <div class="feature-card"><span class="feature-title">当前主题策略</span><p class="feature-description">保存值：{{ form.theme }}；当前解析结果：{{ app.resolvedTheme }}。</p><strong>{{ app.resolvedTheme === 'light' ? '浅色界面' : '深色界面' }}</strong></div>
            <div class="feature-card"><span class="feature-title">三端一致</span><p class="feature-description">Wails、Electron 与 Web 都读取同一 AGMPSettings，并使用统一 Design Token。</p><strong>全局 Theme Source</strong></div>
          </div>
          <div class="settings-footer"><span class="save-message" :class="{ active: message }">{{ message || '选择主题后立即预览，点击保存后持久化。' }}</span><button class="btn btn--primary" :disabled="saving" @click="save"><AppIcon name="check" />{{ saving ? '保存中…' : '保存设置' }}</button></div>
        </section>

        <ModelManagementSection v-else-if="activeSection === 'models'" />
        <XiaoYuIntelligenceSection v-else-if="activeSection === 'intelligence'" />
        <EmailServiceSection v-else-if="activeSection === 'email'" />
        <LicenseSection v-else-if="activeSection === 'license'" />
        <EnvironmentStorageSection v-else-if="activeSection === 'environment'" />
        <UpdateSection v-else-if="activeSection === 'updates'" />

        <section v-else class="settings-section">
          <div class="settings-section__intro"><div class="settings-section__icon"><AppIcon name="logs" /></div><div><strong>开发诊断</strong><span>调试选项只影响 AI Game Manager Panel 自身诊断信息，不会自动修改任何游戏服务器。</span></div></div>
          <label class="feature-card feature-card--toggle"><div><span class="feature-title">调试模式</span><p class="feature-description">保留更多用于开发阶段定位问题的信息。生产环境建议关闭。</p></div><input v-model="form.debug" class="switch" type="checkbox" /></label>
          <div class="settings-footer"><span class="save-message" :class="{ active: message }">{{ message || '调试状态属于 AI Game Manager Panel 全局设置。' }}</span><button class="btn btn--primary" :disabled="saving" @click="save"><AppIcon name="check" />{{ saving ? '保存中…' : '保存设置' }}</button></div>
        </section>
      </article>
    </div>
  </section>
</template>
