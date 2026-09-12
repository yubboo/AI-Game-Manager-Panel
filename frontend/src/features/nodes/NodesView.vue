<script setup lang="ts">
import EmptyState from '../../shared/components/EmptyState.vue'
import StatusPill from '../../shared/components/StatusPill.vue'
import { useAppStore } from '../../shared/store/app'

const app = useAppStore()
</script>

<template>
  <section class="page">
    <div class="page-heading"><div><span class="eyebrow">AI GAME MANAGER NODES</span><h1>节点</h1><p>未来用一套界面管理本机 Windows、远程 Windows、Linux VPS 与 NAS。</p></div></div>
    <article class="panel node-card">
      <div class="node-card__main"><div class="node-card__icon">PC</div><div><strong>本机 Windows</strong><small>{{ app.info?.platform ?? '等待环境信息' }}</small></div></div>
      <StatusPill :tone="app.backendReady ? 'success' : 'warning'" :label="app.backendReady ? '在线' : '检测中'" />
    </article>
    <EmptyState icon="nodes" title="远程节点尚未启用" description="Agent、远程 Windows 和 Linux 节点会在核心执行层稳定后接入；游戏模块不会直接绑定某个操作系统。" />
  </section>
</template>
