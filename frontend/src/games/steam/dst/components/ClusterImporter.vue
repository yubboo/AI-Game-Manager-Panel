<script setup lang="ts">
import { computed, ref } from 'vue'
import StatusPill from '../../../../shared/components/StatusPill.vue'
import { backend } from '../../../../shared/api/backend'
import type { DSTEnvironment, DSTClusterImportResult } from '../../../../shared/types/backend'

const props = defineProps<{ environment?: DSTEnvironment | null; runtimeActive?: boolean }>()
const emit = defineEmits<{ imported: [result: DSTClusterImportResult]; refresh: [] }>()

const sourcePath = ref('')
const targetName = ref('')
const loading = ref(false)
const message = ref('')
const lastResult = ref<DSTClusterImportResult | null>(null)

const localClusters = computed(() => (props.environment?.clusters ?? []).filter(cluster => cluster.distribution === 'steam' && cluster.source === 'local' && cluster.shards.some(shard => shard.name.toLowerCase() === 'master')))

function chooseLocal(path: string, name: string) {
  sourcePath.value = path
  targetName.value = `${name}_Server`
}

async function importCluster() {
  if (!sourcePath.value.trim()) return
  loading.value = true
  message.value = ''
  try {
    const result = await backend.importDSTCluster({ sourcePath: sourcePath.value.trim(), targetName: targetName.value.trim() })
    lastResult.value = result
    message.value = `导入完成：复制 ${result.copiedFiles} 个文件。${result.tokenSkipped ? '旧 Token 已按安全策略跳过，请单独配置新令牌。' : '接下来请配置服务器令牌。'}`
    emit('imported', result)
    emit('refresh')
  } catch (error) {
    message.value = error instanceof Error ? error.message : String(error)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <article class="panel workspace-panel workspace-panel--wide dst-cluster-importer">
    <div class="workspace-panel__heading">
      <div><span class="eyebrow">CLUSTER IMPORT</span><h3>导入已有世界</h3><p>把客户端 LOCAL 世界或其他完整 Cluster 复制到 Klei SERVER 根目录，供 Dedicated Server 使用。原目录不会被修改。</p></div>
      <StatusPill :tone="localClusters.length ? 'success' : 'muted'" :label="`${localClusters.length} 个本地世界`" />
    </div>

    <div v-if="localClusters.length" class="dst-local-cluster-list">
      <button v-for="cluster in localClusters" :key="cluster.path" class="dst-local-cluster" :disabled="loading || runtimeActive" @click="chooseLocal(cluster.path, cluster.name)">
        <div><strong>{{ cluster.name }}</strong><small>{{ cluster.path }}</small></div><span>选中导入 →</span>
      </button>
    </div>
    <div v-else class="notice">没有在当前 Steam 用户目录中发现 LOCAL Cluster。仍可以手动填写其他完整 Cluster 文件夹路径。</div>

    <div class="dst-cluster-import-form">
      <div class="dst-runtime-field"><label for="dst-import-source">源 Cluster 文件夹</label><input id="dst-import-source" v-model="sourcePath" type="text" placeholder="包含 cluster.ini 和 Master/server.ini 的目录" :disabled="loading || runtimeActive" /></div>
      <div class="dst-runtime-field"><label for="dst-import-name">新 SERVER Cluster 名称</label><input id="dst-import-name" v-model="targetName" type="text" placeholder="留空则使用原文件夹名" :disabled="loading || runtimeActive" /></div>
      <button class="btn btn--primary" :disabled="loading || runtimeActive || !sourcePath.trim()" @click="importCluster">{{ loading ? '正在复制世界…' : '复制到 Dedicated Server 根目录' }}</button>
    </div>

    <div class="notice notice--warning">安全规则：导入世界时 <strong>不会复制 cluster_token.txt</strong>。Token 必须通过“令牌管理”单独应用，避免把旧服务器凭据误带到新 Cluster。</div>
    <div v-if="runtimeActive" class="notice notice--warning">服务器运行期间不执行 Cluster 导入，请先停服。</div>
    <div v-if="message" class="notice" :class="{ 'notice--error': !lastResult }">{{ message }}</div>
  </article>
</template>
