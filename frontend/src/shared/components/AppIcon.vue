<script setup lang="ts">
import type { AppIconName } from '../types/ui'

defineProps<{ name: AppIconName }>()
</script>

<template>
  <svg class="app-icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
    <path v-if="name === 'home'" d="M3.5 10.8 12 3.8l8.5 7v8.4a1.8 1.8 0 0 1-1.8 1.8H5.3a1.8 1.8 0 0 1-1.8-1.8v-8.4Z" />
    <path v-else-if="name === 'servers'" d="M5 4h14a2 2 0 0 1 2 2v4H3V6a2 2 0 0 1 2-2Zm-2 10h18v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4Zm4-7h.01M7 17h.01M11 7h6M11 17h6" />
    <path v-else-if="name === 'games'" d="M8.2 8.2 6.4 16a2.3 2.3 0 0 0 4 2l1.6-2h0l1.6 2a2.3 2.3 0 0 0 4-2l-1.8-7.8A3.8 3.8 0 0 0 12 5.3a3.8 3.8 0 0 0-3.8 2.9ZM7.7 11h3.2M9.3 9.4v3.2M15.8 10.2h.01M17.2 12h.01" />
    <path v-else-if="name === 'nodes'" d="M12 3v5m0 8v5M4.5 7.5l4.3 2.4m6.4 4.2 4.3 2.4m0-9-4.3 2.4m-6.4 4.2-4.3 2.4M12 8.5 15.2 10v4L12 15.5 8.8 14v-4L12 8.5Z" />
    <path v-else-if="name === 'files'" d="M3.5 6.5h6l2 2H20.5v10a2 2 0 0 1-2 2h-13a2 2 0 0 1-2-2v-12Z" />
    <path v-else-if="name === 'terminal'" d="M4 5h16v14H4V5Zm3 4 3 3-3 3m5 0h5" />
    <path v-else-if="name === 'logs'" d="M5 4h14v16H5V4Zm3 4h8M8 12h8M8 16h5" />
    <path v-else-if="name === 'scheduler'" d="M5 4v3m14-3v3M4 9h16M5 6h14v14H5V6Zm4 7h3v3H9v-3Z" />
    <path v-else-if="name === 'backup'" d="M4 7h16v12H4V7Zm3-3h10v3H7V4Zm5 6v6m-2-2 2 2 2-2" />
    <path v-else-if="name === 'network'" d="M5 12a7 7 0 0 1 14 0M8 15a4 4 0 0 1 8 0m-4 4h.01" />
    <path v-else-if="name === 'plugins'" d="M9 3h6v5h5v6h-5v7H9v-7H4V8h5V3Z" />
    <path v-else-if="name === 'environment'" d="M4 5h16v11H4V5Zm4 15h8m-4-4v4M7 9h3m-3 3h6" />
    <path v-else-if="name === 'users'" d="M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8Zm7-1a3 3 0 1 0 0-6m-13 16c0-4 2.7-6 6-6s6 2 6 6m1-6c2.8 0 5 1.7 5 5" />
    <path v-else-if="name === 'ai'" d="M12 3v3m0 12v3M3 12h3m12 0h3M7 7l2 2m6 6 2 2m0-10-2 2m-6 6-2 2m2-5a3 3 0 1 0 6 0 3 3 0 0 0-6 0Z" />
    <path v-else-if="name === 'developer'" d="m8 8-4 4 4 4m8-8 4 4-4 4m-3-10-2 12" />
    <path v-else-if="name === 'settings'" d="M12 8.5A3.5 3.5 0 1 0 12 15a3.5 3.5 0 0 0 0-6.5Zm0-5 1 2.1 2.3.5 1.8-1.5 1.9 1.9-1.5 1.8.5 2.3 2.1 1v2.8l-2.1 1-.5 2.3 1.5 1.8-1.9 1.9-1.8-1.5-2.3.5-1 2.1H10l-1-2.1-2.3-.5-1.8 1.5L3 17.5l1.5-1.8-.5-2.3-2.1-1v-2.8l2.1-1 .5-2.3L3 4.5l1.9-1.9 1.8 1.5L9 3.6l1-2.1h2Z" />
    <path v-else-if="name === 'fire'" d="M13.8 3.2c.5 3-1.2 4.4-2.4 5.8-1.1 1.2-1.5 2.4-.7 4.1.7-1.1 1.6-1.8 2.8-2.5 2 1.8 3.1 3.8 2.6 6.1-.5 2.5-2.6 4.3-5.1 4.3-3.3 0-5.8-2.6-5.8-5.9 0-3.9 2.7-6.2 5.1-8.2 1.1-.9 2.3-1.9 3.5-3.7Z" />
    <path v-else-if="name === 'check'" d="m5 12.5 4.1 4.1L19 6.7" />
    <path v-else d="M12 3.5 13.7 9l5.3 1.8-5.3 1.7L12 18l-1.7-5.5L5 10.8 10.3 9 12 3.5Z" />
  </svg>
</template>
