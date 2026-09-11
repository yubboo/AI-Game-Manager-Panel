<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import StatusPill from '../../shared/components/StatusPill.vue'
import { useAppStore } from '../../shared/store/app'
import DstWorkspace from '../../games/steam/dst/DstWorkspace.vue'

const app = useAppStore()
const activeGameId = ref('steam.dst')

const gameTabs = computed(() => (app.platformConfig?.games.templates ?? []).map(item => ({
  ...item,
  mark: item.nameZh.slice(0, 1) || '+',
  state: item.state === 'supported' ? 'active' : 'planned',
  description: item.state === 'supported' ? `${item.family} · 当前已接入` : `${item.family} · 骨架已预留`,
})))
const activeGame = computed(() => gameTabs.value.find((item) => item.id === activeGameId.value) ?? gameTabs.value[0])
const dstState = computed(() => app.gameWorkspace?.game.catalog.id === 'steam.dst' ? app.gameWorkspace.game : null)
const dstReady = computed(() => !!app.dstDedicated?.installation.valid && !!app.dstDedicated?.confDir.valid)

async function refreshDst() {
  await app.refreshDSTWorkspace()
}

async function selectGame(id: string) {
  activeGameId.value = id
  if (id === 'steam.dst') await refreshDst()
}

function tabState(id: string) {
  if (id !== 'steam.dst') return { tone: 'muted' as const, label: '骨架已预留' }
  if (app.gameWorkspaceLoading || app.dstDedicatedLoading) return { tone: 'warning' as const, label: '检测中' }
  if (app.gameWorkspaceError || app.dstDedicatedError) return { tone: 'danger' as const, label: '检测失败' }
  if (dstReady.value) return { tone: 'success' as const, label: '就绪' }
  return { tone: 'warning' as const, label: '需准备' }
}

onMounted(() => {
  if (!app.gameWorkspace && !app.gameWorkspaceLoading) void refreshDst()
})
</script>

<template>
  <section class="page game-workspaces-page">
    <div class="game-workspace-canvas">
      <div class="page-heading game-workspace-heading">
        <div>
          <span class="eyebrow">GAME WORKSPACES</span>
          <h1>游戏库</h1>
          <p>选择要开服的游戏后，进入对应的专属工作台。AI Game Manager Panel 不把 Steam 全部应用塞进主界面。</p>
        </div>
      </div>

      <nav class="game-workspace-tabs" aria-label="游戏开服工作台">
        <button
          v-for="item in gameTabs"
          :key="item.id"
          class="game-workspace-tab"
          :class="{ active: activeGameId === item.id, planned: item.state === 'planned' }"
          @click="selectGame(item.id)"
        >
          <span class="game-workspace-tab__mark">{{ item.mark }}</span>
          <span class="game-workspace-tab__copy"><strong>{{ item.nameZh }}</strong><small>{{ item.description }}</small></span>
          <StatusPill :tone="tabState(item.id).tone" :label="tabState(item.id).label" />
        </button>
      </nav>

      <DstWorkspace
        v-if="activeGameId === 'steam.dst'"
        :game="dstState"
        :loading="app.gameWorkspaceLoading || app.dstDedicatedLoading"
        :error="app.gameWorkspaceError || app.dstDedicatedError"
        :scanned-at="app.gameWorkspace?.scannedAt"
        :duration-ms="app.gameWorkspace?.durationMs"
        :steam-detected="app.gameWorkspace?.steam.detected ?? false"
        :steam-install-path="app.gameWorkspace?.steam.installPath"
        :warnings="[...(app.gameWorkspace?.warnings ?? []), ...(app.dstDedicated?.warnings ?? [])]"
        :dedicated="app.dstDedicated"
        :environment="app.dstEnvironment"
        @refresh="refreshDst"
      />

      <article v-else-if="activeGame && activeGame.state === 'planned'" class="panel game-planned-panel">
        <div class="game-planned-panel__mark">{{ activeGame.mark }}</div>
        <span class="eyebrow">COMING GAME PROVIDER</span>
        <h2>{{ activeGame.nameZh }}</h2>
        <strong>{{ activeGame.nameEn }}</strong>
        <p>当前只预留入口，不伪装成已支持。对应 Game Provider 完成以后才开放真实检测和服务器管理。</p>
        <StatusPill tone="muted" label="开发中" />
      </article>

      <article v-else class="panel game-planned-panel custom-game-panel">
        <div class="game-planned-panel__mark">+</div>
        <span class="eyebrow">GAME CONFIG</span>
        <h2>未找到游戏模板</h2>
        <p>请检查根目录 configs/games.json。</p>
      </article>
    </div>
  </section>
</template>
