<script setup lang="ts">
import { computed, onMounted } from 'vue'
import EmptyState from '../../shared/components/EmptyState.vue'
import StatusPill from '../../shared/components/StatusPill.vue'
import { useAppStore } from '../../shared/store/app'
import MinecraftInstance from '../../games/minecraft/MinecraftInstance.vue'

const app = useAppStore()

const gameNames = computed(() => new Map((app.gamePacks.length ? app.gamePacks : (app.platformConfig?.games.templates ?? [])).map(item => [item.id, item.nameZh])))

function stateTone(state: string) {
  const value = state.toLowerCase()
  if (value.includes('running') || value.includes('healthy')) return 'success' as const
  if (value.includes('starting') || value.includes('unknown')) return 'warning' as const
  if (value.includes('failed') || value.includes('error')) return 'danger' as const
  return 'muted' as const
}

onMounted(() => {
  void app.loadGameInstances()
})
</script>

<template>
  <section class="page">
    <div class="page-heading">
      <div>
        <span class="eyebrow">GAME INSTANCES</span>
        <h1>服务器</h1>
        <p>无论从游戏库点击部署，还是让 XiaoYu 一句话开服，最终都进入同一个 GameInstance 资源模型并在这里长期管理。</p>
      </div>
      <button class="btn" :disabled="app.gameInstancesLoading" @click="app.loadGameInstances()">{{ app.gameInstancesLoading ? '刷新中…' : '刷新实例' }}</button>
    </div>

    <div v-if="app.gameInstancesError" class="inline-alert inline-alert--danger">{{ app.gameInstancesError }}</div>

    <div v-if="app.gameInstances.length" class="settings-grid">
      <article v-for="instance in app.gameInstances" :key="instance.id" class="panel settings-card">
        <div class="settings-card__heading">
          <div>
            <span class="eyebrow">{{ gameNames.get(instance.gameId) ?? instance.gameId }}</span>
            <h2>{{ instance.name }}</h2>
          </div>
          <StatusPill :tone="stateTone(instance.runtimeState)" :label="instance.runtimeState || 'unknown'" />
        </div>
        <p>{{ instance.nodeOs }} / {{ instance.nodeArch }} · {{ instance.origin === 'agent' ? 'XiaoYu 创建' : instance.origin === 'visual' ? '可视化创建' : '现有实例接管' }}</p>
        <p v-if="instance.gameVersion || instance.serverType"><small>{{ instance.gameVersion }}<span v-if="instance.serverType"> · {{ instance.serverType }}</span><span v-if="instance.address"> · {{ instance.address }}</span></small></p>
        <small>{{ instance.installPath }}</small>
        <MinecraftInstance v-if="instance.gameId === 'minecraft.java'" :instance="instance" @changed="app.loadGameInstances()" />
        <div class="chip-row">
          <span v-for="capability in instance.capabilities" :key="capability" class="chip">{{ capability }}</span>
        </div>
      </article>
    </div>

    <EmptyState v-else-if="!app.gameInstancesLoading" icon="servers" title="还没有可管理的服务器实例" description="0.3.0 开始，真实 DST 集群与 Minecraft 可视化/XiaoYu 部署都进入同一个 GameInstance 合同，不维护两套服务器记录。">
      <button class="btn btn--disabled" disabled>从游戏库部署，或让 XiaoYu 一句话开服</button>
    </EmptyState>
  </section>
</template>
