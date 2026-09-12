<script setup lang="ts">
import { computed, onMounted } from 'vue'
import EmptyState from '../../shared/components/EmptyState.vue'
import StatusPill from '../../shared/components/StatusPill.vue'
import { useAppStore } from '../../shared/store/app'

const app = useAppStore()

const nodeTitle = computed(() => {
  const runtime = app.platformRuntime
  if (!runtime) return '当前原生节点'
  const os = runtime.hostOs === 'windows' ? 'Windows' : runtime.hostOs === 'darwin' ? 'macOS' : runtime.hostOs === 'linux' ? 'Linux' : runtime.hostOs
  return `${os} 原生节点`
})

const runtimeLabel = computed(() => {
  const runtime = app.platformRuntime
  if (!runtime) return app.info?.platform ?? '等待运行端信息'
  return `${runtime.nodeKind} · ${runtime.hostArch}`
})

const controlSurfaces = computed(() => app.platformRuntime?.surfaces.filter(item => item.controlsNodes) ?? [])

onMounted(() => {
  if (!app.platformRuntime) void app.loadPlatformRuntime()
})
</script>

<template>
  <section class="page">
    <div class="page-heading">
      <div>
        <span class="eyebrow">AI GAME MANAGER NODES</span>
        <h1>节点</h1>
        <p>控制端和执行端分离：浏览器与桌面端负责管理；游戏、文件和进程始终在目标操作系统的原生 AGMP Runtime 上执行。</p>
      </div>
    </div>

    <article class="panel node-card">
      <div class="node-card__main">
        <div class="node-card__icon">{{ app.platformRuntime?.hostOs === 'linux' ? 'LX' : app.platformRuntime?.hostOs === 'darwin' ? 'MAC' : 'PC' }}</div>
        <div>
          <strong>{{ nodeTitle }}</strong>
          <small>{{ runtimeLabel }}</small>
        </div>
      </div>
      <StatusPill :tone="app.backendReady ? 'success' : 'warning'" :label="app.backendReady ? '在线 · 原生执行' : '检测中'" />
    </article>

    <article v-if="controlSurfaces.length" class="panel">
      <div class="page-heading page-heading--compact">
        <div><span class="eyebrow">CONTROL SURFACES</span><h2>当前平台合同</h2><p>Web 不是桌面程序，Windows/macOS Desktop 也不会模拟 Linux Runtime。</p></div>
      </div>
      <div class="settings-grid">
        <div v-for="surface in controlSurfaces" :key="surface.id" class="settings-card">
          <strong>{{ surface.name }}</strong>
          <small>{{ surface.kind }} · {{ surface.state === 'supported' ? '已接入' : '规划中' }} · {{ surface.browser ? '浏览器访问' : '原生程序' }}</small>
          <p>{{ surface.executesOnNode ? '可在本机原生 Runtime 执行' : '仅控制节点，不在客户端执行游戏进程' }}</p>
        </div>
      </div>
    </article>

    <EmptyState icon="nodes" title="远程节点注册尚未启用" description="后续远程 Windows、macOS、Linux 与 NAS 节点会注册到同一个控制面；远程控制只改变目标节点，不改变它的原生执行系统。" />
  </section>
</template>
