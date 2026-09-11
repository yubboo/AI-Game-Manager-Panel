<script setup lang="ts">
import { ref } from 'vue'
import AppIcon from '../../shared/components/AppIcon.vue'
import TerminalPanel from '../../features/terminal/TerminalPanel.vue'
import { useAppStore } from '../../shared/store/app'

const app = useAppStore()
const active = ref<'terminal' | 'logs' | 'tasks'>('terminal')
</script>

<template>
  <section class="bottom-panel" aria-label="底部面板">
    <header class="bottom-panel__header">
      <nav class="bottom-panel__tabs" aria-label="底部面板标签">
        <button :class="{ active: active === 'terminal' }" type="button" @click="active = 'terminal'"><AppIcon name="terminal" />终端</button>
        <button :class="{ active: active === 'logs' }" type="button" @click="active = 'logs'"><AppIcon name="logs" />日志</button>
        <button :class="{ active: active === 'tasks' }" type="button" @click="active = 'tasks'"><AppIcon name="check" />任务输出</button>
      </nav>
      <button class="pane-icon-button" type="button" title="关闭底部面板" aria-label="关闭底部面板" @click="app.setBottomPanelOpen(false)">×</button>
    </header>

    <div v-if="active === 'terminal'" class="bottom-panel__body bottom-terminal">
      <TerminalPanel />
    </div>
    <div v-else-if="active === 'logs'" class="bottom-panel__body bottom-panel__placeholder"><strong>实时日志</strong><span>这里将承载当前工作区的实时输出；完整历史日志仍进入“日志中心”。</span></div>
    <div v-else class="bottom-panel__body bottom-panel__placeholder"><strong>任务输出</strong><span>后台安装、校验、备份、迁移等任务的实时输出将在这里统一展示。</span></div>
  </section>
</template>
