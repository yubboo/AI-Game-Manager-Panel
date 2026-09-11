<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import StatusPill from '../../../../shared/components/StatusPill.vue'
import { backend } from '../../../../shared/api/backend'
import type { DSTPortConfiguration, DSTPortSettings } from '../../../../shared/types/backend'

const props = defineProps<{
  clusterPath: string
  runtimeActive: boolean
}>()

const emit = defineEmits<{ configured: [] }>()

const loading = ref(false)
const saving = ref(false)
const error = ref('')
const message = ref('')
const configuration = ref<DSTPortConfiguration | null>(null)
const advanced = ref(false)
const form = ref<DSTPortSettings>(emptySettings())

function emptySettings(): DSTPortSettings {
  return {
    shardMasterPort: 10888,
    masterServerPort: 10999,
    masterSteamMasterPort: 27016,
    masterSteamAuthPort: 8766,
    cavesServerPort: 11000,
    cavesSteamMasterPort: 27017,
    cavesSteamAuthPort: 8767,
  }
}

const hasCaves = computed(() => configuration.value?.hasCaves ?? false)
const needsRepair = computed(() => !!configuration.value && !configuration.value.valid)

function copySettings(value: DSTPortSettings) {
  form.value = { ...value }
}

async function refresh() {
  if (!props.clusterPath) {
    configuration.value = null
    return
  }
  loading.value = true
  error.value = ''
  try {
    const value = await backend.dstPortConfiguration({ clusterPath: props.clusterPath })
    configuration.value = value
    copySettings(value.effective)
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err)
  } finally {
    loading.value = false
  }
}

async function applyRecommended() {
  if (!props.clusterPath || props.runtimeActive) return
  saving.value = true
  error.value = ''
  message.value = ''
  try {
    const current = configuration.value ?? await backend.dstPortConfiguration({ clusterPath: props.clusterPath })
    const result = await backend.configureDSTPorts({
      clusterPath: props.clusterPath,
      useRecommended: true,
      settings: current.recommended,
    })
    configuration.value = result.configuration
    copySettings(result.configuration.effective)
    message.value = '推荐端口已应用，AGMP 已自动备份原网络配置并重新检查。'
    emit('configured')
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err)
  } finally {
    saving.value = false
  }
}

async function saveCustom() {
  if (!props.clusterPath || props.runtimeActive) return
  saving.value = true
  error.value = ''
  message.value = ''
  try {
    const result = await backend.configureDSTPorts({
      clusterPath: props.clusterPath,
      useRecommended: false,
      settings: { ...form.value },
    })
    configuration.value = result.configuration
    copySettings(result.configuration.effective)
    message.value = '自定义端口已保存并立即生效。'
    emit('configured')
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err)
  } finally {
    saving.value = false
  }
}

watch(() => props.clusterPath, () => void refresh())
onMounted(() => void refresh())
</script>

<template>
  <section class="panel dst-network-config">
    <div class="dst-network-config__head">
      <div>
        <span class="eyebrow">NETWORK & SHARD</span>
        <h3>网络与 Shard 配置</h3>
        <p>普通用户直接使用 AGMP 推荐配置即可；只有多开服务器或特殊网络环境才需要自定义端口。</p>
      </div>
      <div class="dst-network-config__head-actions">
        <StatusPill v-if="loading" tone="warning" label="读取中" />
        <StatusPill v-else-if="configuration?.valid" tone="success" label="配置正常" />
        <StatusPill v-else tone="danger" label="需要修复" />
        <button class="btn btn--secondary btn--compact" :disabled="loading || saving" @click="refresh">重新检测</button>
      </div>
    </div>

    <div v-if="runtimeActive" class="notice notice--warning">Master / Caves 运行期间不能修改端口，请先正常停止服务器。</div>
    <div v-if="error" class="notice notice--error">{{ error }}</div>
    <div v-if="message" class="notice notice--success">{{ message }}</div>

    <div v-if="configuration" class="dst-network-config__body">
      <article class="dst-network-recommended" :class="{ 'is-needed': needsRepair }">
        <div>
          <span class="eyebrow">RECOMMENDED</span>
          <h4>{{ needsRepair ? '检测到端口缺失或重复' : 'AGMP 推荐端口方案' }}</h4>
          <p v-if="needsRepair">无需理解 server.ini。点击一次即可自动补齐 Master / Caves / Shard 所需端口。</p>
          <p v-else>当前配置已经完整。需要恢复标准方案时也可以随时一键应用。</p>
        </div>
        <div class="dst-network-recommended__ports">
          <span>Shard <b>{{ configuration.recommended.shardMasterPort || '—' }}</b></span>
          <span>地面 <b>{{ configuration.recommended.masterServerPort }}</b></span>
          <span v-if="configuration.hasCaves">洞穴 <b>{{ configuration.recommended.cavesServerPort }}</b></span>
        </div>
        <button class="btn btn--primary" :disabled="saving || runtimeActive" @click="applyRecommended">
          {{ saving ? '正在应用…' : (needsRepair ? '一键修复并应用' : '使用推荐配置') }}
        </button>
      </article>

      <div v-if="configuration.missing.length" class="dst-network-problems">
        <strong>AGMP 将自动补齐：</strong>
        <span v-for="item in configuration.missing" :key="item">{{ item }}</span>
      </div>
      <div v-if="configuration.collisions.length" class="dst-network-problems is-danger">
        <strong>检测到重复端口：</strong>
        <span v-for="item in configuration.collisions" :key="item.port">{{ item.port }} · {{ item.uses.map((use) => `${use.shardName}/${use.purpose}`).join('、') }}</span>
      </div>

      <button type="button" class="dst-network-advanced-toggle" @click="advanced = !advanced">
        <span>{{ advanced ? '收起高级端口设置' : '高级：自定义端口' }}</span>
        <small>{{ advanced ? '▲' : '▼' }}</small>
      </button>

      <form v-if="advanced" class="dst-network-form" @submit.prevent="saveCustom">
        <div class="dst-network-form__section">
          <div><span class="eyebrow">MASTER</span><h4>地面</h4></div>
          <label><span>玩家连接端口</span><input v-model.number="form.masterServerPort" type="number" min="1024" max="65535" /></label>
          <label><span>Steam 主端口</span><input v-model.number="form.masterSteamMasterPort" type="number" min="1024" max="65535" /></label>
          <label><span>Steam 认证端口</span><input v-model.number="form.masterSteamAuthPort" type="number" min="1024" max="65535" /></label>
        </div>
        <div v-if="configuration.hasCaves" class="dst-network-form__section">
          <div><span class="eyebrow">CAVES</span><h4>洞穴</h4></div>
          <label><span>玩家连接端口</span><input v-model.number="form.cavesServerPort" type="number" min="1024" max="65535" /></label>
          <label><span>Steam 主端口</span><input v-model.number="form.cavesSteamMasterPort" type="number" min="1024" max="65535" /></label>
          <label><span>Steam 认证端口</span><input v-model.number="form.cavesSteamAuthPort" type="number" min="1024" max="65535" /></label>
        </div>
        <div v-if="configuration.hasCaves" class="dst-network-form__section dst-network-form__section--shard">
          <div><span class="eyebrow">SHARD LINK</span><h4>地面 / 洞穴内部联动</h4></div>
          <label><span>Shard 内部端口</span><input v-model.number="form.shardMasterPort" type="number" min="1024" max="65535" /></label>
          <p>这是 Master 与 Caves 在本机之间使用的内部联动端口，普通用户保持推荐值即可。</p>
        </div>
        <div class="dst-network-form__actions">
          <button type="button" class="btn btn--secondary" :disabled="saving" @click="configuration && copySettings(configuration.recommended)">恢复推荐值</button>
          <button type="submit" class="btn btn--primary" :disabled="saving || runtimeActive">{{ saving ? '保存中…' : '保存并应用' }}</button>
        </div>
      </form>
    </div>
  </section>
</template>
