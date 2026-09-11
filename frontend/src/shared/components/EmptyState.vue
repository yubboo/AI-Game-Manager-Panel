<script setup lang="ts">
import type { AppIconName } from '../types/ui'
import AppIcon from './AppIcon.vue'

defineProps<{ icon: AppIconName; title: string; description: string; badge?: string }>()
</script>

<template>
  <div class="empty-state panel">
    <div class="empty-state__icon"><AppIcon :name="icon" /></div>
    <span v-if="badge" class="eyebrow">{{ badge }}</span>
    <h2>{{ title }}</h2>
    <p>{{ description }}</p>
    <div class="empty-state__slot"><slot /></div>
  </div>
</template>
