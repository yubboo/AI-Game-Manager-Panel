<script setup lang="ts">
import EmptyState from '../../shared/components/EmptyState.vue'
</script>

<template>
  <section class="page">
    <div class="page-heading"><div><span class="eyebrow">SERVER INSTANCES</span><h1>服务器</h1><p>未来这里集中管理所有游戏的服务器实例，而不是按游戏拆成多个应用。</p></div></div>
    <EmptyState icon="servers" badge="PLATFORM PREVIEW" title="还没有服务器实例" description="游戏业务尚未启用。本页面只用于确认 AI Game Manager Panel 的平台级信息架构和视觉风格。">
      <button class="btn btn--disabled" disabled>＋ 创建服务器（后续版本）</button>
    </EmptyState>
  </section>
</template>
