<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAppStore } from '../../shared/store/app'

const app = useAppStore()
const router = useRouter()
const games = computed(() => app.platformConfig?.games.templates ?? [])
const module = computed(() => app.platformConfig?.modules.modules.find(item => item.id === 'deployment'))
</script>

<template>
  <section class="page deployment-page">
    <div class="page-heading row-between">
      <div><span class="eyebrow">ONE-CLICK DEPLOYMENT</span><h1>一键部署</h1><p>AI 与手动部署共用同一套 AI Game Manager Panel Core；不配置 AI 也能完成全部可视化部署操作。</p></div>
      <span class="phase-badge">{{ module?.phase ?? '部署中心阶段' }}</span>
    </div>

    <div class="deployment-mode-grid">
      <article class="panel deployment-mode-card active">
        <span>MANUAL</span><h2>手动可视化部署</h2><p>适合不配置 AI 的用户。通过向导完成环境检查、游戏下载、配置、端口、备份和启动。</p>
        <button class="btn btn--primary" type="button" @click="router.push('/games')">打开当前游戏工作台</button>
      </article>
      <article class="panel deployment-mode-card">
        <span>AI AGENT</span><h2>AI 一句话部署</h2><p>配置 AI Provider 后，由 Planner 生成计划，通过权限审批后调用同一套 AI Game Manager Panel Tool 执行。</p>
        <button class="btn btn--secondary" type="button" @click="router.push('/')">返回 小鱼</button>
      </article>
    </div>

    <div class="section-heading"><div><span class="eyebrow">GAME TEMPLATES</span><h2>部署模板</h2></div><span class="section-heading__hint">模板来源：configs/games.json</span></div>
    <div class="module-placeholder-grid game-template-grid">
      <article v-for="game in games" :key="game.id" class="panel game-template-card">
        <div><span>{{ game.family }}</span><strong>{{ game.nameZh }}</strong><small>{{ game.nameEn }}</small></div>
        <b :class="game.state">{{ game.state === 'supported' ? '已接入' : '骨架已预留' }}</b>
      </article>
    </div>
  </section>
</template>
