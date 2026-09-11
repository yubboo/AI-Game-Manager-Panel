<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import AppIcon from './AppIcon.vue'
import { useAppStore } from '../store/app'

const props = defineProps<{ feature: string }>()
const app = useAppStore()
const router = useRouter()
const stateText = computed(() => {
  const state = app.license?.state || 'UNLICENSED'
  const map: Record<string, string> = {
    UNLICENSED: '未激活', ACTIVE: '已激活', EXPIRING: '即将到期', EXPIRED: '已过期',
    DEVICE_MISMATCH: '设备不匹配', REVOKED: '已吊销', SEAT_LIMIT: '设备席位已满',
    OFFLINE_GRACE: '离线宽限期', SERVER_UNREACHABLE: '许可证服务器不可达',
  }
  return map[state] || state
})
</script>

<template>
  <section class="page license-gate-page">
    <article class="panel license-gate-card">
      <div class="license-gate-icon"><AppIcon name="settings" /></div>
      <span class="eyebrow">AI GAME MANAGER LICENSE</span>
      <h1>此功能需要有效授权</h1>
      <p>账号系统只负责确认“你是谁”；许可证系统负责确认当前设备可以使用哪些高级功能。登录不会因为未授权而被阻断。</p>
      <div class="license-gate-status"><span>当前状态</span><strong>{{ stateText }}</strong></div>
      <div class="license-gate-status"><span>所需权限</span><code>{{ props.feature }}</code></div>
      <button class="btn btn--primary" type="button" @click="router.push('/settings?section=license')">前往授权与许可证</button>
    </article>
  </section>
</template>
