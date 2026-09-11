<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import type { PlatformApprovalMode } from '../../../shared/types/backend'

const props = defineProps<{
  modes: PlatformApprovalMode[]
  modelValue: string
  disabled?: boolean
  saving?: boolean
}>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
const open = ref(false)
const root = ref<HTMLElement | null>(null)

const current = computed(() => props.modes.find(item => item.id === props.modelValue) ?? props.modes[0])

function close() {
  open.value = false
}

function choose(id: string) {
  if (props.disabled || props.saving) return
  if (id !== props.modelValue) emit('update:modelValue', id)
  close()
}

function onDocumentPointerDown(event: PointerEvent) {
  if (!open.value) return
  const target = event.target
  if (target instanceof Node && root.value?.contains(target)) return
  close()
}

function onDocumentKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape' && open.value) {
    event.preventDefault()
    close()
  }
}

onMounted(() => {
  document.addEventListener('pointerdown', onDocumentPointerDown, true)
  document.addEventListener('keydown', onDocumentKeydown, true)
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', onDocumentPointerDown, true)
  document.removeEventListener('keydown', onDocumentKeydown, true)
})
</script>

<template>
  <div ref="root" class="approval-control">
    <button
      class="approval-trigger"
      type="button"
      :disabled="disabled || saving"
      :aria-expanded="open"
      aria-haspopup="menu"
      @click.stop="open = !open"
    >
      <span class="approval-shield">!</span>
      <span>{{ saving ? '正在保存…' : (current?.label ?? '请求批准') }}</span>
      <span class="approval-chevron">⌄</span>
    </button>

    <div v-if="open" class="approval-popover" role="menu" @click.stop>
      <div class="approval-popover__head"><strong>如何批准 AI 操作？</strong><span>由最高管理员决定</span></div>
      <button
        v-for="mode in modes"
        :key="mode.id"
        type="button"
        role="menuitemradio"
        class="approval-option"
        :class="{ active: mode.id === modelValue }"
        :aria-checked="mode.id === modelValue"
        :disabled="disabled || saving"
        @click="choose(mode.id)"
      >
        <span class="approval-option__icon">{{ mode.id === 'full' ? '!' : mode.id === 'risk' ? '◇' : '☝' }}</span>
        <span><strong>{{ mode.label }}</strong><small>{{ mode.description }}</small></span>
        <b v-if="mode.id === modelValue">✓</b>
      </button>
      <p v-if="disabled">当前账号只能查看审批模式；只有最高管理员可以切换。</p>
      <p v-else>“完全访问权限”允许 AI 自主使用已注册且已启用的 AGMP Tool，但不会关闭参数校验、作用域限制、备份/回滚条件、审计和结果验证。任意 Shell 仍需平台策略显式开放。</p>
    </div>
  </div>
</template>
