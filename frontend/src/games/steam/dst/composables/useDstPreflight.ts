import { ref, watch, type Ref } from 'vue'
import { backend } from '../../../../shared/api/backend'
import type { DSTPreflightResult } from '../../../../shared/types/backend'

export function useDstPreflight(clusterPath: Ref<string>) {
  const result = ref<DSTPreflightResult | null>(null)
  const loading = ref(false)
  const error = ref('')
  let generation = 0

  async function refresh() {
    const path = clusterPath.value.trim()
    const current = ++generation
    if (!path) {
      result.value = null
      error.value = ''
      return
    }
    loading.value = true
    try {
      const value = await backend.dstPreflight(path)
      if (current !== generation) return
      result.value = value
      error.value = ''
    } catch (reason) {
      if (current !== generation) return
      result.value = null
      error.value = reason instanceof Error ? reason.message : String(reason)
    } finally {
      if (current === generation) loading.value = false
    }
  }

  watch(clusterPath, () => void refresh(), { immediate: true })

  return { result, loading, error, refresh }
}
