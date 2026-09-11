<script setup lang="ts">
withDefaults(defineProps<{ tone?: 'success' | 'warning' | 'danger' | 'muted' | 'accent'; label: string; dot?: boolean }>(), {
  tone: 'muted',
  dot: true,
})
</script>

<template>
  <span class="status-pill" :class="`status-pill--${tone}`">
    <span v-if="dot" class="status-pill__dot"></span>
    {{ label }}
  </span>
</template>
