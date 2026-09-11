<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useAppStore } from '../store/app'

const route = useRoute()
const app = useAppStore()
const moduleId = computed(() => String(route.meta.moduleId ?? ''))
const module = computed(() => app.platformConfig?.modules.modules.find(item => item.id === moduleId.value))
</script>

<template>
  <section class="page platform-module-page">
    <div class="page-heading row-between">
      <div>
        <span class="eyebrow">AI GAME MANAGER MODULE</span>
        <h1>{{ module?.name ?? String(route.meta.title ?? '平台模块') }}</h1>
        <p>Phase 1 已冻结入口、目录、配置与职责边界；后续阶段只在既定模块内补充真实 Go Core 能力。</p>
      </div>
      <span class="phase-badge">{{ module?.phase ?? '规划中' }}</span>
    </div>
    <article class="panel module-placeholder-card">
      <div class="module-placeholder-card__head">
        <div><strong>{{ module?.category ?? '平台' }}</strong><span>{{ module?.status ?? 'skeleton' }}</span></div>
        <p>该入口不是“假装已完成”，而是正式项目骨架占位。真实功能完成并通过实机验收后才会解除占位状态。</p>
      </div>
      <div class="module-placeholder-grid">
        <div v-for="feature in module?.features ?? []" :key="feature"><span>规划能力</span><strong>{{ feature }}</strong></div>
      </div>
    </article>
  </section>
</template>
