<script setup lang="ts">
import StatusPill from '../../../../shared/components/StatusPill.vue'
import type { DSTPreflightResult } from '../../../../shared/types/backend'

defineProps<{
  result: DSTPreflightResult | null
  loading?: boolean
  error?: string
}>()

const emit = defineEmits<{ configureToken: []; configureNetwork: []; importWorld: []; refresh: [] }>()

function tone(severity: string) {
  if (severity === 'blocker') return 'danger' as const
  if (severity === 'warning') return 'warning' as const
  return 'success' as const
}
</script>

<template>
  <section class="panel dst-preflight">
    <div class="dst-preflight__head">
      <div><span class="eyebrow">STARTUP PREFLIGHT</span><h3>启动前检查</h3><p>真正创建 Dedicated Server 进程之前检查文件、Cluster、Master 和 Klei 服务器令牌。</p></div>
      <div class="dst-preflight__actions">
        <StatusPill v-if="loading" tone="warning" label="检查中" />
        <StatusPill v-else-if="result?.ready" tone="success" label="允许启动" />
        <StatusPill v-else tone="danger" :label="result ? `${result.blockers} 个阻止项` : '未检查'" />
        <button class="btn btn--secondary btn--compact" :disabled="loading" @click="emit('refresh')">重新检查</button>
      </div>
    </div>
    <div v-if="error" class="notice notice--error">{{ error }}</div>
    <div v-if="result" class="dst-preflight__grid">
      <article v-for="item in result.checks" :key="item.code" class="dst-preflight-check" :class="`is-${item.severity}`">
        <StatusPill :tone="tone(item.severity)" :label="item.severity === 'ok' ? '通过' : item.severity === 'warning' ? '提醒' : '阻止'" />
        <div><strong>{{ item.label }}</strong><p>{{ item.message }}</p><small v-if="item.action">{{ item.action }}</small></div>
        <button v-if="item.code === 'token' && item.severity === 'blocker'" class="btn btn--primary btn--compact" @click="emit('configureToken')">配置令牌</button>
        <button v-else-if="['ports_explicit', 'ports_config', 'ports_warning'].includes(item.code) && item.severity !== 'ok'" class="btn btn--primary btn--compact" @click="emit('configureNetwork')">一键修复 / 设置端口</button>
        <button v-else-if="item.code === 'cluster' && item.severity === 'blocker'" class="btn btn--secondary btn--compact" @click="emit('importWorld')">导入世界</button>
      </article>
    </div>
  </section>
</template>
